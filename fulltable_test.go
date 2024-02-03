package bart_test

import (
	"bufio"
	"compress/gzip"
	crand "crypto/rand"
	"log"
	"math/rand"
	"net/netip"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/gaissmai/bart"
)

// full internet prefix list, gzipped
const prefixFile = "testdata/prefixes.txt.gz"

var (
	routes  []route
	routes4 []route
	routes6 []route
)

type route struct {
	CIDR  netip.Prefix
	Value any
}

func init() {
	fillRouteTables()
}

func TestFullNew(t *testing.T) {
	var startMem, endMem runtime.MemStats
	runtime.ReadMemStats(&startMem)
	nRoutes := make([]route, len(routes))
	copy(nRoutes, routes)
	runtime.ReadMemStats(&endMem)
	rawBytes := endMem.TotalAlloc - startMem.TotalAlloc

	rtBart := bart.Table[any]{}
	runtime.ReadMemStats(&startMem)
	for _, route := range nRoutes {
		rtBart.Insert(route.CIDR, nil)
	}
	runtime.ReadMemStats(&endMem)
	bartBytes := endMem.TotalAlloc - startMem.TotalAlloc

	t.Logf("BART: n: %d routes, raw: %d KBytes, bart: %6d KBytes, mult: %.2f (bart/raw)",
		len(nRoutes), rawBytes/(2<<10), bartBytes/(2<<10), float32(bartBytes)/float32(rawBytes))

	// t.Logf("ART:  n: %d routes, raw: %d KBytes, art:  %6d KBytes, mult: %.2f (art/raw)",
	// 	len(nRoutes), rawBytes/(2<<10), artBytes/(2<<10), float32(artBytes)/float32(rawBytes))
}

func TestFullNewV4(t *testing.T) {
	var startMem, endMem runtime.MemStats
	runtime.ReadMemStats(&startMem)
	nRoutes := make([]route, len(routes4))
	copy(nRoutes, routes4)
	runtime.ReadMemStats(&endMem)
	rawBytes := endMem.TotalAlloc - startMem.TotalAlloc

	rtBart := bart.Table[any]{}
	runtime.ReadMemStats(&startMem)
	for _, route := range nRoutes {
		rtBart.Insert(route.CIDR, nil)
	}
	runtime.ReadMemStats(&endMem)
	bartBytes := endMem.TotalAlloc - startMem.TotalAlloc

	t.Logf("BART: n: %d routes, raw: %d KBytes, bart: %6d KBytes, mult: %.2f (bart/raw)",
		len(nRoutes), rawBytes/(2<<10), bartBytes/(2<<10), float32(bartBytes)/float32(rawBytes))

	// t.Logf("ART:  n: %d routes, raw: %d KBytes, art:  %6d KBytes, mult: %.2f (art/raw)",
	// 	len(nRoutes), rawBytes/(2<<10), artBytes/(2<<10), float32(artBytes)/float32(rawBytes))
}

func TestFullNewV6(t *testing.T) {
	var startMem, endMem runtime.MemStats
	runtime.ReadMemStats(&startMem)
	nRoutes := make([]route, len(routes6))
	copy(nRoutes, routes4)
	runtime.ReadMemStats(&endMem)
	rawBytes := endMem.TotalAlloc - startMem.TotalAlloc

	rtBart := bart.Table[any]{}
	runtime.ReadMemStats(&startMem)
	for _, route := range nRoutes {
		rtBart.Insert(route.CIDR, nil)
	}
	runtime.ReadMemStats(&endMem)
	bartBytes := endMem.TotalAlloc - startMem.TotalAlloc

	t.Logf("BART: n: %d routes, raw: %d KBytes, bart: %6d KBytes, mult: %.2f (bart/raw)",
		len(nRoutes), rawBytes/(2<<10), bartBytes/(2<<10), float32(bartBytes)/float32(rawBytes))

	// t.Logf("ART:  n: %d routes, raw: %d KBytes, art:  %6d KBytes, mult: %.2f (art/raw)",
	// 	len(nRoutes), rawBytes/(2<<10), artBytes/(2<<10), float32(artBytes)/float32(rawBytes))
}

var (
	intSink int
	okSink  bool
)

func BenchmarkFullMatchV4(b *testing.B) {
	var rtBart bart.Table[int]

	for i, route := range routes {
		rtBart.Insert(route.CIDR, i)
	}

	var ip netip.Addr

	// find a random match
	for {
		ip = randomIP4()
		_, ok := rtBart.Get(ip)
		if ok {
			break
		}
	}

	b.Run("Get", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			intSink, okSink = rtBart.Get(ip)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			_, intSink, okSink = rtBart.Lookup(ip)
		}
	})

	b.Run("LookupSCP", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			_, _, okSink = rtBart.LookupShortest(ip)
		}
	})
}

func BenchmarkFullMatchV6(b *testing.B) {
	var rtBart bart.Table[int]

	for i, route := range routes {
		rtBart.Insert(route.CIDR, i)
	}

	var ip netip.Addr

	// find a random match
	for {
		ip = randomIP6()
		_, ok := rtBart.Get(ip)
		if ok {
			break
		}
	}

	b.Run("Get", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			intSink, okSink = rtBart.Get(ip)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			_, intSink, okSink = rtBart.Lookup(ip)
		}
	})

	b.Run("LookupSCP", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			_, _, okSink = rtBart.LookupShortest(ip)
		}
	})
}

func BenchmarkFullMissV4(b *testing.B) {
	var rtBart bart.Table[int]

	for i, route := range routes {
		rtBart.Insert(route.CIDR, i)
	}

	var ip netip.Addr

	for {
		ip = randomIP4()
		_, ok := rtBart.Get(ip)
		if !ok {
			break
		}
	}

	b.Run("Get", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			intSink, okSink = rtBart.Get(ip)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			_, intSink, okSink = rtBart.Lookup(ip)
		}
	})

	b.Run("LookupSCP", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			_, _, okSink = rtBart.LookupShortest(ip)
		}
	})
}

func BenchmarkFullMissV6(b *testing.B) {
	var rtBart bart.Table[int]

	for i, route := range routes {
		rtBart.Insert(route.CIDR, i)
	}

	var ip netip.Addr

	for {
		ip = randomIP6()
		_, ok := rtBart.Get(ip)
		if !ok {
			break
		}
	}

	b.Run("Get", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			intSink, okSink = rtBart.Get(ip)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			_, intSink, okSink = rtBart.Lookup(ip)
		}
	})

	b.Run("LookupSCP", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			_, _, okSink = rtBart.LookupShortest(ip)
		}
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

// randomPrefixes returns n randomly generated prefixes and
// associated values, distributed equally between IPv4 and IPv6.
//
//nolint:unused
func randomPrefix() netip.Prefix {
	if rand.Intn(2) == 1 {
		return randomPrefix4()
	} else {
		return randomPrefix6()
	}
}

//nolint:unused
func randomPrefix4() netip.Prefix {
	bits := rand.Intn(33)
	pfx, err := randomIP4().Prefix(bits)
	if err != nil {
		panic(err)
	}
	return pfx
}

//nolint:unused
func randomPrefix6() netip.Prefix {
	bits := rand.Intn(129)
	pfx, err := randomIP6().Prefix(bits)
	if err != nil {
		panic(err)
	}
	return pfx
}

//nolint:unused
func randomIP() netip.Addr {
	if rand.Intn(2) == 1 {
		return randomIP4()
	} else {
		return randomIP6()
	}
}

//nolint:unused
func randomIP4() netip.Addr {
	var b [4]byte
	if _, err := crand.Read(b[:]); err != nil {
		panic(err)
	}
	return netip.AddrFrom4(b)
}

//nolint:unused
func randomIP6() netip.Addr {
	var b [16]byte
	if _, err := crand.Read(b[:]); err != nil {
		panic(err)
	}
	return netip.AddrFrom16(b)
}
