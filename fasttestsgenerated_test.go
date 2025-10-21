// Code generated from file "aa-alltests_tmpl.go"; DO NOT EDIT.

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

	"github.com/gaissmai/bart/internal/tests/golden"
	"github.com/gaissmai/bart/internal/tests/random"
)

// ############ tests ################################

// flatSorted, just a helper to compare with golden table.
func (t *Fast[V]) flatSorted() golden.Table[V] {
	var flat golden.Table[V]

	for p, v := range t.AllSorted() {
		flat = append(flat, golden.TableItem[V]{Pfx: p, Val: v})
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
	pfxs := random.RealWorldPrefixes(prng, n)

	gold := new(golden.Table[int])
	tbl := new(Fast[int])

	for i, p := range pfxs {
		gold.Insert(p, i)
		tbl.Insert(p, i)
	}

	for range n {
		ip := random.IP(prng)

		_, goldOK := gold.Lookup(ip)
		tblOK := tbl.Contains(ip)

		if goldOK != tblOK {
			t.Fatalf("Contains(%q) = %v, want %v", ip, tblOK, goldOK)
		}
	}
}

func TestTableLookupCompare_Fast(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := random.RealWorldPrefixes(prng, n)

	gold := new(golden.Table[int])
	tbl := new(Fast[int])

	for i, pfx := range pfxs {
		gold.Insert(pfx, i)
		tbl.Insert(pfx, i)
	}

	for range n {
		ip := random.IP(prng)

		goldVal, goldOK := gold.Lookup(ip)
		tblVal, tblOK := tbl.Lookup(ip)

		if goldOK != tblOK {
			t.Fatalf("Lookup(%q) = (_, %v), want %v", ip, tblOK, goldOK)
		}

		// Skip value comparison for liteTable (no real payload)
		if _, isLite := any(tbl).(*liteTable[int]); !isLite {
			if goldVal != tblVal {
				t.Fatalf("Lookup(%q) = (%v, %v), want (%v, %v)", ip, tblVal, tblOK, goldVal, goldOK)
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
	pfxs := random.RealWorldPrefixes(prng, n)

	gold := new(golden.Table[int])
	tbl := new(Fast[int])
	for i, pfx := range pfxs {
		gold.Insert(pfx, i)
		tbl.Insert(pfx, i)
	}

	for range n {
		pfx := random.Prefix(prng)

		goldVal, goldOK := gold.LookupPrefix(pfx)
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
	pfxs := random.RealWorldPrefixes(prng, n)

	gold := new(golden.Table[int])
	tbl := new(Fast[int])
	for i, pfx := range pfxs {
		gold.Insert(pfx, i)
		tbl.Insert(pfx, i)
	}

	for range n {
		pfx := random.Prefix(prng)

		goldLPM, goldVal, goldOK := gold.LookupPrefixLPM(pfx)
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
	pfxs := random.RealWorldPrefixes(prng, n)

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
	pfxs := random.RealWorldPrefixes(prng, n)

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
		all4 := random.RealWorldPrefixes4(prng, numPerFamily)
		all6 := random.RealWorldPrefixes6(prng, numPerFamily)

		pfxs := slices.Concat(all4, all6)
		toDelete := slices.Concat(all4[deleteCut:], all6[deleteCut:])

		gold := new(golden.Table[string])
		tbl := new(Fast[string])

		for _, pfx := range pfxs {
			gold.Insert(pfx, pfx.String())
			tbl.Insert(pfx, pfx.String())
		}

		for _, pfx := range toDelete {
			gold.Delete(pfx)
			tbl.Delete(pfx)
		}

		gold.Sort()

		tblFlat := tbl.flatSorted()

		// Skip value comparison for liteTable (no real payload)
		if _, isLite := any(tbl).(*liteTable[string]); isLite {
			if !slices.Equal(gold.AllSorted(), tblFlat.AllSorted()) {
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
		all4 := random.RealWorldPrefixes4(prng, numPerFamily)
		all6 := random.RealWorldPrefixes6(prng, numPerFamily)

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

	prefixes := random.RealWorldPrefixes(prng, n)

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
		prefixes := random.RealWorldPrefixes(prng, n)

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
	pfx := random.Prefix(prng)

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
	pfxs := random.RealWorldPrefixes(prng, n)

	gold := new(golden.Table[string])
	tbl := new(Fast[string])
	for _, pfx := range pfxs {
		gold.Insert(pfx, pfx.String())
		tbl.Insert(pfx, pfx.String())
	}

	for _, pfx := range pfxs {
		goldVal, goldOK := gold.Get(pfx)
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

	if _, isLite := any(new(Fast[any])).(*liteTable[any]); isLite {
		t.Skip("liteNode has no real payload")
	}

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

		// Ensure the original table isn't modified
		for pfx, wantVal := range tt.prepare {
			val, ok := tbl.Get(pfx)
			if !ok {
				t.Errorf("[%s] original table: key %v should be present", tt.name, pfx)
			}

			if val != wantVal {
				t.Errorf("[%s] original table: val %v is not as expected %v", tt.name, val, wantVal)
			}
		}
	}
}

func TestTableModifyCompare_Fast(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := random.RealWorldPrefixes(prng, n)

	gold := new(golden.Table[int])
	tbl := new(Fast[int])

	// Update as insert
	for i, pfx := range pfxs {
		gold.Insert(pfx, i)
		tbl.Modify(pfx, func(int, bool) (int, bool) { return i, false })
	}

	gold.Sort()
	tblFlat := tbl.flatSorted()

	// Skip value comparison for liteTable (no real payload)
	if _, isLite := any(tbl).(*liteTable[int]); isLite {
		if !slices.Equal(gold.AllSorted(), tblFlat.AllSorted()) {
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
		gold.Update(pfx, cb1)
		tbl.Modify(pfx, cb2)
	}

	gold.Sort()
	tblFlat = tbl.flatSorted()

	// Skip value comparison for liteTable (no real payload)
	if _, isLite := any(tbl).(*liteTable[int]); isLite {
		if !slices.Equal(gold.AllSorted(), tblFlat.AllSorted()) {
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
	pfxs := random.RealWorldPrefixes(prng, n)

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
		all4 := random.RealWorldPrefixes4(prng, numPerFamily)
		all6 := random.RealWorldPrefixes6(prng, numPerFamily)

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
		all4 := random.RealWorldPrefixes4(prng, numPerFamily)
		all6 := random.RealWorldPrefixes6(prng, numPerFamily)

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

	newTable := func(pfx ...string) *Fast[any] {
		tbl := new(Fast[any])
		for _, s := range pfx {
			tbl.Insert(mpp(s), nil)
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

	newTable := func(pfx ...string) *Fast[any] {
		tbl := new(Fast[any])
		for _, s := range pfx {
			tbl.Insert(mpp(s), nil)
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
		pfxs := random.RealWorldPrefixes(prng, n)

		gold := new(golden.Table[string])
		tbl := new(Fast[string])

		for _, pfx := range pfxs {
			gold.Insert(pfx, pfx.String())
			tbl.Insert(pfx, pfx.String())
		}

		pfxs2 := random.RealWorldPrefixes(prng, n)

		gold2 := new(golden.Table[string])
		tbl2 := new(Fast[string])

		for _, pfx := range pfxs2 {
			gold2.Insert(pfx, pfx.String())
			tbl2.Insert(pfx, pfx.String())
		}

		gold.Union(gold2)
		tbl.Union(tbl2)

		// dump as slow table for comparison
		tblFlat := tbl.flatSorted()

		// sort for comparison
		gold.Sort()

		// Skip value comparison for liteTable (no real payload)
		if _, isLite := any(tbl).(*liteTable[string]); isLite {
			if !slices.Equal(gold.AllSorted(), tblFlat.AllSorted()) {
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
		pfxs := random.RealWorldPrefixes(prng, n)

		gold := new(golden.Table[int])
		tbl := new(Fast[int])

		for i, pfx := range pfxs {
			gold.Insert(pfx, i)
			tbl.Insert(pfx, i)
		}

		pfxs2 := random.RealWorldPrefixes(prng, n)

		gold2 := new(golden.Table[int])
		tbl2 := new(Fast[int])

		for i, pfx := range pfxs2 {
			gold2.Insert(pfx, i)
			tbl2.Insert(pfx, i)
		}

		gold.Union(gold2)
		tblP := tbl.UnionPersist(tbl2)

		// dump as slow table for comparison
		flatP := tblP.flatSorted()

		// sort for comparison
		gold.Sort()

		// Skip value comparison for liteTable (no real payload)
		if _, isLite := any(tbl).(*liteTable[int]); isLite {
			if !slices.Equal(gold.AllSorted(), flatP.AllSorted()) {
				t.Fatal("expected Equal")
			}
		} else {
			if !slices.Equal(*gold, flatP) {
				t.Fatal("expected Equal")
			}
		}
	}
}

func TestTableClone_Fast(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := random.RealWorldPrefixes(prng, n)

	var tbl *Fast[int]
	if tbl.Clone() != nil {
		t.Fatal("expected nil")
	}

	tbl = new(Fast[int])
	for i, pfx := range pfxs {
		tbl.Insert(pfx, i)
	}
	clone := tbl.Clone()

	if !tbl.Equal(clone) {
		t.Fatal("expected equal")
	}
}

func TestTableCloneShallow_Fast(t *testing.T) {
	t.Parallel()

	tbl := new(Fast[*int])

	if _, isLite := any(tbl).(*liteTable[*int]); isLite {
		t.Skip("liteNode has no real payload")
	}

	clone := tbl.Clone()
	if tbl.dumpString() != clone.dumpString() {
		t.Errorf("empty Clone: got:\n%swant:\n%s", clone.dumpString(), tbl.dumpString())
	}

	val := 1
	pfx := mpp("10.0.0.1/32")
	tbl.Insert(pfx, &val)

	clone = tbl.Clone()
	want, _ := tbl.Get(pfx)
	got, _ := clone.Get(pfx)

	if *got != *want || got != want {
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

func TestTableCloneDeep_Fast(t *testing.T) {
	t.Parallel()

	tbl := new(Fast[*MyInt])

	if _, isLite := any(tbl).(*liteTable[*MyInt]); isLite {
		t.Skip("liteNode has no real payload")
	}

	clone := tbl.Clone()
	if tbl.dumpString() != clone.dumpString() {
		t.Errorf("empty Clone: got:\n%swant:\n%s", clone.dumpString(), tbl.dumpString())
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

func TestTableUnionShallow_Fast(t *testing.T) {
	t.Parallel()

	tbl1 := new(Fast[*int])
	tbl2 := new(Fast[*int])

	if _, isLite := any(tbl1).(*liteTable[*int]); isLite {
		t.Skip("liteNode has no real payload")
	}

	val := 1
	pfx := mpp("10.0.0.1/32")
	tbl2.Insert(pfx, &val)

	tbl1.Union(tbl2)
	got, _ := tbl1.Get(pfx)
	want, _ := tbl2.Get(pfx)

	if *got != *want || got != want {
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

func TestTableUnionDeep_Fast(t *testing.T) {
	t.Parallel()

	tbl1 := new(Fast[*MyInt])
	tbl2 := new(Fast[*MyInt])

	if _, isLite := any(tbl1).(*liteTable[*MyInt]); isLite {
		t.Skip("liteNode has no real payload")
	}

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
func TestTableOverlapsPrefixEdgeCases_Fast(t *testing.T) {
	t.Parallel()

	type probe struct {
		pfx  netip.Prefix
		want bool
	}

	type probes []probe
	type pfxs []netip.Prefix

	type test struct {
		name   string
		insert pfxs
		probes probes
	}

	tests := []test{
		{
			name:   "empty table",
			insert: nil,
			probes: probes{{mpp("0.0.0.0/0"), false}, {mpp("::/0"), false}},
		},
		{
			name:   "default route I",
			insert: pfxs{mpp("10.0.0.0/9"), mpp("2001:db8::/32")},
			probes: probes{{mpp("0.0.0.0/0"), true}, {mpp("::/0"), true}},
		},
		{
			name:   "default route II",
			insert: pfxs{mpp("0.0.0.0/0"), mpp("::/0")},
			probes: probes{{mpp("10.0.0.0/9"), true}, {mpp("2001:db8::/32"), true}},
		},
		{
			name:   "single IP I",
			insert: pfxs{mpp("10.0.0.0/7"), mpp("2001::/16")},
			probes: probes{{mpp("10.1.2.3/32"), true}, {mpp("2001:db8:affe::cafe/128"), true}},
		},
		{
			name:   "single IP II",
			insert: pfxs{mpp("10.1.2.3/32"), mpp("2001:db8:affe::cafe/128")},
			probes: probes{{mpp("10.0.0.0/7"), true}, {mpp("2001::/16"), true}},
		},
		{
			name:   "same IP",
			insert: pfxs{mpp("10.1.2.3/32"), mpp("2001:db8:affe::cafe/128")},
			probes: probes{{mpp("10.1.2.3/32"), true}, {mpp("2001:db8:affe::cafe/128"), true}},
		},
	}

	for _, tt := range tests {
		tbl := new(Fast[int])
		for _, pfx := range tt.insert {
			tbl.Insert(pfx, 0)
		}

		for _, probe := range tt.probes {
			got := tbl.OverlapsPrefix(probe.pfx)
			if got != probe.want {
				t.Errorf("[%s] OverlapsPrefix(%v) = %v, want %v", tt.name, probe.pfx, got, probe.want)
			}
		}
	}
}

func TestTableSize_Fast(t *testing.T) {
	t.Parallel()

	n := workLoadN()
	prng := rand.New(rand.NewPCG(42, 42))

	tbl := new(Fast[any])
	if tbl.Size() != 0 {
		t.Errorf("empty Table: want: 0, got: %d", tbl.Size())
	}

	if tbl.Size4() != 0 {
		t.Errorf("empty Table: want: 0, got: %d", tbl.Size4())
	}

	if tbl.Size6() != 0 {
		t.Errorf("empty Table: want: 0, got: %d", tbl.Size6())
	}

	pfxs1 := random.RealWorldPrefixes(prng, n)
	pfxs2 := random.RealWorldPrefixes(prng, n)

	for _, pfx := range pfxs1 {
		tbl.Insert(pfx, nil)
	}

	for _, pfx := range pfxs2 {
		tbl.Modify(pfx, func(any, bool) (any, bool) { return nil, false })
	}

	pfxs1 = append(pfxs1, pfxs2...)

	for _, pfx := range pfxs1[:n] {
		tbl.Modify(pfx, func(any, bool) (any, bool) { return nil, false })
	}

	for _, pfx := range random.RealWorldPrefixes(prng, n) {
		tbl.Delete(pfx)
	}

	var allCount4 int
	var allCount6 int

	for range tbl.AllSorted4() {
		allCount4++
	}

	for range tbl.AllSorted6() {
		allCount6++
	}

	if allCount4 != tbl.Size4() {
		t.Errorf("Size4: want: %d, got: %d", allCount4, tbl.Size4())
	}

	if allCount6 != tbl.Size6() {
		t.Errorf("Size6: want: %d, got: %d", allCount6, tbl.Size6())
	}
}

// TestAll tests All with random samples
func TestTableAll_Fast(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := random.RealWorldPrefixes(prng, n)

	tbl := new(Fast[string])

	// Insert all prefixes with their values
	for _, pfx := range pfxs {
		tbl.Insert(pfx, pfx.String())
	}

	// Collect all prefixes from All
	gotPrefixes := make([]netip.Prefix, 0, n)
	gotValues := make([]string, 0, n)

	for pfx, val := range tbl.All() {
		gotPrefixes = append(gotPrefixes, pfx)
		gotValues = append(gotValues, val)
	}

	// Collect all prefixes from All4 and All6
	got4Prefixes := make([]netip.Prefix, 0, n)
	got4Values := make([]string, 0, n)

	for pfx, val := range tbl.All4() {
		got4Prefixes = append(got4Prefixes, pfx)
		got4Values = append(got4Values, val)
	}

	got6Prefixes := make([]netip.Prefix, 0, n)
	got6Values := make([]string, 0, n)
	for pfx, val := range tbl.All6() {
		got6Prefixes = append(got6Prefixes, pfx)
		got6Values = append(got6Values, val)
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
	if !slices.Equal(slices.Concat(got4Values, got6Values), gotValues) {
		t.Fatal("Values: All4 + All6 != All")
	}

	// Verify we can find each original prefix in the results
	// Create a map for O(1) lookup verification
	resultMap := make(map[netip.Prefix]string, n)
	for i, pfx := range gotPrefixes {
		resultMap[pfx] = gotValues[i]
	}

	for _, pfx := range pfxs {
		val, found := resultMap[pfx]
		if !found {
			t.Fatalf("Original prefix %v not found in All results", pfx)
		}

		// Skip value comparison for liteTable (no real payload)
		if _, isLite := any(tbl).(*liteTable[string]); !isLite {
			if val != pfx.String() {
				t.Fatalf("Original prefix %v has wrong value: expected %s, got %s",
					pfx, pfx.String(), val)
			}
		}
	}

	// Verify no duplicates in results
	seen := make(map[netip.Prefix]bool, n)
	for _, pfx := range gotPrefixes {
		if seen[pfx] {
			t.Fatalf("Duplicate prefix %v found in All results", pfx)
		}
		seen[pfx] = true
	}
}

func TestTableAllSorted_Fast(t *testing.T) {
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

			tbl := new(Fast[int])

			// Insert prefixes with index as value
			for i, prefixStr := range tc.prefixes {
				pfx := netip.MustParsePrefix(prefixStr)
				tbl.Insert(pfx, i)
			}

			// Collect sorted results
			var actualOrder []string
			for pfx := range tbl.AllSorted() {
				actualOrder = append(actualOrder, pfx.String())
			}

			// Verify the order matches expected
			if len(actualOrder) != len(tc.expected) {
				t.Fatalf("%s: Expected %d results, got %d", tc.name, len(tc.expected), len(actualOrder))
			}

			// Collect sorted 4 results
			var actual4Order []string
			for pfx := range tbl.AllSorted4() {
				actual4Order = append(actual4Order, pfx.String())
			}

			// Collect sorted 6 results
			var actual6Order []string
			for pfx := range tbl.AllSorted6() {
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

func TestTableSubnets_Fast(t *testing.T) {
	t.Parallel()

	var zeroPfx netip.Prefix

	t.Run("empty table", func(t *testing.T) {
		t.Parallel()
		tbl := new(Fast[string])
		pfx := mpp("::1/128")

		for range tbl.Subnets(pfx) {
			t.Errorf("empty table, must not range over")
		}
	})

	t.Run("invalid prefix", func(t *testing.T) {
		tbl := new(Fast[string])
		pfx := mpp("::1/128")
		val := "foo"
		tbl.Insert(pfx, val)
		for range tbl.Subnets(zeroPfx) {
			t.Errorf("invalid prefix, must not range over")
		}
	})

	t.Run("identity", func(t *testing.T) {
		tbl := new(Fast[string])
		pfx := mpp("::1/128")
		val := "foo"
		tbl.Insert(pfx, val)

		for p, v := range tbl.Subnets(pfx) {
			if p != pfx {
				t.Errorf("Subnet(%v), got: %v, want: %v", pfx, p, pfx)
			}

			// Skip value comparison for liteTable (no real payload)
			if _, isLite := any(tbl).(*liteTable[string]); !isLite {
				if v != val {
					t.Errorf("Subnet(%v), got: %v, want: %v", pfx, v, val)
				}
			}
		}
	})

	t.Run("default gateway", func(t *testing.T) {
		n := workLoadN()
		prng := rand.New(rand.NewPCG(42, 42))

		want4 := n - n/2
		want6 := n + n/2

		tbl := new(Fast[int])
		for i, pfx := range random.RealWorldPrefixes4(prng, want4) {
			tbl.Insert(pfx, i)
		}
		for i, pfx := range random.RealWorldPrefixes6(prng, want6) {
			tbl.Insert(pfx, i)
		}

		// default gateway v4 covers all v4 prefixes in table
		dg4 := mpp("0.0.0.0/0")
		got4 := 0
		for range tbl.Subnets(dg4) {
			got4++
		}

		// default gateway v6 covers all v6 prefixes in table
		dg6 := mpp("::/0")
		got6 := 0
		for range tbl.Subnets(dg6) {
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

func TestTableSubnetsCompare_Fast(t *testing.T) {
	t.Parallel()
	n := workLoadN()
	prng := rand.New(rand.NewPCG(42, 42))

	pfxs := random.RealWorldPrefixes(prng, n)

	gold := new(golden.Table[int])
	tbl := new(Fast[int])

	for i, pfx := range pfxs {
		gold.Insert(pfx, i)
		tbl.Insert(pfx, i)
	}

	for _, pfx := range random.RealWorldPrefixes(prng, n) {
		t.Run("subtest", func(t *testing.T) {
			t.Parallel()

			gotGold := gold.Subnets(pfx)
			gotTbl := []netip.Prefix{}
			for pfx := range tbl.Subnets(pfx) {
				gotTbl = append(gotTbl, pfx)
			}
			if !slices.Equal(gotGold, gotTbl) {
				t.Fatalf("Subnets(%q) = %v, want %v", pfx, gotTbl, gotGold)
			}
		})
	}
}

func TestTableSupernetsEdgeCase_Fast(t *testing.T) {
	t.Parallel()

	var zeroPfx netip.Prefix

	t.Run("empty table", func(t *testing.T) {
		t.Parallel()
		tbl := new(Fast[any])
		pfx := mpp("::1/128")

		tbl.Supernets(pfx)(func(_ netip.Prefix, _ any) bool {
			t.Errorf("empty table, must not range over")
			return false
		})
	})

	t.Run("invalid prefix", func(t *testing.T) {
		tbl := new(Fast[any])
		pfx := mpp("::1/128")
		val := "foo"
		tbl.Insert(pfx, val)

		tbl.Supernets(zeroPfx)(func(_ netip.Prefix, _ any) bool {
			t.Errorf("invalid prefix, must not range over")
			return false
		})
	})

	t.Run("identity", func(t *testing.T) {
		tbl := new(Fast[string])
		pfx := mpp("::1/128")
		val := "foo"
		tbl.Insert(pfx, val)

		for p, v := range tbl.Supernets(pfx) {
			if p != pfx {
				t.Errorf("Supernets(%v), got: %v, want: %v", pfx, p, pfx)
			}

			// Skip value comparison for liteTable (no real payload)
			if _, isLite := any(tbl).(*liteTable[string]); !isLite {
				if v != val {
					t.Errorf("Supernets(%v), got: %v, want: %v", pfx, v, val)
				}
			}
		}
	})
}

func TestTableSupernetsCompare_Fast(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))

	pfxs := random.RealWorldPrefixes(prng, n)

	gold := new(golden.Table[int])
	tbl := new(Fast[int])

	for i, pfx := range pfxs {
		gold.Insert(pfx, i)
		tbl.Insert(pfx, i)
	}

	for _, pfx := range random.RealWorldPrefixes(prng, n) {
		t.Run("subtest", func(t *testing.T) {
			t.Parallel()
			gotGold := gold.Supernets(pfx)
			gotTbl := []netip.Prefix{}

			for p := range tbl.Supernets(pfx) {
				gotTbl = append(gotTbl, p)
			}

			if !slices.Equal(gotGold, gotTbl) {
				t.Fatalf("Supernets(%q) = %v, want %v", pfx, gotTbl, gotGold)
			}
		})
	}
}

func TestTableMarshalText_Fast(t *testing.T) {
	tests := []struct {
		name         string
		expectedData map[netip.Prefix]string
	}{
		{
			name:         "empty",
			expectedData: map[netip.Prefix]string{},
		},
		{
			name: "with_data",
			expectedData: map[netip.Prefix]string{
				mpp("192.168.1.0/24"): "test1",
				mpp("10.0.0.0/8"):     "test2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tbl := new(Fast[string])

			// Insert test data
			for prefix, value := range tt.expectedData {
				tbl.Insert(prefix, value)
			}

			data, err := tbl.MarshalText()
			if err != nil {
				t.Errorf("MarshalText failed: %v", err)
			}

			if len(tt.expectedData) > 0 && len(data) == 0 {
				t.Error("Expected non-empty marshaled text")
			}

			// Check that all expected values appear in marshaled text
			text := string(data)
			for _, value := range tt.expectedData {
				if !strings.Contains(text, value) {
					t.Errorf("Marshaled text doesn't contain expected value: %s", value)
				}
			}
		})
	}
}

func TestTableMarshalJSON_Fast(t *testing.T) {
	tests := []struct {
		name         string
		expectedData map[netip.Prefix]any
	}{
		{
			name:         "empty",
			expectedData: map[netip.Prefix]any{},
		},
		{
			name: "string_values",
			expectedData: map[netip.Prefix]any{
				mpp("192.168.1.0/24"): "net1",
				mpp("10.0.0.0/8"):     "net2",
			},
		},
		{
			name: "mixed_values",
			expectedData: map[netip.Prefix]any{
				mpp("192.168.1.0/24"): "string",
				mpp("10.0.0.0/8"):     42,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tbl := new(Fast[any])

			// Insert test data
			for prefix, value := range tt.expectedData {
				tbl.Insert(prefix, value)
			}

			jsonData, err := tbl.MarshalJSON()
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

func TestTableDumpList4_Fast(t *testing.T) {
	tests := []struct {
		name         string
		expectedData map[netip.Prefix]string
		expectItems  int
	}{
		{
			name:         "empty",
			expectedData: map[netip.Prefix]string{},
			expectItems:  0,
		},
		{
			name: "single_ipv4",
			expectedData: map[netip.Prefix]string{
				mpp("192.168.1.0/24"): "lan",
			},
			expectItems: 1,
		},
		{
			name: "multiple_ipv4",
			expectedData: map[netip.Prefix]string{
				mpp("192.168.1.0/24"): "lan",
				mpp("10.0.0.0/8"):     "private",
			},
			expectItems: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tbl := new(Fast[string])

			// Insert test data
			for prefix, value := range tt.expectedData {
				tbl.Insert(prefix, value)
			}

			dumpList := tbl.DumpList4()

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

func TestTableDumpList6_Fast(t *testing.T) {
	tests := []struct {
		name         string
		expectedData map[netip.Prefix]string
		expectItems  int
	}{
		{
			name:         "empty",
			expectedData: map[netip.Prefix]string{},
			expectItems:  0,
		},
		{
			name: "single_ipv6",
			expectedData: map[netip.Prefix]string{
				mpp("2001:db8::/32"): "doc",
			},
			expectItems: 1,
		},
		{
			name: "multiple_ipv6",
			expectedData: map[netip.Prefix]string{
				mpp("2001:db8::/32"): "doc",
				mpp("fe80::/10"):     "link-local",
			},
			expectItems: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tbl := new(Fast[string])

			// Insert test data
			for prefix, value := range tt.expectedData {
				tbl.Insert(prefix, value)
			}

			dumpList := tbl.DumpList6()

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

func TestTableEqual_Fast(t *testing.T) {
	t.Parallel()

	n := workLoadN()
	prng := rand.New(rand.NewPCG(42, 42))

	at := new(Fast[int])
	for i, pfx := range random.RealWorldPrefixes(prng, n) {
		at.Insert(pfx, i)
	}

	t.Run("clone", func(t *testing.T) {
		t.Parallel()
		bt := at.Clone()
		if !at.Equal(bt) {
			t.Error("expected true, got false")
		}
	})
}
