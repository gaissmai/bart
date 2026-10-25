// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import "testing"

// TestFastAlias verifies that the deprecated Fast is identical to Table.
func TestFastAlias(t *testing.T) {
	t.Parallel()

	fast := new(Fast[int])

	// compiles only if Fast[int] and Table[int] are identical types

	//nolint:staticcheck
	var tbl *Table[int] = fast

	pfx := mpp("10.0.0.0/8")
	fast.Insert(pfx, 42)

	if got, ok := tbl.Get(pfx); !ok || got != 42 {
		t.Errorf("Get(%s) = (%d, %v), want (42, true)", pfx, got, ok)
	}
}
