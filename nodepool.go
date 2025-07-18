package bart

import "sync"

// pool is a type-safe wrapper around sync.Pool specialized for managing *node[V] instances.
//
// The generic type parameter V allows the pool to be used with generic node[V] type.
type pool[V any] struct {
	sync.Pool
}

// newNodePool creates and returns a new pool for nodes of type V.
//
// It initializes the internal sync.Pool with a New function that
// returns a new *node[V].
func newNodePool[V any]() *pool[V] {
	return &pool[V]{
		Pool: sync.Pool{
			New: func() interface{} {
				return new(node[V])
			},
		},
	}
}

// Get retrieves a *node[V] instance from the pool.
//
// If the pool is nil, it returns a newly allocated *node[V].
func (p *pool[V]) Get() *node[V] {
	if p == nil {
		return new(node[V])
	}
	return p.Pool.Get().(*node[V])
}

// Put returns a *node[V] instance back to the pool for potential reuse.
//
// If the pool is nil, the node is discarded.
func (p *pool[V]) Put(n *node[V]) {
	if p == nil {
		return
	}
	p.Pool.Put(n)
}
