// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package sparse

import (
	"math/rand/v2"
	"testing"
)

func TestNewArray(t *testing.T) {
	t.Parallel()
	a := new(Array[int])

	if c := a.Len(); c != 0 {
		t.Errorf("Count, expected 0, got %d", c)
	}
}

func TestSparseArrayCount(t *testing.T) {
	t.Parallel()
	a := new(Array[int])

	for i := range 10_000 {
		a.InsertAt(uint(i), i)
		a.InsertAt(uint(i), i)
	}
	if c := a.Len(); c != 10_000 {
		t.Errorf("Count, expected 10_000, got %d", c)
	}

	for i := range 5_000 {
		a.DeleteAt(uint(i))
		a.DeleteAt(uint(i))
	}
	if c := a.Len(); c != 5_000 {
		t.Errorf("Count, expected 5_000, got %d", c)
	}
}

func TestSparseArrayGet(t *testing.T) {
	t.Parallel()
	a := new(Array[int])

	for i := range 10_000 {
		a.InsertAt(uint(i), i)
	}

	for range 100 {
		i := rand.IntN(10_000)
		v, ok := a.Get(uint(i))
		if !ok {
			t.Errorf("Get, expected true, got %v", ok)
		}
		if v != i {
			t.Errorf("Get, expected %d, got %d", i, v)
		}

		v = a.MustGet(uint(i))
		if v != i {
			t.Errorf("MustGet, expected %d, got %d", i, v)
		}
	}

	_, ok := a.Get(20_000)
	if ok {
		t.Errorf("Get, expected false, got %v", ok)
	}
}

func TestSparseArrayMustGetPanic(t *testing.T) {
	t.Parallel()
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("MustGet, expected panic")
		}
	}()

	a := new(Array[int])

	for i := 5; i <= 10; i++ {
		a.InsertAt(uint(i), i)
	}

	// must panic, runtime error: index out of range [-1]
	a.MustGet(0)
}

func TestSparseArrayUpdate(t *testing.T) {
	t.Parallel()
	a := new(Array[int])

	for i := range 10_000 {
		a.InsertAt(uint(i), i)
	}

	// mult all values * 2
	for i := 15_000; i >= 0; i-- {
		a.UpdateAt(uint(i), func(oldVal int, existsOld bool) int {
			newVal := i * 3
			if existsOld {
				newVal = oldVal * 2
			}
			return newVal
		})
	}

	for i := range 10_000 {
		v, _ := a.Get(uint(i))
		if v != 2*i {
			t.Errorf("UpdateAt, expected %d, got %d", 2*i, v)
		}
	}

	for i := 10_000; i <= 15_000; i++ {
		v, _ := a.Get(uint(i))
		if v != 3*i {
			t.Errorf("UpdateAt, expected %d, got %d", 3*i, v)
		}
	}
}

func TestSparseArrayCompact(t *testing.T) {
	t.Parallel()
	a := new(Array[int])

	for i := range 10_000 {
		a.InsertAt(uint(i), i)
	}

	if l := len(a.Items); l != 10_000 {
		t.Errorf("len, expected 10_000, got %d", l)
	}

	for i := range 7_000 {
		a.DeleteAt(uint(i))
	}
	if l := len(a.Items); l != 3_000 {
		t.Errorf("len, expected 3_000, got %d", l)
	}
	if c := cap(a.Items); c != 3_000 {
		t.Errorf("cap, expected 3_000, got %d", c)
	}
}

func TestSparseArrayCopy(t *testing.T) {
	t.Parallel()
	a := new(Array[int])

	for i := range 10_000 {
		a.InsertAt(uint(i), i)
	}

	// shallow copy
	b := a.Copy()

	// basic values identity
	for i, v := range a.Items {
		if b.Items[i] != v {
			t.Errorf("Clone, expect value: %v, got: %v", v, b.Items[i])
		}
	}

	// update array a
	for i := range 10_000 {
		a.UpdateAt(uint(i), func(u int, _ bool) int { return u + 1 })
	}

	// cloned array must now differ
	for i, v := range a.Items {
		if b.Items[i] == v {
			t.Errorf("update a after Clone, b must now differ: aValue: %v, bValue: %v", b.Items[i], v)
		}
	}
}
