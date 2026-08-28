// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Package bitset provides a compact and efficient implementation of a fixed-length
// bitset for the range [0..255].
//
// This implementation is optimized for internal use in a compressed routing trie
// and prioritizes minimal allocation, performance, and inlining. It supports
// constant-time set/test operations, iteration over set bits, ranking, and masked
// intersections.
//
// Internally, the bitset is represented by four uint64 words, providing
// fast bit-level access through direct indexing and hardware-accelerated primitives.
//
// For external consumers, the API intentionally avoids dynamic allocation except
// when explicitly requested (via Bits()).
package bitset

// can inline (*BitSet256).AsSlice with cost 42
// can inline (*BitSet256).Bits with cost 47
// can inline (*BitSet256).Clear with cost 12
// can inline (*BitSet256).FirstSet with cost 79
// can inline (*BitSet256).Intersects with cost 48
// can inline (*BitSet256).Intersection with cost 53
// can inline (*BitSet256).IntersectionTop with cost 67
// can inline (*BitSet256).IsEmpty with cost 22
// can inline (*BitSet256).LastSet with cost 75
// can inline (*BitSet256).NextSet with cost 65
// can inline (*BitSet256).Rank with cost 57
// can inline (*BitSet256).Set with cost 12
// can inline (*BitSet256).Size with cost 33
// can inline (*BitSet256).Test with cost 15
// can inline (*BitSet256).Union with cost 36

// can inline (*BitSet256).AsSlice with cost 47
// can inline (*BitSet256).Bits with cost 52
// can inline (*BitSet256).Clear with cost 18
// can inline BitSet256.FirstSet with cost 71
// can inline BitSet256.IntersectionTop with cost 80
// can inline BitSet256.Intersection with cost 22
// can inline BitSet256.Intersects with cost 26
// can inline BitSet256.IsEmpty with cost 14
// can inline BitSet256.LastSet with cost 59
// can inline BitSet256.Rank with cost 40
// can inline (*BitSet256).Set with cost 17
// can inline BitSet256.Size with cost 20
// can inline (*BitSet256).Test with cost 20
// can inline BitSet256.Union with cost 22
// cannot inline BitSet256.NextSet: function too complex: cost 121 exceeds budget 80

import (
	"math/bits"
	"unsafe"
)

// BitSet256 represents a fixed-size bitset for the range [0..255],
// stored as four uint64 words (256 bits total).
type BitSet256 struct{ W0, W1, W2, W3 uint64 }

// Set returns a new BitSet256 with the bit at position set to 1.
func (b *BitSet256) Set(bit uint8) {
	words := (*[4]uint64)(unsafe.Pointer(b))
	words[bit>>6] |= uint64(1) << (bit & 63)
}

// Clear returns a new BitSet256 with the bit at position set to 0.
func (b *BitSet256) Clear(bit uint8) {
	words := (*[4]uint64)(unsafe.Pointer(b))
	words[bit>>6] &= ^(uint64(1) << (bit & 63))
}

// Test reports whether the bit at position bit (0..255) is set.
func (b *BitSet256) Test(bit uint8) (result bool) {
	words := (*[4]uint64)(unsafe.Pointer(b))
	return words[bit>>6]&(uint64(1)<<(bit&63)) != 0
}

// FirstSet returns the index of the lowest (first) bit that is set in the BitSet256.
//
// It searches the 256-bit set in ascending order and returns the position of the
// first bit with value 1. If at least one bit is set, ok is true.
// If no bits are set, ok is false and first is undefined.
//
// Example:
//
//	var bs BitSet256
//	bs.Set(17)
//	bs.Set(42)
//	bs.Set(130)
//	bs.Set(255)
//	index, ok := bs.FirstSet()  // index == 17, ok == true
//
// Note: This implementation avoids a for loop for optimal speed.
// On modern CPUs, computing all four trailing-zero counts up front allows
// the CPU to parallelize these operations internally (pipelining), avoiding
// branch misprediction and maximizing instruction throughput. This technique
// is especially effective for bitsets with known, fixed word count.
func (b BitSet256) FirstSet() (first uint8, ok bool) {
	x0 := bits.TrailingZeros64(b.W0)
	x1 := bits.TrailingZeros64(b.W1)
	x2 := bits.TrailingZeros64(b.W2)
	x3 := bits.TrailingZeros64(b.W3)

	switch {
	case x0 != 64:
		return uint8(x0), true
	case x1 != 64:
		return uint8(x1 + 64), true
	case x2 != 64:
		return uint8(x2 + 128), true
	case x3 != 64:
		return uint8(x3 + 192), true
	}

	return
}

// NextSet returns the index of the next set bit that is greater than or equal to bit.
//
// If such a bit exists, it returns its index as next and ok=true.
// Otherwise, ok is false and next is undefined.
//
// The search starts at the given bit index and proceeds toward higher indices, scanning
// across all four 64-bit segments of the internal bitset representation.
//
// Example:
//
//	b.Set(5)
//	b.Set(130)
//	b.NextSet(0)   ->   5, true
//	b.NextSet(5)   ->   5, true
//	b.NextSet(6)   -> 130, true
//	b.NextSet(200) ->   0, false
func (b BitSet256) NextSet(bit uint8) (uint8, bool) {
	wIdx := bit >> 6
	mask := ^uint64(0) << (bit & 63)

	switch wIdx {
	case 0:
		b.W0 &= mask
	case 1:
		b.W0 = 0
		b.W1 &= mask
	case 2:
		b.W0, b.W1 = 0, 0
		b.W2 &= mask
	case 3:
		b.W0, b.W1, b.W2 = 0, 0, 0
		b.W3 &= mask
	}

	switch {
	case b.W0 != 0:
		return uint8(bits.TrailingZeros64(b.W0)), true
	case b.W1 != 0:
		return 64 + uint8(bits.TrailingZeros64(b.W1)), true
	case b.W2 != 0:
		return 128 + uint8(bits.TrailingZeros64(b.W2)), true
	case b.W3 != 0:
		return 192 + uint8(bits.TrailingZeros64(b.W3)), true
	}

	return 0, false
}

// LastSet returns the index of the highest (last) bit that is set in the BitSet256.
//
// It searches the bitset in descending order and returns the position of the
// first bit (top bit) with value 1. If at least one bit is set, ok is true.
// If no bits are set, ok is false and last is 0.
//
// Example:
//
//	var bs BitSet256
//	bs.Set(2)
//	bs.Set(130)
//	bs.Set(214)
//	index, ok := bs.LastSet()  // index == 214, ok == true
func (b BitSet256) LastSet() (last uint8, ok bool) {
	if b.W3 != 0 {
		return uint8(bits.Len64(b.W3) + 191), true
	}
	if b.W2 != 0 {
		return uint8(bits.Len64(b.W2) + 127), true
	}
	if b.W1 != 0 {
		return uint8(bits.Len64(b.W1) + 63), true
	}
	if b.W0 != 0 {
		return uint8(bits.Len64(b.W0) - 1), true
	}
	return 0, false
}

// AsSlice extracts the indices of all set bits in the BitSet256, returning them
// as uint8 values in strictly ascending order.
//
// Performance Considerations:
// To guarantee zero heap allocations and enable compiler inlining, the caller must
// provide a pointer to a 256-byte array (`buf`) as backing storage. The method
// populates this array in-place and returns a sliced view (`[]uint8`) tailored to
// the actual number of set bits.
//
// Safety and Lifecycle:
// The returned slice directly shares the underlying storage of `buf` and is only
// valid until `buf` is modified or reused. This pattern is highly recommended for
// hot paths and performance-critical loops where heap churn must be avoided.
func (b *BitSet256) AsSlice(buf *[256]uint8) []uint8 {
	words := (*[4]uint64)(unsafe.Pointer(b))
	size := 0

	for wIdx, word := range words {
		for ; word != 0; size++ {
			buf[size] = uint8(wIdx<<6 + bits.TrailingZeros64(word))
			word &= word - 1 // clear the rightmost set bit
		}
	}

	// tailor to the actual number of set bits
	return buf[:size]
}

// Bits returns a slice containing the indices of all set bits in strictly
// ascending order as uint8 values.
//
// Performance Considerations:
// Unlike [AsSlice], this method dynamically allocates a new slice on the
// heap to store the result. It is designed for convenience and APIs where the lifecycle
// of the returned slice needs to outlive the immediate caller's stack frame.
//
// Usage Guidance:
// Use Bits when convenience is preferred over raw performance, or when the result
// must be returned across boundaries where stack-allocated buffers cannot safely escape.
// For high-throughput or allocation-free processing, prefer [AsSlice].
func (b *BitSet256) Bits() []uint8 {
	return b.AsSlice(&[256]uint8{})
}

// IntersectionTop computes the intersection of the receiver with c
// and returns the highest (top-most) set bit of the result.
// If the intersection is non-empty, it returns the top bit index and true.
// If the intersection is empty, ok is false and top is 0.
func (b BitSet256) IntersectionTop(c BitSet256) (top uint8, ok bool) {
	b.W3 &= c.W3
	b.W2 &= c.W2
	b.W1 &= c.W1
	b.W0 &= c.W0

	switch {
	case b.W3 != 0:
		return uint8(bits.Len64(b.W3) + 191), true
	case b.W2 != 0:
		return uint8(bits.Len64(b.W2) + 127), true
	case b.W1 != 0:
		return uint8(bits.Len64(b.W1) + 63), true
	case b.W0 != 0:
		return uint8(bits.Len64(b.W0) - 1), true
	}

	return 0, false
}

// Rank returns the number of bits set (i.e., value 1) in the BitSet256
// up to and including the provided index.
//
// The rank is computed efficiently using precomputed bitmasks (rankMask),
// which mask out all bits above the index. For example:
//
//	b.Set(3)
//	b.Set(5)
//	b.Set(120)
//	b.Rank(5)   -> 2     // only bits 3 and 5 are ≤ 5
//	b.Rank(119) -> 2     // only bits 3 and 5 are ≤ 119
//	b.Rank(120) -> 3     // bit 120 is included here
//
// Rank is particularly useful in prefix trees, indexing schemes,
// and data compression techniques where ordinal positions matter.
//
// Internally, the function performs four bitwise-and operations
// between the bitset words and a precomputed mask covering bits 0..idx,
// followed by popcount operations (via bits.OnesCount64).
//
// This avoids dynamic mask construction and enables branch-free, highly
// predictable performance.
func (b BitSet256) Rank(idx uint8) int {
	return bits.OnesCount64(b.W0&rankMask[idx].W0) +
		bits.OnesCount64(b.W1&rankMask[idx].W1) +
		bits.OnesCount64(b.W2&rankMask[idx].W2) +
		bits.OnesCount64(b.W3&rankMask[idx].W3)
}

// IsEmpty reports whether all 256 bits are zero.
func (b BitSet256) IsEmpty() bool {
	return b.W0|b.W1|b.W2|b.W3 == 0
}

// Intersects reports whether the receiver and c have at least one bit in common.
func (b BitSet256) Intersects(c BitSet256) bool {
	return ((b.W0 & c.W0) | (b.W1 & c.W1) | (b.W2 & c.W2) | (b.W3 & c.W3)) != 0
}

// Intersection returns a new BitSet256 containing only the bits
// that are set in both the receiver and c (bitwise AND).
func (b BitSet256) Intersection(c BitSet256) BitSet256 {
	b.W0 &= c.W0
	b.W1 &= c.W1
	b.W2 &= c.W2
	b.W3 &= c.W3
	return b
}

// Union returns a new BitSet256 containing the bitwise OR union of b and c.
func (b BitSet256) Union(c BitSet256) BitSet256 {
	b.W0 |= c.W0
	b.W1 |= c.W1
	b.W2 |= c.W2
	b.W3 |= c.W3
	return b
}

// Size returns the population count, i.e. the number of set bits.
func (b BitSet256) Size() int {
	return bits.OnesCount64(b.W0) +
		bits.OnesCount64(b.W1) +
		bits.OnesCount64(b.W2) +
		bits.OnesCount64(b.W3)
}

// rankMask is a table of bitmasks with all bits set to 1 up to and including a given bit position.
// It is used by BitSet256.Rank() to perform efficient popcount operations.
//
// This approach trades ~8KB of static memory for zero-allocation,
// fully branch-free, and cache-friendly Rank() calls with constant runtime.
//
// Used internally by the trie for position counting, CIDR ordering,
// and fast range-limited bit population counts.
//
//nolint:gochecknoglobals // Precomputed read‑only table used in hot paths.
var rankMask = [256]BitSet256{
	/*   0 */ {0x1, 0x0, 0x0, 0x0},
	/*   1 */ {0x3, 0x0, 0x0, 0x0},
	/*   2 */ {0x7, 0x0, 0x0, 0x0},
	/*   3 */ {0xf, 0x0, 0x0, 0x0},
	/*   4 */ {0x1f, 0x0, 0x0, 0x0},
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
