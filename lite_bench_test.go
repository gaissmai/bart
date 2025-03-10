package bart

import (
	"net/netip"
	"runtime"
	"strconv"
	"testing"
)

func BenchmarkLiteFullMatchV4(b *testing.B) {
	lt := new(Lite)

	for _, route := range routes {
		lt.Insert(route.CIDR)
	}

	var ip netip.Addr

	// find a random match
	for {
		ip = randomIP4()
		if lt.Contains(ip) {
			break
		}
	}

	b.Run("Contains", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			okSink = lt.Contains(ip)
		}
	})
}

func BenchmarkLiteFullMatchV6(b *testing.B) {
	lt := new(Lite)

	for _, route := range routes {
		lt.Insert(route.CIDR)
	}

	var ip netip.Addr

	// find a random match
	for {
		ip = randomIP6()
		if lt.Contains(ip) {
			break
		}
	}

	b.Run("Contains", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			okSink = lt.Contains(ip)
		}
	})
}

func BenchmarkLiteFullMissV4(b *testing.B) {
	lt := new(Lite)

	for _, route := range routes {
		lt.Insert(route.CIDR)
	}

	var ip netip.Addr

	// find a random miss
	for {
		ip = randomIP4()
		if !lt.Contains(ip) {
			break
		}
	}

	b.Run("Contains", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			okSink = lt.Contains(ip)
		}
	})
}

func BenchmarkLiteFullMissV6(b *testing.B) {
	lt := new(Lite)

	for _, route := range routes {
		lt.Insert(route.CIDR)
	}

	var ip netip.Addr

	// find a random miss
	for {
		ip = randomIP6()
		if !lt.Contains(ip) {
			break
		}
	}

	b.Run("Contains", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			okSink = lt.Contains(ip)
		}
	})
}

func BenchmarkLiteFullTableMemoryV4(b *testing.B) {
	var startMem, endMem runtime.MemStats

	lite := new(Lite)
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	b.Run(strconv.Itoa(len(routes4)), func(b *testing.B) {
		for range b.N {
			for _, route := range routes4 {
				lite.Insert(route.CIDR)
			}
		}

		runtime.GC()
		runtime.ReadMemStats(&endMem)

		b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc)/1024, "KByte")
		b.ReportMetric(0, "ns/op")
	})
}

func BenchmarkLiteFullTableMemoryV6(b *testing.B) {
	var startMem, endMem runtime.MemStats

	lite := new(Lite)
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	b.Run(strconv.Itoa(len(routes6)), func(b *testing.B) {
		for range b.N {
			for _, route := range routes6 {
				lite.Insert(route.CIDR)
			}
		}

		runtime.GC()
		runtime.ReadMemStats(&endMem)

		b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc)/1024, "KByte")
		b.ReportMetric(0, "ns/op")
	})
}

func BenchmarkLiteFullTableMemory(b *testing.B) {
	var startMem, endMem runtime.MemStats

	lite := new(Lite)
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	b.Run(strconv.Itoa(len(routes)), func(b *testing.B) {
		for range b.N {
			for _, route := range routes {
				lite.Insert(route.CIDR)
			}
		}

		runtime.GC()
		runtime.ReadMemStats(&endMem)

		b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc)/1024, "KByte")
		b.ReportMetric(0, "ns/op")
	})
}

func BenchmarkLiteRealWorldRandomPfxsMemoryV4(b *testing.B) {
	for _, k := range []int{1_000, 10_000, 100_000, 1_000_000} {
		var startMem, endMem runtime.MemStats

		runtime.GC()
		runtime.ReadMemStats(&startMem)

		b.Run(strconv.Itoa(k), func(b *testing.B) {
			lite := new(Lite)
			for range b.N {
				lite = new(Lite)
				for _, pfx := range randomRealWorldPrefixes4(k) {
					lite.Insert(pfx)
				}
			}

			runtime.GC()
			runtime.ReadMemStats(&endMem)

			stats := lite.root4.nodeStatsRec()
			b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc), "Bytes")
			b.ReportMetric(float64(stats.nodes), "nodes")
			b.ReportMetric(float64(stats.pfxs), "pfxs")
			b.ReportMetric(float64(stats.leaves), "leaves")
			b.ReportMetric(float64(stats.fringes), "fringes")
			b.ReportMetric(0, "ns/op")
		})
	}
}

func BenchmarkLiteRealWorldRandomPfxsMemoryV6(b *testing.B) {
	for _, k := range []int{1_000, 10_000, 100_000, 1_000_000} {
		var startMem, endMem runtime.MemStats

		runtime.GC()
		runtime.ReadMemStats(&startMem)

		b.Run(strconv.Itoa(k), func(b *testing.B) {
			lite := new(Lite)
			for range b.N {
				lite = new(Lite)
				for _, pfx := range randomRealWorldPrefixes6(k) {
					lite.Insert(pfx)
				}
			}

			runtime.GC()
			runtime.ReadMemStats(&endMem)

			stats := lite.root6.nodeStatsRec()
			b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc), "Bytes")
			b.ReportMetric(float64(stats.nodes), "nodes")
			b.ReportMetric(float64(stats.pfxs), "pfxs")
			b.ReportMetric(float64(stats.leaves), "leaves")
			b.ReportMetric(float64(stats.fringes), "fringes")
			b.ReportMetric(0, "ns/op")
		})
	}
}

func BenchmarkLiteRealWorldRandomPfxsMemory(b *testing.B) {
	for _, k := range []int{1_000, 10_000, 100_000, 1_000_000} {
		var startMem, endMem runtime.MemStats

		runtime.GC()
		runtime.ReadMemStats(&startMem)

		b.Run(strconv.Itoa(k), func(b *testing.B) {
			lite := new(Lite)
			for range b.N {
				lite = new(Lite)
				for _, pfx := range randomRealWorldPrefixes(k) {
					lite.Insert(pfx)
				}
			}

			runtime.GC()
			runtime.ReadMemStats(&endMem)

			stats4 := lite.root4.nodeStatsRec()
			stats6 := lite.root6.nodeStatsRec()
			b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc), "Bytes")
			b.ReportMetric(float64(stats4.nodes+stats6.nodes), "nodes")
			b.ReportMetric(float64(stats4.pfxs+stats6.pfxs), "pfxs")
			b.ReportMetric(float64(stats4.leaves+stats6.leaves), "leaves")
			b.ReportMetric(float64(stats4.fringes+stats6.fringes), "fringes")
			b.ReportMetric(0, "ns/op")
		})
	}
}
