// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause
//
// some regression tests modified from github.com/tailscale/art
// for this implementation by:
//
// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"math/rand"
	"net/netip"
	"testing"
)

// ############ tests ################################

func TestLiteInvalid(t *testing.T) {
	t.Parallel()

	tbl := new(Lite)
	var zeroPfx netip.Prefix
	var zeroIP netip.Addr
	var testname string

	testname = "Insert"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid prefix input", testname)
			}
		}(testname)

		tbl.Insert(zeroPfx)
	})

	testname = "InsertPersist"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid prefix input", testname)
			}
		}(testname)

		_ = tbl.InsertPersist(zeroPfx)
	})

	testname = "Delete"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid prefix input", testname)
			}
		}(testname)

		tbl.Delete(zeroPfx)
	})

	testname = "DeletePersist"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid prefix input", testname)
			}
		}(testname)

		_ = tbl.DeletePersist(zeroPfx)
	})

	testname = "Get"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid prefix input", testname)
			}
		}(testname)

		_, _ = tbl.Get(zeroPfx)
	})

	testname = "GetAndDelete"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid prefix input", testname)
			}
		}(testname)

		_, _ = tbl.GetAndDelete(zeroPfx)
	})

	testname = "GetAndDeletePersist"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid prefix input", testname)
			}
		}(testname)

		_, _, _ = tbl.GetAndDeletePersist(zeroPfx)
	})

	testname = "Contains"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid IP input", testname)
			}
		}(testname)

		if tbl.Contains(zeroIP) != false {
			t.Errorf("%s returns true on invalid IP input, expected false", testname)
		}
	})

	testname = "Lookup"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid IP input", testname)
			}
		}(testname)

		_, got := tbl.Lookup(zeroIP)
		if got != false {
			t.Errorf("%s returns true on invalid IP input, expected false", testname)
		}
	})

	testname = "LookupPrefix"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid prefix input", testname)
			}
		}(testname)

		tbl.LookupPrefix(zeroPfx)
	})

	testname = "LookupPrefixLPM"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid prefix input", testname)
			}
		}(testname)

		tbl.LookupPrefixLPM(zeroPfx)
	})

	testname = "OverlapsPrefix"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid prefix input", testname)
			}
		}(testname)

		tbl.OverlapsPrefix(zeroPfx)
	})

	testname = "Contains"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid ip input", testname)
			}
		}(testname)

		tbl.Contains(zeroIP)
	})
}

func TestLiteDeletePersist(t *testing.T) {
	t.Parallel()

	t.Run("table_is_empty", func(t *testing.T) {
		t.Parallel()
		// must not panic
		tbl := new(Lite)
		checkLiteNumNodes(t, tbl, 0)
		tbl = tbl.DeletePersist(randomPrefix())
		checkLiteNumNodes(t, tbl, 0)
	})

	t.Run("prefix_in_root", func(t *testing.T) {
		t.Parallel()
		// Add/remove prefix from root table.
		tbl := new(Lite)
		checkLiteNumNodes(t, tbl, 0)

		tbl.Insert(mpp("10.0.0.0/8"))
		checkLiteNumNodes(t, tbl, 1)
		tbl = tbl.DeletePersist(mpp("10.0.0.0/8"))
		checkLiteNumNodes(t, tbl, 0)
	})

	t.Run("prefix_in_leaf", func(t *testing.T) {
		t.Parallel()
		// Create, then delete a single leaf table.
		tbl := new(Lite)
		checkLiteNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"))
		checkLiteNumNodes(t, tbl, 1)

		tbl = tbl.DeletePersist(mpp("192.168.0.1/32"))
		checkLiteNumNodes(t, tbl, 0)
	})

	t.Run("intermediate_no_routes", func(t *testing.T) {
		t.Parallel()
		// Create an intermediate with 2 leaves, then delete one leaf.
		tbl := new(Lite)
		checkLiteNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"))
		tbl.Insert(mpp("192.180.0.1/32"))
		checkLiteNumNodes(t, tbl, 2)

		tbl = tbl.DeletePersist(mpp("192.180.0.1/32"))
		checkLiteNumNodes(t, tbl, 1)
	})

	t.Run("intermediate_with_route", func(t *testing.T) {
		t.Parallel()
		// Same, but the intermediate carries a route as well.
		tbl := new(Lite)
		checkLiteNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"))
		tbl.Insert(mpp("192.180.0.1/32"))
		tbl.Insert(mpp("192.0.0.0/10"))

		checkLiteNumNodes(t, tbl, 2)

		tbl = tbl.DeletePersist(mpp("192.180.0.1/32"))
		checkLiteNumNodes(t, tbl, 2)
	})

	t.Run("intermediate_many_leaves", func(t *testing.T) {
		t.Parallel()
		// Intermediate with 3 leaves, then delete one leaf.
		tbl := new(Lite)
		checkLiteNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"))
		tbl.Insert(mpp("192.180.0.1/32"))
		tbl.Insert(mpp("192.200.0.1/32"))

		checkLiteNumNodes(t, tbl, 2)

		tbl = tbl.DeletePersist(mpp("192.180.0.1/32"))
		checkLiteNumNodes(t, tbl, 2)
	})

	t.Run("nosuchprefix_missing_child", func(t *testing.T) {
		t.Parallel()
		// Delete non-existent prefix
		tbl := new(Lite)
		checkLiteNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"))
		checkLiteNumNodes(t, tbl, 1)

		tbl = tbl.DeletePersist(mpp("200.0.0.0/32"))
		checkLiteNumNodes(t, tbl, 1)
	})

	t.Run("intermediate_with_deleted_route", func(t *testing.T) {
		t.Parallel()
		// Intermediate node loses its last route and becomes
		// compactable.
		tbl := new(Lite)
		checkLiteNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"))
		tbl.Insert(mpp("192.168.0.0/22"))
		checkLiteNumNodes(t, tbl, 3)

		tbl = tbl.DeletePersist(mpp("192.168.0.0/22"))
		checkLiteNumNodes(t, tbl, 1)
	})

	t.Run("default_route", func(t *testing.T) {
		t.Parallel()
		tbl := new(Lite)
		checkLiteNumNodes(t, tbl, 0)

		tbl.Insert(mpp("0.0.0.0/0"))
		tbl.Insert(mpp("::/0"))
		tbl = tbl.DeletePersist(mpp("0.0.0.0/0"))

		checkLiteNumNodes(t, tbl, 1)
	})

	t.Run("path compressed purge", func(t *testing.T) {
		t.Parallel()
		tbl := new(Lite)
		checkLiteNumNodes(t, tbl, 0)

		tbl.Insert(mpp("10.10.0.0/17"))
		tbl.Insert(mpp("10.20.0.0/17"))
		checkLiteNumNodes(t, tbl, 2)

		tbl = tbl.DeletePersist(mpp("10.20.0.0/17"))
		checkLiteNumNodes(t, tbl, 1)

		tbl = tbl.DeletePersist(mpp("10.10.0.0/17"))
		checkLiteNumNodes(t, tbl, 0)
	})
}

func TestLiteContainsCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()
	pfxs := randomPrefixes(10_000)

	gold := new(goldTable[int]).insertMany(pfxs)
	fast := new(Table[int])

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	for range 10_000 {
		a := randomAddr()

		_, goldOK := gold.lookup(a)
		fastOK := fast.Contains(a)

		if goldOK != fastOK {
			t.Fatalf("Contains(%q) = %v, want %v", a, fastOK, goldOK)
		}
	}
}

func TestLiteLookupCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()
	pfxs := randomPrefixes(10_000)

	fast := new(Table[int])
	gold := new(goldTable[int]).insertMany(pfxs)

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	seenVals4 := map[int]bool{}
	seenVals6 := map[int]bool{}

	for range 10_000 {
		a := randomAddr()

		goldVal, goldOK := gold.lookup(a)
		fastVal, fastOK := fast.Lookup(a)

		if !getsEqual(goldVal, goldOK, fastVal, fastOK) {
			t.Fatalf("Lookup(%q) = (%v, %v), want (%v, %v)", a, fastVal, fastOK, goldVal, goldOK)
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

func TestLiteLookupPrefixUnmasked(t *testing.T) {
	// test that the pfx must not be masked on input for LookupPrefix
	t.Parallel()

	rt := new(Table[any])
	rt.Insert(mpp("10.20.30.0/24"), nil)

	// not normalized pfxs
	tests := []struct {
		probe   netip.Prefix
		wantLPM netip.Prefix
		wantOk  bool
	}{
		{
			probe:   netip.MustParsePrefix("10.20.30.40/0"),
			wantLPM: netip.Prefix{},
			wantOk:  false,
		},
		{
			probe:   netip.MustParsePrefix("10.20.30.40/23"),
			wantLPM: netip.Prefix{},
			wantOk:  false,
		},
		{
			probe:   netip.MustParsePrefix("10.20.30.40/24"),
			wantLPM: mpp("10.20.30.0/24"),
			wantOk:  true,
		},
		{
			probe:   netip.MustParsePrefix("10.20.30.40/25"),
			wantLPM: mpp("10.20.30.0/24"),
			wantOk:  true,
		},
		{
			probe:   netip.MustParsePrefix("10.20.30.40/32"),
			wantLPM: mpp("10.20.30.0/24"),
			wantOk:  true,
		},
	}

	for _, tc := range tests {
		_, got := rt.LookupPrefix(tc.probe)
		if got != tc.wantOk {
			t.Errorf("LookupPrefix non canonical prefix (%s), got: %v, want: %v", tc.probe, got, tc.wantOk)
		}

		lpm, _, got := rt.LookupPrefixLPM(tc.probe)
		if got != tc.wantOk {
			t.Errorf("LookupPrefixLPM non canonical prefix (%s), got: %v, want: %v", tc.probe, got, tc.wantOk)
		}
		if lpm != tc.wantLPM {
			t.Errorf("LookupPrefixLPM non canonical prefix (%s), got: %v, want: %v", tc.probe, lpm, tc.wantLPM)
		}
	}
}

func TestLiteLookupPrefixCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()
	pfxs := randomPrefixes(10_000)

	fast := new(Table[int])
	gold := new(goldTable[int]).insertMany(pfxs)

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	seenVals4 := map[int]bool{}
	seenVals6 := map[int]bool{}

	for range 10_000 {
		pfx := randomPrefix()

		goldVal, goldOK := gold.lookupPfx(pfx)
		fastVal, fastOK := fast.LookupPrefix(pfx)

		if !getsEqual(goldVal, goldOK, fastVal, fastOK) {
			t.Fatalf("LookupPrefix(%q) = (%v, %v), want (%v, %v)", pfx, fastVal, fastOK, goldVal, goldOK)
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

func TestLiteLookupPrefixLPMCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()
	pfxs := randomPrefixes(10_000)

	fast := new(Table[int])
	gold := new(goldTable[int]).insertMany(pfxs)

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	seenVals4 := map[int]bool{}
	seenVals6 := map[int]bool{}

	for range 10_000 {
		pfx := randomPrefix()

		goldLPM, goldVal, goldOK := gold.lookupPfxLPM(pfx)
		fastLPM, fastVal, fastOK := fast.LookupPrefixLPM(pfx)

		if !getsEqual(goldVal, goldOK, fastVal, fastOK) {
			t.Fatalf("LookupPrefixLPM(%q) = (%v, %v), want (%v, %v)", pfx, fastVal, fastOK, goldVal, goldOK)
		}

		if !getsEqual(goldLPM, goldOK, fastLPM, fastOK) {
			t.Fatalf("LookupPrefixLPM(%q) = (%v, %v), want (%v, %v)", pfx, fastLPM, fastOK, goldLPM, goldOK)
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

func TestLiteInsertShuffled(t *testing.T) {
	// The order in which you insert prefixes into a route table
	// should not matter, as long as you're inserting the same set of
	// routes.
	t.Parallel()

	pfxs := randomPrefixes(1000)

	for range 10 {
		pfxs2 := append([]goldTableItem[int](nil), pfxs...)
		rand.Shuffle(len(pfxs2), func(i, j int) { pfxs2[i], pfxs2[j] = pfxs2[j], pfxs2[i] })

		addrs := make([]netip.Addr, 0, 10_000)
		for range 10_000 {
			addrs = append(addrs, randomAddr())
		}

		rt1 := new(Table[int])
		rt2 := new(Table[int])

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
				t.Fatalf("Lookup(%q) = (%v, %v), want (%v, %v)", a, val2, ok2, val1, ok1)
			}
		}
	}
}

func TestLiteInsertPersistShuffled(t *testing.T) {
	// The order in which you insert prefixes into a route table
	// should not matter, as long as you're inserting the same set of
	// routes.
	t.Parallel()

	pfxs := randomPrefixes(1000)

	for range 10 {
		pfxs2 := append([]goldTableItem[int](nil), pfxs...)
		rand.Shuffle(len(pfxs2), func(i, j int) { pfxs2[i], pfxs2[j] = pfxs2[j], pfxs2[i] })

		addrs := make([]netip.Addr, 0, 10_000)
		for range 10_000 {
			addrs = append(addrs, randomAddr())
		}

		rt1 := new(Table[int])
		rt2 := new(Table[int])

		// rt1 is mutable
		for _, pfx := range pfxs {
			rt1.Insert(pfx.pfx, pfx.val)
		}

		// rt2 is persistent
		for _, pfx := range pfxs2 {
			rt2 = rt2.InsertPersist(pfx.pfx, pfx.val)
		}

		if rt1.String() != rt2.String() {
			t.Fatal("mutable and immutable table have different string representation")
		}

		if rt1.dumpString() != rt2.dumpString() {
			t.Fatal("mutable and immutable table have different dumpString representation")
		}

		for _, a := range addrs {
			val1, ok1 := rt1.Lookup(a)
			val2, ok2 := rt2.Lookup(a)

			if !getsEqual(val1, ok1, val2, ok2) {
				t.Fatalf("Lookup(%q) = (%v, %v), want (%v, %v)", a, val2, ok2, val1, ok1)
			}
		}
	}
}

func TestLiteDeleteCompare(t *testing.T) {
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

	pfxs := append([]goldTableItem[int](nil), all4[:deleteCut]...)
	pfxs = append(pfxs, all6[:deleteCut]...)

	toDelete := append([]goldTableItem[int](nil), all4[deleteCut:]...)
	toDelete = append(toDelete, all6[deleteCut:]...)

	fast := new(Table[int])
	gold := new(goldTable[int]).insertMany(pfxs)

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

	for range numProbes {
		a := randomAddr()

		goldVal, goldOK := gold.lookup(a)
		fastVal, fastOK := fast.Lookup(a)

		if !getsEqual(goldVal, goldOK, fastVal, fastOK) {
			t.Fatalf("Lookup(%q) = (%v, %v), want (%v, %v)", a, fastVal, fastOK, goldVal, goldOK)
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

func TestLiteDeleteShuffled(t *testing.T) {
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

	pfxs := append([]goldTableItem[int](nil), all4[:deleteCut]...)
	pfxs = append(pfxs, all6[:deleteCut]...)

	toDelete := append([]goldTableItem[int](nil), all4[deleteCut:]...)
	toDelete = append(toDelete, all6[deleteCut:]...)

	rt1 := new(Table[int])
	for _, pfx := range pfxs {
		rt1.Insert(pfx.pfx, pfx.val)
	}
	for _, pfx := range toDelete {
		rt1.Insert(pfx.pfx, pfx.val)
	}
	for _, pfx := range toDelete {
		rt1.Delete(pfx.pfx)
	}

	for range 10 {
		pfxs2 := append([]goldTableItem[int](nil), pfxs...)
		toDelete2 := append([]goldTableItem[int](nil), toDelete...)
		rand.Shuffle(len(toDelete2), func(i, j int) { toDelete2[i], toDelete2[j] = toDelete2[j], toDelete2[i] })
		rt2 := new(Table[int])
		for _, pfx := range pfxs2 {
			rt2.Insert(pfx.pfx, pfx.val)
		}
		for _, pfx := range toDelete2 {
			rt2.Insert(pfx.pfx, pfx.val)
		}
		for _, pfx := range toDelete2 {
			rt2.Delete(pfx.pfx)
		}

		if rt1.String() != rt2.String() {
			t.Fatal("shuffled table has different string representation")
		}

		if rt1.dumpString() != rt2.dumpString() {
			t.Fatal("shuffled table has different dumpString representation")
		}
	}
}

func TestLiteDeleteIsReverseOfInsert(t *testing.T) {
	t.Parallel()
	// Insert N prefixes, then delete those same prefixes in reverse
	// order. Each deletion should exactly undo the internal structure
	// changes that each insert did.
	const N = 10_000

	tbl := new(Table[int])
	want := tbl.dumpString()

	prefixes := randomPrefixes(N)

	defer func() {
		if t.Failed() {
			t.Logf("the prefixes that fail the test: %v\n", prefixes)
		}
	}()

	for _, p := range prefixes {
		tbl.Insert(p.pfx, p.val)
	}

	for i := len(prefixes) - 1; i >= 0; i-- {
		tbl.Delete(prefixes[i].pfx)
	}
	if got := tbl.dumpString(); got != want {
		t.Fatalf("after delete, mismatch:\n\n got: %s\n\nwant: %s", got, want)
	}
}

func TestLiteGetAndDelete(t *testing.T) {
	t.Parallel()
	// Insert N prefixes, then delete those same prefixes in shuffled
	// order.
	const N = 10_000

	tbl := new(Table[int])
	prefixes := randomPrefixes(N)

	// insert the prefixes
	for _, p := range prefixes {
		tbl.Insert(p.pfx, p.val)
	}

	// shuffle the prefixes
	rand.Shuffle(N, func(i, j int) {
		prefixes[i], prefixes[j] = prefixes[j], prefixes[i]
	})

	for _, p := range prefixes {
		want, _ := tbl.Get(p.pfx)
		val, ok := tbl.GetAndDelete(p.pfx)

		if !ok {
			t.Errorf("GetAndDelete, expected true, got %v", ok)
		}

		if val != want {
			t.Errorf("GetAndDelete, expected %v, got %v", want, val)
		}

		val, ok = tbl.GetAndDelete(p.pfx)
		if ok {
			t.Errorf("GetAndDelete, expected false, got (%v, %v)", val, ok)
		}
	}
}

func TestLiteGet(t *testing.T) {
	t.Parallel()

	t.Run("empty table", func(t *testing.T) {
		t.Parallel()

		rt := new(Table[int])
		pfx := randomPrefix()
		_, ok := rt.Get(pfx)

		if ok {
			t.Errorf("empty table: Get(%v), ok=%v, expected: %v", pfx, ok, false)
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

	rt := new(Table[int])
	for _, tt := range tests {
		rt.Insert(tt.pfx, tt.val)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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

func TestLiteGetCompare(t *testing.T) {
	t.Parallel()

	pfxs := randomPrefixes(10_000)
	fast := new(Table[int])
	gold := new(goldTable[int]).insertMany(pfxs)

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	for _, pfx := range pfxs {
		goldVal, goldOK := gold.get(pfx.pfx)
		fastVal, fastOK := fast.Get(pfx.pfx)

		if !getsEqual(goldVal, goldOK, fastVal, fastOK) {
			t.Fatalf("Get(%q) = (%v, %v), want (%v, %v)", pfx.pfx, fastVal, fastOK, goldVal, goldOK)
		}
	}
}

func TestLiteUnionEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		aTbl := new(Table[int])
		bTbl := new(Table[int])

		// union empty tables
		aTbl.Union(bTbl)

		want := ""
		got := aTbl.String()
		if got != want {
			t.Fatalf("got:\n%v\nwant:\n%v", got, want)
		}
	})

	t.Run("other empty", func(t *testing.T) {
		t.Parallel()
		aTbl := new(Table[int])
		bTbl := new(Table[int])

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
		t.Parallel()
		aTbl := new(Table[int])
		bTbl := new(Table[int])

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
		t.Parallel()
		aTbl := new(Table[string])
		bTbl := new(Table[string])

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
		t.Parallel()
		aTbl := new(Table[int])
		bTbl := new(Table[int])

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
		t.Parallel()
		aTbl := new(Table[int])
		bTbl := new(Table[int])

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
func TestLiteUnionMemoryAliasing(t *testing.T) {
	t.Parallel()

	newTable := func(pfx ...string) *Table[struct{}] {
		t := new(Table[struct{}])
		for _, s := range pfx {
			t.Insert(mpp(s), struct{}{})
		}
		return t
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

func TestLiteUnionCompare(t *testing.T) {
	t.Parallel()

	const numEntries = 200

	for range 100 {
		pfxs := randomPrefixes(numEntries)
		fast := new(Table[int])
		gold := new(goldTable[int]).insertMany(pfxs)

		for _, pfx := range pfxs {
			fast.Insert(pfx.pfx, pfx.val)
		}

		pfxs2 := randomPrefixes(numEntries)
		gold2 := new(goldTable[int]).insertMany(pfxs2)
		fast2 := new(Table[int])
		for _, pfx := range pfxs2 {
			fast2.Insert(pfx.pfx, pfx.val)
		}

		gold.union(gold2)
		fast.Union(fast2)

		// dump as slow table for comparison
		fastAsGoldenTbl := fast.dumpAsGoldTable()

		// sort for comparison
		gold.sort()
		fastAsGoldenTbl.sort()

		for i := range *gold {
			goldItem := (*gold)[i]
			fastItem := fastAsGoldenTbl[i]
			if goldItem != fastItem {
				t.Fatalf("Union(...): items[%d] differ slow(%v) != fast(%v)", i, goldItem, fastItem)
			}
		}

		// check the size
		if fast.Size() != len(*gold) {
			t.Errorf("sizes differ, got: %d, want: %d", fast.Size(), len(*gold))
		}
	}
}

func TestLiteCloneEdgeCases(t *testing.T) {
	t.Parallel()

	tbl := new(Table[int])
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

func TestLiteClone(t *testing.T) {
	t.Parallel()

	pfxs := randomPrefixes(100_000)

	golden := new(Table[int])
	tbl := new(Table[int])
	for _, pfx := range pfxs {
		golden.Insert(pfx.pfx, pfx.val)
		tbl.Insert(pfx.pfx, pfx.val)
	}
	clone := tbl.Clone()

	if golden.dumpString() != clone.dumpString() {
		t.Errorf("Clone: got:\n%swant:\n%s", clone.dumpString(), golden.dumpString())
	}

	if tbl.dumpString() != clone.dumpString() {
		t.Errorf("Clone: got:\n%swant:\n%s", clone.dumpString(), tbl.dumpString())
	}
}

func TestLiteCloneShallow(t *testing.T) {
	t.Parallel()

	tbl := new(Table[*int])
	clone := tbl.Clone()
	if tbl.dumpString() != clone.dumpString() {
		t.Errorf("empty Clone: got:\n%swant:\n%s", clone.String(), tbl.String())
	}

	val := 1
	pfx := mpp("10.0.0.1/32")
	tbl.Insert(pfx, &val)

	clone = tbl.Clone()
	want, _ := tbl.Get(pfx)
	got, _ := clone.Get(pfx)

	if !(*got == *want && got == want) {
		t.Errorf("shallow copy, values and pointers must be equal:\nvalues(%d, %d)\n(ptr(%v, %v)", *got, *want, got, want)
	}

	// update value, shallow copy of values, clone must be equal
	val = 2
	want, _ = tbl.Get(pfx)
	got, _ = clone.Get(pfx)

	if *got != *want {
		t.Errorf("memory aliasing after shallow copy, values must be equal:\nvalues(%d, %d)", *got, *want)
	}
}

func TestLiteCloneDeep(t *testing.T) {
	t.Parallel()

	tbl := new(Table[*MyInt])
	clone := tbl.Clone()
	if tbl.String() != clone.String() {
		t.Errorf("empty Clone: got:\n%swant:\n%s", clone.String(), tbl.String())
	}

	val := MyInt(1)
	pfx := mpp("10.0.0.1/32")
	tbl.Insert(pfx, &val)

	clone = tbl.Clone()
	want, _ := tbl.Get(pfx)
	got, _ := clone.Get(pfx)

	if *got != *want || got == want {
		t.Errorf("value with Cloner interface, pointers must be different:\nvalues(%d, %d)\n(ptr(%v, %v)", *got, *want, got, want)
	}

	// update value, deep copy of values, cloned value must now be different
	val = 2
	want, _ = tbl.Get(pfx)
	got, _ = clone.Get(pfx)

	if *got == *want {
		t.Errorf("memory aliasing after deep copy, values must be different:\nvalues(%d, %d)", *got, *want)
	}
}

func TestLiteUnionShallow(t *testing.T) {
	t.Parallel()

	tbl1 := new(Table[*int])
	tbl2 := new(Table[*int])

	val := 1
	pfx := mpp("10.0.0.1/32")
	tbl2.Insert(pfx, &val)

	tbl1.Union(tbl2)
	got, _ := tbl1.Get(pfx)
	want, _ := tbl2.Get(pfx)

	if !(*got == *want && got == want) {
		t.Errorf("shallow copy, values and pointers must be equal:\nvalues(%d, %d)\n(ptr(%v, %v)", *got, *want, got, want)
	}

	// update value, shallow copy of values, union must be equal
	val = 2
	got, _ = tbl1.Get(pfx)
	want, _ = tbl2.Get(pfx)

	if *got != *want {
		t.Errorf("memory aliasing after shallow copy, values must be equal:\nvalues(%d, %d)", *got, *want)
	}
}

func TestLiteUnionDeep(t *testing.T) {
	t.Parallel()

	tbl1 := new(Table[*MyInt])
	tbl2 := new(Table[*MyInt])

	val := MyInt(1)
	pfx := mpp("10.0.0.1/32")
	tbl2.Insert(pfx, &val)

	tbl1.Union(tbl2)
	got, _ := tbl1.Get(pfx)
	want, _ := tbl2.Get(pfx)

	if *got != *want || got == want {
		t.Errorf("value with Cloner interface, pointers must be different:\nvalues(%d, %d)\n(ptr(%v, %v)", *got, *want, got, want)
	}

	// update value, shallow copy of values, union must be equal
	val = 2
	got, _ = tbl1.Get(pfx)
	want, _ = tbl2.Get(pfx)

	if *got == *want {
		t.Errorf("memory aliasing after deep copy, values must be different:\nvalues(%d, %d)", *got, *want)
	}
}

// test some edge cases
func TestLiteOverlapsPrefixEdgeCases(t *testing.T) {
	t.Parallel()

	tbl := new(Table[int])

	// empty table
	checkOverlapsPrefix(t, tbl, []tableOverlapsTest{
		{"0.0.0.0/0", false},
		{"::/0", false},
	})

	// default route
	tbl.Insert(mpp("10.0.0.0/9"), 0)
	tbl.Insert(mpp("2001:db8::/32"), 0)
	checkOverlapsPrefix(t, tbl, []tableOverlapsTest{
		{"0.0.0.0/0", true},
		{"::/0", true},
	})

	// default route
	tbl = new(Table[int])
	tbl.Insert(mpp("0.0.0.0/0"), 0)
	tbl.Insert(mpp("::/0"), 0)
	checkOverlapsPrefix(t, tbl, []tableOverlapsTest{
		{"10.0.0.0/9", true},
		{"2001:db8::/32", true},
	})

	// single IP
	tbl = new(Table[int])
	tbl.Insert(mpp("10.0.0.0/7"), 0)
	tbl.Insert(mpp("2001::/16"), 0)
	checkOverlapsPrefix(t, tbl, []tableOverlapsTest{
		{"10.1.2.3/32", true},
		{"2001:db8:affe::cafe/128", true},
	})

	// single IP
	tbl = new(Table[int])
	tbl.Insert(mpp("10.1.2.3/32"), 0)
	tbl.Insert(mpp("2001:db8:affe::cafe/128"), 0)
	checkOverlapsPrefix(t, tbl, []tableOverlapsTest{
		{"10.0.0.0/7", true},
		{"2001::/16", true},
	})

	// same IPv
	tbl = new(Table[int])
	tbl.Insert(mpp("10.1.2.3/32"), 0)
	tbl.Insert(mpp("2001:db8:affe::cafe/128"), 0)
	checkOverlapsPrefix(t, tbl, []tableOverlapsTest{
		{"10.1.2.3/32", true},
		{"2001:db8:affe::cafe/128", true},
	})
}

func TestLiteSize(t *testing.T) {
	t.Parallel()

	tbl := new(Table[any])
	if tbl.Size() != 0 {
		t.Errorf("empty Table: want: 0, got: %d", tbl.Size())
	}

	if tbl.Size4() != 0 {
		t.Errorf("empty Table: want: 0, got: %d", tbl.Size4())
	}

	if tbl.Size6() != 0 {
		t.Errorf("empty Table: want: 0, got: %d", tbl.Size6())
	}

	pfxs1 := randomPrefixes(10_000)
	pfxs2 := randomPrefixes(10_000)

	for _, pfx := range pfxs1 {
		tbl.Insert(pfx.pfx, nil)
	}

	for _, pfx := range pfxs2 {
		tbl.Update(pfx.pfx, func(any, bool) any { return nil })
	}

	pfxs1 = append(pfxs1, pfxs2...)

	for _, pfx := range pfxs1[:1_000] {
		tbl.Update(pfx.pfx, func(any, bool) any { return nil })
	}

	for _, pfx := range randomPrefixes(20_000) {
		tbl.Delete(pfx.pfx)
	}

	var allInc4 int
	var allInc6 int

	tbl.AllSorted4()(func(netip.Prefix, any) bool {
		allInc4++
		return true
	})

	tbl.AllSorted6()(func(netip.Prefix, any) bool {
		allInc6++
		return true
	})

	if allInc4 != tbl.Size4() {
		t.Errorf("Size4: want: %d, got: %d", allInc4, tbl.Size4())
	}

	if allInc6 != tbl.Size6() {
		t.Errorf("Size6: want: %d, got: %d", allInc6, tbl.Size6())
	}
}

func checkLiteNumNodes(t *testing.T, tbl *Lite, want int) {
	t.Helper()

	s4 := tbl.root4.nodeStatsRec()
	s6 := tbl.root6.nodeStatsRec()
	nodes := s4.nodes + s6.nodes

	if got := nodes; got != want {
		t.Errorf("wrong table dump, got %d nodes want %d", got, want)
		t.Error(tbl.dumpString())
	}
}
