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
	"bytes"
	"fmt"
	"math/rand"
	"runtime"
	"sort"
	"testing"
)

func TestInverseIndex(t *testing.T) {
	t.Parallel()
	for i := 0; i < maxNodeChildren; i++ {
		for bits := 0; bits <= stride; bits++ {
			addr := i & (0xFF << (stride - bits))
			idx := prefixToBaseIndex(uint(addr), bits)
			addr2, len2 := baseIndexToPrefix(idx)
			if addr2 != uint(addr) || len2 != bits {
				t.Errorf("inverse(index(%d/%d)) != %d/%d", addr, bits, addr2, len2)
			}
		}
	}
}

func TestFringeIndex(t *testing.T) {
	t.Parallel()
	for i := 0; i < maxNodeChildren; i++ {
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
		fast.prefixes.insert(uint(pfx.addr), pfx.bits, pfx.val)
	}

	for i := 0; i < 256; i++ {
		addr := uint(i)
		slowVal, slowOK := slow.get(addr)
		_, fastVal, fastOK := fast.prefixes.lpmByAddr(addr)
		if !getsEqual(fastVal, fastOK, slowVal, slowOK) {
			t.Fatalf("get(%d) = (%v, %v), want (%v, %v)", addr, fastVal, fastOK, slowVal, slowOK)
		}

		slowVal, slowOK = slow.spm(addr)
		_, fastVal, fastOK = fast.prefixes.spmByAddr(addr)
		if !getsEqual(fastVal, fastOK, slowVal, slowOK) {
			t.Fatalf("spm(%d) = (%v, %v), want (%v, %v)", addr, fastVal, fastOK, slowVal, slowOK)
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
		fast.prefixes.insert(pfx.addr, pfx.bits, pfx.val)
	}

	toDelete := pfxs[:50]
	for _, pfx := range toDelete {
		slow.delete(pfx.addr, pfx.bits)
		fast.prefixes.delete(pfx.addr, pfx.bits)
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

		slowVal, slowOK = slow.spm(addr)
		_, fastVal, fastOK = fast.prefixes.spmByAddr(addr)
		if !getsEqual(fastVal, fastOK, slowVal, slowOK) {
			t.Fatalf("spm(%d) = (%v, %v), want (%v, %v)", addr, fastVal, fastOK, slowVal, slowOK)
		}
	}
}

func TestPrefixOverlaps(t *testing.T) {
	t.Parallel()

	pfxs := shufflePrefixes(allPrefixes())[:100]
	slow := slowTable[int]{pfxs}
	fast := newNode[int]()

	for _, pfx := range pfxs {
		fast.prefixes.insert(pfx.addr, pfx.bits, pfx.val)
	}

	for _, tt := range allPrefixes() {
		slowOK := slow.overlapsPrefix(uint8(tt.addr), tt.bits)
		fastOK := fast.overlapsPrefix(tt.addr, tt.bits)
		if slowOK != fastOK {
			t.Fatalf("overlapsPrefix(%d, %d) = %v, want %v", tt.addr, tt.bits, fastOK, slowOK)
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
		slow := slowTable[int]{pfxs}
		fast := newNode[int]()
		for _, pfx := range pfxs {
			fast.prefixes.insert(pfx.addr, pfx.bits, pfx.val)
		}

		inter := all[numEntries : 2*numEntries]
		slowInter := slowTable[int]{inter}
		fastInter := newNode[int]()
		for _, pfx := range inter {
			fastInter.prefixes.insert(pfx.addr, pfx.bits, pfx.val)
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
				if routes[i].bits < routes[j].bits {
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
				rt.prefixes.insert(route.addr, route.bits, val)
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
			rt.prefixes.insert(route.addr, route.bits, val)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			rt2 := rt
			for _, route := range routes {
				rt2.prefixes.delete(route.addr, route.bits)
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
	forPrefixCount(b, func(b *testing.B, routes []slowEntry[int]) {
		val := 0
		rt := newNode[int]()
		for _, route := range routes {
			rt.prefixes.insert(route.addr, route.bits, val)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, writeSink, _ = rt.prefixes.lpmByAddr(uint(uint8(i)))
		}

		lpm := float64(b.N)
		elapsed := float64(b.Elapsed().Nanoseconds())
		b.ReportMetric(elapsed/lpm, "ns/op")
	})
}

func BenchmarkPrefixSPM(b *testing.B) {
	forPrefixCount(b, func(b *testing.B, routes []slowEntry[int]) {
		val := 0
		rt := newNode[int]()
		for _, route := range routes {
			rt.prefixes.insert(route.addr, route.bits, val)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, writeSink, _ = rt.prefixes.spmByAddr(uint(uint8(i)))
		}

		spm := float64(b.N)
		elapsed := float64(b.Elapsed().Nanoseconds())
		b.ReportMetric(elapsed/spm, "ns/op")
	})
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
	bits int
	val  V
}

func (st *slowTable[V]) String() string {
	pfxs := append([]slowEntry[V](nil), st.prefixes...)
	sort.Slice(pfxs, func(i, j int) bool {
		if pfxs[i].bits != pfxs[j].bits {
			return pfxs[i].bits < pfxs[j].bits
		}
		return pfxs[i].addr < pfxs[j].addr
	})
	var ret bytes.Buffer
	for _, pfx := range pfxs {
		fmt.Fprintf(&ret, "%3d/%d (%08b/%08b) = %v\n", pfx.addr, pfx.bits, pfx.addr, pfxMask(pfx.bits), pfx.val)
	}
	return ret.String()
}

func (st *slowTable[V]) delete(addr uint, prefixLen int) {
	pfx := make([]slowEntry[V], 0, len(st.prefixes))
	for _, e := range st.prefixes {
		if e.addr == addr && e.bits == prefixLen {
			continue
		}
		pfx = append(pfx, e)
	}
	st.prefixes = pfx
}

// get, longest-prefix-match
func (st *slowTable[V]) get(addr uint) (ret V, ok bool) {
	const noMatch = -1
	longest := noMatch
	for _, e := range st.prefixes {
		if addr&pfxMask(e.bits) == e.addr && e.bits >= longest {
			ret = e.val
			longest = e.bits
		}
	}
	return ret, longest != noMatch
}

// spm, shortest-prefix-match
func (st *slowTable[V]) spm(addr uint) (ret V, ok bool) {
	const noMatch = 9
	shortest := noMatch
	for _, e := range st.prefixes {
		if addr&pfxMask(e.bits) == e.addr && e.bits <= shortest {
			ret = e.val
			shortest = e.bits
		}
	}
	return ret, shortest != noMatch
}

func (st *slowTable[T]) overlapsPrefix(addr uint8, prefixLen int) bool {
	for _, e := range st.prefixes {
		minBits := prefixLen
		if e.bits < minBits {
			minBits = e.bits
		}
		mask := ^hostMasks[minBits]
		if addr&mask == uint8(e.addr)&mask {
			return true
		}
	}
	return false
}

func (st *slowTable[T]) overlaps(so *slowTable[T]) bool {
	for _, tp := range st.prefixes {
		for _, op := range so.prefixes {
			minBits := tp.bits
			if op.bits < minBits {
				minBits = op.bits
			}
			if tp.addr&pfxMask(minBits) == op.addr&pfxMask(minBits) {
				return true
			}
		}
	}
	return false
}

func pfxMask(pfxLen int) uint {
	return 0xFF << (stride - pfxLen)
}

func allPrefixes() []slowEntry[int] {
	ret := make([]slowEntry[int], 0, maxNodePrefixes-1)
	for idx := 1; idx < maxNodePrefixes; idx++ {
		addr, bits := baseIndexToPrefix(uint(idx))
		ret = append(ret, slowEntry[int]{addr, bits, idx})
	}
	return ret
}

func shufflePrefixes(pfxs []slowEntry[int]) []slowEntry[int] {
	rand.Shuffle(len(pfxs), func(i, j int) { pfxs[i], pfxs[j] = pfxs[j], pfxs[i] })
	return pfxs
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
