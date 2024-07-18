// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

//go:build go1.23

// rangefunc iterators, only test it with 1.23
// or 1.22 and:
//  GOEXPERIMENT=rangefunc go test ...

package bart

import (
	"net/netip"
	"testing"
)

func TestAll4Iter(t *testing.T) {
	pfxs := randomPrefixes4(10_000)
	seen := make(map[netip.Prefix]int, 10_000)

	t.Run("All4Iter", func(t *testing.T) {
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
			seen[item.pfx] = item.val
		}

		// rangefunc iterator
		for pfx, val := range rtbl.All4Iter() {
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

	t.Run("All4Iter with premature exit", func(t *testing.T) {
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
		}

		// check if callback stops prematurely
		count := 0
		for _, _ = range rtbl.All4Iter() {
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

func TestAll6Iter(t *testing.T) {
	pfxs := randomPrefixes6(10_000)
	seen := make(map[netip.Prefix]int, 10_000)

	t.Run("All6Iter", func(t *testing.T) {
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
			seen[item.pfx] = item.val
		}

		// rangefunc iterator
		for pfx, val := range rtbl.All6Iter() {
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

	t.Run("All6Iter with premature exit", func(t *testing.T) {
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
		}

		// check if callback stops prematurely
		count := 0
		for _, _ = range rtbl.All6Iter() {
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

func TestAllIter(t *testing.T) {
	pfxs := randomPrefixes(10_000)
	seen := make(map[netip.Prefix]int, 10_000)

	t.Run("AllIter", func(t *testing.T) {
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
			seen[item.pfx] = item.val
		}

		// rangefunc iterator
		for pfx, val := range rtbl.AllIter() {
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

	t.Run("AllIter with premature exit", func(t *testing.T) {
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
		}

		// check if callback stops prematurely
		count := 0
		for _, _ = range rtbl.AllIter() {
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

	t.Run("All4SortedIter", func(t *testing.T) {
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
			seen[item.pfx] = item.val
		}

		// rangefunc iterator
		for pfx, val := range rtbl.All4SortedIter() {
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

	t.Run("All4SortedIter with premature exit", func(t *testing.T) {
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
		}

		// check if callback stops prematurely
		count := 0
		for _, _ = range rtbl.All4SortedIter() {
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

func TestAll6SortedIter(t *testing.T) {
	pfxs := randomPrefixes6(10_000)
	seen := make(map[netip.Prefix]int, 10_000)

	t.Run("All6SortedIter", func(t *testing.T) {
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
			seen[item.pfx] = item.val
		}

		// rangefunc iterator
		for pfx, val := range rtbl.All6SortedIter() {
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

	t.Run("All6SortedIter with premature exit", func(t *testing.T) {
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
		}

		// check if callback stops prematurely
		count := 0
		for _, _ = range rtbl.All6SortedIter() {
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

func TestAllSortedIter(t *testing.T) {
	pfxs := randomPrefixes(10_000)
	seen := make(map[netip.Prefix]int, 10_000)

	t.Run("AllSortedIter", func(t *testing.T) {
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
			seen[item.pfx] = item.val
		}

		// rangefunc iterator
		for pfx, val := range rtbl.AllSortedIter() {
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

	t.Run("AllSortedIter with premature exit", func(t *testing.T) {
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
		}

		// check if callback stops prematurely
		count := 0
		for _, _ = range rtbl.AllSortedIter() {
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

func TestLookupPrefixIterEdgeCase(t *testing.T) {
	t.Parallel()

	rtbl := new(Table[any])
	pfx := mpp("::1/128")
	for _, _ = range rtbl.LookupPrefixIter(pfx) {
		t.Errorf("empty table, must not range over")
	}

	val := "foo"
	rtbl.Insert(pfx, val)
	for _, _ = range rtbl.LookupPrefixIter(netip.Prefix{}) {
		t.Errorf("invalid prefix, must not range over")
	}

	for p, v := range rtbl.LookupPrefixIter(pfx) {
		if p != pfx {
			t.Errorf("LookupPrefixIter(%v), got: %v, want: %v", pfx, p, pfx)
		}

		if v.(string) != val {
			t.Errorf("LookupPrefixIter(%v), got: %v, want: %v", pfx, v.(string), val)
		}
	}

}

func TestSubnetIter(t *testing.T) {
	t.Parallel()

	rtbl := new(Table[any])
	pfx := mpp("::1/128")
	for _, _ = range rtbl.SubnetIter(pfx) {
		t.Errorf("empty table, must not range over")
	}

	val := "foo"
	rtbl.Insert(pfx, val)
	for _, _ = range rtbl.SubnetIter(netip.Prefix{}) {
		t.Errorf("invalid prefix, must not range over")
	}

	for p, v := range rtbl.SubnetIter(pfx) {
		if p != pfx {
			t.Errorf("SubnetIter(%v), got: %v, want: %v", pfx, p, pfx)
		}

		if v.(string) != val {
			t.Errorf("SubnetIter(%v), got: %v, want: %v", pfx, v.(string), val)
		}
	}

}
