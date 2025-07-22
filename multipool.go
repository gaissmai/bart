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
		node:   newNodePool[V](),
		leaf:   newLeafPool[V](),
		fringe: newFringePool[V](),
	}
}

// getNode retrieves an internal node from the pool, incrementing the live allocation count.
// If the multiPool receiver is nil, indicating no sub-pools exist,
// it returns a newly allocated internal node instance without tracking or reuse.
func (mp *multiPool[V]) getNode() *node[V] {
	if mp == nil {
		return new(node[V])
	}
	mp.node.currentLive.Add(1)
	return mp.node.Get().(*node[V])
}

// getLeaf retrieves a leaf node from the pool, initialized with the given
// prefix and value, incrementing the live allocation count.
// If the multiPool receiver is nil, indicating no sub-pools exist,
// it returns a newly allocated leaf node instance without tracking or reuse.
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

// getFringe retrieves a fringe node from the pool, initialized with the given
// value, incrementing the live allocation count.
// If the multiPool receiver is nil, indicating no sub-pools exist,
// it returns a newly allocated fringe node instance without tracking or reuse.
func (mp *multiPool[V]) getFringe(val V) *fringeNode[V] {
	if mp == nil {
		return &fringeNode[V]{value: val}
	}
	mp.fringe.currentLive.Add(1)
	f := mp.fringe.Get().(*fringeNode[V])
	f.value = val
	return f
}

// putNode returns an internal node to its pool for reuse,
// decrementing the live allocation count.
// If the multiPool receiver is nil, indicating no sub-pools exist,
// the node is discarded since there is no pool to return it to.
func (mp *multiPool[V]) putNode(n *node[V]) {
	if mp != nil {
		n.reset() // reset internal state but keep allocated memory
		mp.node.currentLive.Add(-1)
		mp.node.Put(n)
	}
}

// putLeaf returns a leaf node to its pool for reuse,
// decrementing the live allocation count.
// If the multiPool receiver is nil, indicating no sub-pools exist,
// the node is discarded since there is no pool to return it to.
func (mp *multiPool[V]) putLeaf(l *leafNode[V]) {
	if mp != nil {
		mp.leaf.currentLive.Add(-1)
		mp.leaf.Put(l)
	}
}

// putFringe returns a fringe node to its pool for reuse,
// decrementing the live allocation count.
// If the multiPool receiver is nil, indicating no sub-pools exist,
// the node is discarded since there is no pool to return it to.
func (mp *multiPool[V]) putFringe(f *fringeNode[V]) {
	if mp != nil {
		mp.fringe.currentLive.Add(-1)
		mp.fringe.Put(f)
	}
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

// ##################################################################

// nodePool manages *node[V] instances with a sync.Pool,
// tracking total allocations and in-use count for diagnostics.
type nodePool[V any] struct {
	sync.Pool
	totalAllocated atomic.Int64 // total *node[V] instances allocated
	currentLive    atomic.Int64 // currently checked-out instances
}

// newNodePool creates a nodePool with allocation tracking enabled.
func newNodePool[V any]() *nodePool[V] {
	np := &nodePool[V]{}
	np.New = func() any {
		np.totalAllocated.Add(1)
		return new(node[V])
	}
	return np
}

// ##################################################################

// leafPool manages *leafNode[V] instances using sync.Pool,
// tracking allocations and usage counts for monitoring.
type leafPool[V any] struct {
	sync.Pool
	totalAllocated atomic.Int64 // total *leafNode[V] instances allocated
	currentLive    atomic.Int64 // currently in-use instances
}

// newLeafPool creates a leafPool with allocation tracking.
func newLeafPool[V any]() *leafPool[V] {
	lp := &leafPool[V]{}
	lp.New = func() any {
		lp.totalAllocated.Add(1)
		return new(leafNode[V])
	}
	return lp
}

// ##################################################################

// fringePool manages *fringeNode[V] instances with sync.Pool,
// tracking allocation counts and active usage for profiling and debugging.
type fringePool[V any] struct {
	sync.Pool

	totalAllocated atomic.Int64 // total *fringeNode[V] instances allocated
	currentLive    atomic.Int64 // currently in-use instances
}

// newFringePool creates a fringePool with tracking enabled.
func newFringePool[V any]() *fringePool[V] {
	fp := &fringePool[V]{}
	fp.New = func() any {
		fp.totalAllocated.Add(1)
		return new(fringeNode[V])
	}
	return fp
}
