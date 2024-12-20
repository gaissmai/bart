//go:build go1.23

// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bitset

import (
	"fmt"
	"slices"
	"testing"
)

func TestAllBitSetIter(t *testing.T) {
	t.Parallel()
	tc := []uint{0, 1, 2, 5, 10, 20, 50, 100, 200, 500, 511}

	for _, n := range tc {
		t.Run(fmt.Sprintf("n: %3d", n), func(t *testing.T) {
			t.Parallel()
			var b BitSet
			seen := make(map[uint]bool)

			for u := range n {
				b = b.Set(u)
				seen[u] = true
			}

			// range over func
			for u := range b.All() {
				if seen[u] != true {
					t.Errorf("bit: %d, expected true, got false", u)
				}
				delete(seen, u)
			}

			// check if all entries visited
			if len(seen) != 0 {
				t.Fatalf("traverse error, not all entries visited")
			}
		})
	}
}

func TestAllBitSetIterStop(t *testing.T) {
	t.Parallel()

	var b BitSet

	for u := range uint(20) {
		b = b.Set(u)
	}

	got := []uint{}
	want := []uint{0, 1, 2, 3, 4}

	// range over func
	for u := range b.All() {
		if u > 4 {
			break
		}
		got = append(got, u)
	}

	if !slices.Equal(got, want) {
		t.Fatalf("rangefunc with break condition, expected: %v, got: %v", want, got)
	}
}

func TestAllBitSetCallback(t *testing.T) {
	t.Parallel()
	tc := []uint{0, 1, 2, 5, 10, 20, 50, 100, 200, 500, 511}

	for _, n := range tc {
		t.Run(fmt.Sprintf("n: %3d", n), func(t *testing.T) {
			t.Parallel()
			var b BitSet
			seen := make(map[uint]bool)

			for u := range n {
				b = b.Set(u)
				seen[u] = true
			}

			// All() with callback, no range-over-func before go1.23
			b.All()(func(u uint) bool {
				if seen[u] != true {
					t.Errorf("bit: %d, expected true, got false", u)
				}
				delete(seen, u)
				return true
			})

			// check if all entries visited
			if len(seen) != 0 {
				t.Fatalf("traverse error, not all entries visited")
			}
		})
	}
}
