// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bitset

import (
	"math/rand/v2"
	"slices"
	"testing"
)

func TestNil(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Error("A nil bitset must not panic")
		}
	}()

	b := BitSet(nil)
	b.Set(0)

	b = BitSet(nil)
	_ = b.Clear(1000)

	b = BitSet(nil)
	b.Compact()

	b = BitSet(nil)
	_ = b.Clone()

	b = BitSet(nil)
	b.Size()

	b = BitSet(nil)
	b.Rank(100)

	b = BitSet(nil)
	b.Test(42)

	b = BitSet(nil)
	b.NextSet(0)

	b = BitSet(nil)
	b.AsSlice(nil)

	b = BitSet(nil)
	b.AppendTo(nil)

	b = BitSet(nil)
	c := BitSet(nil)
	b.InPlaceIntersection(c)

	b = BitSet(nil)
	c = BitSet(nil)
	b.InPlaceUnion(c)

	b = BitSet(nil)
	c = BitSet(nil)
	b.IntersectsAny(c)

	b = BitSet(nil)
	c = BitSet(nil)
	b.IntersectionTop(c)
}

func TestZeroValue(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Error("A zero value bitset must not panic")
		}
	}()

	b := BitSet{}
	b.Set(0)

	b = BitSet{}
	_ = b.Clear(1000)

	b = BitSet{}
	b.Compact()

	b = BitSet{}
	b.Clone()

	b = BitSet{}
	b.Size()

	b = BitSet{}
	b.Rank(100)

	b = BitSet{}
	b.Test(42)

	b = BitSet{}
	b.NextSet(0)

	b = BitSet{}
	b.AsSlice(nil)

	b = BitSet{}
	b.AppendTo(nil)

	b = BitSet{}
	c := BitSet{}
	b.InPlaceIntersection(c)

	b = BitSet{}
	c = BitSet{}
	b.InPlaceUnion(c)

	b = BitSet{}
	c = BitSet{}
	b.IntersectsAny(c)

	b = BitSet{}
	c = BitSet{}
	b.IntersectionTop(c)
}

func TestBitSetUntil(t *testing.T) {
	t.Parallel()
	var b BitSet
	var last uint = 900
	b = b.Set(last)
	for i := range last {
		if b.Test(i) {
			t.Errorf("Bit %d is set, and it shouldn't be.", i)
		}
	}
}

func TestExpand(t *testing.T) {
	t.Parallel()
	var b BitSet
	for i := range 512 {
		b = b.Set(uint(i))
	}
	want := 8
	if len(b) != want {
		t.Errorf("Set(511), want len: %d, got: %d", want, len(b))
	}
	if cap(b) != want {
		t.Errorf("Set(511), want cap: %d, got: %d", want, cap(b))
	}

	b = make([]uint64, 0, 4)
	b = b.Set(250)
	want = 4
	if len(b) != want {
		t.Errorf("Set(250), want len: %d, got: %d", want, len(b))
	}
	if cap(b) != want {
		t.Errorf("Set(250), want cap: %d, got: %d", want, cap(b))
	}
}

func TestClone(t *testing.T) {
	t.Parallel()
	var b BitSet
	c := b.Clone()

	if !slices.Equal(b, c) {
		t.Error("clone of nil BitSet should also be nil")
	}

	// make random numbers
	var rands []uint64
	for range 8 {
		rands = append(rands, rand.Uint64())
	}

	b = rands
	c = b.Clone()

	if !slices.Equal(b, c) {
		t.Error("cloned random BitSet is not equal")
	}
}

func TestCompact(t *testing.T) {
	t.Parallel()
	var b BitSet
	for _, i := range []uint{1, 2, 5, 10, 20, 50, 100, 200, 500, 1023} {
		b = b.Set(i)
	}

	want := 16
	if len(b) != want {
		t.Errorf("Set(...), want len: %d, got: %d", want, len(b))
	}
	if cap(b) != want {
		t.Errorf("Set(...), want cap: %d, got: %d", want, cap(b))
	}

	b = b.Clear(1023)
	if len(b) != want {
		t.Errorf("Set(...), want len: %d, got: %d", want, len(b))
	}
	if cap(b) != want {
		t.Errorf("Set(...), want cap: %d, got: %d", want, cap(b))
	}

	b = b.Compact()
	want = 8
	if len(b) != want {
		t.Errorf("Compact(), want len: %d, got: %d", want, len(b))
	}
	if cap(b) != want {
		t.Errorf("Compact(), want cap: %d, got: %d", want, cap(b))
	}

	b = b.Set(10_000)
	b = b.Clear(10_000)
	b = b.Compact()

	want = 8
	if len(b) != want {
		t.Errorf("Compact(), want len: %d, got: %d", want, len(b))
	}
	if cap(b) != want {
		t.Errorf("Compact(), want cap: %d, got: %d", want, cap(b))
	}
}

func TestTest(t *testing.T) {
	t.Parallel()
	var b BitSet
	b = b.Set(100)
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
			set:     []uint{70, 777},
			wantIdx: 70,
			wantOk:  true,
		},
	}

	for _, tc := range testCases {
		var b BitSet
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
			set:     []uint{1, 70, 777},
			del:     []uint{},
			start:   2,
			wantIdx: 70,
			wantOk:  true,
		},
	}

	for _, tc := range testCases {
		var b BitSet
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

func TestAppendTo(t *testing.T) {
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
			buf:      nil,
			wantData: []uint{},
		},
		{
			name:     "zero",
			set:      []uint{0},
			del:      []uint{},
			buf:      nil,
			wantData: []uint{0}, // bit #0 is set
		},
		{
			name:     "1,5",
			set:      []uint{1, 5},
			del:      []uint{},
			buf:      nil,
			wantData: []uint{1, 5},
		},
		{
			name:     "many",
			set:      []uint{1, 65, 130, 190, 250, 300, 380, 420, 480, 511},
			del:      []uint{},
			buf:      nil,
			wantData: []uint{1, 65, 130, 190, 250, 300, 380, 420, 480, 511},
		},
		{
			name:     "special, last return",
			set:      []uint{1},
			del:      []uint{1}, // delete without compact
			buf:      nil,
			wantData: []uint{},
		},
	}

	for _, tc := range testCases {
		var b BitSet
		for _, u := range tc.set {
			b = b.Set(u)
		}

		for _, u := range tc.del {
			b = b.Clear(u) // without compact
		}

		buf := b.AppendTo(tc.buf)

		if !slices.Equal(buf, tc.wantData) {
			t.Errorf("AppendTo, %s: returned buf is not equal as expected:\ngot:  %v\nwant: %v",
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
			set:      []uint{1, 65, 130, 190, 250, 300, 380, 420, 480, 511},
			del:      []uint{},
			buf:      make([]uint, 0, 512),
			wantData: []uint{1, 65, 130, 190, 250, 300, 380, 420, 480, 511},
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
		var b BitSet
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

func TestCount(t *testing.T) {
	t.Parallel()
	var b BitSet
	tot := uint(64*4 + 11) // just an unmagic number
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
func TestCount2(t *testing.T) {
	t.Parallel()
	var b BitSet
	tot := uint(64*4 + 11)
	for i := uint(0); i < tot; i += 3 {
		sz := uint(b.Size())
		if sz != i/3 {
			t.Errorf("Count reported as %d, but it should be %d", sz, i)
			break
		}
		b = b.Set(i)
	}
}

func TestInPlaceUnion(t *testing.T) {
	t.Parallel()
	var a BitSet
	var b BitSet
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

func TestInplaceIntersection(t *testing.T) {
	t.Parallel()
	var a BitSet
	var b BitSet
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

func TestIntersects(t *testing.T) {
	t.Parallel()
	var a BitSet
	var b BitSet

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

func TestIntersectionTop(t *testing.T) {
	t.Parallel()
	var a BitSet
	var b BitSet
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

func TestRank(t *testing.T) {
	t.Parallel()
	u := []uint{2, 3, 5, 7, 11, 70, 150}
	var b BitSet
	for _, v := range u {
		b = b.Set(v)
	}

	if b.Rank(5) != 3 {
		t.Error("Unexpected rank")
		return
	}
	if b.Rank(6) != 3 {
		t.Error("Unexpected rank")
		return
	}
	if b.Rank(63) != 5 {
		t.Error("Unexpected rank")
		return
	}
	if b.Rank(1500) != 7 {
		t.Error("Unexpected rank")
		return
	}
}

func TestPopcntSlice(t *testing.T) {
	t.Parallel()
	s := []uint64{2, 3, 5, 7, 11, 13, 17, 19, 23, 29}
	res := uint64(popcount(s))
	const l uint64 = 27
	if res != l {
		t.Errorf("Wrong popcount %d != %d", res, l)
	}
}

func TestPopcntAndSlice(t *testing.T) {
	t.Parallel()
	s := []uint64{2, 3, 5, 7, 11, 13, 17, 19, 23, 29}
	m := []uint64{31, 37, 41, 43, 47, 53, 59, 61, 67, 71}
	res := uint64(popcountAnd(s, m))
	const l uint64 = 18
	if res != l {
		t.Errorf("Wrong And %d !=  %d", res, l)
	}
}
