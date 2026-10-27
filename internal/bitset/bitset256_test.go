// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bitset

import (
	"fmt"
	"math/bits"
	"math/rand/v2"
	"slices"
	"sync"
	"testing"
)

// #####################################################
//  The tests in this file compare BitSet256 with a simple reference model,
//  one bool per index. The model is independent of the bit tricks of the
//  implementation and serves as the oracle.
// #####################################################

// refModel is the reference implementation of a set of [0..255].
type refModel [256]bool

func refModelOf(idxs ...uint8) refModel {
	var m refModel
	for _, i := range idxs {
		m[i] = true
	}
	return m
}

// bitset builds the BitSet256 from the model by writing the words directly,
// independent of BitSet256.Set, which is tested against it.
func (m refModel) bitset() BitSet256 {
	var b BitSet256
	for i, ok := range m {
		if ok {
			b[i>>6] |= 1 << (i & 63)
		}
	}
	return b
}

// idxs returns the indices in the set in ascending order.
func (m refModel) idxs() []uint8 {
	var s []uint8
	for i, ok := range m {
		if ok {
			s = append(s, uint8(i))
		}
	}
	return s
}

// refSeq returns the indices from (inclusive) to to (exclusive).
func refSeq(from, to int) []uint8 {
	s := make([]uint8, 0, to-from)
	for i := from; i < to; i++ {
		s = append(s, uint8(i))
	}
	return s
}

// refStride returns every step-th index in [0..255], beginning with start.
func refStride(start, step int) []uint8 {
	var s []uint8
	for i := start; i < 256; i += step {
		s = append(s, uint8(i))
	}
	return s
}

// refRandomModel returns a model with exactly nBits random bits set.
func refRandomModel(prng *rand.Rand, nBits int) refModel {
	var m refModel
	for _, i := range prng.Perm(256)[:nBits] {
		m[i] = true
	}
	return m
}

// refIndexSets are structured inputs for the tests, with special attention
// to the borders of the four 64-bit words.
var refIndexSets = []struct {
	name string
	idxs []uint8
}{
	{"empty", nil},
	{"bit 0", []uint8{0}},
	{"bit 63", []uint8{63}},
	{"bit 64", []uint8{64}},
	{"bit 127", []uint8{127}},
	{"bit 128", []uint8{128}},
	{"bit 191", []uint8{191}},
	{"bit 192", []uint8{192}},
	{"bit 255", []uint8{255}},
	{"first of each word", []uint8{0, 64, 128, 192}},
	{"last of each word", []uint8{63, 127, 191, 255}},
	{"word boundaries", []uint8{0, 63, 64, 127, 128, 191, 192, 255}},
	{"unsorted input order", []uint8{200, 3, 129, 64, 63, 255, 0, 128}},
	{"dense first word", refSeq(0, 64)},
	{"dense second word", refSeq(64, 128)},
	{"dense last word", refSeq(192, 256)},
	{"even bits", refStride(0, 2)},
	{"odd bits", refStride(1, 2)},
	{"aligned pairs 2,3 6,7 10,11 ...", slices.Concat(refStride(2, 4), refStride(3, 4))},
	{"every third bit", refStride(0, 3)},
	{"full", refSeq(0, 256)},
}

type refCase struct {
	name string
	m    refModel
}

// refAllCases are the structured inputs plus random sets of different density.
var refAllCases = sync.OnceValue(func() []refCase {
	var cases []refCase
	for _, s := range refIndexSets {
		cases = append(cases, refCase{s.name, refModelOf(s.idxs...)})
	}

	prng := rand.New(rand.NewPCG(42, 42))
	for _, n := range []int{1, 2, 5, 20, 64, 100, 128, 200, 254, 255} {
		for k := range 2 {
			name := fmt.Sprintf("random %d bits #%d", n, k)
			cases = append(cases, refCase{name, refRandomModel(prng, n)})
		}
	}

	return cases
})

// The loop based variants of FirstSet, LastSet and AndTop. The implementations
// in the package unroll the loops, the benchmarks compare both variants,
// TestRefLoopReferences verifies that they are equivalent.

func refFirstSetLoop(b *BitSet256) (first uint8, ok bool) {
	for wIdx, w := range b {
		if w != 0 {
			return uint8(wIdx<<6 + bits.TrailingZeros64(w)), true
		}
	}
	return 0, false
}

func refLastSetLoop(b *BitSet256) (last uint8, ok bool) {
	for wIdx := 3; wIdx >= 0; wIdx-- {
		if w := b[wIdx]; w != 0 {
			return uint8(wIdx<<6 + bits.Len64(w) - 1), true
		}
	}
	return 0, false
}

func refAndTopLoop(b, c *BitSet256) (top uint8, ok bool) {
	for wIdx := 3; wIdx >= 0; wIdx-- {
		if w := b[wIdx] & c[wIdx]; w != 0 {
			return uint8(wIdx<<6 + bits.Len64(w) - 1), true
		}
	}
	return 0, false
}

// #####################################################
//  Set, Clear, Test
// #####################################################

func TestRefZeroValue(t *testing.T) {
	t.Parallel()

	var b BitSet256

	if !b.IsEmpty() {
		t.Error("zero value: IsEmpty() = false, want true")
	}

	if got := b.OnesCount(); got != 0 {
		t.Errorf("zero value: OnesCount() = %d, want 0", got)
	}

	for i := range 256 {
		if b.Test(uint8(i)) {
			t.Fatalf("zero value: Test(%d) = true, want false", i)
		}
	}
}

func TestRefSetClearTest(t *testing.T) {
	t.Parallel()

	for _, tt := range refIndexSets {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			want := refModelOf(tt.idxs...)

			// Set in the given order
			var b BitSet256
			for _, i := range tt.idxs {
				b.Set(i)
			}

			if exp := want.bitset(); b != exp {
				t.Fatalf("after Set: got %#x, want %#x", b, exp)
			}

			for i := range 256 {
				if got := b.Test(uint8(i)); got != want[i] {
					t.Fatalf("Test(%d) = %v, want %v", i, got, want[i])
				}
			}

			// Set is idempotent
			again := b
			for _, i := range tt.idxs {
				again.Set(i)
			}

			if again != b {
				t.Fatalf("second Set changed the set: got %#x, want %#x", again, b)
			}

			// Clear of a bit that is not set changes nothing
			for i := range 256 {
				if want[i] {
					continue
				}

				c := b
				c.Clear(uint8(i))

				if c != b {
					t.Fatalf("Clear(%d) of an unset bit changed the set", i)
				}
			}

			// Clear the bits one by one in reverse order
			for _, idx := range slices.Backward(tt.idxs) {
				b.Clear(idx)
				want[idx] = false

				if exp := want.bitset(); b != exp {
					t.Fatalf("after Clear(%d): got %#x, want %#x", idx, b, exp)
				}
			}

			if b != (BitSet256{}) {
				t.Fatalf("after clearing all bits: got %#x, want the empty set", b)
			}
		})
	}
}

func TestRefOnesCountIsEmpty(t *testing.T) {
	t.Parallel()

	for _, tc := range refAllCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b := tc.m.bitset()
			want := len(tc.m.idxs())

			if got := b.OnesCount(); got != want {
				t.Errorf("OnesCount() = %d, want %d", got, want)
			}

			if got := b.IsEmpty(); got != (want == 0) {
				t.Errorf("IsEmpty() = %v, want %v", got, want == 0)
			}
		})
	}
}

// #####################################################
//  FirstSet, LastSet, NextSet
// #####################################################

func TestRefFirstLastNextSet(t *testing.T) {
	t.Parallel()

	for _, tc := range refAllCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b := tc.m.bitset()
			idxs := tc.m.idxs()

			first, ok := b.FirstSet()
			if ok != (len(idxs) > 0) {
				t.Fatalf("FirstSet() ok = %v, want %v", ok, len(idxs) > 0)
			}

			if ok && first != idxs[0] {
				t.Errorf("FirstSet() = %d, want %d", first, idxs[0])
			}

			last, ok := b.LastSet()
			if ok != (len(idxs) > 0) {
				t.Fatalf("LastSet() ok = %v, want %v", ok, len(idxs) > 0)
			}

			if ok && last != idxs[len(idxs)-1] {
				t.Errorf("LastSet() = %d, want %d", last, idxs[len(idxs)-1])
			}

			// documented: for an empty set last is 0
			if !ok && last != 0 {
				t.Errorf("LastSet() of the empty set = %d, want 0", last)
			}

			// NextSet from every possible start, including 255
			for start := range 256 {
				var want uint8
				var wantOK bool

				for _, i := range idxs {
					if int(i) >= start {
						want, wantOK = i, true
						break
					}
				}

				next, ok := b.NextSet(uint8(start))
				if ok != wantOK {
					t.Fatalf("NextSet(%d) ok = %v, want %v", start, ok, wantOK)
				}

				if ok && next != want {
					t.Fatalf("NextSet(%d) = %d, want %d", start, next, want)
				}
			}
		})
	}
}

// #####################################################
//  All, AllEnumerate, AppendBits, Bits
// #####################################################

func TestRefIterators(t *testing.T) {
	t.Parallel()

	for _, tc := range refAllCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b := tc.m.bitset()
			want := tc.m.idxs()

			if got := slices.Collect(b.All()); !slices.Equal(got, want) {
				t.Fatalf("All() = %v, want %v", got, want)
			}

			pos := 0
			for i, idx := range b.AllEnumerate() {
				if pos >= len(want) {
					t.Fatalf("AllEnumerate() yields more than the %d elements", len(want))
				}

				if int(i) != pos || idx != want[pos] {
					t.Fatalf("AllEnumerate() #%d = (%d, %d), want (%d, %d)", pos, i, idx, pos, want[pos])
				}

				pos++
			}

			if pos != len(want) {
				t.Fatalf("AllEnumerate() yields %d elements, want %d", pos, len(want))
			}
		})
	}
}

// TestRefIteratorsEarlyExit verifies that the iterators stop immediately
// when yield returns false and never call yield again.
func TestRefIteratorsEarlyExit(t *testing.T) {
	t.Parallel()

	for _, tc := range refAllCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b := tc.m.bitset()
			n := len(tc.m.idxs())

			for k := 1; k <= n; k++ {
				calls := 0
				b.All()(func(uint8) bool {
					calls++
					return calls < k
				})

				if calls != k {
					t.Fatalf("All() stopping after %d: yield called %d times", k, calls)
				}

				calls = 0
				b.AllEnumerate()(func(uint8, uint8) bool {
					calls++
					return calls < k
				})

				if calls != k {
					t.Fatalf("AllEnumerate() stopping after %d: yield called %d times", k, calls)
				}
			}
		})
	}
}

func TestRefAppendBitsAndBits(t *testing.T) {
	t.Parallel()

	for _, tc := range refAllCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b := tc.m.bitset()
			want := tc.m.idxs()

			if got := b.AppendBits(nil); !slices.Equal(got, want) {
				t.Fatalf("AppendBits(nil) = %v, want %v", got, want)
			}

			// existing content of buf is kept, the indices are appended
			prefix := []uint8{9, 9, 9}

			got := b.AppendBits(slices.Clone(prefix))
			if !slices.Equal(got[:len(prefix)], prefix) || !slices.Equal(got[len(prefix):], want) {
				t.Fatalf("AppendBits(prefix) = %v, want %v followed by %v", got, prefix, want)
			}

			// Bits returns the indices in a slice with exactly the needed capacity
			bs := b.Bits()
			if !slices.Equal(bs, want) {
				t.Fatalf("Bits() = %v, want %v", bs, want)
			}

			if cap(bs) != len(want) {
				t.Errorf("Bits(): cap = %d, want %d", cap(bs), len(want))
			}
		})
	}
}

// TestRefAppendBitsAllocs verifies the documented allocation behavior.
// It must not run in parallel, testing.AllocsPerRun requires it.
func TestRefAppendBitsAllocs(t *testing.T) {
	sets := []struct {
		name string
		m    refModel
	}{
		{"sparse", refModelOf(3, 70, 130, 250)},
		{"dense", refModelOf(refSeq(0, 256)...)},
	}

	for _, tt := range sets {
		t.Run(tt.name, func(t *testing.T) {
			b := tt.m.bitset()
			buf := make([]uint8, 0, 256)

			allocs := testing.AllocsPerRun(100, func() {
				buf = b.AppendBits(buf[:0])
			})

			if allocs != 0 {
				t.Errorf("AppendBits with sufficient capacity: %v allocs, want 0", allocs)
			}

			if want := tt.m.idxs(); !slices.Equal(buf, want) {
				t.Errorf("AppendBits() = %v, want %v", buf, want)
			}
		})
	}
}

// #####################################################
//  AndTop, AlignedPairs
// #####################################################

func TestRefAndTop(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		a, b   []uint8
		want   uint8
		wantOK bool
	}{
		{"both empty", nil, nil, 0, false},
		{"disjoint", []uint8{1, 2, 3}, []uint8{4, 5, 6}, 0, false},
		{"single common bit", []uint8{1, 200}, []uint8{200}, 200, true},
		{"highest of several", []uint8{3, 70, 130, 250}, []uint8{3, 70, 130}, 130, true},
		{"only in word 0", []uint8{5}, []uint8{5, 100}, 5, true},
		{"63 and 64, only 63 common", []uint8{63, 64}, []uint8{63}, 63, true},
		{"63 and 64, both common", []uint8{63, 64}, []uint8{63, 64}, 64, true},
		{"top bit 255", []uint8{255}, []uint8{255}, 255, true},
		{"bit 0 is a match, not a miss", []uint8{0}, []uint8{0}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := refModelOf(tt.a...).bitset()
			b := refModelOf(tt.b...).bitset()

			// AndTop is symmetric
			for _, order := range []string{"a,b", "b,a"} {
				x, y := &a, &b
				if order == "b,a" {
					x, y = y, x
				}

				top, ok := x.AndTop(y)
				if top != tt.want || ok != tt.wantOK {
					t.Errorf("AndTop(%s) = (%d, %v), want (%d, %v)", order, top, ok, tt.want, tt.wantOK)
				}
			}
		})
	}

	// all pairs of the structured and random sets against the model
	t.Run("all pairs", func(t *testing.T) {
		t.Parallel()

		for _, x := range refAllCases() {
			for _, y := range refAllCases() {
				a, b := x.m.bitset(), y.m.bitset()

				var want uint8
				var wantOK bool

				for i := 255; i >= 0; i-- {
					if x.m[i] && y.m[i] {
						want, wantOK = uint8(i), true
						break
					}
				}

				top, ok := a.AndTop(&b)
				if top != want || ok != wantOK {
					t.Fatalf("%q.AndTop(%q) = (%d, %v), want (%d, %v)", x.name, y.name, top, ok, want, wantOK)
				}
			}
		}
	})
}

func TestRefAlignedPairs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   []uint8
		want []uint8
	}{
		{"empty", nil, nil},
		{"pair 0,1", []uint8{0, 1}, []uint8{0}},
		{"odd start 1,2 is no aligned pair", []uint8{1, 2}, nil},
		{"pair at the end of word 0: 62,63", []uint8{62, 63}, []uint8{62}},
		{"63,64 crosses the word boundary", []uint8{63, 64}, nil},
		{"pair at the start of word 1: 64,65", []uint8{64, 65}, []uint8{64}},
		{"last pair 254,255", []uint8{254, 255}, []uint8{254}},
		{"four in a row 0..3", []uint8{0, 1, 2, 3}, []uint8{0, 2}},
		{"pair plus a loner", []uint8{2, 3, 5}, []uint8{2}},
		{"single bit", []uint8{7}, nil},
		{"full set gives all even bits", refSeq(0, 256), refStride(0, 2)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			in := refModelOf(tt.in...).bitset()
			want := refModelOf(tt.want...).bitset()

			if got := in.AlignedPairs(); got != want {
				t.Errorf("AlignedPairs() = %#x, want %#x", got, want)
			}
		})
	}

	// all structured and random sets against the model
	t.Run("all cases", func(t *testing.T) {
		t.Parallel()

		for _, tc := range refAllCases() {
			in := tc.m.bitset()

			var want refModel
			for n := 0; n < 256; n += 2 {
				want[n] = tc.m[n] && tc.m[n+1]
			}

			if got := in.AlignedPairs(); got != want.bitset() {
				t.Fatalf("%q.AlignedPairs() = %#x, want %#x", tc.name, got, want.bitset())
			}
		}
	})
}

// #####################################################
//  And, Or, Xor, Overlaps
// #####################################################

func TestRefBinaryOps(t *testing.T) {
	t.Parallel()

	for _, x := range refAllCases() {
		for _, y := range refAllCases() {
			a, b := x.m.bitset(), y.m.bitset()

			var wantAnd, wantOr, wantXor refModel
			var wantOverlaps bool

			for i := range 256 {
				wantAnd[i] = x.m[i] && y.m[i]
				wantOr[i] = x.m[i] || y.m[i]
				wantXor[i] = x.m[i] != y.m[i]
				wantOverlaps = wantOverlaps || wantAnd[i]
			}

			if got := a.And(&b); got != wantAnd.bitset() {
				t.Fatalf("%q.And(%q) = %#x, want %#x", x.name, y.name, got, wantAnd.bitset())
			}

			if got := a.Or(&b); got != wantOr.bitset() {
				t.Fatalf("%q.Or(%q) = %#x, want %#x", x.name, y.name, got, wantOr.bitset())
			}

			if got := a.Xor(&b); got != wantXor.bitset() {
				t.Fatalf("%q.Xor(%q) = %#x, want %#x", x.name, y.name, got, wantXor.bitset())
			}

			if got := a.Overlaps(&b); got != wantOverlaps {
				t.Fatalf("%q.Overlaps(%q) = %v, want %v", x.name, y.name, got, wantOverlaps)
			}
		}
	}
}

// TestRefReceiverUnchanged verifies that the binary operations leave both
// the receiver and the argument unchanged.
func TestRefReceiverUnchanged(t *testing.T) {
	t.Parallel()

	for _, x := range refAllCases() {
		for _, y := range refAllCases() {
			a, b := x.m.bitset(), y.m.bitset()
			aBefore, bBefore := a, b

			_ = a.And(&b)
			_ = a.Or(&b)
			_ = a.Xor(&b)
			_ = a.Overlaps(&b)
			_, _ = a.AndTop(&b)
			_ = a.AlignedPairs()

			if a != aBefore || b != bBefore {
				t.Fatalf("operations on %q and %q changed the operands", x.name, y.name)
			}
		}
	}
}

// TestRefAlgebra verifies algebraic properties with random sets.
func TestRefAlgebra(t *testing.T) {
	t.Parallel()

	prng := rand.New(rand.NewPCG(7, 7))
	empty := BitSet256{}

	for range 2000 {
		a := refRandomModel(prng, prng.IntN(257)).bitset()
		b := refRandomModel(prng, prng.IntN(257)).bitset()

		and, or, xor := a.And(&b), a.Or(&b), a.Xor(&b)

		// commutative
		if ba := b.And(&a); and != ba {
			t.Fatalf("And is not commutative for %#x and %#x", a, b)
		}

		if ba := b.Or(&a); or != ba {
			t.Fatalf("Or is not commutative for %#x and %#x", a, b)
		}

		if ba := b.Xor(&a); xor != ba {
			t.Fatalf("Xor is not commutative for %#x and %#x", a, b)
		}

		// idempotent, neutral and self-inverse elements
		if aa := a.And(&a); aa != a {
			t.Fatalf("a AND a != a for %#x", a)
		}

		if ae := a.Or(&empty); ae != a {
			t.Fatalf("a OR empty != a for %#x", a)
		}

		if aa := a.Xor(&a); aa != empty {
			t.Fatalf("a XOR a != empty for %#x", a)
		}

		// a XOR b == (a OR b) XOR (a AND b)
		if x := or.Xor(&and); x != xor {
			t.Fatalf("a XOR b != (a OR b) XOR (a AND b) for %#x and %#x", a, b)
		}

		// Overlaps is the negation of an empty intersection
		if got := a.Overlaps(&b); got == and.IsEmpty() {
			t.Fatalf("Overlaps() = %v, but IsEmpty(a AND b) = %v", got, and.IsEmpty())
		}

		// inclusion-exclusion for the population counts
		na, nb, nAnd := a.OnesCount(), b.OnesCount(), and.OnesCount()

		if got := or.OnesCount(); got != na+nb-nAnd {
			t.Fatalf("OnesCount(a OR b) = %d, want %d", got, na+nb-nAnd)
		}

		if got := xor.OnesCount(); got != na+nb-2*nAnd {
			t.Fatalf("OnesCount(a XOR b) = %d, want %d", got, na+nb-2*nAnd)
		}
	}
}

// #####################################################
//  differential test
// #####################################################

// TestRefRandomOps applies random Set and Clear operations to BitSet256 and
// the reference model and compares them.
func TestRefRandomOps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		seed   uint64
		ops    int
		span   int // indices are drawn from [0..span)
		setPct int // probability of a Set in percent, otherwise Clear
	}{
		{"balanced full range", 1, 5000, 256, 50},
		{"set heavy", 2, 5000, 256, 85},
		{"clear heavy", 3, 5000, 256, 15},
		{"single word", 4, 3000, 64, 50},
		{"crossing the first word boundary", 5, 3000, 130, 60},
		{"tiny range, many collisions", 6, 3000, 8, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			prng := rand.New(rand.NewPCG(tt.seed, tt.seed))

			var b BitSet256
			var m refModel

			for op := range tt.ops {
				idx := uint8(prng.IntN(tt.span))

				if prng.IntN(100) < tt.setPct {
					b.Set(idx)
					m[idx] = true
				} else {
					b.Clear(idx)
					m[idx] = false
				}

				if b.Test(idx) != m[idx] {
					t.Fatalf("op %d: Test(%d) = %v, want %v", op, idx, b.Test(idx), m[idx])
				}

				// full comparison now and then
				if op%25 == 0 {
					if exp := m.bitset(); b != exp {
						t.Fatalf("op %d: got %#x, want %#x", op, b, exp)
					}

					if got, want := b.OnesCount(), len(m.idxs()); got != want {
						t.Fatalf("op %d: OnesCount() = %d, want %d", op, got, want)
					}
				}
			}
		})
	}
}

// #####################################################
//  references for the benchmarks, size
// #####################################################

// TestRefLoopReferences verifies that the loop based variants, which are used
// as reference in the benchmarks, are equivalent to the unrolled
// implementations.
func TestRefLoopReferences(t *testing.T) {
	t.Parallel()

	for _, x := range refAllCases() {
		a := x.m.bitset()

		first, ok := a.FirstSet()
		if gotFirst, gotOK := refFirstSetLoop(&a); gotOK != ok || (ok && gotFirst != first) {
			t.Fatalf("%q: refFirstSetLoop() = (%d, %v), FirstSet() = (%d, %v)", x.name, gotFirst, gotOK, first, ok)
		}

		last, ok := a.LastSet()
		if gotLast, gotOK := refLastSetLoop(&a); gotLast != last || gotOK != ok {
			t.Fatalf("%q: refLastSetLoop() = (%d, %v), LastSet() = (%d, %v)", x.name, gotLast, gotOK, last, ok)
		}

		for _, y := range refAllCases() {
			b := y.m.bitset()

			top, ok := a.AndTop(&b)
			if gotTop, gotOK := refAndTopLoop(&a, &b); gotTop != top || gotOK != ok {
				t.Fatalf("%q, %q: refAndTopLoop() = (%d, %v), AndTop() = (%d, %v)",
					x.name, y.name, gotTop, gotOK, top, ok)
			}
		}
	}
}
