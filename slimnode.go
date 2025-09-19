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

// slimNode is the core building block of the slimmed-down BART trie.
//
// Each slimNode represents one stride (8 bits) of the address space and stores
// both routing prefixes and child pointers for further trie traversal. It is
// designed as a memory-efficient alternative to classic ART-style nodes,
// using compact bitsets and sparse arrays instead of full lookup tables.
//
// A slimNode has two main responsibilities:
//   - **Prefix storage**: Up to 256 possible prefixes (one per stride index) are
//     managed in a BitSet (prefixes). Lookups use longest-prefix match (LPM)
//     via backtracking along the complete binary tree (CBT) encoded in this bitset.
//   - **Child management**: Child pointers are held in a sparse-array of at most
//     256 entries. A child can be another *slimNode[V] for further traversal, or
//     a path-compressed terminal node: *slimLeafNode (explicit prefix storage)
//     or *slimFringeNode (implicit prefix at stride boundary).
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
//	slimNode is *pseudo-generic*: the type parameter V does not occur in the
//	struct fields itself. Instead, it is a **phantom type** used solely to make
//	slimNode[V] satisfy the generic interface nodeReadWriter[V].
//	This allows slimNode, fastNode, and node to be interchangeable under the
//	same interface abstraction, enabling generic algorithms for insertion,
//	lookup, dumping, and traversal, regardless of the internal representation.
//	The compiler enforces type correctness at the interface boundary, while
//	the internal layout of slimNode stays lean (no value payloads).
//
// Memory model:
//   - Prefix presence is tracked only via bitset (values are not stored directly).
//   - No values are stored; Slim tracks presence only.
//   - slimNode acts solely as the internal routing structure.
//
// Usage notes:
//   - Routing insertions place prefixes either into the prefix table (if aligned)
//     or into compressed child nodes (leaf/fringe).
//   - Lookup/contains use the precomputed CBT-backtracking bitset (lpm.LookupTbl)
//     for fast longest-prefix match within stride.
//   - purgeAndCompress reclaims empty / sparse nodes on unwind to keep the trie compact.
type slimNode[V any] struct {
	prefixes bitset.BitSet256
	children sparse.Array256[any]
	pfxCount uint16
}

// isEmpty returns true if the node contains no routing entries (prefixes)
// and no child nodes. Empty nodes are candidates for compression or removal
// during trie optimization.
//
//nolint:unused
func (n *slimNode[V]) isEmpty() bool {
	if n == nil {
		return true
	}
	return n.pfxCount == 0 && n.children.Len() == 0
}

// prefixCount returns the number of prefixes stored in this node.
//
//nolint:unused
func (n *slimNode[V]) prefixCount() int {
	return int(n.pfxCount)
}

// childCount returns the number of slots used in this node.
func (n *slimNode[V]) childCount() int {
	return n.children.Len()
}

// insertPrefix adds a routing entry at the specified index.
// It returns true if a prefix already existed at that index (indicating an update),
// false if this is a new insertion.
func (n *slimNode[V]) insertPrefix(idx uint8, _ V) (exists bool) {
	if exists = n.prefixes.Test(idx); exists {
		return
	}
	n.prefixes.Set(idx)
	n.pfxCount++
	return
}

// prefix is set at the given index.
//
//nolint:unused
func (n *slimNode[V]) getPrefix(idx uint8) (_ V, exists bool) {
	exists = n.prefixes.Test(idx)
	return
}

//nolint:unused
func (n *slimNode[V]) mustGetPrefix(idx uint8) (_ V) {
	return
}

// getIndices returns a slice of all index positions that have prefixes stored in this node.
// The indices correspond to positions in the complete binary tree representation used
// for prefix storage within the 8-bit stride.
//
//nolint:unused
func (n *slimNode[V]) getIndices() []uint8 {
	return n.prefixes.AsSlice(&[256]uint8{})
}

// allIndices returns an iterator over all prefix entries.
// Each iteration yields the prefix index (uint8) and its associated value (V).
//
//nolint:unused
func (n *slimNode[V]) allIndices() iter.Seq2[uint8, V] {
	var zero V
	return func(yield func(uint8, V) bool) {
		for _, idx := range n.prefixes.AsSlice(&[256]uint8{}) {
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
func (n *slimNode[V]) deletePrefix(idx uint8) (_ V, exists bool) {
	if exists = n.prefixes.Test(idx); !exists {
		return
	}
	n.prefixes.Clear(idx)
	n.pfxCount--
	return
}

// insertChild adds a child node at the specified address (0-255).
// The child can be a *slimNode[V], *slimLeafNode, or *slimFringeNode.
// Returns true if a child already existed at that address.
func (n *slimNode[V]) insertChild(addr uint8, child any) (exists bool) {
	return n.children.InsertAt(addr, child)
}

// getChild retrieves the child node at the specified address.
// Returns the child and true if found, or nil and false if not present.
//
//nolint:unused
func (n *slimNode[V]) getChild(addr uint8) (any, bool) {
	return n.children.Get(addr)
}

// getChildAddrs returns a slice of all addresses (0-255) that have children in this node.
// This is useful for iterating over all child nodes without checking every possible address.
//
//nolint:unused
func (n *slimNode[V]) getChildAddrs() []uint8 {
	return n.children.AsSlice(&[256]uint8{})
}

// allChildren returns an iterator over all child nodes.
// Each iteration yields the child's address (uint8) and the child node (any).
//
//nolint:unused
func (n *slimNode[V]) allChildren() iter.Seq2[uint8, any] {
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
func (n *slimNode[V]) mustGetChild(addr uint8) any {
	return n.children.MustGet(addr)
}

// deleteChild removes the child node at the specified address.
// This operation is idempotent - removing a non-existent child is safe.
func (n *slimNode[V]) deleteChild(addr uint8) (exists bool) {
	_, exists = n.children.DeleteAt(addr)
	return
}

// contains returns true if an index (idx) has any matching longest-prefix
// in the current node’s prefix table.
//
// This function performs a presence check.
//
// The prefix table is structured as a complete binary tree (CBT), and LPM testing
// is done via a bitset operation that maps the traversal path from the given index
// toward its possible ancestors.
func (n *slimNode[V]) contains(idx uint8) bool {
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
//nolint:unused
func (n *slimNode[V]) lookupIdx(idx uint8) (lpmIdx uint8, _ V, ok bool) {
	lpmIdx, ok = n.prefixes.IntersectionTop(&lpm.LookupTbl[idx])
	return
}

// lookup is just a simple wrapper for lookupIdx.
//
//nolint:unused
func (n *slimNode[V]) lookup(idx uint8) (_ V, ok bool) {
	_, _, ok = n.lookupIdx(idx)
	return
}

// slimLeafNode represents a path-compressed routing entry that stores only the prefix.
// Leaf nodes are used when a prefix doesn't align with stride boundaries
// and is stored as a compressed path to save memory.
type slimLeafNode struct {
	prefix netip.Prefix
}

// newSlimLeafNode creates a new leaf node with the specified prefix.
func newSlimLeafNode(pfx netip.Prefix) *slimLeafNode {
	return &slimLeafNode{prefix: pfx}
}

// slimFringeNode represents a path-compressed routing entry with an implicit prefix
// defined by the node's position in the trie. No prefix nor value is stored.
// Fringes are used for prefixes that align exactly with stride boundaries
// (/8, /16, /24, etc.) to save memory by not storing redundant prefix information.
type slimFringeNode struct{}

// newSlimFringeNode creates a new fringe node.
func newSlimFringeNode() *slimFringeNode {
	return new(slimFringeNode)
}

// insertAtDepth inserts a network prefix and its associated value into the
// trie starting at the specified byte depth.
//
// The function traverses the prefix address from the given depth and inserts
// the prefix either directly into the node's prefix table or as a compressed
// leaf or fringe node. If a conflicting leaf or fringe exists, it creates
// a new intermediate node to accommodate both entries.
//
// Parameters:
//   - pfx: The network prefix to insert (must be in canonical form)
//   - depth: The current depth in the trie (0-based byte index)
//
// Returns true if a prefix already existed and was updated, false for new insertions.
func (n *slimNode[V]) insertAtDepth(pfx netip.Prefix, depth int) (exists bool) {
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
				return n.insertChild(octet, newSlimFringeNode())
			}
			return n.insertChild(octet, newSlimLeafNode(pfx))
		}

		// ... or descend down the trie
		kid := n.mustGetChild(octet)

		// kid is node or leaf or fringe at addr
		switch kid := kid.(type) {
		case *slimNode[V]:
			n = kid // descend down to next trie level

		case *slimLeafNode:
			// reached a path compressed prefix
			if kid.prefix == pfx {
				// exists
				return true
			}

			// create new node
			// push the leaf down
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(slimNode[V])
			newNode.insertAtDepth(kid.prefix, depth+1)

			n.insertChild(octet, newNode)
			n = newNode

		case *slimFringeNode:
			// reached a path compressed fringe
			if isFringe(depth, pfx) {
				// exists
				return true
			}

			// create new node
			// push the fringe down, it becomes a default route (idx=1)
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(slimNode[V])
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
func (n *slimNode[V]) purgeAndCompress(stack []*slimNode[V], octets []uint8, is4 bool) {
	// unwind the stack
	for depth := len(stack) - 1; depth >= 0; depth-- {
		parent := stack[depth]
		octet := octets[depth]

		childCount := n.childCount()

		switch {
		case n.pfxCount == 0 && childCount == 0:
			// just delete this empty node from parent
			parent.deleteChild(octet)

		case n.pfxCount == 0 && childCount == 1:
			switch kid := n.children.Items[0].(type) {
			case *slimNode[V]:
				// fast exit, we are at an intermediate path node
				// no further delete/compress upwards the stack is possible
				return
			case *slimLeafNode:
				// just one leaf, delete this node and reinsert the leaf above
				parent.deleteChild(octet)

				// ... (re)insert the leaf at parents depth
				parent.insertAtDepth(kid.prefix, depth)
			case *slimFringeNode:
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

		case n.pfxCount == 1 && childCount == 0:
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
