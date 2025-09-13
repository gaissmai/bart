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
	"math/rand/v2"
	"testing"

	"github.com/gaissmai/bart/internal/art"
)

func TestInverseIndex(t *testing.T) {
	t.Parallel()
	for i := range maxItems {
		for bits := range uint8(8) {
			octet := byte(i & (0xFF << (strideLen - bits)))
			idx := art.PfxToIdx(octet, bits)
			octet2, len2 := art.IdxToPfx(idx)
			if octet2 != octet || len2 != bits {
				t.Errorf("inverse(index(%d/%d)) != %d/%d", octet, bits, octet2, len2)
			}
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
		fast.insertPrefix(art.PfxToIdx(pfx.octet, pfx.bits), pfx.val)
	}

	for i := range 256 {
		//nolint:gosec  // G115: integer overflow conversion int -> uint
		octet := byte(i)
		//nolint:gosec  // G115: integer overflow conversion int -> uint
		addr := uint8(i)
		goldVal, goldOK := gold.lpm(octet)
		_, fastVal, fastOK := fast.lpmGet(art.OctetToIdx(addr))
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
		fast.insertPrefix(art.PfxToIdx(pfx.octet, pfx.bits), pfx.val)
	}

	toDelete := pfxs[:50]
	for _, pfx := range toDelete {
		gold.delete(pfx.octet, pfx.bits)
		fast.deletePrefix(art.PfxToIdx(pfx.octet, pfx.bits))
	}

	// Sanity check that slow table seems to have done the right thing.
	if cnt := len(*gold); cnt != 50 {
		t.Fatalf("goldenStride has %d entries after deletes, want 50", cnt)
	}

	for i := range 256 {
		//nolint:gosec  // G115: integer overflow conversion int -> uint
		octet := byte(i)
		//nolint:gosec  // G115: integer overflow conversion int -> uint
		addr := uint8(i)
		goldVal, goldOK := gold.lpm(octet)
		_, fastVal, fastOK := fast.lpmGet(art.OctetToIdx(addr))
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
		fast.insertPrefix(art.PfxToIdx(pfx.octet, pfx.bits), pfx.val)
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
			fast.insertPrefix(art.PfxToIdx(pfx.octet, pfx.bits), pfx.val)
		}

		inter := all[numEntries : 2*numEntries]
		goldInter := new(goldStrideTbl[int]).insertMany(inter)
		fastInter := new(node[int])

		for _, pfx := range inter {
			fastInter.insertPrefix(art.PfxToIdx(pfx.octet, pfx.bits), pfx.val)
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
	prng := rand.New(rand.NewPCG(42, 42))
	routes := shuffleStridePfxs(allStridePfxs())

	for _, nroutes := range prefixCount {
		this := new(node[int])

		for i, route := range routes {
			if i >= nroutes {
				break
			}
			this.insertPrefix(art.PfxToIdx(route.octet, route.bits), 0)
		}

		b.Run(fmt.Sprintf("Into %d", nroutes), func(b *testing.B) {
			route := routes[prng.IntN(len(routes))]
			idx := art.PfxToIdx(route.octet, route.bits)

			for b.Loop() {
				this.insertPrefix(idx, 0)
			}
		})
	}
}

func BenchmarkNodePrefixDelete(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	routes := shuffleStridePfxs(allStridePfxs())

	for _, nroutes := range prefixCount {
		this := new(node[int])

		for i, route := range routes {
			if i >= nroutes {
				break
			}
			this.insertPrefix(art.PfxToIdx(route.octet, route.bits), 0)
		}

		b.Run(fmt.Sprintf("From %d", nroutes), func(b *testing.B) {
			route := routes[prng.IntN(len(routes))]
			idx := art.PfxToIdx(route.octet, route.bits)

			for b.Loop() {
				this.deletePrefix(idx)
			}
		})
	}
}

func BenchmarkNodePrefixLPM(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	routes := shuffleStridePfxs(allStridePfxs())

	for _, nroutes := range prefixCount {
		this := new(node[int])

		for i, route := range routes {
			if i >= nroutes {
				break
			}
			this.insertPrefix(art.PfxToIdx(route.octet, route.bits), 0)
		}

		b.Run(fmt.Sprintf("lpmGet  IN %d", nroutes), func(b *testing.B) {
			route := routes[prng.IntN(len(routes))]
			idx := art.PfxToIdx(route.octet, route.bits)

			for b.Loop() {
				this.lpmGet(uint(idx))
			}
		})

		b.Run(fmt.Sprintf("lpmTest IN %d", nroutes), func(b *testing.B) {
			route := routes[prng.IntN(len(routes))]
			idx := art.PfxToIdx(route.octet, route.bits)

			for b.Loop() {
				this.lpmTest(uint(idx))
			}
		})
	}
}

func BenchmarkNodePrefixesAsSlice(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, nPrefixes := range prefixCount {
		this := new(node[any])

		for range nPrefixes {
			idx := byte(prng.IntN(maxItems))
			this.insertPrefix(idx, nil)
		}

		b.Run(fmt.Sprintf("Set %d", nPrefixes), func(b *testing.B) {
			var buf [256]uint8
			for b.Loop() {
				this.prefixes.AsSlice(&buf)
			}
		})
	}
}

func BenchmarkNodePrefixesAll(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, nPrefixes := range prefixCount {
		this := new(node[any])

		for range nPrefixes {
			idx := byte(prng.IntN(maxItems))
			this.insertPrefix(idx, nil)
		}

		b.Run(fmt.Sprintf("Set %d", nPrefixes), func(b *testing.B) {
			for b.Loop() {
				this.prefixes.Bits()
			}
		})

	}
}

func BenchmarkNodeChildInsert(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, nchilds := range childCount {
		this := new(node[int])

		for range nchilds {
			//nolint:gosec  // G115: integer overflow conversion int -> uint
			octet := uint8(prng.IntN(maxItems))
			this.insertChild(octet, nil)
		}

		b.Run(fmt.Sprintf("Into %d", nchilds), func(b *testing.B) {
			//nolint:gosec  // G115: integer overflow conversion int -> uint
			octet := uint8(prng.IntN(maxItems))

			for b.Loop() {
				this.insertChild(octet, nil)
			}
		})
	}
}

func BenchmarkNodeChildDelete(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, nchilds := range childCount {
		this := new(node[int])

		for range nchilds {
			//nolint:gosec  // G115: integer overflow conversion int -> uint
			octet := uint8(prng.IntN(maxItems))
			this.insertChild(octet, nil)
		}

		b.Run(fmt.Sprintf("From %d", nchilds), func(b *testing.B) {
			//nolint:gosec  // G115: integer overflow conversion int -> uint
			octet := uint8(prng.IntN(maxItems))

			for b.Loop() {
				this.deleteChild(octet)
			}
		})
	}
}

func BenchmarkNodeChildrenAsSlice(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, nchilds := range childCount {
		this := new(node[int])

		for range nchilds {
			octet := byte(prng.IntN(maxItems))
			this.insertChild(octet, nil)
		}

		b.Run(fmt.Sprintf("Set %d", nchilds), func(b *testing.B) {
			var buf [256]uint8
			for b.Loop() {
				this.children.AsSlice(&buf)
			}
		})
	}
}

func BenchmarkNodeChildrenAll(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, nchilds := range childCount {
		this := new(node[int])

		for range nchilds {
			octet := byte(prng.IntN(maxItems))
			this.insertChild(octet, nil)
		}

		b.Run(fmt.Sprintf("Set %d", nchilds), func(b *testing.B) {
			for b.Loop() {
				this.children.Bits()
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
