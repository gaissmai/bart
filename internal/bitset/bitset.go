// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bitset

//  can inline (*BitSet256).IsEmpty with cost 28
//  can inline (*BitSet256).Set with cost 12
//  can inline (*BitSet256).Clear with cost 22
//  can inline (*BitSet256).Test with cost 26
//  can inline (*BitSet256).FirstSet with cost 79
//  can inline (*BitSet256).NextSet with cost 71
//  can inline (*BitSet256).AsSlice with cost 50
//  can inline (*BitSet256).All with cost 56
//  can inline (*BitSet256).IntersectsAny with cost 48
//  can inline (*BitSet256).IntersectionTop with cost 42
//  can inline (*BitSet256).popcntAnd with cost 53
//  can inline (*BitSet256).IntersectionCardinality with cost 57
//  can inline (*BitSet256).InPlaceIntersection with cost 36
//  can inline (*BitSet256).InPlaceUnion with cost 36
//  can inline (*BitSet256).popcnt with cost 33
//  can inline (*BitSet256).Size with cost 36
//  can inline (*BitSet256).Rank0 with cost 52

import (
	"math/bits"
)

type BitSet256 [4]uint64

// IsEmpty returns true if no bit is set.
func (b *BitSet256) IsEmpty() bool {
	return b[0] == 0 && b[1] == 0 && b[2] == 0 && b[3] == 0
}

// Set bit i to 1, the capacity of the bitset is increased accordingly.
// panics if i is > 255
func (b *BitSet256) Set(i uint) {
	b[i>>6] |= 1 << (i & 63)
}

// Clear bit i to 0.
func (b *BitSet256) Clear(i uint) {
	if x := int(i >> 6); x < 4 {
		b[x] &^= 1 << (i & 63)
	}
}

// Test if bit i is set.
func (b *BitSet256) Test(i uint) (ok bool) {
	if x := int(i >> 6); x < 4 {
		return b[x]&(1<<(i&63)) != 0
	}
	return
}

// FirstSet returns the first bit set along with an ok code.
func (b *BitSet256) FirstSet() (first uint, ok bool) {
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
func (b *BitSet256) NextSet(i uint) (uint, bool) {
	x := int(i >> 6)
	if x >= 4 {
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
func (b *BitSet256) AsSlice(buf []uint) []uint {
	buf = buf[:cap(buf)] // use cap as max len

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
func (b *BitSet256) All() []uint {
	return b.AsSlice(make([]uint, 0, 256))
}

// IntersectsAny returns true if the intersection of base set with the compare set
// is not the empty set.
func (b *BitSet256) IntersectsAny(c *BitSet256) bool {
	return b[0]&c[0] != 0 ||
		b[1]&c[1] != 0 ||
		b[2]&c[2] != 0 ||
		b[3]&c[3] != 0
}

// IntersectionTop computes the intersection of base set with the compare set.
// If the result set isn't empty, it returns the top most set bit and true.
func (b *BitSet256) IntersectionTop(c *BitSet256) (top uint, ok bool) {
	for i := 4 - 1; i >= 0; i-- {
		if word := b[i] & c[i]; word != 0 {
			return uint(i<<6+bits.Len64(word)) - 1, true
		}
	}
	return
}

// IntersectionCardinality computes the popcount of the intersection.
func (b *BitSet256) IntersectionCardinality(c *BitSet256) int {
	return b.popcntAnd(c)
}

// InPlaceIntersection overwrites and computes the intersection of
// base set with the compare set. This is the BitSet equivalent of & (and).
func (b *BitSet256) InPlaceIntersection(c *BitSet256) {
	b[0] &= c[0]
	b[1] &= c[1]
	b[2] &= c[2]
	b[3] &= c[3]
}

// InPlaceUnion creates the destructive union of base set with compare set.
// This is the BitSet equivalent of | (or).
func (b *BitSet256) InPlaceUnion(c *BitSet256) {
	b[0] |= c[0]
	b[1] |= c[1]
	b[2] |= c[2]
	b[3] |= c[3]
}

// Size (number of set bits).
func (b *BitSet256) Size() int {
	return b.popcnt()
}

// Rank0 is equal to Rank(i) - 1
func (b *BitSet256) Rank0(i uint) (rnk int) {
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

func (b *BitSet256) popcnt() (cnt int) {
	cnt += bits.OnesCount64(b[0])
	cnt += bits.OnesCount64(b[1])
	cnt += bits.OnesCount64(b[2])
	cnt += bits.OnesCount64(b[3])
	return
}

func (b *BitSet256) popcntAnd(c *BitSet256) (cnt int) {
	cnt += bits.OnesCount64(b[0] & c[0])
	cnt += bits.OnesCount64(b[1] & c[1])
	cnt += bits.OnesCount64(b[2] & c[2])
	cnt += bits.OnesCount64(b[3] & c[3])
	return
}
