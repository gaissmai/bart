// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
)

/*
// Supernets returns an iterator over all CIDRs covering pfx.
// The iteration is in reverse CIDR sort order, from longest-prefix-match to shortest-prefix-match.
func (t *Table2[V]) Supernets(pfx netip.Prefix) func(yield func(netip.Prefix, V) bool) {
	return func(yield func(netip.Prefix, V) bool) {
		if !pfx.IsValid() {
			return
		}

		// canonicalize the prefix
		pfx = pfx.Masked()

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
		octets[lastOctetIdx] = lastOctet

		// stack of the traversed nodes for reverse ordering of supernets
		stack := [maxTreeDepth]*node[V]{}

		// run variable, used after for loop
		var i int
		var ok bool
		var octet byte
		var addr uint

		// find last node along this octet path
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
			octet = octets[depth]
			addr = uint(octet)

			// first check path compressed prefix at this level
			if t.pathCompressed && n.pathcomp.Test(addr) {
				pc := n.pathcomp.MustGet(addr)
				if pc.prefix.Overlaps(pfx) && pc.prefix.Bits() <= pfx.Bits() {
					if !yield(pc.prefix, pc.value) {
						// early exit
						return
					}
				}
			}

			// microbenchmarking
			if n.prefixes.Len() == 0 {
				continue
			}

			// only the lastOctet may have a different prefix len
			// all others are just host routes
			pfxLen := strideLen
			if depth == lastOctetIdx {
				pfxLen = lastOctetBits
			}
			if !n.eachLookupPrefix(path, depth, is4, octet, pfxLen, yield) {
				// early exit
				return
			}
		}
	}
}

// Subnets returns an iterator over all CIDRs covered by pfx.
// The iteration is in natural CIDR sort order.
func (t *Table2[V]) Subnets(pfx netip.Prefix) func(yield func(netip.Prefix, V) bool) {
	return func(yield func(netip.Prefix, V) bool) {
		if !pfx.IsValid() {
			return
		}

		// canonicalize the prefix
		pfx = pfx.Masked()

		// values derived from pfx
		ip := pfx.Addr()
		is4 := ip.Is4()
		bits := pfx.Bits()

		n := t.rootNodeByVersion(is4)

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
		octets[lastOctetIdx] = lastOctet

		var addr uint
		// find the trie node
		for i, octet := range octets {
			addr = uint(octet)

			// already at last octet?
			if i == lastOctetIdx {
				_ = n.eachSubnet(pfx, path, i, is4, lastOctet, lastOctetBits, yield)
				return
			}

			// check path compressed prefix at this slot
			if n.pathcomp.Test(addr) {
				pc := n.pathcomp.MustGet(addr)
				if pfx.Overlaps(pc.prefix) && pfx.Bits() <= pc.prefix.Bits() {
					if !yield(pc.prefix, pc.value) {
						return
					}
				}
			}

			// descend down to next level in tight loop
			if n.children.Test(addr) {
				n = n.children.MustGet(addr)
				continue
			}

			// end of trie along this octets path
			break
		}
	}
}
*/

// All returns an iterator over key-value pairs from Table. The iteration order
// is not specified and is not guaranteed to be the same from one call to the
// next.
func (t *Table2[V]) All() func(yield func(pfx netip.Prefix, val V) bool) {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root4.allRec(zeroPath, 0, true, yield) && t.root6.allRec(zeroPath, 0, false, yield)
	}
}

// All4, like [Table2.All] but only for the v4 routing table.
func (t *Table2[V]) All4() func(yield func(pfx netip.Prefix, val V) bool) {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root4.allRec(zeroPath, 0, true, yield)
	}
}

// All6, like [Table2.All] but only for the v6 routing table.
func (t *Table2[V]) All6() func(yield func(pfx netip.Prefix, val V) bool) {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root6.allRec(zeroPath, 0, false, yield)
	}
}

// AllSorted returns an iterator over key-value pairs from Table2 in natural CIDR sort order.
func (t *Table2[V]) AllSorted() func(yield func(pfx netip.Prefix, val V) bool) {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root4.allRecSorted(zeroPath, 0, true, yield) &&
			t.root6.allRecSorted(zeroPath, 0, false, yield)
	}
}

// AllSorted4, like [Table2.AllSorted] but only for the v4 routing table.
func (t *Table2[V]) AllSorted4() func(yield func(pfx netip.Prefix, val V) bool) {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root4.allRecSorted(zeroPath, 0, true, yield)
	}
}

// AllSorted6, like [Table2.AllSorted] but only for the v6 routing table.
func (t *Table2[V]) AllSorted6() func(yield func(pfx netip.Prefix, val V) bool) {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root6.allRecSorted(zeroPath, 0, false, yield)
	}
}