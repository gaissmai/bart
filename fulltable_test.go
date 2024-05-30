// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"log"
	"math/rand"
	"net/netip"
	"os"
	"runtime"
	"strconv"
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
)

type route struct {
	CIDR  netip.Prefix
	Value any
}

func init() {
	fillRouteTables()

	randRoute4 = routes4[rand.Intn(len(routes4))]
	randRoute6 = routes6[rand.Intn(len(routes6))]
}

var (
	intSink  int
	okSink   bool
	boolSink bool

	pfxSliceSink []netip.Prefix
	cloneSink    *Table[int]
)

func BenchmarkFullTableInsert(b *testing.B) {
	var rt Table[struct{}]

	b.ResetTimer()
	for k := 0; k < b.N; k++ {
		for _, route := range routes6 {
			rt.Insert(route.CIDR, struct{}{})
		}
	}
}

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

	b.Run("Lookup", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			intSink, okSink = rt.Lookup(ip)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			intSink, okSink = rt.LookupPrefix(ipAsPfx)
		}
	})

	b.Run("LookupPrefixLPM", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
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

	b.Run("Lookup", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			intSink, okSink = rt.Lookup(ip)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			intSink, okSink = rt.LookupPrefix(ipAsPfx)
		}
	})

	b.Run("LookupPrefixLPM", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
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

	b.Run("Lookup", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			intSink, okSink = rt.Lookup(ip)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			intSink, okSink = rt.LookupPrefix(ipAsPfx)
		}
	})

	b.Run("LookupPrefixLPM", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
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

	b.Run("Lookup", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			intSink, okSink = rt.Lookup(ip)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			intSink, okSink = rt.LookupPrefix(ipAsPfx)
		}
	})

	b.Run("LookupPrefixLPM", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
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
			for k := 0; k < b.N; k++ {
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
			for k := 0; k < b.N; k++ {
				boolSink = rt.Overlaps(inter)
			}
		})
	}
}

func BenchmarkFullTableOverlaps(b *testing.B) {
	var rt Table[int]

	for i, route := range routes {
		rt.Insert(route.CIDR, i)
	}

	for i := 1; i <= 1024; i *= 2 {
		inter := new(Table[int])
		for j := 0; j <= i; j++ {
			pfx := randomPrefix()
			inter.Insert(pfx, j)
		}

		b.Run(fmt.Sprintf("With_%4d", i), func(b *testing.B) {
			b.ResetTimer()
			for k := 0; k < b.N; k++ {
				boolSink = rt.Overlaps(inter)
			}
		})
	}
}

func BenchmarkFullTableCloneV4(b *testing.B) {
	var rt Table[int]

	for i, route := range routes4 {
		rt.Insert(route.CIDR, i)
	}

	b.ResetTimer()
	for k := 0; k < b.N; k++ {
		cloneSink = rt.Clone()
	}
}

func BenchmarkFullTableCloneV6(b *testing.B) {
	var rt Table[int]

	for i, route := range routes6 {
		rt.Insert(route.CIDR, i)
	}

	b.ResetTimer()
	for k := 0; k < b.N; k++ {
		cloneSink = rt.Clone()
	}
}

func BenchmarkFullTableClone(b *testing.B) {
	var rt Table[int]

	for i, route := range routes {
		rt.Insert(route.CIDR, i)
	}

	b.ResetTimer()
	for k := 0; k < b.N; k++ {
		cloneSink = rt.Clone()
	}
}

func BenchmarkFullTableSubnets(b *testing.B) {
	var rt Table[int]

	for i, route := range routes {
		rt.Insert(route.CIDR, i)
	}

	b.Run("V4", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			pfxSliceSink = rt.Subnets(randRoute4.CIDR)
		}
	})

	b.Run("V6", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			pfxSliceSink = rt.Subnets(randRoute6.CIDR)
		}
	})
}

func BenchmarkFullTableSupernets(b *testing.B) {
	var rt Table[int]

	for i, route := range routes {
		rt.Insert(route.CIDR, i)
	}

	b.Run("V4", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			pfxSliceSink = rt.Supernets(randRoute4.CIDR)
		}
	})

	b.Run("V6", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			pfxSliceSink = rt.Supernets(randRoute6.CIDR)
		}
	})
}

func BenchmarkFullTableMemoryV4(b *testing.B) {
	var startMem, endMem runtime.MemStats

	rt := new(Table[struct{}])
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	b.Run(strconv.Itoa(len(routes4)), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, route := range routes4 {
				rt.Insert(route.CIDR, struct{}{})
			}
		}

		runtime.GC()
		runtime.ReadMemStats(&endMem)

		b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc), "Bytes")
		b.ReportMetric(float64(rt.Size())/float64(rt.nodes()), "Prefix/Node")
		b.ReportMetric(0, "ns/op")
	})
}

func BenchmarkFullTableMemoryV6(b *testing.B) {
	var startMem, endMem runtime.MemStats

	rt := new(Table[struct{}])
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	b.Run(strconv.Itoa(len(routes6)), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, route := range routes6 {
				rt.Insert(route.CIDR, struct{}{})
			}
		}

		runtime.GC()
		runtime.ReadMemStats(&endMem)

		b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc), "Bytes")
		b.ReportMetric(float64(rt.Size())/float64(rt.nodes()), "Prefix/Node")
		b.ReportMetric(0, "ns/op")
	})
}

func BenchmarkFullTableMemory(b *testing.B) {
	var startMem, endMem runtime.MemStats

	rt := new(Table[struct{}])
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	b.Run(strconv.Itoa(len(routes)), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, route := range routes {
				rt.Insert(route.CIDR, struct{}{})
			}
		}

		runtime.GC()
		runtime.ReadMemStats(&endMem)

		b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc), "Bytes")
		b.ReportMetric(float64(rt.Size())/float64(rt.nodes()), "Prefix/Node")
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

	if err := scanner.Err(); err != nil {
		log.Printf("reading from %v, %v", rgz, err)
	}
}

//nolint:unused
func sliceRoutes(n int) []route {
	if n > len(routes) {
		panic("n too big")
	}

	clone := make([]route, 0, n)
	clone = append(clone, routes...)

	rand.Shuffle(len(clone), func(i, j int) {
		clone[i], clone[j] = clone[j], clone[i]
	})
	return clone[:n]
}

// #########################################################

//nolint:unused
func gimmeRandomPrefix4(n int) (pfxs []netip.Prefix) {
	set := map[netip.Prefix]netip.Prefix{}

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
	return
}

//nolint:unused
func gimmeRandomPrefix6(n int) (pfxs []netip.Prefix) {
	set := map[netip.Prefix]netip.Prefix{}

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
	return
}
