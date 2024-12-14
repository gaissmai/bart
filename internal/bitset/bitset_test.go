// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

//
// Some tests are taken and modified from:
//
//  github.com/bits-and-blooms/bitset
//
// All introduced bugs belong to me!
//
// original license:
// ---------------------------------------------------
// Copyright 2014 Will Fitzgerald. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// ---------------------------------------------------

package bitset

import (
	"math/rand/v2"
	"slices"
	"testing"
)

func TestNil(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Error("A nil bitset must not panic")
		}
	}()

	b := BitSet(nil)
	b.Set(0)

	b = BitSet(nil)
	b.Clear(1000)

	b = BitSet(nil)
	b.Compact()

	b = BitSet(nil)
	_ = b.Clone()

	b = BitSet(nil)
	b.Count()

	b = BitSet(nil)
	b.Rank(100)

	b = BitSet(nil)
	b.Test(42)

	b = BitSet(nil)
	b.NextSet(0)

	b = BitSet(nil)
	b.NextSetMany(0, nil)

	b = BitSet(nil)
	c := BitSet(nil)
	b.InPlaceIntersection(c)

	b = BitSet(nil)
	c = BitSet(nil)
	b.InPlaceUnion(c)

	b = BitSet(nil)
	c = BitSet(nil)
	b.IntersectionCardinality(c)
}

func TestZeroValue(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Error("A zero value bitset must not panic")
		}
	}()

	b := BitSet{}
	b.Set(0)

	b = BitSet{}
	b.Clear(1000)

	b = BitSet{}
	b.Compact()

	b = BitSet{}
	b.Clone()

	b = BitSet{}
	b.Count()

	b = BitSet{}
	b.Rank(100)

	b = BitSet{}
	b.Test(42)

	b = BitSet{}
	b.NextSet(0)

	b = BitSet{}
	b.NextSetMany(0, nil)

	b = BitSet{}
	c := BitSet{}
	b.InPlaceIntersection(c)

	b = BitSet{}
	c = BitSet{}
	b.InPlaceUnion(c)

	b = BitSet{}
	c = BitSet{}
	b.IntersectionCardinality(c)
}

func TestBitSetUntil(t *testing.T) {
	var b BitSet
	var last uint = 900
	b.Set(last)
	for i := range last {
		if b.Test(i) {
			t.Errorf("Bit %d is set, and it shouldn't be.", i)
		}
	}
}

func TestExpand(t *testing.T) {
	var b BitSet
	for i := range 512 {
		b.Set(uint(i))
	}
	want := 8
	if len(b) != want {
		t.Errorf("Set(511), want len: %d, got: %d", want, len(b))
	}
	if cap(b) != want {
		t.Errorf("Set(511), want cap: %d, got: %d", want, cap(b))
	}
}

func TestClone(t *testing.T) {
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
	var b BitSet
	for _, i := range []uint{1, 2, 5, 10, 20, 50, 100, 200, 500, 1023} {
		b.Set(i)
	}

	want := 16
	if len(b) != want {
		t.Errorf("Set(...), want len: %d, got: %d", want, len(b))
	}
	if cap(b) != want {
		t.Errorf("Set(...), want cap: %d, got: %d", want, cap(b))
	}

	b.Clear(1023)
	if len(b) != want {
		t.Errorf("Set(...), want len: %d, got: %d", want, len(b))
	}
	if cap(b) != want {
		t.Errorf("Set(...), want cap: %d, got: %d", want, cap(b))
	}

	b.Compact()
	want = 8
	if len(b) != want {
		t.Errorf("Compact(), want len: %d, got: %d", want, len(b))
	}
	if cap(b) != want {
		t.Errorf("Compact(), want cap: %d, got: %d", want, cap(b))
	}

	b.Set(10_000)
	b.Clear(10_000)
	b.Compact()

	want = 8
	if len(b) != want {
		t.Errorf("Compact(), want len: %d, got: %d", want, len(b))
	}
	if cap(b) != want {
		t.Errorf("Compact(), want cap: %d, got: %d", want, cap(b))
	}
}

func TestTest(t *testing.T) {
	var b BitSet
	b.Set(100)
	if !b.Test(100) {
		t.Errorf("Bit %d is clear, and it shouldn't be.", 100)
	}
}

func TestNextSet(t *testing.T) {
	var b BitSet
	b.Set(0)
	b.Set(1)
	b.Set(2)

	data := make([]uint, 3)
	j := 0
	for i, ok := b.NextSet(0); ok; i, ok = b.NextSet(i + 1) {
		data[j] = i
		j++
	}
	if data[0] != 0 {
		t.Errorf("bug 0")
	}
	if data[1] != 1 {
		t.Errorf("bug 1")
	}
	if data[2] != 2 {
		t.Errorf("bug 2")
	}
	b.Set(10)
	b.Set(2000)

	data = make([]uint, 5)
	j = 0
	for i, e := b.NextSet(0); e; i, e = b.NextSet(i + 1) {
		data[j] = i
		j++
	}
	if data[0] != 0 {
		t.Errorf("bug 0")
	}
	if data[1] != 1 {
		t.Errorf("bug 1")
	}
	if data[2] != 2 {
		t.Errorf("bug 2")
	}
	if data[3] != 10 {
		t.Errorf("bug 3")
	}
	if data[4] != 2000 {
		t.Errorf("bug 4")
	}
}

func TestNextSetMany(t *testing.T) {
	testCases := []struct {
		name string
		//
		set []uint
		del []uint
		//
		buf      []uint
		wantData []uint
		//
		startIdx uint
		wantIdx  uint
	}{
		{
			name:     "null",
			set:      []uint{},
			del:      []uint{},
			buf:      make([]uint, 0, 512),
			wantData: []uint{},
			startIdx: 0,
			wantIdx:  0,
		},
		{
			name:     "zero",
			set:      []uint{0},
			del:      []uint{},
			buf:      make([]uint, 0, 512),
			wantData: []uint{0}, // bit #0 is set
			startIdx: 0,
			wantIdx:  0,
		},
		{
			name:     "1,5",
			set:      []uint{1, 5},
			del:      []uint{},
			buf:      make([]uint, 0, 512),
			wantData: []uint{1, 5},
			startIdx: 0,
			wantIdx:  5,
		},
		{
			name:     "many",
			set:      []uint{1, 65, 130, 190, 250, 300, 380, 420, 480, 511},
			del:      []uint{},
			buf:      make([]uint, 0, 512),
			wantData: []uint{1, 65, 130, 190, 250, 300, 380, 420, 480, 511},
			startIdx: 0,
			wantIdx:  511,
		},
		{
			name:     "start idx",
			set:      []uint{1, 65, 130, 190, 250, 300, 380, 420, 480, 511},
			del:      []uint{},
			buf:      make([]uint, 0, 512),
			wantData: []uint{250, 300, 380, 420, 480, 511},
			startIdx: 195,
			wantIdx:  511,
		},
		{
			name:     "zero buffer",
			set:      []uint{1, 2, 3, 4, 511},
			del:      []uint{},
			buf:      make([]uint, 0), // buffer
			wantData: []uint{},
			startIdx: 0,
			wantIdx:  0,
		},
		{
			name:     "buffer too short, first word",
			set:      []uint{1, 2, 3, 4, 5, 6, 7, 8, 9},
			del:      []uint{},
			buf:      make([]uint, 0, 5), // buffer
			wantData: []uint{1, 2, 3, 4, 5},
			startIdx: 0,
			wantIdx:  5,
		},
		{
			name:     "buffer too short",
			set:      []uint{65, 66, 67, 68, 69, 70},
			del:      []uint{},
			buf:      make([]uint, 0, 5), // buffer
			wantData: []uint{65, 66, 67, 68, 69},
			startIdx: 0,
			wantIdx:  69,
		},
		{
			name:     "special, last return",
			set:      []uint{1},
			del:      []uint{1},          // delete without compact
			buf:      make([]uint, 0, 5), // buffer
			wantData: []uint{},
			startIdx: 0,
			wantIdx:  0,
		},
	}

	for _, tc := range testCases {
		var b BitSet
		for _, u := range tc.set {
			b.Set(u)
		}

		for _, u := range tc.del {
			b.Clear(u) // without compact
		}

		idx, buf := b.NextSetMany(tc.startIdx, tc.buf)

		if idx != tc.wantIdx {
			t.Errorf("NextSetMany, %s: got next idx: %d, want: %d", tc.name, idx, tc.wantIdx)
		}

		if !slices.Equal(buf, tc.wantData) {
			t.Errorf("NextSetMany, %s: returned buf is not equal as expected:\ngot:  %v\nwant: %v",
				tc.name, buf, tc.wantData)
		}
	}
}

func TestCount(t *testing.T) {
	var b BitSet
	tot := uint(64*4 + 11) // just an unmagic number
	checkLast := true
	for i := range tot {
		sz := uint(b.Count())
		if sz != i {
			t.Errorf("Count reported as %d, but it should be %d", sz, i)
			checkLast = false
			break
		}
		b.Set(i)
	}
	if checkLast {
		sz := uint(b.Count())
		if sz != tot {
			t.Errorf("After all bits set, size reported as %d, but it should be %d", sz, tot)
		}
	}
}

// test setting every 3rd bit, just in case something odd is happening
func TestCount2(t *testing.T) {
	var b BitSet
	tot := uint(64*4 + 11)
	for i := uint(0); i < tot; i += 3 {
		sz := uint(b.Count())
		if sz != i/3 {
			t.Errorf("Count reported as %d, but it should be %d", sz, i)
			break
		}
		b.Set(i)
	}
}

func TestInPlaceUnion(t *testing.T) {
	var a BitSet
	var b BitSet
	for i := uint(1); i < 100; i += 2 {
		a.Set(i)
		b.Set(i - 1)
	}
	for i := uint(100); i < 200; i++ {
		b.Set(i)
	}
	c := a.Clone()
	c.InPlaceUnion(b)
	d := b.Clone()
	d.InPlaceUnion(a)
	if c.Count() != 200 {
		t.Errorf("Union should have 200 bits set, but had %d", c.Count())
	}
	if d.Count() != 200 {
		t.Errorf("Union should have 200 bits set, but had %d", d.Count())
	}
}

func TestInplaceIntersection(t *testing.T) {
	var a BitSet
	var b BitSet
	for i := uint(1); i < 100; i += 2 {
		a.Set(i)
		b.Set(i - 1)
		b.Set(i)
	}
	for i := uint(100); i < 200; i++ {
		b.Set(i)
	}
	c := a.Clone()
	c.InPlaceIntersection(b)
	d := b.Clone()
	d.InPlaceIntersection(a)
	if c.Count() != 50 {
		t.Errorf("Intersection should have 50 bits set, but had %d", c.Count())
	}
	if d.Count() != 50 {
		t.Errorf("Intersection should have 50 bits set, but had %d", d.Count())
	}
	if a.IntersectionCardinality(b) != c.Count() {
		t.Error("Intersection and IntersectionCardinality differ")
	}
	if b.IntersectionCardinality(a) != c.Count() {
		t.Error("Intersection and IntersectionCardinality differ")
	}
}

func TestRank(t *testing.T) {
	u := []uint{2, 3, 5, 7, 11, 70, 150}
	var b BitSet
	for _, v := range u {
		b.Set(v)
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
	s := []uint64{2, 3, 5, 7, 11, 13, 17, 19, 23, 29}
	res := uint64(popcntSlice(s))
	const l uint64 = 27
	if res != l {
		t.Errorf("Wrong popcount %d != %d", res, l)
	}
}

func TestPopcntAndSlice(t *testing.T) {
	s := []uint64{2, 3, 5, 7, 11, 13, 17, 19, 23, 29}
	m := []uint64{31, 37, 41, 43, 47, 53, 59, 61, 67, 71}
	res := uint64(popcntAndSlice(s, m))
	const l uint64 = 18
	if res != l {
		t.Errorf("Wrong And %d !=  %d", res, l)
	}
}
