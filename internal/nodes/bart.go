// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package nodes

import (
	"github.com/gaissmai/bart/internal/sparse"
)

// BartNode represents a single trie level in the multibit routing table.
//
// Unlike the original ART algorithm, this implementation uses popcount-compressed
// sparse arrays instead of fixed-size allocations. Insertions and lookups rely on
// fast bitset operations and precomputed lookup tables to maximize CPU pipelining
// and cache efficiency.
//
// Each BartNode maintains two distinct sparse arrays:
//
//  1. Prefixes: Stores routing entries (prefix -> value) for the current stride.
//     These are laid out as a complete binary tree using the baseIndex()
//     function mapping from the ART algorithm. Prefixes that match exactly at
//     the maximum trie depth are always stored here.
//
//  2. Children: Holds pointers to the next logical levels with a branching
//     factor of 256 (8 bits per stride).
//
// A slot in the Children array may contain one of three types:
//   - *BartNode[T]:   An internal intermediate node for further trie traversal.
//   - *FringeNode[T]: A path-compressed node if it qualifies as fringe [IsFringe]
//   - *LeafNode[T]:   A path-compressed node otherwise.
//
// Note: LeafNode and FringeNode are created through path compression and
// are automatically split into regular BartNodes when a more specific prefix
// is inserted that requires further branching.
type BartNode[T any] struct {
	Prefixes sparse.Array256[T]
	Children sparse.Array256[any]
}

// PrefixCount returns the number of prefixes stored in this node.
func (n *BartNode[T]) PrefixCount() int {
	return len(n.Prefixes.Items)
}

// InsertPrefix adds or updates a routing entry at the specified index with the given value.
// It returns true if a prefix already existed at that index (indicating an update),
// false if this is a new insertion.
func (n *BartNode[T]) InsertPrefix(idx uint8, val T) (exists bool) {
	return n.Prefixes.InsertAt(idx, val)
}

// DeletePrefix removes the prefix at the specified index.
// Returns true if the prefix existed, otherwise false.
func (n *BartNode[T]) DeletePrefix(idx uint8) (exists bool) {
	_, exists = n.Prefixes.DeleteAt(idx)
	return exists
}

// GetPrefix retrieves the value associated with the prefix at the given index.
// Returns the value and true if found, or zero value and false if not present.
func (n *BartNode[T]) GetPrefix(idx uint8) (val T, exists bool) {
	return n.Prefixes.Get(idx)
}

// MustGetPrefix retrieves the value at the specified index, panicking if not found.
// This method should only be used when the caller is certain the index exists.
func (n *BartNode[T]) MustGetPrefix(idx uint8) (val T) {
	return n.Prefixes.MustGet(idx)
}

// CloneFlat returns a shallow copy of the current node, optionally performing deep copies of values.
//
// If cloneFn is nil, the stored values in prefixes are copied directly without modification.
// Otherwise, cloneFn is applied to each stored value for deep cloning.
// Child nodes are cloned shallowly: LeafNode and FringeNode children are cloned via their clone methods,
// but child nodes of type *BartNode[T] (subnodes) are assigned as-is without recursive cloning.
// This method does not recursively clone descendants beyond the immediate children.
//
// Note: The returned node is a new instance with copied slices but only shallow copies of nested nodes,
// except for LeafNode and FringeNode children which are cloned according to cloneFn.
func (n *BartNode[T]) CloneFlat(cloneFn func(T) T) *BartNode[T] {
	if n == nil {
		return nil
	}

	c := new(BartNode[T])

	// copy ...
	c.Prefixes = *(n.Prefixes.Copy())

	// ... and clone the values
	if cloneFn != nil {
		for i, v := range c.Prefixes.Items {
			c.Prefixes.Items[i] = cloneFn(v)
		}
	}

	// copy ...
	c.Children = *(n.Children.Copy())

	// Iterate over children to flat clone leaf/fringe nodes;
	// for *BartNode[T] children, keep shallow references (no recursive clone)
	for i, anyKid := range c.Children.Items {
		switch kid := anyKid.(type) {
		case *BartNode[T]:
			// Shallow copy
		case *LeafNode[T]:
			// Clone leaf nodes, applying cloneFn as needed
			c.Children.Items[i] = kid.CloneLeaf(cloneFn)
		case *FringeNode[T]:
			// Clone fringe nodes, applying cloneFn as needed
			c.Children.Items[i] = kid.CloneFringe(cloneFn)
		default:
			panic("logic error, wrong node type")
		}
	}

	return c
}
