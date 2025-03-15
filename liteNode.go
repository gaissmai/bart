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

// liteNode, for description see similar node struct, but here without payload V.
// Needs less memory and insert and delete is also a bit faster.
type liteNode struct {
	prefixes bitset.BitSet256
	children sparse.Array256[any] // [any] is a *liteNode or a *liteLeaf
}

// liteLeaf, just a path compressed prefix and a fringe flag.
// For explanation of the fringe flag see the func isFringe()
type liteLeaf struct {
	prefix netip.Prefix
	fringe bool
}

// lpmTest, true if idx has a (any) longest-prefix-match in node.
func (n *liteNode) lpmTest(idx uint) bool {
	return n.prefixes.IntersectsAny(lpmbt.LookupTbl[idx])
}

// insertAtDepth, see the similar method for node, but now simpler without payload V.
func (n *liteNode) insertAtDepth(pfx netip.Prefix, depth int) {
	ip := pfx.Addr()
	bits := pfx.Bits()
	octets := ip.AsSlice()
	lastIdx, lastBits := lastOctetIdxAndBits(bits)

	// find the proper trie node to insert prefix
	// start with prefix octet at depth
	for ; depth < len(octets); depth++ {
		octet := octets[depth]
		addr := uint(octet)

		// last significant octet: set prefix idx in node
		if depth == lastIdx {
			n.prefixes.Set(art.PfxToIdx(octet, lastBits))
			return
		}

		// reached end of trie path ...
		if !n.children.Test(addr) {
			// insert prefix as path-compressed leaf
			n.children.InsertAt(addr, &liteLeaf{pfx, isFringe(depth, bits)})
			return
		}

		// ... or decend down the trie
		kid := n.children.MustGet(addr)

		// kid is node or leaf at addr
		switch kid := kid.(type) {
		case *liteNode:
			n = kid
			continue // descend down to next trie level

		case *liteLeaf:
			// reached a path compressed prefix
			if kid.prefix == pfx {
				// already exists, nothing to do
				return
			}

			// create new node
			// push the leaf down
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(liteNode)
			newNode.insertAtDepth(kid.prefix, depth+1)

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
	for depth := len(parentStack) - 1; depth >= 0; depth-- {
		parent := parentStack[depth]
		addr := uint(childPath[depth])

		prefixCount := n.prefixes.Size()
		childCount := n.children.Len()

		switch {
		case prefixCount == 0 && childCount == 0:
			// just delete this empty node
			parent.children.DeleteAt(addr)

		case prefixCount == 0 && childCount == 1:
			// if child is a leaf (not a node), shift it up one level
			if kid, ok := n.children.Items[0].(*liteLeaf); ok {
				// delete this node
				parent.children.DeleteAt(addr)

				// ... insert prefix at parents depth
				parent.insertAtDepth(kid.prefix, depth)
			}

		case prefixCount == 1 && childCount == 0:
			// get prefix back from idx
			idx, _ := n.prefixes.FirstSet()

			// ... and octet path
			path := stridePath{}
			copy(path[:], childPath)
			pfx := cidrFromPath(path, depth+1, is4, idx)

			// delete this node
			parent.children.DeleteAt(addr)

			// ... insert prefix at parents depth
			parent.insertAtDepth(pfx, depth)
		}

		// climb up the stack
		n = parent
	}
}
