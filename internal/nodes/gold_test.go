package nodes

import (
	"fmt"
	"math/rand/v2"
	"net/netip"
	"slices"
)

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
	set := make(map[netip.Prefix]struct{})
	pfxs := make([]netip.Prefix, 0, n)

	for len(set) < n {
		pfx := randomPrefix4(prng)

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

func randomRealWorldPrefixes6(prng *rand.Rand, n int) []netip.Prefix {
	set := make(map[netip.Prefix]struct{})
	pfxs := make([]netip.Prefix, 0, n)

	for len(set) < n {
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
			set[pfx] = struct{}{}
			pfxs = append(pfxs, pfx)
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

// goldTable is a simple and slow route table, implemented as a slice of prefixes
// and values as a golden reference for bart.Table.
type goldTable[V any] []goldTableItem[V]

type goldTableItem[V any] struct {
	pfx netip.Prefix
	val V
}

func (g goldTableItem[V]) String() string {
	return fmt.Sprintf("(%s, %v)", g.pfx, g.val)
}

func (t *goldTable[V]) insert(pfx netip.Prefix, val V) {
	pfx = pfx.Masked()
	for i, item := range *t {
		if item.pfx == pfx {
			(*t)[i].val = val // de-dupe
			return
		}
	}
	*t = append(*t, goldTableItem[V]{pfx, val})
}

func (t *goldTable[V]) delete(pfx netip.Prefix) (exists bool) {
	pfx = pfx.Masked()

	for i, item := range *t {
		if item.pfx == pfx {
			*t = slices.Delete(*t, i, i+1)
			return true
		}
	}
	return false
}

func (t goldTable[V]) allSorted() []netip.Prefix {
	var result []netip.Prefix

	for _, item := range t {
		result = append(result, item.pfx)
	}
	slices.SortFunc(result, CmpPrefix)
	return result
}

/*

func (t goldTable[V]) get(pfx netip.Prefix) (val V, ok bool) {
	pfx = pfx.Masked()
	for _, item := range t {
		if item.pfx == pfx {
			return item.val, true
		}
	}
	return val, false
}

func (t *goldTable[V]) update(pfx netip.Prefix, cb func(V, bool) V) (val V) {
	pfx = pfx.Masked()
	for i, item := range *t {
		if item.pfx == pfx {
			// update val
			val = cb(item.val, true)
			(*t)[i].val = val
			return val
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

func (t goldTable[V]) lookup(addr netip.Addr) (val V, ok bool) {
	bestLen := -1

	for _, item := range t {
		if item.pfx.Contains(addr) && item.pfx.Bits() > bestLen {
			val = item.val
			ok = true
			bestLen = item.pfx.Bits()
		}
	}
	return val, ok
}

func (t goldTable[V]) lookupPrefix(pfx netip.Prefix) (val V, ok bool) {
	pfx = pfx.Masked()
	bestLen := -1

	for _, item := range t {
		if item.pfx.Overlaps(pfx) && item.pfx.Bits() <= pfx.Bits() && item.pfx.Bits() > bestLen {
			val = item.val
			ok = true
			bestLen = item.pfx.Bits()
		}
	}
	return val, ok
}

func (t goldTable[V]) lookupPrefixLPM(pfx netip.Prefix) (lpm netip.Prefix, val V, ok bool) {
	pfx = pfx.Masked()
	bestLen := -1

	for _, item := range t {
		if item.pfx.Overlaps(pfx) && item.pfx.Bits() <= pfx.Bits() && item.pfx.Bits() > bestLen {
			val = item.val
			lpm = item.pfx
			ok = true
			bestLen = item.pfx.Bits()
		}
	}
	return lpm, val, ok
}

func (t goldTable[V]) subnets(pfx netip.Prefix) []netip.Prefix {
	pfx = pfx.Masked()
	var result []netip.Prefix

	for _, item := range t {
		if pfx.Overlaps(item.pfx) && pfx.Bits() <= item.pfx.Bits() {
			result = append(result, item.pfx)
		}
	}
	slices.SortFunc(result, CmpPrefix)
	return result
}

func (t goldTable[V]) supernets(pfx netip.Prefix) []netip.Prefix {
	pfx = pfx.Masked()
	var result []netip.Prefix

	for _, item := range t {
		if item.pfx.Overlaps(pfx) && item.pfx.Bits() <= pfx.Bits() {
			result = append(result, item.pfx)
		}
	}
	slices.SortFunc(result, CmpPrefix)
	slices.Reverse(result)
	return result
}

//nolint:unused
func (t goldTable[V]) overlapsPrefix(pfx netip.Prefix) bool {
	pfx = pfx.Masked()
	for _, p := range t {
		if p.pfx.Overlaps(pfx) {
			return true
		}
	}
	return false
}

//nolint:unused
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
		return CmpPrefix(a.pfx, b.pfx)
	})
}

*/
