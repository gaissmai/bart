// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Package sparse implements a special sparse array
// with popcount compression for max. 256 items.
package sparse

import (
	"github.com/gaissmai/bart/internal/bitset"
)

// Array256 is a generic implementation of a sparse array
// with popcount compression for max. 256 items with payload T.
type Array256[T any] struct {
	bitset.BitSet256
	Items []T
}

// MustSet of the underlying bitset is forbidden. The bitset and the items are coupled.
// An unsynchronized Set() disturbs the coupling between bitset and Items[].
func (a *Array256[T]) MustSet(uint) {
	panic("forbidden, use InsertAt")
}

// MustClear of the underlying bitset is forbidden. The bitset and the items are coupled.
// An unsynchronized Clear() disturbs the coupling between bitset and Items[].
func (a *Array256[T]) MustClear(uint) {
	panic("forbidden, use DeleteAt")
}

// Get the value at i from sparse array.
//
// example: a.Get(5) -> a.Items[1]
//
//	                        ⬇
//	BitSet256:   [0|0|1|0|0|1|0|...|1] <- 3 bits set
//	Items:       [*|*|*]               <- len(Items) = 3
//	                ⬆
//
//	BitSet256.Test(5):     true
//	BitSet256.popcount(5): 2, for interval [0,5]
//	BitSet256.Rank0(5):    1, equal popcount(5)-1
func (a *Array256[T]) Get(i uint) (value T, ok bool) {
	if a.Test(i) {
		return a.Items[a.Rank0(i)], true
	}
	return
}

// MustGet use it only after a successful test
// or the behavior is undefined, it will NOT PANIC.
func (a *Array256[T]) MustGet(i uint) T {
	return a.Items[a.Rank0(i)]
}

// UpdateAt or set the value at i via callback. The new value is returned
// and true if the value was already present.
func (a *Array256[T]) UpdateAt(i uint, cb func(T, bool) T) (newValue T, wasPresent bool) {
	var rank0 int

	// if already set, get current value
	var oldValue T

	if wasPresent = a.Test(i); wasPresent {
		rank0 = a.Rank0(i)
		oldValue = a.Items[rank0]
	}

	// callback function to get updated or new value
	newValue = cb(oldValue, wasPresent)

	// already set, update and return value
	if wasPresent {
		a.Items[rank0] = newValue

		return newValue, wasPresent
	}

	// new value, insert into bitset ...
	a.BitSet256.MustSet(i)

	// bitset has changed, recalc rank
	rank0 = a.Rank0(i)

	// ... and insert value into slice
	a.insertItem(rank0, newValue)

	return newValue, wasPresent
}

// Len returns the number of items in sparse array.
func (a *Array256[T]) Len() int {
	return len(a.Items)
}

// Copy returns a shallow copy of the Array.
// The elements are copied using assignment, this is no deep clone.
func (a *Array256[T]) Copy() *Array256[T] {
	if a == nil {
		return nil
	}

	// copy the fields
	return &Array256[T]{
		BitSet256: a.BitSet256,
		Items:     append(a.Items[:0:0], a.Items...),
	}
}

// InsertAt a value at i into the sparse array.
// If the value already exists, overwrite it with val and return true.
func (a *Array256[T]) InsertAt(i uint, value T) (exists bool) {
	// slot exists, overwrite value
	if a.Test(i) {
		a.Items[a.Rank0(i)] = value
		return true
	}

	// new, insert into bitset ...
	a.BitSet256.MustSet(i)

	// ... and slice
	a.insertItem(a.Rank0(i), value)

	return false
}

// DeleteAt a value at i from the sparse array, zeroes the tail.
func (a *Array256[T]) DeleteAt(i uint) (value T, exists bool) {
	if a.Len() == 0 || !a.Test(i) {
		return
	}

	rank0 := a.Rank0(i)
	value = a.Items[rank0]

	// delete from slice
	a.deleteItem(rank0)

	// delete from bitset
	a.BitSet256.MustClear(i)

	return value, true
}

// insertItem inserts the item at index i, shift the rest one pos right
//
// It panics if i is out of range.
func (a *Array256[T]) insertItem(i int, item T) {
	if len(a.Items) < cap(a.Items) {
		a.Items = a.Items[:len(a.Items)+1] // fast resize, no alloc
	} else {
		var zero T
		a.Items = append(a.Items, zero) // append one item, mostly enlarge cap by more than one item
	}

	_ = a.Items[i]                   // BCE
	copy(a.Items[i+1:], a.Items[i:]) // shift one slot right, starting at [i]
	a.Items[i] = item                // insert new item at [i]
}

// deleteItem at index i, shift the rest one pos left and clears the tail item
//
// It panics if i is out of range.
func (a *Array256[T]) deleteItem(i int) {
	var zero T

	_ = a.Items[i]                   // BCE
	copy(a.Items[i:], a.Items[i+1:]) // shift left, overwrite item at [i]

	nl := len(a.Items) - 1 // new len

	a.Items[nl] = zero     // clear the tail item
	a.Items = a.Items[:nl] // new len, cap is unchanged
}
