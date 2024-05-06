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

	// some values derived from pfx
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

	// find the proper trie node to insert prefix
	for i, octet := range octets {
		if i == lastOctetIdx {
			// insert prefix into node
			n.insertPrefix(lastOctet, lastOctetBits, val)
			return
		}

		// descend down to next trie level
		c := n.getChild(octet)

		// create and insert missing intermediate child
		if c == nil {
			c = newNode[V]()
			n.insertChild(octet, c)
		}

		// proceed with next level
		n = c
	}
}

// Delete removes pfx from the tree, pfx does not have to be present.
func (t *Table[V]) Delete(pfx netip.Prefix) {
	// some values derived from pfx
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

	// record path to deleted node
	// purge dangling nodes after deletion
	stack := [maxTreeDepth]*node[V]{}

	// run variables, needed after for loop
	var i int
	var octet byte

	// find the trie node
	for i, octet = range octets {
		// push current node on stack for path recording
		stack[i] = n

		if i == lastOctetIdx {
			if !n.deletePrefix(lastOctet, lastOctetBits) {
				// prefix not in tree, nothing deleted
				return
			}

			// escape, but purge dangling path if needed, see below
			break
		}

		// descend down to next level
		if c := n.getChild(octet); c != nil {
			n = c
			continue
		}

		// no child found, nothing to delete
		return
	}

	// purge dangling paths
	for i > 0 {
		if n.isEmpty() {
			// purge empty node from parents children
			parent := stack[i-1]
			parent.deleteChild(octets[i-1])
		}

		// unwind the stack
		i--
		n = stack[i]
	}
}

// Update or set the value at pfx with a callback function.
// The callback function is called with (value, ok) and returns a new value..
//
// If the pfx does not already exist, it is set with the new value.
func (t *Table[V]) Update(pfx netip.Prefix, cb func(val V, ok bool) V) V {
	t.init()

	// some values derived from pfx
	_, ip, bits, is4 := pfxToValues(pfx)

	n := t.rootNodeByVersion(is4)

	// do not allocate
	octets := make([]byte, 0, 16)
	octets = ipToOctets(octets, ip, is4)

	// see comment in Insert()
	lastOctetIdx := (bits - 1) / strideLen
	lastOctet := octets[lastOctetIdx]
	lastOctetBits := bits - (lastOctetIdx * strideLen)

	// find the proper trie node to update prefix
	for i, octet := range octets {
		if i == lastOctetIdx {
			// update/insert prefix into node
			return n.updatePrefix(lastOctet, lastOctetBits, cb)
		}

		// descend down to next trie level
		c := n.getChild(octet)

		// create and insert missing intermediate child
		if c == nil {
			c = newNode[V]()
			n.insertChild(octet, c)
		}

		// proceed with next level
		n = c
	}

	panic("unreachable")
}

// Get returns the associated payload for prefix and true, or false if
// prefix is not set in the routing table.
func (t *Table[V]) Get(pfx netip.Prefix) (val V, ok bool) {
	// some values derived from pfx
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

	// find the proper trie node to update prefix
	for i, octet := range octets {
		if i == lastOctetIdx {
			return n.getValByPrefix(lastOctet, lastOctetBits)
		}

		// descend down to next level
		if c := n.getChild(octet); c != nil {
			n = c
			continue
		}

		return
	}

	panic("unreachable")
}

// Lookup does a route lookup (longest prefix match) for IP and
// returns the associated value and true, or false if no route matched.
func (t *Table[V]) Lookup(ip netip.Addr) (val V, ok bool) {
	is4 := ip.Is4()

	n := t.rootNodeByVersion(is4)
	if n == nil {
		return
	}

	// do not allocate
	octets := make([]byte, 0, 16)
	octets = ipToOctets(octets, ip, is4)

	lastOctetIdx := len(octets) - 1

	// stack of the traversed nodes for fast backtracking, if needed
	stack := [maxTreeDepth]*node[V]{}

	// run variables, used after for loop
	var i int
	var octet byte

	// find leaf node
	for i, octet = range octets {
		// push current node on stack for fast backtracking
		stack[i] = n

		if i == lastOctetIdx {
			break
		}

		// go down in tight loop to leaf node
		if c := n.getChild(octet); c != nil {
			n = c
			continue
		}

		break
	}

	// start backtracking at leaf node in tight loop
	for {
		// longest prefix match?
		if _, val, ok := n.lpmByOctet(octet); ok {
			return val, true
		}

		// if stack is exhausted?
		if i == 0 {
			return
		}

		// unwind the stack
		i--
		octet = octets[i]
		n = stack[i]
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

	if ok {
		// calculate the mask from baseIdx and depth
		mask := baseIndexToPrefixMask(baseIdx, depth)

		// calculate the lpm from ip and mask
		lpm, _ = pfx.Addr().Prefix(mask)
	}

	return lpm, val, ok
}

func (t *Table[V]) lpmByPrefix(pfx netip.Prefix) (depth int, baseIdx uint, val V, ok bool) {
	// some values derived from pfx
	_, ip, bits, is4 := pfxToValues(pfx)

	n := t.rootNodeByVersion(is4)
	if n == nil {
		return
	}

	// record path to leaf node
	stack := [maxTreeDepth]*node[V]{}

	// do not allocate
	octets := make([]byte, 0, 16)
	octets = ipToOctets(octets, ip, is4)

	// see comment in Insert()
	lastOctetIdx := (bits - 1) / strideLen
	lastOctetBits := bits - (lastOctetIdx * strideLen)

	// default, only the lastOctet has a different prefix len
	pfxLen := strideLen

	var octet byte

	// find the trie node
	for depth, octet = range octets {
		// push current node on stack
		stack[depth] = n

		// last significant octet reached
		if depth == lastOctetIdx {
			// only the lastOctet has a different prefix len (prefix route)
			pfxLen = lastOctetBits
			break
		}

		// go down in tight loop to leaf node
		if c := n.getChild(octet); c != nil {
			n = c
			continue
		}

		break
	}

	// start backtracking with last node and octet
	for {
		if baseIdx, val, ok := n.lpmByPrefix(octet, pfxLen); ok {
			return depth, baseIdx, val, true
		}

		// if stack is exhausted?
		if depth == 0 {
			return
		}

		// unwind the stack
		depth--
		pfxLen = strideLen
		octet = octets[depth]
		n = stack[depth]
	}
}

// Subnets, return all prefixes covered by pfx in natural CIDR sort order.
func (t *Table[V]) Subnets(pfx netip.Prefix) []netip.Prefix {
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

	// find the trie node
	for i, octet := range octets {
		if i == lastOctetIdx {
			result := n.subnets(octets[:i], prefixToBaseIndex(lastOctet, lastOctetBits), is4)

			slices.SortFunc(result, cmpPrefix)
			return result
		}

		// descend down to next level
		if c := n.getChild(octet); c != nil {
			n = c
			continue
		}

		return nil
	}

	panic("unreachable")
}

// Supernets, return all matching routes for pfx,
// in natural CIDR sort order.
func (t *Table[V]) Supernets(pfx netip.Prefix) []netip.Prefix {
	var result []netip.Prefix

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

	for i, octet := range octets {
		if i == lastOctetIdx {
			// make an all-prefix-match at last level
			return append(result, n.apmByPrefix(lastOctet, lastOctetBits, i, ip)...)
		}

		// make an all-prefix-match at intermediate level for octet
		result = append(result, n.apmByOctet(octet, i, ip)...)

		// descend down to next trie level
		if c := n.getChild(octet); c != nil {
			n = c
			continue
		}

		return result
	}

	panic("unreachable")
}

// OverlapsPrefix reports whether any IP in pfx matches a route in the table.
func (t *Table[V]) OverlapsPrefix(pfx netip.Prefix) bool {
	// some needed values, see below
	_, ip, bits, is4 := pfxToValues(pfx)

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

	for i, octet := range octets {
		if i == lastOctetIdx {
			return n.overlapsPrefix(lastOctet, lastOctetBits)
		}

		// still in the middle of prefix chunks
		// test if any route overlaps prefix´
		if _, _, ok := n.lpmByOctet(octet); ok {
			return true
		}

		// no overlap so far, go down to next c
		if c := n.getChild(octet); c != nil {
			n = c
			continue
		}

		return false
	}

	panic("unreachable")
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

// All may be used in a for/range loop to iterate
// through all the prefixes.
//
// Prefixes must not be inserted or deleted during iteration, otherwise
// the behavior is undefined. However, value updates are permitted.
//
// The iteration order is not specified and is not part of the
// public interface, you must not rely on it.
func (t *Table[V]) All(yield func(pfx netip.Prefix, val V) bool) {
	t.init()
	// respect premature end of allRec()
	_ = t.rootV4.allRec(nil, true, yield) && t.rootV6.allRec(nil, false, yield)
}

// All4, like [Table.All] but only for the v4 routing table.
func (t *Table[V]) All4(yield func(pfx netip.Prefix, val V) bool) {
	t.init()
	t.rootV4.allRec(nil, true, yield)
}

// All6, like [Table.All] but only for the v6 routing table.
func (t *Table[V]) All6(yield func(pfx netip.Prefix, val V) bool) {
	t.init()
	t.rootV6.allRec(nil, false, yield)
}

// ipToOctets, be careful, do not allocate!
//
// intended use: SA4009: argument octets is overwritten before first use
//
//nolint:staticcheck
func ipToOctets(octets []byte, ip netip.Addr, is4 bool) []byte {
	a16 := ip.As16()
	octets = a16[:]
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
