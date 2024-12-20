package bart

import (
	"net/netip"
	"testing"
)

func TestMy(t *testing.T) {
	pfxs := []netip.Prefix{
		mpp("255.0.0.0/8"),
		mpp("250.255.0.0/16"),
		mpp("250.250.255.0/24"),
		mpp("250.250.250.255/32"),
		mpp("250.250.250.250/32"),
	}

	tbl := new(Table[string])
	for _, p := range pfxs {
		tbl.Insert(p, p.String())
	}

	probe := mpa("250.250.250.251")
	_ = tbl.Contains(probe)
	/*
		t.Errorf("Contains: %v", tbl.Contains(probe))
		t.Error(tbl.dumpString())
		t.Error(tbl)
	*/
}
