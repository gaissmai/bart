// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package golden

import (
	"net/netip"
	"testing"
)

var (
	mpa = netip.MustParseAddr
	mpp = netip.MustParsePrefix
)

func TestTableInsert(t *testing.T) {
	tbl := new(Table[int])

	// Insert IPv4 prefix
	tbl.Insert(mpp("192.168.1.0/24"), 1)
	if len(*tbl) != 1 {
		t.Errorf("expected table length 1, got %d", len(*tbl))
	}

	// Insert duplicate - should update value
	tbl.Insert(mpp("192.168.1.0/24"), 2)
	if len(*tbl) != 1 {
		t.Errorf("expected table length 1 after duplicate insert, got %d", len(*tbl))
	}
	if val, ok := tbl.Get(mpp("192.168.1.0/24")); !ok || val != 2 {
		t.Errorf("expected value 2, got %v, ok=%v", val, ok)
	}

	// Insert non-masked prefix - should auto-mask
	tbl.Insert(mpp("10.1.2.3/16"), 3)
	pfxs := tbl.AllSorted()
	for _, pfx := range pfxs {
		if pfx != pfx.Masked() {
			t.Errorf("prefix %v is not masked", pfx)
		}
	}
}

func TestTableDelete(t *testing.T) {
	tbl := new(Table[int])
	tbl.Insert(mpp("192.168.1.0/24"), 1)
	tbl.Insert(mpp("10.0.0.0/8"), 2)

	// Delete existing prefix
	if !tbl.Delete(mpp("192.168.1.0/24")) {
		t.Error("expected Delete to return true for existing prefix")
	}
	if len(*tbl) != 1 {
		t.Errorf("expected table length 1 after delete, got %d", len(*tbl))
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

func TestTableGet(t *testing.T) {
	tbl := new(Table[string])
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

func TestTableUpdate(t *testing.T) {
	tbl := new(Table[int])
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
	if len(*tbl) != 2 {
		t.Errorf("expected table length 2, got %d", len(*tbl))
	}
}

func TestTableUnion(t *testing.T) {
	tbl1 := new(Table[int])
	tbl1.Insert(mpp("192.168.1.0/24"), 1)
	tbl1.Insert(mpp("10.0.0.0/8"), 2)

	tbl2 := new(Table[int])
	tbl2.Insert(mpp("192.168.1.0/24"), 10) // Overlaps with tbl1
	tbl2.Insert(mpp("172.16.0.0/12"), 3)

	tbl1.Union(tbl2)

	if len(*tbl1) != 3 {
		t.Errorf("expected table length 3 after union, got %d", len(*tbl1))
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

func TestTableLookup(t *testing.T) {
	tbl := new(Table[string])
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
			val, ok := tbl.Lookup(mpa(tt.ip))
			if ok != tt.wantOk || val != tt.wantVal {
				t.Errorf("Lookup(%s) = (%v, %v), want (%v, %v)", tt.ip, val, ok, tt.wantVal, tt.wantOk)
			}
		})
	}
}

func TestTableLookupPrefix(t *testing.T) {
	tbl := new(Table[int])
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
			val, ok := tbl.LookupPrefix(mpp(tt.prefix))
			if ok != tt.wantOk || val != tt.wantVal {
				t.Errorf("LookupPrefix(%s) = (%v, %v), want (%v, %v)", tt.prefix, val, ok, tt.wantVal, tt.wantOk)
			}
		})
	}
}

func TestTableLookupPrefixLPM(t *testing.T) {
	tbl := new(Table[int])
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

func TestTableSubnets(t *testing.T) {
	tbl := new(Table[int])
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

func TestTableSupernets(t *testing.T) {
	tbl := new(Table[int])
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

func TestTableOverlapsPrefix(t *testing.T) {
	tbl := new(Table[int])
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
			got := tbl.OverlapsPrefix(mpp(tt.prefix))
			if got != tt.want {
				t.Errorf("OverlapsPrefix(%s) = %v, want %v", tt.prefix, got, tt.want)
			}
		})
	}
}

func TestTableOverlaps(t *testing.T) {
	tbl1 := new(Table[int])
	tbl1.Insert(mpp("192.168.0.0/16"), 1)
	tbl1.Insert(mpp("10.0.0.0/8"), 2)

	tbl2 := new(Table[int])
	tbl2.Insert(mpp("192.168.1.0/24"), 3) // Overlaps with tbl1

	if !tbl1.Overlaps(tbl2) {
		t.Error("expected tables to overlap")
	}

	tbl3 := new(Table[int])
	tbl3.Insert(mpp("172.16.0.0/12"), 4) // No overlap

	if tbl1.Overlaps(tbl3) {
		t.Error("expected tables not to overlap")
	}
}

func TestTableAllSorted(t *testing.T) {
	tbl := new(Table[int])
	tbl.Insert(mpp("192.168.1.0/24"), 1)
	tbl.Insert(mpp("10.0.0.0/8"), 2)
	tbl.Insert(mpp("192.168.0.0/16"), 3)
	tbl.Insert(mpp("2001:db8::/32"), 4)

	sorted := tbl.AllSorted()

	// Check IPv4 prefixes are sorted
	if len(sorted) != 4 {
		t.Errorf("expected 4 prefixes, got %d", len(sorted))
	}

	// Verify sorted order
	for i := 1; i < len(sorted); i++ {
		if cmpPrefix(sorted[i-1], sorted[i]) >= 0 {
			t.Errorf("prefixes not sorted: %v should be before %v", sorted[i-1], sorted[i])
		}
	}
}

func TestTableSort(t *testing.T) {
	tbl := new(Table[string])
	tbl.Insert(mpp("192.168.1.0/24"), "c")
	tbl.Insert(mpp("10.0.0.0/8"), "a")
	tbl.Insert(mpp("192.168.0.0/16"), "b")

	tbl.Sort()

	// Verify in-place sorting
	for i := 1; i < len(*tbl); i++ {
		if cmpPrefix((*tbl)[i-1].Pfx, (*tbl)[i].Pfx) >= 0 {
			t.Errorf("table not sorted at index %d: %v, %v", i, (*tbl)[i-1].Pfx, (*tbl)[i].Pfx)
		}
	}
}

func TestTableEmpty(t *testing.T) {
	tbl := new(Table[int])

	// Test operations on empty table
	if len(tbl.AllSorted()) != 0 {
		t.Error("expected empty AllSorted result")
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

func TestTableItemString(t *testing.T) {
	item := TableItem[int]{
		Pfx: mpp("192.168.1.0/24"),
		Val: 42,
	}

	str := item.String()
	expected := "(192.168.1.0/24, 42)"
	if str != expected {
		t.Errorf("String() = %q, want %q", str, expected)
	}
}

func TestTableIPv6(t *testing.T) {
	tbl := new(Table[string])
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

func TestCmpPrefix(t *testing.T) {
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

func TestTableMixedIPVersions(t *testing.T) {
	tbl := new(Table[int])
	tbl.Insert(mpp("192.168.1.0/24"), 4)
	tbl.Insert(mpp("2001:db8::/32"), 6)

	sorted := tbl.AllSorted()
	if len(sorted) != 2 {
		t.Errorf("expected 2 prefixes, got %d", len(sorted))
	}

	// IPv4 should come before IPv6 in sorted order
	if !sorted[0].Addr().Is4() {
		t.Error("expected IPv4 prefix first in sorted order")
	}
	if !sorted[1].Addr().Is6() {
		t.Error("expected IPv6 prefix second in sorted order")
	}
}
