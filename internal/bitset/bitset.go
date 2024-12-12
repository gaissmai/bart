/*
Copyright 2014 Will Fitzgerald. All rights reserved.
Use of this source code is governed by a BSD-style
license that can be found in the LICENSE file.
*/

// Package bitset implements bitsets, a mapping
// between non-negative integers and boolean values.
//
// This is a simplified and stripped down version of:
//
//	github.com/bits-and-blooms/bitset
//
// All bugs belong to me.
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

// extendSet adds additional words to incorporate new bits if needed.
func (b *BitSet) extendSet(i uint) {
	nsize := wordsNeeded(i)
	if b == nil {
		*b = make([]uint64, nsize)
	} else if len(*b) < nsize {
		newset := make([]uint64, nsize)
		copy(newset, *b)
		*b = newset
	}
}

// bitsCapacity returns the number of possible bits in the current set.
func (b BitSet) bitsCapacity() uint {
	return uint(len(b) * 64)
}

// wordsNeeded calculates the number of words needed for i bits.
func wordsNeeded(i uint) int {
	return int(i+wordSize) >> log2WordSize
}

// bitsIndex calculates the index of i in a `uint64`
func bitsIndex(i uint) uint {
	return i & (wordSize - 1) // (i % 64) but faster
}

// Test whether bit i is set.
func (b BitSet) Test(i uint) bool {
	if i >= b.bitsCapacity() {
		return false
	}
	return b[i>>log2WordSize]&(1<<bitsIndex(i)) != 0
}

// Set bit i to 1, the capacity of the bitset is increased accordingly.
func (b *BitSet) Set(i uint) {
	if i >= b.bitsCapacity() {
		b.extendSet(i)
	}
	(*b)[i>>log2WordSize] |= (1 << bitsIndex(i))
}

// Clear bit i to 0.
func (b *BitSet) Clear(i uint) {
	if i >= b.bitsCapacity() {
		return
	}
	(*b)[i>>log2WordSize] &^= (1 << bitsIndex(i))
}

// Clone this BitSet, returning a new BitSet that has the same bits set.
func (b BitSet) Clone() BitSet {
	c := BitSet(make([]uint64, len(b)))
	copy(c, b)
	return c
}

// Compact shrinks BitSet so that we preserve all set bits, while minimizing
// memory usage. A new slice is allocated to store the new bits.
func (b *BitSet) Compact() {
	idx := len(*b) - 1

	// find last word with at least one bit set.
	for ; idx >= 0; idx-- {
		if (*b)[idx] != 0 {
			newset := make([]uint64, idx+1)
			copy(newset, (*b)[:idx+1])
			*b = newset
			return
		}
	}

	// not found
	*b = nil
}

// NextSet returns the next bit set from the specified index,
// including possibly the current index along with an ok code.
func (b BitSet) NextSet(i uint) (uint, bool) {
	x := int(i >> log2WordSize)
	if x >= len(b) {
		return 0, false
	}
	word := b[x]
	word = word >> bitsIndex(i)
	if word != 0 {
		return i + uint(bits.TrailingZeros64(word)), true
	}
	x++
	// bounds check elimination in the loop
	if x < 0 {
		return 0, false
	}
	for x < len(b) {
		if b[x] != 0 {
			return uint(x*wordSize + bits.TrailingZeros64(b[x])), true
		}
		x++

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
	myanswer := buffer
	capacity := cap(buffer)
	x := int(i >> log2WordSize)
	if x >= len(b) || capacity == 0 {
		return 0, myanswer[:0]
	}
	word := b[x] >> bitsIndex(i)
	myanswer = myanswer[:capacity]
	size := int(0)
	for word != 0 {
		r := uint(bits.TrailingZeros64(word))
		t := word & ((^word) + 1)
		myanswer[size] = r + i
		size++
		if size == capacity {
			goto End
		}
		word = word ^ t
	}
	x++
	for idx, word := range b[x:] {
		for word != 0 {
			r := uint(bits.TrailingZeros64(word))
			t := word & ((^word) + 1)
			myanswer[size] = r + (uint(x+idx) << 6)
			size++
			if size == capacity {
				goto End
			}
			word = word ^ t
		}
	}
End:
	if size > 0 {
		return myanswer[size-1], myanswer[:size]
	}
	return 0, myanswer[:0]
}

// IntersectionCardinality computes the cardinality of the intersection
func (b BitSet) IntersectionCardinality(c BitSet) uint {
	if len(b) <= len(c) {
		return uint(popcntAndSlice(b, c))
	}
	return uint(popcntAndSlice(c, b))
}

// InPlaceIntersection overwrites and computes the intersection of
// base set with the compare set.
// This is the BitSet equivalent of & (and)
func (b *BitSet) InPlaceIntersection(c BitSet) {
	bLen := len(*b)
	cLen := len(c)

	// intersect b with shorter or equal c
	if bLen >= cLen {
		// bounds check elimination
		_ = (*b)[cLen-1]
		_ = c[cLen-1]

		for i := range cLen {
			(*b)[i] &= c[i]
		}
		for i := cLen; i < bLen; i++ {
			(*b)[i] = 0
		}
		return
	}

	// intersect b with longer c
	// bounds check elimination
	_ = (*b)[bLen-1]
	_ = c[bLen-1]

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
		// bounds check elimination
		_ = (*b)[cLen-1]
		_ = c[cLen-1]

		for i := range cLen {
			(*b)[i] |= c[i]
		}
		return
	}

	// union b with longer c
	newset := make([]uint64, cLen)
	copy(newset, *b)
	*b = newset
	// bounds check elimination
	_ = (*b)[cLen-1]
	_ = c[cLen-1]

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
func (b BitSet) Rank(index uint) int {
	wordIdx := int((index + 1) >> log2WordSize)

	if wordIdx >= len(b) {
		return popcntSlice(b)
	}

	answer := popcntSlice(b[:wordIdx])

	bitsIdx := bitsIndex(index + 1)
	if bitsIdx == 0 {
		return answer
	}

	return answer + bits.OnesCount64(b[wordIdx]<<(64-bitsIdx))
}

func popcntSlice(s []uint64) int {
	var cnt int
	for _, x := range s {
		cnt += bits.OnesCount64(x)
	}
	return cnt
}

func popcntAndSlice(s, m []uint64) int {
	var cnt int
	for i := range s {
		// panics if mask slice m is too short
		cnt += bits.OnesCount64(s[i] & m[i])
	}
	return cnt
}
