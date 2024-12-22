// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// package sparse implements a generic sparse array
// with popcount compression.
package sparse

import (
	"github.com/gaissmai/bart/internal/bitset"
)

// Array, a generic implementation of a sparse array
// with popcount compression and payload T.
type Array[T any] struct {
	bitset.BitSet
	Items []T
}

// Len returns the number of items in sparse array.
func (s *Array[T]) Len() int {
	return len(s.Items)
}

// rank is the key of the popcount compression algorithm,
// mapping between bitset index and slice index.
func (s *Array[T]) rank(i uint) int {
	// adjust offset by one to slice index.
	return s.Rank(i) - 1
}

// InsertAt a value at i into the sparse array.
// If the value already exists, overwrite it with val and return true.
func (s *Array[T]) InsertAt(i uint, val T) (exists bool) {
	// slot exists, overwrite val
	if s.Len() != 0 && s.Test(i) {
		s.Items[s.rank(i)] = val

		return true
	}

	// new, insert into bitset and slice
	s.BitSet = s.Set(i)
	s.insertItem(val, s.rank(i))

	return false
}

// DeleteAt, delete a value at i from the sparse array.
func (s *Array[T]) DeleteAt(i uint) (T, bool) {
	var zero T
	if s.Len() == 0 || !s.Test(i) {
		return zero, false
	}

	rnk := s.rank(i)
	val := s.Items[rnk]

	// delete from slice and (maybe) compact it
	s.deleteItem(rnk)

	// delete from bitset, followed by Compact to reduce memory consumption
	s.BitSet = s.Clear(i).Compact()

	return val, true
}

// Get the value at i from sparse array.
func (s *Array[T]) Get(i uint) (val T, ok bool) {
	var zero T

	if s.Len() != 0 && s.Test(i) {
		return s.Items[s.rank(i)], true
	}

	return zero, false
}

// MustGet, use it only after a successful test
// or the behavior is undefined, maybe it panics.
func (s *Array[T]) MustGet(i uint) T {
	return s.Items[s.rank(i)]
}

// UpdateAt or set the value at i via callback. The new value is returned
// and true if the val was already present.
func (s *Array[T]) UpdateAt(i uint, cb func(T, bool) T) (newVal T, wasPresent bool) {
	var rnk int

	// if already set, get current value
	var oldVal T

	if wasPresent = s.Test(i); wasPresent {
		rnk = s.rank(i)
		oldVal = s.Items[rnk]
	}

	// callback function to get updated or new value
	newVal = cb(oldVal, wasPresent)

	// already set, update and return value
	if wasPresent {
		s.Items[rnk] = newVal

		return newVal, wasPresent
	}

	// new val, insert into bitset ...
	s.BitSet = s.Set(i)

	// bitset has changed, recalc rank
	rnk = s.rank(i)

	// ... and insert value into slice
	s.insertItem(newVal, rnk)

	return newVal, wasPresent
}

// insertItem inserts the item at index i.
//
// insertItem panics if i is out of range.
func (s *Array[T]) insertItem(item T, i int) {
	// in place resize, no alloc
	if len(s.Items) < cap(s.Items) {
		s.Items = s.Items[:len(s.Items)+1] // fast resize, no alloc
		copy(s.Items[i+1:], s.Items[i:])
		s.Items[i] = item
		return
	}

	// make new backing array
	newSlice := make([]T, len(s.Items)+1)
	copy(newSlice, s.Items[:i])
	copy(newSlice[i+1:], s.Items[i:])
	newSlice[i] = item
	(*s).Items = newSlice
}

// deleteItem deletes the item at index i.
// It clears/zeroes the elements s[len(s):] and if cap() >= 2*len() compacts the slice.
//
// deleteItem panics if i is out of range.
func (s *Array[T]) deleteItem(i int) {
	l := len(s.Items) - 1            // new len
	copy(s.Items[i:], s.Items[i+1:]) // overwrite s[i]
	clear(s.Items[l:])               // clear/zeroes the tail
	s.Items = s.Items[:l]            // cut to new len
	if cap(s.Items) >= 2*l {         // compact to new len
		s.Items = s.Items[:l:l]
	}
}
