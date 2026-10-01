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
// so this optimization for speed is possible.
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

// CIDRLeaf represents an immutable, path-compressed routing entry that stores a prefix.
// Immutability & Reference Sharing:
// Instances of CIDRLeaf are treated as strictly immutable after creation. The internal
// netip.Prefix must not be modified after construction.
//
// Consequently, *CIDRLeaf pointers can be safely shared across table and node clones
// (e.g., during CloneFlat and CloneRec) without additional heap allocations, mutexes,
// or risk of data races.
type CIDRLeaf struct {
	// prefix is unexported to prevent mutation after instantiation.
	prefix netip.Prefix
}

// NewCIDRLeaf returns a new immutable *CIDRLeaf initialized with the given prefix.
func NewCIDRLeaf(pfx netip.Prefix) *CIDRLeaf {
	return &CIDRLeaf{prefix: pfx}
}

// Prefix returns the canonical netip.Prefix encapsulated by this leaf.
func (l *CIDRLeaf) Prefix() netip.Prefix {
	return l.prefix
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

// CloneRec performs a recursive deep copy of the FastACLNode and all its trie descendants.
//
// Value-based structures (bitsets, caches, CIDRLeaf entries) are cloned directly,
// while child FastACLNode instances in the sparse array are recursively duplicated
// to ensure complete structural isolation.
//
// Returns a new FastACLNode instance containing a deep clone of the receiver subtree.
func (n *FastACLNode) CloneRec() *FastACLNode {
	if n == nil {
		return nil
	}

	// Perform a flat clone of the current node's local storage and scalar fields.
	c := n.CloneFlat()

	// Recursively deep-copy all intermediate child nodes.
	for i, kidAny := range c.Children.Items {
		if kid, ok := kidAny.(*FastACLNode); ok {
			c.Children.Items[i] = kid.CloneRec()
		}
	}

	return c
}

// CloneFlat returns a shallow copy of the FastACLNode.
//
// All scalar fields, bitsets (Fringes, Prefixes), population counts, lookup
// rank caches, and the sparse array slice header (via Children.Copy()) are duplicated
// by value.
//
// Shallow Copy Semantics:
// The internal Children slice is duplicated, but its element pointers (*CIDRLeaf and
// *FastACLNode) remain shared references to the original child instances.
//
//   - Pointers to immutable *CIDRLeaf instances (netip.Prefix) require no further processing and can
//     be safely shared across table clones without heap re-allocations.
//   - Pointers to mutable *FastACLNode instances are subsequently replaced with deep
//     copies by the calling CloneRec method to ensure complete structural isolation.
//
// Returns nil if the receiver node is nil.
func (n *FastACLNode) CloneFlat() *FastACLNode {
	if n == nil {
		return nil
	}

	c := new(FastACLNode)

	// Copy scalar bitsets and lookup metadata by value.
	c.Fringes = n.Fringes
	c.Prefixes = n.Prefixes
	c.prefixCount = n.prefixCount
	c.childRankCache = n.childRankCache

	// Duplicate sparse array layout and internal slice storage.
	// Element pointers (*CIDRLeaf and *FastACLNode) are shallow-copied into the new slice.
	c.Children = *(n.Children.Copy())

	return c
}

// AggregateRec compresses the FastACLNode in-place by pruning redundant subnets,
// removing child nodes covered by parent prefixes, recursively compressing child
// subtrees, promoting single-entry children to higher-level entries, and
// merging adjacent sibling prefixes or fringe nodes.
//
// The aggregation process executes the following steps in order:
//
//  1. Default Route Purge: If the node contains a default route (CBT index 1), all
//     other local prefixes, fringes, and child nodes in the subtree are pruned immediately.
//
//  2. Prefix Subsumption: Removes more-specific local prefixes that are fully covered
//     by a broader supernet prefix within the same node's CBT bitset.
//
//  3. Child and Fringe Subsumption: Deletes child nodes and fringe entries that are fully
//     covered by an existing local prefix or fringe in the current node.
//
//  4. Recursive Descent & Promotion: Recursively calls AggregateRec on child nodes.
//     Upon return, if a child node has been compressed down to a single total entry,
//     it is promoted in-place at the parent node:
//     - A single local prefix is converted to a CIDR prefix and re-inserted at parent depth.
//     - A single fringe entry is reconstructed as a CIDR prefix and re-inserted at parent depth.
//     - A single grandchild *CIDRLeaf is promoted directly into the parent's child slot.
//
//  5. Fringe Merging: Collapses aligned pairs of adjacent fringe entries (covering a /7 block)
//     into a single supernet prefix inserted into the current node's bitset.
//
//  6. Prefix Merging: Repeatedly combines aligned pairs of adjacent sibling prefixes
//     into their parent supernet prefix in the CBT bitset until no further merges are possible.
//
// Returns modified, the number of structural mutation operations performed during
// the aggregation pass.
func (n *FastACLNode) AggregateRec(ptx PathContext) (modified int) {
	// #########################################################################################
	// 1. Default Route Purge: If node has default route, purge all prefixes, fringes, and children.
	if n.Prefixes.Test(1) {
		*n = FastACLNode{}

		// Restore default route in this node
		n.InsertPrefix(1)

		return modified + 1
	}

	// #########################################################################################
	// 2. Prefix Subsumption: Remove subnets in the bitset that are fully covered by a supernet.
	var ok bool
	var pfxIdx uint8
	var changed bool
	for {
		// Find the next set prefix index, starting search at bit 0
		if pfxIdx, ok = n.Prefixes.NextSet(pfxIdx); !ok {
			break
		}

		// The last prefix index (255) has no potential subnets below it
		if pfxIdx == 255 {
			break
		}

		// Find all prefixes covered by pfxIdx using the allotment lookup table
		// including pfxIdx itself.
		covered := n.Prefixes.And(&allot.PfxRoutesLookupTbl[pfxIdx])
		if covered.OnesCount() > 1 {
			// Clear all covered prefixes, including pfxIdx itself
			n.Prefixes = n.Prefixes.Xor(&covered)

			// Re-enable cleared pfxIdx
			n.Prefixes.Set(pfxIdx)

			// Track structural mutation
			changed = true
		}

		// Advance index to search for the next prefix
		pfxIdx++
	}

	// Recalculate prefix count after modifications
	if changed {
		// Track structural mutation
		modified++

		//nolint:gosec // G115: integer overflow conversion int -> uint16
		n.prefixCount = uint16(n.Prefixes.OnesCount())
	}

	// ###########################################################################
	// 3. Child and Fringe Subsumption: Remove fringe and child nodes covered by any
	// remaining local prefix or fringe in this node.
	//
	// We first accumulate all matching addresses into BitSet256 masks and delete
	// them in batch.
	var batchFringeAddrs bitset.BitSet256
	var batchChildAddrs bitset.BitSet256

	for idx := range n.Prefixes.All() {
		// Collect fringe and child addresses covered by the current prefix
		// using the fringe lookup table
		coveredFringeAddrs := n.Fringes.And(&allot.FringeRoutesLookupTbl[idx])
		coveredChildAddrs := n.Children.And(&allot.FringeRoutesLookupTbl[idx])

		// Accumulate covered addresses
		batchFringeAddrs = batchFringeAddrs.Or(&coveredFringeAddrs)
		batchChildAddrs = batchChildAddrs.Or(&coveredChildAddrs)
	}

	// Batch delete fringe entries covered by local prefixes
	for addr := range batchFringeAddrs.All() {
		n.DeleteFringe(addr)
		modified++
	}

	// Batch delete child nodes covered by local prefixes
	for addr := range batchChildAddrs.All() {
		n.DeleteChild(addr)
		modified++
	}

	// Delete child nodes directly covered by remaining fringes
	for addr := range n.Fringes.All() {
		if exists := n.DeleteChild(addr); exists {
			modified++
		}
	}

	// #########################################################
	// 4. Recursive Descent: Top-down compression of child nodes
	for addr := range n.Children.All() {
		anyKid := n.MustGetChild(addr)
		kid, ok := anyKid.(*FastACLNode)
		// Skip compressed leaf nodes (*CIDRLeaf)
		if !ok {
			continue
		}

		// Recurse down with updated path context
		nextPtx := ptx
		nextPtx.Path[ptx.Depth] = addr
		nextPtx.Depth++

		modified += kid.AggregateRec(nextPtx)

		// #############################################################
		// After the recursive call returns, attempt to promote single-entry children
		pfxCount := kid.PrefixCount()
		fringeCount := kid.FringeCount()
		childCount := kid.ChildCount()

		// Nothing to promote if combined entry count exceeds 1
		if pfxCount+fringeCount+childCount > 1 {
			continue
		}

		// Promote single-entry child nodes to lower-overhead structures at the parent level.
		// Note: nextPtx provides the child's context (depth = ptx.Depth + 1) for CIDR reconstruction,
		// while ptx.Depth represents the current parent node level for insertion.
		switch {
		case pfxCount == 1:
			// Restore CIDR from single local prefix and re-insert at current parent depth
			nextPtx.Slot, _ = kid.Prefixes.FirstSet()
			cidr := PrefixCIDR(nextPtx)
			n.DeleteChild(addr)
			n.Insert(cidr, ptx.Depth)

		case fringeCount == 1:
			// Restore CIDR from single fringe entry and re-insert at current parent depth
			nextPtx.Slot, _ = kid.Fringes.FirstSet()
			cidr := FringeCIDR(nextPtx)
			n.DeleteChild(addr)
			n.Insert(cidr, ptx.Depth)

		case childCount == 1:
			// Directly access the underlying slice without bitset traversal operations.
			// Since there is exactly one child node, it is guaranteed to be the first
			// and only item in the slice.
			switch grandKid := kid.Children.Items[0].(type) {
			case *FastACLNode:
				// Intermediate path node with multi-level depth, leave structural hierarchy intact
				continue

			case *CIDRLeaf:
				// Promote single grandchild leaf directly into parent's child slot
				n.InsertChild(addr, grandKid)
			}
		}
	}

	// #############################################################################
	// 5. Fringe Merging: Collapse aligned fringe pairs (/8 siblings) into a /7 supernet prefix.
	alignedPairs := n.Fringes.AlignedPairs()
	for addr := range alignedPairs.All() {
		// Promote aligned fringe pair (addr and addr+1) to a /7 prefix index
		n.InsertPrefix(art.PfxToIdx(addr, 7))

		// Delete the aligned fringe pair.
		n.DeleteFringe(addr)
		n.DeleteFringe(addr + 1)

		modified++
	}

	// #############################################################
	// 6. Prefix Merging: Repeatedly merge aligned sibling prefixes within the CBT bitset.
	for { // Loop until cascading merges complete
		more := false

		alignedPairs := n.Prefixes.AlignedPairs()
		for idx := range alignedPairs.All() {
			// Insert parent supernet index (idx >> 1 in CBT)
			n.InsertPrefix(idx >> 1)

			// Delete left and right sibling subnets
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

// Insert inserts a prefix into the trie starting at the specified stride depth.
//
// It traverses the trie top-down across 8-bit octet strides. It stores terminal
// prefixes in local Complete Binary Tree (CBT) prefix tables or stride-aligned
// fringe bitsets. Unbranched subtrees are path-compressed into CIDRLeaf nodes.
//
// If a path collision occurs with an existing CIDRLeaf, the leaf is expanded
// into an intermediate FastACLNode and pushed down the trie.
//
// Returns true if the prefix was already present in the trie, or false if it was
// newly inserted.
//
// Note: pfx must be provided in canonical (masked) form.
func (n *FastACLNode) Insert(pfx netip.Prefix, depth int) (exists bool) {
	ip := pfx.Addr()
	pfxLen := pfx.Bits()
	octets := ip.AsSlice()
	strideCount, modBits := DivMod8(pfxLen)

	// Traverse octets starting at the provided stride depth.
	for ; depth < len(octets); depth++ {
		octet := octets[depth]

		// 1. Terminal stride reached: insert directly into local CBT prefix table.
		if depth == strideCount {
			return n.InsertPrefix(art.PfxToIdx(octet, modBits))
		}

		// 2. Stride-aligned boundary match (/8, /16, /24, etc.): insert into fringe bitset.
		if IsFringe(depth, pfxLen) {
			return n.InsertFringe(octet)
		}

		// 3. Unoccupied child slot: path-compress remaining strides into a CIDRLeaf.
		if !n.Children.Test(octet) {
			return n.InsertChild(octet, &CIDRLeaf{prefix: pfx})
		}

		// Retrieve existing child node or path-compressed leaf at current octet slot.
		kid := n.MustGetChild(octet)

		switch kid := kid.(type) {
		case *FastACLNode:
			// Descend deeper into next trie level.
			n = kid

		case *CIDRLeaf:
			// 4. Collision resolution with an existing path-compressed leaf.
			// Verify exact prefix match.
			if kid.prefix == pfx {
				return true
			}

			// Path divergence: allocate intermediate node, push existing leaf down,
			// swap child slot, and descend into the new node to insert pfx.
			newNode := new(FastACLNode)
			newNode.Insert(kid.prefix, depth+1)

			n.InsertChild(octet, newNode)
			n = newNode
		}
	}

	panic("unreachable: trie depth exceeded address bounds")
}

// Delete removes a prefix from the trie rooted at node n.
//
// It traverses the trie top-down across 8-bit octet strides to locate the target
// prefix in local CBT prefix bitsets, fringe bitsets, or path-compressed CIDRLeaf nodes.
//
// Upon successful deletion, it invokes PurgeAndCompress using the accumulated
// ancestor stack to prune redundant intermediate nodes and restore optimal path compression.
//
// Returns true if the prefix existed and was removed, or false if it was not found.
//
// Note: pfx must be provided in canonical (masked) form.
func (n *FastACLNode) Delete(pfx netip.Prefix) (exists bool) {
	ip := pfx.Addr()
	pfxLen := pfx.Bits()
	is4 := ip.Is4()
	octets := ip.AsSlice()
	strideCount, modBits := DivMod8(pfxLen)

	// Record ancestor nodes during descent; PurgeAndCompress uses this stack to
	// walk back up and prune empty or re-compressible nodes post-deletion.
	stack := [MaxTreeDepth]*FastACLNode{}

	for depth, octet := range octets {
		depth &= DepthMask // BCE hint: bounds check elimination for fast-path traversal.

		stack[depth] = n // Record current parent node before descending.

		// 1. Terminal stride boundary reached: delete directly from local CBT prefix table.
		if depth == strideCount {
			if exists = n.DeletePrefix(art.PfxToIdx(octet, modBits)); !exists {
				return false
			}

			// Prune empty nodes and re-compress path upwards.
			n.PurgeAndCompress(stack[:depth], octets, is4)
			return true
		}

		// 2. Stride-aligned boundary match (/8, /16, /24, etc.): delete from fringe bitset.
		if IsFringe(depth, pfxLen) {
			if exists := n.DeleteFringe(octet); !exists {
				return false
			}

			// Prune empty nodes and re-compress path upwards.
			n.PurgeAndCompress(stack[:depth], octets, is4)
			return true
		}

		// 3. Early termination: no child node or leaf exists at the target octet slot.
		if !n.Children.Test(octet) {
			return false
		}

		// Retrieve child node or path-compressed leaf at current octet slot.
		kid := n.MustGetChild(octet)

		switch kid := kid.(type) {
		case *FastACLNode:
			// Descend deeper into next trie level.
			n = kid

		case *CIDRLeaf:
			// 4. Path-compressed leaf encountered: verify exact prefix match.
			if kid.prefix != pfx {
				return false
			}

			n.DeleteChild(octet)

			// Prune empty nodes and re-compress path upwards.
			n.PurgeAndCompress(stack[:depth], octets, is4)
			return true
		}
	}

	panic("unreachable: trie depth exceeded address bounds")
}

// PurgeAndCompress performs bottom-up structural trie optimization to restore path compression
// following a prefix deletion.
//
// It unwinds the provided stack of parent nodes using slices.Backward, evaluating nodes
// that have become sparse (containing at most one entry across prefixes, fringes, and children).
// Redundant sparse nodes are pruned from their parent slot and elevated into path-compressed
// CIDRLeaf representations at higher trie levels.
func (n *FastACLNode) PurgeAndCompress(stack []*FastACLNode, octets []uint8, is4 bool) {
	// Unwind parent ancestor stack bottom-up to collapse redundant path nodes.
	for depth, parent := range slices.Backward(stack) {
		pfxCount := n.PrefixCount()
		fringeCount := n.FringeCount()
		childCount := n.ChildCount()

		// A node with more than one total entry is structurally required and cannot be pruned.
		if pfxCount+fringeCount+childCount > 1 {
			return
		}

		octet := octets[depth]

		switch {
		case childCount == 1:
			// Directly access the underlying slice without bitset traversal operations.
			// Since there is exactly one child node, it is guaranteed to be the first
			// and only item in the slice.
			anyKid := n.Children.Items[0]

			// If the child is an intermediate node, path compression cannot proceed higher.
			if _, ok := anyKid.(*FastACLNode); ok {
				return
			}

			// Elevate the single path-compressed CIDRLeaf to the parent level.
			leaf := anyKid.(*CIDRLeaf)
			parent.DeleteChild(octet)
			parent.Insert(leaf.prefix, depth)

		case fringeCount == 1:
			// Elevate the single fringe entry to the parent level as a path-compressed leaf.
			addr, _ := n.Fringes.FirstSet()
			ptx := NewPathContext(octets, depth+1, addr, is4)

			parent.DeleteChild(octet)
			parent.Insert(FringeCIDR(ptx), depth)

		case pfxCount == 1:
			// Elevate the single CBT local prefix to the parent level as a path-compressed leaf.
			idx, _ := n.Prefixes.FirstSet()
			ptx := NewPathContext(octets, depth+1, idx, is4)

			parent.DeleteChild(octet)
			parent.Insert(PrefixCIDR(ptx), depth)
		}

		// Advance upwards to continue structural pruning at the next parent level.
		n = parent
	}
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
			if nKid.prefix != oKid.prefix {
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
// It returns immediately if n is empty. For each visited internal node
// it calls dump to write the node's representation, then iterates its child
// addresses and recurses into children of type *FastACLNode (internal subnodes).
// The path slice and depth together represent the byte-wise path
// from the root to the current node; depth is incremented for each recursion.
// The is4 flag controls IPv4/IPv6 formatting used by dump.
func (n *FastACLNode) DumpRec(w io.Writer, path StridePath, depth int, is4 bool) {
	if n.IsEmpty() {
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
			pfx := CidrFromPath(path[:], depth, is4, idx)
			fmt.Fprintf(w, " [%d]➜{%s}", idx, pfx)
		}

		fmt.Fprintln(w)
	}

	// print the fringes
	if n.FringeCount() != 0 {
		fmt.Fprintf(w, "%sfringe(#%*d):", indent, width, n.FringeCount())

		for addr := range n.Fringes.All() {
			fringePfx := CidrForFringe(path[:], depth, is4, addr)
			fmt.Fprintf(w, " [%s]➜{%s}", addrFmt(addr, is4), fringePfx)
		}

		fmt.Fprintln(w)
	}

	// print the nodes and leafs
	if n.ChildCount() != 0 {
		fmt.Fprintf(w, "%s child(#%*d):", indent, width, n.ChildCount())

		for addr, child := range n.AllChildren() {
			switch child := child.(type) {
			case *FastACLNode:
				fmt.Fprintf(w, " [%s]↓", addrFmt(addr, is4))

			case *CIDRLeaf:
				fmt.Fprintf(w, " [%s]➜{%s}", addrFmt(addr, is4), child.prefix)

			default:
				panic("logic error, wrong node type")
			}
		}

		fmt.Fprintln(w)
	}
}

// hasType classifies the given node into one of the nodeType values.
//
// It inspects immediate statistics (prefix count, child count, node, leaf and
// fringe counts) for the node and returns:
//   - nullNode: no prefixes and no children
//   - stopNode: no subnodes (nodes == 0)
//   - pathNode: has subnodes only (no prefixes, leaves or fringes)
//   - fullNode: has prefixes or fringes and also has subnodes
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
	case s.Prefixes == 0 && s.Leaves == 0 && s.Fringes == 0:
		return pathNode
	default:
		return fullNode
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

// AllRec recursively traverses the trie starting at the current node,
// applying the provided yield function to every stored prefix.
//
// For each route entry, yield is invoked. If yield returns false,
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
func (n *FastACLNode) AllRec(ptx PathContext, yield func(netip.Prefix) bool) bool {
	if n.IsEmpty() {
		return true
	}

	// 1. Direct local node prefixes
	for idx := range n.Prefixes.All() {
		ptx.Slot = idx
		if !n.yieldPrefix(ptx, yield) {
			return false
		}
	}

	// 2. Fringe prefixes at stride boundaries
	for addr := range n.Fringes.All() {
		ptx.Slot = addr
		if !n.YieldFringe(ptx, yield) {
			return false
		}
	}

	// 3. Child nodes and path-compressed leaves
	for addr := range n.Children.All() {
		ptx.Slot = addr
		if !n.yieldChildRec(ptx, yield) {
			return false
		}
	}

	return true
}

// AllRecSorted recursively traverses the trie node and its descendants in canonical
// CIDR prefix-sorted order, invoking the yield callback for each stored prefix.
//
// The order guarantees that prefixes are yielded by numerical network address,
// followed by prefix length (e.g., 10.0.0.0/8 before 10.0.0.0/16, and 10.0.0.0/16 before 10.1.0.0/16).
//
// Iteration halts immediately if the yield callback returns false, propagating
// the cancellation up the call stack. Returns false if iteration was terminated early,
// or true if all items were processed successfully.
func (n *FastACLNode) AllRecSorted(ptx PathContext, yield func(netip.Prefix) bool) bool {
	// Extract local CBT prefix indices and allocate exact-capacity slice.
	allIndices := n.Prefixes.AppendBits(make([]uint8, 0, n.PrefixCount()))

	// Combine fringes and child subtrees to process all descendant octets in strictly increasing order (0..255).
	fringeOrChild := n.Fringes.Or(&n.Children.BitSet256)
	allAddrs := fringeOrChild.AppendBits(make([]uint8, 0, fringeOrChild.OnesCount()))

	// Sort local prefix indices into canonical CIDR rank order.
	slices.SortFunc(allIndices, CmpIndexRank)

	// Cursor index tracking progression through allAddrs slice across iterations.
	addrCursor := 0

	// Interleave local prefixes, fringes, and child subtrees in canonical CIDR rank order.
	for _, pfxIdx := range allIndices {
		pfxOctet, _ := art.IdxToPfx(pfxIdx)

		// Yield all child subtrees/fringes whose base byte address strictly precedes the current prefix's target octet.
		for ; addrCursor < len(allAddrs); addrCursor++ {
			addr := allAddrs[addrCursor]
			if addr >= pfxOctet {
				break
			}

			ptx.Slot = addr
			if !n.yieldAddr(ptx, yield) {
				return false
			}
		}

		// Yield the local prefix once all preceding addrs have been traversed.
		ptx.Slot = pfxIdx
		if !n.yieldPrefix(ptx, yield) {
			return false
		}
	}

	// Yield any remaining fringe and child positioned after all local prefixes.
	for _, addr := range allAddrs[addrCursor:] {
		ptx.Slot = addr
		if !n.yieldAddr(ptx, yield) {
			return false
		}
	}

	return true
}

// YieldFringe reconstructs and yields a fringe prefix from its byte address.
func (n *FastACLNode) YieldFringe(ptx PathContext, yield func(netip.Prefix) bool) bool {
	return yield(FringeCIDR(ptx))
}

// yieldPrefix reconstructs and yields a local prefix from its CBT index.
func (n *FastACLNode) yieldPrefix(ptx PathContext, yield func(netip.Prefix) bool) bool {
	return yield(PrefixCIDR(ptx))
}

// yieldAddr yields fringe entries and child subtrees positioned at the specified byte address.
func (n *FastACLNode) yieldAddr(ptx PathContext, yield func(netip.Prefix) bool) bool {
	if n.Fringes.Test(ptx.Slot) {
		if !n.YieldFringe(ptx, yield) {
			return false
		}
	}

	if n.Children.Test(ptx.Slot) {
		if !n.yieldChildRecSorted(ptx, yield) {
			return false
		}
	}

	return true
}

// yieldChildRec traverses subtrees or yields path-compressed leaf prefixes at the specified byte address.
func (n *FastACLNode) yieldChildRec(ptx PathContext, yield func(netip.Prefix) bool) bool {
	switch kid := n.MustGetChild(ptx.Slot).(type) {
	case *FastACLNode:
		nextPtx := ptx
		nextPtx.Path[ptx.Depth] = ptx.Slot
		nextPtx.Depth++

		return kid.AllRec(nextPtx, yield)

	case *CIDRLeaf:
		return yield(kid.prefix)
	}

	return false
}

// yieldChildRecSorted traverses subtrees or yields path-compressed leaf prefixes in CIDR sort order
// at the specified byte address.
func (n *FastACLNode) yieldChildRecSorted(ptx PathContext, yield func(netip.Prefix) bool) bool {
	switch kid := n.MustGetChild(ptx.Slot).(type) {
	case *FastACLNode:
		nextPtx := ptx
		nextPtx.Path[ptx.Depth] = ptx.Slot
		nextPtx.Depth++

		return kid.AllRecSorted(nextPtx, yield)

	case *CIDRLeaf:
		return yield(kid.prefix)
	}

	return false
}

// YieldSupernets performs a hierarchical lookup of all matching supernets
// within the current node's 8-bit stride-based table.
//
// It evaluates both stride-aligned fringe entries and configured prefixes:
//  1. Fringe Check: If the target octet matches a fringe bit in [FastACLNode.Fringes],
//     the enclosing stride boundary prefix (e.g., /8, /16, /24) is reconstructed and yielded first
//     provided its length does not exceed pfxLen.
//  2. Prefix Check: Set bits in [FastACLNode.Prefixes] are intersected with a precomputed complete
//     binary tree (CBT) ancestor mask. Matching prefixes are yielded in descending specificity
//     order (longest to shortest prefix length).
//
// Returns false if the yield callback requested an early exit.
//
// Requirement: The caller MUST provide a canonicalized prefix pfx (i.e. pfx == pfx.Masked()).
func (n *FastACLNode) YieldSupernets(pfx netip.Prefix, depth int, octet byte, pfxIdx uint8, yield func(netip.Prefix) bool) (ok bool) {
	ip := pfx.Addr()
	pfxLen := pfx.Bits()

	if n.Fringes.Test(octet) {
		// Reconstruct the stride boundary prefix (e.g., /8, /16, /24).
		fringePfx, _ := ip.Prefix((depth + 1) << 3)

		// Ignore fringe if its prefix length is more specific than the query target.
		if fringePfx.Bits() <= pfxLen {
			if !yield(fringePfx) {
				return false
			}
		}
	}

	// Intersect configured prefixes with precomputed CBT ancestor mask.
	coverage := n.Prefixes.And(&lpm.LookupTbl[pfxIdx])

	// Iterate over matching bit indices in reverse order (longest to shortest prefix).
	for covIdx := range coverage.AllBackward() {

		// Reconstruct stride-relative length and combine with current depth offset.
		_, pfxLen := art.IdxToPfx(covIdx)
		cidr, _ := ip.Prefix(depth<<3 + int(pfxLen))

		if !yield(cidr) {
			return false
		}
	}

	return true
}

// YieldSubnets yields all routes, fringes, and subtrees covered by pfxIdx within the
// current node in canonical CIDR sort order.
//
// It first prunes local prefixes, fringes, and child subtrees by intersecting them
// with precomputed lookup tables (PfxRoutesLookupTbl and FringeRoutesLookupTbl)
// for pfxIdx.
//
// Local covered prefixes are sorted by rank. Local prefixes, fringes, and child subtrees
// are then interleaved in canonical CIDR order based on their byte boundaries (pfxOctet).
// Subtrees are traversed recursively to guarantee deterministic ordering across stride boundaries.
//
// Expects the node to be at the path location specified by octets/depth.
func (n *FastACLNode) YieldSubnets(ptx PathContext, yield func(netip.Prefix) bool) bool {
	// Collect matching local prefix indices within the target subtree slot.
	coveredIndices := n.Prefixes.And(&allot.PfxRoutesLookupTbl[ptx.Slot])
	allCoveredIndices := slices.Collect(coveredIndices.All())

	// Merge bitsets to identify relevant child nodes and fringe entries.
	fringeOrChild := n.Fringes.Or(&n.Children.BitSet256)
	coveredFringeOrChild := fringeOrChild.And(&allot.FringeRoutesLookupTbl[ptx.Slot])
	allCoveredAddrs := slices.Collect(coveredFringeOrChild.All())

	// Sort local prefix indices into canonical CIDR rank order.
	slices.SortFunc(allCoveredIndices, CmpIndexRank)

	// Cursor index tracking progression through allCoveredAddrs slice across iterations.
	addrCursor := 0

	// Interleave local prefixes, fringes, and child subtrees in canonical CIDR rank order.
	// Note: After the initial pruning phase above, this interleaving loop uses the exact same
	// ordering algorithm as AllRecSorted to guarantee deterministic CIDR sequence.
	for _, pfxIdx := range allCoveredIndices {
		pfxOctet, _ := art.IdxToPfx(pfxIdx)

		// Yield all child subtrees/fringes whose base byte address strictly precedes the current
		// prefix's target octet.
		for ; addrCursor < len(allCoveredAddrs); addrCursor++ {
			addr := allCoveredAddrs[addrCursor]
			if addr >= pfxOctet {
				break
			}

			ptx.Slot = addr
			if !n.yieldAddr(ptx, yield) {
				return false
			}
		}

		// Yield the local prefix once all preceding addrs have been traversed.
		ptx.Slot = pfxIdx
		if !n.yieldPrefix(ptx, yield) {
			return false
		}
	}

	// Yield any remaining fringe and child subtrees positioned after all local prefixes.
	for _, addr := range allCoveredAddrs[addrCursor:] {
		ptx.Slot = addr
		if !n.yieldAddr(ptx, yield) {
			return false
		}
	}

	return true
}

// UnionRec recursively merges another node o into the receiver node n in-place.
//
// The depth parameter represents the current 8-bit stride depth in the trie.
// Bitwise operations are used to merge local prefixes and fringe boundaries.
// Existing children are merged using handleMatrix, while missing children from o
// are directly linked into n.
//
// Returns the total number of duplicate prefixes overwritten during the subtree merge.
func (n *FastACLNode) UnionRec(o *FastACLNode, depth int) (duplicates int) {
	// 1. Calculate duplicate local prefixes and apply bitwise OR merge.
	dupBits := n.Prefixes.And(&o.Prefixes)
	duplicates += dupBits.OnesCount()

	// Bitwise union merge for local prefixes and ...
	n.Prefixes = n.Prefixes.Or(&o.Prefixes)

	// ... update prefix count cache
	//nolint:gosec // G115: integer overflow conversion int -> uint16
	n.prefixCount = uint16(n.Prefixes.OnesCount())

	// 2. Calculate duplicate fringe boundaries and apply bitwise OR merge.
	dupBits = n.Fringes.And(&o.Fringes)
	duplicates += dupBits.OnesCount()

	// Bitwise union merge for fringes; no fringe count cache to update
	n.Fringes = n.Fringes.Or(&o.Fringes)

	// 3. Iterate through all child pointers in the source node o.
	for i, addr := range o.Children.AllEnumerate() {
		otherChild := o.Children.Items[i]

		thisChild, thisExists := n.GetChild(addr)
		if !thisExists {
			// Fast path: If the slot is empty in n, directly insert the child pointer from o.
			n.InsertChild(addr, otherChild)
			continue
		}

		// Slot collision: Resolve combination matrix using handleMatrix.
		duplicates += n.handleMatrix(thisChild, otherChild, addr, depth)
	}

	return duplicates
}

// handleMatrix handles the four possible structural combinations when unioning
// two colliding child entries at a specific stride byte address and depth.
//
// Matrix of combinations:
//   - node + node: Recursive descent via UnionRec.
//   - node + leaf: Direct leaf insertion into the existing node.
//   - leaf + node: Creates a new internal node, pushes the existing leaf down, and recurses.
//   - leaf + leaf: Direct equality check (overwrites duplicate) or pushes both leaves into a new node.
//
// Returns the number of duplicate prefixes detected (1 if duplicate, 0 otherwise).
func (n *FastACLNode) handleMatrix(thisChild, otherChild any, addr uint8, depth int) int {
	// Perform type assertions upfront to reduce line noise and branch complexity.
	var (
		thisNode, thisIsNode = thisChild.(*FastACLNode)
		thisLeaf, thisIsLeaf = thisChild.(*CIDRLeaf)

		otherNode, otherIsNode = otherChild.(*FastACLNode)
		otherLeaf, otherIsLeaf = otherChild.(*CIDRLeaf)
	)

	// Case 1: Special case for leaf + leaf collision with identical prefix.
	if thisIsLeaf && otherIsLeaf && thisLeaf.prefix == otherLeaf.prefix {
		return 1
	}

	// Case 2: thisChild is already a full node; no new allocation needed at this level.
	if thisIsNode {
		switch {
		case otherIsNode:
			return thisNode.UnionRec(otherNode, depth+1)
		case otherIsLeaf:
			if thisNode.Insert(otherLeaf.prefix, depth+1) {
				return 1
			}
			return 0
		}
	}

	// Case 3: thisChild is a leaf; allocate a new internal node to push thisChild down.
	nc := new(FastACLNode)
	nc.Insert(thisLeaf.prefix, depth+1)

	// Replace the existing leaf child with the newly created node.
	n.InsertChild(addr, nc)

	// Process otherChild into the newly allocated internal node.
	switch {
	case otherIsNode:
		return nc.UnionRec(otherNode, depth+1)
	case otherIsLeaf:
		if nc.Insert(otherLeaf.prefix, depth+1) {
			return 1
		}
		return 0
	}

	return 0
}

/*
// UnionRec recursively merges another node o into the receiver node n.
//
// Returns the number of duplicate prefixes that were overwritten during merging.
func (n *FastACLNode) UnionRec(o *FastACLNode, depth int) (duplicates int) {
	dupBits := n.Prefixes.And(&o.Prefixes)
	duplicates += dupBits.OnesCount()

	n.Prefixes = n.Prefixes.Or(&o.Prefixes)
	n.prefixCount = uint16(n.Prefixes.OnesCount())

	dupBits = n.Fringes.And(&o.Fringes)
	duplicates += dupBits.OnesCount()

	n.Fringes = n.Fringes.Or(&o.Fringes)
	// no extra fringeCount tracked

	// for all child addrs in other node do ...
	for i, addr := range o.Children.AllEnumerate() {
		otherChild := o.Children.Items[i]

		thisChild, thisExists := n.GetChild(addr)
		if !thisExists {
			// just insert other child at this empty slot
			n.InsertChild(addr, otherChild)
			continue
		}

		// Use helper function to handle all 2x2 combinations
		duplicates += n.handleMatrix(thisChild, otherChild, addr, depth)
	}

	return duplicates
}

// handleMatrix, 4 possible combinations to union this child and other child
//
//	THIS,   OTHER:
//	--------------
//	node,   node    <-- union rec-descent with node
//	node,   leaf    <-- insert leaf at depth+1
//
//	leaf,   node    <-- insert new node, push this leaf down, union rec-descent
//	leaf,   leaf    <-- insert new node, push both leaves down (!first check equality)
func (n *FastACLNode) handleMatrix(thisChild, otherChild any, addr uint8, depth int) int {
	// Do ALL type assertions upfront - reduces line noise
	var (
		thisNode, thisIsNode = thisChild.(*FastACLNode)
		thisLeaf, thisIsLeaf = thisChild.(*CIDRLeaf)

		otherNode, otherIsNode = otherChild.(*FastACLNode)
		otherLeaf, otherIsLeaf = otherChild.(*CIDRLeaf)
	)

	// Case 1: Special cases that DON'T need a new node

	// Special case: leaf + leaf with same prefix -> just overwrite value
	if thisIsLeaf && otherIsLeaf && thisLeaf.prefix == otherLeaf.prefix {
		return 1
	}

	// Case 2: thisChild is already a node - insert into it, no new node needed
	if thisIsNode {
		switch {
		case otherIsNode:
			return thisNode.UnionRec(otherNode, depth+1)
		case otherIsLeaf:
			if thisNode.Insert(otherLeaf.prefix, depth+1) {
				return 1
			}
			return 0
		}
	}

	// Case 3: All remaining cases need a new node
	// (thisChild is leaf or fringe, and we didn't hit the special cases above)

	// Push existing child down into new node
	nc := new(FastACLNode)
	nc.Insert(thisLeaf.prefix, depth+1)

	// Replace child with new node
	n.InsertChild(addr, nc)

	// Now handle other child
	switch {
	case otherIsNode:
		return nc.UnionRec(otherNode, depth+1)
	case otherIsLeaf:
		if nc.Insert(otherLeaf.prefix, depth+1) {
			return 1
		}
		return 0
	}

	return 0
}
*/

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
			return kid.prefix.Overlaps(pfx)

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
		return nNode.OverlapsPrefixAtDepth(oLeaf.prefix, depth)
	case nIsNode && oIsFringe:
		return true

	// LEAF cases
	case nIsLeaf && oIsNode:
		return oNode.OverlapsPrefixAtDepth(nLeaf.prefix, depth)
	case nIsLeaf && oIsLeaf:
		return oLeaf.prefix.Overlaps(nLeaf.prefix)
	case nIsLeaf && oIsFringe:
		return true

	// FRINGE cases
	case nIsFringe:
		return true // fringe overlaps with everything

	default:
		panic("logic error, wrong node type combination")
	}
}

// PathContext encapsulates the active traversal state, stride history,
// and local node position required during recursive trie evaluation.
//
// It tracks path segments across stride boundaries while maintaining
// the active CBT (Complete Binary Tree) prefix index or octet address
// within local nodes.
type PathContext struct {
	// Path contains the accumulated octet path traversed through the trie.
	Path StridePath

	// Depth represents the current stride level or byte depth in the trie.
	Depth int

	// Slot stores the active local position, representing either a 1-based CBT prefix
	// index (1..255) or a 0-based stride octet address (0..255).
	Slot uint8

	// Is4 indicates whether the context evaluates an IPv4 (true) or IPv6 (false) address space.
	Is4 bool
}

func NewPathContext(octets []byte, depth int, slot uint8, is4 bool) PathContext {
	ptx := PathContext{
		Depth: depth,
		Slot:  slot,
		Is4:   is4,
	}
	if octets != nil {
		copy(ptx.Path[:], octets)
	}
	return ptx
}

// HierarchyItem represents a structural node or fringe boundary within
// the longest-prefix match (LPM) containment tree.
//
// It decouples the visual output representation from the internal trie traversal state:
//   - Cidr holds the reconstructed netip.Prefix for rendering.
//   - NextNode points to downstream subtrees (*FastACLNode or *CIDRLeaf)
//     or is nil if this item represents a terminal prefix.
//   - NextCtx holds the pre-computed PathContext for the next recursion step.
type HierarchyItem struct {
	CIDR     netip.Prefix
	NextNode any         // *FastACLNode or *CIDRLeaf (nil for terminal prefixes)
	NextCtx  PathContext // Pre-computed context for downstream traversal
}

// IsDirectlyCoveredBy checks whether idx is directly covered by parentIdx
// within this node's prefix table using CBT (Complete Binary Tree) ancestor tracking.
//
// To verify direct coverage, the method evaluates the longest-prefix match (LPM)
// of idx's immediate parent rather than idx itself. Since a lookup at
// idx would trivially match idx if present in n.Prefixes, shifting right
// by 1 bit (idx >> 1) bypasses the candidate itself to query its enclosing scope.
func (n *FastACLNode) IsDirectlyCoveredBy(idx, parentIdx uint8) bool {
	// Calculate immediate parent index in the CBT
	nextIdx := idx >> 1

	// Fast path: LPM match is mathematically impossible if ancestor index is smaller
	if nextIdx < parentIdx {
		return false
	}

	// Perform longest prefix match lookup for ancestor verification
	lpm, _ := n.LookupIdx(nextIdx)
	return lpm == parentIdx
}

// FprintRec recursively traverses the FastACL trie starting at node n,
// printing a formatted hierarchical ASCII tree of CIDRs to w.
//
// It collects, sorts, and prints immediate descendants for the current
// traversal state before descending recursively into subtrees.
func (n *FastACLNode) FprintRec(w io.Writer, ptx PathContext, pad string) error {
	// Guard clause: avoid processing empty nodes
	if n.IsEmpty() {
		return nil
	}

	// Retrieve all immediate children under parent and sort canonical by prefix
	directItems := n.DirectItems(ptx)
	slices.SortFunc(directItems, func(a, b HierarchyItem) int {
		return CmpPrefix(a.CIDR, b.CIDR)
	})

	lastIdx := len(directItems) - 1
	for i, item := range directItems {
		// Determine ASCII tree branch glyph based on position
		glyph := "├─ "
		space := "│  "
		if i == lastIdx {
			glyph = "└─ "
			space = "   "
		}

		// Print formatted prefix entry
		if _, err := fmt.Fprintf(w, "%s%s\n", pad+glyph, item.CIDR); err != nil {
			return err
		}

		// Recurse into downstream nodes or leaves
		if err := n.fprintNext(w, item, pad+space); err != nil {
			return err
		}
	}

	return nil
}

// fprintNext dispatches recursive tree printing using the NextCtx.
func (n *FastACLNode) fprintNext(w io.Writer, item HierarchyItem, pad string) error {
	switch next := item.NextNode.(type) {
	case *FastACLNode:
		return next.FprintRec(w, item.NextCtx, pad)
	case *CIDRLeaf:
		return next.Fprint(w, pad)
	default:
		return nil
	}
}

// Fprint renders a path-compressed terminal leaf under a fringe boundary.
func (c *CIDRLeaf) Fprint(w io.Writer, pad string) error {
	if c == nil {
		return nil
	}
	_, err := fmt.Fprintf(w, "%s└─ %s\n", pad, c.prefix)
	return err
}

// DirectItems returns all immediate descendant hierarchy items directly covered by parent.
//
// It inspects both local CBT prefixes and stride boundaries (fringes/children)
// to yield items directly beneath parent in the LPM hierarchy.
func (n *FastACLNode) DirectItems(ptx PathContext) []HierarchyItem {
	if n.IsEmpty() {
		return nil
	}

	// Pre-allocate slice capacity based on node statistics to eliminate dynamic re-allocations
	capacityHint := n.PrefixCount() + n.FringeCount() + n.ChildCount()
	items := make([]HierarchyItem, 0, capacityHint)

	// Collect matching elements across CBT prefixes and slot boundaries
	items = n.collectDirectPrefixes(ptx, items)
	items = n.collectDirectChildrenAndFringes(ptx, items)

	return items
}

// collectDirectPrefixes gathers local CBT prefixes directly covered by ptx.Idx.
//
// Iterates through active local bitset prefixes, evaluating direct coverage
// via IsDirectlyCoveredBy before constructing the output hierarchy item.
func (n *FastACLNode) collectDirectPrefixes(ptx PathContext, dst []HierarchyItem) []HierarchyItem {
	for idx := range n.Prefixes.All() {
		if !n.IsDirectlyCoveredBy(idx, ptx.Slot) {
			continue
		}

		// Preserve current path context, only update idx
		nextCtx := ptx
		nextCtx.Slot = idx

		dst = append(dst, HierarchyItem{
			NextNode: n, // nextNode is again this node
			NextCtx:  nextCtx,
			CIDR:     CidrFromPath(ptx.Path[:], ptx.Depth, ptx.Is4, idx),
		})
	}

	return dst
}

// collectDirectChildrenAndFringes scans slot boundaries within node n (combining n.Children
// and n.Fringes) to find subtrees, path-compressed terminal leaves, or fringe boundaries
// directly covered by parent.
//
// Slots are converted to host indices (via art.OctetToIdx) and evaluated against ptx.Idx.
// Matching slots are handed off to appendSlotItems to construct the respective HierarchyItem entries.
//
// Parameters:
//   - ptx: The PathContext defining target scope, CBT index, and stride depth.
//   - dst: The destination slice to accumulate matched items into, avoiding dynamic allocations.
//
// Returns the updated dst slice containing newly appended child and fringe items.
func (n *FastACLNode) collectDirectChildrenAndFringes(ptx PathContext, dst []HierarchyItem) []HierarchyItem {
	// Compute the bitwise union of active child slots and fringe slots.
	// This allows iterating over all relevant 256-ary stride boundaries in a single pass.
	allSlots := n.Children.Or(&n.Fringes)

	// Iterate over all active octet/slot addresses set in the bitset.
	for addr := range allSlots.All() {

		// Step 3: Map the byte-sized slot address to its corresponding host index within the CBT.
		hostIdx := art.OctetToIdx(addr)

		// Verify if this entire stride slot is directly covered by ptx.Idx.
		lpm, _ := n.LookupIdx(hostIdx)
		if lpm != ptx.Slot {
			continue
		}

		// Delegate item construction for the matching slot (fringe, child node, or leaf).
		nextPtx := ptx
		nextPtx.Slot = addr
		nextPtx.Path[ptx.Depth] = addr

		dst = n.appendSlotItems(nextPtx, dst)
	}

	return dst
}

// appendSlotItems evaluates a single octet slot address (addr) within node n
// and appends all directly covered hierarchy items to dst.
//
// A slot address represents a 256-ary branch boundary (e.g., /8, /16, /24 stride boundaries)
// within the multi-bit trie. This method handles three distinct structural states:
//
//  1. Fringe present (with optional child):
//     Constructs a HierarchyItem for the stride fringe. If a child node or leaf also resides
//     at this slot, item.NextNode is populated and nextCtx is calculated (advancing depth,
//     recording the path octet, and resetting the local CBT index to 0).
//
//  2. Child-only (*FastACLNode):
//     Occurs when no explicit fringe boundary exists at this slot, but a deeper subtree exists.
//     The method transparently hoists direct items from the child node upwards by delegating to
//     kid.DirectItems(nextParent), preserving the continuous LPM containment tree.
//
//  3. Child-only (*CIDRLeaf):
//     Appends a path-compressed terminal leaf representing a direct prefix match.
func (n *FastACLNode) appendSlotItems(ptx PathContext, dst []HierarchyItem) []HierarchyItem {
	// Query local bitsets to check if a fringe boundary or child pointer exists at slot addr.
	hasFringe := n.Fringes.Test(ptx.Slot)
	hasChild := n.Children.Test(ptx.Slot)

	switch {
	case hasFringe:
		// Case 1: Stride boundary fringe exists.
		item := HierarchyItem{CIDR: FringeCIDR(ptx)}

		if hasChild {

			// Step across the stride boundary:
			// - Record current slot octet into path history.
			// - Advance depth by 1.
			// - Reset local CBT bit index (Slot) to 0 for downstream traversal.
			nextCtx := ptx
			nextCtx.Depth++
			nextCtx.Slot = 0

			// Attach downstream child node or leaf under the fringe boundary.
			item.NextNode = n.MustGetChild(ptx.Slot)
			item.NextCtx = nextCtx
		}
		return append(dst, item)

	case hasChild:
		// Case 2 & 3: No local fringe boundary exists; evaluate the child slot directly.
		child := n.MustGetChild(ptx.Slot)

		switch kid := child.(type) {
		case *FastACLNode:
			// Case 2: Sub-node exists without a local fringe.
			// prepare rec-descent traversal
			nextPtx := ptx
			nextPtx.Depth++
			nextPtx.Slot = 0

			// Hoist direct items from the deeper sub-node into current scope.
			return append(dst, kid.DirectItems(nextPtx)...)

		case *CIDRLeaf:
			// Case 3: Path-compressed terminal leaf under an un-fringed slot.
			return append(dst, HierarchyItem{
				CIDR: kid.prefix,
			})

		default:
			panic("fastacl: unknown child node type in trie slot")
		}
	}

	return dst
}
