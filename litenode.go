// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"iter"

	"github.com/gaissmai/bart/internal/bitset"
	"github.com/gaissmai/bart/internal/lpm"
	"github.com/gaissmai/bart/internal/sparse"
)

// liteNode is the core building block of the slimmed-down BART trie.
//
// Each liteNode represents one stride (8 bits) of the address space and stores
// both routing prefixes and child pointers for further trie traversal. It is
// designed as a memory-efficient alternative to classic ART-style nodes,
// using compact bitsets and sparse arrays instead of full lookup tables.
//
// A liteNode has two main responsibilities:
//   - **Prefix storage**: Up to 256 possible prefixes (one per stride index) are
//     managed in a BitSet (prefixes). Lookups use longest-prefix match (LPM)
//     via backtracking along the complete binary tree (CBT) encoded in this bitset.
//   - **Child management**: Child pointers are held in a sparse-array of at most
//     256 entries. A child can be another *liteNode[V] for further traversal, or
//     a path-compressed terminal node: *leafNode (explicit prefix storage)
//     or *fringeNode (implicit prefix at stride boundary).
//
// Fields:
//   - prefixes: BitSet256 indicating which prefix indices are occupied.
//   - children: Sparse array holding subnodes or compressed leaf/fringe nodes.
//   - pfxCount: Number of prefix entries actually stored in this node.
//
// Invariants:
//   - pfxCount always matches the number of set bits in prefixes.
//   - children only contains entries at addresses (0–255) explicitly present.
//   - Node emptiness (no prefixes and no children) implies a candidate for removal.
//
// Generic design note:
//
//	liteNode is *pseudo-generic*: the type parameter V does not occur in the
//	struct fields itself. Instead, it is a **phantom type** used solely to make
//	liteNode[V] satisfy the generic interface nodeReadWriter[V].
//	This allows liteNode, fastNode, and node to be interchangeable under the
//	same interface abstraction, enabling generic algorithms for insertion,
//	lookup, dumping, and traversal, regardless of the internal representation.
//	The compiler enforces type correctness at the interface boundary, while
//	the internal layout of liteNode stays lean (no value payloads).
//
// Memory model:
//   - Prefix presence is tracked only via bitset (values are not stored directly).
//   - No values are stored; lite tracks presence only.
//   - liteNode acts solely as the internal routing structure.
//
// Usage notes:
//   - Routing insertions place prefixes either into the prefix table (if aligned)
//     or into compressed child nodes (leaf/fringe).
//   - Lookup/contains use the precomputed CBT-backtracking bitset (lpm.LookupTbl)
//     for fast longest-prefix match within stride.
//   - purgeAndCompress reclaims empty / sparse nodes on unwind to keep the trie compact.
type liteNode[V any] struct {
	children sparse.Array256[any]
	prefixes struct {
		bitset.BitSet256
		// no values
		count uint16
	}
}

// isEmpty returns true if the node contains no routing entries (prefixes)
// and no child nodes. Empty nodes are candidates for compression or removal
// during trie optimization.
func (n *liteNode[V]) isEmpty() bool {
	if n == nil {
		return true
	}
	return n.prefixes.count == 0 && n.children.Len() == 0
}

// prefixCount returns the number of prefixes stored in this node.
func (n *liteNode[V]) prefixCount() int {
	return int(n.prefixes.count)
}

// childCount returns the number of slots used in this node.
func (n *liteNode[V]) childCount() int {
	return n.children.Len()
}

// insertPrefix adds a routing entry at the specified index.
// It returns true if a prefix already existed at that index (indicating an update),
// false if this is a new insertion.
func (n *liteNode[V]) insertPrefix(idx uint8, _ V) (exists bool) {
	if exists = n.prefixes.Test(idx); exists {
		return exists
	}
	n.prefixes.Set(idx)
	n.prefixes.count++
	return exists
}

// prefix is set at the given index.
//
//nolint:unparam
func (n *liteNode[V]) getPrefix(idx uint8) (_ V, exists bool) {
	exists = n.prefixes.Test(idx)
	return
}

//nolint:unparam
func (n *liteNode[V]) mustGetPrefix(idx uint8) (_ V) {
	return
}

// getIndices returns a slice of all index positions that have prefixes stored in this node.
// The indices correspond to positions in the complete binary tree representation used
// for prefix storage within the 8-bit stride.
//
//nolint:unused // used via nodeReader interface
func (n *liteNode[V]) getIndices(buf *[256]uint8) []uint8 {
	return n.prefixes.AsSlice(buf)
}

// allIndices returns an iterator over all prefix entries.
// Each iteration yields the prefix index (uint8) and its associated value (V).
//
//nolint:unused // used via nodeReader interface
func (n *liteNode[V]) allIndices() iter.Seq2[uint8, V] {
	var zero V
	return func(yield func(uint8, V) bool) {
		var buf [256]uint8
		for _, idx := range n.prefixes.AsSlice(&buf) {
			if !yield(idx, zero) {
				return
			}
		}
	}
}

// deletePrefix removes the prefix at the specified index.
// Returns true if the prefix existed, and false otherwise.
//
//nolint:unparam
func (n *liteNode[V]) deletePrefix(idx uint8) (_ V, exists bool) {
	if exists = n.prefixes.Test(idx); !exists {
		return
	}
	n.prefixes.Clear(idx)
	n.prefixes.count--
	return
}

// insertChild adds a child node at the specified address (0-255).
// The child can be a *liteNode[V], *leafNode, or *fringeNode.
// Returns true if a child already existed at that address.
func (n *liteNode[V]) insertChild(addr uint8, child any) (exists bool) {
	return n.children.InsertAt(addr, child)
}

// getChild retrieves the child node at the specified address.
// Returns the child and true if found, or nil and false if not present.
func (n *liteNode[V]) getChild(addr uint8) (any, bool) {
	return n.children.Get(addr)
}

// getChildAddrs returns a slice of all addresses (0-255) that have children in this node.
// This is useful for iterating over all child nodes without checking every possible address.
//
//nolint:unused // used via nodeReader interface
func (n *liteNode[V]) getChildAddrs(buf *[256]uint8) []uint8 {
	return n.children.AsSlice(buf)
}

// allChildren returns an iterator over all child nodes.
// Each iteration yields the child's address (uint8) and the child node (any).
//
//nolint:unused // used via nodeReader interface
func (n *liteNode[V]) allChildren() iter.Seq2[uint8, any] {
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
func (n *liteNode[V]) mustGetChild(addr uint8) any {
	return n.children.MustGet(addr)
}

// deleteChild removes the child node at the specified address.
// This operation is idempotent - removing a non-existent child is safe.
func (n *liteNode[V]) deleteChild(addr uint8) (exists bool) {
	_, exists = n.children.DeleteAt(addr)
	return exists
}

// contains returns true if an index (idx) has any matching longest-prefix
// in the current node’s prefix table.
//
// This function performs a presence check.
//
// The prefix table is structured as a complete binary tree (CBT), and LPM testing
// is done via a bitset operation that maps the traversal path from the given index
// toward its possible ancestors.
func (n *liteNode[V]) contains(idx uint8) bool {
	return n.prefixes.Intersects(&lpm.LookupTbl[idx])
}

// lookupIdx performs a longest-prefix match (LPM) lookup for the given index (idx)
// within the 8-bit stride-based prefix table at this trie depth.
//
// The function returns the matched index and whether a matching prefix
// exists at this level. The value type parameter exists only to satisfy interfaces.
//
// Internally, the prefix table is organized as a complete binary tree (CBT) indexed
// via the baseIndex function. Unlike the original ART algorithm, this implementation
// does not use an allotment-based approach. Instead, it performs CBT backtracking
// using a bitset-based operation with a precomputed backtracking pattern specific to idx.
//
//nolint:unparam,unused // used via nodeReader interface
func (n *liteNode[V]) lookupIdx(idx uint8) (top uint8, _ V, ok bool) {
	top, ok = n.prefixes.IntersectionTop(&lpm.LookupTbl[idx])
	return
}

// lookup is just a simple wrapper for lookupIdx.
//
//nolint:unparam,unused // used via nodeReader interface
func (n *liteNode[V]) lookup(idx uint8) (_ V, ok bool) {
	_, _, ok = n.lookupIdx(idx)
	return
}
