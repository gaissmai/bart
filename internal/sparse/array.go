package sparse

import (
	"slices"

	"github.com/bits-and-blooms/bitset"
)

// Array, a generic implementation of a sparse array
// with popcount compression and payload T.
type Array[T any] struct {
	*bitset.BitSet
	Items []T
}

// NewArray, initialize BitSet with zero value.
func NewArray[T any]() *Array[T] {
	return &Array[T]{
		BitSet: new(bitset.BitSet),
	}
}

// Len returns the number of items in sparse array.
func (s *Array[T]) Len() int {
	return len(s.Items)
}

// rank is the key of the popcount compression algorithm,
// mapping between bitset index and slice index.
func (s *Array[T]) rank(i uint) int {
	// adjust offset by one to slice index.
	return int(s.BitSet.Rank(i)) - 1
}

// InsertAt a value at i into the sparse array.
// If the value already exists, overwrite it with val and return true.
func (s *Array[T]) InsertAt(i uint, val T) (exists bool) {
	// slot exists, overwrite val
	if s.BitSet.Test(i) {
		s.Items[s.rank(i)] = val

		return true
	}

	// new, insert into bitset and slice
	s.BitSet.Set(i)
	s.Items = slices.Insert(s.Items, s.rank(i), val)

	return false
}

// DeleteAt, delete a value at i from the sparse array.
func (s *Array[T]) DeleteAt(i uint) (T, bool) {
	var zero T
	if !s.BitSet.Test(i) {
		return zero, false
	}

	rnk := s.rank(i)
	val := s.Items[rnk]

	// delete from slice
	s.Items = slices.Delete(s.Items, rnk, rnk+1)

	// delete from bitset, followed by Compact to reduce memory consumption
	s.BitSet.Clear(i)
	s.BitSet.Compact()

	return val, true
}

// Get the value at i from sparse array.
func (s *Array[T]) Get(i uint) (val T, ok bool) {
	var zero T

	if s.BitSet.Test(i) {
		return s.Items[s.rank(i)], true
	}

	return zero, false
}

// MustGet, use it only after a successful test
// or the behavior is undefined, maybe it panics.
func (s *Array[T]) MustGet(i uint) T {
	// can't use s.Items[s.rank(i)], make it inlineable
	return s.Items[int(s.BitSet.Rank(i))-1]
}

// UpdateAt or set the value at i via callback. The new value is returned
// and true if the val was already present.
func (s *Array[T]) UpdateAt(i uint, cb func(T, bool) T) (newVal T, wasPresent bool) {
	var rnk int

	// if already set, get current value
	var oldVal T

	if wasPresent = s.BitSet.Test(i); wasPresent {
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
	s.BitSet.Set(i)

	// bitset has changed, recalc rank
	rnk = s.rank(i)

	// ... and insert value into slice
	s.Items = slices.Insert(s.Items, rnk, newVal)

	return newVal, wasPresent
}
