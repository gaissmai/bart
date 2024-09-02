package bart

import (
	"net/netip"
	"reflect"
	"testing"
)

func TestEachSubnetCompare(t *testing.T) {
	t.Parallel()

	pfxs := randomPrefixes(10_000)

	fast := &Table[int]{}
	gold := goldTable[int](pfxs)

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	for range 10_000 {
		pfx := randomPrefix()
		goldPfxs := gold.subnets(pfx)

		var fastPfxs []netip.Prefix
		values := map[netip.Prefix]int{}

		fast.EachSubnet(pfx, func(p netip.Prefix, val int) bool {
			fastPfxs = append(fastPfxs, p)
			values[p] = val
			return true
		})

		if !reflect.DeepEqual(goldPfxs, fastPfxs) {
			t.Fatalf("\nEachSubnets(%q):\ngot:  %v\nwant: %v", pfx, fastPfxs, goldPfxs)
		}

		// also check the values handled by yield function
		for pfx, val := range values {
			got, ok := fast.Get(pfx)

			if !ok || got != val {
				t.Fatalf("EachSubnets: Get(%q), got: %d,%v, want: %d,%v", pfx, got, ok, val, true)
			}
		}
	}
}

func TestEachLookupPrefix(t *testing.T) {
	t.Parallel()

	pfxs := randomPrefixes(10_000)

	fast := Table[int]{}
	gold := goldTable[int](pfxs)

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	var fastPfxs []netip.Prefix

	for range 10_000 {
		pfx := randomPrefix()

		goldPfxs := gold.lookupPrefixReverse(pfx)

		fastPfxs = nil

		fast.EachLookupPrefix(pfx, func(p netip.Prefix, _ int) bool {
			fastPfxs = append(fastPfxs, p)
			return true
		})

		if !reflect.DeepEqual(goldPfxs, fastPfxs) {
			t.Fatalf("\nEachSupernet(%q):\ngot:  %v\nwant: %v", pfx, fastPfxs, goldPfxs)
		}
	}
}
