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

func TestTableNil(t *testing.T) {
	t.Parallel()

	ip4 := mpa("127.0.0.1")
	ip6 := mpa("::1")

	pfx4 := mpp("127.0.0.0/8")
	pfx6 := mpp("::1/128")

	bart2 := new(Table[any])
	bart2.Insert(pfx4, nil)
	bart2.Insert(pfx6, nil)

	var bart1 *Table[any] = nil

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
		mustPanic(t, "ModifyPersist", func() { bart1.ModifyPersist(pfx4, nil) })
		mustPanic(t, "Contains", func() { bart1.Contains(ip4) })
		mustPanic(t, "Lookup", func() { bart1.Lookup(ip6) })
		mustPanic(t, "LookupPrefix", func() { bart1.LookupPrefix(pfx4) })
		mustPanic(t, "LookupPrefixLPM", func() { bart1.LookupPrefixLPM(pfx4) })
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

func TestTableInvalid(t *testing.T) {
	t.Parallel()

	bart1 := new(Table[any])
	bart2 := new(Table[any])

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
	noPanic(t, "Lookup", func() { bart1.Lookup(zeroIP) })
	noPanic(t, "LookupPrefix", func() { bart1.LookupPrefix(zeroPfx) })
	noPanic(t, "LookupPrefixLPM", func() { bart1.LookupPrefixLPM(zeroPfx) })
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

func TestTableContainsCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[int])
	bart := new(Table[int])

	for i, p := range pfxs {
		gold.insert(p, i) // ensures Masked + de-dupe
		bart.Insert(p, i)
	}

	for range n {
		a := randomAddr(prng)

		_, goldOK := gold.lookup(a)
		bartOK := bart.Contains(a)

		if goldOK != bartOK {
			t.Fatalf("Contains(%q) = %v, want %v", a, bartOK, goldOK)
		}
	}
}

func TestTableLookupCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[int])
	bart := new(Table[int])

	for i, pfx := range pfxs {
		gold.insert(pfx, i)
		bart.Insert(pfx, i)
	}

	for range n {
		a := randomAddr(prng)

		goldVal, goldOK := gold.lookup(a)
		bartVal, bartOK := bart.Lookup(a)

		if !getsEqual(goldVal, goldOK, bartVal, bartOK) {
			t.Fatalf("Lookup(%q) = (%v, %v), want (%v, %v)", a, bartVal, bartOK, goldVal, goldOK)
		}
	}
}

func TestTableLookupPrefixUnmasked(t *testing.T) {
	// test that the pfx must not be masked on input for LookupPrefix
	t.Parallel()

	bart := new(Table[any])
	bart.Insert(mpp("10.20.30.0/24"), nil)

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
		_, got := bart.LookupPrefix(tc.probe)
		if got != tc.wantOk {
			t.Errorf("LookupPrefix non canonical prefix (%s), got: %v, want: %v", tc.probe, got, tc.wantOk)
		}

		lpm, _, got := bart.LookupPrefixLPM(tc.probe)
		if got != tc.wantOk {
			t.Errorf("LookupPrefixLPM non canonical prefix (%s), got: %v, want: %v", tc.probe, got, tc.wantOk)
		}
		if lpm != tc.wantLPM {
			t.Errorf("LookupPrefixLPM non canonical prefix (%s), got: %v, want: %v", tc.probe, lpm, tc.wantLPM)
		}
	}
}

func TestTableLookupPrefixCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[int])
	bart := new(Table[int])
	for i, pfx := range pfxs {
		gold.insert(pfx, i)
		bart.Insert(pfx, i)
	}

	for range n {
		pfx := randomPrefix(prng)

		goldVal, goldOK := gold.lookupPfx(pfx)
		bartVal, bartOK := bart.LookupPrefix(pfx)

		if !getsEqual(goldVal, goldOK, bartVal, bartOK) {
			t.Fatalf("LookupPrefix(%q) = (%v, %v), want (%v, %v)", pfx, bartVal, bartOK, goldVal, goldOK)
		}
	}
}

func TestTableLookupPrefixLPMCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[int])

	bart := new(Table[int])
	for i, pfx := range pfxs {
		gold.insert(pfx, i)
		bart.Insert(pfx, i)
	}

	for range n {
		pfx := randomPrefix(prng)

		goldLPM, goldVal, goldOK := gold.lookupPfxLPM(pfx)
		bartLPM, bartVal, bartOK := bart.LookupPrefixLPM(pfx)

		if !getsEqual(goldVal, goldOK, bartVal, bartOK) {
			t.Fatalf("LookupPrefixLPM(%q) = (%v, %v), want (%v, %v)", pfx, bartVal, bartOK, goldVal, goldOK)
		}

		if !getsEqual(goldLPM, goldOK, bartLPM, bartOK) {
			t.Fatalf("LookupPrefixLPM(%q) = (%v, %v), want (%v, %v)", pfx, bartLPM, bartOK, goldLPM, goldOK)
		}
	}
}

func TestTableInsertShuffled(t *testing.T) {
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

		bart1 := new(Table[string])
		bart2 := new(Table[string])

		for _, pfx := range pfxs {
			bart1.Insert(pfx, pfx.String())
			bart1.Insert(pfx, pfx.String()) // idempotent
		}
		for _, pfx := range pfxs2 {
			bart2.Insert(pfx, pfx.String()) // idempotent
		}

		if !bart1.Equal(bart2) {
			t.Fatal("expected Equal")
		}
	}
}

func TestTableInsertPersistShuffled(t *testing.T) {
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

		bart1 := new(Table[string])
		bart2 := new(Table[string])

		// bart1 is mutable
		for _, pfx := range pfxs {
			bart1.Insert(pfx, pfx.String())
		}

		// bart2 is persistent
		for _, pfx := range pfxs2 {
			bart2 = bart2.InsertPersist(pfx, pfx.String())
		}

		if bart1.dumpString() != bart2.dumpString() {
			t.Fatal("mutable and immutable table have different dumpString representation")
		}

		if !bart1.Equal(bart2) {
			t.Fatal("expected Equal")
		}
	}
}

func TestTableDeleteCompare(t *testing.T) {
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

		// pfxs toDelete should be non-overlapping sets
		pfxs := slices.Concat(all4[:deleteCut], all6[:deleteCut])
		toDelete := slices.Concat(all4[deleteCut:], all6[deleteCut:])

		gold := new(goldTable[string])
		bart := new(Table[string])

		for _, pfx := range pfxs {
			gold.insert(pfx, pfx.String())
			bart.Insert(pfx, pfx.String())
		}

		for _, pfx := range toDelete {
			gold.delete(pfx)
			bart.Delete(pfx)
		}

		gold.sort()

		bartGolden := bart.dumpAsGoldTable()
		bartGolden.sort()

		if !slices.Equal(*gold, bartGolden) {
			t.Fatal("expected Equal")
		}
	}
}

func TestTableDeleteShuffled(t *testing.T) {
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

		bart := new(Table[string])

		// insert
		for _, pfx := range pfxs {
			bart.Insert(pfx, pfx.String())
		}
		for _, pfx := range toDelete {
			bart.Insert(pfx, pfx.String())
		}

		// delete
		for _, pfx := range toDelete {
			bart.Delete(pfx)
		}

		pfxs2 := slices.Clone(pfxs)
		toDelete2 := slices.Clone(toDelete)
		prng.Shuffle(len(toDelete2), func(i, j int) { toDelete2[i], toDelete2[j] = toDelete2[j], toDelete2[i] })

		bart2 := new(Table[string])

		// insert
		for _, pfx := range pfxs2 {
			bart2.Insert(pfx, pfx.String())
		}
		for _, pfx := range toDelete2 {
			bart2.Insert(pfx, pfx.String())
		}

		// delete
		for _, pfx := range toDelete2 {
			bart2.Delete(pfx)
		}

		if !bart.Equal(bart2) {
			t.Fatal("expect equal")
		}
	}
}

func TestTableDeleteIsReverseOfInsert(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	// Insert n prefixes, then delete those same prefixes in reverse
	// order. Each deletion should exactly undo the internal structure
	// changes that each insert did.

	bart := new(Table[string])
	want := bart.dumpString()

	prefixes := randomRealWorldPrefixes(prng, n)

	defer func() {
		if t.Failed() {
			t.Logf("the prefixes that fail the test: %v\n", prefixes)
		}
	}()

	for _, p := range prefixes {
		bart.Insert(p, p.String())
	}

	for i := len(prefixes) - 1; i >= 0; i-- {
		bart.Delete(prefixes[i])
	}
	if got := bart.dumpString(); got != want {
		t.Fatalf("after delete, mismatch:\n\n got: %s\n\nwant: %s", got, want)
	}
}

func TestTableDeleteButOne(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))
	// Insert n prefixes, then delete all but one
	n := workLoadN()

	for range 10 {

		bart := new(Table[any])
		prefixes := randomRealWorldPrefixes(prng, n)

		for _, p := range prefixes {
			bart.Insert(p, nil)
		}

		// shuffle the prefixes
		prng.Shuffle(n, func(i, j int) {
			prefixes[i], prefixes[j] = prefixes[j], prefixes[i]
		})

		// skip the first
		for i := 1; i < len(prefixes); i++ {
			bart.Delete(prefixes[i])
		}

		stats4 := bart.root4.StatsRec()
		stats6 := bart.root6.StatsRec()

		if nodes := stats4.Nodes + stats6.Nodes; nodes != 1 {
			t.Fatalf("delete but one, want nodes: 1, got: %d\n%s", nodes, bart.dumpString())
		}

		sum := stats4.Pfxs + stats4.Leaves + stats4.Fringes +
			stats6.Pfxs + stats6.Leaves + stats6.Fringes

		if sum != 1 {
			t.Fatalf("delete but one, only one item must be left, but: %d\n%s", sum, bart.dumpString())
		}
	}
}

func TestTableGet(t *testing.T) {
	t.Parallel()

	t.Run("empty table", func(t *testing.T) {
		t.Parallel()
		prng := rand.New(rand.NewPCG(42, 42))

		bart := new(Table[int])
		pfx := randomPrefix(prng)
		_, ok := bart.Get(pfx)

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

	bart := new(Table[int])
	for _, tt := range tests {
		bart.Insert(tt.pfx, tt.val)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := bart.Get(tt.pfx)

			if !ok {
				t.Errorf("%s: ok=%v, expected: %v", tt.name, ok, true)
			}

			if got != tt.val {
				t.Errorf("%s: val=%v, expected: %v", tt.name, got, tt.val)
			}
		})
	}
}

func TestTableGetCompare(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[string])
	bart := new(Table[string])
	for _, pfx := range pfxs {
		gold.insert(pfx, pfx.String())
		bart.Insert(pfx, pfx.String())
	}

	for _, pfx := range pfxs {
		goldVal, goldOK := gold.get(pfx)
		bartVal, bartOK := bart.Get(pfx)

		if !getsEqual(goldVal, goldOK, bartVal, bartOK) {
			t.Fatalf("Get(%q) = (%v, %v), want (%v, %v)", pfx, bartVal, bartOK, goldVal, goldOK)
		}
	}
}

func TestTableModifySemantics(t *testing.T) {
	t.Parallel()

	type args struct {
		pfx netip.Prefix
		cb  func(val int, found bool) (_ int, del bool)
	}

	type want struct {
		val     int
		deleted bool
	}

	tests := []struct {
		name      string
		prepare   map[netip.Prefix]int // entries to pre-populate the table
		args      args
		want      want
		finalData map[netip.Prefix]int // expected table contents after the operation
	}{
		{
			name:    "Delete existing entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42, mpp("2001:db8::/32"): 4242},
			args: args{
				pfx: mpp("10.0.0.0/8"),
				cb:  func(val int, found bool) (_ int, del bool) { return 0, true },
			},
			want:      want{val: 42, deleted: true},
			finalData: map[netip.Prefix]int{mpp("2001:db8::/32"): 4242},
		},

		{
			name:    "Insert new entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42},
			args: args{
				pfx: mpp("2001:db8::/32"),
				cb:  func(val int, found bool) (_ int, del bool) { return 4242, false },
			},
			want:      want{val: 4242, deleted: false},
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42, mpp("2001:db8::/32"): 4242},
		},

		{
			// For update, the callback gets oldVal, returns newVal, but Modify returns oldVal
			name:    "Update existing entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42, mpp("2001:db8::/32"): 4242},
			args: args{
				pfx: mpp("10.0.0.0/8"),
				cb:  func(val int, found bool) (_ int, del bool) { return -1, false },
			},
			want:      want{val: 42, deleted: false},
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): -1, mpp("2001:db8::/32"): 4242},
		},

		{
			name:    "No-op on missing entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42},
			args: args{
				pfx: mpp("2001:db8::/32"),
				cb:  func(val int, found bool) (_ int, del bool) { return 0, true },
			},
			want:      want{val: 0, deleted: false},
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42},
		},
	}

	for _, tt := range tests {
		bart := new(Table[int])

		// Insert initial entries using Modify
		for pfx, v := range tt.prepare {
			bart.Modify(pfx, func(_ int, _ bool) (_ int, del bool) { return v, false })
		}

		bart.Modify(tt.args.pfx, tt.args.cb)

		// Check the final state of the table using Get, compares expected and actual table
		got := make(map[netip.Prefix]int, len(tt.finalData))
		for pfx, val := range bart.All() {
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

func TestTableModifyPersistSemantics(t *testing.T) {
	t.Parallel()

	type args struct {
		pfx netip.Prefix
		cb  func(val int, found bool) (_ int, del bool)
	}

	type want struct {
		val     int
		deleted bool
	}

	tests := []struct {
		name      string
		prepare   map[netip.Prefix]int // entries to pre-populate the table
		args      args
		want      want
		finalData map[netip.Prefix]int // expected table contents after the operation
	}{
		{
			name:    "Delete existing entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42, mpp("2001:db8::/32"): 4242},
			args: args{
				pfx: mpp("10.0.0.0/8"),
				cb:  func(val int, found bool) (_ int, del bool) { return 0, true },
			},
			want:      want{val: 42, deleted: true},
			finalData: map[netip.Prefix]int{mpp("2001:db8::/32"): 4242},
		},

		{
			name:    "Insert new entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42},
			args: args{
				pfx: mpp("2001:db8::/32"),
				cb:  func(val int, found bool) (_ int, del bool) { return 4242, false },
			},
			want:      want{val: 4242, deleted: false},
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42, mpp("2001:db8::/32"): 4242},
		},

		{
			// For update, the callback gets oldVal, returns newVal, but Modify returns oldVal
			name:    "Update existing entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42, mpp("2001:db8::/32"): 4242},
			args: args{
				pfx: mpp("10.0.0.0/8"),
				cb:  func(val int, found bool) (_ int, del bool) { return -1, false },
			},
			want:      want{val: 42, deleted: false},
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): -1, mpp("2001:db8::/32"): 4242},
		},

		{
			name:    "No-op on missing entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42},
			args: args{
				pfx: mpp("2001:db8::/32"),
				cb:  func(val int, found bool) (_ int, del bool) { return 0, true },
			},
			want:      want{val: 0, deleted: false},
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42},
		},
	}

	for _, tt := range tests {
		bart := new(Table[int])

		// Insert initial entries using Modify
		for pfx, v := range tt.prepare {
			bart.Modify(pfx, func(_ int, _ bool) (_ int, del bool) { return v, false })
		}

		prt := bart.ModifyPersist(tt.args.pfx, tt.args.cb)

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

func TestTableModifyCompare(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[int])
	bart := new(Table[int])

	// Update as insert
	for i, pfx := range pfxs {
		gold.insert(pfx, i)
		bart.Modify(pfx, func(int, bool) (int, bool) { return i, false })
	}

	gold.sort()
	bartGolden := bart.dumpAsGoldTable()
	bartGolden.sort()

	if !slices.Equal(*gold, bartGolden) {
		t.Fatal("expected Equal")
	}

	cb1 := func(val int, _ bool) int { return val + 1 }
	cb2 := func(val int, _ bool) (int, bool) { return val + 1, false }

	// Update as update
	for _, pfx := range pfxs[:len(pfxs)/2] {
		gold.update(pfx, cb1)
		bart.Modify(pfx, cb2)
	}

	gold.sort()
	bartGolden = bart.dumpAsGoldTable()
	bartGolden.sort()

	if !slices.Equal(*gold, bartGolden) {
		t.Fatal("expected Equal")
	}
}

func TestTableModifyPersistCompare(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	mut := new(Table[int])
	imu := new(Table[int])

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

func TestTableModifyShuffled(t *testing.T) {
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

		bart1 := new(Table[string])

		// insert
		for _, pfx := range pfxs {
			bart1.Insert(pfx, pfx.String())
		}
		for _, pfx := range toDelete {
			bart1.Insert(pfx, pfx.String())
		}

		// this callback deletes unconditionally
		cb := func(string, bool) (string, bool) { return "", true }

		// delete
		for _, pfx := range toDelete {
			bart1.Modify(pfx, cb)
		}

		pfxs2 := slices.Clone(pfxs)
		toDelete2 := slices.Clone(toDelete)
		prng.Shuffle(len(toDelete2), func(i, j int) { toDelete2[i], toDelete2[j] = toDelete2[j], toDelete2[i] })

		bart2 := new(Table[string])

		// insert
		for _, pfx := range pfxs2 {
			bart2.Insert(pfx, pfx.String())
		}
		for _, pfx := range toDelete2 {
			bart2.Insert(pfx, pfx.String())
		}

		// delete
		for _, pfx := range toDelete2 {
			bart2.Modify(pfx, cb)
		}

		if !bart1.Equal(bart2) {
			t.Fatal("expected equal")
		}
	}
}

func TestTableModifyPersistShuffled(t *testing.T) {
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

		bart1 := new(Table[string])

		// insert
		for _, pfx := range pfxs {
			bart1.Insert(pfx, pfx.String())
		}
		for _, pfx := range toDelete {
			bart1.Insert(pfx, pfx.String())
		}

		// this callback deletes unconditionally
		cb := func(string, bool) (string, bool) { return "", true }

		// delete
		for _, pfx := range toDelete {
			bart1 = bart1.ModifyPersist(pfx, cb)
		}

		pfxs2 := slices.Clone(pfxs)
		toDelete2 := slices.Clone(toDelete)
		prng.Shuffle(len(toDelete2), func(i, j int) { toDelete2[i], toDelete2[j] = toDelete2[j], toDelete2[i] })

		bart2 := new(Table[string])

		// insert
		for _, pfx := range pfxs2 {
			bart2.Insert(pfx, pfx.String())
		}
		for _, pfx := range toDelete2 {
			bart2.Insert(pfx, pfx.String())
		}

		// delete
		for _, pfx := range toDelete2 {
			bart2 = bart2.ModifyPersist(pfx, cb)
		}

		if !bart1.Equal(bart2) {
			t.Fatal("expected equal")
		}
	}
}

// TestUnionMemoryAliasing tests that the Union method does not alias memory
// between the two tables.
func TestTableUnionMemoryAliasing(t *testing.T) {
	t.Parallel()

	newTable := func(pfx ...string) *Table[struct{}] {
		rt := new(Table[struct{}])
		for _, s := range pfx {
			rt.Insert(mpp(s), struct{}{})
		}
		return rt
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
func TestTableUnionPersistMemoryAliasing(t *testing.T) {
	t.Parallel()

	newTable := func(pfx ...string) *Table[struct{}] {
		rt := new(Table[struct{}])
		for _, s := range pfx {
			rt.Insert(mpp(s), struct{}{})
		}
		return rt
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

func TestTableUnionCompare(t *testing.T) {
	t.Parallel()

	n := workLoadN()
	prng := rand.New(rand.NewPCG(42, 42))

	for range 3 {
		pfxs := randomRealWorldPrefixes(prng, n)

		gold := new(goldTable[string])
		bart := new(Table[string])

		for _, pfx := range pfxs {
			gold.insert(pfx, pfx.String())
			bart.Insert(pfx, pfx.String())
		}

		pfxs2 := randomRealWorldPrefixes(prng, n)

		gold2 := new(goldTable[string])
		bart2 := new(Table[string])

		for _, pfx := range pfxs2 {
			gold2.insert(pfx, pfx.String())
			bart2.Insert(pfx, pfx.String())
		}

		gold.union(gold2)
		bart.Union(bart2)

		// dump as slow table for comparison
		bartAsGoldenTbl := bart.dumpAsGoldTable()

		// sort for comparison
		gold.sort()
		bartAsGoldenTbl.sort()

		if !slices.Equal(*gold, bartAsGoldenTbl) {
			t.Fatal("expected equal")
		}
	}
}

func TestTableUnionPersistCompare(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))

	n := workLoadN()

	for range 3 {
		pfxs := randomRealWorldPrefixes(prng, n)

		gold := new(goldTable[int])
		bart := new(Table[int])

		for i, pfx := range pfxs {
			gold.insert(pfx, i)
			bart.Insert(pfx, i)
		}

		pfxs2 := randomRealWorldPrefixes(prng, n)

		gold2 := new(goldTable[int])
		bart2 := new(Table[int])

		for i, pfx := range pfxs2 {
			gold2.insert(pfx, i)
			bart2.Insert(pfx, i)
		}

		gold.union(gold2)
		bartP := bart.UnionPersist(bart2)

		// dump as slow table for comparison
		bartAsGoldenTbl := bartP.dumpAsGoldTable()

		// sort for comparison
		gold.sort()
		bartAsGoldenTbl.sort()

		if !slices.Equal(*gold, bartAsGoldenTbl) {
			t.Fatal("expected equal")
		}
	}
}

func TestTableClone(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, 100_000)

	var bart *Table[int]
	if bart.Clone() != nil {
		t.Fatal("expected nil")
	}

	bart = new(Table[int])
	for i, pfx := range pfxs {
		bart.Insert(pfx, i)
	}
	clone := bart.Clone()

	if !bart.Equal(clone) {
		t.Fatal("expected equal")
	}
}

func TestTableCloneShallow(t *testing.T) {
	t.Parallel()

	bart := new(Table[*int])
	clone := bart.Clone()
	if bart.dumpString() != clone.dumpString() {
		t.Errorf("empty Clone: got:\n%swant:\n%s", clone.dumpString(), bart.dumpString())
	}

	val := 1
	pfx := mpp("10.0.0.1/32")
	bart.Insert(pfx, &val)

	clone = bart.Clone()
	want, _ := bart.Get(pfx)
	got, _ := clone.Get(pfx)

	if *got != *want || got != want {
		t.Errorf("shallow copy, values and pointers must be equal:\nvalues(%d, %d)\n(ptr(%v, %v)", *got, *want, got, want)
	}

	// update value, shallow copy of values, clone must be equal
	val = 2
	want, _ = bart.Get(pfx)
	got, _ = clone.Get(pfx)

	if *got != *want {
		t.Errorf("memory aliasing after shallow copy, values must be equal:\nvalues(%d, %d)", *got, *want)
	}
}

func TestTableCloneDeep(t *testing.T) {
	t.Parallel()

	bart := new(Table[*MyInt])
	clone := bart.Clone()
	if bart.dumpString() != clone.dumpString() {
		t.Errorf("empty Clone: got:\n%swant:\n%s", clone.dumpString(), bart.dumpString())
	}

	val := MyInt(1)
	pfx := mpp("10.0.0.1/32")
	bart.Insert(pfx, &val)

	clone = bart.Clone()
	want, _ := bart.Get(pfx)
	got, _ := clone.Get(pfx)

	if *got != *want || got == want {
		t.Errorf("value with Cloner interface, pointers must be different:\nvalues(%d, %d)\n(ptr(%v, %v)", *got, *want, got, want)
	}

	// update value, deep copy of values, cloned value must now be different
	val = 2
	want, _ = bart.Get(pfx)
	got, _ = clone.Get(pfx)

	if *got == *want {
		t.Errorf("memory aliasing after deep copy, values must be different:\nvalues(%d, %d)", *got, *want)
	}
}

func TestTableUnionShallow(t *testing.T) {
	t.Parallel()

	bart1 := new(Table[*int])
	bart2 := new(Table[*int])

	val := 1
	pfx := mpp("10.0.0.1/32")
	bart2.Insert(pfx, &val)

	bart1.Union(bart2)
	got, _ := bart1.Get(pfx)
	want, _ := bart2.Get(pfx)

	if *got != *want || got != want {
		t.Errorf("shallow copy, values and pointers must be equal:\nvalues(%d, %d)\n(ptr(%v, %v)", *got, *want, got, want)
	}

	// update value, shallow copy of values, union must be equal
	val = 2
	got, _ = bart1.Get(pfx)
	want, _ = bart2.Get(pfx)

	if *got != *want {
		t.Errorf("memory aliasing after shallow copy, values must be equal:\nvalues(%d, %d)", *got, *want)
	}
}

func TestTableUnionDeep(t *testing.T) {
	t.Parallel()

	bart1 := new(Table[*MyInt])
	bart2 := new(Table[*MyInt])

	val := MyInt(1)
	pfx := mpp("10.0.0.1/32")
	bart2.Insert(pfx, &val)

	bart1.Union(bart2)
	got, _ := bart1.Get(pfx)
	want, _ := bart2.Get(pfx)

	if *got != *want || got == want {
		t.Errorf("value with Cloner interface, pointers must be different:\nvalues(%d, %d)\n(ptr(%v, %v)", *got, *want, got, want)
	}

	// update value, shallow copy of values, union must be equal
	val = 2
	got, _ = bart1.Get(pfx)
	want, _ = bart2.Get(pfx)

	if *got == *want {
		t.Errorf("memory aliasing after deep copy, values must be different:\nvalues(%d, %d)", *got, *want)
	}
}

// test some edge cases
func TestTableOverlapsPrefixEdgeCases(t *testing.T) {
	t.Parallel()

	bart := new(Table[int])

	// empty table
	checkOverlapsPrefix(t, bart, []tableOverlapsTest{
		{"0.0.0.0/0", false},
		{"::/0", false},
	})

	// default route
	bart.Insert(mpp("10.0.0.0/9"), 0)
	bart.Insert(mpp("2001:db8::/32"), 0)
	checkOverlapsPrefix(t, bart, []tableOverlapsTest{
		{"0.0.0.0/0", true},
		{"::/0", true},
	})

	// default route
	bart = new(Table[int])
	bart.Insert(mpp("0.0.0.0/0"), 0)
	bart.Insert(mpp("::/0"), 0)
	checkOverlapsPrefix(t, bart, []tableOverlapsTest{
		{"10.0.0.0/9", true},
		{"2001:db8::/32", true},
	})

	// single IP
	bart = new(Table[int])
	bart.Insert(mpp("10.0.0.0/7"), 0)
	bart.Insert(mpp("2001::/16"), 0)
	checkOverlapsPrefix(t, bart, []tableOverlapsTest{
		{"10.1.2.3/32", true},
		{"2001:db8:affe::cafe/128", true},
	})

	// single IP
	bart = new(Table[int])
	bart.Insert(mpp("10.1.2.3/32"), 0)
	bart.Insert(mpp("2001:db8:affe::cafe/128"), 0)
	checkOverlapsPrefix(t, bart, []tableOverlapsTest{
		{"10.0.0.0/7", true},
		{"2001::/16", true},
	})

	// same IPv
	bart = new(Table[int])
	bart.Insert(mpp("10.1.2.3/32"), 0)
	bart.Insert(mpp("2001:db8:affe::cafe/128"), 0)
	checkOverlapsPrefix(t, bart, []tableOverlapsTest{
		{"10.1.2.3/32", true},
		{"2001:db8:affe::cafe/128", true},
	})
}

func TestTableSize(t *testing.T) {
	t.Parallel()

	n := workLoadN()
	prng := rand.New(rand.NewPCG(42, 42))

	bart := new(Table[any])
	if bart.Size() != 0 {
		t.Errorf("empty Table: want: 0, got: %d", bart.Size())
	}

	if bart.Size4() != 0 {
		t.Errorf("empty Table: want: 0, got: %d", bart.Size4())
	}

	if bart.Size6() != 0 {
		t.Errorf("empty Table: want: 0, got: %d", bart.Size6())
	}

	pfxs1 := randomRealWorldPrefixes(prng, n)
	pfxs2 := randomRealWorldPrefixes(prng, n)

	for _, pfx := range pfxs1 {
		bart.Insert(pfx, nil)
	}

	for _, pfx := range pfxs2 {
		bart.Modify(pfx, func(any, bool) (any, bool) { return nil, false })
	}

	pfxs1 = append(pfxs1, pfxs2...)

	for _, pfx := range pfxs1[:n] {
		bart.Modify(pfx, func(any, bool) (any, bool) { return nil, false })
	}

	for _, pfx := range randomRealWorldPrefixes(prng, n) {
		bart.Delete(pfx)
	}

	var allInc4 int
	var allInc6 int

	for range bart.AllSorted4() {
		allInc4++
	}

	for range bart.AllSorted6() {
		allInc6++
	}

	if allInc4 != bart.Size4() {
		t.Errorf("Size4: want: %d, got: %d", allInc4, bart.Size4())
	}

	if allInc6 != bart.Size6() {
		t.Errorf("Size6: want: %d, got: %d", allInc6, bart.Size6())
	}
}

// TestAll tests All with random samples
func TestTableAll(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	bart := new(Table[string])

	// Insert all prefixes with their values
	for _, pfx := range pfxs {
		bart.Insert(pfx, pfx.String())
	}

	// Collect all prefixes from All
	gotPrefixes := make([]netip.Prefix, 0, n)
	gotValues := make([]string, 0, n)

	for pfx, val := range bart.All() {
		gotPrefixes = append(gotPrefixes, pfx)
		gotValues = append(gotValues, val)
	}

	// Collect all prefixes from All4 and All6
	got4Prefixes := make([]netip.Prefix, 0, n)
	got4Values := make([]string, 0, n)

	for pfx, val := range bart.All4() {
		got4Prefixes = append(got4Prefixes, pfx)
		got4Values = append(got4Values, val)
	}

	got6Prefixes := make([]netip.Prefix, 0, n)
	got6Values := make([]string, 0, n)
	for pfx, val := range bart.All6() {
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

		if val != pfx.String() {
			t.Fatalf("Original prefix %v has wrong value: expected %s, got %s",
				pfx, pfx.String(), val)
		}
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

func TestTableAllSorted(t *testing.T) {
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

			bart := new(Table[int])

			// Insert prefixes with index as value
			for i, prefixStr := range tc.prefixes {
				pfx := netip.MustParsePrefix(prefixStr)
				bart.Insert(pfx, i)
			}

			// Collect sorted results
			var actualOrder []string
			for pfx := range bart.AllSorted() {
				actualOrder = append(actualOrder, pfx.String())
			}

			// Verify the order matches expected
			if len(actualOrder) != len(tc.expected) {
				t.Fatalf("%s: Expected %d results, got %d", tc.name, len(tc.expected), len(actualOrder))
			}

			// Collect sorted 4 results
			var actual4Order []string
			for pfx := range bart.AllSorted4() {
				actual4Order = append(actual4Order, pfx.String())
			}

			// Collect sorted 6 results
			var actual6Order []string
			for pfx := range bart.AllSorted6() {
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

func TestTableSubnets(t *testing.T) {
	t.Parallel()

	var zeroPfx netip.Prefix

	t.Run("empty table", func(t *testing.T) {
		t.Parallel()
		bart := new(Table[string])
		pfx := mpp("::1/128")

		for range bart.Subnets(pfx) {
			t.Errorf("empty table, must not range over")
		}
	})

	t.Run("invalid prefix", func(t *testing.T) {
		bart := new(Table[string])
		pfx := mpp("::1/128")
		val := "foo"
		bart.Insert(pfx, val)
		for range bart.Subnets(zeroPfx) {
			t.Errorf("invalid prefix, must not range over")
		}
	})

	t.Run("identity", func(t *testing.T) {
		bart := new(Table[string])
		pfx := mpp("::1/128")
		val := "foo"
		bart.Insert(pfx, val)

		for p, v := range bart.Subnets(pfx) {
			if p != pfx {
				t.Errorf("Subnet(%v), got: %v, want: %v", pfx, p, pfx)
			}

			if v != val {
				t.Errorf("Subnet(%v), got: %v, want: %v", pfx, v, val)
			}
		}
	})

	t.Run("default gateway", func(t *testing.T) {
		n := workLoadN()
		prng := rand.New(rand.NewPCG(42, 42))

		want4 := n - n/2
		want6 := n + n/2

		bart := new(Table[int])
		for i, pfx := range randomRealWorldPrefixes4(prng, want4) {
			bart.Insert(pfx, i)
		}
		for i, pfx := range randomRealWorldPrefixes6(prng, want6) {
			bart.Insert(pfx, i)
		}

		// default gateway v4 covers all v4 prefixes in table
		dg4 := mpp("0.0.0.0/0")
		got4 := 0
		for range bart.Subnets(dg4) {
			got4++
		}

		// default gateway v6 covers all v6 prefixes in table
		dg6 := mpp("::/0")
		got6 := 0
		for range bart.Subnets(dg6) {
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

func TestTableSubnetsCompare(t *testing.T) {
	t.Parallel()
	n := workLoadN()
	prng := rand.New(rand.NewPCG(42, 42))

	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[int])
	bart := new(Table[int])

	for i, pfx := range pfxs {
		gold.insert(pfx, i)
		bart.Insert(pfx, i)
	}

	for _, pfx := range randomRealWorldPrefixes(prng, n) {
		t.Run("subtest", func(t *testing.T) {
			t.Parallel()

			gotGold := gold.subnets(pfx)
			gotBart := []netip.Prefix{}
			for pfx := range bart.Subnets(pfx) {
				gotBart = append(gotBart, pfx)
			}
			if !slices.Equal(gotGold, gotBart) {
				t.Fatalf("Subnets(%q) = %v, want %v", pfx, gotBart, gotGold)
			}
		})
	}
}

func TestTableSupernetsEdgeCase(t *testing.T) {
	t.Parallel()

	var zeroPfx netip.Prefix

	t.Run("empty table", func(t *testing.T) {
		t.Parallel()
		bart := new(Table[any])
		pfx := mpp("::1/128")

		bart.Supernets(pfx)(func(_ netip.Prefix, _ any) bool {
			t.Errorf("empty table, must not range over")
			return false
		})
	})

	t.Run("invalid prefix", func(t *testing.T) {
		bart := new(Table[any])
		pfx := mpp("::1/128")
		val := "foo"
		bart.Insert(pfx, val)

		bart.Supernets(zeroPfx)(func(_ netip.Prefix, _ any) bool {
			t.Errorf("invalid prefix, must not range over")
			return false
		})
	})

	t.Run("identity", func(t *testing.T) {
		bart := new(Table[string])
		pfx := mpp("::1/128")
		val := "foo"
		bart.Insert(pfx, val)

		for p, v := range bart.Supernets(pfx) {
			if p != pfx {
				t.Errorf("Supernets(%v), got: %v, want: %v", pfx, p, pfx)
			}

			if v != val {
				t.Errorf("Supernets(%v), got: %v, want: %v", pfx, v, val)
			}
		}
	})
}

func TestTableSupernetsCompare(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))

	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[int])
	bart := new(Table[int])

	for i, pfx := range pfxs {
		gold.insert(pfx, i)
		bart.Insert(pfx, i)
	}

	for _, pfx := range randomRealWorldPrefixes(prng, n) {
		t.Run("subtest", func(t *testing.T) {
			t.Parallel()
			gotGold := gold.supernets(pfx)
			gotBart := []netip.Prefix{}

			for p := range bart.Supernets(pfx) {
				gotBart = append(gotBart, p)
			}

			if !slices.Equal(gotGold, gotBart) {
				t.Fatalf("Supernets(%q) = %v, want %v", pfx, gotBart, gotGold)
			}
		})
	}
}

func TestTableMarshalText(t *testing.T) {
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
				netip.MustParsePrefix("192.168.1.0/24"): "test1",
				netip.MustParsePrefix("10.0.0.0/8"):     "test2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bart := new(Table[string])

			// Insert test data
			for prefix, value := range tt.expectedData {
				bart.Insert(prefix, value)
			}

			data, err := bart.MarshalText()
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

func TestTableMarshalJSON(t *testing.T) {
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
				netip.MustParsePrefix("192.168.1.0/24"): "net1",
				netip.MustParsePrefix("10.0.0.0/8"):     "net2",
			},
		},
		{
			name: "mixed_values",
			expectedData: map[netip.Prefix]any{
				netip.MustParsePrefix("192.168.1.0/24"): "string",
				netip.MustParsePrefix("10.0.0.0/8"):     42,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bart := new(Table[any])

			// Insert test data
			for prefix, value := range tt.expectedData {
				bart.Insert(prefix, value)
			}

			jsonData, err := json.Marshal(bart)
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

func TestTableDumpList4(t *testing.T) {
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
				netip.MustParsePrefix("192.168.1.0/24"): "lan",
			},
			expectItems: 1,
		},
		{
			name: "multiple_ipv4",
			expectedData: map[netip.Prefix]string{
				netip.MustParsePrefix("192.168.1.0/24"): "lan",
				netip.MustParsePrefix("10.0.0.0/8"):     "private",
			},
			expectItems: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bart := new(Table[string])

			// Insert test data
			for prefix, value := range tt.expectedData {
				bart.Insert(prefix, value)
			}

			dumpList := bart.DumpList4()

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

func TestTableDumpList6(t *testing.T) {
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
				netip.MustParsePrefix("2001:db8::/32"): "doc",
			},
			expectItems: 1,
		},
		{
			name: "multiple_ipv6",
			expectedData: map[netip.Prefix]string{
				netip.MustParsePrefix("2001:db8::/32"): "doc",
				netip.MustParsePrefix("fe80::/10"):     "link-local",
			},
			expectItems: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bart := new(Table[string])

			// Insert test data
			for prefix, value := range tt.expectedData {
				bart.Insert(prefix, value)
			}

			dumpList := bart.DumpList6()

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

func TestTableEqual(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		buildA    func() *Table[stringVal]
		buildB    func() *Table[stringVal]
		wantEqual bool
	}{
		{
			name:      "empty tables",
			buildA:    func() *Table[stringVal] { return new(Table[stringVal]) },
			buildB:    func() *Table[stringVal] { return new(Table[stringVal]) },
			wantEqual: true,
		},
		{
			name: "same single entry",
			buildA: func() *Table[stringVal] {
				tbl := new(Table[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "foo")
				return tbl
			},
			buildB: func() *Table[stringVal] {
				tbl := new(Table[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "foo")
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "different values for same prefix",
			buildA: func() *Table[stringVal] {
				tbl := new(Table[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "foo")
				return tbl
			},
			buildB: func() *Table[stringVal] {
				tbl := new(Table[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "bar")
				return tbl
			},
			wantEqual: false,
		},
		{
			name: "different entries",
			buildA: func() *Table[stringVal] {
				tbl := new(Table[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "foo")
				return tbl
			},
			buildB: func() *Table[stringVal] {
				tbl := new(Table[stringVal])
				tbl.Insert(mpp("2001:db8::/32"), "foo")
				return tbl
			},
			wantEqual: false,
		},
		{
			name: "same entries, different insert order",
			buildA: func() *Table[stringVal] {
				tbl := new(Table[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "foo")
				tbl.Insert(mpp("198.51.100.0/24"), "bar")
				return tbl
			},
			buildB: func() *Table[stringVal] {
				tbl := new(Table[stringVal])
				tbl.Insert(mpp("198.51.100.0/24"), "bar")
				tbl.Insert(mpp("192.0.2.0/24"), "foo")
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

func TestTableFullEqual(t *testing.T) {
	t.Parallel()
	at := new(Table[int])
	for i, r := range routes {
		at.Insert(r.CIDR, i)
	}

	t.Run("clone", func(t *testing.T) {
		t.Parallel()
		bt := at.Clone()
		if !at.Equal(bt) {
			t.Error("expected true, got false")
		}
	})

	t.Run("modify", func(t *testing.T) {
		t.Parallel()
		ct := at.Clone()

		for i, r := range routes {
			// update value
			if i%42 == 0 {
				ct.Modify(r.CIDR, func(oldVal int, _ bool) (int, bool) { return oldVal + 1, false })
			}
		}

		if at.Equal(ct) {
			t.Error("expected false, got true")
		}
	})
}
