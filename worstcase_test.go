package bart

import (
	"net/netip"
	"testing"
)

var (
	worstCaseProbeIP4 = mpa("255.255.255.255")

	worstCasePfxsIP4 = []netip.Prefix{
		mpp("0.0.0.0/1"),
		mpp("255.0.0.0/9"),
		mpp("255.255.0.0/17"),
		mpp("255.255.255.0/25"),
		mpp("255.255.255.255/32"), // matching prefix
	}

	worstCaseProbeIP6 = mpa("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff")

	worstCasePfxsIP6 = []netip.Prefix{
		mpp("::/1"),
		mpp("ff00::/9"),
		mpp("ffff::/17"),
		mpp("ffff:ff00::/25"),
		mpp("ffff:ffff::/33"),
		mpp("ffff:ffff:ff00::/41"),
		mpp("ffff:ffff:ffff::/49"),
		mpp("ffff:ffff:ffff:ff00::/57"),
		mpp("ffff:ffff:ffff:ffff::/65"),
		mpp("ffff:ffff:ffff:ffff:ff00::/73"),
		mpp("ffff:ffff:ffff:ffff:ffff::/81"),
		mpp("ffff:ffff:ffff:ffff:ffff:ff00::/89"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff::/97"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:ff00::/105"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:ffff::/113"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ff00/121"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:ffff:fffe/128"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff/128"),
	}
)

func TestWorstCase(t *testing.T) {
	t.Parallel()

	t.Run("WorstCaseMatchIP4", func(t *testing.T) {
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

	t.Run("WorstCaseMissIP4", func(t *testing.T) {
		t.Parallel()

		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(mpp("255.255.255.255/32")) // delete matching prefix

		want := false
		ok := tbl.Contains(worstCaseProbeIP4)
		if ok != want {
			t.Errorf("Contains, worst case miss IP4, expected OK: %v, got: %v", want, ok)
		}
	})

	t.Run("WorstCaseMatchIP6", func(t *testing.T) {
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

	t.Run("WorstCaseMissIP6", func(t *testing.T) {
		t.Parallel()

		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(mpp("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff/128")) // delete matching prefix

		want := false
		ok := tbl.Contains(worstCaseProbeIP6)
		if ok != want {
			t.Errorf("Contains, worst case miss IP6, expected OK: %v, got: %v", want, ok)
		}
	})
}

func BenchmarkWorstCase(b *testing.B) {
	b.Run("WorstCaseMatchIP4", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}

		for range b.N {
			_ = tbl.Contains(worstCaseProbeIP4)
		}
	})

	b.Run("WorstCaseMatchIP6", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}

		for range b.N {
			_ = tbl.Contains(worstCaseProbeIP6)
		}
	})

	b.Run("WorstCaseMissIP4", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(mpp("255.255.255.255/32")) // delete matching prefix

		for range b.N {
			_ = tbl.Contains(worstCaseProbeIP4)
		}
	})

	b.Run("WorstCaseMissIP6", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(mpp("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff/128")) // delete matching prefix

		for range b.N {
			_ = tbl.Contains(worstCaseProbeIP6)
		}
	})
}
