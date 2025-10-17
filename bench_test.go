package bart

import (
	"fmt"
	"math"
	"math/rand/v2"
	"net/netip"
	"runtime"
	"testing"

	"github.com/gaissmai/bart/internal/nodes"
)

// roundFloat64 to 2 decimal places
func roundFloat64(f float64) float64 { return math.Round(f*100) / 100 }

func BenchmarkBartMatch4(b *testing.B) {
	bart := new(Table[struct{}])

	for _, route := range routes {
		bart.Insert(route.CIDR, struct{}{})
	}

	b.Run("Contains", func(b *testing.B) {
		for b.Loop() {
			bart.Contains(matchIP4)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		for b.Loop() {
			bart.Lookup(matchIP4)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		for b.Loop() {
			bart.LookupPrefix(matchPfx4)
		}
	})

	b.Run("LookupPfxLPM", func(b *testing.B) {
		for b.Loop() {
			bart.LookupPrefixLPM(matchPfx4)
		}
	})
}

func BenchmarkBartMatch6(b *testing.B) {
	bart := new(Table[struct{}])

	for _, route := range routes {
		bart.Insert(route.CIDR, struct{}{})
	}

	b.Run("Contains", func(b *testing.B) {
		for b.Loop() {
			bart.Contains(matchIP6)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		for b.Loop() {
			bart.Lookup(matchIP6)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		for b.Loop() {
			bart.LookupPrefix(matchPfx6)
		}
	})

	b.Run("LookupPfxLPM", func(b *testing.B) {
		for b.Loop() {
			bart.LookupPrefixLPM(matchPfx6)
		}
	})
}

func BenchmarkBartMiss4(b *testing.B) {
	bart := new(Table[int])

	for i, route := range routes {
		bart.Insert(route.CIDR, i)
	}

	b.Run("Contains", func(b *testing.B) {
		for b.Loop() {
			bart.Contains(missIP4)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		for b.Loop() {
			bart.Lookup(missIP4)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		for b.Loop() {
			bart.LookupPrefix(missPfx4)
		}
	})

	b.Run("LookupPfxLPM", func(b *testing.B) {
		for b.Loop() {
			bart.LookupPrefixLPM(missPfx4)
		}
	})
}

func BenchmarkBartMiss6(b *testing.B) {
	bart := new(Table[int])

	for i, route := range routes {
		bart.Insert(route.CIDR, i)
	}

	b.Run("Contains", func(b *testing.B) {
		for b.Loop() {
			bart.Contains(missIP6)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		for b.Loop() {
			bart.Lookup(missIP6)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		for b.Loop() {
			bart.LookupPrefix(missPfx6)
		}
	})

	b.Run("LookupPfxLPM", func(b *testing.B) {
		for b.Loop() {
			bart.LookupPrefixLPM(missPfx6)
		}
	})
}

func BenchmarkFastMatch4(b *testing.B) {
	fast := new(Fast[struct{}])

	for _, route := range routes {
		fast.Insert(route.CIDR, struct{}{})
	}

	b.Run("Contains", func(b *testing.B) {
		for b.Loop() {
			fast.Contains(matchIP4)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		for b.Loop() {
			fast.Lookup(matchIP4)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		for b.Loop() {
			fast.LookupPrefix(matchPfx4)
		}
	})

	b.Run("LookupPfxLPM", func(b *testing.B) {
		for b.Loop() {
			fast.LookupPrefixLPM(matchPfx4)
		}
	})
}

func BenchmarkFastMatch6(b *testing.B) {
	fast := new(Fast[struct{}])

	for _, route := range routes {
		fast.Insert(route.CIDR, struct{}{})
	}

	b.Run("Contains", func(b *testing.B) {
		for b.Loop() {
			fast.Contains(matchIP6)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		for b.Loop() {
			fast.Lookup(matchIP6)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		for b.Loop() {
			fast.LookupPrefix(matchPfx6)
		}
	})

	b.Run("LookupPfxLPM", func(b *testing.B) {
		for b.Loop() {
			fast.LookupPrefixLPM(matchPfx6)
		}
	})
}

func BenchmarkFastMiss4(b *testing.B) {
	fast := new(Fast[int])

	for i, route := range routes {
		fast.Insert(route.CIDR, i)
	}

	b.Run("Contains", func(b *testing.B) {
		for b.Loop() {
			fast.Contains(missIP4)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		for b.Loop() {
			fast.Lookup(missIP4)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		for b.Loop() {
			fast.LookupPrefix(missPfx4)
		}
	})

	b.Run("LookupPfxLPM", func(b *testing.B) {
		for b.Loop() {
			fast.LookupPrefixLPM(missPfx4)
		}
	})
}

func BenchmarkFastMiss6(b *testing.B) {
	fast := new(Fast[int])

	for i, route := range routes {
		fast.Insert(route.CIDR, i)
	}

	b.Run("Contains", func(b *testing.B) {
		for b.Loop() {
			fast.Contains(missIP6)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		for b.Loop() {
			fast.Lookup(missIP6)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		for b.Loop() {
			fast.LookupPrefix(missPfx6)
		}
	})

	b.Run("LookupPfxLPM", func(b *testing.B) {
		for b.Loop() {
			fast.LookupPrefixLPM(missPfx6)
		}
	})
}

func BenchmarkBartOverlaps4(b *testing.B) {
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
			for b.Loop() {
				lt.Overlaps4(lt2)
			}
		})
	}
}

func BenchmarkBartOverlaps6(b *testing.B) {
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
			for b.Loop() {
				lt.Overlaps6(lt2)
			}
		})
	}
}

func BenchmarkBartMemory4(b *testing.B) {
	var startMem, endMem runtime.MemStats

	bart := new(Table[struct{}])
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	b.Run(fmt.Sprintf("Table[]: %d", len(routes4)), func(b *testing.B) {
		for _, route := range routes4 {
			bart.Insert(route.CIDR, struct{}{})
		}

		runtime.GC()
		runtime.ReadMemStats(&endMem)

		stats := bart.root4.StatsRec()
		if stats.Prefixes == 0 {
			b.Skip("No prefixes inserted")
		}

		bytes := float64(endMem.HeapAlloc - startMem.HeapAlloc)
		b.ReportMetric(roundFloat64(bytes/float64(bart.Size())), "bytes/route")

		b.ReportMetric(float64(stats.Prefixes), "pfxs")
		b.ReportMetric(float64(stats.SubNodes), "nodes")
		b.ReportMetric(float64(stats.Leaves), "leaves")
		b.ReportMetric(float64(stats.Fringes), "fringes")
		b.ReportMetric(0, "ns/op")
	})
}

func BenchmarkBartMemory6(b *testing.B) {
	var startMem, endMem runtime.MemStats

	bart := new(Table[struct{}])
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	b.Run(fmt.Sprintf("Table[]: %d", len(routes6)), func(b *testing.B) {
		for _, route := range routes6 {
			bart.Insert(route.CIDR, struct{}{})
		}

		runtime.GC()
		runtime.ReadMemStats(&endMem)

		stats := bart.root6.StatsRec()
		if stats.Prefixes == 0 {
			b.Skip("No prefixes inserted")
		}

		bytes := float64(endMem.HeapAlloc - startMem.HeapAlloc)
		b.ReportMetric(roundFloat64(bytes/float64(bart.Size())), "bytes/route")

		b.ReportMetric(float64(stats.Prefixes), "pfxs")
		b.ReportMetric(float64(stats.SubNodes), "nodes")
		b.ReportMetric(float64(stats.Leaves), "leaves")
		b.ReportMetric(float64(stats.Fringes), "fringes")
		b.ReportMetric(0, "ns/op")
	})
}

func BenchmarkBartMemory(b *testing.B) {
	var startMem, endMem runtime.MemStats

	bart := new(Table[struct{}])
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	b.Run(fmt.Sprintf("Table[]: %d", len(routes)), func(b *testing.B) {
		for _, route := range routes {
			bart.Insert(route.CIDR, struct{}{})
		}

		runtime.GC()
		runtime.ReadMemStats(&endMem)

		s4 := bart.root4.StatsRec()
		s6 := bart.root6.StatsRec()
		stats := nodes.StatsT{
			Prefixes: s4.Prefixes + s6.Prefixes,
			Children: s4.Children + s6.Children,
			SubNodes: s4.SubNodes + s6.SubNodes,
			Leaves:   s4.Leaves + s6.Leaves,
			Fringes:  s4.Fringes + s6.Fringes,
		}

		if stats.Prefixes == 0 {
			b.Skip("No prefixes inserted")
		}

		bytes := float64(endMem.HeapAlloc - startMem.HeapAlloc)
		b.ReportMetric(roundFloat64(bytes/float64(bart.Size())), "bytes/route")

		b.ReportMetric(float64(stats.Prefixes), "pfxs")
		b.ReportMetric(float64(stats.SubNodes), "nodes")
		b.ReportMetric(float64(stats.Leaves), "leaves")
		b.ReportMetric(float64(stats.Fringes), "fringes")
		b.ReportMetric(0, "ns/op")
	})
}

// worstcase benchmarks

var (
	worstCaseProbeIP4  = mpa("255.255.255.255")
	worstCaseProbePfx4 = mpp("255.255.255.255/32")

	ipv4DefaultRoute = mpp("0.0.0.0/0")
	worstCasePfxsIP4 = []netip.Prefix{
		mpp("0.0.0.0/1"),
		mpp("254.0.0.0/8"),
		mpp("255.0.0.0/9"),
		mpp("255.254.0.0/16"),
		mpp("255.255.0.0/17"),
		mpp("255.255.254.0/24"),
		mpp("255.255.255.0/25"),
		mpp("255.255.255.255/32"), // matching prefix
	}

	worstCaseProbeIP6  = mpa("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff")
	worstCaseProbePfx6 = mpp("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff/128")

	ipv6DefaultRoute = mpp("::/0")
	worstCasePfxsIP6 = []netip.Prefix{
		mpp("::/1"),
		mpp("fe00::/8"),
		mpp("ff00::/9"),
		mpp("fffe::/16"),
		mpp("ffff::/17"),
		mpp("ffff:fe00::/24"),
		mpp("ffff:ff00::/25"),
		mpp("ffff:fffe::/32"),
		mpp("ffff:ffff::/33"),
		mpp("ffff:ffff:fe00::/40"),
		mpp("ffff:ffff:ff00::/41"),
		mpp("ffff:ffff:fffe::/48"),
		mpp("ffff:ffff:ffff::/49"),
		mpp("ffff:ffff:ffff:fe00::/56"),
		mpp("ffff:ffff:ffff:ff00::/57"),
		mpp("ffff:ffff:ffff:fffe::/64"),
		mpp("ffff:ffff:ffff:ffff::/65"),
		mpp("ffff:ffff:ffff:ffff:fe00::/72"),
		mpp("ffff:ffff:ffff:ffff:ff00::/73"),
		mpp("ffff:ffff:ffff:ffff:fffe::/80"),
		mpp("ffff:ffff:ffff:ffff:ffff::/81"),
		mpp("ffff:ffff:ffff:ffff:ffff:fe00::/88"),
		mpp("ffff:ffff:ffff:ffff:ffff:ff00::/89"),
		mpp("ffff:ffff:ffff:ffff:ffff:fffe::/96"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff::/97"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:fe00::/104"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:ff00::/105"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:fffe::/112"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:ffff::/113"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:ffff:fe00/120"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ff00/121"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:ffff:fffe/128"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff/128"),
	}
)

func BenchmarkBartWorstCaseMatch4(b *testing.B) {
	b.Run("Contains", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}

		for b.Loop() {
			tbl.Contains(worstCaseProbeIP4)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}
		tbl.Insert(ipv4DefaultRoute, ipv4DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx4)

		for b.Loop() {
			tbl.Lookup(worstCaseProbeIP4)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}
		tbl.Insert(ipv4DefaultRoute, ipv4DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx4)

		for b.Loop() {
			tbl.LookupPrefix(worstCaseProbePfx4)
		}
	})

	b.Run("LookupPrefixLPM", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}
		tbl.Insert(ipv4DefaultRoute, ipv4DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx4)

		for b.Loop() {
			tbl.LookupPrefixLPM(worstCaseProbePfx4)
		}
	})
}

func BenchmarkFastWorstCaseMatch4(b *testing.B) {
	b.Run("Contains", func(b *testing.B) {
		tbl := new(Fast[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}

		for b.Loop() {
			tbl.Contains(worstCaseProbeIP4)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		tbl := new(Fast[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}
		tbl.Insert(ipv4DefaultRoute, ipv4DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx4)

		for b.Loop() {
			tbl.Lookup(worstCaseProbeIP4)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		tbl := new(Fast[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}
		tbl.Insert(ipv4DefaultRoute, ipv4DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx4)

		for b.Loop() {
			tbl.LookupPrefix(worstCaseProbePfx4)
		}
	})

	b.Run("LookupPrefixLPM", func(b *testing.B) {
		tbl := new(Fast[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}
		tbl.Insert(ipv4DefaultRoute, ipv4DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx4)

		for b.Loop() {
			tbl.LookupPrefixLPM(worstCaseProbePfx4)
		}
	})
}

func BenchmarkBartWorstCaseMiss4(b *testing.B) {
	b.Run("Contains", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx4) // delete matching prefix

		for b.Loop() {
			tbl.Contains(worstCaseProbeIP4)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx4) // delete matching prefix

		for b.Loop() {
			tbl.Lookup(worstCaseProbeIP4)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx4) // delete matching prefix

		for b.Loop() {
			tbl.LookupPrefix(worstCaseProbePfx4)
		}
	})

	b.Run("LookupPrefixLPM", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx4) // delete matching prefix

		for b.Loop() {
			tbl.LookupPrefixLPM(worstCaseProbePfx4)
		}
	})
}

func BenchmarkFastWorstCaseMiss4(b *testing.B) {
	b.Run("Contains", func(b *testing.B) {
		tbl := new(Fast[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx4) // delete matching prefix

		for b.Loop() {
			tbl.Contains(worstCaseProbeIP4)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		tbl := new(Fast[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx4) // delete matching prefix

		for b.Loop() {
			tbl.Lookup(worstCaseProbeIP4)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		tbl := new(Fast[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx4) // delete matching prefix

		for b.Loop() {
			tbl.LookupPrefix(worstCaseProbePfx4)
		}
	})

	b.Run("LookupPrefixLPM", func(b *testing.B) {
		tbl := new(Fast[string])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx4) // delete matching prefix

		for b.Loop() {
			tbl.LookupPrefixLPM(worstCaseProbePfx4)
		}
	})
}

func BenchmarkBartWorstCaseMatch6(b *testing.B) {
	b.Run("Contains", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}

		for b.Loop() {
			tbl.Contains(worstCaseProbeIP6)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}
		tbl.Insert(ipv6DefaultRoute, ipv6DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx6)

		for b.Loop() {
			tbl.Lookup(worstCaseProbeIP6)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}
		tbl.Insert(ipv6DefaultRoute, ipv6DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx6)

		for b.Loop() {
			tbl.LookupPrefix(worstCaseProbePfx6)
		}
	})

	b.Run("LookupPrefixLPM", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}
		tbl.Insert(ipv6DefaultRoute, ipv6DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx6)

		for b.Loop() {
			tbl.LookupPrefixLPM(worstCaseProbePfx6)
		}
	})
}

func BenchmarkFastWorstCaseMatch6(b *testing.B) {
	b.Run("Contains", func(b *testing.B) {
		tbl := new(Fast[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}

		for b.Loop() {
			tbl.Contains(worstCaseProbeIP6)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		tbl := new(Fast[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}
		tbl.Insert(ipv6DefaultRoute, ipv6DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx6)

		for b.Loop() {
			tbl.Lookup(worstCaseProbeIP6)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		tbl := new(Fast[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}
		tbl.Insert(ipv6DefaultRoute, ipv6DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx6)

		for b.Loop() {
			tbl.LookupPrefix(worstCaseProbePfx6)
		}
	})

	b.Run("LookupPrefixLPM", func(b *testing.B) {
		tbl := new(Fast[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}
		tbl.Insert(ipv6DefaultRoute, ipv6DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx6)

		for b.Loop() {
			tbl.LookupPrefixLPM(worstCaseProbePfx6)
		}
	})
}

func BenchmarkBartWorstCaseMiss6(b *testing.B) {
	b.Run("Contains", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx6) // delete matching prefix

		for b.Loop() {
			tbl.Contains(worstCaseProbeIP6)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx6) // delete matching prefix

		for b.Loop() {
			tbl.Lookup(worstCaseProbeIP6)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx6) // delete matching prefix

		for b.Loop() {
			tbl.LookupPrefix(worstCaseProbePfx6)
		}
	})

	b.Run("LookupPrefixLPM", func(b *testing.B) {
		tbl := new(Table[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx6) // delete matching prefix

		for b.Loop() {
			tbl.LookupPrefixLPM(worstCaseProbePfx6)
		}
	})
}

func BenchmarkFastWorstCaseMiss6(b *testing.B) {
	b.Run("Contains", func(b *testing.B) {
		tbl := new(Fast[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx6) // delete matching prefix

		for b.Loop() {
			tbl.Contains(worstCaseProbeIP6)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		tbl := new(Fast[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx6) // delete matching prefix

		for b.Loop() {
			tbl.Lookup(worstCaseProbeIP6)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		tbl := new(Fast[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx6) // delete matching prefix

		for b.Loop() {
			tbl.LookupPrefix(worstCaseProbePfx6)
		}
	})

	b.Run("LookupPrefixLPM", func(b *testing.B) {
		tbl := new(Fast[string])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, p.String())
		}

		tbl.Delete(worstCaseProbePfx6) // delete matching prefix

		for b.Loop() {
			tbl.LookupPrefixLPM(worstCaseProbePfx6)
		}
	})
}
