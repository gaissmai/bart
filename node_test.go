// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause
//
// tests and benchmarks copied from github.com/tailscale/art
// and modified for this implementation by:
//
// Copyright (c) Karl Gaissmaier
// SPDX-License-Identifier: BSD-3-Clause

package bart

import (
	"bytes"
	"fmt"
	"math/rand"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestInverseIndex(t *testing.T) {
	t.Parallel()
	for i := 0; i < maxNodeChilds; i++ {
		for len := 0; len <= stride; len++ {
			addr := i & (0xFF << (stride - len))
			idx := prefixToBaseIndex(uint(addr), len)
			addr2, len2 := baseIndexToPrefix(idx)
			if addr2 != uint(addr) || len2 != len {
				t.Errorf("inverse(index(%d/%d)) != %d/%d", addr, len, addr2, len2)
			}
		}
	}
}

func TestFringeIndex(t *testing.T) {
	t.Parallel()
	for i := 0; i < maxNodeChilds; i++ {
		got := addrToBaseIndex(uint(i))
		want := prefixToBaseIndex(uint(i), 8)
		if got != want {
			t.Errorf("fringeIndex(%d) = %d, want %d", i, got, want)
		}
	}
}

func TestPrefixInsert(t *testing.T) {
	t.Parallel()
	// Verify that lookup results after a bunch of inserts exactly
	// match those of a naive implementation that just scans all prefixes on
	// every lookup. The naive implementation is very slow, but its behavior is
	// easy to verify by inspection.

	pfxs := shufflePrefixes(allPrefixes())[:100]
	slow := slowTable[int]{pfxs}
	fast := newNode[int]()

	for _, pfx := range pfxs {
		fast.prefixes.insert(uint(pfx.addr), pfx.len, pfx.val)
	}

	for i := 0; i < 256; i++ {
		addr := uint(i)
		slowVal, slowOK := slow.get(addr)
		_, fastVal, fastOK := fast.prefixes.lpmByAddr(addr)
		if !getsEqual(fastVal, fastOK, slowVal, slowOK) {
			t.Fatalf("get(%d) = (%v, %v), want (%v, %v)", addr, fastVal, fastOK, slowVal, slowOK)
		}
	}
}

func TestPrefixDelete(t *testing.T) {
	t.Parallel()
	// Compare route deletion to our reference slowTable.
	pfxs := shufflePrefixes(allPrefixes())[:100]
	slow := slowTable[int]{pfxs}
	fast := newNode[int]()

	for _, pfx := range pfxs {
		fast.prefixes.insert(pfx.addr, pfx.len, pfx.val)
	}

	toDelete := pfxs[:50]
	for _, pfx := range toDelete {
		slow.delete(pfx.addr, pfx.len)
		fast.prefixes.delete(pfx.addr, pfx.len)
	}

	// Sanity check that slowTable seems to have done the right thing.
	if cnt := len(slow.prefixes); cnt != 50 {
		t.Fatalf("slowTable has %d entries after deletes, want 50", cnt)
	}

	for i := 0; i < 256; i++ {
		addr := uint(i)
		slowVal, slowOK := slow.get(addr)
		_, fastVal, fastOK := fast.prefixes.lpmByAddr(addr)
		if !getsEqual(fastVal, fastOK, slowVal, slowOK) {
			t.Fatalf("get(%d) = (%v, %v), want (%v, %v)", addr, fastVal, fastOK, slowVal, slowOK)
		}
	}
}

var prefixRouteCount = []int{10, 50, 100, 200}

// forPrefixCount runs the benchmark fn with different sets of routes.
func forPrefixCount(b *testing.B, fn func(b *testing.B, routes []slowEntry[int])) {
	routes := shufflePrefixes(allPrefixes())
	for _, nroutes := range prefixRouteCount {
		b.Run(fmt.Sprint(nroutes), func(b *testing.B) {
			runAndRecord := func(b *testing.B) {
				b.ReportAllocs()
				var startMem, endMem runtime.MemStats
				runtime.ReadMemStats(&startMem)
				fn(b, routes)
				runtime.ReadMemStats(&endMem)
				ops := float64(b.N) * float64(len(routes))
				allocs := float64(endMem.Mallocs - startMem.Mallocs)
				bytes := float64(endMem.TotalAlloc - startMem.TotalAlloc)
				b.ReportMetric(roundFloat64(allocs/ops), "allocs/op")
				b.ReportMetric(roundFloat64(bytes/ops), "B/op")
			}

			routes := append([]slowEntry[int](nil), routes[:nroutes]...)
			b.Run("random_order", runAndRecord)
			sort.Slice(routes, func(i, j int) bool {
				if routes[i].len < routes[j].len {
					return true
				}
				return routes[i].addr < routes[j].addr
			})
		})
	}
}

func BenchmarkPrefixInsertion(b *testing.B) {
	forPrefixCount(b, func(b *testing.B, routes []slowEntry[int]) {
		val := 0
		for i := 0; i < b.N; i++ {
			rt := newNode[int]()
			for _, route := range routes {
				rt.prefixes.insert(route.addr, route.len, val)
			}
		}
		inserts := float64(b.N) * float64(len(routes))
		elapsed := float64(b.Elapsed().Nanoseconds())
		elapsedSec := b.Elapsed().Seconds()
		b.ReportMetric(elapsed/inserts, "ns/op")
		b.ReportMetric(inserts/elapsedSec, "routes/s")
	})
}

func BenchmarkPrefixDeletion(b *testing.B) {
	forPrefixCount(b, func(b *testing.B, routes []slowEntry[int]) {
		val := 0
		rt := newNode[int]()
		for _, route := range routes {
			rt.prefixes.insert(route.addr, route.len, val)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			rt2 := rt
			for _, route := range routes {
				rt2.prefixes.delete(route.addr, route.len)
			}
		}
		deletes := float64(b.N) * float64(len(routes))
		elapsed := float64(b.Elapsed().Nanoseconds())
		elapsedSec := b.Elapsed().Seconds()
		b.ReportMetric(elapsed/deletes, "ns/op")
		b.ReportMetric(deletes/elapsedSec, "routes/s")
	})
}

var writeSink int

func BenchmarkPrefixGet(b *testing.B) {
	// No need to forCountAndOrdering here, route lookup time is independent of
	// the route count.
	routes := shufflePrefixes(allPrefixes())[:100]
	rt := newNode[int]()
	for _, route := range routes {
		rt.prefixes.insert(route.addr, route.len, route.val)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, writeSink, _ = rt.prefixes.lpmByAddr(uint(i))
	}
	gets := float64(b.N)
	elapsedSec := b.Elapsed().Seconds()
	b.ReportMetric(gets/elapsedSec, "routes/s")
}

// slowTable is an 8-bit routing table implemented as a set of prefixes that are
// explicitly scanned in full for every route lookup. It is very slow, but also
// reasonably easy to verify by inspection, and so a good comparison target for
// strideTable.
type slowTable[V any] struct {
	prefixes []slowEntry[V]
}

type slowEntry[V any] struct {
	addr uint
	len  int
	val  V
}

func (t *slowTable[V]) String() string {
	pfxs := append([]slowEntry[V](nil), t.prefixes...)
	sort.Slice(pfxs, func(i, j int) bool {
		if pfxs[i].len != pfxs[j].len {
			return pfxs[i].len < pfxs[j].len
		}
		return pfxs[i].addr < pfxs[j].addr
	})
	var ret bytes.Buffer
	for _, pfx := range pfxs {
		fmt.Fprintf(&ret, "%3d/%d (%08b/%08b) = %v\n", pfx.addr, pfx.len, pfx.addr, pfxMask(pfx.len), pfx.val)
	}
	return ret.String()
}

func (t *slowTable[V]) insert(addr uint, prefixLen int, val V) {
	t.delete(addr, prefixLen) // no-op if prefix doesn't exist

	t.prefixes = append(t.prefixes, slowEntry[V]{addr, prefixLen, val})
}

func (t *slowTable[V]) delete(addr uint, prefixLen int) {
	pfx := make([]slowEntry[V], 0, len(t.prefixes))
	for _, e := range t.prefixes {
		if e.addr == addr && e.len == prefixLen {
			continue
		}
		pfx = append(pfx, e)
	}
	t.prefixes = pfx
}

func (t *slowTable[V]) get(addr uint) (ret V, ok bool) {
	curLen := -1
	for _, e := range t.prefixes {
		if addr&pfxMask(e.len) == e.addr && e.len >= curLen {
			ret = e.val
			curLen = e.len
		}
	}
	return ret, curLen != -1
}

func pfxMask(pfxLen int) uint {
	return 0xFF << (stride - pfxLen)
}

func allPrefixes() []slowEntry[int] {
	ret := make([]slowEntry[int], 0, maxNodePrefixes-1)
	for i := 1; i < maxNodePrefixes; i++ {
		a, l := baseIndexToPrefix(uint(i))
		ret = append(ret, slowEntry[int]{a, l, i})
	}
	return ret
}

func shufflePrefixes(pfxs []slowEntry[int]) []slowEntry[int] {
	rand.Shuffle(len(pfxs), func(i, j int) { pfxs[i], pfxs[j] = pfxs[j], pfxs[i] })
	return pfxs
}

func formatSlowEntriesShort[V any](ents []slowEntry[V]) string {
	var ret []string
	for _, ent := range ents {
		ret = append(ret, fmt.Sprintf("%d/%d", ent.addr, ent.len))
	}
	return "[" + strings.Join(ret, " ") + "]"
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
