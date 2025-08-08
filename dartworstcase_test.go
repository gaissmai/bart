package bart

import (
	"testing"
)

func TestDartWorstCaseMatch4(t *testing.T) {
	t.Parallel()

	t.Run("Contains", func(t *testing.T) {
		t.Parallel()

		tbl := new(Dart[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}

		want := true
		ok := tbl.Contains(worstCaseProbeIP4)
		if ok != want {
			t.Errorf("Contains, worst case match IP4, expected OK: %v, got: %v", want, ok)
		}
	})
}

/*
func TestDartWorstCaseMiss4(t *testing.T) {
	t.Parallel()

	t.Run("Contains", func(t *testing.T) {
		t.Parallel()

		tbl := new(Dart[string])
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
}
*/

func TestDartWorstCaseMatch6(t *testing.T) {
	t.Parallel()

	t.Run("Contains", func(t *testing.T) {
		t.Parallel()

		tbl := new(Dart[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}

		want := true
		ok := tbl.Contains(worstCaseProbeIP6)
		if ok != want {
			t.Errorf("Contains, worst case match IP6, expected OK: %v, got: %v", want, ok)
		}
	})
}

/* TODO Delete
func TestDartWorstCaseMiss6(t *testing.T) {
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
}
*/

func BenchmarkDartWorstCaseMatch4(b *testing.B) {
	b.Run("Contains", func(b *testing.B) {
		tbl := new(Dart[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}

		b.ResetTimer()
		for range b.N {
			boolSink = tbl.Contains(worstCaseProbeIP4)
		}
	})
}

/*
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
*/

func BenchmarkDartWorstCaseMatch6(b *testing.B) {
	b.Run("Contains", func(b *testing.B) {
		tbl := new(Dart[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}

		b.ResetTimer()
		for range b.N {
			boolSink = tbl.Contains(worstCaseProbeIP6)
		}
	})
}

/*
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
*/
