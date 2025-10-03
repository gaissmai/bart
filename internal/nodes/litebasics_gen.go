// Code generated from file "nodebasics_tmpl.go"; DO NOT EDIT.

// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package nodes

import (
	"net/netip"

	"github.com/gaissmai/bart/internal/art"
)

// Insert inserts a network prefix and its associated value into the
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
func (n *LiteNode[V]) Insert(pfx netip.Prefix, val V, depth int) (exists bool) {
	ip := pfx.Addr() // the pfx must be in canonical form
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := LastOctetPlusOneAndLastBits(pfx)

	// find the proper trie node to insert prefix
	// start with prefix octet at depth
	for ; depth < len(octets); depth++ {
		octet := octets[depth]

		// last masked octet: insert/override prefix/val into node
		if depth == lastOctetPlusOne {
			return n.InsertPrefix(art.PfxToIdx(octet, lastBits), val)
		}

		// reached end of trie path ...
		if !n.Children.Test(octet) {
			// insert prefix path compressed as leaf or fringe
			if IsFringe(depth, pfx) {
				return n.InsertChild(octet, NewFringeNode(val))
			}
			return n.InsertChild(octet, NewLeafNode(pfx, val))
		}

		// ... or descend down the trie
		kid := n.MustGetChild(octet)

		// kid is node or leaf at addr
		switch kid := kid.(type) {
		case *LiteNode[V]:
			n = kid // descend down to next trie level

		case *LeafNode[V]:
			// reached a path compressed prefix
			// override value in slot if prefixes are equal
			if kid.Prefix == pfx {
				kid.Value = val
				// exists
				return true
			}

			// create new node
			// push the leaf down
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(LiteNode[V])
			newNode.Insert(kid.Prefix, kid.Value, depth+1)

			n.InsertChild(octet, newNode)
			n = newNode

		case *FringeNode[V]:
			// reached a path compressed fringe
			// override value in slot if pfx is a fringe
			if IsFringe(depth, pfx) {
				kid.Value = val
				// exists
				return true
			}

			// create new node
			// push the fringe down, it becomes a default route (idx=1)
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(LiteNode[V])
			newNode.InsertPrefix(1, kid.Value)

			n.InsertChild(octet, newNode)
			n = newNode

		default:
			panic("logic error, wrong node type")
		}
	}
	panic("unreachable")
}

// InsertPersist is similar to insert but the receiver isn't modified.
// Assumes the caller has pre-cloned the root (COW). It clones the
// internal nodes along the descent path before mutating them.
func (n *LiteNode[V]) InsertPersist(cloneFn CloneFunc[V], pfx netip.Prefix, val V, depth int) (exists bool) {
	ip := pfx.Addr() // the pfx must be in canonical form
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := LastOctetPlusOneAndLastBits(pfx)

	// find the proper trie node to insert prefix
	// start with prefix octet at depth
	for ; depth < len(octets); depth++ {
		octet := octets[depth]

		// last masked octet: insert/override prefix/val into node
		if depth == lastOctetPlusOne {
			return n.InsertPrefix(art.PfxToIdx(octet, lastBits), val)
		}

		// reached end of trie path ...
		if !n.Children.Test(octet) {
			// insert prefix path compressed as leaf or fringe
			if IsFringe(depth, pfx) {
				return n.InsertChild(octet, NewFringeNode(val))
			}
			return n.InsertChild(octet, NewLeafNode(pfx, val))
		}

		// ... or descend down the trie
		kid := n.MustGetChild(octet)

		// kid is node or leaf at addr
		switch kid := kid.(type) {
		case *LiteNode[V]:
			// clone the traversed path

			// kid points now to cloned kid
			kid = kid.CloneFlat(cloneFn)

			// replace kid with clone
			n.InsertChild(octet, kid)

			n = kid
			continue // descend down to next trie level

		case *LeafNode[V]:
			// reached a path compressed prefix
			// override value in slot if prefixes are equal
			if kid.Prefix == pfx {
				kid.Value = val
				// exists
				return true
			}

			// create new node
			// push the leaf down
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(LiteNode[V])
			newNode.Insert(kid.Prefix, kid.Value, depth+1)

			n.InsertChild(octet, newNode)
			n = newNode

		case *FringeNode[V]:
			// reached a path compressed fringe
			// override value in slot if pfx is a fringe
			if IsFringe(depth, pfx) {
				kid.Value = val
				// exists
				return true
			}

			// create new node
			// push the fringe down, it becomes a default route (idx=1)
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(LiteNode[V])
			newNode.InsertPrefix(1, kid.Value)

			n.InsertChild(octet, newNode)
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
func (n *LiteNode[V]) PurgeAndCompress(stack []*LiteNode[V], octets []uint8, is4 bool) {
	// unwind the stack
	for depth := len(stack) - 1; depth >= 0; depth-- {
		parent := stack[depth]
		octet := octets[depth]

		pfxCount := n.PrefixCount()
		childCount := n.ChildCount()

		switch {
		case n.IsEmpty():
			// just delete this empty node from parent
			parent.DeleteChild(octet)

		case pfxCount == 0 && childCount == 1:
			singleAddr, _ := n.Children.FirstSet() // single addr must be first bit set
			anyKid := n.MustGetChild(singleAddr)

			switch kid := anyKid.(type) {
			case *LiteNode[V]:
				// fast exit, we are at an intermediate path node
				// no further delete/compress upwards the stack is possible
				return
			case *LeafNode[V]:
				// just one leaf, delete this node and reinsert the leaf above
				parent.DeleteChild(octet)

				// ... (re)insert the leaf at parents depth
				parent.Insert(kid.Prefix, kid.Value, depth)
			case *FringeNode[V]:
				// just one fringe, delete this node and reinsert the fringe as leaf above
				parent.DeleteChild(octet)

				// rebuild the prefix with octets, depth, ip version and addr
				// depth is the parent's depth, so add +1 here for the kid
				// lastOctet in cidrForFringe is the only addr (singleAddr)
				fringePfx := CidrForFringe(octets, depth+1, is4, singleAddr)

				// ... (re)reinsert prefix/value at parents depth
				parent.Insert(fringePfx, kid.Value, depth)
			}

		case pfxCount == 1 && childCount == 0:
			// just one prefix, delete this node and reinsert the idx as leaf above
			parent.DeleteChild(octet)

			// get prefix back from idx ...
			idx, _ := n.Prefixes.FirstSet() // single idx must be first bit set
			val := n.MustGetPrefix(idx)

			// ... and octet path
			path := StridePath{}
			copy(path[:], octets)

			// depth is the parent's depth, so add +1 here for the kid
			pfx := CidrFromPath(path, depth+1, is4, idx)

			// ... (re)insert prefix/value at parents depth
			parent.Insert(pfx, val, depth)
		}

		// climb up the stack
		n = parent
	}
}

// Delete deletes the prefix and returns the associated value and true if the prefix existed,
// or false otherwise. The prefix must be in canonical form.
func (n *LiteNode[V]) Delete(pfx netip.Prefix) (exists bool) {
	// invariant, prefix must be masked

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := LastOctetPlusOneAndLastBits(pfx)

	// record the nodes on the path to the deleted node, needed to purge
	// and/or path compress nodes after the deletion of a prefix
	stack := [MaxTreeDepth]*LiteNode[V]{}

	// find the trie node
	for depth, octet := range octets {
		depth = depth & DepthMask // BCE, Delete must be fast

		// push current node on stack for path recording
		stack[depth] = n

		// Last “octet” from prefix, update/insert prefix into node.
		// Note: For /32 and /128, depth never reaches lastOctetPlusOne (4/16),
		// so those are handled below via the fringe/leaf path.
		if depth == lastOctetPlusOne {
			// try to delete prefix in trie node
			if exists = n.DeletePrefix(art.PfxToIdx(octet, lastBits)); !exists {
				return false
			}

			// remove now-empty nodes and re-path-compress upwards
			n.PurgeAndCompress(stack[:depth], octets, is4)
			return true
		}

		if !n.Children.Test(octet) {
			return false
		}
		kid := n.MustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *LiteNode[V]:
			n = kid // descend down to next trie level

		case *FringeNode[V]:
			// if pfx is no fringe at this depth, fast exit
			if !IsFringe(depth, pfx) {
				return false
			}

			// pfx is fringe at depth, delete fringe
			n.DeleteChild(octet)

			// remove now-empty nodes and re-path-compress upwards
			n.PurgeAndCompress(stack[:depth], octets, is4)

			return true

		case *LeafNode[V]:
			// Attention: pfx must be masked to be comparable!
			if kid.Prefix != pfx {
				return false
			}

			// prefix is equal leaf, delete leaf
			n.DeleteChild(octet)

			// remove now-empty nodes and re-path-compress upwards
			n.PurgeAndCompress(stack[:depth], octets, is4)

			return true

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
}

// DeletePersist is similar to delete but does not mutate the original trie.
// Assumes the caller has pre-cloned the root (COW). It clones the
// internal nodes along the descent path before mutating them.
func (n *LiteNode[V]) DeletePersist(cloneFn CloneFunc[V], pfx netip.Prefix) (exists bool) {
	ip := pfx.Addr() // the pfx must be in canonical form
	is4 := ip.Is4()
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := LastOctetPlusOneAndLastBits(pfx)

	// Stack to keep track of cloned nodes along the path,
	// needed for purge and path compression after delete.
	stack := [MaxTreeDepth]*LiteNode[V]{}

	// Traverse the trie to locate the prefix to delete.
	for depth, octet := range octets {
		// Keep track of the cloned node at current depth.
		stack[depth] = n

		if depth == lastOctetPlusOne {
			// Attempt to delete the prefix from the node's prefixes.
			if exists = n.DeletePrefix(art.PfxToIdx(octet, lastBits)); !exists {
				// Prefix not found, nothing deleted.
				return false
			}

			// After deletion, purge nodes and compress the path if needed.
			n.PurgeAndCompress(stack[:depth], octets, is4)

			return true
		}

		addr := octet

		// If child node doesn't exist, no prefix to delete.
		if !n.Children.Test(addr) {
			return false
		}

		// Fetch child node at current address.
		kid := n.MustGetChild(addr)

		switch kid := kid.(type) {
		case *LiteNode[V]:
			// Clone the internal node for copy-on-write.
			kid = kid.CloneFlat(cloneFn)

			// Replace child with cloned node.
			n.InsertChild(addr, kid)

			// Descend to cloned child node.
			n = kid
			continue

		case *FringeNode[V]:
			// Reached a path compressed fringe.
			if !IsFringe(depth, pfx) {
				// Prefix to delete not found here.
				return false
			}

			// Delete the fringe node.
			n.DeleteChild(addr)

			// Purge and compress affected path.
			n.PurgeAndCompress(stack[:depth], octets, is4)

			return true

		case *LeafNode[V]:
			// Reached a path compressed leaf node.
			if kid.Prefix != pfx {
				// Leaf prefix does not match; nothing to delete.
				return false
			}

			// Delete leaf node.
			n.DeleteChild(addr)

			// Purge and compress affected path.
			n.PurgeAndCompress(stack[:depth], octets, is4)

			return true

		default:
			// Unexpected node type indicates a logic error.
			panic("logic error, wrong node type")
		}
	}

	// Should never happen: traversal always returns or panics inside loop.
	panic("unreachable")
}

// Get retrieves the value associated with the given network prefix.
// Returns the stored value and true if the prefix exists in this node,
// zero value and false if the prefix is not found.
//
// Parameters:
//   - pfx: The network prefix to look up (must be in canonical form)
//
// Returns:
//   - val: The value associated with the prefix (zero value if not found)
//   - exists: True if the prefix was found, false otherwise
func (n *LiteNode[V]) Get(pfx netip.Prefix) (val V, exists bool) {
	// invariant, prefix must be masked

	// values derived from pfx
	ip := pfx.Addr()
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := LastOctetPlusOneAndLastBits(pfx)

	// find the trie node
	for depth, octet := range octets {
		if depth == lastOctetPlusOne {
			return n.GetPrefix(art.PfxToIdx(octet, lastBits))
		}

		kidAny, ok := n.GetChild(octet)
		if !ok {
			return val, false
		}

		// kid is node or leaf or fringe at octet
		switch kid := kidAny.(type) {
		case *LiteNode[V]:
			n = kid // descend down to next trie level

		case *FringeNode[V]:
			// reached a path compressed fringe, stop traversing
			if IsFringe(depth, pfx) {
				return kid.Value, true
			}
			return val, false

		case *LeafNode[V]:
			// reached a path compressed prefix, stop traversing
			if kid.Prefix == pfx {
				return kid.Value, true
			}
			return val, false

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
}

// Modify performs an in-place modification of a prefix using the provided callback function.
// The callback receives the current value (if found) and existence flag, and returns
// a new value and deletion flag.
//
// modify returns the size delta (-1, 0, +1).
// This method handles path traversal, node creation for new paths, and automatic
// purge/compress operations after deletions.
//
// Parameters:
//   - pfx: The network prefix to modify (must be in canonical form)
//   - cb: Callback function that receives (currentValue, exists) and returns (newValue, deleteFlag)
//
// Returns:
//   - delta: Size change (-1 for delete, 0 for update/noop, +1 for insert)
func (n *LiteNode[V]) Modify(pfx netip.Prefix, cb func(val V, found bool) (_ V, del bool)) (delta int) {
	var zero V

	ip := pfx.Addr()
	is4 := ip.Is4()
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := LastOctetPlusOneAndLastBits(pfx)

	// record the nodes on the path to the deleted node, needed to purge
	// and/or path compress nodes after the deletion of a prefix
	stack := [MaxTreeDepth]*LiteNode[V]{}

	// find the proper trie node to update prefix
	for depth, octet := range octets {
		depth = depth & DepthMask // BCE

		// push current node on stack for path recording
		stack[depth] = n

		// Last “octet” from prefix, update/insert prefix into node.
		// Note: For /32 and /128, depth never reaches lastOctetPlusOne (4/16),
		// so those are handled below via the fringe/leaf path.
		if depth == lastOctetPlusOne {
			idx := art.PfxToIdx(octet, lastBits)

			oldVal, existed := n.GetPrefix(idx)
			newVal, del := cb(oldVal, existed)

			// update size if necessary
			switch {
			case !existed && del: // no-op
				return 0

			case existed && del: // delete
				n.DeletePrefix(idx)
				// remove now-empty nodes and re-path-compress upwards
				n.PurgeAndCompress(stack[:depth], octets, is4)
				return -1

			case !existed: // insert
				n.InsertPrefix(idx, newVal)
				return 1

			case existed: // update
				n.InsertPrefix(idx, newVal)
				return 0

			default:
				panic("unreachable")
			}

		}

		// go down in tight loop to last octet
		if !n.Children.Test(octet) {
			// insert prefix path compressed

			newVal, del := cb(zero, false)
			if del {
				return 0
			}

			// insert
			if IsFringe(depth, pfx) {
				n.InsertChild(octet, NewFringeNode(newVal))
			} else {
				n.InsertChild(octet, NewLeafNode(pfx, newVal))
			}

			return 1
		}

		// n.children.Test(octet) == true
		kid := n.MustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *LiteNode[V]:
			n = kid // descend down to next trie level
			continue

		case *LeafNode[V]:
			oldVal := kid.Value

			// update existing value if prefixes are equal
			if kid.Prefix == pfx {
				newVal, del := cb(oldVal, true)

				if !del {
					kid.Value = newVal
					return 0
				}

				// delete
				n.DeleteChild(octet)

				// remove now-empty nodes and re-path-compress upwards
				n.PurgeAndCompress(stack[:depth], octets, is4)

				return -1
			}

			// stop if this is a no-op for zero values
			newVal, del := cb(zero, false)
			if del {
				return 0
			}

			// create new node
			// insert new child at current leaf position (octet)
			newNode := new(LiteNode[V])
			n.InsertChild(octet, newNode)

			// push the leaf down
			// insert pfx with newVal in new node
			newNode.Insert(kid.Prefix, kid.Value, depth+1)
			newNode.Insert(pfx, newVal, depth+1)

			return 1

		case *FringeNode[V]:
			// update existing value if prefix is fringe
			if IsFringe(depth, pfx) {
				newVal, del := cb(kid.Value, true)
				if !del {
					kid.Value = newVal
					return 0
				}

				// delete
				n.DeleteChild(octet)

				// remove now-empty nodes and re-path-compress upwards
				n.PurgeAndCompress(stack[:depth], octets, is4)

				return -1
			}

			// stop if this is a no-op for zero values
			newVal, del := cb(zero, false)
			if del {
				return 0
			}

			// create new node
			// insert new child at current leaf position (octet)
			newNode := new(LiteNode[V])
			n.InsertChild(octet, newNode)

			// push the fringe down, it becomes a default route (idx=1)
			// insert pfx with newVal in new node
			newNode.InsertPrefix(1, kid.Value)
			newNode.Insert(pfx, newVal, depth+1)

			return 1

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
}

// EqualRec performs recursive structural equality comparison between two nodes.
// Compares prefix and child bitsets, then recursively compares all stored values
// and child nodes. Returns true if the nodes and their entire subtrees are
// structurally and semantically identical, false otherwise.
//
// The comparison handles different node types (internal nodes, leafNodes, fringeNodes)
// and uses the equal function for value comparisons to support custom equality logic.
func (n *LiteNode[V]) EqualRec(o *LiteNode[V]) bool {
	if n == nil || o == nil {
		return n == o
	}
	if n == o {
		return true
	}

	if n.Prefixes.BitSet256 != o.Prefixes.BitSet256 {
		return false
	}

	if n.Children.BitSet256 != o.Children.BitSet256 {
		return false
	}

	for idx, nVal := range n.AllIndices() {
		oVal := o.MustGetPrefix(idx) // mustGet is ok, bitsets are equal
		if !Equal(nVal, oVal) {
			return false
		}
	}

	for addr, nKid := range n.AllChildren() {
		oKid := o.MustGetChild(addr) // mustGet is ok, bitsets are equal

		switch nKid := nKid.(type) {
		case *LiteNode[V]:
			// oKid must also be a node
			oKid, ok := oKid.(*LiteNode[V])
			if !ok {
				return false
			}

			// compare rec-descent
			if !nKid.EqualRec(oKid) {
				return false
			}

		case *LeafNode[V]:
			// oKid must also be a leaf
			oKid, ok := oKid.(*LeafNode[V])
			if !ok {
				return false
			}

			// compare prefixes
			if nKid.Prefix != oKid.Prefix {
				return false
			}

			// compare values
			if !Equal(nKid.Value, oKid.Value) {
				return false
			}

		case *FringeNode[V]:
			// oKid must also be a fringe
			oKid, ok := oKid.(*FringeNode[V])
			if !ok {
				return false
			}

			// compare values
			if !Equal(nKid.Value, oKid.Value) {
				return false
			}

		default:
			panic("logic error, wrong node type")
		}
	}

	return true
}
