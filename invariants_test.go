// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/gaissmai/bart/internal/allot"
	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/bitset"
	"github.com/gaissmai/bart/internal/lpm"
)

// TestLookupTableInvariants validates the core invariants of all precomputed lookup tables
// by testing them against the actual ART allotment logic extended to the full 511 range.
func TestLookupTableInvariants(t *testing.T) {
	t.Parallel()

	t.Run("LPM_LookupTbl_Backtracking", func(t *testing.T) {
		t.Parallel()
		testLPMBacktrackingInvariants(t)
	})

	t.Run("Allot_Tables_vs_Extended_Logic", func(t *testing.T) {
		t.Parallel()
		testAllotTablesAgainstExtendedLogic(t)
	})

	t.Run("ART_Index_Mappings", func(t *testing.T) {
		t.Parallel()
		testARTIndexMappings(t)
	})
}

// testLPMBacktrackingInvariants validates the LPM lookup table against backtracking logic
func testLPMBacktrackingInvariants(t *testing.T) {
	if len(lpm.LookupTbl) != 256 {
		t.Fatalf("lpm.LookupTbl length = %d, want 256", len(lpm.LookupTbl))
	}

	// Test backtracking path generation for all indices
	for i := range 256 {
		//nolint:gosec
		idx := uint8(i)
		entry := lpm.LookupTbl[idx]
		expected := genBacktrackingPath(idx)

		if entry != expected {
			t.Errorf("lpm.LookupTbl[%d] backtracking path mismatch", idx)
		}
	}
}

// testAllotTablesAgainstExtendedLogic validates the allot tables using direct slice generation
func testAllotTablesAgainstExtendedLogic(t *testing.T) {
	// Test key representative indices from different ranges
	testIndices := []uint8{
		1, 2, 3, 4, 7, 8, 15, 16, 31, 32, 63, 64,
		127, 128, 129, 192, 255, // Key boundary values
	}

	for _, startIdx := range testIndices {
		t.Run(fmt.Sprintf("idx_%d", startIdx), func(t *testing.T) {
			// Generate the allotment slice directly for this starting index up to 511
			allotKeys := generateExtendedAllotSlice(startIdx)

			// Test that the specific lookup table entries match the allot tree
			validateTableEntriesAgainstAllot(t, startIdx, allotKeys)
		})
	}
}

// testARTIndexMappings validates the ART index mapping functions
func testARTIndexMappings(t *testing.T) {
	// Test OctetToIdx with representative values
	octets := []uint8{0, 1, 127, 128, 255}
	for _, octet := range octets {
		idx := art.OctetToIdx(octet)
		expected := 128 + octet>>1

		if idx != expected {
			t.Errorf("art.OctetToIdx(%d) = %d, want %d", octet, idx, expected)
		}

		if idx < 128 {
			t.Errorf("art.OctetToIdx(%d) = %d, should be >= 128", octet, idx)
		}
	}

	// Test PfxToIdx with known valid combinations
	pfxCases := []struct {
		octet, pfxLen, want uint8
	}{
		{0, 0, 1},     // Default route
		{0, 1, 2},     // 0.0.0.0/1
		{128, 1, 3},   // 128.0.0.0/1
		{80, 4, 21},   // From art tests
		{255, 7, 255}, // From art tests
	}

	for _, tc := range pfxCases {
		got := art.PfxToIdx(tc.octet, tc.pfxLen)
		if got != tc.want {
			t.Errorf("art.PfxToIdx(%d, %d) = %d, want %d", tc.octet, tc.pfxLen, got, tc.want)
		}

		if got == 0 {
			t.Errorf("art.PfxToIdx(%d, %d) = 0, should never return invalid index", tc.octet, tc.pfxLen)
		}
	}
}

// generateExtendedAllotSlice simulates the allot function logic extended to 511, returning a slice directly
func generateExtendedAllotSlice(startIdx uint8) []uint {
	var allotKeys []uint

	// Use stack-based iteration like the original allot function
	stack := []uint{uint(startIdx)}

	for len(stack) > 0 {
		// pop last idx
		idx := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		// Add this index to our result slice
		allotKeys = append(allotKeys, idx)

		// Continue propagation up to 511
		if idx < 256 {
			// Binary tree children: left = idx*2, right = (idx*2)+1
			leftChild := idx << 1
			rightChild := (idx << 1) + 1

			if leftChild <= 511 {
				stack = append(stack, leftChild)
			}
			if rightChild <= 511 {
				stack = append(stack, rightChild)
			}
		}
	}

	// Sort the result for consistent testing
	slices.Sort(allotKeys)
	return allotKeys
}

// validateTableEntriesAgainstAllot validates that specific lookup table entries match the allot tree
func validateTableEntriesAgainstAllot(t *testing.T, startIdx uint8, expectedKeys []uint) {
	// Get the specific prefix routes bitset for this startIdx
	prefixBitset := allot.PfxRoutesLookupTbl[startIdx]
	prefixSlice := prefixBitset.Bits()

	// Get the specific fringe routes bitset for this startIdx
	fringeBitset := allot.FringeRoutesLookupTbl[startIdx]
	fringeSlice := fringeBitset.Bits()

	// Convert prefix slice to uint
	prefixKeys := make([]uint, len(prefixSlice))
	for i, idx := range prefixSlice {
		prefixKeys[i] = uint(idx)
	}

	// Convert fringe slice - simply add 256 to map back to original range
	fringeKeys := make([]uint, len(fringeSlice))
	for i, idx := range fringeSlice {
		fringeKeys[i] = uint(idx) + 256
	}

	// Combine table-derived keys
	tableKeys := slices.Concat(prefixKeys, fringeKeys)

	// Direct comparison
	if !slices.Equal(tableKeys, expectedKeys) {
		t.Errorf("Lookup table entries mismatch for startIdx %d\n got: %v\nwant: %v",
			startIdx, tableKeys, expectedKeys)
	}
}

// genBacktrackingPath builds a backtracking path bitset for LPM
func genBacktrackingPath(i uint8) (path bitset.BitSet256) {
	for ; i > 0; i >>= 1 {
		path.Set(i)
	}
	return
}
