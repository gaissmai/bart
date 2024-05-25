// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

//go:build go1.23

// rangefunc iterators, only test it with 1.23
// or 1.22 and:
//  GOEXPERIMENT=rangefunc go test ...

package bart

import (
	"net/netip"
	"testing"
)

func TestRange(t *testing.T) {
	pfxs := randomPrefixes(10_000)
	seen := make(map[netip.Prefix]int, 10_000)

	t.Run("All", func(t *testing.T) {
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
			seen[item.pfx] = item.val
		}

		// rangefunc iterator
		for pfx, val := range rtbl.All {
			// check if pfx/val is as expected
			if seen[pfx] != val {
				t.Errorf("%v got value: %v, expected: %v", pfx, val, seen[pfx])
			}
			delete(seen, pfx)
		}

		// check if all entries visited
		if len(seen) != 0 {
			t.Fatalf("traverse error, not all entries visited")
		}
	})

	t.Run("All with premature exit", func(t *testing.T) {
		rtbl := new(Table[int])
		for _, item := range pfxs {
			rtbl.Insert(item.pfx, item.val)
		}

		// check if callback stops prematurely
		count := 0
		for _, _ = range rtbl.All {
			count++
			if count >= 1000 {
				break
			}
		}

		// check if iteration stopped with error
		if count > 1000 {
			t.Fatalf("expected premature stop with error")
		}
	})
}
