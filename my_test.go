package bart

import (
	"math/rand/v2"
	"testing"
)

func TestMy(t *testing.T) {
	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, 10_000)

	gold := new(goldTable[int]).insertMany(pfxs)
	fast := new(Dart[int])

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	for range 10_000 {
		a := randomAddr(prng)

		_, goldOK := gold.lookup(a)
		fastOK := fast.Contains(a)

		if goldOK != fastOK {
			t.Fatalf("Contains(%q) = %v, want %v", a, fastOK, goldOK)
		}
	}
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
