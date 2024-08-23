// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

//go:build !big_endian

package bart

// allotTblFor, returns the precalculated allot table for baseIndex.
// Used for bitset intersections instead of range loops in overlap methods.
func allotTblFor(idx uint) [8]uint64 {
	return allotLookupTblLittleEndian[idx]
}
