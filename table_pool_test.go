package bart

import (
	"net/netip"
	"testing"
)

func TestTablePool_InsertDelete_ReusesTrieNodes(t *testing.T) {
	type Value string

	tbl := &Table[Value]{}
	tbl.WithPool()

	// Insert multiple prefixes that cause new trie nodes to be created.
	// These routes diverge — resulting in logically separate paths
	// in the multibit trie beyond the root node.
	prefixes := []netip.Prefix{
		netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("10.1.0.0/16"),
	}

	for _, pfx := range prefixes {
		tbl.Insert(pfx, Value("route-"+pfx.String()))
	}
	t.Log(tbl.dumpString())

	liveAfterInsert, totalAfterInsert := tbl.pool.Stats()
	t.Logf("after insert: live: %d, total: %d", liveAfterInsert, totalAfterInsert)

	if totalAfterInsert == 0 {
		t.Errorf("expected at least one node allocated after insert #%d", len(prefixes))
	}
	if liveAfterInsert == 0 {
		t.Errorf("expected at least one node live after insert #%d", len(prefixes))
	}

	// Delete the same prefixes
	for _, pfx := range prefixes {
		tbl.Delete(pfx)
		t.Log(tbl.dumpString())
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
