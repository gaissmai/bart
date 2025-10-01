// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Usage: go generate -tags=ignore ./...
//go:generate ./scripts/generate-node-methods.sh
//go:build ignore

package bart

// ### GENERATE DELETE START ###

// stub code for generator types and methods
// useful for gopls during development, deleted during go generate

import (
	"iter"
	"net/netip"

	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/bitset"
)

type _NODE_TYPE[V any] struct {
	prefixes struct{ bitset.BitSet256 }
	children struct{ bitset.BitSet256 }
}

func (n *_NODE_TYPE[V]) isEmpty() (_ bool)                         { return }
func (n *_NODE_TYPE[V]) prefixCount() (_ int)                      { return }
func (n *_NODE_TYPE[V]) childCount() (_ int)                       { return }
func (n *_NODE_TYPE[V]) mustGetPrefix(uint8) (_ V)                 { return }
func (n *_NODE_TYPE[V]) mustGetChild(uint8) (_ any)                { return }
func (n *_NODE_TYPE[V]) insertPrefix(uint8, V) (_ bool)            { return }
func (n *_NODE_TYPE[V]) deletePrefix(uint8) (_ bool)               { return }
func (n *_NODE_TYPE[V]) getChild(uint8) (_ any, _ bool)            { return }
func (n *_NODE_TYPE[V]) getPrefix(uint8) (_ V, _ bool)             { return }
func (n *_NODE_TYPE[V]) insertChild(uint8, any) (_ bool)           { return }
func (n *_NODE_TYPE[V]) deleteChild(uint8) (_ bool)                { return }
func (n *_NODE_TYPE[V]) cloneRec(cloneFunc[V]) (_ *_NODE_TYPE[V])  { return }
func (n *_NODE_TYPE[V]) cloneFlat(cloneFunc[V]) (_ *_NODE_TYPE[V]) { return }
func (n *_NODE_TYPE[V]) allIndices() (seq2 iter.Seq2[uint8, V])    { return }
func (n *_NODE_TYPE[V]) allChildren() (seq2 iter.Seq2[uint8, any]) { return }

// ### GENERATE DELETE END ###

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
func (n *_NODE_TYPE[V]) insert(pfx netip.Prefix, val V, depth int) (exists bool) {
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
			newNode := new(_NODE_TYPE[V])
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

// insertPersist is similar to insert but the receiver isn't modified.
// Assumes the caller has pre-cloned the root (COW). It clones the
// internal nodes along the descent path before mutating them.
func (n *_NODE_TYPE[V]) insertPersist(cloneFn cloneFunc[V], pfx netip.Prefix, val V, depth int) (exists bool) {
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
func (n *_NODE_TYPE[V]) purgeAndCompress(stack []*_NODE_TYPE[V], octets []uint8, is4 bool) {
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
			case *_NODE_TYPE[V]:
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

// delete deletes the prefix and returns the associated value and true if the prefix existed,
// or false otherwise. The prefix must be in canonical form.
func (n *_NODE_TYPE[V]) delete(pfx netip.Prefix) (exists bool) {
	// invariant, prefix must be masked

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)

	// record the nodes on the path to the deleted node, needed to purge
	// and/or path compress nodes after the deletion of a prefix
	stack := [maxTreeDepth]*_NODE_TYPE[V]{}

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
			if exists = n.deletePrefix(art.PfxToIdx(octet, lastBits)); !exists {
				return false
			}

			// remove now-empty nodes and re-path-compress upwards
			n.purgeAndCompress(stack[:depth], octets, is4)
			return true
		}

		if !n.children.Test(octet) {
			return false
		}
		kid := n.mustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *_NODE_TYPE[V]:
			n = kid // descend down to next trie level

		case *fringeNode[V]:
			// if pfx is no fringe at this depth, fast exit
			if !isFringe(depth, pfx) {
				return false
			}

			// pfx is fringe at depth, delete fringe
			n.deleteChild(octet)

			// remove now-empty nodes and re-path-compress upwards
			n.purgeAndCompress(stack[:depth], octets, is4)

			return true

		case *leafNode[V]:
			// Attention: pfx must be masked to be comparable!
			if kid.prefix != pfx {
				return false
			}

			// prefix is equal leaf, delete leaf
			n.deleteChild(octet)

			// remove now-empty nodes and re-path-compress upwards
			n.purgeAndCompress(stack[:depth], octets, is4)

			return true

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
}

// deletePersist is similar to delete but does not mutate the original trie.
// Assumes the caller has pre-cloned the root (COW). It clones the
// internal nodes along the descent path before mutating them.
func (n *_NODE_TYPE[V]) deletePersist(cloneFn cloneFunc[V], pfx netip.Prefix) (exists bool) {
	ip := pfx.Addr() // the pfx must be in canonical form
	is4 := ip.Is4()
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)

	// Stack to keep track of cloned nodes along the path,
	// needed for purge and path compression after delete.
	stack := [maxTreeDepth]*_NODE_TYPE[V]{}

	// Traverse the trie to locate the prefix to delete.
	for depth, octet := range octets {
		// Keep track of the cloned node at current depth.
		stack[depth] = n

		if depth == lastOctetPlusOne {
			// Attempt to delete the prefix from the node's prefixes.
			if exists = n.deletePrefix(art.PfxToIdx(octet, lastBits)); !exists {
				// Prefix not found, nothing deleted.
				return false
			}

			// After deletion, purge nodes and compress the path if needed.
			n.purgeAndCompress(stack[:depth], octets, is4)

			return true
		}

		addr := octet

		// If child node doesn't exist, no prefix to delete.
		if !n.children.Test(addr) {
			return false
		}

		// Fetch child node at current address.
		kid := n.mustGetChild(addr)

		switch kid := kid.(type) {
		case *_NODE_TYPE[V]:
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
				return false
			}

			// Delete the fringe node.
			n.deleteChild(addr)

			// Purge and compress affected path.
			n.purgeAndCompress(stack[:depth], octets, is4)

			return true

		case *leafNode[V]:
			// Reached a path compressed leaf node.
			if kid.prefix != pfx {
				// Leaf prefix does not match; nothing to delete.
				return false
			}

			// Delete leaf node.
			n.deleteChild(addr)

			// Purge and compress affected path.
			n.purgeAndCompress(stack[:depth], octets, is4)

			return true

		default:
			// Unexpected node type indicates a logic error.
			panic("logic error, wrong node type")
		}
	}

	// Should never happen: traversal always returns or panics inside loop.
	panic("unreachable")
}

// get retrieves the value associated with the given network prefix.
// Returns the stored value and true if the prefix exists in this node,
// zero value and false if the prefix is not found.
//
// Parameters:
//   - pfx: The network prefix to look up (must be in canonical form)
//
// Returns:
//   - val: The value associated with the prefix (zero value if not found)
//   - exists: True if the prefix was found, false otherwise
func (n *_NODE_TYPE[V]) get(pfx netip.Prefix) (val V, exists bool) {
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
		case *_NODE_TYPE[V]:
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

// modify performs an in-place modification of a prefix using the provided callback function.
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
func (n *_NODE_TYPE[V]) modify(pfx netip.Prefix, cb func(val V, found bool) (_ V, del bool)) (delta int) {
	var zero V

	ip := pfx.Addr()
	is4 := ip.Is4()
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)

	// record the nodes on the path to the deleted node, needed to purge
	// and/or path compress nodes after the deletion of a prefix
	stack := [maxTreeDepth]*_NODE_TYPE[V]{}

	// find the proper trie node to update prefix
	for depth, octet := range octets {
		depth = depth & depthMask // BCE

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
				return 0

			case existed && del: // delete
				n.deletePrefix(idx)
				// remove now-empty nodes and re-path-compress upwards
				n.purgeAndCompress(stack[:depth], octets, is4)
				return -1

			case !existed: // insert
				n.insertPrefix(idx, newVal)
				return 1

			case existed: // update
				n.insertPrefix(idx, newVal)
				return 0

			default:
				panic("unreachable")
			}

		}

		// go down in tight loop to last octet
		if !n.children.Test(octet) {
			// insert prefix path compressed

			newVal, del := cb(zero, false)
			if del {
				return 0
			}

			// insert
			if isFringe(depth, pfx) {
				n.insertChild(octet, newFringeNode(newVal))
			} else {
				n.insertChild(octet, newLeafNode(pfx, newVal))
			}

			return 1
		}

		// n.children.Test(octet) == true
		kid := n.mustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *_NODE_TYPE[V]:
			n = kid // descend down to next trie level
			continue

		case *leafNode[V]:
			oldVal := kid.value

			// update existing value if prefixes are equal
			if kid.prefix == pfx {
				newVal, del := cb(oldVal, true)

				if !del {
					kid.value = newVal
					return 0
				}

				// delete
				n.deleteChild(octet)

				// remove now-empty nodes and re-path-compress upwards
				n.purgeAndCompress(stack[:depth], octets, is4)

				return -1
			}

			// stop if this is a no-op for zero values
			newVal, del := cb(zero, false)
			if del {
				return 0
			}

			// create new node
			// insert new child at current leaf position (octet)
			newNode := new(_NODE_TYPE[V])
			n.insertChild(octet, newNode)

			// push the leaf down
			// insert pfx with newVal in new node
			newNode.insert(kid.prefix, kid.value, depth+1)
			newNode.insert(pfx, newVal, depth+1)

			return 1

		case *fringeNode[V]:
			// update existing value if prefix is fringe
			if isFringe(depth, pfx) {
				newVal, del := cb(kid.value, true)
				if !del {
					kid.value = newVal
					return 0
				}

				// delete
				n.deleteChild(octet)

				// remove now-empty nodes and re-path-compress upwards
				n.purgeAndCompress(stack[:depth], octets, is4)

				return -1
			}

			// stop if this is a no-op for zero values
			newVal, del := cb(zero, false)
			if del {
				return 0
			}

			// create new node
			// insert new child at current leaf position (octet)
			newNode := new(_NODE_TYPE[V])
			n.insertChild(octet, newNode)

			// push the fringe down, it becomes a default route (idx=1)
			// insert pfx with newVal in new node
			newNode.insertPrefix(1, kid.value)
			newNode.insert(pfx, newVal, depth+1)

			return 1

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
}

// equalRec performs recursive structural equality comparison between two nodes.
// Compares prefix and child bitsets, then recursively compares all stored values
// and child nodes. Returns true if the nodes and their entire subtrees are
// structurally and semantically identical, false otherwise.
//
// The comparison handles different node types (internal nodes, leafNodes, fringeNodes)
// and uses the equal function for value comparisons to support custom equality logic.
func (n *_NODE_TYPE[V]) equalRec(o *_NODE_TYPE[V]) bool {
	if n == nil || o == nil {
		return n == o
	}
	if n == o {
		return true
	}

	if n.prefixes.BitSet256 != o.prefixes.BitSet256 {
		return false
	}

	if n.children.BitSet256 != o.children.BitSet256 {
		return false
	}

	for idx, nVal := range n.allIndices() {
		oVal := o.mustGetPrefix(idx) // mustGet is ok, bitsets are equal
		if !equal(nVal, oVal) {
			return false
		}
	}

	for addr, nKid := range n.allChildren() {
		oKid := o.mustGetChild(addr) // mustGet is ok, bitsets are equal

		switch nKid := nKid.(type) {
		case *_NODE_TYPE[V]:
			// oKid must also be a node
			oKid, ok := oKid.(*_NODE_TYPE[V])
			if !ok {
				return false
			}

			// compare rec-descent
			if !nKid.equalRec(oKid) {
				return false
			}

		case *leafNode[V]:
			// oKid must also be a leaf
			oKid, ok := oKid.(*leafNode[V])
			if !ok {
				return false
			}

			// compare prefixes
			if nKid.prefix != oKid.prefix {
				return false
			}

			// compare values
			if !equal(nKid.value, oKid.value) {
				return false
			}

		case *fringeNode[V]:
			// oKid must also be a fringe
			oKid, ok := oKid.(*fringeNode[V])
			if !ok {
				return false
			}

			// compare values
			if !equal(nKid.value, oKid.value) {
				return false
			}

		default:
			panic("logic error, wrong node type")
		}
	}

	return true
}
