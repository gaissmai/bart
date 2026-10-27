// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bitset

import (
	"fmt"
	"math/rand/v2"
	"sync"
	"testing"
)

// #####################################################
//  Benchmarks for BitSet256
//
//  Design notes:
//
//  - The operations are tiny and inlined, a single call takes about a
//    nanosecond or less. The loop overhead (index calculation, load of the
//    operands, store of the result) is part of the measured time, compare
//    the variants with each other (benchstat -col /impl), not absolute values.
//  - The operands are random sets from a large pool, so the branch predictor
//    cannot memorize the sequence. This matters for FirstSet, LastSet, NextSet
//    and AndTop, whose branches depend on the data.
//  - The benchmarks run for sets with 1, 8, 16, 64, 128, 192 and 256 bits set, from a
//    sparse node to a completely full one. The word-wise operations without
//    branches do not depend on the density, they run only with 128 bits.
//  - impl=loop are loop based reference variants (see the test file), which
//    verify the claim in the source comments that the unrolled versions are
//    faster. impl=unrolled is the implementation in the package.
//  - OnesCount needs the POPCNT instruction. With the default GOAMD64=v1 the
//    compiler checks the CPU feature before every use, build with GOAMD64=v3
//    or higher for the numbers that matter for the library.
// #####################################################

const (
	// setCount random sets are generated per density, 128 KiB per slice.
	setCount = 1 << 12
	setMask  = setCount - 1

	// probeCount random indices, a power of two for cheap masking.
	probeCount = 1 << 16
	probeMask  = probeCount - 1
)

// benchDensities are the numbers of set bits per set.
var benchDensities = []int{1, 8, 16, 64, 128, 192, 256}

// benchInputs holds the random inputs. For every density there are two
// independent slices of sets, a and b, for the binary operations.
type benchInputs struct {
	a, b map[int][]BitSet256
	idxs []uint8
}

// benchData generates the inputs once, on first use.
var benchData = sync.OnceValue(func() *benchInputs {
	prng := rand.New(rand.NewPCG(1, 2))

	d := &benchInputs{
		a:    make(map[int][]BitSet256),
		b:    make(map[int][]BitSet256),
		idxs: make([]uint8, probeCount),
	}

	// exactly n random bits per set
	gen := func(n int) []BitSet256 {
		sets := make([]BitSet256, setCount)
		for i := range sets {
			for _, bit := range prng.Perm(256)[:n] {
				sets[i].Set(uint8(bit))
			}
		}
		return sets
	}

	for _, n := range benchDensities {
		d.a[n] = gen(n)
		d.b[n] = gen(n)
	}

	for i := range d.idxs {
		d.idxs[i] = uint8(prng.UintN(256))
	}

	return d
})

// sinks keep the results alive, the compiler must not drop the calls.
var (
	benchSinkU8   uint8
	benchSinkBool bool
	benchSinkInt  int
	benchSinkSet  BitSet256
)

// eachDensity runs a sub-benchmark for every density. The loops stay in the
// callbacks on purpose, a function value in the loop would prevent inlining.
func eachDensity(b *testing.B, run func(b *testing.B, n int, a, c []BitSet256)) {
	d := benchData()

	for _, n := range benchDensities {
		b.Run(fmt.Sprintf("bits=%d", n), func(b *testing.B) {
			run(b, n, d.a[n], d.b[n])
		})
	}
}

// #####################################################
//  Set, Clear, Test
// #####################################################

func BenchmarkBitSet256SetClearTest(b *testing.B) {
	d := benchData()

	// Set and Clear have no branches, the cost does not depend on the content.
	b.Run("Set", func(b *testing.B) {
		var s BitSet256

		for i := 0; b.Loop(); i++ {
			s.Set(d.idxs[i&probeMask])
		}

		benchSinkSet = s
	})

	b.Run("Clear", func(b *testing.B) {
		var s BitSet256
		for i := range s {
			s[i] = ^uint64(0)
		}

		for i := 0; b.Loop(); i++ {
			s.Clear(d.idxs[i&probeMask])
		}

		benchSinkSet = s
	})

	b.Run("Test", func(b *testing.B) {
		sets := d.a[128]

		for i := 0; b.Loop(); i++ {
			benchSinkBool = sets[i&setMask].Test(d.idxs[i&probeMask])
		}
	})
}

// #####################################################
//  FirstSet, LastSet, NextSet, AndTop
// #####################################################

func BenchmarkBitSet256FirstSet(b *testing.B) {
	eachDensity(b, func(b *testing.B, _ int, a, _ []BitSet256) {
		b.Run("impl=unrolled", func(b *testing.B) {
			for i := 0; b.Loop(); i++ {
				benchSinkU8, benchSinkBool = a[i&setMask].FirstSet()
			}
		})

		b.Run("impl=loop", func(b *testing.B) {
			for i := 0; b.Loop(); i++ {
				benchSinkU8, benchSinkBool = refFirstSetLoop(&a[i&setMask])
			}
		})
	})
}

func BenchmarkBitSet256LastSet(b *testing.B) {
	eachDensity(b, func(b *testing.B, _ int, a, _ []BitSet256) {
		b.Run("impl=unrolled", func(b *testing.B) {
			for i := 0; b.Loop(); i++ {
				benchSinkU8, benchSinkBool = a[i&setMask].LastSet()
			}
		})

		b.Run("impl=loop", func(b *testing.B) {
			for i := 0; b.Loop(); i++ {
				benchSinkU8, benchSinkBool = refLastSetLoop(&a[i&setMask])
			}
		})
	})
}

func BenchmarkBitSet256NextSet(b *testing.B) {
	idxs := benchData().idxs

	eachDensity(b, func(b *testing.B, _ int, a, _ []BitSet256) {
		for i := 0; b.Loop(); i++ {
			benchSinkU8, benchSinkBool = a[i&setMask].NextSet(idxs[i&probeMask])
		}
	})
}

func BenchmarkBitSet256AndTop(b *testing.B) {
	eachDensity(b, func(b *testing.B, _ int, a, c []BitSet256) {
		b.Run("impl=unrolled", func(b *testing.B) {
			for i := 0; b.Loop(); i++ {
				benchSinkU8, benchSinkBool = a[i&setMask].AndTop(&c[i&setMask])
			}
		})

		b.Run("impl=loop", func(b *testing.B) {
			for i := 0; b.Loop(); i++ {
				benchSinkU8, benchSinkBool = refAndTopLoop(&a[i&setMask], &c[i&setMask])
			}
		})
	})
}

// #####################################################
//  Iteration
// #####################################################

// BenchmarkBitSet256Iterate visits all set bits of a set in four ways. The
// documentation of All recommends AppendBits for dense sets, this benchmark
// shows the difference. ns/bit is the time per visited bit.
func BenchmarkBitSet256Iterate(b *testing.B) {
	eachDensity(b, func(b *testing.B, n int, a, _ []BitSet256) {
		perBit := func(b *testing.B) {
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(n), "ns/bit")
		}

		b.Run("impl=All", func(b *testing.B) {
			for i := 0; b.Loop(); i++ {
				var sum uint8
				for idx := range a[i&setMask].All() {
					sum += idx
				}
				benchSinkU8 = sum
			}

			perBit(b)
		})

		b.Run("impl=AllEnumerate", func(b *testing.B) {
			for i := 0; b.Loop(); i++ {
				var sum uint8
				for pos, idx := range a[i&setMask].AllEnumerate() {
					sum += pos ^ idx
				}
				benchSinkU8 = sum
			}

			perBit(b)
		})

		b.Run("impl=AppendBits", func(b *testing.B) {
			var buf [256]uint8

			for i := 0; b.Loop(); i++ {
				var sum uint8
				for _, idx := range a[i&setMask].AppendBits(buf[:0]) {
					sum += idx
				}
				benchSinkU8 = sum
			}

			perBit(b)
		})

		b.Run("impl=Bits", func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; b.Loop(); i++ {
				var sum uint8
				for _, idx := range a[i&setMask].Bits() {
					sum += idx
				}
				benchSinkU8 = sum
			}

			perBit(b)
		})
	})
}

// BenchmarkBitSet256AppendBits measures the conversion alone, without
// visiting the result, into a buffer on the stack.
func BenchmarkBitSet256AppendBits(b *testing.B) {
	eachDensity(b, func(b *testing.B, n int, a, _ []BitSet256) {
		var buf [256]uint8

		for i := 0; b.Loop(); i++ {
			benchSinkInt = len(a[i&setMask].AppendBits(buf[:0]))
		}

		b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(n), "ns/bit")
	})
}

// #####################################################
//  Word-wise operations
// #####################################################

// BenchmarkBitSet256WordOps measures the operations that work on the four
// words without branches. The cost does not depend on the density.
func BenchmarkBitSet256WordOps(b *testing.B) {
	d := benchData()
	a, c := d.a[128], d.b[128]

	b.Run("And", func(b *testing.B) {
		for i := 0; b.Loop(); i++ {
			benchSinkSet = a[i&setMask].And(&c[i&setMask])
		}
	})

	b.Run("Or", func(b *testing.B) {
		for i := 0; b.Loop(); i++ {
			benchSinkSet = a[i&setMask].Or(&c[i&setMask])
		}
	})

	b.Run("Xor", func(b *testing.B) {
		for i := 0; b.Loop(); i++ {
			benchSinkSet = a[i&setMask].Xor(&c[i&setMask])
		}
	})

	b.Run("AlignedPairs", func(b *testing.B) {
		for i := 0; b.Loop(); i++ {
			benchSinkSet = a[i&setMask].AlignedPairs()
		}
	})

	b.Run("Overlaps", func(b *testing.B) {
		for i := 0; b.Loop(); i++ {
			benchSinkBool = a[i&setMask].Overlaps(&c[i&setMask])
		}
	})

	b.Run("IsEmpty", func(b *testing.B) {
		for i := 0; b.Loop(); i++ {
			benchSinkBool = a[i&setMask].IsEmpty()
		}
	})

	b.Run("OnesCount", func(b *testing.B) {
		for i := 0; b.Loop(); i++ {
			benchSinkInt = a[i&setMask].OnesCount()
		}
	})
}
