// Copyright (c) 2024 Karl Gaissmaier
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

func TestWorstCaseMatch4(t *testing.T) {
	t.Parallel()

	t.Run("Contains", func(t *testing.T) {
		t.Parallel()

		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}

		want := true
		ok := tbl.Contains(worstCaseProbeIP4)
		if ok != want {
			t.Errorf("Contains, worst case match IP4, expected OK: %v, got: %v", want, ok)
		}
	})

	t.Run("Lookup", func(t *testing.T) {
		t.Parallel()

		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}
		tbl.Insert(ipv4DefaultRoute, ipv4DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx4)

		wantVal := ipv4DefaultRoute.String()
		want := true
		val, ok := tbl.Lookup(worstCaseProbeIP4)
		if ok != want {
			t.Errorf("Lookup, worst case match IP4, expected OK: %v, got: %v", want, ok)
		}
		if val != wantVal {
			t.Errorf("Lookup, worst case match IP4, expected: %v, got: %v", wantVal, val)
		}
	})

	t.Run("LookupPrefix", func(t *testing.T) {
		t.Parallel()

		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}
		tbl.Insert(ipv4DefaultRoute, ipv4DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx4)

		wantVal := ipv4DefaultRoute.String()
		want := true
		val, ok := tbl.LookupPrefix(worstCaseProbePfx4)
		if ok != want {
			t.Errorf("LookupPrefix, worst case match IP4 pfx, expected OK: %v, got: %v", want, ok)
		}
		if val != wantVal {
			t.Errorf("LookupPrefix, worst case match IP4 pfx, expected: %v, got: %v", wantVal, val)
		}
	})

	t.Run("LookupPfxLPM", func(t *testing.T) {
		t.Parallel()

		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}
		tbl.Insert(ipv4DefaultRoute, ipv4DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx4)

		wantLPM := ipv4DefaultRoute
		wantVal := ipv4DefaultRoute.String()
		want := true
		lpm, val, ok := tbl.LookupPrefixLPM(worstCaseProbePfx4)
		if ok != want {
			t.Errorf("LookupPrefixLPM, worst case match IP4 pfx, expected OK: %v, got: %v", want, ok)
		}
		if val != wantVal {
			t.Errorf("LookupPrefixLPM, worst case match IP4 pfx, expected: %v, got: %v", wantVal, val)
		}
		if lpm != wantLPM {
			t.Errorf("LookupPrefixLPM, worst case match IP4 pfx, expected: %v, got: %v", wantLPM, lpm)
		}
	})
}

func TestWorstCaseMiss4(t *testing.T) {
	t.Parallel()

	t.Run("Contains", func(t *testing.T) {
		t.Parallel()

		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx4) // delete matching prefix

		want := false
		ok := tbl.Contains(worstCaseProbeIP4)
		if ok != want {
			t.Errorf("Contains, worst case miss IP4, expected OK: %v, got: %v", want, ok)
		}
	})

	t.Run("Lookup", func(t *testing.T) {
		t.Parallel()

		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx4) // delete matching prefix

		want := false
		_, ok := tbl.Lookup(worstCaseProbeIP4)
		if ok != want {
			t.Errorf("Lookup, worst case miss IP4, expected OK: %v, got: %v", want, ok)
		}
	})

	t.Run("LookupPrefix", func(t *testing.T) {
		t.Parallel()

		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx4) // delete matching prefix

		want := false
		_, ok := tbl.LookupPrefix(worstCaseProbePfx4)
		if ok != want {
			t.Errorf("LookupPrefix, worst case miss IP4 pfx, expected OK: %v, got: %v", want, ok)
		}
	})

	t.Run("LookupPfxLPM", func(t *testing.T) {
		t.Parallel()

		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx4) // delete matching prefix

		want := false
		_, _, ok := tbl.LookupPrefixLPM(worstCaseProbePfx4)
		if ok != want {
			t.Errorf("LookupPrefixLPM, worst case miss IP4 pfx, expected OK: %v, got: %v", want, ok)
		}
	})
}

func TestWorstCaseMatch6(t *testing.T) {
	t.Parallel()

	t.Run("Contains", func(t *testing.T) {
		t.Parallel()

		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}

		want := true
		ok := tbl.Contains(worstCaseProbeIP6)
		if ok != want {
			t.Errorf("Contains, worst case match IP6, expected OK: %v, got: %v", want, ok)
		}
	})

	t.Run("Lookup", func(t *testing.T) {
		t.Parallel()

		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}
		tbl.Insert(ipv6DefaultRoute, ipv6DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx6)

		wantVal := ipv6DefaultRoute.String()
		want := true
		val, ok := tbl.Lookup(worstCaseProbeIP6)
		if ok != want {
			t.Errorf("Lookup, worst case match IP6, expected OK: %v, got: %v", want, ok)
		}
		if val != wantVal {
			t.Errorf("Lookup, worst case match IP6, expected: %v, got: %v", wantVal, val)
		}
	})

	t.Run("LookupPrefix", func(t *testing.T) {
		t.Parallel()

		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}
		tbl.Insert(ipv6DefaultRoute, ipv6DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx6)

		wantVal := ipv6DefaultRoute.String()
		want := true
		val, ok := tbl.LookupPrefix(worstCaseProbePfx6)
		if ok != want {
			t.Errorf("LookupPrefix, worst case match IP6 pfx, expected OK: %v, got: %v", want, ok)
		}
		if val != wantVal {
			t.Errorf("LookupPrefix, worst case match IP6 pfx, expected: %v, got: %v", wantVal, val)
		}
	})

	t.Run("LookupPfxLPM", func(t *testing.T) {
		t.Parallel()

		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}
		tbl.Insert(ipv6DefaultRoute, ipv6DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx6)

		wantLPM := ipv6DefaultRoute
		wantVal := ipv6DefaultRoute.String()
		want := true
		lpm, val, ok := tbl.LookupPrefixLPM(worstCaseProbePfx6)
		if ok != want {
			t.Errorf("LookupPrefixLPM, worst case match IP6 pfx, expected OK: %v, got: %v", want, ok)
		}
		if val != wantVal {
			t.Errorf("LookupPrefixLPM, worst case match IP6 pfx, expected: %v, got: %v", wantVal, val)
		}
		if lpm != wantLPM {
			t.Errorf("LookupPrefixLPM, worst case match IP6 pfx, expected: %v, got: %v", wantLPM, lpm)
		}
	})
}

func TestWorstCaseMiss6(t *testing.T) {
	t.Parallel()

	t.Run("Contains", func(t *testing.T) {
		t.Parallel()

		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx6) // delete matching prefix

		want := false
		ok := tbl.Contains(worstCaseProbeIP6)
		if ok != want {
			t.Errorf("Contains, worst case miss IP6, expected OK: %v, got: %v", want, ok)
		}
	})

	t.Run("Lookup", func(t *testing.T) {
		t.Parallel()

		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx6)

		want := false
		_, ok := tbl.Lookup(worstCaseProbeIP6)
		if ok != want {
			t.Errorf("Lookup, worst case miss IP6, expected OK: %v, got: %v", want, ok)
		}
	})

	t.Run("LookupPrefix", func(t *testing.T) {
		t.Parallel()

		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx6)

		want := false
		_, ok := tbl.LookupPrefix(worstCaseProbePfx6)
		if ok != want {
			t.Errorf("LookupPrefix, worst case miss IP6 pfx, expected OK: %v, got: %v", want, ok)
		}
	})

	t.Run("LookupPfxLPM", func(t *testing.T) {
		t.Parallel()

		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx6)

		want := false
		_, _, ok := tbl.LookupPrefixLPM(worstCaseProbePfx6)
		if ok != want {
			t.Errorf("LookupPrefixLPM, worst case miss IP6 pfx, expected OK: %v, got: %v", want, ok)
		}
	})
}

func BenchmarkWorstCaseMatch4(b *testing.B) {
	b.Run("Contains", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}

		b.ResetTimer()
		for range b.N {
			boolSink = tbl.Contains(worstCaseProbeIP4)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}
		tbl.Insert(ipv4DefaultRoute, ipv4DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx4)

		b.ResetTimer()
		for range b.N {
			_, boolSink = tbl.Lookup(worstCaseProbeIP4)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}
		tbl.Insert(ipv4DefaultRoute, ipv4DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx4)

		b.ResetTimer()
		for range b.N {
			_, boolSink = tbl.LookupPrefix(worstCaseProbePfx4)
		}
	})

	b.Run("LookupPfxLPM", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}
		tbl.Insert(ipv4DefaultRoute, ipv4DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx4)

		b.ResetTimer()
		for range b.N {
			_, _, boolSink = tbl.LookupPrefixLPM(worstCaseProbePfx4)
		}
	})
}

func BenchmarkWorstCaseMiss4(b *testing.B) {
	b.Run("Contains", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx4) // delete matching prefix

		b.ResetTimer()
		for range b.N {
			boolSink = tbl.Contains(worstCaseProbeIP4)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx4) // delete matching prefix

		b.ResetTimer()
		for range b.N {
			_, boolSink = tbl.Lookup(worstCaseProbeIP4)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx4) // delete matching prefix

		b.ResetTimer()
		for range b.N {
			_, boolSink = tbl.LookupPrefix(worstCaseProbePfx4)
		}
	})

	b.Run("LookupPfxLPM", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx4) // delete matching prefix

		b.ResetTimer()
		for range b.N {
			_, _, boolSink = tbl.LookupPrefixLPM(worstCaseProbePfx4)
		}
	})
}

func BenchmarkWorstCaseMatch6(b *testing.B) {
	b.Run("Contains", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}

		b.ResetTimer()
		for range b.N {
			boolSink = tbl.Contains(worstCaseProbeIP6)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}
		tbl.Insert(ipv6DefaultRoute, ipv6DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx6)

		b.ResetTimer()
		for range b.N {
			_, boolSink = tbl.Lookup(worstCaseProbeIP6)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}
		tbl.Insert(ipv6DefaultRoute, ipv6DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx6)

		b.ResetTimer()
		for range b.N {
			_, boolSink = tbl.LookupPrefix(worstCaseProbePfx6)
		}
	})

	b.Run("LookupPfxLPM", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}
		tbl.Insert(ipv6DefaultRoute, ipv6DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx6)

		b.ResetTimer()
		for range b.N {
			_, _, boolSink = tbl.LookupPrefixLPM(worstCaseProbePfx6)
		}
	})
}

func BenchmarkWorstCaseMiss6(b *testing.B) {
	b.Run("Contains", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx6) // delete matching prefix

		b.ResetTimer()
		for range b.N {
			boolSink = tbl.Contains(worstCaseProbeIP6)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx6) // delete matching prefix

		b.ResetTimer()
		for range b.N {
			_, boolSink = tbl.Lookup(worstCaseProbeIP6)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx6) // delete matching prefix

		b.ResetTimer()
		for range b.N {
			_, boolSink = tbl.LookupPrefix(worstCaseProbePfx6)
		}
	})

	b.Run("LookupPfxLPM", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx6) // delete matching prefix

		b.ResetTimer()
		for range b.N {
			_, _, boolSink = tbl.LookupPrefixLPM(worstCaseProbePfx6)
		}
	})
}
