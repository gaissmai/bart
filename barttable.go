// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Package bart provides a high-performance Balanced Routing Table (BART).
//
// BART is balanced in terms of memory usage and lookup time
// for longest-prefix match (LPM) queries on IPv4 and IPv6 addresses.
//
// Internally, BART is implemented as a multibit trie with a fixed stride of 8 bits.
// Each level node uses a fast mapping function (adapted from D. E. Knuth's ART algorithm)
// to arrange all 256 possible prefixes in a complete binary tree structure.
//
// Instead of allocating full arrays, BART uses popcount-compressed sparse arrays
// and aggressive path compression. This results in up to 100x less memory usage
// than ART, while maintaining or even improving lookup speed.
//
// Lookup operations are entirely bit-vector based and rely on precomputed
// lookup tables. Because the data fits within 256-bit blocks, it allows
// for extremely efficient, cacheline-aligned access and is accelerated by
// CPU instructions such as POPCNT, LZCNT, and TZCNT.
//
// The fixed 256-bit representation (4x uint64) permits loop unrolling in hot paths,
// ensuring predictable and fast performance even under high routing load.
package bart

import (
	"net/netip"
	"sync"

	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/lpm"
)

// Table represents a thread-safe IPv4 and IPv6 routing table with payload V.
//
// The zero value is ready to use.
//
// The Table is safe for concurrent reads, but concurrent reads and writes
// must be externally synchronized. Mutation via Insert/Delete requires locks,
// or alternatively, use ...Persist methods which return a modified copy
// without altering the original table (copy-on-write).
//
// A Table must not be copied by value; always pass by pointer.
//
// Performance note: Do not pass IPv4-in-IPv6 addresses (e.g., ::ffff:192.0.2.1)
// as input. The methods do not perform automatic unmapping to avoid unnecessary
// overhead for the common case where native addresses are used.
// Users should unmap IPv4-in-IPv6 addresses to their native IPv4 form
// (e.g., 192.0.2.1) before calling these methods.
type Table[V any] struct {
	// used by -copylocks checker from `go vet`.
	_ [0]sync.Mutex

	// the root nodes, implemented as popcount compressed multibit tries
	root4 bartNode[V]
	root6 bartNode[V]

	// the number of prefixes in the routing table
	size4 int
	size6 int
}

// rootNodeByVersion, root node getter for ip version.
func (t *Table[V]) rootNodeByVersion(is4 bool) *bartNode[V] {
	if is4 {
		return &t.root4
	}
	return &t.root6
}

// lastOctetPlusOneAndLastBits returns the count of full 8‑bit strides (bits/8)
// and the leftover bits in the final stride (bits%8) for pfx.
//
// lastOctetPlusOne is the count of full 8‑bit strides (bits/8).
// lastBits is the remaining bit count in the final stride (bits%8),
//
// ATTENTION: Split the IP prefixes at 8bit borders, count from 0.
//
//	/7, /15, /23, /31, ..., /127
//
//	BitPos: [0-7],[8-15],[16-23],[24-31],[32]
//	BitPos: [0-7],[8-15],[16-23],[24-31],[32-39],[40-47],[48-55],[56-63],...,[120-127],[128]
//
//	0.0.0.0/0      => lastOctetPlusOne:  0, lastBits: 0 (default route)
//	0.0.0.0/7      => lastOctetPlusOne:  0, lastBits: 7
//	0.0.0.0/8      => lastOctetPlusOne:  1, lastBits: 0 (possible fringe)
//	10.0.0.0/8     => lastOctetPlusOne:  1, lastBits: 0 (possible fringe)
//	10.0.0.0/22    => lastOctetPlusOne:  2, lastBits: 6
//	10.0.0.0/29    => lastOctetPlusOne:  3, lastBits: 5
//	10.0.0.0/32    => lastOctetPlusOne:  4, lastBits: 0 (possible fringe)
//
//	::/0           => lastOctetPlusOne:  0, lastBits: 0 (default route)
//	::1/128        => lastOctetPlusOne: 16, lastBits: 0 (possible fringe)
//	2001:db8::/42  => lastOctetPlusOne:  5, lastBits: 2
//	2001:db8::/56  => lastOctetPlusOne:  7, lastBits: 0 (possible fringe)
//
//	/32 and /128 prefixes are special, they never form a new node,
//	At the end of the trie (IPv4: depth 4, IPv6: depth 16) they are always
//	inserted as a path‑compressed fringe.
//
// We are not splitting at /8, /16, ..., because this would mean that the
// first node would have 512 prefixes, 9 bits from [0-8]. All remaining nodes
// would then only have 8 bits from [9-16], [17-24], [25..32], ...
// but the algorithm would then require a variable length bitset.
//
// If you can commit to a fixed size of [4]uint64, then the algorithm is
// much faster due to modern CPUs.
//
// Perhaps a future Go version that supports SIMD instructions for the [4]uint64 vectors
// will make the algorithm even faster on suitable hardware.
func lastOctetPlusOneAndLastBits(pfx netip.Prefix) (lastOctetPlusOne int, lastBits uint8) {
	// lastOctetPlusOne:  range from 0..4 or 0..16 !ATTENTION: not 0..3 or 0..15
	// lastBits:          range from 0..7
	bits := pfx.Bits()

	//nolint:gosec  // G115: narrowing conversion is safe here (bits in [0..128])
	return bits >> 3, uint8(bits & 7)
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
func (t *Table[V]) Contains(ip netip.Addr) bool {
	// speed is top priority: no explicit test for ip.IsValid
	// if ip is invalid, AsSlice() returns nil, Contains returns false.
	is4 := ip.Is4()
	n := t.rootNodeByVersion(is4)

	for _, octet := range ip.AsSlice() {
		// for contains, any lpm match is good enough, no backtracking needed
		if n.prefixCount() != 0 && n.contains(art.OctetToIdx(octet)) {
			return true
		}

		// stop traversing?
		if !n.children.Test(octet) {
			return false
		}
		kid := n.mustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *bartNode[V]:
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

// Lookup performs a longest prefix match (LPM) lookup for the given address.
// It finds the most specific (longest) prefix in the routing table that
// contains the given address and returns its associated value.
//
// This is the fundamental operation for IP routing decisions, finding the
// best matching route for a destination address.
//
// Returns the associated value and true if a matching prefix is found.
// Returns zero value and false if no prefix contains the address.
func (t *Table[V]) Lookup(ip netip.Addr) (val V, ok bool) {
	if !ip.IsValid() {
		return val, ok
	}

	is4 := ip.Is4()
	octets := ip.AsSlice()

	n := t.rootNodeByVersion(is4)

	// stack of the traversed nodes for fast backtracking, if needed
	stack := [maxTreeDepth]*bartNode[V]{}

	// run variable, used after for loop
	var depth int
	var octet byte

LOOP:
	// find leaf node
	for depth, octet = range octets {
		depth = depth & depthMask // BCE, Lookup must be fast

		// push current node on stack for fast backtracking
		stack[depth] = n

		// go down in tight loop to last octet
		if !n.children.Test(octet) {
			// no more nodes below octet
			break LOOP
		}
		kid := n.mustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *bartNode[V]:
			n = kid
			continue LOOP // descend down to next trie level

		case *fringeNode[V]:
			// fringe is the default-route for all possible nodes below
			return kid.value, true

		case *leafNode[V]:
			if kid.prefix.Contains(ip) {
				return kid.value, true
			}
			// reached a path compressed prefix, stop traversing
			break LOOP

		default:
			panic("logic error, wrong node type")
		}
	}

	// start backtracking, unwind the stack, bounds check eliminated
	for ; depth >= 0; depth-- {
		depth = depth & depthMask // BCE

		n = stack[depth]

		// longest prefix match, skip if node has no prefixes
		if n.prefixCount() != 0 {
			idx := art.OctetToIdx(octets[depth])
			// lookupIdx() manually inlined
			if lpmIdx, ok2 := n.prefixes.IntersectionTop(&lpm.LookupTbl[idx]); ok2 {
				return n.mustGetPrefix(lpmIdx), ok2
			}
		}
	}

	return val, ok
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
func (t *Table[V]) LookupPrefix(pfx netip.Prefix) (val V, ok bool) {
	_, val, ok = t.lookupPrefixLPM(pfx, false)
	return val, ok
}

// LookupPrefixLPM performs a longest prefix match lookup for any address within
// the given prefix. It finds the most specific routing table entry that would
// match any address in the provided prefix range.
//
// This is functionally identical to LookupPrefix but additionally returns the
// matching prefix (lpmPfx) itself along with the value.
//
// This method is slower than LookupPrefix and should only be used if the
// matching lpm entry is also required for other reasons.
//
// Returns the matching prefix, its associated value, and true if found.
// Returns zero values and false if no match exists.
func (t *Table[V]) LookupPrefixLPM(pfx netip.Prefix) (lpmPfx netip.Prefix, val V, ok bool) {
	return t.lookupPrefixLPM(pfx, true)
}

func (t *Table[V]) lookupPrefixLPM(pfx netip.Prefix, withLPM bool) (lpmPfx netip.Prefix, val V, ok bool) {
	if !pfx.IsValid() {
		return lpmPfx, val, ok
	}

	// canonicalize the prefix
	pfx = pfx.Masked()

	ip := pfx.Addr()
	bits := pfx.Bits()
	is4 := ip.Is4()
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)

	n := t.rootNodeByVersion(is4)

	// record path to leaf node
	stack := [maxTreeDepth]*bartNode[V]{}

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
		case *bartNode[V]:
			n = kid
			continue LOOP // descend down to next trie level

		case *leafNode[V]:
			// reached a path compressed prefix, stop traversing
			if kid.prefix.Bits() > bits || !kid.prefix.Contains(ip) {
				break LOOP
			}
			return kid.prefix, kid.value, true

		case *fringeNode[V]:
			// the bits of the fringe are defined by the depth
			// maybe the LPM isn't needed, saves some cycles
			fringeBits := (depth + 1) << 3
			if fringeBits > bits {
				break LOOP
			}

			// the LPM isn't needed, saves some cycles
			if !withLPM {
				return netip.Prefix{}, kid.value, true
			}

			// sic, get the LPM prefix back, it costs some cycles!
			fringePfx := cidrForFringe(octets, depth, is4, octet)
			return fringePfx, kid.value, true

		default:
			panic("logic error, wrong node type")
		}
	}

	// start backtracking, unwind the stack
	for ; depth >= 0; depth-- {
		depth = depth & depthMask // BCE

		n = stack[depth]

		// longest prefix match, skip if node has no prefixes
		if n.prefixes.Len() == 0 {
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
		if topIdx, ok2 := n.prefixes.IntersectionTop(&lpm.LookupTbl[idx]); ok2 {
			val = n.mustGetPrefix(topIdx)

			// called from LookupPrefix
			if !withLPM {
				return netip.Prefix{}, val, ok2
			}

			// called from LookupPrefixLPM

			// get the bits from depth and top idx
			pfxBits := int(art.PfxBits(depth, topIdx))

			// calculate the lpmPfx from incoming ip and new mask
			// netip.Addr.Prefix already canonicalize the prefix
			lpmPfx, _ = ip.Prefix(pfxBits)
			return lpmPfx, val, ok2
		}
	}

	return lpmPfx, val, ok
}
