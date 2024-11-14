// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
)

// EachLookupPrefix calls yield() for each CIDR covering pfx
// in reverse CIDR sort order, from longest-prefix-match to
// shortest-prefix-match.
//
// If the yield function returns false, the iteration ends prematurely.
//
// Deprecated: EachLookupPrefix is deprecated. Use [Table.Supernets] instead.
func (t *Table[V]) EachLookupPrefix(pfx netip.Prefix, yield func(pfx netip.Prefix, val V) bool) {
	if !pfx.IsValid() || !t.isInit() {
		return
	}
	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()

	n := t.rootNodeByVersion(is4)

	// do not allocate
	path := ip.As16()

	octets := path[:]
	if is4 {
		octets = octets[12:]
	}

	copy(path[:], octets)

	// see comment in Insert()
	lastOctetIdx := (bits - 1) / strideLen
	lastOctet := octets[lastOctetIdx]
	lastOctetBits := bits - (lastOctetIdx * strideLen)

	// mask the prefix
	lastOctet &= netMask[lastOctetBits]
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
		c, ok := n.children.Get(uint(octet))
		if !ok {
			break
		}

		n = c
	}

	// start backtracking, unwind the stack
	for depth := i; depth >= 0; depth-- {
		n = stack[depth]

		// microbenchmarking
		if n.prefixes.Len() == 0 {
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

// EachSubnet iterates over all CIDRs covered by pfx in natural CIDR sort order.
//
// If the yield function returns false, the iteration ends prematurely.
//
// Deprecated: EachSubnet is deprecated. Use [Table.Subnets] instead.
func (t *Table[V]) EachSubnet(pfx netip.Prefix, yield func(pfx netip.Prefix, val V) bool) {
	if !pfx.IsValid() || !t.isInit() {
		return
	}

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()

	n := t.rootNodeByVersion(is4)

	// do not allocate
	path := ip.As16()

	octets := path[:]
	if is4 {
		octets = octets[12:]
	}

	copy(path[:], octets)

	// see comment in Insert()
	lastOctetIdx := (bits - 1) / strideLen
	lastOctet := octets[lastOctetIdx]
	lastOctetBits := bits - (lastOctetIdx * strideLen)

	// mask the prefix
	lastOctet &= netMask[lastOctetBits]
	octets[lastOctetIdx] = lastOctet

	// find the trie node
	for i, octet := range octets {
		if i == lastOctetIdx {
			_ = n.eachSubnet(path, i, is4, lastOctet, lastOctetBits, yield)

			return
		}

		c, ok := n.children.Get(uint(octet))
		if !ok {
			break
		}

		n = c
	}
}
