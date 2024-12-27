// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
)

// Supernets returns an iterator over all CIDRs covering pfx.
// The iteration is in reverse CIDR sort order, from longest-prefix-match to shortest-prefix-match.
// TODO, path compression not yet implemented for this method.
func (t *Table[V]) Supernets(pfx netip.Prefix) func(yield func(netip.Prefix, V) bool) {
	if t.pathCompressed {
		panic("TODO, path compression not yet implemented for this method")
	}

	return func(yield func(netip.Prefix, V) bool) {
		if !pfx.IsValid() {
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

		// needed as argument below
		copy(path[:], octets)

		// see comment in Insert()
		lastOctetIdx := (bits - 1) / strideLen
		lastOctet := octets[lastOctetIdx]
		lastOctetBits := bits - (lastOctetIdx * strideLen)

		// mask the prefix
		lastOctet &= netMask(lastOctetBits)
		octets[lastOctetIdx] = lastOctet

		// stack of the traversed nodes for reverse ordering of supernets
		stack := [maxTreeDepth]*node[V]{}

		// run variable, used after for loop
		var i int
		var ok bool
		var octet byte

		// find last node
		for i, octet = range octets[:lastOctetIdx+1] {
			// push current node on stack
			stack[i] = n

			// go down in tight loop
			if n, ok = n.children.Get(uint(octet)); !ok {
				break
			}
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
}

// Subnets returns an iterator over all CIDRs covered by pfx.
// The iteration is in natural CIDR sort order.
// TODO, path compression not yet implemented for this method.
func (t *Table[V]) Subnets(pfx netip.Prefix) func(yield func(netip.Prefix, V) bool) {
	if t.pathCompressed {
		panic("TODO, path compression not yet implemented for this method")
	}

	return func(yield func(netip.Prefix, V) bool) {
		if !pfx.IsValid() {
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
		lastOctet &= netMask(lastOctetBits)
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
}

// All returns an iterator over key-value pairs from Table. The iteration order
// is not specified and is not guaranteed to be the same from one call to the
// next.
func (t *Table[V]) All() func(yield func(pfx netip.Prefix, val V) bool) {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root4.allRec(zeroPath, 0, true, yield) && t.root6.allRec(zeroPath, 0, false, yield)
	}
}

// All4, like [Table.All] but only for the v4 routing table.
func (t *Table[V]) All4() func(yield func(pfx netip.Prefix, val V) bool) {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root4.allRec(zeroPath, 0, true, yield)
	}
}

// All6, like [Table.All] but only for the v6 routing table.
func (t *Table[V]) All6() func(yield func(pfx netip.Prefix, val V) bool) {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root6.allRec(zeroPath, 0, false, yield)
	}
}

// AllSorted returns an iterator over key-value pairs from Table in natural CIDR sort order.
func (t *Table[V]) AllSorted() func(yield func(pfx netip.Prefix, val V) bool) {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root4.allRecSorted(zeroPath, 0, true, yield) &&
			t.root6.allRecSorted(zeroPath, 0, false, yield)
	}
}

// AllSorted4, like [Table.AllSorted] but only for the v4 routing table.
func (t *Table[V]) AllSorted4() func(yield func(pfx netip.Prefix, val V) bool) {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root4.allRecSorted(zeroPath, 0, true, yield)
	}
}

// AllSorted6, like [Table.AllSorted] but only for the v6 routing table.
func (t *Table[V]) AllSorted6() func(yield func(pfx netip.Prefix, val V) bool) {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root6.allRecSorted(zeroPath, 0, false, yield)
	}
}
