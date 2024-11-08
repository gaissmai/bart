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
	prefixes *sparse.Array[V]

	// children, recursively spans the trie with a branching factor of 256
	children *sparse.Array[*node[V]]
}

// newNode with sparse arrays for prefixes and children.
func newNode[V any]() *node[V] {
	return &node[V]{
		prefixes: sparse.NewArray[V](),
		children: sparse.NewArray[*node[V]](),
	}
}

// isEmpty returns true if node has neither prefixes nor children.
func (n *node[V]) isEmpty() bool {
	return n.prefixes.Count() == 0 && n.children.Count() == 0
}

// lpm does a route lookup for idx in the 8-bit (stride) routing table
// at this depth and returns (baseIdx, value, true) if a matching
// longest prefix exists, or ok=false otherwise.
//
// backtracking is fast, it's just a bitset test and, if found, one popcount.
// max steps in backtracking is the stride length.
func (n *node[V]) lpm(idx uint) (baseIdx uint, val V, ok bool) {
	// backtracking the CBT, make it as fast as possible
	for baseIdx = idx; baseIdx > 0; baseIdx >>= 1 {
		// practically it's get, but get is not inlined
		if n.prefixes.BitSet.Test(baseIdx) {
			return baseIdx, n.prefixes.MustGet(baseIdx), true
		}
	}

	// not found (on this level)
	return 0, val, false
}

// lpmTest for faster lpm tests without value returns
func (n *node[V]) lpmTest(idx uint) bool {
	// backtracking the CBT
	for idx := idx; idx > 0; idx >>= 1 {
		if n.prefixes.BitSet.Test(idx) {
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
	bits int,
	yield func(netip.Prefix, V) bool,
) bool {
	// backtracking the CBT
	for idx := pfxToIdx(octet, bits); idx > 0; idx >>= 1 {
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
	pfxLastAddr := uint(octet | ^netMask[pfxLen])

	idxBackingArray := [maxNodePrefixes]uint{}
	allCoveredIndices := idxBackingArray[:0]

	var idx uint

	var ok bool

	for {
		if idx, ok = n.prefixes.BitSet.NextSet(idx); !ok {
			break
		}

		// idx is covered by prefix
		thisOctet, thisPfxLen := idxToPfx(idx)

		thisFirstAddr := uint(thisOctet)
		thisLastAddr := uint(thisOctet | ^netMask[thisPfxLen])

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

	addrBackingArray := [maxNodeChildren]uint{}
	allCoveredAddrs := addrBackingArray[:0]

	var addr uint

	for {
		if addr, ok = n.children.BitSet.NextSet(addr); !ok {
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
	// make backing array, no heap allocs
	idxBacking := make([]uint, maxNodePrefixes)

	// for all prefixes in other node do ...
	for i, oIdx := range o.prefixes.AllSetBits(idxBacking) {
		// insert/overwrite prefix/value from oNode to nNode
		ok := n.prefixes.InsertAt(oIdx, o.prefixes.Items[i])

		// this prefix is duplicate in n and o
		if !ok {
			duplicates++
		}
	}

	// make backing array, no heap allocs
	addrBacking := make([]uint, maxNodeChildren)

	// for all children in other node do ...
	for i, oOctet := range o.children.AllSetBits(addrBacking) {
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
	c := newNode[V]()
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
	idxBacking := make([]uint, maxNodePrefixes)
	// for all prefixes in this node do ...
	for _, idx := range n.prefixes.AllSetBits(idxBacking) {
		cidr, _ := cidrFromPath(path, depth, is4, idx)

		// make the callback for this prefix
		if !yield(cidr, n.prefixes.MustGet(idx)) {
			// early exit
			return false
		}
	}

	addrBacking := make([]uint, maxNodeChildren)
	// for all children in this node do ...
	for i, addr := range n.children.AllSetBits(addrBacking) {
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
	// make backing arrays, no heap allocs
	addrBacking := make([]uint, maxNodeChildren)
	idxBacking := make([]uint, maxNodePrefixes)

	// get slice of all child octets, sorted by addr
	childAddrs := n.children.AllSetBits(addrBacking)

	// get slice of all indexes, sorted by idx
	allIndices := n.prefixes.AllSetBits(idxBacking)

	// sort indices in CIDR sort order
	slices.SortFunc(allIndices, cmpIndexRank)

	childCursor := 0

	// yield indices and childs in CIDR sort order
	for _, idx := range allIndices {
		octet, _ := idxToPfx(idx)

		// yield all childs before idx
		for j := childCursor; j < len(childAddrs); j++ {
			addr := childAddrs[j]

			if addr >= uint(octet) {
				break
			}

			// yield the child for this addr
			c := n.children.Items[j]

			// add (set) this octet to path
			path[depth] = byte(addr)

			// all cidrs under this child are covered by pfx
			if !c.allRecSorted(path, depth+1, is4, yield) {
				// early exit
				return false
			}

			childCursor++
		}

		// yield the prefix for this idx
		cidr, _ := cidrFromPath(path, depth, is4, idx)
		if !yield(cidr, n.prefixes.MustGet(idx)) {
			// early exit
			return false
		}
	}

	// yield the rest of childs, if any
	for j := childCursor; j < len(childAddrs); j++ {
		addr := childAddrs[j]
		c := n.children.Items[j]

		path[depth] = byte(addr)
		if !c.allRecSorted(path, depth+1, is4, yield) {
			// early exit
			return false
		}
	}

	return true
}
