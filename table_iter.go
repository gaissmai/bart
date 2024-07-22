// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
)

// Supernets returns an iterator over all CIDRs covering pfx.
// The iteration is in reverse CIDR sort order, from longest-prefix-match to shortest-prefix-match.
func (t *Table[V]) Supernets(pfx netip.Prefix) func(yield func(netip.Prefix, V) bool) {
	return func(yield func(netip.Prefix, V) bool) {

		// iterator setup
		if !pfx.IsValid() {
			return
		}

		// values derived from pfx
		ip := pfx.Addr()
		is4 := ip.Is4()
		bits := pfx.Bits()

		n := t.rootNodeByVersion(is4)
		if n == nil {
			return
		}

		// do not allocate
		path := ip.As16()
		octets := path[:]
		if is4 {
			octets = octets[12:]
		}
		copy(path[:], octets[:])

		// see comment in Insert()
		lastOctetIdx := (bits - 1) / strideLen
		lastOctet := octets[lastOctetIdx]
		lastOctetBits := bits - (lastOctetIdx * strideLen)

		// mask the prefix
		lastOctet = lastOctet & netMask(lastOctetBits)
		octets[lastOctetIdx] = lastOctet

		// stack of the traversed nodes for reverse ordering of supernets
		stack := [maxTreeDepth]*node[V]{}

		// run variable, used after for loop
		var i int
		var octet byte

		// find last node
		for i, octet = range octets[:lastOctetIdx+1] {
			// push current node on stack
			stack[i] = n

			// go down in tight loop
			c := n.getChild(octet)
			if c == nil {
				break
			}
			n = c
		}

		// start backtracking, unwind the stack
		for depth := i; depth >= 0; depth-- {
			n = stack[depth]

			// microbenchmarking
			if len(n.prefixes) == 0 {
				continue
			}

			// only the lastOctet may have a different prefix len
			if depth == lastOctetIdx {
				if !n.eachLookupPrefix(path, depth, is4, lastOctet, lastOctetBits, yield) {
					// early exit
					return
				}
				continue
			}

			// all others are just host routes
			if !n.eachLookupPrefix(path, depth, is4, octets[depth], strideLen, yield) {
				// early exit
				return
			}
		}
	}

}

// Subnets returns an iterator over all CIDRs covered by pfx.
// The iteration is in natural CIDR sort order.
func (t *Table[V]) Subnets(pfx netip.Prefix) func(yield func(netip.Prefix, V) bool) {
	return func(yield func(netip.Prefix, V) bool) {

		// iterator setup
		if !pfx.IsValid() {
			return
		}

		// values derived from pfx
		ip := pfx.Addr()
		is4 := ip.Is4()
		bits := pfx.Bits()

		n := t.rootNodeByVersion(is4)
		if n == nil {
			return
		}

		// do not allocate
		path := ip.As16()
		octets := path[:]
		if is4 {
			octets = octets[12:]
		}
		copy(path[:], octets[:])

		// see comment in Insert()
		lastOctetIdx := (bits - 1) / strideLen
		lastOctet := octets[lastOctetIdx]
		lastOctetBits := bits - (lastOctetIdx * strideLen)

		// mask the prefix
		lastOctet = lastOctet & netMask(lastOctetBits)
		octets[lastOctetIdx] = lastOctet

		// find the trie node
		for i, octet := range octets {
			if i == lastOctetIdx {
				_ = n.eachSubnet(path, i, is4, lastOctet, lastOctetBits, yield)
				return
			}

			c := n.getChild(octet)
			if c == nil {
				break
			}

			n = c
		}
	}
}

// All may be used in a for/range loop to iterate
// through all the prefixes.
// The sort order is undefined and you must not rely on it!
//
// Prefixes must not be inserted or deleted during iteration, otherwise
// the behavior is undefined. However, value updates are permitted.
//
// If the yield function returns false, the iteration ends prematurely.
func (t *Table[V]) All(yield func(pfx netip.Prefix, val V) bool) {
	t.init()
	// respect early exit
	_ = t.rootV4.allRec(zeroPath, 0, true, yield) &&
		t.rootV6.allRec(zeroPath, 0, false, yield)
}

// All4, like [Table.All] but only for the v4 routing table.
func (t *Table[V]) All4(yield func(pfx netip.Prefix, val V) bool) {
	t.init()
	t.rootV4.allRec(zeroPath, 0, true, yield)
}

// All6, like [Table.All] but only for the v6 routing table.
func (t *Table[V]) All6(yield func(pfx netip.Prefix, val V) bool) {
	t.init()
	t.rootV6.allRec(zeroPath, 0, false, yield)
}

// AllSorted may be used in a for/range loop to iterate
// through all the prefixes in natural CIDR sort order.
//
// Prefixes must not be inserted or deleted during iteration, otherwise
// the behavior is undefined. However, value updates are permitted.
//
// If the yield function returns false, the iteration ends prematurely.
func (t *Table[V]) AllSorted(yield func(pfx netip.Prefix, val V) bool) {
	t.init()
	// respect early exit
	_ = t.rootV4.allRecSorted(zeroPath, 0, true, yield) &&
		t.rootV6.allRecSorted(zeroPath, 0, false, yield)
}

// All4Sorted, like [Table.AllSorted] but only for the v4 routing table.
func (t *Table[V]) All4Sorted(yield func(pfx netip.Prefix, val V) bool) {
	t.init()
	t.rootV4.allRecSorted(zeroPath, 0, true, yield)
}

// All6Sorted, like [Table.AllSorted] but only for the v6 routing table.
func (t *Table[V]) All6Sorted(yield func(pfx netip.Prefix, val V) bool) {
	t.init()
	t.rootV6.allRecSorted(zeroPath, 0, false, yield)
}
