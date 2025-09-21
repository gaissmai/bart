// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

// Cloner is an interface that enables deep cloning of values of type V.
// If a value implements Cloner[V], Table methods such as InsertPersist, UpdatePersist,
// DeletePersist, Clone, and Union will use its Clone method to perform deep copies.
type Cloner[V any] interface {
	Clone() V
}

// cloneFunc is a type definition for a function that takes a value of type V
// and returns the (possibly cloned) value of type V.
type cloneFunc[V any] func(V) V

// cloneFnFactory returns a cloneFunc.
// If V implements Cloner[V], the returned function should perform
// a deep copy using Clone(), otherwise it returns nil.
func cloneFnFactory[V any]() cloneFunc[V] {
	var zero V
	// you can't assert directly on a type parameter
	if _, ok := any(zero).(Cloner[V]); ok {
		return cloneVal[V]
	}
	return nil
}

// cloneVal returns a deep clone of val by calling its Clone method when
// val implements Cloner[V]. If val does not implement Cloner[V] or the
// asserted Cloner is nil, val is returned unchanged.
func cloneVal[V any](val V) V {
	// you can't assert directly on a type parameter
	c, ok := any(val).(Cloner[V])
	if !ok || c == nil {
		return val
	}
	return c.Clone()
}

// copyVal just copies the value.
func copyVal[V any](val V) V {
	return val
}

// cloneLeaf creates and returns a copy of the leafNode receiver.
// If cloneFn is nil, the value is copied directly without modification.
// Otherwise, cloneFn is applied to the value for deep cloning.
// The prefix field is always copied as is.
func (l *leafNode[V]) cloneLeaf(cloneFn cloneFunc[V]) *leafNode[V] {
	if cloneFn == nil {
		return &leafNode[V]{prefix: l.prefix, value: l.value}
	}
	return &leafNode[V]{prefix: l.prefix, value: cloneFn(l.value)}
}

// cloneFringe creates and returns a copy of the fringeNode receiver.
// If cloneFn is nil, the value is copied directly without modification.
// Otherwise, cloneFn is applied to the value for deep cloning.
func (l *fringeNode[V]) cloneFringe(cloneFn cloneFunc[V]) *fringeNode[V] {
	if cloneFn == nil {
		return &fringeNode[V]{value: l.value}
	}
	return &fringeNode[V]{value: cloneFn(l.value)}
}

// ############################################################################
// # why no generic version of cloneFlat and cloneRec?                        #
// # - avoid interface boxing and extra heap allocations                      #
// # - direct Array256.Items access is much faster than generic getIndices/   #
// #   getChildAddrs method calls                                             #
// ############################################################################

// cloneFlat returns a shallow copy of the current node[V], optionally performing deep copies of values.
//
// If cloneFn is nil, the stored values in prefixes are copied directly without modification.
// Otherwise, cloneFn is applied to each stored value for deep cloning.
// Child nodes are cloned shallowly: leafNode and fringeNode children are cloned via their clone methods,
// but child nodes of type *bartNode[V] (subnodes) are assigned as-is without recursive cloning.
// This method does not recursively clone descendants beyond the immediate children.
//
// Note: The returned node is a new instance with copied slices but only shallow copies of nested nodes,
// except for leafNode and fringeNode children which are cloned according to cloneFn.
func (n *bartNode[V]) cloneFlat(cloneFn cloneFunc[V]) *bartNode[V] {
	if n == nil {
		return nil
	}

	c := new(bartNode[V])
	if n.isEmpty() {
		return c
	}

	// copy ...
	c.prefixes = *(n.prefixes.Copy())

	// ... and clone the values
	if cloneFn != nil {
		for i, v := range c.prefixes.Items {
			c.prefixes.Items[i] = cloneFn(v)
		}
	}

	// copy ...
	c.children = *(n.children.Copy())

	// Iterate over children to flat clone leaf/fringe nodes;
	// for *bartNode[V] children, keep shallow references (no recursive clone)
	for i, anyKid := range c.children.Items {
		switch kid := anyKid.(type) {
		case *bartNode[V]:
			// Shallow copy
		case *leafNode[V]:
			// Clone leaf nodes, applying cloneFn as needed
			c.children.Items[i] = kid.cloneLeaf(cloneFn)
		case *fringeNode[V]:
			// Clone fringe nodes, applying cloneFn as needed
			c.children.Items[i] = kid.cloneFringe(cloneFn)
		default:
			panic("logic error, wrong node type")
		}
	}

	return c
}

// cloneRec performs a recursive deep copy of the node[V] and all its descendants.
//
// If cloneFn is nil, the stored values are copied directly without modification.
// Otherwise cloneFn is applied to each stored value for deep cloning.
//
// This method first creates a shallow clone of the current node using cloneFlat,
// applying cloneFn to values as described there. Then it recursively clones all
// child nodes of type *bartNode[V], performing a full deep clone down the subtree.
//
// Child nodes of type *leafNode[V] and *fringeNode[V] are already cloned
// by cloneFlat.
//
// Returns a new instance of node[V] which is a complete deep clone of the
// receiver node with all descendants.
func (n *bartNode[V]) cloneRec(cloneFn cloneFunc[V]) *bartNode[V] {
	if n == nil {
		return nil
	}

	// Perform a flat clone of the current node.
	c := n.cloneFlat(cloneFn)

	// Recursively clone all child nodes of type *bartNode[V]
	for i, kidAny := range c.children.Items {
		if kid, ok := kidAny.(*bartNode[V]); ok {
			c.children.Items[i] = kid.cloneRec(cloneFn)
		}
	}

	return c
}

// cloneFlat returns a shallow copy of the current fastNode[V],
// Its semantics are identical to [node.cloneFlat].
func (n *fastNode[V]) cloneFlat(cloneFn cloneFunc[V]) *fastNode[V] {
	if n == nil {
		return nil
	}

	c := new(fastNode[V])
	if n.isEmpty() {
		return c
	}

	// copy the bitsets
	c.prefixes.BitSet256 = n.prefixes.BitSet256
	c.children.BitSet256 = n.children.BitSet256

	// copy the counters
	c.pfxCount = n.pfxCount
	c.cldCount = n.cldCount

	// it's a clone of the prefixes ...
	// but the allot algorithm makes it more difficult
	// see also insertPrefix
	for _, idx := range n.getIndices() {
		origValPtr := n.prefixes.items[idx]
		newValPtr := new(V)

		if cloneFn == nil {
			*newValPtr = *origValPtr // just copy the value
		} else {
			*newValPtr = cloneFn(*origValPtr) // clone the value
		}

		oldValPtr := c.prefixes.items[idx]
		c.allot(idx, oldValPtr, newValPtr)
	}

	// flat clone of the children
	for _, addr := range n.getChildAddrs() {
		kidAny := *n.children.items[addr]

		switch kid := kidAny.(type) {
		case *fastNode[V]:
			// just copy the pointer
			c.children.items[addr] = n.children.items[addr]

		case *leafNode[V]:
			leafAny := any(kid.cloneLeaf(cloneFn))
			c.children.items[addr] = &leafAny

		case *fringeNode[V]:
			fringeAny := any(kid.cloneFringe(cloneFn))
			c.children.items[addr] = &fringeAny

		default:
			panic("logic error, wrong node type")
		}

	}

	return c
}

// cloneRec performs a recursive deep copy of the fastNode[V] and all its descendants.
// Its semantics are identical to [node.cloneRec].
func (n *fastNode[V]) cloneRec(cloneFn cloneFunc[V]) *fastNode[V] {
	if n == nil {
		return nil
	}

	// Perform a flat clone of the current node.
	c := n.cloneFlat(cloneFn)

	// Recursively clone all child nodes of type *fastNode[V]
	for _, addr := range c.getChildAddrs() {
		kidAny := *c.children.items[addr]

		switch kid := kidAny.(type) {
		case *fastNode[V]:
			nodeAny := any(kid.cloneRec(cloneFn))
			c.children.items[addr] = &nodeAny
		}
	}

	return c
}

// cloneFlat returns a shallow copy of the current node[V].
//
// cloneFn is only used for interface satisfaction.
func (n *liteNode[V]) cloneFlat(_ cloneFunc[V]) *liteNode[V] {
	if n == nil {
		return nil
	}

	c := new(liteNode[V])
	if n.isEmpty() {
		return c
	}

	// copy simple values
	c.pfxCount = n.pfxCount
	c.prefixes = n.prefixes

	// sparse array
	c.children = *(n.children.Copy())

	// no values to copy
	return c
}

// cloneRec performs a recursive deep copy of the node[V] and all its descendants.
//
// cloneFn is only used for interface satisfaction.
//
// It first creates a shallow clone of the current node using cloneFlat.
// Then it recursively clones all child nodes of type *liteNode[V],
// performing a full deep clone down the subtree.
//
// Child nodes of type *liteLeafNode and *liteFringeNode are already copied
// by cloneFlat.
//
// Returns a new instance of liteNode[V] which is a complete deep clone of the
// receiver node with all descendants.
func (n *liteNode[V]) cloneRec(_ cloneFunc[V]) *liteNode[V] {
	if n == nil {
		return nil
	}

	// Perform a flat clone of the current node.
	c := n.cloneFlat(nil)

	// Recursively clone all child nodes of type *liteNode[V]
	for i, kidAny := range c.children.Items {
		if kid, ok := kidAny.(*liteNode[V]); ok {
			c.children.Items[i] = kid.cloneRec(nil)
		}
	}

	return c
}
