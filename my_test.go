package bart

import (
	"net/netip"
	"testing"
)

var pfxs = []netip.Prefix{
	mpp("10.0.0.0/23"),
	mpp("10.0.0.0/32"),
	mpp("10.0.0.1/32"),
	mpp("10.0.42.0/24"),
	mpp("10.0.42.0/25"),
	mpp("10.0.42.0/28"),
	mpp("10.0.42.128/25"),
	mpp("10.10.0.0/16"),
}

func TestMy(t *testing.T) {
	facl := new(FastACL)
	lite := new(Lite)
	for _, pfx := range pfxs {
		facl.Insert(pfx)
		lite.Insert(pfx)
	}

	facl.Fprint(t.Output())
	lite.Fprint(t.Output())
}
