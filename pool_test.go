package bart

import (
	"testing"
)

func TestNodePool_ReuseAndStats(t *testing.T) {
	t.Parallel()

	pool := newPool[string]()

	live0, total0 := pool.Stats()
	if live0 != 0 || total0 != 0 {
		t.Fatalf("initial stats incorrect: live=%d, total=%d", live0, total0)
	}

	// Get a node from the pool
	n1 := pool.Get()
	n1.prefixes.InsertAt(42, "foo")
	n1.children.InsertAt(192, &leafNode[string]{prefix: mpp("192.0.2.0/24"), value: "bar"})

	live1, total1 := pool.Stats()
	if live1 != 1 || total1 != 1 {
		t.Errorf("expected live=1 and total=1 after Get; got live=%d, total=%d", live1, total1)
	}

	// Return to pool
	pool.Put(n1)

	live2, total2 := pool.Stats()
	if live2 != 0 || total2 != 1 {
		t.Errorf("expected live=0, total=1 after Put(); got live=%d, total=%d", live2, total2)
	}

	// Get again: should reuse
	n2 := pool.Get()

	// test is node is reset
	if n2.prefixes.Len() != 0 || n2.children.Len() != 0 {
		t.Error("expected reused node to be reset")
	}
	if val, ok := n2.prefixes.Get(42); ok {
		t.Errorf("expected prefix 5 to be cleared, got value: %v", val)
	}
	if val, ok := n2.children.Get(192); ok {
		t.Errorf("expected leaf at 192 to be cleared, got value: %v", val)
	}

	pool.Put(n2)
}
