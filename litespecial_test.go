// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"testing"
)

func TestTableNil_LiteTable(t *testing.T) {
	t.Parallel()

	ip4 := mpa("127.0.0.1")
	ip6 := mpa("::1")

	pfx4 := mpp("127.0.0.0/8")
	pfx6 := mpp("::1/128")

	tbl2 := new(Lite)
	tbl2.Insert(pfx4)
	tbl2.Insert(pfx6)

	var tbl1 *Lite = nil

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
		mustPanic(t, "Insert", func() { tbl1.Insert(pfx4) })
		mustPanic(t, "InsertPersist", func() { tbl1.InsertPersist(pfx4) })
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

		mustPanic(t, "dump", func() { tbl1.dump(nil) })
		mustPanic(t, "dumpString", func() { tbl1.dumpString() })
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

func TestTableInvalid_LiteTable(t *testing.T) {
	t.Parallel()

	tbl1 := new(Lite)
	tbl2 := new(Lite)

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
	noPanic(t, "Insert", func() { tbl1.Insert(zeroPfx) })
	noPanic(t, "InsertPersist", func() { tbl1.InsertPersist(zeroPfx) })
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
