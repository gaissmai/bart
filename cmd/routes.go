package main

import (
	"bufio"
	"compress/gzip"
	"log"
	"math/rand/v2"
	"net/netip"
	"os"
	"strings"
)

// full internet prefix list, gzipped
const prefixFile = "../testdata/prefixes.txt.gz"

var mpp = netip.MustParsePrefix

func tier1Pfxs() (pfxs []netip.Prefix) {
	file, err := os.Open(prefixFile)
	if err != nil {
		log.Fatal(err)
	}

	rgz, err := gzip.NewReader(file)
	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(rgz)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		cidr := netip.MustParsePrefix(line)
		cidr = cidr.Masked()

		pfxs = append(pfxs, cidr)
	}

	if err = scanner.Err(); err != nil {
		log.Printf("reading from %v, %v", rgz, err)
	}
	return
}

//nolint:unused
func randomTier1RoutesN(prng *rand.Rand, pfxs []netip.Prefix, n int) []netip.Prefix {
	prng.Shuffle(len(pfxs), func(i, j int) {
		pfxs[i], pfxs[j] = pfxs[j], pfxs[i]
	})

	return pfxs[:n]
}

// randomPrefix returns a randomly generated prefix
//
//nolint:unused
func randomPrefix(prng *rand.Rand) netip.Prefix {
	if prng.IntN(2) == 1 {
		return randomPrefix4(prng)
	}
	return randomPrefix6(prng)
}

func randomPrefix4(prng *rand.Rand) netip.Prefix {
	bits := prng.IntN(33)
	pfx, err := randomIP4(prng).Prefix(bits)
	if err != nil {
		panic(err)
	}
	return pfx
}

func randomPrefix6(prng *rand.Rand) netip.Prefix {
	bits := prng.IntN(129)
	pfx, err := randomIP6(prng).Prefix(bits)
	if err != nil {
		panic(err)
	}
	return pfx
}

func randomIP4(prng *rand.Rand) netip.Addr {
	var b [4]byte
	for i := range b {
		b[i] = byte(prng.UintN(256))
	}
	return netip.AddrFrom4(b)
}

func randomIP6(prng *rand.Rand) netip.Addr {
	var b [16]byte
	for i := range b {
		b[i] = byte(prng.UintN(256))
	}
	return netip.AddrFrom16(b)
}

//nolint:unused
func randomAddr(prng *rand.Rand) netip.Addr {
	if prng.IntN(2) == 1 {
		return randomIP4(prng)
	}
	return randomIP6(prng)
}

func randomRealWorldPrefixes4(prng *rand.Rand, n int) []netip.Prefix {
	set := map[netip.Prefix]netip.Prefix{}
	pfxs := make([]netip.Prefix, 0, n)

	for {
		pfx := randomPrefix4(prng)

		// skip too small or too big masks
		if pfx.Bits() < 8 || pfx.Bits() > 28 {
			continue
		}

		// skip multicast ...
		if pfx.Overlaps(mpp("240.0.0.0/8")) {
			continue
		}

		if _, ok := set[pfx]; !ok {
			set[pfx] = pfx
			pfxs = append(pfxs, pfx)
		}

		if len(set) >= n {
			break
		}
	}
	return pfxs
}

func randomRealWorldPrefixes6(prng *rand.Rand, n int) []netip.Prefix {
	set := map[netip.Prefix]netip.Prefix{}
	pfxs := make([]netip.Prefix, 0, n)

	for {
		pfx := randomPrefix6(prng)

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
			set[pfx] = pfx
			pfxs = append(pfxs, pfx)
		}
		if len(set) >= n {
			break
		}
	}
	return pfxs
}

func randomRealWorldPrefixes(prng *rand.Rand, n int) []netip.Prefix {
	pfxs := make([]netip.Prefix, 0, n)
	pfxs = append(pfxs, randomRealWorldPrefixes4(prng, n/2)...)
	pfxs = append(pfxs, randomRealWorldPrefixes6(prng, n-len(pfxs))...)

	prng.Shuffle(len(pfxs), func(i, j int) {
		pfxs[i], pfxs[j] = pfxs[j], pfxs[i]
	})

	return pfxs
}
