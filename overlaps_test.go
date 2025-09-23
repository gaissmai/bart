// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"math/rand/v2"
	"testing"
)

func TestOverlapsCompare(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))

	const numEntries = 6

	for range n {
		pfxs := randomPrefixes(prng, numEntries)

		gold := new(goldTable[int])
		gold.insertMany(pfxs)

		fast := new(Table[int])
		for _, pfx := range pfxs {
			fast.Insert(pfx.pfx, pfx.val)
		}

		inter := randomPrefixes(prng, numEntries)

		goldInter := new(goldTable[int])
		goldInter.insertMany(inter)

		fastInter := new(Table[int])
		for _, pfx := range inter {
			fastInter.Insert(pfx.pfx, pfx.val)
		}

		gotGold := gold.overlaps(goldInter)
		gotFast := fast.Overlaps(fastInter)

		if gotGold != gotFast {
			t.Fatalf("Overlaps(...) = %v, want %v\nTable1:\n%s\nTable:\n%s",
				gotFast, gotGold, fast.String(), fastInter.String())
		}
	}
}

func TestOverlapsPrefixCompare(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, n)

	gold := new(goldTable[int])
	gold.insertMany(pfxs)

	fast := new(Table[int])
	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	tests := randomPrefixes(prng, n)
	for _, tt := range tests {
		gotGold := gold.overlapsPrefix(tt.pfx)
		gotFast := fast.OverlapsPrefix(tt.pfx)
		if gotGold != gotFast {
			t.Fatalf("overlapsPrefix(%q) = %v, want %v", tt.pfx, gotFast, gotGold)
		}
	}
}
