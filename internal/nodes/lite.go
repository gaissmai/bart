// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package nodes

import (
	"iter"
	"slices"

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
//  3. Fringe Merging: Collapses pairs of adjacent FringeNode children into
//     a single supernet prefix inserted into the current node's bitset.
//  4. Leaf Merging: Merges pairs of adjacent LeafNode children covering a contiguous
//     range into a single LeafNode representing their common supernet.
//  5. Prefix Merging: Combines pairs of adjacent sibling prefixes (e.g., sharing
//     the same parent bit sequence) into their higher-level supernet prefix.
//  6. Recursive Descent: Recursively calls Aggregate on child LiteNode instances.
//
// Returns deleted, the net number of prefix/child entries pruned or merged during
// the bottom-up compression pass.
func (n *LiteNode[V]) Aggregate() (deleted int) {
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
			deleted++
		}
	}

	// 2. Child Subsumption: Remove child nodes covered by any prefix in this node.
	for idx := range n.Prefixes.All() {
		covered := n.Children.Intersection(&allot.FringeRoutesLookupTbl[idx])
		for addr := range covered.All() {
			n.DeleteChild(addr)
			deleted++
		}
	}

	// 3. Fringe Merging: Collapse adjacent FringeNode pairs into a supernet prefix.
	more := true
	for more { // loop as long as changes occur, maybe many passes
		more = false

		var lastFringeAddr uint8
		var lastFringe *FringeNode[V]
		for addr := range n.Children.All() {
			anyKid := n.MustGetChild(addr)

			fringe, ok := anyKid.(*FringeNode[V])
			if !ok {
				continue
			}

			// start/restart
			if lastFringe == nil {
				lastFringeAddr = addr
				lastFringe = fringe
				continue
			}

			// check adjacency (even address XOR 1 must equal following odd address)
			// e.g. 8^1 == 9, 7^1 == 6
			if lastFringeAddr^1 != addr {
				lastFringeAddr = addr
				lastFringe = fringe
				continue
			}

			n.InsertPrefix(art.PfxToIdx(lastFringeAddr, 7), zero)
			n.DeleteChild(lastFringeAddr)
			n.DeleteChild(addr)

			deleted++ // 1 prefix inserted, 2 fringes deleted
			more = true

			// reset
			lastFringeAddr = 0
			lastFringe = nil
		}
	}

	// 4. Leaf Merging: Collapse adjacent LeafNode pairs into a single supernet leaf.
	more = true
	for more { // loop as long as changes occur, maybe many passes
		more = false
		var lastLeafAddr uint8
		var lastLeaf *LeafNode[V]

		for _, addr := range n.Children.AppendBits(make([]uint8, 0, n.ChildCount())) {
			anyKid := n.MustGetChild(addr)
			leaf, ok := anyKid.(*LeafNode[V])
			if !ok {
				continue
			}

			// start/restart
			if lastLeaf == nil {
				lastLeafAddr = addr
				lastLeaf = leaf
				continue
			}

			// check adjacency (even address XOR 1 must equal following odd address)
			// e.g. 8^1 == 9, 7^1 == 6
			if lastLeafAddr^1 != addr {
				lastLeafAddr = addr
				lastLeaf = leaf
				continue
			}

			superPfx, ok := Supernet(lastLeaf.Prefix, leaf.Prefix)
			if !ok {
				lastLeafAddr = addr
				lastLeaf = leaf
				continue
			}

			n.InsertChild(lastLeafAddr, NewLeafNode(superPfx, zero))
			n.DeleteChild(addr)
			deleted++ // 1 leaf updated, 1 leaf deleted
			more = true

			// reset
			lastLeafAddr = 0
			lastLeaf = nil
		}
	}

	// 5. Prefix Merging: Merge adjacent prefixes within the bitset.
	idxs := n.Prefixes.AppendBits(make([]uint8, 0, n.PrefixCount()))
	lastIdx := uint8(0)
	for _, idx := range slices.Backward(idxs) {
		if idx <= 1 {
			break
		}

		if lastIdx == 0 {
			lastIdx = idx
			continue
		}

		// test adjacent, 35>>1 = 17, 34>>1 = 17, set 17, delete 34,35
		if lastIdx>>1 == idx>>1 {
			// insert supernet
			n.InsertPrefix(idx>>1, zero)

			// delete subnets
			n.DeletePrefix(lastIdx)
			n.DeletePrefix(idx)

			// 1 inserted, 2 deleted => 1
			deleted++
		}

		lastIdx = idx
	}

	// 6. Recursive Descent: Top-down compression of child LiteNodes.
	for _, anyKid := range n.Children.Items {
		if kid, ok := anyKid.(*LiteNode[V]); ok {
			deleted += kid.Aggregate()
		}
	}

	return deleted
}
