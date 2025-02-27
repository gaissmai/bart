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

		b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc), "Bytes")
		b.ReportMetric(0, "ns/op")
	})
}

func BenchmarkLiteFullTableMemoryV6(b *testing.B) {
	var startMem, endMem runtime.MemStats

	rt := new(Lite)
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	b.Run(strconv.Itoa(len(routes6)), func(b *testing.B) {
		for range b.N {
			for _, route := range routes6 {
				rt.Insert(route.CIDR)
			}
		}

		runtime.GC()
		runtime.ReadMemStats(&endMem)

		b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc), "Bytes")
		b.ReportMetric(0, "ns/op")
	})
}

func BenchmarkLiteFullTableMemory(b *testing.B) {
	var startMem, endMem runtime.MemStats

	rt := new(Lite)
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	b.Run(strconv.Itoa(len(routes)), func(b *testing.B) {
		for range b.N {
			for _, route := range routes {
				rt.Insert(route.CIDR)
			}
		}

		runtime.GC()
		runtime.ReadMemStats(&endMem)

		b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc), "Bytes")
		b.ReportMetric(0, "ns/op")
	})
}
