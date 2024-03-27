package bart_test

import (
	"net/netip"
	"testing"

	"github.com/gaissmai/bart"
)

var sink bool

// worst case scenario, go down 16 levels deep
func TestWorstCaseGet(t *testing.T) {
	pfx := netip.MustParsePrefix("fe80::1/128")
	ip6 := netip.MustParseAddr("fe80::2")

	rt := new(bart.Table[any])
	rt.Insert(pfx, nil)

	for i := 0; i < 10_000_000; i++ {
		_, sink = rt.Get(ip6)
	}
}
