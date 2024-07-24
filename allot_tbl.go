// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import "github.com/bits-and-blooms/bitset"

// allotedPrefixRoutes, overwrite the buffer with the precalculated words in allotment table.
func allotedPrefixRoutes(idx uint, buf []uint64) {
	if idx < 256 {
		// overwrite the backing array of bitset with precalculated bitset
		copy(buf[:], allotLookupTbl[idx][:])
		return
	}
	// upper half in allot tbl, just 1 bit is set, fast calculation at runtime
	bitset.From(buf[:]).Set(idx)
}

// allotedHostRoutes, overwrite the buffer with the precalculated words in allotment table.
func allotedHostRoutes(idx uint, buf []uint64) {
	if idx < 256 {
		// overwrite the backing array of bitset with precalculated bitset
		copy(buf[:], allotLookupTbl[idx][4:])
		return
	}
	// upper half in allot tbl, just 1 bit is set, fast calculation at runtime
	bitset.From(buf[:]).Set(idx - 256)
}
