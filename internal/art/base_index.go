// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Package art summarizes the functions and inverse functions
// for mapping between a prefix and a baseIndex.
//
//	can inline HostIdx with cost 4
//	can inline PfxToIdx with cost 13
//	can inline PfxLen with cost 21
//	can inline IdxToPfx with cost 42
//	can inline IdxToRange with cost 66
//	can inline NetMask with cost 7
//
// Please read the ART paper ./doc/artlookup.pdf
// to understand the baseIndex algorithm.
package art

import "math/bits"

// HostIdx is just PfxToIdx(octet/8) but faster.
func HostIdx(octet uint) uint {
	return octet + 256
}

// PfxToIdx maps 8bit prefixes to numbers. The prefixes range from 0/0 to 255/8
// and the mapped values from:
//
//	  [0x0000_00001 .. 0x0000_0001_1111_1111] = [1 .. 511]
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
func PfxToIdx(octet byte, pfxLen int) uint {
	// uint8() are compiler optimization hints, that the shift amount is
	// smaller than the width of the types
	return uint(octet>>uint8(8-pfxLen)) + (1 << uint8(pfxLen))
}

// IdxToPfx returns the octet and prefix len of baseIdx.
// It's the inverse to pfxToIdx.
//
// It panics on invalid input, valid values for idx are from [1 .. 511]
func IdxToPfx(idx uint) (octet uint8, pfxLen int) {
	if idx == 0 || idx > 511 {
		panic("logic error, idx is out of bounds [1..511]")
	}

	pfxLen = bits.Len64(uint64(idx)) - 1
	shiftBits := 8 - uint8(pfxLen)

	mask := uint8(0xff) >> shiftBits
	octet = (uint8(idx) & mask) << shiftBits

	return
}

// PfxLen returns the bits based on depth and idx.
func PfxLen(depth int, idx uint) int {
	// see IdxToPfx
	if idx == 0 || idx > 511 {
		panic("logic error, idx is out of bounds [1..511]")
	}
	return depth<<3 + bits.Len64(uint64(idx)) - 1
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
