// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"math/bits"
	"slices"
	"strconv"

	"github.com/bits-and-blooms/bitset"
)

const (
	stride          = 8                 // byte
	maxTreeDepth    = 128 / stride      // 16
	maxNodeChildren = 1 << stride       // 256
	maxNodePrefixes = 1 << (stride + 1) // 512
)

type nodeType byte

const (
	nullNode         nodeType = iota // empty node
	fullNode                         // prefixes and childs
	leafNode                         // only prefix(es)
	intermediateNode                 // only child(s)
)

// node, a level node in the multibit-trie.
// A node can have prefixes or child nodes or both.
type node[V any] struct {
	prefixes *prefixCBTree[V]
	children *childTree[V]
}

// prefixCBTree, complete binary tree, popcount-compressed.
type prefixCBTree[V any] struct {
	indexes *bitset.BitSet
	values  []*V
}

// childTree, just a slice with nodes, but also popcount-compressed
type childTree[V any] struct {
	addrs *bitset.BitSet
	nodes []*node[V]
}

// newNode, bitSets have to be initialized.
//
// The maximum length of the bitsets are known in advance
// (maxNodeChilds and maxNodePrefixes), so you could create correspondingly
// large slices for every node, this would save minimal computing time
// during insert, but if you let the bitset slices grow when necessary,
// we save a lot of memory on average.
func newNode[V any]() *node[V] {
	return &node[V]{
		prefixes: &prefixCBTree[V]{
			indexes: bitset.New(0), // init BitSet, zero size
			values:  nil,
		},

		children: &childTree[V]{
			addrs: bitset.New(0), // init BitSet, zero size
			nodes: nil,
		},
	}
}

// isEmpty returns true if node has neither prefixes nor children.
func (n *node[V]) isEmpty() bool {
	return len(n.prefixes.values) == 0 && len(n.children.nodes) == 0
}

// ################## prefixes ##################################

// rank is the key of the popcount compression algorithm,
// mapping between bitset index and slice index.
func (p *prefixCBTree[V]) rank(treeIdx uint) int {
	return int(p.indexes.Rank(treeIdx)) - 1
}

// insert adds the route addr/prefixLen, with value val.
func (p *prefixCBTree[V]) insert(addr uint, prefixLen int, val V) {
	baseIdx := prefixToBaseIndex(addr, prefixLen)

	// prefix exists, overwrite val
	if p.indexes.Test(baseIdx) {
		p.values[p.rank(baseIdx)] = &val
		return
	}

	// new, insert into bitset and slice
	p.indexes.Set(baseIdx)
	p.values = slices.Insert(p.values, p.rank(baseIdx), &val)
}

// delete removes the route addr/prefixLen. Reports whether the
// prefix existed in the table prior to deletion.
func (p *prefixCBTree[V]) delete(addr uint, prefixLen int) (wasPresent bool) {
	baseIdx := prefixToBaseIndex(addr, prefixLen)

	// no route entry
	if !p.indexes.Test(baseIdx) {
		return false
	}

	rnk := p.rank(baseIdx)

	// delete from slice
	// TODO: with go 1.22 the free slot is already clear'd by Delete for GC
	p.values = slices.Delete(p.values, rnk, rnk+1)

	// delete from bitset, followed by Compact to reduce memory consumption
	p.indexes.Clear(baseIdx)
	p.indexes.Compact()

	return true
}

// lpmByIndex does a route lookup for idx in the 8-bit (stride) routing table
// at this depth and returns (baseIdx, value, true) if a matching
// longest prefix exists, or ok=false otherwise.
//
// backtracking is fast, it's just a bitset test and, if found, one popcount.
func (p *prefixCBTree[V]) lpmByIndex(idx uint) (baseIdx uint, val V, ok bool) {
	// max steps in backtracking is the stride length.
	for {
		if p.indexes.Test(idx) {
			// longest prefix match
			return idx, *(p.values[p.rank(idx)]), true
		}

		if idx == 0 {
			break
		}

		// cache friendly backtracking to the next less specific route.
		// thanks to the complete binary tree it's just a shift operation.
		idx = parentIndex(idx)
	}

	// not found (on this level)
	return 0, val, false
}

// lpmByAddr does a route lookup for addr in the 8-bit (stride) routing table.
// It's an adapter to lpmByIndex.
func (p *prefixCBTree[V]) lpmByAddr(addr uint) (baseIdx uint, val V, ok bool) {
	return p.lpmByIndex(addrToBaseIndex(addr))
}

// lpmByPrefix does a route lookup for addr/pfxLen in the 8-bit (stride) routing table
// It's an adapter to lpmByIndex.
//
//nolint:unused
func (p *prefixCBTree[V]) lpmByPrefix(addr uint, prefixLen int) (baseIdx uint, val V, ok bool) {
	return p.lpmByIndex(prefixToBaseIndex(addr, prefixLen))
}

// spmByIndex does a shortest-prefix-match for idx in the 8-bit (stride) routing table
// at this depth and returns (baseIdx, value, true) if a matching
// shortest prefix exists, or ok=false otherwise.
//
// backtracking is stride*bitset-test and, if found, one popcount.
func (p *prefixCBTree[V]) spmByIndex(idx uint) (baseIdx uint, val V, ok bool) {
	var shortest uint
	// steps in backtracking is always the stride length for spm,
	for {
		if p.indexes.Test(idx) {
			shortest = idx
			// no fast exit on match for shortest-prefix-match.
		}

		if idx == 0 {
			break
		}

		// cache friendly backtracking to the next less specific route.
		// thanks to the complete binary tree it's just a shift operation.
		idx = parentIndex(idx)
	}

	if shortest != 0 {
		return shortest, *(p.values[p.rank(shortest)]), true
	}

	// not found (on this level)
	return 0, val, false
}

// spmByAddr does a shortest-prefix-match for addr in the 8-bit (stride) routing table.
// It's an adapter to spmByIndex.
func (p *prefixCBTree[V]) spmByAddr(addr uint) (baseIdx uint, val V, ok bool) {
	return p.spmByIndex(addrToBaseIndex(addr))
}

// overlaps reports whether the route addr/prefixLen overlaps
// with any prefix in this node..
func (p *prefixCBTree[V]) overlaps(addr uint, pfxLen int) bool {
	baseIdx := prefixToBaseIndex(addr, pfxLen)

	// any route in this node overlaps prefix?
	if _, _, ok := p.lpmByIndex(baseIdx); ok {
		return true
	}

	// from here on: reverse direction,
	// test if prefix overlaps any route in this node.

	// lower boundary, idx == baseIdx alreday tested with lpm above,
	// increase it
	idx := baseIdx << 1

	// upper boundary for addr/pfxLen
	lastHostIdx := lastHostIndex(addr, pfxLen)

	var ok bool
	for {
		if idx, ok = p.indexes.NextSet(idx); !ok {
			return false
		}

		// out of addr/pfxLen
		if idx > lastHostIdx {
			return false
		}

		// e.g.: 365 -> 182 -> 91 -> 45 -> 22 -> baseIdx(11) STOP
		//
		for j := idx; j >= baseIdx; j = parentIndex(j) {
			if j == baseIdx {
				return true
			}
		}
		// next round
		idx++
	}
}

// getVal for baseIdx.
func (p *prefixCBTree[V]) getVal(baseIdx uint) *V {
	if p.indexes.Test(baseIdx) {
		return p.values[p.rank(baseIdx)]
	}
	return nil
}

// allIndexes returns all baseIndexes set in this prefix tree in ascending order.
func (p *prefixCBTree[V]) allIndexes() []uint {
	all := make([]uint, 0, maxNodePrefixes)
	_, all = p.indexes.NextSetMany(0, all)
	return all
}

// ################## childs ####################################

// rank is the key of the popcount compression algorithm,
// mapping between bitset index and slice index.
func (c *childTree[V]) rank(addr uint) int {
	return int(c.addrs.Rank(addr)) - 1
}

// insert the child into childTree.
func (c *childTree[V]) insert(addr uint, child *node[V]) {
	// insert into bitset and slice
	c.addrs.Set(addr)
	c.nodes = slices.Insert(c.nodes, c.rank(addr), child)
}

// delete the child at addr. It is valid to delete a non-existent child.
func (c *childTree[V]) delete(addr uint) {
	if !c.addrs.Test(addr) {
		return
	}

	rnk := c.rank(addr)

	// delete from slice
	// TODO: with go 1.22 the free slot is clear'd by Delete for GC
	c.nodes = slices.Delete(c.nodes, rnk, rnk+1)

	// delete from bitset, followed by Compact to reduce memory consumption
	c.addrs.Clear(addr)
	c.addrs.Compact()
}

// get returns the child pointer for addr, or nil if none.
func (c *childTree[V]) get(addr uint) *node[V] {
	if !c.addrs.Test(addr) {
		return nil
	}

	return c.nodes[c.rank(addr)]
}

// overlaps reports whether the prefix addr/pfxLen overlaps
// with any child in this node..
func (c *childTree[V]) overlaps(addr uint, pfxLen int) bool {
	// lower boundary for addr/pfxLen
	baseIdx := prefixToBaseIndex(addr, pfxLen)

	// upper boundary for addr/pfxLen
	lastHostIdx := lastHostIndex(addr, pfxLen)

	var ok bool
	for {
		if addr, ok = c.addrs.NextSet(addr); !ok {
			return false
		}

		// this addrs baseIdx
		hostIdx := addrToBaseIndex(addr)

		// out of addr/pfxLen
		if hostIdx > lastHostIdx {
			return false
		}

		// check if prefix overlaps this child or any of his parents
		// within the limits of addr/pfxLen
		for idx := hostIdx; idx >= baseIdx; idx = parentIndex(idx) {
			if idx == baseIdx {
				return true
			}
		}
		// next round
		addr++
	}
}

// allAddrs returns the addrs of all child nodes in ascending order.
func (c *childTree[V]) allAddrs() []uint {
	all := make([]uint, maxNodeChildren)
	_, all = c.addrs.NextSetMany(0, all)
	return all
}

// ################## helpers ###################################

// prefixToBaseIndex, maps a prefix table as a 'complete binary tree'.
// This is the so-called baseIndex a.k.a heapFunc:
//
// https://cseweb.ucsd.edu//~varghese/TEACH/cs228/artlookup.pdf
func prefixToBaseIndex(addr uint, prefixLen int) uint {
	return (addr >> (stride - prefixLen)) + (1 << prefixLen)
}

// addrToBaseIndex, just prefixToBaseIndex(addr, 8), a.k.a host routes
// but faster, use it for host routes in Get and Lookup.
func addrToBaseIndex(addr uint) uint {
	return addr + 1<<stride
}

// parentIndex returns the index of idx's parent prefix, or 0 if idx
// is the index of 0/0.
func parentIndex(idx uint) uint {
	return idx >> 1
}

// baseIndexToPrefix returns the address and prefix len of baseIdx.
// It's the inverse to prefixToBaseIndex.
func baseIndexToPrefix(baseIdx uint) (addr uint, pfxLen int) {
	nlz := bits.LeadingZeros(baseIdx)
	pfxLen = strconv.IntSize - nlz - 1
	addr = baseIdx & (0xFF >> (stride - pfxLen)) << (stride - pfxLen)
	return addr, pfxLen
}

// baseIndexToPrefixLen returns the prefix len of baseIdx, partly
// the inverse to prefixToBaseIndex.
// Needed for Lookup, it's faster than:
//
//	_, pfxLen := baseIndexToPrefix(idx)
func baseIndexToPrefixLen(baseIdx uint) int {
	return strconv.IntSize - bits.LeadingZeros(baseIdx) - 1
}

var addrMaskTable = []uint{
	0b1111_1111,
	0b0111_1111,
	0b0011_1111,
	0b0001_1111,
	0b0000_1111,
	0b0000_0111,
	0b0000_0011,
	0b0000_0001,
	0b0000_0000,
}

// lastHostIndex returns the array index of the last address in addr/len.
func lastHostIndex(addr uint, bits int) uint {
	return addrToBaseIndex(addr | addrMaskTable[bits])
}
