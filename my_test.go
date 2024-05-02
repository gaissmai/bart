package bart

import (
	"net/netip"
	"runtime"
	"testing"
)

var (
	pfxs = randomPrefixes(1_000_000)
	pfx4 = gimmeRandomPrefix4(10)[7]
	pfx6 = gimmeRandomPrefix6(10)[7]
)

func BenchmarkMyInsert(b *testing.B) {
	rt1 := new(Table[struct{}])
	rt2 := new(Table2[struct{}])

	for _, route := range pfxs {
		rt1.Insert(route.pfx, struct{}{})
		rt2.Insert(route.pfx, struct{}{})
	}

	b.Run("v4-pfx/orig/random 1_000_000 pfxs", func(b *testing.B) {
		for range b.N {
			rt1.Insert(pfx4, struct{}{})
		}
		b.ReportMetric(float64(rt1.numNodes()), "Nodes")
	})

	b.Run("v4-pfx/comp/random 1_000_000 pfxs", func(b *testing.B) {
		for range b.N {
			rt2.Insert(pfx4, struct{}{})
		}
		b.ReportMetric(float64(rt2.numNodes()), "Nodes")
	})

	b.Run("v6-pfx/orig/random 1_000_000 pfxs", func(b *testing.B) {
		for range b.N {
			rt1.Insert(pfx6, struct{}{})
		}
		b.ReportMetric(float64(rt1.numNodes()), "Nodes")
	})

	b.Run("v6-pfx/comp/random 1_000_000 pfxs", func(b *testing.B) {
		for range b.N {
			rt2.Insert(pfx6, struct{}{})
		}
		b.ReportMetric(float64(rt2.numNodes()), "Nodes")
	})
}

func BenchmarkMyLookupV4(b *testing.B) {
	var rt1 Table[int]
	var rt2 Table2[int]

	for i, route := range randomPrefixes4(1_000_000) {
		rt1.Insert(route.pfx, i)
		rt2.Insert(route.pfx, i)
	}

	ip := randomIP4()

	b.Run("orig/random/1_000_000", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			intSink, okSink = rt1.Lookup(ip)
		}
		b.ReportMetric(float64(rt1.numNodes()), "Nodes")
	})

	b.Run("comp/random/1_000_000", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			intSink, okSink = rt2.Lookup(ip)
		}
		b.ReportMetric(float64(rt2.numNodes()), "Nodes")
	})
}

func BenchmarkMyLookupV6(b *testing.B) {
	var rt1 Table[int]
	var rt2 Table2[int]

	for i, route := range randomPrefixes6(1_000_000) {
		rt1.Insert(route.pfx, i)
		rt2.Insert(route.pfx, i)
	}

	ip := randomIP6()

	b.Run("orig/random/1_000_000", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			intSink, okSink = rt1.Lookup(ip)
		}
		b.ReportMetric(float64(rt1.numNodes()), "Nodes")
	})

	b.Run("comp/random/1_000_000", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			intSink, okSink = rt2.Lookup(ip)
		}
		b.ReportMetric(float64(rt2.numNodes()), "Nodes")
	})
}

func BenchmarkMyFullMatchV4(b *testing.B) {
	var rt1 Table[int]
	var rt2 Table2[int]

	for i, route := range routes {
		rt1.Insert(route.CIDR, i)
		rt2.Insert(route.CIDR, i)
	}

	var ip netip.Addr

	// find a random match
	for {
		ip = randomIP4()
		_, ok1 := rt1.Lookup(ip)
		_, ok2 := rt2.Lookup(ip)
		if ok1 != ok2 {
			b.Errorf("ip: %s, ok1 %v, ok2 %v", ip, ok1, ok2)
		}
		if ok1 {
			break
		}
	}

	b.Run("orig", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			_, okSink = rt1.Lookup(ip)
		}
		b.ReportMetric(float64(rt1.numNodes()), "Nodes")
	})

	b.Run("comp", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			_, okSink = rt2.Lookup(ip)
		}
		b.ReportMetric(float64(rt2.numNodes()), "Nodes")
	})
}

func BenchmarkMyFullMatchV6(b *testing.B) {
	var rt1 Table[int]
	var rt2 Table2[int]

	for i, route := range routes {
		rt1.Insert(route.CIDR, i)
		rt2.Insert(route.CIDR, i)
	}

	var ip netip.Addr

	// find a random match
	for {
		ip = randomIP6()
		_, ok1 := rt1.Lookup(ip)
		_, ok2 := rt2.Lookup(ip)
		if ok1 != ok2 {
			b.Errorf("ip: %s, ok1 %v, ok2 %v", ip, ok1, ok2)
		}
		if ok1 {
			break
		}
	}

	b.Run("orig", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			_, okSink = rt1.Lookup(ip)
		}
		b.ReportMetric(float64(rt1.numNodes()), "Nodes")
	})

	b.Run("comp", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			intSink, okSink = rt2.Lookup(ip)
		}
		b.ReportMetric(float64(rt2.numNodes()), "Nodes")
	})
}

func BenchmarkMyFullMissV4(b *testing.B) {
	var rt1 Table[int]
	var rt2 Table2[int]

	for i, route := range routes {
		rt1.Insert(route.CIDR, i)
		rt2.Insert(route.CIDR, i)
	}

	var ip netip.Addr

	// find a random match
	for {
		ip = randomIP4()
		_, ok := rt2.Lookup(ip)
		if !ok {
			break
		}
	}

	b.Run("orig", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			_, okSink = rt1.Lookup(ip)
		}
		b.ReportMetric(float64(rt1.numNodes()), "Nodes")
	})

	b.Run("comp", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			_, okSink = rt2.Lookup(ip)
		}
		b.ReportMetric(float64(rt2.numNodes()), "Nodes")
	})
}

func BenchmarkMyFullMissV6(b *testing.B) {
	var rt1 Table[int]
	var rt2 Table2[int]

	for i, route := range routes {
		rt1.Insert(route.CIDR, i)
		rt2.Insert(route.CIDR, i)
	}

	var ip netip.Addr

	// find a random match
	for {
		ip = randomIP6()
		_, ok := rt2.Lookup(ip)
		if !ok {
			break
		}
	}

	b.Run("orig", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			_, okSink = rt1.Lookup(ip)
		}
		b.ReportMetric(float64(rt1.numNodes()), "Nodes")
	})

	b.Run("comp", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			_, okSink = rt2.Lookup(ip)
		}
		b.ReportMetric(float64(rt2.numNodes()), "Nodes")
	})
}

func BenchmarkMyFullSize(b *testing.B) {
	var startMem, endMem runtime.MemStats

	b.Run("orig", func(b *testing.B) {
		rt1 := new(Table[any])

		for range b.N {
			rt1 = new(Table[any])
			runtime.GC()
			runtime.ReadMemStats(&startMem)

			for _, route := range routes {
				rt1.Insert(route.CIDR, struct{}{})
			}

			runtime.GC()
			runtime.ReadMemStats(&endMem)

			if npfx := rt1.numPrefixes(); npfx != len(routes) {
				b.Fatalf("expect %v prefixes, got %v", len(routes), npfx)
			}

			b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc), "Bytes")
			b.ReportMetric(float64(rt1.numNodes()), "Nodes")
			b.ReportMetric(0, "ns/op") // silence
		}
	})

	b.Run("comp", func(b *testing.B) {
		rt2 := new(Table2[any])

		for range b.N {
			rt2 = new(Table2[any])
			runtime.GC()
			runtime.ReadMemStats(&startMem)

			for _, route := range routes {
				rt2.Insert(route.CIDR, struct{}{})
			}

			runtime.GC()
			runtime.ReadMemStats(&endMem)

			if npfx := rt2.numPrefixes(); npfx != len(routes) {
				b.Fatalf("expect %v prefixes, got %v", len(routes), npfx)
			}

			b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc), "Bytes")
			b.ReportMetric(float64(rt2.numNodes()), "Nodes")
			b.ReportMetric(0, "ns/op") // silence
		}
	})
}
