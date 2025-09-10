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
		t.Errorf("Len, expected 0, got %d", c)
	}
}

func TestSparseArrayLen(t *testing.T) {
	t.Parallel()
	a := new(Array256[uint8])

	var i uint8
	for i = range 255 {
		a.InsertAt(i, i)
	}
	a.InsertAt(255, 255)
	if c := a.Len(); c != 256 {
		t.Errorf("Len, expected 256, got %d", c)
	}

	for i = range 128 {
		a.DeleteAt(i)
	}
	if c := a.Len(); c != 128 {
		t.Errorf("Len, expected 128, got %d", c)
	}
}

func TestSparseArrayGet(t *testing.T) {
	t.Parallel()
	a := new(Array256[uint8])

	var i uint8
	for i = range 255 {
		a.InsertAt(i, i)
	}
	a.InsertAt(255, 255)

	for range 100 {
		//nolint:gosec
		i := uint8(rand.IntN(100))
		v, ok := a.Get(i)
		if !ok {
			t.Errorf("Get, expected true, got %v", ok)
		}
		if v != i {
			t.Errorf("Get, expected %d, got %d", i, v)
		}

		v = a.MustGet(i)
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
			t.Errorf("Set, expected panic")
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
			t.Errorf("Clear, expected panic")
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

	a := new(Array256[uint8])

	var i uint8
	for i = 5; i <= 10; i++ {
		a.InsertAt(i, i)
	}

	// must panic for index out of range
	a.MustGet(0)
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

			// mutate copy and ensure original is unchanged
			if len(aCopy.Items) > 0 {
				old := aCopy.Items[0]
				aCopy.Items[0] = old + 1
				if original.Items[0] == aCopy.Items[0] {
					t.Error("Copy mutation leaked into original")
				}
				aCopy.Items[0] = old
			}
		})
	}
}
