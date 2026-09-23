// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package golden

import (
	"net/netip"
	"slices"
	"testing"
)

var (
	mpa = netip.MustParseAddr
	mpp = netip.MustParsePrefix
)

func TestTable_Equal(t *testing.T) {
	t.Parallel()

	prefixA := mpp("192.168.1.0/24")
	prefixB := mpp("10.0.0.0/8")

	tests := []struct {
		name string
		ta   Table[int]
		tb   Table[int]
		want bool
	}{
		{
			name: "both tables nil",
			ta:   nil,
			tb:   nil,
			want: true,
		},
		{
			name: "both tables empty",
			ta:   Table[int]{},
			tb:   Table[int]{},
			want: true,
		},
		{
			name: "identical key-value pairs",
			ta:   Table[int]{prefixA: 10, prefixB: 20},
			tb:   Table[int]{prefixA: 10, prefixB: 20},
			want: true,
		},
		{
			name: "identical keys with different values",
			ta:   Table[int]{prefixA: 10, prefixB: 20},
			tb:   Table[int]{prefixA: 10, prefixB: 99},
			want: false,
		},
		{
			name: "different keys",
			ta:   Table[int]{prefixA: 10},
			tb:   Table[int]{prefixB: 10},
			want: false,
		},
		{
			name: "one table nil, other non-empty",
			ta:   nil,
			tb:   Table[int]{prefixA: 10},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.ta.Equal(tt.tb); got != tt.want {
				t.Errorf("Table.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

type equalSlice []int

func (v equalSlice) Equal(other equalSlice) bool {
	return slices.Equal(v, other)
}

func TestTable_EqualCustom(t *testing.T) {
	t.Parallel()

	prefixA := mpp("192.168.1.0/24")
	prefixB := mpp("10.0.0.0/8")

	tests := []struct {
		name string
		ta   Table[equalSlice]
		tb   Table[equalSlice]
		want bool
	}{
		{
			name: "equal non-comparable values",
			ta:   Table[equalSlice]{prefixA: {1, 2, 3}},
			tb:   Table[equalSlice]{prefixA: {1, 2, 3}},
			want: true,
		},
		{
			name: "different non-comparable values",
			ta:   Table[equalSlice]{prefixA: {1, 2, 3}},
			tb:   Table[equalSlice]{prefixA: {1, 2, 4}},
			want: false,
		},
		{
			name: "nil and empty slices are custom-equal",
			ta: Table[equalSlice]{
				prefixA: nil,
				prefixB: {1, 2},
			},
			tb: Table[equalSlice]{
				prefixA: {},
				prefixB: {1, 2},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.ta.Equal(tt.tb); got != tt.want {
				t.Errorf("Table.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTable_FlatSorted(t *testing.T) {
	t.Parallel()

	p10 := netip.MustParsePrefix("10.0.0.0/8")
	p192_168_1_0_24 := netip.MustParsePrefix("192.168.1.0/24")
	p192_168_1_0_25 := netip.MustParsePrefix("192.168.1.0/25")
	pIPv6 := netip.MustParsePrefix("2001:db8::/32")

	tests := []struct {
		name  string
		table Table[string]
		want  []Item[string]
	}{
		{
			name:  "nil table",
			table: nil,
			want:  nil,
		},
		{
			name:  "empty table",
			table: Table[string]{},
			want:  nil,
		},
		{
			name: "single element",
			table: Table[string]{
				p10: "A",
			},
			want: []Item[string]{
				{Pfx: p10, Val: "A"},
			},
		},
		{
			name: "unsorted prefixes",
			table: Table[string]{
				p192_168_1_0_24: "C",
				p10:             "A",
				pIPv6:           "D",
				p192_168_1_0_25: "B",
			},
			want: []Item[string]{
				{Pfx: p10, Val: "A"},
				{Pfx: p192_168_1_0_24, Val: "C"},
				{Pfx: p192_168_1_0_25, Val: "B"},
				{Pfx: pIPv6, Val: "D"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.table.FlatSorted()

			if !slices.EqualFunc(got, tt.want, func(a, b Item[string]) bool {
				return a.Pfx == b.Pfx && a.Val == b.Val
			}) {
				t.Errorf("SortedItems() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTableSlice_SortKeys(t *testing.T) {
	t.Parallel()

	p10 := mpp("10.0.0.0/8")
	p192_24 := mpp("192.168.1.0/24")
	p192_25 := mpp("192.168.1.0/25")
	pIPv6 := mpp("2001:db8::/32")

	tests := []struct {
		name  string
		slice TableSlice[string]
		want  []netip.Prefix
	}{
		{
			name:  "nil slice",
			slice: nil,
			want:  nil,
		},
		{
			name:  "empty slice",
			slice: TableSlice[string]{},
			want:  nil,
		},
		{
			name: "single item",
			slice: TableSlice[string]{
				{Pfx: p10, Val: "A"},
			},
			want: []netip.Prefix{p10},
		},
		{
			name: "unsorted prefixes (IPv4 and IPv6)",
			slice: TableSlice[string]{
				{Pfx: p192_24, Val: "C"},
				{Pfx: p10, Val: "A"},
				{Pfx: pIPv6, Val: "D"},
				{Pfx: p192_25, Val: "B"},
			},
			want: []netip.Prefix{p10, p192_24, p192_25, pIPv6},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.slice.SortKeys()

			if !slices.Equal(got, tt.want) {
				t.Errorf("SortKeys() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTable_AllKeys(t *testing.T) {
	t.Parallel()

	p1 := mpp("10.0.0.0/8")
	p2 := mpp("192.168.1.0/24")
	p3 := mpp("2001:db8::/32")

	tests := []struct {
		name       string
		table      Table[int]
		wantKeys   []netip.Prefix
		breakEarly bool
	}{
		{
			name:     "nil table",
			table:    nil,
			wantKeys: nil,
		},
		{
			name:     "empty table",
			table:    Table[int]{},
			wantKeys: nil,
		},
		{
			name: "full iteration",
			table: Table[int]{
				p1: 1,
				p2: 2,
				p3: 3,
			},
			wantKeys: []netip.Prefix{p1, p2, p3},
		},
		{
			name: "early break iteration",
			table: Table[int]{
				p1: 1,
				p2: 2,
				p3: 3,
			},
			wantKeys:   []netip.Prefix{p1, p2, p3},
			breakEarly: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got []netip.Prefix
			for pfx := range tt.table.AllKeys() {
				got = append(got, pfx)
				if tt.breakEarly {
					break // Verifies yield returning false stops iteration
				}
			}

			if tt.breakEarly {
				if len(tt.table) > 0 && len(got) != 1 {
					t.Errorf("AllKeys() early break produced %d keys, want 1", len(got))
				}
				return
			}

			// Sort both slices since map iteration order is non-deterministic
			slices.SortFunc(got, cmpPrefix)
			wantSorted := slices.Clone(tt.wantKeys)
			slices.SortFunc(wantSorted, cmpPrefix)

			if !slices.Equal(got, wantSorted) {
				t.Errorf("AllKeys() produced %v, want %v", got, wantSorted)
			}
		})
	}
}

func TestTable_Insert(t *testing.T) {
	t.Parallel()

	prefixUnmasked := netip.MustParsePrefix("192.168.1.10/24") // normalizes to 192.168.1.0/24
	prefixMasked := mpp("192.168.1.0/24")
	prefixOther := mpp("10.0.0.0/8")

	tests := []struct {
		name       string
		initTable  func() *Table[string]
		pfx        netip.Prefix
		val        string
		wantKey    netip.Prefix
		wantLength int
	}{
		{
			name: "insert into nil pointer (safe no-op)",
			initTable: func() *Table[string] {
				return nil
			},
			pfx:        prefixMasked,
			val:        "data",
			wantLength: 0,
		},
		{
			name: "insert into nil map (allocates in-place)",
			initTable: func() *Table[string] {
				var tbl Table[string] // nil map
				return &tbl
			},
			pfx:        prefixMasked,
			val:        "value-1",
			wantKey:    prefixMasked,
			wantLength: 1,
		},
		{
			name: "insert into existing initialized map",
			initTable: func() *Table[string] {
				tbl := Table[string]{prefixOther: "existing"}
				return &tbl
			},
			pfx:        prefixMasked,
			val:        "value-2",
			wantKey:    prefixMasked,
			wantLength: 2,
		},
		{
			name: "insert normalizes unmasked prefix",
			initTable: func() *Table[string] {
				tbl := make(Table[string])
				return &tbl
			},
			pfx:        prefixUnmasked,
			val:        "normalized-val",
			wantKey:    prefixMasked,
			wantLength: 1,
		},
		{
			name: "overwrite existing key",
			initTable: func() *Table[string] {
				tbl := Table[string]{prefixMasked: "old-value"}
				return &tbl
			},
			pfx:        prefixMasked,
			val:        "new-value",
			wantKey:    prefixMasked,
			wantLength: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tblPtr := tt.initTable()
			tblPtr.Insert(tt.pfx, tt.val)

			if tblPtr == nil {
				if tt.wantLength != 0 {
					t.Fatalf("got nil table pointer, expected length %d", tt.wantLength)
				}
				return
			}

			tbl := *tblPtr
			if len(tbl) != tt.wantLength {
				t.Errorf("len(Table) = %d, want %d", len(tbl), tt.wantLength)
			}

			if tt.wantLength > 0 {
				gotVal, exists := tbl[tt.wantKey]
				if !exists {
					t.Errorf("key %v not found in table", tt.wantKey)
				}
				if gotVal != tt.val {
					t.Errorf("Table[%v] = %v, want %v", tt.wantKey, gotVal, tt.val)
				}
			}
		})
	}
}

func TestTable_Delete(t *testing.T) {
	t.Parallel()
	tbl := Table[int]{}
	tbl.Insert(mpp("192.168.1.0/24"), 1)
	tbl.Insert(mpp("10.0.0.0/8"), 2)

	// Delete existing prefix
	if !tbl.Delete(mpp("192.168.1.0/24")) {
		t.Error("expected Delete to return true for existing prefix")
	}
	if len(tbl) != 1 {
		t.Errorf("expected table length 1 after delete, got %d", len(tbl))
	}

	// Delete non-existing prefix
	if tbl.Delete(mpp("172.16.0.0/12")) {
		t.Error("expected Delete to return false for non-existing prefix")
	}

	// Delete non-masked prefix
	tbl.Insert(mpp("10.1.2.3/16"), 3)
	if !tbl.Delete(mpp("10.1.2.3/16")) {
		t.Error("expected Delete to handle non-masked prefix")
	}
}

func TestTable_Get(t *testing.T) {
	t.Parallel()
	tbl := Table[string]{}
	tbl.Insert(mpp("192.168.1.0/24"), "network")
	tbl.Insert(mpp("2001:db8::/32"), "ipv6")

	// Get existing prefix
	if val, ok := tbl.Get(mpp("192.168.1.0/24")); !ok || val != "network" {
		t.Errorf("expected ('network', true), got (%v, %v)", val, ok)
	}

	// Get non-existing prefix
	if val, ok := tbl.Get(mpp("10.0.0.0/8")); ok {
		t.Errorf("expected (empty, false), got (%v, %v)", val, ok)
	}

	// Get with non-masked prefix
	if val, ok := tbl.Get(mpp("192.168.1.5/24")); !ok || val != "network" {
		t.Errorf("expected ('network', true) for non-masked, got (%v, %v)", val, ok)
	}
}

func TestTable_Update(t *testing.T) {
	t.Parallel()
	tbl := Table[int]{}
	tbl.Insert(mpp("192.168.1.0/24"), 10)

	// Update existing entry
	val := tbl.Update(mpp("192.168.1.0/24"), func(v int, exists bool) int {
		if !exists {
			t.Error("expected exists=true for existing entry")
		}
		if v != 10 {
			t.Errorf("expected value 10, got %d", v)
		}
		return v + 5
	})
	if val != 15 {
		t.Errorf("expected updated value 15, got %d", val)
	}

	// Update non-existing entry (insert)
	val = tbl.Update(mpp("10.0.0.0/8"), func(v int, exists bool) int {
		if exists {
			t.Error("expected exists=false for non-existing entry")
		}
		return 100
	})
	if val != 100 {
		t.Errorf("expected new value 100, got %d", val)
	}
	if len(tbl) != 2 {
		t.Errorf("expected table length 2, got %d", len(tbl))
	}
}

func TestTable_Union(t *testing.T) {
	t.Parallel()
	tbl1 := Table[int]{}
	tbl1.Insert(mpp("192.168.1.0/24"), 1)
	tbl1.Insert(mpp("10.0.0.0/8"), 2)

	tbl2 := Table[int]{}
	tbl2.Insert(mpp("192.168.1.0/24"), 10) // Overlaps with tbl1
	tbl2.Insert(mpp("172.16.0.0/12"), 3)

	tbl1.Union(tbl2)

	if len(tbl1) != 3 {
		t.Errorf("expected table length 3 after union, got %d", len(tbl1))
	}

	// Check that overlapping prefix was updated
	if val, ok := tbl1.Get(mpp("192.168.1.0/24")); !ok || val != 10 {
		t.Errorf("expected value 10 for overlapping prefix, got %v, ok=%v", val, ok)
	}

	// Check that unique prefixes were added
	if val, ok := tbl1.Get(mpp("172.16.0.0/12")); !ok || val != 3 {
		t.Errorf("expected value 3 for new prefix, got %v, ok=%v", val, ok)
	}
}

func TestTable_Lookup(t *testing.T) {
	t.Parallel()
	tbl := Table[string]{}
	tbl.Insert(mpp("192.168.0.0/16"), "large")
	tbl.Insert(mpp("192.168.1.0/24"), "specific")
	tbl.Insert(mpp("10.0.0.0/8"), "ten")

	tests := []struct {
		ip      string
		wantVal string
		wantOk  bool
	}{
		{"192.168.1.5", "specific", true}, // Most specific match
		{"192.168.2.5", "large", true},    // Less specific match
		{"10.5.6.7", "ten", true},         // Match /8
		{"172.16.0.1", "", false},         // No match
		{"2001:db8::1", "", false},        // IPv6, no match
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			t.Parallel()
			val, ok := tbl.Lookup(mpa(tt.ip))
			if ok != tt.wantOk || val != tt.wantVal {
				t.Errorf("Lookup(%s) = (%v, %v), want (%v, %v)", tt.ip, val, ok, tt.wantVal, tt.wantOk)
			}
		})
	}
}

func TestTable_LookupPrefix(t *testing.T) {
	t.Parallel()
	tbl := Table[int]{}
	tbl.Insert(mpp("192.168.0.0/16"), 1)
	tbl.Insert(mpp("192.168.1.0/24"), 2)

	tests := []struct {
		prefix  string
		wantVal int
		wantOk  bool
	}{
		{"192.168.1.0/24", 2, true},  // Exact match
		{"192.168.1.0/25", 2, true},  // More specific, matches /24
		{"192.168.2.0/24", 1, true},  // Matches /16 only
		{"192.168.0.0/15", 0, false}, // Less specific than any entry
		{"10.0.0.0/8", 0, false},     // No match
	}

	for _, tt := range tests {
		t.Run(tt.prefix, func(t *testing.T) {
			t.Parallel()
			val, ok := tbl.LookupPrefix(mpp(tt.prefix))
			if ok != tt.wantOk || val != tt.wantVal {
				t.Errorf("LookupPrefix(%s) = (%v, %v), want (%v, %v)", tt.prefix, val, ok, tt.wantVal, tt.wantOk)
			}
		})
	}
}

func TestTable_LookupPrefixLPM(t *testing.T) {
	t.Parallel()
	tbl := Table[int]{}
	tbl.Insert(mpp("192.168.0.0/16"), 1)
	tbl.Insert(mpp("192.168.1.0/24"), 2)

	tests := []struct {
		prefix  string
		wantLpm string
		wantVal int
		wantOk  bool
	}{
		{"192.168.1.0/24", "192.168.1.0/24", 2, true},
		{"192.168.1.0/25", "192.168.1.0/24", 2, true},
		{"192.168.2.0/24", "192.168.0.0/16", 1, true},
		{"10.0.0.0/8", "", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.prefix, func(t *testing.T) {
			t.Parallel()
			lpm, val, ok := tbl.LookupPrefixLPM(mpp(tt.prefix))
			if ok != tt.wantOk {
				t.Errorf("LookupPrefixLPM(%s) ok = %v, want %v", tt.prefix, ok, tt.wantOk)
			}
			if ok {
				if lpm.String() != tt.wantLpm {
					t.Errorf("LookupPrefixLPM(%s) lpm = %v, want %v", tt.prefix, lpm, tt.wantLpm)
				}
				if val != tt.wantVal {
					t.Errorf("LookupPrefixLPM(%s) val = %v, want %v", tt.prefix, val, tt.wantVal)
				}
			}
		})
	}
}

func TestTable_Subnets(t *testing.T) {
	t.Parallel()
	tbl := Table[int]{}
	tbl.Insert(mpp("192.168.0.0/16"), 1)
	tbl.Insert(mpp("192.168.1.0/24"), 2)
	tbl.Insert(mpp("192.168.1.0/25"), 3)
	tbl.Insert(mpp("192.168.2.0/24"), 4)
	tbl.Insert(mpp("10.0.0.0/8"), 5)

	subnets := tbl.Subnets(mpp("192.168.0.0/16"))

	// Should include /16, /24, /25 in 192.168.0.0/16
	expected := []string{"192.168.0.0/16", "192.168.1.0/24", "192.168.1.0/25", "192.168.2.0/24"}
	if len(subnets) != len(expected) {
		t.Errorf("expected %d subnets, got %d", len(expected), len(subnets))
	}

	for i, exp := range expected {
		if subnets[i].String() != exp {
			t.Errorf("subnet[%d] = %v, want %v", i, subnets[i], exp)
		}
	}

	// Test empty result
	subnets = tbl.Subnets(mpp("172.16.0.0/12"))
	if len(subnets) != 0 {
		t.Errorf("expected 0 subnets for non-overlapping prefix, got %d", len(subnets))
	}
}

func TestTable_Supernets(t *testing.T) {
	t.Parallel()
	tbl := Table[int]{}
	tbl.Insert(mpp("192.168.0.0/16"), 1)
	tbl.Insert(mpp("192.168.1.0/24"), 2)
	tbl.Insert(mpp("192.0.0.0/8"), 3)

	supernets := tbl.Supernets(mpp("192.168.1.0/25"))

	// Should return /24, /16, /8 in reverse order (most specific first)
	expected := []string{"192.168.1.0/24", "192.168.0.0/16", "192.0.0.0/8"}
	if len(supernets) != len(expected) {
		t.Errorf("expected %d supernets, got %d", len(expected), len(supernets))
	}

	for i, exp := range expected {
		if supernets[i].String() != exp {
			t.Errorf("supernet[%d] = %v, want %v", i, supernets[i], exp)
		}
	}
}

func TestTable_OverlapsPrefix(t *testing.T) {
	t.Parallel()
	tbl := Table[int]{}
	tbl.Insert(mpp("192.168.0.0/16"), 1)
	tbl.Insert(mpp("10.0.0.0/8"), 2)

	tests := []struct {
		prefix string
		want   bool
	}{
		{"192.168.1.0/24", true}, // Overlaps with /16
		{"192.168.0.0/15", true}, // Overlaps with /16
		{"10.5.6.0/24", true},    // Overlaps with /8
		{"172.16.0.0/12", false}, // No overlap
		{"2001:db8::/32", false}, // IPv6, no overlap
	}

	for _, tt := range tests {
		t.Run(tt.prefix, func(t *testing.T) {
			t.Parallel()
			got := tbl.OverlapsPrefix(mpp(tt.prefix))
			if got != tt.want {
				t.Errorf("OverlapsPrefix(%s) = %v, want %v", tt.prefix, got, tt.want)
			}
		})
	}
}

func TestTable_Overlaps(t *testing.T) {
	t.Parallel()
	tbl1 := Table[int]{}
	tbl1.Insert(mpp("192.168.0.0/16"), 1)
	tbl1.Insert(mpp("10.0.0.0/8"), 2)

	tbl2 := Table[int]{}
	tbl2.Insert(mpp("192.168.1.0/24"), 3) // Overlaps with tbl1

	if !tbl1.Overlaps(tbl2) {
		t.Error("expected tables to overlap")
	}

	tbl3 := Table[int]{}
	tbl3.Insert(mpp("172.16.0.0/12"), 4) // No overlap

	if tbl1.Overlaps(tbl3) {
		t.Error("expected tables not to overlap")
	}
}

func TestTable_All(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		tbl  Table[int]
		want Table[int]
	}{
		{
			name: "empty table",
			tbl:  Table[int]{},
			want: nil,
		},
		{
			name: "single entry",
			tbl:  Table[int]{mpp("192.168.1.0/24"): 1},
			want: Table[int]{mpp("192.168.1.0/24"): 1},
		},
		{
			name: "multiple IPv4 and IPv6 entries",
			tbl: Table[int]{
				mpp("10.0.0.0/8"):     10,
				mpp("2001:db8::/32"):  30,
				mpp("192.168.1.0/24"): 20,
			},
			want: Table[int]{
				mpp("10.0.0.0/8"):     10,
				mpp("2001:db8::/32"):  30,
				mpp("192.168.1.0/24"): 20,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for pfx, val := range tt.tbl.All() {
				wantVal, ok := tt.want[pfx]
				if !ok {
					t.Errorf("All(): not ok, missing val %v", val)
				} else if val != wantVal {
					t.Errorf("All(): want: %v, got: %v", wantVal, val)
				}
				delete(tt.want, pfx)
			}

			if len(tt.want) != 0 {
				t.Error("All(): not all items iterated")
			}
		})
	}

	t.Run("early break", func(t *testing.T) {
		t.Parallel()
		tbl := Table[int]{
			mpp("10.0.0.0/8"):     1,
			mpp("192.168.1.0/24"): 2,
			mpp("172.16.0.0/12"):  3,
		}

		var count int
		for range tbl.All() {
			count++
			if count == 2 {
				break
			}
		}

		if count != 2 {
			t.Errorf("All() early break failed: processed %d items, want 2", count)
		}
	})
}

func TestTable_Empty(t *testing.T) {
	t.Parallel()
	tbl := Table[int]{}

	// Test operations on empty table
	if len(tbl.FlatSorted()) != 0 {
		t.Error("expected empty SortedItems result")
	}

	if _, ok := tbl.Get(mpp("192.168.1.0/24")); ok {
		t.Error("expected Get to return false on empty table")
	}

	if _, ok := tbl.Lookup(mpa("192.168.1.1")); ok {
		t.Error("expected Lookup to return false on empty table")
	}

	if tbl.Delete(mpp("192.168.1.0/24")) {
		t.Error("expected Delete to return false on empty table")
	}
}

func TestTable_IPv6(t *testing.T) {
	t.Parallel()
	tbl := Table[string]{}
	tbl.Insert(mpp("2001:db8::/32"), "ipv6")
	tbl.Insert(mpp("2001:db8:1::/48"), "specific")

	// Test IPv6 lookup
	val, ok := tbl.Lookup(mpa("2001:db8:1::1"))
	if !ok || val != "specific" {
		t.Errorf("Lookup(2001:db8:1::1) = (%v, %v), want (specific, true)", val, ok)
	}

	// Test IPv6 subnets
	subnets := tbl.Subnets(mpp("2001:db8::/32"))
	if len(subnets) != 2 {
		t.Errorf("expected 2 IPv6 subnets, got %d", len(subnets))
	}
}

func Test_CmpPrefix(t *testing.T) {
	t.Parallel()
	tests := []struct {
		a    string
		b    string
		want int
	}{
		{"10.0.0.0/8", "192.168.0.0/16", -1},     // Different address
		{"192.168.0.0/16", "192.168.0.0/24", -1}, // Same address, different bits
		{"192.168.0.0/24", "192.168.0.0/24", 0},  // Equal
		{"192.168.1.0/24", "192.168.0.0/24", 1},  // Different address
	}

	for _, tt := range tests {
		a := mpp(tt.a)
		b := mpp(tt.b)
		got := cmpPrefix(a, b)

		// Normalize to -1, 0, 1
		if got < 0 {
			got = -1
		} else if got > 0 {
			got = 1
		}

		if got != tt.want {
			t.Errorf("cmpPrefix(%s, %s) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestTable_MixedIPVersions(t *testing.T) {
	t.Parallel()
	tbl := Table[int]{}
	tbl.Insert(mpp("192.168.1.0/24"), 4)
	tbl.Insert(mpp("2001:db8::/32"), 6)

	sorted := tbl.FlatSorted()
	if len(sorted) != 2 {
		t.Errorf("expected 2 prefixes, got %d", len(sorted))
	}

	// IPv4 should come before IPv6 in sorted order
	if !sorted[0].Pfx.Addr().Is4() {
		t.Error("expected IPv4 prefix first in sorted order")
	}
	if !sorted[1].Pfx.Addr().Is6() {
		t.Error("expected IPv6 prefix second in sorted order")
	}
}

func TestTable_Aggregate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   Table[any]
		want Table[any]
	}{
		{
			name: "empty",
			in:   Table[any]{},
			want: Table[any]{},
		},
		{
			name: "single",
			in:   Table[any]{mpp("10.0.0.0/8"): nil},
			want: Table[any]{mpp("10.0.0.0/8"): nil},
		},
		{
			name: "containment",
			in: Table[any]{
				mpp("10.1.0.0/16"): nil,
				mpp("10.0.0.0/8"):  nil,
				mpp("10.1.1.0/24"): nil,
			},
			want: Table[any]{mpp("10.0.0.0/8"): nil},
		},
		{
			name: "adjacency single merge",
			in: Table[any]{
				mpp("192.168.0.0/25"):   nil,
				mpp("192.168.0.128/25"): nil,
			},
			want: Table[any]{mpp("192.168.0.0/24"): nil},
		},
		{
			name: "cascading merge",
			in: Table[any]{
				mpp("10.0.0.0/26"):   nil,
				mpp("10.0.0.64/26"):  nil,
				mpp("10.0.0.128/26"): nil,
				mpp("10.0.0.192/26"): nil,
			},
			want: Table[any]{mpp("10.0.0.0/24"): nil},
		},
		{
			name: "mixed v4 and v6",
			in: Table[any]{
				mpp("2001:db8::/33"):      nil,
				mpp("10.0.0.0/25"):        nil,
				mpp("2001:db8:8000::/33"): nil,
				mpp("10.0.0.128/25"):      nil,
			},
			want: Table[any]{
				mpp("10.0.0.0/24"):   nil,
				mpp("2001:db8::/32"): nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tbl := tt.in
			tbl.Aggregate()

			// don't check the values
			if !tbl.Equal(tt.want) {
				t.Errorf("got %v, want %v", tbl, tt.want)
			}
		})
	}
}
