package bart_test

import (
	"net/netip"
	"testing"

	"github.com/gaissmai/bart"
)

var mpp = netip.MustParsePrefix

func TestMy(t *testing.T) {
	rt := new(bart.Table[string])

	k := 1
	for _, s := range examplePrefixes[:k] {
		pfx := mpp(s)
		rt.Insert(pfx, s)
	}

	if size := rt.Size(); size != k {
		t.Errorf("Size: expected %d, got %d", k, size)
	}
}
