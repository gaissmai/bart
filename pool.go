package bart

import (
	"sync"
	"sync/atomic"
)

// pool is a type-safe wrapper around sync.Pool,
// specialized for managing *node[V] instances.
//
// It efficiently reuses node memory and tracks statistics
// on allocations and active use for debugging and performance tuning.
type pool[V any] struct {
	sync.Pool // embedded Sync Pool for *node[V]

	// TODO: remove it once the code is stable.
	totalAllocated atomic.Int64 // total number of *node[V] ever allocated
	currentLive    atomic.Int64 // number of nodes currently in use (not returned to pool)
}

// newPool creates and returns a new pool for *node[V] instances.
//
// The pool uses sync.Pool internally, and defines a New function
// that creates new nodes with statistical tracking.
func newPool[V any]() *pool[V] {
	p := &pool[V]{}
	p.New = func() any {
		p.totalAllocated.Add(1) // TODO: remove it once the code is stable.

		return new(node[V])
	}
	return p
}

// Get retrieves a *node[V] from the pool, or creates a new one if needed.
//
// If the pool is nil, a new node is returned without tracking.
// Internally increments the live usage counter.
func (p *pool[V]) Get() *node[V] {
	if p == nil {
		return new(node[V])
	}
	p.currentLive.Add(1) // TODO: remove it once the code is stable.

	return p.Pool.Get().(*node[V])
}

// Put returns a *node[V] back to the pool for potential reuse.
//
// The node is reset (cleared) before storage.
// If the pool is nil, the node is discarded and not reused.
// Decrements the live usage counter.
func (p *pool[V]) Put(n *node[V]) {
	if p == nil {
		return
	}
	p.currentLive.Add(-1) // TODO: remove it once the code is stable.

	n.reset() // reset node´s state but retain storage capacity
	p.Pool.Put(n)
}

// Stats returns the number of currently live (checked-out) nodes
// and the total number of *node[V] objects ever allocated by this pool.
//
// TODO: remove it once the code is stable.
func (p *pool[V]) Stats() (live int64, total int64) {
	if p == nil {
		return 0, 0
	}
	return p.currentLive.Load(), p.totalAllocated.Load()
}
