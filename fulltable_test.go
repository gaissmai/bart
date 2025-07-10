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
	"strings"
	"testing"
)

// full internet prefix list, gzipped
const prefixFile = "testdata/prefixes.txt.gz"

var (
	routes  []route
	routes4 []route
	routes6 []route

	randRoute4 route
	randRoute6 route

	matchIP4  netip.Addr
	matchIP6  netip.Addr
	matchPfx4 netip.Prefix
	matchPfx6 netip.Prefix

	missIP4  netip.Addr
	missIP6  netip.Addr
	missPfx4 netip.Prefix
	missPfx6 netip.Prefix
)

type route struct {
	CIDR  netip.Prefix
	Value any
}

func init() {
	prng := rand.New(rand.NewPCG(42, 42))
	fillRouteTables()

	randRoute4 = routes4[prng.IntN(len(routes4))]
	randRoute6 = routes6[prng.IntN(len(routes6))]
}

var (
	intSink  int
	boolSink bool
)

func init() {
	prng := rand.New(rand.NewPCG(42, 42))
	lt := new(Lite)

	for _, route := range routes {
		lt.Insert(route.CIDR)
	}

	// find a random match IP4 and IP6
	for {
		matchIP4 = randomRealWorldPrefixes4(prng, 1)[0].Addr().Next()
		if ok := lt.Contains(matchIP4); ok {
			break
		}
	}
	for {
		matchIP6 = randomRealWorldPrefixes6(prng, 1)[0].Addr().Next()
		if ok := lt.Contains(matchIP6); ok {
			break
		}
	}

	// find a random match Pfx4
	for {
		matchPfx4 = randomRealWorldPrefixes4(prng, 1)[0]
		if _, ok := lt.LookupPrefix(matchPfx4); ok {
			break
		}
	}
	for {
		matchPfx6 = randomRealWorldPrefixes6(prng, 1)[0]
		if _, ok := lt.LookupPrefix(matchPfx6); ok {
			break
		}
	}

	for {
		missIP4 = randomRealWorldPrefixes4(prng, 1)[0].Addr().Next()
		if ok := lt.Contains(missIP4); !ok {
			break
		}
	}
	for {
		missIP6 = randomRealWorldPrefixes6(prng, 1)[0].Addr().Next()
		if ok := lt.Contains(missIP6); !ok {
			break
		}
	}

	for {
		missPfx4 = randomRealWorldPrefixes4(prng, 1)[0]
		if _, ok := lt.LookupPrefix(missPfx4); !ok {
			break
		}
	}
	for {
		missPfx6 = randomRealWorldPrefixes6(prng, 1)[0]
		if _, ok := lt.LookupPrefix(missPfx6); !ok {
			break
		}
	}
}

func BenchmarkFullMatch4(b *testing.B) {
	rt := new(Table[struct{}])

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

	b.Run("LookupPrefix", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			_, boolSink = rt.LookupPrefix(matchPfx4)
		}
	})

	b.Run("LookupPfxLPM", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			_, _, boolSink = rt.LookupPrefixLPM(matchPfx4)
		}
	})
}

func BenchmarkFullMatch6(b *testing.B) {
	rt := new(Table[struct{}])

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

	b.Run("LookupPrefix", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			_, boolSink = rt.LookupPrefix(matchPfx6)
		}
	})

	b.Run("LookupPfxLPM", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			_, _, boolSink = rt.LookupPrefixLPM(matchPfx6)
		}
	})
}

func BenchmarkFullMiss4(b *testing.B) {
	rt := new(Table[int])

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

	b.Run("LookupPrefix", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			intSink, boolSink = rt.LookupPrefix(missPfx4)
		}
	})

	b.Run("LookupPfxLPM", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			_, intSink, boolSink = rt.LookupPrefixLPM(missPfx4)
		}
	})
}

func BenchmarkFullMiss6(b *testing.B) {
	rt := new(Table[int])

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

	b.Run("LookupPrefix", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			intSink, boolSink = rt.LookupPrefix(missPfx6)
		}
	})

	b.Run("LookupPfxLPM", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			_, intSink, boolSink = rt.LookupPrefixLPM(missPfx6)
		}
	})
}

func BenchmarkFullTableOverlaps4(b *testing.B) {
	lt := new(Lite)

	for _, route := range routes4 {
		lt.Insert(route.CIDR)
	}

	for i := 1; i <= 1<<20; i *= 2 {
		prng := rand.New(rand.NewPCG(42, 42))
		lt2 := new(Lite)
		for _, pfx := range randomRealWorldPrefixes4(prng, i) {
			lt2.Insert(pfx)
		}

		b.Run(fmt.Sprintf("With_%4d", i), func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				boolSink = lt.Overlaps(lt2)
			}
		})
	}
}

func BenchmarkFullTableOverlaps6(b *testing.B) {
	lt := new(Lite)

	for _, route := range routes6 {
		lt.Insert(route.CIDR)
	}

	for i := 1; i <= 1<<20; i *= 2 {
		prng := rand.New(rand.NewPCG(42, 42))
		lt2 := new(Lite)
		for _, pfx := range randomRealWorldPrefixes6(prng, i) {
			lt2.Insert(pfx)
		}

		b.Run(fmt.Sprintf("With_%4d", i), func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				boolSink = lt.Overlaps(lt2)
			}
		})
	}
}

func BenchmarkFullTableOverlapsPrefix(b *testing.B) {
	lt := new(Lite)

	for _, route := range routes {
		lt.Insert(route.CIDR)
	}

	prng := rand.New(rand.NewPCG(42, 42))
	pfx := randomRealWorldPrefixes(prng, 1)[0]

	b.ResetTimer()
	for range b.N {
		boolSink = lt.OverlapsPrefix(pfx)
	}
}

func BenchmarkFullTableClone(b *testing.B) {
	rt4 := new(Table[int])

	for i, route := range routes4 {
		rt4.Insert(route.CIDR, i)
	}

	b.ResetTimer()
	b.Run("CloneIP4", func(b *testing.B) {
		for range b.N {
			_ = rt4.Clone()
		}
	})

	rt6 := new(Table[int])

	for i, route := range routes6 {
		rt6.Insert(route.CIDR, i)
	}

	b.ResetTimer()
	b.Run("CloneIP6", func(b *testing.B) {
		for range b.N {
			_ = rt6.Clone()
		}
	})

	rt := new(Table[int])

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

func BenchmarkFullTableMemory4(b *testing.B) {
	var startMem, endMem runtime.MemStats

	rt := new(Table[struct{}])
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
		b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc)/1024, "KByte")
		b.ReportMetric(float64(stats.pfxs), "pfxs")
		b.ReportMetric(float64(stats.nodes), "nodes")
		b.ReportMetric(float64(stats.leaves), "leaves")
		b.ReportMetric(float64(stats.fringes), "fringes")
		b.ReportMetric(0, "ns/op")
	})
}

func BenchmarkFullTableMemory6(b *testing.B) {
	var startMem, endMem runtime.MemStats

	rt := new(Table[struct{}])
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
		b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc)/1024, "KByte")
		b.ReportMetric(float64(stats.pfxs), "pfxs")
		b.ReportMetric(float64(stats.nodes), "nodes")
		b.ReportMetric(float64(stats.leaves), "leaves")
		b.ReportMetric(float64(stats.fringes), "fringes")
		b.ReportMetric(0, "ns/op")
	})
}

func BenchmarkFullTableMemory(b *testing.B) {
	var startMem, endMem runtime.MemStats

	rt := new(Table[struct{}])
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
			s4.pfxs + s6.pfxs,
			s4.childs + s6.childs,
			s4.nodes + s6.nodes,
			s4.leaves + s6.leaves,
			s4.fringes + s6.fringes,
		}

		b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc)/1024, "KByte")
		b.ReportMetric(float64(stats.pfxs), "pfxs")
		b.ReportMetric(float64(stats.nodes), "nodes")
		b.ReportMetric(float64(stats.leaves), "leaves")
		b.ReportMetric(float64(stats.fringes), "fringes")
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

func randomRealWorldPrefixes4(prng *rand.Rand, n int) []netip.Prefix {
	set := map[netip.Prefix]netip.Prefix{}
	pfxs := make([]netip.Prefix, 0, n)

	for {
		pfx := randomPrefix4(prng)

		// skip too small or too big masks
		if pfx.Bits() < 8 || pfx.Bits() > 28 {
			continue
		}

		// skip multicast ...
		if pfx.Overlaps(mpp("240.0.0.0/8")) {
			continue
		}

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

func randomRealWorldPrefixes6(prng *rand.Rand, n int) []netip.Prefix {
	set := map[netip.Prefix]netip.Prefix{}
	pfxs := make([]netip.Prefix, 0, n)

	for {
		pfx := randomPrefix6(prng)

		// skip too small or too big masks
		if pfx.Bits() < 16 || pfx.Bits() > 56 {
			continue
		}

		// skip non global routes seen in the real world
		if !pfx.Overlaps(mpp("2000::/3")) {
			continue
		}
		if pfx.Addr().Compare(mpp("2c0f::/16").Addr()) == 1 {
			continue
		}

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

func randomRealWorldPrefixes(prng *rand.Rand, n int) []netip.Prefix {
	pfxs := make([]netip.Prefix, 0, n)
	pfxs = append(pfxs, randomRealWorldPrefixes4(prng, n/2)...)
	pfxs = append(pfxs, randomRealWorldPrefixes6(prng, n-len(pfxs))...)

	prng.Shuffle(n, func(i, j int) {
		pfxs[i], pfxs[j] = pfxs[j], pfxs[i]
	})

	return pfxs
}
