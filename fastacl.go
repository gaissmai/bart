// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"fmt"
	"io"
	"iter"
	"net/netip"
	"strings"
	"sync"

	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/nodes"
)

// FastACL provides a high-performance routing table and Access Control List (ACL)
// evaluation structure for IPv4 and IPv6 prefixes.
//
// It maintains distinct trie root nodes and prefix counters for IPv4 and IPv6 address
// families to eliminate runtime protocol branching and enable zero-allocation lookups.
type FastACL struct {
	// _ enforces copy-protection via 'go vet' -copylocks without incurring memory overhead.
	_ [0]sync.Mutex

	// Root nodes for IPv4 and IPv6 tries.
	root4 nodes.FastACLNode
	root6 nodes.FastACLNode

	// Total count of active prefixes stored per address family.
	size4 int
	size6 int
}

// rootNodeByVersion returns a pointer to the root node corresponding to the
// specified address family (IPv4 if is4 is true, IPv6 if false).
func (f *FastACL) rootNodeByVersion(is4 bool) *nodes.FastACLNode {
	if is4 {
		return &f.root4
	}
	return &f.root6
}

// sizeUpdate adjusts the prefix count for the specified address family by delta.
func (f *FastACL) sizeUpdate(is4 bool, delta int) {
	if is4 {
		f.size4 += delta
		return
	}
	f.size6 += delta
}

// Insert adds a prefix to the FastACL table.
//
// Operation is idempotent: inserting an existing prefix is a no-op that preserves
// the table state and leaves the size counter unchanged. Non-canonical prefixes
// are automatically normalized using pfx.Masked(). Invalid or uninitialized
// prefixes are silently ignored.
func (f *FastACL) Insert(pfx netip.Prefix) {
	if !pfx.IsValid() {
		return
	}

	// Canonicalize prefix to ensure host bits are zeroed.
	pfx = pfx.Masked()

	is4 := pfx.Addr().Is4()
	n := f.rootNodeByVersion(is4)

	// If the prefix already exists, avoid updating size counters.
	if exists := n.Insert(pfx, 0); exists {
		return
	}

	// Update the prefix count for the corresponding address family.
	f.sizeUpdate(is4, 1)
}

// Delete removes a prefix from the FastACL table.
//
// Operation is idempotent: attempting to delete a prefix that does not exist in
// the table is a no-op that leaves table state and size counters unchanged.
// Non-canonical prefixes are automatically normalized using pfx.Masked().
// Invalid or uninitialized prefixes are silently ignored.
func (f *FastACL) Delete(pfx netip.Prefix) {
	if !pfx.IsValid() {
		return
	}

	// Canonicalize prefix to ensure host bits are zeroed.
	pfx = pfx.Masked()
	is4 := pfx.Addr().Is4()

	n := f.rootNodeByVersion(is4)

	// If the prefix existed and was removed, decrement the corresponding address family counter.
	if exists := n.Delete(pfx); exists {
		f.sizeUpdate(is4, -1)
	}
}

// Contains reports whether any stored prefix in the FastACL table covers the given IP address.
// An invalid netip.Addr returns false.
//
// Any IPv6 zone identifier is stripped and has no effect on the lookup result.
func (f *FastACL) Contains(ip netip.Addr) bool {
	// Speed is top priority: skip explicit ip.IsValid() call.
	// For an invalid netip.Addr{}, ip.AsSlice() returns nil, causing the loop
	// to immediately exit and return false.
	is4 := ip.Is4()

	n := f.rootNodeByVersion(is4)

	for _, octet := range ip.AsSlice() {
		// Short-circuit: any matching local prefix in this node is sufficient
		if n.PrefixCount() != 0 && n.Contains(art.OctetToIdx(octet)) {
			return true
		}

		// Short-circuit: any matching fringe in this node is sufficient
		if n.Fringes.Test(octet) {
			return true
		}

		// Stop traversal if no child or leaf exists for the current octet
		if !n.Children.Test(octet) {
			return false
		}
		kid := n.MustGetChild(octet)

		// Terminal leaf node encountered
		if leaf, ok := kid.(*nodes.CIDRLeaf); ok {
			// Strip IPv6 zone only when evaluating leaf.Prefix.Contains.
			// Deferred to this branch so short-circuiting hits do not pay the stripping penalty.
			//
			// but netip.Addr.withoutZone  is not exported :-(
			// and netip.Addr.WithZone("") is not inlinable, so we have to resort to this clever trick:
			// https://github.com/gaissmai/bart/pull/418#issuecomment-5735613506
			if !is4 {
				ip = netip.PrefixFrom(ip, 0).Addr()
			}
			return leaf.Prefix().Contains(ip)
		}

		// Internal trie node: descend to next level
		n = kid.(*nodes.FastACLNode)
	}

	return false
}

// ContainsPrefix reports whether pfx is covered by any prefix present in the FastACL table.
//
// A prefix is covered if an identical prefix or a broader enclosing supernet exists
// in the trie. Non-canonical prefixes are automatically normalized using pfx.Masked().
//
// It returns false if pfx is invalid or if no covering prefix exists in the table.
func (f *FastACL) ContainsPrefix(pfx netip.Prefix) bool {
	// Guard against uninitialized or invalid prefixes early.
	if !pfx.IsValid() {
		return false
	}

	// Canonicalize input prefix to ensure host bits are zeroed.
	pfx = pfx.Masked()
	pfxLen := pfx.Bits()
	ip := pfx.Addr()
	is4 := ip.Is4()
	octets := ip.AsSlice()

	// Calculate full 8-bit byte strides and remaining bit count for the probe prefix length.
	strideCount, modBits := nodes.DivMod8(pfxLen)

	// Fetch root node for IPv4 or IPv6 address family.
	n := f.rootNodeByVersion(is4)

	// Traverse the trie top-down across byte strides up to the probe's target depth.
	for depth, octet := range octets {
		depth &= nodes.DepthMask // BCE

		// stepped one past the last stride of interest
		if depth > strideCount {
			return false
		}

		// 1. Check for covering local prefixes within the current stride node.
		if n.PrefixCount() != 0 {
			// Intermediate strides evaluate full 8-bit host routes;
			// only the terminal stride uses the remaining bit count (modBits).
			var idx uint8
			if depth == strideCount {
				idx = art.PfxToIdx(octet, modBits)
			} else {
				idx = art.OctetToIdx(octet)
			}

			// Check if any local prefix covers the target index in the Complete Binary Tree (CBT).
			if n.Contains(idx) {
				return true
			}
		}

		// 2. Check for a stride-aligned fringe boundary matching at /8, /16, /24, etc.
		fringeBits := (depth + 1) << 3
		// Verify that the fringe's prefix length is broader/equal and encloses
		// the target prefix.
		if fringeBits <= pfxLen && n.Fringes.Test(octet) {
			return true
		}

		// 3. Early termination: stop traversal if no child node or leaf exists at the target octet slot.
		if !n.Children.Test(octet) {
			return false
		}

		// Retrieve child node or path-compressed leaf at the current octet slot.
		kid := n.MustGetChild(octet)

		switch kid := kid.(type) {
		case *nodes.FastACLNode:
			// Recurse into deeper trie level.
			n = kid
			continue

		case *nodes.CIDRLeaf:
			// Terminal path-compressed leaf reached: verify that the leaf's prefix length
			// is broader/equal and encloses the target prefix.
			if kid.Prefix().Bits() > pfxLen {
				return false
			}
			return kid.Prefix().Contains(ip)
		}
	}

	return false
}

// LookupPrefixLPM performs a longest-prefix-match lookup for any route covering
// the given prefix range. It searches the trie for the most specific prefix
// that encloses pfx.
//
// Non-canonical prefixes are automatically normalized using pfx.Masked().
//
// Returns the matching prefix and true if a covering prefix is found.
// Returns an invalid netip.Prefix{} and false if no match exists or if pfx is invalid.
func (f *FastACL) LookupPrefixLPM(pfx netip.Prefix) (lpmPfx netip.Prefix, ok bool) {
	// Guard against uninitialized or invalid prefixes early.
	if !pfx.IsValid() {
		return lpmPfx, ok
	}

	// Canonicalize input prefix to ensure host bits are zeroed.
	pfx = pfx.Masked()

	ip := pfx.Addr()
	pfxLen := pfx.Bits()
	is4 := ip.Is4()
	octets := ip.AsSlice()

	// Calculate full 8-bit byte strides and remaining bit count for the probe prefix length.
	strideCount, modBits := nodes.DivMod8(pfxLen)

	// Fetch root node for IPv4 or IPv6 address family.
	n := f.rootNodeByVersion(is4)

	// Fixed-size stack on the frame to store path nodes for backtracking.
	stack := [nodes.MaxTreeDepth]*nodes.FastACLNode{}

	var depth int
	var octet byte

	// Top-down traversal: Descend as deep as possible along the octet path.
LOOP:
	// find the last node on the octets path in the trie,
	for depth, octet = range octets {
		depth &= nodes.DepthMask // BCE

		// stepped one past the last stride of interest; back up to last and break
		if depth > strideCount {
			depth--
			break
		}

		// Record current node on the traversal stack for backtracking
		stack[depth] = n

		// Early exit from descent if no child or leaf exists at the target octet slot.
		if !n.Children.Test(octet) {
			break LOOP
		}
		kid := n.MustGetChild(octet)

		switch kid := kid.(type) {
		case *nodes.FastACLNode:
			// Descend deeper into next trie level
			n = kid
			continue LOOP

		case *nodes.CIDRLeaf:
			// Terminal path-compressed leaf reached: verify that the leaf's prefix length
			// is broader/equal and encloses the target prefix/IP range.
			if kid.Prefix().Bits() > pfxLen || !kid.Prefix().Contains(ip) {
				break LOOP
			}
			return kid.Prefix(), true
		}
	}

	// Backtracking phase: Unwind the stack bottom-up to locate the longest matching prefix.
	for ; depth >= 0; depth-- {
		depth &= nodes.DepthMask // BCE: Hint compiler that depth stays within bounds

		n = stack[depth]
		octet = octets[depth]

		// #############################################################################
		// 1. Check for a stride-aligned fringe boundary matching at /8, /16, /24, etc.
		//
		// Verify that the fringe's prefix length is broader/equal and encloses the target prefix.
		fringeBits := (depth + 1) << 3
		if fringeBits <= pfxLen && n.Fringes.Test(octet) {
			// Reconstruct the stride-aligned prefix from the IP address and fringe bit length.
			fringePfx, _ := ip.Prefix(fringeBits)
			return fringePfx, true
		}

		// #############################################################################
		// 2. Check for local prefixes within the current CBT node.
		if n.PrefixCount() == 0 {
			continue
		}

		// Intermediate strides evaluate full 8-bit host routes;
		// only the terminal stride uses the remaining bit count (modBits).
		var idx uint8
		if depth == strideCount {
			idx = art.PfxToIdx(octet, modBits)
		} else {
			idx = art.OctetToIdx(octet)
		}

		var topIdx uint8
		if topIdx, ok = n.LookupIdx(idx); ok {
			// Reconstruct bit length from current depth and CBT top index
			pfxBits := int(art.PfxBits(depth, topIdx))

			// Reconstruct canonical prefix from IP address and calculated bit length.
			// Invariant: art.PfxBits returns valid ranges (v4: 0..32, v6: 0..128).
			lpmPfx, _ = ip.Prefix(pfxBits)
			return lpmPfx, ok
		}
	}

	return lpmPfx, ok
}

// Supernets returns an iterator over all enclosing supernet prefixes in the FastACL table
// that cover the given target prefix range pfx.
//
// The iteration yields matching routes in reverse CIDR order:
//  1. An exact match for pfx itself (if present in the table).
//  2. Enclosing supernet routes ascending from the most specific match (longest prefix match)
//     up to the most general enclosing route (root level /0).
//
// Non-canonical inputs are automatically normalized. If pfx is invalid (!pfx.IsValid()),
// an empty sequence is returned immediately without executing yield.
func (f *FastACL) Supernets(pfx netip.Prefix) iter.Seq[netip.Prefix] {
	// Guard clause: Early exit before any processing or closure allocation logic.
	if !pfx.IsValid() {
		return nodes.EmptySeq
	}

	// Canonicalize prefix into a NEW variable.
	// This enables the compiler to capture canonicalPfx by value instead of by ref,
	// avoiding the 'moved to heap: pfx' allocation.
	canonicalPfx := pfx.Masked()

	// Traversal is performed in two phases:
	//  1. Descent Phase: Downward traversal along the octet path to locate the deepest matching node,
	//     pushing visited internal nodes onto a fixed-size stack. If a compressed path leaf (*CIDRLeaf)
	//     is encountered and covers pfx, its prefix is yielded immediately (most specific match).
	//  2. Backtracking Phase: Unwinding the stack bottom-up, calling [FastACLNode.YieldSupernets]
	//     at each step to yield stride-aligned fringe prefixes and node-internal prefixes in descending length order.
	//
	// Iteration halts immediately if the yield callback returns false.
	return func(yield func(netip.Prefix) bool) {
		pfx := canonicalPfx
		is4 := pfx.Addr().Is4()
		n := f.rootNodeByVersion(is4)

		ip := pfx.Addr()
		pfxLen := pfx.Bits()
		octets := ip.AsSlice()
		strideCount, modBits := nodes.DivMod8(pfxLen)

		// Fixed-size stack tracking traversed nodes for reverse (bottom-up) iteration.
		stack := [nodes.MaxTreeDepth]*nodes.FastACLNode{}

		// Loop tracking variables preserved for the unwinding phase.
		var depth int
		var octet byte

		// Phase 1: Descend down the trie along the octet path.
	LOOP:
		for depth, octet = range octets[:strideCount+1] {
			// Push current node onto stack before descending.
			stack[depth] = n

			// Stop descent if no child pointer exists for this octet.
			if !n.Children.Test(octet) {
				break LOOP
			}
			kid := n.MustGetChild(octet)

			switch kid := kid.(type) {
			case *nodes.FastACLNode:
				n = kid
				continue LOOP // Descend to next trie level.

			case *nodes.CIDRLeaf:
				// Ignore leaf if its prefix length is more specific than the query target.
				if kid.Prefix().Bits() > pfx.Bits() {
					break LOOP
				}

				// Yield leaf prefix if it covers the target IP.
				if kid.Prefix().Contains(ip) {
					if !yield(kid.Prefix()) {
						return
					}
				}

				// End traversal along this path.
				break LOOP
			}
		}

		// Phase 2: Backtrack bottom-up, unwinding the node stack.
		for ; depth >= 0; depth-- {
			n = stack[depth]

			// Derive the prefix index within the node's 8-bit complete binary tree (CBT).
			// Only the terminal stride uses modBits; preceding strides cover full 8-bit octets.
			var pfxIdx uint8
			octet = octets[depth]
			if depth == strideCount {
				pfxIdx = art.PfxToIdx(octet, modBits)
			} else {
				pfxIdx = art.OctetToIdx(octet)
			}

			// Yield matching node-internal prefixes in descending specificity.
			if !n.YieldSupernets(pfx, depth, octet, pfxIdx, yield) {
				return
			}
		}
	}
}

// Subnets returns an iterator over all subnets of the given prefix
// in natural CIDR sort order. This includes prefixes of the same length
// (exact match) and longer (more specific) prefixes contained
// within the given prefix.
//
// Iteration can be stopped early by breaking from the range loop.
// Returns an empty iterator if the prefix is invalid.
func (f *FastACL) Subnets(pfx netip.Prefix) iter.Seq[netip.Prefix] {
	// Guard clause: Early exit before any processing or closure allocation logic.
	if !pfx.IsValid() {
		return nodes.EmptySeq
	}

	// Canonicalize prefix into a NEW variable.
	// This enables the compiler to capture canonicalPfx by value instead of by ref,
	// avoiding the 'moved to heap: pfx' allocation.
	canonicalPfx := pfx.Masked()

	return func(yield func(netip.Prefix) bool) {
		pfx := canonicalPfx
		is4 := pfx.Addr().Is4()
		n := f.rootNodeByVersion(is4)

		ip := pfx.Addr()
		pfxLen := pfx.Bits()
		octets := ip.AsSlice()
		strideCount, modBits := nodes.DivMod8(pfxLen)

		// Traverse trie levels according to byte stride depth.
		for depth, octet := range octets {

			// Reached the target trie level matching the prefix length.
			if depth == strideCount {
				idx := art.PfxToIdx(octet, modBits)
				ptx := nodes.NewPathContext(octets, depth, idx, is4)
				n.YieldSubnets(ptx, yield)
				return
			}

			// Yield matching fringe prefixes at intermediate levels if present.
			if nodes.IsFringe(depth, pfxLen) && n.Fringes.Test(octet) {
				ptx := nodes.NewPathContext(octets, depth, octet, is4)
				if !n.YieldFringe(ptx, yield) {
					return
				}
			}

			// Branch traversal terminates if child byte bucket is empty.
			if !n.Children.Test(octet) {
				return
			}
			kid := n.MustGetChild(octet)

			// Descend deeper or evaluate leaf termination.
			switch kid := kid.(type) {
			case *nodes.FastACLNode:
				n = kid
				continue // Descend down to next trie level.

			case *nodes.CIDRLeaf:
				// Evaluate leaf containment. No subnets exist below a leaf, so iteration terminates immediately.
				if pfx.Bits() <= kid.Prefix().Bits() && pfx.Overlaps(kid.Prefix()) {
					yield(kid.Prefix())
				}
				return // Terminal leaf reached; discontinue trie descent.
			}
		}
	}
}

// Clone returns a deep copy of the FastACL table.
//
// The cloned table is completely decoupled from the original receiver instance.
// Subsequent mutations (insertions or deletions) on either instance will not
// affect the other.
func (f *FastACL) Clone() *FastACL {
	c := new(FastACL)

	c.root4 = *f.root4.CloneRec()
	c.root6 = *f.root6.CloneRec()

	c.size4 = f.size4
	c.size6 = f.size6

	return c
}

// OverlapsPrefix reports whether any prefix in the routing table overlaps with
// the given prefix. Two prefixes overlap if they share any IP addresses.
//
// The check is bidirectional: it returns true if the input prefix is covered by an existing
// route, or if any stored route is itself contained within the input prefix.
//
// Internally, the function normalizes the prefix and descends the relevant trie branch,
// using stride-based logic to identify overlap without performing a full lookup.
//
// This is useful for containment tests, route validation, or policy checks using prefix
// semantics without retrieving exact matches.
func (f *FastACL) OverlapsPrefix(pfx netip.Prefix) bool {
	if !pfx.IsValid() {
		return false
	}

	// canonicalize the prefix
	pfx = pfx.Masked()

	is4 := pfx.Addr().Is4()
	n := f.rootNodeByVersion(is4)

	return n.OverlapsPrefixAtDepth(pfx, 0)
}

// Overlaps reports whether any route in the receiver table overlaps
// with a route in the other table, in either direction.
//
// The overlap check is bidirectional: it returns true if any IP prefix
// in the receiver is covered by the other table, or vice versa.
// This includes partial overlaps, exact matches, and supernet/subnet relationships.
//
// Both IPv4 and IPv6 route trees are compared independently. If either
// tree has overlapping routes, the function returns true.
//
// This is useful for conflict detection, policy enforcement,
// or validating mutually exclusive routing domains.
func (f *FastACL) Overlaps(o *FastACL) bool {
	return f.Overlaps4(o) || f.Overlaps6(o)
}

// Overlaps4 is like [liteTable.Overlaps] but for the v4 routing table only.
func (f *FastACL) Overlaps4(o *FastACL) bool {
	if f.size4 == 0 || o.size4 == 0 {
		return false
	}
	return f.root4.Overlaps(&o.root4, 0)
}

// Overlaps6 is like [liteTable.Overlaps] but for the v6 routing table only.
func (f *FastACL) Overlaps6(o *FastACL) bool {
	if f.size6 == 0 || o.size6 == 0 {
		return false
	}
	return f.root6.Overlaps(&o.root6, 0)
}

// Aggregate compresses the table in-place by merging overlapping
// and adjacent IP prefixes into their minimal covering CIDR blocks.
func (f *FastACL) Aggregate() {
	mod4 := f.root4.AggregateRec(pathContext{Is4: true})
	mod6 := f.root6.AggregateRec(pathContext{Is4: false})

	if mod4 != 0 {
		stats := f.root4.StatsRec()
		f.size4 = stats.Prefixes + stats.Leaves + stats.Fringes
	}

	if mod6 != 0 {
		stats := f.root6.StatsRec()
		f.size6 = stats.Prefixes + stats.Leaves + stats.Fringes
	}
}

// Equal checks whether two tables are structurally and semantically equal.
// It ensures both trees (IPv4-based and IPv6-based) have the same sizes and
// recursively compares their root nodes.
//
// If V implements an `Equal(V) bool` method, its custom equality logic is used.
// Otherwise, values are compared directly using the == operator.
//
// Note: If V implements `Equal(V) bool` with a pointer receiver, the Equal
// method should handle nil receivers gracefully.
//
// ATTENTION: If V is not comparable at runtime (such as a slice or map without an `Equal`
// method), a runtime panic will occur.
func (f *FastACL) Equal(o *FastACL) bool {
	if f.size4 != o.size4 || f.size6 != o.size6 {
		return false
	}
	if o == f {
		return true
	}

	return f.root4.EqualRec(&o.root4) && f.root6.EqualRec(&o.root6)
}

// Size returns the prefix count.
func (f *FastACL) Size() int {
	return f.size4 + f.size6
}

// Size4 returns the IPv4 prefix count.
func (f *FastACL) Size4() int {
	return f.size4
}

// Size6 returns the IPv6 prefix count.
func (f *FastACL) Size6() int {
	return f.size6
}

// All returns an iterator over all canonical prefixes (both IPv4 and IPv6)
// stored in the table. Iteration order is explicitly unspecified.
func (f *FastACL) All() iter.Seq[netip.Prefix] {
	return func(yield func(netip.Prefix) bool) {
		if !f.root4.AllRec(pathContext{Is4: true}, yield) {
			return
		}

		f.root6.AllRec(pathContext{Is4: false}, yield)
	}
}

// All4 returns an iterator over all canonical IPv4 prefixes stored in the table.
// Iteration order is explicitly unspecified.
func (f *FastACL) All4() iter.Seq[netip.Prefix] {
	return func(yield func(netip.Prefix) bool) {
		f.root4.AllRec(pathContext{Is4: true}, yield)
	}
}

// All6 returns an iterator over all canonical IPv6 prefixes stored in the table.
// Iteration order is explicitly unspecified.
func (f *FastACL) All6() iter.Seq[netip.Prefix] {
	return func(yield func(netip.Prefix) bool) {
		f.root6.AllRec(pathContext{Is4: false}, yield)
	}
}

// AllSorted is like [FastACL.All] but the iteration is ordered in canonical
// CIDR prefix sort order.
func (f *FastACL) AllSorted() iter.Seq[netip.Prefix] {
	return func(yield func(netip.Prefix) bool) {
		if !f.root4.AllRecSorted(pathContext{Is4: true}, yield) {
			return
		}

		f.root6.AllRecSorted(pathContext{Is4: false}, yield)
	}
}

// AllSorted4 is like [FastACL.AllSorted] but only for the v4 routing table.
func (f *FastACL) AllSorted4() iter.Seq[netip.Prefix] {
	return func(yield func(netip.Prefix) bool) {
		f.root4.AllRecSorted(pathContext{Is4: true}, yield)
	}
}

// AllSorted6 is like [FastACL.AllSorted] but only for the v6 routing table.
func (f *FastACL) AllSorted6() iter.Seq[netip.Prefix] {
	return func(yield func(netip.Prefix) bool) {
		f.root6.AllRecSorted(pathContext{Is4: false}, yield)
	}
}

// Fprint writes a hierarchical tree diagram of the ordered CIDRs
// with default formatted payload V to w.
//
// The order from top to bottom is in ascending order of the prefix address
// and the subtree structure is determined by the CIDRs coverage.
//
//	▼
//	├─ 10.0.0.0/8 (V)
//	│  ├─ 10.0.0.0/24 (V)
//	│  └─ 10.0.1.0/24 (V)
//	├─ 127.0.0.0/8 (V)
//	│  └─ 127.0.0.1/32 (V)
//	├─ 169.254.0.0/16 (V)
//	├─ 172.16.0.0/12 (V)
//	└─ 192.168.0.0/16 (V)
//	   └─ 192.168.1.0/24 (V)
//	▼
//	└─ ::/0 (V)
//	   ├─ ::1/128 (V)
//	   ├─ 2000::/3 (V)
//	   │  └─ 2001:db8::/32 (V)
//	   └─ fe80::/10 (V)
func (f *FastACL) Fprint(w io.Writer) error {
	if w == nil && f != nil {
		return fmt.Errorf("nil writer")
	}

	// v4
	if err := f.fprint(w, true); err != nil {
		return err
	}

	// v6
	if err := f.fprint(w, false); err != nil {
		return err
	}

	return nil
}

// fprint is the version dependent adapter to fprintRec.
func (f *FastACL) fprint(w io.Writer, is4 bool) error {
	n := f.rootNodeByVersion(is4)
	if n.IsEmpty() {
		return nil
	}

	if _, err := fmt.Fprint(w, "▼\n"); err != nil {
		return err
	}

	return n.FprintRec(w, pathContext{Is4: is4}, "")
}

// dump the table structure and all the nodes to w.
func (f *FastACL) dump(w io.Writer) {
	if f.size4 > 0 {
		stats := f.root4.StatsRec()
		fmt.Fprintln(w)
		fmt.Fprintf(w, "### IPv4: size(%d), subnodes(%d), prefixes(%d), fringes(%d), leaves(%d)",
			f.size4, stats.SubNodes, stats.Prefixes, stats.Fringes, stats.Leaves)

		f.root4.DumpRec(w, stridePath{}, 0, true)
	}

	if f.size6 > 0 {
		stats := f.root6.StatsRec()
		fmt.Fprintln(w)
		fmt.Fprintf(w, "### IPv6: size(%d), subnodes(%d), prefixes(%d), fringes(%d), leaves(%d)",
			f.size6, stats.SubNodes, stats.Prefixes, stats.Fringes, stats.Leaves)

		f.root6.DumpRec(w, stridePath{}, 0, false)
	}
}

// dumpString is just a wrapper for dump.
func (f *FastACL) dumpString() string {
	w := new(strings.Builder)
	f.dump(w)

	return w.String()
}
