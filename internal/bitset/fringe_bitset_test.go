// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bitset

import (
	"fmt"
	"slices"
	"testing"
)

func TestFringeZeroValue(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("A zero value bitset must not panic: %v", r)
		}
	}()

	b := BitSetFringe{}
	b.Set(0)

	b = BitSetFringe{}
	b.Clear(1000)

	b = BitSetFringe{}
	b.Size()

	b = BitSetFringe{}
	b.Rank0(100)

	b = BitSetFringe{}
	b.Test(42)

	b = BitSetFringe{}
	b.NextSet(0)

	b = BitSetFringe{}
	b.AsSlice(nil)

	b = BitSetFringe{}
	b.All()

	b = BitSetFringe{}
	c := BitSetFringe{}
	b.InPlaceIntersection(&c)

	b = BitSetFringe{}
	c = BitSetFringe{}
	b.InPlaceUnion(&c)

	b = BitSetFringe{}
	c = BitSetFringe{}
	b.IntersectsAny(&c)

	b = BitSetFringe{}
	c = BitSetFringe{}
	b.IntersectionTop(&c)
}

func TestFringeTest(t *testing.T) {
	t.Parallel()
	var b BitSetFringe
	b.Set(100)
	if !b.Test(100) {
		t.Errorf("Bit %d is clear, and it shouldn't be.", 100)
	}
}

func TestFringeFirstSet(t *testing.T) {
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
		var b BitSetFringe
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

func TestFringeNextSet(t *testing.T) {
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
		var b BitSetFringe
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

func TestFringeIsEmpty(t *testing.T) {
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
		var b BitSetFringe
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

func TestFringeAll(t *testing.T) {
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
		var b BitSetFringe
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

func TestFringeAsSlice(t *testing.T) {
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
		var b BitSetFringe
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

func TestFringeCount(t *testing.T) {
	t.Parallel()
	var b BitSetFringe

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
func TestFringeCount2(t *testing.T) {
	t.Parallel()
	var b BitSetFringe
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

func TestFringeInPlaceUnion(t *testing.T) {
	t.Parallel()
	var a BitSetFringe
	var b BitSetFringe
	for i := uint(1); i < 100; i += 2 {
		a.Set(i)
		b.Set(i - 1)
	}
	for i := uint(100); i < 200; i++ {
		b.Set(i)
	}
	c := a
	c.InPlaceUnion(&b)
	d := b
	d.InPlaceUnion(&a)
	if c.Size() != 200 {
		t.Errorf("Union should have 200 bits set, but had %d", c.Size())
	}
	if d.Size() != 200 {
		t.Errorf("Union should have 200 bits set, but had %d", d.Size())
	}
}

func TestFringeInplaceIntersection(t *testing.T) {
	t.Parallel()
	var a BitSetFringe
	var b BitSetFringe
	for i := uint(1); i < 100; i += 2 {
		a.Set(i)
		b.Set(i - 1)
		b.Set(i)
	}
	for i := uint(100); i < 200; i++ {
		b.Set(i)
	}
	c := a
	c.InPlaceIntersection(&b)
	d := b
	d.InPlaceIntersection(&a)
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

func TestFringeIntersects(t *testing.T) {
	t.Parallel()
	var a BitSetFringe
	var b BitSetFringe

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

func TestFringeIntersectionTop(t *testing.T) {
	t.Parallel()
	var a BitSetFringe
	var b BitSetFringe
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
func TestFringeRank0(t *testing.T) {
	t.Parallel()
	u := []uint{2, 3, 5, 7, 11, 70, 150}
	var b BitSetFringe
	for _, v := range u {
		b.Set(v)
	}

	if b.Rank0(5) != 2 {
		t.Error("Unexpected rank")
		return
	}
	if b.Rank0(6) != 2 {
		t.Error("Unexpected rank")
		return
	}
	if b.Rank0(63) != 4 {
		t.Error("Unexpected rank")
		return
	}
	if b.Rank0(1500) != 6 {
		t.Error("Unexpected rank")
		return
	}
}

func TestFringePopcntAnd(t *testing.T) {
	t.Parallel()
	s := BitSetFringe{0b0000_1010_1010, 0b0000_1010_1010, 0b0000_1010_1010, 0b0000_1010_1010}
	m := BitSetFringe{0b1111_1111_1111, 0b1111_1111_1111, 0b1111_1111_1111, 0b1111_1111_1111}

	want := 16
	got := popcntAnd(s[:], m[:])
	if got != want {
		t.Errorf("Wrong And %d !=  %d", got, want)
	}
}

func TestFringePopcntCompare(t *testing.T) {
	t.Parallel()
	var i uint64
	for i = range 10_000 {
		bs := BitSetFringe{i, i, i, i}

		got := bs.popcnt()
		want := popcntSlice(bs[:])

		if got != want {
			t.Errorf("Wrong popcount for {%d, %d, %d, %d}: %d != %d", i, i, i, i, got, want)
		}
	}
}

func BenchmarkIntersectsAny(b *testing.B) {
	aa := BitSetFringe{1, 1, 1, 1}

	for i, bb := range []BitSetFringe{
		{1},
		{0, 1},
		{0, 0, 1},
		{0, 0, 0, 1},
		{},
	} {
		b.Run(fmt.Sprintf("IntersectsAnyGo at %d", i), func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				_ = aa.IntersectsAny(&bb)
			}
		})
	}
}

func BenchmarkFringePopcount(b *testing.B) {
	aa := BitSetFringe{0b0000_1010_1010, 0b0000_1010_1010, 0b0000_1010_1010, 0b0000_1010_1010}
	bb := BitSetFringe{0b1111_1111_1111, 0b1111_1111_1111, 0b1111_1111_1111, 0b1111_1111_1111}

	b.Run("PopcountFringe", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			_ = aa.popcnt()
		}
	})

	b.Run("PopcountSlice", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			_ = popcntSlice(aa[:])
		}
	})

	b.Run("PopcountAndFringe", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			_ = aa.popcntAnd(&bb)
		}
	})

	b.Run("PopcountAndSlice", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			_ = popcntAnd(aa[:], bb[:])
		}
	})
}

func BenchmarkFringeRank0(b *testing.B) {
	aa := BitSetFringe{0b0000_1010_1010, 0b0000_1010_1010, 0b0000_1010_1010, 0b0000_1010_1010}
	for _, i := range []uint{10_000, 64*4 - 11, 64*3 - 11, 64*2 - 11, 64*1 - 11} {
		b.Run(fmt.Sprintf("FringeRank0(%d)", i), func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				_ = aa.Rank0(i)
			}
		})
	}
}

func BenchmarkFringeIsEmpty(b *testing.B) {
	for i, bb := range []BitSetFringe{
		{1},
		{0, 1},
		{0, 0, 1},
		{0, 0, 0, 1},
		{},
	} {
		b.Run(fmt.Sprintf("IsEmpty at %d", i), func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				_ = bb.IsEmpty()
			}
		})
	}
}

func BenchmarkFringeFirstSet(b *testing.B) {
	for i, bb := range []BitSetFringe{
		{1},
		{0, 1},
		{0, 0, 1},
		{0, 0, 0, 1},
		{},
	} {
		b.Run(fmt.Sprintf("FirstSet at %d", i), func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				_, _ = bb.FirstSet()
			}
		})
	}
}
