// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"testing"
)

func TestFastWorstCaseMatch4(t *testing.T) {
	t.Parallel()

	t.Run("Contains", func(t *testing.T) {
		t.Parallel()

		tbl := new(Fast[string])
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

		tbl := new(Fast[string])
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
}

func TestFastWorstCaseMiss4(t *testing.T) {
	t.Parallel()

	t.Run("Contains", func(t *testing.T) {
		t.Parallel()

		tbl := new(Fast[string])
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

		tbl := new(Fast[string])
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
}

func TestFastWorstCaseMatch6(t *testing.T) {
	t.Parallel()

	t.Run("Contains", func(t *testing.T) {
		t.Parallel()

		tbl := new(Fast[string])
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

		tbl := new(Fast[string])
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
}

func TestFastWorstCaseMiss6(t *testing.T) {
	t.Parallel()

	t.Run("Contains", func(t *testing.T) {
		t.Parallel()

		tbl := new(Fast[string])
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

		tbl := new(Fast[string])
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
}

func BenchmarkFastWorstCaseMatch4(b *testing.B) {
	b.Run("Contains", func(b *testing.B) {
		tbl := new(Fast[string])

		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}

		for b.Loop() {
			tbl.Contains(worstCaseProbeIP4)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		tbl := new(Fast[string])

		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}
		tbl.Insert(ipv4DefaultRoute, ipv4DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx4)

		for b.Loop() {
			tbl.Lookup(worstCaseProbeIP4)
		}
	})
}

func BenchmarkFastWorstCaseMiss4(b *testing.B) {
	b.Run("Contains", func(b *testing.B) {
		tbl := new(Fast[string])

		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx4) // delete matching prefix

		for b.Loop() {
			tbl.Contains(worstCaseProbeIP4)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		tbl := new(Fast[string])

		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx4) // delete matching prefix

		for b.Loop() {
			tbl.Lookup(worstCaseProbeIP4)
		}
	})
}

func BenchmarkFastWorstCaseMatch6(b *testing.B) {
	b.Run("Contains", func(b *testing.B) {
		tbl := new(Fast[string])

		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}

		for b.Loop() {
			tbl.Contains(worstCaseProbeIP6)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		tbl := new(Fast[string])

		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}
		tbl.Insert(ipv6DefaultRoute, ipv6DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx6)

		for b.Loop() {
			tbl.Lookup(worstCaseProbeIP6)
		}
	})
}

func BenchmarkFastWorstCaseMiss6(b *testing.B) {
	b.Run("Contains", func(b *testing.B) {
		tbl := new(Fast[string])

		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx6) // delete matching prefix

		for b.Loop() {
			tbl.Contains(worstCaseProbeIP6)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		tbl := new(Fast[string])

		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx6) // delete matching prefix

		for b.Loop() {
			tbl.Lookup(worstCaseProbeIP6)
		}
	})
}
