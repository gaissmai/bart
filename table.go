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
type Table[V any] struct {
	rootV4 *node[V]
	rootV6 *node[V]

	// BitSets have to be initialized.
	initOnce sync.Once
}

// init BitSets once, so no constructor is needed
func (t *Table[V]) init() {
	t.initOnce.Do(func() {
		t.rootV4 = newNode[V]()
		t.rootV6 = newNode[V]()
	})
}

// rootNodeByVersion, select root node for ip version.
func (t *Table[V]) rootNodeByVersion(is4 bool) *node[V] {
	if is4 {
		return t.rootV4
	}
	return t.rootV6
}

// Insert adds pfx to the tree, with value val.
// If pfx is already present in the tree, its value is set to val.
func (t *Table[V]) Insert(pfx netip.Prefix, val V) {
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

	for _, octet := range octets {

		// loop stop condition: last significant octet reached
		//
		// insert the (octet, bits, val) into the node at this depth
		//
		// 8.0.0.0/5 ->       depth 0, octet 8,   bits 5
		// 10.0.0.0/8 ->      depth 0, octet 10,  bits 8
		// 192.168.0.0/16  -> depth 1, octet 168, bits 8, (16-1*8 = 8)
		// 192.168.20.0/19 -> depth 2, octet 20,  bits 3, (19-2*8 = 3)
		// 172.16.19.12/32 -> depth 3, octet 12,  bits 8, (32-3*8 = 8)
		//
		if bits <= strideLen {
			n.insertPrefix(octet, bits, val)
			return
		}

		// descend down to next trie level
		child := n.getChild(octet)

		// create and insert missing intermediate child, no path compression!
		if child == nil {
			child = newNode[V]()
			n.insertChild(octet, child)
		}

		// go down
		n = child
		bits -= strideLen
	}
}

// Delete removes pfx from the tree, pfx does not have to be present.
func (t *Table[V]) Delete(pfx netip.Prefix) {
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
	pathStack := [maxTreeDepth]*node[V]{}

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
func (t *Table[V]) Update(pfx netip.Prefix, cb func(val V, ok bool) V) V {
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

	for _, octet := range octets {
		// last significant octet reached
		if bits <= strideLen {
			return n.updatePrefix(octet, bits, cb)
		}

		// descend down to next trie level
		child := n.getChild(octet)

		// create and insert missing intermediate child, no path compression!
		if child == nil {
			child = newNode[V]()
			n.insertChild(octet, child)
		}

		// go down
		n = child
		bits -= strideLen
	}

	panic("unreachable")
}

// Get returns the associated payload for prefix and true, or false if
// prefix is not set in the routing table.
func (t *Table[V]) Get(pfx netip.Prefix) (val V, ok bool) {
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

	for _, octet := range octets {

		// last significant octet reached
		if bits <= strideLen {
			return n.getValByPrefix(octet, bits)
		}

		// descend down to next level, no path compression
		child := n.getChild(octet)
		if child == nil {
			break
		}

		// go down
		bits -= strideLen
		n = child
	}

	// prefix does not exist
	return
}

// Lookup does a route lookup (longest prefix match) for IP and
// returns the associated value and true, or false if no route matched.
func (t *Table[V]) Lookup(ip netip.Addr) (val V, ok bool) {
	is4 := ip.Is4()
	n := t.rootNodeByVersion(is4)
	if n == nil {
		return
	}

	// stack of the traversed nodes for fast backtracking, if needed
	pathStack := [maxTreeDepth]*node[V]{}

	// does not allocate
	a16 := ip.As16()
	octets := a16[:]
	if is4 {
		octets = octets[12:]
	}

	var depth int
	var octet byte

	// find leaf node
	for depth, octet = range octets {

		// push current node on stack for fast backtracking
		pathStack[depth] = n

		// go down in tight loop to leaf node
		if child := n.getChild(octet); child != nil {
			n = child
			continue
		}

		break
	}

	// start backtracking at leaf node in tight loop
	for {
		// lookup only in nodes with prefixes, skip over intermediate nodes
		if len(n.prefixes) != 0 {
			// longest prefix match?
			if _, val, ok := n.lpmByIndex(octetToBaseIndex(octet)); ok {
				return val, true
			}
		}

		// end condition, stack is exhausted
		if depth == 0 {
			return
		}

		// go up, backtracking
		depth--
		octet = octets[depth]
		n = pathStack[depth]
	}
}

// LookupPrefix does a route lookup (longest prefix match) for pfx and
// returns the associated value and true, or false if no route matched.
func (t *Table[V]) LookupPrefix(pfx netip.Prefix) (val V, ok bool) {
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
func (t *Table[V]) LookupPrefixLPM(pfx netip.Prefix) (lpm netip.Prefix, val V, ok bool) {
	depth, baseIdx, val, ok := t.lpmByPrefix(pfx)

	// calculate the mask from baseIdx and depth
	mask := baseIndexToPrefixMask(baseIdx, depth)

	// calculate the lpm from ip and mask
	lpm, _ = pfx.Addr().Prefix(mask)

	return lpm, val, ok
}

func (t *Table[V]) lpmByPrefix(pfx netip.Prefix) (depth int, baseIdx uint, val V, ok bool) {
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
		baseIdx = prefixToBaseIndex(0, 0)
		val, ok = n.getValByIndex(baseIdx)
		return
	}

	// stack of the traversed nodes for fast backtracking, if needed
	pathStack := [maxTreeDepth]*node[V]{}

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

// Subnets, return all prefixes covered by pfx in natural CIDR sort order.
func (t *Table[V]) Subnets(pfx netip.Prefix) []netip.Prefix {
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

// Supernets, return all matching routes for pfx,
// in natural CIDR sort order.
func (t *Table[V]) Supernets(pfx netip.Prefix) []netip.Prefix {
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

// OverlapsPrefix reports whether any IP in pfx matches a route in the table.
func (t *Table[V]) OverlapsPrefix(pfx netip.Prefix) bool {
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

// Overlaps reports whether any IP in the table matches a route in the
// other table.
func (t *Table[V]) Overlaps(o *Table[V]) bool {
	t.init()
	o.init()

	return t.rootV4.overlapsRec(o.rootV4) || t.rootV6.overlapsRec(o.rootV6)
}

// Union combines two tables, changing the receiver table.
// If there are duplicate entries, the value is taken from the other table.
func (t *Table[V]) Union(o *Table[V]) {
	t.init()
	o.init()

	t.rootV4.unionRec(o.rootV4)
	t.rootV6.unionRec(o.rootV6)
}

// Clone returns a copy of the routing table.
// The payloads V are copied using assignment, so this is a shallow clone.
func (t *Table[V]) Clone() *Table[V] {
	t.init()

	c := new(Table[V])
	c.init()

	c.rootV4 = t.rootV4.cloneRec()
	c.rootV6 = t.rootV6.cloneRec()

	return c
}

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
func (t *Table[V]) Walk(cb func(pfx netip.Prefix, val V) error) error {
	t.init()

	if err := t.Walk4(cb); err != nil {
		return err
	}

	return t.Walk6(cb)
}

// Walk4, like [Table.Walk] but only for the v4 routing table.
func (t *Table[V]) Walk4(cb func(pfx netip.Prefix, val V) error) error {
	t.init()
	return t.rootV4.walkRec(nil, true, cb)
}

// Walk6, like [Table.Walk] but only for the v6 routing table.
func (t *Table[V]) Walk6(cb func(pfx netip.Prefix, val V) error) error {
	t.init()
	return t.rootV6.walkRec(nil, false, cb)
}
