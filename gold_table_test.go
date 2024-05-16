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

// goldTable is a simple and slow route table, implemented as a slice of prefixes
// and values as a golden reference for bart.Table.
type goldTable[V any] []goldTableItem[V]

type goldTableItem[V any] struct {
	pfx netip.Prefix
	val V
}

func (t *goldTable[V]) insert(pfx netip.Prefix, val V) {
	pfx = pfx.Masked()
	for i, ent := range *t {
		if ent.pfx == pfx {
			(*t)[i].val = val
			return
		}
	}
	*t = append(*t, goldTableItem[V]{pfx, val})
}

func (t *goldTable[V]) get(pfx netip.Prefix) (val V, ok bool) {
	pfx = pfx.Masked()
	for _, ent := range *t {
		if ent.pfx == pfx {
			return ent.val, true
		}
	}
	return val, false
}

func (t *goldTable[V]) update(pfx netip.Prefix, cb func(V, bool) V) (val V) {
	pfx = pfx.Masked()
	for i, ent := range *t {
		if ent.pfx == pfx {
			// update val
			(*t)[i].val = cb(ent.val, true)
			return
		}
	}
	// new val
	val = cb(val, false)

	*t = append(*t, goldTableItem[V]{pfx, val})
	return val
}

func (ta *goldTable[V]) union(tb *goldTable[V]) {
	for _, bItem := range *tb {
		var match bool
		for i, aItem := range *ta {
			if aItem.pfx == bItem.pfx {
				(*ta)[i] = bItem
				match = true
				break
			}
		}
		if !match {
			*ta = append(*ta, bItem)
		}
	}
}

func (t *goldTable[V]) lookup(addr netip.Addr) (val V, ok bool) {
	bestLen := -1

	for _, item := range *t {
		if item.pfx.Contains(addr) && item.pfx.Bits() > bestLen {
			val = item.val
			ok = true
			bestLen = item.pfx.Bits()
		}
	}
	return
}

func (t *goldTable[V]) lookupPfx(pfx netip.Prefix) (val V, ok bool) {
	bestLen := -1

	for _, item := range *t {
		if item.pfx.Overlaps(pfx) && item.pfx.Bits() <= pfx.Bits() && item.pfx.Bits() > bestLen {
			val = item.val
			ok = true
			bestLen = item.pfx.Bits()
		}
	}
	return
}

func (t *goldTable[V]) lookupPfxLPM(pfx netip.Prefix) (lpm netip.Prefix, val V, ok bool) {
	bestLen := -1

	for _, item := range *t {
		if item.pfx.Overlaps(pfx) && item.pfx.Bits() <= pfx.Bits() && item.pfx.Bits() > bestLen {
			val = item.val
			lpm = item.pfx
			ok = true
			bestLen = item.pfx.Bits()
		}
	}
	return
}

func (t *goldTable[V]) subnets(pfx netip.Prefix) []netip.Prefix {
	var result []netip.Prefix

	for _, item := range *t {
		if pfx.Overlaps(item.pfx) && pfx.Bits() <= item.pfx.Bits() {
			result = append(result, item.pfx)
		}
	}
	slices.SortFunc(result, cmpPrefix)
	return result
}

func (t *goldTable[V]) supernets(pfx netip.Prefix) []netip.Prefix {
	var result []netip.Prefix

	for _, item := range *t {
		if item.pfx.Overlaps(pfx) && item.pfx.Bits() <= pfx.Bits() {
			result = append(result, item.pfx)
		}
	}
	slices.SortFunc(result, cmpPrefix)
	return result
}

func (t *goldTable[V]) overlapsPrefix(pfx netip.Prefix) bool {
	for _, p := range *t {
		if p.pfx.Overlaps(pfx) {
			return true
		}
	}
	return false
}

func (ta *goldTable[V]) overlaps(tb *goldTable[V]) bool {
	for _, aItem := range *ta {
		for _, bItem := range *tb {
			if aItem.pfx.Overlaps(bItem.pfx) {
				return true
			}
		}
	}
	return false
}

// sort, inplace by netip.Prefix, all prefixes are in normalized form
func (t *goldTable[V]) sort() {
	slices.SortFunc(*t, func(a, b goldTableItem[V]) int {
		if cmp := a.pfx.Addr().Compare(b.pfx.Addr()); cmp != 0 {
			return cmp
		}
		return cmp.Compare(a.pfx.Bits(), b.pfx.Bits())
	})
}

// randomPrefixes returns n randomly generated prefixes and associated values,
// distributed equally between IPv4 and IPv6.
func randomPrefixes(n int) []goldTableItem[int] {
	pfxs := randomPrefixes4(n / 2)
	pfxs = append(pfxs, randomPrefixes6(n-len(pfxs))...)
	return pfxs
}

// randomPrefixes4 returns n randomly generated IPv4 prefixes and associated values.
// skip default route
func randomPrefixes4(n int) []goldTableItem[int] {
	pfxs := map[netip.Prefix]bool{}

	for len(pfxs) < n {
		bits := rand.Intn(32)
		bits += 1
		pfx, err := randomAddr4().Prefix(bits)
		if err != nil {
			panic(err)
		}
		pfxs[pfx] = true
	}

	ret := make([]goldTableItem[int], 0, len(pfxs))
	for pfx := range pfxs {
		ret = append(ret, goldTableItem[int]{pfx, rand.Int()})
	}

	return ret
}

// randomPrefixes6 returns n randomly generated IPv4 prefixes and associated values.
// skip default route
func randomPrefixes6(n int) []goldTableItem[int] {
	pfxs := map[netip.Prefix]bool{}

	for len(pfxs) < n {
		bits := rand.Intn(128)
		bits += 1
		pfx, err := randomAddr6().Prefix(bits)
		if err != nil {
			panic(err)
		}
		pfxs[pfx] = true
	}

	ret := make([]goldTableItem[int], 0, len(pfxs))
	for pfx := range pfxs {
		ret = append(ret, goldTableItem[int]{pfx, rand.Int()})
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
