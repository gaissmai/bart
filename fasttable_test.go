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

func TestFastNil(t *testing.T) {
	t.Parallel()

	ip4 := mpa("127.0.0.1")
	ip6 := mpa("::1")

	pfx4 := mpp("127.0.0.0/8")
	pfx6 := mpp("::1/128")

	fast2 := new(Fast[any])
	fast2.Insert(pfx4, nil)
	fast2.Insert(pfx6, nil)

	var fast1 *Fast[any] = nil

	t.Run("mustPanic", func(t *testing.T) {
		t.Parallel()

		mustPanic(t, "sizeUpdate", func() { fast1.sizeUpdate(true, 1) })
		mustPanic(t, "sizeUpdate", func() { fast1.sizeUpdate(false, 1) })
		mustPanic(t, "rootNodeByVersion", func() { fast1.rootNodeByVersion(true) })
		mustPanic(t, "rootNodeByVersion", func() { fast1.rootNodeByVersion(false) })
		mustPanic(t, "fprint", func() { fast1.fprint(nil, true) })
		mustPanic(t, "fprint", func() { fast1.fprint(nil, false) })

		mustPanic(t, "Size", func() { fast1.Size() })
		mustPanic(t, "Size4", func() { fast1.Size4() })
		mustPanic(t, "Size6", func() { fast1.Size6() })

		mustPanic(t, "Get", func() { fast1.Get(pfx4) })
		mustPanic(t, "Insert", func() { fast1.Insert(pfx4, nil) })
		mustPanic(t, "InsertPersist", func() { fast1.InsertPersist(pfx4, nil) })
		mustPanic(t, "Delete", func() { fast1.Delete(pfx4) })
		mustPanic(t, "DeletePersist", func() { fast1.DeletePersist(pfx4) })
		mustPanic(t, "Modify", func() { fast1.Modify(pfx4, nil) })
		mustPanic(t, "ModifyPersist", func() { fast1.Modify(pfx4, nil) })
		mustPanic(t, "Contains", func() { fast1.Contains(ip4) })
		mustPanic(t, "Lookup", func() { fast1.Lookup(ip6) })
		mustPanic(t, "LookupPrefix", func() { fast1.LookupPrefix(pfx4) })
		mustPanic(t, "LookupPrefixLPM", func() { fast1.LookupPrefixLPM(pfx4) })
		mustPanic(t, "Union", func() { fast1.Union(fast2) })
		mustPanic(t, "UnionPersist", func() { fast1.UnionPersist(fast2) })
	})

	t.Run("noPanic", func(t *testing.T) {
		t.Parallel()

		noPanic(t, "Overlaps", func() { fast1.Overlaps(nil) })
		noPanic(t, "Overlaps4", func() { fast1.Overlaps4(nil) })
		noPanic(t, "Overlaps6", func() { fast1.Overlaps6(nil) })

		noPanic(t, "Overlaps", func() { fast2.Overlaps(fast2) })
		noPanic(t, "Overlaps4", func() { fast2.Overlaps4(fast2) })
		noPanic(t, "Overlaps6", func() { fast2.Overlaps6(fast2) })

		mustPanic(t, "Overlaps", func() { fast1.Overlaps(fast2) })
		mustPanic(t, "Overlaps4", func() { fast1.Overlaps4(fast2) })
		mustPanic(t, "Overlaps6", func() { fast1.Overlaps6(fast2) })

		mustPanic(t, "Equal", func() { fast1.Equal(fast2) })
		noPanic(t, "Equal", func() { fast1.Equal(fast1) })
		noPanic(t, "Equal", func() { fast2.Equal(fast2) })

		noPanic(t, "dump", func() { fast1.dump(nil) })
		noPanic(t, "dumpString", func() { fast1.dumpString() })
		noPanic(t, "Clone", func() { fast1.Clone() })
		noPanic(t, "DumpList4", func() { fast1.DumpList4() })
		noPanic(t, "DumpList6", func() { fast1.DumpList6() })
		noPanic(t, "Fprint", func() { fast1.Fprint(nil) })
		noPanic(t, "MarshalJSON", func() { _, _ = fast1.MarshalJSON() })
		noPanic(t, "MarshalText", func() { _, _ = fast1.MarshalText() })
	})

	t.Run("noPanicRangeOverFunc", func(t *testing.T) {
		t.Parallel()

		noPanicRangeOverFunc[any](t, "All", fast1.All)
		noPanicRangeOverFunc[any](t, "All4", fast1.All4)
		noPanicRangeOverFunc[any](t, "All6", fast1.All6)
		noPanicRangeOverFunc[any](t, "AllSorted", fast1.AllSorted)
		noPanicRangeOverFunc[any](t, "AllSorted4", fast1.AllSorted4)
		noPanicRangeOverFunc[any](t, "AllSorted6", fast1.AllSorted6)
		noPanicRangeOverFunc[any](t, "Subnets", fast1.Subnets)
		noPanicRangeOverFunc[any](t, "Supernets", fast1.Supernets)
	})
}

func TestFastInvalid(t *testing.T) {
	t.Parallel()

	fast1 := new(Fast[any])
	fast2 := new(Fast[any])

	var zeroIP netip.Addr
	var zeroPfx netip.Prefix

	noPanic(t, "All", func() { fast1.All() })
	noPanic(t, "All4", func() { fast1.All4() })
	noPanic(t, "All6", func() { fast1.All6() })
	noPanic(t, "AllSorted", func() { fast1.AllSorted() })
	noPanic(t, "AllSorted4", func() { fast1.AllSorted4() })
	noPanic(t, "AllSorted6", func() { fast1.AllSorted6() })
	noPanic(t, "Clone", func() { fast1.Clone() })
	noPanic(t, "Contains", func() { fast1.Contains(zeroIP) })
	noPanic(t, "Delete", func() { fast1.Delete(zeroPfx) })
	noPanic(t, "DeletePersist", func() { fast1.DeletePersist(zeroPfx) })
	noPanic(t, "DumpList4", func() { fast1.DumpList4() })
	noPanic(t, "DumpList6", func() { fast1.DumpList6() })
	noPanic(t, "Equal", func() { fast1.Equal(fast2) })
	noPanic(t, "Fprint", func() { fast1.Fprint(nil) })
	noPanic(t, "Get", func() { fast1.Get(zeroPfx) })
	noPanic(t, "Insert", func() { fast1.Insert(zeroPfx, nil) })
	noPanic(t, "InsertPersist", func() { fast1.InsertPersist(zeroPfx, nil) })
	noPanic(t, "Lookup", func() { fast1.Lookup(zeroIP) })
	noPanic(t, "LookupPrefix", func() { fast1.LookupPrefix(zeroPfx) })
	noPanic(t, "LookupPrefixLPM", func() { fast1.LookupPrefixLPM(zeroPfx) })
	noPanic(t, "MarshalJSON", func() { _, _ = fast1.MarshalJSON() })
	noPanic(t, "MarshalText", func() { _, _ = fast1.MarshalText() })
	noPanic(t, "Modify", func() { fast1.Modify(zeroPfx, nil) })
	noPanic(t, "ModifyPersist", func() { fast1.ModifyPersist(zeroPfx, nil) })
	noPanic(t, "Overlaps", func() { fast1.Overlaps(fast2) })
	noPanic(t, "Overlaps4", func() { fast1.Overlaps4(fast2) })
	noPanic(t, "Overlaps6", func() { fast1.Overlaps6(fast2) })
	noPanic(t, "OverlapsPrefix", func() { fast1.OverlapsPrefix(zeroPfx) })
	noPanic(t, "Size", func() { fast1.Size() })
	noPanic(t, "Size4", func() { fast1.Size4() })
	noPanic(t, "Size6", func() { fast1.Size6() })
	noPanic(t, "Subnets", func() { fast1.Subnets(zeroPfx) })
	noPanic(t, "Supernets", func() { fast1.Supernets(zeroPfx) })
	noPanic(t, "Union", func() { fast1.Union(fast2) })
	noPanic(t, "UnionPersist", func() { fast1.UnionPersist(fast2) })
}

func TestFastContainsCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Fast's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[int])
	fast := new(Fast[int])

	for i, p := range pfxs {
		gold.insert(p, i) // ensures Masked + de-dupe
		fast.Insert(p, i)
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

func TestFastLookupCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Fast's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[int])
	fast := new(Fast[int])

	for i, pfx := range pfxs {
		gold.insert(pfx, i)
		fast.Insert(pfx, i)
	}

	for range n {
		a := randomAddr(prng)

		goldVal, goldOK := gold.lookup(a)
		fastVal, fastOK := fast.Lookup(a)

		if !getsEqual(goldVal, goldOK, fastVal, fastOK) {
			t.Fatalf("Lookup(%q) = (%v, %v), want (%v, %v)", a, fastVal, fastOK, goldVal, goldOK)
		}
	}
}

func TestFastLookupPrefixUnmasked(t *testing.T) {
	// test that the pfx must not be masked on input for LookupPrefix
	t.Parallel()

	fast := new(Fast[any])
	fast.Insert(mpp("10.20.30.0/24"), nil)

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
		_, got := fast.LookupPrefix(tc.probe)
		if got != tc.wantOk {
			t.Errorf("LookupPrefix non canonical prefix (%s), got: %v, want: %v", tc.probe, got, tc.wantOk)
		}

		lpm, _, got := fast.LookupPrefixLPM(tc.probe)
		if got != tc.wantOk {
			t.Errorf("LookupPrefixLPM non canonical prefix (%s), got: %v, want: %v", tc.probe, got, tc.wantOk)
		}
		if lpm != tc.wantLPM {
			t.Errorf("LookupPrefixLPM non canonical prefix (%s), got: %v, want: %v", tc.probe, lpm, tc.wantLPM)
		}
	}
}

func TestFastLookupPrefixCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Fast's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[int])
	fast := new(Fast[int])
	for i, pfx := range pfxs {
		gold.insert(pfx, i)
		fast.Insert(pfx, i)
	}

	for range n {
		pfx := randomPrefix(prng)

		goldVal, goldOK := gold.lookupPfx(pfx)
		fastVal, fastOK := fast.LookupPrefix(pfx)

		if !getsEqual(goldVal, goldOK, fastVal, fastOK) {
			t.Fatalf("LookupPrefix(%q) = (%v, %v), want (%v, %v)", pfx, fastVal, fastOK, goldVal, goldOK)
		}
	}
}

func TestFastLookupPrefixLPMCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Fast's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, n)

	gold := new(goldTable[int])
	gold.insertMany(pfxs)

	fast := new(Fast[int])
	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	for range n {
		pfx := randomPrefix(prng)

		goldLPM, goldVal, goldOK := gold.lookupPfxLPM(pfx)
		fastLPM, fastVal, fastOK := fast.LookupPrefixLPM(pfx)

		if !getsEqual(goldVal, goldOK, fastVal, fastOK) {
			t.Fatalf("LookupPrefixLPM(%q) = (%v, %v), want (%v, %v)", pfx, fastVal, fastOK, goldVal, goldOK)
		}

		if !getsEqual(goldLPM, goldOK, fastLPM, fastOK) {
			t.Fatalf("LookupPrefixLPM(%q) = (%v, %v), want (%v, %v)", pfx, fastLPM, fastOK, goldLPM, goldOK)
		}
	}
}

func TestFastInsertShuffled(t *testing.T) {
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

		fast1 := new(Fast[string])
		fast2 := new(Fast[string])

		for _, pfx := range pfxs {
			fast1.Insert(pfx, pfx.String())
			fast1.Insert(pfx, pfx.String()) // idempotent
		}
		for _, pfx := range pfxs2 {
			fast2.Insert(pfx, pfx.String()) // idempotent
		}

		if !fast1.Equal(fast2) {
			t.Fatal("expected Equal")
		}
	}
}

func TestFastInsertPersistShuffled(t *testing.T) {
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

		fast1 := new(Fast[string])
		fast2 := new(Fast[string])

		// fast1 is mutable
		for _, pfx := range pfxs {
			fast1.Insert(pfx, pfx.String())
		}

		// fast2 is persistent
		for _, pfx := range pfxs2 {
			fast2 = fast2.InsertPersist(pfx, pfx.String())
		}

		if fast1.dumpString() != fast2.dumpString() {
			t.Fatal("mutable and immutable table have different dumpString representation")
		}

		if !fast1.Equal(fast2) {
			t.Fatal("expected Equal")
		}
	}
}

func TestFastDeleteCompare(t *testing.T) {
	// Create large route tables repeatedly, delete half of their
	// prefixes, and compare Fast's behavior to a naive and slow but
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
		fast := new(Fast[string])

		for _, pfx := range pfxs {
			gold.insert(pfx, pfx.String())
			fast.Insert(pfx, pfx.String())
		}

		for _, pfx := range toDelete {
			gold.delete(pfx)
			fast.Delete(pfx)
		}

		gold.sort()

		fastGolden := fast.dumpAsGoldTable()
		fastGolden.sort()

		if !slices.Equal(*gold, fastGolden) {
			t.Fatal("expected Equal")
		}
	}
}

func TestFastDeleteShuffled(t *testing.T) {
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

		fast := new(Fast[string])

		// insert
		for _, pfx := range pfxs {
			fast.Insert(pfx, pfx.String())
		}
		for _, pfx := range toDelete {
			fast.Insert(pfx, pfx.String())
		}

		// delete
		for _, pfx := range toDelete {
			fast.Delete(pfx)
		}

		pfxs2 := slices.Clone(pfxs)
		toDelete2 := slices.Clone(toDelete)
		prng.Shuffle(len(toDelete2), func(i, j int) { toDelete2[i], toDelete2[j] = toDelete2[j], toDelete2[i] })

		fast2 := new(Fast[string])

		// insert
		for _, pfx := range pfxs2 {
			fast2.Insert(pfx, pfx.String())
		}
		for _, pfx := range toDelete2 {
			fast2.Insert(pfx, pfx.String())
		}

		// delete
		for _, pfx := range toDelete2 {
			fast2.Delete(pfx)
		}

		if !fast.Equal(fast2) {
			t.Fatal("expect equal")
		}
	}
}

func TestFastDeleteIsReverseOfInsert(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	// Insert n prefixes, then delete those same prefixes in reverse
	// order. Each deletion should exactly undo the internal structure
	// changes that each insert did.

	fast := new(Fast[string])
	want := fast.dumpString()

	prefixes := randomRealWorldPrefixes(prng, n)

	defer func() {
		if t.Failed() {
			t.Logf("the prefixes that fail the test: %v\n", prefixes)
		}
	}()

	for _, p := range prefixes {
		fast.Insert(p, p.String())
	}

	for i := len(prefixes) - 1; i >= 0; i-- {
		fast.Delete(prefixes[i])
	}
	if got := fast.dumpString(); got != want {
		t.Fatalf("after delete, mismatch:\n\n got: %s\n\nwant: %s", got, want)
	}
}

func TestFastDeleteButOne(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))
	// Insert n prefixes, then delete all but one
	n := workLoadN()

	for range 10 {

		fast := new(Fast[any])
		prefixes := randomRealWorldPrefixes(prng, n)

		for _, p := range prefixes {
			fast.Insert(p, nil)
		}

		// shuffle the prefixes
		prng.Shuffle(n, func(i, j int) {
			prefixes[i], prefixes[j] = prefixes[j], prefixes[i]
		})

		// skip the first
		for i := 1; i < len(prefixes); i++ {
			fast.Delete(prefixes[i])
		}

		stats4 := fast.root4.StatsRec()
		stats6 := fast.root6.StatsRec()

		if nodes := stats4.Nodes + stats6.Nodes; nodes != 1 {
			t.Fatalf("delete but one, want nodes: 1, got: %d\n%s", nodes, fast.dumpString())
		}

		sum := stats4.Pfxs + stats4.Leaves + stats4.Fringes +
			stats6.Pfxs + stats6.Leaves + stats6.Fringes

		if sum != 1 {
			t.Fatalf("delete but one, only one item must be left, but: %d\n%s", sum, fast.dumpString())
		}
	}
}

func TestFastGet(t *testing.T) {
	t.Parallel()

	t.Run("empty table", func(t *testing.T) {
		t.Parallel()
		prng := rand.New(rand.NewPCG(42, 42))

		fast := new(Fast[int])
		pfx := randomPrefix(prng)
		_, ok := fast.Get(pfx)

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

	fast := new(Fast[int])
	for _, tt := range tests {
		fast.Insert(tt.pfx, tt.val)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := fast.Get(tt.pfx)

			if !ok {
				t.Errorf("%s: ok=%v, expected: %v", tt.name, ok, true)
			}

			if got != tt.val {
				t.Errorf("%s: val=%v, expected: %v", tt.name, got, tt.val)
			}
		})
	}
}

func TestFastGetCompare(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[string])
	fast := new(Fast[string])
	for _, pfx := range pfxs {
		gold.insert(pfx, pfx.String())
		fast.Insert(pfx, pfx.String())
	}

	for _, pfx := range pfxs {
		goldVal, goldOK := gold.get(pfx)
		fastVal, fastOK := fast.Get(pfx)

		if !getsEqual(goldVal, goldOK, fastVal, fastOK) {
			t.Fatalf("Get(%q) = (%v, %v), want (%v, %v)", pfx, fastVal, fastOK, goldVal, goldOK)
		}
	}
}

func TestFastModifySemantics(t *testing.T) {
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
		fast := new(Fast[int])

		// Insert initial entries using Modify
		for pfx, v := range tt.prepare {
			fast.Modify(pfx, func(_ int, _ bool) (_ int, del bool) { return v, false })
		}

		fast.Modify(tt.args.pfx, tt.args.cb)

		// Check the final state of the table using Get, compares expected and actual table
		got := make(map[netip.Prefix]int, len(tt.finalData))
		for pfx, val := range fast.All() {
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

func TestFastModifyPersistSemantics(t *testing.T) {
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
		fast := new(Fast[int])

		// Insert initial entries using Modify
		for pfx, v := range tt.prepare {
			fast.Modify(pfx, func(_ int, _ bool) (_ int, del bool) { return v, false })
		}

		prt := fast.ModifyPersist(tt.args.pfx, tt.args.cb)

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

func TestFastModifyCompare(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[int])
	fast := new(Fast[int])

	// Update as insert
	for i, pfx := range pfxs {
		gold.insert(pfx, i)
		fast.Modify(pfx, func(int, bool) (int, bool) { return i, false })
	}

	gold.sort()
	fastGolden := fast.dumpAsGoldTable()
	fastGolden.sort()

	if !slices.Equal(*gold, fastGolden) {
		t.Fatal("expected Equal")
	}

	cb1 := func(val int, _ bool) int { return val + 1 }
	cb2 := func(val int, _ bool) (int, bool) { return val + 1, false }

	// Update as update
	for _, pfx := range pfxs[:len(pfxs)/2] {
		gold.update(pfx, cb1)
		fast.Modify(pfx, cb2)
	}

	gold.sort()
	fastGolden = fast.dumpAsGoldTable()
	fastGolden.sort()

	if !slices.Equal(*gold, fastGolden) {
		t.Fatal("expected Equal")
	}
}

func TestFastModifyPersistCompare(t *testing.T) {
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

func TestFastModifyShuffled(t *testing.T) {
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

		fast1 := new(Fast[string])

		// insert
		for _, pfx := range pfxs {
			fast1.Insert(pfx, pfx.String())
		}
		for _, pfx := range toDelete {
			fast1.Insert(pfx, pfx.String())
		}

		// this callback deletes unconditionally
		cb := func(string, bool) (string, bool) { return "", true }

		// delete
		for _, pfx := range toDelete {
			fast1.Modify(pfx, cb)
		}

		pfxs2 := slices.Clone(pfxs)
		toDelete2 := slices.Clone(toDelete)
		prng.Shuffle(len(toDelete2), func(i, j int) { toDelete2[i], toDelete2[j] = toDelete2[j], toDelete2[i] })

		fast2 := new(Fast[string])

		// insert
		for _, pfx := range pfxs2 {
			fast2.Insert(pfx, pfx.String())
		}
		for _, pfx := range toDelete2 {
			fast2.Insert(pfx, pfx.String())
		}

		// delete
		for _, pfx := range toDelete2 {
			fast2.Modify(pfx, cb)
		}

		if !fast1.Equal(fast2) {
			t.Fatal("expected equal")
		}
	}
}

func TestFastModifyPersistShuffled(t *testing.T) {
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

		fast1 := new(Fast[string])

		// insert
		for _, pfx := range pfxs {
			fast1.Insert(pfx, pfx.String())
		}
		for _, pfx := range toDelete {
			fast1.Insert(pfx, pfx.String())
		}

		// this callback deletes unconditionally
		cb := func(string, bool) (string, bool) { return "", true }

		// delete
		for _, pfx := range toDelete {
			fast1 = fast1.ModifyPersist(pfx, cb)
		}

		pfxs2 := slices.Clone(pfxs)
		toDelete2 := slices.Clone(toDelete)
		prng.Shuffle(len(toDelete2), func(i, j int) { toDelete2[i], toDelete2[j] = toDelete2[j], toDelete2[i] })

		fast2 := new(Fast[string])

		// insert
		for _, pfx := range pfxs2 {
			fast2.Insert(pfx, pfx.String())
		}
		for _, pfx := range toDelete2 {
			fast2.Insert(pfx, pfx.String())
		}

		// delete
		for _, pfx := range toDelete2 {
			fast2 = fast2.ModifyPersist(pfx, cb)
		}

		if !fast1.Equal(fast2) {
			t.Fatal("expected equal")
		}
	}
}

// TestUnionMemoryAliasing tests that the Union method does not alias memory
// between the two tables.
func TestFastUnionMemoryAliasing(t *testing.T) {
	t.Parallel()

	newFast := func(pfx ...string) *Fast[struct{}] {
		rt := new(Fast[struct{}])
		for _, s := range pfx {
			rt.Insert(mpp(s), struct{}{})
		}
		return rt
	}

	// First create two tables with disjoint prefixes.
	stable := newFast("0.0.0.0/24")
	temp := newFast("100.69.1.0/24")

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
func TestFastUnionPersistMemoryAliasing(t *testing.T) {
	t.Parallel()

	newFast := func(pfx ...string) *Fast[struct{}] {
		rt := new(Fast[struct{}])
		for _, s := range pfx {
			rt.Insert(mpp(s), struct{}{})
		}
		return rt
	}
	// First create two tables with disjoint prefixes.
	a := newFast("100.69.1.0/24")
	b := newFast("0.0.0.0/24")

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

func TestFastUnionCompare(t *testing.T) {
	t.Parallel()

	n := workLoadN()
	prng := rand.New(rand.NewPCG(42, 42))

	for range 3 {
		pfxs := randomRealWorldPrefixes(prng, n)

		gold := new(goldTable[string])
		fast := new(Fast[string])

		for _, pfx := range pfxs {
			gold.insert(pfx, pfx.String())
			fast.Insert(pfx, pfx.String())
		}

		pfxs2 := randomRealWorldPrefixes(prng, n)

		gold2 := new(goldTable[string])
		fast2 := new(Fast[string])

		for _, pfx := range pfxs2 {
			gold2.insert(pfx, pfx.String())
			fast2.Insert(pfx, pfx.String())
		}

		gold.union(gold2)
		fast.Union(fast2)

		// dump as slow table for comparison
		fastAsGoldenTbl := fast.dumpAsGoldTable()

		// sort for comparison
		gold.sort()
		fastAsGoldenTbl.sort()

		if !slices.Equal(*gold, fastAsGoldenTbl) {
			t.Fatal("expected equal")
		}
	}
}

func TestFastUnionPersistCompare(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))

	n := workLoadN()

	for range 3 {
		pfxs := randomRealWorldPrefixes(prng, n)

		gold := new(goldTable[int])
		fast := new(Fast[int])

		for i, pfx := range pfxs {
			gold.insert(pfx, i)
			fast.Insert(pfx, i)
		}

		pfxs2 := randomRealWorldPrefixes(prng, n)

		gold2 := new(goldTable[int])
		fast2 := new(Fast[int])

		for i, pfx := range pfxs2 {
			gold2.insert(pfx, i)
			fast2.Insert(pfx, i)
		}

		gold.union(gold2)
		fastP := fast.UnionPersist(fast2)

		// dump as slow table for comparison
		fastAsGoldenTbl := fastP.dumpAsGoldTable()

		// sort for comparison
		gold.sort()
		fastAsGoldenTbl.sort()

		if !slices.Equal(*gold, fastAsGoldenTbl) {
			t.Fatal("expected equal")
		}
	}
}

func TestFastClone(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, 100_000)

	var fast *Fast[int]
	if fast.Clone() != nil {
		t.Fatal("expected nil")
	}

	fast = new(Fast[int])
	for i, pfx := range pfxs {
		fast.Insert(pfx, i)
	}
	clone := fast.Clone()

	if !fast.Equal(clone) {
		t.Fatal("expected equal")
	}
}

func TestFastCloneShallow(t *testing.T) {
	t.Parallel()

	fast := new(Fast[*int])
	clone := fast.Clone()
	if fast.dumpString() != clone.dumpString() {
		t.Errorf("empty Clone: got:\n%swant:\n%s", clone.dumpString(), fast.dumpString())
	}

	val := 1
	pfx := mpp("10.0.0.1/32")
	fast.Insert(pfx, &val)

	clone = fast.Clone()
	want, _ := fast.Get(pfx)
	got, _ := clone.Get(pfx)

	if *got != *want || got != want {
		t.Errorf("shallow copy, values and pointers must be equal:\nvalues(%d, %d)\n(ptr(%v, %v)", *got, *want, got, want)
	}

	// update value, shallow copy of values, clone must be equal
	val = 2
	want, _ = fast.Get(pfx)
	got, _ = clone.Get(pfx)

	if *got != *want {
		t.Errorf("memory aliasing after shallow copy, values must be equal:\nvalues(%d, %d)", *got, *want)
	}
}

func TestFastCloneDeep(t *testing.T) {
	t.Parallel()

	fast := new(Fast[*MyInt])
	clone := fast.Clone()
	if fast.dumpString() != clone.dumpString() {
		t.Errorf("empty Clone: got:\n%swant:\n%s", clone.dumpString(), fast.dumpString())
	}

	val := MyInt(1)
	pfx := mpp("10.0.0.1/32")
	fast.Insert(pfx, &val)

	clone = fast.Clone()
	want, _ := fast.Get(pfx)
	got, _ := clone.Get(pfx)

	if *got != *want || got == want {
		t.Errorf("value with Cloner interface, pointers must be different:\nvalues(%d, %d)\n(ptr(%v, %v)", *got, *want, got, want)
	}

	// update value, deep copy of values, cloned value must now be different
	val = 2
	want, _ = fast.Get(pfx)
	got, _ = clone.Get(pfx)

	if *got == *want {
		t.Errorf("memory aliasing after deep copy, values must be different:\nvalues(%d, %d)", *got, *want)
	}
}

func TestFastUnionShallow(t *testing.T) {
	t.Parallel()

	fast1 := new(Fast[*int])
	fast2 := new(Fast[*int])

	val := 1
	pfx := mpp("10.0.0.1/32")
	fast2.Insert(pfx, &val)

	fast1.Union(fast2)
	got, _ := fast1.Get(pfx)
	want, _ := fast2.Get(pfx)

	if *got != *want || got != want {
		t.Errorf("shallow copy, values and pointers must be equal:\nvalues(%d, %d)\n(ptr(%v, %v)", *got, *want, got, want)
	}

	// update value, shallow copy of values, union must be equal
	val = 2
	got, _ = fast1.Get(pfx)
	want, _ = fast2.Get(pfx)

	if *got != *want {
		t.Errorf("memory aliasing after shallow copy, values must be equal:\nvalues(%d, %d)", *got, *want)
	}
}

func TestFastUnionDeep(t *testing.T) {
	t.Parallel()

	fast1 := new(Fast[*MyInt])
	fast2 := new(Fast[*MyInt])

	val := MyInt(1)
	pfx := mpp("10.0.0.1/32")
	fast2.Insert(pfx, &val)

	fast1.Union(fast2)
	got, _ := fast1.Get(pfx)
	want, _ := fast2.Get(pfx)

	if *got != *want || got == want {
		t.Errorf("value with Cloner interface, pointers must be different:\nvalues(%d, %d)\n(ptr(%v, %v)", *got, *want, got, want)
	}

	// update value, shallow copy of values, union must be equal
	val = 2
	got, _ = fast1.Get(pfx)
	want, _ = fast2.Get(pfx)

	if *got == *want {
		t.Errorf("memory aliasing after deep copy, values must be different:\nvalues(%d, %d)", *got, *want)
	}
}

// test some edge cases
func TestFastOverlapsPrefixEdgeCases(t *testing.T) {
	t.Parallel()

	fast := new(Fast[int])

	// empty table
	checkOverlapsPrefix(t, fast, []tableOverlapsTest{
		{"0.0.0.0/0", false},
		{"::/0", false},
	})

	// default route
	fast.Insert(mpp("10.0.0.0/9"), 0)
	fast.Insert(mpp("2001:db8::/32"), 0)
	checkOverlapsPrefix(t, fast, []tableOverlapsTest{
		{"0.0.0.0/0", true},
		{"::/0", true},
	})

	// default route
	fast = new(Fast[int])
	fast.Insert(mpp("0.0.0.0/0"), 0)
	fast.Insert(mpp("::/0"), 0)
	checkOverlapsPrefix(t, fast, []tableOverlapsTest{
		{"10.0.0.0/9", true},
		{"2001:db8::/32", true},
	})

	// single IP
	fast = new(Fast[int])
	fast.Insert(mpp("10.0.0.0/7"), 0)
	fast.Insert(mpp("2001::/16"), 0)
	checkOverlapsPrefix(t, fast, []tableOverlapsTest{
		{"10.1.2.3/32", true},
		{"2001:db8:affe::cafe/128", true},
	})

	// single IP
	fast = new(Fast[int])
	fast.Insert(mpp("10.1.2.3/32"), 0)
	fast.Insert(mpp("2001:db8:affe::cafe/128"), 0)
	checkOverlapsPrefix(t, fast, []tableOverlapsTest{
		{"10.0.0.0/7", true},
		{"2001::/16", true},
	})

	// same IPv
	fast = new(Fast[int])
	fast.Insert(mpp("10.1.2.3/32"), 0)
	fast.Insert(mpp("2001:db8:affe::cafe/128"), 0)
	checkOverlapsPrefix(t, fast, []tableOverlapsTest{
		{"10.1.2.3/32", true},
		{"2001:db8:affe::cafe/128", true},
	})
}

func TestFastSize(t *testing.T) {
	t.Parallel()

	n := workLoadN()
	prng := rand.New(rand.NewPCG(42, 42))

	fast := new(Fast[any])
	if fast.Size() != 0 {
		t.Errorf("empty Fast: want: 0, got: %d", fast.Size())
	}

	if fast.Size4() != 0 {
		t.Errorf("empty Fast: want: 0, got: %d", fast.Size4())
	}

	if fast.Size6() != 0 {
		t.Errorf("empty Fast: want: 0, got: %d", fast.Size6())
	}

	pfxs1 := randomRealWorldPrefixes(prng, n)
	pfxs2 := randomRealWorldPrefixes(prng, n)

	for _, pfx := range pfxs1 {
		fast.Insert(pfx, nil)
	}

	for _, pfx := range pfxs2 {
		fast.Modify(pfx, func(any, bool) (any, bool) { return nil, false })
	}

	pfxs1 = append(pfxs1, pfxs2...)

	for _, pfx := range pfxs1[:n] {
		fast.Modify(pfx, func(any, bool) (any, bool) { return nil, false })
	}

	for _, pfx := range randomRealWorldPrefixes(prng, n) {
		fast.Delete(pfx)
	}

	var allInc4 int
	var allInc6 int

	for range fast.AllSorted4() {
		allInc4++
	}

	for range fast.AllSorted6() {
		allInc6++
	}

	if allInc4 != fast.Size4() {
		t.Errorf("Size4: want: %d, got: %d", allInc4, fast.Size4())
	}

	if allInc6 != fast.Size6() {
		t.Errorf("Size6: want: %d, got: %d", allInc6, fast.Size6())
	}
}

// TestAll tests All with random samples
func TestFastAll(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	fast := new(Fast[string])

	// Insert all prefixes with their values
	for _, pfx := range pfxs {
		fast.Insert(pfx, pfx.String())
	}

	// Collect all prefixes from All
	gotPrefixes := make([]netip.Prefix, 0, n)
	gotValues := make([]string, 0, n)

	for pfx, val := range fast.All() {
		gotPrefixes = append(gotPrefixes, pfx)
		gotValues = append(gotValues, val)
	}

	// Collect all prefixes from All4 and All6
	got4Prefixes := make([]netip.Prefix, 0, n)
	got4Values := make([]string, 0, n)

	for pfx, val := range fast.All4() {
		got4Prefixes = append(got4Prefixes, pfx)
		got4Values = append(got4Values, val)
	}

	got6Prefixes := make([]netip.Prefix, 0, n)
	got6Values := make([]string, 0, n)
	for pfx, val := range fast.All6() {
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

func TestFastAllSorted(t *testing.T) {
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

			fast := new(Fast[int])

			// Insert prefixes with index as value
			for i, prefixStr := range tc.prefixes {
				pfx := netip.MustParsePrefix(prefixStr)
				fast.Insert(pfx, i)
			}

			// Collect sorted results
			var actualOrder []string
			for pfx := range fast.AllSorted() {
				actualOrder = append(actualOrder, pfx.String())
			}

			// Verify the order matches expected
			if len(actualOrder) != len(tc.expected) {
				t.Fatalf("%s: Expected %d results, got %d", tc.name, len(tc.expected), len(actualOrder))
			}

			// Collect sorted 4 results
			var actual4Order []string
			for pfx := range fast.AllSorted4() {
				actual4Order = append(actual4Order, pfx.String())
			}

			// Collect sorted 6 results
			var actual6Order []string
			for pfx := range fast.AllSorted6() {
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

func TestFastSubnets(t *testing.T) {
	t.Parallel()

	var zeroPfx netip.Prefix

	t.Run("empty table", func(t *testing.T) {
		t.Parallel()
		fast := new(Fast[string])
		pfx := mpp("::1/128")

		for range fast.Subnets(pfx) {
			t.Errorf("empty table, must not range over")
		}
	})

	t.Run("invalid prefix", func(t *testing.T) {
		fast := new(Fast[string])
		pfx := mpp("::1/128")
		val := "foo"
		fast.Insert(pfx, val)
		for range fast.Subnets(zeroPfx) {
			t.Errorf("invalid prefix, must not range over")
		}
	})

	t.Run("identity", func(t *testing.T) {
		fast := new(Fast[string])
		pfx := mpp("::1/128")
		val := "foo"
		fast.Insert(pfx, val)

		for p, v := range fast.Subnets(pfx) {
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

		fast := new(Fast[int])
		for i, pfx := range randomRealWorldPrefixes4(prng, want4) {
			fast.Insert(pfx, i)
		}
		for i, pfx := range randomRealWorldPrefixes6(prng, want6) {
			fast.Insert(pfx, i)
		}

		// default gateway v4 covers all v4 prefixes in table
		dg4 := mpp("0.0.0.0/0")
		got4 := 0
		for range fast.Subnets(dg4) {
			got4++
		}

		// default gateway v6 covers all v6 prefixes in table
		dg6 := mpp("::/0")
		got6 := 0
		for range fast.Subnets(dg6) {
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

func TestFastSubnetsCompare(t *testing.T) {
	t.Parallel()
	n := workLoadN()
	prng := rand.New(rand.NewPCG(42, 42))

	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[int])
	fast := new(Fast[int])

	for i, pfx := range pfxs {
		gold.insert(pfx, i)
		fast.Insert(pfx, i)
	}

	for _, pfx := range randomRealWorldPrefixes(prng, n) {
		t.Run("subtest", func(t *testing.T) {
			t.Parallel()

			gotGold := gold.subnets(pfx)
			gotBart := []netip.Prefix{}
			for pfx := range fast.Subnets(pfx) {
				gotBart = append(gotBart, pfx)
			}
			if !slices.Equal(gotGold, gotBart) {
				t.Fatalf("Subnets(%q) = %v, want %v", pfx, gotBart, gotGold)
			}
		})
	}
}

func TestFastSupernetsEdgeCase(t *testing.T) {
	t.Parallel()

	var zeroPfx netip.Prefix

	t.Run("empty table", func(t *testing.T) {
		t.Parallel()
		fast := new(Fast[any])
		pfx := mpp("::1/128")

		fast.Supernets(pfx)(func(_ netip.Prefix, _ any) bool {
			t.Errorf("empty table, must not range over")
			return false
		})
	})

	t.Run("invalid prefix", func(t *testing.T) {
		fast := new(Fast[any])
		pfx := mpp("::1/128")
		val := "foo"
		fast.Insert(pfx, val)

		fast.Supernets(zeroPfx)(func(_ netip.Prefix, _ any) bool {
			t.Errorf("invalid prefix, must not range over")
			return false
		})
	})

	t.Run("identity", func(t *testing.T) {
		fast := new(Fast[string])
		pfx := mpp("::1/128")
		val := "foo"
		fast.Insert(pfx, val)

		for p, v := range fast.Supernets(pfx) {
			if p != pfx {
				t.Errorf("Supernets(%v), got: %v, want: %v", pfx, p, pfx)
			}

			if v != val {
				t.Errorf("Supernets(%v), got: %v, want: %v", pfx, v, val)
			}
		}
	})
}

func TestFastSupernetsCompare(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))

	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[int])
	fast := new(Fast[int])

	for i, pfx := range pfxs {
		gold.insert(pfx, i)
		fast.Insert(pfx, i)
	}

	for _, pfx := range randomRealWorldPrefixes(prng, n) {
		t.Run("subtest", func(t *testing.T) {
			t.Parallel()
			gotGold := gold.supernets(pfx)
			gotBart := []netip.Prefix{}

			for p := range fast.Supernets(pfx) {
				gotBart = append(gotBart, p)
			}

			if !slices.Equal(gotGold, gotBart) {
				t.Fatalf("Supernets(%q) = %v, want %v", pfx, gotBart, gotGold)
			}
		})
	}
}

func TestFastMarshalText(t *testing.T) {
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
			fast := new(Fast[string])

			// Insert test data
			for prefix, value := range tt.expectedData {
				fast.Insert(prefix, value)
			}

			data, err := fast.MarshalText()
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

func TestFastMarshalJSON(t *testing.T) {
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
			fast := new(Fast[any])

			// Insert test data
			for prefix, value := range tt.expectedData {
				fast.Insert(prefix, value)
			}

			jsonData, err := json.Marshal(fast)
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

func TestFastDumpList4(t *testing.T) {
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
			fast := new(Fast[string])

			// Insert test data
			for prefix, value := range tt.expectedData {
				fast.Insert(prefix, value)
			}

			dumpList := fast.DumpList4()

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

func TestFastDumpList6(t *testing.T) {
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
			fast := new(Fast[string])

			// Insert test data
			for prefix, value := range tt.expectedData {
				fast.Insert(prefix, value)
			}

			dumpList := fast.DumpList6()

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

func TestFastEqual(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		buildA    func() *Fast[stringVal]
		buildB    func() *Fast[stringVal]
		wantEqual bool
	}{
		{
			name:      "empty tables",
			buildA:    func() *Fast[stringVal] { return new(Fast[stringVal]) },
			buildB:    func() *Fast[stringVal] { return new(Fast[stringVal]) },
			wantEqual: true,
		},
		{
			name: "same single entry",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "foo")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "foo")
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "different values for same prefix",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "foo")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "bar")
				return tbl
			},
			wantEqual: false,
		},
		{
			name: "different entries",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "foo")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("2001:db8::/32"), "foo")
				return tbl
			},
			wantEqual: false,
		},
		{
			name: "same entries, different insert order",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "foo")
				tbl.Insert(mpp("198.51.100.0/24"), "bar")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
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

func TestFastFullEqual(t *testing.T) {
	t.Parallel()
	at := new(Fast[int])
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
