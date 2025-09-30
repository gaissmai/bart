// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"iter"

	"github.com/gaissmai/bart/internal/bitset"
	"github.com/gaissmai/bart/internal/lpm"
)

// fastNode is a trie level node in the multibit routing table.
//
// Each fastNode contains two conceptually different fixed sized arrays:
//   - prefixes: representing routes, using a complete binary tree layout
//     driven by the baseIndex() function from the ART algorithm.
//   - children: holding subtries or path-compressed leaves or fringes.
//
// See doc/artlookup.pdf for the mapping mechanics and prefix tree details.
type fastNode[V any] struct {
	prefixes struct {
		bitset.BitSet256
		items [256]*V
	}

	// children.items: **fastNode or path-compressed **leaf- or **fringeNode
	// an array of "pointers to" the empty interface,
	// and not an array of empty interfaces.
	//
	// - any  ( interface{}) takes 2 words, even if nil.
	// - *any (*interface{}) requires only 1 word when nil.
	//
	// Since many slots are nil, this reduces memory by 30%.
	// The added indirection does not have a measurable performance impact,
	// but makes the code uglier.
	children struct {
		bitset.BitSet256
		items [256]*any
	}

	// pfxCount is an O(1) counter tracking the number of prefixes in this node.
	// This replaces expensive prefixesBitSet.Size() calls with direct counter access.
	// Automatically maintained during insertPrefix() and deletePrefix() operations.
	pfxCount uint16

	// cldCount is an O(1) counter tracking the number of child nodes in this node.
	// This replaces expensive childrenBitSet.Size() calls with direct counter access.
	// Automatically maintained during insertChild() and deleteChild() operations.
	cldCount uint16
}

// prefixCount returns the number of prefixes stored in this node.
func (n *fastNode[V]) prefixCount() int {
	return int(n.pfxCount)
}

// childCount returns the number of slots used in this node.
func (n *fastNode[V]) childCount() int {
	return int(n.cldCount)
}

// isEmpty returns true if node has neither prefixes nor children
func (n *fastNode[V]) isEmpty() bool {
	if n == nil {
		return true
	}
	return n.pfxCount == 0 && n.cldCount == 0
}

// getChild returns the child node at the specified address and true if it exists.
// If no child exists at addr, returns nil and false.
func (n *fastNode[V]) getChild(addr uint8) (any, bool) {
	if anyPtr := n.children.items[addr]; anyPtr != nil {
		return *anyPtr, true
	}
	return nil, false
}

// mustGetChild returns the child node at the specified address.
// Panics if no child exists at addr. This method should only be called
// when the caller has verified the child exists.
func (n *fastNode[V]) mustGetChild(addr uint8) any {
	// panics if n.children[addr] is nil
	return *n.children.items[addr]
}

// getChildAddrs returns a slice containing all addresses that have child nodes.
// The addresses are returned in ascending order.
func (n *fastNode[V]) getChildAddrs(buf *[256]uint8) []uint8 {
	return n.children.AsSlice(buf)
}

// allChildren returns an iterator over all child nodes.
// Each iteration yields the child's address (uint8) and the child node (any).
func (n *fastNode[V]) allChildren() iter.Seq2[uint8, any] {
	return func(yield func(addr uint8, child any) bool) {
		var buf [256]uint8
		for _, addr := range n.children.AsSlice(&buf) {
			child := *n.children.items[addr]
			if !yield(addr, child) {
				return
			}
		}
	}
}

// insertChild inserts a child node at the specified address.
// Returns true if a child already existed at addr (overwrite case),
// false if this is a new insertion.
func (n *fastNode[V]) insertChild(addr uint8, child any) (exists bool) {
	if p := n.children.items[addr]; p != nil {
		// Reuse existing *any slot to cut allocations and GC churn
		*p = child // overwrite
		return true
	}

	n.children.Set(addr)
	n.cldCount++

	// pointer to any reduces per-slot memory for nil entries versus storing `any` directly.
	p := new(any)
	*p = child
	n.children.items[addr] = p

	return false
}

// deleteChild removes the child node at the specified address.
// This operation is idempotent - removing a non-existent child is safe.
func (n *fastNode[V]) deleteChild(addr uint8) (exists bool) {
	if n.children.items[addr] == nil {
		return false
	}
	n.cldCount--

	n.children.Clear(addr)
	n.children.items[addr] = nil
	return true
}

// insertPrefix adds or updates a routing entry at the specified index with the given value.
// It returns true if a prefix already existed at that index (indicating an update),
// false if this is a new insertion.
func (n *fastNode[V]) insertPrefix(idx uint8, val V) (exists bool) {
	if exists = n.prefixes.Test(idx); !exists {
		n.prefixes.Set(idx)
		n.pfxCount++
	}

	// insert or update

	// To ensure allot works as intended, every unique prefix in the
	// fastNode must point to a distinct value pointer, even for identical values.
	// Using new() and assignment guarantees each inserted prefix gets its own address,
	valPtr := new(V)
	*valPtr = val

	oldValPtr := n.prefixes.items[idx]

	// overwrite oldValPtr with valPtr
	n.allot(idx, oldValPtr, valPtr)

	return exists
}

// getPrefix returns the value for the given prefix index and true if it exists.
// If no prefix exists at idx, returns the zero value and false.
func (n *fastNode[V]) getPrefix(idx uint8) (val V, exists bool) {
	if exists = n.prefixes.Test(idx); exists {
		val = *n.prefixes.items[idx]
	}
	return val, exists
}

// mustGetPrefix returns the value for the given prefix index.
// Panics if no prefix exists at idx. This method should only be called
// when the caller has verified the prefix exists.
func (n *fastNode[V]) mustGetPrefix(idx uint8) V {
	return *n.prefixes.items[idx]
}

// getIndices returns a slice containing all prefix indices that have values stored.
// The indices are returned in ascending order.
func (n *fastNode[V]) getIndices(buf *[256]uint8) []uint8 {
	return n.prefixes.AsSlice(buf)
}

// allIndices returns an iterator over all prefix entries.
// Each iteration yields the prefix index (uint8) and its associated value (V).
func (n *fastNode[V]) allIndices() iter.Seq2[uint8, V] {
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

// deletePrefix removes the route at the given index.
// Returns true if the prefix existed, otherwise false.
func (n *fastNode[V]) deletePrefix(idx uint8) (exists bool) {
	if exists = n.prefixes.Test(idx); !exists {
		// Route entry doesn't exist
		return exists
	}
	n.pfxCount--

	valPtr := n.prefixes.items[idx]
	parentValPtr := n.prefixes.items[idx>>1]

	// delete -> overwrite valPtr with parentValPtr
	n.allot(idx, valPtr, parentValPtr)

	n.prefixes.Clear(idx)
	return true
}

// contains returns true if the given index has any matching longest-prefix
// in the current node's prefix table.
//
// This function performs a presence check using the ART algorithm's
// hierarchical prefix structure. It tests whether any ancestor prefix
// exists for the given index by probing the slot at idx (children inherit
// ancestor pointers via allot).
func (n *fastNode[V]) contains(idx uint8) (ok bool) {
	return n.prefixes.items[idx] != nil
}

// lookup performs a longest-prefix match (LPM) lookup for the given index
// within the current node's prefix table in O(1).
//
// The function returns the matched value and true if a matching prefix exists;
// otherwise, it returns the zero value and false. The lookup uses the ART
// algorithm's hierarchical structure to find the most specific
// matching prefix.
func (n *fastNode[V]) lookup(idx uint8) (val V, ok bool) {
	if valPtr := n.prefixes.items[idx]; valPtr != nil {
		return *valPtr, true
	}
	return val, ok
}

// lookupIdx performs a longest-prefix match (LPM) lookup for the given index (idx)
// within the 8-bit stride-based prefix table at this trie depth.
//
// The function returns the matched base index, associated value, and true if a
// matching prefix exists at this level; otherwise, ok is false.
//
// Its semantics are identical to [node.lookupIdx].
func (n *fastNode[V]) lookupIdx(idx uint8) (top uint8, val V, ok bool) {
	// top is the idx of the longest-prefix-match
	if top, ok = n.prefixes.IntersectionTop(&lpm.LookupTbl[idx]); ok {
		return top, *n.prefixes.items[top], true
	}
	return top, val, ok
}

// allot updates entries whose stored valPtr matches oldValPtr, in the
// subtree rooted at idx. Matching entries have their stored oldValPtr set to
// valPtr, and their value set to val.
//
// allot is the core of the ART algorithm, enabling efficient insertion/deletion
// while preserving very fast lookups.
//
// See doc/artlookup.pdf for the mapping mechanics and prefix tree details.
//
// Example of (uninterrupted) allotment sequence:
//
//	addr/bits: 0/5 -> {0/5, 0/6, 4/6, 0/7, 2/7, 4/7, 6/7}
//	                    ╭────╮╭─────────┬────╮
//	       idx: 32 ->  32    64   65   128  129 130  131
//	                    ╰─────────╯╰─────────────┴────╯
//
// Using an iterative form ensures better inlining opportunities.
func (n *fastNode[V]) allot(idx uint8, oldValPtr, valPtr *V) {
	// iteration with stack instead of recursion
	stack := make([]uint8, 0, 256)

	// start idx
	stack = append(stack, idx)

	for i := 0; i < len(stack); i++ {
		idx = stack[i]

		// stop this allot path, idx already points to a more specific route.
		if n.prefixes.items[idx] != oldValPtr {
			continue // take next path from stack
		}

		// overwrite
		n.prefixes.items[idx] = valPtr

		// max idx is 255, so stop the duplication at 128 and above
		if idx >= 128 {
			continue
		}

		// child nodes, it's a complete binary tree
		// left:  idx*2
		// right: (idx*2)+1
		stack = append(stack, idx<<1, (idx<<1)+1)
	}
}
