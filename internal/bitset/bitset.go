// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Package bitset implements bitsets, a mapping
// between non-negative integers and boolean values.
//
// Studied [github.com/bits-and-blooms/bitset] inside out
// and rewrote it from scratch for the needs of this project.
package bitset

import "math/bits"

// A BitSet is a slice of words. This is an internal package
// with a wide open public API.
type BitSet []uint64

// xIdx calculates the index of i in a []uint64
// func wIdx(i uint) int {
// 	return int(i >> 6) // like (i / 64) but faster
// }

// bIdx calculates the index of i in a `uint64`
// func bIdx(i uint) uint {
// 	return i & 63 // like (i % 64) but faster
// }

// Set bit i to 1, the capacity of the bitset is increased accordingly.
func (b BitSet) Set(i uint) BitSet {
	// grow?
	if i >= uint(len(b)<<6) {
		words := int((i + 64) >> 6)
		switch {
		case b == nil:
			b = make([]uint64, words)
		case cap(b) >= words:
			b = b[:words]
		default:
			newset := make([]uint64, words)
			copy(newset, b)
			b = newset
		}
	}

	b[i>>6] |= 1 << (i & 63)
	return b
}

// Clear bit i to 0.
func (b BitSet) Clear(i uint) BitSet {
	if x := int(i >> 6); x < len(b) {
		b[x] &^= 1 << (i & 63)
	}
	return b
}

// Test if bit i is set.
func (b BitSet) Test(i uint) bool {
	if x := int(i >> 6); x < len(b) {
		return b[x]&(1<<(i&63)) != 0
	}
	return false
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

// Compact, preserve all set bits, while minimizing memory usage.
// A new slice is allocated to store the new bits.
func (b BitSet) Compact() BitSet {
	last := len(b) - 1

	// find last word with at least one bit set.
	for ; last >= 0; last-- {
		if b[last] != 0 {
			newset := make([]uint64, last+1)
			copy(newset, b[:last+1])
			b = newset
			return b
		}
	}

	// BitSet was empty, shrink to nil
	return nil
}

// NextSet returns the next bit set from the specified index,
// including possibly the current index along with an ok code.
func (b BitSet) NextSet(i uint) (uint, bool) {
	x := int(i >> 6)
	if x >= len(b) {
		return 0, false
	}

	// process the first (maybe partial) first
	first := b[x] >> (i & 63) // i % 64
	if first != 0 {
		return i + uint(bits.TrailingZeros64(first)), true
	}

	// process the following words until next bit is set
	// x < len(b), no out-of-bounds panic in following slice expression
	x++
	for j, word := range b[x:] {
		if word != 0 {
			return uint((x+j)<<6 + bits.TrailingZeros64(word)), true
		}
	}
	return 0, false
}

// AsSlice returns all set bits as slice of uint without
// heap allocations.
//
// This is faster than AppendTo, but also more dangerous,
// it panics if the capacity of buf is < b.Size()
func (b BitSet) AsSlice(buf []uint) []uint {
	buf = buf[:cap(buf)] // len = cap

	size := 0
	for idx, word := range b {
		for ; word != 0; size++ {
			// panics if capacity of buf is exceeded.
			buf[size] = uint(idx<<6 + bits.TrailingZeros64(word))

			// clear the rightmost set bit
			word &= word - 1
		}
	}

	buf = buf[:size]
	return buf
}

// AppendTo appends all set bits to buf and returns the (maybe extended) buf.
// If the capacity of buf is < b.Size() new memory is allocated.
func (b BitSet) AppendTo(buf []uint) []uint {
	for idx, word := range b {
		for word != 0 {
			buf = append(buf, uint(idx<<6+bits.TrailingZeros64(word)))

			// clear the rightmost set bit
			word &= word - 1
		}
	}

	return buf
}

// IntersectionCardinality computes the popcount of the intersection.
func (b BitSet) IntersectionCardinality(c BitSet) int {
	return popcountAnd(b, c)
}

// InPlaceIntersection overwrites and computes the intersection of
// base set with the compare set. This is the BitSet equivalent of & (and).
// If len(c) > len(b), new memory is allocated.
func (b *BitSet) InPlaceIntersection(c BitSet) {
	// bounds check eliminated, range until minLen(b,c)
	for i := 0; i < len(*b) && i < len(c); i++ {
		(*b)[i] &= c[i]
	}

	// b >= c
	if len(*b) >= len(c) {
		// bounds check eliminated
		for i := len(c); i < len(*b); i++ {
			(*b)[i] = 0
		}
		return
	}

	// b < c
	newset := make([]uint64, len(c))
	copy(newset, *b)
	*b = newset
}

// InPlaceUnion creates the destructive union of base set with compare set.
// This is the BitSet equivalent of | (or).
// If len(c) > len(b), new memory is allocated.
func (b *BitSet) InPlaceUnion(c BitSet) {
	// b >= c
	if len(*b) >= len(c) {
		// bounds check eliminated
		for i := 0; i < len(*b) && i < len(c); i++ {
			(*b)[i] |= c[i]
		}

		return
	}

	// b < c
	newset := make([]uint64, len(c))
	copy(newset, *b)
	*b = newset

	// bounds check eliminated
	for i := 0; i < len(*b) && i < len(c); i++ {
		(*b)[i] |= c[i]
	}
}

// Size (number of set bits).
func (b BitSet) Size() int {
	return popcount(b)
}

// Rank returns the number of set bits up to and including the index
// that are set in the bitset.
func (b BitSet) Rank(i uint) int {
	// inlined popcount to make Rank inlineable
	var rnk int

	i++ // Rank count is inclusive
	wordIdx := i >> 6
	bitsIdx := i & 63

	if int(wordIdx) >= len(b) {
		// inlined popcount, whole slice
		for _, x := range b {
			rnk += bits.OnesCount64(x)
		}
		return rnk
	}

	// inlined popcount, partial slice
	for _, x := range b[:wordIdx] {
		rnk += bits.OnesCount64(x)
	}

	if bitsIdx == 0 {
		return rnk
	}

	// plus partial word
	return rnk + bits.OnesCount64(b[wordIdx]<<(64-bitsIdx))
}

// popcount
func popcount(s []uint64) int {
	var cnt int
	for _, x := range s {
		// count all the bits set in slice.
		cnt += bits.OnesCount64(x)
	}
	return cnt
}

// popcountAnd
func popcountAnd(s, m []uint64) int {
	var cnt int
	for j := 0; j < len(s) && j < len(m); j++ {
		// words are bitwise & followed by popcount.
		cnt += bits.OnesCount64(s[j] & m[j])
	}
	return cnt
}
