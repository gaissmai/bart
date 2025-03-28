// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Package art summarizes the functions and inverse functions
// for mapping between a prefix and a baseIndex.
//
//	can inline HostIdx with cost 4
//	can inline PfxToIdx with cost 13
//	can inline IdxToPfx with cost 39
//	can inline IdxToRange with cost 63
//	can inline PfxLen with cost 21
//	can inline NetMask with cost 7
//
// Please read the ART paper ./doc/artlookup.pdf
// to understand the baseIndex algorithm.
package art

import "math/bits"

// HostIdx is just PfxToIdx(octet, 8) but faster.
func HostIdx(octet uint) uint {
	return 256 + octet
}

// PfxToIdx maps a prefix table as a 'complete binary tree'.
func PfxToIdx(octet byte, pfxLen int) uint {
	// uint16() are compiler optimization hints, that the shift amount is
	// smaller than the width of the types
	return uint(octet>>uint8(8-pfxLen)) + (1 << uint8(pfxLen))
}

// IdxToPfx returns the octet and prefix len of baseIdx.
// It's the inverse to pfxToIdx.
func IdxToPfx(idx uint) (octet uint8, pfxLen int) {
	// the idx is in the range [0..511]
	if idx > 255 {
		pfxLen = 8
	} else {
		pfxLen = bits.Len8(uint8(idx)) - 1
	}

	shiftBits := uint8(8 - pfxLen)
	mask := uint8(0xff >> shiftBits)
	octet = (uint8(idx) & mask) << shiftBits

	return
}

// PfxLen returns the bits based on depth and idx.
func PfxLen(depth int, idx uint) int {
	// see IdxToPfx
	if idx > 255 {
		return (depth + 1) << 3
	}
	return depth<<3 + bits.Len8(uint8(idx)) - 1
}

// IdxToRange returns the first and last octet of prefix idx.
func IdxToRange(idx uint) (first, last uint8) {
	first, pfxLen := IdxToPfx(idx)
	last = first | ^NetMask(pfxLen)
	return
}

// NetMask for bits
//
//	0b0000_0000, // bits == 0
//	0b1000_0000, // bits == 1
//	0b1100_0000, // bits == 2
//	0b1110_0000, // bits == 3
//	0b1111_0000, // bits == 4
//	0b1111_1000, // bits == 5
//	0b1111_1100, // bits == 6
//	0b1111_1110, // bits == 7
//	0b1111_1111, // bits == 8
func NetMask(bits int) uint8 {
	return 0b1111_1111 << (8 - uint16(bits))
}
