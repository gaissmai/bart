// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

//
// This is an adapted and simplified version of:
//
//  github.com/bits-and-blooms/bitset
//
// All introduced bugs belong to me!
//
// original license:
// ---------------------------------------------------
// Copyright 2014 Will Fitzgerald. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// ---------------------------------------------------

// Package bitset implements bitsets, a mapping
// between non-negative integers and boolean values.
package bitset

import (
	"math/bits"
)

// the wordSize of a bit set
const wordSize = 64

// log2WordSize is lg(wordSize)
const log2WordSize = 6

// A BitSet is a slice of words. This is an internal package
// with a wide open public API.
type BitSet []uint64

// bitsCapacity returns the number of possible bits in the current set.
func (b BitSet) bitsCapacity() uint {
	return uint(len(b) * 64)
}

// xIdx calculates the index of i in a []uint64
func wIdx(i uint) int {
	return int(i >> log2WordSize) // (i / 64) but faster
}

// bIdx calculates the index of i in a `uint64`
func bIdx(i uint) uint {
	return i & 63 // (i % 64) but faster
}

// wordsNeeded calculates the last word in slice for bit i.
func wordsNeeded(i uint) int {
	return wIdx(i + wordSize)
}

// extendSet adds additional words to incorporate new bits if needed.
func (b BitSet) extendSet(i uint) BitSet {
	size := wordsNeeded(i)

	switch {
	case b == nil:
		b = make([]uint64, size)
	case cap(b) >= size:
		b = b[:size]
	case len(b) < size:
		newset := make([]uint64, size)
		copy(newset, b)
		b = newset
	}
	return b
}

// Test if bit i is set.
func (b BitSet) Test(i uint) bool {
	if i >= b.bitsCapacity() {
		return false
	}
	return b[wIdx(i)]&(1<<bIdx(i)) != 0
}

// Set bit i to 1, the capacity of the bitset is increased accordingly.
func (b *BitSet) Set(i uint) {
	if i >= b.bitsCapacity() {
		*b = b.extendSet(i)
	}
	(*b)[wIdx(i)] |= (1 << bIdx(i))
}

// Clear bit i to 0.
func (b *BitSet) Clear(i uint) {
	if i >= b.bitsCapacity() {
		return
	}
	(*b)[wIdx(i)] &^= (1 << bIdx(i))
}

// Clone this BitSet, returning a new BitSet that has the same bits set.
func (b BitSet) Clone() BitSet {
	if b == nil {
		return nil
	}
	c := BitSet(make([]uint64, len(b)))
	copy(c, b)
	return c
}

// Compact shrinks BitSet so that we preserve all set bits, while minimizing
// memory usage. A new slice is allocated to store the new bits.
func (b *BitSet) Compact() {
	last := len(*b) - 1

	// find last word with at least one bit set.
	for ; last >= 0; last-- {
		if (*b)[last] != 0 {
			newset := make([]uint64, last+1)
			copy(newset, (*b)[:last+1])
			*b = newset
			return
		}
	}

	// not found, shrink to nil
	*b = nil
}

// NextSet returns the next bit set from the specified index,
// including possibly the current index along with an ok code.
func (b BitSet) NextSet(i uint) (uint, bool) {
	x := wIdx(i)
	if x >= len(b) {
		return 0, false
	}

	// process the first word
	word := b[x] >> bIdx(i) // bIdx(i) = i % 64
	if word != 0 {
		return i + uint(bits.TrailingZeros64(word)), true
	}

	// process the following full words
	x++
	for ; x < len(b); x++ {
		if b[x] != 0 {
			return uint(x*wordSize) + uint(bits.TrailingZeros64(b[x])), true
		}
	}
	return 0, false
}

// NextSetMany returns many next bit sets from the specified index,
// including possibly the current index and up to cap(buffer).
// If the returned slice has len zero, then no more set bits were found
//
// It is possible to retrieve all set bits as follow:
//
//	indices := make([]uint, b.Count())
//	b.NextSetMany(0, indices)
func (b BitSet) NextSetMany(i uint, buffer []uint) (uint, []uint) {
	capacity := cap(buffer)
	result := buffer[:capacity]

	x := wIdx(i)
	if x >= len(b) || capacity == 0 {
		return 0, result[:0]
	}

	// process the first word
	word := b[x] >> bIdx(i) // bIdx(i) = i % 64

	size := 0
	for word != 0 {
		result[size] = i + uint(bits.TrailingZeros64(word))

		size++
		if size == capacity {
			return result[size-1], result[:size]
		}

		// clear the rightmost set bit
		word &= word - 1
	}

	// process the following full words
	x++
	for j, word := range b[x:] {
		for word != 0 {
			result[size] = (uint(x+j) << 6) + uint(bits.TrailingZeros64(word))

			size++
			if size == capacity {
				return result[size-1], result[:size]
			}

			// clear the rightmost set bit
			word &= word - 1
		}
	}

	if size > 0 {
		return result[size-1], result[:size]
	}

	return 0, result[:0]
}

// IntersectionCardinality computes the cardinality of the intersection
func (b BitSet) IntersectionCardinality(c BitSet) int {
	return popcntAndSlice(b, c)
}

// InPlaceIntersection overwrites and computes the intersection of
// base set with the compare set.
// This is the BitSet equivalent of & (and)
func (b *BitSet) InPlaceIntersection(c BitSet) {
	bLen := len(*b)
	cLen := len(c)

	// intersect b with shorter or equal c
	if bLen >= cLen && cLen != 0 {
		for i := range cLen {
			(*b)[i] &= c[i]
		}
		for i := cLen; i < bLen; i++ {
			(*b)[i] = 0
		}
		return
	}

	// intersect b with longer c
	for i := range bLen {
		(*b)[i] &= c[i]
	}

	newset := make([]uint64, cLen)
	copy(newset, *b)
	*b = newset
}

// InPlaceUnion creates the destructive union of base set with compare set.
// This is the BitSet equivalent of | (or).
func (b *BitSet) InPlaceUnion(c BitSet) {
	bLen := len(*b)
	cLen := len(c)

	// union b with shorter or equal c
	if bLen >= cLen {
		for i := range cLen {
			(*b)[i] |= c[i]
		}
		return
	}

	// union b with longer c
	newset := make([]uint64, cLen)
	copy(newset, *b)
	*b = newset

	for i := range cLen {
		(*b)[i] |= c[i]
	}
}

// Count (number of set bits).
// Also known as "popcount" or "population count".
func (b BitSet) Count() int {
	return popcntSlice(b)
}

// Rank returns the number of set bits up to and including the index
// that are set in the bitset.
func (b BitSet) Rank(i uint) int {
	if wIdx(i+1) >= len(b) {
		return popcntSlice(b)
	}

	answer := popcntSlice(b[:wIdx(i+1)])

	// word boundary?
	if bIdx(i+1) == 0 {
		return answer
	}

	return answer + bits.OnesCount64(b[wIdx(i+1)]<<(64-bIdx(i+1)))
}

// popcntSlice, count the bits set in slice.
func popcntSlice(s []uint64) int {
	var cnt int
	for _, x := range s {
		cnt += bits.OnesCount64(x)
	}
	return cnt
}

// popcntAndSlice, uint64 words are bitwise & followed by popcount.
func popcntAndSlice(s, m []uint64) int {
	if len(m) < len(s) {
		s, m = m, s
	}

	var cnt int
	for j := range s {
		cnt += bits.OnesCount64(s[j] & m[j])
	}
	return cnt
}
