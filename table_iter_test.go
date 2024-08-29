//go:build go1.23

// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"reflect"
	"testing"
)

func TestTableIsEmptyIterators(t *testing.T) {
	rt := new(Table[byte])

	_ = rt.All()
	if rt.isInit() {
		t.Error("All() changed the init state of Table")
	}

	_ = rt.All4()
	if rt.isInit() {
		t.Error("All4() changed the init state of Table")
	}

	_ = rt.All6()
	if rt.isInit() {
		t.Error("All6() changed the init state of Table")
	}

	_ = rt.AllSorted()
	if rt.isInit() {
		t.Error("AllSorted() changed the init state of Table")
	}

	_ = rt.AllSorted4()
	if rt.isInit() {
		t.Error("AllSorted4() changed the init state of Table")
	}

	_ = rt.AllSorted6()
	if rt.isInit() {
		t.Error("AllSorted6() changed the init state of Table")
	}

	pfx := mpp("1.2.3.4/32")
	_ = rt.Subnets(pfx)
	if rt.isInit() {
		t.Error("Subnets() changed the init state of Table")
	}

	_ = rt.Supernets(pfx)
	if rt.isInit() {
		t.Error("Supernets() changed the init state of Table")
	}
}

func TestAll4RangeOverFunc(t *testing.T) {
	pfxs := randomPrefixes4(10_000)
	seen := make(map[netip.Prefix]int, 10_000)

	t.Run("All4RangeOverFunc", func(t *testing.T) {
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
	pfxs := randomPrefixes6(10_000)
	seen := make(map[netip.Prefix]int, 10_000)

	t.Run("All6RangeOverFunc", func(t *testing.T) {
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
	pfxs := randomPrefixes(10_000)
	seen := make(map[netip.Prefix]int, 10_000)

	t.Run("AllRangeOverFunc", func(t *testing.T) {
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
	pfxs := randomPrefixes4(10_000)
	seen := make(map[netip.Prefix]int, 10_000)

	t.Run("All4SortedRangeOverFunc", func(t *testing.T) {
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
	pfxs := randomPrefixes6(10_000)
	seen := make(map[netip.Prefix]int, 10_000)

	t.Run("All6SortedRangeOverFunc", func(t *testing.T) {
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
	pfxs := randomPrefixes(10_000)
	seen := make(map[netip.Prefix]int, 10_000)

	t.Run("AllSortedRangeOverFunc", func(t *testing.T) {
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

	rtbl := new(Table[any])
	pfx := mpp("::1/128")
	for range rtbl.Subnets(pfx) {
		t.Errorf("empty table, must not range over")
	}

	val := "foo"
	rtbl.Insert(pfx, val)
	for range rtbl.Subnets(netip.Prefix{}) {
		t.Errorf("invalid prefix, must not range over")
	}

	for p, v := range rtbl.Subnets(pfx) {
		if p != pfx {
			t.Errorf("Subnet(%v), got: %v, want: %v", pfx, p, pfx)
		}

		if v.(string) != val {
			t.Errorf("Subnet(%v), got: %v, want: %v", pfx, v.(string), val)
		}
	}
}
