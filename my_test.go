package bart

import (
	"fmt"
	"math/rand/v2"
	"runtime"
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

func BenchmarkDartFullTableMemory4(b *testing.B) {
	var startMem, endMem runtime.MemStats

	rt := new(Dart[struct{}])
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	b.Run(fmt.Sprintf("Table[]: %d", len(routes4)), func(b *testing.B) {
		for range b.N {
			for _, route := range routes4 {
				rt.Insert(route.CIDR, struct{}{})
			}
		}

		runtime.GC()
		runtime.ReadMemStats(&endMem)

		stats := rt.root4.nodeStatsRec()
		b.ReportMetric(float64(int(endMem.HeapAlloc-startMem.HeapAlloc)/stats.pfxs), "bytes/pfx")
		b.ReportMetric(float64(stats.pfxs), "pfxs")
		b.ReportMetric(float64(stats.nodes), "nodes")
		b.ReportMetric(float64(stats.leaves), "leaves")
		b.ReportMetric(float64(stats.fringes), "fringes")
		b.ReportMetric(0, "ns/op")
	})
}

func BenchmarkDartFullTableMemory6(b *testing.B) {
	var startMem, endMem runtime.MemStats

	rt := new(Dart[struct{}])
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	b.Run(fmt.Sprintf("Table[]: %d", len(routes6)), func(b *testing.B) {
		for range b.N {
			for _, route := range routes6 {
				rt.Insert(route.CIDR, struct{}{})
			}
		}

		runtime.GC()
		runtime.ReadMemStats(&endMem)

		stats := rt.root6.nodeStatsRec()
		b.ReportMetric(float64(int(endMem.HeapAlloc-startMem.HeapAlloc)/stats.pfxs), "bytes/pfx")
		b.ReportMetric(float64(stats.pfxs), "pfxs")
		b.ReportMetric(float64(stats.nodes), "nodes")
		b.ReportMetric(float64(stats.leaves), "leaves")
		b.ReportMetric(float64(stats.fringes), "fringes")
		b.ReportMetric(0, "ns/op")
	})
}

func BenchmarkDartFullTableMemory(b *testing.B) {
	var startMem, endMem runtime.MemStats

	rt := new(Dart[struct{}])
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	b.Run(fmt.Sprintf("Table[]: %d", len(routes)), func(b *testing.B) {
		for range b.N {
			for _, route := range routes {
				rt.Insert(route.CIDR, struct{}{})
			}
		}

		runtime.GC()
		runtime.ReadMemStats(&endMem)

		s4 := rt.root4.nodeStatsRec()
		s6 := rt.root6.nodeStatsRec()
		stats := stats{
			pfxs:    s4.pfxs + s6.pfxs,
			childs:  s4.childs + s6.childs,
			nodes:   s4.nodes + s6.nodes,
			leaves:  s4.leaves + s6.leaves,
			fringes: s4.fringes + s6.fringes,
		}

		b.ReportMetric(float64(int(endMem.HeapAlloc-startMem.HeapAlloc)/stats.pfxs), "bytes/pfx")
		b.ReportMetric(float64(stats.pfxs), "pfxs")
		b.ReportMetric(float64(stats.nodes), "nodes")
		b.ReportMetric(float64(stats.leaves), "leaves")
		b.ReportMetric(float64(stats.fringes), "fringes")
		b.ReportMetric(0, "ns/op")
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
