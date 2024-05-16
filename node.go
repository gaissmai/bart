// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"cmp"
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

// node is a level node in the multibit-trie.
// A node has prefixes and children.
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
	prefixesBitset *bitset.BitSet
	childrenBitset *bitset.BitSet

	// popcount compressed slices
	prefixes []V
	children []*node[V]
}

// newNode, BitSets have to be initialized.
func newNode[V any]() *node[V] {
	return &node[V]{
		prefixesBitset: bitset.New(0), // init BitSet
		childrenBitset: bitset.New(0), // init BitSet
	}
}

// isEmpty returns true if node has neither prefixes nor children.
func (n *node[V]) isEmpty() bool {
	return len(n.prefixes) == 0 && len(n.children) == 0
}

// ################## prefixes ################################

// prefixRank, Rank() is the key of the popcount compression algorithm,
// mapping between bitset index and slice index.
func (n *node[V]) prefixRank(baseIdx uint) int {
	// adjust offset by one to slice index
	return int(n.prefixesBitset.Rank(baseIdx)) - 1
}

// insertPrefix adds the route octet/prefixLen, with value val.
// Just an adapter for insertIdx.
func (n *node[V]) insertPrefix(octet byte, prefixLen int, val V) {
	n.insertIdx(prefixToBaseIndex(octet, prefixLen), val)
}

// insertIdx adds the route for baseIdx, with value val.
// incSize reports if the sie counter must incremented.
func (n *node[V]) insertIdx(baseIdx uint, val V) {
	// prefix exists, overwrite val
	if n.prefixesBitset.Test(baseIdx) {
		n.prefixes[n.prefixRank(baseIdx)] = val
		return
	}

	// new, insert into bitset and slice
	n.prefixesBitset.Set(baseIdx)
	n.prefixes = slices.Insert(n.prefixes, n.prefixRank(baseIdx), val)
}

// deletePrefix removes the route octet/prefixLen. Reports whether the
// prefix existed in the table prior to deletion.
func (n *node[V]) deletePrefix(octet byte, prefixLen int) (wasPresent bool) {
	baseIdx := prefixToBaseIndex(octet, prefixLen)

	// no route entry
	if !n.prefixesBitset.Test(baseIdx) {
		return false
	}

	rnk := n.prefixRank(baseIdx)

	// delete from slice
	n.prefixes = slices.Delete(n.prefixes, rnk, rnk+1)

	// delete from bitset, followed by Compact to reduce memory consumption
	n.prefixesBitset.Clear(baseIdx)
	n.prefixesBitset.Compact()

	return true
}

// updatePrefix, update or set the value at prefix via callback.
func (n *node[V]) updatePrefix(octet byte, prefixLen int, cb func(V, bool) V) (val V) {
	// calculate idx once
	baseIdx := prefixToBaseIndex(octet, prefixLen)

	var ok bool
	var rnk int

	// if prefix is set, get current value
	if ok = n.prefixesBitset.Test(baseIdx); ok {
		rnk = n.prefixRank(baseIdx)
		val = n.prefixes[rnk]
	}

	// callback function to get updated or new value
	val = cb(val, ok)

	// prefix is already set, update and return value
	if ok {
		n.prefixes[rnk] = val
		return
	}

	// new prefix, insert into bitset ...
	n.prefixesBitset.Set(baseIdx)

	// bitset has changed, recalc rank
	rnk = n.prefixRank(baseIdx)

	// ... and insert value into slice
	n.prefixes = slices.Insert(n.prefixes, rnk, val)

	return
}

// lpmByIndex does a route lookup for idx in the 8-bit (stride) routing table
// at this depth and returns (baseIdx, value, true) if a matching
// longest prefix exists, or ok=false otherwise.
//
// backtracking is fast, it's just a bitset test and, if found, one popcount.
func (n *node[V]) lpmByIndex(idx uint) (baseIdx uint, val V, ok bool) {
	// max steps in backtracking is the stride length.
	for {
		if n.prefixesBitset.Test(idx) {
			// longest prefix match
			return idx, n.prefixes[n.prefixRank(idx)], true
		}

		if idx == 0 {
			break
		}

		// cache friendly backtracking to the next less specific route.
		// thanks to the complete binary tree it's just a shift operation.
		idx >>= 1
	}

	// not found (on this level)
	return 0, val, false
}

// lpmByOctet is an adapter to lpmByIndex.
func (n *node[V]) lpmByOctet(octet byte) (baseIdx uint, val V, ok bool) {
	return n.lpmByIndex(octetToBaseIndex(octet))
}

// lpmByPrefix is an adapter to lpmByIndex.
func (n *node[V]) lpmByPrefix(octet byte, bits int) (baseIdx uint, val V, ok bool) {
	return n.lpmByIndex(prefixToBaseIndex(octet, bits))
}

// getValByIndex for baseIdx.
func (n *node[V]) getValByIndex(baseIdx uint) (val V, ok bool) {
	if n.prefixesBitset.Test(baseIdx) {
		return n.prefixes[n.prefixRank(baseIdx)], true
	}
	return
}

// getValByPrefix, adapter for getValByIndex.
func (n *node[V]) getValByPrefix(octet byte, bits int) (val V, ok bool) {
	return n.getValByIndex(prefixToBaseIndex(octet, bits))
}

// apmByOctet is an adapter for apmByPrefix.
func (n *node[V]) apmByOctet(octet byte, depth int, ip netip.Addr) (result []netip.Prefix) {
	return n.apmByPrefix(octet, strideLen, depth, ip)
}

// apmByPrefix does an all prefix match in the 8-bit (stride) routing table
// at this depth and returns all matching CIDRs.
func (n *node[V]) apmByPrefix(octet byte, bits int, depth int, ip netip.Addr) (result []netip.Prefix) {
	// skip intermediate nodes
	if len(n.prefixes) == 0 {
		return
	}

	var superIdxs []uint
	baseIdx := prefixToBaseIndex(octet, bits)
	for {
		if n.prefixesBitset.Test(baseIdx) {
			superIdxs = append(superIdxs, baseIdx)
		}

		if baseIdx == 0 {
			break
		}

		// cache friendly backtracking to the next less specific route.
		// thanks to the complete binary tree it's just a shift operation.
		baseIdx >>= 1
	}

	// sort baseIndexes in ascending order
	slices.Sort(superIdxs)

	// make CIDRs
	for _, baseIdx := range superIdxs {
		superPfx, _ := ip.Prefix(baseIndexToPrefixMask(baseIdx, depth))
		result = append(result, superPfx)
	}

	return result
}

// allStrideIndexes returns all baseIndexes set in this stride node in ascending order.
func (n *node[V]) allStrideIndexes() []uint {
	all := make([]uint, 0, maxNodePrefixes)
	_, all = n.prefixesBitset.NextSetMany(0, all)
	return all
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

// allChildAddrs returns the octets of all child nodes in ascending order.
func (n *node[V]) allChildAddrs() []uint {
	all := make([]uint, maxNodeChildren)
	_, all = n.childrenBitset.NextSetMany(0, all)
	return all
}

// #################### nodes #############################################

// overlapsRec returns true if any IP in the nodes n or o overlaps.
// First test the routes, then the children and if no match rec-descent
// for child nodes with same octet.
func (n *node[V]) overlapsRec(o *node[V]) bool {
	// dynamically allot the host routes from prefixes
	nAllotIndex := [maxNodePrefixes]bool{}
	oAllotIndex := [maxNodePrefixes]bool{}

	// 1. test if any routes overlaps?

	nPfxExists := len(n.prefixes) > 0
	oPfxExists := len(o.prefixes) > 0
	var nIdx, oIdx uint

	// zig-zag, for all routes in both nodes ...
	for {
		if nPfxExists {
			// range over bitset, node n
			if nIdx, nPfxExists = n.prefixesBitset.NextSet(nIdx); nPfxExists {
				// get range of host routes for this prefix
				lowerBound, upperBound := lowerUpperBound(nIdx)

				// insert host routes (octet/8) for this prefix,
				// some sort of allotment
				for i := lowerBound; i <= upperBound; i++ {
					// zig-zag, fast return
					if oAllotIndex[i] {
						return true
					}
					nAllotIndex[i] = true
				}
				nIdx++
			}
		}

		if oPfxExists {
			// range over bitset, node o
			if oIdx, oPfxExists = o.prefixesBitset.NextSet(oIdx); oPfxExists {
				// get range of host routes for this prefix
				lowerBound, upperBound := lowerUpperBound(oIdx)

				// insert host routes (octet/8) for this prefix,
				// some sort of allotment
				for i := lowerBound; i <= upperBound; i++ {
					// zig-zag, fast return
					if nAllotIndex[i] {
						return true
					}
					oAllotIndex[i] = true
				}
				oIdx++
			}
		}
		if !nPfxExists && !oPfxExists {
			break
		}
	}

	// full run, zig-zag didn't already match
	if len(n.prefixes) > 0 && len(o.prefixes) > 0 {
		for i := firstHostIndex; i <= lastHostIndex; i++ {
			if nAllotIndex[i] && oAllotIndex[i] {
				return true
			}
		}
	}

	// 2. test if routes overlaps any child

	nOctets := [maxNodeChildren]bool{}
	oOctets := [maxNodeChildren]bool{}

	ncExists := len(n.children) > 0
	ocExists := len(o.children) > 0
	var nOctet, oOctet uint

	// zig-zag, for all octets in both nodes ...
	for {
		// range over bitset, node n
		if ncExists {
			if nOctet, ncExists = n.childrenBitset.NextSet(nOctet); ncExists {
				if oAllotIndex[nOctet+firstHostIndex] {
					return true
				}
				nOctets[nOctet] = true
				nOctet++
			}
		}

		// range over bitset, node o
		if ocExists {
			if oOctet, ocExists = o.childrenBitset.NextSet(oOctet); ocExists {
				if nAllotIndex[oOctet+firstHostIndex] {
					return true
				}
				oOctets[oOctet] = true
				oOctet++
			}
		}

		if !ncExists && !ocExists {
			break
		}
	}

	// 3. rec-descent call for childs with same octet

	if len(n.children) > 0 && len(o.children) > 0 {
		for i := 0; i < len(nOctets); i++ {
			if nOctets[i] && oOctets[i] {
				// get next child node for this octet
				nc := n.getChild(byte(i))
				oc := o.getChild(byte(i))

				// rec-descent
				if nc.overlapsRec(oc) {
					return true
				}
			}
		}
	}

	return false
}

// overlapsPrefix returns true if node overlaps with prefix.
func (n *node[V]) overlapsPrefix(octet byte, pfxLen int) bool {
	// ##################################################
	// 1. test if any route in this node overlaps prefix?

	pfxIdx := prefixToBaseIndex(octet, pfxLen)
	if _, _, ok := n.lpmByIndex(pfxIdx); ok {
		return true
	}

	// #################################################
	// 2. test if prefix overlaps any route in this node

	// lower/upper boundary for host routes
	pfxLowerBound, pfxUpperBound := lowerUpperBound(pfxIdx)

	// increment to 'next' routeIdx for start in bitset search
	// since pfxIdx already testet by lpm in other direction
	routeIdx := pfxIdx * 2
	var ok bool
	for {
		if routeIdx, ok = n.prefixesBitset.NextSet(routeIdx); !ok {
			break
		}

		routeLowerBound, routeUpperBound := lowerUpperBound(routeIdx)
		if routeLowerBound >= pfxLowerBound && routeUpperBound <= pfxUpperBound {
			return true
		}

		// next route
		routeIdx++
	}

	// #################################################
	// 3. test if prefix overlaps any child in this node

	// set start octet in bitset search with prefix octet
	cOctet := uint(octet)
	for {
		if cOctet, ok = n.childrenBitset.NextSet(cOctet); !ok {
			break
		}

		cIdx := cOctet + firstHostIndex
		if cIdx >= pfxLowerBound && cIdx <= pfxUpperBound {
			return true
		}

		// next round
		cOctet++
	}

	return false
}

// unionRec combines two nodes, changing the receiver node.
// If there are duplicate entries, the value is taken from the other node.
func (n *node[V]) unionRec(o *node[V]) {
	// for all prefixes in other node do ...
	for _, oIdx := range o.allStrideIndexes() {
		// insert/overwrite prefix/value from oNode to nNode
		oVal, _ := o.getValByIndex(oIdx)
		n.insertIdx(oIdx, oVal)
	}

	// for all children in other node do ...
	for _, oOctet := range o.allChildAddrs() {
		octet := byte(oOctet)

		// get other child for this octet
		oc := o.getChild(octet)

		// get n child with same octet
		nc := n.getChild(octet)

		if nc == nil {
			// insert cloned child from oNode into nNode
			n.insertChild(octet, oc.cloneRec())
		} else {
			// both nodes have child with octet, call union rec-descent
			nc.unionRec(oc)
		}
	}
}

// cloneRec, clones the node recursive.
func (n *node[V]) cloneRec() *node[V] {
	c := newNode[V]()
	if n.isEmpty() {
		return c
	}

	c.prefixesBitset = n.prefixesBitset.Clone() // deep
	c.prefixes = slices.Clone(n.prefixes)       // shallow values

	c.childrenBitset = n.childrenBitset.Clone() // deep
	c.children = slices.Clone(n.children)       // shallow

	// now clone the children deep
	for i, child := range c.children {
		c.children[i] = child.cloneRec()
	}

	return c
}

// allRec runs recursive the trie, starting at node and
// the yield function is called for each route entry with prefix and value.
// If the yield function returns false the recursion ends prematurely and the
// false value is propagated.
func (n *node[V]) allRec(path []byte, is4 bool, yield func(netip.Prefix, V) bool) bool {
	// for all prefixes in this node do ...
	for _, idx := range n.allStrideIndexes() {
		val, _ := n.getValByIndex(idx)
		pfx := cidrFromPath(path, idx, is4)

		// make the callback for this prefix
		if !yield(pfx, val) {
			// premature end of recursion
			return false
		}
	}

	// for all children in this node do ...
	for _, addr := range n.allChildAddrs() {
		octet := byte(addr)
		path := append(slices.Clone(path), octet)
		child := n.getChild(octet)

		if !child.allRec(path, is4, yield) {
			// premature end of recursion
			return false
		}
	}

	return true
}

// subnets returns all CIDRs covered by parent pfx.
func (n *node[V]) subnets(path []byte, pfxOctet byte, pfxLen int, is4 bool) (result []netip.Prefix) {
	parentIdx := prefixToBaseIndex(pfxOctet, pfxLen)

	// collect all routes covered by this pfx
	// see also algorithm in overlapsPrefix

	// lower/upper boundary for octet/pfxLen host routes
	pfxLowerBound, pfxUpperBound := lowerUpperBound(parentIdx)

	// start in bitset search at parentIdx
	idx := parentIdx
	var ok bool
	for {
		if idx, ok = n.prefixesBitset.NextSet(idx); !ok {
			// no more prefixes in this node
			break
		}

		routeLowerBound, routeUpperBound := lowerUpperBound(idx)
		if routeLowerBound >= pfxLowerBound && routeUpperBound <= pfxUpperBound {
			// get CIDR back for this idx
			pfx := cidrFromPath(path, idx, is4)
			result = append(result, pfx)
		}

		// next prefix idx
		idx++
	}

	// collect all children covered by this pfx
	// see also algorithm in overlapsPrefix

	// set start octet in bitset search with prefix octet
	cOctet := uint(pfxOctet)
	for {
		if cOctet, ok = n.childrenBitset.NextSet(cOctet); !ok {
			// no more children
			break
		}

		cIdx := cOctet + firstHostIndex

		if cIdx >= pfxLowerBound && cIdx <= pfxUpperBound {
			// pfx covers child
			c := n.getChild(byte(cOctet))

			// append octet to path
			path := append(slices.Clone(path), byte(cOctet))

			// all cidrs under this child are covered by pfx
			c.allRec(path, is4, func(pfx netip.Prefix, _ V) bool {
				result = append(result, pfx)
				return true
			})
		}

		// next round
		cOctet++
	}

	return result
}

// cmpPrefix, all cidrs are normalized
func cmpPrefix(a, b netip.Prefix) int {
	if cmp := a.Addr().Compare(b.Addr()); cmp != 0 {
		return cmp
	}
	return cmp.Compare(a.Bits(), b.Bits())
}

// numPrefixesRec, calculate the number of prefixes under n.
func (n *node[V]) numPrefixesRec() int {
	size := len(n.prefixes) // this node
	for _, c := range n.children {
		size += c.numPrefixesRec()
	}
	return size
}

// numNodesRec, calculate the number of nodes under n.
func (n *node[V]) numNodesRec() int {
	size := 1 // this node
	for _, c := range n.children {
		size += c.numNodesRec()
	}
	return size
}
