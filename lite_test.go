// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"fmt"
	"math/rand/v2"
	"net/netip"
	"testing"
)

// ############ tests ################################

func TestLiteInvalid(t *testing.T) {
	t.Parallel()

	tbl1 := new(Lite)
	tbl2 := new(Lite)

	var zeroPfx netip.Prefix
	var zeroIP netip.Addr

	noPanic(t, "Contains", func() { tbl1.Contains(zeroIP) })
	noPanic(t, "Lookup", func() { tbl1.Lookup(zeroIP) })

	noPanic(t, "LookupPrefix", func() { tbl1.LookupPrefix(zeroPfx) })
	noPanic(t, "LookupPrefixLPM", func() { tbl1.LookupPrefixLPM(zeroPfx) })

	noPanic(t, "Exists", func() { tbl1.Exists(zeroPfx) })
	noPanic(t, "Insert", func() { tbl1.Insert(zeroPfx) })
	noPanic(t, "Delete", func() { tbl1.Delete(zeroPfx) })
	noPanic(t, "InsertPersist", func() { tbl1.InsertPersist(zeroPfx) })
	noPanic(t, "DeletePersist", func() { tbl1.DeletePersist(zeroPfx) })

	noPanic(t, "OverlapsPrefix", func() { tbl1.OverlapsPrefix(zeroPfx) })

	noPanic(t, "Overlaps", func() { tbl1.Overlaps(tbl2) })
	noPanic(t, "Overlaps4", func() { tbl1.Overlaps4(tbl2) })
	noPanic(t, "Overlaps6", func() { tbl1.Overlaps6(tbl2) })
}

func TestLiteDeletePersist(t *testing.T) {
	t.Parallel()

	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))
	t.Run("table_is_empty", func(t *testing.T) {
		t.Parallel()
		// must not panic
		tbl := new(Lite)
		checkLiteNumNodes(t, tbl, 0)
		tbl, _ = tbl.DeletePersist(randomPrefix(prng))
		checkLiteNumNodes(t, tbl, 0)
	})

	t.Run("prefix_in_root", func(t *testing.T) {
		t.Parallel()
		// Add/remove prefix from root table.
		tbl := new(Lite)
		checkLiteNumNodes(t, tbl, 0)

		tbl.Insert(mpp("10.0.0.0/8"))
		checkLiteNumNodes(t, tbl, 1)
		tbl, _ = tbl.DeletePersist(mpp("10.0.0.0/8"))
		checkLiteNumNodes(t, tbl, 0)
	})

	t.Run("prefix_in_leaf", func(t *testing.T) {
		t.Parallel()
		// Create, then delete a single leaf table.
		tbl := new(Lite)
		checkLiteNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"))
		checkLiteNumNodes(t, tbl, 1)

		tbl, _ = tbl.DeletePersist(mpp("192.168.0.1/32"))
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

		tbl, _ = tbl.DeletePersist(mpp("192.180.0.1/32"))
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

		tbl, _ = tbl.DeletePersist(mpp("192.180.0.1/32"))
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

		tbl, _ = tbl.DeletePersist(mpp("192.180.0.1/32"))
		checkLiteNumNodes(t, tbl, 2)
	})

	t.Run("nosuchprefix_missing_child", func(t *testing.T) {
		t.Parallel()
		// Delete non-existent prefix
		tbl := new(Lite)
		checkLiteNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"))
		checkLiteNumNodes(t, tbl, 1)

		tbl, _ = tbl.DeletePersist(mpp("200.0.0.0/32"))
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

		tbl, _ = tbl.DeletePersist(mpp("192.168.0.0/22"))
		checkLiteNumNodes(t, tbl, 1)
	})

	t.Run("default_route", func(t *testing.T) {
		t.Parallel()
		tbl := new(Lite)
		checkLiteNumNodes(t, tbl, 0)

		tbl.Insert(mpp("0.0.0.0/0"))
		tbl.Insert(mpp("::/0"))
		tbl, _ = tbl.DeletePersist(mpp("0.0.0.0/0"))

		checkLiteNumNodes(t, tbl, 1)
	})

	t.Run("path compressed purge", func(t *testing.T) {
		t.Parallel()
		tbl := new(Lite)
		checkLiteNumNodes(t, tbl, 0)

		tbl.Insert(mpp("10.10.0.0/17"))
		tbl.Insert(mpp("10.20.0.0/17"))
		checkLiteNumNodes(t, tbl, 2)

		tbl, _ = tbl.DeletePersist(mpp("10.20.0.0/17"))
		checkLiteNumNodes(t, tbl, 1)

		tbl, _ = tbl.DeletePersist(mpp("10.10.0.0/17"))
		checkLiteNumNodes(t, tbl, 0)
	})
}

func TestLiteContainsCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := 10_000
	if testing.Short() {
		n = 1_000
	}

	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, n)

	gold := new(goldTable[int])
	gold.insertMany(pfxs)

	fast := new(Lite)
	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx)
	}

	for range n {
		a := randomAddr(prng)

		_, goldOK := gold.lookup(a)
		fastOK := fast.Contains(a)

		if goldOK != fastOK {
			t.Fatalf("Contains(%q) = %v, want %v", a, fastOK, goldOK)
		}
	}
}

func TestLiteEqual(t *testing.T) {
	t.Parallel()
	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))

	count := 100_000
	if testing.Short() {
		count = 10_000
	}

	rt := new(Lite)
	for _, pfx := range randomRealWorldPrefixes(prng, count) {
		rt.Insert(pfx)
	}

	ct := rt.Clone()
	if !rt.Equal(ct) {
		t.Error("expected true, got false")
	}
}

func TestLiteLookupPrefixUnmasked(t *testing.T) {
	// test that the pfx must not be masked on input for LookupPrefix
	t.Parallel()

	rt := new(Lite)
	rt.Insert(mpp("10.20.30.0/24"))

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

	n := 10_000
	if testing.Short() {
		n = 1_000
	}

	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, n)

	gold := new(goldTable[int])
	gold.insertMany(pfxs)

	fast := new(Lite)
	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx)
	}

	for range n {
		pfx := randomPrefix(prng)

		_, goldOK := gold.lookupPfx(pfx)
		_, fastOK := fast.LookupPrefix(pfx)

		if goldOK != fastOK {
			t.Fatalf("LookupPrefix(%q) = %v, want %v", pfx, fastOK, goldOK)
		}

	}
}

func TestLiteLookupPrefixLPMCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()
	n := 10_000
	if testing.Short() {
		n = 1_000
	}

	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, n)

	gold := new(goldTable[int])
	gold.insertMany(pfxs)

	fast := new(Lite)
	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx)
	}

	for range n {
		pfx := randomPrefix(prng)

		goldLPM, _, goldOK := gold.lookupPfxLPM(pfx)
		fastLPM, _, fastOK := fast.LookupPrefixLPM(pfx)

		if !getsEqual(goldLPM, goldOK, fastLPM, fastOK) {
			t.Fatalf("LookupPrefixLPM(%q) = (%v, %v), want (%v, %v)", pfx, fastLPM, fastOK, goldLPM, goldOK)
		}

	}
}

func TestLiteInsertPersistShuffled(t *testing.T) {
	// The order in which you insert prefixes into a route table
	// should not matter, as long as you're inserting the same set of
	// routes.
	t.Parallel()

	n := 10_000
	if testing.Short() {
		n = 1_000
	}

	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, n)

	for range 10 {
		pfxs2 := append([]goldTableItem[int](nil), pfxs...)
		prng.Shuffle(len(pfxs2), func(i, j int) { pfxs2[i], pfxs2[j] = pfxs2[j], pfxs2[i] })

		addrs := make([]netip.Addr, 0, n)
		for range n {
			addrs = append(addrs, randomAddr(prng))
		}

		rt1 := new(Lite)
		rt2 := new(Lite)

		// rt1 is mutable
		for _, pfx := range pfxs {
			rt1.Insert(pfx.pfx)
		}

		// rt2 is persistent
		for _, pfx := range pfxs2 {
			rt2 = rt2.InsertPersist(pfx.pfx)
		}

		if rt1.String() != rt2.String() {
			t.Fatal("mutable and immutable table have different string representation")
		}

		if rt1.dumpString() != rt2.dumpString() {
			t.Fatal("mutable and immutable table have different dumpString representation")
		}

		for _, a := range addrs {
			ok1 := rt1.Contains(a)
			ok2 := rt2.Contains(a)

			if ok1 != ok2 {
				t.Fatalf("Contains(%q) = %v, want %v", a, ok2, ok1)
			}
		}
	}
}

func TestLiteDeleteCompare(t *testing.T) {
	// Create large route tables repeatedly, delete half of their
	// prefixes, and compare Table's behavior to a naive and slow but
	// correct implementation.
	t.Parallel()
	n := 10_000
	if testing.Short() {
		n = 1_000
	}

	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))

	var (
		numPrefixes  = n // total prefixes to insert (test deletes 50% of them)
		numPerFamily = numPrefixes / 2
		deleteCut    = numPerFamily / 2
		numProbes    = n // random addr lookups to do
	)

	// We have to do this little dance instead of just using allPrefixes,
	// because we want pfxs and toDelete to be non-overlapping sets.
	all4, all6 := randomPrefixes4(prng, numPerFamily), randomPrefixes6(prng, numPerFamily)

	pfxs := append([]goldTableItem[int](nil), all4[:deleteCut]...)
	pfxs = append(pfxs, all6[:deleteCut]...)

	toDelete := append([]goldTableItem[int](nil), all4[deleteCut:]...)
	toDelete = append(toDelete, all6[deleteCut:]...)

	gold := new(goldTable[int])
	gold.insertMany(pfxs)

	fast := new(Lite)
	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx)
	}

	for _, pfx := range toDelete {
		fast.Insert(pfx.pfx)
	}
	for _, pfx := range toDelete {
		fast.Delete(pfx.pfx)
	}

	for range numProbes {
		a := randomAddr(prng)

		_, goldOK := gold.lookup(a)
		fastOK := fast.Contains(a)

		if goldOK != fastOK {
			t.Fatalf("Contains(%q) = %v, want %v", a, fastOK, goldOK)
		}
	}
}

func TestLiteDeleteShuffled(t *testing.T) {
	// The order in which you delete prefixes from a route table
	// should not matter, as long as you're deleting the same set of
	// routes.
	t.Parallel()
	n := 10_000
	if testing.Short() {
		n = 1_000
	}

	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))

	var (
		numPrefixes  = n // prefixes to insert (test deletes 50% of them)
		numPerFamily = numPrefixes / 2
		deleteCut    = numPerFamily / 2
	)

	// We have to do this little dance instead of just using allPrefixes,
	// because we want pfxs and toDelete to be non-overlapping sets.
	all4, all6 := randomPrefixes4(prng, numPerFamily), randomPrefixes6(prng, numPerFamily)

	pfxs := append([]goldTableItem[int](nil), all4[:deleteCut]...)
	pfxs = append(pfxs, all6[:deleteCut]...)

	toDelete := append([]goldTableItem[int](nil), all4[deleteCut:]...)
	toDelete = append(toDelete, all6[deleteCut:]...)

	rt1 := new(Lite)
	for _, pfx := range pfxs {
		rt1.Insert(pfx.pfx)
	}
	for _, pfx := range toDelete {
		rt1.Insert(pfx.pfx)
	}
	for _, pfx := range toDelete {
		rt1.Delete(pfx.pfx)
	}

	for range 10 {
		pfxs2 := append([]goldTableItem[int](nil), pfxs...)
		toDelete2 := append([]goldTableItem[int](nil), toDelete...)
		prng.Shuffle(len(toDelete2), func(i, j int) { toDelete2[i], toDelete2[j] = toDelete2[j], toDelete2[i] })
		rt2 := new(Lite)
		for _, pfx := range pfxs2 {
			rt2.Insert(pfx.pfx)
		}
		for _, pfx := range toDelete2 {
			rt2.Insert(pfx.pfx)
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
	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))
	// Insert count prefixes, then delete those same prefixes in reverse
	// order. Each deletion should exactly undo the internal structure
	// changes that each insert did.

	count := 10_000
	if testing.Short() {
		count = 1_000
	}

	tbl := new(Lite)
	want := tbl.dumpString()

	prefixes := randomPrefixes(prng, count)

	defer func() {
		if t.Failed() {
			t.Logf("the prefixes that fail the test: %v\n", prefixes)
		}
	}()

	for _, p := range prefixes {
		tbl.Insert(p.pfx)
	}

	for i := len(prefixes) - 1; i >= 0; i-- {
		tbl.Delete(prefixes[i].pfx)
	}
	if got := tbl.dumpString(); got != want {
		t.Fatalf("after delete, mismatch:\n\n got: %s\n\nwant: %s", got, want)
	}
}

func TestLiteClone(t *testing.T) {
	t.Parallel()
	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))

	count := 10_000
	if testing.Short() {
		count = 1_000
	}

	pfxs := randomPrefixes(prng, count)

	golden := new(Lite)
	tbl := new(Lite)
	for _, pfx := range pfxs {
		golden.Insert(pfx.pfx)
		tbl.Insert(pfx.pfx)
	}
	clone := tbl.Clone()

	if golden.dumpString() != clone.dumpString() {
		t.Errorf("Clone: got:\n%swant:\n%s", clone.dumpString(), golden.dumpString())
	}

	if tbl.dumpString() != clone.dumpString() {
		t.Errorf("Clone: got:\n%swant:\n%s", clone.dumpString(), tbl.dumpString())
	}
}

func TestLiteUnion(t *testing.T) {
	t.Parallel()

	for i := range 10 {
		t.Run(fmt.Sprintf("Union-%d", i), func(t *testing.T) {
			t.Parallel()
			//nolint:gosec
			prng := rand.New(rand.NewPCG(42, 42))
			pfx1 := randomRealWorldPrefixes(prng, 1_000)
			pfx2 := randomRealWorldPrefixes(prng, 2_000)

			golden := new(Lite)
			for _, pfx := range append(pfx1, pfx2...) {
				golden.Insert(pfx)
			}

			tbl1 := new(Lite)
			for _, pfx := range pfx1 {
				tbl1.Insert(pfx)
			}

			tbl2 := new(Lite)
			for _, pfx := range pfx2 {
				tbl2.Insert(pfx)
			}

			tbl1.Union(tbl2)

			if tbl1.dumpString() != golden.dumpString() {
				t.Errorf("got:\n%swant:\n%s", tbl1.dumpString(), golden.dumpString())
			}
		})
	}
}

func TestLiteUnionPersist(t *testing.T) {
	t.Parallel()

	for i := range 10 {
		t.Run(fmt.Sprintf("UnionPersist-%d", i), func(t *testing.T) {
			t.Parallel()
			//nolint:gosec
			prng := rand.New(rand.NewPCG(42, 42))
			pfx1 := randomRealWorldPrefixes(prng, 1_000)
			pfx2 := randomRealWorldPrefixes(prng, 2_000)

			golden := new(Lite)
			for _, pfx := range append(pfx1, pfx2...) {
				golden.Insert(pfx)
			}

			tbl1 := new(Lite)
			for _, pfx := range pfx1 {
				tbl1.Insert(pfx)
			}

			tbl2 := new(Lite)
			for _, pfx := range pfx2 {
				tbl2.Insert(pfx)
			}

			pTbl := tbl1.UnionPersist(tbl2)

			if pTbl.dumpString() != golden.dumpString() {
				t.Errorf("got:\n%swant:\n%s", pTbl.dumpString(), golden.dumpString())
			}
		})
	}
}

func TestLiteStringEmpty(t *testing.T) {
	t.Parallel()
	tbl := new(Lite)
	want := ""
	got := tbl.String()
	if got != want {
		t.Errorf("table is nil, expected %q, got %q", want, got)
	}
}

func TestLiteStringDefaultRouteV4(t *testing.T) {
	t.Parallel()

	tt := stringTest{
		cidrs: []netip.Prefix{
			mpp("0.0.0.0/0"),
		},
		want: `▼
└─ 0.0.0.0/0
`,
	}

	tbl := new(Lite)
	checkLiteString(t, tbl, tt)
}

func TestLiteStringDefaultRouteV6(t *testing.T) {
	t.Parallel()

	tt := stringTest{
		cidrs: []netip.Prefix{
			mpp("::/0"),
		},
		want: `▼
└─ ::/0
`,
	}

	tbl := new(Lite)
	checkLiteString(t, tbl, tt)
}

func TestLiteStringSample(t *testing.T) {
	t.Parallel()

	tt := stringTest{
		cidrs: []netip.Prefix{
			mpp("fe80::/10"),
			mpp("172.16.0.0/12"),
			mpp("10.0.0.0/24"),
			mpp("::1/128"),
			mpp("192.168.0.0/16"),
			mpp("10.0.0.0/8"),
			mpp("::/0"),
			mpp("10.0.1.0/24"),
			mpp("169.254.0.0/16"),
			mpp("2000::/3"),
			mpp("2001:db8::/32"),
			mpp("127.0.0.0/8"),
			mpp("127.0.0.1/32"),
			mpp("192.168.1.0/24"),
		},
		want: `▼
├─ 10.0.0.0/8
│  ├─ 10.0.0.0/24
│  └─ 10.0.1.0/24
├─ 127.0.0.0/8
│  └─ 127.0.0.1/32
├─ 169.254.0.0/16
├─ 172.16.0.0/12
└─ 192.168.0.0/16
   └─ 192.168.1.0/24
▼
└─ ::/0
   ├─ ::1/128
   ├─ 2000::/3
   │  └─ 2001:db8::/32
   └─ fe80::/10
`,
	}

	tbl := new(Lite)
	checkLiteString(t, tbl, tt)
}

func TestLiteWalkPersist(t *testing.T) {
	type testCase struct {
		name       string
		input      []string
		fn         func(*Lite, netip.Prefix) (*Lite, bool)
		wantRemain []string
	}

	tests := []testCase{
		{
			name: "delete nothing",
			input: []string{
				"192.168.0.0/16",
				"2001:db8::/32",
			},
			fn: func(l *Lite, pfx netip.Prefix) (*Lite, bool) {
				return l, false // early exit
			},
			wantRemain: []string{"192.168.0.0/16", "2001:db8::/32"},
		},
		{
			name: "delete all",
			input: []string{
				"10.0.0.0/8",
				"fd00::/8",
			},
			fn: func(pl *Lite, pfx netip.Prefix) (*Lite, bool) {
				prt, _ := pl.DeletePersist(pfx)
				return prt, true // remove everything
			},
			wantRemain: []string{},
		},
		{
			name: "delete only IPv4",
			input: []string{
				"172.16.0.0/12",
				"2001:db8:1::/48",
			},
			fn: func(pl *Lite, pfx netip.Prefix) (*Lite, bool) {
				if pfx.Addr().Is4() {
					pl, _ = pl.DeletePersist(pfx)
				}
				return pl, true
			},
			wantRemain: []string{"2001:db8:1::/48"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Build initial table.
			tbl := new(Lite)
			for _, pfx := range tc.input {
				tbl.Insert(mpp(pfx))
			}

			// Apply FilterPersist.
			got := tbl.WalkPersist(tc.fn)

			// Collect remaining prefixes from result.
			gotRemain := []string{}
			for pfx := range got.All() {
				gotRemain = append(gotRemain, pfx.String())
			}

			// Compare lengths.
			if len(gotRemain) != len(tc.wantRemain) {
				t.Fatalf("expected %d entries, got %d: %v", len(tc.wantRemain), len(gotRemain), gotRemain)
			}

			// Compare sets (order is not guaranteed).
			wantMap := map[string]bool{}
			for _, w := range tc.wantRemain {
				wantMap[w] = true
			}
			for _, g := range gotRemain {
				if !wantMap[g] {
					t.Errorf("unexpected remaining prefix: %s", g)
				}
			}
		})
	}
}

// ###################################################################

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

func checkLiteString(t *testing.T, tbl *Lite, tt stringTest) {
	t.Helper()

	for _, cidr := range tt.cidrs {
		tbl.Insert(cidr)
	}

	got := tbl.String()
	if tt.want != got {
		t.Errorf("String got:\n%swant:\n%s", got, tt.want)
	}

	gotBytes, err := tbl.MarshalText()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if tt.want != string(gotBytes) {
		t.Errorf("MarshalText got:\n%swant:\n%s", gotBytes, tt.want)
	}
}
