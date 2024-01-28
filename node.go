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
	maxNodeChilds   = 1 << stride       // 256
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
// backtracking is fast, it's just a bitset test and, if found, a popcount.
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
		idx >>= 1
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

// getVal for baseIdx.
func (p *prefixCBTree[V]) getVal(baseIdx uint) *V {
	if p.indexes.Test(baseIdx) {
		return p.values[p.rank(baseIdx)]
	}
	return nil
}

// allIndexes returns all baseIndexes set in this prefixHeap.
func (p *prefixCBTree[V]) allIndexes() []uint {
	all := make([]uint, p.indexes.Count())
	p.indexes.NextSetMany(0, all)
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

// allAddrs returns the addrs of all child nodes.
func (c *childTree[V]) allAddrs() []uint {
	all := make([]uint, c.addrs.Count())
	c.addrs.NextSetMany(0, all)
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

// baseIndexToPrefix returns the address and prefix len of baseIdx.
// It's the inverse to prefixToBaseIndex.
func baseIndexToPrefix(baseIdx uint) (addr uint, pfxLen int) {
	nlz := bits.LeadingZeros(baseIdx)
	pfxLen = strconv.IntSize - nlz - 1
	addr = baseIdx & (0xFF >> (stride - pfxLen)) << (stride - pfxLen)
	return addr, pfxLen
}

// baseIndexToPrefixLen returns the prefix len of baseIdx, partly the inverse to prefixToBaseIndex.
// Needed for Lookup, it's faster than:
//
//	_, pfxLen := baseIndexToPrefix(idx)
func baseIndexToPrefixLen(baseIdx uint) int {
	return strconv.IntSize - bits.LeadingZeros(baseIdx) - 1
}
