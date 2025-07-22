package bart

import (
	"net/netip"
	"sync"
	"sync/atomic"
)

// multiPool groups sub-pools dedicated to internal child nodes, leaf nodes,
// and fringe nodes. Each sub-pool manages allocation, reuse, and
// statistics tracking for its respective node type.
type multiPool[V any] struct {
	node   *nodePool[V]
	leaf   *leafPool[V]
	fringe *fringePool[V]
}

// newMultiPool creates and returns a new multiPool containing
// separate pools for internal nodes, leaf nodes, and fringe nodes.
func newMultiPool[V any]() *multiPool[V] {
	return &multiPool[V]{
		node:   &nodePool[V]{},
		leaf:   &leafPool[V]{},
		fringe: &fringePool[V]{},
	}
}

// nodePool manages *node[V] instances with a sync.Pool,
// tracking total allocations and in-use count for diagnostics.
type nodePool[V any] struct {
	sync.Pool
	totalAllocated atomic.Int64 // total *node[V] instances allocated
	currentLive    atomic.Int64 // currently checked-out instances
}

// leafPool manages *leafNode[V] instances using sync.Pool,
// tracking allocations and usage counts for monitoring.
type leafPool[V any] struct {
	sync.Pool
	totalAllocated atomic.Int64 // total *leafNode[V] instances allocated
	currentLive    atomic.Int64 // currently in-use instances
}

// fringePool manages *fringeNode[V] instances with sync.Pool,
// tracking allocation counts and active usage for profiling and debugging.
type fringePool[V any] struct {
	sync.Pool

	totalAllocated atomic.Int64 // total *fringeNode[V] instances allocated
	currentLive    atomic.Int64 // currently in-use instances
}

// ##########################################################################

// getNode retrieves an internal node from the pool, incrementing the live allocation count.
// If the multiPool receiver is nil, indicating no sub-pools exist,
// it returns a newly allocated internal node instance without tracking or reuse.
func (mp *multiPool[V]) getNode() (n *node[V]) {
	if mp == nil {
		return &node[V]{}
	}

	// Update statistics
	mp.node.currentLive.Add(1)

	// Try to get a *node[V] from the internal sync.Pool
	i := mp.node.Get()
	if i == nil {
		// Update statistics
		mp.node.totalAllocated.Add(1)

		// Fallback for nil return: explicitly allocate a new node
		return &node[V]{}
	}

	// Return the cached node
	return i.(*node[V])
}

// getLeaf retrieves a leaf node from the pool, initialized with the given
// prefix and value, incrementing the live allocation count.
// If the multiPool receiver is nil, indicating no sub-pools exist,
// it returns a newly allocated leaf node instance without tracking or reuse.
func (mp *multiPool[V]) getLeaf(pfx netip.Prefix, val V) *leafNode[V] {
	if mp == nil {
		return &leafNode[V]{prefix: pfx, value: val}
	}

	// Update statistics
	mp.leaf.currentLive.Add(1)

	// Try to get a *leafNode[V] from the internal sync.Pool
	i := mp.leaf.Get()
	if i == nil {
		// Update statistics
		mp.leaf.totalAllocated.Add(1)

		// Fallback for nil return: explicitly allocate a new node
		return &leafNode[V]{prefix: pfx, value: val}
	}

	l := i.(*leafNode[V])
	// Initialize the reused leaf with the caller-provided prefix and value
	l.prefix = pfx
	l.value = val

	// Return the cached node
	return l
}

// getFringe retrieves a fringe node from the pool, initialized with the given
// value, incrementing the live allocation count.
// If the multiPool receiver is nil, indicating no sub-pools exist,
// it returns a newly allocated fringe node instance without tracking or reuse.
func (mp *multiPool[V]) getFringe(val V) *fringeNode[V] {
	if mp == nil {
		return &fringeNode[V]{value: val}
	}

	// Update statistics
	mp.fringe.currentLive.Add(1)

	// Try to get a *fringeNode[V] from the internal sync.Pool
	i := mp.fringe.Get()
	if i == nil {
		// Update statistics
		mp.fringe.totalAllocated.Add(1)

		// Fallback for nil return: explicitly allocate a new node
		return &fringeNode[V]{value: val}
	}

	f := i.(*fringeNode[V])

	// Initialize the reused fringe with the caller-provided value
	f.value = val

	// Return the cached node
	return f
}

// putNode returns an internal node to its pool for reuse,
// decrementing the live allocation count.
// If the multiPool receiver is nil, indicating no sub-pools exist,
// the node is discarded since there is no pool to return it to.
func (mp *multiPool[V]) putNode(n *node[V]) {
	if mp == nil {
		return
	}
	n.reset() // reset internal state but keep allocated memory
	mp.node.currentLive.Add(-1)
	mp.node.Put(n)
}

// putLeaf returns a leaf node to its pool for reuse,
// decrementing the live allocation count.
// If the multiPool receiver is nil, indicating no sub-pools exist,
// the node is discarded since there is no pool to return it to.
func (mp *multiPool[V]) putLeaf(l *leafNode[V]) {
	if mp == nil {
		return
	}
	// reset
	var zero V
	l.value = zero
	l.prefix = netip.Prefix{}

	mp.leaf.currentLive.Add(-1)
	mp.leaf.Put(l)
}

// putFringe returns a fringe node to its pool for reuse,
// decrementing the live allocation count.
// If the multiPool receiver is nil, indicating no sub-pools exist,
// the node is discarded since there is no pool to return it to.
func (mp *multiPool[V]) putFringe(f *fringeNode[V]) {
	if mp == nil {
		return
	}
	// reset
	var zero V
	f.value = zero

	mp.fringe.currentLive.Add(-1)
	mp.fringe.Put(f)
}

// nodeStats returns the count of currently live internal nodes and
// the total number of internal nodes allocated during the pool's lifetime.
func (mp *multiPool[V]) nodeStats() (live int64, total int64) {
	if mp == nil {
		return 0, 0
	}
	return mp.node.currentLive.Load(), mp.node.totalAllocated.Load()
}

// leafStats returns the count of currently live leaf nodes and
// the total number of leaf nodes allocated during the pool's lifetime.
func (mp *multiPool[V]) leafStats() (live int64, total int64) {
	if mp == nil {
		return 0, 0
	}
	return mp.leaf.currentLive.Load(), mp.leaf.totalAllocated.Load()
}

// fringeStats returns the count of currently live fringe nodes and
// the total number of fringe nodes allocated during the pool's lifetime.
func (mp *multiPool[V]) fringeStats() (live int64, total int64) {
	if mp == nil {
		return 0, 0
	}
	return mp.fringe.currentLive.Load(), mp.fringe.totalAllocated.Load()
}
