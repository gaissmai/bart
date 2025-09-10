// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"math/rand"

	"github.com/gaissmai/bart/internal/art"
)

// goldStrideTbl, is an 8-bit slow routing table, implemented as a slice
// as a correctness reference.
type goldStrideTbl[V any] []goldStrideItem[V]

type goldStrideItem[V any] struct {
	octet uint8
	bits  uint8
	val   V
}

func (t *goldStrideTbl[V]) insertMany(strides []goldStrideItem[V]) *goldStrideTbl[V] {
	conv := goldStrideTbl[V](strides)
	t = &conv
	return t
}

// delete prefix
func (t *goldStrideTbl[V]) delete(octet, prefixLen uint8) {
	pfx := make([]goldStrideItem[V], 0, len(*t))
	for _, e := range *t {
		if e.octet == octet && e.bits == prefixLen {
			continue
		}
		pfx = append(pfx, e)
	}
	*t = pfx
}

// lpm, longest-prefix-match
func (t *goldStrideTbl[V]) lpm(octet byte) (ret V, ok bool) {
	const noMatch = -1
	longest := noMatch
	for _, e := range *t {
		if octet&pfxMask(e.bits) == e.octet && int(e.bits) >= longest {
			ret = e.val
			longest = int(e.bits)
		}
	}
	return ret, longest != noMatch
}

// strideOverlapsPrefix
func (t *goldStrideTbl[V]) strideOverlapsPrefix(octet, prefixLen uint8) bool {
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

// strideOverlaps
func (ta *goldStrideTbl[V]) strideOverlaps(tb *goldStrideTbl[V]) bool {
	for _, aItem := range *ta {
		for _, bItem := range *tb {
			minBits := aItem.bits
			if bItem.bits < minBits {
				minBits = bItem.bits
			}
			if aItem.octet&pfxMask(minBits) == bItem.octet&pfxMask(minBits) {
				return true
			}
		}
	}
	return false
}

func pfxMask(pfxLen uint8) uint8 {
	return 0xFF << (8 - pfxLen)
}

func allStridePfxs() []goldStrideItem[int] {
	ret := make([]goldStrideItem[int], 0, maxItems)
	for idx := 1; idx < maxItems; idx++ {
		//nolint:gosec
		octet, bits := art.IdxToPfx(uint8(idx))
		ret = append(ret, goldStrideItem[int]{octet, bits, idx})
	}
	return ret
}

func shuffleStridePfxs(pfxs []goldStrideItem[int]) []goldStrideItem[int] {
	rand.Shuffle(len(pfxs), func(i, j int) { pfxs[i], pfxs[j] = pfxs[j], pfxs[i] })
	return pfxs
}
