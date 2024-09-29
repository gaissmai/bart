// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"slices"

	"github.com/bits-and-blooms/bitset"
)

const (
	strideLen       = 8                    // octet
	maxTreeDepth    = 128 / strideLen      // 16
	maxNodeChildren = 1 << strideLen       // 256
	maxNodePrefixes = 1 << (strideLen + 1) // 512
)

// zero value, used manifold
var zeroPath [16]byte

// node is a level node in the multibit-trie.
// A node has prefixes and children, forming the multibit trie.
//
// The prefixes form a complete binary tree, see the artlookup.pdf
// paper in the doc folder to understand the data structure.
//
// In contrast to the ART algorithm, popcount-compressed slices are used
// instead of fixed-size arrays.
//
// The array slots are also not pre-allocated as in the ART algorithm,
// but backtracking is used for the longest-prefix-match.
//
// The lookup is then slower by a factor of about 2, but this is
// the intended trade-off to prevent memory consumption from exploding.
type node[V any] struct {
	// prefixes contains the payload V
	prefixes []V

	// children, recursively spans the trie with a branching factor of 256
	children []*node[V]

	// Here we would be done if they were fixed arrays, but since they
	// are popcount compressed slices we need bitsets.
	// ---
	// To address a specific element in prefixes or children
	// the popcount of the bitset is calculated up to the desired element,
	// this gives the position of the element in the corresponding slice.
	//
	// e.g. find the value V for prefix 10/7:
	//  pfxToIdx(10/7) -> 133; popcount(133) -> i; V = prefixes[i]
	//
	// e.g. find the next node for octet(253):
	//  popcount(253) -> i; *n = children[i]
	//
	prefixesBitset *bitset.BitSet
	childrenBitset *bitset.BitSet
}

// newNode, the zero-value of BitSet is ready to use,
// not using bitset.New(), this would be not inlineable.
func newNode[V any]() *node[V] {
	return &node[V]{
		prefixesBitset: &bitset.BitSet{},
		childrenBitset: &bitset.BitSet{},
	}
}

// isEmpty returns true if node has neither prefixes nor children.
func (n *node[V]) isEmpty() bool {
	return len(n.prefixes) == 0 && len(n.children) == 0
}

// ################## prefixes ################################

// prefixRank, Rank() is the key of the popcount compression algorithm,
// mapping between bitset index and slice index.
func (n *node[V]) prefixRank(idx uint) int {
	// adjust offset by one to slice index
	return int(n.prefixesBitset.Rank(idx)) - 1
}

// insertPrefix adds the route as baseIdx, with value val.
// If the value already exists, overwrite it with val and return false.
func (n *node[V]) insertPrefix(idx uint, val V) (ok bool) {
	// prefix exists, overwrite val
	if n.prefixesBitset.Test(idx) {
		n.prefixes[n.prefixRank(idx)] = val

		return false
	}

	// new, insert into bitset and slice
	n.prefixesBitset.Set(idx)
	n.prefixes = slices.Insert(n.prefixes, n.prefixRank(idx), val)

	return true
}

// deletePrefix removes the route octet/prefixLen and returns the associated value and true
// or false if there was no prefix to delete (and no value to return).
func (n *node[V]) deletePrefix(octet byte, prefixLen int) (val V, ok bool) {
	idx := pfxToIdx(octet, prefixLen)

	// no route entry
	if !n.prefixesBitset.Test(idx) {
		return val, false
	}

	rnk := n.prefixRank(idx)
	val = n.prefixes[rnk]

	// delete from slice
	n.prefixes = slices.Delete(n.prefixes, rnk, rnk+1)

	// delete from bitset, followed by Compact to reduce memory consumption
	n.prefixesBitset.Clear(idx)
	n.prefixesBitset.Compact()

	return val, true
}

// updatePrefix, update or set the value at prefix via callback. The new value returned
// and a bool wether the prefix was already present in the node.
func (n *node[V]) updatePrefix(octet byte, prefixLen int, cb func(V, bool) V) (newVal V, wasPresent bool) {
	// calculate idx once
	idx := pfxToIdx(octet, prefixLen)

	var rnk int

	// if prefix is set, get current value
	var oldVal V

	if wasPresent = n.prefixesBitset.Test(idx); wasPresent {
		rnk = n.prefixRank(idx)
		oldVal = n.prefixes[rnk]
	}

	// callback function to get updated or new value
	newVal = cb(oldVal, wasPresent)

	// prefix is already set, update and return value
	if wasPresent {
		n.prefixes[rnk] = newVal

		return
	}

	// new prefix, insert into bitset ...
	n.prefixesBitset.Set(idx)

	// bitset has changed, recalc rank
	rnk = n.prefixRank(idx)

	// ... and insert value into slice
	n.prefixes = slices.Insert(n.prefixes, rnk, newVal)

	return
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
		// practically it's getValueOK, but getValueOK is not inlined
		if n.prefixesBitset.Test(baseIdx) {
			return baseIdx, n.prefixes[n.prefixRank(baseIdx)], true
		}
	}

	// not found (on this level)
	return 0, val, false
}

// lpmTest for faster lpm tests without value returns
func (n *node[V]) lpmTest(idx uint) bool {
	// backtracking the CBT
	for idx := idx; idx > 0; idx >>= 1 {
		if n.prefixesBitset.Test(idx) {
			return true
		}
	}

	return false
}

// getValueOK for idx..
func (n *node[V]) getValueOK(idx uint) (val V, ok bool) {
	if n.prefixesBitset.Test(idx) {
		return n.prefixes[n.prefixRank(idx)], true
	}

	return
}

// mustGetValue for idx, use it only after a successful bitset test.
// n.prefixesBitset.Test(idx) must be true
func (n *node[V]) mustGetValue(idx uint) V {
	return n.prefixes[n.prefixRank(idx)]
}

// allStrideIndexes returns all baseIndexes set in this stride node in ascending order.
func (n *node[V]) allStrideIndexes(buffer []uint) []uint {
	_, buffer = n.prefixesBitset.NextSetMany(0, buffer)

	return buffer
}

// ################## children ################################

// childRank, Rank() is the key of the popcount compression algorithm,
// mapping between bitset index and slice index.
func (n *node[V]) childRank(octet byte) int {
	// adjust offset by one to slice index
	return int(n.childrenBitset.Rank(uint(octet))) - 1
}

// insertChild, insert the child
func (n *node[V]) insertChild(octet byte, child *node[V]) {
	// child exists, overwrite it
	if n.childrenBitset.Test(uint(octet)) {
		n.children[n.childRank(octet)] = child

		return
	}

	// new insert into bitset and slice
	n.childrenBitset.Set(uint(octet))
	n.children = slices.Insert(n.children, n.childRank(octet), child)
}

// deleteChild, delete the child at octet. It is valid to delete a non-existent child.
func (n *node[V]) deleteChild(octet byte) {
	if !n.childrenBitset.Test(uint(octet)) {
		return
	}

	rnk := n.childRank(octet)

	// delete from slice
	n.children = slices.Delete(n.children, rnk, rnk+1)

	// delete from bitset, followed by Compact to reduce memory consumption
	n.childrenBitset.Clear(uint(octet))
	n.childrenBitset.Compact()
}

// getChild returns the child pointer for octet, or nil if none.
func (n *node[V]) getChild(octet byte) *node[V] {
	if !n.childrenBitset.Test(uint(octet)) {
		return nil
	}

	return n.children[n.childRank(octet)]
}

// allChildAddrs fills the buffer with the octets of all child nodes in ascending order,
// panics if the buffer isn't big enough.
func (n *node[V]) allChildAddrs(buffer []uint) []uint {
	_, buffer = n.childrenBitset.NextSetMany(0, buffer)

	return buffer
}

// #################### nodes #############################################

// eachLookupPrefix does an all prefix match in the 8-bit (stride) routing table
// at this depth and calls yield() for any matching CIDR.
func (n *node[V]) eachLookupPrefix(path [16]byte, depth int, is4 bool, octet byte, bits int, yield func(netip.Prefix, V) bool) bool {
	// backtracking the CBT
	for idx := pfxToIdx(octet, bits); idx > 0; idx >>= 1 {
		if val, ok := n.getValueOK(idx); ok {
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
func (n *node[V]) eachSubnet(path [16]byte, depth int, is4 bool, octet byte, pfxLen int, yield func(netip.Prefix, V) bool) bool {
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
		if idx, ok = n.prefixesBitset.NextSet(idx); !ok {
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
		if addr, ok = n.childrenBitset.NextSet(addr); !ok {
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
			c := n.getChild(octet)

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
		if !yield(cidr, n.mustGetValue(idx)) {
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
		c := n.getChild(octet)

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
	for i, oIdx := range o.allStrideIndexes(idxBacking) {
		// insert/overwrite prefix/value from oNode to nNode
		ok := n.insertPrefix(oIdx, o.prefixes[i])

		// this prefix is duplicate in n and o
		if !ok {
			duplicates++
		}
	}

	// make backing array, no heap allocs
	addrBacking := make([]uint, maxNodeChildren)

	// for all children in other node do ...
	for i, oOctet := range o.allChildAddrs(addrBacking) {
		octet := byte(oOctet)

		// we know the slice index, faster as o.getChild(octet)
		oc := o.children[i]

		// get n child with same octet,
		// we don't know the slice index in n.children
		nc := n.getChild(octet)

		if nc == nil {
			// insert cloned child from oNode into nNode
			n.insertChild(octet, oc.cloneRec())
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

	c.prefixesBitset = n.prefixesBitset.Clone() // deep
	c.prefixes = slices.Clone(n.prefixes)       // values, shallow copy

	// deep copy if V implements Cloner[V]
	for i, v := range c.prefixes {
		if v, ok := any(v).(Cloner[V]); ok {
			c.prefixes[i] = v.Clone()
		} else {
			break
		}
	}

	c.childrenBitset = n.childrenBitset.Clone() // deep
	c.children = slices.Clone(n.children)       // children, shallow copy

	// deep copy of children
	for i, child := range c.children {
		c.children[i] = child.cloneRec()
	}

	return c
}

// allRec runs recursive the trie, starting at this node and
// the yield function is called for each route entry with prefix and value.
// If the yield function returns false the recursion ends prematurely and the
// false value is propagated.
//
// The iteration order is not defined, just the simplest and fastest recursive implementation.
func (n *node[V]) allRec(path [16]byte, depth int, is4 bool, yield func(netip.Prefix, V) bool) bool {
	idxBacking := make([]uint, maxNodePrefixes)
	// for all prefixes in this node do ...
	for _, idx := range n.allStrideIndexes(idxBacking) {
		cidr, _ := cidrFromPath(path, depth, is4, idx)

		// make the callback for this prefix
		if !yield(cidr, n.mustGetValue(idx)) {
			// early exit
			return false
		}
	}

	addrBacking := make([]uint, maxNodeChildren)
	// for all children in this node do ...
	for i, addr := range n.allChildAddrs(addrBacking) {
		child := n.children[i]
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
func (n *node[V]) allRecSorted(path [16]byte, depth int, is4 bool, yield func(netip.Prefix, V) bool) bool {
	// make backing arrays, no heap allocs
	addrBacking := make([]uint, maxNodeChildren)
	idxBacking := make([]uint, maxNodePrefixes)

	// get slice of all child octets, sorted by addr
	childAddrs := n.allChildAddrs(addrBacking)

	// get slice of all indexes, sorted by idx
	allIndices := n.allStrideIndexes(idxBacking)

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
			c := n.children[j]

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
		if !yield(cidr, n.mustGetValue(idx)) {
			// early exit
			return false
		}
	}

	// yield the rest of childs, if any
	for j := childCursor; j < len(childAddrs); j++ {
		addr := childAddrs[j]
		c := n.children[j]

		path[depth] = byte(addr)
		if !c.allRecSorted(path, depth+1, is4, yield) {
			// early exit
			return false
		}
	}

	return true
}
