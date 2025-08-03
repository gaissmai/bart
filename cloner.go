package bart

// Cloner is an interface that enables deep cloning of values of type V.
// If a value implements Cloner[V], Table methods such as InsertPersist, UpdatePersist,
// DeletePersist, Clone, and Union will use its Clone method to perform deep copies.
type Cloner[V any] interface {
	Clone() V
}

// isCloner reports whether the type of val implements the Cloner[V] interface.
func isCloner[V any](val V) (ok bool) {
	_, ok = any(val).(Cloner[V])
	return
}

// cloneFunc is a type definition for a function that takes a value of type V
// and returns a (possibly cloned) value of type V.
type cloneFunc[V any] func(V) V

// cloneFnFactory returns a cloneFunc.
// If V implements Cloner[V], the returned function performs a deep copy using Clone;
// otherwise, it returns the value directly (shallow copy).
func cloneFnFactory[V any]() cloneFunc[V] {
	var zero V
	if isCloner(zero) {
		return cloneVal[V]
	}
	return nil
}

// cloneVal invokes the Clone method to deeply copy val.
// Assumes that val implements Cloner[V].
func cloneVal[V any](val V) V {
	return any(val).(Cloner[V]).Clone()
}

// cloneLeaf returns a cloned copy of the receiver leafNode
// by applying cloneFn to its value.
func (l *leafNode[V]) cloneLeaf(cloneFn cloneFunc[V]) *leafNode[V] {
	if cloneFn == nil {
		return &leafNode[V]{prefix: l.prefix, value: l.value}
	}
	return &leafNode[V]{prefix: l.prefix, value: cloneFn(l.value)}
}

// cloneFringe returns a cloned copy of the receiver fringeNode
// by applying cloneFn to its value.
func (l *fringeNode[V]) cloneFringe(cloneFn cloneFunc[V]) *fringeNode[V] {
	if cloneFn == nil {
		return &fringeNode[V]{value: l.value}
	}
	return &fringeNode[V]{value: cloneFn(l.value)}
}

// cloneFlat returns a shallow copy of the current node[V], optionally performing deep copies of values.
//
// This method performs a quick, non-recursive clone of the node itself. It copies the node’s fields,
// applying deep cloning only to stored values using cloneFn, while cloning child nodes shallowly.
// It does not recursively clone descendants beyond the immediate children.
func (n *node[V]) cloneFlat(cloneFn cloneFunc[V]) *node[V] {
	if n == nil {
		return nil
	}

	c := new(node[V])
	if n.isEmpty() {
		return c
	}

	// copy ...
	c.prefixes = *(n.prefixes.Copy())

	// ... or deep clone
	if cloneFn != nil {
		for i, v := range c.prefixes.Items {
			c.prefixes.Items[i] = cloneFn(v)
		}
	}

	// copy ...
	c.children = *(n.children.Copy())

	// ... and flat clone, not traversing the node levels
	for i, anyKid := range c.children.Items {
		switch kid := anyKid.(type) {
		case *node[V]:
			// no-op
		case *leafNode[V]:
			c.children.Items[i] = kid.cloneLeaf(cloneFn)
		case *fringeNode[V]:
			c.children.Items[i] = kid.cloneFringe(cloneFn)
		default:
			panic("logic error, wrong node type")
		}
	}

	return c
}

// cloneRec performs a recursive deep copy of the node[V] and all its descendants.
//
// If the value type V implements the Cloner[V] interface, each value is deep-copied using cloneFn.
//
// The method clones the current node using cloneFlat, then recursively clones all child nodes of type *node[V].
func (n *node[V]) cloneRec(cloneFn cloneFunc[V]) *node[V] {
	if n == nil {
		return nil
	}

	c := n.cloneFlat(cloneFn)

	// clone the child nodes rec-descent
	for i, kidAny := range c.children.Items {
		// leaves and fringes are already flat cloned
		if kid, ok := kidAny.(*node[V]); ok {
			c.children.Items[i] = kid.cloneRec(cloneFn)
		}
	}

	return c
}
