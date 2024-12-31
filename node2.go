package bart

// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

import (
	"net/netip"
	"slices"

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
	// the generic item [any] is a node (recursive) or a leaf (prefix and value).
	children sparse.Array[any]
}

// leaf is prefix and value together as path compressed child
type leaf[V any] struct {
	prefix netip.Prefix
	value  V
}

// cloneLeaf returns a copy of the leaf.
// If the value implements the Cloner interface, the values are deeply copied.
func (l *leaf[V]) cloneLeaf() *leaf[V] {
	if l == nil {
		return nil
	}
	return &leaf[V]{l.prefix, cloneValue(l.value)}
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
			// push the leaf down
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

// cloneRec, clones the node recursive.
func (n *node2[V]) cloneRec() *node2[V] {
	c := new(node2[V])
	if n.isEmpty() {
		return c
	}

	// shallow
	c.prefixes = *(n.prefixes.Clone())

	// deep copy if V implements Cloner[V]
	for i, v := range c.prefixes.Items {
		c.prefixes.Items[i] = cloneValue(v)
	}

	// shallow
	c.children = *(n.children.Clone())

	// deep copy of nodes and leaves
	for i, k := range c.children.Items {
		switch k := k.(type) {
		case *node2[V]:
			// clone the child node rec-descent
			c.children.Items[i] = k.cloneRec()
		case *leaf[V]:
			// deep copy if V implements Cloner[V]
			c.children.Items[i] = k.cloneLeaf()
		}
	}

	return c
}

// allRec runs recursive the trie, starting at this node and
// the yield function is called for each route entry with prefix and value.
// If the yield function returns false the recursion ends prematurely and the
// false value is propagated.
//
// The iteration order is not defined, just the simplest and fastest recursive implementation.
func (n *node2[V]) allRec(
	path [16]byte,
	depth int,
	is4 bool,
	yield func(netip.Prefix, V) bool,
) bool {
	// for all prefixes in this node do ...
	allIndices := n.prefixes.AsSlice(make([]uint, 0, maxNodePrefixes))
	for _, idx := range allIndices {
		cidr, _ := cidrFromPath(path, depth, is4, idx)

		// callback for this prefix and val
		if !yield(cidr, n.prefixes.MustGet(idx)) {
			// early exit
			return false
		}
	}

	// for all children (nodes and leaves) in this node do ...
	allChildAddrs := n.children.AsSlice(make([]uint, 0, maxNodeChildren))
	for i, addr := range allChildAddrs {
		switch k := n.children.Items[i].(type) {
		case *node2[V]:
			// rec-descent with this node
			path[depth] = byte(addr)
			if !k.allRec(path, depth+1, is4, yield) {
				// early exit
				return false
			}
		case *leaf[V]:
			// callback for this leaf
			if !yield(k.prefix, k.value) {
				// early exit
				return false
			}
		}
	}

	return true
}

// allRecSorted runs recursive the trie, starting at node and
// the yield function is called for each route entry with prefix and value.
// The iteration is in prefix sort order.
//
// If the yield function returns false the recursion ends prematurely and the
// false value is propagated.
func (n *node2[V]) allRecSorted(
	path [16]byte,
	depth int,
	is4 bool,
	yield func(netip.Prefix, V) bool,
) bool {
	// get slice of all child octets, sorted by addr
	allChildAddrs := n.children.AsSlice(make([]uint, 0, maxNodeChildren))

	// get slice of all indexes, sorted by idx
	allIndices := n.prefixes.AsSlice(make([]uint, 0, maxNodePrefixes))

	// sort indices in CIDR sort order
	slices.SortFunc(allIndices, cmpIndexRank)

	childCursor := 0

	// yield indices and childs in CIDR sort order
	for _, pfxIdx := range allIndices {
		pfxOctet, _ := idxToPfx(pfxIdx)

		// yield all childs before idx
		for j := childCursor; j < len(allChildAddrs); j++ {
			childAddr := allChildAddrs[j]

			if childAddr >= uint(pfxOctet) {
				break
			}

			// yield the node (rec-descent) or leaf
			switch k := n.children.Items[j].(type) {
			case *node2[V]:
				// yield this child rec-descent, if matched
				path[depth] = byte(childAddr)
				if !k.allRecSorted(path, depth+1, is4, yield) {
					// early exit
					return false
				}
			case *leaf[V]:
				if !yield(k.prefix, k.value) {
					// early exit
					return false
				}
			}

			childCursor++
		}

		// yield the prefix for this idx
		cidr, _ := cidrFromPath(path, depth, is4, pfxIdx)
		if !yield(cidr, n.prefixes.MustGet(pfxIdx)) {
			// early exit
			return false
		}
	}

	// yield the rest of leaves and nodes (rec-descent)
	for j := childCursor; j < len(allChildAddrs); j++ {
		addr := allChildAddrs[j]
		switch k := n.children.Items[j].(type) {
		case *node2[V]:
			path[depth] = byte(addr)
			if !k.allRecSorted(path, depth+1, is4, yield) {
				return false
			}
		case *leaf[V]:
			if !yield(k.prefix, k.value) {
				return false
			}
		}
	}

	return true
}

/*
// unionRec combines two nodes, changing the receiver node.
// If there are duplicate entries, the value is taken from the other node.
// Count duplicate entries to adjust the t.size struct members.
func (n *node2[V]) unionRec(o *node2[V]) (duplicates int) {
	allIndices := o.prefixes.AsSlice(make([]uint, 0, maxNodePrefixes))
	// for all prefixes in other node do ...
	for i, oIdx := range allIndices {
		// insert/overwrite prefix/value from oNode to nNode
		exists := n.prefixes.InsertAt(oIdx, o.prefixes.Items[i])

		// this prefix is duplicate in n and o
		if exists {
			duplicates++
		}
	}

	allChildAddrs := o.children.AsSlice(make([]uint, 0, maxNodeChildren))
	// for all children in other node do ...
	for i, oAddr := range allChildAddrs {
		oOctet := byte(oAddr)

		// we know the slice index, faster as o.getChild(octet)
		oc := o.children.Items[i]

		// get n child with same octet,
		// we don't know the slice index in n.children
		if nc, ok := n.children.Get(uint(oOctet)); !ok {
			// insert cloned child from oNode into nNode
			n.children.InsertAt(uint(oOctet), oc.cloneRec())
		} else {
			// both nodes have child with octet, call union rec-descent
			duplicates += nc.unionRec(oc)
		}
	}

	return duplicates
}
*/
