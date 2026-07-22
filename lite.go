// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"sync"

	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/lpm"
	"github.com/gaissmai/bart/internal/nodes"
)

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

	root4 nodes.LiteNode[V]
	root6 nodes.LiteNode[V]

	// the number of prefixes in the routing table
	size4 int
	size6 int
}

// Lookup is just a wrapper for Contains.
// Returns the zero value V and true if a prefix matches ip, otherwise zero value and false.
// This method exists to provide a consistent interface for code generation.
func (l *liteTable[V]) Lookup(ip netip.Addr) (val V, exists bool) {
	return val, l.Contains(ip)
}

// LookupPrefix performs a longest prefix match lookup for any address within
// the given prefix. It finds the most specific routing table entry that would
// match any address in the provided prefix range.
//
// This is functionally identical to LookupPrefixLPM but returns only the
// associated value, not the matching prefix itself.
//
// Returns the zero value and true if a matching prefix is found.
// Returns zero value and false if no match exists.
func (l *liteTable[V]) LookupPrefix(pfx netip.Prefix) (val V, exists bool) {
	_, exists = l.lookupPrefixLPM(pfx, false)
	return
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
// Returns the matching prefix, the zero value, and true if found.
// Returns zero values and false if no match exists.
func (l *liteTable[V]) LookupPrefixLPM(pfx netip.Prefix) (lpm netip.Prefix, val V, exists bool) {
	lpm, exists = l.lookupPrefixLPM(pfx, true)
	return
}

// lookupPrefixLPM performs a longest prefix match lookup for any address within
// the given prefix. It finds the most specific routing table entry that would
// match any address in the provided prefix range. If withLPM is true, it also
// returns the matching longest prefix.
func (l *liteTable[V]) lookupPrefixLPM(pfx netip.Prefix, withLPM bool) (lpmPfx netip.Prefix, ok bool) {
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

	n := l.rootNodeByVersion(is4)

	// record path to leaf node
	stack := [nodes.MaxTreeDepth]*nodes.LiteNode[V]{}

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
		case *nodes.LiteNode[V]:
			n = kid
			continue LOOP // descend down to next trie level

		case *nodes.LeafNode[V]:
			// reached a path compressed prefix, stop traversing
			if kid.Prefix.Bits() > pfxLen || !kid.Prefix.Contains(ip) {
				break LOOP
			}
			return kid.Prefix, true

		case *nodes.FringeNode[V]:
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
		if topIdx, ok = n.Prefixes.IntersectionTop(&lpm.LookupTbl[idx]); ok {
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
