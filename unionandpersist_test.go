// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"fmt"
	"iter"
	"maps"
	"math/rand/v2"
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
		expected  map[string]int
	}{
		{
			name:      "No overlap",
			prefixes1: []string{"10.0.0.0/8", "192.168.1.0/24"},
			values1:   []int{100, 200},
			prefixes2: []string{"172.16.0.0/12", "203.0.113.0/24"},
			values2:   []int{300, 400},
			expected: map[string]int{
				"10.0.0.0/8":     100,
				"192.168.1.0/24": 200,
				"172.16.0.0/12":  300,
				"203.0.113.0/24": 400,
			},
		},
		{
			name:      "Complete overlap - tbl2 values should win",
			prefixes1: []string{"10.0.0.0/8", "192.168.1.0/24"},
			values1:   []int{100, 200},
			prefixes2: []string{"10.0.0.0/8", "192.168.1.0/24"},
			values2:   []int{999, 888},
			expected: map[string]int{
				"10.0.0.0/8":     999,
				"192.168.1.0/24": 888,
			},
		},
		{
			name:      "Partial overlap",
			prefixes1: []string{"10.0.0.0/8", "192.168.1.0/24", "172.16.0.0/12"},
			values1:   []int{100, 200, 300},
			prefixes2: []string{"192.168.1.0/24", "203.0.113.0/24"},
			values2:   []int{777, 400},
			expected: map[string]int{
				"10.0.0.0/8":     100,
				"192.168.1.0/24": 777,
				"172.16.0.0/12":  300,
				"203.0.113.0/24": 400,
			},
		},
		{
			name:      "Mixed IPv4 and IPv6",
			prefixes1: []string{"10.0.0.0/8", "2001:db8::/32"},
			values1:   []int{100, 200},
			prefixes2: []string{"192.168.0.0/16", "2001:db8::/32", "::1/128"},
			values2:   []int{300, 555, 400},
			expected: map[string]int{
				"10.0.0.0/8":     100,
				"2001:db8::/32":  555,
				"192.168.0.0/16": 300,
				"::1/128":        400,
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

			t.Run("liteTable_Union", func(t *testing.T) {
				t.Parallel()
				tbl1 := new(liteTable[int])
				tbl2 := new(liteTable[int])

				for i, pfxStr := range tc.prefixes1 {
					tbl1.Insert(mpp(pfxStr), tc.values1[i])
				}
				for i, pfxStr := range tc.prefixes2 {
					tbl2.Insert(mpp(pfxStr), tc.values2[i])
				}

				tbl1.Union(tbl2)

				// Only verify prefix presence for liteTable
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
		})
	}
}

// BenchmarkUnion benchmarks Union performance
func BenchmarkUnion(b *testing.B) {
	sizes := []struct {
		size1, size2 int
	}{
		{100, 50},
		{1000, 500},
		{5000, 2500},
	}

	for _, size := range sizes {
		// Setup test data once
		prng := rand.New(rand.NewPCG(42, 42))
		prefixes1 := randomPrefixes(prng, size.size1)
		prefixes2 := randomPrefixes(prng, size.size2)

		b.Run(fmt.Sprintf("Table_%dx%d", size.size1, size.size2), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				tbl1 := new(Table[int])
				tbl2 := new(Table[int])

				for _, item := range prefixes1 {
					tbl1.Insert(item.pfx, item.val)
				}
				for _, item := range prefixes2 {
					tbl2.Insert(item.pfx, item.val)
				}

				tbl1.Union(tbl2)
			}
		})

		b.Run(fmt.Sprintf("Fast_%dx%d", size.size1, size.size2), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				tbl1 := new(Fast[int])
				tbl2 := new(Fast[int])

				for _, item := range prefixes1 {
					tbl1.Insert(item.pfx, item.val)
				}
				for _, item := range prefixes2 {
					tbl2.Insert(item.pfx, item.val)
				}

				tbl1.Union(tbl2)
			}
		})

		b.Run(fmt.Sprintf("liteTable_%dx%d", size.size1, size.size2), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				tbl1 := new(liteTable[int])
				tbl2 := new(liteTable[int])

				for _, item := range prefixes1 {
					tbl1.Insert(item.pfx, item.val)
				}
				for _, item := range prefixes2 {
					tbl2.Insert(item.pfx, item.val)
				}

				tbl1.Union(tbl2)
			}
		})
	}
}

// BenchmarkUnionPersist benchmarks UnionPersist performance
func BenchmarkUnionPersist(b *testing.B) {
	sizes := []struct {
		size1, size2 int
	}{
		{100, 50},
		{1000, 500},
		{2000, 1000},
	}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Table_%dx%d", size.size1, size.size2), func(b *testing.B) {
			// Setup test data
			prng := rand.New(rand.NewPCG(42, 42))
			prefixes1 := randomPrefixes(prng, size.size1)
			prefixes2 := randomPrefixes(prng, size.size2)

			// Pre-populate tables
			tbl1 := new(Table[int])
			tbl2 := new(Table[int])
			for _, item := range prefixes1 {
				tbl1.Insert(item.pfx, item.val)
			}
			for _, item := range prefixes2 {
				tbl2.Insert(item.pfx, item.val)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_ = tbl1.UnionPersist(tbl2)
			}
		})
	}
}

// FuzzUnion tests that Union correctly merges two tables
func FuzzTableUnion(f *testing.F) {
	// Seed with some initial test cases
	f.Add(uint64(12345), 50, 30)
	f.Add(uint64(67890), 100, 75)
	f.Add(uint64(11111), 200, 150)

	f.Fuzz(func(t *testing.T, seed uint64, count1, count2 int) {
		// Bound the test size to reasonable limits
		if count1 < 5 || count1 > 500 || count2 < 5 || count2 > 500 {
			t.Skip("counts out of range")
		}

		// Generate random prefixes for both tables
		prng := rand.New(rand.NewPCG(seed, 42))
		prefixes1 := randomPrefixes(prng, count1)
		prefixes2 := randomPrefixes(prng, count2)

		t.Run("Table", func(t *testing.T) {
			testUnionTable(t, prefixes1, prefixes2)
		})
	})
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

// FuzzFastUnion tests that Fast.Union correctly merges two tables
func FuzzFastUnion(f *testing.F) {
	// Seed with some initial test cases
	f.Add(uint64(12345), 50, 30)
	f.Add(uint64(67890), 100, 75)
	f.Add(uint64(11111), 200, 150)

	f.Fuzz(func(t *testing.T, seed uint64, count1, count2 int) {
		// Bound the test size to reasonable limits
		if count1 < 5 || count1 > 500 || count2 < 5 || count2 > 500 {
			t.Skip("counts out of range")
		}

		// Generate random prefixes for both tables
		prng := rand.New(rand.NewPCG(seed, 42))
		prefixes1 := randomPrefixes(prng, count1)
		prefixes2 := randomPrefixes(prng, count2)

		t.Run("Fast", func(t *testing.T) {
			testUnionFast(t, prefixes1, prefixes2)
		})
	})
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

// FuzzLiteUnion tests that Lite.Union correctly merges two tables
func FuzzLiteUnion(f *testing.F) {
	// Seed with some initial test cases
	f.Add(uint64(12345), 50, 30)
	f.Add(uint64(67890), 100, 75)
	f.Add(uint64(11111), 200, 150)

	f.Fuzz(func(t *testing.T, seed uint64, count1, count2 int) {
		// Bound the test size to reasonable limits
		if count1 < 5 || count1 > 500 || count2 < 5 || count2 > 500 {
			t.Skip("counts out of range")
		}

		// Generate random prefixes for both tables
		prng := rand.New(rand.NewPCG(seed, 42))
		prefixes1 := randomPrefixes(prng, count1)
		prefixes2 := randomPrefixes(prng, count2)

		t.Run("Lite", func(t *testing.T) {
			testUnionLite(t, prefixes1, prefixes2)
		})
	})
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

// FuzzUnionPersist tests that UnionPersist correctly merges without modifying originals
func FuzzUnionPersist(f *testing.F) {
	// Seed with some initial test cases
	f.Add(uint64(54321), 40, 60)
	f.Add(uint64(98765), 80, 120)
	f.Add(uint64(22222), 150, 100)

	f.Fuzz(func(t *testing.T, seed uint64, count1, count2 int) {
		// Bound the test size to reasonable limits
		if count1 < 5 || count1 > 300 || count2 < 5 || count2 > 300 {
			t.Skip("counts out of range")
		}

		// Generate random prefixes for both tables
		prng := rand.New(rand.NewPCG(seed, 42))
		prefixes1 := randomPrefixes(prng, count1)
		prefixes2 := randomPrefixes(prng, count2)

		t.Run("Table", func(t *testing.T) {
			testUnionPersistTable(t, prefixes1, prefixes2)
		})
	})
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

// FuzzFastUnionPersist tests that Fast.UnionPersist correctly merges without modifying originals
func FuzzFastUnionPersist(f *testing.F) {
	// Seed with some initial test cases
	f.Add(uint64(54321), 40, 60)
	f.Add(uint64(98765), 80, 120)
	f.Add(uint64(22222), 150, 100)

	f.Fuzz(func(t *testing.T, seed uint64, count1, count2 int) {
		// Bound the test size to reasonable limits
		if count1 < 5 || count1 > 300 || count2 < 5 || count2 > 300 {
			t.Skip("counts out of range")
		}

		// Generate random prefixes for both tables
		prng := rand.New(rand.NewPCG(seed, 42))
		prefixes1 := randomPrefixes(prng, count1)
		prefixes2 := randomPrefixes(prng, count2)

		t.Run("Fast", func(t *testing.T) {
			testUnionPersistFast(t, prefixes1, prefixes2)
		})
	})
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

// FuzzLiteUnionPersist tests that Lite.UnionPersist correctly merges without modifying originals
func FuzzLiteUnionPersist(f *testing.F) {
	// Seed with some initial test cases
	f.Add(uint64(54321), 40, 60)
	f.Add(uint64(98765), 80, 120)
	f.Add(uint64(22222), 150, 100)

	f.Fuzz(func(t *testing.T, seed uint64, count1, count2 int) {
		// Bound the test size to reasonable limits
		if count1 < 5 || count1 > 300 || count2 < 5 || count2 > 300 {
			t.Skip("counts out of range")
		}

		// Generate random prefixes for both tables
		prng := rand.New(rand.NewPCG(seed, 42))
		prefixes1 := randomPrefixes(prng, count1)
		prefixes2 := randomPrefixes(prng, count2)

		t.Run("Lite", func(t *testing.T) {
			testUnionPersistLite(t, prefixes1, prefixes2)
		})
	})
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

// FuzzUnionPersistAliasing tests for potential aliasing/memory sharing bugs
func FuzzUnionPersistAliasing(f *testing.F) {
	// Seed with initial test cases
	f.Add(uint64(12345), uint64(67890), 20, 20, 5)
	f.Add(uint64(11111), uint64(22222), 50, 30, 10)
	f.Add(uint64(99999), uint64(88888), 100, 100, 20)

	f.Fuzz(func(t *testing.T, seed1, seed2 uint64, count1, count2, modifyCount int) {
		// Bound test sizes
		if count1 < 5 || count1 > 200 || count2 < 5 || count2 > 200 || modifyCount < 1 || modifyCount > 50 {
			t.Skip("counts out of range")
		}

		// Generate test data
		prng1 := rand.New(rand.NewPCG(seed1, 42))
		prng2 := rand.New(rand.NewPCG(seed2, 42))

		prefixes1 := randomPrefixes(prng1, count1)
		prefixes2 := randomPrefixes(prng2, count2)
		modifyPrefixes := randomPrefixes(prng1, modifyCount)

		// Create and populate original tables
		tbl1 := new(Table[int])
		tbl2 := new(Table[int])

		for _, item := range prefixes1 {
			tbl1.Insert(item.pfx, item.val)
		}
		for _, item := range prefixes2 {
			tbl2.Insert(item.pfx, item.val+10000)
		}

		// Capture state before UnionPersist
		tbl1BeforeState := captureTableState(tbl1)
		tbl2BeforeState := captureTableState(tbl2)

		// Perform UnionPersist
		resultTable := tbl1.UnionPersist(tbl2)

		// Capture initial result state
		resultInitialState := captureTableState(resultTable)

		// TEST 1: Verify original tables unchanged
		if !maps.Equal(tbl1BeforeState, captureTableState(tbl1)) {
			t.Fatal("UnionPersist modified tbl1 (immutability violation)")
		}
		if !maps.Equal(tbl2BeforeState, captureTableState(tbl2)) {
			t.Fatal("UnionPersist modified tbl2 (immutability violation)")
		}

		// TEST 2: Modify original tbl1 persistent - result should NOT change
		for _, item := range modifyPrefixes {
			tbl1 = tbl1.InsertPersist(item.pfx, item.val+20000)
		}
		tbl1State2 := captureTableState(tbl1)

		if !maps.Equal(resultInitialState, captureTableState(resultTable)) {
			t.Fatal("Result table changed after modifying tbl1 (aliasing bug)")
		}

		// TEST 3: Modify original tbl2 persistent - result should NOT change
		for _, item := range modifyPrefixes {
			tbl2 = tbl2.InsertPersist(item.pfx, item.val+30000)
		}
		tbl2State2 := captureTableState(tbl2)

		if !maps.Equal(resultInitialState, captureTableState(resultTable)) {
			t.Fatal("Result table changed after modifying tbl2 (aliasing bug)")
		}

		// TEST 4: Modify result table persistent - original tables should NOT change
		for _, item := range modifyPrefixes {
			resultTable = resultTable.InsertPersist(item.pfx, item.val+40000)
		}

		if !maps.Equal(tbl1State2, captureTableState(tbl1)) {
			t.Fatal("tbl1 changed after modifying result (reverse aliasing)")
		}
		if !maps.Equal(tbl2State2, captureTableState(tbl2)) {
			t.Fatal("tbl2 changed after modifying result (reverse aliasing)")
		}

		// TEST 5: Multiple UnionPersist operations should be independent
		resultTable2 := tbl1.UnionPersist(tbl2)
		resultTable3 := resultTable.UnionPersist(tbl1)

		// Modify resultTable2
		testPrefix := mpp("10.99.99.0/24")
		_ = resultTable2.InsertPersist(testPrefix, 55555)

		// resultTable3 should not be affected
		if val3, found := resultTable3.Get(testPrefix); found && val3 == 55555 {
			t.Fatal("UnionPersist results share memory (deep aliasing bug)")
		}

		// TEST 6: Nested UnionPersist chain
		chain1 := tbl1.UnionPersist(tbl2)
		chain2 := chain1.UnionPersist(tbl1)
		chain3 := chain2.UnionPersist(tbl2)

		// All should be independent
		testPrefix2 := mpp("192.168.99.0/24")
		chain1 = chain1.InsertPersist(testPrefix2, 111)
		chain2 = chain2.InsertPersist(testPrefix2, 222)
		chain3 = chain3.InsertPersist(testPrefix2, 333)

		val1, _ := chain1.Get(testPrefix2)
		val2, _ := chain2.Get(testPrefix2)
		val3, _ := chain3.Get(testPrefix2)

		if val1 == val2 || val2 == val3 || val1 == val3 {
			t.Fatalf("Chained UnionPersist tables share state: %d, %d, %d", val1, val2, val3)
		}
	})
}

// FuzzFastUnionPersistAliasing tests for potential aliasing/memory sharing bugs in Fast
func FuzzFastUnionPersistAliasing(f *testing.F) {
	// Seed with initial test cases
	f.Add(uint64(12345), uint64(67890), 20, 20, 5)
	f.Add(uint64(11111), uint64(22222), 50, 30, 10)
	f.Add(uint64(99999), uint64(88888), 100, 100, 20)

	f.Fuzz(func(t *testing.T, seed1, seed2 uint64, count1, count2, modifyCount int) {
		// Bound test sizes
		if count1 < 5 || count1 > 200 || count2 < 5 || count2 > 200 || modifyCount < 1 || modifyCount > 50 {
			t.Skip("counts out of range")
		}

		// Generate test data
		prng1 := rand.New(rand.NewPCG(seed1, 42))
		prng2 := rand.New(rand.NewPCG(seed2, 42))

		prefixes1 := randomPrefixes(prng1, count1)
		prefixes2 := randomPrefixes(prng2, count2)
		modifyPrefixes := randomPrefixes(prng1, modifyCount)

		// Create and populate original tables
		tbl1 := new(Fast[int])
		tbl2 := new(Fast[int])

		for _, item := range prefixes1 {
			tbl1.Insert(item.pfx, item.val)
		}
		for _, item := range prefixes2 {
			tbl2.Insert(item.pfx, item.val+10000)
		}

		// Capture state before UnionPersist
		tbl1BeforeState := captureFastState(tbl1)
		tbl2BeforeState := captureFastState(tbl2)

		// Perform UnionPersist
		resultTable := tbl1.UnionPersist(tbl2)

		// Capture initial result state
		resultInitialState := captureFastState(resultTable)

		// TEST 1: Verify original tables unchanged
		if !maps.Equal(tbl1BeforeState, captureFastState(tbl1)) {
			t.Fatal("UnionPersist modified tbl1 (immutability violation)")
		}
		if !maps.Equal(tbl2BeforeState, captureFastState(tbl2)) {
			t.Fatal("UnionPersist modified tbl2 (immutability violation)")
		}

		// TEST 2: Modify original tbl1 persistent - result should NOT change
		for _, item := range modifyPrefixes {
			tbl1 = tbl1.InsertPersist(item.pfx, item.val+20000)
		}
		tbl1State2 := captureFastState(tbl1)

		if !maps.Equal(resultInitialState, captureFastState(resultTable)) {
			t.Fatal("Result table changed after modifying tbl1 (aliasing bug)")
		}

		// TEST 3: Modify original tbl2 persistent - result should NOT change
		for _, item := range modifyPrefixes {
			tbl2 = tbl2.InsertPersist(item.pfx, item.val+30000)
		}
		tbl2State2 := captureFastState(tbl2)

		if !maps.Equal(resultInitialState, captureFastState(resultTable)) {
			t.Fatal("Result table changed after modifying tbl2 (aliasing bug)")
		}

		// TEST 4: Modify result table persistent - original tables should NOT change
		for _, item := range modifyPrefixes {
			resultTable = resultTable.InsertPersist(item.pfx, item.val+40000)
		}

		if !maps.Equal(tbl1State2, captureFastState(tbl1)) {
			t.Fatal("tbl1 changed after modifying result (reverse aliasing)")
		}
		if !maps.Equal(tbl2State2, captureFastState(tbl2)) {
			t.Fatal("tbl2 changed after modifying result (reverse aliasing)")
		}

		// TEST 5: Multiple UnionPersist operations should be independent
		resultTable2 := tbl1.UnionPersist(tbl2)
		resultTable3 := resultTable.UnionPersist(tbl1)

		// Modify resultTable2
		testPrefix := mpp("10.99.99.0/24")
		_ = resultTable2.InsertPersist(testPrefix, 55555)

		// resultTable3 should not be affected
		if val3, found := resultTable3.Get(testPrefix); found && val3 == 55555 {
			t.Fatal("UnionPersist results share memory (deep aliasing bug)")
		}
	})
}

// FuzzLiteUnionPersistAliasing tests for potential aliasing/memory sharing bugs in Lite
func FuzzLiteUnionPersistAliasing(f *testing.F) {
	// Seed with initial test cases
	f.Add(uint64(12345), uint64(67890), 20, 20, 5)
	f.Add(uint64(11111), uint64(22222), 50, 30, 10)
	f.Add(uint64(99999), uint64(88888), 100, 100, 20)

	f.Fuzz(func(t *testing.T, seed1, seed2 uint64, count1, count2, modifyCount int) {
		// Bound test sizes
		if count1 < 5 || count1 > 200 || count2 < 5 || count2 > 200 || modifyCount < 1 || modifyCount > 50 {
			t.Skip("counts out of range")
		}

		// Generate test data
		prng1 := rand.New(rand.NewPCG(seed1, 42))
		prng2 := rand.New(rand.NewPCG(seed2, 42))

		prefixes1 := randomPrefixes(prng1, count1)
		prefixes2 := randomPrefixes(prng2, count2)
		modifyPrefixes := randomPrefixes(prng1, modifyCount)

		// Create and populate original tables
		tbl1 := new(Lite)
		tbl2 := new(Lite)

		for _, item := range prefixes1 {
			tbl1.Insert(item.pfx)
		}
		for _, item := range prefixes2 {
			tbl2.Insert(item.pfx)
		}

		// Capture state before UnionPersist
		tbl1BeforeState := captureLiteState(tbl1)
		tbl2BeforeState := captureLiteState(tbl2)

		// Perform UnionPersist
		resultTable := tbl1.UnionPersist(tbl2)

		// Capture initial result state
		resultInitialState := captureLiteState(resultTable)

		// TEST 1: Verify original tables unchanged
		if !maps.Equal(tbl1BeforeState, captureLiteState(tbl1)) {
			t.Fatal("UnionPersist modified tbl1 (immutability violation)")
		}
		if !maps.Equal(tbl2BeforeState, captureLiteState(tbl2)) {
			t.Fatal("UnionPersist modified tbl2 (immutability violation)")
		}

		// TEST 2: Modify original tbl1 persistent - result should NOT change
		for _, item := range modifyPrefixes {
			tbl1 = tbl1.InsertPersist(item.pfx)
		}
		tbl1State2 := captureLiteState(tbl1)

		if !maps.Equal(resultInitialState, captureLiteState(resultTable)) {
			t.Fatal("Result table changed after modifying tbl1 (aliasing bug)")
		}

		// TEST 3: Modify original tbl2 persistent - result should NOT change
		for _, item := range modifyPrefixes {
			tbl2 = tbl2.InsertPersist(item.pfx)
		}
		tbl2State2 := captureLiteState(tbl2)

		if !maps.Equal(resultInitialState, captureLiteState(resultTable)) {
			t.Fatal("Result table changed after modifying tbl2 (aliasing bug)")
		}

		// TEST 4: Modify result table persistent - original tables should NOT change
		for _, item := range modifyPrefixes {
			resultTable = resultTable.InsertPersist(item.pfx)
		}

		if !maps.Equal(tbl1State2, captureLiteState(tbl1)) {
			t.Fatal("tbl1 changed after modifying result (reverse aliasing)")
		}
		if !maps.Equal(tbl2State2, captureLiteState(tbl2)) {
			t.Fatal("tbl2 changed after modifying result (reverse aliasing)")
		}
	})
}

// Extend TestUnionDeterministic to include Fast and Lite
func TestUnionDeterministicExtended(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		prefixes1 []string
		values1   []int
		prefixes2 []string
		values2   []int
		expected  map[string]int
	}{
		{
			name:      "No overlap",
			prefixes1: []string{"10.0.0.0/8", "192.168.1.0/24"},
			values1:   []int{100, 200},
			prefixes2: []string{"172.16.0.0/12", "203.0.113.0/24"},
			values2:   []int{300, 400},
			expected: map[string]int{
				"10.0.0.0/8":     100,
				"192.168.1.0/24": 200,
				"172.16.0.0/12":  300,
				"203.0.113.0/24": 400,
			},
		},
		{
			name:      "Complete overlap - tbl2 values should win",
			prefixes1: []string{"10.0.0.0/8", "192.168.1.0/24"},
			values1:   []int{100, 200},
			prefixes2: []string{"10.0.0.0/8", "192.168.1.0/24"},
			values2:   []int{999, 888},
			expected: map[string]int{
				"10.0.0.0/8":     999,
				"192.168.1.0/24": 888,
			},
		},
		{
			name:      "Mixed IPv4 and IPv6",
			prefixes1: []string{"10.0.0.0/8", "2001:db8::/32"},
			values1:   []int{100, 200},
			prefixes2: []string{"192.168.0.0/16", "2001:db8::/32", "::1/128"},
			values2:   []int{300, 555, 400},
			expected: map[string]int{
				"10.0.0.0/8":     100,
				"2001:db8::/32":  555,
				"192.168.0.0/16": 300,
				"::1/128":        400,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Test Fast_Union
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

			// Test Lite_Union
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

				// For Lite, we only verify prefix presence
				verifyPrefixPresence[struct{}](t, &tbl1.liteTable, tc.expected)
			})

			// Test Fast_UnionPersist
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

				verifyPrefixPresence[struct{}](t, &result.liteTable, tc.expected)
			})
		})
	}
}

// Extended BenchmarkUnionPersist to include Fast and Lite
func BenchmarkUnionPersistExtended(b *testing.B) {
	sizes := []struct {
		size1, size2 int
	}{
		{100, 50},
		{1000, 500},
		{2000, 1000},
	}

	for _, size := range sizes {
		// Setup test data
		prng := rand.New(rand.NewPCG(42, 42))
		prefixes1 := randomPrefixes(prng, size.size1)
		prefixes2 := randomPrefixes(prng, size.size2)

		b.Run(fmt.Sprintf("Fast_%dx%d", size.size1, size.size2), func(b *testing.B) {
			// Pre-populate tables
			tbl1 := new(Fast[int])
			tbl2 := new(Fast[int])
			for _, item := range prefixes1 {
				tbl1.Insert(item.pfx, item.val)
			}
			for _, item := range prefixes2 {
				tbl2.Insert(item.pfx, item.val)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_ = tbl1.UnionPersist(tbl2)
			}
		})

		b.Run(fmt.Sprintf("Lite_%dx%d", size.size1, size.size2), func(b *testing.B) {
			// Pre-populate tables
			tbl1 := new(Lite)
			tbl2 := new(Lite)
			for _, item := range prefixes1 {
				tbl1.Insert(item.pfx)
			}
			for _, item := range prefixes2 {
				tbl2.Insert(item.pfx)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_ = tbl1.UnionPersist(tbl2)
			}
		})
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
}, expected map[string]int,
) {
	t.Helper()

	actual := make(map[string]int)
	for pfx, val := range tbl.All() {
		actual[pfx.String()] = val
	}

	if len(actual) != len(expected) {
		t.Fatalf("Expected %d prefixes, got %d", len(expected), len(actual))
	}

	for pfxStr, expectedVal := range expected {
		actualVal, found := actual[pfxStr]
		if !found {
			t.Errorf("Expected prefix %s not found", pfxStr)
			continue
		}
		if actualVal != expectedVal {
			t.Errorf("Prefix %s: expected value %d, got %d", pfxStr, expectedVal, actualVal)
		}
	}
}

func verifyPrefixPresence[T any](t *testing.T, tbl interface {
	All() iter.Seq2[netip.Prefix, T]
}, expected map[string]int,
) {
	t.Helper()

	actual := make(map[string]bool)
	for pfx := range tbl.All() {
		actual[pfx.String()] = true
	}

	if len(actual) != len(expected) {
		t.Fatalf("Expected %d prefixes, got %d", len(expected), len(actual))
	}

	for pfxStr := range expected {
		if !actual[pfxStr] {
			t.Errorf("Expected prefix %s not found", pfxStr)
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
