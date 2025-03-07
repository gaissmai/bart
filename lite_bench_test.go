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

		stats := lite.root4.nodeStatsRec()
		b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc), "Bytes")
		b.ReportMetric(float64(stats.pfxs), "pfxs")
		b.ReportMetric(float64(stats.nodes), "nodes")
		b.ReportMetric(float64(stats.leaves), "leaves")
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

		stats := lite.root6.nodeStatsRec()
		b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc), "Bytes")
		b.ReportMetric(float64(stats.pfxs), "pfxs")
		b.ReportMetric(float64(stats.nodes), "nodes")
		b.ReportMetric(float64(stats.leaves), "leaves")
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

		s4 := lite.root4.nodeStatsRec()
		s6 := lite.root6.nodeStatsRec()
		stats := stats{
			s4.pfxs + s6.pfxs,
			s4.childs + s6.childs,
			s4.nodes + s6.nodes,
			s4.leaves + s6.leaves,
			s4.fringes + s6.fringes,
		}

		b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc), "Bytes")
		b.ReportMetric(float64(stats.pfxs), "pfxs")
		b.ReportMetric(float64(stats.nodes), "nodes")
		b.ReportMetric(float64(stats.leaves), "leaves")
		b.ReportMetric(0, "ns/op")
	})
}
