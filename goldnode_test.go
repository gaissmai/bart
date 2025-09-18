// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"math/rand/v2"

	"github.com/gaissmai/bart/internal/art"
)

// goldNode, is an 8-bit slow routing table, implemented as a slice
// as a correctness reference.
type goldNode[V any] []goldNodeItem[V]

type goldNodeItem[V any] struct {
	octet uint8
	bits  uint8
	val   V
}

func (t *goldNode[V]) insertMany(items []goldNodeItem[V]) *goldNode[V] {
	*t = goldNode[V](items)
	return t
}

// deleteItem prefix
func (t *goldNode[V]) deleteItem(octet, prefixLen uint8) {
	pfx := make([]goldNodeItem[V], 0, len(*t))
	for _, e := range *t {
		if e.octet == octet && e.bits == prefixLen {
			continue
		}
		pfx = append(pfx, e)
	}
	*t = pfx
}

// lpm, longest-prefix-match
func (t *goldNode[V]) lpm(octet byte) (ret V, ok bool) {
	const noMatch = -1
	longest := noMatch
	for _, e := range *t {
		if octet&art.NetMask(e.bits) == e.octet && int(e.bits) >= longest {
			ret = e.val
			longest = int(e.bits)
		}
	}
	return ret, longest != noMatch
}

// overlapsPrefix
func (t *goldNode[V]) overlapsPrefix(octet, prefixLen uint8) bool {
	for _, e := range *t {
		minBits := prefixLen
		if e.bits < minBits {
			minBits = e.bits
		}
		mask := art.NetMask(minBits)
		if octet&mask == e.octet&mask {
			return true
		}
	}
	return false
}

// overlaps
func (ta *goldNode[V]) overlaps(tb *goldNode[V]) bool {
	for _, aItem := range *ta {
		for _, bItem := range *tb {
			minBits := aItem.bits
			if bItem.bits < minBits {
				minBits = bItem.bits
			}
			if aItem.octet&art.NetMask(minBits) == bItem.octet&art.NetMask(minBits) {
				return true
			}
		}
	}
	return false
}

func allNodePfxs() []goldNodeItem[int] {
	ret := make([]goldNodeItem[int], 0, maxItems)
	for idx := 1; idx < maxItems; idx++ {
		//nolint:gosec // test-only: idx conversion is safe and deterministic
		octet, bits := art.IdxToPfx(uint8(idx))
		ret = append(ret, goldNodeItem[int]{octet, bits, idx})
	}
	return ret
}

func shuffleNodePfxs(prng *rand.Rand, pfxs []goldNodeItem[int]) []goldNodeItem[int] {
	prng.Shuffle(len(pfxs), func(i, j int) { pfxs[i], pfxs[j] = pfxs[j], pfxs[i] })
	return pfxs
}
