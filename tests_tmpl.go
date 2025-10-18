//go:build generate

// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

//go:generate ./scripts/generate-table-tests.sh

package bart

// ### GENERATE DELETE START ###

// stub code for generator types and methods
// useful for gopls during development, deleted during go generate

import (
	"io"
	"iter"
	"math/rand/v2"
	"net/netip"
	"testing"
)

type (
	_NODE_TYPE[V any]  struct{}
	_TABLE_TYPE[V any] struct{}
)

func (*_TABLE_TYPE[V]) rootNodeByVersion(bool) (_ *_NODE_TYPE[V])                  { return }
func (*_TABLE_TYPE[V]) sizeUpdate(bool, int)                                       { return }
func (*_TABLE_TYPE[V]) dump(io.Writer)                                             { return }
func (*_TABLE_TYPE[V]) dumpString() (_ string)                                     { return }
func (*_TABLE_TYPE[V]) fprint(io.Writer, bool) (_ error)                           { return }
func (*_TABLE_TYPE[V]) Fprint(io.Writer) (_ error)                                 { return }
func (*_TABLE_TYPE[V]) Size() (_ int)                                              { return }
func (*_TABLE_TYPE[V]) Size4() (_ int)                                             { return }
func (*_TABLE_TYPE[V]) Size6() (_ int)                                             { return }
func (*_TABLE_TYPE[V]) Insert(netip.Prefix, V)                                     { return }
func (*_TABLE_TYPE[V]) Get(netip.Prefix) (_ V, _ bool)                             { return }
func (*_TABLE_TYPE[V]) Delete(netip.Prefix)                                        { return }
func (*_TABLE_TYPE[V]) Modify(netip.Prefix, func(V, bool) (V, bool))               { return }
func (*_TABLE_TYPE[V]) Clone() (_ *_TABLE_TYPE[V])                                 { return }
func (*_TABLE_TYPE[V]) Union(*_TABLE_TYPE[V])                                      { return }
func (*_TABLE_TYPE[V]) Equal(*_TABLE_TYPE[V]) (_ bool)                             { return }
func (*_TABLE_TYPE[V]) OverlapsPrefix(netip.Prefix) (_ bool)                       { return }
func (*_TABLE_TYPE[V]) Overlaps(*_TABLE_TYPE[V]) (_ bool)                          { return }
func (*_TABLE_TYPE[V]) Overlaps4(*_TABLE_TYPE[V]) (_ bool)                         { return }
func (*_TABLE_TYPE[V]) Overlaps6(*_TABLE_TYPE[V]) (_ bool)                         { return }
func (*_TABLE_TYPE[V]) Contains(netip.Addr) (_ bool)                               { return }
func (*_TABLE_TYPE[V]) Lookup(netip.Addr) (_ V, _ bool)                            { return }
func (*_TABLE_TYPE[V]) LookupPrefix(netip.Prefix) (_ V, _ bool)                    { return }
func (*_TABLE_TYPE[V]) LookupPrefixLPM(netip.Prefix) (_ netip.Prefix, _ V, _ bool) { return }

func (*_TABLE_TYPE[V]) InsertPersist(netip.Prefix, V) (_ *_TABLE_TYPE[V]) { return }
func (*_TABLE_TYPE[V]) DeletePersist(netip.Prefix) (_ *_TABLE_TYPE[V])    { return }
func (*_TABLE_TYPE[V]) UnionPersist(*_TABLE_TYPE[V]) (_ *_TABLE_TYPE[V])  { return }
func (*_TABLE_TYPE[V]) ModifyPersist(netip.Prefix, func(V, bool) (V, bool)) (_ *_TABLE_TYPE[V]) {
	return
}
func (*_TABLE_TYPE[V]) MarshalText() (_ []byte, _ error) { return }
func (*_TABLE_TYPE[V]) MarshalJSON() (_ []byte, _ error) { return }
func (*_TABLE_TYPE[V]) DumpList4() (_ []DumpListNode[V]) { return }
func (*_TABLE_TYPE[V]) DumpList6() (_ []DumpListNode[V]) { return }

func (*_TABLE_TYPE[V]) All() (_ iter.Seq2[netip.Prefix, V])  { return }
func (*_TABLE_TYPE[V]) All4() (_ iter.Seq2[netip.Prefix, V]) { return }
func (*_TABLE_TYPE[V]) All6() (_ iter.Seq2[netip.Prefix, V]) { return }

func (*_TABLE_TYPE[V]) AllSorted() (_ iter.Seq2[netip.Prefix, V])  { return }
func (*_TABLE_TYPE[V]) AllSorted4() (_ iter.Seq2[netip.Prefix, V]) { return }
func (*_TABLE_TYPE[V]) AllSorted6() (_ iter.Seq2[netip.Prefix, V]) { return }

func (*_TABLE_TYPE[V]) Subnets(netip.Prefix) (_ iter.Seq2[netip.Prefix, V])   { return }
func (*_TABLE_TYPE[V]) Supernets(netip.Prefix) (_ iter.Seq2[netip.Prefix, V]) { return }

// ### GENERATE DELETE END ###

// ############ tests ################################

func TestTableNil__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	ip4 := mpa("127.0.0.1")
	ip6 := mpa("::1")

	pfx4 := mpp("127.0.0.0/8")
	pfx6 := mpp("::1/128")

	tbl2 := new(_TABLE_TYPE[any])
	tbl2.Insert(pfx4, nil)
	tbl2.Insert(pfx6, nil)

	var tbl1 *_TABLE_TYPE[any] = nil

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

		mustPanic(t, "Equal", func() { tbl1.Equal(tbl2) })
		noPanic(t, "Equal", func() { tbl1.Equal(tbl1) })
		noPanic(t, "Equal", func() { tbl2.Equal(tbl2) })

		noPanic(t, "dump", func() { tbl1.dump(nil) })
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

func TestTableInvalid__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	tbl1 := new(_TABLE_TYPE[any])
	tbl2 := new(_TABLE_TYPE[any])

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

func TestTableContainsCompare__TABLE_TYPE(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[int])
	tbl := new(_TABLE_TYPE[int])

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

func TestTableLookupCompare__TABLE_TYPE(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[int])
	tbl := new(_TABLE_TYPE[int])

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

		// liteTable has no real payload
		if _, ok := any(tbl).(*liteTable[int]); !ok {
			if goldVal != tblVal {
				t.Fatalf("Lookup(%q) = (%v, %v), want (%v, %v)", a, tblVal, tblOK, goldVal, goldOK)
			}
		}
	}
}

func TestTableLookupPrefixUnmasked__TABLE_TYPE(t *testing.T) {
	// test that the pfx must not be masked on input for LookupPrefix
	t.Parallel()

	tbl := new(_TABLE_TYPE[any])
	tbl.Insert(mpp("10.20.30.0/24"), nil)

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

func TestTableLookupPrefixCompare__TABLE_TYPE(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[int])
	tbl := new(_TABLE_TYPE[int])
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

		// liteTable has no real payload
		if _, ok := any(tbl).(*liteTable[int]); !ok {
			if goldVal != tblVal {
				t.Fatalf("LookupPrefix(%q) = (%v, %v), want (%v, %v)", pfx, tblVal, tblOK, goldVal, goldOK)
			}
		}
	}
}

func TestTableLookupPrefixLPMCompare__TABLE_TYPE(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	gold := new(goldTable[int])
	tbl := new(_TABLE_TYPE[int])
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

		// liteTable has no real payload
		if _, ok := any(tbl).(*liteTable[int]); !ok {
			if goldVal != tblVal {
				t.Fatalf("LookupPrefixLPM(%q) = (_, %v, _), want (_, %v, _)", pfx, tblVal, goldVal)
			}
		}
	}
}
