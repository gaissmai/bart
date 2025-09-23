// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Code generated from file "nodebasics_tmpl.go"; DO NOT EDIT.

package bart

import (
	"net/netip"

	"github.com/gaissmai/bart/internal/art"
)

// insert inserts a network prefix and its associated value into the
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
func (n *bartNode[V]) insert(pfx netip.Prefix, val V, depth int) (exists bool) {
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
		case *bartNode[V]:
			n = kid // descend down to next trie level

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
			newNode := new(bartNode[V])
			newNode.insert(kid.prefix, kid.value, depth+1)

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
			newNode := new(bartNode[V])
			newNode.insertPrefix(1, kid.value)

			n.insertChild(octet, newNode)
			n = newNode

		default:
			panic("logic error, wrong node type")
		}
	}
	panic("unreachable")
}

// insertPersist inserts a network prefix and its associated value into the
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
func (n *bartNode[V]) insertPersist(cloneFn cloneFunc[V], pfx netip.Prefix, val V, depth int) (exists bool) {
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
		case *bartNode[V]:
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
			newNode := new(bartNode[V])
			newNode.insert(kid.prefix, kid.value, depth+1)

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
			newNode := new(bartNode[V])
			newNode.insertPrefix(1, kid.value)

			n.insertChild(octet, newNode)
			n = newNode

		default:
			panic("logic error, wrong node type")
		}

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
func (n *bartNode[V]) purgeAndCompress(stack []*bartNode[V], octets []uint8, is4 bool) {
	// unwind the stack
	for depth := len(stack) - 1; depth >= 0; depth-- {
		parent := stack[depth]
		octet := octets[depth]

		pfxCount := n.prefixCount()
		childCount := n.childCount()

		switch {
		case n.isEmpty():
			// just delete this empty node from parent
			parent.deleteChild(octet)

		case pfxCount == 0 && childCount == 1:
			addr, _ := n.children.FirstSet() // single addr must be first bit set
			anyKid := n.mustGetChild(addr)

			switch kid := anyKid.(type) {
			case *bartNode[V]:
				// fast exit, we are at an intermediate path node
				// no further delete/compress upwards the stack is possible
				return
			case *leafNode[V]:
				// just one leaf, delete this node and reinsert the leaf above
				parent.deleteChild(octet)

				// ... (re)insert the leaf at parents depth
				parent.insert(kid.prefix, kid.value, depth)
			case *fringeNode[V]:
				// just one fringe, delete this node and reinsert the fringe as leaf above
				parent.deleteChild(octet)

				// get the last octet back, the only item is also the first item
				lastOctet, _ := n.children.FirstSet()

				// rebuild the prefix with octets, depth, ip version and addr
				// depth is the parent's depth, so add +1 here for the kid
				fringePfx := cidrForFringe(octets, depth+1, is4, lastOctet)

				// ... (re)reinsert prefix/value at parents depth
				parent.insert(fringePfx, kid.value, depth)
			}

		case pfxCount == 1 && childCount == 0:
			// just one prefix, delete this node and reinsert the idx as leaf above
			parent.deleteChild(octet)

			// get prefix back from idx ...
			idx, _ := n.prefixes.FirstSet() // single idx must be first bit set
			val := n.mustGetPrefix(idx)

			// ... and octet path
			path := stridePath{}
			copy(path[:], octets)

			// depth is the parent's depth, so add +1 here for the kid
			pfx := cidrFromPath(path, depth+1, is4, idx)

			// ... (re)insert prefix/value at parents depth
			parent.insert(pfx, val, depth)
		}

		// climb up the stack
		n = parent
	}
}
