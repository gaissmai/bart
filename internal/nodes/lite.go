// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package nodes

import (
	"iter"

	"github.com/gaissmai/bart/internal/allot"
	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/bitset"
	"github.com/gaissmai/bart/internal/lpm"
	"github.com/gaissmai/bart/internal/sparse"
)

// LiteNode is a space-optimized version of [BartNode] that tracks prefix existence
// without storing associated values.
type LiteNode[V any] struct {
	Children sparse.Array256[any]
	Prefixes struct {
		// BitSet256 tracks the presence of prefixes at this level.
		bitset.BitSet256
		// Count maintains the current number of set bits, updated on modification
		// to avoid expensive population counting.
		Count uint16
	}
}

// PrefixCount returns the number of prefixes stored in this node.
func (n *LiteNode[V]) PrefixCount() int {
	return int(n.Prefixes.Count)
}

// InsertPrefix adds a routing entry at the specified index.
// It returns true if a prefix already existed at that index,
// false if this is a new insertion.
func (n *LiteNode[V]) InsertPrefix(idx uint8, _ V) (exists bool) {
	if exists = n.Prefixes.Test(idx); exists {
		return exists
	}
	n.Prefixes.Set(idx)
	n.Prefixes.Count++
	return exists
}

// DeletePrefix removes the prefix at the specified index.
// Returns true if the prefix existed, and false otherwise.
func (n *LiteNode[V]) DeletePrefix(idx uint8) (exists bool) {
	if exists = n.Prefixes.Test(idx); !exists {
		return false
	}
	n.Prefixes.Clear(idx)
	n.Prefixes.Count--
	return true
}

func (n *LiteNode[V]) GetPrefix(idx uint8) (_ V, exists bool) {
	// no docstring by intention
	exists = n.Prefixes.Test(idx)
	return
}

func (n *LiteNode[V]) MustGetPrefix(idx uint8) (_ V) {
	// no docstring by intention
	return
}

// AllIndices returns an iterator over all prefix entries.
// Each iteration yields the prefix index (uint8) and its associated value (V).
func (n *LiteNode[V]) AllIndices() iter.Seq2[uint8, V] {
	var zero V
	return func(yield func(uint8, V) bool) {
		for idx := range n.Prefixes.All() {
			if !yield(idx, zero) {
				return
			}
		}
	}
}

// InsertChild adds a child node at the specified address (0-255).
// The child can be a *LiteNode[V], *LeafNode, or *FringeNode.
// Returns true if a child already existed at that address.
func (n *LiteNode[V]) InsertChild(addr uint8, child any) (exists bool) {
	_, exists = n.Children.InsertAt(addr, child)
	return
}

// GetChild retrieves the child node at the specified address.
// Returns the child and true if found, or nil and false if not present.
func (n *LiteNode[V]) GetChild(addr uint8) (any, bool) {
	return n.Children.Get(addr)
}

// MustGetChild retrieves the child at the specified address, panicking if not found.
// This method should only be used when the caller is certain the child exists.
func (n *LiteNode[V]) MustGetChild(addr uint8) any {
	return n.Children.MustGet(addr)
}

// DeleteChild removes the child node at the specified address.
// This operation is idempotent - removing a non-existent child is safe.
func (n *LiteNode[V]) DeleteChild(addr uint8) (exists bool) {
	_, exists = n.Children.DeleteAt(addr)
	return exists
}

// LookupIdx performs a longest-prefix match (LPM) lookup for the given index (idx)
// within the 8-bit stride-based prefix table at this trie depth.
//
// The function returns the matched index and whether a matching prefix
// exists at this level. The value type parameter exists only to satisfy interfaces.
//
// Internally, the prefix table is organized as a complete binary tree (CBT) indexed
// via the baseIndex function. Unlike the original ART algorithm, this implementation
// does not use an allotment-based approach. Instead, it performs CBT backtracking
// using a bitset-based operation with a precomputed backtracking pattern specific to idx.
func (n *LiteNode[V]) LookupIdx(idx uint8) (top uint8, _ V, ok bool) {
	top, ok = n.Prefixes.IntersectionTop(&lpm.LookupTbl[idx])
	return
}

// Lookup is just a simple wrapper for LookupIdx.
func (n *LiteNode[V]) Lookup(idx uint8) (_ V, ok bool) {
	_, _, ok = n.LookupIdx(idx)
	return
}

// CloneFlat returns a shallow copy of the current node.
func (n *LiteNode[V]) CloneFlat(_ func(V) V) *LiteNode[V] {
	if n == nil {
		return nil
	}

	c := new(LiteNode[V])

	// copy simple values
	c.Prefixes = n.Prefixes

	// sparse array
	c.Children = *(n.Children.Copy())

	// no values to copy
	return c
}

// Aggregate compresses the LiteNode in-place by pruning redundant subnets,
// removing child nodes covered by parent prefixes, and merging adjacent siblings
// across prefixes, fringe nodes, and leaf nodes.
//
// The aggregation process executes the following steps in order:
//  1. Prefix Subsumption: Removes more-specific prefixes fully covered by a
//     broader supernet prefix within the same node's bitset.
//  2. Child Subsumption: Deletes child nodes (fringes) that are fully covered
//     by an existing prefix in the current node.
//  3. Recursive Descent: Recursively calls Aggregate on child LiteNode instances.
//  4. Fringe Merging: Collapses pairs of adjacent FringeNode children into
//     a single supernet prefix inserted into the current node's bitset.
//  5. Prefix Merging: Combines pairs of adjacent sibling prefixes (e.g., sharing
//     the same parent bit sequence) into their higher-level supernet prefix.
//
// Returns deleted, the net number of prefix/child entries pruned or merged during
// the bottom-up compression pass.
func (n *LiteNode[V]) Aggregate() (modified int) {
	var zero V

	// 1. Prefix Subsumption: Remove subnets in the bitset that are fully covered by a supernet.
	for super := range n.Prefixes.All() {
		if super == 255 {
			break
		}

		covered := n.Prefixes.Intersection(&allot.PfxRoutesLookupTbl[super])
		for sub := range covered.All() {
			// skip self
			if sub == super {
				continue
			}
			// delete covered subnet
			n.DeletePrefix(sub)
			modified++
		}
	}

	// 2. Child Subsumption: Remove child nodes covered by any prefix in this node.
	for idx := range n.Prefixes.All() {
		covered := n.Children.Intersection(&allot.FringeRoutesLookupTbl[idx])
		for addr := range covered.All() {
			n.DeleteChild(addr)
			modified++
		}
	}

	// 3. Recursive Descent: Top-down compression of child LiteNodes.
	for i, anyKid := range n.Children.Items {
		if kid, ok := anyKid.(*LiteNode[V]); ok {
			modified += kid.Aggregate()

			// no dangling STOP node, can't promote
			if kid.PrefixCount() != 1 || kid.ChildCount() != 0 {
				continue
			}

			// promote as fringe after aggregation
			// a node with just one prefix and default route (idx == 1)
			// is a fringe one level above
			if kid.Prefixes.Test(1) {
				n.Children.Items[i] = NewFringeNode(zero)
			}
		}
	}

	// 4. Fringe Merging: Collapse adjacent FringeNode pairs into a supernet prefix.

	// only aligned pairs are aggregate candidates
	alignedPairs := n.Children.AlignedPairs()
	for addr := range alignedPairs.All() {
		anyKid := n.MustGetChild(addr)
		if _, ok := anyKid.(*FringeNode[V]); !ok {
			continue
		}

		anyKid = n.MustGetChild(addr + 1)
		if _, ok := anyKid.(*FringeNode[V]); !ok {
			continue
		}

		// the aligned child pair are fringes, promote them as prefix: addr/7
		n.InsertPrefix(art.PfxToIdx(addr, 7), zero)
		n.DeleteChild(addr)
		n.DeleteChild(addr + 1)

		modified++
	}

	// 5. Prefix Merging: Merge adjacent prefixes within the bitset.
	for { // loop as long as changes occur, maybe many passes
		more := false

		alignedPairs := n.Prefixes.AlignedPairs()
		for idx := range alignedPairs.All() {
			// insert supernet
			n.InsertPrefix(idx>>1, zero)

			// delete subnets
			n.DeletePrefix(idx)
			n.DeletePrefix(idx + 1)

			modified++
			more = true
		}

		if !more {
			break
		}
	}

	return modified
}
