// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package sparse

import (
	"math/rand/v2"
	"testing"
)

func TestNewArray(t *testing.T) {
	t.Parallel()
	a := new(Array256[int])

	if c := a.Len(); c != 0 {
		t.Errorf("Count, expected 0, got %d", c)
	}
}

func TestSparseArrayCount(t *testing.T) {
	t.Parallel()
	a := new(Array256[int])

	for i := range 255 {
		a.InsertAt(uint8(i), i)
		a.InsertAt(uint8(i), i)
	}
	if c := a.Len(); c != 255 {
		t.Errorf("Count, expected 255, got %d", c)
	}

	for i := range 128 {
		a.DeleteAt(uint8(i))
		a.DeleteAt(uint8(i))
	}
	if c := a.Len(); c != 127 {
		t.Errorf("Count, expected 127, got %d", c)
	}
}

func TestSparseArrayGet(t *testing.T) {
	t.Parallel()
	a := new(Array256[int])

	for i := range 255 {
		a.InsertAt(uint8(i), i)
	}

	for range 100 {
		i := rand.IntN(100)
		v, ok := a.Get(uint8(i))
		if !ok {
			t.Errorf("Get, expected true, got %v", ok)
		}
		if v != i {
			t.Errorf("Get, expected %d, got %d", i, v)
		}

		v = a.MustGet(uint8(i))
		if v != i {
			t.Errorf("MustGet, expected %d, got %d", i, v)
		}
	}
}

func TestSparseArraySetPanic(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("MustSet, expected panic")
		}
	}()

	a := new(Array256[int])

	// must panic
	a.Set(0)
}

func TestSparseArrayClearPanic(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("MustClear, expected panic")
		}
	}()

	a := new(Array256[int])

	// must panic
	a.Clear(0)
}

func TestSparseArrayMustGetPanic(t *testing.T) {
	t.Parallel()
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("MustGet, expected panic")
		}
	}()

	a := new(Array256[int])

	for i := 5; i <= 10; i++ {
		a.InsertAt(uint8(i), i)
	}

	// must panic, runtime error: index out of range [-1]
	a.MustGet(0)
}

func TestSparseArrayUpdate(t *testing.T) {
	t.Parallel()
	a := new(Array256[int])

	for i := range 100 {
		a.InsertAt(uint8(i), i)
	}

	// mult all values * 2
	for i := 150; i >= 0; i-- {
		a.UpdateAt(uint8(i), func(oldVal int, existsOld bool) int {
			newVal := i * 3
			if existsOld {
				newVal = oldVal * 2
			}
			return newVal
		})
	}

	for i := range 100 {
		v, _ := a.Get(uint8(i))
		if v != 2*i {
			t.Errorf("UpdateAt, expected %d, got %d", 2*i, v)
		}
	}

	for i := 100; i <= 150; i++ {
		v, _ := a.Get(uint8(i))
		if v != 3*i {
			t.Errorf("UpdateAt, expected %d, got %d", 3*i, v)
		}
	}
}

func TestSparseArrayCopy(t *testing.T) {
	t.Parallel()
	var a *Array256[int]

	if a.Copy() != nil {
		t.Fatal("copy a nil array, expected nil")
	}

	a = new(Array256[int])

	for i := range 255 {
		a.InsertAt(uint8(i), i)
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
	for i := range 255 {
		a.UpdateAt(uint8(i), func(u int, _ bool) int { return u + 1 })
	}

	// cloned array must now differ
	for i, v := range a.Items {
		if b.Items[i] == v {
			t.Errorf("update a after Clone, b must now differ: aValue: %v, bValue: %v", b.Items[i], v)
		}
	}
}
