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
	strideLen       = 8                    // byte
	maxTreeDepth    = 128 / strideLen      // 16
	maxNodeChildren = 1 << strideLen       // 256
	maxNodePrefixes = 1 << (strideLen + 1) // 512
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
	values  []V
}

// childTree, just a slice with nodes, but also popcount-compressed
type childTree[V any] struct {
	addrs *bitset.BitSet
	nodes []*node[V]
}

// newNode, BitSets have to be initialized.
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
func (p *prefixCBTree[V]) rank(baseIdx uint) int {
	return int(p.indexes.Rank(baseIdx)) - 1
}

// insert adds the route addr/prefixLen, with value val.
// Just an adapter for insertIdx.
func (p *prefixCBTree[V]) insert(addr uint, prefixLen int, val V) {
	p.insertIdx(prefixToBaseIndex(addr, prefixLen), val)
}

// insertIdx adds the route for baseIdx, with value val.
func (p *prefixCBTree[V]) insertIdx(baseIdx uint, val V) {
	// prefix exists, overwrite val
	if p.indexes.Test(baseIdx) {
		p.values[p.rank(baseIdx)] = val
		return
	}

	// new, insert into bitset and slice
	p.indexes.Set(baseIdx)
	p.values = slices.Insert(p.values, p.rank(baseIdx), val)
}

// update or set the value at prefix via callback.
func (p *prefixCBTree[V]) update(addr uint, prefixLen int, cb func(V, bool) V) (val V) {
	// calculate idx once
	baseIdx := prefixToBaseIndex(addr, prefixLen)

	var ok bool
	var rnk int

	// if prefix is set, get current value
	if ok = p.indexes.Test(baseIdx); ok {
		rnk = p.rank(baseIdx)
		val = p.values[rnk]
	}

	// callback function to get updated or new value
	val = cb(val, ok)

	// prefix is already set, update and return value
	if ok {
		p.values[rnk] = val
		return val
	}

	// new prefix, insert into bitset ...
	p.indexes.Set(baseIdx)

	// bitset has changed, recalc rank
	rnk = p.rank(baseIdx)

	// ... and insert value into slice
	p.values = slices.Insert(p.values, rnk, val)

	return val
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
			return idx, p.values[p.rank(idx)], true
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

// lpmByAddr is an adapter to lpmByIndex.
func (p *prefixCBTree[V]) lpmByAddr(addr uint) (baseIdx uint, val V, ok bool) {
	return p.lpmByIndex(addrToBaseIndex(addr))
}

// lpmByPrefix is an adapter to lpmByIndex.
func (p *prefixCBTree[V]) lpmByPrefix(addr uint, bits int) (baseIdx uint, val V, ok bool) {
	return p.lpmByIndex(prefixToBaseIndex(addr, bits))
}

// apmByPrefix does an all prefix match in the 8-bit (stride) routing table
// at this depth and returns all matching baseIdx's.
func (p *prefixCBTree[V]) apmByPrefix(addr uint, bits int) (result []uint) {
	// skip intermediate nodes
	if len(p.values) == 0 {
		return
	}

	idx := prefixToBaseIndex(addr, bits)
	for {
		if p.indexes.Test(idx) {
			result = append(result, idx)
		}

		if idx == 0 {
			break
		}

		idx = parentIndex(idx)
	}

	// sort in ascending order
	slices.Sort(result)
	return result
}

// getValByIndex for baseIdx.
func (p *prefixCBTree[V]) getValByIndex(baseIdx uint) (val V, ok bool) {
	if p.indexes.Test(baseIdx) {
		return p.values[p.rank(baseIdx)], true
	}
	return
}

// getValByPrefix, adapter for getValByIndex.
func (p *prefixCBTree[V]) getValByPrefix(addr uint, bits int) (val V, ok bool) {
	return p.getValByIndex(prefixToBaseIndex(addr, bits))
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

// allAddrs returns the addrs of all child nodes in ascending order.
func (c *childTree[V]) allAddrs() []uint {
	all := make([]uint, maxNodeChildren)
	_, all = c.addrs.NextSetMany(0, all)
	return all
}

// ################## node ###################################

// isEmpty returns true if node has neither prefixes nor children.
func (n *node[V]) isEmpty() bool {
	return len(n.prefixes.values) == 0 && len(n.children.nodes) == 0
}

// overlapsRec returns true if any IP in the nodes n or o overlaps.
// First test the routes, then the children and if no match rec-descent
// for child nodes with same addr.
func (n *node[V]) overlapsRec(o *node[V]) bool {
	// dynamically allot the host routes from prefixes
	nAllotIndex := [maxNodePrefixes]bool{}
	oAllotIndex := [maxNodePrefixes]bool{}

	// 1. test if any routes overlaps?

	nOk := len(n.prefixes.values) > 0
	oOk := len(o.prefixes.values) > 0
	var nIdx, oIdx uint
	// zig-zag, for all routes in both nodes ...
	for {
		if nOk {
			// range over bitset, node n
			if nIdx, nOk = n.prefixes.indexes.NextSet(nIdx); nOk {
				// get range of host routes for this prefix
				lowerBound, upperBound := lowerUpperBound(nIdx)

				// insert host routes (addr/8) for this prefix,
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

		if oOk {
			// range over bitset, node o
			if oIdx, oOk = o.prefixes.indexes.NextSet(oIdx); oOk {
				// get range of host routes for this prefix
				lowerBound, upperBound := lowerUpperBound(oIdx)

				// insert host routes (addr/8) for this prefix,
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
		if !nOk && !oOk {
			break
		}
	}

	// full run, zig-zag didn't already match
	if len(n.prefixes.values) > 0 && len(o.prefixes.values) > 0 {
		for i := firstHostIndex; i <= lastHostIndex; i++ {
			if nAllotIndex[i] && oAllotIndex[i] {
				return true
			}
		}
	}

	// 2. test if routes overlaps any child

	nAddresses := [maxNodeChildren]bool{}
	oAddresses := [maxNodeChildren]bool{}

	nOk = len(n.children.nodes) > 0
	oOk = len(o.children.nodes) > 0
	var nAddr, oAddr uint
	// zig-zag, for all addrs in both nodes ...
	for {
		// range over bitset, node n
		if nOk {
			if nAddr, nOk = n.children.addrs.NextSet(nAddr); nOk {
				if oAllotIndex[nAddr+firstHostIndex] {
					return true
				}
				nAddresses[nAddr] = true
				nAddr++
			}
		}

		// range over bitset, node o
		if oOk {
			if oAddr, oOk = o.children.addrs.NextSet(oAddr); oOk {
				if nAllotIndex[oAddr+firstHostIndex] {
					return true
				}
				oAddresses[oAddr] = true
				oAddr++
			}
		}

		if !nOk && !oOk {
			break
		}
	}

	// 3. rec-descent call for childs with same addr

	if len(n.children.nodes) > 0 && len(o.children.nodes) > 0 {
		for i := 0; i < len(nAddresses); i++ {
			if nAddresses[i] && oAddresses[i] {
				// get next child node for this addr
				nc := n.children.get(uint(i))
				oc := o.children.get(uint(i))

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
func (n *node[V]) overlapsPrefix(addr uint, pfxLen int) bool {
	// ##################################################
	// 1. test if any route in this node overlaps prefix?

	pfxIdx := prefixToBaseIndex(addr, pfxLen)
	if _, _, ok := n.prefixes.lpmByIndex(pfxIdx); ok {
		return true
	}

	// #################################################
	// 2. test if prefix overlaps any route in this node

	// lower/upper boundary for addr/pfxLen host routes
	pfxLowerBound := addr + firstHostIndex
	pfxUpperBound := lastHostIndexOfPrefix(addr, pfxLen)

	// increment to 'next' routeIdx for start in bitset search
	// since pfxIdx already testet by lpm in other direction
	routeIdx := pfxIdx << 1
	var ok bool
	for {
		if routeIdx, ok = n.prefixes.indexes.NextSet(routeIdx); !ok {
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

	// set start address in bitset search with prefix addr
	childAddr := addr
	for {
		if childAddr, ok = n.children.addrs.NextSet(childAddr); !ok {
			break
		}

		childIdx := childAddr + firstHostIndex
		if childIdx >= pfxLowerBound && childIdx <= pfxUpperBound {
			return true
		}

		// next round
		childAddr++
	}

	return false
}

// unionRec combines two nodes, changing the receiver node.
// If there are duplicate entries, the value is taken from the other node.
func (n *node[V]) unionRec(o *node[V]) {
	var oIdx uint
	var oOk bool
	// for all prefixes in other node do ...
	for {
		if oIdx, oOk = o.prefixes.indexes.NextSet(oIdx); !oOk {
			break
		}
		oVal, _ := o.prefixes.getValByIndex(oIdx)
		// insert/overwrite prefix/value from oNode to nNode
		n.prefixes.insertIdx(oIdx, oVal)
		oIdx++
	}

	var oAddr uint
	// for all children in other node do ...
	for {
		if oAddr, oOk = o.children.addrs.NextSet(oAddr); !oOk {
			break
		}
		oNode := o.children.get(oAddr)

		// get nNode with same addr
		nNode := n.children.get(oAddr)
		if nNode == nil {
			// union child from oNode into nNode
			n.children.insert(oAddr, oNode.cloneRec())
		} else {
			// both nodes have child with addr, call union rec-descent
			nNode.unionRec(oNode)
		}
		oAddr++
	}
}

func (n *node[V]) cloneRec() *node[V] {
	c := newNode[V]()
	if n.isEmpty() {
		return c
	}

	c.prefixes.indexes = n.prefixes.indexes.Clone()     // deep
	c.prefixes.values = slices.Clone(n.prefixes.values) // shallow

	c.children.addrs = n.children.addrs.Clone()       // deep
	c.children.nodes = slices.Clone(n.children.nodes) // shallow
	// make it deep
	for i, child := range c.children.nodes {
		c.children.nodes[i] = child.cloneRec()
	}

	return c
}

// walkRec runs recursive the trie, starting at node and
// the cb function is called for each route entry with prefix and value.
// If the cb function returns an error the walk ends prematurely and the
// error is propagated.
func (n *node[V]) walkRec(path []byte, is4 bool, cb func(netip.Prefix, V) error) error {
	// for all prefixes in this node do ...
	for _, idx := range n.prefixes.allIndexes() {
		val, _ := n.prefixes.getValByIndex(idx)
		pfx := cidrFromPath(path, idx, is4)

		// make the callback for this prefix
		if err := cb(pfx, val); err != nil {
			// premature end of recursion
			return err
		}
	}

	// for all children in this node do ...
	for _, addr := range n.children.allAddrs() {
		path := append(slices.Clone(path), byte(addr))
		child := n.children.get(addr)

		if err := child.walkRec(path, is4, cb); err != nil {
			// premature end of recursion
			return err
		}
	}

	return nil
}

// subnets returns all CIDRs covered by parent pfx.
func (n *node[V]) subnets(path []byte, parentIdx uint, is4 bool) (result []netip.Prefix) {
	// for all routes in this node do ...
	for _, idx := range n.prefixes.allIndexes() {
		// is this route covered by pfx?
		for i := idx; i >= parentIdx; i >>= 1 {
			if i == parentIdx { // match
				// get CIDR back for idx
				pfx := cidrFromPath(path, idx, is4)

				result = append(result, pfx)
				break
			}
		}
	}

	// for all children in this node do ...
	for _, addr := range n.children.allAddrs() {
		idx := addrToBaseIndex(addr)

		// is this child covered by pfx?
		for i := idx; i >= parentIdx; i >>= 1 {
			if i == parentIdx { // match
				// get child for addr
				c := n.children.get(addr)

				// append addr to path
				path := append(slices.Clone(path), byte(addr))

				// all cidrs under this child are covered by pfx
				_ = c.walkRec(path, is4, func(pfx netip.Prefix, _ V) error {
					result = append(result, pfx)
					return nil
				})
			}
		}
	}

	return result
}

// sortByPrefix
func sortByPrefix(a, b netip.Prefix) int {
	if cmp := a.Masked().Addr().Compare(b.Masked().Addr()); cmp != 0 {
		return cmp
	}
	return cmp.Compare(a.Bits(), b.Bits())
}
