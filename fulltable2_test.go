// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"testing"
)

func BenchmarkFull2MatchV4(b *testing.B) {
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

	b.Run("Lookup1", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			_, okSink = rt1.Lookup(ip)
		}
		b.ReportMetric(float64(rt1.numNodes()), "Nodes")
	})

	b.Run("Lookup2", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			_, okSink = rt2.Lookup(ip)
		}
		b.ReportMetric(float64(rt2.numNodes()), "Nodes")
	})
}

func BenchmarkFull2MatchV6(b *testing.B) {
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

	b.Run("Lookup1", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			_, okSink = rt1.Lookup(ip)
		}
		b.ReportMetric(float64(rt1.numNodes()), "Nodes")
	})

	b.Run("Lookup2", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			intSink, okSink = rt2.Lookup(ip)
		}
		b.ReportMetric(float64(rt2.numNodes()), "Nodes")
	})
}

func BenchmarkFull2MissV4(b *testing.B) {
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

	b.Run("Lookup1", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			_, okSink = rt1.Lookup(ip)
		}
		b.ReportMetric(float64(rt1.numNodes()), "Nodes")
	})

	b.Run("Lookup2", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			_, okSink = rt2.Lookup(ip)
		}
		b.ReportMetric(float64(rt2.numNodes()), "Nodes")
	})
}

func BenchmarkFull2MissV6(b *testing.B) {
	var rt1 Table[int]
	var rt2 Table2[int]

	for i, route := range routes {
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

	b.Run("Lookup1", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			_, okSink = rt1.Lookup(ip)
		}
		b.ReportMetric(float64(rt1.numNodes()), "Nodes")
	})

	b.Run("Lookup2", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			_, okSink = rt2.Lookup(ip)
		}
		b.ReportMetric(float64(rt2.numNodes()), "Nodes")
	})
}
