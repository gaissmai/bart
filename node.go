// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

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
// The lookup is then slower by a factor of about 2, but this is
// the intended trade-off to prevent memory consumption from exploding.
type node[V any] struct {
	// prefixes contains the routes with payload V
	prefixes sparse.Array[V]

	// children, recursively spans the trie with a branching factor of 256
	children sparse.Array[*node[V]]

	// path compressed items
	pathcomp sparse.Array[*pathItem[V]]
}

type pathItem[V any] struct {
	prefix netip.Prefix
	value  V
}

// isEmpty returns true if node has neither prefixes nor children nor path compressed items.
func (n *node[V]) isEmpty() bool {
	return n.prefixes.Len() == 0 &&
		n.children.Len() == 0 &&
		n.pathcomp.Len() == 0
}

// purgeParents, dangling nodes after successful deletion
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

// ### more complex functions than routing table lookups ###

// eachLookupPrefix does an all prefix match in the 8-bit (stride) routing table
// at this depth and calls yield() for any matching CIDR.
func (n *node[V]) eachLookupPrefix(
	path [16]byte,
	depth int,
	is4 bool,
	octet byte,
	pfxLen int,
	yield func(netip.Prefix, V) bool,
) bool {
	// backtracking the CBT
	for idx := pfxToIdx(octet, pfxLen); idx > 0; idx >>= 1 {
		if val, ok := n.prefixes.Get(idx); ok {
			cidr, _ := cidrFromPath(path, depth, is4, idx)

			if !yield(cidr, val) {
				// early exit
				return false
			}
		}
	}

	return true
}

// eachSubnet calls yield() for any covered CIDR by parent prefix in natural CIDR sort order.
func (n *node[V]) eachSubnet(
	path [16]byte,
	depth int,
	is4 bool,
	octet byte,
	pfxLen int,
	yield func(netip.Prefix, V) bool,
) bool {
	// ###############################################################
	// 1. collect all indices in n covered by prefix
	// ###############################################################
	pfxFirstAddr := uint(octet)
	pfxLastAddr := uint(octet | ^netMask(pfxLen))

	allCoveredIndices := make([]uint, 0, maxNodePrefixes)

	var idx uint
	var ok bool
	for {
		if idx, ok = n.prefixes.NextSet(idx); !ok {
			break
		}

		// idx is covered by prefix
		thisOctet, thisPfxLen := idxToPfx(idx)

		thisFirstAddr := uint(thisOctet)
		thisLastAddr := uint(thisOctet | ^netMask(thisPfxLen))

		if thisFirstAddr >= pfxFirstAddr && thisLastAddr <= pfxLastAddr {
			allCoveredIndices = append(allCoveredIndices, idx)
		}

		idx++
	}

	// sort indices in CIDR sort order
	slices.SortFunc(allCoveredIndices, cmpIndexRank)

	// ###############################################################
	// 2. collect all children in n covered by prefix
	// ###############################################################

	allCoveredAddrs := make([]uint, 0, maxNodeChildren)

	var addr uint

	for {
		if addr, ok = n.children.NextSet(addr); !ok {
			break
		}

		// host addrs are sorted in indexRank order
		if addr > pfxLastAddr {
			break
		}

		if addr >= pfxFirstAddr {
			allCoveredAddrs = append(allCoveredAddrs, addr)
		}

		addr++
	}

	cursor := 0

	// #####################################################
	// 3. yield indices and childs in CIDR sort order
	// #####################################################

	for _, idx := range allCoveredIndices {
		thisOctet, _ := idxToPfx(idx)

		// yield all childs before idx
		for j := cursor; j < len(allCoveredAddrs); j++ {
			addr = allCoveredAddrs[j]

			// yield prefix
			if addr >= uint(thisOctet) {
				break
			}

			// yield child

			octet = byte(addr)
			c, _ := n.children.Get(uint(octet))

			// add (set) this octet to path
			path[depth] = octet

			// all cidrs under this child are covered by pfx
			if !c.allRecSorted(path, depth+1, is4, yield) {
				// early exit
				return false
			}

			cursor++
		}

		// yield the prefix for this idx
		cidr, _ := cidrFromPath(path, depth, is4, idx)
		if !yield(cidr, n.prefixes.MustGet(idx)) {
			// early exit
			return false
		}
	}

	// ###############################################
	// 4. yield the rest of childs, if any
	// ###############################################

	for j := cursor; j < len(allCoveredAddrs); j++ {
		addr = allCoveredAddrs[j]

		octet = byte(addr)
		c, _ := n.children.Get(uint(octet))

		// add (set) this octet to path
		path[depth] = octet

		// all cidrs under this child are covered by pfx
		if !c.allRecSorted(path, depth+1, is4, yield) {
			// early exit
			return false
		}
	}

	return true
}

// unionRec combines two nodes, changing the receiver node.
// If there are duplicate entries, the value is taken from the other node.
// Count duplicate entries to adjust the t.size struct members.
func (n *node[V]) unionRec(o *node[V]) (duplicates int) {
	// no heap allocs
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

	// no heap allocs
	allChildAddrs := o.children.AsSlice(make([]uint, 0, maxNodeChildren))

	// for all children in other node do ...
	for i, oOctet := range allChildAddrs {
		octet := byte(oOctet)

		// we know the slice index, faster as o.getChild(octet)
		oc := o.children.Items[i]

		// get n child with same octet,
		// we don't know the slice index in n.children
		if nc, ok := n.children.Get(uint(octet)); !ok {
			// insert cloned child from oNode into nNode
			n.children.InsertAt(uint(octet), oc.cloneRec())
		} else {
			// both nodes have child with octet, call union rec-descent
			duplicates += nc.unionRec(oc)
		}
	}

	return duplicates
}

// cloneRec, clones the node recursive.
func (n *node[V]) cloneRec() *node[V] {
	c := new(node[V])
	if n.isEmpty() {
		return c
	}

	c.prefixes.BitSet = n.prefixes.BitSet.Clone()     // deep
	c.prefixes.Items = slices.Clone(n.prefixes.Items) // values, shallow copy

	// deep copy if V implements Cloner[V]
	for i, v := range c.prefixes.Items {
		if v, ok := any(v).(Cloner[V]); ok {
			c.prefixes.Items[i] = v.Clone()
		} else {
			break
		}
	}

	c.children.BitSet = n.children.BitSet.Clone()     // deep
	c.children.Items = slices.Clone(n.children.Items) // children, shallow copy

	// deep copy of children
	for i, child := range c.children.Items {
		c.children.Items[i] = child.cloneRec()
	}

	// #######################################
	//          path compression
	// #######################################

	c.pathcomp.BitSet = n.pathcomp.BitSet.Clone()     // deep
	c.pathcomp.Items = slices.Clone(n.pathcomp.Items) // values, shallow copy

	// deep copy
	for i, pc := range c.pathcomp.Items {
		item := *pc

		// deep copy if V implements Cloner[V]
		if v, ok := any(item.value).(Cloner[V]); ok {
			item.value = v.Clone()
		}
		c.pathcomp.Items[i] = &item
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
	allIndices := n.prefixes.AsSlice(make([]uint, 0, maxNodePrefixes))

	// for all prefixes in this node do ...
	for _, idx := range allIndices {
		cidr, _ := cidrFromPath(path, depth, is4, idx)

		// make the callback for this prefix
		if !yield(cidr, n.prefixes.MustGet(idx)) {
			// early exit
			return false
		}
	}

	// for all path compressed items do ...
	for _, pc := range n.pathcomp.Items {
		// make the callback for this prefix
		if !yield(pc.prefix, pc.value) {
			// early exit
			return false
		}
	}

	allChildAddrs := n.children.AsSlice(make([]uint, 0, maxNodeChildren))
	// for all children in this node do ...
	for i, addr := range allChildAddrs {
		child := n.children.Items[i]
		path[depth] = byte(addr)

		if !child.allRec(path, depth+1, is4, yield) {
			// early exit
			return false
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
	// get slice of all indexes, sorted by idx
	allIndices := n.prefixes.AsSlice(make([]uint, 0, maxNodePrefixes))

	// sort indices in CIDR sort order
	slices.SortFunc(allIndices, cmpIndexRank)

	// get all the bits in fast adressable form as a set of bool.
	allChildSet := n.children.AsSet(make([]bool, 0, maxNodeChildren))
	allPathCompSet := n.pathcomp.AsSet(make([]bool, 0, maxNodeChildren))

	// yield indices, pathcomp prefixes and childs in CIDR sort order
	var lower, upper uint

	for _, idx := range allIndices {
		pfxAddr, _ := idxToPfx(idx)
		upper = uint(pfxAddr)

		// for all pathcomp and child items < pfxAddr
		for addr := lower; addr < upper; addr++ {
			// either pathcomp or children match this addr, but not possible for both

			if allPathCompSet[addr] {
				pc := n.pathcomp.MustGet(addr)
				if !yield(pc.prefix, pc.value) {
					return false
				}
			}

			if allChildSet[addr] {
				// yield this child rec-descent, if matched
				c := n.children.MustGet(addr)
				path[depth] = byte(addr)
				if !c.allRecSorted(path, depth+1, is4, yield) {
					return false
				}
			}

		}

		// yield the prefix for this idx
		cidr, _ := cidrFromPath(path, depth, is4, idx)
		if !yield(cidr, n.prefixes.MustGet(idx)) {
			return false
		}

		// forward lower bound for next round
		lower = upper
	}

	// yield the rest of pathcomp and child items, if any
	for addr := lower; addr < maxNodeChildren; addr++ {
		// either pathcomp or children match this addr, but not possible for both

		if allPathCompSet[addr] {
			pc := n.pathcomp.MustGet(addr)
			if !yield(pc.prefix, pc.value) {
				return false
			}
		}

		if allChildSet[addr] {
			c := n.children.MustGet(addr)
			path[depth] = byte(addr)
			if !c.allRecSorted(path, depth+1, is4, yield) {
				return false
			}
		}
	}

	return true
}
