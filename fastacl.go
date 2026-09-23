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
	"github.com/gaissmai/bart/internal/lpm"
	"github.com/gaissmai/bart/internal/nodes"
)

type FastACL struct {
	// used by -copylocks checker from `go vet`.
	_ [0]sync.Mutex

	root4 nodes.FastACLNode
	root6 nodes.FastACLNode

	// the number of prefixes in the routing table
	size4 int
	size6 int
}

// rootNodeByVersion, root node getter for ip version.
func (f *FastACL) rootNodeByVersion(is4 bool) *nodes.FastACLNode {
	if is4 {
		return &f.root4
	}
	return &f.root6
}

func (f *FastACL) sizeUpdate(is4 bool, delta int) {
	if is4 {
		f.size4 += delta
		return
	}
	f.size6 += delta
}

// Contains reports whether any stored prefix covers the given IP address.
// It returns false for invalid IP addresses.
//
// This method performs longest-prefix matching and returns true if any prefix
// in the routing table contains the IP address.
//
// Any IPv6 zone identifier is stripped and has no effect on the lookup result.
func (f *FastACL) Contains(ip netip.Addr) bool {
	// speed is top priority: no explicit test for ip.IsValid
	// if ip is invalid, AsSlice() returns nil, Contains returns false.
	is4 := ip.Is4()

	n := f.rootNodeByVersion(is4)

	for _, octet := range ip.AsSlice() {
		// for contains, any lpm match is good enough, no lpm backtracking needed
		if n.PrefixCount() != 0 && n.Contains(art.OctetToIdx(octet)) {
			return true
		}

		// for contains, any matching fringe is good enough
		if n.Fringes.Test(octet) {
			return true
		}

		// stop traversing?
		if !n.Children.Test(octet) {
			return false
		}
		kid := n.MustGetChild(octet)

		// kid is leaf
		if leaf, ok := kid.(*nodes.CIDRLeaf); ok {
			// Strip IPv6 zone before netip.Prefix.Contains to prevent false returns.
			if !is4 {
				// but netip.Addr.withoutZone  is not exported :-(
				// and netip.Addr.WithZone("") is not inlinable, so we have to resort to this clever trick:
				// https://github.com/gaissmai/bart/pull/418#issuecomment-5735613506
				ip = netip.PrefixFrom(ip, 0).Addr()
			}
			return leaf.Prefix.Contains(ip)
		}

		// kid is node!
		n = kid.(*nodes.FastACLNode)
	}

	return false
}

// LookupPrefix performs a longest prefix match lookup for any address within
// the given prefix. It finds the most specific routing table entry that would
// match any address in the provided prefix range.
//
// This is functionally identical to LookupPrefixLPM but returns only the
// associated value, not the matching prefix itself.
//
// Returns the value and true if a matching prefix is found.
// Returns zero value and false if no match exists.
func (f *FastACL) LookupPrefix(pfx netip.Prefix) (ok bool) {
	_, ok = f.lookupPrefixLPM(pfx, false)
	return ok
}

// LookupPrefixLPM performs a longest prefix match lookup for any address within
// the given prefix. It finds the most specific routing table entry that would
// match any address in the provided prefix range.
//
// This is functionally identical to LookupPrefix but additionally returns the
// matching LPM prefix itself along with the value.
//
// This method is slower than LookupPrefix and should only be used if the
// matching lpm entry is also required for other reasons.
//
// Returns the matching prefix, its associated value, and true if found.
// Returns zero values and false if no match exists.
func (f *FastACL) LookupPrefixLPM(pfx netip.Prefix) (lpmPfx netip.Prefix, ok bool) {
	return f.lookupPrefixLPM(pfx, true)
}

func (f *FastACL) lookupPrefixLPM(pfx netip.Prefix, withLPM bool) (lpmPfx netip.Prefix, ok bool) {
	panic("TODO")

	if !pfx.IsValid() {
		return lpmPfx, ok
	}

	// canonicalize the prefix
	pfx = pfx.Masked()

	ip := pfx.Addr()
	pfxLen := pfx.Bits()
	is4 := ip.Is4()
	octets := ip.AsSlice()
	strideCount, modBits := nodes.DivMod8(pfxLen)

	n := f.rootNodeByVersion(is4)

	// record path to leaf node
	stack := [nodes.MaxTreeDepth]*nodes.FastACLNode{}

	var depth int
	var octet byte

LOOP:
	// find the last node on the octets path in the trie,
	for depth, octet = range octets {
		depth &= nodes.DepthMask // BCE

		// stepped one past the last stride of interest; back up to last and break
		if depth > strideCount {
			depth--
			break
		}
		// push current node on stack
		stack[depth] = n

		// go down in tight loop to leaf node
		if !n.Children.Test(octet) {
			break LOOP
		}
		kid := n.MustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *nodes.FastACLNode:
			n = kid
			continue LOOP // descend down to next trie level

		case *nodes.CIDRLeaf:
			// reached a path compressed prefix, stop traversing
			if kid.Prefix.Bits() > pfxLen || !kid.Prefix.Contains(ip) {
				break LOOP
			}
			return kid.Prefix, true

		case *nodes.FringeLeaf:
			// the bits of the fringe are defined by the depth
			// maybe the LPM isn't needed, saves some cycles
			fringeBits := (depth + 1) << 3
			if fringeBits > pfxLen {
				break LOOP
			}

			// the LPM isn't needed, saves some cycles
			if !withLPM {
				return netip.Prefix{}, true
			}

			// get the LPM prefix back from ip and depth
			// it's a fringe, bits are always /8, /16, /24, ...
			fringePfx, _ := ip.Prefix((depth + 1) << 3)
			return fringePfx, true
		}
	}

	// start backtracking, unwind the stack
	for ; depth >= 0; depth-- {
		depth &= nodes.DepthMask // BCE

		n = stack[depth]

		// longest prefix match, skip if node has no prefixes
		if n.PrefixCount() == 0 {
			continue
		}

		var idx uint8
		octet = octets[depth]

		// only the final stride may have a different prefix len
		// all others are just host routes
		if depth == strideCount {
			idx = art.PfxToIdx(octet, modBits)
		} else {
			idx = art.OctetToIdx(octet)
		}

		// manually inlined: lookupIdx(idx)
		var topIdx uint8
		if topIdx, ok = n.Prefixes.AndTop(&lpm.LookupTbl[idx]); ok {
			// called from LookupPrefix
			if !withLPM {
				return netip.Prefix{}, ok
			}

			// called from LookupPrefixLPM

			// get the bits from depth and top idx
			pfxBits := int(art.PfxBits(depth, topIdx))

			// calculate the lpmPfx from incoming ip and new mask
			// netip.Addr.Prefix canonicalizes. Invariant: art.PfxBits(depth, topIdx)
			// yields a valid mask (v4: 0..32, v6: 0..128), so error is impossible.
			lpmPfx, _ = ip.Prefix(pfxBits)
			return lpmPfx, ok
		}
	}

	return lpmPfx, ok
}

// Insert adds or updates a prefix-value pair in the routing table.
// If the prefix already exists, its value is updated; otherwise a new entry is created.
// Invalid prefixes are silently ignored.
//
// The prefix is automatically canonicalized using pfx.Masked() to ensure
// consistent behavior regardless of host bits in the input.
func (f *FastACL) Insert(pfx netip.Prefix) {
	if !pfx.IsValid() {
		return
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	is4 := pfx.Addr().Is4()
	n := f.rootNodeByVersion(is4)

	if exists := n.Insert(pfx, 0); exists {
		return
	}

	// true insert, update size
	f.sizeUpdate(is4, 1)
}

// Delete removes the exact prefix pfx from the table in-place.
//
// This is an exact-match operation (no LPM). If pfx exists, the entry is
// removed. If pfx does not exist or pfx is invalid, the table is left unchanged.
//
// The prefix is canonicalized (Masked) before lookup.
func (f *FastACL) Delete(pfx netip.Prefix) {
	if !pfx.IsValid() {
		return
	}

	// canonicalize prefix
	pfx = pfx.Masked()
	is4 := pfx.Addr().Is4()

	n := f.rootNodeByVersion(is4)
	if exists := n.Delete(pfx); exists {
		f.sizeUpdate(is4, -1)
	}
}

// Get performs an exact-prefix lookup and returns whether the exact
// prefix exists. The prefix is canonicalized (Masked) before lookup.
//
// This is an exact-match operation (no LPM). The prefix must match exactly
// in both address and prefix length to be found. If pfx exists, the
// associated value (zero value for Lite) and found=true is returned.
// If pfx does not exist or pfx is invalid, the zero value for V and
// exists=false is returned.
//
// For longest-prefix-match (LPM) lookups, use Contains(ip), Lookup(ip),
// LookupPrefix(pfx) or LookupPrefixLPM(pfx) instead.
func (f *FastACL) Get(pfx netip.Prefix) (exists bool) {
	if !pfx.IsValid() {
		return exists
	}
	// canonicalize prefix
	pfx = pfx.Masked()

	is4 := pfx.Addr().Is4()
	n := f.rootNodeByVersion(is4)

	return n.Get(pfx)
}

// Supernets returns an iterator over all supernet routes that cover the given prefix pfx.
//
// The traversal searches both exact-length and shorter (less specific) prefixes that
// include pfx. Starting from the most specific position in the trie,
// it walks upward through parent nodes and yields any matching entries found at each level.
//
// The iteration order is reverse-CIDR: from longest prefix match (LPM) towards
// least-specific routes.
//
// This can be used to enumerate all covering supernet routes in routing-based
// policy engines, diagnostics tools, or fallback resolution logic.
//
// Example:
//
//	for supernet, val := range table.Supernets(netip.MustParsePrefix("192.0.2.128/25")) {
//	    fmt.Println("Covered by:", supernet, "->", val)
//	}
//
// The iteration can be stopped early by breaking from the range loop.
// Returns an empty iterator if the prefix is invalid.
func (f *FastACL) Supernets(pfx netip.Prefix) iter.Seq[netip.Prefix] {
	return func(yield func(netip.Prefix) bool) {
		if !pfx.IsValid() {
			return
		}

		// canonicalize the prefix
		pfx = pfx.Masked()

		is4 := pfx.Addr().Is4()
		n := f.rootNodeByVersion(is4)

		n.Supernets(pfx, yield)
	}
}

// Subnets returns an iterator over all subnets of the given prefix
// in natural CIDR sort order. This includes prefixes of the same length
// (exact match) and longer (more specific) prefixes that are contained
// within the given prefix.
//
// Example:
//
//	for sub, val := range table.Subnets(netip.MustParsePrefix("10.0.0.0/8")) {
//	    fmt.Println("Covered:", sub, "->", val)
//	}
//
// The iteration can be stopped early by breaking from the range loop.
// Returns an empty iterator if the prefix is invalid.
func (f *FastACL) Subnets(pfx netip.Prefix) iter.Seq[netip.Prefix] {
	return func(yield func(netip.Prefix) bool) {
		if !pfx.IsValid() {
			return
		}

		pfx = pfx.Masked()
		is4 := pfx.Addr().Is4()

		n := f.rootNodeByVersion(is4)
		n.Subnets(pfx, yield)
	}
}

// Clone returns a deep copy of the ACL.
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
	mod4 := f.root4.AggregateRec(nodes.StridePath{}, 0, true)
	mod6 := f.root6.AggregateRec(nodes.StridePath{}, 0, false)

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

// All returns an iterator over all prefix–value pairs in the table.
//
// The iteration order is unspecified and may vary between calls; for a stable order,
// use [liteTable.AllSorted].
//
// IMPORTANT: Modifying the table during iteration is not allowed,
// as this would interfere with the internal traversal and may corrupt or
// prematurely terminate the iteration.
func (f *FastACL) All() iter.Seq[netip.Prefix] {
	return func(yield func(netip.Prefix) bool) {
		_ = f.root4.AllRec(stridePath{}, 0, true, yield) && f.root6.AllRec(stridePath{}, 0, false, yield)
	}
}

// All4 is like [liteTable.All] but only for the v4 routing table.
func (f *FastACL) All4() iter.Seq[netip.Prefix] {
	return func(yield func(netip.Prefix) bool) {
		_ = f.root4.AllRec(stridePath{}, 0, true, yield)
	}
}

// All6 is like [liteTable.All] but only for the v6 routing table.
func (f *FastACL) All6() iter.Seq[netip.Prefix] {
	return func(yield func(netip.Prefix) bool) {
		_ = f.root6.AllRec(stridePath{}, 0, false, yield)
	}
}

// AllSorted is like [liteTable.All] but the iteration is ordered in canonical
// CIDR prefix sort order.
func (f *FastACL) AllSorted() iter.Seq[netip.Prefix] {
	return func(yield func(netip.Prefix) bool) {
		_ = f.root4.AllRecSorted(stridePath{}, 0, true, yield) &&
			f.root6.AllRecSorted(stridePath{}, 0, false, yield)
	}
}

// AllSorted4 is like [liteTable.AllSorted] but only for the v4 routing table.
func (f *FastACL) AllSorted4() iter.Seq[netip.Prefix] {
	return func(yield func(netip.Prefix) bool) {
		_ = f.root4.AllRecSorted(stridePath{}, 0, true, yield)
	}
}

// AllSorted6 is like [liteTable.AllSorted] but only for the v6 routing table.
func (f *FastACL) AllSorted6() iter.Seq[netip.Prefix] {
	return func(yield func(netip.Prefix) bool) {
		_ = f.root6.AllRecSorted(stridePath{}, 0, false, yield)
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

	startParent := nodes.TrieItemACL{
		Node: nil,
		Idx:  0,
		Path: stridePath{},
		Is4:  is4,
	}

	return n.FprintRec(w, startParent, "")
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
