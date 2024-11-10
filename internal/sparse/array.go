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
		BitSet: &bitset.BitSet{},
	}
}

// Count, number of items in sparse array.
func (s *Array[T]) Count() int {
	// faster than BitSet.Count()
	return len(s.Items)
}

// rank is the key of the popcount compression algorithm,
// mapping between bitset index and slice index.
func (s *Array[T]) rank(idx uint) int {
	// adjust offset by one to slice index.
	return int(s.BitSet.Rank(idx)) - 1
}

// InsertAt a value at idx into the sparse array.
func (s *Array[T]) InsertAt(idx uint, val T) (ok bool) {
	// prefix exists, overwrite val
	if s.BitSet.Test(idx) {
		s.Items[s.rank(idx)] = val

		return false
	}

	// new, insert into bitset and slice
	s.BitSet.Set(idx)
	s.Items = slices.Insert(s.Items, s.rank(idx), val)

	return true
}

// DeleteAt, delete a value at idx from the sparse array.
func (s *Array[T]) DeleteAt(idx uint) (T, bool) {
	var zero T
	if !s.BitSet.Test(idx) {
		return zero, false
	}

	rnk := s.rank(idx)
	val := s.Items[rnk]

	// delete from slice
	s.Items = slices.Delete(s.Items, rnk, rnk+1)

	// delete from bitset, followed by Compact to reduce memory consumption
	s.BitSet.Clear(idx)
	s.BitSet.Compact()

	return val, true
}

// Get, get the value at idx from sparse array.
func (s *Array[T]) Get(idx uint) (T, bool) {
	var zero T

	if s.BitSet.Test(idx) {
		return s.Items[int(s.BitSet.Rank(idx))-1], true
	}

	return zero, false
}

// MustGet, use it only after a successful test,
// panics otherwise.
func (s *Array[T]) MustGet(idx uint) T {
	return s.Items[int(s.BitSet.Rank(idx))-1]
}

// UpdateAt or set the value at idx via callback. The new value is returned
// and true if the val was already present.
func (s *Array[T]) UpdateAt(idx uint, cb func(T, bool) T) (newVal T, wasPresent bool) {
	var rnk int

	// if already set, get current value
	var oldVal T

	if wasPresent = s.BitSet.Test(idx); wasPresent {
		rnk = s.rank(idx)
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
	s.BitSet.Set(idx)

	// bitset has changed, recalc rank
	rnk = s.rank(idx)

	// ... and insert value into slice
	s.Items = slices.Insert(s.Items, rnk, newVal)

	return newVal, wasPresent
}

// AllSetBits, retrieve all set bits in the sparse array, panics if the buffer isn't big enough.
func (s *Array[T]) AllSetBits(buffer []uint) []uint {
	if cap(buffer) < s.Count() {
		panic("buffer capacity too small")
	}

	_, buffer = s.BitSet.NextSetMany(0, buffer)

	return buffer
}
