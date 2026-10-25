// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Package sparse provides a compact and efficient sparse array
// implementation for addressable key ranges from [0..255].
//
// It is optimized for use in radix tries and lookup structures
// where only a small subset of possible keys contain actual data.
//
// Internally, an Array256 combines a fixed-size BitSet256 with
// a compact packed Items slice. The bitset tracks which key slots
// are occupied, while the Items slice stores the associated payloads.
// The position of a payload in Items is the number of set bits below
// its key, the rank. To keep this mapping cheap, the Array256 carries a
// tiny rank cache with the cumulative popcounts of the bitset words, so
// that a rank needs one popcount instead of four.
//
// Lookup is O(1). Insertion and deletion are O(1) for the bitset and the
// rank cache and O(n) only for shifting the items in Items.
//
// This module avoids memory allocations where possible and provides
// predictable performance even under heavy insert/delete workloads.
package sparse

import (
	"math/bits"

	"github.com/gaissmai/bart/internal/bitset"
)

// Array256 is a popcount-compressed sparse array for up to 256 item slots.
//
// Internally, it consists of:
//   - a BitSet256, where each set bit marks a present slot (i.e., key i used)
//   - a compact Items slice, which stores the payloads only for the set bits
//   - a rank cache, the cumulative popcounts of the bitset words
//
// Each occupied index i ∈ [0..255] can be tested using .Test(i).
// The position of its item within Items is the number of set bits below i.
//
// Invariants, maintained by InsertAt and DeleteAt:
//   - len(Items) equals the number of set bits in the BitSet256
//   - Items[k] is the item of the k-th set bit, counted in ascending index order
//   - cum[w] equals the number of set bits in the bitset words below word w,
//     hence cum[0] is always 0 and no cum value exceeds 192
//
// Callers may read Items and replace its elements in place, but must never
// append to, reslice or reorder it. Direct mutation of the underlying bitset
// (via Set, Clear) is forbidden and will panic,
// because it would break the invariants above.
//
// This structure allows extremely fast and compact prefix-based routing,
// highly efficient child-node containers in prefix tries, or memory-safe
// alternatives to map[uint8]T with deterministic iteration.
//
// On 64-bit platforms an Array256 occupies 64 bytes: 32 bytes for the bitset,
// 24 bytes for the slice header and 4 bytes for the rank cache plus 4 bytes
// of padding.
//
// Example layout:
//
//	BitSet256:   [0 1 0 0 0 1 ...]   // bits 1 and 5 set
//	Items:       [T1, T2]            // a.Items[0] <-> key 1, a.Items[1] <-> key 5
type Array256[T any] struct {
	bitset.BitSet256
	Items []T

	// cum is the rank cache: cum[w] is the number of set bits in the words
	// below w. It is read by rank0 and updated by InsertAt and DeleteAt.
	// The values never exceed 192 (three full words), so uint8 is sufficient.
	cum [4]uint8
}

// Set panics. The bitset is internally coupled with Items[] and the rank cache.
// Use InsertAt to add or overwrite at index i.
func (a *Array256[T]) Set(uint8) {
	panic("forbidden, use InsertAt")
}

// Clear panics. The bitset is internally coupled with Items[] and the rank cache.
// Use DeleteAt to remove index i.
func (a *Array256[T]) Clear(uint8) {
	panic("forbidden, use DeleteAt")
}

// rank0 returns the number of set bits strictly below index i.
//
// If the bit for i is set, rank0(i) is the position of its item in Items.
// If the bit for i is not set, rank0(i) is the position at which an item
// for i must be inserted to keep Items ordered by index.
//
// The result is computed from the cumulative word counts in cum plus one
// popcount of the bits below i in its own word.
// branches.
func (a *Array256[T]) rank0(i uint8) int {
	wi := i >> 6
	mask := uint64(1)<<(i&63) - 1
	return int(a.cum[wi]) + bits.OnesCount64(a.BitSet256[wi]&mask)
}

// incrCum updates the rank cache after a bit was set at index i:
// every word above the word of i now has one more set bit below it.
func (a *Array256[T]) incrCum(i uint8) {
	for w := i>>6 + 1; w < 4; w++ {
		a.cum[w]++
	}
}

// decrCum updates the rank cache after the bit at index i was cleared:
// every word above the word of i now has one less set bit below it.
func (a *Array256[T]) decrCum(i uint8) {
	for w := i>>6 + 1; w < 4; w++ {
		a.cum[w]--
	}
}

// Get returns the value at index i and whether it exists.
//
// If the bit for i is not set, ok is false and value is the zero-value of T.
//
// example: a.Get(5) -> a.Items[1]
//
//	                        ⬇
//	BitSet256:   [0|0|1|0|0|1|0|...|1] <- 3 bits set
//	Items:       [*|*|*]               <- len(Items) = 3
//	                ⬆
//
//	BitSet256.Test(5):     true
//	a.rank0(5):            1   (the set bit at index 2 lies below index 5)
//
// The word of the bitset is loaded once and used for both the presence test
// and the rank, which also keeps the inline cost low.
func (a *Array256[T]) Get(i uint8) (value T, ok bool) {
	wi := i >> 6
	bit := uint64(1) << (i & 63)

	w := a.BitSet256[wi]
	if w&bit == 0 {
		return
	}

	return a.Items[int(a.cum[wi])+bits.OnesCount64(w&(bit-1))], true
}

// MustGet returns the value at index i without checking if it exists.
//
// Use only after ensuring i is set (via Test(i)); otherwise it returns the
// item of the next higher set bit, or panics if there is none. Intended
// only for tight, validated loops.
func (a *Array256[T]) MustGet(i uint8) T {
	return a.Items[a.rank0(i)]
}

// Len returns the number of items in the sparse array.
func (a *Array256[T]) Len() int {
	return len(a.Items)
}

// Copy returns a shallow copy of the Array, including the rank cache.
// The elements are copied using assignment, this is no deep clone.
// The copy has its own backing array for Items, so inserting into or
// deleting from either array does not affect the other.
//
// Copy on a nil receiver returns nil.
func (a *Array256[T]) Copy() *Array256[T] {
	if a == nil {
		return nil
	}

	c := &Array256[T]{
		BitSet256: a.BitSet256,
		Items:     make([]T, len(a.Items)),
		cum:       a.cum,
	}
	copy(c.Items, a.Items)
	return c
}

// InsertAt inserts or overwrites the value at sparse index i and reports
// whether the slot already existed.
//
// If the slot exists, its value is overwritten in place, nothing else changes
// and exists=true is returned.
//
// If the slot is new, the bit for i is set, the rank cache is updated and the
// value is inserted into Items at its sorted position, shifting the following
// items one slot to the right. exists=false is returned.
func (a *Array256[T]) InsertAt(i uint8, value T) (exists bool) {
	// slot exists, just overwrite value
	if a.Test(i) {
		a.Items[a.rank0(i)] = value
		return true
	}

	// new slot: set the bit and update the rank cache first. The words above
	// the word of i are adjusted, so cum[i>>6] and the bits below i in its own
	// word are untouched and rank0(i) is exactly the insertion position.
	a.BitSet256.Set(i)
	a.incrCum(i)

	a.insertItem(a.rank0(i), value)

	return false
}

// DeleteAt removes the value at index i from the sparse array,
// shifting remaining items down in the slice, clearing the bit and
// updating the rank cache.
//
// If the entry exists, it is returned together with true.
// If i is not present, the zero value and false are returned.
func (a *Array256[T]) DeleteAt(i uint8) (value T, exists bool) {
	if !a.Test(i) {
		return value, false
	}

	// the position must be determined while the bit is still set
	// and the rank cache is still unchanged
	value = a.deleteItem(a.rank0(i))

	a.BitSet256.Clear(i)
	a.decrCum(i)

	return value, true
}

// insertItem inserts a new element at the given index position i in the Items slice,
// shifting all following elements one position to the right to make space.
//
// This method must be called with the correct insertion index - that is,
// the rank0 value of the corresponding bit index i in BitSet256.
//
// The slice will be extended by one element. If the capacity allows, this is done
// without reallocation (fast path); otherwise slice growth occurs (slow path).
//
// Example (inserting at rank0 == 2):
//
//	Items before: [A B C D]
//	After insertItem(2, X): [A B X C D]
//
// Panics if i is out of range (i < 0 or i > len(Items)).
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

// deleteItem removes the item at index i from the Items slice and returns it,
// shifting all subsequent items one position to the left,
// and clearing the final (now duplicate) slot.
//
// This is used to remove Items[i] in response to a BitSet256.Clear(i)
// operation, where i is guaranteed to be set and valid.
//
// Example (deleting rank0 == 1):
//
//	Items before: [A B C D]
//	After deleteItem(1): [A C D]
//
// The tail item is explicitly cleared (assigned the zero-value of T)
// to avoid holding references (important for GC with pointer types).
//
// Panics if i is out of range (i < 0 or i >= len(Items)).
func (a *Array256[T]) deleteItem(i int) (item T) {
	var zero T

	item = a.Items[i]
	copy(a.Items[i:], a.Items[i+1:]) // shift left, overwrite item at [i]

	nl := len(a.Items) - 1 // new len

	a.Items[nl] = zero     // clear the tail item
	a.Items = a.Items[:nl] // new len, cap is unchanged

	return item
}
