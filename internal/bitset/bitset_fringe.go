// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bitset

import (
	"math/bits"
)

const words = 4

type BitSetFringe [words]uint64

// IsEmpty returns true if no bit is set.
func (b BitSetFringe) IsEmpty() bool {
	for _, w := range b {
		if w != 0 {
			return false
		}
	}
	return true
}

// Set bit i to 1, the capacity of the bitset is increased accordingly.
// panics if i is > 255
func (b BitSetFringe) Set(i uint) BitSetFringe {
	b[i>>6] |= 1 << (i & 63)
	return b
}

// Clear bit i to 0.
func (b BitSetFringe) Clear(i uint) BitSetFringe {
	if x := int(i >> 6); x < len(b) {
		b[x] &^= 1 << (i & 63)
	}
	return b
}

// Test if bit i is set.
func (b BitSetFringe) Test(i uint) (ok bool) {
	if x := int(i >> 6); x < len(b) {
		return b[x]&(1<<(i&63)) != 0
	}
	return
}

// Clone this BitSet, returning a new BitSet that has the same bits set.
func (b BitSetFringe) Clone() BitSetFringe {
	return b
}

// FirstSet returns the first bit set along with an ok code.
func (b BitSetFringe) FirstSet() (uint, bool) {
	for x, word := range b {
		if word != 0 {
			return uint(x<<6 + bits.TrailingZeros64(word)), true
		}
	}
	return 0, false
}

// NextSet returns the next bit set from the specified index,
// including possibly the current index along with an ok code.
func (b BitSetFringe) NextSet(i uint) (uint, bool) {
	x := int(i >> 6)
	if x >= len(b) {
		return 0, false
	}

	// process the first (maybe partial) word
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
// This is faster than All, but also more dangerous,
// it panics if the capacity of buf is < b.Size()
func (b BitSetFringe) AsSlice(buf []uint) []uint {
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

// All returns all set bits. This has a simpler API but is slower than AsSlice.
func (b BitSetFringe) All() []uint {
	return b.AsSlice(make([]uint, 0, popcntSlice(b[:])))
}

// IntersectsAny returns true if the intersection of base set with the compare set
// is not the empty set.
func (b BitSetFringe) IntersectsAny(c BitSetFringe) bool {
	for i := words - 1; i >= 0; i-- {
		if b[i]&c[i] != 0 {
			return true
		}
	}
	return false
}

// IntersectionTop computes the intersection of base set with the compare set.
// If the result set isn't empty, it returns the top most set bit and true.
func (b BitSetFringe) IntersectionTop(c BitSetFringe) (top uint, ok bool) {
	for i := words - 1; i >= 0; i-- {
		if word := b[i] & c[i]; word != 0 {
			return uint(i<<6+bits.Len64(word)) - 1, true
		}
	}
	return
}

// IntersectionCardinality computes the popcount of the intersection.
func (b BitSetFringe) IntersectionCardinality(c BitSetFringe) int {
	return popcntAnd(b[:], c[:])
}

// InPlaceIntersection overwrites and computes the intersection of
// base set with the compare set. This is the BitSet equivalent of & (and).
// If len(c) > len(b), new memory is allocated.
func (b *BitSetFringe) InPlaceIntersection(c BitSetFringe) {
	// bounds check eliminated, range until minLen(b,c)
	for i := words - 1; i >= 0; i-- {
		(*b)[i] &= c[i]
	}
}

// InPlaceUnion creates the destructive union of base set with compare set.
// This is the BitSet equivalent of | (or).
// If len(c) > len(b), new memory is allocated.
func (b *BitSetFringe) InPlaceUnion(c BitSetFringe) {
	for i := words - 1; i >= 0; i-- {
		(*b)[i] |= c[i]
	}
}

// Size (number of set bits).
func (b BitSetFringe) Size() int {
	return popcntSlice(b[:])
}

// Rank0 is equal to Rank(i) - 1
//
// With inlined popcount to make Rank0 itself inlineable.
func (b BitSetFringe) Rank0(i uint) (rnk int) {
	// Rank count is inclusive
	i++

	if wordIdx := int(i >> 6); wordIdx >= len(b) {
		// inlined popcount, whole slice
		for _, x := range b {
			// don't test for x != 0, less branches
			rnk += bits.OnesCount64(x)
		}
	} else {
		// inlined popcount, partial slice ...
		for _, x := range b[:wordIdx] {
			rnk += bits.OnesCount64(x)
		}

		// ... plus partial word, unconditional
		// don't test i&63 != 0, less branches
		rnk += bits.OnesCount64(b[wordIdx] << (64 - i&63))
	}

	// correct for offset by one
	return rnk - 1
}
