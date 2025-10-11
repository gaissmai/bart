// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"encoding/json"
	"math/rand/v2"
	"net/netip"
	"slices"
	"strings"
	"testing"
)

// ############ tests ################################

func TestLiteNil(t *testing.T) {
	t.Parallel()

	ip4 := mpa("127.0.0.1")
	ip6 := mpa("::1")

	pfx4 := mpp("127.0.0.0/8")
	pfx6 := mpp("::1/128")

	lite2 := new(Lite)
	lite2.Insert(pfx4)
	lite2.Insert(pfx6)

	var lite1 *Lite = nil

	t.Run("mustPanic", func(t *testing.T) {
		t.Parallel()

		mustPanic(t, "sizeUpdate", func() { lite1.sizeUpdate(true, 1) })
		mustPanic(t, "sizeUpdate", func() { lite1.sizeUpdate(false, 1) })
		mustPanic(t, "rootNodeByVersion", func() { lite1.rootNodeByVersion(true) })
		mustPanic(t, "rootNodeByVersion", func() { lite1.rootNodeByVersion(false) })
		mustPanic(t, "fprint", func() { lite1.fprint(nil, true) })
		mustPanic(t, "fprint", func() { lite1.fprint(nil, false) })
		mustPanic(t, "dump", func() { lite1.dump(nil) })
		mustPanic(t, "dumpString", func() { lite1.dumpString() })

		mustPanic(t, "Size", func() { lite1.Size() })
		mustPanic(t, "Size4", func() { lite1.Size4() })
		mustPanic(t, "Size6", func() { lite1.Size6() })

		mustPanic(t, "Get", func() { lite1.Get(pfx4) })
		mustPanic(t, "Insert", func() { lite1.Insert(pfx4) })
		mustPanic(t, "InsertPersist", func() { lite1.InsertPersist(pfx4) })
		mustPanic(t, "Delete", func() { lite1.Delete(pfx4) })
		mustPanic(t, "DeletePersist", func() { lite1.DeletePersist(pfx4) })
		mustPanic(t, "Modify", func() { lite1.Modify(pfx4, nil) })
		mustPanic(t, "ModifyPersist", func() { lite1.Modify(pfx4, nil) })
		mustPanic(t, "Contains", func() { lite1.Contains(ip4) })
		mustPanic(t, "Lookup", func() { lite1.Lookup(ip6) })
		mustPanic(t, "LookupPrefix", func() { lite1.LookupPrefix(pfx4) })
		mustPanic(t, "LookupPrefixLPM", func() { lite1.LookupPrefixLPM(pfx4) })
		mustPanic(t, "Union", func() { lite1.Union(lite2) })
		mustPanic(t, "UnionPersist", func() { lite1.UnionPersist(lite2) })
		mustPanic(t, "DumpList4", func() { lite1.DumpList4() })
		mustPanic(t, "DumpList6", func() { lite1.DumpList6() })
		mustPanic(t, "Fprint", func() { lite1.Fprint(nil) })
		mustPanic(t, "MarshalJSON", func() { _, _ = lite1.MarshalJSON() })
		mustPanic(t, "MarshalText", func() { _, _ = lite1.MarshalText() })
	})

	t.Run("noPanic", func(t *testing.T) {
		t.Parallel()

		noPanic(t, "Clone", func() { lite1.Clone() })

		noPanic(t, "Overlaps", func() { lite1.Overlaps(nil) })
		noPanic(t, "Overlaps4", func() { lite1.Overlaps4(nil) })
		noPanic(t, "Overlaps6", func() { lite1.Overlaps6(nil) })

		noPanic(t, "Overlaps", func() { lite2.Overlaps(lite2) })
		noPanic(t, "Overlaps4", func() { lite2.Overlaps4(lite2) })
		noPanic(t, "Overlaps6", func() { lite2.Overlaps6(lite2) })

		mustPanic(t, "Overlaps", func() { lite1.Overlaps(lite2) })
		mustPanic(t, "Overlaps4", func() { lite1.Overlaps4(lite2) })
		mustPanic(t, "Overlaps6", func() { lite1.Overlaps6(lite2) })

		mustPanic(t, "Equal", func() { lite1.Equal(lite2) })
		noPanic(t, "Equal", func() { lite1.Equal(lite1) })
		noPanic(t, "Equal", func() { lite2.Equal(lite2) })
	})

	t.Run("noPanicRangeOverFunc", func(t *testing.T) {
		t.Parallel()

		noPanicRangeOverFunc[any](t, "All", lite1.All)
		noPanicRangeOverFunc[any](t, "All4", lite1.All4)
		noPanicRangeOverFunc[any](t, "All6", lite1.All6)
		noPanicRangeOverFunc[any](t, "AllSorted", lite1.AllSorted)
		noPanicRangeOverFunc[any](t, "AllSorted4", lite1.AllSorted4)
		noPanicRangeOverFunc[any](t, "AllSorted6", lite1.AllSorted6)
		noPanicRangeOverFunc[any](t, "Subnets", lite1.Subnets)
		noPanicRangeOverFunc[any](t, "Supernets", lite1.Supernets)
	})
}

func TestLiteInvalid(t *testing.T) {
	t.Parallel()

	lite1 := new(Lite)
	lite2 := new(Lite)

	var zeroIP netip.Addr
	var zeroPfx netip.Prefix

	noPanic(t, "All", func() { lite1.All() })
	noPanic(t, "All4", func() { lite1.All4() })
	noPanic(t, "All6", func() { lite1.All6() })
	noPanic(t, "AllSorted", func() { lite1.AllSorted() })
	noPanic(t, "AllSorted4", func() { lite1.AllSorted4() })
	noPanic(t, "AllSorted6", func() { lite1.AllSorted6() })
	noPanic(t, "Clone", func() { lite1.Clone() })
	noPanic(t, "Contains", func() { lite1.Contains(zeroIP) })
	noPanic(t, "Delete", func() { lite1.Delete(zeroPfx) })
	noPanic(t, "DeletePersist", func() { lite1.DeletePersist(zeroPfx) })
	noPanic(t, "DumpList4", func() { lite1.DumpList4() })
	noPanic(t, "DumpList6", func() { lite1.DumpList6() })
	noPanic(t, "Equal", func() { lite1.Equal(lite2) })
	noPanic(t, "Fprint", func() { lite1.Fprint(nil) })
	noPanic(t, "Get", func() { lite1.Get(zeroPfx) })
	noPanic(t, "Insert", func() { lite1.Insert(zeroPfx) })
	noPanic(t, "InsertPersist", func() { lite1.InsertPersist(zeroPfx) })
	noPanic(t, "Lookup", func() { lite1.Lookup(zeroIP) })
	noPanic(t, "LookupPrefix", func() { lite1.LookupPrefix(zeroPfx) })
	noPanic(t, "LookupPrefixLPM", func() { lite1.LookupPrefixLPM(zeroPfx) })
	noPanic(t, "MarshalJSON", func() { _, _ = lite1.MarshalJSON() })
	noPanic(t, "MarshalText", func() { _, _ = lite1.MarshalText() })
	noPanic(t, "Modify", func() { lite1.Modify(zeroPfx, nil) })
	noPanic(t, "ModifyPersist", func() { lite1.ModifyPersist(zeroPfx, nil) })
	noPanic(t, "Overlaps", func() { lite1.Overlaps(lite2) })
	noPanic(t, "Overlaps4", func() { lite1.Overlaps4(lite2) })
	noPanic(t, "Overlaps6", func() { lite1.Overlaps6(lite2) })
	noPanic(t, "OverlapsPrefix", func() { lite1.OverlapsPrefix(zeroPfx) })
	noPanic(t, "Size", func() { lite1.Size() })
	noPanic(t, "Size4", func() { lite1.Size4() })
	noPanic(t, "Size6", func() { lite1.Size6() })
	noPanic(t, "Subnets", func() { lite1.Subnets(zeroPfx) })
	noPanic(t, "Supernets", func() { lite1.Supernets(zeroPfx) })
	noPanic(t, "Union", func() { lite1.Union(lite2) })
	noPanic(t, "UnionPersist", func() { lite1.UnionPersist(lite2) })
}

func TestLiteTableNil(t *testing.T) {
	t.Parallel()

	ip4 := mpa("127.0.0.1")
	ip6 := mpa("::1")

	pfx4 := mpp("127.0.0.0/8")
	pfx6 := mpp("::1/128")

	bart2 := new(liteTable[any])
	bart2.Insert(pfx4, nil)
	bart2.Insert(pfx6, nil)

	var bart1 *liteTable[any] = nil

	t.Run("mustPanic", func(t *testing.T) {
		t.Parallel()

		mustPanic(t, "sizeUpdate", func() { bart1.sizeUpdate(true, 1) })
		mustPanic(t, "sizeUpdate", func() { bart1.sizeUpdate(false, 1) })
		mustPanic(t, "rootNodeByVersion", func() { bart1.rootNodeByVersion(true) })
		mustPanic(t, "rootNodeByVersion", func() { bart1.rootNodeByVersion(false) })
		mustPanic(t, "fprint", func() { bart1.fprint(nil, true) })
		mustPanic(t, "fprint", func() { bart1.fprint(nil, false) })

		mustPanic(t, "Size", func() { bart1.Size() })
		mustPanic(t, "Size4", func() { bart1.Size4() })
		mustPanic(t, "Size6", func() { bart1.Size6() })

		mustPanic(t, "Get", func() { bart1.Get(pfx4) })
		mustPanic(t, "Insert", func() { bart1.Insert(pfx4, nil) })
		mustPanic(t, "InsertPersist", func() { bart1.InsertPersist(pfx4, nil) })
		mustPanic(t, "Delete", func() { bart1.Delete(pfx4) })
		mustPanic(t, "DeletePersist", func() { bart1.DeletePersist(pfx4) })
		mustPanic(t, "Modify", func() { bart1.Modify(pfx4, nil) })
		mustPanic(t, "ModifyPersist", func() { bart1.Modify(pfx4, nil) })
		mustPanic(t, "Contains", func() { bart1.Contains(ip4) })
		mustPanic(t, "Contains", func() { bart1.Contains(ip6) })
		mustPanic(t, "lookupPrefixLPM", func() { bart1.lookupPrefixLPM(pfx4, true) })
		mustPanic(t, "lookupPrefixLPM", func() { bart1.lookupPrefixLPM(pfx6, false) })
		mustPanic(t, "Union", func() { bart1.Union(bart2) })
		mustPanic(t, "UnionPersist", func() { bart1.UnionPersist(bart2) })
	})

	t.Run("noPanic", func(t *testing.T) {
		t.Parallel()

		noPanic(t, "Overlaps", func() { bart1.Overlaps(nil) })
		noPanic(t, "Overlaps4", func() { bart1.Overlaps4(nil) })
		noPanic(t, "Overlaps6", func() { bart1.Overlaps6(nil) })

		noPanic(t, "Overlaps", func() { bart2.Overlaps(bart2) })
		noPanic(t, "Overlaps4", func() { bart2.Overlaps4(bart2) })
		noPanic(t, "Overlaps6", func() { bart2.Overlaps6(bart2) })

		mustPanic(t, "Overlaps", func() { bart1.Overlaps(bart2) })
		mustPanic(t, "Overlaps4", func() { bart1.Overlaps4(bart2) })
		mustPanic(t, "Overlaps6", func() { bart1.Overlaps6(bart2) })

		mustPanic(t, "Equal", func() { bart1.Equal(bart2) })
		noPanic(t, "Equal", func() { bart1.Equal(bart1) })
		noPanic(t, "Equal", func() { bart2.Equal(bart2) })

		noPanic(t, "dump", func() { bart1.dump(nil) })
		noPanic(t, "dumpString", func() { bart1.dumpString() })
		noPanic(t, "Clone", func() { bart1.Clone() })
		noPanic(t, "DumpList4", func() { bart1.DumpList4() })
		noPanic(t, "DumpList6", func() { bart1.DumpList6() })
		noPanic(t, "Fprint", func() { bart1.Fprint(nil) })
		noPanic(t, "MarshalJSON", func() { _, _ = bart1.MarshalJSON() })
		noPanic(t, "MarshalText", func() { _, _ = bart1.MarshalText() })
	})

	t.Run("noPanicRangeOverFunc", func(t *testing.T) {
		t.Parallel()

		noPanicRangeOverFunc[any](t, "All", bart1.All)
		noPanicRangeOverFunc[any](t, "All4", bart1.All4)
		noPanicRangeOverFunc[any](t, "All6", bart1.All6)
		noPanicRangeOverFunc[any](t, "AllSorted", bart1.AllSorted)
		noPanicRangeOverFunc[any](t, "AllSorted4", bart1.AllSorted4)
		noPanicRangeOverFunc[any](t, "AllSorted6", bart1.AllSorted6)
		noPanicRangeOverFunc[any](t, "Subnets", bart1.Subnets)
		noPanicRangeOverFunc[any](t, "Supernets", bart1.Supernets)
	})
}

func TestLiteTableInvalid(t *testing.T) {
	t.Parallel()

	bart1 := new(liteTable[any])
	bart2 := new(liteTable[any])

	var zeroIP netip.Addr
	var zeroPfx netip.Prefix

	noPanic(t, "All", func() { bart1.All() })
	noPanic(t, "All4", func() { bart1.All4() })
	noPanic(t, "All6", func() { bart1.All6() })
	noPanic(t, "AllSorted", func() { bart1.AllSorted() })
	noPanic(t, "AllSorted4", func() { bart1.AllSorted4() })
	noPanic(t, "AllSorted6", func() { bart1.AllSorted6() })
	noPanic(t, "Clone", func() { bart1.Clone() })
	noPanic(t, "Contains", func() { bart1.Contains(zeroIP) })
	noPanic(t, "Delete", func() { bart1.Delete(zeroPfx) })
	noPanic(t, "DeletePersist", func() { bart1.DeletePersist(zeroPfx) })
	noPanic(t, "DumpList4", func() { bart1.DumpList4() })
	noPanic(t, "DumpList6", func() { bart1.DumpList6() })
	noPanic(t, "Equal", func() { bart1.Equal(bart2) })
	noPanic(t, "Fprint", func() { bart1.Fprint(nil) })
	noPanic(t, "Get", func() { bart1.Get(zeroPfx) })
	noPanic(t, "Insert", func() { bart1.Insert(zeroPfx, nil) })
	noPanic(t, "InsertPersist", func() { bart1.InsertPersist(zeroPfx, nil) })
	noPanic(t, "LookupPrefixLPM", func() { bart1.lookupPrefixLPM(zeroPfx, true) })
	noPanic(t, "LookupPrefixLPM", func() { bart1.lookupPrefixLPM(zeroPfx, false) })
	noPanic(t, "MarshalJSON", func() { _, _ = bart1.MarshalJSON() })
	noPanic(t, "MarshalText", func() { _, _ = bart1.MarshalText() })
	noPanic(t, "Modify", func() { bart1.Modify(zeroPfx, nil) })
	noPanic(t, "ModifyPersist", func() { bart1.ModifyPersist(zeroPfx, nil) })
	noPanic(t, "Overlaps", func() { bart1.Overlaps(bart2) })
	noPanic(t, "Overlaps4", func() { bart1.Overlaps4(bart2) })
	noPanic(t, "Overlaps6", func() { bart1.Overlaps6(bart2) })
	noPanic(t, "OverlapsPrefix", func() { bart1.OverlapsPrefix(zeroPfx) })
	noPanic(t, "Size", func() { bart1.Size() })
	noPanic(t, "Size4", func() { bart1.Size4() })
	noPanic(t, "Size6", func() { bart1.Size6() })
	noPanic(t, "Subnets", func() { bart1.Subnets(zeroPfx) })
	noPanic(t, "Supernets", func() { bart1.Supernets(zeroPfx) })
	noPanic(t, "Union", func() { bart1.Union(bart2) })
	noPanic(t, "UnionPersist", func() { bart1.UnionPersist(bart2) })
}

func TestLiteContainsCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Lite's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[int])
	lite := new(Lite)

	for i, p := range pfxs {
		gold.insert(p, i) // ensures Masked + de-dupe
		lite.Insert(p)
	}

	for range n {
		a := randomAddr(prng)

		_, goldOK := gold.lookup(a)
		liteOK := lite.Contains(a)

		if goldOK != liteOK {
			t.Fatalf("Contains(%q) = %v, want %v", a, liteOK, goldOK)
		}
	}
}

func TestLiteLookupCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Lite's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[int])
	lite := new(Lite)

	for i, pfx := range pfxs {
		gold.insert(pfx, i)
		lite.Insert(pfx)
	}

	for range n {
		a := randomAddr(prng)

		_, goldOK := gold.lookup(a)
		liteOK := lite.Lookup(a)

		if goldOK != liteOK {
			t.Fatalf("Lookup(%q) = %v, want %v", a, liteOK, goldOK)
		}
	}
}

func TestLiteLookupPrefixUnmasked(t *testing.T) {
	// test that the pfx must not be masked on input for LookupPrefix
	t.Parallel()

	lite := new(Lite)
	lite.Insert(mpp("10.20.30.0/24"))

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
			wantLPM: netip.MustParsePrefix("10.20.30.0/24"),
			wantOk:  true,
		},
		{
			probe:   netip.MustParsePrefix("10.20.30.40/25"),
			wantLPM: netip.MustParsePrefix("10.20.30.0/24"),
			wantOk:  true,
		},
		{
			probe:   netip.MustParsePrefix("10.20.30.40/32"),
			wantLPM: netip.MustParsePrefix("10.20.30.0/24"),
			wantOk:  true,
		},
	}

	for _, tc := range tests {
		got := lite.LookupPrefix(tc.probe)
		if got != tc.wantOk {
			t.Errorf("LookupPrefix non canonical prefix (%s), got: %v, want: %v", tc.probe, got, tc.wantOk)
		}

		lpm, got := lite.LookupPrefixLPM(tc.probe)
		if got != tc.wantOk {
			t.Errorf("LookupPrefixLPM non canonical prefix (%s), got: %v, want: %v", tc.probe, got, tc.wantOk)
		}
		if lpm != tc.wantLPM {
			t.Errorf("LookupPrefixLPM non canonical prefix (%s), got: %v, want: %v", tc.probe, lpm, tc.wantLPM)
		}
	}
}

func TestLiteLookupPrefixCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Lite's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[int])
	lite := new(Lite)
	for i, pfx := range pfxs {
		gold.insert(pfx, i)
		lite.Insert(pfx)
	}

	for range n {
		pfx := randomPrefix(prng)

		_, goldOK := gold.lookupPfx(pfx)
		liteOK := lite.LookupPrefix(pfx)

		if goldOK != liteOK {
			t.Fatalf("LookupPrefix(%q) = %v, want %v", pfx, liteOK, goldOK)
		}
	}
}

func TestLiteLookupPrefixLPMCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Lite's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, n)

	gold := new(goldTable[int])
	gold.insertMany(pfxs)

	lite := new(Lite)
	for _, pfx := range pfxs {
		lite.Insert(pfx.pfx)
	}

	for range n {
		pfx := randomPrefix(prng)

		goldLPM, _, goldOK := gold.lookupPfxLPM(pfx)
		liteLPM, liteOK := lite.LookupPrefixLPM(pfx)

		if goldOK != liteOK {
			t.Fatalf("LookupPrefixLPM(%q) = %v, want %v", pfx, liteOK, goldOK)
		}

		if goldLPM != liteLPM {
			t.Fatalf("LookupPrefixLPM(%q) = (%v, %v), want (%v, %v)", pfx, liteLPM, liteOK, goldLPM, goldOK)
		}
	}
}

func TestLiteInsertShuffled(t *testing.T) {
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

		lite1 := new(Lite)
		lite2 := new(Lite)

		for _, pfx := range pfxs {
			lite1.Insert(pfx)
			lite1.Insert(pfx) // idempotent
		}
		for _, pfx := range pfxs2 {
			lite2.Insert(pfx) // idempotent
		}

		if !lite1.Equal(lite2) {
			t.Fatal("expected Equal")
		}
	}
}

func TestLiteInsertPersistShuffled(t *testing.T) {
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

		lite1 := new(Lite)
		lite2 := new(Lite)

		// lite1 is mutable
		for _, pfx := range pfxs {
			lite1.Insert(pfx)
		}

		// lite2 is persistent
		for _, pfx := range pfxs2 {
			lite2 = lite2.InsertPersist(pfx)
		}

		if lite1.dumpString() != lite2.dumpString() {
			t.Fatal("mutable and immutable table have different dumpString representation")
		}

		if !lite1.Equal(lite2) {
			t.Fatal("expected Equal")
		}
	}
}

func TestLiteDeleteCompare(t *testing.T) {
	// Create large route tables repeatedly, delete half of their
	// prefixes, and compare Lite's behavior to a naive and slow but
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

		// pfxs toDelete should be non-overlapping sets
		pfxs := slices.Concat(all4[:deleteCut], all6[:deleteCut])
		toDelete := slices.Concat(all4[deleteCut:], all6[deleteCut:])

		gold := new(goldTable[string])
		lite := new(Lite)

		for _, pfx := range pfxs {
			gold.insert(pfx, pfx.String())
			lite.Insert(pfx)
		}

		for _, pfx := range toDelete {
			gold.delete(pfx)
			lite.Delete(pfx)
		}

		liteGolden := dumpAsGoldTable[string](lite)

		if !slices.Equal(gold.allSorted(), liteGolden.allSorted()) {
			t.Fatal("expected Equal")
		}
	}
}

func TestLiteDeleteShuffled(t *testing.T) {
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

		// pfxs toDelete should be non-overlapping sets
		pfxs := slices.Concat(all4[:deleteCut], all6[:deleteCut])
		toDelete := slices.Concat(all4[deleteCut:], all6[deleteCut:])

		lite := new(Lite)

		// insert
		for _, pfx := range pfxs {
			lite.Insert(pfx)
		}
		for _, pfx := range toDelete {
			lite.Insert(pfx)
		}

		// delete
		for _, pfx := range toDelete {
			lite.Delete(pfx)
		}

		pfxs2 := slices.Clone(pfxs)
		toDelete2 := slices.Clone(toDelete)
		prng.Shuffle(len(toDelete2), func(i, j int) { toDelete2[i], toDelete2[j] = toDelete2[j], toDelete2[i] })

		lite2 := new(Lite)

		// insert
		for _, pfx := range pfxs2 {
			lite2.Insert(pfx)
		}
		for _, pfx := range toDelete2 {
			lite2.Insert(pfx)
		}

		// delete
		for _, pfx := range toDelete2 {
			lite2.Delete(pfx)
		}

		if !lite.Equal(lite2) {
			t.Fatal("expect equal")
		}
	}
}

func TestLiteDeleteIsReverseOfInsert(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	// Insert n prefixes, then delete those same prefixes in reverse
	// order. Each deletion should exactly undo the internal structure
	// changes that each insert did.

	lite := new(Lite)
	want := lite.dumpString()

	prefixes := randomRealWorldPrefixes(prng, n)

	defer func() {
		if t.Failed() {
			t.Logf("the prefixes that fail the test: %v\n", prefixes)
		}
	}()

	for _, p := range prefixes {
		lite.Insert(p)
	}

	for i := len(prefixes) - 1; i >= 0; i-- {
		lite.Delete(prefixes[i])
	}
	if got := lite.dumpString(); got != want {
		t.Fatalf("after delete, mismatch:\n\n got: %s\n\nwant: %s", got, want)
	}
}

func TestLiteDeleteButOne(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))
	// Insert n prefixes, then delete all but one
	n := workLoadN()

	for range 10 {

		lite := new(Lite)
		prefixes := randomRealWorldPrefixes(prng, n)

		for _, p := range prefixes {
			lite.Insert(p)
		}

		// shuffle the prefixes
		prng.Shuffle(n, func(i, j int) {
			prefixes[i], prefixes[j] = prefixes[j], prefixes[i]
		})

		// skip the first
		for i := 1; i < len(prefixes); i++ {
			lite.Delete(prefixes[i])
		}

		stats4 := lite.root4.StatsRec()
		stats6 := lite.root6.StatsRec()

		if nodes := stats4.Nodes + stats6.Nodes; nodes != 1 {
			t.Fatalf("delete but one, want nodes: 1, got: %d\n%s", nodes, lite.dumpString())
		}

		sum := stats4.Pfxs + stats4.Leaves + stats4.Fringes +
			stats6.Pfxs + stats6.Leaves + stats6.Fringes

		if sum != 1 {
			t.Fatalf("delete but one, only one item must be left, but: %d\n%s", sum, lite.dumpString())
		}
	}
}

func TestLiteGet(t *testing.T) {
	t.Parallel()

	t.Run("empty table", func(t *testing.T) {
		t.Parallel()
		prng := rand.New(rand.NewPCG(42, 42))

		lite := new(Lite)
		pfx := randomPrefix(prng)
		ok := lite.Get(pfx)

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

	lite := new(Lite)
	for _, tt := range tests {
		lite.Insert(tt.pfx)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ok := lite.Get(tt.pfx)

			if !ok {
				t.Errorf("%s: ok=%v, expected: %v", tt.name, ok, true)
			}
		})
	}
}

func TestLiteGetCompare(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[string])
	lite := new(Lite)
	for _, pfx := range pfxs {
		gold.insert(pfx, pfx.String())
		lite.Insert(pfx)
	}

	for _, pfx := range pfxs {
		_, goldOK := gold.get(pfx)
		liteOK := lite.Get(pfx)

		if goldOK != liteOK {
			t.Fatalf("Get(%q) = %v, want %v", pfx, liteOK, goldOK)
		}
	}
}

func TestLiteModifySemantics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		prepare   []netip.Prefix // entries to pre-populate the table
		modify    []netip.Prefix // entries to mofify
		cb        func(bool) bool
		finalData []netip.Prefix // expected table contents after the operation
	}{
		{
			name:      "Delete existing entries",
			prepare:   []netip.Prefix{mpp("10.0.0.0/8"), mpp("2001:db8::/32")},
			modify:    []netip.Prefix{mpp("10.0.0.0/8"), mpp("2001:db8::/32")},
			cb:        func(exists bool) bool { return true },
			finalData: []netip.Prefix{},
		},

		{
			name:      "Insert new entry",
			prepare:   nil,
			modify:    []netip.Prefix{mpp("10.0.0.0/8"), mpp("2001:db8::/32")},
			cb:        func(exists bool) bool { return false },
			finalData: []netip.Prefix{mpp("10.0.0.0/8"), mpp("2001:db8::/32")},
		},
	}

	for _, tt := range tests {
		lite := new(Lite)

		// Insert initial entries using Modify
		for _, pfx := range tt.prepare {
			lite.Modify(pfx, func(bool) bool { return false })
		}

		for _, pfx := range tt.modify {
			lite.Modify(pfx, tt.cb)
		}

		if lite.Size() != len(tt.finalData) {
			t.Fatalf("[%s] final table size mismatch: got %d, want %d", tt.name, lite.Size(), len(tt.finalData))
		}

		collect := []netip.Prefix{}
		for pfx := range lite.AllSorted() {
			collect = append(collect, pfx)
		}

		if !slices.Equal(collect, tt.finalData) {
			t.Fatalf("[%s] final table not equal expected", tt.name)
		}
	}
}

func TestLiteModifyPersistSemantics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		prepare   []netip.Prefix // entries to pre-populate the table
		modify    []netip.Prefix // entries to mofify
		cb        func(bool) bool
		finalData []netip.Prefix // expected table contents after the operation
	}{
		{
			name:      "Delete existing entries",
			prepare:   []netip.Prefix{mpp("10.0.0.0/8"), mpp("2001:db8::/32")},
			modify:    []netip.Prefix{mpp("10.0.0.0/8"), mpp("2001:db8::/32")},
			cb:        func(exists bool) bool { return true },
			finalData: []netip.Prefix{},
		},

		{
			name:      "Insert new entry",
			prepare:   nil,
			modify:    []netip.Prefix{mpp("10.0.0.0/8"), mpp("2001:db8::/32")},
			cb:        func(exists bool) bool { return false },
			finalData: []netip.Prefix{mpp("10.0.0.0/8"), mpp("2001:db8::/32")},
		},
	}

	for _, tt := range tests {
		lite := new(Lite)

		// Insert initial entries using Modify
		for _, pfx := range tt.prepare {
			lite = lite.ModifyPersist(pfx, func(bool) bool { return false })
		}

		for _, pfx := range tt.modify {
			lite = lite.ModifyPersist(pfx, tt.cb)
		}

		if lite.Size() != len(tt.finalData) {
			t.Fatalf("[%s] final table size mismatch: got %d, want %d", tt.name, lite.Size(), len(tt.finalData))
		}

		collect := []netip.Prefix{}
		for pfx := range lite.AllSorted() {
			collect = append(collect, pfx)
		}

		if !slices.Equal(collect, tt.finalData) {
			t.Fatalf("[%s] final table not equal expected", tt.name)
		}
	}
}

func TestLiteModifyCompare(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[int])
	lite := new(Lite)

	// Update as insert
	for i, pfx := range pfxs {
		gold.insert(pfx, i)
		lite.Modify(pfx, func(bool) bool { return false })
	}

	liteGolden := dumpAsGoldTable[int](lite)

	if !slices.Equal(gold.allSorted(), liteGolden.allSorted()) {
		t.Fatal("expected Equal")
	}
}

func TestLiteModifyPersistCompare(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	mut := new(Lite)
	imu := new(Lite)

	// Update as insert
	for _, pfx := range pfxs {
		mut.Modify(pfx, func(bool) bool { return false })
		imu = imu.ModifyPersist(pfx, func(bool) bool { return false })
	}

	if !mut.Equal(imu) {
		t.Fatal("expected Equal")
	}
}

func TestLiteModifyShuffled(t *testing.T) {
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

		lite1 := new(Lite)

		// insert
		for _, pfx := range pfxs {
			lite1.Insert(pfx)
		}
		for _, pfx := range toDelete {
			lite1.Insert(pfx)
		}

		// this callback deletes unconditionally
		cb := func(bool) bool { return true }

		// delete
		for _, pfx := range toDelete {
			lite1.Modify(pfx, cb)
		}

		pfxs2 := slices.Clone(pfxs)
		toDelete2 := slices.Clone(toDelete)
		prng.Shuffle(len(toDelete2), func(i, j int) { toDelete2[i], toDelete2[j] = toDelete2[j], toDelete2[i] })

		lite2 := new(Lite)

		// insert
		for _, pfx := range pfxs2 {
			lite2.Insert(pfx)
		}
		for _, pfx := range toDelete2 {
			lite2.Insert(pfx)
		}

		// delete
		for _, pfx := range toDelete2 {
			lite2.Modify(pfx, cb)
		}

		if !lite1.Equal(lite2) {
			t.Fatal("expected equal")
		}
	}
}

func TestLiteModifyPersistShuffled(t *testing.T) {
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

		lite1 := new(Lite)

		// insert
		for _, pfx := range pfxs {
			lite1.Insert(pfx)
		}
		for _, pfx := range toDelete {
			lite1.Insert(pfx)
		}

		// this callback deletes unconditionally
		cb := func(bool) bool { return true }

		// delete
		for _, pfx := range toDelete {
			lite1 = lite1.ModifyPersist(pfx, cb)
		}

		pfxs2 := slices.Clone(pfxs)
		toDelete2 := slices.Clone(toDelete)
		prng.Shuffle(len(toDelete2), func(i, j int) { toDelete2[i], toDelete2[j] = toDelete2[j], toDelete2[i] })

		lite2 := new(Lite)

		// insert
		for _, pfx := range pfxs2 {
			lite2.Insert(pfx)
		}
		for _, pfx := range toDelete2 {
			lite2.Insert(pfx)
		}

		// delete
		for _, pfx := range toDelete2 {
			lite2 = lite2.ModifyPersist(pfx, cb)
		}

		if !lite1.Equal(lite2) {
			t.Fatal("expected equal")
		}
	}
}

// TestUnionMemoryAliasing tests that the Union method does not alias memory
// between the two tables.
func TestLiteUnionMemoryAliasing(t *testing.T) {
	t.Parallel()

	newLite := func(pfx ...string) *Lite {
		rt := new(Lite)
		for _, s := range pfx {
			rt.Insert(mpp(s))
		}
		return rt
	}

	// First create two tables with disjoint prefixes.
	stable := newLite("0.0.0.0/24")
	temp := newLite("100.69.1.0/24")

	// Verify that the tables are disjoint.
	if stable.Overlaps(temp) {
		t.Error("stable should not overlap temp")
	}

	// Now union them.
	temp.Union(stable)

	// Add a new prefix to temp.
	temp.Insert(mpp("0.0.1.0/24"))

	// Ensure that stable is unchanged.
	ok := stable.Lookup(mpa("0.0.1.1"))
	if ok {
		t.Error("stable should not contain 0.0.1.1")
	}
	if stable.OverlapsPrefix(mpp("0.0.1.1/32")) {
		t.Error("stable should not overlap 0.0.1.1/32")
	}
}

// TestUnionPersistMemoryAliasing tests that the Union method does not alias memory
// between the tables.
func TestLiteUnionPersistMemoryAliasing(t *testing.T) {
	t.Parallel()

	newLite := func(pfx ...string) *Lite {
		rt := new(Lite)
		for _, s := range pfx {
			rt.Insert(mpp(s))
		}
		return rt
	}
	// First create two tables with disjoint prefixes.
	a := newLite("100.69.1.0/24")
	b := newLite("0.0.0.0/24")

	// Verify that the tables are disjoint.
	if a.Overlaps(b) {
		t.Error("this should not overlap other")
	}

	// Now union them with copy-on-write.
	pTbl := a.UnionPersist(b)

	// Add a new prefix to new union
	pTbl.Insert(mpp("0.0.1.0/24"))

	// Ensure that a is unchanged.
	ok := a.Lookup(mpa("0.0.1.1"))
	if ok {
		t.Error("a should not contain 0.0.1.1")
	}
	if a.OverlapsPrefix(mpp("0.0.1.1/32")) {
		t.Error("a should not overlap 0.0.1.1/32")
	}
}

func TestLiteUnionCompare(t *testing.T) {
	t.Parallel()

	n := workLoadN()
	prng := rand.New(rand.NewPCG(42, 42))

	for range 3 {
		pfxs := randomRealWorldPrefixes(prng, n)

		gold := new(goldTable[any])
		lite := new(Lite)

		for _, pfx := range pfxs {
			gold.insert(pfx, nil)
			lite.Insert(pfx)
		}

		pfxs2 := randomRealWorldPrefixes(prng, n)

		gold2 := new(goldTable[any])
		lite2 := new(Lite)

		for _, pfx := range pfxs2 {
			gold2.insert(pfx, nil)
			lite2.Insert(pfx)
		}

		gold.union(gold2)
		lite.Union(lite2)

		// dump as slow table for comparison
		liteAsGoldenTbl := dumpAsGoldTable[any](lite)

		if !slices.Equal(gold.allSorted(), liteAsGoldenTbl.allSorted()) {
			t.Fatal("expected equal")
		}
	}
}

func TestLiteUnionPersistCompare(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))

	n := workLoadN()

	for range 3 {
		pfxs := randomRealWorldPrefixes(prng, n)

		gold := new(goldTable[int])
		lite := new(Lite)

		for i, pfx := range pfxs {
			gold.insert(pfx, i)
			lite.Insert(pfx)
		}

		pfxs2 := randomRealWorldPrefixes(prng, n)

		gold2 := new(goldTable[int])
		lite2 := new(Lite)

		for i, pfx := range pfxs2 {
			gold2.insert(pfx, i)
			lite2.Insert(pfx)
		}

		gold.union(gold2)
		liteP := lite.UnionPersist(lite2)

		// dump as slow table for comparison
		liteAsGoldenTbl := dumpAsGoldTable[int](liteP)

		// sort for comparison
		if !slices.Equal(gold.allSorted(), liteAsGoldenTbl.allSorted()) {
			t.Fatal("expected equal")
		}
	}
}

func TestLiteClone(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, 100_000)

	var lite *Lite
	if lite.Clone() != nil {
		t.Fatal("expected nil")
	}

	lite = new(Lite)
	for _, pfx := range pfxs {
		lite.Insert(pfx)
	}
	clone := lite.Clone()

	if !lite.Equal(clone) {
		t.Fatal("expected equal")
	}
}

// test some edge cases
func TestLiteOverlapsPrefixEdgeCases(t *testing.T) {
	t.Parallel()

	lite := new(Lite)

	// empty table
	checkOverlapsPrefix(t, lite, []tableOverlapsTest{
		{"0.0.0.0/0", false},
		{"::/0", false},
	})

	// default route
	lite.Insert(mpp("10.0.0.0/9"))
	lite.Insert(mpp("2001:db8::/32"))
	checkOverlapsPrefix(t, lite, []tableOverlapsTest{
		{"0.0.0.0/0", true},
		{"::/0", true},
	})

	// default route
	lite = new(Lite)
	lite.Insert(mpp("0.0.0.0/0"))
	lite.Insert(mpp("::/0"))
	checkOverlapsPrefix(t, lite, []tableOverlapsTest{
		{"10.0.0.0/9", true},
		{"2001:db8::/32", true},
	})

	// single IP
	lite = new(Lite)
	lite.Insert(mpp("10.0.0.0/7"))
	lite.Insert(mpp("2001::/16"))
	checkOverlapsPrefix(t, lite, []tableOverlapsTest{
		{"10.1.2.3/32", true},
		{"2001:db8:affe::cafe/128", true},
	})

	// single IP
	lite = new(Lite)
	lite.Insert(mpp("10.1.2.3/32"))
	lite.Insert(mpp("2001:db8:affe::cafe/128"))
	checkOverlapsPrefix(t, lite, []tableOverlapsTest{
		{"10.0.0.0/7", true},
		{"2001::/16", true},
	})

	// same IPv
	lite = new(Lite)
	lite.Insert(mpp("10.1.2.3/32"))
	lite.Insert(mpp("2001:db8:affe::cafe/128"))
	checkOverlapsPrefix(t, lite, []tableOverlapsTest{
		{"10.1.2.3/32", true},
		{"2001:db8:affe::cafe/128", true},
	})
}

func TestLiteSize(t *testing.T) {
	t.Parallel()

	n := workLoadN()
	prng := rand.New(rand.NewPCG(42, 42))

	lite := new(Lite)
	if lite.Size() != 0 {
		t.Errorf("empty Lite: want: 0, got: %d", lite.Size())
	}

	if lite.Size4() != 0 {
		t.Errorf("empty Lite: want: 0, got: %d", lite.Size4())
	}

	if lite.Size6() != 0 {
		t.Errorf("empty Lite: want: 0, got: %d", lite.Size6())
	}

	pfxs1 := randomRealWorldPrefixes(prng, n)
	pfxs2 := randomRealWorldPrefixes(prng, n)

	for _, pfx := range pfxs1 {
		lite.Insert(pfx)
	}

	for _, pfx := range pfxs2 {
		lite.Modify(pfx, func(bool) bool { return false })
	}

	pfxs1 = append(pfxs1, pfxs2...)

	for _, pfx := range pfxs1[:n] {
		lite.Modify(pfx, func(bool) bool { return false })
	}

	for _, pfx := range randomRealWorldPrefixes(prng, n) {
		lite.Delete(pfx)
	}

	var allInc4 int
	var allInc6 int

	for range lite.AllSorted4() {
		allInc4++
	}

	for range lite.AllSorted6() {
		allInc6++
	}

	if allInc4 != lite.Size4() {
		t.Errorf("Size4: want: %d, got: %d", allInc4, lite.Size4())
	}

	if allInc6 != lite.Size6() {
		t.Errorf("Size6: want: %d, got: %d", allInc6, lite.Size6())
	}
}

// TestAll tests All with random samples
func TestLiteAll(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	lite := new(Lite)

	// Insert all prefixes with their values
	for _, pfx := range pfxs {
		lite.Insert(pfx)
	}

	// Collect all prefixes from All
	gotPrefixes := make([]netip.Prefix, 0, n)

	for pfx := range lite.All() {
		gotPrefixes = append(gotPrefixes, pfx)
	}

	// Collect all prefixes from All4 and All6
	got4Prefixes := make([]netip.Prefix, 0, n)

	for pfx := range lite.All4() {
		got4Prefixes = append(got4Prefixes, pfx)
	}

	got6Prefixes := make([]netip.Prefix, 0, n)
	for pfx := range lite.All6() {
		got6Prefixes = append(got6Prefixes, pfx)
	}

	// Verify we got exactly the same number of prefixes
	if len(gotPrefixes) != len(pfxs) {
		t.Fatalf("Expected %d prefixes, got %d", len(pfxs), len(gotPrefixes))
	}

	if len(got4Prefixes)+len(got6Prefixes) != len(pfxs) {
		t.Fatalf("Expected %d prefixes, got %d", len(pfxs), len(got4Prefixes)+len(got6Prefixes))
	}

	if !slices.Equal(slices.Concat(got4Prefixes, got6Prefixes), gotPrefixes) {
		t.Fatal("Prefixes: All4 + All6 != All")
	}

	// Verify no duplicates in results (since randomRealWorldPrefixes guarantees no duplicates)
	seen := make(map[netip.Prefix]bool, n)
	for _, pfx := range gotPrefixes {
		if seen[pfx] {
			t.Fatalf("Duplicate prefix %v found in All results", pfx)
		}
		seen[pfx] = true
	}
}

func TestLiteAllSorted(t *testing.T) {
	t.Parallel()

	// Test cases with known CIDR sort order
	testCases := []struct {
		name     string
		prefixes []string
		expected []string // Expected order after sorting
	}{
		{
			name: "Mixed IPv4 addresses and prefix lengths",
			prefixes: []string{
				"10.0.0.0/16",
				"10.0.0.0/8",
				"192.168.1.0/24",
				"10.0.0.0/24",
				"172.16.0.0/12",
			},
			expected: []string{
				"10.0.0.0/8",     // Same address, shorter prefix first
				"10.0.0.0/16",    // Same address, longer prefix
				"10.0.0.0/24",    // Same address, longest prefix
				"172.16.0.0/12",  // Next address
				"192.168.1.0/24", // Highest address
			},
		},
		{
			name: "Mixed IPv6 addresses and prefix lengths",
			prefixes: []string{
				"2001:db8::/32",
				"2001:db8::/64",
				"2000::/16",
				"2001:db8:1::/48",
			},
			expected: []string{
				"2000::/16",       // Lowest address
				"2001:db8::/32",   // Same address, shorter prefix first
				"2001:db8::/64",   // Same address, longer prefix
				"2001:db8:1::/48", // Higher address
			},
		},
		{
			name: "Mixed IPv4 and IPv6",
			prefixes: []string{
				"192.168.1.0/24",
				"2001:db8::/32",
				"10.0.0.0/8",
				"::1/128",
			},
			expected: []string{
				"10.0.0.0/8",     // IPv4 addresses come first (lower in comparison)
				"192.168.1.0/24", // Next IPv4 address
				"::1/128",        // IPv6 addresses after IPv4
				"2001:db8::/32",  // Higher IPv6 address
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			lite := new(Lite)

			// Insert prefixes with index as value
			for _, prefixStr := range tc.prefixes {
				pfx := mpp(prefixStr)
				lite.Insert(pfx)
			}

			// Collect sorted results
			var actualOrder []string
			for pfx := range lite.AllSorted() {
				actualOrder = append(actualOrder, pfx.String())
			}

			// Verify the order matches expected
			if len(actualOrder) != len(tc.expected) {
				t.Fatalf("%s: Expected %d results, got %d", tc.name, len(tc.expected), len(actualOrder))
			}

			// Collect sorted 4 results
			var actual4Order []string
			for pfx := range lite.AllSorted4() {
				actual4Order = append(actual4Order, pfx.String())
			}

			// Collect sorted 6 results
			var actual6Order []string
			for pfx := range lite.AllSorted6() {
				actual6Order = append(actual6Order, pfx.String())
			}

			if !slices.Equal(slices.Concat(actual4Order, actual6Order), actualOrder) {
				t.Fatalf("%s: Prefixes: AllSorted4 + AllSorted6 != AllSorted", tc.name)
			}

			for i, expected := range tc.expected {
				if actualOrder[i] != expected {
					t.Errorf("%s:At position %d: expected %s, got %s", tc.name, i, expected, actualOrder[i])
					t.Errorf("%s:Full expected order: %v", tc.name, tc.expected)
					t.Errorf("%s:Full actual order:   %v", tc.name, actualOrder)
					break
				}
			}
		})
	}
}

func TestLiteSubnets(t *testing.T) {
	t.Parallel()

	var zeroPfx netip.Prefix

	t.Run("empty table", func(t *testing.T) {
		t.Parallel()
		lite := new(Lite)
		pfx := mpp("::1/128")

		for range lite.Subnets(pfx) {
			t.Errorf("empty table, must not range over")
		}
	})

	t.Run("invalid prefix", func(t *testing.T) {
		lite := new(Lite)
		pfx := mpp("::1/128")
		lite.Insert(pfx)
		for range lite.Subnets(zeroPfx) {
			t.Errorf("invalid prefix, must not range over")
		}
	})

	t.Run("identity", func(t *testing.T) {
		lite := new(Lite)
		pfx := mpp("::1/128")
		lite.Insert(pfx)

		for p := range lite.Subnets(pfx) {
			if p != pfx {
				t.Errorf("Subnet(%v), got: %v, want: %v", pfx, p, pfx)
			}
		}
	})

	t.Run("default gateway", func(t *testing.T) {
		n := workLoadN()
		prng := rand.New(rand.NewPCG(42, 42))

		want4 := n - n/2
		want6 := n + n/2

		lite := new(Lite)
		for _, pfx := range randomRealWorldPrefixes4(prng, want4) {
			lite.Insert(pfx)
		}
		for _, pfx := range randomRealWorldPrefixes6(prng, want6) {
			lite.Insert(pfx)
		}

		// default gateway v4 covers all v4 prefixes in table
		dg4 := mpp("0.0.0.0/0")
		got4 := 0
		for range lite.Subnets(dg4) {
			got4++
		}

		// default gateway v6 covers all v6 prefixes in table
		dg6 := mpp("::/0")
		got6 := 0
		for range lite.Subnets(dg6) {
			got6++
		}

		if got4 != want4 {
			t.Errorf("Subnets v4, want: %d, got: %d", want4, got4)
		}
		if got6 != want6 {
			t.Errorf("Subnets v6, want: %d, got: %d", want6, got6)
		}
	})
}

func TestLiteSubnetsCompare(t *testing.T) {
	t.Parallel()
	n := workLoadN()
	prng := rand.New(rand.NewPCG(42, 42))

	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[int])
	lite := new(Lite)

	for i, pfx := range pfxs {
		gold.insert(pfx, i)
		lite.Insert(pfx)
	}

	for _, pfx := range randomRealWorldPrefixes(prng, n) {
		t.Run("subtest", func(t *testing.T) {
			t.Parallel()

			gotGold := gold.subnets(pfx)
			gotBart := []netip.Prefix{}
			for pfx := range lite.Subnets(pfx) {
				gotBart = append(gotBart, pfx)
			}
			if !slices.Equal(gotGold, gotBart) {
				t.Fatalf("Subnets(%q) = %v, want %v", pfx, gotBart, gotGold)
			}
		})
	}
}

func TestLiteSupernetsEdgeCase(t *testing.T) {
	t.Parallel()

	var zeroPfx netip.Prefix

	t.Run("empty table", func(t *testing.T) {
		t.Parallel()
		lite := new(Lite)
		pfx := mpp("::1/128")

		lite.Supernets(pfx)(func(netip.Prefix) bool {
			t.Errorf("empty table, must not range over")
			return false
		})
	})

	t.Run("invalid prefix", func(t *testing.T) {
		lite := new(Lite)
		pfx := mpp("::1/128")
		lite.Insert(pfx)

		lite.Supernets(zeroPfx)(func(netip.Prefix) bool {
			t.Errorf("invalid prefix, must not range over")
			return false
		})
	})

	t.Run("identity", func(t *testing.T) {
		lite := new(Lite)
		pfx := mpp("::1/128")
		lite.Insert(pfx)

		for p := range lite.Supernets(pfx) {
			if p != pfx {
				t.Errorf("Supernets(%v), got: %v, want: %v", pfx, p, pfx)
			}
		}
	})
}

func TestLiteSupernetsCompare(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))

	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[int])
	lite := new(Lite)

	for i, pfx := range pfxs {
		gold.insert(pfx, i)
		lite.Insert(pfx)
	}

	for _, pfx := range randomRealWorldPrefixes(prng, n) {
		t.Run("subtest", func(t *testing.T) {
			t.Parallel()
			gotGold := gold.supernets(pfx)
			gotBart := []netip.Prefix{}

			for p := range lite.Supernets(pfx) {
				gotBart = append(gotBart, p)
			}

			if !slices.Equal(gotGold, gotBart) {
				t.Fatalf("Supernets(%q) = %v, want %v", pfx, gotBart, gotGold)
			}
		})
	}
}

func TestLiteMarshalText(t *testing.T) {
	tests := []struct {
		name         string
		expectedData []string
	}{
		{
			name:         "empty",
			expectedData: []string{},
		},
		{
			name: "with_data",
			expectedData: []string{
				"192.168.1.0/24",
				"10.0.0.0/8",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lite := new(Lite)

			// Insert test data
			for _, prefix := range tt.expectedData {
				lite.Insert(mpp(prefix))
			}

			data, err := lite.MarshalText()
			if err != nil {
				t.Errorf("MarshalText failed: %v", err)
			}

			if len(tt.expectedData) > 0 && len(data) == 0 {
				t.Error("Expected non-empty marshaled text")
			}

			// Check that all expected values appear in marshaled text
			text := string(data)
			for _, prefix := range tt.expectedData {
				if !strings.Contains(text, prefix) {
					t.Errorf("Marshaled text doesn't contain expected value: %s", prefix)
				}
			}
		})
	}
}

func TestLiteMarshalJSON(t *testing.T) {
	tests := []struct {
		name         string
		expectedData []string
	}{
		{
			name:         "empty",
			expectedData: []string{},
		},
		{
			name: "string_values",
			expectedData: []string{
				"192.168.1.0/24",
				"10.0.0.0/8",
			},
		},
		{
			name: "mixed_values",
			expectedData: []string{
				"192.168.1.0/24",
				"10.0.0.0/8",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lite := new(Lite)

			// Insert test data
			for _, prefix := range tt.expectedData {
				lite.Insert(mpp(prefix))
			}

			jsonData, err := json.Marshal(lite)
			if err != nil {
				t.Errorf("JSON marshaling failed: %v", err)
			}

			if len(jsonData) == 0 {
				t.Error("Expected valid JSON")
			}

			// Should be valid JSON
			var result interface{}
			if err := json.Unmarshal(jsonData, &result); err != nil {
				t.Errorf("Invalid JSON produced: %v", err)
			}
		})
	}
}

func TestLiteDumpList4(t *testing.T) {
	tests := []struct {
		name         string
		expectedData []string
		expectItems  int
	}{
		{
			name:         "empty",
			expectedData: []string{},
			expectItems:  0,
		},
		{
			name: "single_ipv4",
			expectedData: []string{
				"192.168.1.0/24",
			},
			expectItems: 1,
		},
		{
			name: "multiple_ipv4",
			expectedData: []string{
				"192.168.1.0/24",
				"10.0.0.0/8",
			},
			expectItems: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lite := new(Lite)

			// Insert test data
			for _, prefix := range tt.expectedData {
				lite.Insert(mpp(prefix))
			}

			dumpList := lite.DumpList4()

			// Count total nodes in the tree (including nested)
			totalNodes := countDumpListNodes(dumpList)
			if totalNodes != tt.expectItems {
				t.Errorf("DumpList4() total nodes (%d) does not match expected (%d)", totalNodes, tt.expectItems)
			}

			// Verify all nodes are IPv4
			verifyAllIPv4Nodes(t, dumpList)
		})
	}
}

func TestLiteDumpList6(t *testing.T) {
	tests := []struct {
		name         string
		expectedData []string
		expectItems  int
	}{
		{
			name:         "empty",
			expectedData: []string{},
			expectItems:  0,
		},
		{
			name: "single_ipv6",
			expectedData: []string{
				"2001:db8::/32",
			},
			expectItems: 1,
		},
		{
			name: "multiple_ipv6",
			expectedData: []string{
				"2001:db8::/32",
				"fe80::/10",
			},
			expectItems: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lite := new(Lite)

			// Insert test data
			for _, prefix := range tt.expectedData {
				lite.Insert(mpp(prefix))
			}

			dumpList := lite.DumpList6()

			// Count total nodes in the tree (including nested)
			totalNodes := countDumpListNodes(dumpList)
			if totalNodes != tt.expectItems {
				t.Errorf("DumpList6() total nodes (%d) does not match expected (%d)", totalNodes, tt.expectItems)
			}

			// Verify all nodes are IPv6
			verifyAllIPv6Nodes(t, dumpList)
		})
	}
}

func TestLiteEqual(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		buildA    func() *Lite
		buildB    func() *Lite
		wantEqual bool
	}{
		{
			name:      "empty tables",
			buildA:    func() *Lite { return new(Lite) },
			buildB:    func() *Lite { return new(Lite) },
			wantEqual: true,
		},
		{
			name: "same single entry",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.0/24"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.0/24"))
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "different entries",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.0/24"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("2001:db8::/32"))
				return tbl
			},
			wantEqual: false,
		},
		{
			name: "same entries, different insert order",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.0/24"))
				tbl.Insert(mpp("198.51.100.0/24"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("198.51.100.0/24"))
				tbl.Insert(mpp("192.0.2.0/24"))
				return tbl
			},
			wantEqual: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			a := tc.buildA()
			b := tc.buildB()

			got := a.Equal(b)
			if got != tc.wantEqual {
				t.Errorf("Equal() = %v, want %v", got, tc.wantEqual)
			}
		})
	}
}

func TestLiteFullEqual(t *testing.T) {
	t.Parallel()
	at := new(Lite)
	for _, r := range routes {
		at.Insert(r.CIDR)
	}

	t.Run("clone", func(t *testing.T) {
		t.Parallel()
		bt := at.Clone()
		if !at.Equal(bt) {
			t.Error("expected true, got false")
		}
	})
}
