// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package sparse

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"
)

// #####################################################
//  Benchmarks for Array256
//
//  Design notes:
//
//  - The probe sequences are long (1 MiB of uint8) and random. A short or
//    repeating sequence is memorized by the branch predictor and would hide
//    the cost of data dependent branches.
//  - The "fixed" probe repeats one index. It is the fully predictable best
//    case and only meant as a lower bound.
//  - The reads are compared against the former implementation based on
//    BitSet256.Rank (impl=rank) in the same binary, so that both variants
//    run under identical conditions. The writes have no in-binary reference,
//    compare them across commits with benchstat.
//  - Every benchmark runs for several fill levels, from a sparse trie node
//    to a completely full one.
// #####################################################

const (
	// probeCount probes are generated per benchmark, a power of two for cheap masking.
	probeCount = 1 << 20
	probeMask  = probeCount - 1
)

// fills are the numbers of occupied slots.
var fills = []int{8, 40, 128, 256}

// sinks keep the results alive, the compiler must not drop the calls.
var (
	sinkInt  int
	sinkBool bool
)

// #####################################################
//  helpers
// #####################################################

// pick returns n distinct indices in random order.
func pick(prng *rand.Rand, n int) []uint8 {
	perm := prng.Perm(256)

	idxs := make([]uint8, n)
	for k := range idxs {
		idxs[k] = uint8(perm[k])
	}
	return idxs
}

// allIndices returns all indices [0..255] in ascending order.
func allIndices() []uint8 {
	s := make([]uint8, 256)
	for i := range s {
		s[i] = uint8(i)
	}
	return s
}

// absentOf returns the indices in [0..255] that are not in present.
func absentOf(present []uint8) []uint8 {
	var in [256]bool
	for _, i := range present {
		in[i] = true
	}

	var out []uint8
	for i := range 256 {
		if !in[i] {
			out = append(out, uint8(i))
		}
	}
	return out
}

// probesFrom returns probeCount random picks from idxs.
func probesFrom(prng *rand.Rand, idxs []uint8) []uint8 {
	p := make([]uint8, probeCount)
	for k := range p {
		p[k] = idxs[prng.IntN(len(idxs))]
	}
	return p
}

// fixedProbes returns probeCount times the same index.
func fixedProbes(idx uint8) []uint8 {
	p := make([]uint8, probeCount)
	for k := range p {
		p[k] = idx
	}
	return p
}

// newFilled builds an Array256 by inserting the indices in the given order.
func newFilled(idxs []uint8) *Array256[int] {
	a := new(Array256[int])
	for k, i := range idxs {
		a.InsertAt(i, k)
	}
	return a
}

// getUnfused is the previous implementation of Get: Test followed by rank0.
func getUnfused[T any](a *Array256[T], i uint8) (value T, ok bool) {
	if a.Test(i) {
		return a.Items[a.rank0(i)], true
	}
	return
}

type ordering struct {
	name string
	idxs []uint8
}

// orderingsOf returns the given random order plus the ascending and the
// descending order of the same indices.
func orderingsOf(random []uint8) []ordering {
	asc := slices.Clone(random)
	slices.Sort(asc)

	desc := slices.Clone(asc)
	slices.Reverse(desc)

	return []ordering{
		{"random", random},
		{"ascending", asc},
		{"descending", desc},
	}
}

// #####################################################
//  reads
// #####################################################

// BenchmarkArray256Get measures the lookup of a slot, with Get and with the
// Test+MustGet pair used by the trie traversal, each against the former
// Rank based implementation.
func BenchmarkArray256Get(b *testing.B) {
	for _, n := range fills {
		prng := rand.New(rand.NewPCG(uint64(n), 1))

		present := pick(prng, n)
		a := newFilled(present)

		probeSets := []struct {
			name   string
			probes []uint8
		}{
			{"hit", probesFrom(prng, present)},
			{"mixed", probesFrom(prng, allIndices())}, // hits and misses
			{"fixed", fixedProbes(present[0])},
		}

		for _, ps := range probeSets {
			probes := ps.probes
			prefix := fmt.Sprintf("n=%d/probe=%s", n, ps.name)

			b.Run(prefix+"/form=Get/impl=cum", func(b *testing.B) {
				for i := 0; b.Loop(); i++ {
					sinkInt, sinkBool = a.Get(probes[i&probeMask])
				}
			})

			b.Run(prefix+"/form=Test+MustGet/impl=cum", func(b *testing.B) {
				for i := 0; b.Loop(); i++ {
					idx := probes[i&probeMask]
					if a.Test(idx) {
						sinkInt = a.MustGet(idx)
					}
				}
			})

			b.Run(prefix+"/form=Get/impl=unfused", func(b *testing.B) {
				for i := 0; b.Loop(); i++ {
					sinkInt, sinkBool = getUnfused(a, probes[i&probeMask])
				}
			})
		}
	}
}

// BenchmarkArray256MustGet measures MustGet in isolation, only with indices
// that are present.
func BenchmarkArray256MustGet(b *testing.B) {
	for _, n := range fills {
		prng := rand.New(rand.NewPCG(uint64(n), 2))

		present := pick(prng, n)
		a := newFilled(present)

		probeSets := []struct {
			name   string
			probes []uint8
		}{
			{"hit", probesFrom(prng, present)},
			{"fixed", fixedProbes(present[0])},
		}

		for _, ps := range probeSets {
			probes := ps.probes
			prefix := fmt.Sprintf("n=%d/probe=%s", n, ps.name)

			b.Run(prefix+"/impl=cum", func(b *testing.B) {
				for i := 0; b.Loop(); i++ {
					sinkInt = a.MustGet(probes[i&probeMask])
				}
			})
		}
	}
}

// #####################################################
//  writes
// #####################################################

// BenchmarkArray256InsertAt measures inserting new slots and overwriting
// existing ones.
//
//   - op=grow-alloc: from empty to n items, the Items slice has to grow
//     (allocations included, see allocs/op).
//   - op=grow: the same, but the capacity of Items is reused, so only the
//     shifting of the items and the update of the rank cache is measured.
//     The orders differ in the amount of shifting: ascending appends at
//     the end, descending always inserts at the front.
//   - op=overwrite: all slots exist, no shifting and no rank cache update.
//
// The measured time is per InsertAt, the reset after n inserts is included
// and amortized over n operations.
func BenchmarkArray256InsertAt(b *testing.B) {
	for _, n := range fills {
		prng := rand.New(rand.NewPCG(uint64(n), 3))

		order := pick(prng, n)
		orders := orderingsOf(order)

		b.Run(fmt.Sprintf("n=%d/op=grow-alloc/order=random", n), func(b *testing.B) {
			b.ReportAllocs()

			var a Array256[int]
			k := 0
			for b.Loop() {
				if k == n {
					a = Array256[int]{} // drop the backing array, Items has to grow again
					k = 0
				}
				sinkBool = a.InsertAt(order[k], k)
				k++
			}
		})

		for _, o := range orders {
			idxs := o.idxs

			b.Run(fmt.Sprintf("n=%d/op=grow/order=%s", n, o.name), func(b *testing.B) {
				a := Array256[int]{Items: make([]int, 0, n)}
				k := 0
				for b.Loop() {
					if k == n {
						// clear bitset and rank cache, keep the capacity of Items
						a = Array256[int]{Items: a.Items[:0]}
						k = 0
					}
					sinkBool = a.InsertAt(idxs[k], k)
					k++
				}
			})
		}

		b.Run(fmt.Sprintf("n=%d/op=overwrite", n), func(b *testing.B) {
			a := newFilled(order)
			probes := probesFrom(prng, order)

			for i := 0; b.Loop(); i++ {
				sinkBool = a.InsertAt(probes[i&probeMask], i)
			}
		})
	}
}

// BenchmarkArray256DeleteAt measures removing existing slots and deleting
// indices that are not present.
//
//   - op=shrink: from n items to empty. The orders differ in the amount of
//     shifting: descending removes the last item, ascending always the first.
//   - op=absent: the index is not set, only the bit test is executed.
//     Not available for n=256.
//
// The measured time is per DeleteAt, the restore of the full array after n
// deletes (a copy of n items) is included and amortized over n operations.
func BenchmarkArray256DeleteAt(b *testing.B) {
	for _, n := range fills {
		prng := rand.New(rand.NewPCG(uint64(n), 4))

		present := pick(prng, n)
		full := newFilled(present)

		for _, o := range orderingsOf(present) {
			idxs := o.idxs

			b.Run(fmt.Sprintf("n=%d/op=shrink/order=%s", n, o.name), func(b *testing.B) {
				a := full.Copy()
				k := 0
				for b.Loop() {
					if k == n {
						// restore the full array, reuse the backing array of Items
						a.BitSet256 = full.BitSet256
						a.cum = full.cum
						a.Items = a.Items[:n]
						copy(a.Items, full.Items)
						k = 0
					}
					sinkInt, sinkBool = a.DeleteAt(idxs[k])
					k++
				}
			})
		}

		if absent := absentOf(present); len(absent) > 0 {
			probes := probesFrom(prng, absent)

			b.Run(fmt.Sprintf("n=%d/op=absent", n), func(b *testing.B) {
				a := full.Copy()
				for i := 0; b.Loop(); i++ {
					sinkInt, sinkBool = a.DeleteAt(probes[i&probeMask])
				}
			})
		}
	}
}
