package main

import (
	"math/rand/v2"
	"net/netip"

	"github.com/gaissmai/bart"
)

var (
	prng      = rand.New(rand.NewPCG(42, 42))
	tbl       = &bart.Table[struct{}]{}
	ipProbes  = []netip.Addr{}
	pfxProbes = []netip.Prefix{}
)

func main() {
	for range 11 {
		ipProbes = append(ipProbes, randomAddr())
	}

	for range 11 {
		pfxProbes = append(pfxProbes, randomPrefix())
	}

	for range 10_000 {
		tbl.Insert(randomPrefix(), struct{}{})
	}

	for i := range 1_000_000_000 {
		tbl.LookupPrefixLPM(pfxProbes[i&10])
	}
}

// randomPrefix returns a randomly generated prefix
func randomPrefix() netip.Prefix {
	if prng.IntN(2) == 1 {
		return randomPrefix4()
	}
	return randomPrefix6()
}

func randomPrefix4() netip.Prefix {
	bits := prng.IntN(33)
	pfx, err := randomIP4().Prefix(bits)
	if err != nil {
		panic(err)
	}
	return pfx
}

func randomPrefix6() netip.Prefix {
	bits := prng.IntN(129)
	pfx, err := randomIP6().Prefix(bits)
	if err != nil {
		panic(err)
	}
	return pfx
}

func randomIP4() netip.Addr {
	var b [4]byte
	for i := range b {
		b[i] = byte(prng.Uint32() & 0xff)
	}
	return netip.AddrFrom4(b)
}

func randomIP6() netip.Addr {
	var b [16]byte
	for i := range b {
		b[i] = byte(prng.Uint32() & 0xff)
	}
	return netip.AddrFrom16(b)
}

func randomAddr() netip.Addr {
	if prng.IntN(2) == 1 {
		return randomIP4()
	}
	return randomIP6()
}
