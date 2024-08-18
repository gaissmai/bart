// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import "github.com/bits-and-blooms/bitset"

// allotedPrefixRoutes, returns the precalculated words as array from lookup table.
func allotedPrefixRoutes(idx uint) (a8 [8]uint64) {
	if idx < 256 {
		// use precalculated bitset
		return allotLookupTbl[idx]
	}
	// upper half in allot tbl, just 1 bit is set, fast calculation at runtime
	bitset.From(a8[:]).Set(idx)
	return a8
}

// allotedHostRoutes, returns the precalculated words as array from lookup table.
func allotedHostRoutes(idx uint) (a4 [4]uint64) {
	if idx < 256 {
		// use precalculated bitset
		copy(a4[:], allotLookupTbl[idx][4:])
		return a4
	}
	// upper half in allot tbl, just 1 bit is set, fast calculation at runtime
	bitset.From(a4[:]).Set(idx - 256)
	return a4
}
