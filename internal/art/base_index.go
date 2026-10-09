// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Package art summarizes the functions and inverse functions
// for mapping between a prefix and a baseIndex.
//
//	can inline OctetToIdx with cost 6
//	can inline PfxToIdx with cost 10
//	can inline PfxBits with cost 21
//	can inline IdxToPfx with cost 29
//	can inline IdxToRange with cost 47
package art

import "math/bits"

// PfxToIdx maps 8bit prefixes to numbers. The prefixes range from 0/0 to 255/7
// The return values range from 1 to 255.
//
//	  [0x0000_0001 .. 0x1111_1111] = [1 .. 255]
//
//		example: octet/pfxLen: 160/3 = 0b1010_0000/3 => IdxToPfx(160/3) => 13
//
//		                0b1010_0000 => 0b0000_0101
//		                  ^^^ >> (8-3)         ^^^
//
//		                0b0000_0001 => 0b0000_1000
//		                          ^ << 3      ^
//		                 + -----------------------
//		                               0b0000_1101 = 13
//
// ATTENTION: For pfxLen values > 7 the result is undefined!
func PfxToIdx(octet, pfxLen uint8) uint8 {
	//  use '|' instead of '+', maybe a little bit faster
	return octet>>(8-pfxLen) | 1<<pfxLen
}

// OctetToIdx maps octet/8 prefixes from [256..511] => [128..255] to start
// with the parent (>>1) in the complete binary tree.
//
// Same formula as PfxToIdx, but for octet/8 and a shift (>>1).
//
//	octet>>(8-pfxLen) + 1<<pfxLen
//	     octet>>(8-8) + 1<<8
//	            octet + 256
//
//	got to parent idx:
//	    (octet+256)>>1 == octet>>1 + 128
func OctetToIdx(octet uint8) uint8 {
	//  use '|' instead of '+', maybe a little bit faster
	return octet>>1 | 128
}

// IdxToPfx converts a baseIndex (1..255) back into its original octet and prefix length.
// It is the inverse operation of PfxToIdx. Panics if idx is 0.
func IdxToPfx(idx uint8) (octet, pfxLen uint8) {
	if idx == 0 {
		panic("IdxToPfx: invalid idx 0")
	}

	// The prefix length matches the 0-based position of the most significant bit.
	// Since bits.Len8 returns the 1-based bit count (1..8 for valid uint8 values),
	// subtracting 1 yields the exact prefix length.
	//
	//nolint:gosec //G115: integer overflow conversion int -> uint8
	pfxLen = uint8(bits.Len8(idx)) - 1

	// Strip the MSB (1 << pfxLen) and shift the remaining prefix bits
	// back to their original position in the octet.
	octet = (idx ^ (1 << pfxLen)) << (8 - pfxLen)

	return octet, pfxLen
}

// PfxBits returns the total prefix length in bits for a given base index
// and trie depth (measured in 8-bit strides).
//
// Depth represents the number of preceding full octets (e.g., depth 0 = 0 bits,
// depth 2 = 16 bits). The base index encodes the prefix length within the current stride.
//
// For example:
//
//	depth = 2 (2 preceding octets = 16 bits)
//	idx = 13 (0b1101) (baseIndex 0b1101: MSB at 1-based position 4 => 3 bits in current stride)
//
//	=> PfxBits = 2*8 + 3 = 19
func PfxBits(depth int, idx uint8) uint8 {
	// Since bits.Len8 returns the 1-based bit count (1..8 for valid uint8 values),
	// subtracting 1 yields the exact prefix length.
	pfxLenInStride := bits.Len8(idx) - 1

	// Each preceding trie level represents 8 bits (one octet).
	baseBits := depth << 3 // depth * 8

	// Total prefix length in bits, [0..128]
	//nolint:gosec // G115: integer overflow conversion int -> uint8
	return uint8(baseBits + pfxLenInStride)
}

// IdxToRange returns the first and last octets covered by a base index.
//
// The base index encodes a prefix of up to 8 bits inside a single stride (octet).
// This function computes the numerical start and end of the value range for that prefix.
//
// For example:
//
//   - A 0-bit prefix (idx == 1) covers the full range:   0..255
//
//   - A 3-bit prefix like 0b101xxxx (idx == 13) covers:  160..191
//
//     idx := PfxToIdx(0b10100000, 3) // 13
//     first, last := IdxToRange(13)  // 160, 191
//
// Internally, this function decodes (octet, prefixLength) via IdxToPfx,
// then computes the broadcast address (last octet) by filling all bits
// below the prefix length.
func IdxToRange(idx uint8) (first, last uint8) {
	// Decode the prefix base (octet) and length (number of fixed bits)
	first, pfxLen := IdxToPfx(idx)

	// Compute the "broadcast" value by filling trailing (host) bits with 1s.
	// This gives the maximum octet value that still matches the prefix.
	last = first | netMask[pfxLen]

	return first, last
}

var netMask = [9]byte{
	0b11111111,
	0b01111111,
	0b00111111,
	0b00011111,
	0b00001111,
	0b00000111,
	0b00000011,
	0b00000001,
	0b00000000,
}
