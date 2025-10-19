package bart

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"iter"
	"math/rand/v2"
	"net/netip"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/gaissmai/bart/internal/tests/random"
)

// location of the full tier1 routing info
const prefixFile = "./internal/tests/testdata/prefixes.txt.gz"

// workLoadN to adjust loops for tests with -short
func workLoadN() int {
	if testing.Short() {
		return 100
	}
	return 1_000
}

// this file contains helpers for other test functions

// holds the tier1 routes
var tier1 = &tier1T{}

type tier1T struct {
	_once sync.Once // load and parse it only once

	_routes  []netip.Prefix // all tier1 routes
	_routes4 []netip.Prefix // all v4 tier1 routes
	_routes6 []netip.Prefix // all v6 tier1 routes

	_matchIP4 netip.Addr // matching v4 address in tier1 routes
	_matchIP6 netip.Addr // matching v6 address in tier1 routes

	_missIP4 netip.Addr // missing v4 address in tier1 routes
	_missIP6 netip.Addr // missing v6 address in tier1 routes

	_matchPfx4 netip.Prefix // matching v4 prefix in tier1 routes
	_matchPfx6 netip.Prefix // matching v6 prefix in tier1 routes

	_missPfx4 netip.Prefix // missing v4 prefix in tier1 routes
	_missPfx6 netip.Prefix // missing v6 prefix in tier1 routes
}

// abbreviation
var mpa = netip.MustParseAddr

// abbreviation and apnic on non masked input
var mpp = func(s string) netip.Prefix {
	pfx := netip.MustParsePrefix(s)
	if pfx == pfx.Masked() {
		return pfx
	}
	panic(fmt.Sprintf("%s is not canonicalized as %s", s, pfx.Masked()))
}

// #########################################################

// tests for deep copies with Cloner interface
type MyInt int

// implement the Cloner interface
func (i *MyInt) Clone() *MyInt {
	a := *i
	return &a
}

// Helper functions
func countDumpListNodes[V any](nodes []DumpListNode[V]) int {
	count := len(nodes)
	for _, node := range nodes {
		count += countDumpListNodes(node.Subnets)
	}
	return count
}

func verifyAllIPv4Nodes[V any](t *testing.T, nodes []DumpListNode[V]) {
	for i, node := range nodes {
		if !node.CIDR.Addr().Is4() {
			t.Errorf("Node %d is not IPv4 prefix: %v", i, node.CIDR)
		}
		// Recursively check subnets
		verifyAllIPv4Nodes(t, node.Subnets)
	}
}

func verifyAllIPv6Nodes[V any](t *testing.T, nodes []DumpListNode[V]) {
	for i, node := range nodes {
		if !node.CIDR.Addr().Is6() {
			t.Errorf("Node %d is not IPv6 prefix: %v", i, node.CIDR)
		}
		// Recursively check subnets
		verifyAllIPv6Nodes(t, node.Subnets)
	}
}

func mustPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatalf("%s must panic", name)
		}
	}()
	fn()
}

func noPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("%s panicked: %v", name, r)
		}
	}()
	fn()
}

func noPanicRangeOverFunc[V any](t *testing.T, name string, fn any) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("%s panicked: %v", name, r)
		}
	}()

	switch iterFunc := fn.(type) {
	case func() iter.Seq[netip.Prefix]:
		for range iterFunc() {
		}
	case func() iter.Seq2[netip.Prefix, V]:
		for range iterFunc() {
		}
	case func(netip.Prefix) iter.Seq[netip.Prefix]:
		pfx := mpp("1.2.3.4/32")
		for range iterFunc(pfx) {
		}
	case func(netip.Prefix) iter.Seq2[netip.Prefix, V]:
		pfx := mpp("1.2.3.4/32")
		for range iterFunc(pfx) {
		}
	default:
		t.Fatalf("%s unknown iter function: %T", name, iterFunc)
	}
}

// init parses the tier1 route table once at first use and caches it
func (t1 *tier1T) init() {
	t1._once.Do(func() {
		file, err := os.Open(prefixFile)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		rgz, err := gzip.NewReader(file)
		if err != nil {
			panic(err)
		}
		defer rgz.Close()

		scanner := bufio.NewScanner(rgz)
		for scanner.Scan() {
			line := scanner.Text()
			line = strings.TrimSpace(line)

			cidr := netip.MustParsePrefix(line)
			cidr = cidr.Masked()

			t1._routes = append(t1._routes, cidr)
		}

		if err = scanner.Err(); err != nil {
			panic(fmt.Errorf("reading %s, %w", prefixFile, err))
		}

		// shuffle the routes
		prng := rand.New(rand.NewPCG(42, 42))
		prng.Shuffle(len(t1._routes), func(i, j int) {
			t1._routes[i], t1._routes[j] = t1._routes[j], t1._routes[i]
		})

		// split into v4 and v6 prefixes
		for _, pfx := range t1._routes {
			switch pfx.Addr().Is4() {
			case true:
				t1._routes4 = append(t1._routes4, pfx)
			case false:
				t1._routes6 = append(t1._routes6, pfx)
			}
		}

		// calculate match/miss info once
		t1._doMatchIP4()
		t1._doMatchIP6()

		t1._doMissIP4()
		t1._doMissIP6()

		t1._doMatchPfx4(prng)
		t1._doMatchPfx6(prng)

		t1._doMissPfx4(prng)
		t1._doMissPfx6(prng)
	})
}

// getters for the tier1 routing info
func (t1 *tier1T) routes() []netip.Prefix {
	t1.init()
	return t1._routes
}

func (t1 *tier1T) routes4() []netip.Prefix {
	t1.init()
	return t1._routes4
}

func (t1 *tier1T) routes6() []netip.Prefix {
	t1.init()
	return t1._routes6
}

func (t1 *tier1T) matchIP4() netip.Addr {
	t1.init()
	return t1._matchIP4
}

func (t1 *tier1T) matchIP6() netip.Addr {
	t1.init()
	return t1._matchIP6
}

func (t1 *tier1T) missIP4() netip.Addr {
	t1.init()
	return t1._missIP4
}

func (t1 *tier1T) missIP6() netip.Addr {
	t1.init()
	return t1._missIP6
}

func (t1 *tier1T) matchPfx4() netip.Prefix {
	t1.init()
	return t1._matchPfx4
}

func (t1 *tier1T) matchPfx6() netip.Prefix {
	t1.init()
	return t1._matchPfx6
}

func (t1 *tier1T) missPfx4() netip.Prefix {
	t1.init()
	return t1._missPfx4
}

func (t1 *tier1T) missPfx6() netip.Prefix {
	t1.init()
	return t1._missPfx6
}

// process the match/miss info once during t1.init()
func (t1 *tier1T) _doMatchIP4() {
	lt := new(Lite)
	for _, pfx := range t1._routes4 {
		lt.Insert(pfx)
	}

	count := 0
	for _, pfx := range t1._routes4 {
		probe := pfx.Addr().Next()
		if !probe.IsValid() {
			continue
		}

		if ok := lt.Contains(probe); ok {
			t1._matchIP4 = probe
			break
		}

		if count++; count > 1_000_000 {
			panic("find matching IP4, giving up!")
		}
	}
}

func (t1 *tier1T) _doMatchIP6() {
	lt := new(Lite)
	for _, pfx := range t1._routes6 {
		lt.Insert(pfx)
	}

	count := 0
	for _, pfx := range t1._routes6 {
		probe := pfx.Addr().Next()
		if !probe.IsValid() {
			continue
		}

		if ok := lt.Contains(probe); ok {
			t1._matchIP6 = probe
			break
		}

		if count++; count > 1_000_000 {
			panic("find matching IP6, giving up!")
		}
	}
}

func (t1 *tier1T) _doMissIP4() {
	lt := new(Lite)
	for _, pfx := range t1._routes4 {
		lt.Insert(pfx)
	}

	count := 0
	for _, pfx := range t1._routes4 {
		probe := pfx.Addr().Prev()
		if !probe.IsValid() {
			continue
		}

		if ok := lt.Contains(probe); !ok {
			t1._missIP4 = probe
			break
		}

		if count++; count > 1_000_000 {
			panic("find missing IP4, giving up!")
		}
	}
}

func (t1 *tier1T) _doMissIP6() {
	lt := new(Lite)
	for _, pfx := range t1._routes6 {
		lt.Insert(pfx)
	}

	count := 0
	for _, pfx := range t1._routes6 {
		probe := pfx.Addr().Prev()
		if !probe.IsValid() {
			continue
		}

		if ok := lt.Contains(probe); !ok {
			t1._missIP6 = probe
			break
		}

		if count++; count > 1_000_000 {
			panic("find missing IP6, giving up!")
		}
	}
}

func (t1 *tier1T) _doMatchPfx4(prng *rand.Rand) {
	lt := new(Lite)
	for _, pfx := range t1._routes4 {
		lt.Insert(pfx)
	}

	count := 0
	for {
		probe := random.Prefix4(prng)
		if !probe.IsValid() {
			continue
		}

		if ok := lt.LookupPrefix(probe); ok {
			t1._matchPfx4 = probe
			break
		}

		if count++; count > 1_000_000 {
			panic("find matching Pfx4, giving up!")
		}
	}
}

func (t1 *tier1T) _doMatchPfx6(prng *rand.Rand) {
	lt := new(Lite)
	for _, pfx := range t1._routes6 {
		lt.Insert(pfx)
	}

	count := 0
	for {
		probe := random.Prefix6(prng)
		if !probe.IsValid() {
			continue
		}

		if ok := lt.LookupPrefix(probe); ok {
			t1._matchPfx6 = probe
			break
		}

		if count++; count > 1_000_000 {
			panic("find matching Pfx6, giving up!")
		}
	}
}

func (t1 *tier1T) _doMissPfx4(prng *rand.Rand) {
	lt := new(Lite)
	for _, pfx := range t1._routes4 {
		lt.Insert(pfx)
	}

	count := 0
	for {
		probe := random.Prefix4(prng)
		if !probe.IsValid() {
			continue
		}

		if ok := lt.LookupPrefix(probe); !ok {
			t1._missPfx4 = probe
			break
		}

		if count++; count > 1_000_000 {
			panic("find matching Pfx4, giving up!")
		}
	}
}

func (t1 *tier1T) _doMissPfx6(prng *rand.Rand) {
	lt := new(Lite)
	for _, pfx := range t1._routes6 {
		lt.Insert(pfx)
	}

	count := 0
	for {
		probe := random.Prefix6(prng)
		if !probe.IsValid() {
			continue
		}

		if ok := lt.LookupPrefix(probe); !ok {
			t1._missPfx6 = probe
			break
		}

		if count++; count > 1_000_000 {
			panic("find matching Pfx6, giving up!")
		}
	}
}
