package bart

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"iter"
	"log"
	"math/rand/v2"
	"net/netip"
	"os"
	"strings"
	"testing"

	"github.com/gaissmai/bart/internal/golden"
)

// this file contains init functions and helpers for test functions

var mpa = netip.MustParseAddr

var mpp = func(s string) netip.Prefix {
	pfx := netip.MustParsePrefix(s)
	if pfx == pfx.Masked() {
		return pfx
	}
	panic(fmt.Sprintf("%s is not canonicalized as %s", s, pfx.Masked()))
}

// workLoadN to adjust loops for tests with -short
func workLoadN() int {
	if testing.Short() {
		return 100
	}
	return 1_000
}

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
		matchIP4 = golden.RandomRealWorldPrefixes4(prng, 1)[0].Addr().Next()
		if ok := lt.Contains(matchIP4); ok {
			break
		}
	}
	for {
		matchIP6 = golden.RandomRealWorldPrefixes6(prng, 1)[0].Addr().Next()
		if ok := lt.Contains(matchIP6); ok {
			break
		}
	}

	// find a random match Pfx4
	for {
		matchPfx4 = golden.RandomRealWorldPrefixes4(prng, 1)[0]
		if ok := lt.LookupPrefix(matchPfx4); ok {
			break
		}
	}
	for {
		matchPfx6 = golden.RandomRealWorldPrefixes6(prng, 1)[0]
		if ok := lt.LookupPrefix(matchPfx6); ok {
			break
		}
	}

	for {
		missIP4 = golden.RandomRealWorldPrefixes4(prng, 1)[0].Addr().Next()
		if ok := lt.Contains(missIP4); !ok {
			break
		}
	}
	for {
		missIP6 = golden.RandomRealWorldPrefixes6(prng, 1)[0].Addr().Next()
		if ok := lt.Contains(missIP6); !ok {
			break
		}
	}

	for {
		missPfx4 = golden.RandomRealWorldPrefixes4(prng, 1)[0]
		if ok := lt.LookupPrefix(missPfx4); !ok {
			break
		}
	}
	for {
		missPfx6 = golden.RandomRealWorldPrefixes6(prng, 1)[0]
		if ok := lt.LookupPrefix(missPfx6); !ok {
			break
		}
	}
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

// tests for deep copies with Cloner interface
type MyInt int

// implement the Cloner interface
func (i *MyInt) Clone() *MyInt {
	a := *i
	return &a
}

// A simple type that implements Equaler for testing.
type stringVal string

func (v stringVal) Equal(other stringVal) bool {
	return v == other
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
