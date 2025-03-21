// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Package bitset implements bitsets, a mapping
// between non-negative integers and boolean values.
//
// Studied [github.com/bits-and-blooms/bitset] inside out
// and rewrote needed parts from scratch for this project.
//
// This implementation is heavily optimized for this internal use case.
package bitset

//  can inline (*BitSet256).All with cost 56
//  can inline (*BitSet256).AsSlice with cost 50
//  can inline (*BitSet256).FirstSet with cost 79
//  can inline (*BitSet256).IntersectionCardinality with cost 53
//  can inline (*BitSet256).IntersectionTop with cost 42
//  can inline (*BitSet256).Intersection with cost 53
//  can inline (*BitSet256).IntersectsAny with cost 48
//  can inline (*BitSet256).IsEmpty with cost 28
//  can inline (*BitSet256).MustClear with cost 12
//  can inline (*BitSet256).MustSet with cost 12
//  can inline (*BitSet256).NextSet with cost 73
//  can inline (*BitSet256).popcnt with cost 33
//  can inline (*BitSet256).Rank0 with cost 64
//  can inline (*BitSet256).Size with cost 36
//  can inline (*BitSet256).Test with cost 28
//  can inline (*BitSet256).Union with cost 53
//  cannot inline (*BitSet256).String: function too complex: cost 121 exceeds budget 80

import (
	"fmt"
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

func (b *BitSet256) String() string {
	return fmt.Sprint(b.All())
}

// MustSet sets the bit, it panic's if bit is > 255 by intention!
func (b *BitSet256) MustSet(bit uint) {
	b[bit>>6] |= 1 << (bit & 63)
}

// MustClear clear the bit, it panic's if bit is > 255 by intention!
func (b *BitSet256) MustClear(bit uint) {
	b[bit>>6] &^= 1 << (bit & 63)
}

// Test if the bit is set.
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

// Rank0 returns the set bits up to and including to idx, minus 1.
//
// Rank0 is used in the hot path, if idx > 255 it does NOT PANIC,
// the bounds check is eliminated, we press the pedal to the metal!
func (b *BitSet256) Rank0(idx uint) (rnk int) {
	rnk += bits.OnesCount64(b[0] & rankMask[uint8(idx)][0]) // uint8() is BCE
	rnk += bits.OnesCount64(b[1] & rankMask[uint8(idx)][1])
	rnk += bits.OnesCount64(b[2] & rankMask[uint8(idx)][2])
	rnk += bits.OnesCount64(b[3] & rankMask[uint8(idx)][3])

	// Rank0: if bit 0 is set, Rank(0) is 1 (up to and including), but
	// here it's only used as slice index, so we decrement it already by 1.
	rnk--
	return
}

/*
// Rank0 is equal to Rank(idx) - 1
func (b *BitSet256) Rank0(idx uint) (rnk int) {
	// Rank count is inclusive
	idx++
	wIdx := min(4, int((idx)>>6))

	// sum up the popcounts until wIdx ...
	// don't test x == 0, just add, less branches
	for jIdx := range wIdx {
		rnk += bits.OnesCount64(b[jIdx])
	}

	// ... plus partial word at wIdx,
	if wIdx < 4 {
		rnk += bits.OnesCount64(b[wIdx&3] << (64 - idx&63))
	}

	// decrement for offset by one
	rnk--
	return
}
*/

// IsEmpty returns true if no bit is set.
func (b *BitSet256) IsEmpty() bool {
	return b[3] == 0 &&
		b[2] == 0 &&
		b[1] == 0 &&
		b[0] == 0
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

// rankMask, all 1 until and including bit pos, the rest is zero, example:
//
//	bs.Rank(7) = popcnt(bs & rankMask[7]) and rankMask[7] = 0000...0000_1111_1111
var rankMask = [256]BitSet256{
	/*   0 */ {0x1, 0x0, 0x0, 0x0}, // 256 bits: 0000...0_0001
	/*   1 */ {0x3, 0x0, 0x0, 0x0}, // 256 bits: 0000...0_0011
	/*   2 */ {0x7, 0x0, 0x0, 0x0}, // 256 bits: 0000...0_0111
	/*   3 */ {0xf, 0x0, 0x0, 0x0}, // 256 bits: 0000...0_1111
	/*   4 */ {0x1f, 0x0, 0x0, 0x0}, // ...
	/*   5 */ {0x3f, 0x0, 0x0, 0x0},
	/*   6 */ {0x7f, 0x0, 0x0, 0x0},
	/*   7 */ {0xff, 0x0, 0x0, 0x0},
	/*   8 */ {0x1ff, 0x0, 0x0, 0x0},
	/*   9 */ {0x3ff, 0x0, 0x0, 0x0},
	/*  10 */ {0x7ff, 0x0, 0x0, 0x0},
	/*  11 */ {0xfff, 0x0, 0x0, 0x0},
	/*  12 */ {0x1fff, 0x0, 0x0, 0x0},
	/*  13 */ {0x3fff, 0x0, 0x0, 0x0},
	/*  14 */ {0x7fff, 0x0, 0x0, 0x0},
	/*  15 */ {0xffff, 0x0, 0x0, 0x0},
	/*  16 */ {0x1ffff, 0x0, 0x0, 0x0},
	/*  17 */ {0x3ffff, 0x0, 0x0, 0x0},
	/*  18 */ {0x7ffff, 0x0, 0x0, 0x0},
	/*  19 */ {0xfffff, 0x0, 0x0, 0x0},
	/*  20 */ {0x1fffff, 0x0, 0x0, 0x0},
	/*  21 */ {0x3fffff, 0x0, 0x0, 0x0},
	/*  22 */ {0x7fffff, 0x0, 0x0, 0x0},
	/*  23 */ {0xffffff, 0x0, 0x0, 0x0},
	/*  24 */ {0x1ffffff, 0x0, 0x0, 0x0},
	/*  25 */ {0x3ffffff, 0x0, 0x0, 0x0},
	/*  26 */ {0x7ffffff, 0x0, 0x0, 0x0},
	/*  27 */ {0xfffffff, 0x0, 0x0, 0x0},
	/*  28 */ {0x1fffffff, 0x0, 0x0, 0x0},
	/*  29 */ {0x3fffffff, 0x0, 0x0, 0x0},
	/*  30 */ {0x7fffffff, 0x0, 0x0, 0x0},
	/*  31 */ {0xffffffff, 0x0, 0x0, 0x0},
	/*  32 */ {0x1ffffffff, 0x0, 0x0, 0x0},
	/*  33 */ {0x3ffffffff, 0x0, 0x0, 0x0},
	/*  34 */ {0x7ffffffff, 0x0, 0x0, 0x0},
	/*  35 */ {0xfffffffff, 0x0, 0x0, 0x0},
	/*  36 */ {0x1fffffffff, 0x0, 0x0, 0x0},
	/*  37 */ {0x3fffffffff, 0x0, 0x0, 0x0},
	/*  38 */ {0x7fffffffff, 0x0, 0x0, 0x0},
	/*  39 */ {0xffffffffff, 0x0, 0x0, 0x0},
	/*  40 */ {0x1ffffffffff, 0x0, 0x0, 0x0},
	/*  41 */ {0x3ffffffffff, 0x0, 0x0, 0x0},
	/*  42 */ {0x7ffffffffff, 0x0, 0x0, 0x0},
	/*  43 */ {0xfffffffffff, 0x0, 0x0, 0x0},
	/*  44 */ {0x1fffffffffff, 0x0, 0x0, 0x0},
	/*  45 */ {0x3fffffffffff, 0x0, 0x0, 0x0},
	/*  46 */ {0x7fffffffffff, 0x0, 0x0, 0x0},
	/*  47 */ {0xffffffffffff, 0x0, 0x0, 0x0},
	/*  48 */ {0x1ffffffffffff, 0x0, 0x0, 0x0},
	/*  49 */ {0x3ffffffffffff, 0x0, 0x0, 0x0},
	/*  50 */ {0x7ffffffffffff, 0x0, 0x0, 0x0},
	/*  51 */ {0xfffffffffffff, 0x0, 0x0, 0x0},
	/*  52 */ {0x1fffffffffffff, 0x0, 0x0, 0x0},
	/*  53 */ {0x3fffffffffffff, 0x0, 0x0, 0x0},
	/*  54 */ {0x7fffffffffffff, 0x0, 0x0, 0x0},
	/*  55 */ {0xffffffffffffff, 0x0, 0x0, 0x0},
	/*  56 */ {0x1ffffffffffffff, 0x0, 0x0, 0x0},
	/*  57 */ {0x3ffffffffffffff, 0x0, 0x0, 0x0},
	/*  58 */ {0x7ffffffffffffff, 0x0, 0x0, 0x0},
	/*  59 */ {0xfffffffffffffff, 0x0, 0x0, 0x0},
	/*  60 */ {0x1fffffffffffffff, 0x0, 0x0, 0x0},
	/*  61 */ {0x3fffffffffffffff, 0x0, 0x0, 0x0},
	/*  62 */ {0x7fffffffffffffff, 0x0, 0x0, 0x0},
	/*  63 */ {0xffffffffffffffff, 0x0, 0x0, 0x0},
	/*  64 */ {0xffffffffffffffff, 0x1, 0x0, 0x0},
	/*  65 */ {0xffffffffffffffff, 0x3, 0x0, 0x0},
	/*  66 */ {0xffffffffffffffff, 0x7, 0x0, 0x0},
	/*  67 */ {0xffffffffffffffff, 0xf, 0x0, 0x0},
	/*  68 */ {0xffffffffffffffff, 0x1f, 0x0, 0x0},
	/*  69 */ {0xffffffffffffffff, 0x3f, 0x0, 0x0},
	/*  70 */ {0xffffffffffffffff, 0x7f, 0x0, 0x0},
	/*  71 */ {0xffffffffffffffff, 0xff, 0x0, 0x0},
	/*  72 */ {0xffffffffffffffff, 0x1ff, 0x0, 0x0},
	/*  73 */ {0xffffffffffffffff, 0x3ff, 0x0, 0x0},
	/*  74 */ {0xffffffffffffffff, 0x7ff, 0x0, 0x0},
	/*  75 */ {0xffffffffffffffff, 0xfff, 0x0, 0x0},
	/*  76 */ {0xffffffffffffffff, 0x1fff, 0x0, 0x0},
	/*  77 */ {0xffffffffffffffff, 0x3fff, 0x0, 0x0},
	/*  78 */ {0xffffffffffffffff, 0x7fff, 0x0, 0x0},
	/*  79 */ {0xffffffffffffffff, 0xffff, 0x0, 0x0},
	/*  80 */ {0xffffffffffffffff, 0x1ffff, 0x0, 0x0},
	/*  81 */ {0xffffffffffffffff, 0x3ffff, 0x0, 0x0},
	/*  82 */ {0xffffffffffffffff, 0x7ffff, 0x0, 0x0},
	/*  83 */ {0xffffffffffffffff, 0xfffff, 0x0, 0x0},
	/*  84 */ {0xffffffffffffffff, 0x1fffff, 0x0, 0x0},
	/*  85 */ {0xffffffffffffffff, 0x3fffff, 0x0, 0x0},
	/*  86 */ {0xffffffffffffffff, 0x7fffff, 0x0, 0x0},
	/*  87 */ {0xffffffffffffffff, 0xffffff, 0x0, 0x0},
	/*  88 */ {0xffffffffffffffff, 0x1ffffff, 0x0, 0x0},
	/*  89 */ {0xffffffffffffffff, 0x3ffffff, 0x0, 0x0},
	/*  90 */ {0xffffffffffffffff, 0x7ffffff, 0x0, 0x0},
	/*  91 */ {0xffffffffffffffff, 0xfffffff, 0x0, 0x0},
	/*  92 */ {0xffffffffffffffff, 0x1fffffff, 0x0, 0x0},
	/*  93 */ {0xffffffffffffffff, 0x3fffffff, 0x0, 0x0},
	/*  94 */ {0xffffffffffffffff, 0x7fffffff, 0x0, 0x0},
	/*  95 */ {0xffffffffffffffff, 0xffffffff, 0x0, 0x0},
	/*  96 */ {0xffffffffffffffff, 0x1ffffffff, 0x0, 0x0},
	/*  97 */ {0xffffffffffffffff, 0x3ffffffff, 0x0, 0x0},
	/*  98 */ {0xffffffffffffffff, 0x7ffffffff, 0x0, 0x0},
	/*  99 */ {0xffffffffffffffff, 0xfffffffff, 0x0, 0x0},
	/* 100 */ {0xffffffffffffffff, 0x1fffffffff, 0x0, 0x0},
	/* 101 */ {0xffffffffffffffff, 0x3fffffffff, 0x0, 0x0},
	/* 102 */ {0xffffffffffffffff, 0x7fffffffff, 0x0, 0x0},
	/* 103 */ {0xffffffffffffffff, 0xffffffffff, 0x0, 0x0},
	/* 104 */ {0xffffffffffffffff, 0x1ffffffffff, 0x0, 0x0},
	/* 105 */ {0xffffffffffffffff, 0x3ffffffffff, 0x0, 0x0},
	/* 106 */ {0xffffffffffffffff, 0x7ffffffffff, 0x0, 0x0},
	/* 107 */ {0xffffffffffffffff, 0xfffffffffff, 0x0, 0x0},
	/* 108 */ {0xffffffffffffffff, 0x1fffffffffff, 0x0, 0x0},
	/* 109 */ {0xffffffffffffffff, 0x3fffffffffff, 0x0, 0x0},
	/* 110 */ {0xffffffffffffffff, 0x7fffffffffff, 0x0, 0x0},
	/* 111 */ {0xffffffffffffffff, 0xffffffffffff, 0x0, 0x0},
	/* 112 */ {0xffffffffffffffff, 0x1ffffffffffff, 0x0, 0x0},
	/* 113 */ {0xffffffffffffffff, 0x3ffffffffffff, 0x0, 0x0},
	/* 114 */ {0xffffffffffffffff, 0x7ffffffffffff, 0x0, 0x0},
	/* 115 */ {0xffffffffffffffff, 0xfffffffffffff, 0x0, 0x0},
	/* 116 */ {0xffffffffffffffff, 0x1fffffffffffff, 0x0, 0x0},
	/* 117 */ {0xffffffffffffffff, 0x3fffffffffffff, 0x0, 0x0},
	/* 118 */ {0xffffffffffffffff, 0x7fffffffffffff, 0x0, 0x0},
	/* 119 */ {0xffffffffffffffff, 0xffffffffffffff, 0x0, 0x0},
	/* 120 */ {0xffffffffffffffff, 0x1ffffffffffffff, 0x0, 0x0},
	/* 121 */ {0xffffffffffffffff, 0x3ffffffffffffff, 0x0, 0x0},
	/* 122 */ {0xffffffffffffffff, 0x7ffffffffffffff, 0x0, 0x0},
	/* 123 */ {0xffffffffffffffff, 0xfffffffffffffff, 0x0, 0x0},
	/* 124 */ {0xffffffffffffffff, 0x1fffffffffffffff, 0x0, 0x0},
	/* 125 */ {0xffffffffffffffff, 0x3fffffffffffffff, 0x0, 0x0},
	/* 126 */ {0xffffffffffffffff, 0x7fffffffffffffff, 0x0, 0x0},
	/* 127 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x0, 0x0},
	/* 128 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x1, 0x0},
	/* 129 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x3, 0x0},
	/* 130 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x7, 0x0},
	/* 131 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xf, 0x0},
	/* 132 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x1f, 0x0},
	/* 133 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x3f, 0x0},
	/* 134 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x7f, 0x0},
	/* 135 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xff, 0x0},
	/* 136 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x1ff, 0x0},
	/* 137 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x3ff, 0x0},
	/* 138 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x7ff, 0x0},
	/* 139 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xfff, 0x0},
	/* 140 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x1fff, 0x0},
	/* 141 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x3fff, 0x0},
	/* 142 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x7fff, 0x0},
	/* 143 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffff, 0x0},
	/* 144 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x1ffff, 0x0},
	/* 145 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x3ffff, 0x0},
	/* 146 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x7ffff, 0x0},
	/* 147 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xfffff, 0x0},
	/* 148 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x1fffff, 0x0},
	/* 149 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x3fffff, 0x0},
	/* 150 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x7fffff, 0x0},
	/* 151 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffff, 0x0},
	/* 152 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x1ffffff, 0x0},
	/* 153 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x3ffffff, 0x0},
	/* 154 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x7ffffff, 0x0},
	/* 155 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xfffffff, 0x0},
	/* 156 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x1fffffff, 0x0},
	/* 157 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x3fffffff, 0x0},
	/* 158 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x7fffffff, 0x0},
	/* 159 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffff, 0x0},
	/* 160 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x1ffffffff, 0x0},
	/* 161 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x3ffffffff, 0x0},
	/* 162 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x7ffffffff, 0x0},
	/* 163 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xfffffffff, 0x0},
	/* 164 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x1fffffffff, 0x0},
	/* 165 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x3fffffffff, 0x0},
	/* 166 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x7fffffffff, 0x0},
	/* 167 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffff, 0x0},
	/* 168 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x1ffffffffff, 0x0},
	/* 169 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x3ffffffffff, 0x0},
	/* 170 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x7ffffffffff, 0x0},
	/* 171 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xfffffffffff, 0x0},
	/* 172 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x1fffffffffff, 0x0},
	/* 173 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x3fffffffffff, 0x0},
	/* 174 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x7fffffffffff, 0x0},
	/* 175 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffff, 0x0},
	/* 176 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x1ffffffffffff, 0x0},
	/* 177 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x3ffffffffffff, 0x0},
	/* 178 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x7ffffffffffff, 0x0},
	/* 179 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xfffffffffffff, 0x0},
	/* 180 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x1fffffffffffff, 0x0},
	/* 181 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x3fffffffffffff, 0x0},
	/* 182 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x7fffffffffffff, 0x0},
	/* 183 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffff, 0x0},
	/* 184 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x1ffffffffffffff, 0x0},
	/* 185 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x3ffffffffffffff, 0x0},
	/* 186 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x7ffffffffffffff, 0x0},
	/* 187 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xfffffffffffffff, 0x0},
	/* 188 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x1fffffffffffffff, 0x0},
	/* 189 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x3fffffffffffffff, 0x0},
	/* 190 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x7fffffffffffffff, 0x0},
	/* 191 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x0},
	/* 192 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x1},
	/* 193 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x3},
	/* 194 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x7},
	/* 195 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xf},
	/* 196 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x1f},
	/* 197 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x3f},
	/* 198 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x7f},
	/* 199 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xff},
	/* 200 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x1ff},
	/* 201 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x3ff},
	/* 202 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x7ff},
	/* 203 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xfff},
	/* 204 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x1fff},
	/* 205 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x3fff},
	/* 206 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x7fff},
	/* 207 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xffff},
	/* 208 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x1ffff},
	/* 209 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x3ffff},
	/* 210 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x7ffff},
	/* 211 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xfffff},
	/* 212 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x1fffff},
	/* 213 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x3fffff},
	/* 214 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x7fffff},
	/* 215 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xffffff},
	/* 216 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x1ffffff},
	/* 217 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x3ffffff},
	/* 218 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x7ffffff},
	/* 219 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xfffffff},
	/* 220 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x1fffffff},
	/* 221 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x3fffffff},
	/* 222 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x7fffffff},
	/* 223 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xffffffff},
	/* 224 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x1ffffffff},
	/* 225 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x3ffffffff},
	/* 226 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x7ffffffff},
	/* 227 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xfffffffff},
	/* 228 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x1fffffffff},
	/* 229 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x3fffffffff},
	/* 230 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x7fffffffff},
	/* 231 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffff},
	/* 232 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x1ffffffffff},
	/* 233 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x3ffffffffff},
	/* 234 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x7ffffffffff},
	/* 235 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xfffffffffff},
	/* 236 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x1fffffffffff},
	/* 237 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x3fffffffffff},
	/* 238 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x7fffffffffff},
	/* 239 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffff},
	/* 240 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x1ffffffffffff},
	/* 241 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x3ffffffffffff},
	/* 242 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x7ffffffffffff},
	/* 243 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xfffffffffffff},
	/* 244 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x1fffffffffffff},
	/* 245 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x3fffffffffffff},
	/* 246 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x7fffffffffffff},
	/* 247 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffff},
	/* 248 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x1ffffffffffffff},
	/* 249 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x3ffffffffffffff},
	/* 250 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x7ffffffffffffff},
	/* 251 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xfffffffffffffff},
	/* 252 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x1fffffffffffffff},
	/* 253 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x3fffffffffffffff},
	/* 254 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0x7fffffffffffffff},
	/* 255 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff},
}
