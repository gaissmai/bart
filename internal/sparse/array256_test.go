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

func TestArray256Clone(t *testing.T) {
	type testCase struct {
		name      string
		setup     func() *Array256[int]
		cloneFunc func(int) int
		verify    func(t *testing.T, orig, clone *Array256[int])
	}

	tests := []testCase{
		{
			name: "Clone returns nil for nil receiver",
			setup: func() *Array256[int] {
				return nil
			},
			cloneFunc: func(v int) int { return v },
			verify: func(t *testing.T, orig, clone *Array256[int]) {
				if clone != nil {
					t.Errorf("expected nil clone, got %v", clone)
				}
			},
		},

		{
			name: "Clone empty array",
			setup: func() *Array256[int] {
				return &Array256[int]{}
			},
			cloneFunc: func(v int) int { return v },
			verify: func(t *testing.T, orig, clone *Array256[int]) {
				if clone == orig {
					t.Error("clone pointer must differ from original")
				}
				if clone.BitSet256 != orig.BitSet256 {
					t.Error("BitSet256 must be equal")
				}
				if len(clone.Items) != 0 {
					t.Errorf("Items must be empty, got: %v", clone.Items)
				}
			},
		},

		{
			name: "Clone after InsertAt several elements",
			setup: func() *Array256[int] {
				a := &Array256[int]{}
				a.InsertAt(7, 100)
				a.InsertAt(42, 200)
				a.InsertAt(127, 300)
				return a
			},
			cloneFunc: func(v int) int { return v },
			verify: func(t *testing.T, orig, clone *Array256[int]) {
				if clone == orig {
					t.Error("clone pointer must differ from original")
				}
				if clone.BitSet256 != orig.BitSet256 {
					t.Errorf("BitSet256 mismatch got=%v want=%v", clone.BitSet256, orig.BitSet256)
				}
				if !slices.Equal(clone.Items, orig.Items) {
					t.Errorf("Items mismatch got=%v want=%v", clone.Items, orig.Items)
				}
				if &clone.Items[0] == &orig.Items[0] && len(orig.Items) > 0 {
					t.Error("Items backing arrays must be distinct")
				}
			},
		},

		{
			name: "Clone after InsertAt and DeleteAt",
			setup: func() *Array256[int] {
				a := &Array256[int]{}
				a.InsertAt(5, 55)
				a.InsertAt(10, 110)
				a.InsertAt(15, 150)
				a.DeleteAt(10)
				a.InsertAt(20, 200)
				return a
			},
			cloneFunc: func(v int) int { return v * 10 },
			verify: func(t *testing.T, orig, clone *Array256[int]) {
				if clone == orig {
					t.Error("clone pointer must differ from original")
				}
				if clone.BitSet256 != orig.BitSet256 {
					t.Errorf("BitSet256 mismatch got=%v want=%v", clone.BitSet256, orig.BitSet256)
				}
				expected := make([]int, len(orig.Items))
				for i, v := range orig.Items {
					expected[i] = v * 10
				}
				if !slices.Equal(clone.Items, expected) {
					t.Errorf("Items mismatch got=%v want=%v", clone.Items, expected)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			orig := tc.setup()
			clone := orig.Clone(tc.cloneFunc)

			tc.verify(t, orig, clone)

			// Confirm modifying original does not affect clone
			if orig != nil && len(orig.Items) > 0 && clone != nil {
				orig.Items[0] = 9999
				if clone.Items[0] == 9999 {
					t.Error("Modifying original affected clone")
				}
			}
			// Confirm modifying clone does not affect original
			if clone != nil && len(clone.Items) > 0 && orig != nil {
				clone.Items[0] = -9999
				if orig.Items[0] == -9999 {
					t.Error("Modifying clone affected original")
				}
			}
		})
	}
}

func TestArray256Clone_WithPtrValues(t *testing.T) {
	type testCase struct {
		name      string
		setup     func() *Array256[*int]
		cloneFunc func(*int) *int
		verify    func(t *testing.T, orig, clone *Array256[*int])
	}

	tests := []testCase{
		{
			name: "Clone with pointer values",
			setup: func() *Array256[*int] {
				a := &Array256[*int]{}
				v1 := 10
				v2 := 20
				v3 := 30
				a.InsertAt(1, &v1)
				a.InsertAt(2, &v2)
				a.InsertAt(3, &v3)
				return a
			},
			cloneFunc: func(p *int) *int {
				if p == nil {
					return nil
				}
				val := *p
				return &val
			},
			verify: func(t *testing.T, orig, clone *Array256[*int]) {
				if clone == orig {
					t.Error("clone pointer must differ from original")
				}
				if clone.BitSet256 != orig.BitSet256 {
					t.Errorf("BitSet256 mismatch got=%v want=%v", clone.BitSet256, orig.BitSet256)
				}
				if len(clone.Items) != len(orig.Items) {
					t.Fatalf("Items length mismatch got=%d want=%d", len(clone.Items), len(orig.Items))
				}
				for i := range orig.Items {
					origVal := orig.Items[i]
					cloneVal := clone.Items[i]
					if origVal == cloneVal {
						t.Errorf("item pointer at index %d is the same between original and clone; want distinct pointer", i)
					}
					if origVal == nil && cloneVal != nil {
						t.Errorf("original pointer at index %d is nil but clone is not", i)
					} else if origVal != nil && cloneVal == nil {
						t.Errorf("clone pointer at index %d is nil but original is not", i)
					} else if origVal != nil && cloneVal != nil && *origVal != *cloneVal {
						t.Errorf("values differ at index %d: got=%v want=%v", i, *cloneVal, *origVal)
					}
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			orig := tc.setup()
			clone := orig.Clone(tc.cloneFunc)
			tc.verify(t, orig, clone)

			// modify original pointee to ensure clone unaffected
			if orig != nil && len(orig.Items) > 0 && clone != nil {
				if orig.Items[0] != nil {
					*orig.Items[0] = 9999
					if clone.Items[0] != nil && *clone.Items[0] == 9999 {
						t.Error("Modifying original pointer value affected clone")
					}
				}
			}

			// modify clone pointee to ensure original unaffected
			if clone != nil && len(clone.Items) > 0 && orig != nil {
				if clone.Items[0] != nil {
					*clone.Items[0] = -9999
					if orig.Items[0] != nil && *orig.Items[0] == -9999 {
						t.Error("Modifying clone pointer value affected original")
					}
				}
			}
		})
	}
}
