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
		n.prefixes.insert(0, 0, val)
		return
	}

	// the ip is chunked in bytes, the multibit stride is 8
	bs := ip.AsSlice()

	// depth index for the child trie
	depth := 0
	for {
		addr := uint(bs[depth]) // stride = 8!

		// loop stop condition:
		// last non-masked addr chunk of prefix, insert the
		// byte and bits into the prefixHeap on this depth
		//
		// 8.0.0.0/5 ->       depth 0, addr byte  8,  bits 5
		// 10.0.0.0/8 ->      depth 0, addr byte  10, bits 8
		// 192.168.0.0/16  -> depth 1, addr byte 168, bits 8, (16-1*8 = 8)
		// 192.168.20.0/19 -> depth 2, addr byte  20, bits 3, (19-2*8 = 3)
		// 172.16.19.12/32 -> depth 3, addr byte  12, bits 8, (32-3*8 = 8)
		//
		if bits <= stride {
			n.prefixes.insert(addr, bits, val)
			return
		}

		// descend down to next child level
		child := n.children.get(addr)

		// create and insert missing intermediate child, no path compression!
		if child == nil {
			child = newNode[V]()
			n.children.insert(addr, child)
		}

		// go down
		depth++
		n = child
		bits -= stride
	}
}

// Update or set the value at pfx with a callback function.
//
// If the pfx does not yet exist, then ok=false and the pfx
// is set with the new value.
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
		return n.prefixes.update(0, 0, cb)
	}

	// keep this method alloc free, don't use ip.AsSlice here
	a16 := ip.As16()
	bs := a16[:]
	if is4 {
		bs = bs[12:]
	}

	// depth index for the child trie
	depth := 0
	for {
		addr := uint(bs[depth]) // stride = 8!

		// loop stop condition
		if bits <= stride {
			return n.prefixes.update(addr, bits, cb)
		}

		// descend down to next child level
		child := n.children.get(addr)

		// create and insert missing intermediate child, no path compression!
		if child == nil {
			child = newNode[V]()
			n.children.insert(addr, child)
		}

		// go down
		depth++
		n = child
		bits -= stride
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
		n.prefixes.delete(0, 0)
		return
	}

	// stack of the traversed child path in order to
	// purge dangling paths after deletion
	pathStack := [maxTreeDepth]*node[V]{}

	bs := ip.AsSlice()
	depth := 0
	for {
		addr := uint(bs[depth]) // stride = 8!

		// push current node on stack for path recording
		pathStack[depth] = n

		// last non-masked byte
		if bits <= stride {
			// found a child on proper depth ...
			if !n.prefixes.delete(addr, bits) {
				// ... but prefix not in tree, nothing deleted
				return
			}

			// purge dangling path, if needed
			break
		}

		// descend down to next level, no path compression
		child := n.children.get(addr)
		if child == nil {
			// no child, nothing to delete
			return
		}

		// go down
		depth++
		bits -= stride
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
			parent.children.delete(uint(bs[depth-1]))
		}

		// go up
		depth--
		n = pathStack[depth]
	}
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
		return n.prefixes.getValByPrefix(0, 0)
	}

	// keep this method alloc free, don't use ip.AsSlice here
	a16 := ip.As16()
	bs := a16[:]
	if is4 {
		bs = bs[12:]
	}

	depth := 0
	for {
		addr := uint(bs[depth]) // stride = 8!

		// last non-masked byte
		if bits <= stride {
			return n.prefixes.getValByPrefix(addr, bits)
		}

		// descend down to next level, no path compression
		child := n.children.get(addr)
		if child == nil {
			// no child, prefix is not set
			return
		}

		// go down
		depth++
		bits -= stride
		n = child
	}
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

	// keep the lpm alloc free, don't use ip.AsSlice here
	a16 := ip.As16()
	bs := a16[:]
	if is4 {
		bs = bs[12:]
	}

	depth := 0
	addr := uint(bs[depth]) // bytewise, stride = 8
	// find leaf tree
	for {

		// push current node on stack for fast backtracking
		pathStack[depth] = n

		// go down in tight loop to leaf tree
		if child := n.children.get(addr); child != nil {
			depth++
			addr = uint(bs[depth])
			n = child
			continue
		}

		break
	}

	// start backtracking at leaf node in tight loop
	for {
		// lookup only in nodes with prefixes, skip over intermediate nodes
		if len(n.prefixes.values) != 0 {
			// longest prefix match?
			if _, val, ok := n.prefixes.lpmByIndex(addrToBaseIndex(addr)); ok {
				// return also baseIdx and the depth, needed to
				// calculate the lpm prefix by the Lookup method.
				return val, true
			}
		}

		// end condition, stack is exhausted
		if depth == 0 {
			return
		}

		// go up, backtracking
		depth--
		addr = uint(bs[depth])
		n = pathStack[depth]
	}
}

// Lookup2 is similar to [Table.Lookup], but has a prefix as input parameter
// and returns the lpm prefix in addition to value,ok.
//
// This method is about 20-30% slower than Lookup and should
// only be used if you either explicitly have a prefix as an input parameter
// or the prefix of the matching lpm entry is also required for other reasons.
//
// If Lookup2 is to be used for IP addresses,
// they must be converted to /32 or /128 prefixes.
func (t *Table[V]) Lookup2(pfx netip.Prefix) (lpm netip.Prefix, val V, ok bool) {
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
		if val, ok = n.prefixes.getValByPrefix(0, 0); !ok {
			// default route not set in table
			return
		}
		return pfx, val, ok
	}

	// stack of the traversed nodes for fast backtracking, if needed
	pathStack := [maxTreeDepth]*node[V]{}

	// keep the lpm alloc free, don't use ip.AsSlice here
	a16 := ip.As16()
	bs := a16[:]
	if is4 {
		bs = bs[12:]
	}

	depth := 0
	addr := uint(bs[depth]) // bytewise, stride = 8
	// find leaf tree
	for {

		// push current node on stack for fast backtracking
		pathStack[depth] = n

		// already at leaf node?
		if bits <= stride {
			break
		}

		// go down in tight loop to leaf node
		if child := n.children.get(addr); child != nil {
			depth++
			bits -= stride
			addr = uint(bs[depth])
			n = child
			continue
		}

		if bits > stride {
			bits = stride
		}
		break
	}

	// start backtracking at leaf node in tight loop
	for {

		// lookup only in nodes with prefixes, skip over intermediate nodes
		if len(n.prefixes.values) != 0 {
			// longest prefix match?
			if baseIdx, val, ok := n.prefixes.lpmByPrefix(addr, bits); ok {
				// calculate the mask from baseIdx and depth
				mask := depth*stride + baseIndexToPrefixLen(baseIdx)

				// calculate the lpm from ip and mask
				lpm = netip.PrefixFrom(ip, mask).Masked()

				// match
				return lpm, val, true
			}
		}

		// bits must be full stride for all upper levels
		if bits < stride {
			bits = stride
		}

		// end condition, stack is exhausted
		if depth == 0 {
			return
		}

		// go up, backtracking
		depth--
		addr = uint(bs[depth])
		n = pathStack[depth]
	}
}

// Subnets return all prefixes covered by pfx in natural CIDR sort order.
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

		// return all routes for this IP version.
		_ = n.walkRec(nil, is4, func(pfx netip.Prefix, _ V) error {
			result = append(result, pfx)
			return nil
		})

		slices.SortFunc(result, sortByPrefix)
		return result
	}

	bs := ip.AsSlice()
	depth := 0

	for {
		addr := uint(bs[depth])

		// already at leaf node?
		if bits <= stride {
			result = n.subnets(bs[:depth], prefixToBaseIndex(addr, bits), is4)

			slices.SortFunc(result, sortByPrefix)
			return result
		}

		// descend down to next child level
		child := n.children.get(addr)

		// stop condition, no more childs
		if child == nil {
			return nil
		}

		// next round
		depth++
		n = child
		bits -= stride
	}
}

// Supernets return all matching routes for pfx,
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
		if _, ok := n.prefixes.getValByPrefix(0, 0); ok {
			result = append(result, pfx)
		}
		return result
	}

	bs := ip.AsSlice()
	depth := 0

	for {
		addr := uint(bs[depth])

		pfxLen := 8
		// already at leaf node?
		if bits <= stride {
			pfxLen = bits
		}

		// make an all-prefix-match at this level
		superStrides := n.prefixes.apmByPrefix(addr, pfxLen)

		// get back the matching prefix from baseIdx
		for _, baseIdx := range superStrides {
			// calc supernet mask
			mask := depth*stride + baseIndexToPrefixLen(baseIdx)

			// lookup ip, masked with supernet mask
			matchPfx := netip.PrefixFrom(ip, mask).Masked()

			result = append(result, matchPfx)
		}

		// stop condition
		if bits <= stride {
			break
		}

		// descend down to next child level
		child := n.children.get(addr)

		// stop condition, no more childs
		if child == nil {
			break
		}

		// next round
		depth++
		n = child
		bits -= stride
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

	// keep the overlaps alloc free, don't use ip.AsSlice here
	a16 := ip.As16()
	bs := a16[:]
	if is4 {
		bs = bs[12:]
	}

	// depth index for the child trie
	depth := 0
	addr := uint(bs[depth])

	for {

		// last prefix chunk reached
		if bits <= stride {
			return n.overlapsPrefix(addr, bits)
		}

		// still in the middle of prefix chunks
		// test if any route overlaps prefix´ addr chunk so far

		// but skip intermediate nodes, no routes to test?
		if len(n.prefixes.values) != 0 {
			if _, _, ok := n.prefixes.lpmByAddr(addr); ok {
				return true
			}
		}

		// no overlap so far, go down to next child
		child := n.children.get(addr)

		// no more children to explore, there can't be an overlap
		if child == nil {
			return false
		}

		// next round
		depth++
		addr = uint(bs[depth])
		bits -= stride
		n = child
	}
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
// The sort order is not specified and is not part of the
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
