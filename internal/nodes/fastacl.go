// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package nodes

import (
	"fmt"
	"io"
	"iter"
	"net/netip"
	"slices"
	"strings"

	"github.com/gaissmai/bart/internal/allot"
	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/bitset"
	"github.com/gaissmai/bart/internal/lpm"
	"github.com/gaissmai/bart/internal/sparse"
)

// FastACLNode is based on [LiteNode], but it also uses a cache ([256]uint8)
// per node to speed up traversal of the multi-bit trie.
// Lookups become faster, but this requires more memory per prefix,
// and updates (insertions/deletions) also become slower due to the
// overhead of managing the cache.
// FastACLNode also uses a Fringes bitset. Fringes are not stored
// in the Children sparse.Array256, since Fringes don't carry a payload
// so this optimization for speed is possible TODO ...
type FastACLNode struct {
	Prefixes bitset.BitSet256
	Fringes  bitset.BitSet256

	// map addr to slice idx (aka rank)
	childRankCache [256]uint8
	Children       sparse.Array256[any]

	// prefixCount maintains the current number of set prefix bits
	// to avoid population counting in the hot path.
	prefixCount uint16
}

// PrefixCount returns the number of prefixes stored in this node.
// A dedicated prefixCount is tracked to avoid popcount on the hot-path.
func (n *FastACLNode) PrefixCount() int {
	return int(n.prefixCount)
}

// FringeCount returns the number of fringes stored in this node.
// No dedicated fringeCount is tracked. The fringe count is not needed on the hot path.
func (n *FastACLNode) FringeCount() int {
	return n.Fringes.OnesCount()
}

// IsEmpty returns true if the node contains no routing entries (prefixes),
// no fringes and no child nodes.
// Empty nodes are candidates for compression or removal during trie optimization.
func (n *FastACLNode) IsEmpty() bool {
	if n == nil {
		return true
	}
	return n.PrefixCount()+n.ChildCount()+n.FringeCount() == 0
}

// InsertPrefix adds a routing entry at the specified index.
// It returns true if a prefix already existed at that index,
// false if this is a new insertion.
func (n *FastACLNode) InsertPrefix(idx uint8) (exists bool) {
	if exists = n.Prefixes.Test(idx); exists {
		return exists
	}

	n.Prefixes.Set(idx)
	n.prefixCount++ // tracked

	return exists
}

// DeletePrefix removes the prefix at the specified index.
// Returns true if the prefix existed, and false otherwise.
func (n *FastACLNode) DeletePrefix(idx uint8) (exists bool) {
	if exists = n.Prefixes.Test(idx); !exists {
		return false
	}

	n.Prefixes.Clear(idx)
	n.prefixCount-- // tracked

	return true
}

// InsertFringe adds a fringe node at the specified address (0-255).
// Returns true if a fringe already existed at that address.
func (n *FastACLNode) InsertFringe(addr uint8) (exists bool) {
	if exists = n.Fringes.Test(addr); exists {
		return exists
	}
	n.Fringes.Set(addr)
	return
}

// DeleteFringe removes the fringe at the specified address.
// Returns true if the fringe existed, and false otherwise.
func (n *FastACLNode) DeleteFringe(idx uint8) (exists bool) {
	if exists = n.Fringes.Test(idx); !exists {
		return false
	}
	n.Fringes.Clear(idx)
	return true
}

// InsertChild adds a child node at the specified address (0-255).
// The child can be a *FastACLNode or *LeafNodeACL.
// Returns true if a child already existed at that address.
func (n *FastACLNode) InsertChild(addr uint8, child any) (exists bool) {
	var rank0 int
	rank0, exists = n.Children.InsertAt(addr, child)
	if exists {
		// Update only: the value at addr is overwritten in-place by InsertAt.
		// childRankCache[addr] is not refreshed because the rank is unchanged
		// the position of addr in the sparse slice does not shift on an update.
		return
	}

	// new child inserted? cache the rank value for this addr
	//nolint:gosec // G115: integer overflow conversion int -> uint8
	n.childRankCache[addr] = uint8(rank0)

	// increment all cached ranks after addr
	for i := 255; i > int(addr); i-- {
		n.childRankCache[i]++
	}
	return
}

// GetChild retrieves the child node at the specified address.
// Returns the child and true if found, or nil and false if not present.
func (n *FastACLNode) GetChild(addr uint8) (any, bool) {
	if n.Children.Test(addr) {
		rank0 := n.childRankCache[addr]
		return n.Children.Items[rank0], true
	}
	return nil, false
}

// MustGetChild retrieves the child at addr using the pre-cached rank stored in
// childRankCache[addr] for a direct O(1) array access without an existence check.
//
// The caller must guarantee that addr is present (Children.Test(addr) == true).
// If addr is absent, childRankCache[addr] contains the number of occupied
// addresses less than addr (maintained by InsertChild/DeleteChild), so the
// behaviour is undefined: either a wrong child is returned silently, or the
// call panics with an index-out-of-range error.
func (n *FastACLNode) MustGetChild(addr uint8) any {
	rank0 := n.childRankCache[addr]
	return n.Children.Items[rank0]
}

// DeleteChild removes the child node at the specified address.
// This operation is idempotent - removing a non-existent child is safe.
func (n *FastACLNode) DeleteChild(addr uint8) (exists bool) {
	_, exists = n.Children.DeleteAt(addr)

	// nothing deleted
	if !exists {
		return exists
	}

	// decrement all cached ranks after addr
	for i := 255; i > int(addr); i-- {
		n.childRankCache[i]--
	}

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
func (n *FastACLNode) LookupIdx(idx uint8) (top uint8, ok bool) {
	return n.Prefixes.AndTop(&lpm.LookupTbl[idx])
}

// Lookup is just a simple wrapper for LookupIdx.
func (n *FastACLNode) Lookup(idx uint8) (ok bool) {
	_, ok = n.LookupIdx(idx)
	return
}

// CloneFlat returns a shallow copy of the current node.
func (n *FastACLNode) CloneFlat() *FastACLNode {
	if n == nil {
		return nil
	}

	c := new(FastACLNode)

	// copy simple values
	c.Fringes = n.Fringes
	c.Prefixes = n.Prefixes
	c.prefixCount = n.prefixCount
	c.childRankCache = n.childRankCache

	// sparse array
	c.Children = *(n.Children.Copy())

	// no values to copy
	return c
}

// CloneRec performs a recursive deep copy of the node and all its descendants.
//
// Returns a new instance of FastACLNode which is a complete deep clone of the
// receiver node with all descendants.
func (n *FastACLNode) CloneRec() *FastACLNode {
	if n == nil {
		return nil
	}

	// Perform a flat clone of the current node.
	c := n.CloneFlat()

	// Recursively clone all child nodes of type *FastACLNode
	for i, kidAny := range c.Children.Items {
		if kid, ok := kidAny.(*FastACLNode); ok {
			c.Children.Items[i] = kid.CloneRec()
		}
	}

	return c
}

// AggregateRec compresses the FastACLNode in-place by pruning redundant subnets,
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
func (n *FastACLNode) AggregateRec(path StridePath, depth int, is4 bool) (modified int) {
	panic("TODO")

	// #########################################################################################
	// 1. Default Route Purge: If node has default route, purge all prefixes and children.
	if n.Prefixes.Test(1) {
		*n = FastACLNode{}

		// Restore default route in this node
		n.InsertPrefix(1)

		return modified + 1
	}

	// #########################################################################################
	// 2. Prefix Subsumption: Remove subnets in the bitset that are fully covered by a supernet.
	oldPfxCount := n.prefixCount
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
		n.Prefixes = n.Prefixes.Xor(&covered)

		// Re-enable cleared pfxIdx
		n.Prefixes.Set(pfxIdx)

		// Advance index to search for the next prefix
		pfxIdx++
	}
	// Recalculate prefix count after deletions
	//nolint:gosec // G115: integer overflow conversion int -> uint16
	n.prefixCount = uint16(n.Prefixes.OnesCount())

	// Track number of subsumed prefixes removed
	modified += int(oldPfxCount - n.prefixCount)

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

		kid, ok := anyKid.(*FastACLNode)
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
				n.Children.Items[i] = nil /* a fringe */
			} else {
				// Convert prefix back to LeafNode and promote
				idx, _ := kid.Prefixes.FirstSet()
				leafPrefix := CidrFromPath(path, depth+1, is4, idx)
				n.Children.Items[i] = &CIDRLeaf{leafPrefix}
			}

		case childCount == 1:
			// Promote single grandchild to parent's child slot
			switch grandKid := kid.Children.Items[0].(type) {
			case *FastACLNode:
				// Intermediate path node, leave as is
				continue

			case *CIDRLeaf:
				// Promote LeafNode directly
				n.Children.Items[i] = grandKid

			case *FringeLeaf:
				// Convert FringeNode back to LeafNode and promote
				fringeByte, _ := kid.Children.FirstSet()
				fringePrefix := CidrForFringe(path[:], depth+1, is4, fringeByte)
				n.Children.Items[i] = &CIDRLeaf{fringePrefix}
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
		if _, ok := anyKid.(*FringeLeaf); !ok {
			continue
		}
		anyKid = n.MustGetChild(addr + 1)
		if _, ok := anyKid.(*FringeLeaf); !ok {
			continue
		}

		// The aligned child pair are fringes; promote them as prefix: addr/7
		n.InsertPrefix(art.PfxToIdx(addr, 7))
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
			n.InsertPrefix(idx >> 1)

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

// Get retrieves the value associated with the given network prefix.
// Traversal descends through the trie using the prefix's octets.
//
// The lookup handles path compression transparently:
//   - If the path matches an internal node at the target stride, the value is retrieved
//     from the node's prefix table.
//   - If the path leads to a compressed LeafNode or FringeNode, the function verifies
//     the prefix match before returning the value.
//
// Parameters:
//   - pfx: The network prefix to look up (must be in canonical form).
//
// Returns:
//   - exists: True if the prefix was found, false otherwise.
func (n *FastACLNode) Get(pfx netip.Prefix) (exists bool) {
	panic("TODO")

	// The prefix must be provided in canonical (masked) form for correct trie traversal.
	ip := pfx.Addr()
	pfxLen := pfx.Bits()
	octets := ip.AsSlice()
	strideCount, modBits := DivMod8(pfxLen)

	// Traverse the trie octet by octet based on the prefix path.
	for depth, octet := range octets {
		// At the target stride boundary, the prefix is expected in this node's
		// prefix table.
		if depth == strideCount {
			return n.Prefixes.Test(art.PfxToIdx(octet, modBits))
		}

		// If no child exists at this path, the prefix is not in the trie.
		kidAny, ok := n.GetChild(octet)
		if !ok {
			return false
		}

		// Identify the node type at this path segment.
		switch kid := kidAny.(type) {
		case *FastACLNode:
			// Standard intermediate node: descend to the next level.
			n = kid

		case *FringeLeaf:
			// Reached a path-compressed FringeNode.
			// Verify if the prefix qualifies as a fringe at this depth to return a match.
			if IsFringe(depth, pfxLen) {
				return true
			}
			return false

		case *CIDRLeaf:
			// Reached a path-compressed LeafNode.
			// Check if the stored prefix matches the lookup prefix exactly.
			if kid.Prefix == pfx {
				return true
			}
			return false

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
}

// #################################################################
// TODO: generate some methods for fastacl
// #################################################################

// ChildCount returns the number of slots used in this node.
func (n *FastACLNode) ChildCount() int {
	return n.Children.Len()
}

// AllChildren returns an iterator over all child nodes.
// Each iteration yields the child's address (uint8) and the child node (any).
func (n *FastACLNode) AllChildren() iter.Seq2[uint8, any] {
	return func(yield func(addr uint8, child any) bool) {
		for i, addr := range n.Children.AllEnumerate() {
			if !yield(addr, n.Children.Items[i]) {
				return
			}
		}
	}
}

// Insert adds or updates a network prefix and its associated value in the trie.
// Traversal begins at the specified byte depth.
//
// The trie utilizes path compression to conserve memory. A prefix is inserted:
//   - Uncompressed into a node's prefix table at depth == strideCount.
//   - As a path-compressed FringeNode if it qualifies as fringe [IsFringe].
//   - As a path-compressed LeafNode otherwise.
//
// When a new prefix collides with an existing compressed node (Leaf or Fringe),
// Insert resolves the collision by creating a new intermediate node, pushing
// the existing entry down to the next level, and continuing traversal.
//
// Parameters:
//   - pfx: The network prefix to insert (must be in canonical/masked form).
//   - val: The value to associate with the prefix.
//   - depth: The current depth in the trie (0-based byte index).
//
// Returns true if an existing prefix was updated, false if a new insertion occurred.
func (n *FastACLNode) Insert(pfx netip.Prefix, depth int) (exists bool) {
	ip := pfx.Addr() // the pfx must be in canonical form
	pfxLen := pfx.Bits()
	octets := ip.AsSlice()
	strideCount, modBits := DivMod8(pfxLen)

	// Traverse the prefix's octets. Each depth corresponds to an 8-bit stride.
	// We descend through the trie until we either reach the final stride (depth == strideCount)
	// or find an empty child slot where we can path-compress the remaining strides.
	for ; depth < len(octets); depth++ {
		octet := octets[depth]

		// The current depth matches the prefix's stride count, meaning this is the final
		// node for this prefix. We insert it directly into this node's prefix table.
		if depth == strideCount {
			return n.InsertPrefix(art.PfxToIdx(octet, modBits))
		}

		// If the prefix is perfectly aligned with the next stride boundary (e.g., /16 at depth 1),
		// it acts as a default route for everything below it. We store it as a FringeNode.
		if IsFringe(depth, pfxLen) {
			return n.InsertFringe(octet)
		}

		// No child exists at this octet path.
		// We path-compress the rest of the prefix as leaf into the child slot at octet.
		if !n.Children.Test(octet) {
			return n.InsertChild(octet, &CIDRLeaf{pfx})
		}

		// A child already exists at this octet path. Retrieve it to either continue
		// our descent along the strides or resolve a structural collision with a compressed node.
		kid := n.MustGetChild(octet)

		switch kid := kid.(type) {
		case *FastACLNode:
			// Standard intermediate node: descend to the next trie level.
			n = kid

		case *CIDRLeaf:
			// Collision with an existing path-compressed LeafNode.
			if kid.Prefix == pfx {
				return true
			}

			// Collision resolution: the paths diverge.
			// 1. Create a new intermediate node.
			// 2. Push the existing leaf down into this new node.
			// 3. Replace the current child slot with the new node.
			// 4. Descend into the new node to continue inserting 'pfx'.
			newNode := new(FastACLNode)
			newNode.Insert(kid.Prefix, depth+1)

			n.InsertChild(octet, newNode)
			n = newNode

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
}

// PurgeAndCompress performs bottom-up trie maintenance to restore path compression
// after a deletion. It unwinds the provided stack of parent nodes, identifying
// nodes that have become sparse (i.e., containing only a single prefix or fringe
// or a single child node) and prunes them by promoting the underlying entries to
// the parent level.
//
// This ensures the trie remains memory-efficient by collapsing redundant intermediate
// nodes back into path-compressed CIDRLeafs whenever possible.
//
// Parameters:
//   - stack: Array of parent nodes to process during bottom-up unwinding.
//   - octets: The full path of octets leading to the current node.
//   - is4: True for IPv4 processing, false for IPv6.
func (n *FastACLNode) PurgeAndCompress(stack []*FastACLNode, octets []uint8, is4 bool) {
	// Iterate backwards through the ancestor stack to prune nodes from the bottom up.
	for depth := len(stack) - 1; depth >= 0; depth-- {
		parent := stack[depth]
		octet := octets[depth]

		// Check if the current node is redundant.
		// A node may be redundant if it contains exactly one entry (either a prefix or
		// or a fringe or a compressed CIDRLeaf).
		pfxCount := n.PrefixCount()
		fringeCount := n.FringeCount()
		childCount := n.ChildCount()

		// If it contains more than one entry, it is always structurally significant and cannot be pruned.
		if pfxCount+fringeCount+childCount > 1 {
			return
		}

		switch {
		case childCount == 1:

			// The node has exactly one child.
			anyKid := n.Children.Items[0]

			// If the child is an intermediate path node; the tree structure is required
			// at this level. Compression cannot proceed further up.
			if _, ok := anyKid.(*FastACLNode); ok {
				return
			}

			// The child must be a compressed LeafNode. Prune the current node
			// and re-insert the leaf into the parent to elevate it.
			leaf := anyKid.(*CIDRLeaf)
			parent.DeleteChild(octet)

			parent.Insert(leaf.Prefix, depth)

		case fringeCount == 1:
			// The node has exactly one fringe. Prune the node and elevate the
			// fringe to the parent level as a leaf.
			parent.DeleteChild(octet)

			// Reconstruct the full prefix for the fringe, as path compression
			// requires the entire CIDR path, not just the remainder.
			// depth is the parent's depth, so we offset by 1 for the kid's position.
			singleAddr, _ := n.Fringes.FirstSet()
			fringePfx := CidrForFringe(octets, depth+1, is4, singleAddr)

			parent.Insert(fringePfx, depth)

		case pfxCount == 1:
			// The node has exactly one prefix. Prune the node and elevate the
			// prefix to the parent level as a leaf/fringe.
			parent.DeleteChild(octet)

			// Retrieve the single prefix stored in this node.
			idx, _ := n.Prefixes.FirstSet()

			// Reconstruct the prefix from the path for re-insertion.
			path := StridePath{}
			copy(path[:], octets)
			pfx := CidrFromPath(path, depth+1, is4, idx)

			parent.Insert(pfx, depth)
		default:
			panic("unreachable")
		}

		// Move up to the next parent in the stack to continue pruning.
		n = parent
	}
}

// Delete removes the prefix from the trie rooted at n and returns true if the
// prefix existed, false if it was not found. The prefix must be in canonical
// (masked) form.
//
// The trie uses path compression, so a prefix may be stored in one of three ways:
//   - In the current node's prefix bitset when the prefix length aligns exactly
//     with the stride boundary at this depth (depth == strideCount).
//   - In the current node's fringe bistet for stride-aligned prefixes
//     (e.g. /8, /16, /24) at depth == strideCount-1.
//   - As a path-compressed CIDRLeaf in a child slot for prefixes otherwise.
//
// After a successful deletion, PurgeAndCompress walks the ancestor stack to prune
// now-empty nodes and restore path compression upward.
func (n *FastACLNode) Delete(pfx netip.Prefix) (exists bool) {
	ip := pfx.Addr() // pfx must be in canonical (masked) form
	pfxLen := pfx.Bits()
	is4 := ip.Is4()
	octets := ip.AsSlice()
	strideCount, modBits := DivMod8(pfxLen)

	// Record ancestor nodes as we descend; PurgeAndCompress uses this stack to
	// walk back up and clean up empty or re-compressible nodes after deletion.
	stack := [MaxTreeDepth]*FastACLNode{}

	for depth, octet := range octets {
		depth &= DepthMask // BCE hint; keep Delete on the fast path

		stack[depth] = n // record current node before descending

		// At the stride boundary, the prefix is stored directly in this node's
		// prefix table.
		if depth == strideCount {
			if exists = n.DeletePrefix(art.PfxToIdx(octet, modBits)); !exists {
				return false
			}

			// prune now-empty nodes and re-compress the path upwards
			n.PurgeAndCompress(stack[:depth], octets, is4)
			return true
		}

		// If the prefix is perfectly aligned with the next stride boundary (e.g., /16 at depth 1),
		// it acts as a default route for everything below it. We store it as a FringeNode.
		if IsFringe(depth, pfxLen) {
			if exists := n.DeleteFringe(octet); !exists {
				return false
			}

			// prune now-empty nodes and re-compress the path upwards
			n.PurgeAndCompress(stack[:depth], octets, is4)
			return true
		}

		// no child node exists at this octet; the prefix is not in the trie
		if !n.Children.Test(octet) {
			return false
		}

		// A child node exists at this octet path, retrieve it.
		kid := n.MustGetChild(octet)

		switch kid := kid.(type) {
		case *FastACLNode:
			n = kid // descend to the next trie level

		case *CIDRLeaf:
			// A LeafNode holds exactly one path-compressed prefix.
			// Compare using the canonical (masked) form for an exact match.
			if kid.Prefix != pfx {
				return false
			}

			n.DeleteChild(octet)

			// prune now-empty nodes and re-compress the path upwards
			n.PurgeAndCompress(stack[:depth], octets, is4)
			return true

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
}

// Contains returns true if an index (idx) has any matching longest-prefix
// in the current node’s prefix table.
//
// This function performs a presence check without retrieving the associated value.
// It is faster than a full lookup, as it only tests for intersection with the
// backtracking bitset for the given index.
//
// The prefix table is structured as a complete binary tree (CBT), and LPM testing
// is done via a bitset operation that maps the traversal path from the given index
// toward its possible ancestors.
func (n *FastACLNode) Contains(idx uint8) bool {
	return n.Prefixes.Overlaps(&lpm.LookupTbl[idx])
}

// EqualRec performs recursive structural equality comparison between two nodes.
// Compares prefix and child bitsets, then recursively compares all
// child nodes. Returns true if the nodes and their entire subtrees are
// structurally and semantically identical, false otherwise.
//
// The comparison handles different node types (internal nodes, leafNodes, fringeNodes).
func (n *FastACLNode) EqualRec(o *FastACLNode) bool {
	if n == nil || o == nil {
		return n == o
	}
	if n == o {
		return true
	}

	if n.Prefixes != o.Prefixes {
		return false
	}

	if n.Fringes != o.Fringes {
		return false
	}

	if n.Children.BitSet256 != o.Children.BitSet256 {
		return false
	}

	for addr, nKid := range n.AllChildren() {
		oKid := o.MustGetChild(addr) // MustGet is ok, bitsets are equal

		switch nKid := nKid.(type) {
		case *FastACLNode:
			// oKid must also be a node
			oKid, ok := oKid.(*FastACLNode)
			if !ok {
				return false
			}

			// compare rec-descent
			if !nKid.EqualRec(oKid) {
				return false
			}

		case *CIDRLeaf:
			// oKid must also be a leaf
			oKid, ok := oKid.(*CIDRLeaf)
			if !ok {
				return false
			}

			// compare prefixes
			if nKid.Prefix != oKid.Prefix {
				return false
			}

		default:
			panic("logic error, wrong node type")
		}
	}

	return true
}

// DumpRec recursively descends the trie rooted at n and writes a human-readable
// representation of each visited node to w.
//
// It returns immediately if n is nil or empty. For each visited internal node
// it calls dump to write the node's representation, then iterates its child
// addresses and recurses into children of type *FastACLNode (internal subnodes).
// The path slice and depth together represent the byte-wise path
// from the root to the current node; depth is incremented for each recursion.
// The is4 flag controls IPv4/IPv6 formatting used by dump.
func (n *FastACLNode) DumpRec(w io.Writer, path StridePath, depth int, is4 bool) {
	if n == nil || n.IsEmpty() {
		return
	}

	// dump this node
	n.dump(w, path, depth, is4)

	// node may have children, rec-descent down
	for addr, child := range n.AllChildren() {
		if kid, ok := child.(*FastACLNode); ok {
			path[depth] = addr
			kid.DumpRec(w, path, depth+1, is4)
		}
	}
}

// dump writes a human-readable representation of the node to `w`.
// It prints the node type, depth, formatted path (IPv4 vs IPv6 controlled by `is4`),
// and bit count, followed by any stored prefixes (and their values when applicable),
// the set of child octets, and any path-compressed leaves or fringe entries.
func (n *FastACLNode) dump(w io.Writer, path StridePath, depth int, is4 bool) {
	bits := depth * strideLen
	indent := strings.Repeat(".", depth)

	// node type with depth and octet path and bits.
	fmt.Fprintf(w, "\n%s[%s] depth:  %d path: [%s] / %d\n",
		indent, n.hasType(), depth, ipStridePath(path, depth, is4), bits)

	// format width for %*d
	width := Len256(max(n.PrefixCount(), n.FringeCount(), n.ChildCount()))

	// print the prefixes
	if n.PrefixCount() != 0 {
		fmt.Fprintf(w, "%sprefix(#%*d):", indent, width, n.PrefixCount())

		for idx := range n.Prefixes.All() {
			pfx := CidrFromPath(path, depth, is4, idx)
			fmt.Fprintf(w, " %d:{%s}", idx, pfx)
		}

		fmt.Fprintln(w)
	}

	// print the fringes
	if n.FringeCount() != 0 {
		fmt.Fprintf(w, "%sfringe(#%*d):", indent, width, n.FringeCount())

		for addr := range n.Fringes.All() {
			fringePfx := CidrForFringe(path[:], depth, is4, addr)
			fmt.Fprintf(w, " %s:{%s}", addrFmt(addr, is4), fringePfx)
		}

		fmt.Fprintln(w)
	}

	nodeAddrs := make([]uint8, 0, n.ChildCount())
	leafAddrs := make([]uint8, 0, n.ChildCount())

	// the node has recursive child nodes or path-compressed leaves
	for addr, child := range n.AllChildren() {
		switch child.(type) {
		case *FastACLNode:
			nodeAddrs = append(nodeAddrs, addr)
			continue

		case *CIDRLeaf:
			leafAddrs = append(leafAddrs, addr)

		default:
			panic("logic error, wrong node type")
		}
	}

	// print the leafs
	if len(leafAddrs) > 0 {
		fmt.Fprintf(w, "%s  leaf(#%*d):", indent, width, len(leafAddrs))

		for _, addr := range leafAddrs {
			leaf := n.MustGetChild(addr).(*CIDRLeaf)
			fmt.Fprintf(w, " %s:{%s}", addrFmt(addr, is4), leaf.Prefix)
		}

		fmt.Fprintln(w)
	}

	// print the nodes
	if len(nodeAddrs) > 0 {
		fmt.Fprintf(w, "%s  node(#%*d):", indent, width, len(nodeAddrs))

		for _, addr := range nodeAddrs {
			fmt.Fprintf(w, " %s", addrFmt(addr, is4))
		}

		fmt.Fprintln(w)
	}
}

// DumpString traverses the trie to the node at the specified depth along the given
// octet path and returns its string representation via Dump.
//
// If the path is invalid or encounters an unexpected node type during traversal,
// it returns an error message string instead.
//
// Parameters:
//   - octets: The path of octets to follow from the root
//   - depth: Target depth to reach before dumping (0-based byte index)
//   - is4: True for IPv4 formatting, false for IPv6
//
// Returns a formatted string representation of the target node or an error message.
func (n *FastACLNode) DumpString(octets []uint8, depth int, is4 bool) string {
	panic("TODO")

	path := StridePath{}
	copy(path[:], octets)

	buf := new(strings.Builder)
	for i := range depth {
		anyKid, ok := n.GetChild(path[i])
		if !ok {
			return fmt.Sprintf("ERROR: kid for %v[%d] is NOT set in node\n", octets, i)
		}

		kid, ok := anyKid.(*FastACLNode)
		if !ok {
			return fmt.Sprintf("ERROR: kid for %v[%d] is NO %s\n", octets, i, "FastACLNode")
		}

		// traverse
		n = kid
	}

	n.dump(buf, path, depth, is4)
	return buf.String()
}

// hasType classifies the given node into one of the nodeType values.
//
// It inspects immediate statistics (prefix count, child count, node, leaf and
// fringe counts) for the node and returns:
//   - nullNode: no prefixes and no children
//   - stopNode: has children but no subnodes (nodes == 0)
//   - halfNode: contains at least one leaf or fringe and also has subnodes, but
//     no prefixes
//   - fullNode: has prefixes and also has subnodes
//   - pathNode: has subnodes only (no prefixes, leaves, or fringes)
//
// The order of these checks is significant to ensure the correct classification.
func (n *FastACLNode) hasType() nodeType {
	if n.IsEmpty() {
		return nullNode
	}

	s := n.Stats()

	// the order is important
	switch {
	case s.SubNodes == 0:
		return stopNode
	case (s.Leaves > 0 || s.Fringes > 0) && s.SubNodes > 0 && s.Prefixes == 0 && s.Fringes == 0:
		return halfNode
	case (s.Prefixes > 0 || s.Leaves > 0 || s.Fringes > 0) && s.SubNodes > 0:
		return fullNode
	case (s.Prefixes == 0 && s.Leaves == 0 && s.Fringes == 0) && s.SubNodes > 0:
		return pathNode
	default:
		panic(fmt.Sprintf("UNREACHABLE: pfx: %d, fringe: %d, chld: %d, node: %d, leaf: %d",
			s.Prefixes, s.Fringes, s.Children, s.SubNodes, s.Leaves))
	}
}

// Stats returns immediate statistics for n: counts of prefixes and children,
// and a classification of each child into nodes, leaves, or fringes.
// It inspects only the direct children of n (not the whole subtree).
// Panics if a child has an unexpected concrete type.
func (n *FastACLNode) Stats() StatsT {
	stats := StatsT{}
	stats.Prefixes = n.PrefixCount()
	stats.Children = n.ChildCount()
	stats.Fringes = n.FringeCount()

	for _, child := range n.AllChildren() {
		switch child.(type) {
		case *FastACLNode:
			stats.SubNodes++

		case *CIDRLeaf:
			stats.Leaves++

		default:
			panic("logic error, wrong node type")
		}
	}

	return stats
}

// StatsRec returns aggregated statistics for the subtree rooted at n.
//
// It walks the node tree recursively and sums immediate counts (prefixes and
// child slots) plus the number of nodes, leaves, and fringe nodes in the
// subtree. If n is nil or empty, a zeroed stats is returned. The returned
// SubNodes count includes the current node. The function will panic if a child
// has an unexpected concrete type.
func (n *FastACLNode) StatsRec() (s StatsT) {
	if n == nil || n.IsEmpty() {
		return s
	}

	s.Prefixes = n.PrefixCount()
	s.Fringes = n.FringeCount()
	s.Children = n.ChildCount()
	s.SubNodes = 1 // this node
	s.Leaves = 0

	for _, child := range n.AllChildren() {
		switch kid := child.(type) {
		case *FastACLNode:
			// rec-descent
			rs := kid.StatsRec()

			s.Prefixes += rs.Prefixes
			s.Fringes += rs.Fringes
			s.Children += rs.Children
			s.SubNodes += rs.SubNodes
			s.Leaves += rs.Leaves

		case *CIDRLeaf:
			s.Leaves++

		default:
			panic("logic error, wrong node type")
		}
	}

	return s
}

// TrieItemACL, TODO
type TrieItemACL struct {
	// for traversing, Path/Depth/Idx is needed to get the CIDR back from the trie.
	Node  any // BartNode, FastNode, LiteNode
	Is4   bool
	Path  StridePath
	Depth int
	Idx   uint8

	// for printing
	Cidr netip.Prefix
}

// FprintRec recursively prints a hierarchical CIDR tree representation
// starting from this node to the provided writer. The output shows the
// routing table structure in human-readable format for debugging and analysis.
func (n *FastACLNode) FprintRec(w io.Writer, parent TrieItemACL, pad string) error {
	panic("TODO")

	// recursion stop condition
	if n == nil || n.IsEmpty() {
		return nil
	}

	// get direct covered childs for this parent ...
	directItems := n.DirectItemsRec(parent.Idx, parent.Path, parent.Depth, parent.Is4)

	// sort them by netip.Prefix, not by baseIndex
	slices.SortFunc(directItems, func(a, b TrieItemACL) int {
		return CmpPrefix(a.Cidr, b.Cidr)
	})

	// for all direct item under this node ...
	for i, item := range directItems {
		// symbols used in tree
		glyph := "├─ "
		space := "│  "

		// ... treat last kid special
		if i == len(directItems)-1 {
			glyph = "└─ "
			space = "   "
		}

		var err error
		_, err = fmt.Fprintf(w, "%s%s\n", pad+glyph, item.Cidr)

		if err != nil {
			return err
		}

		// rec-descent with this item as parent
		nextNode, _ := item.Node.(*FastACLNode)
		if err = nextNode.FprintRec(w, item, pad+space); err != nil {
			return err
		}
	}

	return nil
}

// DirectItemsRec, returns the direct covered items by parent.
// It's a complex recursive function, you have to know the data structure
// by heart to understand this function!
func (n *FastACLNode) DirectItemsRec(parentIdx uint8, path StridePath, depth int, is4 bool) (directItems []TrieItemACL) {
	panic("TODO")

	// recursion stop condition
	if n == nil || n.IsEmpty() {
		return nil
	}

	// prefixes:
	// for all idx's (prefixes mapped by baseIndex) in this node
	// do a longest-prefix-match
	for idx := range n.Prefixes.All() {
		// tricky part, skip self
		// test with next possible lpm (idx>>1), it's a complete binary tree
		nextIdx := idx >> 1

		// fast skip, lpm not possible
		if nextIdx < parentIdx {
			continue
		}

		// do a longest-prefix-match
		lpm, _ := n.LookupIdx(nextIdx)

		// be aware, 0 is here a possible value for parentIdx and lpm (if not found)
		if lpm == parentIdx {
			// prefix is directly covered by parent

			item := TrieItemACL{
				Node:  n,
				Is4:   is4,
				Path:  path,
				Depth: depth,
				Idx:   idx,
				// get the prefix back from trie
				Cidr: CidrFromPath(path, depth, is4, idx),
			}

			directItems = append(directItems, item)
		}
	}

	// children:
	for addr, child := range n.AllChildren() {
		hostIdx := art.OctetToIdx(addr)

		// do a longest-prefix-match
		lpm, _ := n.LookupIdx(hostIdx)

		// be aware, 0 is here a possible value for parentIdx and lpm (if not found)
		if lpm == parentIdx {
			// child is directly covered by parent
			switch kid := child.(type) {
			case *FastACLNode: // traverse rec-descent, call with next child node,
				// next trie level, set parentIdx to 0, adjust path and depth
				path[depth] = addr
				directItems = append(directItems, kid.DirectItemsRec(0, path, depth+1, is4)...)

			case *CIDRLeaf: // path-compressed child, stops recursion for this child
				item := TrieItemACL{
					Node: nil,
					Is4:  is4,
					Cidr: kid.Prefix,
				}
				directItems = append(directItems, item)

			case *FringeLeaf: // path-compressed fringe, stops recursion for this child
				item := TrieItemACL{
					Node: nil,
					Is4:  is4,
					// get the prefix back from trie
					Cidr: CidrForFringe(path[:], depth, is4, addr),
				}
				directItems = append(directItems, item)

			default:
				panic("logic error, wrong node type")
			}
		}
	}

	return directItems
}

// AllRec recursively traverses the trie starting at the current node,
// applying the provided yield function to every stored prefix and value.
//
// For each route entry (prefix and value), yield is invoked. If yield returns false,
// the traversal stops immediately, and false is propagated upwards,
// enabling early termination.
//
// The function handles all prefix entries in the current node, as well as any children -
// including sub-nodes, leaf nodes with full prefixes, and fringe nodes
// representing path-compressed prefixes. IP prefix reconstruction is performed on-the-fly
// from the current path and depth.
//
// The traversal order is not defined. This implementation favors simplicity
// and runtime efficiency over consistency of iteration sequence.
func (n *FastACLNode) AllRec(path StridePath, depth int, is4 bool, yield func(netip.Prefix) bool) bool {
	panic("TODO")

	for idx := range n.Prefixes.All() {
		cidr := CidrFromPath(path, depth, is4, idx)

		// callback for this prefix and val
		if !yield(cidr) {
			// early exit
			return false
		}
	}

	// for all children (nodes and leaves) in this node do ...
	i := 0
	for addr := range n.Children.All() {
		anyKid := n.Children.Items[i]
		switch kid := anyKid.(type) {
		case *FastACLNode:
			// rec-descent with this node
			path[depth] = addr
			if !kid.AllRec(path, depth+1, is4, yield) {
				// early exit
				return false
			}
		case *CIDRLeaf:
			// callback for this leaf
			if !yield(kid.Prefix) {
				// early exit
				return false
			}
		case *FringeLeaf:
			fringePfx := CidrForFringe(path[:], depth, is4, addr)
			// callback for this fringe
			if !yield(fringePfx) {
				// early exit
				return false
			}

		default:
			panic("logic error, wrong node type")
		}

		i++
	}

	return true
}

// AllRecSorted recursively traverses the trie in prefix-sorted order and applies
// the given yield function to each stored prefix and value.
//
// Unlike AllRec, this implementation ensures that route entries are visited in
// canonical prefix sort order. To achieve this,
// both the prefixes and children of the current node are gathered, sorted,
// and then interleaved during traversal based on logical octet positioning.
//
// The function first sorts relevant entries by their prefix index and address value,
// using a comparison function that ranks prefixes according to their mask length and position.
// Then it walks the trie, always yielding child entries that fall before the current prefix,
// followed by the prefix itself. Remaining children are processed once all prefixes have been visited.
//
// Prefixes are reconstructed on-the-fly from the traversal path, and iteration includes all child types:
// inner nodes (recursive descent), leaf nodes, and fringe (compressed) prefixes.
//
// The order is stable and predictable, making the function suitable for use cases
// like table exports, comparisons or serialization.
//
// Parameters:
//   - path: the current traversal path through the trie
//   - depth: current depth in the trie (0-based)
//   - is4: true for IPv4 processing, false for IPv6
//   - yield: callback function invoked for each prefix/value pair
//
// Returns false if yield function requests early termination.
func (n *FastACLNode) AllRecSorted(path StridePath, depth int, is4 bool, yield func(netip.Prefix) bool) bool {
	panic("TODO")

	allIndices := n.Prefixes.AppendBits(make([]uint8, 0, n.PrefixCount()))
	// allFringeAddrs := n.Fringes.AppendBits(make([]uint8, 0, n.FringeCount()))
	allChildAddrs := n.Children.AppendBits(make([]uint8, 0, n.ChildCount()))

	// Sort local prefix indices into canonical CIDR rank order.
	slices.SortFunc(allIndices, CmpIndexRank)

	// Helper to process and yield any child node type (inner node, leaf, or fringe).
	yieldChild := func(addr uint8) bool {
		switch kid := n.MustGetChild(addr).(type) {
		case *FastACLNode:
			path[depth] = addr
			return kid.AllRecSorted(path, depth+1, is4, yield)

		case *CIDRLeaf:
			return yield(kid.Prefix)

		case *FringeLeaf:
			fringePfx := CidrForFringe(path[:], depth, is4, addr)
			return yield(fringePfx)

		default:
			panic("logic error: unknown child node type")
		}
	}

	childCursor := 0

	// Interleave local prefixes and child subtrees in CIDR rank order.
	for _, pfxIdx := range allIndices {
		pfxOctet, _ := art.IdxToPfx(pfxIdx)

		// Yield all child subtrees whose base address precedes the current prefix octet.
		for childCursor < len(allChildAddrs) {
			childAddr := allChildAddrs[childCursor]
			if childAddr >= pfxOctet {
				break
			}

			if !yieldChild(childAddr) {
				return false
			}
			childCursor++
		}

		// Yield the local prefix for this index.
		cidr := CidrFromPath(path, depth, is4, pfxIdx)
		if !yield(cidr) {
			return false
		}
	}

	// Yield remaining child subtrees strictly positioned after all local prefixes.
	for _, addr := range allChildAddrs[childCursor:] {
		if !yieldChild(addr) {
			return false
		}
	}

	return true
}

// EachLookupPrefix performs a hierarchical lookup of all matching prefixes
// in the current node’s 8-bit stride-based prefix table.
//
// The function walks up the trie-internal complete binary tree (CBT),
// testing each possible prefix length mask (in decreasing order of specificity),
// and invokes the yield function for every matching entry.
//
// The given idx refers to the position for this stride's prefix and is used
// to derive a backtracking path through the CBT by repeatedly halving the index.
// At each step, if a prefix exists in the table, its corresponding CIDR is
// reconstructed and yielded. If yield returns false, traversal stops early.
//
// This function is intended for internal use during supernet traversal and
// does not descend the trie further.
func (n *FastACLNode) EachLookupPrefix(ip netip.Addr, depth int, pfxIdx uint8, yield func(netip.Prefix) bool) (ok bool) {
	panic("TODO")

	for ; pfxIdx > 0; pfxIdx >>= 1 {
		if n.Prefixes.Test(pfxIdx) {
			// get the CIDR back
			_, pfxLen := art.IdxToPfx(pfxIdx)
			cidr, _ := ip.Prefix(depth<<3 + int(pfxLen))

			if !yield(cidr) {
				return false
			}
		}
	}

	return true
}

// EachSubnet yields all routes and subtrees covered by pfxIdx within the current node
// in canonical CIDR sort order.
//
// It intersects the node's prefixes and child subtrees with precomputed lookup
// tables for pfxIdx. Covered prefixes are sorted by rank, and child subtrees are merged
// interleaved before and after prefixes based on their byte boundaries (pfxOctet).
//
// Subtrees are traversed recursively using AllRecSorted
// to guarantee deterministic ordering across stride boundaries.
//
// Expects the node to be at the path location specified by octets/depth.
func (n *FastACLNode) EachSubnet(octets []byte, depth int, is4 bool, pfxIdx uint8, yield func(netip.Prefix) bool) bool {
	panic("TODO")

	// octets as array, needed below more than once
	var path StridePath
	copy(path[:], octets)

	var tmp bitset.BitSet256

	// bitset & node entries against precomputed allot tables for pfxIdx.
	tmp = n.Prefixes.And(&allot.PfxRoutesLookupTbl[pfxIdx])
	allCoveredIndices := tmp.AppendBits(make([]uint8, 0, tmp.OnesCount()))

	tmp = n.Children.And(&allot.FringeRoutesLookupTbl[pfxIdx])
	allCoveredChildAddrs := tmp.AppendBits(make([]uint8, 0, tmp.OnesCount()))

	// Sort covered prefix indices into canonical CIDR order.
	slices.SortFunc(allCoveredIndices, CmpIndexRank)

	// Helper to process and yield child entries (nodes, leaves, or fringes).
	yieldChild := func(addr uint8) bool {
		switch kid := n.MustGetChild(addr).(type) {
		case *FastACLNode:
			path[depth] = addr
			return kid.AllRecSorted(path, depth+1, is4, yield)

		case *CIDRLeaf:
			return yield(kid.Prefix)

		case *FringeLeaf:
			fringePfx := CidrForFringe(path[:], depth, is4, addr)
			return yield(fringePfx)

		default:
			panic("logic error: unknown child node type")
		}
	}

	addrCursor := 0

	// Interleave local prefixes and child subtrees in CIDR rank order.
	for _, pfxIdx := range allCoveredIndices {
		pfxOctet, _ := art.IdxToPfx(pfxIdx)

		// Yield all child subtrees whose base address falls before the current prefix scope.
		for j := addrCursor; j < len(allCoveredChildAddrs); j++ {
			addr := allCoveredChildAddrs[j]
			if addr >= pfxOctet {
				break
			}

			if !yieldChild(addr) {
				return false
			}
			addrCursor++
		}

		// Yield the local prefix entry itself.
		cidr := CidrFromPath(path, depth, is4, pfxIdx)
		if !yield(cidr) {
			return false
		}
	}

	// Yield remaining child subtrees strictly after all local prefixes.
	for _, addr := range allCoveredChildAddrs[addrCursor:] {
		if !yieldChild(addr) {
			return false
		}
	}

	return true
}

// Supernets yields all supernet prefixes of pfx that exist in the trie,
// in reverse order (most-specific first, least-specific last).
//
// It traverses upward from the given prefix toward the root, collecting
// matching prefixes along the path. The traversal uses a stack to yield
// results in reverse order, so that more-specific supernets appear before
// less-specific ones.
//
// The function handles all node types (internal nodes, leaves, and fringes)
// and stops early if the yield callback returns false.
//
// Parameters:
//   - pfx: The prefix for which to find supernets
//   - yield: Callback function invoked for each supernet prefix/value pair
//
// The yield function receives prefix/value pairs and returns false to stop
// the iteration early.
func (n *FastACLNode) Supernets(pfx netip.Prefix, yield func(netip.Prefix) bool) {
	panic("TODO")

	ip := pfx.Addr()
	pfxLen := pfx.Bits()
	is4 := ip.Is4()
	octets := ip.AsSlice()
	strideCount, modBits := DivMod8(pfxLen)

	// stack of the traversed nodes for reverse ordering of supernets
	stack := [MaxTreeDepth]*FastACLNode{}

	// run variable, used after for loop
	var depth int
	var octet byte

	// find last node along this octet path
LOOP:
	for depth, octet = range octets {
		// stepped one past the last stride of interest; back up to last and exit
		if depth > strideCount {
			depth--
			break
		}
		// push current node on stack
		stack[depth] = n

		// descend down the trie
		if !n.Children.Test(octet) {
			break LOOP
		}
		kid := n.MustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *FastACLNode:
			n = kid
			continue LOOP // descend down to next trie level

		case *CIDRLeaf:
			if kid.Prefix.Bits() > pfx.Bits() {
				break LOOP
			}

			if kid.Prefix.Overlaps(pfx) {
				if !yield(kid.Prefix) {
					// early exit
					return
				}
			}
			// end of trie along this octets path
			break LOOP

		case *FringeLeaf:
			fringePfx := CidrForFringe(octets, depth, is4, octet)
			if fringePfx.Bits() > pfx.Bits() {
				break LOOP
			}

			if fringePfx.Overlaps(pfx) {
				if !yield(fringePfx) {
					// early exit
					return
				}
			}
			// end of trie along this octets path
			break LOOP

		default:
			panic("logic error, wrong node type")
		}
	}

	// start backtracking, unwind the stack
	for ; depth >= 0; depth-- {
		n = stack[depth]

		// only the lastOctet may have a different prefix len
		// all others are just host routes
		var idx uint8
		octet = octets[depth]
		// Last “octet” from prefix
		// Note: For /32 and /128, depth never reaches strideCount (4/16),
		if depth == strideCount {
			idx = art.PfxToIdx(octet, modBits)
		} else {
			idx = art.OctetToIdx(octet)
		}

		// micro benchmarking, skip if there is no match
		if !n.Contains(idx) {
			continue
		}

		// yield all the matching prefixes, not just the lpm
		if !n.EachLookupPrefix(ip, depth, idx, yield) {
			// early exit
			return
		}
	}
}

// Subnets yields all subnet prefixes covered by pfx that exist in the trie,
// in CIDR sort order.
//
// It first locates the trie node corresponding to pfx, then recursively
// yields all prefixes and child entries contained within that subtree.
// The traversal uses sorted iteration to maintain canonical CIDR ordering.
//
// The function handles various node types (internal nodes, leaves, and fringes)
// and uses EachSubnet and AllRecSorted for sorted traversal of covered prefixes.
//
// Parameters:
//   - pfx: The parent prefix whose subnets should be yielded
//   - yield: Callback function invoked for each subnet prefix/value pair
//
// The yield function receives prefix/value pairs and returns false to stop
// the iteration early. If pfx doesn't exist in the trie, no prefixes are yielded.
func (n *FastACLNode) Subnets(pfx netip.Prefix, yield func(netip.Prefix) bool) {
	panic("TODO")

	// values derived from pfx
	ip := pfx.Addr()
	pfxLen := pfx.Bits()
	is4 := ip.Is4()
	octets := ip.AsSlice()
	strideCount, modBits := DivMod8(pfxLen)

	// find the trie node
	for depth, octet := range octets {
		// Last “octet” from prefix
		// Note: For /32 and /128, depth never reaches strideCount (4/16),
		// so those are handled below via the fringe/leaf path.
		if depth == strideCount {
			idx := art.PfxToIdx(octet, modBits)
			n.EachSubnet(octets, depth, is4, idx, yield)
			return
		}

		if !n.Children.Test(octet) {
			return
		}
		kid := n.MustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *FastACLNode:
			n = kid
			continue // descend down to next trie level

		case *CIDRLeaf:
			if pfx.Bits() <= kid.Prefix.Bits() && pfx.Overlaps(kid.Prefix) {
				yield(kid.Prefix)
			}
			return // immediate return

		case *FringeLeaf:
			// get the LPM prefix back from ip and depth
			// it's a fringe, bits are always /8, /16, /24, ...
			fringePfx, _ := ip.Prefix((depth + 1) << 3)

			if pfx.Bits() <= fringePfx.Bits() && pfx.Overlaps(fringePfx) {
				yield(fringePfx)
			}
			return // immediate return

		default:
			panic("logic error, wrong node type")
		}
	}
}

// Overlaps recursively compares two trie nodes and returns true
// if any of their prefixes or descendants overlap.
//
// The implementation checks for:
// 1. Direct overlapping prefixes on this node level
// 2. Prefixes in one node overlapping with children in the other
// 3. Matching child addresses in both nodes, which are recursively compared
//
// All 12 possible type combinations for child entries (node, leaf, fringe) are supported.
//
// The function is optimized for early exit on first match and uses heuristics to
// choose between set-based and loop-based matching for performance.
func (n *FastACLNode) Overlaps(o *FastACLNode, depth int) bool {
	panic("TODO")

	nPfxCount := n.PrefixCount()
	oPfxCount := o.PrefixCount()

	nChildCount := n.ChildCount()
	oChildCount := o.ChildCount()

	// ##############################
	// 1. Test if any routes overlaps
	// ##############################

	// full cross check
	if nPfxCount > 0 && oPfxCount > 0 {
		if n.OverlapsRoutes(o) {
			return true
		}
	}

	// ####################################
	// 2. Test if routes overlaps any child
	// ####################################

	// swap nodes to help chance on its way,
	// if the first call to expensive overlapsChildrenIn() is already true,
	// if both orders are false it doesn't help either
	if nChildCount > oChildCount {
		n, o = o, n

		nPfxCount = n.PrefixCount()
		oPfxCount = o.PrefixCount()

		nChildCount = n.ChildCount()
		oChildCount = o.ChildCount()
	}

	if nPfxCount > 0 && oChildCount > 0 {
		if n.OverlapsChildrenIn(o) {
			return true
		}
	}

	// symmetric reverse
	if oPfxCount > 0 && nChildCount > 0 {
		if o.OverlapsChildrenIn(n) {
			return true
		}
	}

	// ############################################
	// 3. children with same octet in nodes n and o
	// ############################################

	// stop condition, n or o have no children
	if nChildCount == 0 || oChildCount == 0 {
		return false
	}

	// stop condition, no child with identical octet in n and o
	if !n.Children.Overlaps(&o.Children.BitSet256) {
		return false
	}

	return n.OverlapsSameChildren(o, depth)
}

// OverlapsRoutes compares the prefix sets of two nodes (n and o).
//
// It first checks for direct bitset intersection (identical indices),
// then walks both prefix sets using the Contains method to detect if any
// of the n-prefixes is contained in o, or vice versa.
func (n *FastACLNode) OverlapsRoutes(o *FastACLNode) bool {
	panic("TODO")

	// some prefixes are identical, trivial overlap
	if n.Prefixes.Overlaps(&o.Prefixes) {
		return true
	}

	// get the lowest idx (biggest prefix)
	nFirstIdx, _ := n.Prefixes.FirstSet()
	oFirstIdx, _ := o.Prefixes.FirstSet()

	// start with other min value
	nIdx := oFirstIdx
	oIdx := nFirstIdx

	nOK := true
	oOK := true

	// zip, range over n and o together to help chance on its way
	for nOK || oOK {
		if nOK {
			// does any route in o overlap this prefix from n
			if nIdx, nOK = n.Prefixes.NextSet(nIdx); nOK {
				if o.Contains(nIdx) {
					return true
				}

				if nIdx == 255 {
					// stop, don't overflow uint8!
					nOK = false
				} else {
					nIdx++
				}
			}
		}

		if oOK {
			// does any route in n overlap this prefix from o
			if oIdx, oOK = o.Prefixes.NextSet(oIdx); oOK {
				if n.Contains(oIdx) {
					return true
				}

				if oIdx == 255 {
					// stop, don't overflow uint8!
					oOK = false
				} else {
					oIdx++
				}
			}
		}
	}

	return false
}

// OverlapsChildrenIn checks whether the prefixes in node n
// overlap with any children (by address range) in node o.
//
// Uses bitset intersection or manual iteration heuristically,
// depending on prefix and child count.
//
// Bitset-based matching uses precomputed coverage tables
// to avoid per-address looping. This is critical for high fan-out nodes.
func (n *FastACLNode) OverlapsChildrenIn(o *FastACLNode) bool {
	panic("TODO")

	pfxCount := n.PrefixCount()
	childCount := o.ChildCount()

	// heuristic: 15 is the crossover point where bitset operations become
	// more efficient than iteration, determined by micro benchmarks on typical
	// routing table distributions
	const overlapsRangeCutoff = 15

	doRange := childCount < overlapsRangeCutoff || pfxCount > overlapsRangeCutoff

	// do range over, not so many children and maybe too many prefixes for other algo below
	if doRange {
		for addr := range o.Children.All() {
			if n.Contains(art.OctetToIdx(addr)) {
				return true
			}
		}
		return false
	}

	// do bitset intersection, alloted route table with child octets
	// maybe too many children for range-over or not so many prefixes to
	// build the alloted routing table from them

	// use allot table with prefixes as bitsets, bitsets are precalculated.
	for idx := range n.Prefixes.All() {
		if o.Children.Overlaps(&allot.FringeRoutesLookupTbl[idx]) {
			return true
		}
	}

	return false
}

// OverlapsSameChildren compares all matching child addresses (octets)
// between node n and node o recursively.
//
// For each shared address, the corresponding child nodes (of any type)
// are compared using FastACLNodeOverlapsTwoChildren, which handles all
// node/leaf/fringe combinations.
func (n *FastACLNode) OverlapsSameChildren(o *FastACLNode, depth int) bool {
	panic("TODO")

	// intersect the child bitsets from n with o
	commonChildren := n.Children.And(&o.Children.BitSet256)

	for addr, ok := commonChildren.NextSet(0); ok; {
		nChild := n.MustGetChild(addr)
		oChild := o.MustGetChild(addr)

		if n.OverlapsTwoChildren(nChild, oChild, depth+1) {
			return true
		}

		if addr == 255 {
			break // Prevent uint8 overflow
		}

		addr, ok = commonChildren.NextSet(addr + 1)
	}
	return false
}

// OverlapsPrefixAtDepth returns true if any route in the subtree rooted at this node
// overlaps with the given pfx, starting the comparison at the specified depth.
//
// This function supports structural overlap detection even in compressed or sparse
// paths within the trie, including fringe and leaf nodes. Matching is directional:
// it returns true if a route fully covers pfx, or if pfx covers an existing route.
//
// At each step, it checks for visible prefixes and children that may intersect the
// target prefix via stride-based longest-prefix test. The walk terminates early as
// soon as a structural overlap is found.
//
// This function underlies the top-level OverlapsPrefix behavior and handles details of
// trie traversal across varying prefix lengths and compression levels.
func (n *FastACLNode) OverlapsPrefixAtDepth(pfx netip.Prefix, depth int) bool {
	panic("TODO")

	ip := pfx.Addr()
	pfxLen := pfx.Bits()
	octets := ip.AsSlice()
	strideCount, modBits := DivMod8(pfxLen)

	for ; depth < len(octets); depth++ {
		if depth > strideCount {
			break
		}

		octet := octets[depth]

		// full octet path in node trie, check overlap with last prefix octet
		if depth == strideCount {
			return n.OverlapsIdx(art.PfxToIdx(octet, modBits))
		}

		// test if any route overlaps prefix´ so far
		// no best match needed, forward tests without backtracking
		if n.PrefixCount() != 0 && n.Contains(art.OctetToIdx(octet)) {
			return true
		}

		if !n.Children.Test(octet) {
			return false
		}

		// next child, node or leaf
		switch kid := n.MustGetChild(octet).(type) {
		case *FastACLNode:
			n = kid
			continue

		case *CIDRLeaf:
			return kid.Prefix.Overlaps(pfx)

		case *FringeLeaf:
			return true

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable: " + pfx.String())
}

// OverlapsIdx returns true if the given prefix index overlaps with any entry in this node.
//
// The overlap detection considers three categories:
//
//  1. Whether any stored prefix in this node covers the requested prefix (LPM test)
//  2. Whether the requested prefix covers any stored route in the node
//  3. Whether the requested prefix overlaps with any fringe or child entry
//
// Internally, it leverages precomputed bitsets from the allotment model,
// using fast bitwise set intersections instead of explicit range comparisons.
// This enables high-performance overlap checks on a single stride level
// without descending further into the trie.
func (n *FastACLNode) OverlapsIdx(idx uint8) bool {
	panic("TODO")

	// 1. Test if any route in this node overlaps prefix?
	if n.Contains(idx) {
		return true
	}

	// 2. Test if prefix overlaps any route in this node
	if n.Prefixes.Overlaps(&allot.PfxRoutesLookupTbl[idx]) {
		return true
	}

	// 3. Test if prefix overlaps any child in this node
	return n.Children.Overlaps(&allot.FringeRoutesLookupTbl[idx])
}

// OverlapsTwoChildren handles all 3x3 combinations of
// node kinds (node, leaf, fringe).
//
//	3x3 possible different combinations for n and o
//
//	node, node    --> overlaps rec descent
//	node, leaf    --> overlapsPrefixAtDepth
//	node, fringe  --> true
//
//	leaf, node    --> overlapsPrefixAtDepth
//	leaf, leaf    --> netip.Prefix.Overlaps
//	leaf, fringe  --> true
//
//	fringe, node    --> true
//	fringe, leaf    --> true
//	fringe, fringe  --> true
func (n *FastACLNode) OverlapsTwoChildren(nChild, oChild any, depth int) bool {
	panic("TODO")

	// child type detection
	nNode, nIsNode := nChild.(*FastACLNode)
	nLeaf, nIsLeaf := nChild.(*CIDRLeaf)
	_, nIsFringe := nChild.(*FringeLeaf)

	oNode, oIsNode := oChild.(*FastACLNode)
	oLeaf, oIsLeaf := oChild.(*CIDRLeaf)
	_, oIsFringe := oChild.(*FringeLeaf)

	// Handle all 9 combinations with a single expression
	switch {
	// NODE cases
	case nIsNode && oIsNode:
		return nNode.Overlaps(oNode, depth)
	case nIsNode && oIsLeaf:
		return nNode.OverlapsPrefixAtDepth(oLeaf.Prefix, depth)
	case nIsNode && oIsFringe:
		return true

	// LEAF cases
	case nIsLeaf && oIsNode:
		return oNode.OverlapsPrefixAtDepth(nLeaf.Prefix, depth)
	case nIsLeaf && oIsLeaf:
		return oLeaf.Prefix.Overlaps(nLeaf.Prefix)
	case nIsLeaf && oIsFringe:
		return true

	// FRINGE cases
	case nIsFringe:
		return true // fringe overlaps with everything

	default:
		panic("logic error, wrong node type combination")
	}
}
