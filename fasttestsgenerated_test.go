// Code generated from file "tests_tmpl.go"; DO NOT EDIT.

// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"math/rand/v2"
	"net/netip"
	"slices"
	"testing"
)

// ############ tests ################################

// flatSorted, just a helper to compare with golden table.
func (t *Fast[V]) flatSorted() goldTable[V] {
	var flat goldTable[V]

	for p, v := range t.AllSorted() {
		flat = append(flat, goldTableItem[V]{pfx: p, val: v})
	}

	return flat
}

func TestTableNil_Fast(t *testing.T) {
	t.Parallel()

	ip4 := mpa("127.0.0.1")
	ip6 := mpa("::1")

	pfx4 := mpp("127.0.0.0/8")
	pfx6 := mpp("::1/128")

	tbl2 := new(Fast[any])
	tbl2.Insert(pfx4, nil)
	tbl2.Insert(pfx6, nil)

	var tbl1 *Fast[any] = nil

	t.Run("mustPanic", func(t *testing.T) {
		t.Parallel()

		mustPanic(t, "sizeUpdate", func() { tbl1.sizeUpdate(true, 1) })
		mustPanic(t, "sizeUpdate", func() { tbl1.sizeUpdate(false, 1) })
		mustPanic(t, "rootNodeByVersion", func() { tbl1.rootNodeByVersion(true) })
		mustPanic(t, "rootNodeByVersion", func() { tbl1.rootNodeByVersion(false) })
		mustPanic(t, "fprint", func() { tbl1.fprint(nil, true) })
		mustPanic(t, "fprint", func() { tbl1.fprint(nil, false) })

		mustPanic(t, "Size", func() { tbl1.Size() })
		mustPanic(t, "Size4", func() { tbl1.Size4() })
		mustPanic(t, "Size6", func() { tbl1.Size6() })

		mustPanic(t, "Get", func() { tbl1.Get(pfx4) })
		mustPanic(t, "Insert", func() { tbl1.Insert(pfx4, nil) })
		mustPanic(t, "InsertPersist", func() { tbl1.InsertPersist(pfx4, nil) })
		mustPanic(t, "Delete", func() { tbl1.Delete(pfx4) })
		mustPanic(t, "DeletePersist", func() { tbl1.DeletePersist(pfx4) })
		mustPanic(t, "Modify", func() { tbl1.Modify(pfx4, nil) })
		mustPanic(t, "ModifyPersist", func() { tbl1.ModifyPersist(pfx4, nil) })
		mustPanic(t, "Contains", func() { tbl1.Contains(ip4) })
		mustPanic(t, "Lookup", func() { tbl1.Lookup(ip6) })
		mustPanic(t, "LookupPrefix", func() { tbl1.LookupPrefix(pfx4) })
		mustPanic(t, "LookupPrefixLPM", func() { tbl1.LookupPrefixLPM(pfx4) })
		mustPanic(t, "Union", func() { tbl1.Union(tbl2) })
		mustPanic(t, "UnionPersist", func() { tbl1.UnionPersist(tbl2) })
	})

	t.Run("noPanic", func(t *testing.T) {
		t.Parallel()

		noPanic(t, "Overlaps", func() { tbl1.Overlaps(nil) })
		noPanic(t, "Overlaps4", func() { tbl1.Overlaps4(nil) })
		noPanic(t, "Overlaps6", func() { tbl1.Overlaps6(nil) })

		noPanic(t, "Overlaps", func() { tbl2.Overlaps(tbl2) })
		noPanic(t, "Overlaps4", func() { tbl2.Overlaps4(tbl2) })
		noPanic(t, "Overlaps6", func() { tbl2.Overlaps6(tbl2) })

		mustPanic(t, "Overlaps", func() { tbl1.Overlaps(tbl2) })
		mustPanic(t, "Overlaps4", func() { tbl1.Overlaps4(tbl2) })
		mustPanic(t, "Overlaps6", func() { tbl1.Overlaps6(tbl2) })
		mustPanic(t, "OverlapsPrefix", func() { tbl1.OverlapsPrefix(pfx4) })
		mustPanic(t, "OverlapsPrefix", func() { tbl1.OverlapsPrefix(pfx6) })

		mustPanic(t, "Equal", func() { tbl1.Equal(tbl2) })
		noPanic(t, "Equal", func() { tbl1.Equal(tbl1) })
		noPanic(t, "Equal", func() { tbl2.Equal(tbl2) })

		noPanic(t, "dump", func() { tbl1.dump(nil) })
		noPanic(t, "dumpString", func() { tbl1.dumpString() })
		noPanic(t, "Clone", func() { tbl1.Clone() })
		noPanic(t, "DumpList4", func() { tbl1.DumpList4() })
		noPanic(t, "DumpList6", func() { tbl1.DumpList6() })
		noPanic(t, "Fprint", func() { tbl1.Fprint(nil) })
		noPanic(t, "MarshalJSON", func() { _, _ = tbl1.MarshalJSON() })
		noPanic(t, "MarshalText", func() { _, _ = tbl1.MarshalText() })
	})

	t.Run("noPanicRangeOverFunc", func(t *testing.T) {
		t.Parallel()

		noPanicRangeOverFunc[any](t, "All", tbl1.All)
		noPanicRangeOverFunc[any](t, "All4", tbl1.All4)
		noPanicRangeOverFunc[any](t, "All6", tbl1.All6)
		noPanicRangeOverFunc[any](t, "AllSorted", tbl1.AllSorted)
		noPanicRangeOverFunc[any](t, "AllSorted4", tbl1.AllSorted4)
		noPanicRangeOverFunc[any](t, "AllSorted6", tbl1.AllSorted6)
		noPanicRangeOverFunc[any](t, "Subnets", tbl1.Subnets)
		noPanicRangeOverFunc[any](t, "Supernets", tbl1.Supernets)
	})
}

func TestTableInvalid_Fast(t *testing.T) {
	t.Parallel()

	tbl1 := new(Fast[any])
	tbl2 := new(Fast[any])

	var zeroIP netip.Addr
	var zeroPfx netip.Prefix

	noPanic(t, "All", func() { tbl1.All() })
	noPanic(t, "All4", func() { tbl1.All4() })
	noPanic(t, "All6", func() { tbl1.All6() })
	noPanic(t, "AllSorted", func() { tbl1.AllSorted() })
	noPanic(t, "AllSorted4", func() { tbl1.AllSorted4() })
	noPanic(t, "AllSorted6", func() { tbl1.AllSorted6() })
	noPanic(t, "Clone", func() { tbl1.Clone() })
	noPanic(t, "Contains", func() { tbl1.Contains(zeroIP) })
	noPanic(t, "Delete", func() { tbl1.Delete(zeroPfx) })
	noPanic(t, "DeletePersist", func() { tbl1.DeletePersist(zeroPfx) })
	noPanic(t, "DumpList4", func() { tbl1.DumpList4() })
	noPanic(t, "DumpList6", func() { tbl1.DumpList6() })
	noPanic(t, "Equal", func() { tbl1.Equal(tbl2) })
	noPanic(t, "Fprint", func() { tbl1.Fprint(nil) })
	noPanic(t, "Get", func() { tbl1.Get(zeroPfx) })
	noPanic(t, "Insert", func() { tbl1.Insert(zeroPfx, nil) })
	noPanic(t, "InsertPersist", func() { tbl1.InsertPersist(zeroPfx, nil) })
	noPanic(t, "Lookup", func() { tbl1.Lookup(zeroIP) })
	noPanic(t, "LookupPrefix", func() { tbl1.LookupPrefix(zeroPfx) })
	noPanic(t, "LookupPrefixLPM", func() { tbl1.LookupPrefixLPM(zeroPfx) })
	noPanic(t, "MarshalJSON", func() { _, _ = tbl1.MarshalJSON() })
	noPanic(t, "MarshalText", func() { _, _ = tbl1.MarshalText() })
	noPanic(t, "Modify", func() { tbl1.Modify(zeroPfx, nil) })
	noPanic(t, "ModifyPersist", func() { tbl1.ModifyPersist(zeroPfx, nil) })
	noPanic(t, "Overlaps", func() { tbl1.Overlaps(tbl2) })
	noPanic(t, "Overlaps4", func() { tbl1.Overlaps4(tbl2) })
	noPanic(t, "Overlaps6", func() { tbl1.Overlaps6(tbl2) })
	noPanic(t, "OverlapsPrefix", func() { tbl1.OverlapsPrefix(zeroPfx) })
	noPanic(t, "Size", func() { tbl1.Size() })
	noPanic(t, "Size4", func() { tbl1.Size4() })
	noPanic(t, "Size6", func() { tbl1.Size6() })
	noPanic(t, "Subnets", func() { tbl1.Subnets(zeroPfx) })
	noPanic(t, "Supernets", func() { tbl1.Supernets(zeroPfx) })
	noPanic(t, "Union", func() { tbl1.Union(tbl2) })
	noPanic(t, "UnionPersist", func() { tbl1.UnionPersist(tbl2) })
}

func TestTableContainsCompare_Fast(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[int])
	tbl := new(Fast[int])

	for i, p := range pfxs {
		gold.insert(p, i)
		tbl.Insert(p, i)
	}

	for range n {
		a := randomAddr(prng)

		_, goldOK := gold.lookup(a)
		tblOK := tbl.Contains(a)

		if goldOK != tblOK {
			t.Fatalf("Contains(%q) = %v, want %v", a, tblOK, goldOK)
		}
	}
}

func TestTableLookupCompare_Fast(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[int])
	tbl := new(Fast[int])

	for i, pfx := range pfxs {
		gold.insert(pfx, i)
		tbl.Insert(pfx, i)
	}

	for range n {
		a := randomAddr(prng)

		goldVal, goldOK := gold.lookup(a)
		tblVal, tblOK := tbl.Lookup(a)

		if goldOK != tblOK {
			t.Fatalf("Lookup(%q) = (_, %v), want %v", a, tblOK, goldOK)
		}

		// Skip value comparison for liteTable (no real payload)
		if _, isLite := any(tbl).(*liteTable[int]); !isLite {
			if goldVal != tblVal {
				t.Fatalf("Lookup(%q) = (%v, %v), want (%v, %v)", a, tblVal, tblOK, goldVal, goldOK)
			}
		}
	}
}

func TestTableLookupPrefixUnmasked_Fast(t *testing.T) {
	// test that the pfx must not be masked on input for LookupPrefix
	t.Parallel()

	tbl := new(Fast[any])
	tbl.Insert(mpp("10.20.30.0/24"), nil)
	tbl.Insert(mpp("2001:db8::/32"), nil)

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
		// IPv6 counterparts
		{
			probe:   netip.MustParsePrefix("2001:db8::1/0"),
			wantLPM: netip.Prefix{},
			wantOk:  false,
		},
		{
			probe:   netip.MustParsePrefix("2001:db8::1/31"),
			wantLPM: netip.Prefix{},
			wantOk:  false,
		},
		{
			probe:   netip.MustParsePrefix("2001:db8::1/32"),
			wantLPM: mpp("2001:db8::/32"),
			wantOk:  true,
		},
		{
			probe:   netip.MustParsePrefix("2001:db8::1/64"),
			wantLPM: mpp("2001:db8::/32"),
			wantOk:  true,
		},
	}

	for _, tc := range tests {
		_, got := tbl.LookupPrefix(tc.probe)
		if got != tc.wantOk {
			t.Errorf("LookupPrefix non canonical prefix (%s), got: %v, want: %v", tc.probe, got, tc.wantOk)
		}

		lpm, _, got := tbl.LookupPrefixLPM(tc.probe)
		if got != tc.wantOk {
			t.Errorf("LookupPrefixLPM non canonical prefix (%s), got: %v, want: %v", tc.probe, got, tc.wantOk)
		}
		if lpm != tc.wantLPM {
			t.Errorf("LookupPrefixLPM non canonical prefix (%s), got: %v, want: %v", tc.probe, lpm, tc.wantLPM)
		}
	}
}

func TestTableLookupPrefixCompare_Fast(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[int])
	tbl := new(Fast[int])
	for i, pfx := range pfxs {
		gold.insert(pfx, i)
		tbl.Insert(pfx, i)
	}

	for range n {
		pfx := randomPrefix(prng)

		goldVal, goldOK := gold.lookupPfx(pfx)
		tblVal, tblOK := tbl.LookupPrefix(pfx)

		if goldOK != tblOK {
			t.Fatalf("LookupPrefix(%q) = (_, %v), want (_, %v)", pfx, tblOK, goldOK)
		}

		// Skip value comparison for liteTable (no real payload)
		if _, isLite := any(tbl).(*liteTable[int]); !isLite {
			if goldVal != tblVal {
				t.Fatalf("LookupPrefix(%q) = (%v, %v), want (%v, %v)", pfx, tblVal, tblOK, goldVal, goldOK)
			}
		}
	}
}

func TestTableLookupPrefixLPMCompare_Fast(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[int])
	tbl := new(Fast[int])
	for i, pfx := range pfxs {
		gold.insert(pfx, i)
		tbl.Insert(pfx, i)
	}

	for range n {
		pfx := randomPrefix(prng)

		goldLPM, goldVal, goldOK := gold.lookupPfxLPM(pfx)
		tblLPM, tblVal, tblOK := tbl.LookupPrefixLPM(pfx)

		if goldOK != tblOK {
			t.Fatalf("LookupPrefixLPM(%q) = (_, _, %v), want (_, _, %v)", pfx, tblOK, goldOK)
		}

		if goldLPM != tblLPM {
			t.Fatalf("LookupPrefixLPM(%q) = ( %v, _, _), want ( %v, _, _)", pfx, tblLPM, goldLPM)
		}

		// Skip value comparison for liteTable (no real payload)
		if _, isLite := any(tbl).(*liteTable[int]); !isLite {
			if goldVal != tblVal {
				t.Fatalf("LookupPrefixLPM(%q) = (_, %v, _), want (_, %v, _)", pfx, tblVal, goldVal)
			}
		}
	}
}

func TestTableInsertShuffled_Fast(t *testing.T) {
	// The order in which you insert prefixes into a route table
	// should not matter, as long as you're inserting the same set of
	// routes.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	for range 10 {
		pfxs2 := slices.Clone(pfxs)
		prng.Shuffle(len(pfxs2), func(i, j int) { pfxs2[i], pfxs2[j] = pfxs2[j], pfxs2[i] })

		tbl1 := new(Fast[string])
		tbl2 := new(Fast[string])

		for _, pfx := range pfxs {
			tbl1.Insert(pfx, pfx.String())
			tbl1.Insert(pfx, pfx.String()) // idempotent
		}
		for _, pfx := range pfxs2 {
			tbl2.Insert(pfx, pfx.String()) // idempotent
		}

		if tbl1.dumpString() != tbl2.dumpString() {
			t.Fatal("tbl1 and tbl2 have different dumpString representation")
		}
		if !tbl1.Equal(tbl2) {
			t.Fatal("expected Equal")
		}
	}
}

func TestTableInsertPersistShuffled_Fast(t *testing.T) {
	// The order in which you insert prefixes into a route table
	// should not matter, as long as you're inserting the same set of
	// routes.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	for range 10 {
		pfxs2 := slices.Clone(pfxs)
		prng.Shuffle(len(pfxs2), func(i, j int) { pfxs2[i], pfxs2[j] = pfxs2[j], pfxs2[i] })

		tbl1 := new(Fast[string])
		tbl2 := new(Fast[string])

		// bart1 is mutable
		for _, pfx := range pfxs {
			tbl1.Insert(pfx, pfx.String())
		}

		// bart2 is persistent
		for _, pfx := range pfxs2 {
			tbl2 = tbl2.InsertPersist(pfx, pfx.String())
		}

		if tbl1.dumpString() != tbl2.dumpString() {
			t.Fatal("mutable and immutable table have different dumpString representation")
		}

		if !tbl1.Equal(tbl2) {
			t.Fatal("expected Equal")
		}
	}
}

func TestTableDeleteCompare_Fast(t *testing.T) {
	// Create large route tables repeatedly, delete half of their
	// prefixes, and compare Table's behavior to a naive and slow but
	// correct implementation.
	t.Parallel()

	var (
		n            = workLoadN()
		prng         = rand.New(rand.NewPCG(42, 42))
		numPrefixes  = n // total prefixes to insert (test deletes 50% of them)
		numPerFamily = numPrefixes / 2
		deleteCut    = numPerFamily / 2
		probes       = 3
	)

	for range probes {
		all4 := randomRealWorldPrefixes4(prng, numPerFamily)
		all6 := randomRealWorldPrefixes6(prng, numPerFamily)

		pfxs := slices.Concat(all4, all6)
		toDelete := slices.Concat(all4[deleteCut:], all6[deleteCut:])

		gold := new(goldTable[string])
		tbl := new(Fast[string])

		for _, pfx := range pfxs {
			gold.insert(pfx, pfx.String())
			tbl.Insert(pfx, pfx.String())
		}

		for _, pfx := range toDelete {
			gold.delete(pfx)
			tbl.Delete(pfx)
		}

		gold.sort()

		tblFlat := tbl.flatSorted()

		// Skip value comparison for liteTable (no real payload)
		if _, isLite := any(tbl).(*liteTable[string]); isLite {
			if !slices.Equal(gold.allSorted(), tblFlat.allSorted()) {
				t.Fatal("expected Equal")
			}
		} else {
			if !slices.Equal(*gold, tblFlat) {
				t.Fatal("expected Equal")
			}
		}
	}
}

func TestTableDeleteShuffled_Fast(t *testing.T) {
	// The order in which you delete prefixes from a route table
	// should not matter, as long as you're deleting the same set of
	// routes.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))

	var (
		numPrefixes  = n // prefixes to insert (test deletes 50% of them)
		numPerFamily = numPrefixes / 2
		deleteCut    = numPerFamily / 2
	)

	for range 10 {
		all4 := randomRealWorldPrefixes4(prng, numPerFamily)
		all6 := randomRealWorldPrefixes6(prng, numPerFamily)

		pfxs := slices.Concat(all4, all6)
		toDelete := slices.Concat(all4[deleteCut:], all6[deleteCut:])

		tbl := new(Fast[string])

		// insert
		for _, pfx := range pfxs {
			tbl.Insert(pfx, pfx.String())
		}

		// delete
		for _, pfx := range toDelete {
			tbl.Delete(pfx)
		}

		pfxs2 := slices.Clone(pfxs)
		toDelete2 := slices.Clone(toDelete)
		prng.Shuffle(len(toDelete2), func(i, j int) { toDelete2[i], toDelete2[j] = toDelete2[j], toDelete2[i] })

		tbl2 := new(Fast[string])

		// insert
		for _, pfx := range pfxs2 {
			tbl2.Insert(pfx, pfx.String())
		}

		// delete
		for _, pfx := range toDelete2 {
			tbl2.Delete(pfx)
		}

		if !tbl.Equal(tbl2) {
			t.Fatal("expect equal")
		}
	}
}

func TestTableDeleteIsReverseOfInsert_Fast(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	// Insert n prefixes, then delete those same prefixes in reverse
	// order. Each deletion should exactly undo the internal structure
	// changes that each insert did.

	tbl := new(Fast[string])
	want := tbl.dumpString()

	prefixes := randomRealWorldPrefixes(prng, n)

	defer func() {
		if t.Failed() {
			t.Logf("the prefixes that fail the test: %v\n", prefixes)
		}
	}()

	for _, p := range prefixes {
		tbl.Insert(p, p.String())
	}

	for i := len(prefixes) - 1; i >= 0; i-- {
		tbl.Delete(prefixes[i])
	}
	if got := tbl.dumpString(); got != want {
		t.Fatalf("after delete, mismatch:\n\n got: %s\n\nwant: %s", got, want)
	}
}

func TestTableDeleteButOne_Fast(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))
	// Insert n prefixes, then delete all but one
	n := workLoadN()

	for range 10 {

		tbl := new(Fast[any])
		prefixes := randomRealWorldPrefixes(prng, n)

		for _, p := range prefixes {
			tbl.Insert(p, nil)
		}

		// shuffle the prefixes
		prng.Shuffle(n, func(i, j int) {
			prefixes[i], prefixes[j] = prefixes[j], prefixes[i]
		})

		// skip the first
		for i := 1; i < len(prefixes); i++ {
			tbl.Delete(prefixes[i])
		}

		if size := tbl.Size(); size != 1 {
			t.Fatalf("Size(), got %d, want 1", size)
		}

		stats4 := tbl.root4.StatsRec()
		stats6 := tbl.root6.StatsRec()

		if nodes := stats4.SubNodes + stats6.SubNodes; nodes != 1 {
			t.Fatalf("delete but one, want nodes: 1, got: %d\n%s", nodes, tbl.dumpString())
		}

		sum := stats4.Prefixes + stats4.Leaves + stats4.Fringes +
			stats6.Prefixes + stats6.Leaves + stats6.Fringes

		if sum != 1 {
			t.Fatalf("delete but one, only one item must be left, but: %d\n%s", sum, tbl.dumpString())
		}
	}
}

func TestTableGet_Fast(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))
	pfx := randomPrefix(prng)

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

	tbl := new(Fast[int])
	if _, ok := tbl.Get(pfx); ok {
		t.Errorf("empty table: Get(%v), ok=%v, expected: %v", pfx, ok, false)
	}

	for _, tt := range tests {
		tbl.Insert(tt.pfx, tt.val)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := tbl.Get(tt.pfx)

			if !ok {
				t.Errorf("%s: ok=%v, expected: %v", tt.name, ok, true)
			}

			if got != tt.val {
				t.Errorf("%s: val=%v, expected: %v", tt.name, got, tt.val)
			}
		})
	}
}

func TestTableGetCompare_Fast(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[string])
	tbl := new(Fast[string])
	for _, pfx := range pfxs {
		gold.insert(pfx, pfx.String())
		tbl.Insert(pfx, pfx.String())
	}

	for _, pfx := range pfxs {
		goldVal, goldOK := gold.get(pfx)
		tblVal, tblOK := tbl.Get(pfx)

		if goldOK != tblOK {
			t.Fatalf("Get(%q) = (_, %v), want (_, %v)", pfx, tblOK, goldOK)
		}

		// Skip value comparison for liteTable (no real payload)
		if _, isLite := any(tbl).(*liteTable[string]); !isLite {
			if goldVal != tblVal {
				t.Fatalf("Get(%q) = (%v, %v), want (%v, %v)", pfx, tblVal, tblOK, goldVal, goldOK)
			}
		}
	}
}

func TestTableModifySemantics_Fast(t *testing.T) {
	t.Parallel()

	type args struct {
		pfx netip.Prefix
		cb  func(val int, found bool) (_ int, del bool)
	}

	tests := []struct {
		name      string
		prepare   map[netip.Prefix]int // entries to pre-populate the table
		args      args
		finalData map[netip.Prefix]int // expected table contents after the operation
	}{
		{
			name:    "Delete existing entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42, mpp("2001:db8::/32"): 4242},
			args: args{
				pfx: mpp("10.0.0.0/8"),
				cb:  func(val int, found bool) (_ int, del bool) { return 0, true },
			},
			finalData: map[netip.Prefix]int{mpp("2001:db8::/32"): 4242},
		},

		{
			name:    "Insert new entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42},
			args: args{
				pfx: mpp("2001:db8::/32"),
				cb:  func(val int, found bool) (_ int, del bool) { return 4242, false },
			},
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42, mpp("2001:db8::/32"): 4242},
		},

		{
			name:    "Update existing entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42, mpp("2001:db8::/32"): 4242},
			args: args{
				pfx: mpp("10.0.0.0/8"),
				cb:  func(val int, found bool) (_ int, del bool) { return -1, false },
			},
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): -1, mpp("2001:db8::/32"): 4242},
		},

		{
			name:    "No-op on missing entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42},
			args: args{
				pfx: mpp("2001:db8::/32"),
				cb:  func(val int, found bool) (_ int, del bool) { return 0, true },
			},
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42},
		},
	}

	for _, tt := range tests {
		tbl := new(Fast[int])

		// Insert initial entries using Modify
		for pfx, v := range tt.prepare {
			tbl.Modify(pfx, func(_ int, _ bool) (_ int, del bool) { return v, false })
		}

		tbl.Modify(tt.args.pfx, tt.args.cb)

		// Check the final state of the table using Get, compares expected and actual table
		got := make(map[netip.Prefix]int, len(tt.finalData))
		for pfx, val := range tbl.All() {
			got[pfx] = val
		}
		if len(got) != len(tt.finalData) {
			t.Fatalf("[%s] final table size mismatch: got %d, want %d", tt.name, len(got), len(tt.finalData))
		}
		for pfx, wantVal := range tt.finalData {
			gotVal, ok := got[pfx]
			if !ok || gotVal != wantVal {
				t.Fatalf("[%s] final table: key %v = %v (present=%v), want %v", tt.name, pfx, gotVal, ok, wantVal)
			}
		}
	}
}

func TestTableModifyPersistSemantics_Fast(t *testing.T) {
	t.Parallel()

	type args struct {
		pfx netip.Prefix
		cb  func(val int, found bool) (_ int, del bool)
	}

	tests := []struct {
		name      string
		prepare   map[netip.Prefix]int // entries to pre-populate the table
		args      args
		finalData map[netip.Prefix]int // expected table contents after the operation
	}{
		{
			name:    "Delete existing entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42, mpp("2001:db8::/32"): 4242},
			args: args{
				pfx: mpp("10.0.0.0/8"),
				cb:  func(val int, found bool) (_ int, del bool) { return 0, true },
			},
			finalData: map[netip.Prefix]int{mpp("2001:db8::/32"): 4242},
		},

		{
			name:    "Insert new entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42},
			args: args{
				pfx: mpp("2001:db8::/32"),
				cb:  func(val int, found bool) (_ int, del bool) { return 4242, false },
			},
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42, mpp("2001:db8::/32"): 4242},
		},

		{
			name:    "Update existing entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42, mpp("2001:db8::/32"): 4242},
			args: args{
				pfx: mpp("10.0.0.0/8"),
				cb:  func(val int, found bool) (_ int, del bool) { return -1, false },
			},
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): -1, mpp("2001:db8::/32"): 4242},
		},

		{
			name:    "No-op on missing entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42},
			args: args{
				pfx: mpp("2001:db8::/32"),
				cb:  func(val int, found bool) (_ int, del bool) { return 0, true },
			},
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42},
		},
	}

	for _, tt := range tests {
		tbl := new(Fast[int])

		// Insert initial entries using Modify
		for pfx, v := range tt.prepare {
			tbl.Modify(pfx, func(_ int, _ bool) (_ int, del bool) { return v, false })
		}

		prt := tbl.ModifyPersist(tt.args.pfx, tt.args.cb)

		// Check the final state of the table using Get, compares expected and actual table
		for pfx, wantVal := range tt.finalData {
			gotVal, ok := prt.Get(pfx)
			if !ok || gotVal != wantVal {
				t.Errorf("[%s] final table: key %v = %v (ok=%v), want %v (ok=true)", tt.name, pfx, gotVal, ok, wantVal)
			}
		}
		// Ensure there are no unexpected entries
		for pfx := range tt.prepare {
			if _, expect := tt.finalData[pfx]; !expect {
				if _, ok := prt.Get(pfx); ok {
					t.Errorf("[%s] final table: key %v should not be present", tt.name, pfx)
				}
			}
		}
	}
}

func TestTableModifyCompare_Fast(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[int])
	tbl := new(Fast[int])

	// Update as insert
	for i, pfx := range pfxs {
		gold.insert(pfx, i)
		tbl.Modify(pfx, func(int, bool) (int, bool) { return i, false })
	}

	gold.sort()
	tblFlat := tbl.flatSorted()

	// Skip value comparison for liteTable (no real payload)
	if _, isLite := any(tbl).(*liteTable[int]); isLite {
		if !slices.Equal(gold.allSorted(), tblFlat.allSorted()) {
			t.Fatal("expected Equal")
		}
	} else {
		if !slices.Equal(*gold, tblFlat) {
			t.Fatal("expected Equal")
		}
	}

	cb1 := func(val int, _ bool) int { return val + 1 }
	cb2 := func(val int, _ bool) (int, bool) { return val + 1, false }

	// Update as update
	for _, pfx := range pfxs[:len(pfxs)/2] {
		gold.update(pfx, cb1)
		tbl.Modify(pfx, cb2)
	}

	gold.sort()
	tblFlat = tbl.flatSorted()

	// Skip value comparison for liteTable (no real payload)
	if _, isLite := any(tbl).(*liteTable[int]); isLite {
		if !slices.Equal(gold.allSorted(), tblFlat.allSorted()) {
			t.Fatal("expected Equal")
		}
	} else {
		if !slices.Equal(*gold, tblFlat) {
			t.Fatal("expected Equal")
		}
	}
}

func TestTableModifyPersistCompare_Fast(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	mut := new(Fast[int])
	imu := new(Fast[int])

	// Update as insert
	for i, pfx := range pfxs {
		mut.Modify(pfx, func(int, bool) (int, bool) { return i, false })
		imu = imu.ModifyPersist(pfx, func(int, bool) (int, bool) { return i, false })
	}

	if !mut.Equal(imu) {
		t.Fatal("expected Equal")
	}

	cb := func(val int, _ bool) (int, bool) { return val + 1, false }

	// Update as update
	for _, pfx := range pfxs[:len(pfxs)/2] {
		mut.Modify(pfx, cb)
		imu = imu.ModifyPersist(pfx, cb)
	}

	if !mut.Equal(imu) {
		t.Fatal("expected Equal")
	}
}

func TestTableModifyShuffled_Fast(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))

	var (
		numPrefixes  = n // prefixes to insert (test deletes 50% of them)
		numPerFamily = numPrefixes / 2
		deleteCut    = numPerFamily / 2
	)

	for range 10 {
		all4 := randomRealWorldPrefixes4(prng, numPerFamily)
		all6 := randomRealWorldPrefixes6(prng, numPerFamily)

		// pfxs toDelete should be non-overlapping sets
		pfxs := slices.Concat(all4[:deleteCut], all6[:deleteCut])
		toDelete := slices.Concat(all4[deleteCut:], all6[deleteCut:])

		tbl1 := new(Fast[string])

		// insert
		for _, pfx := range pfxs {
			tbl1.Insert(pfx, pfx.String())
		}

		// this callback deletes unconditionally
		cb := func(string, bool) (string, bool) { return "", true }

		// delete
		for _, pfx := range toDelete {
			tbl1.Modify(pfx, cb)
		}

		pfxs2 := slices.Clone(pfxs)
		toDelete2 := slices.Clone(toDelete)
		prng.Shuffle(len(toDelete2), func(i, j int) { toDelete2[i], toDelete2[j] = toDelete2[j], toDelete2[i] })

		tbl2 := new(Fast[string])

		// insert
		for _, pfx := range pfxs2 {
			tbl2.Insert(pfx, pfx.String())
		}

		// delete
		for _, pfx := range toDelete2 {
			tbl2.Modify(pfx, cb)
		}

		if !tbl1.Equal(tbl2) {
			t.Fatal("expected equal")
		}
	}
}

func TestTableModifyPersistShuffled_Fast(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))

	var (
		numPrefixes  = n // prefixes to insert (test deletes 50% of them)
		numPerFamily = numPrefixes / 2
		deleteCut    = numPerFamily / 2
	)

	for range 10 {
		all4 := randomRealWorldPrefixes4(prng, numPerFamily)
		all6 := randomRealWorldPrefixes6(prng, numPerFamily)

		// pfxs toDelete should be non-overlapping sets
		pfxs := slices.Concat(all4, all6)
		toDelete := slices.Concat(all4[deleteCut:], all6[deleteCut:])

		tbl1 := new(Fast[string])

		// insert
		for _, pfx := range pfxs {
			tbl1.Insert(pfx, pfx.String())
		}

		// this callback deletes unconditionally
		cb := func(string, bool) (string, bool) { return "", true }

		// delete
		for _, pfx := range toDelete {
			tbl1 = tbl1.ModifyPersist(pfx, cb)
		}

		pfxs2 := slices.Clone(pfxs)
		toDelete2 := slices.Clone(toDelete)
		prng.Shuffle(len(toDelete2), func(i, j int) { toDelete2[i], toDelete2[j] = toDelete2[j], toDelete2[i] })

		tbl2 := new(Fast[string])

		// insert
		for _, pfx := range pfxs2 {
			tbl2.Insert(pfx, pfx.String())
		}

		// delete
		for _, pfx := range toDelete2 {
			tbl2 = tbl2.ModifyPersist(pfx, cb)
		}

		if !tbl1.Equal(tbl2) {
			t.Fatal("expected equal")
		}
	}
}

// TestUnionMemoryAliasing tests that the Union method does not alias memory
// between the two tables.
func TestTableUnionMemoryAliasing_Fast(t *testing.T) {
	t.Parallel()

	newTable := func(pfx ...string) *Fast[struct{}] {
		tbl := new(Fast[struct{}])
		for _, s := range pfx {
			tbl.Insert(mpp(s), struct{}{})
		}
		return tbl
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

// TestUnionPersistMemoryAliasing tests that the Union method does not alias memory
// between the tables.
func TestTableUnionPersistMemoryAliasing_Fast(t *testing.T) {
	t.Parallel()

	newTable := func(pfx ...string) *Fast[struct{}] {
		tbl := new(Fast[struct{}])
		for _, s := range pfx {
			tbl.Insert(mpp(s), struct{}{})
		}
		return tbl
	}
	// First create two tables with disjoint prefixes.
	a := newTable("100.69.1.0/24")
	b := newTable("0.0.0.0/24")

	// Verify that the tables are disjoint.
	if a.Overlaps(b) {
		t.Error("this should not overlap other")
	}

	// Now union them with copy-on-write.
	pTbl := a.UnionPersist(b)

	// Add a new prefix to new union
	pTbl.Insert(mpp("0.0.1.0/24"), struct{}{})

	// Ensure that a is unchanged.
	_, ok := a.Lookup(mpa("0.0.1.1"))
	if ok {
		t.Error("a should not contain 0.0.1.1")
	}
	if a.OverlapsPrefix(mpp("0.0.1.1/32")) {
		t.Error("a should not overlap 0.0.1.1/32")
	}
}

func TestTableUnionCompare_Fast(t *testing.T) {
	t.Parallel()

	n := workLoadN()
	prng := rand.New(rand.NewPCG(42, 42))

	for range 3 {
		pfxs := randomRealWorldPrefixes(prng, n)

		gold := new(goldTable[string])
		tbl := new(Fast[string])

		for _, pfx := range pfxs {
			gold.insert(pfx, pfx.String())
			tbl.Insert(pfx, pfx.String())
		}

		pfxs2 := randomRealWorldPrefixes(prng, n)

		gold2 := new(goldTable[string])
		tbl2 := new(Fast[string])

		for _, pfx := range pfxs2 {
			gold2.insert(pfx, pfx.String())
			tbl2.Insert(pfx, pfx.String())
		}

		gold.union(gold2)
		tbl.Union(tbl2)

		// dump as slow table for comparison
		tblFlat := tbl.flatSorted()

		// sort for comparison
		gold.sort()

		// Skip value comparison for liteTable (no real payload)
		if _, isLite := any(tbl).(*liteTable[string]); isLite {
			if !slices.Equal(gold.allSorted(), tblFlat.allSorted()) {
				t.Fatal("expected Equal")
			}
		} else {
			if !slices.Equal(*gold, tblFlat) {
				t.Fatal("expected Equal")
			}
		}
	}
}

func TestTableUnionPersistCompare_Fast(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))

	n := workLoadN()

	for range 3 {
		pfxs := randomRealWorldPrefixes(prng, n)

		gold := new(goldTable[int])
		tbl := new(Fast[int])

		for i, pfx := range pfxs {
			gold.insert(pfx, i)
			tbl.Insert(pfx, i)
		}

		pfxs2 := randomRealWorldPrefixes(prng, n)

		gold2 := new(goldTable[int])
		tbl2 := new(Fast[int])

		for i, pfx := range pfxs2 {
			gold2.insert(pfx, i)
			tbl2.Insert(pfx, i)
		}

		gold.union(gold2)
		tblP := tbl.UnionPersist(tbl2)

		// dump as slow table for comparison
		flatP := tblP.flatSorted()

		// sort for comparison
		gold.sort()

		// Skip value comparison for liteTable (no real payload)
		if _, isLite := any(tbl).(*liteTable[int]); isLite {
			if !slices.Equal(gold.allSorted(), flatP.allSorted()) {
				t.Fatal("expected Equal")
			}
		} else {
			if !slices.Equal(*gold, flatP) {
				t.Fatal("expected Equal")
			}
		}
	}
}
