// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package random

import (
	"math/rand/v2"
	"net/netip"
)

// NestedPrefixes returns up to n distinct random canonical prefixes of one
// address family, the bytes of the addresses are drawn from pool.
//
// In contrast to [Prefix] or [RealWorldPrefixes] the prefixes are made to
// collide: a small pool of byte values lets them overlap and nest, and the
// prefix lengths favor the borders of the 8-bit strides of the trie, where the
// path compression (leaf, fringe) and the splitting and merging of nodes
// happen. That makes the generator suited to test the structure of a routing
// table, it is not suited to benchmark it.
//
// With nest, for about a quarter of the prefixes both more specific halves are
// added, so the result contains up to 3*n prefixes. The first prefixes of the
// result are the same with and without nest.
//
// The result is deterministic for a given state of prng.
func NestedPrefixes(prng *rand.Rand, is4 bool, pool []byte, n int, nest bool) []netip.Prefix {
	size, maxBits := 16, 128
	if is4 {
		size, maxBits = 4, 32
	}

	set := make(map[netip.Prefix]struct{}, n)
	pfxs := make([]netip.Prefix, 0, n)

	add := func(pfx netip.Prefix) {
		if _, ok := set[pfx]; !ok {
			set[pfx] = struct{}{}
			pfxs = append(pfxs, pfx)
		}
	}

	for range n {
		var bits int

		switch prng.IntN(3) {
		case 0: // any length
			bits = prng.IntN(maxBits + 1)
		case 1: // on a stride border
			bits = prng.IntN(size+1) * 8
		default: // near a stride border
			bits = prng.IntN(size+1)*8 + prng.IntN(5) - 2
		}

		bits = max(0, min(maxBits, bits))

		raw := make([]byte, size)
		for i := range raw {
			raw[i] = pool[prng.IntN(len(pool))]
		}

		var addr netip.Addr
		if is4 {
			addr = netip.AddrFrom4([4]byte(raw))
		} else {
			addr = netip.AddrFrom16([16]byte(raw))
		}

		add(netip.PrefixFrom(addr, bits).Masked())
	}

	if nest {
		// the halves are added behind the generated prefixes and
		// are not nested again
		generated := len(pfxs)

		for i := range generated {
			pfx := pfxs[i]

			if prng.IntN(4) != 0 || pfx.Bits() == maxBits {
				continue
			}

			lo, hi := halves(pfx)
			add(lo)
			add(hi)
		}
	}

	return pfxs
}

// halves returns the two prefixes that are one bit more specific than pfx.
func halves(pfx netip.Prefix) (lo, hi netip.Prefix) {
	raw := pfx.Addr().AsSlice()
	raw[pfx.Bits()/8] |= 0x80 >> (pfx.Bits() % 8)

	hiAddr, _ := netip.AddrFromSlice(raw)

	return netip.PrefixFrom(pfx.Addr(), pfx.Bits()+1), netip.PrefixFrom(hiAddr, pfx.Bits()+1)
}
