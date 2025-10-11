package bart

import (
	"fmt"
	"math"
	"math/rand/v2"
	"runtime"
	"strconv"
	"testing"

	"github.com/gaissmai/bart/internal/nodes"
)

var benchRouteCount = []int{1, 2, 5, 10, 100, 1000, 10_000, 100_000, 200_000}

// roundFloat64 to 2 decimal places
func roundFloat64(f float64) float64 { return math.Round(f*100) / 100 }

func BenchmarkTableModifyRandom(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, n := range benchRouteCount {
		randomPfxs := randomRealWorldPrefixes(prng, n)

		bart := new(Table[int])
		for i, pfx := range randomPfxs {
			bart.Insert(pfx, i)
		}

		prt := bart

		probe := randomPfxs[prng.IntN(len(randomPfxs))]

		b.Run(fmt.Sprintf("mutable into %d", n), func(b *testing.B) {
			for b.Loop() {
				bart.Modify(probe, func(int, bool) (int, bool) { return 42, false })
			}
		})

		b.Run(fmt.Sprintf("persist into %d", n), func(b *testing.B) {
			for b.Loop() {
				prt = prt.ModifyPersist(probe, func(int, bool) (int, bool) { return 42, false })
			}
		})

	}
}

func BenchmarkTableDelete(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, n := range []int{1_000, 10_000, 100_000, 1_000_000} {
		pfxs := randomPrefixes(prng, n)

		b.Run(fmt.Sprintf("mutable from_%d", n), func(b *testing.B) {
			for b.Loop() {
				b.StopTimer()
				bart := new(Table[*MyInt])

				for i, route := range pfxs {
					myInt := MyInt(i)
					bart.Insert(route.pfx, &myInt)
				}
				b.StartTimer()

				for _, route := range pfxs {
					bart.Delete(route.pfx)
				}
			}
			b.ReportMetric(float64(b.Elapsed())/float64(b.N)/float64(len(pfxs)), "ns/route")
			b.ReportMetric(0, "ns/op")
		})

		b.Run(fmt.Sprintf("persist from_%d", n), func(b *testing.B) {
			for b.Loop() {
				b.StopTimer()
				bart := new(Table[*MyInt])

				for i, route := range pfxs {
					myInt := MyInt(i)
					bart.Insert(route.pfx, &myInt)
				}
				b.StartTimer()

				for _, route := range pfxs {
					bart = bart.DeletePersist(route.pfx)
				}
			}
			b.ReportMetric(float64(b.Elapsed())/float64(b.N)/float64(len(pfxs)), "ns/route")
			b.ReportMetric(0, "ns/op")
		})
	}
}

func BenchmarkTableGet(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		for _, nroutes := range benchRouteCount {
			bart := new(Table[int])
			for _, route := range rng(prng, nroutes) {
				bart.Insert(route.pfx, route.val)
			}

			probe := rng(prng, 1)[0]

			b.Run(fmt.Sprintf("%s/From_%d", fam, nroutes), func(b *testing.B) {
				for b.Loop() {
					bart.Get(probe.pfx)
				}
			})
		}
	}
}

func BenchmarkTableLPM(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		for _, nroutes := range benchRouteCount {
			bart := new(Table[int])
			for _, route := range rng(prng, nroutes) {
				bart.Insert(route.pfx, route.val)
			}

			probe := rng(prng, 1)[0]

			b.Run(fmt.Sprintf("%s/In_%6d/%s", fam, nroutes, "Contains"), func(b *testing.B) {
				for b.Loop() {
					bart.Contains(probe.pfx.Addr())
				}
			})

			b.Run(fmt.Sprintf("%s/In_%6d/%s", fam, nroutes, "Lookup"), func(b *testing.B) {
				for b.Loop() {
					bart.Lookup(probe.pfx.Addr())
				}
			})

			b.Run(fmt.Sprintf("%s/In_%6d/%s", fam, nroutes, "Prefix"), func(b *testing.B) {
				for b.Loop() {
					bart.LookupPrefix(probe.pfx)
				}
			})

			b.Run(fmt.Sprintf("%s/In_%6d/%s", fam, nroutes, "PrefixLPM"), func(b *testing.B) {
				for b.Loop() {
					bart.LookupPrefixLPM(probe.pfx)
				}
			})
		}
	}
}

func BenchmarkTableOverlapsPrefix(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		for _, nroutes := range benchRouteCount {
			bart := new(Table[int])
			for _, route := range rng(prng, nroutes) {
				bart.Insert(route.pfx, route.val)
			}

			probe := rng(prng, 1)[0]

			b.Run(fmt.Sprintf("%s/With_%d", fam, nroutes), func(b *testing.B) {
				for b.Loop() {
					bart.OverlapsPrefix(probe.pfx)
				}
			})
		}
	}
}

func BenchmarkTableOverlaps(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		for _, nroutes := range benchRouteCount {
			bart := new(Table[int])
			for _, route := range rng(prng, nroutes) {
				bart.Insert(route.pfx, route.val)
			}

			inter := new(Table[int])
			for _, route := range rng(prng, nroutes) {
				inter.Insert(route.pfx, route.val)
			}

			b.Run(fmt.Sprintf("%s/%d_with_%d", fam, nroutes, nroutes), func(b *testing.B) {
				for b.Loop() {
					bart.Overlaps(inter)
				}
			})
		}
	}
}

func BenchmarkTableClone(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		for _, nroutes := range benchRouteCount {
			bart := new(Table[int])
			for _, route := range rng(prng, nroutes) {
				bart.Insert(route.pfx, route.val)
			}

			b.Run(fmt.Sprintf("%s/%d", fam, nroutes), func(b *testing.B) {
				for b.Loop() {
					bart.Clone()
				}
			})
		}
	}
}

func BenchmarkFullMatch4(b *testing.B) {
	rt := new(Table[struct{}])

	for _, route := range routes {
		rt.Insert(route.CIDR, struct{}{})
	}

	b.Run("Contains", func(b *testing.B) {
		for b.Loop() {
			rt.Contains(matchIP4)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		for b.Loop() {
			rt.Lookup(matchIP4)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		for b.Loop() {
			rt.LookupPrefix(matchPfx4)
		}
	})

	b.Run("LookupPfxLPM", func(b *testing.B) {
		for b.Loop() {
			rt.LookupPrefixLPM(matchPfx4)
		}
	})
}

func BenchmarkFullMatch6(b *testing.B) {
	rt := new(Table[struct{}])

	for _, route := range routes {
		rt.Insert(route.CIDR, struct{}{})
	}

	b.Run("Contains", func(b *testing.B) {
		for b.Loop() {
			rt.Contains(matchIP6)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		for b.Loop() {
			rt.Lookup(matchIP6)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		for b.Loop() {
			rt.LookupPrefix(matchPfx6)
		}
	})

	b.Run("LookupPfxLPM", func(b *testing.B) {
		for b.Loop() {
			rt.LookupPrefixLPM(matchPfx6)
		}
	})
}

func BenchmarkFullMiss4(b *testing.B) {
	rt := new(Table[int])

	for i, route := range routes {
		rt.Insert(route.CIDR, i)
	}

	b.Run("Contains", func(b *testing.B) {
		for b.Loop() {
			rt.Contains(missIP4)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		for b.Loop() {
			rt.Lookup(missIP4)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		for b.Loop() {
			rt.LookupPrefix(missPfx4)
		}
	})

	b.Run("LookupPfxLPM", func(b *testing.B) {
		for b.Loop() {
			rt.LookupPrefixLPM(missPfx4)
		}
	})
}

func BenchmarkFullMiss6(b *testing.B) {
	rt := new(Table[int])

	for i, route := range routes {
		rt.Insert(route.CIDR, i)
	}

	b.Run("Contains", func(b *testing.B) {
		for b.Loop() {
			rt.Contains(missIP6)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		for b.Loop() {
			rt.Lookup(missIP6)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		for b.Loop() {
			rt.LookupPrefix(missPfx6)
		}
	})

	b.Run("LookupPfxLPM", func(b *testing.B) {
		for b.Loop() {
			rt.LookupPrefixLPM(missPfx6)
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
			for b.Loop() {
				lt.Overlaps4(lt2)
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
			for b.Loop() {
				lt.Overlaps6(lt2)
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

	for b.Loop() {
		lt.OverlapsPrefix(pfx)
	}
}

func BenchmarkFullTableClone(b *testing.B) {
	rt4 := new(Table[int])

	for i, route := range routes4 {
		rt4.Insert(route.CIDR, i)
	}

	b.Run("CloneIP4", func(b *testing.B) {
		for b.Loop() {
			_ = rt4.Clone()
		}
	})

	rt6 := new(Table[int])

	for i, route := range routes6 {
		rt6.Insert(route.CIDR, i)
	}

	b.Run("CloneIP6", func(b *testing.B) {
		for b.Loop() {
			_ = rt6.Clone()
		}
	})

	rt := new(Table[int])

	for i, route := range routes {
		rt.Insert(route.CIDR, i)
	}

	b.Run("Clone", func(b *testing.B) {
		for b.Loop() {
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
		for _, route := range routes4 {
			rt.Insert(route.CIDR, struct{}{})
		}

		runtime.GC()
		runtime.ReadMemStats(&endMem)

		stats := rt.root4.StatsRec()
		if stats.Pfxs == 0 {
			b.Skip("No prefixes inserted")
		}

		bytes := float64(endMem.HeapAlloc - startMem.HeapAlloc)
		b.ReportMetric(roundFloat64(bytes/float64(rt.Size())), "bytes/route")

		b.ReportMetric(float64(stats.Pfxs), "pfxs")
		b.ReportMetric(float64(stats.Nodes), "nodes")
		b.ReportMetric(float64(stats.Leaves), "leaves")
		b.ReportMetric(float64(stats.Fringes), "fringes")
		b.ReportMetric(0, "ns/op")
	})
}

func BenchmarkFullTableMemory6(b *testing.B) {
	var startMem, endMem runtime.MemStats

	rt := new(Table[struct{}])
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	b.Run(fmt.Sprintf("Table[]: %d", len(routes6)), func(b *testing.B) {
		for _, route := range routes6 {
			rt.Insert(route.CIDR, struct{}{})
		}

		runtime.GC()
		runtime.ReadMemStats(&endMem)

		stats := rt.root6.StatsRec()
		if stats.Pfxs == 0 {
			b.Skip("No prefixes inserted")
		}

		bytes := float64(endMem.HeapAlloc - startMem.HeapAlloc)
		b.ReportMetric(roundFloat64(bytes/float64(rt.Size())), "bytes/route")

		b.ReportMetric(float64(stats.Pfxs), "pfxs")
		b.ReportMetric(float64(stats.Nodes), "nodes")
		b.ReportMetric(float64(stats.Leaves), "leaves")
		b.ReportMetric(float64(stats.Fringes), "fringes")
		b.ReportMetric(0, "ns/op")
	})
}

func BenchmarkFullTableMemory(b *testing.B) {
	var startMem, endMem runtime.MemStats

	rt := new(Table[struct{}])
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	b.Run(fmt.Sprintf("Table[]: %d", len(routes)), func(b *testing.B) {
		for _, route := range routes {
			rt.Insert(route.CIDR, struct{}{})
		}

		runtime.GC()
		runtime.ReadMemStats(&endMem)

		s4 := rt.root4.StatsRec()
		s6 := rt.root6.StatsRec()
		stats := nodes.StatsT{
			Pfxs:    s4.Pfxs + s6.Pfxs,
			Childs:  s4.Childs + s6.Childs,
			Nodes:   s4.Nodes + s6.Nodes,
			Leaves:  s4.Leaves + s6.Leaves,
			Fringes: s4.Fringes + s6.Fringes,
		}

		if stats.Pfxs == 0 {
			b.Skip("No prefixes inserted")
		}

		bytes := float64(endMem.HeapAlloc - startMem.HeapAlloc)
		b.ReportMetric(roundFloat64(bytes/float64(rt.Size())), "bytes/route")

		b.ReportMetric(float64(stats.Pfxs), "pfxs")
		b.ReportMetric(float64(stats.Nodes), "nodes")
		b.ReportMetric(float64(stats.Leaves), "leaves")
		b.ReportMetric(float64(stats.Fringes), "fringes")
		b.ReportMetric(0, "ns/op")
	})
}

func BenchmarkMemIP4(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, k := range []int{1_000, 10_000, 100_000, 1_000_000} {
		var startMem, endMem runtime.MemStats

		runtime.GC()
		runtime.ReadMemStats(&startMem)

		b.Run(strconv.Itoa(k), func(b *testing.B) {
			bart := new(Table[struct{}])
			for b.Loop() {
				bart = new(Table[struct{}])
				for _, pfx := range randomRealWorldPrefixes4(prng, k) {
					bart.Insert(pfx, struct{}{})
				}
			}

			runtime.GC()
			runtime.ReadMemStats(&endMem)

			stats := bart.root4.StatsRec()

			bytes := float64(endMem.HeapAlloc - startMem.HeapAlloc)
			b.ReportMetric(roundFloat64(bytes/float64(stats.Pfxs)), "bytes/route")
			b.ReportMetric(float64(stats.Pfxs), "pfxs")
			b.ReportMetric(float64(stats.Nodes), "nodes")
			b.ReportMetric(float64(stats.Leaves), "leaves")
			b.ReportMetric(float64(stats.Fringes), "fringes")
			b.ReportMetric(0, "ns/op")
		})
	}
}

func BenchmarkMemIP6(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, k := range []int{1_000, 10_000, 100_000, 1_000_000} {
		var startMem, endMem runtime.MemStats

		runtime.GC()
		runtime.ReadMemStats(&startMem)

		b.Run(strconv.Itoa(k), func(b *testing.B) {
			bart := new(Table[struct{}])
			for b.Loop() {
				bart = new(Table[struct{}])
				for _, pfx := range randomRealWorldPrefixes6(prng, k) {
					bart.Insert(pfx, struct{}{})
				}
			}

			runtime.GC()
			runtime.ReadMemStats(&endMem)

			stats := bart.root6.StatsRec()

			bytes := float64(endMem.HeapAlloc - startMem.HeapAlloc)
			b.ReportMetric(roundFloat64(bytes/float64(stats.Pfxs)), "bytes/route")
			b.ReportMetric(float64(stats.Pfxs), "pfxs")
			b.ReportMetric(float64(stats.Nodes), "nodes")
			b.ReportMetric(float64(stats.Leaves), "leaves")
			b.ReportMetric(float64(stats.Fringes), "fringes")
			b.ReportMetric(0, "ns/op")
		})
	}
}

func BenchmarkMem(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, k := range []int{1_000, 10_000, 100_000, 1_000_000} {
		var startMem, endMem runtime.MemStats

		runtime.GC()
		runtime.ReadMemStats(&startMem)

		b.Run(strconv.Itoa(k), func(b *testing.B) {
			bart := new(Table[struct{}])
			for b.Loop() {
				bart = new(Table[struct{}])
				for _, pfx := range randomRealWorldPrefixes(prng, k) {
					bart.Insert(pfx, struct{}{})
				}
			}

			runtime.GC()
			runtime.ReadMemStats(&endMem)

			s4 := bart.root4.StatsRec()
			s6 := bart.root6.StatsRec()
			stats := nodes.StatsT{
				Pfxs:    s4.Pfxs + s6.Pfxs,
				Childs:  s4.Childs + s6.Childs,
				Nodes:   s4.Nodes + s6.Nodes,
				Leaves:  s4.Leaves + s6.Leaves,
				Fringes: s4.Fringes + s6.Fringes,
			}

			bytes := float64(endMem.HeapAlloc - startMem.HeapAlloc)
			b.ReportMetric(roundFloat64(bytes/float64(stats.Pfxs)), "bytes/route")
			b.ReportMetric(float64(stats.Pfxs), "pfxs")
			b.ReportMetric(float64(stats.Nodes), "nodes")
			b.ReportMetric(float64(stats.Leaves), "leaves")
			b.ReportMetric(float64(stats.Fringes), "fringes")
			b.ReportMetric(0, "ns/op")
		})
	}
}
