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

// Copy returns a shallow copy of the Array.
// The elements are copied using assignment, this is no deep clone.
func (s *Array[T]) Copy() *Array[T] {
	if s == nil {
		return nil
	}

	var items []T

	if s.Items != nil {
		items = make([]T, len(s.Items), cap(s.Items))
		copy(items, s.Items) // shallow
	}

	return &Array[T]{
		s.BitSet.Clone(),
		items,
	}
}

// rank is the key of the popcount compression algorithm,
// mapping between bitset index and slice index.
func (s *Array[T]) rank(i uint) int {
	// adjust offset by one to slice index.
	return s.Rank(i) - 1
}

// InsertAt a value at i into the sparse array.
// If the value already exists, overwrite it with val and return true.
// The capacity is identical to the length after insertion.
func (s *Array[T]) InsertAt(i uint, val T) (exists bool) {
	// slot exists, overwrite val
	if s.Len() != 0 && s.Test(i) {
		s.Items[s.rank(i)] = val

		return true
	}

	// new, insert into bitset ...
	s.BitSet = s.Set(i)

	// ... and slice
	s.insertItem(val, s.rank(i))

	return false
}

// DeleteAt a value at i from the sparse array, zeroes the tail.
func (s *Array[T]) DeleteAt(i uint) (val T, exists bool) {
	if s.Len() == 0 || !s.Test(i) {
		return
	}

	rnk := s.rank(i)
	val = s.Items[rnk]

	// delete from slice
	s.deleteItem(rnk)

	// delete from bitset
	s.BitSet = s.Clear(i)

	return val, true
}

// Get the value at i from sparse array.
func (s *Array[T]) Get(i uint) (val T, ok bool) {
	if s.Test(i) {
		return s.Items[s.Rank(i)-1], true
	}
	return
}

// MustGet, use it only after a successful test
// or the behavior is undefined, maybe it panics.
func (s *Array[T]) MustGet(i uint) T {
	return s.Items[s.Rank(i)-1]
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

// insertItem inserts the item at index i, shift the rest one pos right
//
// It panics if i is out of range.
func (s *Array[T]) insertItem(item T, i int) {
	if len(s.Items) < cap(s.Items) {
		s.Items = s.Items[:len(s.Items)+1] // fast resize, no alloc
	} else {
		var zero T
		s.Items = append(s.Items, zero) // appends maybe more than just one item
	}
	copy(s.Items[i+1:], s.Items[i:])
	s.Items[i] = item
	return
}

// deleteItem at index i, shift the rest one pos left and clears the tail item
//
// It panics if i is out of range.
func (s *Array[T]) deleteItem(i int) {
	var zero T
	l := len(s.Items) - 1            // new len
	copy(s.Items[i:], s.Items[i+1:]) // overwrite s[i]
	s.Items[l] = zero                // clear the tail item
	s.Items = s.Items[:l]            // new len, cap is unchanged
}
