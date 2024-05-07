// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause
//
// some tests copied from github.com/tailscale/art
// and modified for this implementation by:
//
// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"fmt"
	"math/rand"
	"net/netip"
	"reflect"
	"runtime"
	"testing"
)

func TestRegression2(t *testing.T) {
	t.Parallel()
	// original comment by tailscale for ART,
	//
	// These tests are specific triggers for subtle correctness issues
	// that came up during initial implementation. Even if they seem
	// arbitrary, please do not clean them up. They are checking edge
	// cases that are very easy to get wrong, and quite difficult for
	// the other statistical tests to trigger promptly.
	//
	// ... but the BART implementation is different and has other edge cases.

	t.Run("prefixes_aligned_on_stride_boundary", func(t *testing.T) {
		fast := &Table2[int]{}
		slow := slowRT[int]{}

		fast.Insert(mpp("226.205.197.0/24"), 1)
		slow.insert(mpp("226.205.197.0/24"), 1)

		fast.Insert(mpp("226.205.0.0/16"), 2)
		slow.insert(mpp("226.205.0.0/16"), 2)

		probe := mpa("226.205.121.152")
		got, gotOK := fast.Lookup(probe)
		want, wantOK := slow.lookup(probe)
		if !getsEqual(got, gotOK, want, wantOK) {
			t.Fatalf("got (%v, %v), want (%v, %v)", got, gotOK, want, wantOK)
		}
	})

	t.Run("parent_prefix_inserted_in_different_orders", func(t *testing.T) {
		t1, t2 := &Table2[int]{}, &Table2[int]{}

		t1.Insert(mpp("136.20.0.0/16"), 1)
		t1.Insert(mpp("136.20.201.62/32"), 2)

		t2.Insert(mpp("136.20.201.62/32"), 2)
		t2.Insert(mpp("136.20.0.0/16"), 1)

		a := mpa("136.20.54.139")
		got1, ok1 := t1.Lookup(a)
		got2, ok2 := t2.Lookup(a)
		if !getsEqual(got1, ok1, got2, ok2) {
			t.Errorf("Lookup(%q) is insertion order dependent: t1=(%v, %v), t2=(%v, %v)", a, got1, ok1, got2, ok2)
		}
	})
}

func TestInsert2(t *testing.T) {
	tbl := &Table2[int]{}

	// Create a new leaf strideTable, with compressed path
	tbl.Insert(mpp("192.168.0.1/32"), 1)
	checkRoutes2(t, tbl, []tableTest{
		{"192.168.0.1", 1},
		{"192.168.0.2", -1},
		{"192.168.0.3", -1},
		{"192.168.0.255", -1},
		{"192.168.1.1", -1},
		{"192.170.1.1", -1},
		{"192.180.0.1", -1},
		{"192.180.3.5", -1},
		{"10.0.0.5", -1},
		{"10.0.0.15", -1},
	})

	// Insert into previous leaf, no tree changes
	tbl.Insert(mpp("192.168.0.2/32"), 2)
	checkRoutes2(t, tbl, []tableTest{
		{"192.168.0.1", 1},
		{"192.168.0.2", 2},
		{"192.168.0.3", -1},
		{"192.168.0.255", -1},
		{"192.168.1.1", -1},
		{"192.170.1.1", -1},
		{"192.180.0.1", -1},
		{"192.180.3.5", -1},
		{"10.0.0.5", -1},
		{"10.0.0.15", -1},
	})

	// Insert into previous leaf, unaligned prefix covering the /32s
	tbl.Insert(mpp("192.168.0.0/26"), 7)
	checkRoutes2(t, tbl, []tableTest{
		{"192.168.0.1", 1},
		{"192.168.0.2", 2},
		{"192.168.0.3", 7},
		{"192.168.0.255", -1},
		{"192.168.1.1", -1},
		{"192.170.1.1", -1},
		{"192.180.0.1", -1},
		{"192.180.3.5", -1},
		{"10.0.0.5", -1},
		{"10.0.0.15", -1},
	})

	// Create a different leaf elsewhere
	tbl.Insert(mpp("10.0.0.0/27"), 3)
	checkRoutes2(t, tbl, []tableTest{
		{"192.168.0.1", 1},
		{"192.168.0.2", 2},
		{"192.168.0.3", 7},
		{"192.168.0.255", -1},
		{"192.168.1.1", -1},
		{"192.170.1.1", -1},
		{"192.180.0.1", -1},
		{"192.180.3.5", -1},
		{"10.0.0.5", 3},
		{"10.0.0.15", 3},
	})

	// Insert that creates a new intermediate table and a new child
	tbl.Insert(mpp("192.168.1.1/32"), 4)
	checkRoutes2(t, tbl, []tableTest{
		{"192.168.0.1", 1},
		{"192.168.0.2", 2},
		{"192.168.0.3", 7},
		{"192.168.0.255", -1},
		{"192.168.1.1", 4},
		{"192.170.1.1", -1},
		{"192.180.0.1", -1},
		{"192.180.3.5", -1},
		{"10.0.0.5", 3},
		{"10.0.0.15", 3},
	})

	// Insert that creates a new intermediate table but no new child
	tbl.Insert(mpp("192.170.0.0/16"), 5)
	checkRoutes2(t, tbl, []tableTest{
		{"192.168.0.1", 1},
		{"192.168.0.2", 2},
		{"192.168.0.3", 7},
		{"192.168.0.255", -1},
		{"192.168.1.1", 4},
		{"192.170.1.1", 5},
		{"192.180.0.1", -1},
		{"192.180.3.5", -1},
		{"10.0.0.5", 3},
		{"10.0.0.15", 3},
	})

	// New leaf in a different subtree, so the next insert can test a
	// variant of decompression.
	tbl.Insert(mpp("192.180.0.1/32"), 8)
	checkRoutes2(t, tbl, []tableTest{
		{"192.168.0.1", 1},
		{"192.168.0.2", 2},
		{"192.168.0.3", 7},
		{"192.168.0.255", -1},
		{"192.168.1.1", 4},
		{"192.170.1.1", 5},
		{"192.180.0.1", 8},
		{"192.180.3.5", -1},
		{"10.0.0.5", 3},
		{"10.0.0.15", 3},
	})

	// Insert that creates a new intermediate table but no new child,
	// with an unaligned intermediate
	tbl.Insert(mpp("192.180.0.0/21"), 9)
	checkRoutes2(t, tbl, []tableTest{
		{"192.168.0.1", 1},
		{"192.168.0.2", 2},
		{"192.168.0.3", 7},
		{"192.168.0.255", -1},
		{"192.168.1.1", 4},
		{"192.170.1.1", 5},
		{"192.180.0.1", 8},
		{"192.180.3.5", 9},
		{"10.0.0.5", 3},
		{"10.0.0.15", 3},
	})

	// Insert a default route, those have their own codepath.
	tbl.Insert(mpp("0.0.0.0/0"), 6)
	checkRoutes2(t, tbl, []tableTest{
		{"192.168.0.1", 1},
		{"192.168.0.2", 2},
		{"192.168.0.3", 7},
		{"192.168.0.255", 6},
		{"192.168.1.1", 4},
		{"192.170.1.1", 5},
		{"192.180.0.1", 8},
		{"192.180.3.5", 9},
		{"10.0.0.5", 3},
		{"10.0.0.15", 3},
	})

	// Now all of the above again, but for IPv6.

	// Create a new leaf strideTable, with compressed path
	tbl.Insert(mpp("ff:aaaa::1/128"), 1)
	checkRoutes2(t, tbl, []tableTest{
		{"ff:aaaa::1", 1},
		{"ff:aaaa::2", -1},
		{"ff:aaaa::3", -1},
		{"ff:aaaa::255", -1},
		{"ff:aaaa:aaaa::1", -1},
		{"ff:aaaa:aaaa:bbbb::1", -1},
		{"ff:cccc::1", -1},
		{"ff:cccc::ff", -1},
		{"ffff:bbbb::5", -1},
		{"ffff:bbbb::15", -1},
	})

	// Insert into previous leaf, no tree changes
	tbl.Insert(mpp("ff:aaaa::2/128"), 2)
	checkRoutes2(t, tbl, []tableTest{
		{"ff:aaaa::1", 1},
		{"ff:aaaa::2", 2},
		{"ff:aaaa::3", -1},
		{"ff:aaaa::255", -1},
		{"ff:aaaa:aaaa::1", -1},
		{"ff:aaaa:aaaa:bbbb::1", -1},
		{"ff:cccc::1", -1},
		{"ff:cccc::ff", -1},
		{"ffff:bbbb::5", -1},
		{"ffff:bbbb::15", -1},
	})

	// Insert into previous leaf, unaligned prefix covering the /128s
	tbl.Insert(mpp("ff:aaaa::/125"), 7)
	checkRoutes2(t, tbl, []tableTest{
		{"ff:aaaa::1", 1},
		{"ff:aaaa::2", 2},
		{"ff:aaaa::3", 7},
		{"ff:aaaa::255", -1},
		{"ff:aaaa:aaaa::1", -1},
		{"ff:aaaa:aaaa:bbbb::1", -1},
		{"ff:cccc::1", -1},
		{"ff:cccc::ff", -1},
		{"ffff:bbbb::5", -1},
		{"ffff:bbbb::15", -1},
	})

	// Create a different leaf elsewhere
	tbl.Insert(mpp("ffff:bbbb::/120"), 3)
	checkRoutes2(t, tbl, []tableTest{
		{"ff:aaaa::1", 1},
		{"ff:aaaa::2", 2},
		{"ff:aaaa::3", 7},
		{"ff:aaaa::255", -1},
		{"ff:aaaa:aaaa::1", -1},
		{"ff:aaaa:aaaa:bbbb::1", -1},
		{"ff:cccc::1", -1},
		{"ff:cccc::ff", -1},
		{"ffff:bbbb::5", 3},
		{"ffff:bbbb::15", 3},
	})

	// Insert that creates a new intermediate table and a new child
	tbl.Insert(mpp("ff:aaaa:aaaa::1/128"), 4)
	checkRoutes2(t, tbl, []tableTest{
		{"ff:aaaa::1", 1},
		{"ff:aaaa::2", 2},
		{"ff:aaaa::3", 7},
		{"ff:aaaa::255", -1},
		{"ff:aaaa:aaaa::1", 4},
		{"ff:aaaa:aaaa:bbbb::1", -1},
		{"ff:cccc::1", -1},
		{"ff:cccc::ff", -1},
		{"ffff:bbbb::5", 3},
		{"ffff:bbbb::15", 3},
	})

	// Insert that creates a new intermediate table but no new child
	tbl.Insert(mpp("ff:aaaa:aaaa:bb00::/56"), 5)
	checkRoutes2(t, tbl, []tableTest{
		{"ff:aaaa::1", 1},
		{"ff:aaaa::2", 2},
		{"ff:aaaa::3", 7},
		{"ff:aaaa::255", -1},
		{"ff:aaaa:aaaa::1", 4},
		{"ff:aaaa:aaaa:bbbb::1", 5},
		{"ff:cccc::1", -1},
		{"ff:cccc::ff", -1},
		{"ffff:bbbb::5", 3},
		{"ffff:bbbb::15", 3},
	})

	// New leaf in a different subtree, so the next insert can test a
	// variant of decompression.
	tbl.Insert(mpp("ff:cccc::1/128"), 8)
	checkRoutes2(t, tbl, []tableTest{
		{"ff:aaaa::1", 1},
		{"ff:aaaa::2", 2},
		{"ff:aaaa::3", 7},
		{"ff:aaaa::255", -1},
		{"ff:aaaa:aaaa::1", 4},
		{"ff:aaaa:aaaa:bbbb::1", 5},
		{"ff:cccc::1", 8},
		{"ff:cccc::ff", -1},
		{"ffff:bbbb::5", 3},
		{"ffff:bbbb::15", 3},
	})

	// Insert that creates a new intermediate table but no new child,
	// with an unaligned intermediate
	tbl.Insert(mpp("ff:cccc::/37"), 9)
	checkRoutes2(t, tbl, []tableTest{
		{"ff:aaaa::1", 1},
		{"ff:aaaa::2", 2},
		{"ff:aaaa::3", 7},
		{"ff:aaaa::255", -1},
		{"ff:aaaa:aaaa::1", 4},
		{"ff:aaaa:aaaa:bbbb::1", 5},
		{"ff:cccc::1", 8},
		{"ff:cccc::ff", 9},
		{"ffff:bbbb::5", 3},
		{"ffff:bbbb::15", 3},
	})

	// Insert a default route, those have their own codepath.
	tbl.Insert(mpp("::/0"), 6)
	checkRoutes2(t, tbl, []tableTest{
		{"ff:aaaa::1", 1},
		{"ff:aaaa::2", 2},
		{"ff:aaaa::3", 7},
		{"ff:aaaa::255", 6},
		{"ff:aaaa:aaaa::1", 4},
		{"ff:aaaa:aaaa:bbbb::1", 5},
		{"ff:cccc::1", 8},
		{"ff:cccc::ff", 9},
		{"ffff:bbbb::5", 3},
		{"ffff:bbbb::15", 3},
	})
}

func TestDelete2(t *testing.T) {
	t.Parallel()

	t.Run("prefix_in_root", func(t *testing.T) {
		// Add/remove prefix from root table.
		rtbl := &Table2[int]{}
		checkSize2(t, rtbl, 2)

		rtbl.Insert(mpp("10.0.0.0/8"), 1)
		checkRoutes2(t, rtbl, []tableTest{
			{"10.0.0.1", 1},
			{"255.255.255.255", -1},
		})
		checkSize2(t, rtbl, 2)

		rtbl.Delete(mpp("10.0.0.0/8"))
		checkRoutes2(t, rtbl, []tableTest{
			{"10.0.0.1", -1},
			{"255.255.255.255", -1},
		})
		checkSize2(t, rtbl, 2)
	})

	t.Run("prefix_in_leaf", func(t *testing.T) {
		// Create, then delete a single leaf table.
		rtbl := &Table2[int]{}
		checkSize2(t, rtbl, 2)

		rtbl.Insert(mpp("192.168.0.1/32"), 1)
		checkRoutes2(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"255.255.255.255", -1},
		})

		rtbl.Delete(mpp("192.168.0.1/32"))
		checkRoutes2(t, rtbl, []tableTest{
			{"192.168.0.1", -1},
			{"255.255.255.255", -1},
		})
		checkSize2(t, rtbl, 2)
	})

	t.Run("intermediate_no_routes", func(t *testing.T) {
		// Create an intermediate with 2 children, then delete one leaf.
		rtbl := &Table2[int]{}

		rtbl.Insert(mpp("192.168.0.1/32"), 1)
		rtbl.Insert(mpp("192.180.0.1/32"), 2)

		checkRoutes2(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", 2},
			{"192.40.0.1", -1},
		})
		checkSize2(t, rtbl, 5)

		rtbl.Delete(mpp("192.180.0.1/32"))

		checkRoutes2(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", -1},
			{"192.40.0.1", -1},
		})
		checkSize2(t, rtbl, 3)
	})

	t.Run("intermediate_with_route", func(t *testing.T) {
		// Same, but the intermediate carries a route as well.
		rtbl := &Table2[int]{}
		rtbl.Insert(mpp("192.168.0.1/32"), 1)
		rtbl.Insert(mpp("192.180.0.1/32"), 2)
		rtbl.Insert(mpp("192.0.0.0/10"), 3)

		checkRoutes2(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", 2},
			{"192.40.0.1", 3},
			{"192.255.0.1", -1},
		})
		checkSize2(t, rtbl, 5)

		rtbl.Delete(mpp("192.180.0.1/32"))

		checkRoutes2(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", -1},
			{"192.40.0.1", 3},
			{"192.255.0.1", -1},
		})
		checkSize2(t, rtbl, 4) // 2 roots, 1 intermediate, 1 leaf
	})

	t.Run("intermediate_many_leaves", func(t *testing.T) {
		// Intermediate with 3 leaves, then delete one leaf.
		rtbl := &Table2[int]{}
		rtbl.Insert(mpp("192.168.0.1/32"), 1)
		rtbl.Insert(mpp("192.180.0.1/32"), 2)
		rtbl.Insert(mpp("192.200.0.1/32"), 3)

		checkRoutes2(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", 2},
			{"192.200.0.1", 3},
			{"192.255.0.1", -1},
		})
		checkSize2(t, rtbl, 6)

		rtbl.Delete(mpp("192.180.0.1/32"))

		checkRoutes2(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", -1},
			{"192.200.0.1", 3},
			{"192.255.0.1", -1},
		})
		checkSize2(t, rtbl, 5)
	})

	t.Run("nosuchprefix_missing_child", func(t *testing.T) {
		// Delete non-existent prefix, missing strideTable path.
		rtbl := &Table2[int]{}
		rtbl.Insert(mpp("192.168.0.1/32"), 1)

		checkRoutes2(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.255.0.1", -1},
		})
		checkSize2(t, rtbl, 3)

		rtbl.Delete(mpp("200.0.0.0/32")) // lookup miss in root

		checkRoutes2(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.255.0.1", -1},
		})
		checkSize2(t, rtbl, 3)
	})

	t.Run("nosuchprefix_not_in_leaf", func(t *testing.T) {
		// Delete non-existent prefix, strideTable path exists but
		// leaf doesn't contain route.
		rtbl := &Table2[int]{}
		rtbl.Insert(mpp("192.168.0.1/32"), 1)

		checkRoutes2(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.255.0.1", -1},
		})
		checkSize2(t, rtbl, 3)

		rtbl.Delete(mpp("192.168.0.5/32")) // right leaf, no route

		checkRoutes2(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.255.0.1", -1},
		})
		checkSize2(t, rtbl, 3)
	})

	t.Run("intermediate_with_deleted_route", func(t *testing.T) {
		// Intermediate table loses its last route and becomes
		// compactable.
		rtbl := &Table2[int]{}
		rtbl.Insert(mpp("192.168.0.1/32"), 1)
		rtbl.Insert(mpp("192.168.0.0/22"), 2)

		checkRoutes2(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.168.0.2", 2},
			{"192.255.0.1", -1},
		})
		checkSize2(t, rtbl, 4)

		rtbl.Delete(mpp("192.168.0.0/22"))

		checkRoutes2(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.168.0.2", -1},
			{"192.255.0.1", -1},
		})
		checkSize2(t, rtbl, 3)
	})

	t.Run("default_route", func(t *testing.T) {
		// Default routes have a special case in the code.
		rtbl := &Table2[int]{}

		rtbl.Insert(mpp("0.0.0.0/0"), 1)
		rtbl.Insert(mpp("::/0"), 1)

		rtbl.Delete(mpp("0.0.0.0/0"))

		checkRoutes2(t, rtbl, []tableTest{
			{"1.2.3.4", -1},
			{"::1", 1},
		})
		checkSize2(t, rtbl, 2)
	})
}

func TestDeleteCompare2(t *testing.T) {
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

	pfxs := append([]slowRTEntry[int](nil), all4[:deleteCut]...)
	pfxs = append(pfxs, all6[:deleteCut]...)

	toDelete := append([]slowRTEntry[int](nil), all4[deleteCut:]...)
	toDelete = append(toDelete, all6[deleteCut:]...)

	slow := slowRT[int]{pfxs}
	fast := Table2[int]{}

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

		slowVal, slowOK := slow.lookup(a)
		fastVal, fastOK := fast.Lookup(a)

		if !getsEqual(slowVal, slowOK, fastVal, fastOK) {
			t.Fatalf("get(%q) = (%v, %v), want (%v, %v)", a, fastVal, fastOK, slowVal, slowOK)
		}

		if a.Is6() {
			seenVals6[fastVal] = true
		} else {
			seenVals4[fastVal] = true
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

func TestDeleteShuffled2(t *testing.T) {
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

	pfxs := append([]slowRTEntry[int](nil), all4[:deleteCut]...)
	pfxs = append(pfxs, all6[:deleteCut]...)

	toDelete := append([]slowRTEntry[int](nil), all4[deleteCut:]...)
	toDelete = append(toDelete, all6[deleteCut:]...)

	rt1 := Table2[int]{}
	for _, pfx := range pfxs {
		rt1.Insert(pfx.pfx, pfx.val)
	}
	for _, pfx := range toDelete {
		rt1.Insert(pfx.pfx, pfx.val)
	}
	for _, pfx := range toDelete {
		rt1.Delete(pfx.pfx)
	}

	for i := 0; i < 10; i++ {
		pfxs2 := append([]slowRTEntry[int](nil), pfxs...)
		toDelete2 := append([]slowRTEntry[int](nil), toDelete...)
		rand.Shuffle(len(toDelete2), func(i, j int) { toDelete2[i], toDelete2[j] = toDelete2[j], toDelete2[i] })
		rt2 := Table2[int]{}
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
			val1, ok1 := rt1.Lookup(a)
			val2, ok2 := rt2.Lookup(a)
			if !getsEqual(val1, ok1, val2, ok2) {
				t.Errorf("get(%q) = (%v, %v), want (%v, %v)", a, val2, ok2, val1, ok1)
			}
		}
	}
}

func TestDeleteIsReverseOfInsert2(t *testing.T) {
	t.Parallel()
	// Insert N prefixes, then delete those same prefixes in reverse
	// order. Each deletion should exactly undo the internal structure
	// changes that each insert did.
	const N = 100

	var tab Table2[int]
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

func TestGet2(t *testing.T) {
	t.Parallel()

	rt := new(Table2[int])
	t.Run("empty table", func(t *testing.T) {
		_, ok := rt.Get(randomPrefix4())

		if ok {
			t.Errorf("empty table: ok=%v, expected: %v", ok, false)
		}
	})

	tests := []struct {
		name string
		pfx  netip.Prefix
		val  int
	}{
		{
			name: "default route v4",
			pfx:  mpp("0.0.0.0/0"),
			val:  0,
		},
		{
			name: "default route v6",
			pfx:  mpp("::/0"),
			val:  0,
		},
		{
			name: "set v4",
			pfx:  mpp("1.2.3.4/32"),
			val:  1234,
		},
		{
			name: "set v6",
			pfx:  mpp("2001:db8::/32"),
			val:  2001,
		},
	}

	rt = new(Table2[int])

	for _, tt := range tests {
		rt.Insert(tt.pfx, tt.val)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := rt.Get(tt.pfx)

			if !ok {
				t.Errorf("%s: ok=%v, expected: %v", tt.name, ok, true)
			}

			if got != tt.val {
				t.Errorf("%s: val=%v, expected: %v", tt.name, got, tt.val)
			}
		})
	}
}

func TestGetCompare2(t *testing.T) {
	t.Parallel()

	pfxs := randomPrefixes(10_000)
	slow := slowRT[int]{pfxs}
	fast := Table2[int]{}

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	for _, pfx := range pfxs {
		slowVal, slowOK := slow.get(pfx.pfx)
		fastVal, fastOK := fast.Get(pfx.pfx)

		if !getsEqual(slowVal, slowOK, fastVal, fastOK) {
			t.Fatalf("get(%q) = (%v, %v), want (%v, %v)", pfx.pfx, fastVal, fastOK, slowVal, slowOK)
		}
	}
}

func TestUpdateCompare2(t *testing.T) {
	t.Parallel()

	pfxs := randomPrefixes(10_000)
	slow := slowRT[int]{pfxs}
	fast := Table2[int]{}

	// Update as insert
	for _, pfx := range pfxs {
		fast.Update(pfx.pfx, func(int, bool) int { return pfx.val })
	}

	for _, pfx := range pfxs {
		slowVal, slowOK := slow.get(pfx.pfx)
		fastVal, fastOK := fast.Get(pfx.pfx)

		if !getsEqual(slowVal, slowOK, fastVal, fastOK) {
			t.Fatalf("get(%q) = (%v, %v), want (%v, %v)", pfx.pfx, fastVal, fastOK, slowVal, slowOK)
		}
	}

	cb := func(val int, _ bool) int { return val + 1 }

	// Update as update
	for _, pfx := range pfxs[:len(pfxs)/2] {
		slow.update(pfx.pfx, cb)
		fast.Update(pfx.pfx, cb)
	}

	for _, pfx := range pfxs {
		slowVal, slowOK := slow.get(pfx.pfx)
		fastVal, fastOK := fast.Get(pfx.pfx)

		if !getsEqual(slowVal, slowOK, fastVal, fastOK) {
			t.Fatalf("get(%q) = (%v, %v), want (%v, %v)", pfx.pfx, fastVal, fastOK, slowVal, slowOK)
		}
	}
}

func TestUpdate2(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		pfx  netip.Prefix
	}{
		{
			name: "default route v4",
			pfx:  mpp("0.0.0.0/0"),
		},
		{
			name: "default route v6",
			pfx:  mpp("::/0"),
		},
		{
			name: "set v4",
			pfx:  mpp("1.2.3.4/32"),
		},
		{
			name: "set v6",
			pfx:  mpp("2001:db8::/32"),
		},
	}

	rt := new(Table2[int])

	// just increment val
	cb := func(val int, ok bool) int {
		if ok {
			return val + 1
		}
		return 0
	}

	// update as insert
	for _, tt := range tests {
		t.Run(fmt.Sprintf("insert: %s", tt.name), func(t *testing.T) {
			val := rt.Update(tt.pfx, cb)
			got, ok := rt.Get(tt.pfx)

			if !ok {
				t.Errorf("%s: ok=%v, expected: %v", tt.name, ok, true)
			}

			if got != 0 || got != val {
				t.Errorf("%s: got=%v, expected: %v", tt.name, got, 0)
			}
		})
	}

	// update as update
	for _, tt := range tests {
		t.Run(fmt.Sprintf("update: %s", tt.name), func(t *testing.T) {
			val := rt.Update(tt.pfx, cb)
			got, ok := rt.Get(tt.pfx)

			if !ok {
				t.Errorf("%s: ok=%v, expected: %v", tt.name, ok, true)
			}

			if got != 1 || got != val {
				t.Errorf("%s: got=%v, expected: %v", tt.name, got, 1)
			}
		})
	}
}

func TestLookupCompare2(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()
	pfxs := randomPrefixes(10_000)

	slow := slowRT[int]{pfxs}
	fast := Table2[int]{}

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	seenVals4 := map[int]bool{}
	seenVals6 := map[int]bool{}

	for i := 0; i < 10_000; i++ {
		a := randomAddr()

		slowVal, slowOK := slow.lookup(a)
		fastVal, fastOK := fast.Lookup(a)

		if !getsEqual(slowVal, slowOK, fastVal, fastOK) {
			t.Fatalf("get(%q) = (%v, %v), want (%v, %v)", a, fastVal, fastOK, slowVal, slowOK)
		}

		if a.Is6() {
			seenVals6[fastVal] = true
		} else {
			seenVals4[fastVal] = true
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

func TestLookupPrefix2(t *testing.T) {
	t.Parallel()

	rt := new(Table2[int])

	t.Run("empty table", func(t *testing.T) {
		_, ok := rt.LookupPrefix(randomPrefix4())

		if ok {
			t.Errorf("empty table: ok=%v, expected: %v", ok, false)
		}
	})

	tests := []struct {
		name string
		pfx  netip.Prefix
		val  int
	}{
		{
			name: "default route v4",
			pfx:  mpp("0.0.0.0/0"),
			val:  0,
		},
		{
			name: "default route v6",
			pfx:  mpp("::/0"),
			val:  0,
		},
		{
			name: "set v4",
			pfx:  mpp("1.2.3.4/32"),
			val:  1234,
		},
		{
			name: "set v6",
			pfx:  mpp("2001:db8::/32"),
			val:  2001,
		},
	}

	rt = new(Table2[int])

	for _, tt := range tests {
		rt.Insert(tt.pfx, tt.val)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := rt.LookupPrefix(tt.pfx)

			if !ok {
				t.Errorf("%s: ok=%v, expected: %v", tt.name, ok, true)
			}

			if got != tt.val {
				t.Errorf("%s: val=%v, expected: %v", tt.name, got, tt.val)
			}
		})
	}
}

func TestLookupPrefixCompare2(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()
	pfxs := randomPrefixes(10_000)

	slow := slowRT[int]{pfxs}
	fast := Table2[int]{}

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	seenVals4 := map[int]bool{}
	seenVals6 := map[int]bool{}

	for i := 0; i < 10_000; i++ {
		pfx := randomPrefix()

		slowVal, slowOK := slow.lookupPfx(pfx)
		fastVal, fastOK := fast.LookupPrefix(pfx)

		if !getsEqual(slowVal, slowOK, fastVal, fastOK) {
			t.Fatalf("get(%q) = (%v, %v), want (%v, %v)", pfx, fastVal, fastOK, slowVal, slowOK)
		}

		if pfx.Addr().Is6() {
			seenVals6[fastVal] = true
		} else {
			seenVals4[fastVal] = true
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

func TestLookupPrefixLPMCompare2(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()
	pfxs := randomPrefixes(10_000)

	slow := slowRT[int]{pfxs}
	fast := Table2[int]{}

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	seenVals4 := map[int]bool{}
	seenVals6 := map[int]bool{}

	for i := 0; i < 10_000; i++ {
		pfx := randomPrefix()

		slowLPM, slowVal, slowOK := slow.lookupPfxLPM(pfx)
		fastLPM, fastVal, fastOK := fast.LookupPrefixLPM(pfx)

		if !getsEqual(slowVal, slowOK, fastVal, fastOK) {
			t.Fatalf("get(%q) = (%v, %v), want (%v, %v)", pfx, fastVal, fastOK, slowVal, slowOK)
		}

		if !getsEqual(slowLPM, slowOK, fastLPM, fastOK) {
			t.Fatalf("get(%q) = (%v, %v), want (%v, %v)", pfx, fastLPM, fastOK, slowLPM, slowOK)
		}

		if pfx.Addr().Is6() {
			seenVals6[fastVal] = true
		} else {
			seenVals4[fastVal] = true
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

func TestInsertShuffled2(t *testing.T) {
	// The order in which you insert prefixes into a route table
	// should not matter, as long as you're inserting the same set of
	// routes.
	t.Parallel()

	pfxs := randomPrefixes(1000)
	/* uncomment for failure debugging
	var pfxs2 []slowRTEntry[int]
	defer func() {
		if t.Failed() {
			t.Logf("pre-shuffle: %#v", pfxs)
			t.Logf("post-shuffle: %#v", pfxs2)
		}
	}()
	*/

	for i := 0; i < 10; i++ {
		pfxs2 := append([]slowRTEntry[int](nil), pfxs...)
		rand.Shuffle(len(pfxs2), func(i, j int) { pfxs2[i], pfxs2[j] = pfxs2[j], pfxs2[i] })

		addrs := make([]netip.Addr, 0, 10_000)
		for i := 0; i < 10_000; i++ {
			addrs = append(addrs, randomAddr())
		}

		rt1 := Table2[int]{}
		rt2 := Table2[int]{}

		for _, pfx := range pfxs {
			rt1.Insert(pfx.pfx, pfx.val)
		}
		for _, pfx := range pfxs2 {
			rt2.Insert(pfx.pfx, pfx.val)
		}

		for _, a := range addrs {
			val1, ok1 := rt1.Lookup(a)
			val2, ok2 := rt2.Lookup(a)

			if !getsEqual(val1, ok1, val2, ok2) {
				t.Fatalf("get(%q) = (%v, %v), want (%v, %v)", a, val2, ok2, val1, ok1)
			}
		}
	}
}

// After go version 1.22 we can use range iterators
func TestAll2(t *testing.T) {
	pfxs := randomPrefixes(10_000)
	seen := make(map[netip.Prefix]int, 10_000)

	t.Run("All", func(t *testing.T) {
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
			seen[item.pfx] = item.val
		}

		// check if pfx/val is as expected
		rtbl.All(func(pfx netip.Prefix, val int) bool {
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

	// make an interation and update the values in the callback
	t.Run("All and Update", func(t *testing.T) {
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
			seen[item.pfx] = item.val + 1
		}

		// update callback, add 1 to val
		updateValue := func(val int, ok bool) int {
			return val + 1
		}

		yield := func(pfx netip.Prefix, val int) bool {
			rtbl.Update(pfx, updateValue)
			return true
		}

		// iterate and update the values
		rtbl.All(yield)

		// test if all values got updated, yield now as closure
		rtbl.All(func(pfx netip.Prefix, val int) bool {
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
		rtbl.All(func(pfx netip.Prefix, val int) bool {
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

func TestTableClone2(t *testing.T) {
	t.Parallel()

	tbl := new(Table2[int])
	clone := tbl.Clone()
	if tbl.String() != clone.String() {
		t.Errorf("empty Clone: got:\n%swant:\n%s", clone.String(), tbl.String())
	}

	tbl.Insert(mpp("10.0.0.1/32"), 1)
	tbl.Insert(mpp("::1/128"), 1)

	clone = tbl.Clone()

	if tbl.String() != clone.String() {
		t.Errorf("Clone: got:\n%swant:\n%s", clone.String(), tbl.String())
	}

	// overwrite value
	tbl.Insert(mpp("::1/128"), 2)
	if tbl.String() == clone.String() {
		t.Errorf("overwrite, clone must be different: clone:\n%sorig:\n%s", clone.String(), tbl.String())
	}

	tbl.Delete(mpp("10.0.0.1/32"))
	if tbl.String() == clone.String() {
		t.Errorf("delete, clone must be different: clone:\n%sorig:\n%s", clone.String(), tbl.String())
	}
}

func TestSubnetsEdgeCases2(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		pfxs []netip.Prefix // input prefixes
		spfx netip.Prefix   // subnet search prefix
		want []netip.Prefix
	}{
		{
			name: "empty V4",
			pfxs: nil,
			spfx: mpp("0.0.0.0/0"),
			want: nil,
		},
		{
			name: "empty V6",
			pfxs: nil,
			spfx: mpp("::/0"),
			want: nil,
		},
		{
			name: "default V4",
			pfxs: []netip.Prefix{mpp("0.0.0.0/0")},
			spfx: mpp("0.0.0.0/0"),
			want: []netip.Prefix{mpp("0.0.0.0/0")},
		},
		{
			name: "default V6",
			pfxs: []netip.Prefix{mpp("::/0")},
			spfx: mpp("::/0"),
			want: []netip.Prefix{mpp("::/0")},
		},
		{
			name: "self V4",
			pfxs: []netip.Prefix{mpp("192.168.128.0/19")},
			spfx: mpp("192.168.128.0/19"),
			want: []netip.Prefix{mpp("192.168.128.0/19")},
		},
		{
			name: "self V6",
			pfxs: []netip.Prefix{mpp("2001:db8::dead:beef/128")},
			spfx: mpp("2001:db8::dead:beef/128"),
			want: []netip.Prefix{mpp("2001:db8::dead:beef/128")},
		},
		{
			name: "same leaf V4",
			pfxs: []netip.Prefix{mpp("10.0.0.1/32")},
			spfx: mpp("10.0.0.0/29"),
			want: []netip.Prefix{mpp("10.0.0.1/32")},
		},
		{
			name: "same leaf V6",
			pfxs: []netip.Prefix{mpp("::1/128")},
			spfx: mpp("::0/120"),
			want: []netip.Prefix{mpp("::1/128")},
		},
		{
			name: "intermediate V4",
			pfxs: []netip.Prefix{mpp("10.0.0.1/32"), mpp("10.0.1.1/32")},
			spfx: mpp("10.0.0.0/16"),
			want: []netip.Prefix{mpp("10.0.0.1/32"), mpp("10.0.1.1/32")},
		},
		{
			name: "intermediate V6",
			pfxs: []netip.Prefix{mpp("2001:db8::1/128"), mpp("2001:db8:dead:beaf::/64")},
			spfx: mpp("2001:db8::/32"),
			want: []netip.Prefix{mpp("2001:db8::1/128"), mpp("2001:db8:dead:beaf::/64")},
		},
		{
			name: "nope V4",
			pfxs: []netip.Prefix{mpp("10.0.0.0/16"), mpp("10.4.0.0/14")},
			spfx: mpp("10.1.0.0/16"),
			want: nil,
		},
		{
			name: "nope V6",
			pfxs: []netip.Prefix{mpp("2001:db0::/32"), mpp("2001:db4::/30")},
			spfx: mpp("2001:db1::/32"),
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rtbl := new(Table2[any])

			for _, pfx := range tt.pfxs {
				rtbl.Insert(pfx, nil)
			}

			got := rtbl.Subnets(tt.spfx)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("%s: got:\n%v\nwant:\n%v", tt.name, got, tt.want)
			}
		})
	}
}

func TestSubnetsCompare2(t *testing.T) {
	t.Parallel()

	pfxs := randomPrefixes(10_000)

	slow := slowRT[int]{pfxs}
	fast := Table2[int]{}

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	for i := 0; i < 10_000; i++ {
		pfx := randomPrefix()

		slowPfxs := slow.subnets(pfx)
		fastPfxs := fast.Subnets(pfx)

		if !reflect.DeepEqual(slowPfxs, fastPfxs) {
			t.Fatalf("Subnets(%q), got: %v\nwant: %v", pfx, fastPfxs, slowPfxs)
		}

	}
}

func TestSupernetsEdgeCases2(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		pfxs []netip.Prefix // input prefixes
		spfx netip.Prefix   // supernet search prefix
		want []netip.Prefix
	}{
		{
			name: "empty V4",
			pfxs: nil,
			spfx: mpp("0.0.0.0/0"),
			want: nil,
		},
		{
			name: "empty V6",
			pfxs: nil,
			spfx: mpp("::/0"),
			want: nil,
		},
		{
			name: "default V4",
			pfxs: []netip.Prefix{mpp("0.0.0.0/0")},
			spfx: mpp("0.0.0.0/0"),
			want: []netip.Prefix{mpp("0.0.0.0/0")},
		},
		{
			name: "default V6",
			pfxs: []netip.Prefix{mpp("::/0")},
			spfx: mpp("::/0"),
			want: []netip.Prefix{mpp("::/0")},
		},
		{
			name: "self V4",
			pfxs: []netip.Prefix{mpp("192.168.128.0/19")},
			spfx: mpp("192.168.128.0/19"),
			want: []netip.Prefix{mpp("192.168.128.0/19")},
		},
		{
			name: "self V6",
			pfxs: []netip.Prefix{mpp("2001:db8::dead:beef/128")},
			spfx: mpp("2001:db8::dead:beef/128"),
			want: []netip.Prefix{mpp("2001:db8::dead:beef/128")},
		},
		{
			name: "same leaf V4",
			pfxs: []netip.Prefix{mpp("10.0.0.0/29")},
			spfx: mpp("10.0.0.1/32"),
			want: []netip.Prefix{mpp("10.0.0.0/29")},
		},
		{
			name: "same leaf V6",
			pfxs: []netip.Prefix{mpp("::0/121")},
			spfx: mpp("::1/128"),
			want: []netip.Prefix{mpp("::0/121")},
		},
		{
			name: "intermediate V4",
			pfxs: []netip.Prefix{mpp("10.10.0.0/16"), mpp("10.0.0.0/13"), mpp("10.0.0.1/32"), mpp("10.0.1.1/32")},
			spfx: mpp("10.0.0.0/16"),
			want: []netip.Prefix{mpp("10.0.0.0/13")},
		},
		{
			name: "intermediate V6",
			pfxs: []netip.Prefix{mpp("2001:db8::0/32"), mpp("2001:db8:dead:beaf::/64")},
			spfx: mpp("2001:db8::/64"),
			want: []netip.Prefix{mpp("2001:db8::0/32")},
		},
		{
			name: "nope V4",
			pfxs: []netip.Prefix{mpp("10.0.0.0/16"), mpp("10.4.0.0/14")},
			spfx: mpp("10.1.0.0/16"),
			want: nil,
		},
		{
			name: "nope V6",
			pfxs: []netip.Prefix{mpp("2001:db0::/32"), mpp("2001:db4::/30")},
			spfx: mpp("2001:db1::/32"),
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rtbl := new(Table2[string])

			for _, pfx := range tt.pfxs {
				rtbl.Insert(pfx, tt.name)
			}

			got := rtbl.Supernets(tt.spfx)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("%s: got:\n%v\nwant:\n%v", tt.name, got, tt.want)
			}
		})
	}
}

func TestSupernetsCompare2(t *testing.T) {
	t.Parallel()

	pfxs := randomPrefixes(10_000)

	slow := slowRT[int]{pfxs}
	fast := Table2[int]{}

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	for i := 0; i < 10_000; i++ {
		pfx := randomPrefix()

		slowPfxs := slow.supernets(pfx)
		fastPfxs := fast.Supernets(pfx)

		if !reflect.DeepEqual(slowPfxs, fastPfxs) {
			t.Fatalf("Supernets(%q), got: %v\nwant: %v", pfx, fastPfxs, slowPfxs)
		}
	}
}

func TestOverlapsPrefixEdgeCases2(t *testing.T) {
	t.Parallel()

	tbl := &Table2[int]{}

	// empty table
	checkOverlaps2(t, tbl, []tableOverlapsTest{
		{"0.0.0.0/0", false},
		{"::/0", false},
	})

	// default route
	tbl.Insert(mpp("10.0.0.0/8"), 0)
	tbl.Insert(mpp("2001:db8::/32"), 0)
	checkOverlaps2(t, tbl, []tableOverlapsTest{
		{"0.0.0.0/0", true},
		{"::/0", true},
	})

	// default route
	tbl = &Table2[int]{}
	tbl.Insert(mpp("0.0.0.0/0"), 0)
	tbl.Insert(mpp("::/0"), 0)
	checkOverlaps2(t, tbl, []tableOverlapsTest{
		{"10.0.0.0/8", true},
		{"2001:db8::/32", true},
	})

	// single IP
	tbl = &Table2[int]{}
	tbl.Insert(mpp("10.0.0.0/7"), 0)
	tbl.Insert(mpp("2001::/16"), 0)
	checkOverlaps2(t, tbl, []tableOverlapsTest{
		{"10.1.2.3/32", true},
		{"2001:db8:affe::cafe/128", true},
	})

	// single IPv
	tbl = &Table2[int]{}
	tbl.Insert(mpp("10.1.2.3/32"), 0)
	tbl.Insert(mpp("2001:db8:affe::cafe/128"), 0)
	checkOverlaps2(t, tbl, []tableOverlapsTest{
		{"10.0.0.0/7", true},
		{"2001::/16", true},
	})

	// same IPv
	tbl = &Table2[int]{}
	tbl.Insert(mpp("10.1.2.3/32"), 0)
	tbl.Insert(mpp("2001:db8:affe::cafe/128"), 0)
	checkOverlaps2(t, tbl, []tableOverlapsTest{
		{"10.1.2.3/32", true},
		{"2001:db8:affe::cafe/128", true},
	})
}

func TestOverlapsPrefixCompare2(t *testing.T) {
	t.Parallel()
	pfxs := randomPrefixes(100_000)

	slow := slowRT[int]{pfxs}
	fast := Table2[int]{}

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

func TestUnionRegression2(t *testing.T) {
	t.Run("reg 1", func(t *testing.T) {
		aTbl := &Table2[string]{}
		bTbl := &Table2[string]{}

		aTbl.Insert(mpp("219.0.0.0/9"), "219.0.0.0/9")
		bTbl.Insert(mpp("219.126.65.199/32"), "219.126.65.199/32")

		aTbl.Union(bTbl)
		want := `▼
└─ 219.0.0.0/9 (219.0.0.0/9)
   └─ 219.126.65.199/32 (219.126.65.199/32)
`
		got := aTbl.String()
		if got != want {
			t.Fatalf("got:\n%v\nwant:\n%v", got, want)
		}
	})

	t.Run("reg 2", func(t *testing.T) {
		aTbl := &Table2[string]{}
		bTbl := &Table2[string]{}

		aTbl.Insert(mpp("219.0.0.0/8"), "219.0.0.0/8")
		aTbl.Insert(mpp("219.126.0.0/15"), "219.126.0.0/15")

		bTbl.Insert(mpp("219.126.3.4/32"), "219.126.3.4/32")

		aTbl.Union(bTbl)
		want := `▼
└─ 219.0.0.0/8 (219.0.0.0/8)
   └─ 219.126.0.0/15 (219.126.0.0/15)
      └─ 219.126.3.4/32 (219.126.3.4/32)
`
		got := aTbl.String()
		if got != want {
			t.Fatalf("got:\n%v\nwant:\n%v", got, want)
		}
	})

	t.Run("reg 3", func(t *testing.T) {
		aTbl := &Table2[string]{}
		bTbl := &Table2[string]{}

		aTbl.Insert(mpp("219.0.0.0/8"), "219.0.0.0/8")
		aTbl.Insert(mpp("219.126.3.4/32"), "219.126.3.4/32")

		bTbl.Insert(mpp("219.126.0.0/15"), "219.126.0.0/15")

		aTbl.Union(bTbl)
		want := `▼
└─ 219.0.0.0/8 (219.0.0.0/8)
   └─ 219.126.0.0/15 (219.126.0.0/15)
      └─ 219.126.3.4/32 (219.126.3.4/32)
`
		got := aTbl.String()
		if got != want {
			t.Fatalf("got:\n%v\nwant:\n%v", got, want)
		}
	})
}

func TestUnionEdgeCases2(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		aTbl := &Table2[int]{}
		bTbl := &Table2[int]{}

		// union empty tables
		aTbl.Union(bTbl)

		want := ""
		got := aTbl.String()
		if got != want {
			t.Fatalf("got:\n%v\nwant:\n%v", got, want)
		}
	})

	t.Run("other empty", func(t *testing.T) {
		aTbl := &Table2[int]{}
		bTbl := &Table2[int]{}

		// one empty table, b
		aTbl.Insert(mpp("0.0.0.0/0"), 0)

		aTbl.Union(bTbl)
		want := `▼
└─ 0.0.0.0/0 (0)
`
		got := aTbl.String()
		if got != want {
			t.Fatalf("got:\n%v\nwant:\n%v", got, want)
		}
	})

	t.Run("other empty", func(t *testing.T) {
		aTbl := &Table2[int]{}
		bTbl := &Table2[int]{}

		// one empty table, a
		bTbl.Insert(mpp("0.0.0.0/0"), 0)

		aTbl.Union(bTbl)
		want := `▼
└─ 0.0.0.0/0 (0)
`
		got := aTbl.String()
		if got != want {
			t.Fatalf("got:\n%v\nwant:\n%v", got, want)
		}
	})

	t.Run("duplicate prefix", func(t *testing.T) {
		aTbl := &Table2[string]{}
		bTbl := &Table2[string]{}

		// one empty table
		aTbl.Insert(mpp("::/0"), "orig value")
		bTbl.Insert(mpp("::/0"), "overwrite")

		aTbl.Union(bTbl)
		want := `▼
└─ ::/0 (overwrite)
`
		got := aTbl.String()
		if got != want {
			t.Fatalf("got:\n%v\nwant:\n%v", got, want)
		}
	})

	t.Run("different IP versions", func(t *testing.T) {
		aTbl := &Table2[int]{}
		bTbl := &Table2[int]{}

		// one empty table
		aTbl.Insert(mpp("0.0.0.0/0"), 1)
		bTbl.Insert(mpp("::/0"), 2)

		aTbl.Union(bTbl)
		want := `▼
└─ 0.0.0.0/0 (1)
▼
└─ ::/0 (2)
`
		got := aTbl.String()
		if got != want {
			t.Fatalf("got:\n%v\nwant:\n%v", got, want)
		}
	})

	t.Run("same children", func(t *testing.T) {
		aTbl := &Table2[int]{}
		bTbl := &Table2[int]{}

		aTbl.Insert(mpp("127.0.0.1/32"), 1)
		aTbl.Insert(mpp("::1/128"), 1)

		bTbl.Insert(mpp("127.0.0.2/32"), 2)
		bTbl.Insert(mpp("::2/128"), 2)

		aTbl.Union(bTbl)
		want := `▼
├─ 127.0.0.1/32 (1)
└─ 127.0.0.2/32 (2)
▼
├─ ::1/128 (1)
└─ ::2/128 (2)
`
		got := aTbl.String()
		if got != want {
			t.Fatalf("got:\n%v\nwant:\n%v", got, want)
		}
	})
}

// TestUnionMemoryAliasing tests that the Union method does not alias memory
// between the two tables.
func TestUnionMemoryAliasing2(t *testing.T) {
	newTable := func(pfx ...string) *Table2[struct{}] {
		var t Table2[struct{}]
		for _, s := range pfx {
			t.Insert(mpp(s), struct{}{})
		}
		return &t
	}
	// First create two tables with disjoint prefixes.
	stable := newTable("0.0.0.0/24")
	temp := newTable("100.69.1.0/24")

	// Verify that the tables are disjoint.
	if stable.Overlaps(temp) {
		t.Error("stable should not overlap temp")
	}

	// Now union them.
	temp.Union(stable)

	// Add a new prefix to temp.
	temp.Insert(mpp("0.0.1.0/24"), struct{}{})

	// Ensure that stable is unchanged.
	_, ok := stable.Lookup(mpa("0.0.1.1"))
	if ok {
		t.Error("stable should not contain 0.0.1.1")
	}
	if stable.OverlapsPrefix(mpp("0.0.1.1/32")) {
		t.Error("stable should not overlap 0.0.1.1/32")
	}
}

func TestUnionCompare2(t *testing.T) {
	t.Parallel()

	const numEntries = 1

	var pfxsA []slowRTEntry[int]
	var pfxsB []slowRTEntry[int]

	var slowA slowRT[int]
	var fastA Table2[int]

	var slowAC slowRT[int]
	var fastAC Table2[int]

	var slowB slowRT[int]
	var fastB Table2[int]

	var slowBC slowRT[int]
	var fastBC Table2[int]

	defer func() {
		if t.Failed() {
			t.Logf("slowAC:\n%s", slowAC.dumpString())
			t.Logf("slowBC:\n%s", slowBC.dumpString())
			t.Logf("fastAC:\n%s", fastAC.String())
			t.Logf("fastBC:\n%s", fastBC.String())

			t.Logf("slowA:\n%s", slowA.dumpString())
			t.Logf("fastA:\n%s", fastA.String())
		}
	}()

	for i := 0; i < 1_000; i++ {
		pfxsA = randomPrefixes4(numEntries)

		slowA = slowRT[int]{pfxsA}
		slowAC = slowRT[int]{pfxsA}
		fastA = Table2[int]{}
		fastAC = Table2[int]{}
		for _, pfx := range pfxsA {
			fastA.Insert(pfx.pfx, pfx.val)
			fastAC.Insert(pfx.pfx, pfx.val)
		}

		pfxsB = randomPrefixes4(numEntries)

		slowB = slowRT[int]{pfxsB}
		slowBC = slowRT[int]{pfxsB}
		fastB = Table2[int]{}
		fastBC = Table2[int]{}
		for _, pfx := range pfxsB {
			fastB.Insert(pfx.pfx, pfx.val)
			fastBC.Insert(pfx.pfx, pfx.val)
		}

		slowA.union(&slowB)
		fastA.Union(&fastB)

		// dump as slow table for comparison
		fastAsSlowTable := fastA.dumpAsPrefixTable()

		// sort for comparison
		slowA.sort()
		fastAsSlowTable.sort()

		for i := range slowA.entries {
			slowI := slowA.entries[i]
			fastI := fastAsSlowTable.entries[i]

			if slowI != fastI {
				t.Fatalf("Union(...): items[%d] differ slow(%v) != fast(%v)", i, slowI, fastI)
			}
		}
	}
}

// ############################################################################

// checkOverlaps2 verifies that the overlaps lookups in tt return the
// expected results on tbl.
func checkOverlaps2(t *testing.T, tbl *Table2[int], tests []tableOverlapsTest) {
	for _, tt := range tests {
		got := tbl.OverlapsPrefix(mpp(tt.prefix))
		if got != tt.want {
			t.Log(tbl.String())
			t.Errorf("OverlapsPrefix(%v) = %v, want %v", mpp(tt.prefix), got, tt.want)
		}
	}
}

// dumpAsPrefixTable, just a helper to compare with slowPrefixTable
func (t *Table2[V]) dumpAsPrefixTable() slowRT[V] {
	pfxs := []slowRTEntry[V]{}

	pfxs = dumpListRec2(pfxs, t.DumpList4())
	pfxs = dumpListRec2(pfxs, t.DumpList6())

	ret := slowRT[V]{pfxs}
	return ret
}

func dumpListRec2[V any](pfxs []slowRTEntry[V], dumpList []DumpListNode[V]) []slowRTEntry[V] {
	for _, node := range dumpList {
		pfxs = append(pfxs, slowRTEntry[V]{pfx: node.CIDR, val: node.Value})
		pfxs = append(pfxs, dumpListRec[V](nil, node.Subnets)...)
	}
	return pfxs
}

// #########################################################

// checkRoutes verifies that the route lookups in tt return the
// expected results on tbl.
func checkRoutes2(t *testing.T, tbl *Table2[int], tt []tableTest) {
	t.Helper()
	for _, tc := range tt {
		v, ok := tbl.Lookup(mpa(tc.addr))

		if !ok && tc.want != -1 {
			t.Errorf("Lookup %q got (%v, %v), want (%v, true)", tc.addr, v, ok, tc.want)
		}
		if ok && v != tc.want {
			t.Errorf("Lookup %q got (%v, %v), want (%v, true)", tc.addr, v, ok, tc.want)
		}
	}
}

func checkSize2(t *testing.T, tbl *Table2[int], want int) {
	tbl.init()
	t.Helper()
	if got := tbl.numNodes(); got != want {
		t.Errorf("wrong table size, got %d strides want %d", got, want)
	}
}

func (t *Table2[V]) numNodes() int {
	t.init()
	return t.numNodesRec(t.rootV4) + t.numNodesRec(t.rootV6)
}

func (t *Table2[V]) numNodesRec(n *node2[V]) int {
	ret := 1 // this node
	for _, c := range n.children {
		ret += t.numNodesRec(c)
	}
	return ret
}

func (t *Table2[V]) numPrefixes() int {
	t.init()
	return t.numPrefixesRec(t.rootV4) + t.numPrefixesRec(t.rootV6)
}

func (t *Table2[V]) numPrefixesRec(n *node2[V]) int {
	ret := len(n.prefixes)
	for _, c := range n.children {
		ret += t.numPrefixesRec(c)
	}
	return ret
}

func BenchmarkTableLookup2(b *testing.B) {
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		// same routes
		for _, nroutes := range []int{10, 100, 1_000, 10_000, 100_000, 1_000_000} {
			runtime.GC()

			var rt1 Table[int]
			var rt2 Table2[int]
			for _, route := range rng(nroutes) {
				rt1.Insert(route.pfx, route.val)
				rt2.Insert(route.pfx, route.val)
			}

			// same probe
			probe := rng(1)[0]

			b.ResetTimer()
			b.Run(fmt.Sprintf("orig/IP:  %s/In_%7d", fam, nroutes), func(b *testing.B) {
				for range b.N {
					writeSink, _ = rt1.Lookup(probe.pfx.Addr())
				}
				b.ReportMetric(float64(rt1.numNodes()), "Nodes")
			})

			b.ResetTimer()
			b.Run(fmt.Sprintf("comp/IP:  %s/In_%7d", fam, nroutes), func(b *testing.B) {
				for range b.N {
					writeSink, _ = rt2.Lookup(probe.pfx.Addr())
				}
				b.ReportMetric(float64(rt2.numNodes()), "Nodes")
			})

		}
	}
}

func BenchmarkTableLookupPrefix2(b *testing.B) {
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		// same routes
		for _, nroutes := range []int{10, 100, 1_000, 10_000, 100_000, 1_000_000} {
			runtime.GC()

			var rt1 Table[int]
			var rt2 Table2[int]
			for _, route := range rng(nroutes) {
				rt1.Insert(route.pfx, route.val)
				rt2.Insert(route.pfx, route.val)
			}

			// same probe
			probe := rng(1)[0]

			b.ResetTimer()
			b.Run(fmt.Sprintf("orig/Pfx: %s/In_%7d", fam, nroutes), func(b *testing.B) {
				for range b.N {
					writeSink, _ = rt1.LookupPrefix(probe.pfx)
				}
				b.ReportMetric(float64(rt1.numNodes()), "Nodes")
			})

			b.ResetTimer()
			b.Run(fmt.Sprintf("comp/Pfx: %s/In_%7d", fam, nroutes), func(b *testing.B) {
				for range b.N {
					writeSink, _ = rt2.LookupPrefix(probe.pfx)
				}
				b.ReportMetric(float64(rt2.numNodes()), "Nodes")
			})

		}
	}
}

func BenchmarkTableLookupPrefixLPM2(b *testing.B) {
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		// same routes
		for _, nroutes := range []int{10, 100, 1_000, 10_000, 100_000, 1_000_000} {
			runtime.GC()

			var rt1 Table[int]
			var rt2 Table2[int]
			for _, route := range rng(nroutes) {
				rt1.Insert(route.pfx, route.val)
				rt2.Insert(route.pfx, route.val)
			}

			// same probe
			probe := rng(1)[0]

			b.ResetTimer()
			b.Run(fmt.Sprintf("orig/LPM: %s/In_%7d", fam, nroutes), func(b *testing.B) {
				for range b.N {
					_, writeSink, _ = rt1.LookupPrefixLPM(probe.pfx)
				}
				b.ReportMetric(float64(rt1.numNodes()), "Nodes")
			})

			b.ResetTimer()
			b.Run(fmt.Sprintf("comp/LPM: %s/In_%7d", fam, nroutes), func(b *testing.B) {
				for range b.N {
					_, writeSink, _ = rt2.LookupPrefixLPM(probe.pfx)
				}
				b.ReportMetric(float64(rt2.numNodes()), "Nodes")
			})

		}
	}
}

func BenchmarkSize2Random(b *testing.B) {
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		var startMem, endMem runtime.MemStats
		for _, nroutes := range []int{10, 100, 1_000, 10_000, 100_000, 1_000_000} {
			rt1 := new(Table[struct{}])
			rt2 := new(Table2[struct{}])

			b.Run(fmt.Sprintf("orig:%7d/%s", nroutes, fam), func(b *testing.B) {
				b.ResetTimer()

				for range b.N {
					rt1 = new(Table[struct{}])
					runtime.GC()
					runtime.ReadMemStats(&startMem)

					for _, route := range rng(nroutes) {
						rt1.Insert(route.pfx, struct{}{})
					}

					runtime.GC()
					runtime.ReadMemStats(&endMem)
					if npfx := rt1.numPrefixes(); npfx != nroutes {
						b.Fatalf("expect %v prefixes, got %v", nroutes, npfx)
					}

					b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc), "Bytes")
					b.ReportMetric(float64(rt1.numNodes()), "Nodes")
					b.ReportMetric(0, "ns/op") // silence
				}
			})
			b.Run(fmt.Sprintf("comp:%7d/%s", nroutes, fam), func(b *testing.B) {
				b.ResetTimer()

				for range b.N {
					rt2 = new(Table2[struct{}])
					runtime.GC()
					runtime.ReadMemStats(&startMem)

					for _, route := range rng(nroutes) {
						rt2.Insert(route.pfx, struct{}{})
					}

					runtime.GC()
					runtime.ReadMemStats(&endMem)
					if npfx := rt2.numPrefixes(); npfx != nroutes {
						b.Fatalf("expect %v prefixes, got %v", nroutes, npfx)
					}

					b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc), "Bytes")
					b.ReportMetric(float64(rt2.numNodes()), "Nodes")
					b.ReportMetric(0, "ns/op") // silence
				}
			})
		}
	}
}

func BenchmarkTableInsert2(b *testing.B) {
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		for _, nroutes := range benchRouteCount {
			var rt1 Table[struct{}]
			var rt2 Table2[struct{}]
			for _, route := range rng(nroutes) {
				rt1.Insert(route.pfx, struct{}{})
				rt2.Insert(route.pfx, struct{}{})
			}

			probe := rng(1)[0]

			b.ResetTimer()
			b.Run(fmt.Sprintf("orig/%s/Into_%d", fam, nroutes), func(b *testing.B) {
				for range b.N {
					rt1.Insert(probe.pfx, struct{}{})
				}
			})

			b.ResetTimer()
			b.Run(fmt.Sprintf("comp/%s/Into_%d", fam, nroutes), func(b *testing.B) {
				for range b.N {
					rt2.Insert(probe.pfx, struct{}{})
				}
			})
		}
	}
}

func BenchmarkTableGet2(b *testing.B) {
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		for _, nroutes := range benchRouteCount {
			var rt1 Table[struct{}]
			var rt2 Table2[struct{}]
			for _, route := range rng(nroutes) {
				rt1.Insert(route.pfx, struct{}{})
				rt2.Insert(route.pfx, struct{}{})
			}

			probe := rng(1)[0]

			b.ResetTimer()
			b.Run(fmt.Sprintf("orig/%s/From_%d", fam, nroutes), func(b *testing.B) {
				for range b.N {
					_, boolSink = rt1.Get(probe.pfx)
				}
			})

			b.ResetTimer()
			b.Run(fmt.Sprintf("comp/%s/From_%d", fam, nroutes), func(b *testing.B) {
				for range b.N {
					_, boolSink = rt2.Get(probe.pfx)
				}
			})

		}
	}
}

func BenchmarkTableDelete2Match(b *testing.B) {
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		for _, nroutes := range benchRouteCount {
			var rt1 Table[int]
			var rt2 Table2[int]

			routes := rng(nroutes)

			for _, route := range routes {
				rt1.Insert(route.pfx, route.val)
				rt2.Insert(route.pfx, route.val)
			}

			nodes1 := rt1.numNodes()
			nodes2 := rt2.numNodes()

			probe := routes[rand.Intn(nroutes)]

			b.ResetTimer()
			b.Run(fmt.Sprintf("orig/%s/From_%d", fam, nroutes), func(b *testing.B) {
				for range b.N {
					rt1.Delete(probe.pfx)
				}
				b.ReportMetric(float64(rt1.numNodes()-nodes1), "delta_Nodes")
			})

			b.ResetTimer()
			b.Run(fmt.Sprintf("comp/%s/From_%d", fam, nroutes), func(b *testing.B) {
				for range b.N {
					rt2.Delete(probe.pfx)
				}
				b.ReportMetric(float64(rt2.numNodes()-nodes2), "delta_Nodes")
			})
		}
	}
}
