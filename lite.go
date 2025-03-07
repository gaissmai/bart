// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"

	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/bitset"
	"github.com/gaissmai/bart/internal/lpmbt"
	"github.com/gaissmai/bart/internal/sparse"
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

	n := l.rootNodeByVersion(is4)

	lastIdx, lastBits := lastOctetIdxAndBits(bits)

	octets := ip.AsSlice()
	octets = octets[:lastIdx+1]

	// record path to deleted node
	// needed to purge and/or path compress nodes after deletion
	stack := [maxTreeDepth]*liteNode{}

	// find the trie node
	for depth, octet := range octets {
		// push current node on stack for path recording
		stack[depth] = n

		// delete prefix in trie node
		if depth == lastIdx {
			n.prefixes = n.prefixes.Clear(art.PfxToIdx(octet, lastBits))
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

		case *prefixNode:
			// reached a path compressed prefix, stop traversing
			if kid.Prefix != pfx {
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

		case *prefixNode:
			// kid is a path-compressed prefix
			return kid.Contains(ip)

		default:
			panic("logic error, wrong node type")
		}
	}

	// invalid IP
	return false
}

// ###################################################################

// liteNode, see the node struct, but without payload V.
// Needs less memory and insert and delete is also a bit faster.
type liteNode struct {
	prefixes bitset.BitSet
	children sparse.Array[any] // [any] is a *liteNode or a *prefixNode
}

// prefixNode, just a path compressed prefix.
type prefixNode struct {
	netip.Prefix
}

// lpmTest, any longest prefix match
func (n *liteNode) lpmTest(idx uint) bool {
	return n.prefixes.IntersectsAny(lpmbt.LookupTbl[idx])
}

// insertAtDepth, see the similar method for node, but now simpler without payload V.
func (n *liteNode) insertAtDepth(pfx netip.Prefix, depth int) {
	ip := pfx.Addr()
	bits := pfx.Bits()

	lastIdx, lastBits := lastOctetIdxAndBits(bits)

	octets := ip.AsSlice()
	octets = octets[:lastIdx+1]

	// find the proper trie node to insert prefix
	// start with prefix octet at depth
	for ; depth < len(octets); depth++ {
		octet := octets[depth]
		addr := uint(octet)

		// last significant octet: insert/override prefix into node
		if depth == lastIdx {
			// just set a bit, no payload to insert
			n.prefixes = n.prefixes.Set(art.PfxToIdx(octet, lastBits))
			return
		}

		if !n.children.Test(addr) {
			// insert prefix as path-compressed leaf
			n.children.InsertAt(addr, &prefixNode{pfx})
			return
		}
		kid := n.children.MustGet(addr)

		// kid is node or leaf at addr
		switch kid := kid.(type) {
		case *liteNode:
			n = kid
			continue // descend down to next trie level

		case *prefixNode:
			// reached a path-compressed leaf, just a netip.Prefix
			if kid.Prefix == pfx {
				// already exists, nothing to do
				return
			}

			// create new node
			// push the leaf down
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(liteNode)
			newNode.insertAtDepth(kid.Prefix, depth+1)

			n.children.InsertAt(addr, newNode)
			n = newNode

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
}

// purgeAndCompress, purge empty nodes or compress nodes with single prefix or leaf.
// similar to the same helper method for node, but without payload V.
func (n *liteNode) purgeAndCompress(parentStack []*liteNode, childPath []uint8, is4 bool) {
	// unwind the stack
	for i := len(parentStack) - 1; i >= 0; i-- {
		parent := parentStack[i]
		addr := uint(childPath[i])

		prefixCount := n.prefixes.Size()
		childCount := n.children.Len()

		switch {
		case prefixCount == 0 && childCount == 0:
			// just delete this empty node
			parent.children.DeleteAt(addr)

		case prefixCount == 0 && childCount == 1:
			// if single child is a path-compressed leaf, shift it up one level
			// and override current node with this leaf
			if pfx, ok := n.children.Items[0].(netip.Prefix); ok {
				parent.children.InsertAt(addr, &prefixNode{pfx})
			}

		case prefixCount == 1 && childCount == 0:
			// make prefix from idx, shift leaf one level up
			// and override current node with new leaf
			idx, _ := n.prefixes.FirstSet()

			path := stridePath{}
			copy(path[:], childPath)
			pfx := cidrFromPath(path, i+1, is4, idx)

			parent.children.InsertAt(addr, &prefixNode{pfx})
		}

		n = parent
	}
}
