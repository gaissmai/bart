// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"math/rand/v2"
	"net/netip"
	"testing"
)

// expectedOverlaps computes a slow O(n*m) ground-truth overlap between two sets.
func expectedOverlaps(as, bs []netip.Prefix) bool {
	for _, a := range as {
		for _, b := range bs {
			if a.Overlaps(b) {
				return true
			}
		}
	}
	return false
}

// expectedOverlapsPrefix computes whether any prefix in set as overlaps q.
func expectedOverlapsPrefix(as []netip.Prefix, q netip.Prefix) bool {
	for _, a := range as {
		if a.Overlaps(q) {
			return true
		}
	}
	return false
}

// TestOverlapsDeterministic_EdgeCases exercises thorough edge coverage for Overlaps across all table types:
// - IPv6: siblings (/33 under /32), default route (::/0), host routes (/128), /127 semantics
// - IPv4: default route (0.0.0.0/0), host routes (/32), /31 semantics, adjacent siblings
// - Mixed-family disjointness
func TestOverlapsDeterministic_EdgeCases(t *testing.T) {
	t.Parallel()

	type pair struct {
		name  string
		pfxsA []string
		pfxsB []string
		want  bool // expected Overlaps(A,B)
	}

	cases := []pair{
		// IPv6: two halves of a /32 are siblings and do not overlap
		{
			name:  "IPv6_siblings_/33",
			pfxsA: []string{"2001:db8::/33"},
			pfxsB: []string{"2001:db8:8000::/33"},
			want:  false,
		},
		// IPv6: default route overlaps everything in IPv6 space
		{
			name:  "IPv6_default_overlaps_db8_32",
			pfxsA: []string{"::/0"},
			pfxsB: []string{"2001:db8::/32"},
			want:  true,
		},
		// IPv6: host vs default
		{
			name:  "IPv6_host_vs_default_zero",
			pfxsA: []string{"::/128"},
			pfxsB: []string{"::/0"},
			want:  true,
		},
		// IPv6: max host vs default
		{
			name:  "IPv6_max_host_vs_default",
			pfxsA: []string{"ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff/128"},
			pfxsB: []string{"::/0"},
			want:  true,
		},
		// IPv6: exact host
		{
			name:  "IPv6_exact_host",
			pfxsA: []string{"2001:db8::1/128"},
			pfxsB: []string{"2001:db8::1/128"},
			want:  true,
		},
		// IPv6: adjacent /64 siblings
		{
			name:  "IPv6_adjacent_/64",
			pfxsA: []string{"2001:db8::/64"},
			pfxsB: []string{"2001:db8:0:1::/64"},
			want:  false,
		},
		// IPv6: /127 covers exactly two hosts
		{
			name:  "IPv6_/127_covers_two_hosts",
			pfxsA: []string{"2001:db8::/127"},
			pfxsB: []string{"2001:db8::/128", "2001:db8::1/128"},
			want:  true,
		},
		// IPv6: /127 excludes next host outside its range
		{
			name:  "IPv6_/127_excludes_next_host",
			pfxsA: []string{"2001:db8::/127"},
			pfxsB: []string{"2001:db8::2/128"},
			want:  false,
		},
		// IPv6: adjacent /127 ranges do not overlap
		{
			name:  "IPv6_adjacent_/127",
			pfxsA: []string{"2001:db8::/127"},
			pfxsB: []string{"2001:db8::2/127"},
			want:  false,
		},
		// IPv6: /127 overlapped by its /126 supernet
		{
			name:  "IPv6_/127_vs_/126",
			pfxsA: []string{"2001:db8::/127"},
			pfxsB: []string{"2001:db8::/126"},
			want:  true,
		},
		// IPv4: default route overlaps everything in IPv4 space
		{
			name:  "IPv4_default_vs_host",
			pfxsA: []string{"0.0.0.0/0"},
			pfxsB: []string{"255.255.255.255/32"},
			want:  true,
		},
		// IPv4: adjacent /25 siblings do not overlap
		{
			name:  "IPv4_adjacent_/25",
			pfxsA: []string{"10.0.0.0/25"},
			pfxsB: []string{"10.0.0.128/25"},
			want:  false,
		},
		// IPv4: exact host
		{
			name:  "IPv4_exact_host",
			pfxsA: []string{"192.168.1.1/32"},
			pfxsB: []string{"192.168.1.1/32"},
			want:  true,
		},
		// IPv4: /31 covers exactly two hosts
		{
			name:  "IPv4_/31_contains_two_hosts",
			pfxsA: []string{"192.0.2.0/31"},
			pfxsB: []string{"192.0.2.0/32", "192.0.2.1/32"},
			want:  true,
		},
		// IPv4: /31 excludes next host outside its range
		{
			name:  "IPv4_/31_excludes_next_host",
			pfxsA: []string{"192.0.2.0/31"},
			pfxsB: []string{"192.0.2.2/32"},
			want:  false,
		},
		// IPv4: adjacent /31 ranges do not overlap
		{
			name:  "IPv4_adjacent_/31",
			pfxsA: []string{"192.0.2.0/31"},
			pfxsB: []string{"192.0.2.2/31"},
			want:  false,
		},
		// IPv4: /31 overlapped by its /30 supernet
		{
			name:  "IPv4_/31_vs_/30",
			pfxsA: []string{"192.0.2.0/31"},
			pfxsB: []string{"192.0.2.0/30"},
			want:  true,
		},
		// Mixed-family: IPv4 and IPv6 do not overlap
		{
			name:  "Mixed_v4_vs_v6",
			pfxsA: []string{"10.0.0.0/8"},
			pfxsB: []string{"2001:db8::/32"},
			want:  false,
		},
		// Multi-entry: at least one overlapping pair exists
		{
			name:  "Multi_entry_overlap",
			pfxsA: []string{"10.0.0.0/8", "172.16.0.0/12"},
			pfxsB: []string{"192.168.0.0/16", "10.64.0.0/10"},
			want:  true,
		},
		// Multi-entry: completely disjoint sets
		{
			name:  "Multi_entry_disjoint",
			pfxsA: []string{"10.0.0.0/16", "172.16.0.0/12"},
			pfxsB: []string{"10.1.0.0/16", "203.0.113.0/24"},
			want:  false,
		},
	}

	for _, tc := range cases {
		t.Run("Table_"+tc.name, func(t *testing.T) {
			t.Parallel()
			a := new(Table[int])
			b := new(Table[int])
			for i, s := range tc.pfxsA {
				a.Insert(netip.MustParsePrefix(s), i)
			}
			for i, s := range tc.pfxsB {
				b.Insert(netip.MustParsePrefix(s), i)
			}
			if got := a.Overlaps(b); got != tc.want {
				t.Fatalf("Table.Overlaps: want %v, got %v (A=%v B=%v)", tc.want, got, tc.pfxsA, tc.pfxsB)
			}
		})

		t.Run("Fast_"+tc.name, func(t *testing.T) {
			t.Parallel()
			a := new(Fast[int])
			b := new(Fast[int])
			for i, s := range tc.pfxsA {
				a.Insert(netip.MustParsePrefix(s), i)
			}
			for i, s := range tc.pfxsB {
				b.Insert(netip.MustParsePrefix(s), i)
			}
			if got := a.Overlaps(b); got != tc.want {
				t.Fatalf("Fast.Overlaps: want %v, got %v (A=%v B=%v)", tc.want, got, tc.pfxsA, tc.pfxsB)
			}
		})

		t.Run("liteTable_"+tc.name, func(t *testing.T) {
			t.Parallel()
			a := new(liteTable[int])
			b := new(liteTable[int])
			for i, s := range tc.pfxsA {
				a.Insert(netip.MustParsePrefix(s), i)
			}
			for i, s := range tc.pfxsB {
				b.Insert(netip.MustParsePrefix(s), i)
			}
			if got := a.Overlaps(b); got != tc.want {
				t.Fatalf("liteTable.Overlaps: want %v, got %v (A=%v B=%v)", tc.want, got, tc.pfxsA, tc.pfxsB)
			}
		})

		t.Run("Lite_"+tc.name, func(t *testing.T) {
			t.Parallel()
			a := new(Lite)
			b := new(Lite)
			for _, s := range tc.pfxsA {
				a.Insert(netip.MustParsePrefix(s))
			}
			for _, s := range tc.pfxsB {
				b.Insert(netip.MustParsePrefix(s))
			}
			if got := a.Overlaps(b); got != tc.want {
				t.Fatalf("Lite.Overlaps: want %v, got %v (A=%v B=%v)", tc.want, got, tc.pfxsA, tc.pfxsB)
			}
		})
	}
}

// TestOverlapsPrefixDeterministic_EdgeCases exercises corner queries against a mixed content set
// for all table types. It includes: default routes, host routes, IPv6 /127, IPv4 /31, mixed-family.
func TestOverlapsPrefixDeterministic_EdgeCases(t *testing.T) {
	t.Parallel()

	// Mixed IPv4/IPv6 contents, covering host routes and small subnets.
	contents := []string{
		"10.0.0.0/8",
		"192.0.2.0/31",
		"203.0.113.128/25",
		"2001:db8::/32",
		"2001:db8::/127",
		"::/128",
	}

	// Queries and expected OverlapsPrefix results with the content above.
	queries := []struct {
		probes string
		want   bool
	}{
		// IPv4 /31 semantics and supernets
		{"192.0.2.0/32", true},
		{"192.0.2.1/32", true},
		{"192.0.2.2/32", false},
		{"192.0.2.0/31", true},
		{"192.0.2.0/30", true},
		// Another IPv4 network
		{"203.0.113.255/32", true}, // within 203.0.113.128/25
		{"203.0.113.0/25", false},  // different /25
		{"0.0.0.0/0", true},        // supernet of all IPv4

		// IPv6 /127 and host routes
		{"2001:db8::/127", true},
		{"2001:db8::/128", true},
		{"2001:db8::1/128", true},
		{"2001:db8::2/128", true},
		{"2001:db9::1/128", false},
		{"::/0", true},     // supernet of all IPv6
		{"::1/128", false}, // different host than ::/128
	}

	var cPfx []netip.Prefix
	for _, s := range contents {
		cPfx = append(cPfx, netip.MustParsePrefix(s))
	}

	// Table
	t.Run("Table", func(t *testing.T) {
		t.Parallel()
		tbl := new(Table[int])
		for i, p := range cPfx {
			tbl.Insert(p, i)
		}
		for _, q := range queries {
			got := tbl.OverlapsPrefix(netip.MustParsePrefix(q.probes))
			if got != q.want {
				t.Fatalf("Table.OverlapsPrefix(%s): want %v, got %v", q.probes, q.want, got)
			}
		}
	})

	// Fast
	t.Run("Fast", func(t *testing.T) {
		t.Parallel()
		ft := new(Fast[int])
		for i, p := range cPfx {
			ft.Insert(p, i)
		}
		for _, q := range queries {
			got := ft.OverlapsPrefix(netip.MustParsePrefix(q.probes))
			if got != q.want {
				t.Fatalf("Fast.OverlapsPrefix(%s): want %v, got %v", q.probes, q.want, got)
			}
		}
	})

	// liteTable
	t.Run("liteTable", func(t *testing.T) {
		t.Parallel()
		lt := new(liteTable[int])
		for i, p := range cPfx {
			lt.Insert(p, i)
		}
		for _, q := range queries {
			got := lt.OverlapsPrefix(netip.MustParsePrefix(q.probes))
			if got != q.want {
				t.Fatalf("liteTable.OverlapsPrefix(%s): want %v, got %v", q.probes, q.want, got)
			}
		}
	})

	// Lite
	t.Run("Lite", func(t *testing.T) {
		t.Parallel()
		L := new(Lite)
		for _, p := range cPfx {
			L.Insert(p)
		}
		for _, q := range queries {
			got := L.OverlapsPrefix(netip.MustParsePrefix(q.probes))
			if got != q.want {
				t.Fatalf("Lite.OverlapsPrefix(%s): want %v, got %v", q.probes, q.want, got)
			}
		}
	})
}

func TestOverlapsGoldenCompare(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))

	const numEntries = 10

	for range n {
		pfxs := randomPrefixes(prng, numEntries)
		inter := randomPrefixes(prng, numEntries)

		gold := new(goldTable[int])
		gold.insertMany(pfxs)

		goldInter := new(goldTable[int])
		goldInter.insertMany(inter)
		gotGold := gold.overlaps(goldInter)

		// Table

		bart := new(Table[int])
		for _, pfx := range pfxs {
			bart.Insert(pfx.pfx, pfx.val)
		}

		bartInter := new(Table[int])
		for _, pfx := range inter {
			bartInter.Insert(pfx.pfx, pfx.val)
		}

		gotBart := bart.Overlaps(bartInter)

		if gotGold != gotBart {
			t.Fatalf("Overlaps(...) = %v, want %v\nbart1:\n%s\nbart2:\n%s",
				gotBart, gotGold, bart.String(), bartInter.String())
		}

		// Fast

		fast := new(Fast[int])
		for _, pfx := range pfxs {
			fast.Insert(pfx.pfx, pfx.val)
		}

		fastInter := new(Fast[int])
		for _, pfx := range inter {
			fastInter.Insert(pfx.pfx, pfx.val)
		}

		gotFast := fast.Overlaps(fastInter)

		if gotGold != gotFast {
			t.Fatalf("Overlaps(...) = %v, want %v\nfast1:\n%s\nfast2:\n%s",
				gotFast, gotGold, fast.String(), fastInter.String())
		}

		// Lite

		lite := new(Lite)
		for _, pfx := range pfxs {
			lite.Insert(pfx.pfx)
		}

		liteInter := new(Lite)
		for _, pfx := range inter {
			liteInter.Insert(pfx.pfx)
		}

		gotLite := lite.Overlaps(liteInter)

		if gotGold != gotLite {
			t.Fatalf("Overlaps(...) = %v, want %v\nlite1:\n%s\nlite2:\n%s",
				gotLite, gotGold, lite.String(), liteInter.String())
		}
	}
}

func TestOverlapsPrefixGoldenCompare(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, n)

	gold := new(goldTable[int])
	gold.insertMany(pfxs)

	bart := new(Table[int])
	fast := new(Fast[int])
	lite := new(liteTable[int])
	for _, pfx := range pfxs {
		bart.Insert(pfx.pfx, pfx.val)
		fast.Insert(pfx.pfx, pfx.val)
		lite.Insert(pfx.pfx, pfx.val)
	}

	t.Run("Table", func(t *testing.T) {
		t.Parallel()

		tests := randomPrefixes(prng, n)
		for _, tt := range tests {
			gotGold := gold.overlapsPrefix(tt.pfx)
			gotBart := bart.OverlapsPrefix(tt.pfx)
			if gotGold != gotBart {
				t.Fatalf("overlapsPrefix(%q) = %v, want %v", tt.pfx, gotBart, gotGold)
			}
		}
	})

	t.Run("Fast", func(t *testing.T) {
		t.Parallel()

		tests := randomPrefixes(prng, n)
		for _, tt := range tests {
			gotGold := gold.overlapsPrefix(tt.pfx)
			gotFast := fast.OverlapsPrefix(tt.pfx)
			if gotGold != gotFast {
				t.Fatalf("overlapsPrefix(%q) = %v, want %v", tt.pfx, gotFast, gotGold)
			}
		}
	})

	t.Run("liteTable", func(t *testing.T) {
		t.Parallel()

		tests := randomPrefixes(prng, n)
		for _, tt := range tests {
			gotGold := gold.overlapsPrefix(tt.pfx)
			gotLite := lite.OverlapsPrefix(tt.pfx)
			if gotGold != gotLite {
				t.Fatalf("overlapsPrefix(%q) = %v, want %v", tt.pfx, gotLite, gotGold)
			}
		}
	})
}
