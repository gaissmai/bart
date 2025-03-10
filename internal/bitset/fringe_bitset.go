// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bitset

//  can inline (*BitSetFringe).IsEmpty with cost 28
//  can inline (*BitSetFringe).Set with cost 12
//  can inline (*BitSetFringe).Clear with cost 22
//  can inline (*BitSetFringe).Test with cost 26
//  can inline (*BitSetFringe).FirstSet with cost 79
//  can inline (*BitSetFringe).NextSet with cost 71
//  can inline (*BitSetFringe).AsSlice with cost 50
//  can inline (*BitSetFringe).popcnt with cost 33
//  can inline (*BitSetFringe).All with cost 71
//  can inline (*BitSetFringe).IntersectsAny with cost 48
//  can inline (*BitSetFringe).IntersectionTop with cost 42
//  can inline (*BitSetFringe).IntersectionCardinality with cost 57
//  can inline (*BitSetFringe).InPlaceIntersection with cost 21
//  can inline (*BitSetFringe).InPlaceUnion with cost 21
//  can inline (*BitSetFringe).Size with cost 36
//  can inline (*BitSetFringe).Rank0 with cost 51
//  can inline (*BitSetFringe).popcntAnd with cost 53

import (
	"math/bits"
)

const length = 4

type BitSetFringe [length]uint64

// IsEmpty returns true if no bit is set.
func (b *BitSetFringe) IsEmpty() bool {
	return b[0] == 0 && b[1] == 0 && b[2] == 0 && b[3] == 0
}

// Set bit i to 1, the capacity of the bitset is increased accordingly.
// panics if i is > 255
func (b *BitSetFringe) Set(i uint) {
	b[i>>6] |= 1 << (i & 63)
}

// Clear bit i to 0.
func (b *BitSetFringe) Clear(i uint) {
	if x := int(i >> 6); x < length {
		b[x] &^= 1 << (i & 63)
	}
}

// Test if bit i is set.
func (b *BitSetFringe) Test(i uint) (ok bool) {
	if x := int(i >> 6); x < length {
		return b[x]&(1<<(i&63)) != 0
	}
	return
}

// FirstSet returns the first bit set along with an ok code.
func (b *BitSetFringe) FirstSet() (first uint, ok bool) {
	// optimized for pipelining, can still inline with cost 79
	if x := bits.TrailingZeros64(b[0]); x != 64 {
		return uint(x), true
	} else if x := bits.TrailingZeros64(b[1]); x != 64 {
		return uint(x + 64), true
	} else if x := bits.TrailingZeros64(b[2]); x != 64 {
		return uint(x + 128), true
	} else if x := bits.TrailingZeros64(b[3]); x != 64 {
		return uint(x + 192), true
	}
	return
}

// NextSet returns the next bit set from the specified index,
// including possibly the current index along with an ok code.
func (b *BitSetFringe) NextSet(i uint) (uint, bool) {
	x := int(i >> 6)
	if x >= length {
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
func (b *BitSetFringe) AsSlice(buf []uint) []uint {
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
func (b *BitSetFringe) All() []uint {
	return b.AsSlice(make([]uint, 0, popcntSlice(b[:])))
}

// IntersectsAny returns true if the intersection of base set with the compare set
// is not the empty set.
func (b *BitSetFringe) IntersectsAny(c *BitSetFringe) bool {
	return b[0]&c[0] != 0 ||
		b[1]&c[1] != 0 ||
		b[2]&c[2] != 0 ||
		b[3]&c[3] != 0
}

// IntersectionTop computes the intersection of base set with the compare set.
// If the result set isn't empty, it returns the top most set bit and true.
func (b *BitSetFringe) IntersectionTop(c *BitSetFringe) (top uint, ok bool) {
	for i := length - 1; i >= 0; i-- {
		if word := b[i] & c[i]; word != 0 {
			return uint(i<<6+bits.Len64(word)) - 1, true
		}
	}
	return
}

// IntersectionCardinality computes the popcount of the intersection.
func (b *BitSetFringe) IntersectionCardinality(c *BitSetFringe) int {
	return b.popcntAnd(c)
}

// InPlaceIntersection overwrites and computes the intersection of
// base set with the compare set. This is the BitSet equivalent of & (and).
// If len(c) > len(b), new memory is allocated.
func (b *BitSetFringe) InPlaceIntersection(c *BitSetFringe) {
	for i := length - 1; i >= 0; i-- {
		b[i] &= c[i]
	}
}

// InPlaceUnion creates the destructive union of base set with compare set.
// This is the BitSet equivalent of | (or).
func (b *BitSetFringe) InPlaceUnion(c *BitSetFringe) {
	for i := length - 1; i >= 0; i-- {
		b[i] |= c[i]
	}
}

// Size (number of set bits).
func (b *BitSetFringe) Size() int {
	return b.popcnt()
}

// Rank0 is equal to Rank(i) - 1
func (b *BitSetFringe) Rank0(i uint) (rnk int) {
	i++ // Rank count is inclusive
	wordIdx := min(int(i>>6), len(b))

	// sum up the popcounts until wordIdx ...
	// don't test x == 0, just add, less branches
	for j := range wordIdx {
		rnk += bits.OnesCount64(b[j&3]) // [j&3] is BCE
	}

	// ... plus partial word at wordIdx,
	// don't test i&63 != 0, just add, less branches
	if wordIdx < len(b) {
		rnk += bits.OnesCount64(b[wordIdx&3] << (64 - i&63)) // [x&3] is BCE
	}

	// decrement for offset by one
	rnk--
	return
}

func (b *BitSetFringe) popcnt() (cnt int) {
	cnt += bits.OnesCount64(b[0])
	cnt += bits.OnesCount64(b[1])
	cnt += bits.OnesCount64(b[2])
	cnt += bits.OnesCount64(b[3])
	return
}

func (b *BitSetFringe) popcntAnd(c *BitSetFringe) (cnt int) {
	cnt += bits.OnesCount64(b[0] & c[0])
	cnt += bits.OnesCount64(b[1] & c[1])
	cnt += bits.OnesCount64(b[2] & c[2])
	cnt += bits.OnesCount64(b[3] & c[3])
	return
}
