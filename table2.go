// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"bytes"
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
		t.rootV4 = newNode2[V]()
		t.rootV6 = newNode2[V]()
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

	// always normalize the prefix
	pfx = pfx.Masked()

	// some needed values, see below
	bits := pfx.Bits()
	ip := pfx.Addr()
	is4 := ip.Is4()

	// get the root node of the routing table
	n := t.rootNodeByVersion(is4)

	// insert default route, easy peasy
	if bits == 0 {
		n.insertPrefix(0, 0, val)
		return
	}

	a16 := ip.As16()
	octets := a16[:]
	if is4 {
		octets = octets[12:]
	}

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

	// find the proper trie node to insert prefix
	for octetIdx, octet := range octets {

		// last octet reached
		if octetIdx == lastOctetIdx {
			// insert prefix into node
			n.insertPrefix(lastOctet, lastOctetBits, val)
			return
		}

		// descend down the trie
		child := n.getChild(octet)

		// child not nil and not path compressed, tight loop
		if child != nil && child.pathLen() == octetIdx+1 {
			n = child
			continue
		}

		// #########################################
		//  path compression, here will be dragons!
		// #########################################

		// make new node, already set path and insert prefix
		other := newNode2[V]()
		other.pathSet(octets[:lastOctetIdx])
		other.insertPrefix(lastOctet, lastOctetBits, val)

		// just insert new leaf node
		if child == nil {
			n.insertChild(octet, other)
			return
		}

		// just insert prefix into existing child
		if child.pathEqual(other) {
			child.insertPrefix(lastOctet, lastOctetBits, val)
			return
		}

		// other is prefix for child
		if child.pathHasPrefix(other) {
			commonPathIdx := child.commonPathIdx(octetIdx, other)
			nextOctet := child.pathAsSlice()[commonPathIdx+1]

			// move current child under new node
			other.insertChild(nextOctet, child)

			// link new node under current n
			n.insertChild(octet, other)
			return
		}

		// child is prefix for other
		if other.pathHasPrefix(child) {
			// is there already a child under octet slot
			if next := child.getChild(octet); next != nil {
				// just insert the prefix
				next.insertPrefix(lastOctet, lastOctetBits, val)
				return
			}
			// insert other
			child.insertChild(octet, other)
			return
		}

		// from here we need a new intermediate node

		// find the idx until they differ, insert intermediate node,
		// insert old and new child into intermdiate node
		commonPathIdx := child.commonPathIdx(octetIdx, other)

		// make intermediate node with path until divergence
		imed := newNode2[V]()
		imed.pathSet(child.pathAsSlice()[:commonPathIdx+1])

		// insert old and new child into intermediate node
		imed.insertChild(child.pathAsSlice()[commonPathIdx+1], child)
		imed.insertChild(other.pathAsSlice()[commonPathIdx+1], other)

		// insert intermediate node into n, overwrites the old link
		n.insertChild(octet, imed)
		return
	}
}

// findCommonPathIdx until they differ, but we know they must be equal until start.
func findCommonPathIdx(start int, a, b []byte) int {
	idx := start
	for i := start; i < min(len(a), len(b)); i++ {
		if a[i] != b[i] {
			return idx
		}
		idx = i
	}
	return idx
}

// TODO path compressed algo
// Delete removes pfx from the tree, pfx does not have to be present.
func (t *Table2[V]) Delete(pfx netip.Prefix) {
	// always normalize the prefix
	pfx = pfx.Masked()

	bits := pfx.Bits()
	ip := pfx.Addr()
	is4 := ip.Is4()

	n := t.rootNodeByVersion(is4)
	if n == nil {
		return
	}

	// delete default route, easy peasy
	if bits == 0 {
		n.deletePrefix(0, 0)
		return
	}

	// stack of the traversed child path in order to
	// purge dangling paths after deletion
	pathStack := [maxTreeDepth]*node2[V]{}

	// does not allocate
	a16 := ip.As16()
	octets := a16[:]
	if is4 {
		octets = octets[12:]
	}

	var depth int
	var octet byte

	for depth, octet = range octets {
		// push current node on stack for path recording
		pathStack[depth] = n

		// last significant octet reached
		if bits <= strideLen {
			// found a child on proper depth ...
			if !n.deletePrefix(octet, bits) {
				// ... but prefix not in tree, nothing deleted
				return
			}

			// purge dangling path, if needed
			break
		}

		// descend down to next level, no path compression
		child := n.getChild(octet)
		if child == nil {
			// no child, nothing to delete
			return
		}

		// go down
		bits -= strideLen
		n = child
	}

	// check for dangling path
	for {

		// loop stop condition
		if depth == 0 {
			break
		}

		// an empty node?
		if n.isEmpty() {
			// purge this node from parents children
			parent := pathStack[depth-1]
			parent.deleteChild(octets[depth-1])
		}

		// go up
		depth--
		n = pathStack[depth]
	}
}

// Update or set the value at pfx with a callback function.
// The callback function is called with (value, ok) and returns a new value..
//
// If the pfx does not already exist, it is set with the new value.
func (t *Table2[V]) Update(pfx netip.Prefix, cb func(val V, ok bool) V) V {
	t.init()

	// always normalize the prefix
	pfx = pfx.Masked()

	// some needed values, see below
	bits := pfx.Bits()
	ip := pfx.Addr()
	is4 := ip.Is4()

	// get the root node of the routing table
	n := t.rootNodeByVersion(is4)

	// update default route, easy peasy
	if bits == 0 {
		return n.updatePrefix(0, 0, cb)
	}

	// does not allocate
	a16 := ip.As16()
	octets := a16[:]
	if is4 {
		octets = octets[12:]
	}

	lastOctetIdx := (bits - 1) / strideLen
	lastOctet := octets[lastOctetIdx]
	lastOctetBits := bits - (lastOctetIdx * strideLen)

	// find the proper trie node to update or insert prefix
	for octetIdx, octet := range octets {

		// last octet reached
		if octetIdx == lastOctetIdx {
			return n.updatePrefix(octet, lastOctetBits, cb)
		}

		// descend down to next trie level
		c := n.getChild(octet)

		// child not nil and not path compressed, tight loop
		if c != nil && c.pathLen() == octetIdx+1 {
			n = c
			continue
		}

		// from here on pretty similar to Insert()

		// #########################################
		//  path compression, here will be dragons!
		// #########################################

		// make new node and already set path and update prefix
		nn := newNode2[V]()
		nn.pathSet(octets[:lastOctetIdx])
		val := nn.updatePrefix(lastOctet, lastOctetBits, cb)

		// just insert new leaf node
		if c == nil {
			n.insertChild(octet, nn)
			return val
		}

		// just insert prefix into existing child
		if c.pathEqual(nn) {
			return c.updatePrefix(lastOctet, lastOctetBits, cb)
		}

		// octets[...] is prefix for c.path
		if c.pathHasPrefix(nn) {

			// move current child under new node
			nn.insertChild(lastOctet, c)

			// link new node under current n
			n.insertChild(octet, nn)
			return val
		}

		// TODO TODO
		// from here we need a new intermediate node

		// find the idx until they differ, insert intermediate node,
		// insert old and new child into intermdiate node
		commonPathIdx := c.commonPathIdx(octetIdx, nn)

		// make intermediate node with path until divergence
		imedNode := newNode2[V]()
		imedNode.pathSet(c.pathAsSlice()[:commonPathIdx])

		// insert old and new child into intermediate node
		imedNode.insertChild(c.pathAsSlice()[commonPathIdx+1], c)
		imedNode.insertChild(octets[commonPathIdx+1], nn)

		// insert intermediate node into n, overwrites the old link
		n.insertChild(octet, imedNode)
		return val
	}

	panic("unreachable")
}

// Get returns the associated payload for prefix and true, or false if
// prefix is not set in the routing table.
func (t *Table2[V]) Get(pfx netip.Prefix) (val V, ok bool) {
	// always normalize the prefix
	pfx = pfx.Masked()

	bits := pfx.Bits()
	ip := pfx.Addr()
	is4 := ip.Is4()

	n := t.rootNodeByVersion(is4)
	if n == nil {
		return
	}

	// get value for default route, easy peasy
	if bits == 0 {
		return n.getValByPrefix(0, 0)
	}

	// does not allocate
	a16 := ip.As16()
	octets := a16[:]
	if is4 {
		octets = octets[12:]
	}

	lastOctetIdx := (bits - 1) / strideLen
	lastOctet := octets[lastOctetIdx]
	lastOctetBits := bits - (lastOctetIdx * strideLen)

	for octetIdx, octet := range octets {

		// last non-masked octet reached
		if octetIdx == lastOctetIdx {
			return n.getValByPrefix(octet, bits)
		}

		// descend down to next level
		c := n.getChild(octet)
		if c == nil {
			return
		}

		if bytes.Equal(c.pathAsSlice(), octets[:lastOctetIdx]) {
			return c.getValByPrefix(lastOctet, lastOctetBits)
		}

		n = c
	}

	panic("unreachable")
}

// Lookup does a route lookup (longest prefix match) for IP and
// returns the associated value and true, or false if no route matched.
func (t *Table2[V]) Lookup(ip netip.Addr) (val V, ok bool) {
	is4 := ip.Is4()
	n := t.rootNodeByVersion(is4)
	if n == nil {
		return
	}

	// stacks of the traversed nodes for fast backtracking, if needed
	nodeStack := [maxTreeDepth]*node2[V]{}
	octetStack := [maxTreeDepth]byte{}

	// does not allocate
	a16 := ip.As16()
	octets := a16[:]
	if is4 {
		octets = octets[12:]
	}

	// octetIdx is allowed to make jumps due to path compression
	octetIdx := 0

	// stackIndex is monotonic
	stackIdx := 0

	// find leaf node
	for {
		octet := octets[octetIdx]

		// insert node and corresponding octet into stacks for backtracking
		nodeStack[stackIdx] = n
		octetStack[stackIdx] = octet

		// end of childs
		c := n.getChild(octet)
		if c == nil {
			break
		}

		// octets does not overlap c.path
		if !bytes.HasPrefix(octets, c.pathAsSlice()) {
			break
		}

		// path compression, allowed to make jumps
		octetIdx = c.pathLen()

		stackIdx++
		n = c
	}

	// start backtracking at leaf node in tight loop
	for {
		n := nodeStack[stackIdx]
		octet := octetStack[stackIdx]

		// LPM lookup only in nodes with prefixes, skip over intermediate nodes
		if len(n.prefixes) != 0 {
			if _, val, ok := n.lpmByIndex(octetToBaseIndex(octet)); ok {
				return val, true
			}
		}

		// next round?
		if stackIdx == 0 {
			break
		}
		stackIdx--
	}

	return
}

// TODO path compressed algo
// LookupPrefix does a route lookup (longest prefix match) for pfx and
// returns the associated value and true, or false if no route matched.
func (t *Table2[V]) LookupPrefix(pfx netip.Prefix) (val V, ok bool) {
	_, _, val, ok = t.lpmByPrefix(pfx)
	return
}

// TODO path compressed algo
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

// TODO path compressed algo
func (t *Table2[V]) lpmByPrefix(pfx netip.Prefix) (depth int, baseIdx uint, val V, ok bool) {
	// always normalize the prefix
	pfx = pfx.Masked()

	bits := pfx.Bits()
	ip := pfx.Addr()
	is4 := ip.Is4()
	n := t.rootNodeByVersion(is4)
	if n == nil {
		return
	}

	// pfx is default route, easy peasy
	if bits == 0 {
		val, ok = n.getValByPrefix(0, 0)
		return
	}

	// stack of the traversed nodes for fast backtracking, if needed
	pathStack := [maxTreeDepth]*node2[V]{}

	// does not allocate
	a16 := ip.As16()
	octets := a16[:]
	if is4 {
		octets = octets[12:]
	}

	var octet byte

	// go down to stride node for pfx
	for depth, octet = range octets {

		// push current node on stack for fast backtracking
		pathStack[depth] = n

		// last significant octet reached
		if bits <= strideLen {
			break
		}

		// go down in tight loop the trie levels
		if child := n.getChild(octet); child != nil {
			bits -= strideLen
			n = child
			continue
		}

		// stop condition was missing child and not bits len,
		// so cut the bits to strideLen
		bits = strideLen

		break
	}

	// start backtracking at matching stride node in tight loop
	for {

		// lookup only in nodes with prefixes, skip over intermediate nodes
		if len(n.prefixes) != 0 {
			if baseIdx, val, ok := n.lpmByPrefix(octet, bits); ok {
				return depth, baseIdx, val, true
			}
		}

		// end condition, stack is exhausted
		if depth == 0 {
			return
		}

		// go up, backtracking
		// bits are now full strideLen for all upper levels
		depth--
		bits = strideLen
		octet = octets[depth]
		n = pathStack[depth]
	}
}

// TODO path compressed algo
// Subnets, return all prefixes covered by pfx in natural CIDR sort order.
func (t *Table2[V]) Subnets(pfx netip.Prefix) []netip.Prefix {
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

		// return *all* routes for this IP version, sic!
		_ = n.walkRec(nil, is4, func(pfx netip.Prefix, _ V) (err error) {
			result = append(result, pfx)
			return
		})

		// walk order is wierd, needed sort after walk
		slices.SortFunc(result, cmpPrefix)

		return result
	}

	// heap allocation does not matter here
	octets := ip.AsSlice()

	for depth, octet := range octets {
		// last significant octet reached
		if bits <= strideLen {
			result := n.subnets(octets[:depth], prefixToBaseIndex(octet, bits), is4)

			slices.SortFunc(result, cmpPrefix)
			return result
		}

		// descend down to next trie level
		child := n.getChild(octet)

		// stop condition, found no matching stride node
		if child == nil {
			return nil
		}

		// next round
		n = child
		bits -= strideLen
	}

	return result
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

	a16 := ip.As16()
	octets := a16[:]
	if is4 {
		octets = octets[12:]
	}

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

	// does not allocate
	a16 := ip.As16()
	octets := a16[:]
	if is4 {
		octets = octets[12:]
	}

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

// TODO path compressed algo
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

// TODO path compressed algo
// Walk runs through the routing table and calls the cb function
// for each route entry with prefix and value.
// If the cb function returns an error,
// the walk ends prematurely and the error is propagated.
//
// Prefixes must not be inserted or deleted by the callback function, otherwise
// the behavior is undefined. However, value updates are permitted.
//
// The walk order is not specified and is not part of the
// public interface, you must not rely on it.
func (t *Table2[V]) Walk(cb func(pfx netip.Prefix, val V) error) error {
	t.init()

	if err := t.Walk4(cb); err != nil {
		return err
	}

	return t.Walk6(cb)
}

// TODO path compressed algo
// Walk4, like [Table.Walk] but only for the v4 routing table.
func (t *Table2[V]) Walk4(cb func(pfx netip.Prefix, val V) error) error {
	t.init()
	return t.rootV4.walkRec(nil, true, cb)
}

// TODO path compressed algo
// Walk6, like [Table.Walk] but only for the v6 routing table.
func (t *Table2[V]) Walk6(cb func(pfx netip.Prefix, val V) error) error {
	t.init()
	return t.rootV6.walkRec(nil, false, cb)
}
