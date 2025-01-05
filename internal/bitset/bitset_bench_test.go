//go:build go1.23

// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bitset

import (
	"fmt"
	"maps"
	"math/rand/v2"
	"slices"
	"testing"
)

func BenchmarkBitSetRankChild(b *testing.B) {
	var bs BitSet
	for range 200 {
		bs = bs.Set(uint(rand.IntN(256)))
	}

	// make unique random numbers
	randsUnique := map[int]bool{}
	for range 10 {
		randsUnique[rand.IntN(256)] = true
	}

	// sort them ascending
	rands := slices.Collect(maps.Keys(randsUnique))
	slices.Sort(rands)

	// benchmark Rank with them
	for _, r := range rands {
		b.Run(fmt.Sprintf("%3d", r), func(b *testing.B) {
			for range b.N {
				_ = bs.Rank(uint(r))
			}
		})
	}
}

func BenchmarkBitSetRankPrefix(b *testing.B) {
	var bs BitSet
	for range 200 {
		bs = bs.Set(uint(rand.IntN(512)))
	}

	// make uniques random numbers
	randsUnique := map[int]bool{}
	for range 10 {
		randsUnique[rand.IntN(512)] = true
	}

	// sort them ascending
	rands := slices.Collect(maps.Keys(randsUnique))
	slices.Sort(rands)

	// benchmark Rank with them
	for _, r := range rands {
		b.Run(fmt.Sprintf("%3d", r), func(b *testing.B) {
			for range b.N {
				_ = bs.Rank(uint(r))
			}
		})
	}
}

func BenchmarkBitSetInPlace(b *testing.B) {
	bs := BitSet([]uint64{})
	cs := BitSet([]uint64{})
	for range 200 {
		bs = bs.Set(uint(rand.IntN(512)))
	}
	for range 200 {
		cs = cs.Set(uint(rand.IntN(512)))
	}

	b.Run("InPlaceIntersection len(b)==len(c)", func(b *testing.B) {
		for range b.N {
			(&bs).InPlaceIntersection(cs)
		}
	})

	bs = BitSet([]uint64{})
	cs = BitSet([]uint64{})
	for range 200 {
		bs = bs.Set(uint(rand.IntN(512)))
	}
	for range 200 {
		cs = cs.Set(uint(rand.IntN(512)))
	}
	b.Run("InPlaceUnion len(b)==len(c)", func(b *testing.B) {
		for range b.N {
			(&bs).InPlaceUnion(cs)
		}
	})

	bs = BitSet([]uint64{})
	cs = BitSet([]uint64{})
	for range 200 {
		bs = bs.Set(uint(rand.IntN(512)))
	}
	for range 200 {
		cs = cs.Set(uint(rand.IntN(256)))
	}
	b.Run("InPlaceIntersection len(b)>len(c)", func(b *testing.B) {
		for range b.N {
			(&bs).InPlaceIntersection(cs)
		}
	})

	bs = BitSet([]uint64{})
	cs = BitSet([]uint64{})
	for range 200 {
		bs = bs.Set(uint(rand.IntN(512)))
	}
	for range 200 {
		cs = cs.Set(uint(rand.IntN(256)))
	}
	b.Run("InPlaceUnion len(b)>len(c)", func(b *testing.B) {
		for range b.N {
			(&bs).InPlaceUnion(cs)
		}
	})
	bs = BitSet([]uint64{})
	cs = BitSet([]uint64{})
	for range 200 {
		bs = bs.Set(uint(rand.IntN(256)))
	}
	for range 200 {
		cs = cs.Set(uint(rand.IntN(512)))
	}
	b.Run("InPlaceIntersection len(b)<len(c)", func(b *testing.B) {
		for range b.N {
			(&bs).InPlaceIntersection(cs)
		}
	})

	bs = BitSet([]uint64{})
	cs = BitSet([]uint64{})
	for range 200 {
		bs = bs.Set(uint(rand.IntN(256)))
	}
	for range 200 {
		cs = cs.Set(uint(rand.IntN(512)))
	}
	b.Run("InPlaceUnion len(b)<len(c)", func(b *testing.B) {
		for range b.N {
			(&bs).InPlaceUnion(cs)
		}
	})
}

func BenchmarkWorstCaseBitSetLPM(b *testing.B) {
	idx := BitSet{}
	for _, i := range []uint{1, 3, 7, 15, 31, 62, 125, 511} {
		idx = idx.Set(i)
	}

	b.Run("lpmTestWorstCase", func(b *testing.B) {
		pfx := BitSet{}.Set(510)

		b.ResetTimer()
		for range b.N {
			pfx.IntersectsAny(idx)
		}
	})

	b.Run("lpmGetWorstCase", func(b *testing.B) {
		pfx := BitSet{}.Set(510)

		b.ResetTimer()
		for range b.N {
			pfx.IntersectionTop(idx)
		}
	})
}

func BenchmarkWorstCaseIterLPM(b *testing.B) {
	b.Run("lpmTestWorstCase", func(b *testing.B) {
		pfx := BitSet{}.Set(1)

		b.ResetTimer()
		for range b.N {
			for idx := uint(511); idx > 0; idx >>= 1 {
				if pfx.Test(idx) {
					break
				}
			}
		}
	})
}
