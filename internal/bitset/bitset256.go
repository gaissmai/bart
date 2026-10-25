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

// can inline (*BitSet256).AlignedPairs with cost 54
// can inline (*BitSet256).All with cost 17
// can inline (*BitSet256).AllEnumerate with cost 17
// can inline (*BitSet256).And with cost 53
// can inline (*BitSet256).AndTop with cost 67
// can inline (*BitSet256).AppendBits with cost 31
// can inline (*BitSet256).Bits with cost 66
// can inline (*BitSet256).Clear with cost 12
// can inline (*BitSet256).FirstSet with cost 79
// can inline (*BitSet256).IsEmpty with cost 22
// can inline (*BitSet256).LastSet with cost 75
// can inline (*BitSet256).NextSet with cost 65
// can inline (*BitSet256).OnesCount with cost 28
// can inline (*BitSet256).Or with cost 53
// can inline (*BitSet256).Overlaps with cost 42
// can inline (*BitSet256).Set with cost 12
// can inline (*BitSet256).Test with cost 15
// can inline (*BitSet256).Xor with cost 53

import (
	"iter"
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

// BitSet256 represents a fixed-size bitset for the range [0..255],
// stored as four uint64 words (256 bits total).
type BitSet256 [4]uint64

// Set sets the bit at position bit (0..255).
func (b *BitSet256) Set(bit uint8) {
	b[bit>>6] |= 1 << (bit & 63)
}

// Clear clears the bit at position bit (0..255).
func (b *BitSet256) Clear(bit uint8) {
	b[bit>>6] &^= 1 << (bit & 63)
}

// Test reports whether the bit at position bit (0..255) is set.
func (b *BitSet256) Test(bit uint8) (ok bool) {
	return b[bit>>6]&(1<<(bit&63)) != 0
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
//
//nolint:gosec  // G115: integer overflow conversion int -> uint8
func (b *BitSet256) FirstSet() (first uint8, ok bool) {
	// optimized for pipelining, can still inline with cost 79
	x0 := bits.TrailingZeros64(b[0])
	x1 := bits.TrailingZeros64(b[1])
	x2 := bits.TrailingZeros64(b[2])
	x3 := bits.TrailingZeros64(b[3])

	if x0 != 64 {
		return uint8(x0), true
	}
	if x1 != 64 {
		return uint8(x1 + 64), true
	}
	if x2 != 64 {
		return uint8(x2 + 128), true
	}
	if x3 != 64 {
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
//
//nolint:gosec  // G115: integer overflow conversion int -> uint8
func (b *BitSet256) NextSet(bit uint8) (next uint8, ok bool) {
	wIdx := int(bit >> 6)

	// process the first (maybe partial) word
	first := b[wIdx] >> (bit & 63)
	if first != 0 {
		return bit + uint8(bits.TrailingZeros64(first)), true
	}

	// process the following words until next bit is set
	for wIdx++; wIdx < 4; wIdx++ {
		if next := b[wIdx]; next != 0 {
			return uint8(wIdx<<6 + bits.TrailingZeros64(next)), true
		}
	}
	return
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
//
//nolint:gosec  // G115: integer overflow conversion int -> uint8
func (b *BitSet256) LastSet() (last uint8, ok bool) {
	// optimized by unrolling the loop.
	// This enables compiler inlining and is ~20% faster.
	if b[3] != 0 {
		return uint8(bits.Len64(b[3]) + 191), true
	}
	if b[2] != 0 {
		return uint8(bits.Len64(b[2]) + 127), true
	}
	if b[1] != 0 {
		return uint8(bits.Len64(b[1]) + 63), true
	}
	if b[0] != 0 {
		return uint8(bits.Len64(b[0]) - 1), true
	}
	return 0, false
}

// All returns an iterator over the indices of all set bits in the BitSet256
// in strictly ascending order.
//
// Note on Dense BitSets:
// While iter.Seq provides clean iterator semantics and supports early breaking,
// calling yield() in a dense iteration loop introduces slight state-machine
// and yield-inlining overhead compared to batch writes. For maximum throughput
// in dense bitsets, consider using AppendBits with a pre-allocated slice buffer.
//
//nolint:gosec // G115: integer overflow conversion int -> uint8
func (b *BitSet256) All() iter.Seq[uint8] {
	return func(yield func(uint8) bool) {
		for wIdx, word := range b {
			for word != 0 {
				bitIdx := uint8(wIdx<<6 + bits.TrailingZeros64(word))
				if !yield(bitIdx) {
					return
				}
				word &= word - 1
			}
		}
	}
}

// AllEnumerate returns a pair iterator yielding a zero-based iteration sequence number
// and the bit index for each set bit in the BitSet256 in strictly ascending order.
//
// See [BitSet256.All] for performance considerations on dense bitsets.
//
//nolint:gosec // G115: integer overflow conversion int -> uint8
func (b *BitSet256) AllEnumerate() iter.Seq2[uint8, uint8] {
	return func(yield func(uint8, uint8) bool) {
		var i uint8
		for wIdx, word := range b {
			for word != 0 {
				bitIdx := uint8(wIdx<<6 + bits.TrailingZeros64(word))
				if !yield(i, bitIdx) {
					return
				}
				i++
				word &= word - 1
			}
		}
	}
}

// AppendBits appends the indices of all set bits in the BitSet256 to buf
// in strictly ascending order and returns the extended slice.
//
// Performance Considerations:
// To achieve zero heap allocations in performance-critical loops, the caller should
// pass a slice backed by stack memory (e.g., slicing a local array `buf[:0]` or a pre-allocated buffer).
// The function appends directly to `buf`, avoiding heap churn if `cap(buf)` is sufficient.
//
// Safety and Lifecycle:
// If `buf` has sufficient capacity, the returned slice shares its underlying storage.
// If capacity is exceeded, standard slice growth rules apply and a new backing array
// will be allocated.
//
//nolint:gosec // G115: integer overflow conversion int -> uint8
func (b *BitSet256) AppendBits(buf []uint8) []uint8 {
	for wIdx, word := range b {
		for word != 0 {
			buf = append(buf, uint8(wIdx<<6+bits.TrailingZeros64(word)))
			word &= word - 1
		}
	}
	return buf
}

// Bits returns a newly allocated slice containing the indices of all set bits
// in strictly ascending order as uint8 values.
//
// Performance Considerations:
// Unlike [BitSet256.AppendBits], this method allocates backing storage on the heap.
// It is designed for convenience when caller lifecycle management across stack
// boundaries is required.
//
// For high-throughput or allocation-free processing in hot paths, prefer [BitSet256.AppendBits].
func (b *BitSet256) Bits() []uint8 {
	// Directly allocate heap slice with exact capacity to avoid re-allocations in append.
	return b.AppendBits(make([]uint8, 0, b.OnesCount()))
}

// AndTop computes the intersection of the receiver with c
// and returns the highest (top-most) set bit of the result.
// If the intersection is non-empty, it returns the top bit index and true.
// If the intersection is empty, ok is false and top is 0.
//
//nolint:gosec  // G115: integer overflow conversion int -> uint8
func (b *BitSet256) AndTop(c *BitSet256) (top uint8, ok bool) {
	// optimized by unrolling the first word check.
	// This enables compiler inlining and is ~15% faster.
	if w := b[3] & c[3]; w != 0 {
		return uint8(191 + bits.Len64(w)), true
	}
	for wIdx := 2; wIdx >= 0; wIdx-- {
		if w := b[wIdx] & c[wIdx]; w != 0 {
			return uint8(wIdx<<6 + bits.Len64(w) - 1), true
		}
	}
	return
}

// AlignedPairs returns a new BitSet256 where bit n is set if and only if
// n is EVEN, and both bit n and bit n+1 were set in the original bitset.
func (b *BitSet256) AlignedPairs() BitSet256 {
	// Mask with bits set at all even positions (0, 2, 4, 6, ..., 62)
	const evenBits uint64 = 0x5555_5555_5555_5555

	// Note: No cross-word carry is needed here!
	// Since n must be strictly EVEN (0, 2, 4...), an aligned pair (n, n+1)
	// always resides within the exact same uint64 word (e.g., bits 62 & 63 in b[0],
	// or bits 64 & 65 in b[1]). Cross-boundary pairs like (63, 64) start on an ODD
	// index and are intentionally excluded by the aligned evenBits constraint.
	return BitSet256{
		b[0] & (b[0] >> 1) & evenBits,
		b[1] & (b[1] >> 1) & evenBits,
		b[2] & (b[2] >> 1) & evenBits,
		b[3] & (b[3] >> 1) & evenBits,
	}
}

// IsEmpty reports whether all 256 bits are zero.
func (b *BitSet256) IsEmpty() bool {
	return b[0]|b[1]|b[2]|b[3] == 0
}

// Overlaps reports whether the receiver and c have at least one bit in common.
func (b *BitSet256) Overlaps(c *BitSet256) bool {
	return (b[0]&c[0] | b[1]&c[1] | b[2]&c[2] | b[3]&c[3]) != 0
}

// And returns a new [BitSet256] representing the bitwise intersection (AND) of b and c.
// It performs a component-wise bitwise AND operation across all words and leaves
// the original receiver unchanged.
func (b *BitSet256) And(c *BitSet256) (bs BitSet256) {
	bs[0] = b[0] & c[0]
	bs[1] = b[1] & c[1]
	bs[2] = b[2] & c[2]
	bs[3] = b[3] & c[3]
	return
}

// Xor returns a new [BitSet256] representing the bitwise symmetric difference (XOR) of b and c.
// It performs a component-wise bitwise XOR operation across all words and leaves
// the original receiver unchanged.
func (b *BitSet256) Xor(c *BitSet256) (bs BitSet256) {
	bs[0] = b[0] ^ c[0]
	bs[1] = b[1] ^ c[1]
	bs[2] = b[2] ^ c[2]
	bs[3] = b[3] ^ c[3]
	return
}

// Or returns a new [BitSet256] representing the bitwise union (OR) of b and c.
// It performs a component-wise bitwise OR operation across all words and leaves
// the original receiver unchanged.
func (b *BitSet256) Or(c *BitSet256) (bs BitSet256) {
	bs[0] = b[0] | c[0]
	bs[1] = b[1] | c[1]
	bs[2] = b[2] | c[2]
	bs[3] = b[3] | c[3]
	return
}

// OnesCount returns the population count, i.e. the number of set bits.
func (b *BitSet256) OnesCount() int {
	return bits.OnesCount64(b[0]) +
		bits.OnesCount64(b[1]) +
		bits.OnesCount64(b[2]) +
		bits.OnesCount64(b[3])
}
