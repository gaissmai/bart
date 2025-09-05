package bart

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"log"
	"math/rand/v2"
	"net/netip"
	"os"
	"strings"
)

// full internet prefix list, gzipped
const prefixFile = "testdata/prefixes.txt.gz"

var benchRouteCount = []int{10, 100, 1_000, 10_000, 100_000, 1_000_000}

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

var mpa = netip.MustParseAddr

var mpp = func(s string) netip.Prefix {
	pfx := netip.MustParsePrefix(s)
	if pfx == pfx.Masked() {
		return pfx
	}
	panic(fmt.Sprintf("%s is not canonicalized as %s", s, pfx.Masked()))
}

type stringTest struct {
	cidrs []netip.Prefix
	want  string
}

type tableTest struct {
	// addr is an IP address string to look up in a route table.
	addr string
	// want is the expected >=0 value associated with the route, or -1
	// if we expect a lookup miss.
	want int
}

func getsEqual[V comparable](a V, aOK bool, b V, bOK bool) bool {
	if !aOK && !bOK {
		return true
	}
	if aOK != bOK {
		return false
	}
	return a == b
}

func init() {
	prng := rand.New(rand.NewPCG(42, 42))
	fillRouteTables()

	randRoute4 = routes4[prng.IntN(len(routes4))]
	randRoute6 = routes6[prng.IntN(len(routes6))]
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
