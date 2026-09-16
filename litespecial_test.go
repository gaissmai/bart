// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"slices"
	"testing"
)

func TestTableNilReceiver_LiteTable(t *testing.T) {
	t.Parallel()

	ip4 := mpa("127.0.0.1")
	ip6 := mpa("::1")

	pfx4 := mpp("127.0.0.0/8")
	pfx6 := mpp("::1/128")

	var tbl1 *Lite = nil

	t.Run("mustPanic", func(t *testing.T) {
		t.Parallel()

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
		mustPanic(t, "Aggregate", func() { tbl1.Aggregate() })
		mustPanic(t, "Clone", func() { tbl1.Clone() })
		mustPanic(t, "Union", func() { tbl1.Union(nil) })
		mustPanic(t, "UnionPersist", func() { tbl1.UnionPersist(nil) })
		mustPanic(t, "Overlaps", func() { tbl1.Overlaps(nil) })
		mustPanic(t, "Overlaps4", func() { tbl1.Overlaps4(nil) })
		mustPanic(t, "Overlaps6", func() { tbl1.Overlaps6(nil) })
		mustPanic(t, "OverlapsPrefix", func() { tbl1.OverlapsPrefix(pfx4) })
		mustPanic(t, "OverlapsPrefix", func() { tbl1.OverlapsPrefix(pfx6) })
		mustPanic(t, "Equal", func() { tbl1.Equal(nil) })
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

func TestLiteIteratorsEarlyExit(t *testing.T) {
	t.Parallel()

	tbl := new(Lite)
	tbl.Insert(mpp("10.0.0.0/8"))
	tbl.Insert(mpp("10.20.0.0/16"))
	tbl.Insert(mpp("2001:db8::/32"))

	// Test All early exit
	count := 0
	for range tbl.All() {
		count++
		break // early exit
	}
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}

	// Test AllSorted early exit
	count = 0
	for range tbl.AllSorted() {
		count++
		break // early exit
	}
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}

	// Test Subnets early exit
	count = 0
	for range tbl.Subnets(mpp("10.0.0.0/8")) {
		count++
		break
	}
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}

	// Test Supernets early exit
	count = 0
	for range tbl.Supernets(mpp("10.20.0.0/16")) {
		count++
		break
	}
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}
}

// helper
func parseAndCollect(pfxStrings []string) []netip.Prefix {
	pfxs := make([]netip.Prefix, len(pfxStrings))
	for i, value := range pfxStrings {
		pfxs[i] = mpp(value)
	}
	return pfxs
}

func TestLiteAggregate(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name: "empty",
		},
		{
			name:  "duplicate canonical prefix",
			input: []string{"192.0.2.0/24", "192.0.2.0/24"},
			want:  []string{"192.0.2.0/24"},
		},
		{
			name:  "prefix subsumption",
			input: []string{"10.1.2.0/24", "10.1.0.0/16", "10.0.0.0/8"},
			want:  []string{"10.0.0.0/8"},
		},
		{
			name:  "child subsumption",
			input: []string{"192.0.2.1/32", "192.0.2.128/25", "192.0.2.0/24"},
			want:  []string{"192.0.2.0/24"},
		},
		{
			name:  "ipv4 adjacent prefixes",
			input: []string{"192.0.2.0/25", "192.0.2.128/25"},
			want:  []string{"192.0.2.0/24"},
		},
		{
			name: "ipv4 recursive leaf merging",
			input: []string{
				"192.0.2.0/32",
				"192.0.2.1/32",
				"192.0.2.2/32",
				"192.0.2.3/32",
			},
			want: []string{"192.0.2.0/30"},
		},
		{
			name: "Promote each child after recursive aggregation",
			input: []string{
				"10.0.0.0/25",
				"10.0.0.128/25",
				"10.0.1.0/24",
			},
			want: []string{"10.0.0.0/23"},
		},
		{
			name: "ipv4 recursive fringe merging",
			input: []string{
				"8.0.0.0/8",
				"9.0.0.0/8",
				"10.0.0.0/8",
				"11.0.0.0/8",
			},
			want: []string{"8.0.0.0/6"},
		},
		{
			name:  "non siblings remain separate",
			input: []string{"10.0.0.0/8", "12.0.0.0/8"},
			want:  []string{"10.0.0.0/8", "12.0.0.0/8"},
		},
		{
			name:  "different prefix lengths remain separate",
			input: []string{"192.0.2.0/25", "192.0.2.128/26"},
			want:  []string{"192.0.2.0/25", "192.0.2.128/26"},
		},
		{
			name:  "ipv6 adjacent prefixes",
			input: []string{"2001:db8::/65", "2001:db8:0:0:8000::/65"},
			want:  []string{"2001:db8::/64"},
		},
		{
			name: "ipv6 recursive leaf merging",
			input: []string{
				"2001:db8::/128",
				"2001:db8::1/128",
				"2001:db8::2/128",
				"2001:db8::3/128",
			},
			want: []string{"2001:db8::/126"},
		},
		{
			name:  "default routes subsume their family",
			input: []string{"0.0.0.0/0", "10.0.0.0/8", "128.0.0.0/1", "::/0", "2001:db8::/32"},
			want:  []string{"0.0.0.0/0", "::/0"},
		},
		{
			name:  "both address families",
			input: []string{"192.0.2.0/25", "192.0.2.128/25", "2001:db8::/65", "2001:db8:0:0:8000::/65"},
			want:  []string{"192.0.2.0/24", "2001:db8::/64"},
		},

		// more corner cases
		{
			name:  "adjacent leaf siblings ipv4",
			input: []string{"10.0.0.0/24", "10.0.1.0/24"},
			want:  []string{"10.0.0.0/23"},
		},
		{
			name:  "adjacent non-byte-aligned siblings ipv4",
			input: []string{"10.0.0.0/25", "10.0.0.128/25"},
			want:  []string{"10.0.0.0/24"},
		},
		{
			name:  "adjacent siblings ipv6",
			input: []string{"2001:db8::/53", "2001:db8:0:0800::/53"},
			want:  []string{"2001:db8::/52"},
		},
		{
			name:  "zero address fringe pair ipv4",
			input: []string{"0.0.0.0/8", "1.0.0.0/8"},
			want:  []string{"0.0.0.0/7"},
		},
		{
			name:  "zero address fringe pair ipv6",
			input: []string{"::/8", "100::/8"},
			want:  []string{"::/7"},
		},
		{
			name: "covered child with multiple routes",
			input: []string{
				"10.0.0.0/8",
				"10.1.0.0/17",
				"10.1.128.0/17",
				"10.2.0.0/17",
				"10.2.128.0/17",
			},
			want: []string{"10.0.0.0/8"},
		},
		{
			name: "cascading prefix merges",
			input: []string{
				"0.0.0.0/5",
				"8.0.0.0/5",
				"16.0.0.0/5",
				"24.0.0.0/5",
			},
			want: []string{"0.0.0.0/3"},
		},
		{
			name:  "partial sibling set",
			input: []string{"0.0.0.0/8", "1.0.0.0/8", "2.0.0.0/8"},
			want:  []string{"0.0.0.0/7", "2.0.0.0/8"},
		},
		{
			name:  "non-adjacent prefixes remain separate",
			input: []string{"10.0.0.0/24", "10.0.2.0/24"},
			want:  []string{"10.0.0.0/24", "10.0.2.0/24"},
		},
		{
			name:  "single prefix",
			input: []string{"192.0.2.1/32"},
			want:  []string{"192.0.2.1/32"},
		},
		{
			name:  "default route per family",
			input: []string{"0.0.0.0/0", "10.0.0.0/8", "::/0", "2001:db8::/32"},
			want:  []string{"0.0.0.0/0", "::/0"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			table := new(Lite)
			for _, prefix := range parseAndCollect(test.input) {
				table.Insert(prefix)
			}

			table.Aggregate()

			want := parseAndCollect(test.want)
			got := slices.Collect(table.AllSorted())
			if !slices.Equal(got, want) {
				t.Fatalf("%s: Aggregate(), got: %v, want: %v", test.name, got, want)
			}

			want4, want6 := 0, 0
			for _, prefix := range want {
				if prefix.Addr().Is4() {
					want4++
				} else {
					want6++
				}
			}
			if table.Size() != len(want) || table.Size4() != want4 || table.Size6() != want6 {
				t.Fatalf("%s: sizes, got: (%d, %d, %d), want: (%d, %d, %d)",
					test.name, table.Size(), table.Size4(), table.Size6(), len(want), want4, want6)
			}

			first := slices.Collect(table.AllSorted())
			table.Aggregate()
			second := slices.Collect(table.AllSorted())
			if !slices.Equal(second, first) {
				t.Fatalf("%s: Aggregate() is not idempotent: first %v, second %v", test.name, first, second)
			}
		})
	}
}

func TestLiteAggregatePreservesMembership(t *testing.T) {
	tests := []struct {
		name   string
		input  []string
		probes []string
	}{
		{
			name:  "ipv4 boundaries",
			input: []string{"192.0.2.0/25", "192.0.2.128/25"},
			probes: []string{
				"192.0.1.255",
				"192.0.2.0",
				"192.0.2.127",
				"192.0.2.128",
				"192.0.2.255",
				"192.0.3.0",
			},
		},
		{
			name:  "ipv6 boundaries",
			input: []string{"2001:db8::/65", "2001:db8:0:0:8000::/65"},
			probes: []string{
				"2001:db7:ffff:ffff:ffff:ffff:ffff:ffff",
				"2001:db8::",
				"2001:db8:0:0:ffff:ffff:ffff:ffff",
				"2001:db9::",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := new(Lite)
			for _, prefix := range parseAndCollect(test.input) {
				before.Insert(prefix)
			}

			after := before.Clone()
			after.Aggregate()

			for _, address := range test.probes {
				ip := netip.MustParseAddr(address)

				beforeContains := before.Contains(ip)
				afterContains := after.Contains(ip)

				if afterContains != beforeContains {
					t.Errorf("Contains(%s) changed from %v to %v", address, beforeContains, afterContains)
				}
			}
		})
	}
}
