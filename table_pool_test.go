package bart

import (
	"math/rand/v2"
	"slices"
	"testing"
)

func TestTablePool_InsertDelete_ReusesTrieNodes(t *testing.T) {
	t.Parallel()

	// tbl := new(Table[Value]).withDebugPool()
	tbl := new(Table[string]).WithPool()

	// Insert multiple prefixes that cause new trie nodes to be created.
	prng := rand.New(rand.NewPCG(42, 42))
	for _, r := range routes {
		tbl.Insert(r.CIDR, "route-"+r.CIDR.String())
	}

	nodeLive1, nodeTotal1 := tbl.multiPool.nodeStats()
	t.Logf("node   after insert: live: %7d, total: %7d", nodeLive1, nodeTotal1)
	if nodeTotal1 == 0 {
		t.Errorf("node   expected at least one node allocated after insert #%7d", len(routes))
	}
	if nodeLive1 == 0 {
		t.Errorf("node   expected at least one node live after insert #%7d", len(routes))
	}

	leafLive1, leafTotal1 := tbl.multiPool.leafStats()
	t.Logf("leaf   after insert: live: %7d, total: %7d", leafLive1, leafTotal1)
	if leafTotal1 == 0 {
		t.Errorf("leaf   expected at least one leaf allocated after insert #%7d", len(routes))
	}
	if leafLive1 == 0 {
		t.Errorf("leaf   expected at least one leaf live after insert #%7d", len(routes))
	}

	fringeLive1, fringeTotal1 := tbl.multiPool.fringeStats()
	t.Logf("fringe after insert: live: %7d, total: %7d", fringeLive1, fringeTotal1)
	if fringeTotal1 == 0 {
		t.Errorf("fringe expected at least one fringe allocated after insert #%7d", len(routes))
	}
	if fringeLive1 == 0 {
		t.Errorf("fringe expected at least one fringe live after insert #%7d", len(routes))
	}

	cloneRoutes := slices.Clone(routes)

	// Delete the same prefixes, shuffeled
	prng.Shuffle(len(cloneRoutes), func(i, j int) {
		cloneRoutes[i], cloneRoutes[j] = cloneRoutes[j], cloneRoutes[i]
	})

	for _, r := range cloneRoutes {
		tbl.Delete(r.CIDR)
	}

	// Check pool stats after deletes
	nodeLive2, nodeTotal2 := tbl.multiPool.nodeStats()
	t.Logf("node   after delete: live: %7d, total: %7d", nodeLive2, nodeTotal2)
	if nodeLive2 != 0 {
		t.Errorf("node   expected all nodes returned to pool after deletes, got %7d live", nodeLive2)
	}
	if nodeTotal2 != nodeTotal1 {
		t.Errorf("node   expected total allocated to remain the same, got %7d (was %7d)", nodeTotal2, nodeTotal1)
	}

	leafLive2, leafTotal2 := tbl.multiPool.leafStats()
	t.Logf("leaf   after delete: live: %7d, total: %7d", leafLive2, leafTotal2)
	if leafLive2 != 0 {
		t.Errorf("leaf   expected all leafs returned to pool after deletes, got %7d live", leafLive2)
	}

	fringeLive2, fringeTotal2 := tbl.multiPool.fringeStats()
	t.Logf("fringe after delete: live: %7d, total: %7d", fringeLive2, fringeTotal2)
	if fringeLive2 != 0 {
		t.Errorf("fringe expected all fringes returned to pool after deletes, got %7d live", fringeLive2)
	}
}
