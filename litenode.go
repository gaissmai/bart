// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"iter"
	"net/netip"

	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/bitset"
	"github.com/gaissmai/bart/internal/lpm"
	"github.com/gaissmai/bart/internal/sparse"
)

type liteNode[V any] struct {
	prefixes bitset.BitSet256
	children sparse.Array256[any]
	pfxCount uint16
}

// isEmpty returns true if the node contains no routing entries (prefixes)
// and no child nodes. Empty nodes are candidates for compression or removal
// during trie optimization.
func (n *liteNode[V]) isEmpty() bool {
	if n == nil {
		return true
	}
	return n.pfxCount == 0 && n.children.Len() == 0
}

// prefixCount returns the number of prefixes stored in this node.
func (n *liteNode[V]) prefixCount() int {
	return int(n.pfxCount)
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
		return
	}
	n.prefixes.Set(idx)
	n.pfxCount++
	return
}

// prefix is set at the given index.
func (n *liteNode[V]) getPrefix(idx uint8) (val V, exists bool) {
	exists = n.prefixes.Test(idx)
	return
}

func (n *liteNode[V]) mustGetPrefix(idx uint8) (val V) {
	return
}

// getIndices returns a slice of all index positions that have prefixes stored in this node.
// The indices correspond to positions in the complete binary tree representation used
// for prefix storage within the 8-bit stride.
//
//nolint:unused
func (n *liteNode[V]) getIndices() []uint8 {
	return n.prefixes.AsSlice(&[256]uint8{})
}

// allIndices returns an iterator over all prefix entries.
// Each iteration yields the prefix index (uint8) and its associated value (V).
//
//nolint:unused
func (n *liteNode[V]) allIndices() iter.Seq2[uint8, V] {
	var zero V
	return func(yield func(uint8, V) bool) {
		for _, idx := range n.prefixes.AsSlice(&[256]uint8{}) {
			if !yield(idx, zero) {
				return
			}
		}
	}
}

// deletePrefix removes the prefix at the specified index and returns its value.
// Returns the deleted value and true if the prefix existed, or zero value and false otherwise.
func (n *liteNode[V]) deletePrefix(idx uint8) (val V, exists bool) {
	if exists = n.prefixes.Test(idx); !exists {
		return
	}
	n.prefixes.Clear(idx)
	n.pfxCount--
	return
}

// insertChild adds a child node at the specified address (0-255).
// The child can be a *liteNode, *leafNode[V], or *fringeNode[V].
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
//nolint:unused
func (n *liteNode[V]) getChildAddrs() []uint8 {
	return n.children.AsSlice(&[256]uint8{})
}

// allChildren returns an iterator over all child nodes.
// Each iteration yields the child's address (uint8) and the child node (any).
//
//nolint:unused
func (n *liteNode[V]) allChildren() iter.Seq2[uint8, any] {
	return func(yield func(addr uint8, child any) bool) {
		addrs := n.children.AsSlice(&[256]uint8{})
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
	return
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
func (n *liteNode[V]) contains(idx uint8) bool {
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
func (n *liteNode[V]) lookupIdx(idx uint8) (baseIdx uint8, val V, ok bool) {
	if top, ok := n.prefixes.IntersectionTop(&lpm.LookupTbl[idx]); ok {
		return top, val, true
	}
	return
}

// lookup is just a simple wrapper for lookupIdx.
func (n *liteNode[V]) lookup(idx uint8) (val V, ok bool) {
	_, _, ok = n.lookupIdx(idx)
	return
}

// leafNode represents a path-compressed routing entry that stores both prefix and value.
// Leaf nodes are used when a prefix doesn't align with trie stride boundaries
// and needs to be stored as a compressed path to save memory.
type liteLeafNode struct {
	prefix netip.Prefix
}

// newLeafNode creates a new leaf node with the specified prefix and value.
func newLiteLeafNode(pfx netip.Prefix) *liteLeafNode {
	return &liteLeafNode{prefix: pfx}
}

// fringeNode represents a path-compressed routing entry that stores only a value.
// The prefix is implicitly defined by the node's position in the trie.
// Fringe nodes are used for prefixes that align exactly with stride boundaries
// (/8, /16, /24, etc.) to save memory by not storing redundant prefix information.
type liteFringeNode struct{}

// newFringeNode creates a new fringe node with the specified value.
func newLiteFringeNode() *liteFringeNode {
	return new(liteFringeNode)
}

// insertAtDepth inserts a network prefix and its associated value into the
// trie starting at the specified byte depth.
//
// The function traverses the prefix address from the given depth and inserts
// the value either directly into the node's prefix table or as a compressed
// leaf or fringe node. If a conflicting leaf or fringe exists, it creates
// a new intermediate node to accommodate both entries.
//
// Parameters:
//   - pfx: The network prefix to insert (must be in canonical form)
//   - val: The value to associate with the prefix
//   - depth: The current depth in the trie (0-based byte index)
//
// Returns true if a prefix already existed and was updated, false for new insertions.
func (n *liteNode[V]) insertAtDepth(pfx netip.Prefix, depth int) (exists bool) {
	var zero V
	ip := pfx.Addr() // the pfx must be in canonical form
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)

	// find the proper trie node to insert prefix
	// start with prefix octet at depth
	for _, octet := range octets[depth:] {
		// last masked octet: insert/override prefix/val into node
		if depth == lastOctetPlusOne {
			return n.insertPrefix(art.PfxToIdx(octet, lastBits), zero)
		}

		// reached end of trie path ...
		if !n.children.Test(octet) {
			// insert prefix path compressed as leaf or fringe
			if isFringe(depth, pfx) {
				return n.insertChild(octet, newLiteFringeNode())
			}
			return n.insertChild(octet, newLiteLeafNode(pfx))
		}

		// ... or descend down the trie
		kid := n.mustGetChild(octet)

		// kid is node or leaf at addr
		switch kid := kid.(type) {
		case *liteNode[V]:
			n = kid // descend down to next trie level

		case *liteLeafNode:
			// reached a path compressed prefix
			if kid.prefix == pfx {
				// exists
				return true
			}

			// create new node
			// push the leaf down
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(liteNode[V])
			newNode.insertAtDepth(kid.prefix, depth+1)

			n.insertChild(octet, newNode)
			n = newNode

		case *liteFringeNode:
			// reached a path compressed fringe
			if isFringe(depth, pfx) {
				// exists
				return true
			}

			// create new node
			// push the fringe down, it becomes a default route (idx=1)
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(liteNode[V])
			newNode.insertPrefix(1, zero)

			n.insertChild(octet, newNode)
			n = newNode

		default:
			panic("logic error, wrong node type")
		}

		depth++
	}

	panic("unreachable")
}

// purgeAndCompress performs bottom-up compression of the trie by removing
// empty nodes and converting sparse branches into compressed leaf/fringe nodes.
//
// The function unwinds the provided stack of parent nodes, checking each level
// for compression opportunities based on child count and prefix distribution.
// It may convert:
//   - Nodes with a single prefix into leafNode (path compression)
//   - Nodes at lastOctet with a single prefix into fringeNode
//   - Empty intermediate nodes are removed entirely
//
// Parameters:
//   - stack: Array of parent nodes to process during unwinding
//   - octets: The path of octets taken to reach the current position
//   - is4: True for IPv4 processing, false for IPv6
func (n *liteNode[V]) purgeAndCompress(stack []*liteNode[V], octets []uint8, is4 bool) {
	// unwind the stack
	for depth := len(stack) - 1; depth >= 0; depth-- {
		parent := stack[depth]
		octet := octets[depth]

		pfxCount := n.prefixCount()
		childCount := n.childCount()

		switch {
		case pfxCount == 0 && childCount == 0:
			// just delete this empty node from parent
			parent.deleteChild(octet)

		case pfxCount == 0 && childCount == 1:
			switch kid := n.children.Items[0].(type) {
			case *liteNode[V]:
				// fast exit, we are at an intermediate path node
				// no further delete/compress upwards the stack is possible
				return
			case *liteLeafNode:
				// just one leaf, delete this node and reinsert the leaf above
				parent.deleteChild(octet)

				// ... (re)insert the leaf at parents depth
				parent.insertAtDepth(kid.prefix, depth)
			case *liteFringeNode:
				// just one fringe, delete this node and reinsert the fringe as leaf above
				parent.deleteChild(octet)

				// get the last octet back, the only item is also the first item
				lastOctet, _ := n.children.FirstSet()

				// rebuild the prefix with octets, depth, ip version and addr
				// depth is the parent's depth, so add +1 here for the kid
				fringePfx := cidrForFringe(octets, depth+1, is4, lastOctet)

				// ... (re)reinsert prefix/value at parents depth
				parent.insertAtDepth(fringePfx, depth)
			}

		case pfxCount == 1 && childCount == 0:
			// just one prefix, delete this node and reinsert the idx as leaf above
			parent.deleteChild(octet)

			// get prefix back from idx ...
			idx, _ := n.prefixes.FirstSet() // single idx must be first bit set

			// ... and octet path
			path := stridePath{}
			copy(path[:], octets)

			// depth is the parent's depth, so add +1 here for the kid
			pfx := cidrFromPath(path, depth+1, is4, idx)

			// ... (re)insert prefix/value at parents depth
			parent.insertAtDepth(pfx, depth)
		}

		// climb up the stack
		n = parent
	}
}
