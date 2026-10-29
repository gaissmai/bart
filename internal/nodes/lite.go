// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package nodes

import (
	"github.com/gaissmai/bart/internal/allot"
	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/bitset"
	"github.com/gaissmai/bart/internal/sparse"
)

// LiteNode is a space-optimized version of [BartNode] that tracks prefix existence
// without storing associated values.
type LiteNode[T any] struct {
	Children sparse.Array256[any]
	Prefixes struct {
		bitset.BitSet256
		count uint16
	}
}

// PrefixCount returns the number of prefixes stored in this node.
func (n *LiteNode[T]) PrefixCount() int {
	return int(n.Prefixes.count)
}

// InsertPrefix adds a routing entry at the specified index.
// It returns true if a prefix already existed at that index,
// false if this is a new insertion.
func (n *LiteNode[T]) InsertPrefix(idx uint8, _ T) (exists bool) {
	if exists = n.Prefixes.Test(idx); exists {
		return exists
	}
	n.Prefixes.Set(idx)
	n.Prefixes.count++
	return exists
}

// DeletePrefix removes the prefix at the specified index.
// Returns true if the prefix existed, and false otherwise.
func (n *LiteNode[T]) DeletePrefix(idx uint8) (exists bool) {
	if exists = n.Prefixes.Test(idx); !exists {
		return false
	}
	n.Prefixes.Clear(idx)
	n.Prefixes.count--
	return true
}

func (n *LiteNode[T]) GetPrefix(idx uint8) (_ T, exists bool) {
	// no docstring by intention
	exists = n.Prefixes.Test(idx)
	return
}

func (n *LiteNode[T]) MustGetPrefix(idx uint8) (_ T) {
	// no docstring by intention
	return
}

// CloneFlat returns a shallow copy of the current node.
func (n *LiteNode[T]) CloneFlat(_ func(T) T) *LiteNode[T] {
	if n == nil {
		return nil
	}

	c := new(LiteNode[T])

	// copy simple values
	c.Prefixes = n.Prefixes

	// sparse array
	c.Children = *(n.Children.Copy())

	// no values to copy
	return c
}

// AggregateRec compresses the LiteNode in-place by pruning redundant subnets,
// removing child nodes covered by parent prefixes, recursively compressing child
// nodes with promotion of eligible single-entry children to FringeNode or LeafNode
// instances, and merging adjacent sibling prefixes or fringe nodes.
//
// The aggregation process executes the following steps in order:
//
//  1. Default Route Purge: If the node contains a default route (index 1), all
//     other prefixes and children in the subtree are pruned immediately.
//
//  2. Prefix Subsumption: Removes more-specific prefixes fully covered by a
//     broader supernet prefix within the same node's bitset.
//
//  3. Child Subsumption: Deletes child nodes that are fully covered
//     by an existing prefix in the current node.
//
//  4. Recursive Descent: Recursively calls AggregateRec on child node instances.
//     After the recursive call returns, if a child node has been compressed down to
//     a single entry (a prefix or a child node), it is promoted in-place in the
//     parent's child array:
//     - A single default prefix (index 1) is promoted to a FringeNode.
//     - Any other single prefix is promoted to a LeafNode with its reconstructed CIDR.
//     - A single child *LeafNode is promoted directly.
//     - A single child *FringeNode is reconstructed into a LeafNode and promoted.
//
//  5. Fringe Merging: Collapses pairs of adjacent FringeNode children into
//     a single supernet prefix inserted into the current node's bitset.
//
//  6. Prefix Merging: Repeatedly combines pairs of adjacent sibling prefixes
//     into their higher-level supernet prefix until no more merges are possible.
//
// Returns modified, the number of structural mutation operations performed
// during the aggregation pass. Note that pruning an entire child node counts
// as a single mutation event, regardless of how many nested prefixes it contained.
func (n *LiteNode[T]) AggregateRec(path StridePath, depth int, is4 bool) (modified int) {
	var zero T

	// #########################################################################################
	// 1. Default Route Purge: If node has default route, purge all prefixes and children.
	if n.Prefixes.Test(1) {
		*n = LiteNode[T]{}

		// Restore default route in this node
		n.InsertPrefix(1, zero)

		return modified + 1
	}

	// #########################################################################################
	// 2. Prefix Subsumption: Remove subnets in the bitset that are fully covered by a supernet.
	oldPfxCount := n.PrefixCount()
	var pfxIdx uint8
	var ok bool
	for {
		// Find the next set prefix index, starting search at bit 0
		if pfxIdx, ok = n.Prefixes.NextSet(pfxIdx); !ok {
			break
		}

		// The last prefix only overlaps with itself
		if pfxIdx == 255 {
			break
		}

		// Find all prefixes covered by pfxIdx using the allotment lookup table
		covered := n.Prefixes.And(&allot.PfxRoutesLookupTbl[pfxIdx])

		// Clear all covered prefixes, including pfxIdx itself
		n.Prefixes.BitSet256 = n.Prefixes.Xor(&covered)

		// Re-enable cleared pfxIdx
		n.Prefixes.Set(pfxIdx)

		// Advance index to search for the next prefix
		pfxIdx++
	}
	// Recalculate prefix count after deletions
	//nolint:gosec // G115: integer overflow conversion int -> uint16
	n.Prefixes.count = uint16(n.Prefixes.OnesCount())

	// Track number of subsumed prefixes removed
	modified += oldPfxCount - n.PrefixCount()

	// ###########################################################################
	// 3. Child Subsumption: Remove child nodes covered by any prefix in this node.
	//
	// We first accumulate all matching child addresses into a BitSet256 and delete
	// them in a second pass.
	var batchAddrs bitset.BitSet256
	for idx := range n.Prefixes.All() {
		// Collect child addresses covered by the current prefix using the fringe lookup table
		covered := n.Children.And(&allot.FringeRoutesLookupTbl[idx])
		batchAddrs = batchAddrs.Or(&covered)
	}

	// Batch delete accumulated child nodes
	oldChildCount := n.ChildCount()
	for addr := range batchAddrs.All() {
		n.DeleteChild(addr)
	}

	// Track total number of subsumed children removed
	modified += oldChildCount - n.ChildCount()

	// #########################################################
	// 4. Recursive Descent: Top-down compression of child nodes
	for i, addr := range n.Children.AllEnumerate() {
		anyKid := n.Children.Items[i]

		kid, ok := anyKid.(*LiteNode[T])
		// Leaf or Fringe, skip over
		if !ok {
			continue
		}

		// Recurse down
		path[depth] = addr
		modified += kid.AggregateRec(path, depth+1, is4)

		pfxCount := kid.PrefixCount()
		childCount := kid.ChildCount()

		// Nothing to promote if combined entry count is 2 or more
		if pfxCount+childCount >= 2 {
			continue
		}

		// Promote single-entry child nodes to lower-overhead structures
		switch {
		case pfxCount == 1:
			// Promote single prefix to FringeNode or LeafNode
			if kid.Prefixes.Test(1) {
				n.Children.Items[i] = NewFringeNode(zero)
			} else {
				// Convert prefix back to LeafNode and promote
				idx, _ := kid.Prefixes.FirstSet()
				leafPrefix := CidrFromPath(path[:], depth+1, is4, idx)
				n.Children.Items[i] = NewLeafNode(leafPrefix, zero)
			}

		case childCount == 1:
			// Promote single grandchild to parent's child slot
			switch grandKid := kid.Children.Items[0].(type) {
			case *LiteNode[T]:
				// Intermediate path node, leave as is
				continue

			case *LeafNode[T]:
				// Promote LeafNode directly
				n.Children.Items[i] = grandKid

			case *FringeNode[T]:
				// Convert FringeNode back to LeafNode and promote
				fringeByte, _ := kid.Children.FirstSet()
				fringePrefix := CidrForFringe(path[:], depth+1, is4, fringeByte)
				n.Children.Items[i] = NewLeafNode(fringePrefix, zero)
			}
		}
	}

	// #############################################################################
	// 5. Fringe Merging: Collapse adjacent FringeNode pairs into a supernet prefix.

	// Only aligned pairs are aggregation candidates
	alignedPairs := n.Children.AlignedPairs()
	for addr := range alignedPairs.All() {
		// addr, addr+1 is an aligned pair
		anyKid := n.MustGetChild(addr)
		if _, ok := anyKid.(*FringeNode[T]); !ok {
			continue
		}
		anyKid = n.MustGetChild(addr + 1)
		if _, ok := anyKid.(*FringeNode[T]); !ok {
			continue
		}

		// The aligned child pair are fringes; promote them as prefix: addr/7
		n.InsertPrefix(art.PfxToIdx(addr, 7), zero)
		n.DeleteChild(addr)
		n.DeleteChild(addr + 1)

		modified++
	}

	// #############################################################
	// 6. Prefix Merging: Merge adjacent prefixes within the bitset.
	for { // Repeat in multiple passes to handle cascading merges
		more := false

		alignedPairs := n.Prefixes.AlignedPairs()
		for idx := range alignedPairs.All() {
			// Insert supernet
			n.InsertPrefix(idx>>1, zero)

			// Delete subnets
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
