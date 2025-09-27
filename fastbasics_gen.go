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
func (n *fastNode[V]) insert(pfx netip.Prefix, val V, depth int) (exists bool) {
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
		case *fastNode[V]:
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
			newNode := new(fastNode[V])
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
			newNode := new(fastNode[V])
			newNode.insertPrefix(1, kid.value)

			n.insertChild(octet, newNode)
			n = newNode

		default:
			panic("logic error, wrong node type")
		}
	}
	panic("unreachable")
}

// insertPersist is similar to insert but the receiver isn't modified.
// All nodes touched during insert are cloned.
func (n *fastNode[V]) insertPersist(cloneFn cloneFunc[V], pfx netip.Prefix, val V, depth int) (exists bool) {
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
		case *fastNode[V]:
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
			newNode := new(fastNode[V])
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
			newNode := new(fastNode[V])
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
func (n *fastNode[V]) purgeAndCompress(stack []*fastNode[V], octets []uint8, is4 bool) {
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
			singleAddr, _ := n.children.FirstSet() // single addr must be first bit set
			anyKid := n.mustGetChild(singleAddr)

			switch kid := anyKid.(type) {
			case *fastNode[V]:
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

				// rebuild the prefix with octets, depth, ip version and addr
				// depth is the parent's depth, so add +1 here for the kid
				// lastOctet in cidrForFringe is the only addr (singleAddr)
				fringePfx := cidrForFringe(octets, depth+1, is4, singleAddr)

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

// delete the prefix and returns the associated value and true if the prefix existed,
// or zero value and false otherwise. The prefix must be in canonical form.
func (n *fastNode[V]) delete(pfx netip.Prefix) (val V, exists bool) {
	// invariant, prefix must be masked

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)

	// record the nodes on the path to the deleted node, needed to purge
	// and/or path compress nodes after the deletion of a prefix
	stack := [maxTreeDepth]*fastNode[V]{}

	// find the trie node
	for depth, octet := range octets {
		depth = depth & depthMask // BCE, Delete must be fast

		// push current node on stack for path recording
		stack[depth] = n

		// Last “octet” from prefix, update/insert prefix into node.
		// Note: For /32 and /128, depth never reaches lastOctetPlusOne (4/16),
		// so those are handled below via the fringe/leaf path.
		if depth == lastOctetPlusOne {
			// try to delete prefix in trie node
			val, exists = n.deletePrefix(art.PfxToIdx(octet, lastBits))
			if !exists {
				return val, exists
			}

			// remove now-empty nodes and re-path-compress upwards
			n.purgeAndCompress(stack[:depth], octets, is4)
			return val, true
		}

		if !n.children.Test(octet) {
			return val, exists
		}
		kid := n.mustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *fastNode[V]:
			n = kid // descend down to next trie level

		case *fringeNode[V]:
			// if pfx is no fringe at this depth, fast exit
			if !isFringe(depth, pfx) {
				return val, exists
			}

			// pfx is fringe at depth, delete fringe
			n.deleteChild(octet)

			// remove now-empty nodes and re-path-compress upwards
			n.purgeAndCompress(stack[:depth], octets, is4)

			return kid.value, true

		case *leafNode[V]:
			// Attention: pfx must be masked to be comparable!
			if kid.prefix != pfx {
				return val, exists
			}

			// prefix is equal leaf, delete leaf
			n.deleteChild(octet)

			// remove now-empty nodes and re-path-compress upwards
			n.purgeAndCompress(stack[:depth], octets, is4)

			return kid.value, true

		default:
			panic("logic error, wrong node type")
		}
	}

	return val, exists
}

// deletePersist is similar to delete but the receiver isn't modified.
// All nodes touched during insert are cloned.
func (n *fastNode[V]) deletePersist(cloneFn cloneFunc[V], pfx netip.Prefix) (val V, exists bool) {
	ip := pfx.Addr() // the pfx must be in canonical form
	is4 := ip.Is4()
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)

	// Stack to keep track of cloned nodes along the path,
	// needed for purge and path compression after delete.
	stack := [maxTreeDepth]*fastNode[V]{}

	// Traverse the trie to locate the prefix to delete.
	for depth, octet := range octets {
		// Keep track of the cloned node at current depth.
		stack[depth] = n

		if depth == lastOctetPlusOne {
			// Attempt to delete the prefix from the node's prefixes.
			val, exists = n.deletePrefix(art.PfxToIdx(octet, lastBits))
			if !exists {
				// Prefix not found, nothing deleted.
				return val, false
			}

			// After deletion, purge nodes and compress the path if needed.
			n.purgeAndCompress(stack[:depth], octets, is4)

			return val, true
		}

		addr := octet

		// If child node doesn't exist, no prefix to delete.
		if !n.children.Test(addr) {
			return val, false
		}

		// Fetch child node at current address.
		kid := n.mustGetChild(addr)

		switch kid := kid.(type) {
		case *fastNode[V]:
			// Clone the internal node for copy-on-write.
			kid = kid.cloneFlat(cloneFn)

			// Replace child with cloned node.
			n.insertChild(addr, kid)

			// Descend to cloned child node.
			n = kid
			continue

		case *fringeNode[V]:
			// Reached a path compressed fringe.
			if !isFringe(depth, pfx) {
				// Prefix to delete not found here.
				return val, false
			}

			// Delete the fringe node.
			n.deleteChild(addr)

			// Purge and compress affected path.
			n.purgeAndCompress(stack[:depth], octets, is4)

			return kid.value, true

		case *leafNode[V]:
			// Reached a path compressed leaf node.
			if kid.prefix != pfx {
				// Leaf prefix does not match; nothing to delete.
				return val, false
			}

			// Delete leaf node.
			n.deleteChild(addr)

			// Purge and compress affected path.
			n.purgeAndCompress(stack[:depth], octets, is4)

			return kid.value, true

		default:
			// Unexpected node type indicates a logic error.
			panic("logic error, wrong node type")
		}
	}

	// Should never happen: traversal always returns or panics inside loop.
	panic("unreachable")
}

func (n *fastNode[V]) get(pfx netip.Prefix) (val V, exists bool) {
	// invariant, prefix must be masked

	// values derived from pfx
	ip := pfx.Addr()
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)

	// find the trie node
	for depth, octet := range octets {
		if depth == lastOctetPlusOne {
			return n.getPrefix(art.PfxToIdx(octet, lastBits))
		}

		kidAny, ok := n.getChild(octet)
		if !ok {
			return val, false
		}

		// kid is node or leaf or fringe at octet
		switch kid := kidAny.(type) {
		case *fastNode[V]:
			n = kid // descend down to next trie level

		case *fringeNode[V]:
			// reached a path compressed fringe, stop traversing
			if isFringe(depth, pfx) {
				return kid.value, true
			}
			return val, false

		case *leafNode[V]:
			// reached a path compressed prefix, stop traversing
			if kid.prefix == pfx {
				return kid.value, true
			}
			return val, false

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
}

func (n *fastNode[V]) modify(pfx netip.Prefix, cb func(val V, found bool) (_ V, del bool)) (delta int, _ V, deleted bool) {
	var zero V

	ip := pfx.Addr()
	is4 := ip.Is4()
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)

	// record the nodes on the path to the deleted node, needed to purge
	// and/or path compress nodes after the deletion of a prefix
	stack := [maxTreeDepth]*fastNode[V]{}

	// find the proper trie node to update prefix
	for depth, octet := range octets {
		// push current node on stack for path recording
		stack[depth] = n

		// Last “octet” from prefix, update/insert prefix into node.
		// Note: For /32 and /128, depth never reaches lastOctetPlusOne (4/16),
		// so those are handled below via the fringe/leaf path.
		if depth == lastOctetPlusOne {
			idx := art.PfxToIdx(octet, lastBits)

			oldVal, existed := n.getPrefix(idx)
			newVal, del := cb(oldVal, existed)

			// update size if necessary
			switch {
			case !existed && del: // no-op
				return 0, zero, false

			case existed && del: // delete
				n.deletePrefix(idx)
				// remove now-empty nodes and re-path-compress upwards
				n.purgeAndCompress(stack[:depth], octets, is4)
				return -1, oldVal, true

			case !existed: // insert
				n.insertPrefix(idx, newVal)
				return 1, newVal, false

			case existed: // update
				n.insertPrefix(idx, newVal)
				return 0, oldVal, false

			default:
				panic("unreachable")
			}

		}

		// go down in tight loop to last octet
		if !n.children.Test(octet) {
			// insert prefix path compressed

			newVal, del := cb(zero, false)
			if del {
				return 0, zero, false // no-op
			}

			// insert
			if isFringe(depth, pfx) {
				n.insertChild(octet, newFringeNode(newVal))
			} else {
				n.insertChild(octet, newLeafNode(pfx, newVal))
			}

			return 1, newVal, false
		}

		// n.children.Test(octet) == true
		kid := n.mustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *fastNode[V]:
			n = kid // descend down to next trie level
			continue

		case *leafNode[V]:
			oldVal := kid.value

			// update existing value if prefixes are equal
			if kid.prefix == pfx {
				newVal, del := cb(oldVal, true)

				if !del {
					kid.value = newVal
					return 0, oldVal, false // update
				}

				// delete
				n.deleteChild(octet)

				// remove now-empty nodes and re-path-compress upwards
				n.purgeAndCompress(stack[:depth], octets, is4)

				return -1, oldVal, true
			}

			// stop if this is a no-op for zero values
			newVal, del := cb(zero, false)
			if del {
				return 0, zero, false
			}

			// create new node
			// insert new child at current leaf position (octet)
			newNode := new(fastNode[V])
			n.insertChild(octet, newNode)

			// push the leaf down
			// insert pfx with newVal in new node
			newNode.insert(kid.prefix, kid.value, depth+1)
			newNode.insert(pfx, newVal, depth+1)

			return 1, newVal, false

		case *fringeNode[V]:
			oldVal := kid.value

			// update existing value if prefix is fringe
			if isFringe(depth, pfx) {
				newVal, del := cb(kid.value, true)
				if !del {
					kid.value = newVal
					return 0, oldVal, false // update
				}

				// delete
				n.deleteChild(octet)

				// remove now-empty nodes and re-path-compress upwards
				n.purgeAndCompress(stack[:depth], octets, is4)

				return -1, oldVal, true
			}

			// stop if this is a no-op for zero values
			newVal, del := cb(zero, false)
			if del {
				return 0, zero, false
			}

			// create new node
			// insert new child at current leaf position (octet)
			newNode := new(fastNode[V])
			n.insertChild(octet, newNode)

			// push the fringe down, it becomes a default route (idx=1)
			// insert pfx with newVal in new node
			newNode.insertPrefix(1, kid.value)
			newNode.insert(pfx, newVal, depth+1)

			return 1, newVal, false

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
}

func (n *fastNode[V]) modifyPersist(cloneFn cloneFunc[V], pfx netip.Prefix, cb func(val V, found bool) (_ V, del bool)) (delta int, _ V, deleted bool) {
	var zero V

	ip := pfx.Addr()
	is4 := ip.Is4()
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)

	// record the nodes on the path to the deleted node, needed to purge
	// and/or path compress nodes after the deletion of a prefix
	stack := [maxTreeDepth]*fastNode[V]{}

	// find the proper trie node to update prefix
	for depth, octet := range octets {
		// push current node on stack for path recording
		stack[depth] = n

		// Last “octet” from prefix, update/insert prefix into node.
		// Note: For /32 and /128, depth never reaches lastOctetPlusOne (4/16),
		// so those are handled below via the fringe/leaf path.
		if depth == lastOctetPlusOne {
			idx := art.PfxToIdx(octet, lastBits)

			oldVal, existed := n.getPrefix(idx)
			newVal, del := cb(oldVal, existed)

			// update size if necessary
			switch {
			case !existed && del: // no-op
				return 0, zero, false

			case existed && del: // delete
				n.deletePrefix(idx)
				// remove now-empty nodes and re-path-compress upwards
				n.purgeAndCompress(stack[:depth], octets, is4)
				return -1, oldVal, true

			case !existed: // insert
				n.insertPrefix(idx, newVal)
				return 1, newVal, false

			case existed: // update
				n.insertPrefix(idx, newVal)
				return 0, oldVal, false

			default:
				panic("unreachable")
			}

		}

		// go down in tight loop to last octet
		if !n.children.Test(octet) {
			// insert prefix path compressed

			newVal, del := cb(zero, false)
			if del {
				return 0, zero, false // no-op
			}

			// insert
			if isFringe(depth, pfx) {
				n.insertChild(octet, newFringeNode(newVal))
			} else {
				n.insertChild(octet, newLeafNode(pfx, newVal))
			}

			return 1, newVal, false
		}

		// n.children.Test(octet) == true
		kid := n.mustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *fastNode[V]:
			// Clone the node along the traversed path to respect copy-on-write.
			kid = kid.cloneFlat(cloneFn)

			// Replace original child with the cloned child.
			n.insertChild(octet, kid)

			n = kid // descend down to next trie level
			continue

		case *leafNode[V]:
			oldVal := kid.value

			// update existing value if prefixes are equal
			if kid.prefix == pfx {
				newVal, del := cb(oldVal, true)

				if !del {
					kid.value = newVal
					return 0, oldVal, false // update
				}

				// delete
				n.deleteChild(octet)

				// remove now-empty nodes and re-path-compress upwards
				n.purgeAndCompress(stack[:depth], octets, is4)

				return -1, oldVal, true
			}

			// stop if this is a no-op for zero values
			newVal, del := cb(zero, false)
			if del {
				return 0, zero, false
			}

			// create new node
			// insert new child at current leaf position (octet)
			newNode := new(fastNode[V])
			n.insertChild(octet, newNode)

			// push the leaf down
			// insert pfx with newVal in new node
			newNode.insert(kid.prefix, kid.value, depth+1)
			newNode.insert(pfx, newVal, depth+1)

			return 1, newVal, false

		case *fringeNode[V]:
			oldVal := kid.value

			// update existing value if prefix is fringe
			if isFringe(depth, pfx) {
				newVal, del := cb(kid.value, true)
				if !del {
					kid.value = newVal
					return 0, oldVal, false // update
				}

				// delete
				n.deleteChild(octet)

				// remove now-empty nodes and re-path-compress upwards
				n.purgeAndCompress(stack[:depth], octets, is4)

				return -1, oldVal, true
			}

			// stop if this is a no-op for zero values
			newVal, del := cb(zero, false)
			if del {
				return 0, zero, false
			}

			// create new node
			// insert new child at current leaf position (octet)
			newNode := new(fastNode[V])
			n.insertChild(octet, newNode)

			// push the fringe down, it becomes a default route (idx=1)
			// insert pfx with newVal in new node
			newNode.insertPrefix(1, kid.value)
			newNode.insert(pfx, newVal, depth+1)

			return 1, newVal, false

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
}
