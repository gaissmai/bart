// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

//go:build big_endian

package bart

import "math/bits"

// allotTblFor, returns the precalculated allot table for baseIndex.
// Used for bitset intersections instead of range loops in overlap methods.
func allotTblFor(idx uint) [8]uint64 {
	a8 := allotLookupTblLittleEndian[idx]
	for i := range 8 {
		// convert uint64 values from little to big endian
		a8[i] = bits.ReverseBytes64(a8[i])
	}
	return a8
}
