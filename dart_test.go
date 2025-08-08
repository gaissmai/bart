package bart

import (
	"math/rand/v2"
	"testing"
)

func TestDartContainsCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, 10_000)

	bart := new(Table[int])
	dart := new(Dart[int])

	for _, pfx := range pfxs {
		bart.Insert(pfx.pfx, pfx.val)
		dart.Insert(pfx.pfx, pfx.val)
	}

	if bart.Size() != dart.Size() {
		t.Errorf("Size() %d != %d", bart.Size(), dart.Size())
	}

	for range 100_000 {
		a := randomAddr(prng)

		bartOk := bart.Contains(a)
		dartOk := dart.Contains(a)

		if bartOk != dartOk {
			t.Fatalf("Contains(%q) = %v, want %v", a, dartOk, bartOk)
		}
	}
}

func TestDartLookupCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, 10_000)

	bart := new(Table[int])
	dart := new(Dart[int])

	for _, pfx := range pfxs {
		bart.Insert(pfx.pfx, pfx.val)
		dart.Insert(pfx.pfx, pfx.val)
	}

	if bart.Size() != dart.Size() {
		t.Errorf("Size() %d != %d", bart.Size(), dart.Size())
	}

	for range 100_000 {
		a := randomAddr(prng)

		bartVal, bartOk := bart.Lookup(a)
		dartVal, dartOk := dart.Lookup(a)

		if bartOk != dartOk {
			t.Fatalf("Lookup(%q) = %v, want %v", a, dartOk, bartOk)
		}

		if bartOk == dartOk && bartVal != dartVal {
			t.Fatalf("Lookup(%q) = %v, want %v", a, dartVal, bartVal)
		}
	}
}

func BenchmarkDartContains(b *testing.B) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	prng := rand.New(rand.NewPCG(42, 24))

	bart := new(Table[int])
	dart := new(Dart[int])

	for i, r := range routes {
		bart.Insert(r.CIDR, i)
		dart.Insert(r.CIDR, i)
	}

	ip := randomIP4(prng)
	ip = mpa("134.60.1.1")

	b.Run("DART: 134.60.1.1", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			boolSink = dart.Contains(ip)
		}
	})

	b.Run("BART: 134.60.1.1", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			boolSink = bart.Contains(ip)
		}
	})

	ip = mpa("2001:7c0:3100::1")

	b.Run("DART: 2001:7c0:3100::1", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			boolSink = dart.Contains(ip)
		}
	})

	b.Run("BART: 2001:7c0:3100::1", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			boolSink = bart.Contains(ip)
		}
	})
}

func BenchmarkDartLookup(b *testing.B) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	prng := rand.New(rand.NewPCG(42, 24))

	bart := new(Table[int])
	dart := new(Dart[int])

	for i, r := range routes {
		bart.Insert(r.CIDR, i)
		dart.Insert(r.CIDR, i)
	}

	ip := randomIP4(prng)
	ip = mpa("134.60.1.1")

	b.Run("DART: 134.60.1.1", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			_, boolSink = dart.Lookup(ip)
		}
	})

	b.Run("BART: 134.60.1.1", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			_, boolSink = bart.Lookup(ip)
		}
	})

	ip = mpa("2001:7c0:3100::1")

	b.Run("DART: 2001:7c0:3100::1", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			_, boolSink = dart.Lookup(ip)
		}
	})

	b.Run("BART: 2001:7c0:3100::1", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			_, boolSink = bart.Lookup(ip)
		}
	})
}

func BenchmarkDartFullMatch4(b *testing.B) {
	rt := new(Dart[struct{}])

	for _, route := range routes {
		rt.Insert(route.CIDR, struct{}{})
	}

	b.Log(matchIP4)
	b.Log(matchPfx4)

	b.Run("Contains", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			boolSink = rt.Contains(matchIP4)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			_, boolSink = rt.Lookup(matchIP4)
		}
	})
}

func BenchmarkDartFullMatch6(b *testing.B) {
	rt := new(Dart[struct{}])

	for _, route := range routes {
		rt.Insert(route.CIDR, struct{}{})
	}

	b.Log(matchIP6)
	b.Log(matchPfx6)

	b.Run("Contains", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			boolSink = rt.Contains(matchIP6)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			_, boolSink = rt.Lookup(matchIP6)
		}
	})
}

func BenchmarkDartFullMiss4(b *testing.B) {
	rt := new(Dart[int])

	for i, route := range routes {
		rt.Insert(route.CIDR, i)
	}

	b.Log(missIP4)
	b.Log(missPfx4)

	b.Run("Contains", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			boolSink = rt.Contains(missIP4)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			intSink, boolSink = rt.Lookup(missIP4)
		}
	})
}

func BenchmarkDartFullMiss6(b *testing.B) {
	rt := new(Dart[int])

	for i, route := range routes {
		rt.Insert(route.CIDR, i)
	}

	b.Log(missIP6)
	b.Log(missPfx6)

	b.Run("Contains", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			boolSink = rt.Contains(missIP6)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			intSink, boolSink = rt.Lookup(missIP6)
		}
	})
}
