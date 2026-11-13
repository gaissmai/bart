// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package sparse

import (
	"maps"
	"math/bits"
	"math/rand/v2"
	"slices"
	"testing"
)

// #####################################################
//  helpers
// #####################################################

// valueFor derives the payload from the index, so that the index-to-item
// mapping can be verified without a second table.
func valueFor(i uint8) int { return int(i) + 1000 }

// seq returns the indices from (inclusive) to to (exclusive).
func seq(from, to int) []uint8 {
	s := make([]uint8, 0, to-from)
	for i := from; i < to; i++ {
		s = append(s, uint8(i))
	}
	return s
}

// stride returns every step-th index in [0..255], starting at 0.
func stride(step int) []uint8 {
	var s []uint8
	for i := 0; i < 256; i += step {
		s = append(s, uint8(i))
	}
	return s
}

// reversed returns a reversed copy of s.
func reversed(s []uint8) []uint8 {
	r := slices.Clone(s)
	slices.Reverse(r)
	return r
}

// newArray builds an Array256 by inserting the indices in the given order.
func newArray(idxs ...uint8) *Array256[int] {
	a := new(Array256[int])
	for _, i := range idxs {
		a.InsertAt(i, valueFor(i))
	}
	return a
}

// modelOf is the reference model for newArray(idxs...).
func modelOf(idxs ...uint8) map[uint8]int {
	m := make(map[uint8]int, len(idxs))
	for _, i := range idxs {
		m[i] = valueFor(i)
	}
	return m
}

func firstAbsent(m map[uint8]int) (uint8, bool) {
	for i := range 256 {
		if _, ok := m[uint8(i)]; !ok {
			return uint8(i), true
		}
	}
	return 0, false
}

func firstPresent(m map[uint8]int) (uint8, bool) {
	for i := range 256 {
		if _, ok := m[uint8(i)]; ok {
			return uint8(i), true
		}
	}
	return 0, false
}

// checkInvariants verifies the coupling between bitset, rank cache and items.
func checkInvariants[T any](t *testing.T, a *Array256[T]) {
	t.Helper()

	if got, want := a.Len(), a.OnesCount(); got != want {
		t.Fatalf("Len() = %d, want OnesCount() = %d", got, want)
	}

	var wantCum [4]uint8
	sum := 0
	for w := range 4 {
		wantCum[w] = uint8(sum)
		sum += bits.OnesCount64(a.BitSet256[w])
	}

	if a.cum != wantCum {
		t.Fatalf("cum = %v, want %v", a.cum, wantCum)
	}
}

// assertMatches verifies the Array256 against the reference model:
// invariants, item order, Get for all 256 indices.
func assertMatches(t *testing.T, a *Array256[int], want map[uint8]int) {
	t.Helper()

	checkInvariants(t, a)

	idxs := make([]uint8, 0, len(want))
	for i := range want {
		idxs = append(idxs, i)
	}
	slices.Sort(idxs)

	if a.Len() != len(idxs) {
		t.Fatalf("Len() = %d, want %d", a.Len(), len(idxs))
	}

	for pos, i := range idxs {
		if a.items[pos] != want[i] {
			t.Fatalf("items[%d] = %d, want %d (index %d)", pos, a.items[pos], want[i], i)
		}
	}

	for i := range 256 {
		idx := uint8(i)
		wantVal, wantOK := want[idx]

		gotVal, gotOK := a.Get(idx)
		if gotOK != wantOK || gotVal != wantVal {
			t.Fatalf("Get(%d) = (%d, %v), want (%d, %v)", idx, gotVal, gotOK, wantVal, wantOK)
		}
	}
}

// indexSets are the shared inputs for the read-only tests.
var indexSets = []struct {
	name string
	idxs []uint8
}{
	{"empty", nil},
	{"single first", []uint8{0}},
	{"single last", []uint8{255}},
	{"one per word", []uint8{10, 70, 130, 200}},
	{"first of each word", []uint8{0, 64, 128, 192}},
	{"last of each word", []uint8{63, 127, 191, 255}},
	{"word boundaries", []uint8{0, 63, 64, 127, 128, 191, 192, 255}},
	{"unsorted insertion order", []uint8{200, 3, 129, 64, 63, 255, 0, 128}},
	{"dense first word", seq(0, 64)},
	{"dense second word", seq(64, 128)},
	{"dense last word", seq(192, 256)},
	{"every third bit", stride(3)},
	{"full", seq(0, 256)},
}

// #####################################################
//  tests
// #####################################################

func TestRank0(t *testing.T) {
	t.Parallel()

	for _, tt := range indexSets {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := newArray(tt.idxs...)
			checkInvariants(t, a)

			sorted := slices.Clone(tt.idxs)
			slices.Sort(sorted)
			sorted = slices.Compact(sorted)

			for i := range 256 {
				idx := uint8(i)

				// for a set bit this is its position in items,
				// for a clear bit it is the insertion position
				want, _ := slices.BinarySearch(sorted, idx)

				if got := a.rank0(idx); got != want {
					t.Errorf("rank0(%d) = %d, want %d", idx, got, want)
				}
			}
		})
	}
}

// TestMustGetFirst verifies that MustGetFirst correctly retrieves the value
// associated with the lowest present index (the first element in items).
func TestMustGetFirst(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		idxs []uint8
	}{
		{"single element at zero", []uint8{0}},
		{"single element non-zero", []uint8{42}},
		{"multiple sorted indices", []uint8{5, 10, 20}},
		{"multiple unsorted indices", []uint8{50, 10, 5, 20}},
		{"across word boundaries", []uint8{64, 128, 192}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := newArray(tt.idxs...)

			// Determine the minimum index which corresponds to items[0]
			minIdx := tt.idxs[0]
			for _, idx := range tt.idxs {
				if idx < minIdx {
					minIdx = idx
				}
			}

			wantVal := valueFor(minIdx)
			if gotVal := a.MustGetFirst(); gotVal != wantVal {
				t.Errorf("MustGetFirst() = %v, want %v", gotVal, wantVal)
			}
		})
	}
}

func TestGetMustGet(t *testing.T) {
	t.Parallel()

	for _, tt := range indexSets {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := newArray(tt.idxs...)
			want := modelOf(tt.idxs...)

			for i := range 256 {
				idx := uint8(i)
				wantVal, wantOK := want[idx]

				gotVal, gotOK := a.Get(idx)
				if gotOK != wantOK || gotVal != wantVal {
					t.Errorf("Get(%d) = (%d, %v), want (%d, %v)", idx, gotVal, gotOK, wantVal, wantOK)
				}

				if wantOK {
					if got := a.MustGet(idx); got != wantVal {
						t.Errorf("MustGet(%d) = %d, want %d", idx, got, wantVal)
					}
				}
			}
		})
	}
}

func TestInsertAt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		idxs []uint8 // insertion order, duplicates overwrite
	}{
		{"ascending", []uint8{1, 2, 3, 4}},
		{"descending", []uint8{200, 100, 50, 0}},
		{"duplicates overwrite", []uint8{5, 5, 5}},
		{"overwrite in the middle", []uint8{10, 20, 30, 20}},
		{"across word boundaries", []uint8{255, 0, 64, 63, 128, 127, 192, 191}},
		{"interleaved words", []uint8{0, 64, 1, 65, 128, 129, 192, 193}},
		{"fill first word backwards", reversed(seq(0, 64))},
		{"fill everything ascending", seq(0, 256)},
		{"fill everything descending", reversed(seq(0, 256))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := new(Array256[int])
			want := make(map[uint8]int)

			for step, i := range tt.idxs {
				val := valueFor(i) + step // distinct per step, overwrites become visible

				_, wantExists := want[i]
				want[i] = val

				if got := a.InsertAt(i, val); got != wantExists {
					t.Fatalf("step %d: InsertAt(%d) exists = %v, want %v", step, i, got, wantExists)
				}

				assertMatches(t, a, want)
			}
		})
	}
}

func TestDeleteAt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		initial []uint8
		deletes []uint8
	}{
		{"absent from empty", nil, []uint8{7}},
		{"absent from non-empty", []uint8{1, 2, 3}, []uint8{0, 4, 255}},
		{"delete first", []uint8{10, 20, 30}, []uint8{10}},
		{"delete middle", []uint8{10, 20, 30}, []uint8{20}},
		{"delete last", []uint8{10, 20, 30}, []uint8{30}},
		{"delete twice", []uint8{5, 6}, []uint8{5, 5}},
		{"delete all ascending", []uint8{1, 2, 3}, []uint8{1, 2, 3}},
		{"delete all descending", []uint8{1, 2, 3}, []uint8{3, 2, 1}},
		{
			"word boundaries",
			[]uint8{0, 63, 64, 127, 128, 191, 192, 255},
			[]uint8{64, 255, 0, 191, 127, 63},
		},
		{"drain full ascending", seq(0, 256), seq(0, 256)},
		{"drain full descending", seq(0, 256), reversed(seq(0, 256))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := newArray(tt.initial...)
			want := modelOf(tt.initial...)
			assertMatches(t, a, want) // sanity check of the starting point

			for step, i := range tt.deletes {
				wantVal, wantExists := want[i]
				delete(want, i)

				gotVal, gotExists := a.DeleteAt(i)
				if gotExists != wantExists || gotVal != wantVal {
					t.Fatalf("step %d: DeleteAt(%d) = (%d, %v), want (%d, %v)",
						step, i, gotVal, gotExists, wantVal, wantExists)
				}

				assertMatches(t, a, want)
			}
		})
	}
}

// TestClone verifies the behavior of Clone with and without a custom clone function,
// ensuring nil safety, proper shallow/deep copying semantics, and isolation from mutations.
func TestClone(t *testing.T) {
	t.Parallel()

	t.Run("nil receiver", func(t *testing.T) {
		t.Parallel()
		var a *Array256[int]
		if got := a.Clone(nil); got != nil {
			t.Errorf("Clone(nil) of nil = %v, want nil", got)
		}
		if got := a.Clone(func(v int) int { return v * 2 }); got != nil {
			t.Errorf("Clone(cloneFn) of nil = %v, want nil", got)
		}
	})

	t.Run("nil cloneFn acts as copy", func(t *testing.T) {
		t.Parallel()
		for _, tt := range indexSets {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				orig := newArray(tt.idxs...)
				origWant := modelOf(tt.idxs...)

				cp := orig.Clone(nil)
				assertMatches(t, cp, origWant)

				// mutate the copy only
				cpWant := modelOf(tt.idxs...)

				if extra, ok := firstAbsent(origWant); ok {
					cp.InsertAt(extra, valueFor(extra))
					cpWant[extra] = valueFor(extra)
				}

				if first, ok := firstPresent(origWant); ok {
					cp.DeleteAt(first)
					delete(cpWant, first)
				}

				assertMatches(t, cp, cpWant)
				assertMatches(t, orig, origWant) // the original must be untouched
			})
		}
	})

	t.Run("with cloneFn transformation", func(t *testing.T) {
		t.Parallel()
		for _, tt := range indexSets {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				orig := newArray(tt.idxs...)
				origWant := modelOf(tt.idxs...)

				// cloneFn scales each value by 2 to simulate a deep copy/transformation
				cloneFn := func(v int) int {
					return v * 2
				}

				cp := orig.Clone(cloneFn)

				cloneWant := make(map[uint8]int, len(origWant))
				for k, v := range origWant {
					cloneWant[k] = cloneFn(v)
				}

				assertMatches(t, cp, cloneWant)
				assertMatches(t, orig, origWant) // original remains completely unaffected
			})
		}
	})
}

// TestAll verifies the iter.Seq2 iterator returned by All, ensuring all index-value
// pairs are correctly yielded in ascending order and early loop termination is supported.
func TestAll(t *testing.T) {
	t.Parallel()

	for _, tt := range indexSets {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := newArray(tt.idxs...)
			want := modelOf(tt.idxs...)

			got := maps.Collect(a.All())
			if !maps.Equal(got, want) {
				t.Fatalf("All() result isn't equal to want")
			}
		})
	}

	t.Run("early termination", func(t *testing.T) {
		t.Parallel()
		a := newArray(10, 20, 30, 40, 50)
		count := 0
		for range a.All() {
			count++
			break // terminate early after the first yielded element
		}
		if count != 1 {
			t.Errorf("expected 1 iteration after break, got %d", count)
		}
	})
}

// TestAllPtr verifies the iter.Seq2 iterator returned by AllPtr, ensuring all index-value
// pointer pairs are correctly yielded in ascending order, allows in-place mutation,
// and supports early loop termination.
func TestAllPtr(t *testing.T) {
	t.Parallel()

	for _, tt := range indexSets {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := newArray(tt.idxs...)
			want := modelOf(tt.idxs...)

			// Test reading and in-place mutation via AllPtr
			for idx, valPtr := range a.AllPtr() {
				wantVal, ok := want[idx]
				if !ok || *valPtr != wantVal {
					t.Errorf("AllPtr() yielded unexpected value at %d: got %v, want %v", idx, *valPtr, wantVal)
				}
				// Mutate the slot in-place by doubling the value
				*valPtr *= 2
			}

			// Verify that the mutations were correctly applied in the sparse array
			for idx, wantVal := range want {
				gotVal, ok := a.Get(idx)
				if !ok || gotVal != wantVal*2 {
					t.Errorf("After AllPtr mutation, Get(%d) = %v, want %v", idx, gotVal, wantVal*2)
				}
			}
		})
	}

	t.Run("early termination", func(t *testing.T) {
		t.Parallel()
		a := newArray(10, 20, 30, 40, 50)
		count := 0
		for range a.AllPtr() {
			count++
			break // terminate early after the first yielded element
		}
		if count != 1 {
			t.Errorf("expected 1 iteration after break, got %d", count)
		}
	})
}

func TestForbiddenMutators(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   func(*Array256[int])
	}{
		{"Set", func(a *Array256[int]) { a.Set(1) }},
		{"Clear", func(a *Array256[int]) { a.Clear(1) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := newArray(1, 2, 3)

			func() {
				defer func() {
					if recover() == nil {
						t.Errorf("%s must panic", tt.name)
					}
				}()
				tt.fn(a)
			}()

			// the state must be unchanged after the panic
			assertMatches(t, a, modelOf(1, 2, 3))
		})
	}
}

// TestRandomOps is a differential test against a map as reference model.
func TestRandomOps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		seed      uint64
		ops       int
		span      int // indices are drawn from [0..span)
		insertPct int // probability of an insert in percent, otherwise delete
	}{
		{"balanced full range", 1, 5000, 256, 50},
		{"insert heavy full range", 2, 5000, 256, 85},
		{"delete heavy full range", 3, 5000, 256, 15},
		{"single word", 4, 3000, 64, 50},
		{"crossing the first word boundary", 5, 3000, 130, 60},
		{"tiny range, many collisions", 6, 3000, 8, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			prng := rand.New(rand.NewPCG(tt.seed, tt.seed))

			a := new(Array256[int])
			want := make(map[uint8]int)

			for op := range tt.ops {
				i := uint8(prng.IntN(tt.span))

				if prng.IntN(100) < tt.insertPct {
					val := prng.IntN(1 << 20)
					_, wantExists := want[i]
					want[i] = val

					if got := a.InsertAt(i, val); got != wantExists {
						t.Fatalf("op %d: InsertAt(%d) exists = %v, want %v", op, i, got, wantExists)
					}
				} else {
					wantVal, wantExists := want[i]
					delete(want, i)

					gotVal, gotExists := a.DeleteAt(i)
					if gotExists != wantExists || gotVal != wantVal {
						t.Fatalf("op %d: DeleteAt(%d) = (%d, %v), want (%d, %v)",
							op, i, gotVal, gotExists, wantVal, wantExists)
					}
				}

				// cheap check after every operation, full comparison now and then
				checkInvariants(t, a)
				if op%97 == 0 {
					assertMatches(t, a, want)
				}
			}

			assertMatches(t, a, want)
		})
	}
}
