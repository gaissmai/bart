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

	"github.com/gaissmai/bart/internal/art"
)

func TestInverseIndex(t *testing.T) {
	t.Parallel()
	for i := range maxItems {
		for bits := 0; bits <= strideLen; bits++ {
			octet := byte(i & (0xFF << (strideLen - bits)))
			idx := art.PfxToIdx(octet, bits)
			octet2, len2 := art.IdxToPfx(idx)
			if octet2 != octet || len2 != bits {
				t.Errorf("inverse(index(%d/%d)) != %d/%d", octet, bits, octet2, len2)
			}
		}
	}
}

func TestFringeIndex(t *testing.T) {
	t.Parallel()
	for i := range maxItems {
		got := art.HostIdx(uint(i))
		want := art.PfxToIdx(byte(i), 8)
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
	gold := new(goldStrideTbl[int]).insertMany(pfxs)
	fast := new(node[int])

	for _, pfx := range pfxs {
		fast.prefixes.InsertAt(art.PfxToIdx(pfx.octet, pfx.bits), pfx.val)
	}

	for i := range 256 {
		octet := byte(i)
		addr := uint(i)
		goldVal, goldOK := gold.lpm(octet)
		_, fastVal, fastOK := fast.lpmGet(art.HostIdx(addr))
		if !getsEqual(fastVal, fastOK, goldVal, goldOK) {
			t.Fatalf("get(%d) = (%v, %v), want (%v, %v)", octet, fastVal, fastOK, goldVal, goldOK)
		}
	}
}

func TestPrefixDelete(t *testing.T) {
	t.Parallel()
	// Compare route deletion to our reference table.
	pfxs := shuffleStridePfxs(allStridePfxs())[:100]
	gold := new(goldStrideTbl[int]).insertMany(pfxs)
	fast := new(node[int])

	for _, pfx := range pfxs {
		fast.prefixes.InsertAt(art.PfxToIdx(pfx.octet, pfx.bits), pfx.val)
	}

	toDelete := pfxs[:50]
	for _, pfx := range toDelete {
		gold.delete(pfx.octet, pfx.bits)
		fast.prefixes.DeleteAt(art.PfxToIdx(pfx.octet, pfx.bits))
	}

	// Sanity check that slow table seems to have done the right thing.
	if cnt := len(*gold); cnt != 50 {
		t.Fatalf("goldenStride has %d entries after deletes, want 50", cnt)
	}

	for i := range 256 {
		octet := byte(i)
		addr := uint(i)
		goldVal, goldOK := gold.lpm(octet)
		_, fastVal, fastOK := fast.lpmGet(art.HostIdx(addr))
		if !getsEqual(fastVal, fastOK, goldVal, goldOK) {
			t.Fatalf("get(%d) = (%v, %v), want (%v, %v)", octet, fastVal, fastOK, goldVal, goldOK)
		}
	}
}

func TestOverlapsPrefix(t *testing.T) {
	t.Parallel()

	pfxs := shuffleStridePfxs(allStridePfxs())[:100]
	gold := new(goldStrideTbl[int]).insertMany(pfxs)
	fast := new(node[int])

	for _, pfx := range pfxs {
		fast.prefixes.InsertAt(art.PfxToIdx(pfx.octet, pfx.bits), pfx.val)
	}

	for _, tt := range allStridePfxs() {
		goldOK := gold.strideOverlapsPrefix(tt.octet, tt.bits)
		fastOK := fast.overlapsIdx(art.PfxToIdx(tt.octet, tt.bits))
		if goldOK != fastOK {
			t.Fatalf("overlapsPrefix(%d, %d) = %v, want %v", tt.octet, tt.bits, fastOK, goldOK)
		}
	}
}

func TestOverlapsRoutes(t *testing.T) {
	t.Parallel()

	const numEntries = 2
	all := allStridePfxs()

	for range 100_000 {
		shuffleStridePfxs(all)
		pfxs := all[:numEntries]

		gold := new(goldStrideTbl[int]).insertMany(pfxs)
		fast := new(node[int])

		for _, pfx := range pfxs {
			fast.prefixes.InsertAt(art.PfxToIdx(pfx.octet, pfx.bits), pfx.val)
		}

		inter := all[numEntries : 2*numEntries]
		goldInter := new(goldStrideTbl[int]).insertMany(inter)
		fastInter := new(node[int])

		for _, pfx := range inter {
			fastInter.prefixes.InsertAt(art.PfxToIdx(pfx.octet, pfx.bits), pfx.val)
		}

		gotGold := gold.strideOverlaps(goldInter)
		gotFast := fast.overlapsRoutes(fastInter)
		if gotGold != gotFast {
			t.Fatalf("node.overlaps = %v, want %v", gotFast, gotGold)
		}
	}
}

var (
	prefixCount = []int{10, 20, 50, 100, 200, maxItems - 1}
	childCount  = []int{10, 20, 50, 100, 200, maxItems - 1}
)

func BenchmarkNodePrefixInsert(b *testing.B) {
	routes := shuffleStridePfxs(allStridePfxs())

	for _, nroutes := range prefixCount {
		this := new(node[int])

		for i, route := range routes {
			if i >= nroutes {
				break
			}
			this.prefixes.InsertAt(art.PfxToIdx(route.octet, route.bits), 0)
		}

		b.Run(fmt.Sprintf("Into %d", nroutes), func(b *testing.B) {
			route := routes[rand.Intn(len(routes))]
			idx := art.PfxToIdx(route.octet, route.bits)

			b.ResetTimer()
			for range b.N {
				this.prefixes.InsertAt(idx, 0)
			}
		})
	}
}

func BenchmarkNodePrefixUpdate(b *testing.B) {
	routes := shuffleStridePfxs(allStridePfxs())

	for _, nroutes := range prefixCount {
		this := new(node[int])

		for i, route := range routes {
			if i >= nroutes {
				break
			}
			this.prefixes.InsertAt(art.PfxToIdx(route.octet, route.bits), 0)
		}

		b.Run(fmt.Sprintf("In %d", nroutes), func(b *testing.B) {
			route := routes[rand.Intn(len(routes))]
			idx := art.PfxToIdx(route.octet, route.bits)

			b.ResetTimer()
			for range b.N {
				this.prefixes.UpdateAt(idx, func(int, bool) int { return 1 })
			}
		})
	}
}

func BenchmarkNodePrefixDelete(b *testing.B) {
	routes := shuffleStridePfxs(allStridePfxs())

	for _, nroutes := range prefixCount {
		this := new(node[int])

		for i, route := range routes {
			if i >= nroutes {
				break
			}
			this.prefixes.InsertAt(art.PfxToIdx(route.octet, route.bits), 0)
		}

		b.Run(fmt.Sprintf("From %d", nroutes), func(b *testing.B) {
			route := routes[rand.Intn(len(routes))]
			idx := art.PfxToIdx(route.octet, route.bits)

			b.ResetTimer()
			for range b.N {
				this.prefixes.DeleteAt(idx)
			}
		})
	}
}

var writeSink int

func BenchmarkNodePrefixLPM(b *testing.B) {
	routes := shuffleStridePfxs(allStridePfxs())

	for _, nroutes := range prefixCount {
		this := new(node[int])

		for i, route := range routes {
			if i >= nroutes {
				break
			}
			this.prefixes.InsertAt(art.PfxToIdx(route.octet, route.bits), 0)
		}

		b.Run(fmt.Sprintf("lpmGet  IN %d", nroutes), func(b *testing.B) {
			route := routes[rand.Intn(len(routes))]
			idx := art.PfxToIdx(route.octet, route.bits)

			b.ResetTimer()
			for range b.N {
				this.lpmGet(idx)
			}
		})

		b.Run(fmt.Sprintf("lpmTest IN %d", nroutes), func(b *testing.B) {
			route := routes[rand.Intn(len(routes))]
			idx := art.PfxToIdx(route.octet, route.bits)

			b.ResetTimer()
			for range b.N {
				this.lpmTest(idx)
			}
		})
	}
}

func BenchmarkNodePrefixesAsSlice(b *testing.B) {
	for _, nPrefixes := range prefixCount {
		this := new(node[any])

		for range nPrefixes {
			idx := byte(rand.Intn(maxItems))
			this.prefixes.InsertAt(uint(idx), nil)
		}

		b.Run(fmt.Sprintf("Set %d", nPrefixes), func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				_ = this.prefixes.AsSlice(make([]uint, 0, maxItems))
			}
		})
	}
}

func BenchmarkNodePrefixesAll(b *testing.B) {
	for _, nPrefixes := range prefixCount {
		this := new(node[any])

		for range nPrefixes {
			idx := byte(rand.Intn(maxItems))
			this.prefixes.InsertAt(uint(idx), nil)
		}

		b.Run(fmt.Sprintf("Set %d", nPrefixes), func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				_ = this.prefixes.All()
			}
		})

	}
}

func BenchmarkNodePrefixIntersectionCardinality(b *testing.B) {
	routes1 := shuffleStridePfxs(allStridePfxs())
	routes2 := shuffleStridePfxs(allStridePfxs())

	for _, nroutes := range prefixCount {
		this := new(node[int])
		other := new(node[int])

		for i, route := range routes1 {
			if i >= nroutes {
				break
			}
			this.prefixes.InsertAt(art.PfxToIdx(route.octet, route.bits), 0)
		}

		for i, route := range routes2 {
			if i >= nroutes {
				break
			}
			other.prefixes.InsertAt(art.PfxToIdx(route.octet, route.bits), 0)
		}

		b.Run(fmt.Sprintf("With %d", nroutes), func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				this.prefixes.IntersectionCardinality(&other.prefixes.BitSet256)
			}
		})
	}
}

func BenchmarkNodeChildInsert(b *testing.B) {
	for _, nchilds := range childCount {
		this := new(node[int])

		for range nchilds {
			octet := rand.Intn(maxItems)
			this.children.InsertAt(uint(octet), nil)
		}

		b.Run(fmt.Sprintf("Into %d", nchilds), func(b *testing.B) {
			octet := rand.Intn(maxItems)

			b.ResetTimer()
			for range b.N {
				this.children.InsertAt(uint(octet), nil)
			}
		})
	}
}

func BenchmarkNodeChildDelete(b *testing.B) {
	for _, nchilds := range childCount {
		this := new(node[int])

		for range nchilds {
			octet := rand.Intn(maxItems)
			this.children.InsertAt(uint(octet), nil)
		}

		b.Run(fmt.Sprintf("From %d", nchilds), func(b *testing.B) {
			octet := rand.Intn(maxItems)

			b.ResetTimer()
			for range b.N {
				this.children.DeleteAt(uint(octet))
			}
		})
	}
}

func BenchmarkNodeChildrenAsSlice(b *testing.B) {
	for _, nchilds := range childCount {
		this := new(node[int])

		for range nchilds {
			octet := byte(rand.Intn(maxItems))
			this.children.InsertAt(uint(octet), nil)
		}

		b.Run(fmt.Sprintf("Set %d", nchilds), func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				this.children.AsSlice(make([]uint, 0, maxItems))
			}
		})
	}
}

func BenchmarkNodeChildrenAll(b *testing.B) {
	for _, nchilds := range childCount {
		this := new(node[int])

		for range nchilds {
			octet := byte(rand.Intn(maxItems))
			this.children.InsertAt(uint(octet), nil)
		}

		b.Run(fmt.Sprintf("Set %d", nchilds), func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				this.children.All()
			}
		})
	}
}

func BenchmarkNodeChildIntersectionCardinality(b *testing.B) {
	for _, nchilds := range childCount {
		this := new(node[int])
		other := new(node[int])

		for range nchilds {
			octet := byte(rand.Intn(maxItems))
			this.children.InsertAt(uint(octet), nil)

			octet = byte(rand.Intn(maxItems))
			other.children.InsertAt(uint(octet), nil)
		}

		b.Run(fmt.Sprintf("With %d", nchilds), func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				this.children.IntersectionCardinality(&other.children.BitSet256)
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
