// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"

	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/bitset"
	"github.com/gaissmai/bart/internal/lpm"
)

// fatNode is a trie level node in the multibit routing table.
//
// Each fatNode contains two conceptually different fixed sized arrays:
//   - prefixes: representing routes, using a complete binary tree layout
//     driven by the baseIndex() function from the ART algorithm.
//   - children: holding subtries or path-compressed leaves or fringes.
//
// See doc/artlookup.pdf for the mapping mechanics and prefix tree details.
type fatNode[V any] struct {
	prefixes [256]*V
	children [256]*any // **fatNode or path-compressed **leaf- or **fringeNode
	// an array of "pointers to" the empty interface,
	// and not an array of empty interfaces.
	//
	// - any  ( interface{}) takes 2 words, even if nil.
	// - *any (*interface{}) requires only 1 word when nil.
	//
	// Since many slots are nil, this reduces memory by 30%.
	// The added indirection does not have a measurable performance impact,
	// but makes the code uglier.

	prefixesBitSet bitset.BitSet256 // for count and fast bitset operations
	childrenBitSet bitset.BitSet256 // for count and fast bitset operations
}

// prefixCount returns the number of prefixes stored in this node.
func (n *fatNode[V]) prefixCount() int {
	return n.prefixesBitSet.Size()
}

// childCount returns the number of slots used in this node.
func (n *fatNode[V]) childCount() int {
	return n.childrenBitSet.Size()
}

// isEmpty returns true if node has neither prefixes nor children
func (n *fatNode[V]) isEmpty() bool {
	if n == nil {
		return true
	}
	return (n.prefixesBitSet.Size() + n.childrenBitSet.Size()) == 0
}

// getChild returns the child node at the specified address and true if it exists.
// If no child exists at addr, returns nil and false.
//
//nolint:unused
func (n *fatNode[V]) getChild(addr uint8) (any, bool) {
	if anyPtr := n.children[addr]; anyPtr != nil {
		return *anyPtr, true
	}
	return nil, false
}

// mustGetChild returns the child node at the specified address.
// Panics if no child exists at addr. This method should only be called
// when the caller has verified the child exists.
//
//nolint:unused
func (n *fatNode[V]) mustGetChild(addr uint8) any {
	// panics if n.children[addr] is nil
	return *n.children[addr]
}

// getChildAddrs returns a slice containing all addresses that have child nodes.
// The addresses are returned in ascending order.
//
//nolint:unused
func (n *fatNode[V]) getChildAddrs() []uint8 {
	return n.childrenBitSet.AsSlice(&[256]uint8{})
}

// insertChild inserts a child node at the specified address.
// Returns true if a child already existed at addr (overwrite case),
// false if this is a new insertion.
func (n *fatNode[V]) insertChild(addr uint8, child any) (exists bool) {
	if n.children[addr] == nil {
		exists = false
		n.childrenBitSet.Set(addr)
	} else {
		exists = true
	}

	// force clear ownership; taking &c makes it escape to heap so the pointer remains valid.
	// This reduces per-slot memory for nil entries versus storing `any` directly.
	c := child
	n.children[addr] = &c

	return exists
}

// deleteChild removes the child node at the specified address.
// This operation is idempotent - removing a non-existent child is safe.
func (n *fatNode[V]) deleteChild(addr uint8) (exists bool) {
	if n.children[addr] == nil {
		return false
	}

	n.childrenBitSet.Clear(addr)
	n.children[addr] = nil
	return true
}

// insertPrefix adds a routing entry at the specified index with the given value.
// It returns true if a prefix already existed at that index (indicating an update),
// false if this is a new insertion.
func (n *fatNode[V]) insertPrefix(idx uint8, val V) (exists bool) {
	if exists = n.prefixesBitSet.Test(idx); !exists {
		n.prefixesBitSet.Set(idx)
	}

	// insert or update

	// To ensure allot works as intended, every unique prefix in the
	// fatNode must point to a distinct value pointer, even for identical values.
	// Using new() and assignment guarantees each inserted prefix gets its own address,
	valPtr := new(V)
	*valPtr = val

	oldValPtr := n.prefixes[idx]

	// overwrite oldValPtr with valPtr
	n.allot(idx, oldValPtr, valPtr)

	return
}

// getPrefix returns the value for the given prefix index and true if it exists.
// If no prefix exists at idx, returns the zero value and false.
func (n *fatNode[V]) getPrefix(idx uint8) (val V, exists bool) {
	if exists = n.prefixesBitSet.Test(idx); exists {
		val = *n.prefixes[idx]
	}
	return
}

// mustGetPrefix returns the value for the given prefix index.
// Panics if no prefix exists at idx. This method should only be called
// when the caller has verified the prefix exists.
//
//nolint:unused
func (n *fatNode[V]) mustGetPrefix(idx uint8) V {
	return *n.prefixes[idx]
}

// getIndices returns a slice containing all prefix indices that have values stored.
// The indices are returned in ascending order.
//
//nolint:unused
func (n *fatNode[V]) getIndices() []uint8 {
	return n.prefixesBitSet.AsSlice(&[256]uint8{})
}

// deletePrefix removes the route at the given index.
// Returns the removed value and true if the prefix existed,
// or the zero value and false if no prefix was found at idx.
func (n *fatNode[V]) deletePrefix(idx uint8) (val V, exists bool) {
	if exists = n.prefixesBitSet.Test(idx); !exists {
		// Route entry doesn't exist
		return
	}

	valPtr := n.prefixes[idx]
	parentValPtr := n.prefixes[idx>>1]

	// delete -> overwrite valPtr with parentValPtr
	n.allot(idx, valPtr, parentValPtr)

	n.prefixesBitSet.Clear(idx)
	return *valPtr, true
}

// contains returns true if the given index has any matching longest-prefix
// in the current node's prefix table.
//
// This function performs a presence check using the ART algorithm's
// hierarchical prefix structure. It tests whether any ancestor prefix
// exists for the given index by probing the slot at idx (children inherit
// ancestor pointers via allot), after normalizing the ART stride index.
// idx must come from art.PfxToIdx (1..255) or art.OctetToIdx(octet) (256..511).
// Host-route indices [256..511] are normalized to their parent prefix slot
// [128..255] via idx >>= 1.
func (n *fatNode[V]) contains(idx uint) (ok bool) {
	normalizedIdx := normalizeIdx(idx)
	return n.prefixes[normalizedIdx] != nil
}

// lookup performs a longest-prefix match (LPM) lookup for the given index
// within the current node's prefix table in O(1).
//
// The function returns the matched value and true if a matching prefix exists;
// otherwise, it returns the zero value and false. The lookup uses the ART
// algorithm's hierarchical structure to find the most specific
// matching prefix.
func (n *fatNode[V]) lookup(idx uint) (val V, ok bool) {
	normalizedIdx := normalizeIdx(idx)
	if valPtr := n.prefixes[normalizedIdx]; valPtr != nil {
		return *valPtr, true
	}
	return
}

// lookupIdx performs a longest-prefix match (LPM) lookup for the given index (idx)
// within the 8-bit stride-based prefix table at this trie depth.
//
// idx must be an ART stride index as returned by art.OctetToIdx (range 256..511) or
// art.PfxToIdx (range 1..255)
//
// The function returns the matched base index, associated value, and true if a
// matching prefix exists at this level; otherwise, ok is false.
//
// Its semantics are identical to [node.lookupIdx].
func (n *fatNode[V]) lookupIdx(idx uint) (baseIdx uint8, val V, ok bool) {
	normalizedIdx := normalizeIdx(idx)
	// top is the idx of the longest-prefix-match
	if top, ok := n.prefixesBitSet.IntersectionTop(lpm.BackTrackingBitset(normalizedIdx)); ok {
		return top, n.mustGetPrefix(top), true
	}
	return
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
func (n *fatNode[V]) allot(idx uint8, oldValPtr, valPtr *V) {
	// iteration with stack instead of recursion
	stack := make([]uint8, 0, 256)

	// start idx
	stack = append(stack, idx)

	for i := 0; i < len(stack); i++ {
		idx = stack[i]

		// stop this allot path, idx already points to a more specific route.
		if n.prefixes[idx] != oldValPtr {
			continue // take next path from stack
		}

		// overwrite
		n.prefixes[idx] = valPtr

		// max idx is 255, so stop the duplication at 128 and above
		if idx >= 128 {
			continue
		}

		// child nodes, it's a complete binary tree
		// left:  idx*2
		// right: idx*2+1
		stack = append(stack, idx<<1, idx<<1+1)
	}
}

// insertAtDepth inserts a network prefix and its associated value into the
// trie starting at the specified byte depth.
//
// The function walks the prefix address from the given depth and inserts the value either directly into
// the node's prefix table or as a compressed leaf or fringe node. If a conflicting leaf or fringe exists,
// it is pushed down via a new intermediate node. Existing entries with the same prefix are overwritten.
func (n *fatNode[V]) insertAtDepth(pfx netip.Prefix, val V, depth int) (exists bool) {
	ip := pfx.Addr() // the pfx must be in canonical form
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)

	// find the proper trie node to insert prefix
	// start with prefix octet at depth
	for _, octet := range octets[depth:] {
		// last masked octet: insert/override prefix/val into node
		if depth == lastOctetPlusOne {
			return n.insertPrefix(art.PfxToIdx(octet, lastBits), val)
		}

		kidAny, ok := n.getChild(octet)
		// reached end of trie path ...
		if !ok {
			// insert prefix path compressed as leaf or fringe
			if isFringe(depth, pfx) {
				return n.insertChild(octet, newFringeNode(val))
			}
			return n.insertChild(octet, newLeafNode(pfx, val))
		}

		// kid is node or leaf at addr
		switch kid := kidAny.(type) {
		case *fatNode[V]:
			n = kid // descend down to next trie level

		case *leafNode[V]:
			// reached a path compressed prefix
			// override value in slot if prefixes are equal
			if kid.prefix == pfx {
				kid.value = val
				// update
				return true
			}

			// create new node
			// push the leaf down
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(fatNode[V])
			_ = newNode.insertAtDepth(kid.prefix, kid.value, depth+1)
			_ = n.insertChild(octet, newNode)
			n = newNode

		case *fringeNode[V]:
			// reached a path compressed fringe
			// override value in slot if pfx is a fringe
			if isFringe(depth, pfx) {
				kid.value = val
				// exists
				return true
			}

			// create new node
			// push the fringe down, it becomes a default route (idx=1)
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(fatNode[V])
			_ = newNode.insertPrefix(1, kid.value)
			_ = n.insertChild(octet, newNode)
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
// It unwinds the provided stack of parent nodes, checking each level for
// compression opportunities based on child count and prefix distribution.
//
// The compression may convert:
//   - Nodes with single prefix into leafNode (path compression)
//   - Nodes at maxDepth-1 with single prefix into fringeNode
//   - Empty intermediate nodes are removed entirely
//
// The reconstruction of prefixes for compressed nodes uses the traversal
// path stored in octets and the parent's depth information.
func (n *fatNode[V]) purgeAndCompress(stack []*fatNode[V], octets []uint8, is4 bool) {
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
			addr, _ := n.childrenBitSet.FirstSet() // single child must be first child
			kidAny := *n.children[addr]

			switch kid := kidAny.(type) {
			case *fatNode[V]:
				// fast exit, we are at an intermediate path node
				// no further delete/compress upwards the stack is possible
				return
			case *leafNode[V]:
				// just one leaf, delete this node and reinsert the leaf above
				parent.deleteChild(octet)

				// ... (re)insert the leaf at parents depth
				parent.insertAtDepth(kid.prefix, kid.value, depth)
			case *fringeNode[V]:
				// just one fringe, delete this node and reinsert the fringe as leaf above
				parent.deleteChild(octet)

				// rebuild the prefix with octets, depth, ip version and addr
				// depth is the parent's depth, so add +1 here for the kid
				fringePfx := cidrForFringe(octets, depth+1, is4, addr)

				// ... (re)reinsert prefix/value at parents depth
				parent.insertAtDepth(fringePfx, kid.value, depth)
			}

		case pfxCount == 1 && childCount == 0:
			// just one prefix, delete this node and reinsert the idx as leaf above
			parent.deleteChild(octet)

			// get prefix/val back from idx ...
			idx, _ := n.prefixesBitSet.FirstSet() // single idx must be first bit set
			val := *n.prefixes[idx]

			// ... and octet path
			path := stridePath{}
			copy(path[:], octets)

			// depth is the parent's depth, so add +1 here for the kid
			pfx := cidrFromPath(path, depth+1, is4, idx)

			// ... (re)insert prefix/value at parents depth
			parent.insertAtDepth(pfx, val, depth)
		}

		// climb up the stack
		n = parent
	}
}
