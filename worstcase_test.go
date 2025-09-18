// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"testing"
)

var (
	worstCaseProbeIP4  = mpa("255.255.255.255")
	worstCaseProbePfx4 = mpp("255.255.255.255/32")

	ipv4DefaultRoute = mpp("0.0.0.0/0")
	worstCasePfxsIP4 = []netip.Prefix{
		mpp("0.0.0.0/1"),
		mpp("254.0.0.0/8"),
		mpp("255.0.0.0/9"),
		mpp("255.254.0.0/16"),
		mpp("255.255.0.0/17"),
		mpp("255.255.254.0/24"),
		mpp("255.255.255.0/25"),
		mpp("255.255.255.255/32"), // matching prefix
	}

	worstCaseProbeIP6  = mpa("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff")
	worstCaseProbePfx6 = mpp("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff/128")

	ipv6DefaultRoute = mpp("::/0")
	worstCasePfxsIP6 = []netip.Prefix{
		mpp("::/1"),
		mpp("fe00::/8"),
		mpp("ff00::/9"),
		mpp("fffe::/16"),
		mpp("ffff::/17"),
		mpp("ffff:fe00::/24"),
		mpp("ffff:ff00::/25"),
		mpp("ffff:fffe::/32"),
		mpp("ffff:ffff::/33"),
		mpp("ffff:ffff:fe00::/40"),
		mpp("ffff:ffff:ff00::/41"),
		mpp("ffff:ffff:fffe::/48"),
		mpp("ffff:ffff:ffff::/49"),
		mpp("ffff:ffff:ffff:fe00::/56"),
		mpp("ffff:ffff:ffff:ff00::/57"),
		mpp("ffff:ffff:ffff:fffe::/64"),
		mpp("ffff:ffff:ffff:ffff::/65"),
		mpp("ffff:ffff:ffff:ffff:fe00::/72"),
		mpp("ffff:ffff:ffff:ffff:ff00::/73"),
		mpp("ffff:ffff:ffff:ffff:fffe::/80"),
		mpp("ffff:ffff:ffff:ffff:ffff::/81"),
		mpp("ffff:ffff:ffff:ffff:ffff:fe00::/88"),
		mpp("ffff:ffff:ffff:ffff:ffff:ff00::/89"),
		mpp("ffff:ffff:ffff:ffff:ffff:fffe::/96"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff::/97"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:fe00::/104"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:ff00::/105"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:fffe::/112"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:ffff::/113"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:ffff:fe00/120"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ff00/121"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:ffff:fffe/128"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff/128"),
	}
)

type tabler[V any] interface {
	Delete(netip.Prefix) (V, bool)
	Insert(netip.Prefix, V)
	Contains(netip.Addr) bool
	Lookup(netip.Addr) (V, bool)
	LookupPrefix(netip.Prefix) (V, bool)
	LookupPrefixLPM(netip.Prefix) (netip.Prefix, V, bool)
}

func TestWorstCaseMatch4(t *testing.T) {
	t.Parallel()

	type tables struct {
		name    string
		builder func() tabler[string]
	}

	testCases := []tables{
		{
			name:    "Table",
			builder: func() tabler[string] { return &Table[string]{} },
		},
		{
			name:    "Fast",
			builder: func() tabler[string] { return &Fast[string]{} },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name+"_Contains", func(t *testing.T) {
			t.Parallel()

			tbl := tc.builder()
			for _, p := range worstCasePfxsIP4 {
				tbl.Insert(p, p.String())
			}

			want := true
			ok := tbl.Contains(worstCaseProbeIP4)
			if ok != want {
				t.Errorf("%s: Contains, worst case match IP4, expected OK: %v, got: %v", tc.name, want, ok)
			}
		})

		t.Run(tc.name+"_Lookup", func(t *testing.T) {
			t.Parallel()

			tbl := tc.builder()
			for _, p := range worstCasePfxsIP4 {
				tbl.Insert(p, p.String())
			}

			want := true
			_, ok := tbl.Lookup(worstCaseProbeIP4)
			if ok != want {
				t.Errorf("%s: Lookup, worst case match IP4, expected OK: %v, got: %v", tc.name, want, ok)
			}
		})

		t.Run(tc.name+"_LookupPrefix", func(t *testing.T) {
			t.Parallel()

			tbl := tc.builder()
			for _, p := range worstCasePfxsIP4 {
				tbl.Insert(p, p.String())
			}

			want := true
			_, ok := tbl.LookupPrefix(worstCaseProbePfx4)
			if ok != want {
				t.Errorf("%s: LookupPrefix, worst case match IP4 pfx, expected OK: %v, got: %v", tc.name, want, ok)
			}
		})

		t.Run(tc.name+"_LookupPrefixLPM", func(t *testing.T) {
			t.Parallel()

			tbl := tc.builder()
			for _, p := range worstCasePfxsIP4 {
				tbl.Insert(p, p.String())
			}

			want := true
			_, _, ok := tbl.LookupPrefixLPM(worstCaseProbePfx4)
			if ok != want {
				t.Errorf("%s: LookupPrefixLPM, worst case match IP4 pfx, expected OK: %v, got: %v", tc.name, want, ok)
			}
		})
	}
}

func TestWorstCaseMiss4(t *testing.T) {
	t.Parallel()

	type tables struct {
		name    string
		builder func() tabler[string]
	}

	testCases := []tables{
		{
			name:    "Table",
			builder: func() tabler[string] { return &Table[string]{} },
		},
		{
			name:    "Fast",
			builder: func() tabler[string] { return &Fast[string]{} },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name+"_Contains", func(t *testing.T) {
			t.Parallel()

			tbl := tc.builder()
			for _, p := range worstCasePfxsIP4 {
				tbl.Insert(p, p.String())
			}

			tbl.Delete(worstCaseProbePfx4) // delete matching prefix

			want := false
			ok := tbl.Contains(worstCaseProbeIP4)
			if ok != want {
				t.Errorf("%s: Contains, worst case miss IP4, expected OK: %v, got: %v", tc.name, want, ok)
			}
		})

		t.Run(tc.name+"_Lookup", func(t *testing.T) {
			t.Parallel()

			tbl := tc.builder()
			for _, p := range worstCasePfxsIP4 {
				tbl.Insert(p, p.String())
			}

			tbl.Delete(worstCaseProbePfx4) // delete matching prefix

			want := false
			_, ok := tbl.Lookup(worstCaseProbeIP4)
			if ok != want {
				t.Errorf("%s: Lookup, worst case miss IP4, expected OK: %v, got: %v", tc.name, want, ok)
			}
		})

		t.Run(tc.name+"_LookupPrefix", func(t *testing.T) {
			t.Parallel()

			tbl := tc.builder()
			for _, p := range worstCasePfxsIP4 {
				tbl.Insert(p, p.String())
			}

			tbl.Delete(worstCaseProbePfx4) // delete matching prefix

			want := false
			_, ok := tbl.LookupPrefix(worstCaseProbePfx4)
			if ok != want {
				t.Errorf("%s: LookupPrefix, worst case miss IP4 pfx, expected OK: %v, got: %v", tc.name, want, ok)
			}
		})

		t.Run(tc.name+"_LookupPfxLPM", func(t *testing.T) {
			t.Parallel()

			tbl := tc.builder()
			for _, p := range worstCasePfxsIP4 {
				tbl.Insert(p, p.String())
			}

			tbl.Delete(worstCaseProbePfx4) // delete matching prefix

			want := false
			_, _, ok := tbl.LookupPrefixLPM(worstCaseProbePfx4)
			if ok != want {
				t.Errorf("%s: LookupPrefixLPM, worst case miss IP4 pfx, expected OK: %v, got: %v", tc.name, want, ok)
			}
		})
	}
}

func TestWorstCaseMatch6(t *testing.T) {
	t.Parallel()

	type tables struct {
		name    string
		builder func() tabler[string]
	}

	testCases := []tables{
		{
			name:    "Table",
			builder: func() tabler[string] { return &Table[string]{} },
		},
		{
			name:    "Fast",
			builder: func() tabler[string] { return &Fast[string]{} },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name+"_Contains", func(t *testing.T) {
			t.Parallel()

			tbl := tc.builder()
			for _, p := range worstCasePfxsIP6 {
				tbl.Insert(p, p.String())
			}

			want := true
			ok := tbl.Contains(worstCaseProbeIP6)
			if ok != want {
				t.Errorf("%s: Contains, worst case match IP6, expected OK: %v, got: %v", tc.name, want, ok)
			}
		})

		t.Run(tc.name+"_Lookup", func(t *testing.T) {
			t.Parallel()

			tbl := tc.builder()
			for _, p := range worstCasePfxsIP6 {
				tbl.Insert(p, p.String())
			}

			want := true
			_, ok := tbl.Lookup(worstCaseProbeIP6)
			if ok != want {
				t.Errorf("%s: Lookup, worst case match IP6, expected OK: %v, got: %v", tc.name, want, ok)
			}
		})

		t.Run(tc.name+"_LookupPrefix", func(t *testing.T) {
			t.Parallel()

			tbl := tc.builder()
			for _, p := range worstCasePfxsIP6 {
				tbl.Insert(p, p.String())
			}

			want := true
			_, ok := tbl.LookupPrefix(worstCaseProbePfx6)
			if ok != want {
				t.Errorf("%s: LookupPrefix, worst case match IP6 pfx, expected OK: %v, got: %v", tc.name, want, ok)
			}
		})

		t.Run(tc.name+"_LookupPrefixLPM", func(t *testing.T) {
			t.Parallel()

			tbl := tc.builder()
			for _, p := range worstCasePfxsIP6 {
				tbl.Insert(p, p.String())
			}

			want := true
			_, _, ok := tbl.LookupPrefixLPM(worstCaseProbePfx6)
			if ok != want {
				t.Errorf("%s: LookupPrefixLPM, worst case match IP6 pfx, expected OK: %v, got: %v", tc.name, want, ok)
			}
		})
	}
}

func TestWorstCaseMiss6(t *testing.T) {
	t.Parallel()

	type tables struct {
		name    string
		builder func() tabler[string]
	}

	testCases := []tables{
		{
			name:    "Table",
			builder: func() tabler[string] { return &Table[string]{} },
		},
		{
			name:    "Fast",
			builder: func() tabler[string] { return &Fast[string]{} },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name+"_Contains", func(t *testing.T) {
			t.Parallel()

			tbl := tc.builder()
			for _, p := range worstCasePfxsIP6 {
				tbl.Insert(p, p.String())
			}

			tbl.Delete(worstCaseProbePfx6) // delete matching prefix

			want := false
			ok := tbl.Contains(worstCaseProbeIP6)
			if ok != want {
				t.Errorf("%s: Contains, worst case miss IP6, expected OK: %v, got: %v", tc.name, want, ok)
			}
		})

		t.Run("Lookup", func(t *testing.T) {
			t.Parallel()

			tbl := tc.builder()
			for _, p := range worstCasePfxsIP6 {
				tbl.Insert(p, p.String())
			}

			tbl.Delete(worstCaseProbePfx6) // delete matching prefix

			want := false
			_, ok := tbl.Lookup(worstCaseProbeIP6)
			if ok != want {
				t.Errorf("%s: Lookup, worst case miss IP6, expected OK: %v, got: %v", tc.name, want, ok)
			}
		})

		t.Run("LookupPrefix", func(t *testing.T) {
			t.Parallel()

			tbl := tc.builder()
			for _, p := range worstCasePfxsIP6 {
				tbl.Insert(p, p.String())
			}

			tbl.Delete(worstCaseProbePfx6) // delete matching prefix

			want := false
			_, ok := tbl.LookupPrefix(worstCaseProbePfx6)
			if ok != want {
				t.Errorf("%s: LookupPrefix, worst case miss IP6 pfx, expected OK: %v, got: %v", tc.name, want, ok)
			}
		})

		t.Run("LookupPfxLPM", func(t *testing.T) {
			t.Parallel()

			tbl := tc.builder()
			for _, p := range worstCasePfxsIP6 {
				tbl.Insert(p, p.String())
			}

			tbl.Delete(worstCaseProbePfx6) // delete matching prefix

			want := false
			_, _, ok := tbl.LookupPrefixLPM(worstCaseProbePfx6)
			if ok != want {
				t.Errorf("%s: LookupPrefixLPM, worst case miss IP6 pfx, expected OK: %v, got: %v", tc.name, want, ok)
			}
		})
	}
}

func BenchmarkWorstCaseMatch4(b *testing.B) {
	type tables struct {
		name    string
		builder func() tabler[string]
	}

	testCases := []tables{
		{
			name:    "Table",
			builder: func() tabler[string] { return &Table[string]{} },
		},
		{
			name:    "Fast",
			builder: func() tabler[string] { return &Fast[string]{} },
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name+"_Contains", func(b *testing.B) {
			tbl := tc.builder()
			for _, p := range worstCasePfxsIP4 {
				tbl.Insert(p, p.String())
			}

			for b.Loop() {
				tbl.Contains(worstCaseProbeIP4)
			}
		})

		b.Run(tc.name+"_Lookup", func(b *testing.B) {
			tbl := tc.builder()
			for _, p := range worstCasePfxsIP4 {
				tbl.Insert(p, p.String())
			}
			tbl.Insert(ipv4DefaultRoute, ipv4DefaultRoute.String())
			tbl.Delete(worstCaseProbePfx4)

			for b.Loop() {
				tbl.Lookup(worstCaseProbeIP4)
			}
		})

		b.Run(tc.name+"_LookupPrefix", func(b *testing.B) {
			tbl := tc.builder()
			for _, p := range worstCasePfxsIP4 {
				tbl.Insert(p, p.String())
			}
			tbl.Insert(ipv4DefaultRoute, ipv4DefaultRoute.String())
			tbl.Delete(worstCaseProbePfx4)

			for b.Loop() {
				tbl.LookupPrefix(worstCaseProbePfx4)
			}
		})

		b.Run(tc.name+"_LookupPrefixLPM", func(b *testing.B) {
			tbl := tc.builder()
			for _, p := range worstCasePfxsIP4 {
				tbl.Insert(p, p.String())
			}
			tbl.Insert(ipv4DefaultRoute, ipv4DefaultRoute.String())
			tbl.Delete(worstCaseProbePfx4)

			for b.Loop() {
				tbl.LookupPrefixLPM(worstCaseProbePfx4)
			}
		})
	}
}

func BenchmarkWorstCaseMiss4(b *testing.B) {
	type tables struct {
		name    string
		builder func() tabler[string]
	}

	testCases := []tables{
		{
			name:    "Table",
			builder: func() tabler[string] { return &Table[string]{} },
		},
		{
			name:    "Fast",
			builder: func() tabler[string] { return &Fast[string]{} },
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name+"_Contains", func(b *testing.B) {
			tbl := tc.builder()
			for _, p := range worstCasePfxsIP4 {
				tbl.Insert(p, p.String())
			}

			tbl.Delete(worstCaseProbePfx4) // delete matching prefix

			for b.Loop() {
				tbl.Contains(worstCaseProbeIP4)
			}
		})

		b.Run(tc.name+"_Lookup", func(b *testing.B) {
			tbl := tc.builder()
			for _, p := range worstCasePfxsIP4 {
				tbl.Insert(p, p.String())
			}

			tbl.Delete(worstCaseProbePfx4) // delete matching prefix

			for b.Loop() {
				tbl.Lookup(worstCaseProbeIP4)
			}
		})

		b.Run(tc.name+"_LookupPrefix", func(b *testing.B) {
			tbl := tc.builder()
			for _, p := range worstCasePfxsIP4 {
				tbl.Insert(p, p.String())
			}

			tbl.Delete(worstCaseProbePfx4) // delete matching prefix

			for b.Loop() {
				tbl.LookupPrefix(worstCaseProbePfx4)
			}
		})

		b.Run(tc.name+"_LookupPrefixLPM", func(b *testing.B) {
			tbl := tc.builder()
			for _, p := range worstCasePfxsIP4 {
				tbl.Insert(p, p.String())
			}

			tbl.Delete(worstCaseProbePfx4) // delete matching prefix

			for b.Loop() {
				tbl.LookupPrefixLPM(worstCaseProbePfx4)
			}
		})
	}
}

func BenchmarkWorstCaseMatch6(b *testing.B) {
	type tables struct {
		name    string
		builder func() tabler[string]
	}

	testCases := []tables{
		{
			name:    "Table",
			builder: func() tabler[string] { return &Table[string]{} },
		},
		{
			name:    "Fast",
			builder: func() tabler[string] { return &Fast[string]{} },
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name+"_Contains", func(b *testing.B) {
			tbl := tc.builder()
			for _, p := range worstCasePfxsIP6 {
				tbl.Insert(p, p.String())
			}

			for b.Loop() {
				tbl.Contains(worstCaseProbeIP6)
			}
		})

		b.Run(tc.name+"_Lookup", func(b *testing.B) {
			tbl := tc.builder()
			for _, p := range worstCasePfxsIP6 {
				tbl.Insert(p, p.String())
			}
			tbl.Insert(ipv6DefaultRoute, ipv6DefaultRoute.String())
			tbl.Delete(worstCaseProbePfx6)

			for b.Loop() {
				tbl.Lookup(worstCaseProbeIP6)
			}
		})

		b.Run(tc.name+"_LookupPrefix", func(b *testing.B) {
			tbl := tc.builder()
			for _, p := range worstCasePfxsIP6 {
				tbl.Insert(p, p.String())
			}
			tbl.Insert(ipv6DefaultRoute, ipv6DefaultRoute.String())
			tbl.Delete(worstCaseProbePfx6)

			for b.Loop() {
				tbl.LookupPrefix(worstCaseProbePfx6)
			}
		})

		b.Run(tc.name+"_LookupPrefixLPM", func(b *testing.B) {
			tbl := tc.builder()
			for _, p := range worstCasePfxsIP6 {
				tbl.Insert(p, p.String())
			}
			tbl.Insert(ipv6DefaultRoute, ipv6DefaultRoute.String())
			tbl.Delete(worstCaseProbePfx6)

			for b.Loop() {
				tbl.LookupPrefixLPM(worstCaseProbePfx6)
			}
		})
	}
}

func BenchmarkWorstCaseMiss6(b *testing.B) {
	type tables struct {
		name    string
		builder func() tabler[string]
	}

	testCases := []tables{
		{
			name:    "Table",
			builder: func() tabler[string] { return &Table[string]{} },
		},
		{
			name:    "Fast",
			builder: func() tabler[string] { return &Fast[string]{} },
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name+"_Contains", func(b *testing.B) {
			tbl := tc.builder()
			for _, p := range worstCasePfxsIP6 {
				tbl.Insert(p, p.String())
			}

			tbl.Delete(worstCaseProbePfx6) // delete matching prefix

			for b.Loop() {
				tbl.Contains(worstCaseProbeIP6)
			}
		})

		b.Run(tc.name+"_Lookup", func(b *testing.B) {
			tbl := tc.builder()
			for _, p := range worstCasePfxsIP6 {
				tbl.Insert(p, p.String())
			}

			tbl.Delete(worstCaseProbePfx6) // delete matching prefix

			for b.Loop() {
				tbl.Lookup(worstCaseProbeIP6)
			}
		})

		b.Run(tc.name+"_LookupPrefix", func(b *testing.B) {
			tbl := tc.builder()
			for _, p := range worstCasePfxsIP6 {
				tbl.Insert(p, p.String())
			}

			tbl.Delete(worstCaseProbePfx6) // delete matching prefix

			for b.Loop() {
				tbl.LookupPrefix(worstCaseProbePfx6)
			}
		})

		b.Run(tc.name+"_LookupPrefixLPM", func(b *testing.B) {
			tbl := tc.builder()
			for _, p := range worstCasePfxsIP6 {
				tbl.Insert(p, p.String())
			}

			tbl.Delete(worstCaseProbePfx6) // delete matching prefix

			for b.Loop() {
				tbl.LookupPrefixLPM(worstCaseProbePfx6)
			}
		})
	}
}
