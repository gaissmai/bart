// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bitset

import (
	"fmt"
	"math"
	"slices"
	"testing"
)

func TestZeroValue(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("A zero value bitset must not panic: %v", r)
		}
	}()

	var b BitSet256

	b = BitSet256{}
	b.Set(0)

	b = BitSet256{}
	b.Clear(100)

	b = BitSet256{}
	b.Size()

	b = BitSet256{}
	b.Rank0(100)

	b = BitSet256{}
	b.Test(42)

	b = BitSet256{}
	b.NextSet(0)

	b = BitSet256{}
	b.AsSlice(nil)

	b = BitSet256{}
	b.All()

	b = BitSet256{}
	c := BitSet256{}
	b = b.Union(&c)

	b = BitSet256{}
	c = BitSet256{}
	b = b.Intersection(&c)

	b = BitSet256{}
	c = BitSet256{}
	b.IntersectsAny(&c)

	b = BitSet256{}
	c = BitSet256{}
	b.IntersectionTop(&c)
}

func TestBitsetSetOutOfBounds(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Error("A Set() out of bounds MUST panic")
		}
	}()

	b := BitSet256{}
	b.Set(256)
}

func TestBitsetClearOutOfBounds(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Error("A Clear() out of bounds MUST panic")
		}
	}()

	b := BitSet256{}
	b.Clear(256)
}

func TestTest(t *testing.T) {
	t.Parallel()
	var b BitSet256
	b.Set(100)
	if !b.Test(100) {
		t.Errorf("Bit %d is clear, and it shouldn't be.", 100)
	}
}

func TestFirstSet(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		set     []uint
		wantIdx uint
		wantOk  bool
	}{
		{
			name:    "null",
			set:     []uint{},
			wantIdx: 0,
			wantOk:  false,
		},
		{
			name:    "zero",
			set:     []uint{0},
			wantIdx: 0,
			wantOk:  true,
		},
		{
			name:    "1,5",
			set:     []uint{1, 5},
			wantIdx: 1,
			wantOk:  true,
		},
		{
			name:    "5,7",
			set:     []uint{5, 7},
			wantIdx: 5,
			wantOk:  true,
		},
		{
			name:    "2. word",
			set:     []uint{70, 255},
			wantIdx: 70,
			wantOk:  true,
		},
	}

	for _, tc := range testCases {
		var b BitSet256
		for _, u := range tc.set {
			b.Set(u)
		}

		idx, ok := b.FirstSet()

		if ok != tc.wantOk {
			t.Errorf("FirstSet, %s: got ok: %v, want: %v", tc.name, ok, tc.wantOk)
		}

		if idx != tc.wantIdx {
			t.Errorf("FirstSet, %s: got idx: %d, want: %d", tc.name, idx, tc.wantIdx)
		}
	}
}

func TestNextSet(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		//
		set   []uint
		del   []uint
		start uint
		//
		wantIdx uint
		wantOk  bool
	}{
		{
			name:    "null",
			set:     []uint{},
			del:     []uint{},
			start:   0,
			wantIdx: 0,
			wantOk:  false,
		},
		{
			name:    "zero",
			set:     []uint{0},
			del:     []uint{},
			start:   0,
			wantIdx: 0,
			wantOk:  true,
		},
		{
			name:    "1,5",
			set:     []uint{1, 5},
			del:     []uint{},
			start:   0,
			wantIdx: 1,
			wantOk:  true,
		},
		{
			name:    "1,5",
			set:     []uint{1, 5},
			del:     []uint{},
			start:   2,
			wantIdx: 5,
			wantOk:  true,
		},
		{
			name:    "1,5",
			set:     []uint{1, 5},
			del:     []uint{},
			start:   6,
			wantIdx: 0,
			wantOk:  false,
		},
		{
			name:    "1,5,7",
			set:     []uint{1, 5, 7},
			del:     []uint{5},
			start:   2,
			wantIdx: 7,
			wantOk:  true,
		},
		{
			name:    "2. word",
			set:     []uint{1, 70, 255},
			del:     []uint{},
			start:   2,
			wantIdx: 70,
			wantOk:  true,
		},
	}

	for _, tc := range testCases {
		var b BitSet256
		for _, u := range tc.set {
			b.Set(u)
		}

		for _, u := range tc.del {
			b.Clear(u) // without compact
		}

		idx, ok := b.NextSet(tc.start)

		if ok != tc.wantOk {
			t.Errorf("NextSet, %s: got ok: %v, want: %v", tc.name, ok, tc.wantOk)
		}

		if idx != tc.wantIdx {
			t.Errorf("NextSet, %s: got idx: %d, want: %d", tc.name, idx, tc.wantIdx)
		}
	}
}

func TestIsEmpty(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		//
		set []uint
		del []uint
		//
		want bool
	}{
		{
			name: "null",
			set:  []uint{},
			del:  []uint{},
			want: true,
		},
		{
			name: "zero",
			set:  []uint{0},
			del:  []uint{},
			want: false,
		},
		{
			name: "1,5",
			set:  []uint{1, 5},
			del:  []uint{},
			want: false,
		},
		{
			name: "many",
			set:  []uint{1, 65, 130, 190, 250},
			del:  []uint{},
			want: false,
		},
		{
			name: "set clear",
			set:  []uint{1},
			del:  []uint{1},
			want: true,
		},
	}

	for _, tc := range testCases {
		var b BitSet256
		for _, u := range tc.set {
			b.Set(u)
		}

		for _, u := range tc.del {
			b.Clear(u) // without compact
		}

		got := b.IsEmpty()

		if got != tc.want {
			t.Errorf("IsEmpty, %s: got: %v, want: %v", tc.name, got, tc.want)
		}
	}
}

func TestAll(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		//
		set []uint
		del []uint
		//
		wantData []uint
	}{
		{
			name:     "null",
			set:      []uint{},
			del:      []uint{},
			wantData: []uint{},
		},
		{
			name:     "zero",
			set:      []uint{0},
			del:      []uint{},
			wantData: []uint{0}, // bit #0 is set
		},
		{
			name:     "1,5",
			set:      []uint{1, 5},
			del:      []uint{},
			wantData: []uint{1, 5},
		},
		{
			name:     "many",
			set:      []uint{1, 65, 130, 190, 250},
			del:      []uint{},
			wantData: []uint{1, 65, 130, 190, 250},
		},
		{
			name:     "special, last return",
			set:      []uint{1},
			del:      []uint{1}, // delete without compact
			wantData: []uint{},
		},
	}

	for _, tc := range testCases {
		var b BitSet256
		for _, u := range tc.set {
			b.Set(u)
		}

		for _, u := range tc.del {
			b.Clear(u) // without compact
		}

		buf := b.All()

		if !slices.Equal(buf, tc.wantData) {
			t.Errorf("All, %s: returned buf is not equal as expected:\ngot:  %v\nwant: %v",
				tc.name, buf, tc.wantData)
		}
	}
}

func TestAsSlice(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		//
		set []uint
		del []uint
		//
		buf      []uint
		wantData []uint
	}{
		{
			name:     "null",
			set:      []uint{},
			del:      []uint{},
			buf:      make([]uint, 0, 512),
			wantData: []uint{},
		},
		{
			name:     "zero",
			set:      []uint{0},
			del:      []uint{},
			buf:      make([]uint, 0, 512),
			wantData: []uint{0}, // bit #0 is set
		},
		{
			name:     "1,5",
			set:      []uint{1, 5},
			del:      []uint{},
			buf:      make([]uint, 0, 512),
			wantData: []uint{1, 5},
		},
		{
			name:     "many",
			set:      []uint{1, 65, 130, 190, 250},
			del:      []uint{},
			buf:      make([]uint, 0, 512),
			wantData: []uint{1, 65, 130, 190, 250},
		},
		{
			name:     "special, last return",
			set:      []uint{1},
			del:      []uint{1},          // delete without compact
			buf:      make([]uint, 0, 5), // buffer
			wantData: []uint{},
		},
	}

	for _, tc := range testCases {
		var b BitSet256
		for _, u := range tc.set {
			b.Set(u)
		}

		for _, u := range tc.del {
			b.Clear(u) // without compact
		}

		buf := b.AsSlice(tc.buf)

		if !slices.Equal(buf, tc.wantData) {
			t.Errorf("AsSlice, %s: returned buf is not equal as expected:\ngot:  %v\nwant: %v",
				tc.name, buf, tc.wantData)
		}
	}
}

func TestCount(t *testing.T) {
	t.Parallel()
	var b BitSet256

	tot := uint(255)
	checkLast := true

	for i := range tot {
		sz := uint(b.Size())
		if sz != i {
			t.Logf("%v", b)
			t.Errorf("Count reported as %d, but it should be %d", sz, i)
			checkLast = false
			break
		}
		b.Set(i)
	}

	if checkLast {
		sz := uint(b.Size())
		if sz != tot {
			t.Errorf("After all bits set, size reported as %d, but it should be %d", sz, tot)
		}
	}
}

// test setting every 3rd bit, just in case something odd is happening
func TestCount2(t *testing.T) {
	t.Parallel()
	var b BitSet256
	tot := uint(64*3 + 11)
	for i := uint(0); i < tot; i += 3 {
		sz := uint(b.Size())
		if sz != i/3 {
			t.Errorf("Count reported as %d, but it should be %d", sz, i)
			break
		}
		b.Set(i)
	}
}

func TestUnion(t *testing.T) {
	t.Parallel()

	var a BitSet256
	var b BitSet256

	for i := uint(1); i < 100; i += 2 {
		a.Set(i)
		b.Set(i - 1)
	}

	for i := uint(100); i < 200; i++ {
		b.Set(i)
	}

	c := a
	c = c.Union(&b)

	d := b
	d = d.Union(&a)

	if c.Size() != 200 {
		t.Errorf("Union should have 200 bits set, but had %d", c.Size())
	}
	if d.Size() != 200 {
		t.Errorf("Union should have 200 bits set, but had %d", d.Size())
	}
}

func TestInplaceIntersection(t *testing.T) {
	t.Parallel()
	var a BitSet256
	var b BitSet256
	for i := uint(1); i < 100; i += 2 {
		a.Set(i)
		b.Set(i - 1)
		b.Set(i)
	}
	for i := uint(100); i < 200; i++ {
		b.Set(i)
	}

	c := a
	c = c.Intersection(&b)

	d := b
	d = d.Intersection(&a)
	if c.Size() != 50 {
		t.Errorf("Intersection should have 50 bits set, but had %d", c.Size())
	}
	if d.Size() != 50 {
		t.Errorf("Intersection should have 50 bits set, but had %d", d.Size())
	}

	if a.IntersectionCardinality(&b) != c.Size() {
		t.Error("Intersection and IntersectionCardinality differ")
	}
	if b.IntersectionCardinality(&a) != c.Size() {
		t.Error("Intersection and IntersectionCardinality differ")
	}
}

func TestIntersects(t *testing.T) {
	t.Parallel()
	var a BitSet256
	var b BitSet256

	for i := uint(1); i < 100; i++ {
		a.Set(i)
	}
	for i := uint(100); i < 200; i++ {
		b.Set(i)
	}

	want := false
	got := a.IntersectsAny(&b)
	if want != got {
		t.Errorf("Intersection should be %v, but got: %v", want, got)
	}

	b = a
	want = true
	got = a.IntersectsAny(&b)
	if want != got {
		t.Errorf("Intersection should be %v, but got: %v", want, got)
	}
}

func TestIntersectionTop(t *testing.T) {
	t.Parallel()
	var a BitSet256
	var b BitSet256
	for i := uint(1); i < 100; i += 2 {
		a.Set(i)
		b.Set(i - 1)
		b.Set(i)
	}
	for i := uint(100); i < 200; i++ {
		b.Set(i)
	}

	wantTop, wantOk := uint(99), true
	gotTop, gotOk := a.IntersectionTop(&b)

	if wantOk != gotOk {
		t.Errorf("IntersectionTop, want %v, got %v", wantOk, gotOk)
	}
	if wantTop != gotTop {
		t.Errorf("IntersectionTop, want %v, got %v", wantTop, gotTop)
	}

	wantTop, wantOk = uint(99), true
	gotTop, gotOk = b.IntersectionTop(&a)

	if wantOk != gotOk {
		t.Errorf("IntersectionTop, want %v, got %v", wantOk, gotOk)
	}

	if wantTop != gotTop {
		t.Errorf("IntersectionTop, want %v, got %v", wantTop, gotTop)
	}
}

// Rank0 is popcount-1
func TestRank0(t *testing.T) {
	t.Parallel()
	u := []uint{0, 3, 5, 7, 11, 62, 63, 64, 70, 150, 255}

	tests := []struct {
		idx  uint
		want int
	}{
		{
			idx:  0,
			want: 0,
		},
		{
			idx:  1,
			want: 0,
		},
		{
			idx:  2,
			want: 0,
		},
		{
			idx:  3,
			want: 1,
		},
		{
			idx:  4,
			want: 1,
		},
		{
			idx:  62,
			want: 5,
		},
		{
			idx:  63,
			want: 6,
		},
		{
			idx:  64,
			want: 7,
		},
		{
			idx:  150,
			want: 9,
		},
		{
			idx:  254,
			want: 9,
		},
		{
			idx:  255,
			want: 10,
		},
	}

	var b BitSet256
	for _, v := range u {
		b.Set(v)
	}

	for _, tc := range tests {
		if got := b.Rank0(tc.idx); got != tc.want {
			t.Errorf("Rank0(%d): want: %d, got: %d", tc.idx, tc.want, got)
		}
	}
}

func TestIntersectionCardinality(t *testing.T) {
	t.Parallel()
	s := BitSet256{0b0000_1010_1010, 0b0000_1010_1010, 0b0000_1010_1010, 0b0000_1010_1010}
	m := BitSet256{0b1111_1111_1111, 0b1111_1111_1111, 0b1111_1111_1111, 0b1111_1111_1111}

	want := 16
	got := s.IntersectionCardinality(&m)
	if got != want {
		t.Errorf("Wrong And %d !=  %d", got, want)
	}
}

func BenchmarkIntersectsAny(b *testing.B) {
	aa := BitSet256{1, 1, 1, 1}

	for i, bb := range []BitSet256{
		{1},
		{0, 1},
		{0, 0, 1},
		{0, 0, 0, 1},
		{},
	} {
		b.Run(fmt.Sprintf("at %d", i), func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				_ = aa.IntersectsAny(&bb)
			}
		})
	}
}

func BenchmarkUnion(b *testing.B) {
	b.Run("Union", func(b *testing.B) {
		aa := &BitSet256{0b0000_1010_1010, 0b0000_1010_1010, 0b0000_1010_1010, 0b0000_1010_1010}
		bb := &BitSet256{0b1111_1111_1111, 0b1111_1111_1111, 0b1111_1111_1111, 0b1111_1111_1111}
		b.ResetTimer()
		for range b.N {
			_ = aa.Union(bb)
		}
	})
}

func BenchmarkIntersection(b *testing.B) {
	aa := &BitSet256{0b0000_1010_1010, 0b0000_1010_1010, 0b0000_1010_1010, 0b0000_1010_1010}
	bb := &BitSet256{0b1111_1111_1111, 0b1111_1111_1111, 0b1111_1111_1111, 0b1111_1111_1111}
	b.ResetTimer()
	for range b.N {
		_ = aa.Intersection(bb)
	}
}

func BenchmarkPopcount(b *testing.B) {
	aa := BitSet256{0b0000_1010_1010, 0b0000_1010_1010, 0b0000_1010_1010, 0b0000_1010_1010}
	bb := BitSet256{0b1111_1111_1111, 0b1111_1111_1111, 0b1111_1111_1111, 0b1111_1111_1111}

	b.ResetTimer()
	for range b.N {
		_ = aa.IntersectionCardinality(&bb)
	}
}

func BenchmarkIntersectionCardinality(b *testing.B) {
	aa := BitSet256{0b0000_1010_1010, 0b0000_1010_1010, 0b0000_1010_1010, 0b0000_1010_1010}

	b.ResetTimer()
	for range b.N {
		_ = aa.popcnt()
	}
}

func BenchmarkRank0(b *testing.B) {
	aa := BitSet256{0b0000_1010_1010, 0b0000_1010_1010, 0b0000_1010_1010, 0b0000_1010_1010}
	for _, i := range []uint{64*4 - 1, 64*3 - 11, 64*2 - 11, 64*1 - 11, 1, 0} {
		b.Run(fmt.Sprintf("for %d", i), func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				_ = aa.Rank0(i)
			}
		})
	}
}

func BenchmarkIsEmpty(b *testing.B) {
	for i, bb := range []BitSet256{
		{1},
		{0, 1},
		{0, 0, 1},
		{0, 0, 0, 1},
		{},
	} {
		b.Run(fmt.Sprintf("at %d", i), func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				_ = bb.IsEmpty()
			}
		})
	}
}

func BenchmarkFirstSet(b *testing.B) {
	for i, bb := range []BitSet256{
		{1},
		{0, 1},
		{0, 0, 1},
		{0, 0, 0, 1},
		{},
	} {
		b.Run(fmt.Sprintf("at %d", i), func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				_, _ = bb.FirstSet()
			}
		})
	}
}

func BenchmarkNextSet(b *testing.B) {
	for i, bb := range []BitSet256{
		{1},
		{0, 1},
		{0, 0, 1},
		{0, 0, 0, 1},
		{},
	} {
		b.Run(fmt.Sprintf("at %d", i), func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				_, _ = bb.NextSet(0)
			}
		})
	}
}

func BenchmarkIntersectionTop(b *testing.B) {
	for i, aa := range []BitSet256{
		{1},
		{0, 1},
		{0, 0, 1},
		{0, 0, 0, 1},
		{0},
		{0},
		{0},
		{0},
	} {
		b.Run(fmt.Sprintf("at %d", i), func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				_, _ = aa.IntersectionTop(&aa)
			}
		})
	}
}

func BenchmarkAsSlice(b *testing.B) {
	for i, aa := range []BitSet256{
		{1},
		{1, 1},
		{1, 1, 1},
		{1, 1, 1, 1},
	} {
		b.Run(fmt.Sprintf("sparse at %d", i), func(b *testing.B) {
			buf := make([]uint, 256)
			b.ResetTimer()
			for range b.N {
				_ = aa.AsSlice(buf)
			}
		})
	}

	for i, aa := range []BitSet256{
		{math.MaxUint64},
		{math.MaxUint64, math.MaxUint64},
		{math.MaxUint64, math.MaxUint64, math.MaxUint64},
		{math.MaxUint64, math.MaxUint64, math.MaxUint64, math.MaxUint64},
	} {
		b.Run(fmt.Sprintf("dense at %d", i), func(b *testing.B) {
			buf := make([]uint, 256)
			b.ResetTimer()
			for range b.N {
				_ = aa.AsSlice(buf)
			}
		})
	}
}
