//go:build go1.23

// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"reflect"
	"slices"
	"testing"
)

func TestAll4RangeOverFunc(t *testing.T) {
	t.Parallel()
	pfxs := randomPrefixes4(10_000)
	seen := make(map[netip.Prefix]int, 10_000)

	t.Run("All4RangeOverFunc", func(t *testing.T) {
		t.Parallel()
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
			seen[item.pfx] = item.val
		}

		// rangefunc iterator
		for pfx, val := range rtbl.All4() {
			// check if pfx/val is as expected
			if seen[pfx] != val {
				t.Errorf("%v got value: %v, expected: %v", pfx, val, seen[pfx])
			}
			delete(seen, pfx)
		}

		// check if all entries visited
		if len(seen) != 0 {
			t.Fatalf("traverse error, not all entries visited")
		}
	})

	t.Run("All4RangeOverFunc with premature exit", func(t *testing.T) {
		t.Parallel()
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
		}

		// check if callback stops prematurely
		count := 0
		for range rtbl.All4() {
			count++
			if count >= 1000 {
				break
			}
		}

		// check if iteration stopped with error
		if count > 1000 {
			t.Fatalf("expected premature stop with error")
		}
	})
}

func TestAll6RangeOverFunc(t *testing.T) {
	t.Parallel()
	pfxs := randomPrefixes6(10_000)
	seen := make(map[netip.Prefix]int, 10_000)

	t.Run("All6RangeOverFunc", func(t *testing.T) {
		t.Parallel()
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
			seen[item.pfx] = item.val
		}

		// rangefunc iterator
		for pfx, val := range rtbl.All6() {
			// check if pfx/val is as expected
			if seen[pfx] != val {
				t.Errorf("%v got value: %v, expected: %v", pfx, val, seen[pfx])
			}
			delete(seen, pfx)
		}

		// check if all entries visited
		if len(seen) != 0 {
			t.Fatalf("traverse error, not all entries visited")
		}
	})

	t.Run("All6RangeOverFunc with premature exit", func(t *testing.T) {
		t.Parallel()
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
		}

		// check if callback stops prematurely
		count := 0
		for range rtbl.All6() {
			count++
			if count >= 1000 {
				break
			}
		}

		// check if iteration stopped with error
		if count > 1000 {
			t.Fatalf("expected premature stop with error")
		}
	})
}

func TestAllRangeOverFunc(t *testing.T) {
	t.Parallel()
	pfxs := randomPrefixes(10_000)
	seen := make(map[netip.Prefix]int, 10_000)

	t.Run("AllRangeOverFunc", func(t *testing.T) {
		t.Parallel()
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
			seen[item.pfx] = item.val
		}

		// rangefunc iterator
		for pfx, val := range rtbl.All() {
			// check if pfx/val is as expected
			if seen[pfx] != val {
				t.Errorf("%v got value: %v, expected: %v", pfx, val, seen[pfx])
			}
			delete(seen, pfx)
		}

		// check if all entries visited
		if len(seen) != 0 {
			t.Fatalf("traverse error, not all entries visited")
		}
	})

	t.Run("AllRangeOverFunc with premature exit", func(t *testing.T) {
		t.Parallel()
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
		}

		// check if callback stops prematurely
		count := 0
		for range rtbl.All() {
			count++
			if count >= 1000 {
				break
			}
		}

		// check if iteration stopped with error
		if count > 1000 {
			t.Fatalf("expected premature stop with error")
		}
	})
}

func TestAll4SortedIter(t *testing.T) {
	t.Parallel()
	pfxs := randomPrefixes4(10_000)
	seen := make(map[netip.Prefix]int, 10_000)

	t.Run("All4SortedRangeOverFunc", func(t *testing.T) {
		t.Parallel()
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
			seen[item.pfx] = item.val
		}

		// rangefunc iterator
		for pfx, val := range rtbl.AllSorted4() {
			// check if pfx/val is as expected
			if seen[pfx] != val {
				t.Errorf("%v got value: %v, expected: %v", pfx, val, seen[pfx])
			}
			delete(seen, pfx)
		}

		// check if all entries visited
		if len(seen) != 0 {
			t.Fatalf("traverse error, not all entries visited")
		}
	})

	t.Run("All4SortedRangeOverFunc with premature exit", func(t *testing.T) {
		t.Parallel()
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
		}

		// check if callback stops prematurely
		count := 0
		for range rtbl.AllSorted4() {
			count++
			if count >= 1000 {
				break
			}
		}

		// check if iteration stopped with error
		if count > 1000 {
			t.Fatalf("expected premature stop with error")
		}
	})
}

func TestAll6SortedRangeOverFunc(t *testing.T) {
	t.Parallel()
	pfxs := randomPrefixes6(10_000)
	seen := make(map[netip.Prefix]int, 10_000)

	t.Run("All6SortedRangeOverFunc", func(t *testing.T) {
		t.Parallel()
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
			seen[item.pfx] = item.val
		}

		// rangefunc iterator
		for pfx, val := range rtbl.AllSorted6() {
			// check if pfx/val is as expected
			if seen[pfx] != val {
				t.Errorf("%v got value: %v, expected: %v", pfx, val, seen[pfx])
			}
			delete(seen, pfx)
		}

		// check if all entries visited
		if len(seen) != 0 {
			t.Fatalf("traverse error, not all entries visited")
		}
	})

	t.Run("All6SortedRangeOverFunc with premature exit", func(t *testing.T) {
		t.Parallel()
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
		}

		// check if callback stops prematurely
		count := 0
		for range rtbl.AllSorted6() {
			count++
			if count >= 1000 {
				break
			}
		}

		// check if iteration stopped with error
		if count > 1000 {
			t.Fatalf("expected premature stop with error")
		}
	})
}

func TestAllSortedRangeOverFunc(t *testing.T) {
	t.Parallel()
	pfxs := randomPrefixes(10_000)
	seen := make(map[netip.Prefix]int, 10_000)

	t.Run("AllSortedRangeOverFunc", func(t *testing.T) {
		t.Parallel()
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
			seen[item.pfx] = item.val
		}

		// rangefunc iterator
		for pfx, val := range rtbl.AllSorted() {
			// check if pfx/val is as expected
			if seen[pfx] != val {
				t.Errorf("%v got value: %v, expected: %v", pfx, val, seen[pfx])
			}
			delete(seen, pfx)
		}

		// check if all entries visited
		if len(seen) != 0 {
			t.Fatalf("traverse error, not all entries visited")
		}
	})

	t.Run("AllSortedRangeOverFunc with premature exit", func(t *testing.T) {
		t.Parallel()
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
		}

		// check if callback stops prematurely
		count := 0
		for range rtbl.AllSorted() {
			count++
			if count >= 1000 {
				break
			}
		}

		// check if iteration stopped with error
		if count > 1000 {
			t.Fatalf("expected premature stop with error")
		}
	})
}

func TestSupernets(t *testing.T) {
	t.Parallel()

	pfxs := randomPrefixes(10_000)

	fast := Table[int]{}
	gold := goldTable[int](pfxs)

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	var fastPfxs []netip.Prefix
	for range 10_000 {
		pfx := randomPrefix()

		goldPfxs := gold.lookupPrefixReverse(pfx)

		fastPfxs = nil
		for p := range fast.Supernets(pfx) {
			fastPfxs = append(fastPfxs, p)
		}

		if !reflect.DeepEqual(goldPfxs, fastPfxs) {
			t.Fatalf("\nEachSupernet(%q):\ngot:  %v\nwant: %v", pfx, fastPfxs, goldPfxs)
		}
	}
}

func TestSupernetsEdgeCase(t *testing.T) {
	t.Parallel()

	rtbl := new(Table[any])
	pfx := mpp("::1/128")
	for range rtbl.Supernets(pfx) {
		t.Errorf("empty table, must not range over")
	}

	val := "foo"
	rtbl.Insert(pfx, val)
	for range rtbl.Supernets(netip.Prefix{}) {
		t.Errorf("invalid prefix, must not range over")
	}

	for p, v := range rtbl.Supernets(pfx) {
		if p != pfx {
			t.Errorf("Supernets(%v), got: %v, want: %v", pfx, p, pfx)
		}

		if v.(string) != val {
			t.Errorf("Supernets(%v), got: %v, want: %v", pfx, v.(string), val)
		}
	}
}

func TestSubnets(t *testing.T) {
	t.Parallel()

	var zeroPfx netip.Prefix

	t.Run("empty table", func(t *testing.T) {
		rtbl := new(Table[string])
		pfx := mpp("::1/128")

		for range rtbl.Subnets(pfx) {
			t.Errorf("empty table, must not range over")
		}
	})

	t.Run("invalid prefix", func(t *testing.T) {
		rtbl := new(Table[string])
		pfx := mpp("::1/128")
		val := "foo"
		rtbl.Insert(pfx, val)
		for range rtbl.Subnets(zeroPfx) {
			t.Errorf("invalid prefix, must not range over")
		}
	})

	t.Run("identity", func(t *testing.T) {
		rtbl := new(Table[string])
		pfx := mpp("::1/128")
		val := "foo"
		rtbl.Insert(pfx, val)

		for p, v := range rtbl.Subnets(pfx) {
			if p != pfx {
				t.Errorf("Subnet(%v), got: %v, want: %v", pfx, p, pfx)
			}

			if v != val {
				t.Errorf("Subnet(%v), got: %v, want: %v", pfx, v, val)
			}
		}
	})

	t.Run("default gateway", func(t *testing.T) {
		want4 := 95_555
		want6 := 105_555

		rtbl := new(Table[int])
		for i, pfx := range gimmeRandomPrefixes4(want4) {
			rtbl.Insert(pfx, i)
		}
		for i, pfx := range gimmeRandomPrefixes6(want6) {
			rtbl.Insert(pfx, i)
		}

		// default gateway v4 covers all v4 prefixes in table
		dg4 := mpp("0.0.0.0/0")
		got4 := 0
		for range rtbl.Subnets(dg4) {
			got4++
		}

		// default gateway v6 covers all v6 prefixes in table
		dg6 := mpp("::/0")
		got6 := 0
		for range rtbl.Subnets(dg6) {
			got6++
		}

		if got4 != want4 {
			t.Errorf("Subnets v4, want: %d, got: %d", want4, got4)
		}
		if got6 != want6 {
			t.Errorf("Subnets v6, want: %d, got: %d", want6, got6)
		}
	})
}

func TestSubnetsPC(t *testing.T) {
	t.Parallel()

	var zeroPfx netip.Prefix

	t.Run("empty table", func(t *testing.T) {
		rtbl := new(Table[string]).WithPathCompression()
		pfx := mpp("::1/128")

		for range rtbl.Subnets(pfx) {
			t.Errorf("empty table, must not range over")
		}
	})

	t.Run("invalid prefix", func(t *testing.T) {
		rtbl := new(Table[string]).WithPathCompression()
		pfx := mpp("::1/128")
		val := "foo"
		rtbl.Insert(pfx, val)
		for range rtbl.Subnets(zeroPfx) {
			t.Errorf("invalid prefix, must not range over")
		}
	})

	t.Run("identity", func(t *testing.T) {
		rtbl := new(Table[string]).WithPathCompression()
		pfx := mpp("::1/128")
		val := "foo"
		rtbl.Insert(pfx, val)

		for p, v := range rtbl.Subnets(pfx) {
			if p != pfx {
				t.Errorf("Subnet(%v), got: %v, want: %v", pfx, p, pfx)
			}

			if v != val {
				t.Errorf("Subnet(%v), got: %v, want: %v", pfx, v, val)
			}
		}
	})

	t.Run("default gateway", func(t *testing.T) {
		want4 := 95_555
		want6 := 105_555

		rtbl := new(Table[int]).WithPathCompression()
		for i, pfx := range gimmeRandomPrefixes4(want4) {
			rtbl.Insert(pfx, i)
		}
		for i, pfx := range gimmeRandomPrefixes6(want6) {
			rtbl.Insert(pfx, i)
		}

		// default gateway v4 covers all v4 prefixes in table
		dg4 := mpp("0.0.0.0/0")
		got4 := 0
		for range rtbl.Subnets(dg4) {
			got4++
		}

		// default gateway v6 covers all v6 prefixes in table
		dg6 := mpp("::/0")
		got6 := 0
		for range rtbl.Subnets(dg6) {
			got6++
		}

		if got4 != want4 {
			t.Errorf("Subnets v4, want: %d, got: %d", want4, got4)
		}
		if got6 != want6 {
			t.Errorf("Subnets v6, want: %d, got: %d", want6, got6)
		}
	})
}

func TestSubnetsCompare(t *testing.T) {
	t.Parallel()

	pfxs := gimmeRandomPrefixes(10_000)

	fast := new(Table[int])
	gold := goldTable[int]{}

	for i, pfx := range pfxs {
		fast.Insert(pfx, i)
		gold.insert(pfx, i)
	}

	tests := randomPrefixes(200)
	for _, tt := range tests {
		gotGold := gold.subnets(tt.pfx)
		gotFast := []netip.Prefix{}
		for pfx := range fast.Subnets(tt.pfx) {
			gotFast = append(gotFast, pfx)
		}
		if !slices.Equal(gotGold, gotFast) {
			t.Fatalf("Subnets(%q) = %v, want %v", tt.pfx, gotFast, gotGold)
		}
	}
}

func TestSubnetsComparePC(t *testing.T) {
	t.Parallel()

	pfxs := gimmeRandomPrefixes(10_000)

	fast := new(Table[int]).WithPathCompression()
	gold := goldTable[int]{}

	for i, pfx := range pfxs {
		fast.Insert(pfx, i)
		gold.insert(pfx, i)
	}

	tests := randomPrefixes(200)
	for _, tt := range tests {
		gotGold := gold.subnets(tt.pfx)
		gotFast := []netip.Prefix{}
		for pfx := range fast.Subnets(tt.pfx) {
			gotFast = append(gotFast, pfx)
		}
		if !slices.Equal(gotGold, gotFast) {
			t.Fatalf("Subnets(%q) = %v, want %v", tt.pfx, gotFast, gotGold)
		}
	}
}

func (t *goldTable[V]) lookupPrefixReverse(pfx netip.Prefix) []netip.Prefix {
	var result []netip.Prefix

	for _, item := range *t {
		if item.pfx.Overlaps(pfx) && item.pfx.Bits() <= pfx.Bits() {
			result = append(result, item.pfx)
		}
	}

	// b,a reverse sort order!
	slices.SortFunc(result, func(a, b netip.Prefix) int {
		return cmpPrefix(b, a)
	})
	return result
}
