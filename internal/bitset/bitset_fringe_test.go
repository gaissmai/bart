// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bitset

import (
	"math/rand/v2"
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
	_ = b.Clear(1000)

	b = BitSetFringe{}
	b.Clone()

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
	b.InPlaceIntersection(c)

	b = BitSetFringe{}
	c = BitSetFringe{}
	b.InPlaceUnion(c)

	b = BitSetFringe{}
	c = BitSetFringe{}
	b.IntersectsAny(c)

	b = BitSetFringe{}
	c = BitSetFringe{}
	b.IntersectionTop(c)
}

func TestFringeClone(t *testing.T) {
	t.Parallel()
	var b BitSetFringe
	c := b.Clone()

	if c != b {
		t.Error("clone of nil BitSetArray should also be nil")
	}

	// make random numbers
	var rands [words]uint64
	for i := range 4 {
		rands[i] = rand.Uint64()
	}

	b = rands
	c = b.Clone()

	if b != c {
		t.Error("cloned random BitSetArray is not equal")
	}
}

func TestFringeTest(t *testing.T) {
	t.Parallel()
	var b BitSetFringe
	b = b.Set(100)
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
			b = b.Set(u)
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
			b = b.Set(u)
		}

		for _, u := range tc.del {
			b = b.Clear(u) // without compact
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
			b = b.Set(u)
		}

		for _, u := range tc.del {
			b = b.Clear(u) // without compact
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
			b = b.Set(u)
		}

		for _, u := range tc.del {
			b = b.Clear(u) // without compact
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
			b = b.Set(u)
		}

		for _, u := range tc.del {
			b = b.Clear(u) // without compact
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
	tot := uint(64*3 + 11) // just an unmagic number
	checkLast := true
	for i := range tot {
		sz := uint(b.Size())
		if sz != i {
			t.Errorf("Count reported as %d, but it should be %d", sz, i)
			checkLast = false
			break
		}
		b = b.Set(i)
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
		b = b.Set(i)
	}
}

func TestFringeInPlaceUnion(t *testing.T) {
	t.Parallel()
	var a BitSetFringe
	var b BitSetFringe
	for i := uint(1); i < 100; i += 2 {
		a = a.Set(i)
		b = b.Set(i - 1)
	}
	for i := uint(100); i < 200; i++ {
		b = b.Set(i)
	}
	c := a.Clone()
	c.InPlaceUnion(b)
	d := b.Clone()
	d.InPlaceUnion(a)
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
		a = a.Set(i)
		b = b.Set(i - 1)
		b = b.Set(i)
	}
	for i := uint(100); i < 200; i++ {
		b = b.Set(i)
	}
	c := a.Clone()
	c.InPlaceIntersection(b)
	d := b.Clone()
	d.InPlaceIntersection(a)
	if c.Size() != 50 {
		t.Errorf("Intersection should have 50 bits set, but had %d", c.Size())
	}
	if d.Size() != 50 {
		t.Errorf("Intersection should have 50 bits set, but had %d", d.Size())
	}
	if a.IntersectionCardinality(b) != c.Size() {
		t.Error("Intersection and IntersectionCardinality differ")
	}
	if b.IntersectionCardinality(a) != c.Size() {
		t.Error("Intersection and IntersectionCardinality differ")
	}
}

func TestFringeIntersects(t *testing.T) {
	t.Parallel()
	var a BitSetFringe
	var b BitSetFringe

	for i := uint(1); i < 100; i++ {
		a = a.Set(i)
	}
	for i := uint(100); i < 200; i++ {
		b = b.Set(i)
	}

	want := false
	got := a.IntersectsAny(b)
	if want != got {
		t.Errorf("Intersection should be %v, but got: %v", want, got)
	}

	b = a.Clone()
	want = true
	got = a.IntersectsAny(b)
	if want != got {
		t.Errorf("Intersection should be %v, but got: %v", want, got)
	}
}

func TestFringeIntersectionTop(t *testing.T) {
	t.Parallel()
	var a BitSetFringe
	var b BitSetFringe
	for i := uint(1); i < 100; i += 2 {
		a = a.Set(i)
		b = b.Set(i - 1)
		b = b.Set(i)
	}
	for i := uint(100); i < 200; i++ {
		b = b.Set(i)
	}

	wantTop, wantOk := uint(99), true
	gotTop, gotOk := a.IntersectionTop(b)

	if wantOk != gotOk {
		t.Errorf("IntersectionTop, want %v, got %v", wantOk, gotOk)
	}
	if wantTop != gotTop {
		t.Errorf("IntersectionTop, want %v, got %v", wantTop, gotTop)
	}

	wantTop, wantOk = uint(99), true
	gotTop, gotOk = b.IntersectionTop(a)

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
		b = b.Set(v)
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

func TestFringePopcntSlice(t *testing.T) {
	t.Parallel()
	s := []uint64{2, 3, 5, 7, 11, 13, 17, 19, 23, 29}
	res := uint64(popcntSlice(s))
	const l uint64 = 27
	if res != l {
		t.Errorf("Wrong popcount %d != %d", res, l)
	}
}

func TestFringePopcntAndSlice(t *testing.T) {
	t.Parallel()
	s := []uint64{2, 3, 5, 7, 11, 13, 17, 19, 23, 29}
	m := []uint64{31, 37, 41, 43, 47, 53, 59, 61, 67, 71}
	res := uint64(popcntAnd(s, m))
	const l uint64 = 18
	if res != l {
		t.Errorf("Wrong And %d !=  %d", res, l)
	}
}
