// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause
//
// tests and benchmarks copied from github.com/tailscale/art
// and massive modified for this implementation by:
//
// Copyright (c) Karl Gaissmaier
// SPDX-License-Identifier: MIT

package nodes

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/gaissmai/bart/internal/art"
)

// workLoadN to adjust loops for tests with -short
func workLoadN() int {
	if testing.Short() {
		return 1_000
	}
	return 10_000
}

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

	prng := rand.New(rand.NewPCG(42, 42))

	pfxs := shuffleNodePfxs(prng, allNodePfxs())[:100]
	gold := new(goldNode[int]).insertMany(pfxs)
	fast := new(BartNode[int])

	for _, pfx := range pfxs {
		fast.InsertPrefix(art.PfxToIdx(pfx.octet, pfx.bits), pfx.val)
	}

	for i := range maxItems {
		//nolint:gosec  // G115: integer overflow conversion int -> uint
		octet := byte(i)
		//nolint:gosec  // G115: integer overflow conversion int -> uint
		addr := uint8(i)
		goldVal, goldOK := gold.lpm(octet)
		_, fastVal, fastOK := fast.LookupIdx(art.OctetToIdx(addr))
		if !getsEqual(fastVal, fastOK, goldVal, goldOK) {
			t.Fatalf("get(%d) = (%v, %v), want (%v, %v)", octet, fastVal, fastOK, goldVal, goldOK)
		}
	}
}

func TestPrefixDelete(t *testing.T) {
	t.Parallel()
	// Compare route deletion to our reference table.
	prng := rand.New(rand.NewPCG(42, 42))

	pfxs := shuffleNodePfxs(prng, allNodePfxs())[:100]
	gold := new(goldNode[int]).insertMany(pfxs)
	fast := new(BartNode[int])

	for _, pfx := range pfxs {
		fast.InsertPrefix(art.PfxToIdx(pfx.octet, pfx.bits), pfx.val)
	}

	toDelete := pfxs[:50]
	for _, pfx := range toDelete {
		gold.deleteItem(pfx.octet, pfx.bits)
		fast.DeletePrefix(art.PfxToIdx(pfx.octet, pfx.bits))
	}

	// Sanity check that slow table seems to have done the right thing.
	if cnt := len(*gold); cnt != 50 {
		t.Fatalf("goldNode has %d entries after deletes, want 50", cnt)
	}

	for i := range maxItems {
		//nolint:gosec  // G115: integer overflow conversion int -> uint
		octet := byte(i)
		//nolint:gosec  // G115: integer overflow conversion int -> uint
		addr := uint8(i)
		goldVal, goldOK := gold.lpm(octet)
		_, fastVal, fastOK := fast.LookupIdx(art.OctetToIdx(addr))
		if !getsEqual(fastVal, fastOK, goldVal, goldOK) {
			t.Fatalf("get(%d) = (%v, %v), want (%v, %v)", octet, fastVal, fastOK, goldVal, goldOK)
		}
	}
}

func TestOverlapsPrefix(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))

	pfxs := shuffleNodePfxs(prng, allNodePfxs())[:100]
	gold := new(goldNode[int]).insertMany(pfxs)
	bart := new(BartNode[int])

	for _, pfx := range pfxs {
		bart.InsertPrefix(art.PfxToIdx(pfx.octet, pfx.bits), pfx.val)
	}

	for _, tt := range allNodePfxs() {
		goldOK := gold.overlapsPrefix(tt.octet, tt.bits)
		fastOK := bart.OverlapsIdx(art.PfxToIdx(tt.octet, tt.bits))
		if goldOK != fastOK {
			t.Fatalf("overlapsPrefix(%d, %d) = %v, want %v", tt.octet, tt.bits, fastOK, goldOK)
		}
	}
}

func TestOverlapsRoutes(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))

	const numEntries = 2
	all := allNodePfxs()

	for range n {
		shuffleNodePfxs(prng, all)
		pfxs := all[:numEntries]

		gold := new(goldNode[int]).insertMany(pfxs)
		fast := new(BartNode[int])

		for _, pfx := range pfxs {
			fast.InsertPrefix(art.PfxToIdx(pfx.octet, pfx.bits), pfx.val)
		}

		inter := all[numEntries : 2*numEntries]
		goldInter := new(goldNode[int]).insertMany(inter)
		fastInter := new(BartNode[int])

		for _, pfx := range inter {
			fastInter.InsertPrefix(art.PfxToIdx(pfx.octet, pfx.bits), pfx.val)
		}

		gotGold := gold.overlaps(goldInter)
		gotFast := fast.OverlapsRoutes(fastInter)
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
	routes := shuffleNodePfxs(prng, allNodePfxs())

	for _, nroutes := range prefixCount {
		this := new(BartNode[int])

		for i, route := range routes {
			if i >= nroutes {
				break
			}
			this.InsertPrefix(art.PfxToIdx(route.octet, route.bits), 0)
		}

		b.Run(fmt.Sprintf("Into %d", nroutes), func(b *testing.B) {
			var i int
			for b.Loop() {
				route := routes[i%len(routes)]
				idx := art.PfxToIdx(route.octet, route.bits)
				this.InsertPrefix(idx, 0)
				i++
			}
		})
	}
}

func BenchmarkNodePrefixDelete(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	routes := shuffleNodePfxs(prng, allNodePfxs())

	for _, nroutes := range prefixCount {
		this := new(BartNode[int])

		for i, route := range routes {
			if i >= nroutes {
				break
			}
			this.InsertPrefix(art.PfxToIdx(route.octet, route.bits), 0)
		}

		b.Run(fmt.Sprintf("From %d", nroutes), func(b *testing.B) {
			var i int
			for b.Loop() {
				route := routes[i%len(routes)]
				idx := art.PfxToIdx(route.octet, route.bits)
				this.DeletePrefix(idx)
				i++
			}
		})
	}
}

func BenchmarkNodesPrefixLPM(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	routes := shuffleNodePfxs(prng, allNodePfxs())

	for _, nroutes := range prefixCount {
		this := new(BartNode[int])
		that := new(FastNode[int])

		for i, route := range routes {
			if i >= nroutes {
				break
			}
			this.InsertPrefix(art.PfxToIdx(route.octet, route.bits), 0)
			that.InsertPrefix(art.PfxToIdx(route.octet, route.bits), 0)
		}

		b.Run(fmt.Sprintf("node:    lookup   IN %d", nroutes), func(b *testing.B) {
			route := routes[prng.IntN(len(routes))]
			idx := art.PfxToIdx(route.octet, route.bits)

			for b.Loop() {
				this.Lookup(idx)
			}
		})

		b.Run(fmt.Sprintf("fastNode: lookup   IN %d", nroutes), func(b *testing.B) {
			route := routes[prng.IntN(len(routes))]
			idx := art.PfxToIdx(route.octet, route.bits)

			for b.Loop() {
				that.Lookup(idx)
			}
		})

		b.Run(fmt.Sprintf("node:    contains IN %d", nroutes), func(b *testing.B) {
			route := routes[prng.IntN(len(routes))]
			idx := art.PfxToIdx(route.octet, route.bits)

			for b.Loop() {
				this.Contains(idx)
			}
		})

		b.Run(fmt.Sprintf("fastNode: contains IN %d", nroutes), func(b *testing.B) {
			route := routes[prng.IntN(len(routes))]
			idx := art.PfxToIdx(route.octet, route.bits)

			for b.Loop() {
				that.Contains(idx)
			}
		})
	}
}

func BenchmarkNodePrefixesAsSlice(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, nPrefixes := range prefixCount {
		this := new(BartNode[any])

		for range nPrefixes {
			idx := byte(prng.IntN(maxItems))
			this.InsertPrefix(idx, nil)
		}

		b.Run(fmt.Sprintf("Set %d", nPrefixes), func(b *testing.B) {
			var buf [maxItems]uint8
			for b.Loop() {
				this.Prefixes.AsSlice(&buf)
			}
		})
	}
}

func BenchmarkNodePrefixesAll(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, nPrefixes := range prefixCount {
		this := new(BartNode[any])

		for range nPrefixes {
			idx := byte(prng.IntN(maxItems))
			this.InsertPrefix(idx, nil)
		}

		b.Run(fmt.Sprintf("Set %d", nPrefixes), func(b *testing.B) {
			for b.Loop() {
				this.Prefixes.Bits()
			}
		})

	}
}

func BenchmarkNodeChildInsert(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, nchilds := range childCount {
		this := new(BartNode[int])

		for range nchilds {
			//nolint:gosec  // G115: integer overflow conversion int -> uint
			octet := uint8(prng.IntN(maxItems))
			this.InsertChild(octet, nil)
		}

		b.Run(fmt.Sprintf("Into %d", nchilds), func(b *testing.B) {
			//nolint:gosec  // G115: integer overflow conversion int -> uint
			octet := uint8(prng.IntN(maxItems))

			for b.Loop() {
				this.InsertChild(octet, nil)
			}
		})
	}
}

func BenchmarkNodeChildDelete(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, nchilds := range childCount {
		this := new(BartNode[int])

		for range nchilds {
			//nolint:gosec  // G115: integer overflow conversion int -> uint
			octet := uint8(prng.IntN(maxItems))
			this.InsertChild(octet, nil)
		}

		b.Run(fmt.Sprintf("From %d", nchilds), func(b *testing.B) {
			//nolint:gosec  // G115: integer overflow conversion int -> uint
			octet := uint8(prng.IntN(maxItems))

			for b.Loop() {
				this.DeleteChild(octet)
			}
		})
	}
}

func BenchmarkNodeChildrenAsSlice(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, nchilds := range childCount {
		this := new(BartNode[int])

		for range nchilds {
			octet := byte(prng.IntN(maxItems))
			this.InsertChild(octet, nil)
		}

		b.Run(fmt.Sprintf("Set %d", nchilds), func(b *testing.B) {
			var buf [maxItems]uint8
			for b.Loop() {
				this.Children.AsSlice(&buf)
			}
		})
	}
}

func BenchmarkNodeChildrenAll(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, nchilds := range childCount {
		this := new(BartNode[int])

		for range nchilds {
			octet := byte(prng.IntN(maxItems))
			this.InsertChild(octet, nil)
		}

		b.Run(fmt.Sprintf("Set %d", nchilds), func(b *testing.B) {
			for b.Loop() {
				this.Children.Bits()
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
