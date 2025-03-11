// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Package bitset implements bitsets, a mapping
// between non-negative integers up to 255 and boolean values.
//
// Studied [github.com/bits-and-blooms/bitset] inside out
// and rewrote needed parts from scratch for this project.
//
// This implementation is optimized for the needed use case.
package bitset

//  can inline (*BitSet256).All with cost 56
//  can inline (*BitSet256).AsSlice with cost 50
//  can inline (*BitSet256).Clear with cost 12
//  can inline (*BitSet256).FirstSet with cost 79
//  can inline (*BitSet256).IntersectionCardinality with cost 57
//  can inline (*BitSet256).IntersectionTop with cost 42
//  can inline (*BitSet256).Intersection with cost 53
//  can inline (*BitSet256).IntersectsAny with cost 48
//  can inline (*BitSet256).IsEmpty with cost 28
//  can inline (*BitSet256).NextSet with cost 73
//  can inline (*BitSet256).popcnt with cost 33
//  can inline (*BitSet256).Rank0 with cost 50
//  can inline (*BitSet256).Set with cost 12
//  can inline (*BitSet256).Size with cost 36
//  can inline (*BitSet256).Test with cost 28
//  can inline (*BitSet256).Union with cost 53

import (
	"math/bits"
)

//   wordIdx calculates the wordIndex of bit i in a []uint64
//   func wordIdx(i uint) int {
//   	return int(i >> 6) // like (i / 64) but faster
//   }

//   bitIdx calculates the bitIndex of i in an `uint64`
//   func bitIdx(i uint) uint {
//   	return i & 63 // like (i % 64) but mostly faster
//   }
//
// just as an explanation of the expressions,
//
//   i>>6 or i<<6 and i&63
//
// not factored out as functions to make most of the methods
// inlineable with minimal costs.

// BitSet256 represents a fixed size bitset from [0..255]
type BitSet256 [4]uint64

// Set the bit, must panic if bit is > 255 by intention!
func (b *BitSet256) Set(bit uint) {
	b[bit>>6] |= 1 << (bit & 63)
}

// Clear the bit, must panic if bit is > 255 by intention!
func (b *BitSet256) Clear(bit uint) {
	b[bit>>6] &^= 1 << (bit & 63)
}

// Test if bit is set.
func (b *BitSet256) Test(bit uint) (ok bool) {
	if x := int(bit >> 6); x < 4 {
		return b[x&3]&(1<<(bit&63)) != 0 // [x&3] is bounds check elimination (BCE)
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

// NextSet returns the next bit set from the specified start bit,
// including possibly the current bit along with an ok code.
func (b *BitSet256) NextSet(bit uint) (uint, bool) {
	wIdx := int(bit >> 6)
	if wIdx >= 4 {
		return 0, false
	}
	// wIdx is < 4

	// process the first (maybe partial) word
	first := b[wIdx&3] >> (bit & 63) // i % 64
	if first != 0 {
		return bit + uint(bits.TrailingZeros64(first)), true
	}

	// process the following words until next bit is set
	wIdx++ // wIdx is <= 4
	for jIdx, word := range b[wIdx:] {
		if word != 0 {
			return uint((wIdx+jIdx)<<6 + bits.TrailingZeros64(word)), true
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
	for wIdx, word := range b {
		for ; word != 0; size++ {
			// panics if capacity of buf is exceeded.
			buf[size] = uint(wIdx<<6 + bits.TrailingZeros64(word))

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

// IntersectionTop computes the intersection of base set with the compare set.
// If the result set isn't empty, it returns the top most set bit and true.
func (b *BitSet256) IntersectionTop(c *BitSet256) (top uint, ok bool) {
	for wIdx := 4 - 1; wIdx >= 0; wIdx-- {
		if word := b[wIdx] & c[wIdx]; word != 0 {
			return uint(wIdx<<6+bits.Len64(word)) - 1, true
		}
	}
	return
}

// Rank0 is equal to Rank(idx) - 1
func (b *BitSet256) Rank0(idx uint) (rnk int) {
	idx++ // Rank count is inclusive
	wIdx := min(4, int(idx>>6))

	// sum up the popcounts until wIdx ...
	// don't test x == 0, just add, less branches
	for jIdx := range wIdx {
		rnk += bits.OnesCount64(b[jIdx])
	}

	// ... plus partial word at wIdx,
	if wIdx < 4 {
		rnk += bits.OnesCount64(b[wIdx&3] << (64 - idx&63)) // with BCE
	}

	// decrement for offset by one
	rnk--
	return
}

// IsEmpty returns true if no bit is set.
func (b *BitSet256) IsEmpty() bool {
	return b[0] == 0 &&
		b[1] == 0 &&
		b[2] == 0 &&
		b[3] == 0
}

// IntersectsAny returns true if the intersection of base set with the compare set
// is not the empty set.
func (b *BitSet256) IntersectsAny(c *BitSet256) bool {
	return b[0]&c[0] != 0 ||
		b[1]&c[1] != 0 ||
		b[2]&c[2] != 0 ||
		b[3]&c[3] != 0
}

// Intersection computes the intersection of base set with the compare set.
// This is the BitSet equivalent of & (and).
func (b *BitSet256) Intersection(c *BitSet256) (bs BitSet256) {
	bs[0] = b[0] & c[0]
	bs[1] = b[1] & c[1]
	bs[2] = b[2] & c[2]
	bs[3] = b[3] & c[3]
	return
}

// Union creates the union of base set with compare set.
// This is the BitSet equivalent of | (or).
func (b *BitSet256) Union(c *BitSet256) (bs BitSet256) {
	bs[0] = b[0] | c[0]
	bs[1] = b[1] | c[1]
	bs[2] = b[2] | c[2]
	bs[3] = b[3] | c[3]
	return
}

// IntersectionCardinality computes the popcount of the intersection.
func (b *BitSet256) IntersectionCardinality(c *BitSet256) (cnt int) {
	cnt += bits.OnesCount64(b[0] & c[0])
	cnt += bits.OnesCount64(b[1] & c[1])
	cnt += bits.OnesCount64(b[2] & c[2])
	cnt += bits.OnesCount64(b[3] & c[3])
	return
}

// Size is the number of set bits (popcount).
func (b *BitSet256) Size() int {
	return b.popcnt()
}

// popcnt, count all the set bits
func (b *BitSet256) popcnt() (cnt int) {
	cnt += bits.OnesCount64(b[0])
	cnt += bits.OnesCount64(b[1])
	cnt += bits.OnesCount64(b[2])
	cnt += bits.OnesCount64(b[3])
	return
}
