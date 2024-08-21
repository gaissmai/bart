// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"reflect"
	"slices"
	"testing"
)

func TestEachSubnetCompare(t *testing.T) {
	t.Parallel()

	pfxs := randomPrefixes(10_000)

	fast := &Table[int]{}
	gold := goldTable[int](pfxs)

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	for range 10_000 {
		pfx := randomPrefix()
		goldPfxs := gold.subnets(pfx)

		var fastPfxs []netip.Prefix
		values := map[netip.Prefix]int{}

		fast.EachSubnet(pfx, func(p netip.Prefix, val int) bool {
			fastPfxs = append(fastPfxs, p)
			values[p] = val
			return true
		})

		if !reflect.DeepEqual(goldPfxs, fastPfxs) {
			t.Fatalf("\nEachSubnets(%q):\ngot:  %v\nwant: %v", pfx, fastPfxs, goldPfxs)
		}

		// also check the values handled by yield function
		for pfx, val := range values {
			got, ok := fast.Get(pfx)

			if !ok || got != val {
				t.Fatalf("EachSubnets: Get(%q), got: %d,%v, want: %d,%v", pfx, got, ok, val, true)
			}
		}
	}
}

func TestEachLookupPrefix(t *testing.T) {
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
		fast.EachLookupPrefix(pfx, func(p netip.Prefix, _ int) bool {
			fastPfxs = append(fastPfxs, p)
			return true
		})

		if !reflect.DeepEqual(goldPfxs, fastPfxs) {
			t.Fatalf("\nEachSupernet(%q):\ngot:  %v\nwant: %v", pfx, fastPfxs, goldPfxs)
		}

	}
}

func TestSupernetsEdgeCaseCB(t *testing.T) {
	t.Parallel()

	rtbl := new(Table[any])
	pfx := mpp("::1/128")
	rtbl.Supernets(pfx)(func(_ netip.Prefix, _ any) bool {
		t.Errorf("empty table, must not range over")
		return false
	})

	val := "foo"
	rtbl.Insert(pfx, val)
	rtbl.Supernets(netip.Prefix{})(func(_ netip.Prefix, _ any) bool {
		t.Errorf("invalid prefix, must not range over")
		return false
	})

	rtbl.Supernets(netip.Prefix{})(func(p netip.Prefix, v any) bool {
		if p != pfx {
			t.Errorf("Supernets(%v), got: %v, want: %v", pfx, p, pfx)
			return false
		}

		if v.(string) != val {
			t.Errorf("Supernets(%v), got: %v, want: %v", pfx, v.(string), val)
			return false
		}
		return true
	})
}

func TestSubnetsCB(t *testing.T) {
	t.Parallel()

	rtbl := new(Table[any])
	pfx := mpp("::1/128")
	rtbl.Supernets(pfx)(func(_ netip.Prefix, _ any) bool {
		t.Errorf("empty table, must not range over")
		return false
	})

	val := "foo"
	rtbl.Insert(pfx, val)
	rtbl.Supernets(netip.Prefix{})(func(_ netip.Prefix, _ any) bool {
		t.Errorf("invalid prefix, must not range over")
		return false
	})

	rtbl.Supernets(pfx)(func(p netip.Prefix, v any) bool {
		if p != pfx {
			t.Errorf("Subnet(%v), got: %v, want: %v", pfx, p, pfx)
			return false
		}

		if v.(string) != val {
			t.Errorf("Subnet(%v), got: %v, want: %v", pfx, v.(string), val)
			return false
		}
		return true
	})
}

func TestAll(t *testing.T) {
	t.Parallel()

	pfxs := randomPrefixes(10_000)

	t.Run("All", func(t *testing.T) {
		rtbl := new(Table[int])
		seen := make(map[netip.Prefix]int, 10_000)
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
			seen[item.pfx] = item.val
		}

		// check if pfx/val is as expected
		rtbl.All()(func(pfx netip.Prefix, val int) bool {
			if seen[pfx] != val {
				t.Errorf("%v got value: %v, expected: %v", pfx, val, seen[pfx])
			}
			delete(seen, pfx)
			return true
		})

		// check if all entries visited
		if len(seen) != 0 {
			t.Fatalf("traverse error, not all entries visited")
		}
	})

	t.Run("All_4&6", func(t *testing.T) {
		rtbl := new(Table[int])
		seen := make(map[netip.Prefix]int, 10_000)
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
			seen[item.pfx] = item.val
		}

		// check if pfx/val is as expected
		rtbl.All4()(func(pfx netip.Prefix, val int) bool {
			if seen[pfx] != val {
				t.Errorf("%v got value: %v, expected: %v", pfx, val, seen[pfx])
			}
			delete(seen, pfx)
			return true
		})

		rtbl.All6()(func(pfx netip.Prefix, val int) bool {
			if seen[pfx] != val {
				t.Errorf("%v got value: %v, expected: %v", pfx, val, seen[pfx])
			}
			delete(seen, pfx)
			return true
		})

		// check if all entries visited
		if len(seen) != 0 {
			t.Fatalf("traverse error, not all entries visited")
		}
	})

	// make an iteration and update the values in the callback
	t.Run("All and Update", func(t *testing.T) {
		rtbl := new(Table[int])
		seen := make(map[netip.Prefix]int, 10_000)
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
			seen[item.pfx] = item.val + 1
		}

		// update callback, add 1 to val
		updateValue := func(val int, _ bool) int {
			return val + 1
		}

		yield := func(pfx netip.Prefix, _ int) bool {
			rtbl.Update(pfx, updateValue)
			return true
		}

		// iterate and update the values
		rtbl.All()(yield)

		// test if all values got updated, yield now as closure
		rtbl.All()(func(pfx netip.Prefix, val int) bool {
			if seen[pfx] != val {
				t.Errorf("%v got value: %v, expected: %v", pfx, val, seen[pfx])
			}
			return true
		})
	})

	t.Run("All with premature exit", func(t *testing.T) {
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
		}

		// check if callback stops prematurely
		countV6 := 0
		rtbl.All()(func(pfx netip.Prefix, _ int) bool {
			// max 1000 IPv6 prefixes
			if !pfx.Addr().Is4() {
				countV6++
			}

			// premature STOP condition
			return countV6 < 1000
		})

		// check if iteration stopped with error
		if countV6 > 1000 {
			t.Fatalf("expected premature stop with error")
		}
	})
}

// After go version 1.22 we can use range iterators
func TestAllSorted(t *testing.T) {
	t.Parallel()

	n := 10_000

	pfxs := randomPrefixes(n)

	t.Run("All versus slices.SortFunc", func(t *testing.T) {
		t.Parallel()
		expect := make([]netip.Prefix, 0, n)
		got := make([]netip.Prefix, 0, n)

		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
			expect = append(expect, item.pfx)
		}

		slices.SortFunc(expect, cmpPrefix)

		rtbl.AllSorted()(func(pfx netip.Prefix, _ int) bool {
			got = append(got, pfx)
			return true
		})

		if !reflect.DeepEqual(got, expect) {
			t.Fatalf("All differs with slices.SortFunc")
		}
	})
}

func BenchmarkAll(b *testing.B) {
	n := 100_000

	rtbl := new(Table[int])
	for _, item := range randomPrefixes(n) {
		rtbl.Insert(item.pfx, item.val)
	}

	b.Run("All", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			rtbl.All()(func(_ netip.Prefix, _ int) bool {
				return true
			})
		}
	})

	b.Run("AllSorted", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			rtbl.AllSorted()(func(_ netip.Prefix, _ int) bool {
				return true
			})
		}
	})
}
