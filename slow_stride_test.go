// Copyright (c) Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"math/rand"
)

// slowST is an 8-bit slow routing table, implemented as a slice
// as a correctness reference.
type slowST[V any] struct {
	entries []slowSTEntry[V]
}

type slowSTEntry[V any] struct {
	octet uint
	bits  int
	val   V
}

func (st *slowST[V]) delete(octet uint, prefixLen int) {
	pfx := make([]slowSTEntry[V], 0, len(st.entries))
	for _, e := range st.entries {
		if e.octet == octet && e.bits == prefixLen {
			continue
		}
		pfx = append(pfx, e)
	}
	st.entries = pfx
}

// lpm, longest-prefix-match
func (st *slowST[V]) lpm(octet uint) (ret V, ok bool) {
	const noMatch = -1
	longest := noMatch
	for _, e := range st.entries {
		if octet&pfxMask(e.bits) == e.octet && e.bits >= longest {
			ret = e.val
			longest = e.bits
		}
	}
	return ret, longest != noMatch
}

func (st *slowST[T]) overlapsPrefix(octet uint8, prefixLen int) bool {
	for _, e := range st.entries {
		minBits := prefixLen
		if e.bits < minBits {
			minBits = e.bits
		}
		mask := ^hostMasks[minBits]
		if octet&mask == uint8(e.octet)&mask {
			return true
		}
	}
	return false
}

func (st *slowST[T]) overlaps(so *slowST[T]) bool {
	for _, tp := range st.entries {
		for _, op := range so.entries {
			minBits := tp.bits
			if op.bits < minBits {
				minBits = op.bits
			}
			if tp.octet&pfxMask(minBits) == op.octet&pfxMask(minBits) {
				return true
			}
		}
	}
	return false
}

func pfxMask(pfxLen int) uint {
	return 0xFF << (strideLen - pfxLen)
}

func allPrefixes() []slowSTEntry[int] {
	ret := make([]slowSTEntry[int], 0, maxNodePrefixes-1)
	for idx := 1; idx < maxNodePrefixes; idx++ {
		octet, bits := baseIndexToPrefix(uint(idx))
		ret = append(ret, slowSTEntry[int]{octet, bits, idx})
	}
	return ret
}

func shufflePrefixes(pfxs []slowSTEntry[int]) []slowSTEntry[int] {
	rand.Shuffle(len(pfxs), func(i, j int) { pfxs[i], pfxs[j] = pfxs[j], pfxs[i] })
	return pfxs
}
