// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package sparse

import (
	"math/rand/v2"
	"slices"
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

	a.DeleteAt(0)
	_, ok := a.Get(0)
	if ok {
		t.Errorf("Get, expected false, got %v", ok)
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

func TestSparseArrayModifyAt(t *testing.T) {
	t.Parallel()
	a := new(Array256[int])

	for i := range 100 {
		a.InsertAt(uint8(i), i)
	}

	// mult all values * 2
	for i := 150; i >= 0; i-- {
		a.ModifyAt(uint8(i), func(oldVal int, existsOld bool) (int, bool) {
			newVal := i * 3
			if existsOld {
				newVal = oldVal * 2
			}
			return newVal, false
		})
	}

	for i := range 100 {
		v, _ := a.Get(uint8(i))
		if v != 2*i {
			t.Errorf("ModifyAt, expected %d, got %d", 2*i, v)
		}
	}

	for i := 100; i <= 150; i++ {
		v, _ := a.Get(uint8(i))
		if v != 3*i {
			t.Errorf("ModifyAt, expected %d, got %d", 3*i, v)
		}
	}

	// delete all items
	for i := range 151 {
		a.ModifyAt(uint8(i), func(_ int, _ bool) (int, bool) {
			// always delete
			return 0, true
		})
	}

	if a.Len() != 0 {
		t.Errorf("ModifyAt, Len(), expected %d, got %d", 0, a.Len())
	}
}

func TestSparseArrayCopy(t *testing.T) {
	type testCase struct {
		name  string
		setup func() *Array256[int]
	}

	tests := []testCase{
		{
			name: "Copy of nil returns nil",
			setup: func() *Array256[int] {
				return nil
			},
		},
		{
			name: "Copy of empty Array256",
			setup: func() *Array256[int] {
				return &Array256[int]{}
			},
		},
		{
			name: "Copy after InsertAt few elements",
			setup: func() *Array256[int] {
				a := &Array256[int]{}
				a.InsertAt(10, 100)
				a.InsertAt(20, 200)
				a.InsertAt(30, 300)
				return a
			},
		},
		{
			name: "Copy after Insert and Delete",
			setup: func() *Array256[int] {
				a := &Array256[int]{}
				a.InsertAt(1, 11)
				a.InsertAt(2, 22)
				a.DeleteAt(1)
				a.InsertAt(3, 33)
				return a
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			original := tc.setup()
			aCopy := original.Copy()

			if original == nil {
				if aCopy != nil {
					t.Errorf("Copy of nil should be nil, got %v", aCopy)
				}
				return
			}

			if aCopy == original {
				t.Error("Copy() returned same pointer as original, want distinct copy")
			}

			if aCopy.BitSet256 != original.BitSet256 {
				t.Errorf("BitSet256 not copied properly. got=%v, want=%v", aCopy.BitSet256, original.BitSet256)
			}

			if !slices.Equal(aCopy.Items, original.Items) {
				t.Errorf("Items slice not copied properly. got=%v, want=%v", aCopy.Items, original.Items)
			}

			if len(original.Items) > 0 && len(aCopy.Items) > 0 {
				if &aCopy.Items[0] == &original.Items[0] {
					t.Error("Items backing array not copied, pointers are equal")
				}
			}
		})
	}
}
