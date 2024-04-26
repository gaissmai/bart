package bart

// rangefunc iterators, only test it with 1.22 and newer

/*

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

		// check if pfx/val is as expected
		for pfx, val := range rtbl.All {
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
		countV6 := 0
		for pfx, _ := range rtbl.All {
			// max 1000 IPv6 prefixes
			if !pfx.Addr().Is4() {
				countV6++
			}

			// premature STOP condition
			if countV6 >= 1000 {
				break
			}
		}

		// check if iteration stopped with error
		if countV6 > 1000 {
			t.Fatalf("expected premature stop with error")
		}
	})
}

*/
