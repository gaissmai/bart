package bart

import (
	"net/netip"
	"sync"
	"sync/atomic"
)

// multiPool groups sub-pools for internal node, leaf, and fringe types.
// Each sub-multiPool handles allocation, reuse, and statistics tracking
// for its corresponding node type.
type multiPool[V any] struct {
	node   *nodePool[V]
	leaf   *leafPool[V]
	fringe *fringePool[V]
}

// newMultiPool initializes and returns a new pool structure with sub-pools
// for internal, leaf, and fringe nodes.
func newMultiPool[V any]() *multiPool[V] {
	return &multiPool[V]{
		node:   newNodePool[V](),
		leaf:   newLeafPool[V](),
		fringe: newFringePool[V](),
	}
}

// getNode obtains a *node[V] from the pool.
// If the parent pool is nil, a new instance is returned without tracking.
func (mp *multiPool[V]) getNode() *node[V] {
	if mp == nil {
		return new(node[V])
	}
	mp.node.currentLive.Add(1)
	return mp.node.Get().(*node[V])
}

// getLeaf obtains a *leafNode[V] from the pool, initialized with
// a prefix and value. If the pool is nil, a fresh instance is created.
func (mp *multiPool[V]) getLeaf(pfx netip.Prefix, val V) *leafNode[V] {
	if mp == nil {
		return &leafNode[V]{prefix: pfx, value: val}
	}
	mp.leaf.currentLive.Add(1)
	l := mp.leaf.Get().(*leafNode[V])
	l.prefix = pfx
	l.value = val
	return l
}

// getFringe obtains a *fringeNode[V] from the pool, initialized with a value.
// If the pool is nil, a new instance is returned without tracking.
func (mp *multiPool[V]) getFringe(val V) *fringeNode[V] {
	if mp == nil {
		return &fringeNode[V]{value: val}
	}
	mp.fringe.currentLive.Add(1)
	f := mp.fringe.Get().(*fringeNode[V])
	f.value = val
	return f
}

// putNode returns an internal node back to its pool for reuse.
// If the pool is nil, the node is discarded.
func (mp *multiPool[V]) putNode(n *node[V]) {
	if mp != nil {
		n.reset() // clear internal state but keep allocated memory
		mp.node.currentLive.Add(-1)
		mp.node.Put(n)
	}
}

// putLeaf returns a leaf node back to its pool for reuse.
// If the pool is nil, the node is discarded.
func (mp *multiPool[V]) putLeaf(l *leafNode[V]) {
	if mp != nil {
		mp.leaf.currentLive.Add(-1)
		mp.leaf.Put(l)
	}
}

// putFringe returns a fringe node back to its pool for reuse.
// If the pool is nil, the node is discarded.
func (mp *multiPool[V]) putFringe(f *fringeNode[V]) {
	if mp != nil {
		mp.fringe.currentLive.Add(-1)
		mp.fringe.Put(f)
	}
}

// nodeStats returns the number of currently live (checked-out) nodes
// and the total number of *node[V] objects ever allocated by this pool.
func (mp *multiPool[V]) nodeStats() (live int64, total int64) {
	if mp == nil {
		return 0, 0
	}
	return mp.node.currentLive.Load(), mp.node.totalAllocated.Load()
}

// leafStats returns the current number of in-use leaf nodes and
// the total number created across the pool's lifetime.
func (mp *multiPool[V]) leafStats() (live int64, total int64) {
	if mp == nil {
		return 0, 0
	}
	return mp.leaf.currentLive.Load(), mp.leaf.totalAllocated.Load()
}

// leafStats returns the current number of in-use fringe nodes and
// the total number created across the pool's lifetime.
func (mp *multiPool[V]) fringeStats() (live int64, total int64) {
	if mp == nil {
		return 0, 0
	}
	return mp.fringe.currentLive.Load(), mp.fringe.totalAllocated.Load()
}

// ##################################################################

// nodePool is a type-safe wrapper around sync.Pool,
// specialized for managing *node[V] instances.
//
// It supports efficient memory reuse and tracks allocation
// and usage statistics to aid debugging and profiling.
type nodePool[V any] struct {
	sync.Pool
	totalAllocated atomic.Int64 // total number of *node[V] instances ever created
	currentLive    atomic.Int64 // number of currently checked-out (in-use) nodes
}

// newNodePool constructs and returns a nodePool with tracking enabled.
func newNodePool[V any]() *nodePool[V] {
	np := &nodePool[V]{}
	np.New = func() any {
		np.totalAllocated.Add(1)
		return new(node[V])
	}
	return np
}

// ##################################################################

// leafPool is a sync.Pool wrapper for *leafNode[V] objects.
// It tracks allocation and reuse statistics for monitoring purposes.
type leafPool[V any] struct {
	sync.Pool
	totalAllocated atomic.Int64
	currentLive    atomic.Int64
}

// newLeafPool initializes a leafPool instance with a node constructor.
func newLeafPool[V any]() *leafPool[V] {
	lp := &leafPool[V]{}
	lp.New = func() any {
		lp.totalAllocated.Add(1)
		return new(leafNode[V])
	}
	return lp
}

// ##################################################################

// fringePool is a type-safe wrapper around sync.Pool,
// specialized for managing *node[V] instances.
//
// It efficiently reuses node memory and tracks statistics
// on allocations and active use for debugging and performance tuning.
type fringePool[V any] struct {
	sync.Pool // embedded Sync Pool for *node[V]

	totalAllocated atomic.Int64 // total number of *node[V] ever allocated
	currentLive    atomic.Int64 // number of nodes currently in use (not returned to pool)
}

// newFringePool creates and returns a new pool for *fringeNode[V] instances.
//
// The pool uses sync.Pool internally, and defines a New function
// that creates new nodes with statistical tracking.
func newFringePool[V any]() *fringePool[V] {
	fp := &fringePool[V]{}
	fp.New = func() any {
		fp.totalAllocated.Add(1)

		return new(fringeNode[V])
	}
	return fp
}
