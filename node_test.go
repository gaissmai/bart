// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause
//
// tests and benchmarks copied from github.com/tailscale/art
// and massive modified for this implementation by:
//
// Copyright (c) Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestInverseIndex(t *testing.T) {
	t.Parallel()
	for i := 0; i < maxNodeChildren; i++ {
		for bits := 0; bits <= strideLen; bits++ {
			octet := byte(i & (0xFF << (strideLen - bits)))
			idx := prefixToBaseIndex(byte(octet), bits)
			octet2, len2 := baseIndexToPrefix(idx)
			if octet2 != octet || len2 != bits {
				t.Errorf("inverse(index(%d/%d)) != %d/%d", octet, bits, octet2, len2)
			}
		}
	}
}

func TestFringeIndex(t *testing.T) {
	t.Parallel()
	for i := 0; i < maxNodeChildren; i++ {
		got := octetToBaseIndex(byte(i))
		want := prefixToBaseIndex(byte(i), 8)
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

	pfxs := shuffleStridePfxs(allStridePfxs())[:100]
	gold := goldStrideTbl[int](pfxs)
	fast := newNode[int]()

	for _, pfx := range pfxs {
		fast.insertPrefix(prefixToBaseIndex(pfx.octet, pfx.bits), pfx.val)
	}

	for i := 0; i < 256; i++ {
		octet := byte(i)
		goldVal, goldOK := gold.lpm(octet)
		_, fastVal, fastOK := fast.lpm(octetToBaseIndex(octet))
		if !getsEqual(fastVal, fastOK, goldVal, goldOK) {
			t.Fatalf("get(%d) = (%v, %v), want (%v, %v)", octet, fastVal, fastOK, goldVal, goldOK)
		}
	}
}

func TestPrefixDelete(t *testing.T) {
	t.Parallel()
	// Compare route deletion to our reference table.
	pfxs := shuffleStridePfxs(allStridePfxs())[:100]
	gold := goldStrideTbl[int](pfxs)
	fast := newNode[int]()

	for _, pfx := range pfxs {
		fast.insertPrefix(prefixToBaseIndex(pfx.octet, pfx.bits), pfx.val)
	}

	toDelete := pfxs[:50]
	for _, pfx := range toDelete {
		gold.delete(pfx.octet, pfx.bits)
		fast.deletePrefix(pfx.octet, pfx.bits)
	}

	// Sanity check that slow table seems to have done the right thing.
	if cnt := len(gold); cnt != 50 {
		t.Fatalf("goldenStride has %d entries after deletes, want 50", cnt)
	}

	for i := 0; i < 256; i++ {
		octet := byte(i)
		goldVal, goldOK := gold.lpm(octet)
		_, fastVal, fastOK := fast.lpm(octetToBaseIndex(octet))
		if !getsEqual(fastVal, fastOK, goldVal, goldOK) {
			t.Fatalf("get(%d) = (%v, %v), want (%v, %v)", octet, fastVal, fastOK, goldVal, goldOK)
		}
	}
}

func TestPrefixOverlaps(t *testing.T) {
	t.Parallel()

	pfxs := shuffleStridePfxs(allStridePfxs())[:100]
	gold := goldStrideTbl[int](pfxs)
	fast := newNode[int]()

	for _, pfx := range pfxs {
		fast.insertPrefix(prefixToBaseIndex(pfx.octet, pfx.bits), pfx.val)
	}

	for _, tt := range allStridePfxs() {
		goldOK := gold.strideOverlapsPrefix(uint8(tt.octet), tt.bits)
		fastOK := fast.overlapsPrefix(tt.octet, tt.bits)
		if goldOK != fastOK {
			t.Fatalf("overlapsPrefix(%d, %d) = %v, want %v", tt.octet, tt.bits, fastOK, goldOK)
		}
	}
}

func TestNodeOverlaps(t *testing.T) {
	t.Parallel()

	// Empirically, between 5 and 6 routes per table results in ~50%
	// of random pairs overlapping. Cool example of the birthday paradox!
	const numEntries = 6
	all := allStridePfxs()

	seenResult := map[bool]int{}
	for i := 0; i < 100_000; i++ {
		shuffleStridePfxs(all)
		pfxs := all[:numEntries]

		gold := goldStrideTbl[int](pfxs)
		fast := newNode[int]()

		for _, pfx := range pfxs {
			fast.insertPrefix(prefixToBaseIndex(pfx.octet, pfx.bits), pfx.val)
		}

		inter := all[numEntries : 2*numEntries]
		goldInter := goldStrideTbl[int](inter)
		fastInter := newNode[int]()

		for _, pfx := range inter {
			fastInter.insertPrefix(prefixToBaseIndex(pfx.octet, pfx.bits), pfx.val)
		}

		gotGold := gold.strideOverlaps(&goldInter)
		gotFast := fast.overlapsRec(fastInter)
		if gotGold != gotFast {
			t.Fatalf("node.overlaps = %v, want %v", gotFast, gotGold)
		}
		seenResult[gotFast]++
	}
	t.Log(seenResult)
	if len(seenResult) != 2 { // saw both intersections and non-intersections
		t.Fatalf("didn't see both intersections and non-intersections\nIntersects: %d\nNon-intersects: %d", seenResult[true], seenResult[false])
	}
}

var (
	prefixCount = []int{10, 20, 50, 100, 200, 500}
	childCount  = []int{10, 20, 50, 100, 200, 250}
)

func BenchmarkNodePrefixInsert(b *testing.B) {
	routes := shuffleStridePfxs(allStridePfxs())

	for _, nroutes := range prefixCount {
		node := newNode[int]()

		for i, route := range routes {
			if i >= nroutes {
				break
			}
			node.insertPrefix(prefixToBaseIndex(route.octet, route.bits), 0)
		}

		b.Run(fmt.Sprintf("Into %d", nroutes), func(b *testing.B) {
			route := routes[rand.Intn(len(routes))]

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				node.insertPrefix(prefixToBaseIndex(route.octet, route.bits), 0)
			}
		})
	}
}

func BenchmarkNodePrefixUpdate(b *testing.B) {
	routes := shuffleStridePfxs(allStridePfxs())

	for _, nroutes := range prefixCount {
		node := newNode[int]()

		for i, route := range routes {
			if i >= nroutes {
				break
			}
			node.insertPrefix(prefixToBaseIndex(route.octet, route.bits), 0)
		}

		b.Run(fmt.Sprintf("In %d", nroutes), func(b *testing.B) {
			route := routes[rand.Intn(len(routes))]

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				node.updatePrefix(route.octet, route.bits, func(int, bool) int { return 1 })
			}
		})
	}
}

func BenchmarkNodePrefixDelete(b *testing.B) {
	routes := shuffleStridePfxs(allStridePfxs())

	for _, nroutes := range prefixCount {
		node := newNode[int]()

		for i, route := range routes {
			if i >= nroutes {
				break
			}
			node.insertPrefix(prefixToBaseIndex(route.octet, route.bits), 0)
		}

		b.Run(fmt.Sprintf("From %d", nroutes), func(b *testing.B) {
			route := routes[rand.Intn(len(routes))]

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				node.deletePrefix(route.octet, route.bits)
			}
		})
	}
}

var writeSink int

func BenchmarkNodePrefixLPM(b *testing.B) {
	routes := shuffleStridePfxs(allStridePfxs())

	for _, nroutes := range prefixCount {
		node := newNode[int]()

		for i, route := range routes {
			if i >= nroutes {
				break
			}
			node.insertPrefix(prefixToBaseIndex(route.octet, route.bits), 0)
		}

		b.Run(fmt.Sprintf("IN %d", nroutes), func(b *testing.B) {
			route := routes[rand.Intn(len(routes))]

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, writeSink, _ = node.lpm(prefixToBaseIndex(route.octet, route.bits))
			}
		})
	}
}

func BenchmarkNodePrefixRank(b *testing.B) {
	routes := shuffleStridePfxs(allStridePfxs())

	for _, nroutes := range prefixCount {
		node := newNode[int]()

		for i, route := range routes {
			if i >= nroutes {
				break
			}
			node.insertPrefix(prefixToBaseIndex(route.octet, route.bits), 0)
		}

		b.Run(fmt.Sprintf("IN %d", nroutes), func(b *testing.B) {
			route := routes[rand.Intn(len(routes))]
			baseIdx := prefixToBaseIndex(route.octet, route.bits)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				writeSink = node.prefixRank(baseIdx)
			}
		})
	}
}

func BenchmarkNodeChildInsert(b *testing.B) {
	for _, nchilds := range childCount {
		node := newNode[int]()

		for i := 0; i < nchilds; i++ {
			octet := rand.Intn(maxNodeChildren)
			node.insertChild(byte(octet), nil)
		}

		b.Run(fmt.Sprintf("Into %d", nchilds), func(b *testing.B) {
			octet := rand.Intn(maxNodeChildren)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				node.insertChild(byte(octet), nil)
			}
		})
	}
}

func BenchmarkNodeChildDelete(b *testing.B) {
	for _, nchilds := range childCount {
		node := newNode[int]()

		for i := 0; i < nchilds; i++ {
			octet := rand.Intn(maxNodeChildren)
			node.insertChild(byte(octet), nil)
		}

		b.Run(fmt.Sprintf("From %d", nchilds), func(b *testing.B) {
			octet := rand.Intn(maxNodeChildren)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				node.deleteChild(byte(octet))
			}
		})
	}
}

func BenchmarkNodeChildRank(b *testing.B) {
	for _, nchilds := range childCount {
		node := newNode[int]()

		for i := 0; i < nchilds; i++ {
			octet := byte(rand.Intn(maxNodeChildren))
			node.insertChild(octet, nil)
		}

		b.Run(fmt.Sprintf("In %d", nchilds), func(b *testing.B) {
			octet := byte(rand.Intn(maxNodeChildren))
			baseIdx := octetToBaseIndex(octet)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				node.childRank(byte(baseIdx))
			}
		})
	}
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
