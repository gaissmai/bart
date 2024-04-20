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
	prefixes *strideTree[V]
	children *childSlice[V]
}

// strideTree, complete-binary-tree, popcount-compressed.
type strideTree[V any] struct {
	*bitset.BitSet
	values []V
}

// childSlice, a slice with nodes, popcount-compressed
type childSlice[V any] struct {
	*bitset.BitSet
	childs []*node[V]
}

// newNode, BitSets have to be initialized.
func newNode[V any]() *node[V] {
	return &node[V]{
		prefixes: &strideTree[V]{
			BitSet: bitset.New(0), // init BitSet
			values: nil,
		},

		children: &childSlice[V]{
			BitSet: bitset.New(0), // init BitSet
			childs: nil,
		},
	}
}

// ################## prefixes ##################################

// rank is the key of the popcount compression algorithm,
// mapping between bitset index and slice index.
func (p *strideTree[V]) rank(baseIdx uint) int {
	return int(p.Rank(baseIdx)) - 1
}

// insert adds the route octet/prefixLen, with value val.
// Just an adapter for insertIdx.
func (p *strideTree[V]) insert(octet uint, prefixLen int, val V) {
	p.insertIdx(prefixToBaseIndex(octet, prefixLen), val)
}

// insertIdx adds the route for baseIdx, with value val.
func (p *strideTree[V]) insertIdx(baseIdx uint, val V) {
	// prefix exists, overwrite val
	if p.Test(baseIdx) {
		p.values[p.rank(baseIdx)] = val
		return
	}

	// new, insert into bitset and slice
	p.Set(baseIdx)
	p.values = slices.Insert(p.values, p.rank(baseIdx), val)
}

// update or set the value at prefix via callback.
func (p *strideTree[V]) update(octet uint, prefixLen int, cb func(V, bool) V) (val V) {
	// calculate idx once
	baseIdx := prefixToBaseIndex(octet, prefixLen)

	var ok bool
	var rnk int

	// if prefix is set, get current value
	if ok = p.Test(baseIdx); ok {
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
	p.Set(baseIdx)

	// bitset has changed, recalc rank
	rnk = p.rank(baseIdx)

	// ... and insert value into slice
	p.values = slices.Insert(p.values, rnk, val)

	return val
}

// delete removes the route octet/prefixLen. Reports whether the
// prefix existed in the table prior to deletion.
func (p *strideTree[V]) delete(octet uint, prefixLen int) (wasPresent bool) {
	baseIdx := prefixToBaseIndex(octet, prefixLen)

	// no route entry
	if !p.Test(baseIdx) {
		return false
	}

	rnk := p.rank(baseIdx)

	// delete from slice
	p.values = slices.Delete(p.values, rnk, rnk+1)

	// delete from bitset, followed by Compact to reduce memory consumption
	p.Clear(baseIdx)
	p.Compact()

	return true
}

// lpmByIndex does a route lookup for idx in the 8-bit (stride) routing table
// at this depth and returns (baseIdx, value, true) if a matching
// longest prefix exists, or ok=false otherwise.
//
// backtracking is fast, it's just a bitset test and, if found, one popcount.
func (p *strideTree[V]) lpmByIndex(idx uint) (baseIdx uint, val V, ok bool) {
	// max steps in backtracking is the stride length.
	for {
		if p.Test(idx) {
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

// lpmByOctet is an adapter to lpmByIndex.
func (p *strideTree[V]) lpmByOctet(octet uint) (baseIdx uint, val V, ok bool) {
	return p.lpmByIndex(octetToBaseIndex(octet))
}

// lpmByPrefix is an adapter to lpmByIndex.
func (p *strideTree[V]) lpmByPrefix(octet uint, bits int) (baseIdx uint, val V, ok bool) {
	return p.lpmByIndex(prefixToBaseIndex(octet, bits))
}

// apmByPrefix does an all prefix match in the 8-bit (stride) routing table
// at this depth and returns all matching baseIdx's.
func (p *strideTree[V]) apmByPrefix(octet uint, bits int) (result []uint) {
	// skip intermediate nodes
	if len(p.values) == 0 {
		return
	}

	idx := prefixToBaseIndex(octet, bits)
	for {
		if p.Test(idx) {
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
func (p *strideTree[V]) getValByIndex(baseIdx uint) (val V, ok bool) {
	if p.Test(baseIdx) {
		return p.values[p.rank(baseIdx)], true
	}
	return
}

// getValByPrefix, adapter for getValByIndex.
func (p *strideTree[V]) getValByPrefix(octet uint, bits int) (val V, ok bool) {
	return p.getValByIndex(prefixToBaseIndex(octet, bits))
}

// allIndexes returns all baseIndexes set in this prefix tree in ascending order.
func (p *strideTree[V]) allIndexes() []uint {
	all := make([]uint, 0, maxNodePrefixes)
	_, all = p.NextSetMany(0, all)
	return all
}

// ################## childs ####################################

// rank is the key of the popcount compression algorithm,
// mapping between bitset index and slice index.
func (c *childSlice[V]) rank(octet uint) int {
	return int(c.Rank(octet)) - 1
}

// insert the child into childTree.
func (c *childSlice[V]) insert(octet uint, child *node[V]) {
	// insert into bitset and slice
	c.Set(octet)
	c.childs = slices.Insert(c.childs, c.rank(octet), child)
}

// delete the child at octet. It is valid to delete a non-existent child.
func (c *childSlice[V]) delete(octet uint) {
	if !c.Test(octet) {
		return
	}

	rnk := c.rank(octet)

	// delete from slice
	c.childs = slices.Delete(c.childs, rnk, rnk+1)

	// delete from bitset, followed by Compact to reduce memory consumption
	c.Clear(octet)
	c.Compact()
}

// get returns the child pointer for octet, or nil if none.
func (c *childSlice[V]) get(octet uint) *node[V] {
	if !c.Test(octet) {
		return nil
	}

	return c.childs[c.rank(octet)]
}

// allOctets returns the octets of all child nodes in ascending order.
func (c *childSlice[V]) allOctets() []uint {
	all := make([]uint, maxNodeChildren)
	_, all = c.NextSetMany(0, all)
	return all
}

// ################## node ###################################

// isEmpty returns true if node has neither prefixes nor children.
func (n *node[V]) isEmpty() bool {
	return len(n.prefixes.values) == 0 && len(n.children.childs) == 0
}

// overlapsRec returns true if any IP in the nodes n or o overlaps.
// First test the routes, then the children and if no match rec-descent
// for child nodes with same octet.
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
			if nIdx, nOk = n.prefixes.NextSet(nIdx); nOk {
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

		if oOk {
			// range over bitset, node o
			if oIdx, oOk = o.prefixes.NextSet(oIdx); oOk {
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

	nOctets := [maxNodeChildren]bool{}
	oOctets := [maxNodeChildren]bool{}

	nOk = len(n.children.childs) > 0
	oOk = len(o.children.childs) > 0
	var nOctet, oOctet uint
	// zig-zag, for all octets in both nodes ...
	for {
		// range over bitset, node n
		if nOk {
			if nOctet, nOk = n.children.NextSet(nOctet); nOk {
				if oAllotIndex[nOctet+firstHostIndex] {
					return true
				}
				nOctets[nOctet] = true
				nOctet++
			}
		}

		// range over bitset, node o
		if oOk {
			if oOctet, oOk = o.children.NextSet(oOctet); oOk {
				if nAllotIndex[oOctet+firstHostIndex] {
					return true
				}
				oOctets[oOctet] = true
				oOctet++
			}
		}

		if !nOk && !oOk {
			break
		}
	}

	// 3. rec-descent call for childs with same octet

	if len(n.children.childs) > 0 && len(o.children.childs) > 0 {
		for i := 0; i < len(nOctets); i++ {
			if nOctets[i] && oOctets[i] {
				// get next child node for this octet
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
func (n *node[V]) overlapsPrefix(octet uint, pfxLen int) bool {
	// ##################################################
	// 1. test if any route in this node overlaps prefix?

	pfxIdx := prefixToBaseIndex(octet, pfxLen)
	if _, _, ok := n.prefixes.lpmByIndex(pfxIdx); ok {
		return true
	}

	// #################################################
	// 2. test if prefix overlaps any route in this node

	// lower/upper boundary for octet/pfxLen host routes
	pfxLowerBound := octet + firstHostIndex
	pfxUpperBound := lastHostIndexOfPrefix(octet, pfxLen)

	// increment to 'next' routeIdx for start in bitset search
	// since pfxIdx already testet by lpm in other direction
	routeIdx := pfxIdx << 1
	var ok bool
	for {
		if routeIdx, ok = n.prefixes.NextSet(routeIdx); !ok {
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
	childOctet := octet
	for {
		if childOctet, ok = n.children.NextSet(childOctet); !ok {
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

// unionRec combines two nodes, changing the receiver node.
// If there are duplicate entries, the value is taken from the other node.
func (n *node[V]) unionRec(o *node[V]) {
	var oIdx uint
	var oOk bool
	// for all prefixes in other node do ...
	for {
		if oIdx, oOk = o.prefixes.NextSet(oIdx); !oOk {
			break
		}
		oVal, _ := o.prefixes.getValByIndex(oIdx)
		// insert/overwrite prefix/value from oNode to nNode
		n.prefixes.insertIdx(oIdx, oVal)
		oIdx++
	}

	var oOctet uint
	// for all children in other node do ...
	for {
		if oOctet, oOk = o.children.NextSet(oOctet); !oOk {
			break
		}
		oNode := o.children.get(oOctet)

		// get nNode with same octet
		nNode := n.children.get(oOctet)
		if nNode == nil {
			// union child from oNode into nNode
			n.children.insert(oOctet, oNode.cloneRec())
		} else {
			// both nodes have child with octet, call union rec-descent
			nNode.unionRec(oNode)
		}
		oOctet++
	}
}

func (n *node[V]) cloneRec() *node[V] {
	c := newNode[V]()
	if n.isEmpty() {
		return c
	}

	c.prefixes.BitSet = n.prefixes.BitSet.Clone()       // deep
	c.prefixes.values = slices.Clone(n.prefixes.values) // shallow

	c.children.BitSet = n.children.BitSet.Clone()       // deep
	c.children.childs = slices.Clone(n.children.childs) // shallow
	// make it deep
	for i, child := range c.children.childs {
		c.children.childs[i] = child.cloneRec()
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
	for _, octet := range n.children.allOctets() {
		path := append(slices.Clone(path), byte(octet))
		child := n.children.get(octet)

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
	for _, octet := range n.children.allOctets() {
		idx := octetToBaseIndex(octet)

		// is this child covered by pfx?
		for i := idx; i >= parentIdx; i >>= 1 {
			if i == parentIdx { // match
				// get child for octet
				c := n.children.get(octet)

				// append octet to path
				path := append(slices.Clone(path), byte(octet))

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
