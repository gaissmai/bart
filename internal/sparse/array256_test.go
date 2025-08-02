// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package sparse

import (
	"math/rand/v2"
	"reflect"
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

// ----- The actual test -----
func TestArray256_Clone_PointerItem_InsertAt(t *testing.T) {
	t.Skip("not yet ready")

	type payload struct{ V int }

	buildArray256 := func(items map[uint8]*payload) *Array256[*payload] {
		arr := &Array256[*payload]{}
		for i, v := range items {
			arr.InsertAt(i, v)
		}
		return arr
	}

	cloneFunc := func(src *payload) *payload {
		if src == nil {
			return nil
		}
		out := *src
		return &out
	}

	tests := []struct {
		name      string
		inItems   map[uint8]*payload
		cloneFunc func(*payload) *payload
		wantItems map[uint8]*payload
	}{
		{
			name:      "nil receiver",
			inItems:   nil,
			cloneFunc: cloneFunc,
			wantItems: nil,
		},
		{
			name:      "empty array",
			inItems:   map[uint8]*payload{},
			cloneFunc: cloneFunc,
			wantItems: map[uint8]*payload{},
		},
		{
			name: "with pointers and nil",
			inItems: map[uint8]*payload{
				5:   {V: 1},
				42:  {V: 2},
				200: nil,
			},
			cloneFunc: cloneFunc,
			wantItems: map[uint8]*payload{
				5:   {V: 1},
				42:  {V: 2},
				200: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var in *Array256[*payload]
			if tt.inItems != nil {
				in = buildArray256(tt.inItems)
			}
			var want *Array256[*payload]
			if tt.wantItems != nil {
				want = buildArray256(tt.wantItems)
			}
			got := in.Clone(tt.cloneFunc)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("Clone() = %#v, want %#v", got, want)
			}

			// --- Memory aliasing checks ---
			if got != nil && in != nil {
				for i, gotPtr := range got.Items {
					if i >= len(in.Items) {
						panic("insertItem grew the slice") // TODO
					}
					inPtr := in.Items[i]
					switch {
					case gotPtr == nil && inPtr == nil:
						// ok
					case gotPtr == nil || inPtr == nil:
						t.Errorf("Clone nil mismatch at index %d: got %v, in %v", i, gotPtr, inPtr)
					case gotPtr == inPtr:
						t.Errorf("Aliasing detected at index %d: got.Items[%d] and in.Items[%d] are same pointer (%p)", i, i, i, gotPtr)
					case gotPtr.V != inPtr.V:
						t.Errorf("Value mismatch at index %d: got.V=%v, in.V=%v", i, gotPtr.V, inPtr.V)
					}
				}

				// Mutate a value and ensure original is unchanged.
				for i, gotPtr := range got.Items {
					if gotPtr != nil && i < len(in.Items) && in.Items[i] != nil {
						orig := in.Items[i].V
						gotPtr.V++
						if in.Items[i].V != orig {
							t.Errorf("Aliasing: modifying got.Items[%d].V affected in.Items[%d].V (got %v, in %v)", i, i, gotPtr.V, in.Items[i].V)
						}
					}
				}
			}
		})
	}
}
