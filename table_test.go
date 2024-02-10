// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause
//
// tests and benchmarks copied from github.com/tailscale/art
// and modified for this implementation by:
//
// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	crand "crypto/rand"
	"fmt"
	"math/rand"
	"net/netip"
	"runtime"
	"strconv"
	"testing"
	"time"
)

func TestRegression(t *testing.T) {
	// original comment by tailscale for ART,
	// but the BART implementation is different and has other edge cases.
	//
	// These tests are specific triggers for subtle correctness issues
	// that came up during initial implementation. Even if they seem
	// arbitrary, please do not clean them up. They are checking edge
	// cases that are very easy to get wrong, and quite difficult for
	// the other statistical tests to trigger promptly.

	t.Run("prefixes_aligned_on_stride_boundary", func(t *testing.T) {
		tbl := &Table[int]{}
		slow := slowPrefixTable[int]{}
		p := netip.MustParsePrefix

		tbl.Insert(p("226.205.197.0/24"), 1)
		slow.insert(p("226.205.197.0/24"), 1)
		tbl.Insert(p("226.205.0.0/16"), 2)
		slow.insert(p("226.205.0.0/16"), 2)

		probe := netip.MustParseAddr("226.205.121.152")
		got, gotOK := tbl.Get(probe)
		want, wantOK := slow.get(probe)
		if !getsEqual(got, gotOK, want, wantOK) {
			t.Fatalf("got (%v, %v), want (%v, %v)", got, gotOK, want, wantOK)
		}
	})

	t.Run("parent_prefix_inserted_in_different_orders", func(t *testing.T) {
		t1, t2 := &Table[int]{}, &Table[int]{}
		p := netip.MustParsePrefix

		t1.Insert(p("136.20.0.0/16"), 1)
		t1.Insert(p("136.20.201.62/32"), 2)

		t2.Insert(p("136.20.201.62/32"), 2)
		t2.Insert(p("136.20.0.0/16"), 1)

		a := netip.MustParseAddr("136.20.54.139")
		got1, ok1 := t1.Get(a)
		got2, ok2 := t2.Get(a)
		if !getsEqual(got1, ok1, got2, ok2) {
			t.Errorf("Get(%q) is insertion order dependent: t1=(%v, %v), t2=(%v, %v)", a, got1, ok1, got2, ok2)
		}
	})

	t.Run("overlaps_divergent_children_with_parent_route_entry", func(t *testing.T) {
		t1, t2 := Table[int]{}, Table[int]{}
		p := netip.MustParsePrefix

		t1.Insert(p("128.0.0.0/2"), 1)
		t1.Insert(p("99.173.128.0/17"), 1)
		t1.Insert(p("219.150.142.0/23"), 1)
		t1.Insert(p("164.148.190.250/31"), 1)
		t1.Insert(p("48.136.229.233/32"), 1)

		t2.Insert(p("217.32.0.0/11"), 1)
		t2.Insert(p("38.176.0.0/12"), 1)
		t2.Insert(p("106.16.0.0/13"), 1)
		t2.Insert(p("164.85.192.0/23"), 1)
		t2.Insert(p("225.71.164.112/31"), 1)

		if !t1.Overlaps(&t2) {
			t.Fatalf("tables unexpectedly do not overlap")
		}
	})

	t.Run("overlaps_parent_child_comparison_with_route_in_parent", func(t *testing.T) {
		t1, t2 := Table[int]{}, Table[int]{}
		p := netip.MustParsePrefix

		t1.Insert(p("226.0.0.0/8"), 1)
		t1.Insert(p("81.128.0.0/9"), 1)
		t1.Insert(p("152.0.0.0/9"), 1)
		t1.Insert(p("151.220.0.0/16"), 1)
		t1.Insert(p("89.162.61.0/24"), 1)

		t2.Insert(p("54.0.0.0/9"), 1)
		t2.Insert(p("35.89.128.0/19"), 1)
		t2.Insert(p("72.33.53.0/24"), 1)
		t2.Insert(p("2.233.60.32/27"), 1)
		t2.Insert(p("152.42.142.160/28"), 1)

		if !t1.Overlaps(&t2) {
			t.Fatalf("tables unexpectedly do not overlap")
		}
	})
}

func TestDelete(t *testing.T) {
	t.Parallel()
	p := func(s string) netip.Prefix {
		pfx := netip.MustParsePrefix(s)
		if pfx.Addr() != pfx.Masked().Addr() {
			panic(fmt.Sprintf("%s is not normalized", s))
		}
		return pfx
	}

	t.Run("prefix_in_root", func(t *testing.T) {
		// Add/remove prefix from root table.
		tbl := &Table[int]{}
		checkSize(t, tbl, 2)

		tbl.Insert(p("10.0.0.0/8"), 1)
		checkRoutes(t, tbl, []tableTest{
			{"10.0.0.1", 1, 1},
			{"255.255.255.255", -1, -1},
		})
		checkSize(t, tbl, 2)
		tbl.Delete(p("10.0.0.0/8"))
		checkRoutes(t, tbl, []tableTest{
			{"10.0.0.1", -1, -1},
			{"255.255.255.255", -1, -1},
		})
		checkSize(t, tbl, 2)
	})

	t.Run("prefix_in_leaf", func(t *testing.T) {
		// Create, then delete a single leaf table.
		tbl := &Table[int]{}
		checkSize(t, tbl, 2)

		tbl.Insert(p("192.168.0.1/32"), 1)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1, 1},
			{"255.255.255.255", -1, -1},
		})
		tbl.Delete(p("192.168.0.1/32"))
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", -1, -1},
			{"255.255.255.255", -1, -1},
		})
		checkSize(t, tbl, 2)
	})

	t.Run("intermediate_no_routes", func(t *testing.T) {
		// Create an intermediate with 2 children, then delete one leaf.
		tbl := &Table[int]{}
		checkSize(t, tbl, 2)
		tbl.Insert(p("192.168.0.1/32"), 1)
		tbl.Insert(p("192.180.0.1/32"), 2)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1, 1},
			{"192.180.0.1", 2, 2},
			{"192.40.0.1", -1, -1},
		})
		checkSize(t, tbl, 7) // 2 roots, 3 intermediate, 2 leaves
		tbl.Delete(p("192.180.0.1/32"))
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1, 1},
			{"192.180.0.1", -1, -1},
			{"192.40.0.1", -1, -1},
		})
		checkSize(t, tbl, 5) // 2 roots, 2 intermediates, 1 leaf
	})

	t.Run("intermediate_with_route", func(t *testing.T) {
		// Same, but the intermediate carries a route as well.
		tbl := &Table[int]{}
		checkSize(t, tbl, 2)
		tbl.Insert(p("192.168.0.1/32"), 1)
		tbl.Insert(p("192.180.0.1/32"), 2)
		tbl.Insert(p("192.0.0.0/10"), 3)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1, 1},
			{"192.180.0.1", 2, 2},
			{"192.40.0.1", 3, 3},
			{"192.255.0.1", -1, -1},
		})
		checkSize(t, tbl, 7) // 2 roots, 2 intermediates, 2 leaves
		tbl.Delete(p("192.180.0.1/32"))
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1, 1},
			{"192.180.0.1", -1, -1},
			{"192.40.0.1", 3, 3},
			{"192.255.0.1", -1, -1},
		})
		checkSize(t, tbl, 5) // 2 roots, 1 full, 1 intermediate, 1 leaf
	})

	t.Run("intermediate_many_leaves", func(t *testing.T) {
		// Intermediate with 3 leaves, then delete one leaf.
		tbl := &Table[int]{}
		checkSize(t, tbl, 2)
		tbl.Insert(p("192.168.0.1/32"), 1)
		tbl.Insert(p("192.180.0.1/32"), 2)
		tbl.Insert(p("192.200.0.1/32"), 3)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1, 1},
			{"192.180.0.1", 2, 2},
			{"192.200.0.1", 3, 3},
			{"192.255.0.1", -1, -1},
		})
		checkSize(t, tbl, 9) // 2 roots, 4 intermediate, 3 leaves
		tbl.Delete(p("192.180.0.1/32"))
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1, 1},
			{"192.180.0.1", -1, -1},
			{"192.200.0.1", 3, 3},
			{"192.255.0.1", -1, -1},
		})
		checkSize(t, tbl, 7) // 2 roots, 3 intermediate, 2 leaves
	})

	t.Run("nosuchprefix_missing_child", func(t *testing.T) {
		// Delete non-existent prefix, missing strideTable path.
		tbl := &Table[int]{}
		checkSize(t, tbl, 2)
		tbl.Insert(p("192.168.0.1/32"), 1)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1, 1},
			{"192.255.0.1", -1, -1},
		})
		checkSize(t, tbl, 5)          // 2 roots, 2 intermediate, 1 leaf
		tbl.Delete(p("200.0.0.0/32")) // lookup miss in root
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1, 1},
			{"192.255.0.1", -1, -1},
		})
		checkSize(t, tbl, 5) // 2 roots, 2 intermediate, 1 leaf
	})

	t.Run("nosuchprefix_not_in_leaf", func(t *testing.T) {
		// Delete non-existent prefix, strideTable path exists but
		// leaf doesn't contain route.
		tbl := &Table[int]{}
		checkSize(t, tbl, 2)
		tbl.Insert(p("192.168.0.1/32"), 1)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1, 1},
			{"192.255.0.1", -1, -1},
		})
		checkSize(t, tbl, 5)            // 2 roots, 2 intermediate, 1 leaf
		tbl.Delete(p("192.168.0.5/32")) // right leaf, no route
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1, 1},
			{"192.255.0.1", -1, -1},
		})
		checkSize(t, tbl, 5) // 2 roots, 2 intermediate, 1 leaf
	})

	t.Run("intermediate_with_deleted_route", func(t *testing.T) {
		// Intermediate table loses its last route and becomes
		// compactable.
		tbl := &Table[int]{}
		checkSize(t, tbl, 2)
		tbl.Insert(p("192.168.0.1/32"), 1)
		tbl.Insert(p("192.168.0.0/22"), 2)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1, 2},
			{"192.168.0.2", 2, 2},
			{"192.255.0.1", -1, -1},
		})
		checkSize(t, tbl, 5) // 2 roots, 1 intermediate, 1 full, 1 leaf
		tbl.Delete(p("192.168.0.0/22"))
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1, 1},
			{"192.168.0.2", -1, -1},
			{"192.255.0.1", -1, -1},
		})
		checkSize(t, tbl, 5) // 2 roots, 2 intermediate, 1 leaf
	})

	t.Run("default_route", func(t *testing.T) {
		// Default routes have a special case in the code.
		tbl := &Table[int]{}

		tbl.Insert(p("0.0.0.0/0"), 1)
		tbl.Insert(p("::/0"), 1)
		tbl.Delete(p("0.0.0.0/0"))

		checkRoutes(t, tbl, []tableTest{
			{"1.2.3.4", -1, -1},
			{"::1", 1, 1},
		})
		checkSize(t, tbl, 2) // 2 roots
	})
}

func TestInsertCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()
	pfxs := randomPrefixes(10_000)

	slow := slowPrefixTable[int]{pfxs}
	fast := Table[int]{}

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	seenVals4 := map[int]bool{}
	seenVals6 := map[int]bool{}
	for i := 0; i < 10_000; i++ {
		a := randomAddr()
		slowVal, slowOK := slow.get(a)
		fastVal, fastOK := fast.Get(a)
		if !getsEqual(slowVal, slowOK, fastVal, fastOK) {
			t.Fatalf("get(%q) = (%v, %v), want (%v, %v)", a, fastVal, fastOK, slowVal, slowOK)
		}

		if a.Is6() {
			seenVals6[fastVal] = true
		} else {
			seenVals4[fastVal] = true
		}

		slowPfx, slowVal, slowOK := slow.lpm(a)
		fastPfx, fastVal, fastOK := fast.Lookup(a)
		if slowPfx != fastPfx {
			t.Fatalf("lpm(%q) = (%v, %v, %v), want (%v, %v, %v)", a, fastPfx, fastVal, fastOK, slowPfx, slowVal, slowOK)
		}
		if !getsEqual(slowVal, slowOK, fastVal, fastOK) {
			t.Fatalf("lpm(%q) = (%v, %v), want (%v, %v)", a, fastVal, fastOK, slowVal, slowOK)
		}
	}

	// Empirically, 10k probes into 5k v4 prefixes and 5k v6 prefixes results in
	// ~1k distinct values for v4 and ~300 for v6. distinct routes. This sanity
	// check that we didn't just return a single route for everything should be
	// very generous indeed.
	if cnt := len(seenVals4); cnt < 10 {
		t.Fatalf("saw %d distinct v4 route results, statistically expected ~1000", cnt)
	}
	if cnt := len(seenVals6); cnt < 10 {
		t.Fatalf("saw %d distinct v6 route results, statistically expected ~300", cnt)
	}
}

func TestInsertShuffled(t *testing.T) {
	// The order in which you insert prefixes into a route table
	// should not matter, as long as you're inserting the same set of
	// routes.
	t.Parallel()
	pfxs := randomPrefixes(1000)
	var pfxs2 []slowPrefixEntry[int]

	defer func() {
		if t.Failed() {
			t.Logf("pre-shuffle: %#v", pfxs)
			t.Logf("post-shuffle: %#v", pfxs2)
		}
	}()

	for i := 0; i < 10; i++ {
		pfxs2 := append([]slowPrefixEntry[int](nil), pfxs...)
		rand.Shuffle(len(pfxs2), func(i, j int) { pfxs2[i], pfxs2[j] = pfxs2[j], pfxs2[i] })

		addrs := make([]netip.Addr, 0, 10_000)
		for i := 0; i < 10_000; i++ {
			addrs = append(addrs, randomAddr())
		}

		rt := Table[int]{}
		rt2 := Table[int]{}

		for _, pfx := range pfxs {
			rt.Insert(pfx.pfx, pfx.val)
		}
		for _, pfx := range pfxs2 {
			rt2.Insert(pfx.pfx, pfx.val)
		}

		for _, a := range addrs {
			val1, ok1 := rt.Get(a)
			val2, ok2 := rt2.Get(a)
			if !getsEqual(val1, ok1, val2, ok2) {
				t.Fatalf("get(%q) = (%v, %v), want (%v, %v)", a, val2, ok2, val1, ok1)
			}
		}
	}
}

func TestDeleteCompare(t *testing.T) {
	// Create large route tables repeatedly, delete half of their
	// prefixes, and compare Table's behavior to a naive and slow but
	// correct implementation.
	t.Parallel()

	const (
		numPrefixes  = 10_000 // total prefixes to insert (test deletes 50% of them)
		numPerFamily = numPrefixes / 2
		deleteCut    = numPerFamily / 2
		numProbes    = 10_000 // random addr lookups to do
	)

	// We have to do this little dance instead of just using allPrefixes,
	// because we want pfxs and toDelete to be non-overlapping sets.
	all4, all6 := randomPrefixes4(numPerFamily), randomPrefixes6(numPerFamily)
	pfxs := append([]slowPrefixEntry[int](nil), all4[:deleteCut]...)
	pfxs = append(pfxs, all6[:deleteCut]...)
	toDelete := append([]slowPrefixEntry[int](nil), all4[deleteCut:]...)
	toDelete = append(toDelete, all6[deleteCut:]...)

	defer func() {
		if t.Failed() {
			for _, pfx := range pfxs {
				fmt.Printf("%q, ", pfx.pfx)
			}
			fmt.Println("")
			for _, pfx := range toDelete {
				fmt.Printf("%q, ", pfx.pfx)
			}
			fmt.Println("")
		}
	}()

	slow := slowPrefixTable[int]{pfxs}
	fast := Table[int]{}

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	for _, pfx := range toDelete {
		fast.Insert(pfx.pfx, pfx.val)
	}
	for _, pfx := range toDelete {
		fast.Delete(pfx.pfx)
	}

	seenVals4 := map[int]bool{}
	seenVals6 := map[int]bool{}
	for i := 0; i < numProbes; i++ {
		a := randomAddr()
		slowVal, slowOK := slow.get(a)
		fastVal, fastOK := fast.Get(a)
		if !getsEqual(slowVal, slowOK, fastVal, fastOK) {
			t.Fatalf("get(%q) = (%v, %v), want (%v, %v)", a, fastVal, fastOK, slowVal, slowOK)
		}
		if a.Is6() {
			seenVals6[fastVal] = true
		} else {
			seenVals4[fastVal] = true
		}

		slowPfx, slowVal, slowOK := slow.lpm(a)
		fastPfx, fastVal, fastOK := fast.Lookup(a)
		if slowPfx != fastPfx {
			t.Fatalf("lpm(%q) = (%v, %v, %v), want (%v, %v, %v)", a, fastPfx, fastVal, fastOK, slowPfx, slowVal, slowOK)
		}
		if !getsEqual(slowVal, slowOK, fastVal, fastOK) {
			t.Fatalf("lpm(%q) = (%v, %v), want (%v, %v)", a, fastVal, fastOK, slowVal, slowOK)
		}
	}
	// Empirically, 10k probes into 5k v4 prefixes and 5k v6 prefixes results in
	// ~1k distinct values for v4 and ~300 for v6. distinct routes. This sanity
	// check that we didn't just return a single route for everything should be
	// very generous indeed.
	if cnt := len(seenVals4); cnt < 10 {
		t.Fatalf("saw %d distinct v4 route results, statistically expected ~1000", cnt)
	}
	if cnt := len(seenVals6); cnt < 10 {
		t.Fatalf("saw %d distinct v6 route results, statistically expected ~300", cnt)
	}
}

func TestDeleteShuffled(t *testing.T) {
	// The order in which you delete prefixes from a route table
	// should not matter, as long as you're deleting the same set of
	// routes.
	t.Parallel()

	const (
		numPrefixes  = 10_000 // prefixes to insert (test deletes 50% of them)
		numPerFamily = numPrefixes / 2
		deleteCut    = numPerFamily / 2
		numProbes    = 10_000 // random addr lookups to do
	)

	// We have to do this little dance instead of just using allPrefixes,
	// because we want pfxs and toDelete to be non-overlapping sets.
	all4, all6 := randomPrefixes4(numPerFamily), randomPrefixes6(numPerFamily)
	pfxs := append([]slowPrefixEntry[int](nil), all4[:deleteCut]...)
	pfxs = append(pfxs, all6[:deleteCut]...)
	toDelete := append([]slowPrefixEntry[int](nil), all4[deleteCut:]...)
	toDelete = append(toDelete, all6[deleteCut:]...)

	rt := Table[int]{}
	for _, pfx := range pfxs {
		rt.Insert(pfx.pfx, pfx.val)
	}
	for _, pfx := range toDelete {
		rt.Insert(pfx.pfx, pfx.val)
	}
	for _, pfx := range toDelete {
		rt.Delete(pfx.pfx)
	}

	for i := 0; i < 10; i++ {
		pfxs2 := append([]slowPrefixEntry[int](nil), pfxs...)
		toDelete2 := append([]slowPrefixEntry[int](nil), toDelete...)
		rand.Shuffle(len(toDelete2), func(i, j int) { toDelete2[i], toDelete2[j] = toDelete2[j], toDelete2[i] })
		rt2 := Table[int]{}
		for _, pfx := range pfxs2 {
			rt2.Insert(pfx.pfx, pfx.val)
		}
		for _, pfx := range toDelete2 {
			rt2.Insert(pfx.pfx, pfx.val)
		}
		for _, pfx := range toDelete2 {
			rt2.Delete(pfx.pfx)
		}

		// Diffing a deep tree of tables gives cmp.Diff a nervous breakdown, so
		// test for equivalence statistically with random probes instead.
		for i := 0; i < numProbes; i++ {
			a := randomAddr()
			val1, ok1 := rt.Get(a)
			val2, ok2 := rt2.Get(a)
			if !getsEqual(val1, ok1, val2, ok2) {
				t.Errorf("get(%q) = (%v, %v), want (%v, %v)", a, val2, ok2, val1, ok1)
			}
		}
	}
}

func TestDeleteIsReverseOfInsert(t *testing.T) {
	// Insert N prefixes, then delete those same prefixes in reverse
	// order. Each deletion should exactly undo the internal structure
	// changes that each insert did.
	const N = 100

	var tab Table[int]
	prefixes := randomPrefixes(N)

	defer func() {
		if t.Failed() {
			fmt.Printf("the prefixes that fail the test: %v\n", prefixes)
		}
	}()

	want := tab.dumpString()
	for _, p := range prefixes {
		tab.Insert(p.pfx, p.val)
	}

	for i := len(prefixes) - 1; i >= 0; i-- {
		tab.Delete(prefixes[i].pfx)
	}
	if got := tab.dumpString(); got != want {
		t.Fatalf("after delete, mismatch:\n\n got: %s\n\nwant: %s", got, want)
	}
}

func TestShortestCompare(t *testing.T) {
	t.Parallel()
	pfxs := randomPrefixes(10_000)

	slow := slowPrefixTable[int]{pfxs}
	fast := Table[int]{}

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	for i := 0; i < 10_000; i++ {
		a := randomAddr()
		slowPfx, slowVal, slowOK := slow.spm(a)
		fastPfx, fastVal, fastOK := fast.LookupShortest(a)
		if slowPfx != fastPfx {
			t.Fatalf("spm(%q) = (%v, %v, %v), want (%v, %v, %v)", a, fastPfx, fastVal, fastOK, slowPfx, slowVal, slowOK)
		}
		if !getsEqual(slowVal, slowOK, fastVal, fastOK) {
			t.Fatalf("spm(%q) = (%v, %v), want (%v, %v)", a, fastVal, fastOK, slowVal, slowOK)
		}
	}
}

func TestOverlapsCompare(t *testing.T) {
	t.Parallel()

	// Empirically, between 5 and 6 routes per table results in ~50%
	// of random pairs overlapping. Cool example of the birthday
	// paradox!
	const numEntries = 6

	seen := map[bool]int{}
	for i := 0; i < 10000; i++ {
		pfxs := randomPrefixes(numEntries)
		slow := slowPrefixTable[int]{pfxs}
		fast := Table[int]{}
		for _, pfx := range pfxs {
			fast.Insert(pfx.pfx, pfx.val)
		}

		inter := randomPrefixes(numEntries)
		slowInter := slowPrefixTable[int]{inter}
		fastInter := Table[int]{}
		for _, pfx := range inter {
			fastInter.Insert(pfx.pfx, pfx.val)
		}

		gotSlow := slow.overlaps(&slowInter)
		gotFast := fast.Overlaps(&fastInter)

		if gotSlow != gotFast {
			t.Fatalf("Overlaps(...) = %v, want %v\nTable1:\n%s\nTable2:\n%v",
				gotFast, gotSlow, fast.String(), fastInter.String())
		}

		seen[gotFast]++
	}

	t.Log(seen)
}

func TestOverlapsPrefixCompare(t *testing.T) {
	t.Parallel()
	pfxs := randomPrefixes(100_000)

	slow := slowPrefixTable[int]{pfxs}
	fast := Table[int]{}

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	tests := randomPrefixes(10_000)
	for _, tt := range tests {
		gotSlow := slow.overlapsPrefix(tt.pfx)
		gotFast := fast.OverlapsPrefix(tt.pfx)
		if gotSlow != gotFast {
			t.Fatalf("overlapsPrefix(%q) = %v, want %v", tt.pfx, gotFast, gotSlow)
		}
	}
}

// test some edge cases
func TestOverlapsPrefixEdgeCases(t *testing.T) {
	t.Parallel()

	p := func(s string) netip.Prefix {
		pfx := netip.MustParsePrefix(s)
		if pfx.Addr() != pfx.Masked().Addr() {
			panic(fmt.Sprintf("%s is not normalized", s))
		}
		return pfx
	}

	tbl := &Table[int]{}

	// empty table
	checkOverlaps(t, tbl, []tableOverlapsTest{
		{"0.0.0.0/0", false},
		{"::/0", false},
	})

	// default route
	tbl.Insert(p("10.0.0.0/8"), 0)
	tbl.Insert(p("2001:db8::/32"), 0)
	checkOverlaps(t, tbl, []tableOverlapsTest{
		{"0.0.0.0/0", true},
		{"::/0", true},
	})

	// default route
	tbl = &Table[int]{}
	tbl.Insert(p("0.0.0.0/0"), 0)
	tbl.Insert(p("::/0"), 0)
	checkOverlaps(t, tbl, []tableOverlapsTest{
		{"10.0.0.0/8", true},
		{"2001:db8::/32", true},
	})

	// single IP
	tbl = &Table[int]{}
	tbl.Insert(p("10.0.0.0/7"), 0)
	tbl.Insert(p("2001::/16"), 0)
	checkOverlaps(t, tbl, []tableOverlapsTest{
		{"10.1.2.3/32", true},
		{"2001:db8:affe::cafe/128", true},
	})

	// single IPv
	tbl = &Table[int]{}
	tbl.Insert(p("10.1.2.3/32"), 0)
	tbl.Insert(p("2001:db8:affe::cafe/128"), 0)
	checkOverlaps(t, tbl, []tableOverlapsTest{
		{"10.0.0.0/7", true},
		{"2001::/16", true},
	})

	// same IPv
	tbl = &Table[int]{}
	tbl.Insert(p("10.1.2.3/32"), 0)
	tbl.Insert(p("2001:db8:affe::cafe/128"), 0)
	checkOverlaps(t, tbl, []tableOverlapsTest{
		{"10.1.2.3/32", true},
		{"2001:db8:affe::cafe/128", true},
	})
}

type tableOverlapsTest struct {
	prefix string
	want   bool
}

// checkOverlaps verifies that the overlaps lookups in tt return the
// expected results on tbl.
func checkOverlaps(t *testing.T, tbl *Table[int], tests []tableOverlapsTest) {
	p := func(s string) netip.Prefix {
		pfx := netip.MustParsePrefix(s)
		if pfx.Addr() != pfx.Masked().Addr() {
			panic(fmt.Sprintf("%s is not normalized", s))
		}
		return pfx
	}

	for _, tt := range tests {
		got := tbl.OverlapsPrefix(p(tt.prefix))
		if got != tt.want {
			t.Log(tbl.String())
			t.Errorf("OverlapsPrefix(%v) = %v, want %v", p(tt.prefix), got, tt.want)
		}
	}
}

type tableTest struct {
	// addr is an IP address string to look up in a route table.
	addr string
	// want is the expected >=0 value associated with the route, or -1
	// if we expect a lookup miss.
	want int
	// spm is the expected >=0 value associated with the spm route, or -1
	// if we expect a lookup miss.
	spm int
}

// checkRoutes verifies that the route lookups in tt return the
// expected results on tbl.
func checkRoutes(t *testing.T, tbl *Table[int], tt []tableTest) {
	a := netip.MustParseAddr
	t.Helper()
	for _, tc := range tt {
		v, ok := tbl.Get(a(tc.addr))

		if !ok && tc.want != -1 {
			t.Errorf("Get %q got (%v, %v), want (_, false)", tc.addr, v, ok)
		}
		if ok && v != tc.want {
			t.Errorf("Get %q got (%v, %v), want (%v, true)", tc.addr, v, ok, tc.want)
		}

		_, v, ok = tbl.Lookup(a(tc.addr))
		if !ok && tc.want != -1 {
			t.Errorf("Lookup %q got (%v, %v), want (_, false)", tc.addr, v, ok)
		}
		if ok && v != tc.want {
			t.Errorf("Lookup %q got (%v, %v), want (%v, true)", tc.addr, v, ok, tc.want)
		}

		_, v, ok = tbl.LookupShortest(a(tc.addr))
		if !ok && tc.spm != -1 {
			t.Errorf("LookupShortest %q got (%v, %v), want (_, false)", tc.addr, v, ok)
		}
		if ok && v != tc.spm {
			t.Errorf("LookupShortest %q got (%v, %v), want (%v, true)", tc.addr, v, ok, tc.spm)
		}
	}
}

var benchRouteCount = []int{10, 100, 1000, 10_000, 100_000}

// forFamilyAndCount runs the benchmark fn with different sets of
// routes.
//
// fn is called once for each combination of {addr_family, num_routes},
// where addr_family is ipv4 or ipv6, num_routes is the values in
// benchRouteCount.
func forFamilyAndCount(b *testing.B, fn func(b *testing.B, routes []slowPrefixEntry[int])) {
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}
		b.Run(fam, func(b *testing.B) {
			for _, nroutes := range benchRouteCount {
				routes := rng(nroutes)
				b.Run(fmt.Sprint(nroutes), func(b *testing.B) {
					fn(b, routes)
				})
			}
		})
	}
}

func BenchmarkTableInsertion(b *testing.B) {
	forFamilyAndCount(b, func(b *testing.B, routes []slowPrefixEntry[int]) {
		b.StopTimer()
		b.ResetTimer()
		var startMem, endMem runtime.MemStats
		runtime.ReadMemStats(&startMem)
		b.StartTimer()
		for i := 0; i < b.N; i++ {
			var rt Table[int]
			for _, route := range routes {
				rt.Insert(route.pfx, route.val)
			}
		}
		b.StopTimer()
		runtime.ReadMemStats(&endMem)
		inserts := float64(b.N) * float64(len(routes))
		allocs := float64(endMem.Mallocs - startMem.Mallocs)
		bytes := float64(endMem.TotalAlloc - startMem.TotalAlloc)
		elapsed := float64(b.Elapsed().Nanoseconds())
		elapsedSec := b.Elapsed().Seconds()
		b.ReportMetric(elapsed/inserts, "ns/op")
		b.ReportMetric(inserts/elapsedSec, "routes/s")
		b.ReportMetric(roundFloat64(allocs/inserts), "avg-allocs/op")
		b.ReportMetric(roundFloat64(bytes/inserts), "avg-B/op")
	})
}

func BenchmarkTableDelete(b *testing.B) {
	forFamilyAndCount(b, func(b *testing.B, routes []slowPrefixEntry[int]) {
		// Collect memstats for one round of insertions, so we can remove it
		// from the total at the end and get only the deletion alloc count.
		insertAllocs, insertBytes := getMemCost(func() {
			var rt Table[int]
			for _, route := range routes {
				rt.Insert(route.pfx, route.val)
			}
		})
		insertAllocs *= float64(b.N)
		insertBytes *= float64(b.N)

		var t runningTimer
		allocs, bytes := getMemCost(func() {
			for i := 0; i < b.N; i++ {
				var rt Table[int]
				for _, route := range routes {
					rt.Insert(route.pfx, route.val)
				}
				t.Start()
				for _, route := range routes {
					rt.Delete(route.pfx)
				}
				t.Stop()
			}
		})
		inserts := float64(b.N) * float64(len(routes))
		allocs -= insertAllocs
		bytes -= insertBytes
		elapsed := float64(t.Elapsed().Nanoseconds())
		elapsedSec := t.Elapsed().Seconds()
		b.ReportMetric(elapsed/inserts, "ns/op")
		b.ReportMetric(inserts/elapsedSec, "routes/s")
		b.ReportMetric(roundFloat64(allocs/inserts), "avg-allocs/op")
		b.ReportMetric(roundFloat64(bytes/inserts), "avg-B/op")
	})
}

func BenchmarkTableGet(b *testing.B) {
	forFamilyAndCount(b, func(b *testing.B, routes []slowPrefixEntry[int]) {
		genAddr := randomAddr4
		if routes[0].pfx.Addr().Is6() {
			genAddr = randomAddr6
		}
		var rt Table[int]
		for _, route := range routes {
			rt.Insert(route.pfx, route.val)
		}
		addrAllocs, addrBytes := getMemCost(func() {
			// Have to run genAddr more than once, otherwise the reported
			// cost is 16 bytes - presumably due to some amortized costs in
			// the memory allocator? Either way, empirically 100 iterations
			// reliably reports the correct cost.
			for i := 0; i < 100; i++ {
				_ = genAddr()
			}
		})
		addrAllocs /= 100
		addrBytes /= 100
		var t runningTimer
		allocs, bytes := getMemCost(func() {
			for i := 0; i < b.N; i++ {
				addr := genAddr()
				t.Start()
				writeSink, _ = rt.Get(addr)
				t.Stop()
			}
		})
		b.ReportAllocs() // Enables the output, but we report manually below
		allocs -= (addrAllocs * float64(b.N))
		bytes -= (addrBytes * float64(b.N))
		lookups := float64(b.N)
		elapsed := float64(t.Elapsed().Nanoseconds())
		elapsedSec := float64(t.Elapsed().Seconds())
		b.ReportMetric(elapsed/lookups, "ns/op")
		b.ReportMetric(lookups/elapsedSec, "addrs/s")
		b.ReportMetric(allocs/lookups, "allocs/op")
		b.ReportMetric(bytes/lookups, "B/op")
	})
}

func BenchmarkTableLookup(b *testing.B) {
	forFamilyAndCount(b, func(b *testing.B, routes []slowPrefixEntry[int]) {
		genAddr := randomAddr4
		if routes[0].pfx.Addr().Is6() {
			genAddr = randomAddr6
		}
		var rt Table[int]
		for _, route := range routes {
			rt.Insert(route.pfx, route.val)
		}
		addrAllocs, addrBytes := getMemCost(func() {
			// Have to run genAddr more than once, otherwise the reported
			// cost is 16 bytes - presumably due to some amortized costs in
			// the memory allocator? Either way, empirically 100 iterations
			// reliably reports the correct cost.
			for i := 0; i < 100; i++ {
				_ = genAddr()
			}
		})
		addrAllocs /= 100
		addrBytes /= 100
		var t runningTimer
		allocs, bytes := getMemCost(func() {
			for i := 0; i < b.N; i++ {
				addr := genAddr()
				t.Start()
				_, writeSink, _ = rt.Lookup(addr)
				t.Stop()
			}
		})
		b.ReportAllocs() // Enables the output, but we report manually below
		allocs -= (addrAllocs * float64(b.N))
		bytes -= (addrBytes * float64(b.N))
		lookups := float64(b.N)
		elapsed := float64(t.Elapsed().Nanoseconds())
		elapsedSec := float64(t.Elapsed().Seconds())
		b.ReportMetric(elapsed/lookups, "ns/op")
		b.ReportMetric(lookups/elapsedSec, "addrs/s")
		b.ReportMetric(allocs/lookups, "allocs/op")
		b.ReportMetric(bytes/lookups, "B/op")
	})
}

func BenchmarkTableLookupSPM(b *testing.B) {
	forFamilyAndCount(b, func(b *testing.B, routes []slowPrefixEntry[int]) {
		genAddr := randomAddr4
		if routes[0].pfx.Addr().Is6() {
			genAddr = randomAddr6
		}
		var rt Table[int]
		for _, route := range routes {
			rt.Insert(route.pfx, route.val)
		}
		addrAllocs, addrBytes := getMemCost(func() {
			// Have to run genAddr more than once, otherwise the reported
			// cost is 16 bytes - presumably due to some amortized costs in
			// the memory allocator? Either way, empirically 100 iterations
			// reliably reports the correct cost.
			for i := 0; i < 100; i++ {
				_ = genAddr()
			}
		})
		addrAllocs /= 100
		addrBytes /= 100
		var t runningTimer
		allocs, bytes := getMemCost(func() {
			for i := 0; i < b.N; i++ {
				addr := genAddr()
				t.Start()
				_, writeSink, _ = rt.LookupShortest(addr)
				t.Stop()
			}
		})
		b.ReportAllocs() // Enables the output, but we report manually below
		allocs -= (addrAllocs * float64(b.N))
		bytes -= (addrBytes * float64(b.N))
		lookups := float64(b.N)
		elapsed := float64(t.Elapsed().Nanoseconds())
		elapsedSec := float64(t.Elapsed().Seconds())
		b.ReportMetric(elapsed/lookups, "ns/op")
		b.ReportMetric(lookups/elapsedSec, "addrs/s")
		b.ReportMetric(allocs/lookups, "allocs/op")
		b.ReportMetric(bytes/lookups, "B/op")
	})
}

var boolSink bool

func BenchmarkTablePrefixOverlaps(b *testing.B) {
	forFamilyAndCount(b, func(b *testing.B, routes []slowPrefixEntry[int]) {
		var rt Table[int]
		for _, route := range routes {
			rt.Insert(route.pfx, route.val)
		}

		genPfxs := randomPrefixes4
		if routes[0].pfx.Addr().Is6() {
			genPfxs = randomPrefixes6
		}
		const count = 10_000
		pfxs := genPfxs(count)
		b.ResetTimer()
		allocs, bytes := getMemCost(func() {
			for i := 0; i < b.N; i++ {
				boolSink = rt.OverlapsPrefix(pfxs[i%count].pfx)
			}
		})
		b.StopTimer()

		b.ReportAllocs() // Enables the output, but we report manually below
		lookups := float64(b.N)
		elapsed := float64(b.Elapsed().Nanoseconds())
		elapsedSec := float64(b.Elapsed().Seconds())
		b.ReportMetric(elapsed/lookups, "ns/op")
		b.ReportMetric(lookups/elapsedSec, "addrs/s")
		b.ReportMetric(allocs/lookups, "allocs/op")
		b.ReportMetric(bytes/lookups, "B/op")
	})
}

func BenchmarkTableOverlaps(b *testing.B) {
	forFamilyAndCount(b, func(b *testing.B, routes []slowPrefixEntry[int]) {
		var rt Table[int]
		for _, route := range routes {
			rt.Insert(route.pfx, route.val)
		}

		genPfxs := randomPrefixes4
		if routes[0].pfx.Addr().Is6() {
			genPfxs = randomPrefixes6
		}

		const (
			intersectSize = 10
			numIntersects = 1_000
		)

		intersects := make([]*Table[int], numIntersects)
		for i := range intersects {
			inter := &Table[int]{}
			for _, route := range genPfxs(intersectSize) {
				inter.Insert(route.pfx, route.val)
			}
			intersects[i] = inter
		}

		var t runningTimer
		allocs, bytes := getMemCost(func() {
			for i := 0; i < b.N; i++ {
				t.Start()
				boolSink = rt.Overlaps(intersects[i%numIntersects])
				t.Stop()
			}
		})

		b.ReportAllocs() // Enables the output, but we report manually below
		lookups := float64(b.N)
		elapsed := float64(t.Elapsed().Nanoseconds())
		elapsedSec := t.Elapsed().Seconds()
		b.ReportMetric(elapsed/lookups, "ns/op")
		b.ReportMetric(lookups/elapsedSec, "tables/s")
		b.ReportMetric(allocs/lookups, "allocs/op")
		b.ReportMetric(bytes/lookups, "B/op")
	})
}

// getMemCost runs fn 100 times and returns the number of allocations and bytes
// allocated by each call to fn.
//
// Note that if your fn allocates very little memory (less than ~16 bytes), you
// should make fn run its workload ~100 times and divide the results of
// getMemCost yourself. Otherwise, the byte count you get will be rounded up due
// to the memory allocator's bucketing granularity.
func getMemCost(fn func()) (allocs, bytes float64) {
	var start, end runtime.MemStats
	runtime.ReadMemStats(&start)
	fn()
	runtime.ReadMemStats(&end)
	return float64(end.Mallocs - start.Mallocs), float64(end.TotalAlloc - start.TotalAlloc)
}

// runningTimer is a timer that keeps track of the cumulative time it's spent
// running since creation. A newly created runningTimer is stopped.
//
// This timer exists because some of our benchmarks have to interleave costly
// ancillary logic in each benchmark iteration, rather than being able to
// front-load all the work before a single b.ResetTimer().
//
// As it turns out, b.StartTimer() and b.StopTimer() are expensive function
// calls, because they do costly memory allocation accounting on every call.
// Starting and stopping the benchmark timer in every b.N loop iteration slows
// the benchmarks down by orders of magnitude.
//
// So, rather than rely on testing.B's timing facility, we use this very
// lightweight timer combined with getMemCost to do our own accounting more
// efficiently.
type runningTimer struct {
	cumulative time.Duration
	start      time.Time
}

func (t *runningTimer) Start() {
	t.Stop()
	t.start = time.Now()
}

func (t *runningTimer) Stop() {
	if t.start.IsZero() {
		return
	}
	t.cumulative += time.Since(t.start)
	t.start = time.Time{}
}

func (t *runningTimer) Elapsed() time.Duration {
	return t.cumulative
}

func checkSize(t *testing.T, tbl *Table[int], want int) {
	tbl.init()
	t.Helper()
	if got := tbl.numNodes(); got != want {
		t.Errorf("wrong table size, got %d strides want %d", got, want)
	}
}

func (t *Table[V]) numNodes() int {
	seen := map[*node[V]]bool{}
	return t.numNodesRec(seen, t.rootV4) + t.numNodesRec(seen, t.rootV6)
}

func (t *Table[V]) numNodesRec(seen map[*node[V]]bool, n *node[V]) int {
	ret := 1
	if len(n.children.nodes) == 0 {
		return ret
	}
	for _, c := range n.children.nodes {
		if seen[c] {
			continue
		}
		seen[c] = true
		ret += t.numNodesRec(seen, c)
	}
	return ret
}

// slowPrefixTable is a routing table implemented as a set of prefixes that are
// explicitly scanned in full for every route lookup. It is very slow, but also
// reasonably easy to verify by inspection, and so a good correctness reference
// for Table.
type slowPrefixTable[V any] struct {
	prefixes []slowPrefixEntry[V]
}

type slowPrefixEntry[V any] struct {
	pfx netip.Prefix
	val V
}

func (st *slowPrefixTable[T]) insert(pfx netip.Prefix, val T) {
	pfx = pfx.Masked()
	for i, ent := range st.prefixes {
		if ent.pfx == pfx {
			st.prefixes[i].val = val
			return
		}
	}
	st.prefixes = append(st.prefixes, slowPrefixEntry[T]{pfx, val})
}

func (st *slowPrefixTable[V]) get(addr netip.Addr) (val V, ok bool) {
	_, val, ok = st.lpm(addr)
	return
}

func (st *slowPrefixTable[V]) lpm(addr netip.Addr) (lpm netip.Prefix, val V, ok bool) {
	bestLen := -1

	for _, item := range st.prefixes {
		if item.pfx.Contains(addr) && item.pfx.Bits() > bestLen {
			lpm = item.pfx
			val = item.val
			bestLen = item.pfx.Bits()
		}
	}
	return lpm, val, bestLen != -1
}

func (st *slowPrefixTable[V]) spm(addr netip.Addr) (spm netip.Prefix, val V, ok bool) {
	bestLen := 129

	for _, item := range st.prefixes {
		if item.pfx.Contains(addr) && item.pfx.Bits() < bestLen {
			spm = item.pfx
			val = item.val
			bestLen = item.pfx.Bits()
		}
	}
	return spm, val, bestLen != 129
}

func (st *slowPrefixTable[T]) overlapsPrefix(pfx netip.Prefix) bool {
	for _, p := range st.prefixes {
		if p.pfx.Overlaps(pfx) {
			return true
		}
	}
	return false
}

func (st *slowPrefixTable[T]) overlaps(so *slowPrefixTable[T]) bool {
	for _, tp := range st.prefixes {
		for _, op := range so.prefixes {
			if tp.pfx.Overlaps(op.pfx) {
				return true
			}
		}
	}
	return false
}

// randomPrefixes returns n randomly generated prefixes and associated values,
// distributed equally between IPv4 and IPv6.
func randomPrefixes(n int) []slowPrefixEntry[int] {
	pfxs := randomPrefixes4(n / 2)
	pfxs = append(pfxs, randomPrefixes6(n-len(pfxs))...)
	return pfxs
}

// randomPrefixes4 returns n randomly generated IPv4 prefixes and associated values.
func randomPrefixes4(n int) []slowPrefixEntry[int] {
	pfxs := map[netip.Prefix]bool{}

	for len(pfxs) < n {
		bits := rand.Intn(33)
		pfx, err := randomAddr4().Prefix(bits)
		if err != nil {
			panic(err)
		}
		pfxs[pfx] = true
	}

	ret := make([]slowPrefixEntry[int], 0, len(pfxs))
	for pfx := range pfxs {
		ret = append(ret, slowPrefixEntry[int]{pfx, rand.Int()})
	}

	return ret
}

// randomPrefixes6 returns n randomly generated IPv4 prefixes and associated values.
func randomPrefixes6(n int) []slowPrefixEntry[int] {
	pfxs := map[netip.Prefix]bool{}

	for len(pfxs) < n {
		bits := rand.Intn(129)
		pfx, err := randomAddr6().Prefix(bits)
		if err != nil {
			panic(err)
		}
		pfxs[pfx] = true
	}

	ret := make([]slowPrefixEntry[int], 0, len(pfxs))
	for pfx := range pfxs {
		ret = append(ret, slowPrefixEntry[int]{pfx, rand.Int()})
	}

	return ret
}

// randomAddr returns a randomly generated IP address.
func randomAddr() netip.Addr {
	if rand.Intn(2) == 1 {
		return randomAddr6()
	}
	return randomAddr4()
}

// randomAddr4 returns a randomly generated IPv4 address.
func randomAddr4() netip.Addr {
	var b [4]byte
	if _, err := crand.Read(b[:]); err != nil {
		panic(err)
	}
	return netip.AddrFrom4(b)
}

// randomAddr6 returns a randomly generated IPv6 address.
func randomAddr6() netip.Addr {
	var b [16]byte
	if _, err := crand.Read(b[:]); err != nil {
		panic(err)
	}
	return netip.AddrFrom16(b)
}

// roundFloat64 rounds f to 2 decimal places, for display.
//
// It round-trips through a float->string->float conversion, so should not be
// used in a performance critical setting.
func roundFloat64(f float64) float64 {
	s := fmt.Sprintf("%.2f", f)
	ret, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic(err)
	}
	return ret
}
