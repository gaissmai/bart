// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"cmp"
	crand "crypto/rand"
	"math/rand"
	"net/netip"
	"slices"
)

// slowRT is a simple and slow route table, implemented as a slice of prefixes
// as a correctness reference for bart.Table.
type slowRT[V any] struct {
	entries []slowRTEntry[V]
}

type slowRTEntry[V any] struct {
	pfx netip.Prefix
	val V
}

func (s *slowRT[V]) insert(pfx netip.Prefix, val V) {
	pfx = pfx.Masked()
	for i, ent := range s.entries {
		if ent.pfx == pfx {
			s.entries[i].val = val
			return
		}
	}
	s.entries = append(s.entries, slowRTEntry[V]{pfx, val})
}

func (s *slowRT[T]) union(o *slowRT[T]) {
	for _, op := range o.entries {
		var match bool
		for i, sp := range s.entries {
			if sp.pfx == op.pfx {
				s.entries[i] = op
				match = true
				break
			}
		}
		if !match {
			s.entries = append(s.entries, op)
		}
	}
}

func (s *slowRT[V]) lookup(addr netip.Addr) (val V, ok bool) {
	bestLen := -1

	for _, item := range s.entries {
		if item.pfx.Contains(addr) && item.pfx.Bits() > bestLen {
			val = item.val
			ok = true
			bestLen = item.pfx.Bits()
		}
	}
	return
}

func (s *slowRT[V]) lookupPfx(pfx netip.Prefix) (val V, ok bool) {
	bestLen := -1

	for _, item := range s.entries {
		if item.pfx.Overlaps(pfx) && item.pfx.Bits() <= pfx.Bits() && item.pfx.Bits() > bestLen {
			val = item.val
			ok = true
			bestLen = item.pfx.Bits()
		}
	}
	return
}

func (s *slowRT[V]) lookupPfxLPM(pfx netip.Prefix) (lpm netip.Prefix, val V, ok bool) {
	bestLen := -1

	for _, item := range s.entries {
		if item.pfx.Overlaps(pfx) && item.pfx.Bits() <= pfx.Bits() && item.pfx.Bits() > bestLen {
			val = item.val
			lpm = item.pfx
			ok = true
			bestLen = item.pfx.Bits()
		}
	}
	return
}

func (s *slowRT[V]) subnets(pfx netip.Prefix) []netip.Prefix {
	var result []netip.Prefix

	for _, item := range s.entries {
		if pfx.Overlaps(item.pfx) && pfx.Bits() <= item.pfx.Bits() {
			result = append(result, item.pfx)
		}
	}
	slices.SortFunc(result, sortByPrefix)
	return result
}

func (s *slowRT[V]) supernets(pfx netip.Prefix) []netip.Prefix {
	var result []netip.Prefix

	for _, item := range s.entries {
		if item.pfx.Overlaps(pfx) && item.pfx.Bits() <= pfx.Bits() {
			result = append(result, item.pfx)
		}
	}
	slices.SortFunc(result, sortByPrefix)
	return result
}

func (s *slowRT[T]) overlapsPrefix(pfx netip.Prefix) bool {
	for _, p := range s.entries {
		if p.pfx.Overlaps(pfx) {
			return true
		}
	}
	return false
}

func (s *slowRT[T]) overlaps(o *slowRT[T]) bool {
	for _, tp := range s.entries {
		for _, op := range o.entries {
			if tp.pfx.Overlaps(op.pfx) {
				return true
			}
		}
	}
	return false
}

// sort, inplace by netip.Prefix, all prefixes are in normalized form
func (s *slowRT[T]) sort() {
	slices.SortFunc(s.entries, func(a, b slowRTEntry[T]) int {
		if cmp := a.pfx.Addr().Compare(b.pfx.Addr()); cmp != 0 {
			return cmp
		}
		return cmp.Compare(a.pfx.Bits(), b.pfx.Bits())
	})
}

// randomPrefixes returns n randomly generated prefixes and associated values,
// distributed equally between IPv4 and IPv6.
func randomPrefixes(n int) []slowRTEntry[int] {
	pfxs := randomPrefixes4(n / 2)
	pfxs = append(pfxs, randomPrefixes6(n-len(pfxs))...)
	return pfxs
}

// randomPrefixes4 returns n randomly generated IPv4 prefixes and associated values.
func randomPrefixes4(n int) []slowRTEntry[int] {
	pfxs := map[netip.Prefix]bool{}

	for len(pfxs) < n {
		bits := rand.Intn(33)
		pfx, err := randomAddr4().Prefix(bits)
		if err != nil {
			panic(err)
		}
		pfxs[pfx] = true
	}

	ret := make([]slowRTEntry[int], 0, len(pfxs))
	for pfx := range pfxs {
		ret = append(ret, slowRTEntry[int]{pfx, rand.Int()})
	}

	return ret
}

// randomPrefixes6 returns n randomly generated IPv4 prefixes and associated values.
func randomPrefixes6(n int) []slowRTEntry[int] {
	pfxs := map[netip.Prefix]bool{}

	for len(pfxs) < n {
		bits := rand.Intn(129)
		pfx, err := randomAddr6().Prefix(bits)
		if err != nil {
			panic(err)
		}
		pfxs[pfx] = true
	}

	ret := make([]slowRTEntry[int], 0, len(pfxs))
	for pfx := range pfxs {
		ret = append(ret, slowRTEntry[int]{pfx, rand.Int()})
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
