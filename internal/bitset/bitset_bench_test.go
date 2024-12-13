//go:build go1.23

package bitset

import (
	"fmt"
	"maps"
	"math/rand/v2"
	"slices"
	"testing"
)

func BenchmarkRankChild(b *testing.B) {
	var bs BitSet
	for range 200 {
		bs.Set(uint(rand.IntN(256)))
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

func BenchmarkRankPrefix(b *testing.B) {
	var bs BitSet
	for range 200 {
		bs.Set(uint(rand.IntN(512)))
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
