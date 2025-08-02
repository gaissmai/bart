package bart

// Cloner is an interface, if implemented by payload of type V the values are deeply cloned
// during [Table.InsertPersist], [Table.UpdatePersist], [Table.DeletePersist], [Table.Clone] and [Table.Union].
type Cloner[V any] interface {
	Clone() V
}

// isCloner returns true if the value type V implements the Cloner[V] interface.
func isCloner[V any](val V) (ok bool) {
	_, ok = any(val).(Cloner[V])
	return
}

// cloneFnFactory returns a function making either a deep copy or a shallow copy of the given value v,
// depending on whether the value type V implements the Cloner[V] interface.
func cloneFnFactory[V any]() (fn func(V) V) {
	var zero V
	if isCloner(zero) {
		return cloneVal[V]
	}
	return copyVal[V]
}

// cloneVal, if the provided value implements Cloner[V], its Clone method is invoked to produce
// a deep copy.
func cloneVal[V any](val V) V {
	return any(val).(Cloner[V]).Clone()
}

// copyVal is just a shallow copy.
func copyVal[V any](val V) V {
	return val
}

// cloneLeaf creates a clone or copy of the current leafNode[V].
func (l *leafNode[V]) cloneLeaf(cloneFn func(val V) V) *leafNode[V] {
	return &leafNode[V]{prefix: l.prefix, value: cloneFn(l.value)}
}

// cloneFringe creates a clone or copy of the current fringeNode[V].
func (l *fringeNode[V]) cloneFringe(cloneFn func(val V) V) *fringeNode[V] {
	return &fringeNode[V]{value: cloneFn(l.value)}
}

// cloneLeafOrFringe takes a value of dynamic type 'any' that represents a pointer to a node,
// and returns a cloned copy of that node depending on its actual concrete type.
func cloneLeafOrFringe[V any](anyKid any) any {
	cloneFn := cloneFnFactory[V]()

	switch kid := anyKid.(type) {
	case *node[V]:
		return any(kid) // just copy
	case *leafNode[V]:
		return any(kid.cloneLeaf(cloneFn))
	case *fringeNode[V]:
		return any(kid.cloneFringe(cloneFn))
	default:
		panic("logic error, wrong node type")
	}
}

// cloneFlat creates a shallow copy of the current node[V], with optional deep copies of values.
//
// This method is intended for fast, non-recursive cloning of a node structure. It copies only
// the current node and selectively performs deep copies of stored values, without recursively
// cloning child nodes.
func (n *node[V]) cloneFlat() *node[V] {
	if n == nil {
		return nil
	}

	c := new(node[V])
	if n.isEmpty() {
		return c
	}

	cloneFn := cloneFnFactory[V]()

	// deep clone
	c.prefixes = *(n.prefixes.Clone(cloneFn))

	// flat clone, not traversing the node levels
	c.children = *(n.children.Clone(cloneLeafOrFringe[V]))

	return c
}

// cloneRec performs a recursive deep copy of the node[V].
//
// If the value type V implements the Cloner[V] interface,
// each value is deep-copied.
func (n *node[V]) cloneRec() *node[V] {
	if n == nil {
		return nil
	}

	c := n.cloneFlat()

	// clone the child nodes rec-descent
	for i, kidAny := range c.children.Items {
		// leaves and fringes are already flat cloned
		if kid, ok := kidAny.(*node[V]); ok {
			c.children.Items[i] = kid.cloneRec()
		}
	}

	return c
}
