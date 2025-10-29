// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"math/rand/v2"
	"net/netip"
	"os"
	"strings"
	"sync"
)

// location of the full tier1 routing info
const prefixFile = "./internal/tests/testdata/prefixes.txt.gz"

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

// holds the tier1 routes
var tier1 = &tier1T{}

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

		t1._doMatchPfx4()
		t1._doMatchPfx6()

		t1._doMissPfx4()
		t1._doMissPfx6()
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

	for _, pfx := range t1._routes4 {
		probe := pfx.Addr().Next()
		if !probe.IsValid() {
			continue
		}

		if ok := lt.Contains(probe); ok {
			t1._matchIP4 = probe
			return
		}

	}
	panic(fmt.Sprintf("no matching IPv4 address found after checking %d routes", len(t1._routes4)))
}

func (t1 *tier1T) _doMatchIP6() {
	lt := new(Lite)
	for _, pfx := range t1._routes6 {
		lt.Insert(pfx)
	}

	for _, pfx := range t1._routes6 {
		probe := pfx.Addr().Next()
		if !probe.IsValid() {
			continue
		}

		if ok := lt.Contains(probe); ok {
			t1._matchIP6 = probe
			return
		}

	}
	panic(fmt.Sprintf("no matching IPv6 address found after checking %d routes", len(t1._routes6)))
}

func (t1 *tier1T) _doMissIP4() {
	lt := new(Lite)
	for _, pfx := range t1._routes4 {
		lt.Insert(pfx)
	}

	for _, pfx := range t1._routes4 {
		probe := pfx.Addr().Prev()
		if !probe.IsValid() {
			continue
		}

		if ok := lt.Contains(probe); !ok {
			t1._missIP4 = probe
			return
		}
	}
	panic(fmt.Sprintf("no missing IPv4 address found after checking %d routes", len(t1._routes4)))
}

func (t1 *tier1T) _doMissIP6() {
	lt := new(Lite)
	for _, pfx := range t1._routes6 {
		lt.Insert(pfx)
	}

	for _, pfx := range t1._routes6 {
		probe := pfx.Addr().Prev()
		if !probe.IsValid() {
			continue
		}

		if ok := lt.Contains(probe); !ok {
			t1._missIP6 = probe
			return
		}
	}
	panic(fmt.Sprintf("no missing IPv6 address found after checking %d routes", len(t1._routes6)))
}

func (t1 *tier1T) _doMatchPfx4() {
	lt := new(Lite)
	for _, pfx := range t1._routes4 {
		lt.Insert(pfx)
	}

	for _, pfx := range t1._routes4 {
		probe := netip.PrefixFrom(pfx.Addr(), pfx.Bits()+1)
		if !probe.IsValid() {
			continue
		}

		if ok := lt.LookupPrefix(probe); ok {
			t1._matchPfx4 = probe
			return
		}

	}
	panic(fmt.Sprintf("no matching IPv4 prefix found after checking %d routes", len(t1._routes4)))
}

func (t1 *tier1T) _doMatchPfx6() {
	lt := new(Lite)
	for _, pfx := range t1._routes6 {
		lt.Insert(pfx)
	}

	for _, pfx := range t1._routes6 {
		probe := netip.PrefixFrom(pfx.Addr(), pfx.Bits()+1)
		if !probe.IsValid() {
			continue
		}

		if ok := lt.LookupPrefix(probe); ok {
			t1._matchPfx6 = probe
			return
		}
	}
	panic(fmt.Sprintf("no matching IPv6 prefix found after checking %d routes", len(t1._routes6)))
}

func (t1 *tier1T) _doMissPfx4() {
	lt := new(Lite)
	for _, pfx := range t1._routes4 {
		lt.Insert(pfx)
	}

	for _, pfx := range t1._routes4 {
		probe := netip.PrefixFrom(pfx.Addr(), pfx.Bits()-1)
		if !probe.IsValid() {
			continue
		}

		if ok := lt.LookupPrefix(probe); !ok {
			t1._missPfx4 = probe
			return
		}
	}
	panic(fmt.Sprintf("no missing IPv6 prefix found after checking %d routes", len(t1._routes4)))
}

func (t1 *tier1T) _doMissPfx6() {
	lt := new(Lite)
	for _, pfx := range t1._routes6 {
		lt.Insert(pfx)
	}

	for _, pfx := range t1._routes6 {
		probe := netip.PrefixFrom(pfx.Addr(), pfx.Bits()-1)
		if !probe.IsValid() {
			continue
		}

		if ok := lt.LookupPrefix(probe); !ok {
			t1._missPfx6 = probe
			return
		}
	}
	panic(fmt.Sprintf("no missing IPv6 prefix found after checking %d routes", len(t1._routes6)))
}
