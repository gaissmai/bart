// Copyright 2014 Will Fitzgerald. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file tests bit sets

package bitset

import (
	"fmt"
	"math"
	"testing"
)

func TestEmptyBitSet(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Error("A zero-length bitset should be fine")
		}
	}()
	b := MustNew(0)
	if b.bitsCapacity() != 0 {
		t.Errorf("Empty set should have capacity 0, not %d", b.bitsCapacity())
	}
}

func TestZeroValueBitSet(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Error("A zero-length bitset should be fine")
		}
	}()
	var b BitSet
	if b.bitsCapacity() != 0 {
		t.Errorf("Empty set should have capacity 0, not %d", b.bitsCapacity())
	}
}

func TestBitSetNew(t *testing.T) {
	v := MustNew(16)
	if v.Test(0) {
		t.Errorf("Unable to make a bit set and read its 0th value.")
	}
}

func TestBitSetHuge(t *testing.T) {
	v := MustNew(uint(math.MaxUint32))
	if v.Test(0) {
		t.Errorf("Unable to make a huge bit set and read its 0th value.")
	}
}

func TestBitSetIsClear(t *testing.T) {
	v := MustNew(1000)
	for i := range 1000 {
		if v.Test(uint(i)) {
			t.Errorf("Bit %d is set, and it shouldn't be.", i)
		}
	}
}

func TestExtendOnBoundary(t *testing.T) {
	v := MustNew(32)
	defer func() {
		if r := recover(); r != nil {
			t.Error("Border out of index error should not have caused a panic")
		}
	}()
	v.Set(32)
}

func TestExpand(t *testing.T) {
	v := MustNew(0)
	defer func() {
		if r := recover(); r != nil {
			t.Error("Expansion should not have caused a panic")
		}
	}()
	for i := range 512 {
		v.Set(uint(i))
	}
}

func TestBitSetAndGet(t *testing.T) {
	v := MustNew(1000)
	v.Set(100)
	if !v.Test(100) {
		t.Errorf("Bit %d is clear, and it shouldn't be.", 100)
	}
}

func TestIterate(t *testing.T) {
	v := MustNew(10000)
	v.Set(0)
	v.Set(1)
	v.Set(2)
	data := make([]uint, 3)
	c := 0
	for i, e := v.NextSet(0); e; i, e = v.NextSet(i + 1) {
		data[c] = i
		c++
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
	v.Set(10)
	v.Set(2000)
	data = make([]uint, 5)
	c = 0
	for i, e := v.NextSet(0); e; i, e = v.NextSet(i + 1) {
		data[c] = i
		c++
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

func TestOutOfBoundsLong(t *testing.T) {
	v := MustNew(64)
	defer func() {
		if r := recover(); r != nil {
			t.Error("Long distance out of index error should not have caused a panic")
		}
	}()
	v.Set(511)
}

func TestOutOfBoundsClose(t *testing.T) {
	v := MustNew(65)
	defer func() {
		if r := recover(); r != nil {
			t.Error("Local out of index error should not have caused a panic")
		}
	}()
	v.Set(66)
}

func TestCount(t *testing.T) {
	tot := uint(64*4 + 11) // just some multi unit64 number
	v := MustNew(tot)
	checkLast := true
	for i := range tot {
		sz := v.Count()
		if sz != i {
			t.Errorf("Count reported as %d, but it should be %d", sz, i)
			checkLast = false
			break
		}
		v.Set(i)
	}
	if checkLast {
		sz := v.Count()
		if sz != tot {
			t.Errorf("After all bits set, size reported as %d, but it should be %d", sz, tot)
		}
	}
}

// test setting every 3rd bit, just in case something odd is happening
func TestCount2(t *testing.T) {
	tot := uint(64*4 + 11) // just some multi unit64 number
	v := MustNew(tot)
	for i := uint(0); i < tot; i += 3 {
		sz := v.Count()
		if sz != i/3 {
			t.Errorf("Count reported as %d, but it should be %d", sz, i)
			break
		}
		v.Set(i)
	}
}

// nil tests
func TestNullTest(t *testing.T) {
	var v *BitSet
	defer func() {
		if r := recover(); r == nil {
			t.Error("Checking bit of null reference should have caused a panic")
		}
	}()
	v.Test(66)
}

func TestNullSet(t *testing.T) {
	var v *BitSet
	defer func() {
		if r := recover(); r == nil {
			t.Error("Setting bit of null reference should have caused a panic")
		}
	}()
	v.Set(66)
}

func TestNullClear(t *testing.T) {
	var v *BitSet
	defer func() {
		if r := recover(); r == nil {
			t.Error("Clearning bit of null reference should have caused a panic")
		}
	}()
	v.Clear(66)
}

func TestNullCount(t *testing.T) {
	var v BitSet
	defer func() {
		if r := recover(); r != nil {
			t.Error("Counting null reference should not have caused a panic")
		}
	}()
	cnt := v.Count()
	if cnt != 0 {
		t.Errorf("Count reported as %d, but it should be 0", cnt)
	}
}

func TestMustNew(t *testing.T) {
	testCases := []struct {
		length uint
		nwords int
	}{
		{
			length: 0,
			nwords: 0,
		},
		{
			length: 1,
			nwords: 1,
		},
		{
			length: 64,
			nwords: 1,
		},
		{
			length: 65,
			nwords: 2,
		},
		{
			length: 512,
			nwords: 8,
		},
		{
			length: 513,
			nwords: 9,
		},
	}

	for _, tc := range testCases {
		b := MustNew(tc.length)
		if len(b.set) != tc.nwords {
			t.Errorf("length = %d, len(b.set) got: %d, want: %d", tc.length, len(b.set), tc.nwords)
		}
	}
}

func TestInPlaceUnion(t *testing.T) {
	a := MustNew(100)
	b := MustNew(200)
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
	a := MustNew(100)
	b := MustNew(200)
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
}

func TestFrom(t *testing.T) {
	u := []uint64{2, 3, 5, 7, 11}
	b := From(u)
	outType := fmt.Sprintf("%T", b)
	expType := "bitset.BitSet"
	if outType != expType {
		t.Error("Expecting type: ", expType, ", gotf:", outType)
		return
	}
}

func TestWords(t *testing.T) {
	b := new(BitSet)
	c := b.Words()
	outType := fmt.Sprintf("%T", c)
	expType := "[]uint64"
	if outType != expType {
		t.Error("Expecting type: ", expType, ", gotf:", outType)
		return
	}
	if len(c) != 0 {
		t.Error("The slice should be empty")
		return
	}
}

func TestRank(t *testing.T) {
	u := []uint{2, 3, 5, 7, 11, 70, 150}
	b := BitSet{}
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
	if b.Rank(1500) != 7 {
		t.Error("Unexpected rank")
		return
	}
}

func TestNextSetError(t *testing.T) {
	b := new(BitSet)
	c, d := b.NextSet(1)
	if c != 0 || d {
		t.Error("Unexpected values")
		return
	}
}

func TestPopcntSlice(t *testing.T) {
	s := []uint64{2, 3, 5, 7, 11, 13, 17, 19, 23, 29}
	res := popcntSlice(s)
	const l uint64 = 27
	if res != l {
		t.Errorf("Wrong popcount %d != %d", res, l)
	}
}

func TestPopcntAndSlice(t *testing.T) {
	s := []uint64{2, 3, 5, 7, 11, 13, 17, 19, 23, 29}
	m := []uint64{31, 37, 41, 43, 47, 53, 59, 61, 67, 71}
	res := popcntAndSlice(s, m)
	const l uint64 = 18
	if res != l {
		t.Errorf("Wrong And %d !=  %d", res, l)
	}
}
