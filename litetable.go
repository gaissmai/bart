// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"sync"

	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/lpm"
)

type Foo struct {
	// used by -copylocks checker from `go vet`.
	_ [0]sync.Mutex

	root4 liteNode[any]
	root6 liteNode[any]

	// the number of prefixes in the routing table
	size4 int
	size6 int
}

// rootNodeByVersion, root node getter for ip version.
func (l *Foo) rootNodeByVersion(is4 bool) *liteNode[any] {
	if is4 {
		return &l.root4
	}
	return &l.root6
}

// Insert adds a pfx to the tree, with given val.
// If pfx is already present in the tree, its value is set to val.
func (l *Foo) Insert(pfx netip.Prefix) {
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

// Delete the prefix and returns the associated payload for prefix and true if found
// or the zero value and false if prefix is not set in the routing table.
func (l *Foo) Delete(pfx netip.Prefix) (found bool) {
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
	stack := [maxTreeDepth]*liteNode[any]{}

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
			return true
		}

		if !n.children.Test(octet) {
			return
		}
		kid := n.mustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *liteNode[any]:
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

			return true

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

			return true

		default:
			panic("logic error, wrong node type")
		}
	}

	return
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
func (l *Foo) Contains(ip netip.Addr) bool {
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
		case *liteNode[any]:
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

// Lookup performs longest-prefix matching for the given IP address and returns
// the associated value of the most specific matching prefix.
// Returns the zero value of V and false if no prefix matches.
// Returns false for invalid IP addresses.
//
// This is the core routing table operation used for packet forwarding decisions.
func (l *Foo) Lookup(ip netip.Addr) (ok bool) {
	if !ip.IsValid() {
		return
	}

	is4 := ip.Is4()
	octets := ip.AsSlice()

	n := l.rootNodeByVersion(is4)

	// stack of the traversed nodes for fast backtracking, if needed
	stack := [maxTreeDepth]*liteNode[any]{}

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
		case *liteNode[any]:
			n = kid
			continue LOOP // descend down to next trie level

		case *liteFringeNode:
			// fringe is the default-route for all possible nodes below
			return true

		case *liteLeafNode:
			if kid.prefix.Contains(ip) {
				return true
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
		if n.pfxCount != 0 {
			idx := art.OctetToIdx(octets[depth])
			// lookupIdx() manually inlined
			if _, ok := n.prefixes.IntersectionTop(&lpm.LookupTbl[idx]); ok {
				return true
			}
		}
	}

	return
}

// Size returns the prefix count.
func (l *Foo) Size() int {
	return l.size4 + l.size6
}

// Size4 returns the IPv4 prefix count.
func (l *Foo) Size4() int {
	return l.size4
}

// Size6 returns the IPv6 prefix count.
func (l *Foo) Size6() int {
	return l.size6
}

func (l *Foo) sizeUpdate(is4 bool, n int) {
	if is4 {
		l.size4 += n
		return
	}
	l.size6 += n
}
