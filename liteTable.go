// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"

	"github.com/gaissmai/bart/internal/art"
)

// Lite is the little sister of [Table]. Lite is ideal for simple
// IP access-control-lists, a.k.a. longest-prefix matches
// with plain true/false results.
//
// For all other tasks the much more powerful [Table] must be used.
type Lite struct {
	// used by -copylocks checker from `go vet`.
	_ noCopy

	// the root nodes, implemented as popcount compressed multibit tries
	root4 liteNode
	root6 liteNode
}

// rootNodeByVersion, root node getter for ip version.
func (l *Lite) rootNodeByVersion(is4 bool) *liteNode {
	if is4 {
		return &l.root4
	}
	return &l.root6
}

// Insert adds pfx to the trie.
func (l *Lite) Insert(pfx netip.Prefix) {
	if !pfx.IsValid() {
		return
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	is4 := pfx.Addr().Is4()
	n := l.rootNodeByVersion(is4)

	n.insertAtDepth(pfx, 0)
}

// Delete removes pfx from the trie.
func (l *Lite) Delete(pfx netip.Prefix) {
	if !pfx.IsValid() {
		return
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()
	octets := ip.AsSlice()
	lastIdx, lastBits := lastOctetIdxAndBits(bits)

	n := l.rootNodeByVersion(is4)

	// record path to deleted node
	// needed to purge and/or path compress nodes after deletion
	stack := [maxTreeDepth]*liteNode{}

	// find the trie node
	for depth, octet := range octets {
		depth = depth & 0xf // BCE

		if depth > lastIdx {
			break
		}

		// push current node on stack for path recording
		stack[depth] = n

		// delete prefix in trie node
		if depth == lastIdx {
			n.prefixes.MustClear(art.PfxToIdx(octet, lastBits))
			n.purgeAndCompress(stack[:depth], octets, is4)
			return
		}

		addr := uint(octet)
		if !n.children.Test(addr) {
			return
		}
		kid := n.children.MustGet(addr)

		// kid is node or leaf at addr
		switch kid := kid.(type) {
		case *liteNode:
			n = kid
			continue // descend down to next trie level

		case *liteLeaf:
			// reached a path compressed prefix, stop traversing
			if kid.prefix != pfx {
				// nothing to delete
				return
			}

			// prefix is equal leaf, delete leaf
			n.children.DeleteAt(addr)
			n.purgeAndCompress(stack[:depth], octets, is4)

			return

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
}

// Contains performs a longest-prefix match for the IP address
// and returns true if any route matches, otherwise false.
func (l *Lite) Contains(ip netip.Addr) bool {
	// if ip is invalid, Is4() returns false and AsSlice() returns nil
	is4 := ip.Is4()
	n := l.rootNodeByVersion(is4)

	for _, octet := range ip.AsSlice() {
		addr := uint(octet)

		// for contains, any lpm match is good enough, no backtracking needed
		if n.lpmTest(art.HostIdx(addr)) {
			return true
		}

		// stop traversing?
		if !n.children.Test(addr) {
			return false
		}
		kid := n.children.MustGet(addr)

		// kid is node or leaf at addr
		switch kid := kid.(type) {
		case *liteNode:
			n = kid
			continue // descend down to next trie level

		case *liteLeaf:
			// fringe is the default-route for all nodes below
			return kid.fringe || kid.prefix.Contains(ip)

		default:
			panic("logic error, wrong node type")
		}
	}

	// invalid IP
	return false
}
