// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package golden

import (
	"fmt"
	"math/rand/v2"
	"net/netip"
)

//nolint:gochecknoglobals
var mpp = func(s string) netip.Prefix {
	pfx := netip.MustParsePrefix(s)
	if pfx == pfx.Masked() {
		return pfx
	}
	panic(fmt.Sprintf("%s is not canonicalized as %s", s, pfx.Masked()))
}

// RandomPrefix returns a randomly generated prefix
func RandomPrefix(prng *rand.Rand) netip.Prefix {
	if prng.IntN(2) == 1 {
		return RandomPrefix4(prng)
	}
	return RandomPrefix6(prng)
}

func RandomPrefix4(prng *rand.Rand) netip.Prefix {
	bits := prng.IntN(33)
	return netip.PrefixFrom(RandomIP4(prng), bits).Masked()
}

func RandomPrefix6(prng *rand.Rand) netip.Prefix {
	bits := prng.IntN(129)
	return netip.PrefixFrom(RandomIP6(prng), bits).Masked()
}

func RandomIP4(prng *rand.Rand) netip.Addr {
	var b [4]byte
	for i := range b {
		b[i] = byte(prng.UintN(256))
	}
	return netip.AddrFrom4(b)
}

func RandomIP6(prng *rand.Rand) netip.Addr {
	var b [16]byte
	for i := range b {
		b[i] = byte(prng.UintN(256))
	}
	return netip.AddrFrom16(b)
}

func RandomAddr(prng *rand.Rand) netip.Addr {
	if prng.IntN(2) == 1 {
		return RandomIP4(prng)
	}
	return RandomIP6(prng)
}

func RandomRealWorldPrefixes4(prng *rand.Rand, n int) []netip.Prefix {
	set := make(map[netip.Prefix]struct{})
	pfxs := make([]netip.Prefix, 0, n)

	for len(set) < n {
		pfx := RandomPrefix4(prng)

		// skip too small or too big masks
		if pfx.Bits() < 8 || pfx.Bits() > 28 {
			continue
		}

		// skip reserved/experimental ranges (e.g., 240.0.0.0/8)
		if pfx.Overlaps(mpp("240.0.0.0/8")) {
			continue
		}

		if _, ok := set[pfx]; !ok {
			set[pfx] = struct{}{}
			pfxs = append(pfxs, pfx)
		}
	}
	return pfxs
}

func RandomRealWorldPrefixes6(prng *rand.Rand, n int) []netip.Prefix {
	set := make(map[netip.Prefix]struct{})
	pfxs := make([]netip.Prefix, 0, n)

	for len(set) < n {
		pfx := RandomPrefix6(prng)

		// skip too small or too big masks
		if pfx.Bits() < 16 || pfx.Bits() > 56 {
			continue
		}

		// skip non global routes seen in the real world
		if !pfx.Overlaps(mpp("2000::/3")) {
			continue
		}
		if pfx.Addr().Compare(mpp("2c0f::/16").Addr()) == 1 {
			continue
		}

		if _, ok := set[pfx]; !ok {
			set[pfx] = struct{}{}
			pfxs = append(pfxs, pfx)
		}
	}
	return pfxs
}

func RandomRealWorldPrefixes(prng *rand.Rand, n int) []netip.Prefix {
	pfxs := make([]netip.Prefix, 0, n)
	pfxs = append(pfxs, RandomRealWorldPrefixes4(prng, n/2)...)
	pfxs = append(pfxs, RandomRealWorldPrefixes6(prng, n-len(pfxs))...)

	prng.Shuffle(len(pfxs), func(i, j int) {
		pfxs[i], pfxs[j] = pfxs[j], pfxs[i]
	})

	return pfxs
}
