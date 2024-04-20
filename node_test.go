// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause
//
// tests and benchmarks copied from github.com/tailscale/art
// and modified for this implementation by:
//
// Copyright (c) Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"fmt"
	"runtime"
	"sort"
	"testing"
)

func TestInverseIndex(t *testing.T) {
	t.Parallel()
	for i := 0; i < maxNodeChildren; i++ {
		for bits := 0; bits <= strideLen; bits++ {
			octet := i & (0xFF << (strideLen - bits))
			idx := prefixToBaseIndex(uint(octet), bits)
			octet2, len2 := baseIndexToPrefix(idx)
			if octet2 != uint(octet) || len2 != bits {
				t.Errorf("inverse(index(%d/%d)) != %d/%d", octet, bits, octet2, len2)
			}
		}
	}
}

func TestFringeIndex(t *testing.T) {
	t.Parallel()
	for i := 0; i < maxNodeChildren; i++ {
		got := octetToBaseIndex(uint(i))
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
	slow := slowST[int]{pfxs}
	fast := newNode[int]()

	for _, pfx := range pfxs {
		fast.prefixes.insert(uint(pfx.octet), pfx.bits, pfx.val)
	}

	for i := 0; i < 256; i++ {
		octet := uint(i)
		slowVal, slowOK := slow.lpm(octet)
		_, fastVal, fastOK := fast.prefixes.lpmByOctet(octet)
		if !getsEqual(fastVal, fastOK, slowVal, slowOK) {
			t.Fatalf("get(%d) = (%v, %v), want (%v, %v)", octet, fastVal, fastOK, slowVal, slowOK)
		}
	}
}

func TestPrefixDelete(t *testing.T) {
	t.Parallel()
	// Compare route deletion to our reference slowTable.
	pfxs := shufflePrefixes(allPrefixes())[:100]
	slow := slowST[int]{pfxs}
	fast := newNode[int]()

	for _, pfx := range pfxs {
		fast.prefixes.insert(pfx.octet, pfx.bits, pfx.val)
	}

	toDelete := pfxs[:50]
	for _, pfx := range toDelete {
		slow.delete(pfx.octet, pfx.bits)
		fast.prefixes.delete(pfx.octet, pfx.bits)
	}

	// Sanity check that slowTable seems to have done the right thing.
	if cnt := len(slow.entries); cnt != 50 {
		t.Fatalf("slowTable has %d entries after deletes, want 50", cnt)
	}

	for i := 0; i < 256; i++ {
		octet := uint(i)
		slowVal, slowOK := slow.lpm(octet)
		_, fastVal, fastOK := fast.prefixes.lpmByOctet(octet)
		if !getsEqual(fastVal, fastOK, slowVal, slowOK) {
			t.Fatalf("get(%d) = (%v, %v), want (%v, %v)", octet, fastVal, fastOK, slowVal, slowOK)
		}
	}
}

func TestPrefixOverlaps(t *testing.T) {
	t.Parallel()

	pfxs := shufflePrefixes(allPrefixes())[:100]
	slow := slowST[int]{pfxs}
	fast := newNode[int]()

	for _, pfx := range pfxs {
		fast.prefixes.insert(pfx.octet, pfx.bits, pfx.val)
	}

	for _, tt := range allPrefixes() {
		slowOK := slow.overlapsPrefix(uint8(tt.octet), tt.bits)
		fastOK := fast.overlapsPrefix(tt.octet, tt.bits)
		if slowOK != fastOK {
			t.Fatalf("overlapsPrefix(%d, %d) = %v, want %v", tt.octet, tt.bits, fastOK, slowOK)
		}
	}
}

func TestNodeOverlaps(t *testing.T) {
	t.Parallel()

	// Empirically, between 5 and 6 routes per table results in ~50%
	// of random pairs overlapping. Cool example of the birthday
	// paradox!
	const numEntries = 6
	all := allPrefixes()

	seenResult := map[bool]int{}
	for i := 0; i < 100_000; i++ {
		shufflePrefixes(all)
		pfxs := all[:numEntries]
		slow := slowST[int]{pfxs}
		fast := newNode[int]()
		for _, pfx := range pfxs {
			fast.prefixes.insert(pfx.octet, pfx.bits, pfx.val)
		}

		inter := all[numEntries : 2*numEntries]
		slowInter := slowST[int]{inter}
		fastInter := newNode[int]()
		for _, pfx := range inter {
			fastInter.prefixes.insert(pfx.octet, pfx.bits, pfx.val)
		}

		gotSlow := slow.overlaps(&slowInter)
		gotFast := fast.overlapsRec(fastInter)
		if gotSlow != gotFast {
			t.Fatalf("node.overlaps = %v, want %v", gotFast, gotSlow)
		}
		seenResult[gotFast]++
	}
	t.Log(seenResult)
	if len(seenResult) != 2 { // saw both intersections and non-intersections
		t.Fatalf("didn't see both intersections and non-intersections\nIntersects: %d\nNon-intersects: %d", seenResult[true], seenResult[false])
	}
}

var prefixRouteCount = []int{10, 20, 50, 100, 200, 500}

// forPrefixCount runs the benchmark fn with different sets of routes.
func forPrefixCount(b *testing.B, fn func(b *testing.B, routes []slowSTEntry[int])) {
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

			routes := append([]slowSTEntry[int](nil), routes[:nroutes]...)
			b.Run("random_order", runAndRecord)
			sort.Slice(routes, func(i, j int) bool {
				if routes[i].bits < routes[j].bits {
					return true
				}
				return routes[i].octet < routes[j].octet
			})
		})
	}
}

func BenchmarkPrefixInsertion(b *testing.B) {
	forPrefixCount(b, func(b *testing.B, routes []slowSTEntry[int]) {
		val := 0
		for i := 0; i < b.N; i++ {
			rt := newNode[int]()
			for _, route := range routes {
				rt.prefixes.insert(route.octet, route.bits, val)
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
	forPrefixCount(b, func(b *testing.B, routes []slowSTEntry[int]) {
		val := 0
		rt := newNode[int]()
		for _, route := range routes {
			rt.prefixes.insert(route.octet, route.bits, val)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			rt2 := rt
			for _, route := range routes {
				rt2.prefixes.delete(route.octet, route.bits)
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

func BenchmarkPrefixLPM(b *testing.B) {
	forPrefixCount(b, func(b *testing.B, routes []slowSTEntry[int]) {
		val := 0
		rt := newNode[int]()
		for _, route := range routes {
			rt.prefixes.insert(route.octet, route.bits, val)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, writeSink, _ = rt.prefixes.lpmByOctet(uint(uint8(i)))
		}

		lpm := float64(b.N)
		elapsed := float64(b.Elapsed().Nanoseconds())
		b.ReportMetric(elapsed/lpm, "ns/op")
	})
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
