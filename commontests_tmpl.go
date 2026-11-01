//go:build generate

// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

//go:generate ./scripts/generate-table-tests.sh

package bart

// ### GENERATE DELETE START ###

// stub code for generator types and methods
// useful for gopls during development, deleted during go generate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"maps"
	"math/rand/v2"
	"net/netip"
	"slices"
	"strings"
	"testing"

	"github.com/gaissmai/bart/internal/nodes"
	"github.com/gaissmai/bart/internal/tests/golden"
	"github.com/gaissmai/bart/internal/tests/random"
)

type (
	_NODE_TYPE[T any] struct{}

	_TABLE_TYPE[T any] struct {
		root4 _NODE_TYPE[T]
		root6 _NODE_TYPE[T]
	}
)

func (*_NODE_TYPE[T]) StatsRec() (_ nodes.StatsT) { return }

func (*_TABLE_TYPE[T]) rootNodeByVersion(bool) (_ *_NODE_TYPE[T])                  { return }
func (*_TABLE_TYPE[T]) sizeUpdate(bool, int)                                       { return }
func (*_TABLE_TYPE[T]) dump(io.Writer)                                             { return }
func (*_TABLE_TYPE[T]) dumpString() (_ string)                                     { return }
func (*_TABLE_TYPE[T]) fprint(io.Writer, bool) (_ error)                           { return }
func (*_TABLE_TYPE[T]) Fprint(io.Writer) (_ error)                                 { return }
func (*_TABLE_TYPE[T]) Size() (_ int)                                              { return }
func (*_TABLE_TYPE[T]) Size4() (_ int)                                             { return }
func (*_TABLE_TYPE[T]) Size6() (_ int)                                             { return }
func (*_TABLE_TYPE[T]) Insert(netip.Prefix, T)                                     { return }
func (*_TABLE_TYPE[T]) Get(netip.Prefix) (_ T, _ bool)                             { return }
func (*_TABLE_TYPE[T]) Delete(netip.Prefix)                                        { return }
func (*_TABLE_TYPE[T]) Modify(netip.Prefix, func(T, bool) (T, bool))               { return }
func (*_TABLE_TYPE[T]) Clone() (_ *_TABLE_TYPE[T])                                 { return }
func (*_TABLE_TYPE[T]) Union(*_TABLE_TYPE[T])                                      { return }
func (*_TABLE_TYPE[T]) Equal(*_TABLE_TYPE[T]) (_ bool)                             { return }
func (*_TABLE_TYPE[T]) OverlapsPrefix(netip.Prefix) (_ bool)                       { return }
func (*_TABLE_TYPE[T]) Overlaps(*_TABLE_TYPE[T]) (_ bool)                          { return }
func (*_TABLE_TYPE[T]) Overlaps4(*_TABLE_TYPE[T]) (_ bool)                         { return }
func (*_TABLE_TYPE[T]) Overlaps6(*_TABLE_TYPE[T]) (_ bool)                         { return }
func (*_TABLE_TYPE[T]) Contains(netip.Addr) (_ bool)                               { return }
func (*_TABLE_TYPE[T]) Lookup(netip.Addr) (_ T, _ bool)                            { return }
func (*_TABLE_TYPE[T]) LookupPrefix(netip.Prefix) (_ T, _ bool)                    { return }
func (*_TABLE_TYPE[T]) LookupPrefixLPM(netip.Prefix) (_ netip.Prefix, _ T, _ bool) { return }

func (*_TABLE_TYPE[T]) InsertPersist(netip.Prefix, T) (_ *_TABLE_TYPE[T]) { return }
func (*_TABLE_TYPE[T]) DeletePersist(netip.Prefix) (_ *_TABLE_TYPE[T])    { return }
func (*_TABLE_TYPE[T]) UnionPersist(*_TABLE_TYPE[T]) (_ *_TABLE_TYPE[T])  { return }
func (*_TABLE_TYPE[T]) ModifyPersist(netip.Prefix, func(T, bool) (T, bool)) (_ *_TABLE_TYPE[T]) {
	return
}
func (*_TABLE_TYPE[T]) MarshalText() (_ []byte, _ error) { return }
func (*_TABLE_TYPE[T]) MarshalJSON() (_ []byte, _ error) { return }
func (*_TABLE_TYPE[T]) DumpList4() (_ []DumpListNode[T]) { return }
func (*_TABLE_TYPE[T]) DumpList6() (_ []DumpListNode[T]) { return }

func (*_TABLE_TYPE[T]) All() (_ iter.Seq2[netip.Prefix, T])  { return }
func (*_TABLE_TYPE[T]) All4() (_ iter.Seq2[netip.Prefix, T]) { return }
func (*_TABLE_TYPE[T]) All6() (_ iter.Seq2[netip.Prefix, T]) { return }

func (*_TABLE_TYPE[T]) AllSorted() (_ iter.Seq2[netip.Prefix, T])  { return }
func (*_TABLE_TYPE[T]) AllSorted4() (_ iter.Seq2[netip.Prefix, T]) { return }
func (*_TABLE_TYPE[T]) AllSorted6() (_ iter.Seq2[netip.Prefix, T]) { return }

func (*_TABLE_TYPE[T]) Subnets(netip.Prefix) (_ iter.Seq2[netip.Prefix, T])   { return }
func (*_TABLE_TYPE[T]) Supernets(netip.Prefix) (_ iter.Seq2[netip.Prefix, T]) { return }

// ### GENERATE DELETE END ###

// ############ tests ################################

// flatSorted, just a helper to compare with golden table.
func (t *_TABLE_TYPE[T]) flatSorted() golden.TableSlice[T] {
	flat := golden.TableSlice[T](make([]golden.Item[T], 0, t.Size()))

	for p, v := range t.AllSorted() {
		flat = append(flat, golden.Item[T]{Pfx: p, Val: v})
	}

	return flat
}

func TestTableNilReceiver__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	var tbl1 *_TABLE_TYPE[any] = nil

	ip4 := mpa("127.0.0.1")
	ip6 := mpa("::1")

	pfx4 := mpp("127.0.0.0/8")
	pfx6 := mpp("::1/128")

	tbl2 := new(_TABLE_TYPE[any])
	tbl2.Insert(pfx4, true)
	tbl2.Insert(pfx6, true)

	t.Run("mustPanicNilReceiver", func(t *testing.T) {
		t.Parallel()

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
		mustPanic(t, "Clone", func() { tbl1.Clone() })
		mustPanic(t, "Union", func() { tbl1.Union(tbl2) })
		mustPanic(t, "UnionPersist", func() { tbl1.UnionPersist(tbl2) })
		mustPanic(t, "Overlaps", func() { tbl1.Overlaps(tbl2) })
		mustPanic(t, "Overlaps4", func() { tbl1.Overlaps4(tbl2) })
		mustPanic(t, "Overlaps6", func() { tbl1.Overlaps6(tbl2) })
		mustPanic(t, "OverlapsPrefix", func() { tbl1.OverlapsPrefix(pfx4) })
		mustPanic(t, "OverlapsPrefix", func() { tbl1.OverlapsPrefix(pfx6) })
		mustPanic(t, "Equal", func() { tbl1.Equal(tbl2) })
		mustPanic(t, "DumpList4", func() { tbl1.DumpList4() })
		mustPanic(t, "DumpList6", func() { tbl1.DumpList6() })
		mustPanic(t, "Fprint", func() { tbl1.Fprint(nil) })
		mustPanic(t, "MarshalJSON", func() { _, _ = tbl1.MarshalJSON() })
		mustPanic(t, "MarshalText", func() { _, _ = tbl1.MarshalText() })

		mustPanicRangeOverFunc[any](t, "All", tbl1.All)
		mustPanicRangeOverFunc[any](t, "All4", tbl1.All4)
		mustPanicRangeOverFunc[any](t, "All6", tbl1.All6)
		mustPanicRangeOverFunc[any](t, "AllSorted", tbl1.AllSorted)
		mustPanicRangeOverFunc[any](t, "AllSorted4", tbl1.AllSorted4)
		mustPanicRangeOverFunc[any](t, "AllSorted6", tbl1.AllSorted6)
		mustPanicRangeOverFunc[any](t, "Subnets", tbl1.Subnets)
		mustPanicRangeOverFunc[any](t, "Supernets", tbl1.Supernets)
	})
}

func TestTableNilArguments__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	t.Run("mustPanicNilArgs", func(t *testing.T) {
		t.Parallel()
		tbl := new(_TABLE_TYPE[bool])

		tbl.Insert(mpp("127.0.0.1/32"), true)
		tbl.Insert(mpp("::1/128"), true)

		mustPanic(t, "Union(nil)", func() { tbl.Union(nil) })
		mustPanic(t, "UnionPersist(nil)", func() { tbl.UnionPersist(nil) })
		mustPanic(t, "Overlaps(nil)", func() { tbl.Overlaps(nil) })
		mustPanic(t, "Overlaps4(nil)", func() { tbl.Overlaps4(nil) })
		mustPanic(t, "Overlaps6(nil)", func() { tbl.Overlaps6(nil) })
		mustPanic(t, "Equal(nil)", func() { tbl.Equal(nil) })
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
	pfxs := random.RealWorldPrefixes(prng, n)

	gold := new(golden.Table[int])
	tbl := new(_TABLE_TYPE[int])

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

func TestTableLookupCompare__TABLE_TYPE(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := random.RealWorldPrefixes(prng, n)

	gold := new(golden.Table[int])
	tbl := new(_TABLE_TYPE[int])

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

func TestZonedLookup__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	check := func(t *testing.T, table *_TABLE_TYPE[string], ip netip.Addr, want string, wantOK bool) {
		t.Helper()

		for _, probe := range []struct {
			name string
			ip   netip.Addr
		}{
			{name: "plain", ip: ip},
			{name: "zoned", ip: ip.WithZone("eth0")},
		} {

			t.Run(probe.name, func(t *testing.T) {

				if got := table.Contains(probe.ip); got != wantOK {
					t.Fatalf("Contains(%q) = %v, want %v", probe.ip, got, wantOK)
				}

				// Skip value comparison for liteTable (no real payload)
				if _, isLite := any(table).(*liteTable[string]); !isLite {
					got, ok := table.Lookup(probe.ip)
					if got != want || ok != wantOK {
						t.Fatalf("Lookup(%q) = (%q, %v), want (%q, %v)", probe.ip, got, ok, want, wantOK)
					}
				} else {
					_, ok := table.Lookup(probe.ip)
					if ok != wantOK {
						t.Fatalf("Lookup(%q) = (_, %v), want (_, %v)", probe.ip, ok, wantOK)
					}
				}
			})
		}
	}

	type route struct {
		cidr  string
		value string
	}

	tests := []struct {
		name   string
		routes []route
		probe  string
		want   string
		wantOK bool
	}{
		{
			name: "compressed leaf hit",
			routes: []route{
				{cidr: "2001:db8:1:2::/65", value: "specific"},
			},
			probe:  "2001:db8:1:2::1",
			want:   "specific",
			wantOK: true,
		},
		{
			name: "leaf miss",
			routes: []route{
				{cidr: "2001:db8:1:2::/65", value: "specific"},
			},
			probe:  "2001:db8:1:2:8000::1",
			wantOK: false,
		},
		{
			name: "leaf miss falls back to less specific route",
			routes: []route{
				{cidr: "2001:db8:1::/48", value: "fallback"},
				{cidr: "2001:db8:1:2::/65", value: "specific"},
			},
			probe:  "2001:db8:1:2:8000::1",
			want:   "fallback",
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			table := new(_TABLE_TYPE[string])
			for _, route := range tt.routes {
				table.Insert(mpp(route.cidr), route.value)
			}

			check(t, table, mpa(tt.probe), tt.want, tt.wantOK)
		})
	}
}

func TestTableLookupPrefixUnmasked__TABLE_TYPE(t *testing.T) {
	// test that the pfx must not be masked on input for LookupPrefix
	t.Parallel()

	tbl := new(_TABLE_TYPE[any])
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

func TestTableLookupPrefixCompare__TABLE_TYPE(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := random.RealWorldPrefixes(prng, n)

	gold := new(golden.Table[int])
	tbl := new(_TABLE_TYPE[int])
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

func TestTableLookupPrefixLPMCompare__TABLE_TYPE(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := random.RealWorldPrefixes(prng, n)

	gold := new(golden.Table[int])
	tbl := new(_TABLE_TYPE[int])
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

func TestTableInsertShuffled__TABLE_TYPE(t *testing.T) {
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

		tbl1 := new(_TABLE_TYPE[string])
		tbl2 := new(_TABLE_TYPE[string])

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

func TestTableInsertPersistShuffled__TABLE_TYPE(t *testing.T) {
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

		tbl1 := new(_TABLE_TYPE[string])
		tbl2 := new(_TABLE_TYPE[string])

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

func TestTableDeleteCompare__TABLE_TYPE(t *testing.T) {
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
		tbl := new(_TABLE_TYPE[string])

		for _, pfx := range pfxs {
			gold.Insert(pfx, pfx.String())
			tbl.Insert(pfx, pfx.String())
		}

		for _, pfx := range toDelete {
			gold.Delete(pfx)
			tbl.Delete(pfx)
		}

		goldFlat := gold.FlatSorted()
		tblFlat := tbl.flatSorted()

		// Skip value comparison for liteTable (no real payload)
		if _, isLite := any(tbl).(*liteTable[string]); isLite {
			if !slices.Equal(goldFlat.SortKeys(), tblFlat.SortKeys()) {
				t.Fatal("expected Equal")
			}
		} else {
			if !slices.Equal(goldFlat, tblFlat) {
				t.Fatal("expected Equal")
			}
		}
	}
}

func TestTableDeleteShuffled__TABLE_TYPE(t *testing.T) {
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

		tbl := new(_TABLE_TYPE[string])

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

		tbl2 := new(_TABLE_TYPE[string])

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

func TestTableDeleteIsReverseOfInsert__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	// Insert n prefixes, then delete those same prefixes in reverse
	// order. Each deletion should exactly undo the internal structure
	// changes that each insert did.

	tbl := new(_TABLE_TYPE[string])
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

func TestTableDeleteButOne__TABLE_TYPE(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))
	// Insert n prefixes, then delete all but one
	n := workLoadN()

	for range 10 {

		tbl := new(_TABLE_TYPE[any])
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

func TestTableGet__TABLE_TYPE(t *testing.T) {
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

	tbl := new(_TABLE_TYPE[int])
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

func TestTableGetCompare__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := random.RealWorldPrefixes(prng, n)

	gold := new(golden.Table[string])
	tbl := new(_TABLE_TYPE[string])
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

func TestTableModifySemantics__TABLE_TYPE(t *testing.T) {
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
		tbl := new(_TABLE_TYPE[int])

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

func TestTableModifyPersistSemantics__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	if _, isLite := any(new(_TABLE_TYPE[any])).(*liteTable[any]); isLite {
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
		tbl := new(_TABLE_TYPE[int])

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

func TestTableModifyCompare__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := random.RealWorldPrefixes(prng, n)

	gold := new(golden.Table[int])
	tbl := new(_TABLE_TYPE[int])

	// Update as insert
	for i, pfx := range pfxs {
		gold.Insert(pfx, i)
		tbl.Modify(pfx, func(int, bool) (int, bool) { return i, false })
	}

	goldFlat := gold.FlatSorted()
	tblFlat := tbl.flatSorted()

	// Skip value comparison for liteTable (no real payload)
	if _, isLite := any(tbl).(*liteTable[int]); isLite {
		if !slices.Equal(goldFlat.SortKeys(), tblFlat.SortKeys()) {
			t.Fatal("expected Equal")
		}
	} else {
		if !slices.Equal(goldFlat, tblFlat) {
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

	goldFlat = gold.FlatSorted()
	tblFlat = tbl.flatSorted()

	// Skip value comparison for liteTable (no real payload)
	if _, isLite := any(tbl).(*liteTable[int]); isLite {
		if !slices.Equal(goldFlat.SortKeys(), tblFlat.SortKeys()) {
			t.Fatal("expected Equal")
		}
	} else {
		if !slices.Equal(goldFlat, tblFlat) {
			t.Fatal("expected Equal")
		}
	}
}

func TestTableModifyPersistCompare__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := random.RealWorldPrefixes(prng, n)

	mut := new(_TABLE_TYPE[int])
	imu := new(_TABLE_TYPE[int])

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

func TestTableModifyShuffled__TABLE_TYPE(t *testing.T) {
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

		tbl1 := new(_TABLE_TYPE[string])

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

		tbl2 := new(_TABLE_TYPE[string])

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

func TestTableModifyPersistShuffled__TABLE_TYPE(t *testing.T) {
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

		tbl1 := new(_TABLE_TYPE[string])

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

		tbl2 := new(_TABLE_TYPE[string])

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
func TestTableUnionMemoryAliasing__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	newTable := func(pfx ...string) *_TABLE_TYPE[any] {
		tbl := new(_TABLE_TYPE[any])
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
func TestTableUnionPersistMemoryAliasing__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	newTable := func(pfx ...string) *_TABLE_TYPE[any] {
		tbl := new(_TABLE_TYPE[any])
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

func TestTableUnionCompare__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	n := workLoadN()
	prng := rand.New(rand.NewPCG(42, 42))

	for range 3 {
		pfxs := random.RealWorldPrefixes(prng, n)

		gold := new(golden.Table[string])
		tbl := new(_TABLE_TYPE[string])

		for _, pfx := range pfxs {
			gold.Insert(pfx, pfx.String())
			tbl.Insert(pfx, pfx.String())
		}

		pfxs2 := random.RealWorldPrefixes(prng, n)

		gold2 := new(golden.Table[string])
		tbl2 := new(_TABLE_TYPE[string])

		for _, pfx := range pfxs2 {
			gold2.Insert(pfx, pfx.String())
			tbl2.Insert(pfx, pfx.String())
		}

		gold.Union(*gold2)
		tbl.Union(tbl2)

		goldFlat := gold.FlatSorted()
		tblFlat := tbl.flatSorted()

		// Skip value comparison for liteTable (no real payload)
		if _, isLite := any(tbl).(*liteTable[string]); isLite {
			if !slices.Equal(goldFlat.SortKeys(), tblFlat.SortKeys()) {
				t.Fatal("expected Equal")
			}
		} else {
			if !slices.Equal(goldFlat, tblFlat) {
				t.Fatal("expected Equal")
			}
		}
	}
}

func TestTableUnionPersistCompare__TABLE_TYPE(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))

	n := workLoadN()

	for range 3 {
		pfxs := random.RealWorldPrefixes(prng, n)

		gold := new(golden.Table[int])
		tbl := new(_TABLE_TYPE[int])

		for i, pfx := range pfxs {
			gold.Insert(pfx, i)
			tbl.Insert(pfx, i)
		}

		pfxs2 := random.RealWorldPrefixes(prng, n)

		gold2 := new(golden.Table[int])
		tbl2 := new(_TABLE_TYPE[int])

		for i, pfx := range pfxs2 {
			gold2.Insert(pfx, i)
			tbl2.Insert(pfx, i)
		}

		gold.Union(*gold2)
		tblP := tbl.UnionPersist(tbl2)

		goldFlat := gold.FlatSorted()
		tblFlat := tblP.flatSorted()

		// Skip value comparison for liteTable (no real payload)
		if _, isLite := any(tbl).(*liteTable[int]); isLite {
			if !slices.Equal(goldFlat.SortKeys(), tblFlat.SortKeys()) {
				t.Fatal("expected Equal")
			}
		} else {
			if !slices.Equal(goldFlat, tblFlat) {
				t.Fatal("expected Equal")
			}
		}
	}
}

func TestTableClone__TABLE_TYPE(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := random.RealWorldPrefixes(prng, n)

	tbl := new(_TABLE_TYPE[int])
	for i, pfx := range pfxs {
		tbl.Insert(pfx, i)
	}
	clone := tbl.Clone()

	if !tbl.Equal(clone) {
		t.Fatal("expected equal")
	}
}

func TestTableCloneShallow__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	tbl := new(_TABLE_TYPE[*int])

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

func TestTableCloneDeep__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	tbl := new(_TABLE_TYPE[*MyInt])

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

func TestTableUnionShallow__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	tbl1 := new(_TABLE_TYPE[*int])
	tbl2 := new(_TABLE_TYPE[*int])

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

func TestTableUnionDeep__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	tbl1 := new(_TABLE_TYPE[*MyInt])
	tbl2 := new(_TABLE_TYPE[*MyInt])

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
func TestTableOverlapsPrefixEdgeCases__TABLE_TYPE(t *testing.T) {
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
		tbl := new(_TABLE_TYPE[int])
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

func TestTableSize__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	n := workLoadN()
	prng := rand.New(rand.NewPCG(42, 42))

	tbl := new(_TABLE_TYPE[any])
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
func TestTableAll__TABLE_TYPE(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := random.RealWorldPrefixes(prng, n)

	tbl := new(_TABLE_TYPE[string])

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

func TestTableAllSorted__TABLE_TYPE(t *testing.T) {
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

			tbl := new(_TABLE_TYPE[int])

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

func TestTableSubnets__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	var zeroPfx netip.Prefix

	t.Run("empty table", func(t *testing.T) {
		t.Parallel()
		tbl := new(_TABLE_TYPE[string])
		pfx := mpp("::1/128")

		for range tbl.Subnets(pfx) {
			t.Errorf("empty table, must not range over")
		}
	})

	t.Run("invalid prefix", func(t *testing.T) {
		tbl := new(_TABLE_TYPE[string])
		pfx := mpp("::1/128")
		val := "foo"
		tbl.Insert(pfx, val)
		for range tbl.Subnets(zeroPfx) {
			t.Errorf("invalid prefix, must not range over")
		}
	})

	t.Run("identity", func(t *testing.T) {
		tbl := new(_TABLE_TYPE[string])
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

		tbl := new(_TABLE_TYPE[int])
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

func TestTableSubnetsCompare__TABLE_TYPE(t *testing.T) {
	t.Parallel()
	n := workLoadN()
	prng := rand.New(rand.NewPCG(42, 42))

	pfxs := random.RealWorldPrefixes(prng, n)

	gold := new(golden.Table[int])
	tbl := new(_TABLE_TYPE[int])

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

func TestTableSupernetsEdgeCase__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	var zeroPfx netip.Prefix

	t.Run("empty table", func(t *testing.T) {
		t.Parallel()
		tbl := new(_TABLE_TYPE[any])
		pfx := mpp("::1/128")

		tbl.Supernets(pfx)(func(_ netip.Prefix, _ any) bool {
			t.Errorf("empty table, must not range over")
			return false
		})
	})

	t.Run("invalid prefix", func(t *testing.T) {
		tbl := new(_TABLE_TYPE[any])
		pfx := mpp("::1/128")
		val := "foo"
		tbl.Insert(pfx, val)

		tbl.Supernets(zeroPfx)(func(_ netip.Prefix, _ any) bool {
			t.Errorf("invalid prefix, must not range over")
			return false
		})
	})

	t.Run("identity", func(t *testing.T) {
		tbl := new(_TABLE_TYPE[string])
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

func TestTableSupernetsCompare__TABLE_TYPE(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))

	pfxs := random.RealWorldPrefixes(prng, n)

	gold := new(golden.Table[int])
	tbl := new(_TABLE_TYPE[int])

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

func TestTableMarshalText__TABLE_TYPE(t *testing.T) {
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
			tbl := new(_TABLE_TYPE[string])

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

func TestTableMarshalJSON__TABLE_TYPE(t *testing.T) {
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
			tbl := new(_TABLE_TYPE[any])

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

func TestTableDumpList4__TABLE_TYPE(t *testing.T) {
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
			tbl := new(_TABLE_TYPE[string])

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

func TestTableDumpList6__TABLE_TYPE(t *testing.T) {
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
			tbl := new(_TABLE_TYPE[string])

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

func TestTableEqual__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	n := workLoadN()
	prng := rand.New(rand.NewPCG(42, 42))

	at := new(_TABLE_TYPE[int])
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

func TestTableExtra__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	// 1. Equal: checking self equality
	t1 := new(_TABLE_TYPE[int])
	t1.Insert(mpp("10.0.0.0/8"), 42)
	t1.Insert(mpp("2001:db8::/32"), 42)

	if !t1.Equal(t1) {
		t.Error("expected table to be equal to itself")
	}

	// 2. Equal: checking different size4 / size6
	t2 := new(_TABLE_TYPE[int])
	t2.Insert(mpp("10.0.0.0/8"), 42)
	if t1.Equal(t2) {
		t.Error("expected false for different size6")
	}

	t3 := new(_TABLE_TYPE[int])
	t3.Insert(mpp("2001:db8::/32"), 42)
	if t1.Equal(t3) {
		t.Error("expected false for different size4")
	}

	// 3. Overlaps6: checking when size6 is 0
	emptyT := new(_TABLE_TYPE[int])
	if t1.Overlaps6(emptyT) {
		t.Error("expected false when other table has no IPv6 prefixes")
	}
	if emptyT.Overlaps6(t1) {
		t.Error("expected false when this table has no IPv6 prefixes")
	}

	// 4. Fprint: nil writer check
	var nilWriter io.Writer = nil
	if err := t1.Fprint(nilWriter); err == nil {
		t.Error("expected error when writing to nil writer")
	}

	// 5. ModifyPersist: no-op deletion of non-existing key
	t4 := new(_TABLE_TYPE[int])
	t5 := t4.ModifyPersist(mpp("10.0.0.0/8"), func(val int, ok bool) (int, bool) {
		return 0, true
	})
	if t5 != t4 {
		t.Error("expected same table pointer for no-op ModifyPersist")
	}
}

// ############ model tests ##########################
//
// The model tests compare the table with the golden table and check three
// properties that a comparison of the content alone does not see:
//
//   - the structure is canonical: after any sequence of operations the table
//     has the same structure as a table that was freshly built from the same
//     prefixes. A wrongly compressed trie still returns the right lookup
//     results, but wastes memory and breaks Equal.
//   - the ...Persist methods leave the original table unchanged: structure,
//     values and sizes.
//   - the size counters are correct, also after a Union with duplicates.
//
// The random tests use prefixes made of a small pool of byte values, so that
// they collide, overlap and nest, see random.NestedPrefixes. The helpers and
// the test data are in zz-model_test.go.

// dumpStructure is dumpString for the comparison of structures.
//
// The leaves and fringes of a liteTable have no payload that matters, they are
// shared on purpose between a node and its clone, see LiteNode.CloneFlat. This
// is invisible for the struct{} payload of the public Lite table, but the dump
// shows the values if the liteTable is instantiated with int, as the tests do.
// Therefore the payload of the leaves and fringes is removed from the dump of
// a liteTable.
func (t *_TABLE_TYPE[T]) dumpStructure() string {
	s := t.dumpString()

	if _, isLite := any(t).(*liteTable[T]); isLite {
		return modelStripPayload(s)
	}

	return s
}

// modelBuild__TABLE_TYPE builds a table, the prefixes are inserted in the given
// order with the values base, base+1, ...
func modelBuild__TABLE_TYPE(pfxs []netip.Prefix, base int) *_TABLE_TYPE[int] {
	tbl := new(_TABLE_TYPE[int])

	for i, pfx := range pfxs {
		tbl.Insert(pfx, base+i)
	}

	return tbl
}

// modelFresh__TABLE_TYPE builds a table from the golden table, the prefixes are
// inserted in random order.
func modelFresh__TABLE_TYPE(gold golden.Table[int], prng *rand.Rand) *_TABLE_TYPE[int] {
	pfxs := gold.FlatSorted().SortKeys()
	prng.Shuffle(len(pfxs), func(i, j int) { pfxs[i], pfxs[j] = pfxs[j], pfxs[i] })

	tbl := new(_TABLE_TYPE[int])

	for _, pfx := range pfxs {
		tbl.Insert(pfx, gold[pfx])
	}

	return tbl
}

// modelCheck__TABLE_TYPE verifies the table against the golden table: the size,
// the content in canonical order, the values (not for a liteTable, it has no
// payload) and the canonical structure: the table must have the same structure
// as a table that was freshly built from the golden table.
func modelCheck__TABLE_TYPE(tb testing.TB, what string, tbl *_TABLE_TYPE[int], gold golden.Table[int], isLite bool, prng *rand.Rand) {
	tb.Helper()

	if got, want := tbl.Size(), len(gold); got != want {
		tb.Fatalf("%s: Size() = %d, want %d", what, got, want)
	}

	goldFlat, tblFlat := gold.FlatSorted(), tbl.flatSorted()

	if isLite {
		if !slices.Equal(goldFlat.SortKeys(), tblFlat.SortKeys()) {
			tb.Fatalf("%s: the prefixes differ from the golden table", what)
		}
	} else if !slices.Equal(goldFlat, tblFlat) {
		tb.Fatalf("%s: the content differs from the golden table", what)
	}

	got, want := tbl.dumpStructure(), modelFresh__TABLE_TYPE(gold, prng).dumpStructure()
	if got != want {
		tb.Fatalf("%s: the structure differs from a freshly built table, it is not canonical:\n%s\nwant:\n%s", what, got, want)
	}
}

// modelUnionCheck__TABLE_TYPE verifies Union and UnionPersist for the tables
// made of pfxsA and pfxsB: content, structure and size of the result, the other
// table is unchanged, and for UnionPersist the receiver is unchanged too.
func modelUnionCheck__TABLE_TYPE(t *testing.T, what string, pfxsA, pfxsB []netip.Prefix, isLite bool, prng *rand.Rand) {
	t.Helper()

	// the values of the second table differ from those of the first,
	// the second wins at a duplicate
	gold := modelUnion(modelGold(pfxsA, 100), modelGold(pfxsB, 200))

	// in place
	a, b := modelBuild__TABLE_TYPE(pfxsA, 100), modelBuild__TABLE_TYPE(pfxsB, 200)
	bSnap := b.dumpStructure()

	a.Union(b)
	modelCheck__TABLE_TYPE(t, what+": Union", a, gold, isLite, prng)

	if b.dumpStructure() != bSnap {
		t.Fatalf("%s: Union modified the other table", what)
	}

	// persistent
	a, b = modelBuild__TABLE_TYPE(pfxsA, 100), modelBuild__TABLE_TYPE(pfxsB, 200)
	aSnap, bSnap := a.dumpStructure(), b.dumpStructure()

	p := a.UnionPersist(b)
	modelCheck__TABLE_TYPE(t, what+": UnionPersist", p, gold, isLite, prng)

	if a.dumpStructure() != aSnap {
		t.Fatalf("%s: UnionPersist modified the receiver (copy-on-write violated)", what)
	}

	if b.dumpStructure() != bSnap {
		t.Fatalf("%s: UnionPersist modified the other table", what)
	}
}

// TestTableUnionCombinations tests the 12 combinations of the children of a
// node that Union has to handle, for IPv4 and IPv6, with and without duplicates.
func TestTableUnionCombinations__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	_, isLite := any(new(_TABLE_TYPE[int])).(*liteTable[int])

	for _, tt := range modelUnionCases {
		for _, is4 := range []bool{true, false} {
			t.Run(modelName(tt.name, is4), func(t *testing.T) {
				t.Parallel()

				pfxsA, pfxsB := modelPrefixes(is4, tt.a), modelPrefixes(is4, tt.b)

				// the number of duplicates in the test table must be right
				goldA, goldB := modelGold(pfxsA, 100), modelGold(pfxsB, 200)
				if dups := len(goldA) + len(goldB) - len(modelUnion(goldA, goldB)); dups != tt.dups {
					t.Fatalf("error in the test table: %d duplicates, want %d", dups, tt.dups)
				}

				prng := rand.New(rand.NewPCG(1, 2))
				modelUnionCheck__TABLE_TYPE(t, "Union", pfxsA, pfxsB, isLite, prng)
			})
		}
	}
}

// TestTableUnionDense tests Union with random tables that overlap and nest.
func TestTableUnionDense__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	_, isLite := any(new(_TABLE_TYPE[int])).(*liteTable[int])

	for _, cfg := range modelConfigs {
		t.Run(cfg.name, func(t *testing.T) {
			t.Parallel()

			for seed := range modelSeeds() {
				prng := rand.New(rand.NewPCG(uint64(seed), 42))

				pfxsA := random.NestedPrefixes(prng, cfg.is4, cfg.pool, prng.IntN(cfg.maxN+1), true)
				pfxsB := modelRelated(prng, cfg, pfxsA)

				modelUnionCheck__TABLE_TYPE(t, fmt.Sprintf("seed %d", seed), pfxsA, pfxsB, isLite, prng)
			}
		})
	}
}

// TestTablePersistIsolation tests that InsertPersist and DeletePersist return
// the right table and leave the original table unchanged, including the
// leaves and fringes that are shared between the tables.
func TestTablePersistIsolation__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	_, isLite := any(new(_TABLE_TYPE[int])).(*liteTable[int])

	for _, cfg := range modelConfigs {
		t.Run(cfg.name, func(t *testing.T) {
			t.Parallel()

			for seed := range modelSeeds() {
				prng := rand.New(rand.NewPCG(uint64(seed), 43))
				what := fmt.Sprintf("seed %d", seed)

				pfxs := random.NestedPrefixes(prng, cfg.is4, cfg.pool, prng.IntN(cfg.maxN+1), true)
				gold := modelGold(pfxs, 100)

				tbl := modelBuild__TABLE_TYPE(pfxs, 100)
				snap := tbl.dumpStructure()

				for range 5 {
					pfx := random.NestedPrefixes(prng, cfg.is4, cfg.pool, 1, false)[0]
					val := prng.IntN(1_000_000)

					// InsertPersist
					goldIns := maps.Clone(gold)
					goldIns.Insert(pfx, val)

					ins := tbl.InsertPersist(pfx, val)
					modelCheck__TABLE_TYPE(t, fmt.Sprintf("%s: InsertPersist(%s)", what, pfx), ins, goldIns, isLite, prng)

					if tbl.dumpStructure() != snap {
						t.Fatalf("%s: InsertPersist(%s) modified the original table (copy-on-write violated)", what, pfx)
					}

					// DeletePersist of an existing prefix most of the time
					del := pfx
					if len(pfxs) > 0 && prng.IntN(10) < 7 {
						del = pfxs[prng.IntN(len(pfxs))]
					}

					deleted := tbl.DeletePersist(del)

					if _, exists := gold[del]; exists {
						goldDel := maps.Clone(gold)
						goldDel.Delete(del)

						modelCheck__TABLE_TYPE(t, fmt.Sprintf("%s: DeletePersist(%s)", what, del), deleted, goldDel, isLite, prng)
					} else if deleted != tbl {
						t.Fatalf("%s: DeletePersist(%s) of a missing prefix must return the receiver", what, del)
					}

					if tbl.dumpStructure() != snap {
						t.Fatalf("%s: DeletePersist(%s) modified the original table (copy-on-write violated)", what, del)
					}
				}
			}
		})
	}
}

// TestTableMixedOperations applies random sequences of inserts and deletes,
// in place and persistent, and compares the table with the golden table now
// and then, including the canonical structure.
func TestTableMixedOperations__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	_, isLite := any(new(_TABLE_TYPE[int])).(*liteTable[int])

	for _, cfg := range modelConfigs {
		t.Run(cfg.name, func(t *testing.T) {
			t.Parallel()

			for seed := range modelSeeds() {
				prng := rand.New(rand.NewPCG(uint64(seed), 44))

				gold := make(golden.Table[int])
				tbl := new(_TABLE_TYPE[int])

				for op := range 120 {
					pfx := random.NestedPrefixes(prng, cfg.is4, cfg.pool, 1, false)[0]
					val := prng.IntN(1_000_000)

					// half of the deletes hit an existing prefix
					del := pfx
					if len(gold) > 0 && prng.IntN(2) == 0 {
						keys := gold.FlatSorted().SortKeys()
						del = keys[prng.IntN(len(keys))]
					}

					switch prng.IntN(4) {
					case 0:
						tbl.Insert(pfx, val)
						gold.Insert(pfx, val)
					case 1:
						tbl.Delete(del)
						gold.Delete(del)
					case 2:
						// the former table is not used anymore, it does not matter
						// that the persistent table shares nodes with it
						tbl = tbl.InsertPersist(pfx, val)
						gold.Insert(pfx, val)
					default:
						tbl = tbl.DeletePersist(del)
						gold.Delete(del)
					}

					if op%10 == 9 {
						modelCheck__TABLE_TYPE(t, fmt.Sprintf("seed %d, op %d", seed, op), tbl, gold, isLite, prng)
					}
				}
			}
		})
	}
}

// TestTableOverlapsCombinations tests Overlaps and OverlapsPrefix for the
// combinations of node, leaf, fringe and prefix table entries.
func TestTableOverlapsCombinations__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	for _, tt := range modelOverlapCases {
		for _, is4 := range []bool{true, false} {
			t.Run(modelName(tt.name, is4), func(t *testing.T) {
				t.Parallel()

				pfxsA, pfxsB := modelPrefixes(is4, tt.a), modelPrefixes(is4, tt.b)
				goldA, goldB := modelGold(pfxsA, 1), modelGold(pfxsB, 1)

				// the expectation in the test table must agree with the golden table
				if got := goldA.Overlaps(goldB); got != tt.want {
					t.Fatalf("error in the test table: the golden table says %v, want %v", got, tt.want)
				}

				a, b := modelBuild__TABLE_TYPE(pfxsA, 1), modelBuild__TABLE_TYPE(pfxsB, 1)

				if got := a.Overlaps(b); got != tt.want {
					t.Errorf("Overlaps() = %v, want %v", got, tt.want)
				}

				if got := b.Overlaps(a); got != tt.want {
					t.Errorf("Overlaps() with swapped tables = %v, want %v", got, tt.want)
				}

				for _, pfx := range pfxsB {
					if got, want := a.OverlapsPrefix(pfx), goldA.OverlapsPrefix(pfx); got != want {
						t.Errorf("OverlapsPrefix(%s) = %v, want %v", pfx, got, want)
					}
				}

				for _, pfx := range pfxsA {
					if got, want := b.OverlapsPrefix(pfx), goldB.OverlapsPrefix(pfx); got != want {
						t.Errorf("OverlapsPrefix(%s) of the other table = %v, want %v", pfx, got, want)
					}
				}
			})
		}
	}
}

// TestTableOverlapsSinglePrefixPairs tests Overlaps for two tables with ONE
// prefix each: all pairs of the 255 prefixes in the prefix table of the root
// node, then random pairs including the prefixes below the octet 10 (leaves
// and a fringe). It is often only the containment of the prefix of one table in
// the prefix of the other table that makes the overlap. An exhaustive test finds
// a direction that is missing, random tables hit it rarely.
func TestTableOverlapsSinglePrefixPairs__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	inRoot, belowTen := modelStridePrefixes()
	all := slices.Concat(inRoot, belowTen)

	tables := make(map[netip.Prefix]*_TABLE_TYPE[int], len(all))

	for i, pfx := range all {
		tbl := new(_TABLE_TYPE[int])
		tbl.Insert(pfx, i)
		tables[pfx] = tbl
	}

	for _, p := range inRoot {
		for _, q := range inRoot {
			if got, want := tables[p].Overlaps(tables[q]), p.Overlaps(q); got != want {
				t.Fatalf("{%s}.Overlaps({%s}) = %v, want %v", p, q, got, want)
			}
		}
	}

	prng := rand.New(rand.NewPCG(99, 99))

	for range 20_000 {
		p, q := all[prng.IntN(len(all))], all[prng.IntN(len(all))]

		if got, want := tables[p].Overlaps(tables[q]), p.Overlaps(q); got != want {
			t.Fatalf("{%s}.Overlaps({%s}) = %v, want %v", p, q, got, want)
		}
	}
}

// TestTableOverlapsDense tests Overlaps and OverlapsPrefix with small random
// tables that overlap and nest, many of them have only a few prefixes.
func TestTableOverlapsDense__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	for _, is4 := range []bool{true, false} {
		t.Run(modelName("dense", is4), func(t *testing.T) {
			t.Parallel()

			var overlapping, disjoint int

			for seed := range modelSeeds() * 2 {
				prng := rand.New(rand.NewPCG(uint64(seed), 45))

				poolA := modelOverlapPools[prng.IntN(len(modelOverlapPools))]
				poolB := modelOverlapPools[prng.IntN(len(modelOverlapPools))]

				pfxsA := random.NestedPrefixes(prng, is4, poolA, prng.IntN(9), prng.IntN(3) == 0)
				pfxsB := random.NestedPrefixes(prng, is4, poolB, prng.IntN(9), prng.IntN(3) == 0)

				goldA, goldB := modelGold(pfxsA, 1), modelGold(pfxsB, 1)
				a, b := modelBuild__TABLE_TYPE(pfxsA, 1), modelBuild__TABLE_TYPE(pfxsB, 1)

				want := goldA.Overlaps(goldB)
				if want {
					overlapping++
				} else {
					disjoint++
				}

				if got := a.Overlaps(b); got != want {
					t.Fatalf("seed %d: Overlaps() = %v, want %v\na: %v\nb: %v", seed, got, want, pfxsA, pfxsB)
				}

				if got := b.Overlaps(a); got != want {
					t.Fatalf("seed %d: Overlaps() with swapped tables = %v, want %v", seed, got, want)
				}

				for range 8 {
					probe := random.NestedPrefixes(prng, is4, modelOverlapPools[prng.IntN(len(modelOverlapPools))], 1, false)[0]

					if got, want := a.OverlapsPrefix(probe), goldA.OverlapsPrefix(probe); got != want {
						t.Fatalf("seed %d: OverlapsPrefix(%s) = %v, want %v", seed, probe, got, want)
					}
				}
			}

			// the test is only meaningful if both outcomes occur
			if overlapping == 0 || disjoint == 0 {
				t.Errorf("unbalanced random cases: %d overlapping, %d disjoint", overlapping, disjoint)
			}
		})
	}
}

// TestTableSubnetsSupernetsEdgeCases tests Subnets and Supernets with explicit
// expectations: the probe is stored or not, a host route, wider than all stored
// prefixes, disjoint, with and without a default route.
func TestTableSubnetsSupernetsEdgeCases__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	for _, tt := range modelEdgeCases {
		for _, is4 := range []bool{true, false} {
			t.Run(modelName(tt.name, is4), func(t *testing.T) {
				t.Parallel()

				pfxs := modelPrefixes(is4, tt.stored)
				gold := modelGold(pfxs, 1)
				tbl := modelBuild__TABLE_TYPE(pfxs, 1)

				probe := tt.probe.pfx(is4)
				wantSub, wantSuper := modelPrefixes(is4, tt.subnets), modelPrefixes(is4, tt.supernets)

				// the expectations in the test table must agree with the golden table
				if got := gold.Subnets(probe); !slices.Equal(got, wantSub) {
					t.Fatalf("error in the test table: the golden table says subnets %v, want %v", got, wantSub)
				}

				if got := gold.Supernets(probe); !slices.Equal(got, wantSuper) {
					t.Fatalf("error in the test table: the golden table says supernets %v, want %v", got, wantSuper)
				}

				var gotSub, gotSuper []netip.Prefix

				for pfx := range tbl.Subnets(probe) {
					gotSub = append(gotSub, pfx)
				}

				for pfx := range tbl.Supernets(probe) {
					gotSuper = append(gotSuper, pfx)
				}

				if !slices.Equal(gotSub, wantSub) {
					t.Errorf("Subnets(%s) = %v, want %v", probe, gotSub, wantSub)
				}

				if !slices.Equal(gotSuper, wantSuper) {
					t.Errorf("Supernets(%s) = %v, want %v", probe, gotSuper, wantSuper)
				}
			})
		}
	}
}

// TestTableSubnetsSupernetsDense tests Subnets and Supernets with random tables
// that overlap and nest, for stored prefixes, random prefixes and the default
// route as probe. The probes are not necessarily stored in the table.
func TestTableSubnetsSupernetsDense__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	_, isLite := any(new(_TABLE_TYPE[int])).(*liteTable[int])

	for _, cfg := range modelConfigs {
		t.Run(cfg.name, func(t *testing.T) {
			t.Parallel()

			for seed := range modelSeeds() {
				prng := rand.New(rand.NewPCG(uint64(seed), 46))

				pfxs := random.NestedPrefixes(prng, cfg.is4, cfg.pool, prng.IntN(cfg.maxN+1), true)
				gold := modelGold(pfxs, 1)
				tbl := modelBuild__TABLE_TYPE(pfxs, 1)

				modelCheck__TABLE_TYPE(t, fmt.Sprintf("seed %d", seed), tbl, gold, isLite, prng)

				probes := []netip.Prefix{modelPfx(cfg.is4, 0, 0, 0)}
				probes = append(probes, pfxs[:min(15, len(pfxs))]...)
				probes = append(probes, random.NestedPrefixes(prng, cfg.is4, cfg.pool, 15, false)...)

				for _, probe := range probes {
					var gotSub, gotSuper []netip.Prefix

					for pfx := range tbl.Subnets(probe) {
						gotSub = append(gotSub, pfx)
					}

					for pfx := range tbl.Supernets(probe) {
						gotSuper = append(gotSuper, pfx)
					}

					if want := gold.Subnets(probe); !slices.Equal(gotSub, want) {
						t.Fatalf("seed %d: Subnets(%s) = %v, want %v", seed, probe, gotSub, want)
					}

					if want := gold.Supernets(probe); !slices.Equal(gotSuper, want) {
						t.Fatalf("seed %d: Supernets(%s) = %v, want %v", seed, probe, gotSuper, want)
					}
				}
			}
		})
	}
}

// TestTableEqualCloneDense tests Equal and Clone with random tables: the same
// prefixes inserted in different orders are equal and have the same structure,
// a table with one prefix less or one value different is not equal, and a clone
// is equal and independent of the original.
func TestTableEqualCloneDense__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	_, isLite := any(new(_TABLE_TYPE[int])).(*liteTable[int])

	for _, cfg := range modelConfigs {
		t.Run(cfg.name, func(t *testing.T) {
			t.Parallel()

			for seed := range modelSeeds() {
				prng := rand.New(rand.NewPCG(uint64(seed), 47))

				pfxs := random.NestedPrefixes(prng, cfg.is4, cfg.pool, prng.IntN(cfg.maxN+1), true)
				gold := modelGold(pfxs, 100)

				x, y := modelFresh__TABLE_TYPE(gold, prng), modelFresh__TABLE_TYPE(gold, prng)

				if !x.Equal(y) || !y.Equal(x) {
					t.Fatalf("seed %d: tables with the same prefixes are not equal", seed)
				}

				if x.dumpStructure() != y.dumpStructure() {
					t.Fatalf("seed %d: tables with the same prefixes have different structures", seed)
				}

				if len(pfxs) > 0 {
					pfx := pfxs[prng.IntN(len(pfxs))]

					less := maps.Clone(gold)
					less.Delete(pfx)

					if z := modelFresh__TABLE_TYPE(less, prng); x.Equal(z) || z.Equal(x) {
						t.Fatalf("seed %d: tables are equal, but one has the prefix %s less", seed, pfx)
					}

					// a liteTable has no values that can differ
					if !isLite {
						diff := maps.Clone(gold)
						diff.Insert(pfx, gold[pfx]+1)

						if z := modelFresh__TABLE_TYPE(diff, prng); x.Equal(z) || z.Equal(x) {
							t.Fatalf("seed %d: tables are equal, but the value of %s differs", seed, pfx)
						}
					}
				}

				c := x.Clone()
				if !c.Equal(x) {
					t.Fatalf("seed %d: the clone is not equal to the original", seed)
				}

				snap := x.dumpStructure()

				for range 6 {
					c.Insert(random.NestedPrefixes(prng, cfg.is4, cfg.pool, 1, false)[0], 1)
				}

				for _, pfx := range pfxs[:min(4, len(pfxs))] {
					c.Delete(pfx)
				}

				if x.dumpStructure() != snap {
					t.Fatalf("seed %d: modifying the clone changed the original table", seed)
				}
			}
		})
	}
}

// TestTableUnionSelfAndEmpty tests Union and UnionPersist with an empty table
// and with the table itself as argument, nothing changes.
func TestTableUnionSelfAndEmpty__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	pfxs := []netip.Prefix{
		mpp("10.0.0.0/8"), mpp("10.1.0.0/16"), mpp("10.1.2.0/24"),
		mpp("2001:db8::/32"), mpp("2001:db8:1::/48"),
	}

	tests := []struct {
		name string
		do   func(tbl, empty *_TABLE_TYPE[int]) *_TABLE_TYPE[int]
	}{
		{"Union with an empty table", func(tbl, empty *_TABLE_TYPE[int]) *_TABLE_TYPE[int] {
			tbl.Union(empty)
			return tbl
		}},
		{"Union with itself", func(tbl, _ *_TABLE_TYPE[int]) *_TABLE_TYPE[int] {
			tbl.Union(tbl)
			return tbl
		}},
		{"UnionPersist with an empty table", func(tbl, empty *_TABLE_TYPE[int]) *_TABLE_TYPE[int] {
			return tbl.UnionPersist(empty)
		}},
		{"UnionPersist with itself", func(tbl, _ *_TABLE_TYPE[int]) *_TABLE_TYPE[int] {
			return tbl.UnionPersist(tbl)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tbl := modelBuild__TABLE_TYPE(pfxs, 100)
			snap := tbl.dumpStructure()

			got := tt.do(tbl, new(_TABLE_TYPE[int]))

			if tbl.dumpStructure() != snap {
				t.Fatal("the receiver was modified")
			}

			if got.Size() != len(pfxs) || !got.Equal(tbl) {
				t.Fatal("the result is not equal to the receiver")
			}
		})
	}
}

// TestTableEqualDifferences tests that Equal finds the difference of tables with
// the same number of prefixes in every place where the nodes are compared: the
// prefixes of a node, the child slots, and the kind of the children, node, leaf
// and fringe. Tables with different sizes are not compared node by node.
func TestTableEqualDifferences__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	for _, tt := range modelEqualCases {
		for _, is4 := range []bool{true, false} {
			t.Run(modelName(tt.name, is4), func(t *testing.T) {
				t.Parallel()

				pfxsA, pfxsB := modelPrefixes(is4, tt.a), modelPrefixes(is4, tt.b)
				goldA, goldB := modelGold(pfxsA, 1), modelGold(pfxsB, 1)

				// the sizes must be equal, otherwise Equal does not compare the nodes
				if len(goldA) != len(goldB) {
					t.Fatalf("error in the test table: the sizes are %d and %d", len(goldA), len(goldB))
				}

				if goldA.Equal(goldB) {
					t.Fatal("error in the test table: the tables are equal")
				}

				a, b := modelBuild__TABLE_TYPE(pfxsA, 1), modelBuild__TABLE_TYPE(pfxsB, 1)

				// both directions, the difference of the kind of a child is
				// found from both sides
				if a.Equal(b) {
					t.Error("a.Equal(b) = true, want false")
				}

				if b.Equal(a) {
					t.Error("b.Equal(a) = true, want false")
				}
			})
		}
	}
}

// TestTableEqualValues tests that Equal compares the values of every kind of
// entry: the prefix table of a node, a node, a leaf and a fringe. A liteTable
// has no payload, the same prefixes with different values are equal.
func TestTableEqualValues__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	_, isLite := any(new(_TABLE_TYPE[int])).(*liteTable[int])

	for _, tt := range modelValueCases {
		for _, is4 := range []bool{true, false} {
			t.Run(modelName(tt.name, is4), func(t *testing.T) {
				t.Parallel()

				pfxs := modelPrefixes(is4, tt.specs)

				a, b := modelBuild__TABLE_TYPE(pfxs, 1), modelBuild__TABLE_TYPE(pfxs, 1001)

				if got := a.Equal(b); got != isLite {
					t.Errorf("Equal() with different values = %v, want %v", got, isLite)
				}

				if got := b.Equal(a); got != isLite {
					t.Errorf("Equal() with different values, swapped = %v, want %v", got, isLite)
				}
			})
		}
	}
}

// TestTableEqualSharedNodes tests Equal for tables that share nodes: an
// InsertPersist of a prefix with its own value gives an equal table that shares
// all untouched nodes with the original, the comparison of the shared nodes is a
// shortcut.
func TestTableEqualSharedNodes__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	for _, cfg := range modelConfigs {
		t.Run(cfg.name, func(t *testing.T) {
			t.Parallel()

			for seed := range modelSeeds() {
				prng := rand.New(rand.NewPCG(uint64(seed), 48))

				pfxs := random.NestedPrefixes(prng, cfg.is4, cfg.pool, 1+prng.IntN(cfg.maxN), true)
				gold := modelGold(pfxs, 100)
				tbl := modelBuild__TABLE_TYPE(pfxs, 100)

				pfx := pfxs[prng.IntN(len(pfxs))]
				same := tbl.InsertPersist(pfx, gold[pfx])

				if !tbl.Equal(same) || !same.Equal(tbl) {
					t.Fatalf("seed %d: InsertPersist of %s with its own value gives an unequal table", seed, pfx)
				}
			}
		})
	}
}

// TestTableFprintErrors tests that Fprint returns the error of the writer, for
// every position of the failure: the header and every line of the IPv4 tree and
// of the IPv6 tree.
func TestTableFprintErrors__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	tbl := new(_TABLE_TYPE[int])

	for i, s := range []string{
		"10.0.0.0/8", "10.1.0.0/16", "10.1.2.0/24", "192.168.0.0/16",
		"2001:db8::/32", "2001:db8:1::/48", "fe80::/10",
	} {
		tbl.Insert(mpp(s), i)
	}

	// count the writes of a successful Fprint
	counter := &modelFailWriter{failAt: -1}
	if err := tbl.Fprint(counter); err != nil {
		t.Fatalf("Fprint() = %v, want no error", err)
	}

	// two headers and at least one line per prefix
	if counter.writes < 2+7 {
		t.Fatalf("Fprint() made %d writes, the test needs more", counter.writes)
	}

	for failAt := range counter.writes {
		w := &modelFailWriter{failAt: failAt}

		if err := tbl.Fprint(w); !errors.Is(err, errModelWrite) {
			t.Errorf("Fprint() with a failing write #%d = %v, want the error of the writer", failAt, err)
		}
	}

	if err := tbl.Fprint(nil); err == nil {
		t.Error("Fprint(nil) = nil, want an error")
	}
}

// TestTableMarshalJSONError tests that MarshalJSON returns the error of the
// JSON encoder, here for a payload type that can not be marshaled.
func TestTableMarshalJSONError__TABLE_TYPE(t *testing.T) {
	t.Parallel()

	tbl := new(_TABLE_TYPE[func()])
	tbl.Insert(mpp("10.0.0.0/8"), nil)

	_, err := tbl.MarshalJSON()

	var unsupported *json.UnsupportedTypeError
	if !errors.As(err, &unsupported) {
		t.Fatalf("MarshalJSON() = %v, want a *json.UnsupportedTypeError", err)
	}
}
