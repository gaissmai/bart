package bart

// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

import (
	"net/netip"
	"slices"

	"github.com/gaissmai/bart/internal/sparse"
)

const (
	strideLen       = 8   // octet
	maxTreeDepth    = 16  // 16 for IPv6
	maxNodeChildren = 256 // 256
	maxNodePrefixes = 512 // 512
)

// a zero value, used manifold
var zeroPath [16]byte

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
type node[V any] struct {
	// prefixes contains the routes as complete binary tree with payload V
	prefixes sparse.Array[V]

	// children, recursively spans the trie with a branching factor of 256
	// the generic child with empty interface is a node (recursive) or
	// a path compressed leaf (prefix and value).
	children sparse.Array[interface{}]
}

// leaf is prefix and value together as path compressed child
type leaf[V any] struct {
	prefix netip.Prefix
	value  V
}

// cloneValue, deep copy if v implements the Cloner interface.
func cloneValue[V any](v V) V {
	if k, ok := any(v).(Cloner[V]); ok {
		return k.Clone()
	}
	return v
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
func (n *node[V]) isEmpty() bool {
	return n.prefixes.Len() == 0 && n.children.Len() == 0
}

// nodeAndLeafCount
func (n *node[V]) nodeAndLeafCount() (nodes int, leaves int) {
	for i := range n.children.AsSlice(make([]uint, 0, maxNodeChildren)) {
		switch n.children.Items[i].(type) {
		case *node[V]:
			nodes++
		case *leaf[V]:
			leaves++
		}
	}
	return
}

// nodeAndLeafCountRec, calculate the number of nodes and leaves under n, rec-descent.
func (n *node[V]) nodeAndLeafCountRec() (int, int) {
	if n == nil || n.isEmpty() {
		return 0, 0
	}

	nodes := 1 // this node
	leaves := 0

	for _, c := range n.children.Items {
		switch k := c.(type) {
		case *node[V]:
			// rec-descent
			ns, ls := k.nodeAndLeafCountRec()
			nodes += ns
			leaves += ls

		case *leaf[V]:
			leaves++
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
func (n *node[V]) insertAtDepth(pfx netip.Prefix, val V, depth int) (exists bool) {
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
		case *node[V]:
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
			c := new(node[V])
			c.insertAtDepth(k.prefix, k.value, depth+1)

			n.children.InsertAt(addr, c)
			n = c
		}
	}

	// last significant octet: insert/override prefix/val into node
	return n.prefixes.InsertAt(pfxToIdx(sigOctet, sigOctetBits), val)
}

// TODO, path compress after purging dangling leaves
func (n *node[V]) purgeParents(parentStack []*node[V], childPath []byte) {
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
func (n *node[V]) lpm(idx uint) (baseIdx uint, val V, ok bool) {
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
func (n *node[V]) lpmTest(idx uint) bool {
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
func (n *node[V]) cloneRec() *node[V] {
	c := new(node[V])
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
		case *node[V]:
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
func (n *node[V]) allRec(
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
		case *node[V]:
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
func (n *node[V]) allRecSorted(
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
			case *node[V]:
				path[depth] = byte(childAddr)
				if !k.allRecSorted(path, depth+1, is4, yield) {
					return false
				}
			case *leaf[V]:
				if !yield(k.prefix, k.value) {
					return false
				}
			}

			childCursor++
		}

		// yield the prefix for this idx
		cidr, _ := cidrFromPath(path, depth, is4, pfxIdx)
		// n.prefixes.Items[i] not possible after sorting allIndices
		if !yield(cidr, n.prefixes.MustGet(pfxIdx)) {
			return false
		}
	}

	// yield the rest of leaves and nodes (rec-descent)
	for j := childCursor; j < len(allChildAddrs); j++ {
		addr := allChildAddrs[j]
		switch k := n.children.Items[j].(type) {
		case *node[V]:
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

// unionRec combines two nodes, changing the receiver node.
// If there are duplicate entries, the value is taken from the other node.
// Count duplicate entries to adjust the t.size struct members.
func (n *node[V]) unionRec(o *node[V], depth int) (duplicates int) {
	// for all prefixes in other node do ...
	allIndices := o.prefixes.AsSlice(make([]uint, 0, maxNodePrefixes))
	for i, oIdx := range allIndices {
		// insert/overwrite prefix/value from oNode to nNode
		exists := n.prefixes.InsertAt(oIdx, o.prefixes.Items[i])

		// this prefix is duplicate in n and o
		if exists {
			duplicates++
		}
	}

	// for all child addrs in other node do ...
	allOtherChildAddrs := o.children.AsSlice(make([]uint, 0, maxNodeChildren))
LOOP:
	for i, addr := range allOtherChildAddrs {
		//  6 possible combinations for this child and other child child
		//
		//  THIS, OTHER:
		//  ----------
		//  NULL, node  <-- easy,    insert at cloned node
		//  NULL, leaf  <-- easy,    insert at cloned leaf
		//  node, node  <-- easy,    union rec-descent
		//  node, leaf  <-- easy,    insert other cloned leaf at depth+1
		//  leaf, node  <-- complex, push this leaf down, union rec-descent
		//  leaf, leaf  <-- complex, push this leaf down, insert other cloned leaf at depth+1
		//
		// try to get child at same addr from n
		thisChild, thisExists := n.children.Get(addr)
		if !thisExists {
			switch otherChild := o.children.Items[i].(type) {

			case *node[V]: // NULL, node
				if !thisExists {
					n.children.InsertAt(addr, otherChild.cloneRec())
					continue LOOP
				}

			case *leaf[V]: // NULL, leaf
				if !thisExists {
					n.children.InsertAt(addr, otherChild.cloneLeaf())
					continue LOOP
				}
			}
		}

		switch otherChild := o.children.Items[i].(type) {

		case *node[V]:
			switch this := thisChild.(type) {

			case *node[V]: // node, node
				// both childs have node in octet, call union rec-descent on child nodes
				duplicates += this.unionRec(otherChild, depth+1)
				continue LOOP

			case *leaf[V]: // leaf, node
				// create new node
				nc := new(node[V])

				// push this leaf down
				nc.insertAtDepth(this.prefix, this.value, depth+1)

				// insert new node at current addr
				n.children.InsertAt(addr, nc)

				// union rec-descent new node with other node
				duplicates += nc.unionRec(otherChild, depth+1)
				continue LOOP
			}

		case *leaf[V]:
			switch this := thisChild.(type) {

			case *node[V]: // node, leaf
				clonedLeaf := otherChild.cloneLeaf()
				if this.insertAtDepth(clonedLeaf.prefix, clonedLeaf.value, depth+1) {
					duplicates++
				}
				continue LOOP

			case *leaf[V]: // leaf, leaf
				// create new node
				nc := new(node[V])

				// push this leaf down
				nc.insertAtDepth(this.prefix, this.value, depth+1)

				// insert at depth cloned leaf
				clonedLeaf := otherChild.cloneLeaf()
				if nc.insertAtDepth(clonedLeaf.prefix, clonedLeaf.value, depth+1) {
					duplicates++
				}

				// insert the new node at current addr
				n.children.InsertAt(addr, nc)
				continue LOOP
			}
		}
	}

	return duplicates
}

// eachLookupPrefix does an all prefix match in the 8-bit (stride) routing table
// at this depth and calls yield() for any matching CIDR.
func (n *node[V]) eachLookupPrefix(octets []byte, depth int, is4 bool, pfxLen int, yield func(netip.Prefix, V) bool) (ok bool) {
	var path [16]byte
	copy(path[:], octets)

	if n.prefixes.Len() == 0 {
		return true
	}

	// backtracking the CBT
	for idx := pfxToIdx(octets[depth], pfxLen); idx > 0; idx >>= 1 {
		if n.prefixes.Test(idx) {
			val := n.prefixes.MustGet(idx)
			cidr, _ := cidrFromPath(path, depth, is4, idx)

			if !yield(cidr, val) {
				return false
			}
		}
	}

	return true
}

// eachSubnet calls yield() for any covered CIDR by parent prefix in natural CIDR sort order.
func (n *node[V]) eachSubnet(octets []byte, depth int, is4 bool, pfxLen int, yield func(netip.Prefix, V) bool) bool {
	// octets as array, needed below more than once
	var path [16]byte
	copy(path[:], octets)

	pfxFirstAddr := uint(octets[depth])
	pfxLastAddr := uint(octets[depth] | ^netMask(pfxLen))

	// 1. collect all indices in n covered by prefix

	allCoveredIndices := make([]uint, 0, maxNodePrefixes)
	for _, idx := range n.prefixes.AsSlice(make([]uint, 0, maxNodePrefixes)) {
		thisOctet, thisPfxLen := idxToPfx(idx)

		thisFirstAddr := uint(thisOctet)
		thisLastAddr := uint(thisOctet | ^netMask(thisPfxLen))

		if thisFirstAddr >= pfxFirstAddr && thisLastAddr <= pfxLastAddr {
			allCoveredIndices = append(allCoveredIndices, idx)
		}
	}

	// sort indices in CIDR sort order
	slices.SortFunc(allCoveredIndices, cmpIndexRank)

	// 2. collect all covered child addrs by prefix

	allCoveredChildAddrs := make([]uint, 0, maxNodeChildren)
	for _, addr := range n.children.AsSlice(make([]uint, 0, maxNodeChildren)) {
		if addr >= pfxFirstAddr && addr <= pfxLastAddr {
			allCoveredChildAddrs = append(allCoveredChildAddrs, addr)
		}
	}

	// 3. yield covered indices, pathcomp prefixes and childs in CIDR sort order

	addrCursor := 0

	// yield indices and childs in CIDR sort order
	for _, pfxIdx := range allCoveredIndices {
		pfxOctet, _ := idxToPfx(pfxIdx)

		// yield all childs before idx
		for j := addrCursor; j < len(allCoveredChildAddrs); j++ {
			addr := allCoveredChildAddrs[j]
			if addr >= uint(pfxOctet) {
				break
			}

			// yield the node or leaf?
			switch k := n.children.MustGet(addr).(type) {

			case *node[V]:
				path[depth] = byte(addr)
				if !k.allRecSorted(path, depth+1, is4, yield) {
					return false
				}

			case *leaf[V]:
				if !yield(k.prefix, k.value) {
					return false
				}
			}

			addrCursor++
		}

		// yield the prefix for this idx
		cidr, _ := cidrFromPath(path, depth, is4, pfxIdx)
		// n.prefixes.Items[i] not possible after sorting allIndices
		if !yield(cidr, n.prefixes.MustGet(pfxIdx)) {
			return false
		}
	}

	// yield the rest of leaves and nodes (rec-descent)
	for _, addr := range allCoveredChildAddrs[addrCursor:] {
		// yield the node or leaf?
		switch k := n.children.MustGet(addr).(type) {

		case *node[V]:
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
