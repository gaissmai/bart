// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import "github.com/bits-and-blooms/bitset"

// see comment in allot_tbl_little_endian.go

// allotedPrefixRoutes, returns the precalculated words as array from lookup table.
func allotedPrefixRoutes(idx uint) (a8 [8]uint64) {
	if idx < firstHostIndex {
		// use precalculated bitset
		return allotLookupTbl[idx]
	}
	// upper half in allot tbl, just 1 bit is set, fast calculation at runtime
	bitset.From(a8[:]).Set(idx)
	return a8
}
