// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"iter"

	"github.com/gaissmai/bart/internal/lpm"
	"github.com/gaissmai/bart/internal/sparse"
)

// bartNode is a trie level bartNode in the multibit routing table.
//
// Each bartNode contains two conceptually different arrays:
//   - prefixes: representing routes, using a complete binary tree layout
//     driven by the baseIndex() function from the ART algorithm.
//   - children: holding subtries or path-compressed leaves/fringes with
//     a branching factor of 256 (8 bits per stride).
//
// Unlike the original ART, this implementation uses popcount-compressed sparse arrays
// instead of fixed-size arrays. Array slots are not pre-allocated; insertion
// and lookup rely on fast bitset operations and precomputed rank indexes.
//
// See doc/artlookup.pdf for the mapping mechanics and prefix tree details.
type bartNode[V any] struct {
	// prefixes stores routing entries (prefix -> value),
	// laid out as a complete binary tree using baseIndex().
	prefixes sparse.Array256[V]

	// children holds subnodes for the 256 possible next-hop paths
	// at this trie level (8-bit stride).
	//
	// Entries in children may be:
	//   - *bartNode[V]   -> internal child node for further traversal
	//   - *leafNode[V]   -> path-comp. node (depth < maxDepth - 1)
	//   - *fringeNode[V] -> path-comp. node (depth == maxDepth - 1, stride-aligned: /8, /16, ... /128)
	//
	// Note: Both *leafNode and *fringeNode entries are only created by path compression.
	// Prefixes that match exactly at the maximum trie depth (depth == maxDepth) are
	// never stored as children, but always directly in the prefixes array at that level.
	children sparse.Array256[any]
}

// isEmpty returns true if the node contains no routing entries (prefixes)
// and no child nodes. Empty nodes are candidates for compression or removal
// during trie optimization.
func (n *bartNode[V]) isEmpty() bool {
	if n == nil {
		return true
	}
	return n.prefixes.Len() == 0 && n.children.Len() == 0
}

// prefixCount returns the number of prefixes stored in this node.
func (n *bartNode[V]) prefixCount() int {
	return n.prefixes.Len()
}

// childCount returns the number of slots used in this node.
func (n *bartNode[V]) childCount() int {
	return n.children.Len()
}

// insertPrefix adds or updates a routing entry at the specified index with the given value.
// It returns true if a prefix already existed at that index (indicating an update),
// false if this is a new insertion.
func (n *bartNode[V]) insertPrefix(idx uint8, val V) (exists bool) {
	return n.prefixes.InsertAt(idx, val)
}

// getPrefix retrieves the value associated with the prefix at the given index.
// Returns the value and true if found, or zero value and false if not present.
func (n *bartNode[V]) getPrefix(idx uint8) (val V, exists bool) {
	return n.prefixes.Get(idx)
}

// getIndices returns a slice of all index positions that have prefixes stored in this node.
// The indices correspond to positions in the complete binary tree representation used
// for prefix storage within the 8-bit stride.
//
//nolint:unused // used via nodeReader interface
func (n *bartNode[V]) getIndices(buf *[256]uint8) []uint8 {
	return n.prefixes.AsSlice(buf)
}

// allIndices returns an iterator over all prefix entries.
// Each iteration yields the prefix index (uint8) and its associated value (V).
//
//nolint:unused // used via nodeReader interface
func (n *bartNode[V]) allIndices() iter.Seq2[uint8, V] {
	return func(yield func(uint8, V) bool) {
		var buf [256]uint8
		for _, idx := range n.prefixes.AsSlice(&buf) {
			val := n.mustGetPrefix(idx)
			if !yield(idx, val) {
				return
			}
		}
	}
}

// mustGetPrefix retrieves the value at the specified index, panicking if not found.
// This method should only be used when the caller is certain the index exists.
func (n *bartNode[V]) mustGetPrefix(idx uint8) (val V) {
	return n.prefixes.MustGet(idx)
}

// deletePrefix removes the prefix at the specified index.
// Returns true if the prefix existed, otherwise false.
func (n *bartNode[V]) deletePrefix(idx uint8) (exists bool) {
	_, exists = n.prefixes.DeleteAt(idx)
	return exists
}

// insertChild adds a child node at the specified address (0-255).
// The child can be a *bartNode[V], *leafNode[V], or *fringeNode[V].
// Returns true if a child already existed at that address.
func (n *bartNode[V]) insertChild(addr uint8, child any) (exists bool) {
	return n.children.InsertAt(addr, child)
}

// getChild retrieves the child node at the specified address.
// Returns the child and true if found, or nil and false if not present.
func (n *bartNode[V]) getChild(addr uint8) (any, bool) {
	return n.children.Get(addr)
}

// getChildAddrs returns a slice of all addresses (0-255) that have children in this node.
// This is useful for iterating over all child nodes without checking every possible address.
//
//nolint:unused // used via nodeReader interface
func (n *bartNode[V]) getChildAddrs(buf *[256]uint8) []uint8 {
	return n.children.AsSlice(buf)
}

// allChildren returns an iterator over all child nodes.
// Each iteration yields the child's address (uint8) and the child node (any).
//
//nolint:unused // used via nodeReader interface
func (n *bartNode[V]) allChildren() iter.Seq2[uint8, any] {
	return func(yield func(addr uint8, child any) bool) {
		var buf [256]uint8
		addrs := n.children.AsSlice(&buf)
		for i, addr := range addrs {
			child := n.children.Items[i]
			if !yield(addr, child) {
				return
			}
		}
	}
}

// mustGetChild retrieves the child at the specified address, panicking if not found.
// This method should only be used when the caller is certain the child exists.
func (n *bartNode[V]) mustGetChild(addr uint8) any {
	return n.children.MustGet(addr)
}

// deleteChild removes the child node at the specified address.
// This operation is idempotent - removing a non-existent child is safe.
func (n *bartNode[V]) deleteChild(addr uint8) (exists bool) {
	_, exists = n.children.DeleteAt(addr)
	return exists
}

// contains returns true if an index (idx) has any matching longest-prefix
// in the current node’s prefix table.
//
// This function performs a presence check without retrieving the associated value.
// It is faster than a full lookup, as it only tests for intersection with the
// backtracking bitset for the given index.
//
// The prefix table is structured as a complete binary tree (CBT), and LPM testing
// is done via a bitset operation that maps the traversal path from the given index
// toward its possible ancestors.
func (n *bartNode[V]) contains(idx uint8) bool {
	return n.prefixes.Intersects(&lpm.LookupTbl[idx])
}

// lookupIdx performs a longest-prefix match (LPM) lookup for the given index (idx)
// within the 8-bit stride-based prefix table at this trie depth.
//
// The function returns the matched base index, associated value, and true if a
// matching prefix exists at this level; otherwise, ok is false.
//
// Internally, the prefix table is organized as a complete binary tree (CBT) indexed
// via the baseIndex function. Unlike the original ART algorithm, this implementation
// does not use an allotment-based approach. Instead, it performs CBT backtracking
// using a bitset-based operation with a precomputed backtracking pattern specific to idx.
func (n *bartNode[V]) lookupIdx(idx uint8) (top uint8, val V, ok bool) {
	// top is the idx of the longest-prefix-match
	if top, ok = n.prefixes.IntersectionTop(&lpm.LookupTbl[idx]); ok {
		return top, n.mustGetPrefix(top), true
	}
	return top, val, ok
}

// lookup is just a simple wrapper for lookupIdx.
func (n *bartNode[V]) lookup(idx uint8) (val V, ok bool) {
	_, val, ok = n.lookupIdx(idx)
	return val, ok
}
