// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"fmt"
	"net/netip"
	"runtime"
	"testing"
)

func BenchmarkFullLookup2MatchV4(b *testing.B) {
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

func BenchmarkFullLookup2MatchV6(b *testing.B) {
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

func BenchmarkFullLookup2MissV4(b *testing.B) {
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

func BenchmarkFullLookup2MissV6(b *testing.B) {
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

func BenchmarkFull2Size2(b *testing.B) {

	var startMem, endMem runtime.MemStats

	for _, nroutes := range []int{10, 100, 1_000, 10_000, 100_000, 1_000_000} {
		rt1 := new(Table[any])
		rt2 := new(Table2[any])

		b.Run(fmt.Sprintf("orig:%7d", nroutes), func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				rt1 = new(Table[any])
				runtime.GC()
				runtime.ReadMemStats(&startMem)

				for j, route := range routes {
					if j >= nroutes {
						break
					}
					rt1.Insert(route.CIDR, struct{}{})
				}

				runtime.GC()
				runtime.ReadMemStats(&endMem)

				if npfx := rt1.numPrefixes(); npfx != nroutes {
					b.Fatalf("expect %v prefixes, got %v", nroutes, npfx)
				}

				b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc), "Bytes")
				b.ReportMetric(float64(rt1.numNodes()), "Nodes")
				b.ReportMetric(0, "ns/op") // silence
			}
		})

		b.Run(fmt.Sprintf("comp:%7d", nroutes), func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				rt2 = new(Table2[any])
				runtime.GC()
				runtime.ReadMemStats(&startMem)

				for j, route := range routes {
					if j >= nroutes {
						break
					}
					rt2.Insert(route.CIDR, struct{}{})
				}

				runtime.GC()
				runtime.ReadMemStats(&endMem)

				if npfx := rt2.numPrefixes(); npfx != nroutes {
					b.Fatalf("expect %v prefixes, got %v", nroutes, npfx)
				}

				b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc), "Bytes")
				b.ReportMetric(float64(rt2.numNodes()), "Nodes")
				b.ReportMetric(0, "ns/op") // silence
			}
		})
	}
}
