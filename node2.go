// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"bytes"
	"net/netip"
	"slices"

	"github.com/bits-and-blooms/bitset"
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
type node2[V any] struct {
	// BitSets for popcount compression
	prefixesBitset *bitset.BitSet
	childrenBitset *bitset.BitSet

	// popcount compressed slices
	prefixes []V
	children []*node2[V]

	// for path compression
	path pathT

	// IP version of this node
	is4 bool
}

// pathT, for path compression.
// Use array and length instead of slice, 17 bytes instead of 40 bytes (24 + <= 16 bytes))
type pathT struct {
	octets [16]byte
	length uint8
}

// newNode2, BitSets and path have to be initialized.
func newNode2[V any](path []byte, is4 bool) *node2[V] {
	n := &node2[V]{
		prefixesBitset: bitset.New(0), // init BitSet
		childrenBitset: bitset.New(0), // init BitSet
	}

	copy(n.path.octets[:], path)
	n.path.length = uint8(len(path))

	n.is4 = is4
	return n
}

func (n *node2[V]) pathAsSlice() (octets []byte) {
	return n.path.octets[:n.path.length]
}

func (n *node2[V]) pathLen() int {
	return int(n.path.length)
}

func (n *node2[V]) pathIsPrefixOrEqual(start int, buf []byte) bool {
	return bytes.HasPrefix(buf[start:], n.pathAsSlice()[start:])
}

// commonPathIdx, return idx for last in common byte.
// Start is the idx we know already they are in common (time optimization).
func (n *node2[V]) commonPathIdx(start int, o *node2[V]) int {
	idx := start
	for i := start; i < min(n.pathLen(), o.pathLen()); i++ {
		if n.path.octets[i] != o.path.octets[i] {
			return idx
		}
		idx = i
	}
	return idx
}

// isEmpty returns true if node has neither prefixes nor children.
func (n *node2[V]) isEmpty() bool {
	return len(n.prefixes) == 0 && len(n.children) == 0
}

// ################## prefixes ################################

// prefixRank, Rank() is the key of the popcount compression algorithm,
// mapping between bitset index and slice index.
func (n *node2[V]) prefixRank(baseIdx uint) int {
	// adjust offset by one to slice index
	return int(n.prefixesBitset.Rank(baseIdx)) - 1
}

// insertPrefix adds the route octet/prefixLen, with value val.
// Just an adapter for insertIdx.
func (n *node2[V]) insertPrefix(octet byte, prefixLen int, val V) (wasPresent bool) {
	return n.insertIdx(prefixToBaseIndex(octet, prefixLen), val)
}

// insertIdx adds the route for baseIdx, with value val.
func (n *node2[V]) insertIdx(baseIdx uint, val V) (wasPresent bool) {
	// prefix exists, overwrite val
	if n.prefixesBitset.Test(baseIdx) {
		n.prefixes[n.prefixRank(baseIdx)] = val
		return true
	}

	// new, insert into bitset and slice
	n.prefixesBitset.Set(baseIdx)
	n.prefixes = slices.Insert(n.prefixes, n.prefixRank(baseIdx), val)
	return false
}

// deletePrefix removes the route octet/prefixLen. Reports whether the
// prefix existed in the table prior to deletion.
func (n *node2[V]) deletePrefix(octet byte, prefixLen int) (wasPresent bool) {
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
func (n *node2[V]) updatePrefix(octet byte, prefixLen int, cb func(V, bool) V) (val V, wasPresent bool) {
	// calculate idx once
	baseIdx := prefixToBaseIndex(octet, prefixLen)

	var rnk int

	// if prefix is set, get current value
	if wasPresent = n.prefixesBitset.Test(baseIdx); wasPresent {
		rnk = n.prefixRank(baseIdx)
		val = n.prefixes[rnk]
	}

	// callback function to get updated or new value
	val = cb(val, wasPresent)

	// prefix is already set, update and return value
	if wasPresent {
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
func (n *node2[V]) lpmByIndex(idx uint) (baseIdx uint, val V, ok bool) {
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
func (n *node2[V]) lpmByOctet(octet byte) (baseIdx uint, val V, ok bool) {
	return n.lpmByIndex(octetToBaseIndex(octet))
}

// lpmByPrefix is an adapter to lpmByIndex.
func (n *node2[V]) lpmByPrefix(octet byte, bits int) (baseIdx uint, val V, ok bool) {
	return n.lpmByIndex(prefixToBaseIndex(octet, bits))
}

// getValByIndex for baseIdx.
func (n *node2[V]) getValByIndex(baseIdx uint) (val V, ok bool) {
	if n.prefixesBitset.Test(baseIdx) {
		return n.prefixes[n.prefixRank(baseIdx)], true
	}
	return
}

// getValByPrefix, adapter for getValByIndex.
func (n *node2[V]) getValByPrefix(octet byte, bits int) (val V, ok bool) {
	return n.getValByIndex(prefixToBaseIndex(octet, bits))
}

// apmByPrefix does an all prefix match in the 8-bit (stride) routing table
// at this depth and returns all matching CIDRs.
func (n *node2[V]) apmByPrefix(octet byte, bits int) []netip.Prefix {
	// skip intermediate nodes
	if len(n.prefixes) == 0 {
		return nil
	}

	var super []uint

	idx := prefixToBaseIndex(octet, bits)
	for {
		if n.prefixesBitset.Test(idx) {
			super = append(super, idx)
		}

		if idx == 0 {
			break
		}

		// cache friendly backtracking to the next less specific route.
		// thanks to the complete binary tree it's just a shift operation.
		idx >>= 1
	}

	// sort indices in ascending order
	slices.Sort(super)

	var result []netip.Prefix
	for _, baseIdx := range super {
		result = append(result, n.cidrFromIndex(baseIdx))
	}

	return result
}

// allStrideIndexes returns all baseIndexes set in this stride node in ascending order.
func (n *node2[V]) allStrideIndexes() []uint {
	all := make([]uint, 0, maxNodePrefixes)
	_, all = n.prefixesBitset.NextSetMany(0, all)
	return all
}

// ################## children ################################

// childRank, Rank() is the key of the popcount compression algorithm,
// mapping between bitset index and slice index.
func (n *node2[V]) childRank(octet byte) int {
	// adjust offset by one to slice index
	return int(n.childrenBitset.Rank(uint(octet))) - 1
}

// insertChild, insert the child
func (n *node2[V]) insertChild(octet byte, child *node2[V]) {
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
func (n *node2[V]) deleteChild(octet byte) {
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
func (n *node2[V]) getChild(octet byte) *node2[V] {
	if !n.childrenBitset.Test(uint(octet)) {
		return nil
	}

	return n.children[n.childRank(octet)]
}

// allChildAddrs returns the octets of all child nodes in ascending order.
func (n *node2[V]) allChildAddrs() []uint {
	all := make([]uint, maxNodeChildren)
	_, all = n.childrenBitset.NextSetMany(0, all)
	return all
}

// #################### nodes #############################################

// overlapsPrefix returns true if node overlaps with prefix.
func (n *node2[V]) overlapsPrefix(octet byte, pfxLen int) bool {
	// ##################################################
	// 1. test if any route in this node overlaps prefix?

	pfxIdx := prefixToBaseIndex(octet, pfxLen)
	if _, _, ok := n.lpmByIndex(pfxIdx); ok {
		return true
	}

	// #################################################
	// 2. test if prefix overlaps any route in this node

	// lower/upper boundary for octet/pfxLen host routes
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
	childOctet := uint(octet)
	for {
		if childOctet, ok = n.childrenBitset.NextSet(childOctet); !ok {
			break
		}

		childIdx := childOctet + firstHostIndex
		if childIdx >= pfxLowerBound && childIdx <= pfxUpperBound {
			return true
		}

		// next round
		childOctet++
	}

	return false
}

// cloneRec, clones the node recursive.
func (n *node2[V]) cloneRec() *node2[V] {
	c := newNode2[V](n.pathAsSlice(), n.is4)

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

// allRec2 runs recursive the trie, starting at node and
// the yield function is called for each route entry with prefix and value.
// If the yield function returns false the recursion ends prematurely and the
// false value is propagated.
func (n *node2[V]) allRec2(yield func(netip.Prefix, V) bool) bool {
	// for all prefixes in this node do ...
	for _, idx := range n.allStrideIndexes() {
		val, _ := n.getValByIndex(idx)
		pfx := n.cidrFromIndex(idx)

		// make the callback for this prefix
		if !yield(pfx, val) {
			// premature end of recursion
			return false
		}
	}

	// for all children in this node do ...
	for _, addr := range n.allChildAddrs() {
		octet := byte(addr)
		child := n.getChild(octet)

		if !child.allRec2(yield) {
			// premature end of recursion
			return false
		}
	}

	return true
}

// subnetsRec2 returns all CIDRs covered by parent pfx.
func (n *node2[V]) subnetsRec2(parentIdx uint) (result []netip.Prefix) {
	// for all routes in this node do ...
	for _, idx := range n.allStrideIndexes() {
		// is this route covered by pfx?
		for i := idx; i >= parentIdx; i >>= 1 {
			if i == parentIdx { // match
				pfx := n.cidrFromIndex(idx)
				result = append(result, pfx)
				break
			}
		}
	}

	// for all children in this node do ...
	for _, addr := range n.allChildAddrs() {
		octet := byte(addr)
		idx := octetToBaseIndex(octet)

		// is this child covered by pfx?
		for i := idx; i >= parentIdx; i >>= 1 {
			if i == parentIdx {
				// all CIDRs under this child are covered by pfx
				c := n.getChild(octet)
				c.allRec2(func(pfx netip.Prefix, _ V) bool {
					result = append(result, pfx)
					return true
				})
			}
		}
	}

	return result
}

// cidrFromIndex, make prefix from baseIndex.
func (n *node2[V]) cidrFromIndex(idx uint) (pfx netip.Prefix) {
	octet, pfxLen := baseIndexToPrefix(idx)

	// append last (partially) masked byte to path and
	// calc bits with pathLen and pfxLen
	octets := append(n.pathAsSlice(), octet)
	bits := n.pathLen()*strideLen + pfxLen

	// make ip addr from octets
	var ip netip.Addr
	if n.is4 {
		b4 := [4]byte{}
		copy(b4[:], octets)
		ip = netip.AddrFrom4(b4)
	} else {
		b16 := [16]byte{}
		copy(b16[:], octets)
		ip = netip.AddrFrom16(b16)
	}

	// make a normalized prefix from ip/bits
	pfx, _ = ip.Prefix(bits)
	return
}

// toplevelSupernetsRec runs recursive the trie, starting at node.
// A top level supernet prefix is defined as having no other prefix as supernet.
// All other prefixes in the trie overlaps with exactly one toplevel supernet.
//
// The yield function is called for each top level supernet.
// If the yield function returns false the recursion ends prematurely and the
// false value is propagated.
//
//	example:
//
//	  ▼
//	  ├─ 6.238.0.0/15
//	  ├─ 8.0.0.0/7
//	  ├─ 25.12.76.64/29
//	  ├─ 52.154.212.32/28
//	  ├─ 115.14.128.0/17
//	  └─ 128.0.0.0/1
//	     ├─ 129.222.24.0/25
//	     ├─ 130.101.156.0/24
//	     ├─ 150.200.0.0/13
//	     ├─ 153.203.41.64/27
//	     ├─ 186.22.98.220/31
//	     ├─ 200.0.0.0/5
//	     │  └─ 205.39.238.0/23
//	     ├─ 212.84.85.0/24
//	     └─ 252.0.0.0/6
//	        └─ 252.196.0.0/18
//
//	toplevel supernets:
//
//	  6.238.0.0/15
//	  8.0.0.0/7
//	  25.12.76.64/29
//	  52.154.212.32/28
//	  115.14.128.0/17
//	  128.0.0.0/1
func (n *node2[V]) toplevelSupernetsRec(yield func(pfx netip.Prefix) bool) bool {
	// allotTbl, filled with prefix entries
	var allotTbl [maxNodePrefixes]bool

	// DEFAULT ROUTE, very fast exit, masks all prefixes and children
	if n.prefixesBitset.Test(1) {
		return yield(n.cidrFromIndex(1))
	}

	// for all prefixes do
	for _, idx := range n.allStrideIndexes() {
		// not yet masked by supernet
		if !allotTbl[idx] {
			if !yield(n.cidrFromIndex(idx)) {
				return false
			}
			// this is a top level supernet, mask allot slots
			allot(&allotTbl, idx)
		}
	}

	// for all children do
	for _, addr := range n.allChildAddrs() {
		// host route already set with supernet prefix?
		if allotTbl[addr+firstHostIndex] {
			continue
		}

		// child not masked by supernet, rec-descent
		c := n.getChild(byte(addr))
		if !c.toplevelSupernetsRec(yield) {
			return false
		}
	}
	return true
}
