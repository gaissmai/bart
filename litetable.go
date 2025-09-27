// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"iter"
	"net/netip"
	"sync"

	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/lpm"
)

// Lite is a prefix-only table that embeds a private liteTable[struct{}].
//
// It is intentionally not nil-receiver safe: calling methods on a nil *Lite
// will panic by design.
type Lite struct {
	liteTable[struct{}]
}

// Insert adds a pfx to the tree.
// If pfx is already present in the tree, it's a no-op.
func (l *Lite) Insert(pfx netip.Prefix) {
	l.liteTable.Insert(pfx, struct{}{})
}

// dropSeq2 converts a Seq2[netip.Prefix, V] into a Seq[netip.Prefix] by discarding the value.
func dropSeq2[V any](seq2 iter.Seq2[netip.Prefix, V]) iter.Seq[netip.Prefix] {
	return func(yield func(netip.Prefix) bool) {
		seq2(func(p netip.Prefix, _ V) bool {
			return yield(p)
		})
	}
}

// All returns an iterator over all prefixes in the table.
//
// The entries from both IPv4 and IPv6 subtries are yielded using an internal recursive traversal.
// The iteration order is unspecified and may vary between calls; for a stable order, use AllSorted.
//
// You can use All directly in a for-range loop without providing a yield function.
// The Go compiler automatically synthesizes the yield callback for you:
//
//	for prefix := range t.All() {
//	    fmt.Println(prefix)
//	}
//
// Under the hood, the loop body is passed as a yield function to the iterator.
// If you break or return from the loop, iteration stops early as expected.
//
// IMPORTANT: Deleting entries during iteration is not allowed,
// as this would interfere with the internal traversal and may corrupt or
// prematurely terminate the iteration.
//
// If mutation of the table during traversal is required,
// use [Lite.WalkPersist] instead.
func (l *Lite) All() iter.Seq[netip.Prefix] {
	return dropSeq2(l.liteTable.All())
}

// All4 is like [Lite.All] but only for the v4 routing table.
func (l *Lite) All4() iter.Seq[netip.Prefix] {
	return dropSeq2(l.liteTable.All4())
}

// All6 is like [Lite.All] but only for the v6 routing table.
func (l *Lite) All6() iter.Seq[netip.Prefix] {
	return dropSeq2(l.liteTable.All6())
}

// AllSorted returns an iterator over all prefixes in the table,
// ordered in canonical CIDR prefix sort order.
//
// This can be used directly with a for-range loop; the Go compiler provides the yield function implicitly.
//
//	for prefix := range t.AllSorted() {
//	    fmt.Println(prefix)
//	}
//
// The traversal is stable and predictable across calls.
// Iteration stops early if you break out of the loop.
//
// IMPORTANT: Deleting entries during iteration is not allowed,
// as this would interfere with the internal traversal and may corrupt or
// prematurely terminate the iteration.
//
// If mutation of the table during traversal is required,
// use [Lite.WalkPersist] instead.
func (l *Lite) AllSorted() iter.Seq[netip.Prefix] {
	return dropSeq2(l.liteTable.AllSorted())
}

// AllSorted4 is like [Lite.AllSorted] but only for the v4 routing table.
func (l *Lite) AllSorted4() iter.Seq[netip.Prefix] {
	return dropSeq2(l.liteTable.AllSorted4())
}

// AllSorted6 is like [Lite.AllSorted] but only for the v6 routing table.
func (l *Lite) AllSorted6() iter.Seq[netip.Prefix] {
	return dropSeq2(l.liteTable.AllSorted6())
}

// Subnets returns an iterator over all prefix–value pairs in the routing table
// that are fully contained within the given prefix pfx.
//
// Entries are returned in CIDR sort order.
//
// Example:
//
//	for sub := range table.Subnets(netip.MustParsePrefix("10.0.0.0/8")) {
//	    fmt.Println("Covered:", sub)
//	}
func (l *Lite) Subnets(pfx netip.Prefix) iter.Seq[netip.Prefix] {
	return dropSeq2(l.liteTable.Subnets(pfx))
}

// Supernets returns an iterator over all supernet routes that cover the given prefix pfx.
//
// The traversal searches both exact-length and shorter (less specific) prefixes that
// overlap or include pfx. Starting from the most specific position in the trie,
// it walks upward through parent nodes and yields any matching entries found at each level.
//
// The iteration order is reverse-CIDR: from longest prefix match (LPM) towards
// least-specific routes.
//
// The search is protocol-specific (IPv4 or IPv6) and stops immediately if the yield
// function returns false. If pfx is invalid, the function silently returns.
//
// This can be used to enumerate all covering supernet routes in routing-based
// policy engines, diagnostics tools, or fallback resolution logic.
//
// Example:
//
//	for supernet := range table.Supernets(netip.MustParsePrefix("192.0.2.128/25")) {
//	    fmt.Println("Matched covering route:", supernet)
//	}
func (l *Lite) Supernets(pfx netip.Prefix) iter.Seq[netip.Prefix] {
	return dropSeq2(l.liteTable.Supernets(pfx))
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
//
// It is intentionally not nil-receiver safe: calling with a nil *Lite
// will panic by design.
func (l *Lite) Overlaps(o *Lite) bool {
	if o == nil {
		return false
	}
	return l.liteTable.Overlaps(&o.liteTable)
}

// Union merges another routing table into the receiver table, modifying it in-place.
//
// All prefixes from the other table (o) are inserted into the receiver.
func (l *Lite) Union(o *Lite) {
	if o == nil {
		return
	}
	l.liteTable.Union(&o.liteTable)
}

// UnionPersist is similar to [Union] but the receiver isn't modified.
//
// All nodes touched during union are cloned and a new *Lite is returned.
// If o is nil or empty, no nodes are touched and the receiver may be
// returned unchanged.
func (l *Lite) UnionPersist(o *Lite) *Lite {
	if o == nil || (o.size4 == 0 && o.size6 == 0) {
		return l
	}
	tbl := l.liteTable.UnionPersist(&o.liteTable)
	//nolint:govet // copy of *tbl is here by intention
	return &Lite{*tbl}
}

// Equal checks whether two tables are structurally and semantically equal.
// It ensures both trees (IPv4-based and IPv6-based) have the same sizes and
// recursively compares their root nodes.
func (l *Lite) Equal(o *Lite) bool {
	if o == nil || l.size4 != o.size4 || l.size6 != o.size6 {
		return false
	}
	return l.liteTable.Equal(&o.liteTable)
}

// Clone returns a copy of the routing table.
func (l *Lite) Clone() *Lite {
	return &Lite{*l.liteTable.Clone()}
}

// liteTable follows the BART design but with no payload.
// It is ideal for simple IP ACLs (access-control-lists) with plain
// true/false results with the smallest memory consumption.
//
// Performance note: Do not pass IPv4-in-IPv6 addresses (e.g., ::ffff:192.0.2.1)
// as input. The methods do not perform automatic unmapping to avoid unnecessary
// overhead for the common case where native addresses are used.
// Users should unmap IPv4-in-IPv6 addresses to their native IPv4 form
// (e.g., 192.0.2.1) before calling these methods.
type liteTable[V any] struct {
	// used by -copylocks checker from `go vet`.
	_ [0]sync.Mutex

	root4 liteNode[V]
	root6 liteNode[V]

	// the number of prefixes in the routing table
	size4 int
	size6 int
}

// rootNodeByVersion, root node getter for ip version.
func (l *liteTable[V]) rootNodeByVersion(is4 bool) *liteNode[V] {
	if is4 {
		return &l.root4
	}
	return &l.root6
}

// Contains reports whether any stored prefix covers the given IP address.
// Returns false for invalid IP addresses.
//
// This performs longest-prefix matching and returns true if any prefix
// in the routing table contains the IP address.
func (l *liteTable[V]) Contains(ip netip.Addr) bool {
	// speed is top priority: no explicit test for ip.IsValid
	// if ip is invalid, AsSlice() returns nil, Contains returns false.
	is4 := ip.Is4()
	n := l.rootNodeByVersion(is4)

	for _, octet := range ip.AsSlice() {
		// for contains, any lpm match is good enough, no backtracking needed
		if n.prefixes.count != 0 && n.contains(art.OctetToIdx(octet)) {
			return true
		}

		// stop traversing?
		if !n.children.Test(octet) {
			return false
		}
		kid := n.mustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *liteNode[V]:
			n = kid // descend down to next trie level

		case *fringeNode[V]:
			// fringe is the default-route for all possible octets below
			return true

		case *leafNode[V]:
			return kid.prefix.Contains(ip)

		default:
			panic("logic error, wrong node type")
		}
	}

	return false
}

// Lookup, only for interface satisfaction.
//
//nolint:unparam
func (l *liteTable[V]) Lookup(ip netip.Addr) (_ V, ok bool) {
	var zero V
	return zero, l.Contains(ip)
}

// LookupPrefix does a route lookup (longest prefix match) for pfx and
// returns true, or false if no route matched.
//
//nolint:unparam
func (l *liteTable[V]) LookupPrefix(pfx netip.Prefix) (_ V, ok bool) {
	_, _, ok = l.lookupPrefixLPM(pfx, false)
	return
}

// LookupPrefixLPM is similar to [Lite.LookupPrefix],
// but it returns the lpm prefix in addition to value,ok.
//
// This method is about 20-30% slower than LookupPrefix and should only
// be used if the matching lpm entry is also required for other reasons.
//
// If LookupPrefixLPM is to be used for IP address lookups,
// they must be converted to /32 or /128 prefixes.
//
//nolint:unparam
func (l *liteTable[V]) LookupPrefixLPM(pfx netip.Prefix) (lpmPfx netip.Prefix, _ V, ok bool) {
	return l.lookupPrefixLPM(pfx, true)
}

//nolint:unparam
func (l *liteTable[V]) lookupPrefixLPM(pfx netip.Prefix, withLPM bool) (lpmPfx netip.Prefix, _ V, ok bool) {
	var zero V

	if !pfx.IsValid() {
		return
	}

	// canonicalize the prefix
	pfx = pfx.Masked()

	ip := pfx.Addr()
	bits := pfx.Bits()
	is4 := ip.Is4()
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)

	n := l.rootNodeByVersion(is4)

	// record path to leaf node
	stack := [maxTreeDepth]*liteNode[V]{}

	var depth int
	var octet byte

LOOP:
	// find the last node on the octets path in the trie,
	for depth, octet = range octets {
		depth = depth & depthMask // BCE

		// stepped one past the last stride of interest; back up to last and break
		if depth > lastOctetPlusOne {
			depth--
			break
		}
		// push current node on stack
		stack[depth] = n

		// go down in tight loop to leaf node
		if !n.children.Test(octet) {
			break LOOP
		}
		kid := n.mustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *liteNode[V]:
			n = kid
			continue LOOP // descend down to next trie level

		case *leafNode[V]:
			// reached a path compressed prefix, stop traversing
			if kid.prefix.Bits() > bits || !kid.prefix.Contains(ip) {
				break LOOP
			}
			return kid.prefix, zero, true

		case *fringeNode[V]:
			// the bits of the fringe are defined by the depth
			// maybe the LPM isn't needed, saves some cycles
			fringeBits := (depth + 1) << 3
			if fringeBits > bits {
				break LOOP
			}

			// the LPM isn't needed, saves some cycles
			if !withLPM {
				return netip.Prefix{}, zero, true
			}

			// sic, get the LPM prefix back, it costs some cycles!
			fringePfx := cidrForFringe(octets, depth, is4, octet)
			return fringePfx, zero, true

		default:
			panic("logic error, wrong node type")
		}
	}

	// start backtracking, unwind the stack
	for ; depth >= 0; depth-- {
		depth = depth & depthMask // BCE

		n = stack[depth]

		// longest prefix match, skip if node has no prefixes
		if n.prefixes.count == 0 {
			continue
		}

		// only the lastOctet may have a different prefix len
		// all others are just host routes
		var idx uint8
		octet = octets[depth]
		// Last “octet” from prefix, update/insert prefix into node.
		// Note: For /32 and /128, depth never reaches lastOctetPlusOne (4 or 16),
		// so those are handled below via the fringe/leaf path.
		if depth == lastOctetPlusOne {
			idx = art.PfxToIdx(octet, lastBits)
		} else {
			idx = art.OctetToIdx(octet)
		}

		// manually inlined: lookupIdx(idx)
		if topIdx, ok := n.prefixes.IntersectionTop(&lpm.LookupTbl[idx]); ok {
			// called from LookupPrefix
			if !withLPM {
				return netip.Prefix{}, zero, ok
			}

			// called from LookupPrefixLPM

			// get the bits from depth and top idx
			pfxBits := int(art.PfxBits(depth, topIdx))

			// calculate the lpmPfx from incoming ip and new mask
			lpmPfx, _ = ip.Prefix(pfxBits)
			return lpmPfx, zero, ok
		}
	}

	return
}
