// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"iter"
	"maps"
	"net/netip"
	"testing"
)

// TestUnionDeterministic tests union with known prefix combinations
func TestUnionDeterministic(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		prefixes1 []string
		values1   []int
		prefixes2 []string
		values2   []int
		expected  map[netip.Prefix]int
	}{
		{
			name:      "No overlap",
			prefixes1: []string{"10.0.0.0/8", "192.168.1.0/24"},
			values1:   []int{100, 200},
			prefixes2: []string{"172.16.0.0/12", "203.0.113.0/24"},
			values2:   []int{300, 400},
			expected: map[netip.Prefix]int{
				mpp("10.0.0.0/8"):     100,
				mpp("192.168.1.0/24"): 200,
				mpp("172.16.0.0/12"):  300,
				mpp("203.0.113.0/24"): 400,
			},
		},
		{
			name:      "Complete overlap - tbl2 values should win",
			prefixes1: []string{"10.0.0.0/8", "192.168.1.0/24"},
			values1:   []int{100, 200},
			prefixes2: []string{"10.0.0.0/8", "192.168.1.0/24"},
			values2:   []int{999, 888},
			expected: map[netip.Prefix]int{
				mpp("10.0.0.0/8"):     999,
				mpp("192.168.1.0/24"): 888,
			},
		},
		{
			name:      "Partial overlap",
			prefixes1: []string{"10.0.0.0/8", "192.168.1.0/24", "172.16.0.0/12"},
			values1:   []int{100, 200, 300},
			prefixes2: []string{"192.168.1.0/24", "203.0.113.0/24"},
			values2:   []int{777, 400},
			expected: map[netip.Prefix]int{
				mpp("10.0.0.0/8"):     100,
				mpp("192.168.1.0/24"): 777,
				mpp("172.16.0.0/12"):  300,
				mpp("203.0.113.0/24"): 400,
			},
		},
		{
			name:      "Mixed IPv4 and IPv6",
			prefixes1: []string{"10.0.0.0/8", "2001:db8::/32"},
			values1:   []int{100, 200},
			prefixes2: []string{"192.168.0.0/16", "2001:db8::/32", "::1/128"},
			values2:   []int{300, 555, 400},
			expected: map[netip.Prefix]int{
				mpp("10.0.0.0/8"):     100,
				mpp("2001:db8::/32"):  555,
				mpp("192.168.0.0/16"): 300,
				mpp("::1/128"):        400,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Test Union for each table type
			t.Run("Table_Union", func(t *testing.T) {
				t.Parallel()
				tbl1 := new(Table[int])
				tbl2 := new(Table[int])

				for i, pfxStr := range tc.prefixes1 {
					tbl1.Insert(mpp(pfxStr), tc.values1[i])
				}
				for i, pfxStr := range tc.prefixes2 {
					tbl2.Insert(mpp(pfxStr), tc.values2[i])
				}

				tbl1.Union(tbl2)

				verifyResults[int](t, tbl1, tc.expected)
			})

			t.Run("Fast_Union", func(t *testing.T) {
				t.Parallel()
				tbl1 := new(Fast[int])
				tbl2 := new(Fast[int])

				for i, pfxStr := range tc.prefixes1 {
					tbl1.Insert(mpp(pfxStr), tc.values1[i])
				}
				for i, pfxStr := range tc.prefixes2 {
					tbl2.Insert(mpp(pfxStr), tc.values2[i])
				}

				tbl1.Union(tbl2)

				verifyResults[int](t, tbl1, tc.expected)
			})

			t.Run("Lite_Union", func(t *testing.T) {
				t.Parallel()
				tbl1 := new(Lite)
				tbl2 := new(Lite)

				for _, pfxStr := range tc.prefixes1 {
					tbl1.Insert(mpp(pfxStr))
				}
				for _, pfxStr := range tc.prefixes2 {
					tbl2.Insert(mpp(pfxStr))
				}

				tbl1.Union(tbl2)

				// Only verify prefix presence for Lite
				verifyPrefixPresence(t, tbl1, tc.expected)
			})

			// Test UnionPersist for Table
			t.Run("Table_UnionPersist", func(t *testing.T) {
				t.Parallel()
				tbl1 := new(Table[int])
				tbl2 := new(Table[int])

				for i, pfxStr := range tc.prefixes1 {
					tbl1.Insert(mpp(pfxStr), tc.values1[i])
				}
				for i, pfxStr := range tc.prefixes2 {
					tbl2.Insert(mpp(pfxStr), tc.values2[i])
				}

				// Save original state
				original1 := captureTableState(tbl1)

				result := tbl1.UnionPersist(tbl2)

				// Verify immutability
				current1 := captureTableState(tbl1)
				if !maps.Equal(original1, current1) {
					t.Fatal("UnionPersist modified original table")
				}

				verifyResults[int](t, result, tc.expected)
			})

			// Test UnionPersist for Fast
			t.Run("Fast_UnionPersist", func(t *testing.T) {
				t.Parallel()
				tbl1 := new(Fast[int])
				tbl2 := new(Fast[int])

				for i, pfxStr := range tc.prefixes1 {
					tbl1.Insert(mpp(pfxStr), tc.values1[i])
				}
				for i, pfxStr := range tc.prefixes2 {
					tbl2.Insert(mpp(pfxStr), tc.values2[i])
				}

				// Save original state
				original1 := captureFastState(tbl1)

				result := tbl1.UnionPersist(tbl2)

				// Verify immutability
				current1 := captureFastState(tbl1)
				if !maps.Equal(original1, current1) {
					t.Fatal("UnionPersist modified original table")
				}

				verifyResults[int](t, result, tc.expected)
			})

			// Test Lite_UnionPersist
			t.Run("Lite_UnionPersist", func(t *testing.T) {
				t.Parallel()
				tbl1 := new(Lite)
				tbl2 := new(Lite)

				for _, pfxStr := range tc.prefixes1 {
					tbl1.Insert(mpp(pfxStr))
				}
				for _, pfxStr := range tc.prefixes2 {
					tbl2.Insert(mpp(pfxStr))
				}

				// Save original state
				original1 := captureLiteState(tbl1)

				result := tbl1.UnionPersist(tbl2)

				// Verify immutability
				current1 := captureLiteState(tbl1)
				if !maps.Equal(original1, current1) {
					t.Fatal("UnionPersist modified original table")
				}

				verifyPrefixPresence(t, result, tc.expected)
			})
		})
	}
}

func testUnionTable(t *testing.T, prefixes1, prefixes2 []goldTableItem[int]) {
	tbl1 := new(Table[int])
	tbl2 := new(Table[int])

	// Populate tables
	expected := make(map[netip.Prefix]int)
	for _, item := range prefixes1 {
		tbl1.Insert(item.pfx, item.val)
		expected[item.pfx] = item.val
	}
	for _, item := range prefixes2 {
		tbl2.Insert(item.pfx, item.val+10000)
		expected[item.pfx] = item.val + 10000 // tbl2 overwrites
	}

	// Perform union
	tbl1.Union(tbl2)

	// Verify results
	actual := make(map[netip.Prefix]int)
	for pfx, val := range tbl1.All() {
		actual[pfx] = val
	}

	if len(actual) != len(expected) {
		t.Fatalf("Expected %d prefixes, got %d", len(expected), len(actual))
	}

	for pfx, expectedVal := range expected {
		actualVal, found := actual[pfx]
		if !found {
			t.Fatalf("Expected prefix %v not found", pfx)
		}
		if actualVal != expectedVal {
			t.Fatalf("Prefix %v: expected %d, got %d", pfx, expectedVal, actualVal)
		}
	}
}

func testUnionFast(t *testing.T, prefixes1, prefixes2 []goldTableItem[int]) {
	tbl1 := new(Fast[int])
	tbl2 := new(Fast[int])

	// Populate tables
	expected := make(map[netip.Prefix]int)
	for _, item := range prefixes1 {
		tbl1.Insert(item.pfx, item.val)
		expected[item.pfx] = item.val
	}
	for _, item := range prefixes2 {
		tbl2.Insert(item.pfx, item.val+10000)
		expected[item.pfx] = item.val + 10000 // tbl2 overwrites
	}

	// Perform union
	tbl1.Union(tbl2)

	// Verify results
	actual := make(map[netip.Prefix]int)
	for pfx, val := range tbl1.All() {
		actual[pfx] = val
	}

	if len(actual) != len(expected) {
		t.Fatalf("Expected %d prefixes, got %d", len(expected), len(actual))
	}

	for pfx, expectedVal := range expected {
		actualVal, found := actual[pfx]
		if !found {
			t.Fatalf("Expected prefix %v not found", pfx)
		}
		if actualVal != expectedVal {
			t.Fatalf("Prefix %v: expected %d, got %d", pfx, expectedVal, actualVal)
		}
	}
}

func testUnionLite(t *testing.T, prefixes1, prefixes2 []goldTableItem[int]) {
	tbl1 := new(Lite)
	tbl2 := new(Lite)

	// Populate tables - Lite only stores prefixes, not values
	expectedPrefixes := make(map[netip.Prefix]bool)
	for _, item := range prefixes1 {
		tbl1.Insert(item.pfx)
		expectedPrefixes[item.pfx] = true
	}
	for _, item := range prefixes2 {
		tbl2.Insert(item.pfx)
		expectedPrefixes[item.pfx] = true // Union includes all prefixes
	}

	// Perform union
	tbl1.Union(tbl2)

	// Verify results
	actualPrefixes := make(map[netip.Prefix]bool)
	for pfx := range tbl1.All() {
		actualPrefixes[pfx] = true
	}

	if len(actualPrefixes) != len(expectedPrefixes) {
		t.Fatalf("Expected %d prefixes, got %d", len(expectedPrefixes), len(actualPrefixes))
	}

	for pfx := range expectedPrefixes {
		if !actualPrefixes[pfx] {
			t.Fatalf("Expected prefix %v not found", pfx)
		}
	}
}

func testUnionPersistTable(t *testing.T, prefixes1, prefixes2 []goldTableItem[int]) {
	tbl1 := new(Table[int])
	tbl2 := new(Table[int])

	// Populate tables
	expected := make(map[netip.Prefix]int)
	for _, item := range prefixes1 {
		tbl1.Insert(item.pfx, item.val)
		expected[item.pfx] = item.val
	}
	for _, item := range prefixes2 {
		tbl2.Insert(item.pfx, item.val+10000)
		expected[item.pfx] = item.val + 10000 // tbl2 overwrites
	}

	// Save original states
	original1 := captureTableState(tbl1)
	original2 := captureTableState(tbl2)

	// Perform persistent union
	result := tbl1.UnionPersist(tbl2)

	// Verify immutability
	current1 := captureTableState(tbl1)
	current2 := captureTableState(tbl2)

	if !maps.Equal(original1, current1) {
		t.Fatal("UnionPersist modified tbl1")
	}
	if !maps.Equal(original2, current2) {
		t.Fatal("UnionPersist modified tbl2")
	}

	// Verify result
	actual := captureTableState(result)
	if len(actual) != len(expected) {
		t.Fatalf("Expected %d prefixes, got %d", len(expected), len(actual))
	}

	for pfx, expectedVal := range expected {
		actualVal, found := actual[pfx]
		if !found {
			t.Fatalf("Expected prefix %v not found", pfx)
		}
		if actualVal != expectedVal {
			t.Fatalf("Prefix %v: expected %d, got %d", pfx, expectedVal, actualVal)
		}
	}
}

func testUnionPersistFast(t *testing.T, prefixes1, prefixes2 []goldTableItem[int]) {
	tbl1 := new(Fast[int])
	tbl2 := new(Fast[int])

	// Populate tables
	expected := make(map[netip.Prefix]int)
	for _, item := range prefixes1 {
		tbl1.Insert(item.pfx, item.val)
		expected[item.pfx] = item.val
	}
	for _, item := range prefixes2 {
		tbl2.Insert(item.pfx, item.val+10000)
		expected[item.pfx] = item.val + 10000 // tbl2 overwrites
	}

	// Save original states
	original1 := captureFastState(tbl1)
	original2 := captureFastState(tbl2)

	// Perform persistent union
	result := tbl1.UnionPersist(tbl2)

	// Verify immutability
	current1 := captureFastState(tbl1)
	current2 := captureFastState(tbl2)

	if !maps.Equal(original1, current1) {
		t.Fatal("UnionPersist modified tbl1")
	}
	if !maps.Equal(original2, current2) {
		t.Fatal("UnionPersist modified tbl2")
	}

	// Verify result
	actual := captureFastState(result)
	if len(actual) != len(expected) {
		t.Fatalf("Expected %d prefixes, got %d", len(expected), len(actual))
	}

	for pfx, expectedVal := range expected {
		actualVal, found := actual[pfx]
		if !found {
			t.Fatalf("Expected prefix %v not found", pfx)
		}
		if actualVal != expectedVal {
			t.Fatalf("Prefix %v: expected %d, got %d", pfx, expectedVal, actualVal)
		}
	}
}

func testUnionPersistLite(t *testing.T, prefixes1, prefixes2 []goldTableItem[int]) {
	tbl1 := new(Lite)
	tbl2 := new(Lite)

	// Populate tables - Lite only stores prefixes
	expectedPrefixes := make(map[netip.Prefix]bool)
	for _, item := range prefixes1 {
		tbl1.Insert(item.pfx)
		expectedPrefixes[item.pfx] = true
	}
	for _, item := range prefixes2 {
		tbl2.Insert(item.pfx)
		expectedPrefixes[item.pfx] = true // Union includes all prefixes
	}

	// Save original states
	original1 := captureLiteState(tbl1)
	original2 := captureLiteState(tbl2)

	// Perform persistent union
	result := tbl1.UnionPersist(tbl2)

	// Verify immutability
	current1 := captureLiteState(tbl1)
	current2 := captureLiteState(tbl2)

	if !maps.Equal(original1, current1) {
		t.Fatal("UnionPersist modified tbl1")
	}
	if !maps.Equal(original2, current2) {
		t.Fatal("UnionPersist modified tbl2")
	}

	// Verify result
	actualPrefixes := captureLiteState(result)
	if len(actualPrefixes) != len(expectedPrefixes) {
		t.Fatalf("Expected %d prefixes, got %d", len(expectedPrefixes), len(actualPrefixes))
	}

	for pfx := range expectedPrefixes {
		if !actualPrefixes[pfx] {
			t.Fatalf("Expected prefix %v not found", pfx)
		}
	}
}

// Helper functions

func captureTableState[V comparable](t *Table[V]) map[netip.Prefix]V {
	state := make(map[netip.Prefix]V)
	for pfx, val := range t.All() {
		state[pfx] = val
	}
	return state
}

func verifyResults[T any](t *testing.T, tbl interface {
	All() iter.Seq2[netip.Prefix, int]
}, expected map[netip.Prefix]int,
) {
	t.Helper()

	actual := make(map[netip.Prefix]int)
	for pfx, val := range tbl.All() {
		actual[pfx] = val
	}

	if len(actual) != len(expected) {
		t.Fatalf("Expected %d prefixes, got %d", len(expected), len(actual))
	}

	for pfx, expectedVal := range expected {
		actualVal, found := actual[pfx]
		if !found {
			t.Errorf("Expected prefix %s not found", pfx)
			continue
		}
		if actualVal != expectedVal {
			t.Errorf("Prefix %s: expected value %d, got %d", pfx, expectedVal, actualVal)
		}
	}
}

func verifyPrefixPresence(t *testing.T, tbl *Lite, expected map[netip.Prefix]int) {
	t.Helper()

	actual := make(map[netip.Prefix]bool)
	for pfx := range tbl.All() {
		actual[pfx] = true
	}

	if len(actual) != len(expected) {
		t.Fatalf("Expected %d prefixes, got %d", len(expected), len(actual))
	}

	for pfx := range expected {
		if !actual[pfx] {
			t.Errorf("Expected prefix %s not found", pfx)
		}
	}
}

func captureFastState[V comparable](t *Fast[V]) map[netip.Prefix]V {
	state := make(map[netip.Prefix]V)
	for pfx, val := range t.All() {
		state[pfx] = val
	}
	return state
}

func captureLiteState(t *Lite) map[netip.Prefix]bool {
	state := make(map[netip.Prefix]bool)
	for pfx := range t.All() {
		state[pfx] = true
	}
	return state
}
