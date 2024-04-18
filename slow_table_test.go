// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause
//
// some tests and benchmarks copied from github.com/tailscale/art
// and modified for this implementation by:
//
// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"cmp"
	crand "crypto/rand"
	"fmt"
	"math/rand"
	"net/netip"
	"slices"
	"strconv"
)

// slowPrefixTable is a routing table implemented as a set of prefixes that are
// explicitly scanned in full for every route lookup. It is very slow, but also
// reasonably easy to verify by inspection, and so a good correctness reference
// for Table.
type slowPrefixTable[V any] struct {
	prefixes []slowPrefixEntry[V]
}

type slowPrefixEntry[V any] struct {
	pfx netip.Prefix
	val V
}

func (s *slowPrefixTable[V]) insert(pfx netip.Prefix, val V) {
	pfx = pfx.Masked()
	for i, ent := range s.prefixes {
		if ent.pfx == pfx {
			s.prefixes[i].val = val
			return
		}
	}
	s.prefixes = append(s.prefixes, slowPrefixEntry[V]{pfx, val})
}

func (s *slowPrefixTable[T]) union(o *slowPrefixTable[T]) {
	for _, op := range o.prefixes {
		var match bool
		for i, sp := range s.prefixes {
			if sp.pfx == op.pfx {
				s.prefixes[i] = op
				match = true
				break
			}
		}
		if !match {
			s.prefixes = append(s.prefixes, op)
		}
	}
}

func (s *slowPrefixTable[V]) lookup(addr netip.Addr) (val V, ok bool) {
	bestLen := -1

	for _, item := range s.prefixes {
		if item.pfx.Contains(addr) && item.pfx.Bits() > bestLen {
			val = item.val
			bestLen = item.pfx.Bits()
		}
	}
	return val, bestLen != -1
}

func (s *slowPrefixTable[V]) subnets(pfx netip.Prefix) []netip.Prefix {
	var result []netip.Prefix

	for _, item := range s.prefixes {
		if pfx.Overlaps(item.pfx) && pfx.Bits() <= item.pfx.Bits() {
			result = append(result, item.pfx)
		}
	}
	slices.SortFunc(result, sortByPrefix)
	return result
}

func (s *slowPrefixTable[V]) supernets(pfx netip.Prefix) []netip.Prefix {
	var result []netip.Prefix

	for _, item := range s.prefixes {
		if item.pfx.Overlaps(pfx) && item.pfx.Bits() <= pfx.Bits() {
			result = append(result, item.pfx)
		}
	}
	slices.SortFunc(result, sortByPrefix)
	return result
}

func (s *slowPrefixTable[T]) overlapsPrefix(pfx netip.Prefix) bool {
	for _, p := range s.prefixes {
		if p.pfx.Overlaps(pfx) {
			return true
		}
	}
	return false
}

func (s *slowPrefixTable[T]) overlaps(o *slowPrefixTable[T]) bool {
	for _, tp := range s.prefixes {
		for _, op := range o.prefixes {
			if tp.pfx.Overlaps(op.pfx) {
				return true
			}
		}
	}
	return false
}

// sort, inplace by netip.Prefix
func (s *slowPrefixTable[T]) sort() {
	slices.SortFunc(s.prefixes, func(a, b slowPrefixEntry[T]) int {
		if cmp := a.pfx.Masked().Addr().Compare(b.pfx.Masked().Addr()); cmp != 0 {
			return cmp
		}
		return cmp.Compare(a.pfx.Bits(), b.pfx.Bits())
	})
}

// randomPrefixes returns n randomly generated prefixes and associated values,
// distributed equally between IPv4 and IPv6.
func randomPrefixes(n int) []slowPrefixEntry[int] {
	pfxs := randomPrefixes4(n / 2)
	pfxs = append(pfxs, randomPrefixes6(n-len(pfxs))...)
	return pfxs
}

// randomPrefixes4 returns n randomly generated IPv4 prefixes and associated values.
func randomPrefixes4(n int) []slowPrefixEntry[int] {
	pfxs := map[netip.Prefix]bool{}

	for len(pfxs) < n {
		bits := rand.Intn(33)
		pfx, err := randomAddr4().Prefix(bits)
		if err != nil {
			panic(err)
		}
		pfxs[pfx] = true
	}

	ret := make([]slowPrefixEntry[int], 0, len(pfxs))
	for pfx := range pfxs {
		ret = append(ret, slowPrefixEntry[int]{pfx, rand.Int()})
	}

	return ret
}

// randomPrefixes6 returns n randomly generated IPv4 prefixes and associated values.
func randomPrefixes6(n int) []slowPrefixEntry[int] {
	pfxs := map[netip.Prefix]bool{}

	for len(pfxs) < n {
		bits := rand.Intn(129)
		pfx, err := randomAddr6().Prefix(bits)
		if err != nil {
			panic(err)
		}
		pfxs[pfx] = true
	}

	ret := make([]slowPrefixEntry[int], 0, len(pfxs))
	for pfx := range pfxs {
		ret = append(ret, slowPrefixEntry[int]{pfx, rand.Int()})
	}

	return ret
}

// randomAddr returns a randomly generated IP address.
func randomAddr() netip.Addr {
	if rand.Intn(2) == 1 {
		return randomAddr6()
	}
	return randomAddr4()
}

// randomAddr4 returns a randomly generated IPv4 address.
func randomAddr4() netip.Addr {
	var b [4]byte
	if _, err := crand.Read(b[:]); err != nil {
		panic(err)
	}
	return netip.AddrFrom4(b)
}

// randomAddr6 returns a randomly generated IPv6 address.
func randomAddr6() netip.Addr {
	var b [16]byte
	if _, err := crand.Read(b[:]); err != nil {
		panic(err)
	}
	return netip.AddrFrom16(b)
}

// roundFloat64 rounds f to 2 decimal places, for display.
//
// It round-trips through a float->string->float conversion, so should not be
// used in a performance critical setting.
func roundFloat64(f float64) float64 {
	s := fmt.Sprintf("%.2f", f)
	ret, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic(err)
	}
	return ret
}

// #####################################################################

// randomPrefixes returns n randomly generated prefixes and
// associated values, distributed equally between IPv4 and IPv6.
func randomPrefix() netip.Prefix {
	if rand.Intn(2) == 1 {
		return randomPrefix4()
	} else {
		return randomPrefix6()
	}
}

func randomPrefix4() netip.Prefix {
	bits := rand.Intn(33)
	pfx, err := randomIP4().Prefix(bits)
	if err != nil {
		panic(err)
	}
	return pfx
}

func randomPrefix6() netip.Prefix {
	bits := rand.Intn(129)
	pfx, err := randomIP6().Prefix(bits)
	if err != nil {
		panic(err)
	}
	return pfx
}

func randomIP4() netip.Addr {
	var b [4]byte
	if _, err := crand.Read(b[:]); err != nil {
		panic(err)
	}
	return netip.AddrFrom4(b)
}

func randomIP6() netip.Addr {
	var b [16]byte
	if _, err := crand.Read(b[:]); err != nil {
		panic(err)
	}
	return netip.AddrFrom16(b)
}
