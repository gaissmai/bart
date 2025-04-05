// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Package art summarizes the functions and inverse functions
// for mapping between a prefix and a baseIndex.
//
//	can inline HostIdx with cost 5
//	can inline IdxToPfx256 with cost 37
//	can inline IdxToRange256 with cost 61
//	can inline NetMask with cost 7
//	can inline PfxLen256 with cost 18
//	can inline PfxToIdx256 with cost 29
//	can inline pfxToIdx with cost 11
//
// Please read the ART paper ./doc/artlookup.pdf
// to understand the baseIndex algorithm.
package art

import "math/bits"

// HostIdx is just PfxToIdx(octet/8) but faster.
func HostIdx(octet uint8) uint {
	return uint(octet) + 256
}

// pfxToIdx maps 8bit prefixes to numbers. The prefixes range from 0/0 to 255/8
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
func pfxToIdx(octet, pfxLen uint8) uint {
	return uint(octet>>(8-pfxLen)) + uint(1<<pfxLen)
}

// PfxToIdx256 maps 8bit prefixes to numbers. The values range [1 .. 255].
// Values > 255 are shifted by >> 1.
func PfxToIdx256(octet, pfxLen uint8) uint8 {
	idx := pfxToIdx(octet, pfxLen)
	if idx > 255 {
		idx >>= 1
	}
	return uint8(idx)
}

// IdxToPfx256 returns the octet and prefix len of baseIdx.
// It's the inverse to pfxToIdx256.
//
// It panics on invalid input.
func IdxToPfx256(idx uint8) (octet, pfxLen uint8) {
	if idx == 0 {
		panic("logic error, idx is 0")
	}

	pfxLen = uint8(bits.Len8(idx)) - 1
	shiftBits := 8 - pfxLen

	mask := uint8(0xff) >> shiftBits
	octet = (idx & mask) << shiftBits

	return
}

// PfxLen256 returns the bits based on depth and idx.
func PfxLen256(depth int, idx uint8) uint8 {
	// see IdxToPfx256
	if idx == 0 {
		panic("logic error, idx is 0")
	}
	return uint8(depth<<3 + bits.Len8(idx) - 1)
}

// IdxToRange256 returns the first and last octet of prefix idx.
func IdxToRange256(idx uint8) (first, last uint8) {
	first, pfxLen := IdxToPfx256(idx)
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
func NetMask(bits uint8) uint8 {
	return 0b1111_1111 << (8 - uint16(bits))
}
