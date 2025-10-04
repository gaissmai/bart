// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"net/netip"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/gaissmai/bart/internal/nodes"
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

	if len(routes4) == 0 || len(routes6) == 0 {
		log.Fatal("no routes loaded from " + prefixFile)
	}

	randRoute4 = routes4[prng.IntN(len(routes4))]
	randRoute6 = routes6[prng.IntN(len(routes6))]

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
		if ok := lt.LookupPrefix(matchPfx4); ok {
			break
		}
	}
	for {
		matchPfx6 = randomRealWorldPrefixes6(prng, 1)[0]
		if ok := lt.LookupPrefix(matchPfx6); ok {
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
		if ok := lt.LookupPrefix(missPfx4); !ok {
			break
		}
	}
	for {
		missPfx6 = randomRealWorldPrefixes6(prng, 1)[0]
		if ok := lt.LookupPrefix(missPfx6); !ok {
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

	b.Log(matchIP6)
	b.Log(matchPfx6)

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

	b.Log(missIP4)
	b.Log(missPfx4)

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

	b.Log(missIP6)
	b.Log(missPfx6)

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
				lt.Overlaps(lt2)
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
				lt.Overlaps(lt2)
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

func fillRouteTables() {
	file, err := os.Open(prefixFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	rgz, err := gzip.NewReader(file)
	if err != nil {
		log.Fatal(err)
	}
	defer rgz.Close()

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
		log.Fatalf("reading %s, %v", prefixFile, err)
	}
}

// #########################################################

func randomRealWorldPrefixes4(prng *rand.Rand, n int) []netip.Prefix {
	set := make(map[netip.Prefix]struct{})
	pfxs := make([]netip.Prefix, 0, n)

	for len(set) < n {
		pfx := randomPrefix4(prng)

		// skip too small or too big masks
		if pfx.Bits() < 8 || pfx.Bits() > 28 {
			continue
		}

		// skip reserved/experimental ranges (e.g., 240.0.0.0/8)
		if pfx.Overlaps(mpp("240.0.0.0/8")) {
			continue
		}

		if _, ok := set[pfx]; !ok {
			set[pfx] = struct{}{}
			pfxs = append(pfxs, pfx)
		}
	}
	return pfxs
}

func randomRealWorldPrefixes6(prng *rand.Rand, n int) []netip.Prefix {
	set := make(map[netip.Prefix]struct{})
	pfxs := make([]netip.Prefix, 0, n)

	for len(set) < n {
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
			set[pfx] = struct{}{}
			pfxs = append(pfxs, pfx)
		}
	}
	return pfxs
}

func randomRealWorldPrefixes(prng *rand.Rand, n int) []netip.Prefix {
	pfxs := make([]netip.Prefix, 0, n)
	pfxs = append(pfxs, randomRealWorldPrefixes4(prng, n/2)...)
	pfxs = append(pfxs, randomRealWorldPrefixes6(prng, n-len(pfxs))...)

	prng.Shuffle(len(pfxs), func(i, j int) {
		pfxs[i], pfxs[j] = pfxs[j], pfxs[i]
	})

	return pfxs
}

// roundFloat64 to 2 decimal places
func roundFloat64(f float64) float64 { return math.Round(f*100) / 100 }

// ------------------------
// Unit tests (appended)
// ------------------------

func TestInitLoadedRoutesAndSplits(t *testing.T) {
	t.Parallel()

	// Ensure init() ran and loaded data
	if len(routes) == 0 {
		t.Fatalf("expected routes to be loaded from %s", prefixFile)
	}
	if len(routes4) == 0 {
		t.Fatalf("expected IPv4 routes to be loaded")
	}
	if len(routes6) == 0 {
		t.Fatalf("expected IPv6 routes to be loaded")
	}
	// routes should equal routes4 + routes6
	if got, want := len(routes), len(routes4)+len(routes6); got != want {
		t.Fatalf("routes count mismatch: got %d want %d", got, want)
	}
	// Validate family partition
	for _, r := range routes4 {
		if !r.CIDR.Addr().Is4() {
			t.Fatalf("routes4 contains non-IPv4 prefix: %v", r.CIDR)
		}
	}
	for _, r := range routes6 {
		if r.CIDR.Addr().Is4() {
			t.Fatalf("routes6 contains IPv4 prefix: %v", r.CIDR)
		}
	}
}

func TestTableContainsAndLookup_MatchAndMiss_IPv4(t *testing.T) {
	t.Parallel()

	rt := new(Table[struct{}])
	for _, r := range routes {
		rt.Insert(r.CIDR, struct{}{})
	}

	// Happy paths
	if ok := rt.Contains(matchIP4); !ok {
		t.Fatalf("expected Contains(%v) to be true", matchIP4)
	}
	if _, ok := rt.Lookup(matchIP4); !ok {
		t.Fatalf("expected Lookup(%v) to find a value", matchIP4)
	}
	if _, ok := rt.LookupPrefix(matchPfx4); !ok {
		t.Fatalf("expected LookupPrefix(%v) to find a value", matchPfx4)
	}
	if _, _, ok := rt.LookupPrefixLPM(matchPfx4); !ok {
		t.Fatalf("expected LookupPrefixLPM(%v) to find a value", matchPfx4)
	}

	// Miss paths
	if ok := rt.Contains(missIP4); ok {
		t.Fatalf("expected Contains(%v) to be false", missIP4)
	}
	if _, ok := rt.Lookup(missIP4); ok {
		t.Fatalf("expected Lookup(%v) to miss", missIP4)
	}
	if _, ok := rt.LookupPrefix(missPfx4); ok {
		t.Fatalf("expected LookupPrefix(%v) to miss", missPfx4)
	}
	if _, _, ok := rt.LookupPrefixLPM(missPfx4); ok {
		t.Fatalf("expected LookupPrefixLPM(%v) to miss", missPfx4)
	}
}

func TestTableContainsAndLookup_MatchAndMiss_IPv6(t *testing.T) {
	t.Parallel()

	rt := new(Table[int])
	for i, r := range routes {
		rt.Insert(r.CIDR, i)
	}

	// Happy paths
	if ok := rt.Contains(matchIP6); !ok {
		t.Fatalf("expected Contains(%v) to be true", matchIP6)
	}
	if _, ok := rt.Lookup(matchIP6); !ok {
		t.Fatalf("expected Lookup(%v) to find a value", matchIP6)
	}
	if _, ok := rt.LookupPrefix(matchPfx6); !ok {
		t.Fatalf("expected LookupPrefix(%v) to find a value", matchPfx6)
	}
	if _, _, ok := rt.LookupPrefixLPM(matchPfx6); !ok {
		t.Fatalf("expected LookupPrefixLPM(%v) to find a value", matchPfx6)
	}

	// Miss paths
	if ok := rt.Contains(missIP6); ok {
		t.Fatalf("expected Contains(%v) to be false", missIP6)
	}
	if _, ok := rt.Lookup(missIP6); ok {
		t.Fatalf("expected Lookup(%v) to miss", missIP6)
	}
	if _, ok := rt.LookupPrefix(missPfx6); ok {
		t.Fatalf("expected LookupPrefix(%v) to miss", missPfx6)
	}
	if _, _, ok := rt.LookupPrefixLPM(missPfx6); ok {
		t.Fatalf("expected LookupPrefixLPM(%v) to miss", missPfx6)
	}
}

func TestLiteOverlapsPrefix_IPv6(t *testing.T) {
	t.Parallel()

	prng := rand.New(rand.NewPCG(42, 42))
	lt := new(Lite)
	// Insert a controlled IPv6 prefix and verify overlaps with its super/sub prefixes
	pfxs := randomRealWorldPrefixes6(prng, 1)
	if len(pfxs) == 0 {
		t.Skip("no IPv6 random prefixes produced")
	}
	p := pfxs[0]
	lt.Insert(p)

	// Subprefix should overlap
	subBits := p.Bits() + 8
	if subBits > 56 {
		subBits = p.Bits()
	}
	sub := netip.PrefixFrom(p.Addr(), subBits)
	if ok := lt.OverlapsPrefix(sub); !ok {
		t.Fatalf("expected OverlapsPrefix to be true for subprefix %v of %v", sub, p)
	}

	// Superprefix should overlap
	superBits := p.Bits() - 4
	if superBits < 16 {
		superBits = p.Bits()
	}
	super := netip.PrefixFrom(p.Addr(), superBits)
	if ok := lt.OverlapsPrefix(super); !ok {
		t.Fatalf("expected OverlapsPrefix to be true for superprefix %v of %v", super, p)
	}

	// Non-overlapping: pick an address outside global 2000::/3 (e.g., fc00::/7 is ULA)
	nonGlobal := mpp("fc00::/7")
	if ok := lt.OverlapsPrefix(nonGlobal); ok {
		t.Fatalf("expected no overlap with ULA space %v", nonGlobal)
	}
}

func TestRandomRealWorldPrefixes4_ConstraintsAndUniqueness(t *testing.T) {
	t.Parallel()

	prng := rand.New(rand.NewPCG(42, 42))
	n := 256
	pfxs := randomRealWorldPrefixes4(prng, n)
	if len(pfxs) != n {
		t.Fatalf("expected %d prefixes, got %d", n, len(pfxs))
	}
	seen := make(map[netip.Prefix]struct{}, n)
	reserved := mpp("240.0.0.0/8")

	for _, p := range pfxs {
		if _, dup := seen[p]; dup {
			t.Fatalf("duplicate prefix returned: %v", p)
		}
		seen[p] = struct{}{}

		if p.Bits() < 8 || p.Bits() > 28 {
			t.Fatalf("mask size out of bounds for IPv4: %v", p)
		}
		if p.Overlaps(reserved) {
			t.Fatalf("unexpected reserved/experimental range included: %v", p)
		}
		if !p.Addr().Is4() {
			t.Fatalf("non-IPv4 prefix returned: %v", p)
		}
	}
}

func TestRandomRealWorldPrefixes6_ConstraintsAndUniqueness(t *testing.T) {
	t.Parallel()

	prng := rand.New(rand.NewPCG(42, 42))
	n := 256
	pfxs := randomRealWorldPrefixes6(prng, n)
	if len(pfxs) != n {
		t.Fatalf("expected %d prefixes, got %d", n, len(pfxs))
	}
	seen := make(map[netip.Prefix]struct{}, n)
	global := mpp("2000::/3")
	upper := mpp("2c0f::/16").Addr() // limit used in generator

	for _, p := range pfxs {
		if _, dup := seen[p]; dup {
			t.Fatalf("duplicate prefix returned: %v", p)
		}
		seen[p] = struct{}{}

		if p.Bits() < 16 || p.Bits() > 56 {
			t.Fatalf("mask size out of bounds for IPv6: %v", p)
		}
		if !p.Overlaps(global) {
			t.Fatalf("expected global unicast range, got %v", p)
		}
		// Ensure address is not above upper bound used in generator
		if p.Addr().Compare(upper) == 1 {
			t.Fatalf("address %v exceeds upper bound %v", p.Addr(), upper)
		}
		if p.Addr().Is4() {
			t.Fatalf("IPv4 prefix returned in IPv6 generator: %v", p)
		}
	}
}

func TestRandomRealWorldPrefixes_MixedAndShuffled(t *testing.T) {
	t.Parallel()

	prng := rand.New(rand.NewPCG(42, 42))
	n := 51
	pfxs := randomRealWorldPrefixes(prng, n)
	if len(pfxs) != n {
		t.Fatalf("expected %d prefixes, got %d", n, len(pfxs))
	}
	// Should contain at least one v4 and one v6 when n > 1
	has4, has6 := false, false
	for _, p := range pfxs {
		if p.Addr().Is4() {
			has4 = true
		} else {
			has6 = true
		}
	}
	if !has4 || !has6 {
		t.Fatalf("expected mixed families (has4=%v, has6=%v) for n=%d", has4, has6, n)
	}
}
