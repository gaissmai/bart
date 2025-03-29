// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"math/rand"
	"net/netip"
	"testing"
)

// ############ tests ################################

func TestLiteDeprecated(t *testing.T) {
	t.Parallel()

	var testname string

	testname = "Update"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r == nil {
				t.Fatalf("Lite.%s is deprecated, should panic", testname)
			}
		}(testname)

		tbl := new(Lite)
		tbl.Update()
	})

	testname = "UpdatePersist"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r == nil {
				t.Fatalf("Lite.%s is deprecated, should panic", testname)
			}
		}(testname)

		tbl := new(Lite)
		tbl.UpdatePersist()
	})

	testname = "GetAndDelete"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r == nil {
				t.Fatalf("Lite.%s is deprecated, should panic", testname)
			}
		}(testname)

		tbl := new(Lite)
		tbl.GetAndDelete()
	})

	testname = "GetAndDeletePersist"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r == nil {
				t.Fatalf("Lite.%s is deprecated, should panic", testname)
			}
		}(testname)

		tbl := new(Lite)
		tbl.GetAndDeletePersist()
	})
}

func TestLiteInvalid(t *testing.T) {
	t.Parallel()

	tbl1 := new(Lite)
	var zeroPfx netip.Prefix
	var zeroIP netip.Addr
	var testname string

	testname = "Exists"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid prefix input", testname)
			}
		}(testname)

		tbl1.Exists(zeroPfx)
	})

	testname = "Insert"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid prefix input", testname)
			}
		}(testname)

		tbl1.Insert(zeroPfx)
	})

	testname = "InsertPersist"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid prefix input", testname)
			}
		}(testname)

		_ = tbl1.InsertPersist(zeroPfx)
	})

	testname = "Delete"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid prefix input", testname)
			}
		}(testname)

		tbl1.Delete(zeroPfx)
	})

	testname = "DeletePersist"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid prefix input", testname)
			}
		}(testname)

		_ = tbl1.DeletePersist(zeroPfx)
	})

	testname = "Contains"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		if tbl1.Contains(zeroIP) != false {
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

		tbl1.LookupPrefix(zeroPfx)
	})

	testname = "LookupPrefixLPM"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid prefix input", testname)
			}
		}(testname)

		tbl1.LookupPrefixLPM(zeroPfx)
	})

	testname = "OverlapsPrefix"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid prefix input", testname)
			}
		}(testname)

		tbl1.OverlapsPrefix(zeroPfx)
	})

	testname = "Overlaps"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on empty table", testname)
			}
		}(testname)

		tbl2 := new(Lite)
		tbl1.Overlaps(tbl2)
		tbl1.Overlaps4(tbl2)
		tbl1.Overlaps6(tbl2)
	})

	testname = "Contains"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid ip input", testname)
			}
		}(testname)

		tbl1.Contains(zeroIP)
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
	fast := new(Lite)

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx)
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
	pfxs := randomPrefixes(10_000)

	fast := new(Lite)
	gold := new(goldTable[int]).insertMany(pfxs)

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx)
	}

	for range 10_000 {
		pfx := randomPrefix()

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
	pfxs := randomPrefixes(10_000)

	fast := new(Lite)
	gold := new(goldTable[int]).insertMany(pfxs)

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx)
	}

	for range 10_000 {
		pfx := randomPrefix()

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

	pfxs := randomPrefixes(1000)

	for range 10 {
		pfxs2 := append([]goldTableItem[int](nil), pfxs...)
		rand.Shuffle(len(pfxs2), func(i, j int) { pfxs2[i], pfxs2[j] = pfxs2[j], pfxs2[i] })

		addrs := make([]netip.Addr, 0, 10_000)
		for range 10_000 {
			addrs = append(addrs, randomAddr())
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

	fast := new(Lite)
	gold := new(goldTable[int]).insertMany(pfxs)

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
		a := randomAddr()

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
		rand.Shuffle(len(toDelete2), func(i, j int) { toDelete2[i], toDelete2[j] = toDelete2[j], toDelete2[i] })
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
	// Insert N prefixes, then delete those same prefixes in reverse
	// order. Each deletion should exactly undo the internal structure
	// changes that each insert did.
	const N = 10_000

	tbl := new(Lite)
	want := tbl.dumpString()

	prefixes := randomPrefixes(N)

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

	pfxs := randomPrefixes(100_000)

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

	for range 10 {
		t.Run("Union", func(t *testing.T) {
			t.Parallel()
			pfx1 := randomRealWorldPrefixes(1_000)
			pfx2 := randomRealWorldPrefixes(2_000)

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
				t.Errorf("Union: got:\n%swant:\n%s", tbl1.dumpString(), golden.dumpString())
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
