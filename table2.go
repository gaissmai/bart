// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"slices"
	"sync"
)

// Table2 is an IPv4 and IPv6 routing table with payload V.
// The zero value is ready to use.
//
// The Table2 is safe for concurrent readers but not for
// concurrent readers and/or writers.
//
// The Table2 is an evolution of Table, but now with path compression.
// Path compression introduces a lot of complexity!
//
// "A complex system that works is invariably found to have evolved from
// a simple system that worked. A complex system designed from scratch
// never works and cannot be patched up to make it work.
// You have to start over with a working simple system." (John Gall)
type Table2[V any] struct {
	rootV4 *node2[V]
	rootV6 *node2[V]

	// number of prefixes in trie
	sizeV4 int
	sizeV6 int

	// BitSets have to be initialized.
	initOnce sync.Once
}

// init BitSets once, so no constructor is needed
func (t *Table2[V]) init() {
	t.initOnce.Do(func() {
		t.rootV4 = newNode2[V](nil, true)
		t.rootV6 = newNode2[V](nil, false)
	})
}

// rootNodeByVersion, select root node for ip version.
func (t *Table2[V]) rootNodeByVersion(is4 bool) *node2[V] {
	if is4 {
		return t.rootV4
	}
	return t.rootV6
}

// Insert adds pfx to the tree, with value val.
// If pfx is already present in the tree, its value is set to val.
func (t *Table2[V]) Insert(pfx netip.Prefix, val V) {
	t.init()
	if !pfx.IsValid() {
		return
	}

	// some needed values, see below
	ip, bits, is4 := pfxToValues(pfx)

	// get the root node of the routing table
	n := t.rootNodeByVersion(is4)

	// do not allocate
	octets := make([]byte, 0, 16)
	octets = ipToOctets(octets, ip, is4)

	// 10.0.0.0/8    -> 0
	// 10.12.0.0/15  -> 1
	// 10.12.0.0/16  -> 1
	// 10.12.10.9/32 -> 3
	lastOctetIdx := (bits - 1) / strideLen

	// 10.0.0.0/8    -> 10
	// 10.12.0.0/15  -> 12
	// 10.12.0.0/16  -> 12
	// 10.12.10.9/32 -> 9
	lastOctet := octets[lastOctetIdx]

	// 10.0.0.0/8    -> 8
	// 10.12.0.0/15  -> 7
	// 10.12.0.0/16  -> 8
	// 10.12.10.9/32 -> 8
	lastOctetBits := bits - (lastOctetIdx * strideLen)

	// path of pfx to insert
	path := octets[:lastOctetIdx]

	idx := 0
	cursor := octets[idx]

	// find the proper trie node to insert prefix
	for {
		// last octet reached
		if idx == lastOctetIdx {
			// insert prefix into node
			wasPresent := n.insertPrefix(lastOctet, lastOctetBits, val)
			if !wasPresent {
				t.incDecSize(+1, is4)
			}
			return
		}

		// descend down the trie
		c := n.getChild(cursor)

		// just insert other node as new leaf
		if c == nil {
			// make new node
			o := newNode2[V](path, n.is4)
			o.insertPrefix(lastOctet, lastOctetBits, val)
			t.incDecSize(+1, is4)

			n.insertChild(cursor, o)
			return
		}

		// child is prefix for other node
		if c.pathIsPrefixOrEqual(idx, path) {
			// go down, path compression -> idx may jump
			idx = c.pathLen()
			cursor = octets[idx]
			n = c
			continue
		}

		// make new node
		o := newNode2[V](path, n.is4)
		o.insertPrefix(lastOctet, lastOctetBits, val)
		t.incDecSize(+1, is4)

		// other is prefix for child node
		if o.pathIsPrefixOrEqual(idx, c.pathAsSlice()) {
			// splice other between n and child: n -> other -> child
			// path compression -> idx jump
			idx = o.pathLen()
			o.insertChild(c.pathAsSlice()[idx], c)
			n.insertChild(cursor, o)
			return
		}

		// The paths are different from a certain index.
		commonPathIdx := c.commonPathIdx(idx, o)

		// make intermediate node with path until divergence
		imed := newNode2[V](c.pathAsSlice()[:commonPathIdx+1], n.is4)

		// insert old and new child into intermediate node
		imed.insertChild(c.pathAsSlice()[commonPathIdx+1], c)
		imed.insertChild(o.pathAsSlice()[commonPathIdx+1], o)

		// splice intermediate: n -> imed -> (child, other)
		n.insertChild(cursor, imed)
		return
	}
}

// Delete removes pfx from the tree, pfx does not have to be present.
func (t *Table2[V]) Delete(pfx netip.Prefix) {
	if !pfx.IsValid() {
		return
	}
	// some needed values, see below
	ip, bits, is4 := pfxToValues(pfx)

	n := t.rootNodeByVersion(is4)
	if n == nil {
		return
	}

	// record path to deleted node
	// purge or compact dangling nodes after deletion
	stack := [maxTreeDepth]struct {
		node  *node2[V]
		octet byte
	}{}

	// do not allocate
	octets := make([]byte, 0, 16)
	octets = ipToOctets(octets, ip, is4)

	// see comment in Insert()
	lastOctetIdx := (bits - 1) / strideLen
	lastOctet := octets[lastOctetIdx]
	lastOctetBits := bits - (lastOctetIdx * strideLen)

	// path of pfx to delete
	path := octets[:lastOctetIdx]

	// stackIndex is monotonic
	stackIdx := 0

	// path compression, octet index may jump
	idx := 0

	cursor := octets[idx]

	// find the trie node
	for {
		// insert cursor and corresponding node into stacks for path recording
		stack[stackIdx].octet = cursor
		stack[stackIdx].node = n

		// last significant octet reached
		if idx == lastOctetIdx {
			// found a child on proper depth ...
			if wasPresent := n.deletePrefix(lastOctet, lastOctetBits); !wasPresent {
				// prefix not in tree
				return
			}

			// escape, but purge dangling path if needed, see below
			t.incDecSize(-1, is4)
			break
		}

		// descend down to next level
		c := n.getChild(cursor)

		// no child, stopp
		if c == nil {
			return
		}

		// no match, stopp
		if !c.pathIsPrefixOrEqual(idx, path) {
			return
		}

		// go down, path compression, idx may jump
		idx = c.pathLen()

		// increase stackIdx monotonic
		stackIdx++

		cursor = octets[idx]
		n = c
	}

	var parent *node2[V]

	// pruning parent nodes along the path?
	for {
		// stop condition
		if stackIdx == 0 {
			return
		}

		switch {
		case n.isEmpty():
			// get parent and slot
			parent = stack[stackIdx-1].node
			cursor = stack[stackIdx-1].octet

			// purge this node from parents children
			parent.deleteChild(cursor)

		case len(n.prefixes) == 0 && len(n.children) == 1:
			// n is pure intermediate and has only one child left, compact path

			// get this single child
			child := n.children[0]

			// get parent and slot
			parent = stack[stackIdx-1].node
			cursor = stack[stackIdx-1].octet

			// overwrite intermediate node with child, compacting the path
			parent.insertChild(cursor, child)
		}

		// cascade up
		stackIdx--
		n = stack[stackIdx].node
		continue
	}
}

// Update or set the value at pfx with a callback function.
// The callback function is called with (value, ok) and returns a new value..
//
// If the pfx does not already exist, it is set with the new value.
func (t *Table2[V]) Update(pfx netip.Prefix, cb func(val V, ok bool) V) V {
	t.init()
	if !pfx.IsValid() {
		var zero V
		return zero
	}

	// some needed values, see below
	ip, bits, is4 := pfxToValues(pfx)

	n := t.rootNodeByVersion(is4)

	// do not allocate
	octets := make([]byte, 0, 16)
	octets = ipToOctets(octets, ip, is4)

	// see comment in Insert()
	lastOctetIdx := (bits - 1) / strideLen
	lastOctet := octets[lastOctetIdx]
	lastOctetBits := bits - (lastOctetIdx * strideLen)

	// path of pfx to update
	path := octets[:lastOctetIdx]

	idx := 0
	cursor := octets[idx]

	// find the proper trie node to update prefix
	for {
		// last octet reached
		if idx == lastOctetIdx {
			// update/insert prefix into node
			val, wasPresent := n.updatePrefix(lastOctet, lastOctetBits, cb)
			if !wasPresent {
				t.incDecSize(+1, is4)
			}
			return val
		}

		// descend down the trie
		c := n.getChild(cursor)

		// just insert other node as new leaf
		if c == nil {
			// make new node, already set path and insert prefix
			o := newNode2[V](path, n.is4)
			n.insertChild(cursor, o)

			val, wasPresent := o.updatePrefix(lastOctet, lastOctetBits, cb)
			if !wasPresent {
				t.incDecSize(+1, is4)
			}
			return val
		}

		// child is prefix or equal to other node
		if c.pathIsPrefixOrEqual(idx, path) {
			// go down, path compression, idx may jump
			idx = c.pathLen()
			cursor = octets[idx]
			n = c
			continue
		}

		o := newNode2[V](path, n.is4)

		// other is prefix for child node
		if o.pathIsPrefixOrEqual(idx, c.pathAsSlice()) {
			// splice other between n and child: n -> other -> child
			// path compression, idx may jump
			idx = o.pathLen()

			o.insertChild(c.pathAsSlice()[idx], c)
			n.insertChild(cursor, o)

			val, wasPresent := o.updatePrefix(lastOctet, lastOctetBits, cb)
			if !wasPresent {
				t.incDecSize(+1, is4)
			}
			return val
		}

		// The paths are different from a certain index.
		commonPathIdx := c.commonPathIdx(idx, o)

		// make intermediate node with path until divergence
		imed := newNode2[V](c.pathAsSlice()[:commonPathIdx+1], n.is4)

		// insert old and new child into intermediate node
		imed.insertChild(c.pathAsSlice()[commonPathIdx+1], c)
		imed.insertChild(o.pathAsSlice()[commonPathIdx+1], o)

		// splice intermediate: n -> imed -> (child, other)
		n.insertChild(cursor, imed)

		val, wasPresent := o.updatePrefix(lastOctet, lastOctetBits, cb)
		if !wasPresent {
			t.incDecSize(+1, is4)
		}
		return val
	}
}

// Get returns the associated payload for prefix and true, or false if
// prefix is not set in the routing table.
func (t *Table2[V]) Get(pfx netip.Prefix) (val V, ok bool) {
	if !pfx.IsValid() {
		return
	}
	// some needed values, see below
	ip, bits, is4 := pfxToValues(pfx)

	n := t.rootNodeByVersion(is4)
	if n == nil {
		return
	}

	// do not allocate
	octets := make([]byte, 0, 16)
	octets = ipToOctets(octets, ip, is4)

	// see comment in Insert()
	lastOctetIdx := (bits - 1) / strideLen
	lastOctet := octets[lastOctetIdx]
	lastOctetBits := bits - (lastOctetIdx * strideLen)

	// path of pfx to get
	path := octets[:lastOctetIdx]

	idx := 0
	cursor := octets[idx]

	// find the trie node
	for {
		// last non-masked octet reached
		if idx == lastOctetIdx {
			return n.getValByPrefix(lastOctet, lastOctetBits)
		}

		// descend down to next level
		c := n.getChild(cursor)
		if c == nil {
			return
		}

		// child is prefix for other node
		if c.pathIsPrefixOrEqual(idx, path) {
			// go down, path compression, idx may jump
			idx = c.pathLen()
			cursor = octets[idx]
			n = c
			continue
		}
		return
	}
}

// Lookup does a route lookup (longest prefix match) for IP and
// returns the associated value and true, or false if no route matched.
func (t *Table2[V]) Lookup(ip netip.Addr) (val V, ok bool) {
	if !ip.IsValid() {
		return
	}
	is4 := ip.Is4()
	n := t.rootNodeByVersion(is4)
	if n == nil {
		return
	}

	// do not allocate
	octets := make([]byte, 16)
	octets = ipToOctets(octets, ip, is4)

	lastOctetIdx := len(octets) - 1

	// stack of the traversed nodes for fast backtracking, if needed
	stack := [maxTreeDepth]struct {
		node  *node2[V]
		octet byte
	}{}

	// stackIndex is monotonic
	var stackIdx int

	// idx is allowed to make jumps due to path compression
	var idx int

	// current octet in loop
	cursor := octets[idx]

	// find leaf node
	for {
		// insert node and corresponding octet into stacks for backtracking
		stack[stackIdx].node = n
		stack[stackIdx].octet = cursor

		if idx == lastOctetIdx {
			break
		}

		c := n.getChild(cursor)

		// end of childs
		if c == nil {
			break
		}

		// c is no prefix for octets
		if !c.pathIsPrefixOrEqual(idx, octets) {
			break
		}

		// path compression, allowed to make jumps
		idx = c.pathLen()

		stackIdx++
		cursor = octets[idx]
		n = c
	}

	// start backtracking in tight loop
	for {
		if _, val, ok := n.lpmByIndex(octetToBaseIndex(cursor)); ok {
			return val, true
		}

		// next round?
		if stackIdx == 0 {
			return
		}
		stackIdx--

		cursor = stack[stackIdx].octet
		n = stack[stackIdx].node
	}
}

// LookupPrefix does a route lookup (longest prefix match) for pfx and
// returns the associated value and true, or false if no route matched.
func (t *Table2[V]) LookupPrefix(pfx netip.Prefix) (val V, ok bool) {
	if !pfx.IsValid() {
		return
	}
	_, _, val, ok = t.lpmByPrefix(pfx)
	return
}

// LookupPrefixLPM is similar to [Table.LookupPrefix],
// but it returns the lpm prefix in addition to value,ok.
//
// This method is about 20-30% slower than LookupPrefix and should only
// be used if the matching lpm entry is also required for other reasons.
//
// If LookupPrefixLPM is to be used for IP addresses,
// they must be converted to /32 or /128 prefixes.
func (t *Table2[V]) LookupPrefixLPM(pfx netip.Prefix) (lpm netip.Prefix, val V, ok bool) {
	if !pfx.IsValid() {
		return
	}
	depth, baseIdx, val, ok := t.lpmByPrefix(pfx)

	if ok {
		// calculate the mask from baseIdx and depth
		mask := baseIndexToPrefixMask(baseIdx, depth)

		// calculate the lpm from ip and mask
		lpm, _ = pfx.Addr().Prefix(mask)
	}

	return lpm, val, ok
}

func (t *Table2[V]) lpmByPrefix(pfx netip.Prefix) (depth int, baseIdx uint, val V, ok bool) {
	// some needed values, see below
	ip, bits, is4 := pfxToValues(pfx)

	n := t.rootNodeByVersion(is4)
	if n == nil {
		return
	}

	// record path to leaf node
	stack := [maxTreeDepth]struct {
		node  *node2[V]
		octet byte
		bits  int
	}{}

	// do not allocate
	octets := make([]byte, 0, 16)
	octets = ipToOctets(octets, ip, is4)

	// see comment in Insert()
	lastOctetIdx := (bits - 1) / strideLen
	lastOctetBits := bits - (lastOctetIdx * strideLen)

	// path of pfx to lookup
	path := octets[:lastOctetIdx]

	// stackIndex is monotonic
	stackIdx := 0

	// path compression, octet index may jump
	idx := 0

	// start with first octet
	cursor := octets[idx]

	// find the trie node
	for {
		// insert cursor and corresponding node into stacks for path recording
		stack[stackIdx].octet = cursor
		stack[stackIdx].node = n

		// last significant octet reached
		if idx == lastOctetIdx {
			// only the lastOctet has a different prefix len (prefix route)
			stack[stackIdx].bits = lastOctetBits
			break
		}

		// for all other octets it's equal to strideLen (host route)
		stack[stackIdx].bits = strideLen

		// get next child
		c := n.getChild(cursor)

		// no child, stopp
		if c == nil {
			break
		}

		// no match, stopp
		if !c.pathIsPrefixOrEqual(idx, path) {
			break
		}

		// go down, path compression, idx may jump
		idx = c.pathLen()
		cursor = octets[idx]
		n = c

		// increase stackIdx monotonic
		stackIdx++
	}

	// start backtracking with last node and cursor
	for {
		if baseIdx, val, ok := n.lpmByPrefix(cursor, stack[stackIdx].bits); ok {
			return n.pathLen(), baseIdx, val, true
		}

		// if stack is exhausted?
		if stackIdx == 0 {
			return
		}
		stackIdx--

		cursor = stack[stackIdx].octet
		n = stack[stackIdx].node
	}
}

// Subnets, return all prefixes covered by pfx in natural CIDR sort order.
func (t *Table2[V]) Subnets(pfx netip.Prefix) []netip.Prefix {
	if !pfx.IsValid() {
		return nil
	}
	// some needed values, see below
	ip, bits, is4 := pfxToValues(pfx)

	n := t.rootNodeByVersion(is4)
	if n == nil {
		return nil
	}

	octets := make([]byte, 0, 16)
	octets = ipToOctets(octets, ip, is4)

	// see comment in Insert()
	lastOctetIdx := (bits - 1) / strideLen
	lastOctet := octets[lastOctetIdx]
	lastOctetBits := bits - (lastOctetIdx * strideLen)

	// path of pfx to get all subnets
	path := octets[:lastOctetIdx]

	idx := 0
	cursor := octets[idx]

	// search prefixes and child below this stride index
	parentIndex := prefixToBaseIndex(lastOctet, lastOctetBits)

	// find the trie node
	for {
		if idx == lastOctetIdx {
			result := n.subnetsRec2(parentIndex)

			slices.SortFunc(result, cmpPrefix)
			return result
		}

		// descend down to next level
		c := n.getChild(cursor)
		if c == nil {
			return nil
		}

		// is child is prefix in search path?
		if c.pathIsPrefixOrEqual(idx, path) {
			// go down, path compression, idx may jump
			idx = c.pathLen()
			cursor = octets[idx]
			n = c
			continue
		}

		// this search path is not in the trie,
		// make temp new search node with this pfx
		search := newNode2[V](path, n.is4)

		// if search is prefix for child node?
		if search.pathIsPrefixOrEqual(idx, c.pathAsSlice()) {
			// insert child into temp search node
			idx = search.pathLen()
			octet := c.pathAsSlice()[idx]
			search.insertChild(octet, c)

			// subnet search, starting at this tmp search node
			result := search.subnetsRec2(parentIndex)

			slices.SortFunc(result, cmpPrefix)

			return result
		}

		return nil
	}
}

// Supernets, return all matching routes for pfx,
// in natural CIDR sort order.
func (t *Table2[V]) Supernets(pfx netip.Prefix) []netip.Prefix {
	if !pfx.IsValid() {
		return nil
	}
	var result []netip.Prefix

	// some needed values, see below
	ip, bits, is4 := pfxToValues(pfx)

	n := t.rootNodeByVersion(is4)
	if n == nil {
		return nil
	}

	octets := make([]byte, 0, 16)
	octets = ipToOctets(octets, ip, is4)

	// see comment in Insert()
	lastOctetIdx := (bits - 1) / strideLen
	lastOctet := octets[lastOctetIdx]
	lastOctetBits := bits - (lastOctetIdx * strideLen)

	// path of pfx to get all supernets
	path := octets[:lastOctetIdx]

	idx := 0
	cursor := octets[idx]

	for {
		// stop condition, last octet
		if idx == lastOctetIdx {
			// make an all-prefix-match at last level
			result = append(result, n.apmByPrefix(lastOctet, lastOctetBits)...)
			break
		}

		// make an all-prefix-match at intermediate level for cursor and strideLen
		result = append(result, n.apmByPrefix(cursor, strideLen)...)

		// descend down to next trie level
		c := n.getChild(cursor)
		if c == nil {
			break
		}

		// is child is prefix in search path?
		if c.pathIsPrefixOrEqual(idx, path) {
			// go down, path compression, idx may jump
			idx = c.pathLen()
			cursor = octets[idx]
			n = c
			continue
		}

		break
	}

	return result
}

// OverlapsPrefix reports whether any IP in pfx matches a route in the table.
func (t *Table2[V]) OverlapsPrefix(pfx netip.Prefix) bool {
	if !pfx.IsValid() {
		return false
	}
	// some needed values, see below
	ip, bits, is4 := pfxToValues(pfx)

	// get the root node of the routing table
	n := t.rootNodeByVersion(is4)
	if n == nil {
		return false
	}

	octets := make([]byte, 0, 16)
	octets = ipToOctets(octets, ip, is4)

	// see comment in Insert()
	lastOctetIdx := (bits - 1) / strideLen
	lastOctet := octets[lastOctetIdx]
	lastOctetBits := bits - (lastOctetIdx * strideLen)

	// path of pfx for overlaps
	path := octets[:lastOctetIdx]

	idx := 0
	cursor := octets[idx]

	for {
		if idx == lastOctetIdx {
			return n.overlapsPrefix(lastOctet, lastOctetBits)
		}

		// still in the middle of prefix chunks
		// test if any route overlaps prefix´
		if _, _, ok := n.lpmByOctet(cursor); ok {
			return true
		}

		// no overlap so far, go down to next c
		c := n.getChild(cursor)
		if c == nil {
			return false
		}

		// is child is prefix in search path?
		if c.pathIsPrefixOrEqual(idx, path) {
			// go down, path compression, idx may jump
			idx = c.pathLen()
			cursor = octets[idx]
			n = c
			continue
		}

		// dummy generic value for insertPrefix
		var zeroV V

		// make temp search node with this pfx
		search := newNode2[V](path, n.is4)
		search.insertPrefix(lastOctet, lastOctetBits, zeroV)

		// if search is prefix for child node?
		if search.pathIsPrefixOrEqual(idx, c.pathAsSlice()) {
			idx = search.pathLen()
			octet := c.pathAsSlice()[idx]

			// pfx overlaps this child octet?
			_, _, ok := search.lpmByOctet(octet)
			return ok
		}

		return false
	}
}

// Overlaps reports whether any IP in the table matches a route in the
// other table.
func (t *Table2[V]) Overlaps(o *Table2[V]) bool {
	t.init()
	o.init()

	// negates the result of OverlapsPrefix, stop recursion on first overlap
	yield := func(pfx netip.Prefix) bool {
		return !t.OverlapsPrefix(pfx)
	}

	// The algorithm works most efficiently when table t is the larger of the two tables
	if t.sizeV4 < o.sizeV4 {
		t, o = o, t
	}

	// return on first overlap, re-negate the result
	if !o.rootV4.toplevelSupernetsRec(yield) {
		return true
	}

	// The algorithm works most efficiently when table t is the larger of the two tables
	if t.sizeV6 < o.sizeV6 {
		t, o = o, t
	}

	return !o.rootV6.toplevelSupernetsRec(yield)
}

// Union combines two tables, changing the receiver table.
// If there are duplicate entries, the value is taken from the other table.
func (t *Table2[V]) Union(o *Table2[V]) {
	t.init()
	o.init()

	// unionRec is too complex for path compressed nodes
	// just walk over all nodes in o and insert pfx/val into t.
	o.All(func(pfx netip.Prefix, val V) bool {
		t.Insert(pfx, val)
		return true
	})
}

// Clone returns a copy of the routing table.
// The payloads V are copied using assignment, so this is a shallow clone.
func (t *Table2[V]) Clone() *Table2[V] {
	t.init()

	c := new(Table2[V])
	c.init()

	c.rootV4 = t.rootV4.cloneRec()
	c.rootV6 = t.rootV6.cloneRec()

	return c
}

// All may be used in a for/range loop to iterate
// through all the prefixes.
//
// Prefixes must not be inserted or deleted during iteration, otherwise
// the behavior is undefined. However, value updates are permitted.
//
// The iteration order is not specified and is not part of the
// public interface, you must not rely on it.
func (t *Table2[V]) All(yield func(pfx netip.Prefix, val V) bool) {
	t.init()
	// respect premature end of allRec()
	_ = t.rootV4.allRec2(yield) && t.rootV6.allRec2(yield)
}

// All4, like [Table.All] but only for the v4 routing table.
func (t *Table2[V]) All4(yield func(pfx netip.Prefix, val V) bool) {
	t.init()
	t.rootV4.allRec2(yield)
}

// All6, like [Table.All] but only for the v6 routing table.
func (t *Table2[V]) All6(yield func(pfx netip.Prefix, val V) bool) {
	t.init()
	t.rootV6.allRec2(yield)
}

// Size returns the sum of the IPv4 and IPv6 refixes.
func (t *Table2[V]) Size() int {
	t.init()
	return t.sizeV4 + t.sizeV6
}

// Size4 returns the number of IPv4 refixes.
func (t *Table2[V]) Size4() int {
	t.init()
	return t.sizeV4
}

// Size6 returns the number of IPv6 refixes.
func (t *Table2[V]) Size6() int {
	t.init()
	return t.sizeV6
}

func (t *Table2[V]) incDecSize(val int, is4 bool) {
	if is4 {
		t.sizeV4 = t.sizeV4 + val
	} else {
		t.sizeV6 = t.sizeV6 + val
	}
}
