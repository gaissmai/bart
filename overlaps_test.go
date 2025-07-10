// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause
//
// some tests modified from github.com/tailscale/art
// for this implementation by:
//
// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"math/rand/v2"
	"net/netip"
	"testing"
)

func TestRegressionOverlaps(t *testing.T) {
	t.Parallel()

	t.Run("overlaps_divergent_children_with_parent_route_entry", func(t *testing.T) {
		t.Parallel()
		t1, t2 := new(Table[int]), new(Table[int])

		t1.Insert(mpp("128.0.0.0/2"), 1)
		t1.Insert(mpp("99.173.128.0/17"), 1)
		t1.Insert(mpp("219.150.142.0/23"), 1)
		t1.Insert(mpp("164.148.190.250/31"), 1)
		t1.Insert(mpp("48.136.229.233/32"), 1)

		t2.Insert(mpp("217.32.0.0/11"), 1)
		t2.Insert(mpp("38.176.0.0/12"), 1)
		t2.Insert(mpp("106.16.0.0/13"), 1)
		t2.Insert(mpp("164.85.192.0/23"), 1)
		t2.Insert(mpp("225.71.164.112/31"), 1)

		if !t1.Overlaps(t2) {
			t.Fatal("tables unexpectedly do not overlap")
		}
	})

	t.Run("overlaps_parent_child_comparison_with_route_in_parent", func(t *testing.T) {
		t.Parallel()
		t1, t2 := new(Table[int]), new(Table[int])

		t1.Insert(mpp("226.0.0.0/8"), 1)
		t1.Insert(mpp("81.128.0.0/9"), 1)
		t1.Insert(mpp("152.0.0.0/9"), 1)
		t1.Insert(mpp("151.220.0.0/16"), 1)
		t1.Insert(mpp("89.162.61.0/24"), 1)

		t2.Insert(mpp("54.0.0.0/9"), 1)
		t2.Insert(mpp("35.89.128.0/19"), 1)
		t2.Insert(mpp("72.33.53.0/24"), 1)
		t2.Insert(mpp("2.233.60.32/27"), 1)
		t2.Insert(mpp("152.42.142.160/28"), 1)

		if !t1.Overlaps(t2) {
			t.Fatal("tables unexpectedly do not overlap")
		}
	})
}

func TestOverlapsCompare(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))

	// Empirically, between 5 and 6 routes per table results in ~50%
	// of random pairs overlapping. Cool example of the birthday paradox!
	const numEntries = 6

	seen := map[bool]int{}
	for range 10_000 {
		pfxs := randomPrefixes(prng, numEntries)
		fast := new(Table[int])
		gold := new(goldTable[int]).insertMany(pfxs)

		for _, pfx := range pfxs {
			fast.Insert(pfx.pfx, pfx.val)
		}

		inter := randomPrefixes(prng, numEntries)
		goldInter := new(goldTable[int]).insertMany(inter)
		fastInter := new(Table[int])
		for _, pfx := range inter {
			fastInter.Insert(pfx.pfx, pfx.val)
		}

		gotGold := gold.overlaps(goldInter)
		gotFast := fast.Overlaps(fastInter)

		if gotGold != gotFast {
			t.Fatalf("Overlaps(...) = %v, want %v\nTable1:\n%s\nTable:\n%v",
				gotFast, gotGold, fast.String(), fastInter.String())
		}

		seen[gotFast]++
	}
}

func TestOverlapsPrefixCompare(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, 100_000)

	fast := new(Table[int])
	gold := new(goldTable[int]).insertMany(pfxs)

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	tests := randomPrefixes(prng, 10_000)
	for _, tt := range tests {
		gotGold := gold.overlapsPrefix(tt.pfx)
		gotFast := fast.OverlapsPrefix(tt.pfx)
		if gotGold != gotFast {
			t.Fatalf("overlapsPrefix(%q) = %v, want %v", tt.pfx, gotFast, gotGold)
		}
	}
}

func TestOverlapsChildren(t *testing.T) {
	t.Parallel()
	pfxs1 := []netip.Prefix{
		// pfxs
		mpp("10.0.0.0/8"),
		mpp("11.0.0.0/8"),
		mpp("12.0.0.0/8"),
		mpp("13.0.0.0/8"),
		mpp("14.0.0.0/8"),
		// chi5dren
		mpp("10.100.0.0/17"),
		mpp("11.100.0.0/17"),
		mpp("12.100.0.0/17"),
		mpp("13.100.0.0/17"),
		mpp("14.100.0.0/17"),
		mpp("15.100.0.0/17"),
		mpp("16.100.0.0/17"),
		mpp("17.100.0.0/17"),
		mpp("18.100.0.0/17"),
		mpp("19.100.0.0/17"),
		mpp("20.100.0.0/17"),
		mpp("21.100.0.0/17"),
		mpp("22.100.0.0/17"),
		mpp("23.100.0.0/17"),
		mpp("24.100.0.0/17"),
		mpp("25.100.0.0/17"),
		mpp("26.100.0.0/17"),
		mpp("27.100.0.0/17"),
		mpp("28.100.0.0/17"),
	}
	pfxs2 := []netip.Prefix{
		mpp("200.0.0.0/8"),
		mpp("201.0.0.0/8"),
		mpp("202.0.0.0/8"),
		mpp("203.0.0.0/8"),
		mpp("204.0.0.0/8"),
		// children
		mpp("201.200.0.0/18"),
		mpp("202.200.0.0/18"),
		mpp("203.200.0.0/18"),
		mpp("204.200.0.0/18"),
		mpp("205.200.0.0/18"),
		mpp("206.200.0.0/18"),
		mpp("207.200.0.0/18"),
		mpp("208.200.0.0/18"),
		mpp("209.200.0.0/18"),
		mpp("210.200.0.0/18"),
		mpp("211.200.0.0/18"),
		mpp("212.200.0.0/18"),
		mpp("213.200.0.0/18"),
		mpp("214.200.0.0/18"),
		mpp("215.200.0.0/18"),
		mpp("216.200.0.0/18"),
		mpp("217.200.0.0/18"),
		mpp("218.200.0.0/18"),
		mpp("219.200.0.0/18"),
	}

	tbl1 := new(Table[string])
	for _, pfx := range pfxs1 {
		tbl1.Insert(pfx, pfx.String())
	}

	tbl2 := new(Table[string])
	for _, pfx := range pfxs2 {
		tbl2.Insert(pfx, pfx.String())
	}
	if tbl1.Overlaps(tbl2) {
		t.Fatal("tables unexpectedly do overlap")
	}
}
