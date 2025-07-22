package bart

import (
	"testing"
)

func TestMultiPool(t *testing.T) {
	// Use simple value type for testing (e.g., string)
	type testVal = string

	mp := newMultiPool[testVal]()

	// Initial stats -> all should be zero
	if live, total := mp.nodeStats(); live != 0 || total != 0 {
		t.Fatalf("expected node stats to start at 0/0, got %d/%d", live, total)
	}
	if live, total := mp.leafStats(); live != 0 || total != 0 {
		t.Fatalf("expected leaf stats to start at 0/0, got %d/%d", live, total)
	}
	if live, total := mp.fringeStats(); live != 0 || total != 0 {
		t.Fatalf("expected fringe stats to start at 0/0, got %d/%d", live, total)
	}

	// allocate node
	n := mp.getNode()
	if n == nil {
		t.Fatal("expected non-nil node")
	}
	if live, _ := mp.nodeStats(); live != 1 {
		t.Fatalf("expected 1 live node, got %d", live)
	}

	// allocate leaf with dummy prefix
	pfx := mpp("10.0.0.0/8")
	l := mp.getLeaf(pfx, "leaf1")
	if l == nil || l.value != "leaf1" || l.prefix != pfx {
		t.Fatalf("unexpected leaf node state: %+v", l)
	}
	if live, _ := mp.leafStats(); live != 1 {
		t.Fatalf("expected 1 live leaf, got %d", live)
	}

	// allocate fringe
	f := mp.getFringe("fringe1")
	if f == nil || f.value != "fringe1" {
		t.Fatalf("unexpected fringe node state: %+v", f)
	}
	if live, _ := mp.fringeStats(); live != 1 {
		t.Fatalf("expected 1 live fringe, got %d", live)
	}

	// return all to pool
	mp.putNode(n)
	mp.putLeaf(l)
	mp.putFringe(f)

	if live, _ := mp.nodeStats(); live != 0 {
		t.Errorf("expected 0 live nodes after put, got %d", live)
	}
	if live, _ := mp.leafStats(); live != 0 {
		t.Errorf("expected 0 live leaf nodes after put, got %d", live)
	}
	if live, _ := mp.fringeStats(); live != 0 {
		t.Errorf("expected 0 live fringe nodes after put, got %d", live)
	}

	// re-allocate from pool (should reuse)
	n2 := mp.getNode()
	mp.putNode(n2)

	leaf2 := mp.getLeaf(pfx, "leaf2")
	if leaf2.value != "leaf2" || leaf2.prefix != pfx {
		t.Errorf("expected reused leaf to be properly reinitialized")
	}
	mp.putLeaf(leaf2)

	fringe2 := mp.getFringe("fringe2")
	if fringe2.value != "fringe2" {
		t.Errorf("expected reused fringe to be properly reinitialized")
	}
	mp.putFringe(fringe2)

	// check total allocated count ≥ 1 (may be zero if sync.Pool reused already-constructed object)
	if _, total := mp.nodeStats(); total < 1 {
		t.Errorf("expected at least 1 node to be allocated, got %d", total)
	}
	if _, total := mp.leafStats(); total < 1 {
		t.Errorf("expected at least 1 leaf to be allocated, got %d", total)
	}
	if _, total := mp.fringeStats(); total < 1 {
		t.Errorf("expected at least 1 fringe to be allocated, got %d", total)
	}
}

func TestMultiPool_NilFallback(t *testing.T) {
	var mp *multiPool[string] // nil pool

	n := mp.getNode()
	if n == nil {
		t.Fatal("expected fallback node instance")
	}

	pfx := mpp("192.168.0.0/16")
	l := mp.getLeaf(pfx, "hello")
	if l == nil || l.value != "hello" || l.prefix != pfx {
		t.Fatalf("unexpected leaf from nil pool: %+v", l)
	}

	f := mp.getFringe("hi")
	if f == nil || f.value != "hi" {
		t.Fatalf("unexpected fringe from nil pool: %+v", f)
	}

	// put functions should not panic on nil pool
	mp.putNode(n)
	mp.putLeaf(l)
	mp.putFringe(f)
}
