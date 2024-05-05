// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"slices"
	"sync"
)

// Table is an IPv4 and IPv6 routing table with payload V.
// The zero value is ready to use.
//
// The Table is safe for concurrent readers but not for
// concurrent readers and/or writers.
type Table2[V any] struct {
	rootV4 *node2[V]
	rootV6 *node2[V]

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

	// some needed values, see below
	_, ip, bits, is4 := pfxToValues(pfx)

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
			n.insertPrefix(lastOctet, lastOctetBits, val)
			return
		}

		// descend down the trie
		c := n.getChild(cursor)

		// just insert other node as new leaf
		if c == nil {
			// make new node
			o := newNode2[V](path, n.is4)
			o.insertPrefix(lastOctet, lastOctetBits, val)

			n.insertChild(cursor, o)
			return
		}

		// child is prefix for other node
		if c.pathIsPrefixOrEqual(path) {
			// go down, path compression -> idx may jump
			idx = c.pathLen()
			cursor = octets[idx]
			n = c
			continue
		}

		// make new node
		o := newNode2[V](path, n.is4)
		o.insertPrefix(lastOctet, lastOctetBits, val)

		// other is prefix for child node
		if o.pathIsPrefixOrEqual(c.pathAsSlice()) {
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
	// some needed values, see below
	_, ip, bits, is4 := pfxToValues(pfx)

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
			if !n.deletePrefix(lastOctet, lastOctetBits) {
				// ... but prefix not in tree, nothing deleted
				return
			}

			// escape, but purge dangling path if needed, see below
			break
		}

		// descend down to next level
		c := n.getChild(cursor)

		// no child, stopp
		if c == nil {
			return
		}

		// no match, stopp
		if !c.pathIsPrefixOrEqual(path) {
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

	// some needed values, see below
	_, ip, bits, is4 := pfxToValues(pfx)

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
			return n.updatePrefix(lastOctet, lastOctetBits, cb)
		}

		// descend down the trie
		c := n.getChild(cursor)

		// just insert other node as new leaf
		if c == nil {
			// make new node, already set path and insert prefix
			o := newNode2[V](path, n.is4)
			n.insertChild(cursor, o)

			return o.updatePrefix(lastOctet, lastOctetBits, cb)
		}

		// child is prefix or equal to other node
		if c.pathIsPrefixOrEqual(path) {
			// go down, path compression, idx may jump
			idx = c.pathLen()
			cursor = octets[idx]
			n = c
			continue
		}

		o := newNode2[V](path, n.is4)

		// other is prefix for child node
		if o.pathIsPrefixOrEqual(c.pathAsSlice()) {
			// splice other between n and child: n -> other -> child
			// path compression, idx may jump
			idx = o.pathLen()

			o.insertChild(c.pathAsSlice()[idx], c)
			n.insertChild(cursor, o)

			return o.updatePrefix(lastOctet, lastOctetBits, cb)
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
		return o.updatePrefix(lastOctet, lastOctetBits, cb)
	}
}

// Get returns the associated payload for prefix and true, or false if
// prefix is not set in the routing table.
func (t *Table2[V]) Get(pfx netip.Prefix) (val V, ok bool) {
	// some needed values, see below
	_, ip, bits, is4 := pfxToValues(pfx)

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
		if c.pathIsPrefixOrEqual(path) {
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
		if !c.pathIsPrefixOrEqual(octets) {
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
	_, ip, bits, is4 := pfxToValues(pfx)

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
		if !c.pathIsPrefixOrEqual(path) {
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
	// some needed values, see below
	_, ip, bits, is4 := pfxToValues(pfx)

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
			result := n.subnetsRec(parentIndex)

			slices.SortFunc(result, cmpPrefix)
			return result
		}

		// descend down to next level
		c := n.getChild(cursor)
		if c == nil {
			return nil
		}

		// is child is prefix in search path?
		if c.pathIsPrefixOrEqual(path) {
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
		if search.pathIsPrefixOrEqual(c.pathAsSlice()) {
			// insert child into temp search node
			idx = search.pathLen()
			octet := c.pathAsSlice()[idx]
			search.insertChild(octet, c)

			// subnet search, starting at this tmp search node
			result := search.subnetsRec(parentIndex)

			slices.SortFunc(result, cmpPrefix)

			return result
		}

		return nil
	}
}

// TODO path compressed algo
// Supernets, return all matching routes for pfx,
// in natural CIDR sort order.
func (t *Table2[V]) Supernets(pfx netip.Prefix) []netip.Prefix {
	var result []netip.Prefix

	// always normalize the prefix
	pfx = pfx.Masked()

	// some needed values, see below
	bits := pfx.Bits()
	ip := pfx.Addr()
	is4 := ip.Is4()

	n := t.rootNodeByVersion(is4)
	if n == nil {
		return nil
	}

	// pfx is default route
	if bits == 0 {
		// test if default route is in table
		if _, ok := n.getValByPrefix(0, 0); ok {
			result = append(result, pfx)
		}
		return result
	}

	octets := make([]byte, 0, 16)
	octets = ipToOctets(octets, ip, is4)

	for depth, octet := range octets {
		// max bits in baseIndex functions is strideLen
		pfxLen := strideLen

		// last significant octet reached
		if bits <= strideLen {
			pfxLen = bits
		}

		// make an all-prefix-match at this level
		superIndexes := n.apmByPrefix(octet, pfxLen)

		// get back the matching prefix from baseIdx
		for _, baseIdx := range superIndexes {
			// calc supernet mask
			mask := baseIndexToPrefixMask(baseIdx, depth)

			// calculate the pfx from ip and mask
			superPfx, _ := ip.Prefix(mask)

			result = append(result, superPfx)
		}

		// last significant octet reached
		if bits <= strideLen {
			break
		}

		// descend down to next trie level
		child := n.getChild(octet)

		// stop condition, no more childs
		if child == nil {
			break
		}

		// next round
		n = child
		bits -= strideLen
	}

	return result
}

// TODO path compressed algo
// OverlapsPrefix reports whether any IP in pfx matches a route in the table.
func (t *Table2[V]) OverlapsPrefix(pfx netip.Prefix) bool {
	// always normalize the prefix
	pfx = pfx.Masked()

	// some needed values, see below
	bits := pfx.Bits()
	ip := pfx.Addr()
	is4 := ip.Is4()

	// get the root node of the routing table
	n := t.rootNodeByVersion(is4)
	if n == nil {
		return false
	}

	// do not allocate
	octets := make([]byte, 0, 16)
	octets = ipToOctets(octets, ip, is4)

	for _, octet := range octets {

		// last significant octet reached
		if bits <= strideLen {
			return n.overlapsPrefix(octet, bits)
		}

		// still in the middle of prefix chunks
		// test if any route overlaps prefix´ octet chunk so far

		// but skip intermediate nodes, no routes to test?
		if len(n.prefixes) != 0 {
			if _, _, ok := n.lpmByOctet(octet); ok {
				return true
			}
		}

		// no overlap so far, go down to next child
		child := n.getChild(octet)

		// no more children to explore, there can't be an overlap
		if child == nil {
			return false
		}

		// next round
		bits -= strideLen
		n = child
	}

	return false
}

// TODO path compressed algo
// Overlaps reports whether any IP in the table matches a route in the
// other table.
func (t *Table2[V]) Overlaps(o *Table2[V]) bool {
	t.init()
	o.init()

	return t.rootV4.overlapsRec(o.rootV4) || t.rootV6.overlapsRec(o.rootV6)
}

// TODO path compressed algo
// Union combines two tables, changing the receiver table.
// If there are duplicate entries, the value is taken from the other table.
func (t *Table2[V]) Union(o *Table2[V]) {
	t.init()
	o.init()

	t.rootV4.unionRec(o.rootV4)
	t.rootV6.unionRec(o.rootV6)
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
	_ = t.rootV4.allRec(yield) && t.rootV6.allRec(yield)
}

// All4, like [Table.All] but only for the v4 routing table.
func (t *Table2[V]) All4(yield func(pfx netip.Prefix, val V) bool) {
	t.init()
	t.rootV4.allRec(yield)
}

// All6, like [Table.All] but only for the v6 routing table.
func (t *Table2[V]) All6(yield func(pfx netip.Prefix, val V) bool) {
	t.init()
	t.rootV6.allRec(yield)
}

// ipToOctets, be careful, do not allocate!
// intended use: SA4009: argument octets is overwritten before first use
func ipToOctets(octets []byte, ip netip.Addr, is4 bool) []byte { //nolint:staticcheck
	a16 := ip.As16()
	octets = a16[:] //nolint:staticcheck
	if is4 {
		octets = octets[12:]
	}
	return octets
}

// pfxToValues, a helper function.
func pfxToValues(pfx netip.Prefix) (masked netip.Prefix, ip netip.Addr, bits int, is4 bool) {
	masked = pfx.Masked() // normalized
	bits = pfx.Bits()
	ip = pfx.Addr()
	is4 = ip.Is4()
	return
}
