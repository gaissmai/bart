package bart

import (
	"testing"
)

func TestMyMatch(t *testing.T) {
	lt := new(Lite)
	rt := new(Table[string])

	for _, p := range worstCasePfxsIP4 {
		lt.Insert(p)
		rt.Insert(p, p.String())
	}

	if lt.Contains(worstCaseProbeIP4) != rt.Contains(worstCaseProbeIP4) {
		t.Error("Lite and Table differs in Contains()")
	}

	t.Log(rt)
	t.Log(rt.dumpString())
}

func TestMyMiss(t *testing.T) {
	lt := new(Lite)
	rt := new(Table[string])

	for _, p := range worstCasePfxsIP4 {
		lt.Insert(p)
		rt.Insert(p, p.String())
	}
	lt.Delete(worstCaseProbePfx4)
	rt.Delete(worstCaseProbePfx4)

	if lt.Contains(worstCaseProbeIP4) != rt.Contains(worstCaseProbeIP4) {
		t.Error("Lite and Table differs in Contains()")
	}

	t.Log(rt)
	t.Log(rt.dumpString())
}

func BenchmarkMyMatch(b *testing.B) {
	lt := new(Lite)
	rt := new(Table[any])

	for _, p := range worstCasePfxsIP4 {
		lt.Insert(p)
		rt.Insert(p, nil)
	}

	b.Run("Table", func(b *testing.B) {
		for range b.N {
			_ = rt.Contains(worstCaseProbeIP4)
		}
	})

	b.Run("Lite", func(b *testing.B) {
		for range b.N {
			_ = lt.Contains(worstCaseProbeIP4)
		}
	})
}

func BenchmarkMyMiss(b *testing.B) {
	lt := new(Lite)
	rt := new(Table[any])

	for _, p := range worstCasePfxsIP4 {
		lt.Insert(p)
		rt.Insert(p, nil)
	}
	lt.Delete(worstCaseProbePfx4)
	rt.Delete(worstCaseProbePfx4)

	b.Run("Table", func(b *testing.B) {
		for range b.N {
			_ = rt.Contains(worstCaseProbeIP4)
		}
	})

	b.Run("Lite", func(b *testing.B) {
		for range b.N {
			_ = lt.Contains(worstCaseProbeIP4)
		}
	})
}
