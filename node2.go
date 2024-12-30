package bart

// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

import (
	"net/netip"

	"github.com/gaissmai/bart/internal/sparse"
)

// node is a level node in the multibit-trie.
// A node has prefixes and children, forming the multibit trie.
//
// The prefixes form a complete binary tree, see the artlookup.pdf
// paper in the doc folder to understand the data structure.
//
// In contrast to the ART algorithm, sparse arrays
// (popcount-compressed slices) are used instead of fixed-size arrays.
//
// The array slots are also not pre-allocated (alloted) as described
// in the ART algorithm, but backtracking is used for the longest-prefix-match.
//
// The sparse child array recursively spans the trie with a branching factor of 256
// and also records path-compressed leaves in the free child slots.
type node2[V any] struct {
	// prefixes contains the routes as complete binary tree with payload V
	prefixes sparse.Array[V]

	// children, recursively spans the trie with a branching factor of 256
	// the interface item (any) is a node (recursive) or a leaf (prefix and value).
	children sparse.Array[any]
}

// leaf is prefix and value together as path compressed child
type leaf[V any] struct {
	prefix netip.Prefix
	value  V
}

// isEmpty returns true if node has neither prefixes nor children
func (n *node2[V]) isEmpty() bool {
	return n.prefixes.Len() == 0 && n.children.Len() == 0
}

// nodeAndLeafCount
func (n *node2[V]) nodeAndLeafCount() (nodes int, leaves int) {
	for i := range n.children.AsSlice(make([]uint, 0, maxNodeChildren)) {
		switch n.children.Items[i].(type) {
		case *node2[V]:
			nodes++
		case *leaf[V]:
			leaves++
		}
	}
	return
}

// nodeAndLeafCountRec, calculate the number of nodes and leaves under n, rec-descent.
func (n *node2[V]) nodeAndLeafCountRec() (int, int) {
	if n == nil || n.isEmpty() {
		return 0, 0
	}

	nodes := 1 // this node
	leaves := 0

	for _, c := range n.children.Items {
		switch k := c.(type) {
		case *node2[V]:
			// rec-descent
			ns, ls := k.nodeAndLeafCountRec()
			nodes += ns
			leaves += ls
		case *leaf[V]:
			leaves++
		default:
			panic("logic error")
		}
	}

	return nodes, leaves
}

// insertAtDepth insert a prefix/val into a node tree at depth.
// n must not be nil, prefix must be valid and already in canonical form.
//
// If a path compression has to be resolved because a new value is added
// that collides with a leaf, the compressed leaf is then reinserted
// one depth down in the node trie.
func (n *node2[V]) insertAtDepth(pfx netip.Prefix, val V, depth int) (exists bool) {
	octets := pfx.Addr().AsSlice()
	bits := pfx.Bits()

	// 10.0.0.0/8    -> 0
	// 10.12.0.0/15  -> 1
	// 10.12.0.0/16  -> 1
	// 10.12.10.9/32 -> 3
	sigOctetIdx := (bits - 1) / strideLen

	// 10.0.0.0/8    -> 10
	// 10.12.0.0/15  -> 12
	// 10.12.0.0/16  -> 12
	// 10.12.10.9/32 -> 9
	sigOctet := octets[sigOctetIdx]

	// 10.0.0.0/8    -> 8
	// 10.12.0.0/15  -> 7
	// 10.12.0.0/16  -> 8
	// 10.12.10.9/32 -> 8
	sigOctetBits := bits - (sigOctetIdx * strideLen)

	// find the proper trie node to insert prefix
	// start with prefix octet at depth
	for ; depth < sigOctetIdx; depth++ {
		addr := uint(octets[depth])

		if !n.children.Test(addr) {
			// insert prefix path compressed
			return n.children.InsertAt(addr, &leaf[V]{pfx, val})
		}

		// get the child: node or leaf
		switch k := n.children.MustGet(addr).(type) {
		case *node2[V]:
			// descend down to next trie level
			n = k
		case *leaf[V]:
			// reached a path compressed prefix
			// override value in slot if prefixes are equal
			if k.prefix == pfx {
				k.value = val
				// exists
				return true
			}

			// create new node
			// push leaf down
			// insert new child at cureent leaf position (addr)
			// descend down, replace n with new child
			c := new(node2[V])
			c.insertAtDepth(k.prefix, k.value, depth+1)

			n.children.InsertAt(addr, c)
			n = c
			// continue to next octet
		}
	}

	// last significant octet: insert/override prefix/val into node
	return n.prefixes.InsertAt(pfxToIdx(sigOctet, sigOctetBits), val)
}

// TODO, path compress after purging dangling leaves
func (n *node2[V]) purgeParents(parentStack []*node2[V], childPath []byte) {
	for i := len(parentStack) - 1; i >= 0; i-- {
		if n.isEmpty() {
			parent := parentStack[i]
			parent.children.DeleteAt(uint(childPath[i]))
		}
		n = parentStack[i]
	}
}

// lpm does a route lookup for idx in the 8-bit (stride) routing table
// at this depth and returns (baseIdx, value, true) if a matching
// longest prefix exists, or ok=false otherwise.
//
// backtracking is fast, it's just a bitset test and, if found, one popcount.
// max steps in backtracking is the stride length.
func (n *node2[V]) lpm(idx uint) (baseIdx uint, val V, ok bool) {
	// shortcut optimization
	minIdx, ok := n.prefixes.FirstSet()
	if !ok {
		return 0, val, false
	}

	// backtracking the CBT
	for baseIdx = idx; baseIdx >= minIdx; baseIdx >>= 1 {
		// practically it's get, but get is not inlined
		if n.prefixes.Test(baseIdx) {
			return baseIdx, n.prefixes.MustGet(baseIdx), true
		}
	}

	// not found (on this level)
	return 0, val, false
}

// lpmTest for faster lpm tests without value returns
func (n *node2[V]) lpmTest(idx uint) bool {
	// shortcut optimization
	minIdx, ok := n.prefixes.FirstSet()
	if !ok {
		return false
	}

	// backtracking the CBT
	for idx := idx; idx >= minIdx; idx >>= 1 {
		if n.prefixes.Test(idx) {
			return true
		}
	}

	return false
}
