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
	b.Rank(100)

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

func TestTest(t *testing.T) {
	t.Parallel()
	var b BitSet256
	b.Set(100)
	if !b.Test(100) {
		t.Errorf("Test(%d) is false", 100)
	}
}

func TestString(t *testing.T) {
	t.Parallel()
	bs := BitSet256{}
	bs.Set(0)
	bs.Set(42)
	bs.Set(255)

	want := "[0 42 255]"
	got := bs.String()
	if got != want {
		t.Errorf("String(), expectet: %s, got: %s", want, got)
	}
}

func TestFirstSet(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		set     []uint8
		wantIdx uint8
		wantOk  bool
	}{
		{
			name:    "null",
			set:     []uint8{},
			wantIdx: 0,
			wantOk:  false,
		},
		{
			name:    "zero",
			set:     []uint8{0},
			wantIdx: 0,
			wantOk:  true,
		},
		{
			name:    "1,5",
			set:     []uint8{1, 5},
			wantIdx: 1,
			wantOk:  true,
		},
		{
			name:    "5,7",
			set:     []uint8{5, 7},
			wantIdx: 5,
			wantOk:  true,
		},
		{
			name:    "2. word",
			set:     []uint8{70, 255},
			wantIdx: 70,
			wantOk:  true,
		},
		{
			name:    "3. word",
			set:     []uint8{150, 255},
			wantIdx: 150,
			wantOk:  true,
		},
		{
			name:    "4. word",
			set:     []uint8{233, 255},
			wantIdx: 233,
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
		set   []uint8
		del   []uint8
		start uint8
		//
		wantIdx uint8
		wantOk  bool
	}{
		{
			name:    "null",
			set:     []uint8{},
			del:     []uint8{},
			start:   0,
			wantIdx: 0,
			wantOk:  false,
		},
		{
			name:    "zero",
			set:     []uint8{0},
			del:     []uint8{},
			start:   0,
			wantIdx: 0,
			wantOk:  true,
		},
		{
			name:    "1,5",
			set:     []uint8{1, 5},
			del:     []uint8{},
			start:   0,
			wantIdx: 1,
			wantOk:  true,
		},
		{
			name:    "1,5",
			set:     []uint8{1, 5},
			del:     []uint8{},
			start:   2,
			wantIdx: 5,
			wantOk:  true,
		},
		{
			name:    "1,5",
			set:     []uint8{1, 5},
			del:     []uint8{},
			start:   6,
			wantIdx: 0,
			wantOk:  false,
		},
		{
			name:    "1,5,7",
			set:     []uint8{1, 5, 7},
			del:     []uint8{5},
			start:   2,
			wantIdx: 7,
			wantOk:  true,
		},
		{
			name:    "2. word",
			set:     []uint8{1, 70, 255},
			del:     []uint8{},
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
		set []uint8
		del []uint8
		//
		want bool
	}{
		{
			name: "null",
			set:  []uint8{},
			del:  []uint8{},
			want: true,
		},
		{
			name: "zero",
			set:  []uint8{0},
			del:  []uint8{},
			want: false,
		},
		{
			name: "1,5",
			set:  []uint8{1, 5},
			del:  []uint8{},
			want: false,
		},
		{
			name: "many",
			set:  []uint8{1, 65, 130, 190, 250},
			del:  []uint8{},
			want: false,
		},
		{
			name: "set clear",
			set:  []uint8{1},
			del:  []uint8{1},
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
		set []uint8
		del []uint8
		//
		wantData []uint8
	}{
		{
			name:     "null",
			set:      []uint8{},
			del:      []uint8{},
			wantData: []uint8{},
		},
		{
			name:     "zero",
			set:      []uint8{0},
			del:      []uint8{},
			wantData: []uint8{0}, // bit #0 is set
		},
		{
			name:     "1,5",
			set:      []uint8{1, 5},
			del:      []uint8{},
			wantData: []uint8{1, 5},
		},
		{
			name:     "many",
			set:      []uint8{1, 65, 130, 190, 250},
			del:      []uint8{},
			wantData: []uint8{1, 65, 130, 190, 250},
		},
		{
			name:     "special, last return",
			set:      []uint8{1},
			del:      []uint8{1}, // delete without compact
			wantData: []uint8{},
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
		set []uint8
		del []uint8
		//
		buf      []uint8
		wantData []uint8
	}{
		{
			name:     "null",
			set:      []uint8{},
			del:      []uint8{},
			buf:      make([]uint8, 0, 256),
			wantData: []uint8{},
		},
		{
			name:     "zero",
			set:      []uint8{0},
			del:      []uint8{},
			buf:      make([]uint8, 0, 256),
			wantData: []uint8{0}, // bit #0 is set
		},
		{
			name:     "1,5",
			set:      []uint8{1, 5},
			del:      []uint8{},
			buf:      make([]uint8, 0, 256),
			wantData: []uint8{1, 5},
		},
		{
			name:     "many",
			set:      []uint8{1, 65, 130, 190, 250},
			del:      []uint8{},
			buf:      make([]uint8, 0, 256),
			wantData: []uint8{1, 65, 130, 190, 250},
		},
		{
			name:     "special, last return",
			set:      []uint8{1},
			del:      []uint8{1},          // delete without compact
			buf:      make([]uint8, 0, 5), // buffer
			wantData: []uint8{},
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

	tot := uint8(255)
	checkLast := true

	for i := range tot {
		sz := uint8(b.Size())
		if sz != i {
			t.Logf("%v", b)
			t.Errorf("Count reported as %d, but it should be %d", sz, i)
			checkLast = false
			break
		}
		b.Set(i)
	}

	if checkLast {
		sz := uint8(b.Size())
		if sz != tot {
			t.Errorf("After all bits set, size reported as %d, but it should be %d", sz, tot)
		}
	}
}

// test setting every 3rd bit, just in case something odd is happening
func TestCount2(t *testing.T) {
	t.Parallel()
	var b BitSet256
	tot := uint8(64*3 + 11)
	for i := uint8(0); i < tot; i += 3 {
		sz := uint8(b.Size())
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

	for i := uint8(1); i < 100; i += 2 {
		a.Set(i)
		b.Set(i - 1)
	}

	for i := uint8(100); i < 200; i++ {
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
	for i := uint8(1); i < 100; i += 2 {
		a.Set(i)
		b.Set(i - 1)
		b.Set(i)
	}
	for i := uint8(100); i < 200; i++ {
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

func TestIntersectsAny(t *testing.T) {
	t.Parallel()
	var a BitSet256
	var b BitSet256

	for i := uint8(1); i < 100; i++ {
		a.Set(i)
	}
	for i := uint8(100); i < 200; i++ {
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
	for i := uint8(1); i < 100; i += 2 {
		a.Set(i)
		b.Set(i - 1)
		b.Set(i)
	}
	for i := uint8(100); i < 200; i++ {
		b.Set(i)
	}

	wantTop, wantOk := uint8(99), true
	gotTop, gotOk := a.IntersectionTop(&b)

	if wantOk != gotOk {
		t.Errorf("IntersectionTop, want %v, got %v", wantOk, gotOk)
	}
	if wantTop != gotTop {
		t.Errorf("IntersectionTop, want %v, got %v", wantTop, gotTop)
	}

	wantTop, wantOk = uint8(99), true
	gotTop, gotOk = b.IntersectionTop(&a)

	if wantOk != gotOk {
		t.Errorf("IntersectionTop, want %v, got %v", wantOk, gotOk)
	}

	if wantTop != gotTop {
		t.Errorf("IntersectionTop, want %v, got %v", wantTop, gotTop)
	}
}

// Rank is popcount-1
func TestRank(t *testing.T) {
	t.Parallel()
	u := []uint8{0, 3, 5, 7, 11, 62, 63, 64, 70, 150, 255}

	tests := []struct {
		idx  uint8
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
		if got := b.Rank(tc.idx) - 1; got != tc.want {
			t.Errorf("Rank(%d): want: %d, got: %d", tc.idx, tc.want, got)
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

var (
	boolSink       bool
	intSink        int
	uint8SliceSink []uint8
	bitsetSink     BitSet256
)

func BenchmarkTest(b *testing.B) {
	aa := BitSet256{0b0000_1010_1010, 0b0000_1010_1010, 0b0000_1010_1010, 0b0000_1010_1010}
	for _, i := range []uint8{64*4 - 1, 64*3 - 11, 64*2 - 11, 64*1 - 11, 1, 0} {
		b.Run(fmt.Sprintf("Test: for %d", i), func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				boolSink = aa.Test(i)
			}
		})
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
		b.Run(fmt.Sprintf("Any: at %d", i), func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				boolSink = aa.IntersectsAny(&bb)
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
			bitsetSink = aa.Union(bb)
		}
	})
}

func BenchmarkIntersection(b *testing.B) {
	aa := &BitSet256{0b0000_1010_1010, 0b0000_1010_1010, 0b0000_1010_1010, 0b0000_1010_1010}
	bb := &BitSet256{0b1111_1111_1111, 0b1111_1111_1111, 0b1111_1111_1111, 0b1111_1111_1111}
	b.ResetTimer()
	for range b.N {
		bitsetSink = aa.Intersection(bb)
	}
}

func BenchmarkIntersectionCardinality(b *testing.B) {
	aa := BitSet256{0b0000_1010_1010, 0b0000_1010_1010, 0b0000_1010_1010, 0b0000_1010_1010}
	bb := BitSet256{0b1111_1111_1111, 0b1111_1111_1111, 0b1111_1111_1111, 0b1111_1111_1111}

	b.ResetTimer()
	for range b.N {
		intSink = aa.IntersectionCardinality(&bb)
	}
}

func BenchmarkPopcount(b *testing.B) {
	aa := BitSet256{0b0000_1010_1010, 0b0000_1010_1010, 0b0000_1010_1010, 0b0000_1010_1010}

	b.ResetTimer()
	for range b.N {
		intSink = aa.popcnt()
	}
}

func BenchmarkRank(b *testing.B) {
	aa := BitSet256{0b0000_1010_1010, 0b0000_1010_1010, 0b0000_1010_1010, 0b0000_1010_1010}
	for _, i := range []uint8{64*4 - 1, 64*3 - 11, 64*2 - 11, 64*1 - 11, 1, 0} {
		b.Run(fmt.Sprintf("for %d", i), func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				intSink = aa.Rank(i)
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
				boolSink = bb.IsEmpty()
			}
		})
	}
}

func BenchmarkFirstSet(b *testing.B) {
	for i, bb := range []*BitSet256{
		{1},
		{0, 1},
		{0, 0, 1},
		{0, 0, 0, 1},
		{},
	} {
		b.Run(fmt.Sprintf("FirstSet, at %d", i), func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				_, boolSink = bb.FirstSet()
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
				_, boolSink = bb.NextSet(0)
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
		b.Run(fmt.Sprintf("Top: at %d", i), func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				_, boolSink = aa.IntersectionTop(&aa)
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
			buf := make([]uint8, 256)
			b.ResetTimer()
			for range b.N {
				uint8SliceSink = aa.AsSlice(buf)
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
			buf := make([]uint8, 256)
			b.ResetTimer()
			for range b.N {
				uint8SliceSink = aa.AsSlice(buf)
			}
		})
	}
}
