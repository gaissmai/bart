// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Usage: go generate tags=ignore
//go:generate ./scripts/gen-monomorphized-methods.sh
//go:build ignore

package bart

// ### GENERATE DELETE START ###
import (
	"net/netip"

	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/bitset"
)

type _NODE_TYPE[V any] struct {
	prefixes struct{ bitset.BitSet256 }
	children struct{ bitset.BitSet256 }
}

func (n *_NODE_TYPE[V]) mustGetPrefix(uint8) (val V)                      { return val }
func (n *_NODE_TYPE[V]) mustGetChild(uint8) (child any)                   { return child }
func (n *_NODE_TYPE[V]) insertPrefix(uint8, V) (exists bool)              { return exists }
func (n *_NODE_TYPE[V]) getChild(uint8) (child any, ok bool)              { return child, ok }
func (n *_NODE_TYPE[V]) insertChild(uint8, any) (exists bool)             { return exists }
func (n *_NODE_TYPE[V]) cloneRec(cloneFunc[V]) (c *_NODE_TYPE[V])         { return c }
func (n *_NODE_TYPE[V]) cloneFlat(cloneFunc[V]) (c *_NODE_TYPE[V])        { return c }
func (n *_NODE_TYPE[V]) insertAtDepth(netip.Prefix, V, int) (exists bool) { return exists }

// ### GENERATE DELETE END ###

// insertAtDepth inserts a network prefix and its associated value into the
// trie starting at the specified byte depth.
//
// The function traverses the prefix address from the given depth and inserts
// the value either directly into the node's prefix table or as a compressed
// leaf or fringe node. If a conflicting leaf or fringe exists, it creates
// a new intermediate node to accommodate both entries.
//
// All nodes touched during insert are cloned.
//
// Parameters:
//   - pfx: The network prefix to insert (must be in canonical form)
//   - val: The value to associate with the prefix
//   - depth: The current depth in the trie (0-based byte index)
//
// Returns true if a prefix already existed and was updated, false for new insertions.
func (n *_NODE_TYPE[V]) insertAtDepthPersist(cloneFn cloneFunc[V], pfx netip.Prefix, val V, depth int) (exists bool) {
	ip := pfx.Addr() // the pfx must be in canonical form
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)

	// find the proper trie node to insert prefix
	// start with prefix octet at depth
	for ; depth < len(octets); depth++ {
		octet := octets[depth]

		// last masked octet: insert/override prefix/val into node
		if depth == lastOctetPlusOne {
			return n.insertPrefix(art.PfxToIdx(octet, lastBits), val)
		}

		// reached end of trie path ...
		if !n.children.Test(octet) {
			// insert prefix path compressed as leaf or fringe
			if isFringe(depth, pfx) {
				return n.insertChild(octet, newFringeNode(val))
			}
			return n.insertChild(octet, newLeafNode(pfx, val))
		}

		// ... or descend down the trie
		kid := n.mustGetChild(octet)

		// kid is node or leaf at addr
		switch kid := kid.(type) {
		case *_NODE_TYPE[V]:
			// clone the traversed path

			// kid points now to cloned kid
			kid = kid.cloneFlat(cloneFn)

			// replace kid with clone
			n.insertChild(octet, kid)

			n = kid
			continue // descend down to next trie level

		case *leafNode[V]:
			// reached a path compressed prefix
			// override value in slot if prefixes are equal
			if kid.prefix == pfx {
				kid.value = val
				// exists
				return true
			}

			// create new node
			// push the leaf down
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(_NODE_TYPE[V])
			newNode.insertAtDepth(kid.prefix, kid.value, depth+1)

			n.insertChild(octet, newNode)
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
			newNode := new(_NODE_TYPE[V])
			newNode.insertPrefix(1, kid.value)

			n.insertChild(octet, newNode)
			n = newNode

		default:
			panic("logic error, wrong node type")
		}

	}

	panic("unreachable")
}
