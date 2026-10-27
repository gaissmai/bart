// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Package art implements the base index mapping of the Allotment Routing
// Table (ART) algorithm by Donald E. Knuth. It summarizes the functions and
// their inverse functions for mapping between a prefix within one 8-bit
// stride and its base index.
//
// The base index numbers the prefixes of a stride like the nodes of a complete
// binary tree, in the same way as the elements of a binary heap:
//
//   - the prefix /0, the default route of the stride, has the index 1 (the root)
//   - the children of idx are 2*idx and 2*idx+1, the prefixes with one more bit
//   - the parent of idx is idx>>1, the prefix with one bit less
//   - the prefix length is the level in the tree: idx 1 is /0, idx 2..3 are /1,
//     idx 4..7 are /2, and so on up to idx 128..255 for /7
//   - an ancestor always has a smaller index than its descendants
//
// The indices 1..255 are the prefixes /0 up to /7 and fit into a uint8.
// The /8 prefixes of the stride would have the indices 256..511. They are not
// stored as prefixes in a node, they are the child addresses (the octets),
// see [OctetToIdx].
//
// The functions are small enough to be inlined, the compiler reports:
//
// can inline OctetToIdx with cost 6
// can inline PfxToIdx with cost 10
// can inline PfxBits with cost 21
// can inline IdxToPfx with cost 25
// can inline IdxToRange with cost 40
package art

import "math/bits"

// PfxToIdx maps a prefix of one stride, given as octet and prefix length
// pfxLen, to its base index. The prefixes range from 0/0 to 254/7, the
// return values range from 1 to 255:
//
//	[0b0000_0001 .. 0b1111_1111] = [1 .. 255]
//
// The host bits of octet, all bits behind the first pfxLen bits, are ignored.
//
// The base index is the pfxLen prefix bits with a leading 1 bit in front as
// marker, which makes the index unique across all prefix lengths:
//
//	idx = 1<<pfxLen | octet>>(8-pfxLen)
//
// Example: octet/pfxLen = 160/3 = 0b1010_0000/3 => PfxToIdx(160, 3) => 13
//
//	octet>>(8-pfxLen):  0b1010_0000 >> 5 =  0b0000_0101   the 3 prefix bits
//	        1<<pfxLen:            1 << 3 =  0b0000_1000   the marker bit
//	                                        -------------
//	or:                                     0b0000_1101 = 13
//
// ATTENTION: For pfxLen values > 7 the result is undefined. The function does
// not check its arguments, it is used on the hot path. pfxLen == 8 returns the
// octet, which is the index of another prefix, larger values return 0.
func PfxToIdx(octet, pfxLen uint8) uint8 {
	return octet>>(8-pfxLen) | 1<<pfxLen
}

// OctetToIdx maps an octet to the base index of the /7 prefix that contains it.
//
// The octet is the /8 prefix of the stride with the index 256+octet, which does
// not fit into a uint8 and is not stored as a prefix in a node. Its parent (>>1)
// in the complete binary tree is the /7 prefix, the most specific prefix a node
// can store. All less specific prefixes covering the octet are the ancestors of
// this index, see lpm.LookupTbl.
//
// Same formula as PfxToIdx, but for octet/8 and a shift (>>1) to the parent:
//
//	octet>>(8-pfxLen) + 1<<pfxLen
//	     octet>>(8-8) + 1<<8
//	            octet + 256
//
//	go to parent idx:
//	    (octet+256)>>1 == octet>>1 + 128
//
// Hence OctetToIdx(octet) is the same as PfxToIdx(octet, 7).
func OctetToIdx(octet uint8) uint8 {
	return octet>>1 | 128
}

// IdxToPfx returns the octet and the prefix length pfxLen of the base index idx.
// It's the inverse to [PfxToIdx], the host bits of the returned octet are zero.
//
// IdxToPfx panics for the invalid index 0.
func IdxToPfx(idx uint8) (octet, pfxLen uint8) {
	if idx == 0 {
		panic("IdxToPfx: invalid idx 0")
	}
	// The index is the prefix bits with a leading 1 bit as marker in front,
	// so the position of this marker is the prefix length.
	// bits.Len8 returns the number of bits needed to represent idx as binary,
	// so we subtract 1 to recover the prefix length (which is always >= 0).
	// Invariant: idx==0 is invalid

	//nolint:gosec  //G115: integer overflow conversion int -> uint8
	pfxLen = uint8(bits.Len8(idx)) - 1
	octet = idx << (8 - pfxLen)

	return octet, pfxLen
}

// PfxBits returns the prefix length in bits of the prefix represented by the
// base index idx in a node at the given trie depth.
//
// Each trie level represents an 8-bit stride (one octet), a node at depth d is
// preceded by d complete strides. The base index contains enough information
// to recover the prefix length within the stride, see [IdxToPfx]. This function
// adds both: 8 bits for every complete stride above plus the prefix length
// within the stride.
//
// For example:
//
//	depth = 2 (i.e. third trie level)
//	idx = 13 (which encodes a prefix of length 3 bits within that stride)
//
//	=> PfxBits = 2*8 + 3 = 19
//
// The result is at most 127, since the host routes /32 and /128 are never stored
// as a prefix in a node, they are fringes.
func PfxBits(depth int, idx uint8) uint8 {
	// bits.Len8(idx) gives the number of significant bits (i.e. prefix length + 1)
	// subtract 1 to get the actual prefix length used in this stride
	pfxLenInStride := bits.Len8(idx) - 1

	// Each trie level represents 8 bits (an octet), max depth is 16
	baseBits := depth << 3 // same as depth * 8

	// Total prefix length in bits, [0..127]
	//nolint:gosec   // G115: integer overflow conversion int -> uint8
	return uint8(baseBits + pfxLenInStride)
}

// IdxToRange returns the first and the last octet covered by a base index.
//
// The base index encodes a prefix of up to 7 bits inside a single stride (octet).
// This function computes the numerical start and end of the octet range for
// that prefix.
//
// For example:
//
//	idx == 1  (the 0-bit prefix)             covers 0..255
//	idx == 13 (the 3-bit prefix 0b101x_xxxx) covers 160..191
//
//	idx := PfxToIdx(0b1010_0000, 3) // 13
//	first, last := IdxToRange(13)   // 160, 191
//
// Internally, this function decodes (octet, prefixLength) via [IdxToPfx],
// then computes the last octet of the range by setting all host bits to 1.
//
// IdxToRange panics for the invalid index 0.
func IdxToRange(idx uint8) (first, last uint8) {
	first, pfxLen := IdxToPfx(idx)
	return first, first | 0xff>>pfxLen
}
