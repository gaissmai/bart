package bart

import (
	"math/rand/v2"
	"testing"
)

func TestTablePool_InsertDelete_ReusesTrieNodes(t *testing.T) {
	t.Parallel()

	// tbl := new(Table[Value]).withDebugPool()
	tbl := new(Table[string]).WithPool()

	// Insert multiple prefixes that cause new trie nodes to be created.
	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, 500_000)

	for _, pfx := range pfxs {
		tbl.Insert(pfx, "route-"+pfx.String())
	}

	liveAfterInsert, totalAfterInsert := tbl.pool.Stats()
	t.Logf("after insert: live: %d, total: %d", liveAfterInsert, totalAfterInsert)

	if totalAfterInsert == 0 {
		t.Errorf("expected at least one node allocated after insert #%d", len(pfxs))
	}
	if liveAfterInsert == 0 {
		t.Errorf("expected at least one node live after insert #%d", len(pfxs))
	}

	// Delete the same prefixes, shuffeled
	prng.Shuffle(len(pfxs), func(i, j int) {
		pfxs[i], pfxs[j] = pfxs[j], pfxs[i]
	})

	for _, pfx := range pfxs {
		tbl.Delete(pfx)
	}

	// Check pool stats after deletes
	liveAfterDelete, totalAfterDelete := tbl.pool.Stats()
	t.Logf("after delete: live: %d, total: %d", liveAfterDelete, totalAfterDelete)

	if liveAfterDelete != 0 {
		t.Errorf("expected all nodes returned to pool after deletes, got %d live", liveAfterDelete)
	}
	if totalAfterDelete != totalAfterInsert {
		t.Errorf("expected total allocated to remain the same, got %d (was %d)", totalAfterDelete, totalAfterInsert)
	}
}
