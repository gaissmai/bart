// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"

	"github.com/gaissmai/bart/internal/art"
)

// InsertPersist is similar to Insert but the receiver isn't modified.
//
// All nodes touched during insert are cloned and a new Table is returned.
// This is not a full [Table.Clone], all untouched nodes are still referenced
// from both Tables.
//
// If the payload type V contains pointers or needs deep copying,
// it must implement the [bart.Cloner] interface to support correct cloning.
//
// This is orders of magnitude slower than Insert,
// typically taking μsec instead of nsec.
//
// The bulk table load could be done with [Table.Insert] and then you can
// use InsertPersist, [Table.UpdatePersist] and [Table.DeletePersist] for lock-free lookups.
func (t *Table[V]) InsertPersist(pfx netip.Prefix, val V) *Table[V] {
	if !pfx.IsValid() {
		return t
	}

	// canonicalize prefix
	pfx = pfx.Masked()
	is4 := pfx.Addr().Is4()

	// share size counters; root nodes cloned selectively.
	pt := &Table[V]{
		size4: t.size4,
		size6: t.size6,
	}

	cloneFn := cloneFnFactory[V]()

	// Clone the root node corresponding to the address family:
	// For the address family in use, perform a shallow clone with copy-on-write semantics.
	// The other root node (IPv4 or IPv6) is simply copied by value (shared).
	if is4 {
		pt.root4 = *t.root4.cloneFlat(cloneFn)
		pt.root6 = t.root6
	} else {
		pt.root4 = t.root4
		pt.root6 = *t.root6.cloneFlat(cloneFn)
	}

	// Get a pointer to the root node we will modify in this operation.
	n := pt.rootNodeByVersion(is4)

	// Insert the prefix and value using the persist insert method that clones nodes
	// along the path. If insertAtDepthPersist returns true, the prefix existed,
	// so no size increment is necessary.
	if n.insertAtDepthPersist(cloneFn, pfx, val, 0) {
		return pt
	}

	// True insert: prefix did not previously exist.
	// Update the prefix count accordingly.
	pt.sizeUpdate(is4, 1)

	return pt
}

// UpdatePersist is similar to Update but does not modify the receiver.
//
// It performs a copy-on-write update, cloning all nodes touched during the update,
// and returns a new Table instance reflecting the update.
// Untouched nodes remain shared between the original and returned Tables.
//
// If the payload type V contains pointers or needs deep copying,
// it must implement the [bart.Cloner] interface to support correct cloning.
//
// Due to cloning overhead, UpdatePersist is significantly slower than Update,
// typically taking μsec instead of nsec.
func (t *Table[V]) UpdatePersist(pfx netip.Prefix, cb func(val V, ok bool) V) (pt *Table[V], newVal V) {
	var zero V // zero value of V for default initialization

	if !pfx.IsValid() {
		return t, zero
	}

	// Normalize prefix by masking host bits.
	pfx = pfx.Masked()

	// Extract address, version info and prefix length.
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()

	// share size counters; root nodes cloned selectively.
	pt = &Table[V]{
		size4: t.size4,
		size6: t.size6,
	}

	cloneFn := cloneFnFactory[V]()

	// Clone root node corresponding to the IP version, for copy-on-write.
	if is4 {
		pt.root4 = *t.root4.cloneFlat(cloneFn)
		pt.root6 = t.root6
	} else {
		pt.root4 = t.root4
		pt.root6 = *t.root6.cloneFlat(cloneFn)
	}

	// Prepare traversal info.
	maxDepth, lastBits := maxDepthAndLastBits(bits)
	octets := ip.AsSlice()

	// Select the root node to operate on.
	n := pt.rootNodeByVersion(is4)

	// Traverse the trie by octets to find the node to update.
	for depth, octet := range octets {
		// If at the last relevant octet, update or insert the prefix in this node.
		if depth == maxDepth {
			newVal, exists := n.prefixes.UpdateAt(art.PfxToIdx(octet, lastBits), cb)
			// If prefix did not previously exist, increment size counter.
			if !exists {
				pt.sizeUpdate(is4, 1)
			}
			return pt, newVal
		}

		addr := octet

		// If child node for this address does not exist, insert new leaf or fringe.
		if !n.children.Test(addr) {
			newVal := cb(zero, false)
			if isFringe(depth, bits) {
				n.children.InsertAt(addr, newFringeNode(newVal))
			} else {
				n.children.InsertAt(addr, newLeafNode(pfx, newVal))
			}

			// New prefix addition updates size.
			pt.sizeUpdate(is4, 1)
			return pt, newVal
		}

		// Child exists - retrieve it.
		kid := n.children.MustGet(addr)

		// kid is node or leaf at addr
		switch kid := kid.(type) {
		case *node[V]:
			// Clone the node along the traversed path to respect copy-on-write.
			kid = kid.cloneFlat(cloneFn)

			// Replace original child with the cloned child.
			n.children.InsertAt(addr, kid)

			// Descend into cloned child for further traversal.
			n = kid
			continue

		case *leafNode[V]:
			// If the leaf's prefix matches, update the value using callback.
			if kid.prefix == pfx {
				newVal = cb(kid.value, true)

				// Replace the existing leaf with an updated one.
				n.children.InsertAt(addr, newLeafNode(pfx, newVal))

				return pt, newVal
			}

			// Prefixes differ - need to push existing leaf down the trie,
			// create a new internal node, and insert the original leaf under it.
			newNode := new(node[V])
			newNode.insertAtDepth(kid.prefix, kid.value, depth+1)

			// Replace leaf with new node and descend.
			n.children.InsertAt(addr, newNode)
			n = newNode

		case *fringeNode[V]:
			// If current node corresponds to a fringe prefix, update its value.
			if isFringe(depth, bits) {
				newVal = cb(kid.value, true)
				// Replace fringe node with updated value.
				n.children.InsertAt(addr, newFringeNode(newVal))
				return pt, newVal
			}

			// Else convert fringe node into an internal node with fringe value
			// pushed down as default route (idx=1).
			newNode := new(node[V])
			newNode.prefixes.InsertAt(1, kid.value)

			// Replace fringe with newly created internal node and descend.
			n.children.InsertAt(addr, newNode)
			n = newNode

		default:
			// Unexpected node type indicates logic error.
			panic("logic error, wrong node type")
		}
	}

	// Should never reach here: the loop should always return or panic.
	panic("unreachable")
}

// DeletePersist is similar to Delete but does not modify the receiver.
//
// It performs a copy-on-write delete operation, cloning all nodes touched during
// deletion and returning a new Table reflecting the change.
//
// If the payload type V contains pointers or requires deep copying,
// it must implement the [bart.Cloner] interface for correct cloning.
//
// Due to cloning overhead, DeletePersist is significantly slower than Delete,
// typically taking μsec instead of nsec.
func (t *Table[V]) DeletePersist(pfx netip.Prefix) *Table[V] {
	pt, _, _ := t.getAndDeletePersist(pfx)
	return pt
}

// GetAndDeletePersist is similar to GetAndDelete but does not modify the receiver.
//
// It performs a copy-on-write delete operation, cloning all nodes touched during
// deletion and returning a new Table reflecting the change.
//
// If the payload type V contains pointers or requires deep copying,
// it must implement the [bart.Cloner] interface for correct cloning.
//
// Due to cloning overhead, GetAndDeletePersist is significantly slower than GetAndDelete,
// typically taking μsec instead of nsec.
func (t *Table[V]) GetAndDeletePersist(pfx netip.Prefix) (pt *Table[V], val V, ok bool) {
	return t.getAndDeletePersist(pfx)
}

// getAndDeletePersist is the internal implementation of GetAndDeletePersist,
// performing the copy-on-write delete without modifying the receiver.
func (t *Table[V]) getAndDeletePersist(pfx netip.Prefix) (pt *Table[V], val V, exists bool) {
	if !pfx.IsValid() {
		return t, val, false
	}

	// Normalize prefix by masking host bits.
	pfx = pfx.Masked()

	// Extract address, IP version, and prefix length.
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()

	// root nodes cloned selectively for copy-on-write.
	pt = &Table[V]{
		size4: t.size4,
		size6: t.size6,
	}

	cloneFn := cloneFnFactory[V]()

	// Clone the root node for the IP version involved.
	if is4 {
		pt.root4 = *t.root4.cloneFlat(cloneFn)
		pt.root6 = t.root6
	} else {
		pt.root4 = t.root4
		pt.root6 = *t.root6.cloneFlat(cloneFn)
	}

	// Prepare traversal context.
	maxDepth, lastBits := maxDepthAndLastBits(bits)
	octets := ip.AsSlice()

	// Stack to keep track of cloned nodes along the path,
	// needed for purge and path compression after delete.
	stack := [maxTreeDepth]*node[V]{}

	// Start at the root node for the given IP version.
	n := pt.rootNodeByVersion(is4)

	// Traverse the trie to locate the prefix to delete.
	for depth, octet := range octets {
		// Keep track of the cloned node at current depth.
		stack[depth] = n

		if depth == maxDepth {
			// Attempt to delete the prefix from the node's prefixes.
			val, exists = n.prefixes.DeleteAt(art.PfxToIdx(octet, lastBits))
			if !exists {
				// Prefix not found, nothing deleted.
				return pt, val, false
			}

			// Adjust stored prefix count for deletion.
			pt.sizeUpdate(is4, -1)

			// After deletion, purge nodes and compress the path if needed.
			n.purgeAndCompress(stack[:depth], octets, is4)

			return pt, val, exists
		}

		addr := octet

		// If child node doesn't exist, no prefix to delete.
		if !n.children.Test(addr) {
			return pt, val, false
		}

		// Fetch child node at current address.
		kid := n.children.MustGet(addr)

		switch kid := kid.(type) {
		case *node[V]:
			// Clone the internal node for copy-on-write.
			kid = kid.cloneFlat(cloneFn)

			// Replace child with cloned node.
			n.children.InsertAt(addr, kid)

			// Descend to cloned child node.
			n = kid
			continue

		case *fringeNode[V]:
			// Reached a path compressed fringe.
			if !isFringe(depth, bits) {
				// Prefix to delete not found here.
				return pt, val, false
			}

			// Delete the fringe node.
			n.children.DeleteAt(addr)

			// Update size to reflect deletion.
			pt.sizeUpdate(is4, -1)

			// Purge and compress affected path.
			n.purgeAndCompress(stack[:depth], octets, is4)

			return pt, kid.value, true

		case *leafNode[V]:
			// Reached a path compressed leaf node.
			if kid.prefix != pfx {
				// Leaf prefix does not match; nothing to delete.
				return pt, val, false
			}

			// Delete leaf node.
			n.children.DeleteAt(addr)

			// Update size to reflect deletion.
			pt.sizeUpdate(is4, -1)

			// Purge and compress affected path.
			n.purgeAndCompress(stack[:depth], octets, is4)

			return pt, kid.value, true

		default:
			// Unexpected node type indicates a logic error.
			panic("logic error, wrong node type")
		}
	}

	// Should never happen: traversal always returns or panics inside loop.
	panic("unreachable")
}
