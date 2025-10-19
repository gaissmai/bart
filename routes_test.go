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

	"github.com/gaissmai/bart/internal/tests/random"
)

const prefixFile = "./testdata/prefixes.txt.gz"

type tier1T struct {
	_once sync.Once

	_routes  []netip.Prefix
	_routes4 []netip.Prefix
	_routes6 []netip.Prefix

	_matchIP4 netip.Addr
	_matchIP6 netip.Addr

	_missIP4 netip.Addr
	_missIP6 netip.Addr

	_matchPfx4 netip.Prefix
	_matchPfx6 netip.Prefix

	_missPfx4 netip.Prefix
	_missPfx6 netip.Prefix
}

var tier1 = &tier1T{}

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

// tier1 parses the testdata route table (once) and returns the prefixes.
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
