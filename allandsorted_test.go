package bart

import (
	"fmt"
	"iter"
	"math/rand/v2"
	"net/netip"
	"testing"
)

// FuzzAllSorted tests that AllSorted() visits all prefixes in natural CIDR sort order
func FuzzAllSorted(f *testing.F) {
	// Seed with some initial test cases
	f.Add(uint64(12345), 100)
	f.Add(uint64(67890), 250)
	f.Add(uint64(11111), 500)

	f.Fuzz(func(t *testing.T, seed uint64, count int) {
		// Bound the test size to reasonable limits
		if count < 10 || count > 1000 {
			t.Skip("count out of range")
		}

		// Generate random prefixes using the existing utility
		prng := rand.New(rand.NewPCG(seed, 42))
		prefixItems := randomPrefixes(prng, count)

		type tabler interface {
			Insert(netip.Prefix, int)
			AllSorted() iter.Seq2[netip.Prefix, int]
		}

		// Test all table types that support AllSorted
		tables := []struct {
			name    string
			builder func() tabler
		}{
			{"Fast", func() tabler { return new(Fast[int]) }},
			{"Table", func() tabler { return new(Table[int]) }},
			{"liteTable", func() tabler { return new(liteTable[int]) }},
		}

		for _, tt := range tables {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				tbl := tt.builder()

				// Insert all prefixes with their values
				for _, item := range prefixItems {
					tbl.Insert(item.pfx, item.val)
				}

				// Collect all prefixes from AllSorted
				sortedPrefixes := make([]netip.Prefix, 0, count)
				sortedValues := make([]int, 0, count)

				for pfx, val := range tbl.AllSorted() {
					sortedPrefixes = append(sortedPrefixes, pfx)
					sortedValues = append(sortedValues, val)
				}

				// Verify we got exactly the same number of prefixes
				if len(sortedPrefixes) != len(prefixItems) {
					t.Fatalf("%s: Expected %d prefixes, got %d", tt.name, len(prefixItems), len(sortedPrefixes))
				}

				// Verify all prefixes are in natural CIDR sort order using existing cmpPrefix function
				for i := 1; i < len(sortedPrefixes); i++ {
					if cmpPrefix(sortedPrefixes[i-1], sortedPrefixes[i]) > 0 {
						t.Fatalf("%s: CIDR sort order violated at index %d: %v should come before %v",
							tt.name, i-1, sortedPrefixes[i-1], sortedPrefixes[i])
					}
				}

				// Verify we can find each original prefix in the sorted results
				// Create a map for O(1) lookup verification
				resultMap := make(map[netip.Prefix]int, count)
				for i, pfx := range sortedPrefixes {
					resultMap[pfx] = sortedValues[i]
				}

				for _, originalItem := range prefixItems {
					val, found := resultMap[originalItem.pfx]
					if !found {
						t.Fatalf("%s: Original prefix %v not found in AllSorted results", tt.name, originalItem.pfx)
					}

					// liteTable has no payload
					if _, ok := tbl.(*liteTable[int]); !ok {
						if val != originalItem.val {
							t.Fatalf("%s: Original prefix %v has wrong value: expected %d, got %d",
								tt.name, originalItem.pfx, originalItem.val, val)
						}
					}
				}

				// Verify no duplicates in results (since randomPrefixes guarantees no duplicates)
				seen := make(map[netip.Prefix]bool, count)
				for _, pfx := range sortedPrefixes {
					if seen[pfx] {
						t.Fatalf("%s: Duplicate prefix %v found in AllSorted results", tt.name, pfx)
					}
					seen[pfx] = true
				}
			})
		}
	})
}

// TestAll tests All with random samples
func TestAll(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	prefixItems := randomPrefixes(prng, n)

	type tabler interface {
		Insert(netip.Prefix, int)
		All() iter.Seq2[netip.Prefix, int]
	}

	// Test all table types that support AllSorted
	tables := []struct {
		name    string
		builder func() tabler
	}{
		{"Fast", func() tabler { return new(Fast[int]) }},
		{"Table", func() tabler { return new(Table[int]) }},
		{"liteTable", func() tabler { return new(liteTable[int]) }},
	}

	for _, tt := range tables {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tbl := tt.builder()

			// Insert all prefixes with their values
			for _, item := range prefixItems {
				tbl.Insert(item.pfx, item.val)
			}

			// Collect all prefixes from AllSorted
			gotPrefixes := make([]netip.Prefix, 0, n)
			gotValues := make([]int, 0, n)

			for pfx, val := range tbl.All() {
				gotPrefixes = append(gotPrefixes, pfx)
				gotValues = append(gotValues, val)
			}

			// Verify we got exactly the same number of prefixes
			if len(gotPrefixes) != len(prefixItems) {
				t.Fatalf("%s: Expected %d prefixes, got %d", tt.name, len(prefixItems), len(gotPrefixes))
			}

			// Verify we can find each original prefix in the sorted results
			// Create a map for O(1) lookup verification
			resultMap := make(map[netip.Prefix]int, n)
			for i, pfx := range gotPrefixes {
				resultMap[pfx] = gotValues[i]
			}

			for _, originalItem := range prefixItems {
				val, found := resultMap[originalItem.pfx]
				if !found {
					t.Fatalf("%s: Original prefix %v not found in All results", tt.name, originalItem.pfx)
				}

				// liteTable has no payload
				if _, ok := tbl.(*liteTable[int]); !ok {
					if val != originalItem.val {
						t.Fatalf("%s: Original prefix %v has wrong value: expected %d, got %d",
							tt.name, originalItem.pfx, originalItem.val, val)
					}
				}
			}

			// Verify no duplicates in results (since randomPrefixes guarantees no duplicates)
			seen := make(map[netip.Prefix]bool, n)
			for _, pfx := range gotPrefixes {
				if seen[pfx] {
					t.Fatalf("%s: Duplicate prefix %v found in AllSorted results", tt.name, pfx)
				}
				seen[pfx] = true
			}
		})
	}
}

// TestAllSortedCIDROrder tests CIDR sort order with known examples
func TestAllSorted(t *testing.T) {
	t.Parallel()

	// Test cases with known CIDR sort order
	testCases := []struct {
		name     string
		prefixes []string
		expected []string // Expected order after sorting
	}{
		{
			name: "Mixed IPv4 addresses and prefix lengths",
			prefixes: []string{
				"10.0.0.0/16",
				"10.0.0.0/8",
				"192.168.1.0/24",
				"10.0.0.0/24",
				"172.16.0.0/12",
			},
			expected: []string{
				"10.0.0.0/8",     // Same address, shorter prefix first
				"10.0.0.0/16",    // Same address, longer prefix
				"10.0.0.0/24",    // Same address, longest prefix
				"172.16.0.0/12",  // Next address
				"192.168.1.0/24", // Highest address
			},
		},
		{
			name: "Mixed IPv6 addresses and prefix lengths",
			prefixes: []string{
				"2001:db8::/32",
				"2001:db8::/64",
				"2000::/16",
				"2001:db8:1::/48",
			},
			expected: []string{
				"2000::/16",       // Lowest address
				"2001:db8::/32",   // Same address, shorter prefix first
				"2001:db8::/64",   // Same address, longer prefix
				"2001:db8:1::/48", // Higher address
			},
		},
		{
			name: "Mixed IPv4 and IPv6",
			prefixes: []string{
				"192.168.1.0/24",
				"2001:db8::/32",
				"10.0.0.0/8",
				"::1/128",
			},
			expected: []string{
				"10.0.0.0/8",     // IPv4 addresses come first (lower in comparison)
				"192.168.1.0/24", // Next IPv4 address
				"::1/128",        // IPv6 addresses after IPv4
				"2001:db8::/32",  // Higher IPv6 address
			},
		},
	}

	type tabler interface {
		Insert(netip.Prefix, int)
		AllSorted() iter.Seq2[netip.Prefix, int]
	}

	// Test all table types that support AllSorted
	tables := []struct {
		name    string
		builder func() tabler
	}{
		{"liteTable", func() tabler { return new(liteTable[int]) }},
		{"Table", func() tabler { return new(Table[int]) }},
		{"Fast", func() tabler { return new(Fast[int]) }},
	}

	for _, tt := range tables {
		for _, tc := range testCases {
			t.Run(tt.name+"_"+tc.name, func(t *testing.T) {
				t.Parallel()

				tbl := tt.builder()

				// Insert prefixes with index as value
				for i, prefixStr := range tc.prefixes {
					pfx := netip.MustParsePrefix(prefixStr)
					tbl.Insert(pfx, i)
				}

				// Collect sorted results
				var actualOrder []string
				for pfx := range tbl.AllSorted() {
					actualOrder = append(actualOrder, pfx.String())
				}

				// Verify the order matches expected
				if len(actualOrder) != len(tc.expected) {
					t.Fatalf("%s_%s: Expected %d results, got %d", tt.name, tc.name, len(tc.expected), len(actualOrder))
				}

				for i, expected := range tc.expected {
					if actualOrder[i] != expected {
						t.Errorf("%s_%s:At position %d: expected %s, got %s", tt.name, tc.name, i, expected, actualOrder[i])
						t.Errorf("%s_%s:Full expected order: %v", tt.name, tc.name, tc.expected)
						t.Errorf("%s_%s:Full actual order:   %v", tt.name, tc.name, actualOrder)
						break
					}
				}
			})
		}
	}
}

func TestAllLite(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	prefixItems := randomPrefixes(prng, n)

	tbl := new(Lite)

	// Insert all prefixes with their values
	for _, item := range prefixItems {
		tbl.Insert(item.pfx)
	}

	// Collect all prefixes from AllSorted
	gotPrefixes := make([]netip.Prefix, 0, n)

	for pfx := range tbl.All() {
		gotPrefixes = append(gotPrefixes, pfx)
	}

	// Verify we got exactly the same number of prefixes
	if len(gotPrefixes) != len(prefixItems) {
		t.Fatalf("Expected %d prefixes, got %d", len(prefixItems), len(gotPrefixes))
	}

	// Verify we can find each original prefix in the sorted results
	// Create a map for O(1) lookup verification
	resultMap := make(map[netip.Prefix]bool, n)
	for _, pfx := range gotPrefixes {
		resultMap[pfx] = true
	}

	for _, originalItem := range prefixItems {
		_, found := resultMap[originalItem.pfx]
		if !found {
			t.Fatalf("Original prefix %v not found in All results", originalItem.pfx)
		}
	}

	// Verify no duplicates in results (since randomPrefixes guarantees no duplicates)
	seen := make(map[netip.Prefix]bool, n)
	for _, pfx := range gotPrefixes {
		if seen[pfx] {
			t.Fatalf("Duplicate prefix %v found in AllSorted results", pfx)
		}
		seen[pfx] = true
	}
}

func TestAllSortedLite(t *testing.T) {
	t.Parallel()

	// Test cases with known CIDR sort order
	testCases := []struct {
		name     string
		prefixes []string
		expected []string // Expected order after sorting
	}{
		{
			name: "Mixed IPv4 addresses and prefix lengths",
			prefixes: []string{
				"10.0.0.0/16",
				"10.0.0.0/8",
				"192.168.1.0/24",
				"10.0.0.0/24",
				"172.16.0.0/12",
			},
			expected: []string{
				"10.0.0.0/8",     // Same address, shorter prefix first
				"10.0.0.0/16",    // Same address, longer prefix
				"10.0.0.0/24",    // Same address, longest prefix
				"172.16.0.0/12",  // Next address
				"192.168.1.0/24", // Highest address
			},
		},
		{
			name: "Mixed IPv6 addresses and prefix lengths",
			prefixes: []string{
				"2001:db8::/32",
				"2001:db8::/64",
				"2000::/16",
				"2001:db8:1::/48",
			},
			expected: []string{
				"2000::/16",       // Lowest address
				"2001:db8::/32",   // Same address, shorter prefix first
				"2001:db8::/64",   // Same address, longer prefix
				"2001:db8:1::/48", // Higher address
			},
		},
		{
			name: "Mixed IPv4 and IPv6",
			prefixes: []string{
				"192.168.1.0/24",
				"2001:db8::/32",
				"10.0.0.0/8",
				"::1/128",
			},
			expected: []string{
				"10.0.0.0/8",     // IPv4 addresses come first (lower in comparison)
				"192.168.1.0/24", // Next IPv4 address
				"::1/128",        // IPv6 addresses after IPv4
				"2001:db8::/32",  // Higher IPv6 address
			},
		},
	}

	for _, tc := range testCases {
		tbl := new(Lite)

		// Insert prefixes with index as value
		for _, prefixStr := range tc.prefixes {
			pfx := netip.MustParsePrefix(prefixStr)
			tbl.Insert(pfx)
		}

		// Collect sorted results
		var actualOrder []string
		for pfx := range tbl.AllSorted() {
			actualOrder = append(actualOrder, pfx.String())
		}

		// Verify the order matches expected
		if len(actualOrder) != len(tc.expected) {
			t.Fatalf("%s: Expected %d results, got %d", tc.name, len(tc.expected), len(actualOrder))
		}

		for i, expected := range tc.expected {
			if actualOrder[i] != expected {
				t.Errorf("%s:At position %d: expected %s, got %s", tc.name, i, expected, actualOrder[i])
				t.Errorf("%s:Full expected order: %v", tc.name, tc.expected)
				t.Errorf("%s:Full actual order:   %v", tc.name, actualOrder)
				break
			}
		}
	}
}

// BenchmarkAllSorted ensures AllSorted performance is reasonable
func BenchmarkAllSorted(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	type tabler interface {
		Insert(netip.Prefix, int)
		AllSorted() iter.Seq2[netip.Prefix, int]
	}

	// Test all table types that support AllSorted
	tables := []struct {
		name    string
		builder func() tabler
	}{
		{"liteTable", func() tabler { return new(liteTable[int]) }},
		{"Table", func() tabler { return new(Table[int]) }},
		{"Fast", func() tabler { return new(Fast[int]) }},
	}

	for _, size := range sizes {
		for _, tt := range tables {
			b.Run(fmt.Sprintf("%s_size_%d", tt.name, size), func(b *testing.B) {
				// Setup test data
				prng := rand.New(rand.NewPCG(42, 42))
				prefixItems := randomPrefixes(prng, size)

				tbl := tt.builder()
				for _, item := range prefixItems {
					tbl.Insert(item.pfx, item.val)
				}

				for b.Loop() {
					for range tbl.AllSorted() {
					}
				}
			})
		}
	}
}
