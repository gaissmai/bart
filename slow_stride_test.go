// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause
//
// tests and benchmarks copied from github.com/tailscale/art
// and modified for this implementation by:
//
// Copyright (c) Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"bytes"
	"fmt"
	"math/rand"
	"sort"
)

// slowTable is an 8-bit routing table implemented as a set of prefixes that are
// explicitly scanned in full for every route lookup. It is very slow, but also
// reasonably easy to verify by inspection, and so a good comparison target for
// strideTable.
type slowTable[V any] struct {
	prefixes []slowEntry[V]
}

type slowEntry[V any] struct {
	addr uint
	bits int
	val  V
}

func (st *slowTable[V]) String() string {
	pfxs := append([]slowEntry[V](nil), st.prefixes...)
	sort.Slice(pfxs, func(i, j int) bool {
		if pfxs[i].bits != pfxs[j].bits {
			return pfxs[i].bits < pfxs[j].bits
		}
		return pfxs[i].addr < pfxs[j].addr
	})
	var ret bytes.Buffer
	for _, pfx := range pfxs {
		fmt.Fprintf(&ret, "%3d/%d (%08b/%08b) = %v\n", pfx.addr, pfx.bits, pfx.addr, pfxMask(pfx.bits), pfx.val)
	}
	return ret.String()
}

func (st *slowTable[V]) delete(addr uint, prefixLen int) {
	pfx := make([]slowEntry[V], 0, len(st.prefixes))
	for _, e := range st.prefixes {
		if e.addr == addr && e.bits == prefixLen {
			continue
		}
		pfx = append(pfx, e)
	}
	st.prefixes = pfx
}

// get, longest-prefix-match
func (st *slowTable[V]) get(addr uint) (ret V, ok bool) {
	const noMatch = -1
	longest := noMatch
	for _, e := range st.prefixes {
		if addr&pfxMask(e.bits) == e.addr && e.bits >= longest {
			ret = e.val
			longest = e.bits
		}
	}
	return ret, longest != noMatch
}

func (st *slowTable[T]) overlapsPrefix(addr uint8, prefixLen int) bool {
	for _, e := range st.prefixes {
		minBits := prefixLen
		if e.bits < minBits {
			minBits = e.bits
		}
		mask := ^hostMasks[minBits]
		if addr&mask == uint8(e.addr)&mask {
			return true
		}
	}
	return false
}

func (st *slowTable[T]) overlaps(so *slowTable[T]) bool {
	for _, tp := range st.prefixes {
		for _, op := range so.prefixes {
			minBits := tp.bits
			if op.bits < minBits {
				minBits = op.bits
			}
			if tp.addr&pfxMask(minBits) == op.addr&pfxMask(minBits) {
				return true
			}
		}
	}
	return false
}

func pfxMask(pfxLen int) uint {
	return 0xFF << (strideLen - pfxLen)
}

func allPrefixes() []slowEntry[int] {
	ret := make([]slowEntry[int], 0, maxNodePrefixes-1)
	for idx := 1; idx < maxNodePrefixes; idx++ {
		addr, bits := baseIndexToPrefix(uint(idx))
		ret = append(ret, slowEntry[int]{addr, bits, idx})
	}
	return ret
}

func shufflePrefixes(pfxs []slowEntry[int]) []slowEntry[int] {
	rand.Shuffle(len(pfxs), func(i, j int) { pfxs[i], pfxs[j] = pfxs[j], pfxs[i] })
	return pfxs
}
