// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"log"
	"math/rand/v2"
	"net/netip"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

var prng = rand.New(rand.NewPCG(42, 42))

// full internet prefix list, gzipped
const prefixFile = "testdata/prefixes.txt.gz"

var (
	routes  []route
	routes4 []route
	routes6 []route

	randRoute4 route
	randRoute6 route
)

type route struct {
	CIDR  netip.Prefix
	Value any
}

func init() {
	fillRouteTables()

	randRoute4 = routes4[prng.IntN(len(routes4))]
	randRoute6 = routes6[prng.IntN(len(routes6))]
}

var (
	intSink  int
	okSink   bool
	boolSink bool
)

func BenchmarkFullMatchV4(b *testing.B) {
	var rt Table[int]

	for i, route := range routes {
		rt.Insert(route.CIDR, i)
	}

	var ip netip.Addr
	var ipAsPfx netip.Prefix

	// find a random match
	for {
		ip = randomIP4()
		_, ok := rt.Lookup(ip)
		if ok {
			ipAsPfx, _ = ip.Prefix(ip.BitLen())
			break
		}
	}

	b.Run("Contains", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			okSink = rt.Contains(ip)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			intSink, okSink = rt.Lookup(ip)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			intSink, okSink = rt.LookupPrefix(ipAsPfx)
		}
	})

	b.Run("LookupPrefixLPM", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			_, intSink, okSink = rt.LookupPrefixLPM(ipAsPfx)
		}
	})
}

func BenchmarkFullMatchV6(b *testing.B) {
	var rt Table[int]

	for i, route := range routes {
		rt.Insert(route.CIDR, i)
	}

	var ip netip.Addr
	var ipAsPfx netip.Prefix

	// find a random match
	for {
		ip = randomIP6()
		_, ok := rt.Lookup(ip)
		if ok {
			ipAsPfx, _ = ip.Prefix(ip.BitLen())
			break
		}
	}

	b.Run("Contains", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			okSink = rt.Contains(ip)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			intSink, okSink = rt.Lookup(ip)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			intSink, okSink = rt.LookupPrefix(ipAsPfx)
		}
	})

	b.Run("LookupPrefixLPM", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			_, intSink, okSink = rt.LookupPrefixLPM(ipAsPfx)
		}
	})
}

func BenchmarkFullMissV4(b *testing.B) {
	var rt Table[int]

	for i, route := range routes {
		rt.Insert(route.CIDR, i)
	}

	var ip netip.Addr
	var ipAsPfx netip.Prefix

	// find a random miss
	for {
		ip = randomIP4()
		_, ok := rt.Lookup(ip)
		if !ok {
			ipAsPfx, _ = ip.Prefix(ip.BitLen())
			break
		}
	}

	b.Run("Contains", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			okSink = rt.Contains(ip)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			intSink, okSink = rt.Lookup(ip)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			intSink, okSink = rt.LookupPrefix(ipAsPfx)
		}
	})

	b.Run("LookupPrefixLPM", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			_, intSink, okSink = rt.LookupPrefixLPM(ipAsPfx)
		}
	})
}

func BenchmarkFullMissV6(b *testing.B) {
	var rt Table[int]

	for i, route := range routes {
		rt.Insert(route.CIDR, i)
	}

	var ip netip.Addr
	var ipAsPfx netip.Prefix

	// find a random miss
	for {
		ip = randomIP6()
		_, ok := rt.Lookup(ip)
		if !ok {
			ipAsPfx, _ = ip.Prefix(ip.BitLen())
			break
		}
	}

	b.Run("Contains", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			okSink = rt.Contains(ip)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			intSink, okSink = rt.Lookup(ip)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			intSink, okSink = rt.LookupPrefix(ipAsPfx)
		}
	})

	b.Run("LookupPrefixLPM", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			_, intSink, okSink = rt.LookupPrefixLPM(ipAsPfx)
		}
	})
}

func BenchmarkFullTableOverlapsV4(b *testing.B) {
	var rt Table[int]

	for i, route := range routes4 {
		rt.Insert(route.CIDR, i)
	}

	for i := 1; i <= 1024; i *= 2 {
		inter := new(Table[int])
		for j := 0; j <= i; j++ {
			pfx := randomPrefix4()
			inter.Insert(pfx, j)
		}

		b.Run(fmt.Sprintf("With_%4d", i), func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				boolSink = rt.Overlaps(inter)
			}
		})
	}
}

func BenchmarkFullTableOverlapsV6(b *testing.B) {
	var rt Table[int]

	for i, route := range routes6 {
		rt.Insert(route.CIDR, i)
	}

	for i := 1; i <= 1024; i *= 2 {
		inter := new(Table[int])
		for j := 0; j <= i; j++ {
			pfx := randomPrefix6()
			inter.Insert(pfx, j)
		}

		b.Run(fmt.Sprintf("With_%4d", i), func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				boolSink = rt.Overlaps(inter)
			}
		})
	}
}

func BenchmarkFullTableOverlapsPrefix(b *testing.B) {
	var rt Table[int]

	for i, route := range routes {
		rt.Insert(route.CIDR, i)
	}

	pfx := randomPrefix()

	b.ResetTimer()
	for range b.N {
		boolSink = rt.OverlapsPrefix(pfx)
	}
}

func BenchmarkFullTableClone(b *testing.B) {
	var rt4 Table[int]

	for i, route := range routes4 {
		rt4.Insert(route.CIDR, i)
	}

	b.ResetTimer()
	b.Run("CloneIP4", func(b *testing.B) {
		for range b.N {
			_ = rt4.Clone()
		}
	})

	var rt6 Table[int]

	for i, route := range routes6 {
		rt6.Insert(route.CIDR, i)
	}

	b.ResetTimer()
	b.Run("CloneIP6", func(b *testing.B) {
		for range b.N {
			_ = rt6.Clone()
		}
	})

	var rt Table[int]

	for i, route := range routes {
		rt.Insert(route.CIDR, i)
	}

	b.ResetTimer()
	b.Run("Clone", func(b *testing.B) {
		for range b.N {
			_ = rt.Clone()
		}
	})
}

func BenchmarkFullTableMemoryV4(b *testing.B) {
	var startMem, endMem runtime.MemStats

	rt := new(Table[struct{}])
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	b.Run(strconv.Itoa(len(routes4)), func(b *testing.B) {
		for range b.N {
			for _, route := range routes4 {
				rt.Insert(route.CIDR, struct{}{})
			}
		}

		runtime.GC()
		runtime.ReadMemStats(&endMem)

		stats := rt.root4.nodeStatsRec()
		b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc), "Bytes")
		b.ReportMetric(float64(rt.Size()), "size")
		b.ReportMetric(float64(stats.pfxs), "pfxs")
		b.ReportMetric(float64(stats.nodes), "nodes")
		b.ReportMetric(float64(stats.leaves), "leaves")
		b.ReportMetric(0, "ns/op")
	})
}

func BenchmarkFullTableMemoryV6(b *testing.B) {
	var startMem, endMem runtime.MemStats

	rt := new(Table[struct{}])
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	b.Run(strconv.Itoa(len(routes6)), func(b *testing.B) {
		for range b.N {
			for _, route := range routes6 {
				rt.Insert(route.CIDR, struct{}{})
			}
		}

		runtime.GC()
		runtime.ReadMemStats(&endMem)

		stats := rt.root6.nodeStatsRec()
		b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc), "Bytes")
		b.ReportMetric(float64(rt.Size()), "size")
		b.ReportMetric(float64(stats.pfxs), "pfxs")
		b.ReportMetric(float64(stats.nodes), "nodes")
		b.ReportMetric(float64(stats.leaves), "leaves")
		b.ReportMetric(0, "ns/op")
	})
}

func BenchmarkFullTableMemory(b *testing.B) {
	var startMem, endMem runtime.MemStats

	rt := new(Table[struct{}])
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	b.Run(strconv.Itoa(len(routes)), func(b *testing.B) {
		for range b.N {
			for _, route := range routes {
				rt.Insert(route.CIDR, struct{}{})
			}
		}

		runtime.GC()
		runtime.ReadMemStats(&endMem)

		s4 := rt.root4.nodeStatsRec()
		s6 := rt.root6.nodeStatsRec()
		stats := stats{s4.pfxs + s6.pfxs, s4.childs + s6.childs, s4.nodes + s6.nodes, s4.leaves + s6.leaves}

		b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc), "Bytes")
		b.ReportMetric(float64(rt.Size()), "size")
		b.ReportMetric(float64(stats.pfxs), "pfxs")
		b.ReportMetric(float64(stats.nodes), "nodes")
		b.ReportMetric(float64(stats.leaves), "leaves")
		b.ReportMetric(0, "ns/op")
	})
}

func fillRouteTables() {
	file, err := os.Open(prefixFile)
	if err != nil {
		log.Fatal(err)
	}

	rgz, err := gzip.NewReader(file)
	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(rgz)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		cidr := netip.MustParsePrefix(line)
		cidr = cidr.Masked()

		routes = append(routes, route{cidr, cidr})

		if cidr.Addr().Is4() {
			routes4 = append(routes4, route{cidr, cidr})
		} else {
			routes6 = append(routes6, route{cidr, cidr})
		}
	}

	if err = scanner.Err(); err != nil {
		log.Printf("reading from %v, %v", rgz, err)
	}
}

// #########################################################

func gimmeRandomPrefixes4(n int) []netip.Prefix {
	set := map[netip.Prefix]netip.Prefix{}
	pfxs := make([]netip.Prefix, 0, n)

	for {
		pfx := randomPrefix4()
		if _, ok := set[pfx]; !ok {
			set[pfx] = pfx
			pfxs = append(pfxs, pfx)
		}
		if len(set) >= n {
			break
		}
	}
	return pfxs
}

func gimmeRandomPrefixes6(n int) []netip.Prefix {
	set := map[netip.Prefix]netip.Prefix{}
	pfxs := make([]netip.Prefix, 0, n)

	for {
		pfx := randomPrefix6()
		if _, ok := set[pfx]; !ok {
			set[pfx] = pfx
			pfxs = append(pfxs, pfx)
		}
		if len(set) >= n {
			break
		}
	}
	return pfxs
}

func gimmeRandomPrefixes(n int) []netip.Prefix {
	pfxs := make([]netip.Prefix, 0, n)
	pfxs = append(pfxs, gimmeRandomPrefixes4(n/2)...)
	pfxs = append(pfxs, gimmeRandomPrefixes6(n-len(pfxs))...)

	prng.Shuffle(n, func(i, j int) {
		pfxs[i], pfxs[j] = pfxs[j], pfxs[i]
	})

	return pfxs
}
