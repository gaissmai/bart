package bart

import (
	"maps"
	"math/rand/v2"
	"net/netip"
	"testing"

	"github.com/gaissmai/bart/internal/nodes"
)

func FuzzTableSubnets(f *testing.F) {
	// Seed corpus
	f.Add(uint64(12345), 150, 30)
	f.Add(uint64(67890), 400, 60)
	f.Add(uint64(54321), 800, 100)
	// Edge-case leaning seeds
	f.Add(uint64(0), 64, 16)    // bias towards small sets
	f.Add(^uint64(0), 1024, 64) // large sets

	f.Fuzz(func(t *testing.T, seed uint64, n, nq int) {
		if n < 10 || n > 5000 || nq < 1 || nq > 200 {
			t.Skip("bounds")
		}

		prng := rand.New(rand.NewPCG(seed, 13))
		pfxs := randomPrefixes(prng, n)
		queries := randomPrefixes(prng, nq)

		bart := new(Table[int])
		for i, it := range pfxs {
			bart.Insert(it.pfx, i)
		}
		for _, q := range queries {
			want := map[netip.Prefix]bool{}
			for _, it := range pfxs {
				if isSubnetOf(it.pfx, q.pfx) {
					want[it.pfx] = true
				}
			}
			got := map[netip.Prefix]bool{}
			for p := range bart.Subnets(q.pfx) {
				if got[p] {
					t.Fatalf("Subnets duplicate: %v", p)
				}
				got[p] = true
			}
			if len(got) != len(want) {
				t.Fatalf("Subnets size mismatch for %v: want %d got %d", q.pfx, len(want), len(got))
			}
			for p := range want {
				if !got[p] {
					t.Fatalf("Subnets missing %v for %v", p, q.pfx)
				}
			}
		}
	})
}

func FuzzFastSubnets(f *testing.F) {
	// Seed corpus
	f.Add(uint64(12345), 150, 30)
	f.Add(uint64(67890), 400, 60)
	f.Add(uint64(54321), 800, 100)
	// Edge-case leaning seeds
	f.Add(uint64(0), 64, 16)    // bias towards small sets
	f.Add(^uint64(0), 1024, 64) // large sets

	f.Fuzz(func(t *testing.T, seed uint64, n, nq int) {
		if n < 10 || n > 5000 || nq < 1 || nq > 200 {
			t.Skip("bounds")
		}

		prng := rand.New(rand.NewPCG(seed, 13))
		pfxs := randomPrefixes(prng, n)
		queries := randomPrefixes(prng, nq)

		fast := new(Fast[int])
		for i, it := range pfxs {
			fast.Insert(it.pfx, i)
		}
		for _, q := range queries {
			want := map[netip.Prefix]bool{}
			for _, it := range pfxs {
				if isSubnetOf(it.pfx, q.pfx) {
					want[it.pfx] = true
				}
			}
			got := map[netip.Prefix]bool{}
			for p := range fast.Subnets(q.pfx) {
				if got[p] {
					t.Fatalf("Subnets duplicate: %v", p)
				}
				got[p] = true
			}
			if len(got) != len(want) {
				t.Fatalf("Subnets size mismatch for %v: want %d got %d", q.pfx, len(want), len(got))
			}
			for p := range want {
				if !got[p] {
					t.Fatalf("Subnets missing %v for %v", p, q.pfx)
				}
			}
		}
	})
}

func FuzzLiteSubnets(f *testing.F) {
	// Seed corpus
	f.Add(uint64(12345), 150, 30)
	f.Add(uint64(67890), 400, 60)
	f.Add(uint64(54321), 800, 100)
	// Edge-case leaning seeds
	f.Add(uint64(0), 64, 16)    // bias towards small sets
	f.Add(^uint64(0), 1024, 64) // large sets

	f.Fuzz(func(t *testing.T, seed uint64, n, nq int) {
		if n < 10 || n > 5000 || nq < 1 || nq > 200 {
			t.Skip("bounds")
		}

		prng := rand.New(rand.NewPCG(seed, 13))
		pfxs := randomPrefixes(prng, n)
		queries := randomPrefixes(prng, nq)

		lite := new(Lite)
		for _, it := range pfxs {
			lite.Insert(it.pfx)
		}
		for _, q := range queries {
			want := map[netip.Prefix]bool{}
			for _, it := range pfxs {
				if isSubnetOf(it.pfx, q.pfx) {
					want[it.pfx] = true
				}
			}
			got := map[netip.Prefix]bool{}
			for p := range lite.Subnets(q.pfx) {
				if got[p] {
					t.Fatalf("Subnets duplicate: %v", p)
				}
				got[p] = true
			}
			if len(got) != len(want) {
				t.Fatalf("Subnets size mismatch for %v: want %d got %d", q.pfx, len(want), len(got))
			}
			for p := range want {
				if !got[p] {
					t.Fatalf("Subnets missing %v for %v", p, q.pfx)
				}
			}
		}
	})
}

func FuzzTableSupernets(f *testing.F) {
	// Seed corpus
	f.Add(uint64(222), 150, 30)
	f.Add(uint64(333), 400, 60)
	f.Add(uint64(444), 800, 100)
	// Edge-case leaning seeds
	f.Add(uint64(0), 64, 16)    // bias towards small sets
	f.Add(^uint64(0), 1024, 64) // large sets

	f.Fuzz(func(t *testing.T, seed uint64, n, nq int) {
		if n < 10 || n > 5000 || nq < 1 || nq > 200 {
			t.Skip("bounds")
		}

		prng := rand.New(rand.NewPCG(seed, 17))
		pfxs := randomPrefixes(prng, n)
		queries := randomPrefixes(prng, nq)

		bart := new(Table[int])
		for i, it := range pfxs {
			bart.Insert(it.pfx, i)
		}
		for _, q := range queries {
			want := map[netip.Prefix]bool{}
			for _, it := range pfxs {
				if isSupernetOf(it.pfx, q.pfx) {
					want[it.pfx] = true
				}
			}
			got := map[netip.Prefix]bool{}
			for p := range bart.Supernets(q.pfx) {
				if got[p] {
					t.Fatalf("Supernets duplicate: %v", p)
				}
				got[p] = true
			}
			if len(got) != len(want) {
				t.Fatalf("Supernets size mismatch for %v: want %d got %d", q.pfx, len(want), len(got))
			}
			for p := range want {
				if !got[p] {
					t.Fatalf("Supernets missing %v for %v", p, q.pfx)
				}
			}
		}
	})
}

func FuzzFastSupernets(f *testing.F) {
	// Seed corpus
	f.Add(uint64(222), 150, 30)
	f.Add(uint64(333), 400, 60)
	f.Add(uint64(444), 800, 100)
	// Edge-case leaning seeds
	f.Add(uint64(0), 64, 16)    // bias towards small sets
	f.Add(^uint64(0), 1024, 64) // large sets

	f.Fuzz(func(t *testing.T, seed uint64, n, nq int) {
		if n < 10 || n > 5000 || nq < 1 || nq > 200 {
			t.Skip("bounds")
		}

		prng := rand.New(rand.NewPCG(seed, 17))
		pfxs := randomPrefixes(prng, n)
		queries := randomPrefixes(prng, nq)

		fast := new(Fast[int])
		for i, it := range pfxs {
			fast.Insert(it.pfx, i)
		}
		for _, q := range queries {
			want := map[netip.Prefix]bool{}
			for _, it := range pfxs {
				if isSupernetOf(it.pfx, q.pfx) {
					want[it.pfx] = true
				}
			}
			got := map[netip.Prefix]bool{}
			for p := range fast.Supernets(q.pfx) {
				if got[p] {
					t.Fatalf("Supernets duplicate: %v", p)
				}
				got[p] = true
			}
			if len(got) != len(want) {
				t.Fatalf("Supernets size mismatch for %v: want %d got %d", q.pfx, len(want), len(got))
			}
			for p := range want {
				if !got[p] {
					t.Fatalf("Supernets missing %v for %v", p, q.pfx)
				}
			}
		}
	})
}

func FuzzLiteSupernets(f *testing.F) {
	// Seed corpus
	f.Add(uint64(222), 150, 30)
	f.Add(uint64(333), 400, 60)
	f.Add(uint64(444), 800, 100)
	// Edge-case leaning seeds
	f.Add(uint64(0), 64, 16)    // bias towards small sets
	f.Add(^uint64(0), 1024, 64) // large sets

	f.Fuzz(func(t *testing.T, seed uint64, n, nq int) {
		if n < 10 || n > 5000 || nq < 1 || nq > 200 {
			t.Skip("bounds")
		}

		prng := rand.New(rand.NewPCG(seed, 17))
		pfxs := randomPrefixes(prng, n)
		queries := randomPrefixes(prng, nq)

		lite := new(Lite)
		for _, it := range pfxs {
			lite.Insert(it.pfx)
		}
		for _, q := range queries {
			want := map[netip.Prefix]bool{}
			for _, it := range pfxs {
				if isSupernetOf(it.pfx, q.pfx) {
					want[it.pfx] = true
				}
			}
			got := map[netip.Prefix]bool{}
			for p := range lite.Supernets(q.pfx) {
				if got[p] {
					t.Fatalf("Supernets duplicate: %v", p)
				}
				got[p] = true
			}
			if len(got) != len(want) {
				t.Fatalf("Supernets size mismatch for %v: want %d got %d", q.pfx, len(want), len(got))
			}
			for p := range want {
				if !got[p] {
					t.Fatalf("Supernets missing %v for %v", p, q.pfx)
				}
			}
		}
	})
}

func FuzzLiteModifyComprehensive(f *testing.F) {
	seeds := []struct {
		seed  uint64
		count int
		op    uint8 // 0=insert, 2=delete, 3=no-op (skip update since no payload)
	}{
		{12345, 50, 0},
		{67890, 25, 2},
		{11111, 75, 3},
		{22222, 30, 0},
		{33333, 10, 2},
	}

	for _, seed := range seeds {
		f.Add(seed.seed, seed.count, seed.op)
	}

	f.Fuzz(func(t *testing.T, seed uint64, count int, op uint8) {
		if count < 5 || count > 100 {
			return
		}

		prng := rand.New(rand.NewPCG(seed, 42))
		prefixItems := randomPrefixes(prng, count)

		if len(prefixItems) == 0 {
			return
		}

		targetIdx := prng.IntN(len(prefixItems))
		targetPrefix := prefixItems[targetIdx].pfx

		lite := &Lite{}

		// Setup: Insert first half of prefixes
		halfCount := len(prefixItems) / 2
		for i := range halfCount {
			item := prefixItems[i]
			lite.Modify(item.pfx, func(bool) bool {
				return false // insert
			})
		}

		initialSize := lite.Size()
		initialFound := lite.Get(targetPrefix)

		// Expected outcome tracking
		var expectedSize int
		var expectedFound bool

		// Execute modify operation - skip update ops since Lite has no meaningful payload
		lite.Modify(targetPrefix, func(found bool) bool {
			// Verify callback parameters
			if found != initialFound {
				t.Errorf("callback found=%v, but actual found=%v", found, initialFound)
			}

			// Map op to valid operations for liteTable (no update)
			switch op % 3 { // Use mod 3 to skip update operation
			case 0: // insert if not found
				if !found {
					expectedSize = initialSize + 1
					expectedFound = true
					return false // insert
				}
				// Already exists, no change
				expectedSize = initialSize
				expectedFound = true
				return false // keep existing

			case 1: // delete if found (mod 3 case 1)
				if found {
					expectedSize = initialSize - 1
					expectedFound = false
					return true // delete
				}
				// Not found, no-op
				expectedSize = initialSize
				expectedFound = false
				return true // no-op with del=true

			case 2: // no-op always (mod 3 case 2)
				expectedSize = initialSize
				expectedFound = found

				if found {
					return false // keep existing
				} else {
					return true // no-op with del=true
				}
			}

			panic("unreachable")
		})

		// Verify results
		if lite.Size() != expectedSize {
			t.Errorf("Size inconsistent: got %d, expected %d (op=%d, initialFound=%v)",
				lite.Size(), expectedSize, op%3, initialFound)
		}

		actualFound := lite.Get(targetPrefix)
		if actualFound != expectedFound {
			t.Errorf("Get found inconsistent: got %v, expected %v (op=%d, initialFound=%v)",
				actualFound, expectedFound, op%3, initialFound)
		}
	})
}

func FuzzTableModifyComprehensive(f *testing.F) {
	seeds := []struct {
		seed  uint64
		count int
		value int
		op    uint8
	}{
		{12345, 50, 100, 0},
		{67890, 25, 200, 1},
		{11111, 75, 300, 2},
		{22222, 30, 400, 3},
		{33333, 10, 500, 0},
	}

	for _, seed := range seeds {
		f.Add(seed.seed, seed.count, seed.value, seed.op)
	}

	f.Fuzz(func(t *testing.T, seed uint64, count int, value int, op uint8) {
		if count < 5 || count > 100 {
			return
		}

		prng := rand.New(rand.NewPCG(seed, 42))
		prefixItems := randomPrefixes(prng, count)

		if len(prefixItems) == 0 {
			return
		}

		targetIdx := prng.IntN(len(prefixItems))
		targetPrefix := prefixItems[targetIdx].pfx

		bart := new(Table[int])

		// Setup: Insert first half of prefixes
		halfCount := len(prefixItems) / 2
		for i := range halfCount {
			item := prefixItems[i]
			bart.Modify(item.pfx, func(_ int, _ bool) (int, bool) {
				return item.val, false
			})
		}

		initialSize := bart.Size()
		initialVal, initialFound := bart.Get(targetPrefix)

		var expectedSize int
		var expectedFound bool
		var expectedVal int

		bart.Modify(targetPrefix, func(val int, found bool) (int, bool) {
			// Verify callback parameters match actual state
			if found != initialFound {
				t.Errorf("callback found=%v, but actual found=%v", found, initialFound)
			}
			if found && val != initialVal {
				t.Errorf("callback val=%v, but actual val=%v", val, initialVal)
			}

			switch op % 4 {
			case 0: // insert if not found
				if !found {
					expectedSize = initialSize + 1
					expectedFound = true
					expectedVal = value
					return value, false // insert new value
				}
				// Already exists, keep existing
				expectedSize = initialSize
				expectedFound = true
				expectedVal = val
				return val, false // no change

			case 1: // update if found
				if found {
					expectedSize = initialSize
					expectedFound = true
					expectedVal = value
					return value, false // update to new value
				}
				// Not found, no-op
				expectedSize = initialSize
				expectedFound = false
				expectedVal = 0
				return 0, true // del=true means no-op

			case 2: // delete if found
				if found {
					expectedSize = initialSize - 1
					expectedFound = false
					expectedVal = 0
					return val, true // delete existing
				}
				// Not found, no-op
				expectedSize = initialSize
				expectedFound = false
				expectedVal = 0
				return 0, true // del=true means no-op

			case 3: // no-op always
				expectedSize = initialSize
				expectedFound = found

				if found {
					expectedVal = val
					return val, false // keep existing value unchanged
				} else {
					expectedVal = 0
					return 0, true // del=true means no-op for non-existent
				}
			}

			return 0, false
		})

		// Verify all results
		if bart.Size() != expectedSize {
			t.Errorf("Size inconsistent: got %d, expected %d (op=%d, initialFound=%v)",
				bart.Size(), expectedSize, op%4, initialFound)
		}

		actualVal, actualFound := bart.Get(targetPrefix)
		if actualFound != expectedFound {
			t.Errorf("Get found inconsistent: got %v, expected %v (op=%d, initialFound=%v)",
				actualFound, expectedFound, op%4, initialFound)
		}

		if expectedFound && actualVal != expectedVal {
			t.Errorf("Get value inconsistent: got %v, expected %v (op=%d)",
				actualVal, expectedVal, op%4)
		}
	})
}

func FuzzFastModifyComprehensive(f *testing.F) {
	seeds := []struct {
		seed  uint64
		count int
		value int
		op    uint8
	}{
		{12345, 50, 100, 0},
		{67890, 25, 200, 1},
		{11111, 75, 300, 2},
		{22222, 30, 400, 3},
		{33333, 10, 500, 0},
	}

	for _, seed := range seeds {
		f.Add(seed.seed, seed.count, seed.value, seed.op)
	}

	f.Fuzz(func(t *testing.T, seed uint64, count int, value int, op uint8) {
		if count < 5 || count > 100 {
			return
		}

		prng := rand.New(rand.NewPCG(seed, 42))
		prefixItems := randomPrefixes(prng, count)

		if len(prefixItems) == 0 {
			return
		}

		targetIdx := prng.IntN(len(prefixItems))
		targetPrefix := prefixItems[targetIdx].pfx

		fast := new(Fast[int])

		// Setup: Insert first half of prefixes
		halfCount := len(prefixItems) / 2
		for i := range halfCount {
			item := prefixItems[i]
			fast.Modify(item.pfx, func(_ int, _ bool) (int, bool) {
				return item.val, false
			})
		}

		initialSize := fast.Size()
		initialVal, initialFound := fast.Get(targetPrefix)

		var expectedSize int
		var expectedFound bool
		var expectedVal int

		fast.Modify(targetPrefix, func(val int, found bool) (int, bool) {
			// Verify callback parameters match actual state
			if found != initialFound {
				t.Errorf("callback found=%v, but actual found=%v", found, initialFound)
			}
			if found && val != initialVal {
				t.Errorf("callback val=%v, but actual val=%v", val, initialVal)
			}

			switch op % 4 {
			case 0: // insert if not found
				if !found {
					expectedSize = initialSize + 1
					expectedFound = true
					expectedVal = value
					return value, false // insert new value
				}
				// Already exists, keep existing
				expectedSize = initialSize
				expectedFound = true
				expectedVal = val
				return val, false // no change

			case 1: // update if found
				if found {
					expectedSize = initialSize
					expectedFound = true
					expectedVal = value
					return value, false // update to new value
				}
				// Not found, no-op
				expectedSize = initialSize
				expectedFound = false
				expectedVal = 0
				return 0, true // del=true means no-op

			case 2: // delete if found
				if found {
					expectedSize = initialSize - 1
					expectedFound = false
					expectedVal = 0
					return val, true // delete existing
				}
				// Not found, no-op
				expectedSize = initialSize
				expectedFound = false
				expectedVal = 0
				return 0, true // del=true means no-op

			case 3: // no-op always
				expectedSize = initialSize
				expectedFound = found

				if found {
					expectedVal = val
					return val, false // keep existing value unchanged
				} else {
					expectedVal = 0
					return 0, true // del=true means no-op for non-existent
				}
			}

			return 0, false
		})

		// Verify all results
		if fast.Size() != expectedSize {
			t.Errorf("Size inconsistent: got %d, expected %d (op=%d, initialFound=%v)",
				fast.Size(), expectedSize, op%4, initialFound)
		}

		actualVal, actualFound := fast.Get(targetPrefix)
		if actualFound != expectedFound {
			t.Errorf("Get found inconsistent: got %v, expected %v (op=%d, initialFound=%v)",
				actualFound, expectedFound, op%4, initialFound)
		}

		if expectedFound && actualVal != expectedVal {
			t.Errorf("Get value inconsistent: got %v, expected %v (op=%d)",
				actualVal, expectedVal, op%4)
		}
	})
}

func FuzzTableAllSorted(f *testing.F) {
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

		bart := new(Table[int])

		// Insert all prefixes with their values
		for _, item := range prefixItems {
			bart.Insert(item.pfx, item.val)
		}

		// Collect all prefixes from AllSorted
		sortedPrefixes := make([]netip.Prefix, 0, count)
		sortedValues := make([]int, 0, count)

		for pfx, val := range bart.AllSorted() {
			sortedPrefixes = append(sortedPrefixes, pfx)
			sortedValues = append(sortedValues, val)
		}

		// Verify we got exactly the same number of prefixes
		if len(sortedPrefixes) != len(prefixItems) {
			t.Fatalf("Expected %d prefixes, got %d", len(prefixItems), len(sortedPrefixes))
		}

		// Verify all prefixes are in natural CIDR sort order using existing cmpPrefix function
		for i := 1; i < len(sortedPrefixes); i++ {
			if nodes.CmpPrefix(sortedPrefixes[i-1], sortedPrefixes[i]) > 0 {
				t.Fatalf("CIDR sort order violated at index %d: %v should come before %v",
					i-1, sortedPrefixes[i-1], sortedPrefixes[i])
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
				t.Fatalf("Original prefix %v not found in AllSorted results", originalItem.pfx)
			}

			if val != originalItem.val {
				t.Fatalf("Original prefix %v has wrong value: expected %d, got %d",
					originalItem.pfx, originalItem.val, val)
			}
		}

		// Verify no duplicates in results (since randomPrefixes guarantees no duplicates)
		seen := make(map[netip.Prefix]bool, count)
		for _, pfx := range sortedPrefixes {
			if seen[pfx] {
				t.Fatalf("Duplicate prefix %v found in AllSorted results", pfx)
			}
			seen[pfx] = true
		}
	})
}

func FuzzFastAllSorted(f *testing.F) {
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

		fast := new(Fast[int])

		// Insert all prefixes with their values
		for _, item := range prefixItems {
			fast.Insert(item.pfx, item.val)
		}

		// Collect all prefixes from AllSorted
		sortedPrefixes := make([]netip.Prefix, 0, count)
		sortedValues := make([]int, 0, count)

		for pfx, val := range fast.AllSorted() {
			sortedPrefixes = append(sortedPrefixes, pfx)
			sortedValues = append(sortedValues, val)
		}

		// Verify we got exactly the same number of prefixes
		if len(sortedPrefixes) != len(prefixItems) {
			t.Fatalf("Expected %d prefixes, got %d", len(prefixItems), len(sortedPrefixes))
		}

		// Verify all prefixes are in natural CIDR sort order using existing cmpPrefix function
		for i := 1; i < len(sortedPrefixes); i++ {
			if nodes.CmpPrefix(sortedPrefixes[i-1], sortedPrefixes[i]) > 0 {
				t.Fatalf("CIDR sort order violated at index %d: %v should come before %v",
					i-1, sortedPrefixes[i-1], sortedPrefixes[i])
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
				t.Fatalf("Original prefix %v not found in AllSorted results", originalItem.pfx)
			}

			if val != originalItem.val {
				t.Fatalf("Original prefix %v has wrong value: expected %d, got %d",
					originalItem.pfx, originalItem.val, val)
			}
		}

		// Verify no duplicates in results (since randomPrefixes guarantees no duplicates)
		seen := make(map[netip.Prefix]bool, count)
		for _, pfx := range sortedPrefixes {
			if seen[pfx] {
				t.Fatalf("Duplicate prefix %v found in AllSorted results", pfx)
			}
			seen[pfx] = true
		}
	})
}

func FuzzLiteAllSorted(f *testing.F) {
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

		lite := new(Lite)

		// Insert all prefixes with their values
		for _, item := range prefixItems {
			lite.Insert(item.pfx)
		}

		// Collect all prefixes from AllSorted
		sortedPrefixes := make([]netip.Prefix, 0, count)

		for pfx := range lite.AllSorted() {
			sortedPrefixes = append(sortedPrefixes, pfx)
		}

		// Verify we got exactly the same number of prefixes
		if len(sortedPrefixes) != len(prefixItems) {
			t.Fatalf("Expected %d prefixes, got %d", len(prefixItems), len(sortedPrefixes))
		}

		// Verify all prefixes are in natural CIDR sort order using existing cmpPrefix function
		for i := 1; i < len(sortedPrefixes); i++ {
			if nodes.CmpPrefix(sortedPrefixes[i-1], sortedPrefixes[i]) > 0 {
				t.Fatalf("CIDR sort order violated at index %d: %v should come before %v",
					i-1, sortedPrefixes[i-1], sortedPrefixes[i])
			}
		}

		// Verify we can find each original prefix in the sorted results
		// Create a map for O(1) lookup verification
		resultMap := make(map[netip.Prefix]bool, count)
		for _, pfx := range sortedPrefixes {
			resultMap[pfx] = true
		}

		for _, originalItem := range prefixItems {
			_, found := resultMap[originalItem.pfx]
			if !found {
				t.Fatalf("Original prefix %v not found in AllSorted results", originalItem.pfx)
			}
		}

		// Verify no duplicates in results (since randomPrefixes guarantees no duplicates)
		seen := make(map[netip.Prefix]bool, count)
		for _, pfx := range sortedPrefixes {
			if seen[pfx] {
				t.Fatalf("Duplicate prefix %v found in AllSorted results", pfx)
			}
			seen[pfx] = true
		}
	})
}

func FuzzTableOverlaps(f *testing.F) {
	// Seed corpus
	f.Add(uint64(12345), 50, 50)
	f.Add(uint64(67890), 150, 75)
	f.Add(uint64(22222), 300, 300)

	f.Fuzz(func(t *testing.T, seed uint64, n1, n2 int) {
		if n1 < 1 || n1 > 1000 || n2 < 1 || n2 > 1000 {
			t.Skip("counts out of range")
		}

		prng := rand.New(rand.NewPCG(seed, 42))
		aPfxs := randomRealWorldPrefixes(prng, n1) // invariant: no duplicates
		bPfxs := randomRealWorldPrefixes(prng, n2)

		gt := expectedOverlaps(aPfxs, bPfxs)

		t1 := new(Table[int])
		t2 := new(Table[int])
		for i, pfx := range aPfxs {
			t1.Insert(pfx, i)
		}
		for i, pfx := range bPfxs {
			t2.Insert(pfx, i)
		}
		if got := t1.Overlaps(t2); got != gt {
			t.Fatalf("Overlaps mismatch: want %v, got %v", gt, got)
		}
	})
}

func FuzzFastOverlaps(f *testing.F) {
	// Seed corpus
	f.Add(uint64(12345), 50, 50)
	f.Add(uint64(67890), 150, 75)
	f.Add(uint64(22222), 300, 300)

	f.Fuzz(func(t *testing.T, seed uint64, n1, n2 int) {
		if n1 < 1 || n1 > 1000 || n2 < 1 || n2 > 1000 {
			t.Skip("counts out of range")
		}

		prng := rand.New(rand.NewPCG(seed, 42))
		aPfxs := randomRealWorldPrefixes(prng, n1) // invariant: no duplicates
		bPfxs := randomRealWorldPrefixes(prng, n2)

		gt := expectedOverlaps(aPfxs, bPfxs)

		t1 := new(Fast[int])
		t2 := new(Fast[int])
		for i, pfx := range aPfxs {
			t1.Insert(pfx, i)
		}
		for i, pfx := range bPfxs {
			t2.Insert(pfx, i)
		}
		if got := t1.Overlaps(t2); got != gt {
			t.Fatalf("Overlaps mismatch: want %v, got %v", gt, got)
		}
	})
}

func FuzzLiteOverlaps(f *testing.F) {
	// Seed corpus
	f.Add(uint64(12345), 50, 50)
	f.Add(uint64(67890), 150, 75)
	f.Add(uint64(22222), 300, 300)

	f.Fuzz(func(t *testing.T, seed uint64, n1, n2 int) {
		if n1 < 1 || n1 > 1000 || n2 < 1 || n2 > 1000 {
			t.Skip("counts out of range")
		}

		prng := rand.New(rand.NewPCG(seed, 42))
		aPfxs := randomRealWorldPrefixes(prng, n1) // invariant: no duplicates
		bPfxs := randomRealWorldPrefixes(prng, n2)

		gt := expectedOverlaps(aPfxs, bPfxs)

		t1 := new(Lite)
		t2 := new(Lite)
		for _, pfx := range aPfxs {
			t1.Insert(pfx)
		}
		for _, pfx := range bPfxs {
			t2.Insert(pfx)
		}
		if got := t1.Overlaps(t2); got != gt {
			t.Fatalf("Overlaps mismatch: want %v, got %v", gt, got)
		}
	})
}

func FuzzTableOverlapsPrefix(f *testing.F) {
	// Seed corpus
	f.Add(uint64(111), 200, 60)
	f.Add(uint64(222), 600, 120)
	f.Add(uint64(333), 1200, 180)

	f.Fuzz(func(t *testing.T, seed uint64, n, nq int) {
		if n < 1 || n > 4000 || nq < 1 || nq > 400 {
			t.Skip("counts out of range")
		}

		prng := rand.New(rand.NewPCG(seed, 7))
		items := randomRealWorldPrefixes(prng, n)  // table contents
		query := randomRealWorldPrefixes(prng, nq) // queries

		bart := new(Table[int])
		for i, pfx := range items {
			bart.Insert(pfx, i)
		}
		for _, pfx := range query {
			gt := expectedOverlapsPrefix(items, pfx)
			got := bart.OverlapsPrefix(pfx)
			if got != gt {
				t.Fatalf("OverlapsPrefix(%v) mismatch: want %v, got %v", pfx, gt, got)
			}
		}
	})
}

func FuzzFastOverlapsPrefix(f *testing.F) {
	// Seed corpus
	f.Add(uint64(111), 200, 60)
	f.Add(uint64(222), 600, 120)
	f.Add(uint64(333), 1200, 180)

	f.Fuzz(func(t *testing.T, seed uint64, n, nq int) {
		if n < 1 || n > 4000 || nq < 1 || nq > 400 {
			t.Skip("counts out of range")
		}

		prng := rand.New(rand.NewPCG(seed, 7))
		items := randomRealWorldPrefixes(prng, n)  // table contents
		query := randomRealWorldPrefixes(prng, nq) // queries

		fast := new(Fast[int])
		for i, pfx := range items {
			fast.Insert(pfx, i)
		}
		for _, pfx := range query {
			gt := expectedOverlapsPrefix(items, pfx)
			got := fast.OverlapsPrefix(pfx)
			if got != gt {
				t.Fatalf("OverlapsPrefix(%v) mismatch: want %v, got %v", pfx, gt, got)
			}
		}
	})
}

func FuzzLiteOverlapsPrefix(f *testing.F) {
	// Seed corpus
	f.Add(uint64(111), 200, 60)
	f.Add(uint64(222), 600, 120)
	f.Add(uint64(333), 1200, 180)

	f.Fuzz(func(t *testing.T, seed uint64, n, nq int) {
		if n < 1 || n > 4000 || nq < 1 || nq > 400 {
			t.Skip("counts out of range")
		}

		prng := rand.New(rand.NewPCG(seed, 7))
		items := randomRealWorldPrefixes(prng, n)  // table contents
		query := randomRealWorldPrefixes(prng, nq) // queries

		lite := new(Lite)
		for _, pfx := range items {
			lite.Insert(pfx)
		}
		for _, pfx := range query {
			gt := expectedOverlapsPrefix(items, pfx)
			got := lite.OverlapsPrefix(pfx)
			if got != gt {
				t.Fatalf("OverlapsPrefix(%v) mismatch: want %v, got %v", pfx, gt, got)
			}
		}
	})
}

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

func FuzzTableUnionPersist(f *testing.F) {
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

func FuzzTableUnionPersistAliasing(f *testing.F) {
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
