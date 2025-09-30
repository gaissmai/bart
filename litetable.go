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

// Lite follows the BART design but with no payload.
// It is ideal for simple IP ACLs (access-control-lists) with plain
// true/false results with the smallest memory consumption.
//
// The zero value is ready to use.
//
// A Lite table must not be copied by value; always pass by pointer.
//
// Performance note: Do not pass IPv4-in-IPv6 addresses (e.g., ::ffff:192.0.2.1)
// as input. The methods do not perform automatic unmapping to avoid unnecessary
// overhead for the common case where native addresses are used.
// Users should unmap IPv4-in-IPv6 addresses to their native IPv4 form
// (e.g., 192.0.2.1) before calling these methods.
type Lite struct {
	liteTable[struct{}]
}

// BEGIN OF liteTable WRAPPER

// Get performs an exact-prefix lookup and returns whether the exact
// prefix exists. The prefix is canonicalized (Masked) before lookup.
//
// This is an exact-match operation (no LPM). The prefix must match exactly
// in both address and prefix length to be found.
// If pfx is valid and exists, true is returned, otherwise false.
//
// For longest-prefix-match (LPM) lookups, use Contains(ip), Lookup(ip),
// LookupPrefix(pfx) or LookupPrefixLPM(pfx) instead.
func (l *Lite) Get(pfx netip.Prefix) bool {
	_, ok := l.liteTable.Get(pfx)
	return ok
}

// Lookup performs a longest-prefix-match (LPM) for addr.
//
// Note: Lite stores no payload values, so this method is rarely useful.
// Prefer Contains(addr) to check whether any prefix matches the address.
// For exact prefix existence use Get(pfx). For prefix-based LPM use
// LookupPrefix or LookupPrefixLPM.
//
// Returns true if any prefix matches addr, otherwise false.
func (l *Lite) Lookup(ip netip.Addr) bool {
	return l.Contains(ip)
}

// LookupPrefix performs a longest prefix match lookup for any address within
// the given prefix.
//
// Returns true if a matching prefix is found, otherwise false.
func (l *Lite) LookupPrefix(pfx netip.Prefix) bool {
	_, _, ok := l.lookupPrefixLPM(pfx, false)
	return ok
}

// LookupPrefixLPM performs a longest prefix match lookup for any address within
// the given prefix. It finds the most specific routing table entry that would
// match any address in the provided prefix range.
//
// This is functionally identical to LookupPrefix but returns the
// matching prefix (lpmPfx) itself.
//
// This method is slower than LookupPrefix and should only be used if the
// matching lpm entry is also required for other reasons.
//
// Returns the matching prefix and true if found, otherwise the zero value and false.
func (l *Lite) LookupPrefixLPM(pfx netip.Prefix) (lpmPfx netip.Prefix, ok bool) {
	lpmPfx, _, ok = l.lookupPrefixLPM(pfx, true)
	return lpmPfx, ok
}

// Insert adds a prefix to the routing table.
// If the prefix already exists, it's a no-op; otherwise a new entry is created.
// Invalid prefixes are silently ignored.
//
// The prefix is automatically canonicalized using pfx.Masked() to ensure
// consistent behavior regardless of host bits in the input.
func (l *Lite) Insert(pfx netip.Prefix) {
	l.liteTable.Insert(pfx, struct{}{})
}

// InsertPersist is similar to Insert but the receiver isn't modified.
//
// All nodes touched during insert are cloned and a new Table is returned.
// This is not a full [Lite.Clone], all untouched nodes are still referenced
// from both Tables.
//
// This is orders of magnitude slower than Insert,
// typically taking μsec instead of nsec.
//
// The bulk table load could be done with [Lite.Insert] and then you can
// use [Lite.InsertPersist], [Lite.ModifyPersist] and [Lite.DeletePersist]
// for further lock-free ops.
func (l *Lite) InsertPersist(pfx netip.Prefix) *Lite {
	lp := l.liteTable.InsertPersist(pfx, struct{}{})
	//nolint:govet // copy of *lp is here by intention
	return &Lite{*lp}
}

// DeletePersist removes a prefix using copy-on-write semantics.
// It creates a new Lite instance with the prefix removed while leaving the
// original unchanged. This enables functional programming patterns and safe
// concurrent access.
//
// Returns a new Lite instance without the specified prefix and a boolean
// indicating whether the prefix was found and deleted. If the prefix doesn't
// exist, it returns a structurally identical copy and false.
//
// Due to cloning overhead, DeletePersist is significantly slower than Delete,
// typically taking μsec instead of nsec.
func (l *Lite) DeletePersist(pfx netip.Prefix) *Lite {
	lp := l.liteTable.DeletePersist(pfx)
	//nolint:govet // copy of *lp is here by intention
	return &Lite{*lp}
}

// Modify applies a modification callback to a prefix in-place.
// The callback function receives a boolean indicating whether the prefix exists.
// It should return a boolean indicating whether to delete the prefix.
// Since Lite is prefix-only, no value handling is needed.
//
//   - for cb(exists==true)  -> del=true,  delete prefix
//   - for cb(exists==false) -> del=false, insert prefix
//   - for cb(exists==true)  -> del=false, no-op
//
// Modify returns a boolean indicating whether a prefix was deleted during
// the operation.
func (l *Lite) Modify(pfx netip.Prefix, cb func(exists bool) (del bool)) {
	// Adapt the callback to work with liteTable's signature
	adaptedCb := func(_ struct{}, exists bool) (_ struct{}, del bool) {
		return struct{}{}, cb(exists)
	}

	l.liteTable.Modify(pfx, adaptedCb)
}

// Clone returns a copy of the routing table.
func (l *Lite) Clone() *Lite {
	return &Lite{*l.liteTable.Clone()}
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
	lp := l.liteTable.UnionPersist(&o.liteTable)
	//nolint:govet // copy of *lp is here by intention
	return &Lite{*lp}
}

// ModifyPersist applies a modification callback to a prefix using copy-on-write semantics.
// It creates a new Lite instance with the modification applied while leaving the original
// unchanged. This enables functional programming patterns and safe concurrent access.
//
// The callback function receives a boolean indicating whether the prefix exists.
// It should return a boolean indicating whether to delete the prefix.
// Since Lite is prefix-only, no value handling is needed.
//
// Returns a new Lite instance with the modification applied and a boolean
// indicating whether a prefix was deleted during the operation.
func (l *Lite) ModifyPersist(pfx netip.Prefix, cb func(exists bool) (del bool)) *Lite {
	// Adapt the callback to work with liteTable's signature
	wrappedFn := func(_ struct{}, exists bool) (_ struct{}, del bool) {
		return struct{}{}, cb(exists)
	}

	lp := l.liteTable.ModifyPersist(pfx, wrappedFn)
	//nolint:govet // copy of *lp is here by intention
	return &Lite{*lp}
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

// Subnets returns an iterator over all subnets of the given prefix
// in natural CIDR sort order. This includes prefixes of the same length
// (exact match) and longer (more specific) prefixes that are contained
// within the given prefix.
//
// Example:
//
//	for sub := range table.Subnets(netip.MustParsePrefix("10.0.0.0/8")) {
//	    fmt.Println("Covered:", sub)
//	}
//
// The iteration can be stopped early by breaking from the range loop.
// Returns an empty iterator if the prefix is invalid.
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

// Equal checks whether two tables are structurally and semantically equal.
// It ensures both trees (IPv4-based and IPv6-based) have the same sizes and
// recursively compares their root nodes.
func (l *Lite) Equal(o *Lite) bool {
	if o == nil || l.size4 != o.size4 || l.size6 != o.size6 {
		return false
	}
	return l.liteTable.Equal(&o.liteTable)
}

// END OF liteTable WRAPPER

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
// in the routing table contains the IP address, regardless of the associated value.
//
// It does not return the value nor the prefix of the matching item,
// but as a test against an allow-/deny-list it's often sufficient
// and even few nanoseconds faster than [Table.Lookup].
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

// Lookup performs a longest-prefix-match (LPM) for addr.
//
// Note: Lite stores no payload values. The returned value is always the zero
// value (struct{}{}), so this method is rarely useful. Prefer Contains(addr)
// to check whether any prefix matches the address. For exact prefix existence
// use Get(pfx). For prefix-based LPM use LookupPrefix or LookupPrefixLPM.
//
// This method exists to satisfy shared interfaces and code generation; its
// behavior is equivalent to Contains(addr) plus returning a meaningless value.
//
// Returns (struct{}{}, true) if any prefix matches addr, otherwise (struct{}{}, false).
func (l *liteTable[V]) Lookup(ip netip.Addr) (_ V, ok bool) {
	var zero V
	return zero, l.Contains(ip)
}

// LookupPrefix performs a longest prefix match lookup for any address within
// the given prefix. It finds the most specific routing table entry that would
// match any address in the provided prefix range.
//
// Returns the value and true if a matching prefix is found.
// Returns zero value and false if no match exists.
func (l *liteTable[V]) LookupPrefix(pfx netip.Prefix) (_ V, ok bool) {
	_, _, ok = l.lookupPrefixLPM(pfx, false)
	return
}

// LookupPrefixLPM performs a longest prefix match lookup for any address within
// the given prefix. It finds the most specific routing table entry that would
// match any address in the provided prefix range.
//
// This is functionally identical to LookupPrefix but additionally returns the
// matching prefix (lpmPfx) itself.
//
// This method is slower than LookupPrefix and should only be used if the
// matching lpm entry is also required for other reasons.
//
// Returns the matching prefix, the zero value (struct{}{}) and true if found.
// Returns zero values and false if no match exists.
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
		if n.prefixCount() == 0 {
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
			// netip.Addr.Prefix canonicalizes. Invariant: art.PfxBits(depth, topIdx)
			// yields a valid mask (v4: 0..32, v6: 0..128), so error is impossible.
			lpmPfx, _ = ip.Prefix(pfxBits)
			return lpmPfx, zero, ok
		}
	}

	return
}
