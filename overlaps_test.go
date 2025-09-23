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

	const numEntries = 10

	for range n {
		pfxs := randomPrefixes(prng, numEntries)
		inter := randomPrefixes(prng, numEntries)

		gold := new(goldTable[int])
		gold.insertMany(pfxs)

		goldInter := new(goldTable[int])
		goldInter.insertMany(inter)
		gotGold := gold.overlaps(goldInter)

		// Table

		bart := new(Table[int])
		for _, pfx := range pfxs {
			bart.Insert(pfx.pfx, pfx.val)
		}

		bartInter := new(Table[int])
		for _, pfx := range inter {
			bartInter.Insert(pfx.pfx, pfx.val)
		}

		gotBart := bart.Overlaps(bartInter)

		if gotGold != gotBart {
			t.Fatalf("Overlaps(...) = %v, want %v\nbart1:\n%s\nbart2:\n%s",
				gotBart, gotGold, bart.String(), bartInter.String())
		}

		// Fast

		fast := new(Fast[int])
		for _, pfx := range pfxs {
			fast.Insert(pfx.pfx, pfx.val)
		}

		fastInter := new(Fast[int])
		for _, pfx := range inter {
			fastInter.Insert(pfx.pfx, pfx.val)
		}

		gotFast := fast.Overlaps(fastInter)

		if gotGold != gotFast {
			t.Fatalf("Overlaps(...) = %v, want %v\nfast1:\n%s\nfast2:\n%s",
				gotFast, gotGold, fast.String(), fastInter.String())
		}

		// Lite

		lite := new(Lite)
		for _, pfx := range pfxs {
			lite.Insert(pfx.pfx)
		}

		liteInter := new(Lite)
		for _, pfx := range inter {
			liteInter.Insert(pfx.pfx)
		}

		gotLite := lite.Overlaps(liteInter)

		if gotGold != gotLite {
			t.Fatalf("Overlaps(...) = %v, want %v\nlite1:\n%s\nlite2:\n%s",
				gotFast, gotGold, lite.String(), liteInter.String())
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
