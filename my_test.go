package bart

import (
	"fmt"
	"net/netip"
	"testing"
)

func TestShowPathCompression(t *testing.T) {
	pfxs := []netip.Prefix{
		netip.MustParsePrefix("4.0.0.0/7"),
		netip.MustParsePrefix("89.103.192.0/18"),
		netip.MustParsePrefix("89.103.192.0/24"),
	}
	rt := new(Table[int])
	for i, pfx := range pfxs {
		rt.Insert(pfx, i)
		t.Logf("AFTER insert(%s)\n%s", pfx, rt.dumpString())
	}

	pfx := pfxs[1]
	rt.Delete(pfx)
	t.Logf("AFTER  delete(%s)\n%s", pfx, rt.dumpString())
}

func TestMy(t *testing.T) {
	at := new(ArtTable[int])
	bt := new(Table[int])

	for i, route := range routes4 {
		at.Insert(route.CIDR, i)
		bt.Insert(route.CIDR, i)

		if at.dumpString() != bt.dumpString() {
			t.Logf("after insert(%s, %d)", route.CIDR, i)
			t.Logf("ART:\n%s", at.dumpString())
			t.Logf("BART:\n%s", bt.dumpString())
			t.Log(bt.String())
			t.Fatal()
		}
	}

	aStats := at.root4.nodeStatsRec()
	bStats := bt.root4.nodeStatsRec()

	fmt.Printf("ART:  pfxs: %d, nodes: %d, leaves: %d, fringes: %d, sum: %d\n",
		aStats.pfxs,
		aStats.nodes,
		aStats.leaves,
		aStats.fringes,
		aStats.pfxs+aStats.leaves+aStats.fringes,
	)

	fmt.Printf("BART: pfxs: %d, nodes: %d, leaves: %d, fringes: %d, sum: %d\n",
		bStats.pfxs,
		bStats.nodes,
		bStats.leaves,
		bStats.fringes,
		bStats.pfxs+bStats.leaves+bStats.fringes,
	)
}
