// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"sync"

	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/lpm"
)

// adapter type
type Lite struct {
	// used by -copylocks checker from `go vet`.
	_ [0]sync.Mutex

	liteTable[any]
}

// adapter method, not delegated
func (t *Lite) Insert(pfx netip.Prefix) {
	t.liteTable.Insert(pfx, nil)
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

// insert adds a prefix to the table (idempotent).
// If the prefix already exists, the operation is a no-op.
func (l *liteTable[V]) Insert(pfx netip.Prefix, _ V) {
	if !pfx.IsValid() {
		return
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	is4 := pfx.Addr().Is4()
	n := l.rootNodeByVersion(is4)

	if exists := n.insertAtDepth(pfx, 0); exists {
		return
	}

	// true insert, update size
	l.sizeUpdate(is4, 1)
}

// Delete removes the prefix and returns true if it was present, false otherwise.
func (l *liteTable[V]) Delete(pfx netip.Prefix) (_ V, found bool) {
	var zero V

	if !pfx.IsValid() {
		return
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)

	n := l.rootNodeByVersion(is4)

	// record the nodes on the path to the deleted node, needed to purge
	// and/or path compress nodes after the deletion of a prefix
	stack := [maxTreeDepth]*liteNode[V]{}

	// find the trie node
	for depth, octet := range octets {
		depth = depth & depthMask // BCE, Delete must be fast

		// push current node on stack for path recording
		stack[depth] = n

		// Last “octet” from prefix, update/insert prefix into node.
		// Note: For /32 and /128, depth never reaches lastOctetPlusOne (4/16),
		// so those are handled below via the fringe/leaf path.
		if depth == lastOctetPlusOne {
			// try to delete prefix in trie node
			_, found = n.deletePrefix(art.PfxToIdx(octet, lastBits))
			if !found {
				return
			}

			l.sizeUpdate(is4, -1)
			// remove now-empty nodes and re-path-compress upwards
			n.purgeAndCompress(stack[:depth], octets, is4)
			return zero, true
		}

		if !n.children.Test(octet) {
			return
		}
		kid := n.mustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *liteNode[V]:
			n = kid // descend down to next trie level

		case *liteFringeNode:
			// if pfx is no fringe at this depth, fast exit
			if !isFringe(depth, pfx) {
				return
			}

			// pfx is fringe at depth, delete fringe
			n.deleteChild(octet)

			l.sizeUpdate(is4, -1)
			// remove now-empty nodes and re-path-compress upwards
			n.purgeAndCompress(stack[:depth], octets, is4)

			return zero, true

		case *liteLeafNode:
			// Attention: pfx must be masked to be comparable!
			if kid.prefix != pfx {
				return
			}

			// prefix is equal leaf, delete leaf
			n.deleteChild(octet)

			l.sizeUpdate(is4, -1)
			// remove now-empty nodes and re-path-compress upwards
			n.purgeAndCompress(stack[:depth], octets, is4)

			return zero, true

		default:
			panic("logic error, wrong node type")
		}
	}

	return
}

// Get returns the associated payload for prefix and true, or false if
// prefix is not set in the routing table.
func (l *liteTable[V]) Get(pfx netip.Prefix) (_ V, ok bool) {
	var zero V

	if !pfx.IsValid() {
		return
	}

	// canonicalize the prefix
	pfx = pfx.Masked()

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()

	n := l.rootNodeByVersion(is4)

	lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)

	octets := ip.AsSlice()

	// find the trie node
	for depth, octet := range octets {
		// Last “octet” from prefix, update/insert prefix into node.
		// Note: For /32 and /128, depth never reaches lastOctetPlusOne (4/16),
		// so those are handled below via the fringe/leaf path.
		if depth == lastOctetPlusOne {
			return n.getPrefix(art.PfxToIdx(octet, lastBits))
		}

		if !n.children.Test(octet) {
			return
		}
		kid := n.mustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *liteNode[V]:
			n = kid // descend down to next trie level

		case *liteFringeNode:
			// reached a path compressed fringe, stop traversing
			if isFringe(depth, pfx) {
				return zero, true
			}
			return

		case *liteLeafNode:
			// reached a path compressed prefix, stop traversing
			if kid.prefix == pfx {
				return zero, true
			}
			return

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
}

// Contains reports whether any stored prefix covers the given IP address.
// Returns false for invalid IP addresses.
//
// This performs longest-prefix matching and returns true if any prefix
// in the routing table contains the IP address.
func (l *liteTable[V]) Contains(ip netip.Addr) bool {
	// speed is top priority: no explicit test for ip.Isvalid
	// if ip is invalid, AsSlice() returns nil, Contains returns false.
	is4 := ip.Is4()
	n := l.rootNodeByVersion(is4)

	for _, octet := range ip.AsSlice() {
		// for contains, any lpm match is good enough, no backtracking needed
		if n.pfxCount != 0 && n.contains(art.OctetToIdx(octet)) {
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

		case *liteFringeNode:
			// fringe is the default-route for all possible octets below
			return true

		case *liteLeafNode:
			return kid.prefix.Contains(ip)

		default:
			panic("logic error, wrong node type")
		}
	}

	return false
}

// Lookup, only for interface satisfaction.
func (l *liteTable[V]) Lookup(ip netip.Addr) (_ V, ok bool) {
	var zero V
	return zero, l.Contains(ip)
}

// LookupPrefix does a route lookup (longest prefix match) for pfx and
// returns true, or false if no route matched.
func (l *liteTable[V]) LookupPrefix(pfx netip.Prefix) (_ V, ok bool) {
	_, _, ok = l.lookupPrefixLPM(pfx, false)
	return
}

// LookupPrefixLPM is similar to [Table.LookupPrefix],
// but it returns the lpm prefix in addition to value,ok.
//
// This method is about 20-30% slower than LookupPrefix and should only
// be used if the matching lpm entry is also required for other reasons.
//
// If LookupPrefixLPM is to be used for IP address lookups,
// they must be converted to /32 or /128 prefixes.
func (l *liteTable[V]) LookupPrefixLPM(pfx netip.Prefix) (lpmPfx netip.Prefix, _ V, ok bool) {
	return l.lookupPrefixLPM(pfx, true)
}

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

		case *liteLeafNode:
			// reached a path compressed prefix, stop traversing
			if kid.prefix.Bits() > bits || !kid.prefix.Contains(ip) {
				break LOOP
			}
			return kid.prefix, zero, true

		case *liteFringeNode:
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
		if n.pfxCount == 0 {
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

// Size returns the prefix count.
func (l *liteTable[V]) Size() int {
	return l.size4 + l.size6
}

// Size4 returns the IPv4 prefix count.
func (l *liteTable[V]) Size4() int {
	return l.size4
}

// Size6 returns the IPv6 prefix count.
func (l *liteTable[V]) Size6() int {
	return l.size6
}

func (l *liteTable[V]) sizeUpdate(is4 bool, delta int) {
	if is4 {
		l.size4 += delta
		return
	}
	l.size6 += delta
}
