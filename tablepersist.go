// Copyright (c) 2025 Karl Gaissmaier
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

	// Extract address, IP version, and prefix length.
	ip := pfx.Addr()
	is4 := ip.Is4()

	// share size counters; root nodes cloned selectively.
	pt := &Table[V]{
		size4: t.size4,
		size6: t.size6,
	}

	// Pointer to the root node we will modify in this operation.
	var n *bartNode[V]

	// Create a cloning function for deep copying values;
	// returns nil if V does not implement the Cloner interface.
	cloneFn := cloneFnFactory[V]()

	// Clone root node corresponding to the IP version, for copy-on-write.
	if is4 {
		pt.root6 = t.root6
		pt.root4 = *t.root4.cloneFlat(cloneFn)

		n = &pt.root4
	} else {
		pt.root4 = t.root4
		pt.root6 = *t.root6.cloneFlat(cloneFn)

		n = &pt.root6
	}

	// Prepare traversal info.
	lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)
	octets := ip.AsSlice()

	// Insert the prefix and value using the persist insert method that clones nodes
	// along the path.
	for depth, octet := range octets {
		// last masked octet: insert/override prefix/val into node
		if depth == lastOctetPlusOne {
			exists := n.insertPrefix(art.PfxToIdx(octet, lastBits), val)
			// If prefix did not previously exist, increment size counter.
			if !exists {
				pt.sizeUpdate(is4, 1)
			}
			return pt
		}

		if !n.children.Test(octet) {
			// insert prefix path compressed as leaf or fringe
			if isFringe(depth, pfx) {
				n.insertChild(octet, newFringeNode(val))
			} else {
				n.insertChild(octet, newLeafNode(pfx, val))
			}

			// New prefix addition path compressed, update size.
			pt.sizeUpdate(is4, 1)
			return pt
		}

		kid := n.mustGetChild(octet)

		// kid is node or leaf or fringe at octet
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
				return pt
			}

			// create new node
			// push the leaf down
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(bartNode[V])
			newNode.insertAtDepth(kid.prefix, kid.value, depth+1)

			n.insertChild(octet, newNode)
			n = newNode

		case *fringeNode[V]:
			// reached a path compressed fringe
			// override value in slot if pfx is a fringe
			if isFringe(depth, pfx) {
				kid.value = val
				// exists
				return pt
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

	// Should never happen: traversal always returns or panics inside loop.
	panic("unreachable")
}

// Deprecated: use [Table.ModifyPersist] instead.
//
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

	// canonicalize prefix
	pfx = pfx.Masked()

	// Extract address, version info and prefix length.
	ip := pfx.Addr()
	is4 := ip.Is4()

	// share size counters; root nodes cloned selectively.
	pt = &Table[V]{
		size4: t.size4,
		size6: t.size6,
	}

	// Pointer to the root node we will modify in this operation.
	var n *bartNode[V]

	// Create a cloning function for deep copying values;
	// returns nil if V does not implement the Cloner interface.
	cloneFn := cloneFnFactory[V]()

	// Clone root node corresponding to the IP version, for copy-on-write.
	if is4 {
		pt.root6 = t.root6
		pt.root4 = *t.root4.cloneFlat(cloneFn)

		n = &pt.root4
	} else {
		pt.root4 = t.root4
		pt.root6 = *t.root6.cloneFlat(cloneFn)

		n = &pt.root6
	}

	// Prepare traversal info.
	lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)
	octets := ip.AsSlice()

	// Traverse the trie by octets to find the node to update.
	for depth, octet := range octets {
		// If at the last relevant octet, update or insert the prefix in this node.
		if depth == lastOctetPlusOne {
			idx := art.PfxToIdx(octet, lastBits)

			oldVal, existed := n.getPrefix(idx)
			newVal := cb(oldVal, existed)
			n.insertPrefix(idx, newVal)

			if !existed {
				pt.sizeUpdate(is4, 1)
			}
			return pt, newVal
		}

		addr := octet

		// If child node for this address does not exist, insert new leaf or fringe.
		if !n.children.Test(addr) {
			newVal := cb(zero, false)
			if isFringe(depth, pfx) {
				n.insertChild(addr, newFringeNode(newVal))
			} else {
				n.insertChild(addr, newLeafNode(pfx, newVal))
			}

			// New prefix addition updates size.
			pt.sizeUpdate(is4, 1)
			return pt, newVal
		}

		// Child exists - retrieve it.
		kid := n.mustGetChild(addr)

		// kid is node or leaf at addr
		switch kid := kid.(type) {
		case *bartNode[V]:
			// Clone the node along the traversed path to respect copy-on-write.
			kid = kid.cloneFlat(cloneFn)

			// Replace original child with the cloned child.
			n.insertChild(addr, kid)

			// Descend into cloned child for further traversal.
			n = kid
			continue

		case *leafNode[V]:
			// If the leaf's prefix matches, update the value using callback.
			if kid.prefix == pfx {
				newVal = cb(kid.value, true)

				// Replace the existing leaf with an updated one.
				n.insertChild(addr, newLeafNode(pfx, newVal))

				return pt, newVal
			}

			// Prefixes differ - need to push existing leaf down the trie,
			// create a new internal node, and insert the original leaf under it.
			newNode := new(bartNode[V])
			newNode.insertAtDepth(kid.prefix, kid.value, depth+1)

			// Replace leaf with new node and descend.
			n.insertChild(addr, newNode)
			n = newNode

		case *fringeNode[V]:
			// If current node corresponds to a fringe prefix, update its value.
			if isFringe(depth, pfx) {
				newVal = cb(kid.value, true)
				// Replace fringe node with updated value.
				n.insertChild(addr, newFringeNode(newVal))
				return pt, newVal
			}

			// Else convert fringe node into an internal node with fringe value
			// pushed down as default route (idx=1).
			newNode := new(bartNode[V])
			newNode.insertPrefix(1, kid.value)

			// Replace fringe with newly created internal node and descend.
			n.insertChild(addr, newNode)
			n = newNode

		default:
			// Unexpected node type indicates logic error.
			panic("logic error, wrong node type")
		}
	}

	// Should never reach here: the loop should always return or panic.
	panic("unreachable")
}

// ModifyPersist is similar to Modify but does not modify the receiver.
//
// It performs a copy-on-write update, cloning all nodes touched during the update,
// and returns a new Table instance reflecting the update.
// Untouched nodes remain shared between the original and returned Tables.
//
// If the payload type V contains pointers or needs deep copying,
// it must implement the [bart.Cloner] interface to support correct cloning.
//
// Due to cloning overhead, ModifyPersist is significantly slower than Modify,
// typically taking μsec instead of nsec.
func (t *Table[V]) ModifyPersist(pfx netip.Prefix, cb func(val V, ok bool) (newVal V, del bool)) (pt *Table[V], newVal V, deleted bool) {
	var zero V // zero value of V for default initialization

	if !pfx.IsValid() {
		return t, zero, false
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	// Extract address, version info and prefix length.
	ip := pfx.Addr()
	is4 := ip.Is4()

	// share size counters; root nodes cloned selectively.
	pt = &Table[V]{
		size4: t.size4,
		size6: t.size6,
	}

	// Pointer to the root node we will modify in this operation.
	var n *bartNode[V]

	// Create a cloning function for deep copying values;
	// returns nil if V does not implement the Cloner interface.
	cloneFn := cloneFnFactory[V]()

	// Clone root node corresponding to the IP version, for copy-on-write.
	if is4 {
		pt.root6 = t.root6
		pt.root4 = *t.root4.cloneFlat(cloneFn)

		n = &pt.root4
	} else {
		pt.root4 = t.root4
		pt.root6 = *t.root6.cloneFlat(cloneFn)

		n = &pt.root6
	}

	// Prepare traversal info.
	lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)
	octets := ip.AsSlice()

	// record the nodes on the path to the deleted node, needed to purge
	// and/or path compress nodes after the deletion of a prefix
	stack := [maxTreeDepth]*bartNode[V]{}

	// find the proper trie node to update prefix
	for depth, octet := range octets {
		// push current node on stack for path recording
		stack[depth] = n

		// last octet from prefix, update/insert/delete prefix
		if depth == lastOctetPlusOne {
			idx := art.PfxToIdx(octet, lastBits)

			oldVal, existed := n.getPrefix(idx)
			newVal, del := cb(oldVal, existed)

			// update size if necessary
			switch {
			case !existed && del: // no-op
				return pt, zero, false

			case existed && del: // delete
				n.deletePrefix(idx)
				pt.sizeUpdate(is4, -1)
				n.purgeAndCompress(stack[:depth], octets, is4)
				return pt, oldVal, true

			case !existed: // insert
				n.insertPrefix(idx, newVal)
				pt.sizeUpdate(is4, 1)
				return pt, newVal, false

			case existed: // update
				n.insertPrefix(idx, newVal)
				return pt, oldVal, false

			default:
				panic("unreachable")
			}
		}

		// go down in tight loop to last octet
		if !n.children.Test(octet) {
			// insert prefix path compressed

			newVal, del := cb(zero, false)
			if del { // no-op
				return pt, zero, false
			}

			// insert
			if isFringe(depth, pfx) {
				n.insertChild(octet, newFringeNode(newVal))
			} else {
				n.insertChild(octet, newLeafNode(pfx, newVal))
			}

			pt.sizeUpdate(is4, 1)
			return pt, newVal, false
		}

		// Child exists - retrieve it.
		kid := n.mustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *bartNode[V]:
			// Clone the node along the traversed path to respect copy-on-write.
			kid = kid.cloneFlat(cloneFn)

			// Replace original child with the cloned child.
			n.insertChild(octet, kid)

			n = kid // descend down to next trie level

		case *leafNode[V]:
			oldVal := kid.value

			// update existing value if prefixes are equal
			if kid.prefix == pfx {
				newVal, del := cb(oldVal, true)
				if !del {
					kid.value = newVal
					return pt, oldVal, false // update
				}

				// delete
				n.deleteChild(octet)

				pt.sizeUpdate(is4, -1)
				n.purgeAndCompress(stack[:depth], octets, is4)

				return pt, oldVal, true
			}

			// create new node
			// push the leaf down
			// insert new child at current leaf position (octet
			// descend down, replace n with new child
			newNode := new(bartNode[V])
			newNode.insertAtDepth(kid.prefix, kid.value, depth+1)

			n.insertChild(octet, newNode)
			n = newNode

		case *fringeNode[V]:
			oldVal := kid.value

			// update existing value if prefix is fringe
			if isFringe(depth, pfx) {
				newVal, del := cb(oldVal, true)
				if !del {
					kid.value = newVal
					return pt, oldVal, false // update
				}

				// delete
				n.deleteChild(octet)

				pt.sizeUpdate(is4, -1)
				n.purgeAndCompress(stack[:depth], octets, is4)

				return pt, oldVal, true
			}

			// create new node
			// push the fringe down, it becomes a default route (idx=1)
			// insert new child at current leaf position (octet
			// descend down, replace n with new child
			newNode := new(bartNode[V])
			newNode.insertPrefix(1, kid.value)

			n.insertChild(octet, newNode)
			n = newNode

		default:
			panic("logic error, wrong node type")
		}
	}

	// Should never reach here: the loop should always return or panic.
	panic("unreachable")
}

// Deprecated: use [Table.DeletePersist] instead.
func (t *Table[V]) GetAndDeletePersist(pfx netip.Prefix) (pt *Table[V], val V, found bool) {
	return t.DeletePersist(pfx)
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
func (t *Table[V]) DeletePersist(pfx netip.Prefix) (pt *Table[V], val V, found bool) {
	if !pfx.IsValid() {
		return t, val, false
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	// Extract address, IP version, and prefix length.
	ip := pfx.Addr()
	is4 := ip.Is4()

	// share size counters; root nodes cloned selectively.
	pt = &Table[V]{
		size4: t.size4,
		size6: t.size6,
	}

	// Pointer to the root node we will modify in this operation.
	var n *bartNode[V]

	// Create a cloning function for deep copying values;
	// returns nil if V does not implement the Cloner interface.
	cloneFn := cloneFnFactory[V]()

	// Clone root node corresponding to the IP version, for copy-on-write.
	if is4 {
		pt.root6 = t.root6
		pt.root4 = *t.root4.cloneFlat(cloneFn)

		n = &pt.root4
	} else {
		pt.root4 = t.root4
		pt.root6 = *t.root6.cloneFlat(cloneFn)

		n = &pt.root6
	}

	// Prepare traversal context.
	lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)
	octets := ip.AsSlice()

	// Stack to keep track of cloned nodes along the path,
	// needed for purge and path compression after delete.
	stack := [maxTreeDepth]*bartNode[V]{}

	// Traverse the trie to locate the prefix to delete.
	for depth, octet := range octets {
		// Keep track of the cloned node at current depth.
		stack[depth] = n

		if depth == lastOctetPlusOne {
			// Attempt to delete the prefix from the node's prefixes.
			val, found = n.deletePrefix(art.PfxToIdx(octet, lastBits))
			if !found {
				// Prefix not found, nothing deleted.
				return pt, val, false
			}

			// Adjust stored prefix count for deletion.
			pt.sizeUpdate(is4, -1)

			// After deletion, purge nodes and compress the path if needed.
			n.purgeAndCompress(stack[:depth], octets, is4)

			return pt, val, true
		}

		addr := octet

		// If child node doesn't exist, no prefix to delete.
		if !n.children.Test(addr) {
			return pt, val, false
		}

		// Fetch child node at current address.
		kid := n.mustGetChild(addr)

		switch kid := kid.(type) {
		case *bartNode[V]:
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
				return pt, val, false
			}

			// Delete the fringe node.
			n.deleteChild(addr)

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
			n.deleteChild(addr)

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

// WalkPersist traverses all prefix/value pairs in the table and calls the
// provided callback function for each entry. The callback receives the
// current persistent table, the prefix, and the associated value.
//
// The callback must return a (potentially updated) persistent table and a
// boolean flag indicating whether traversal should continue. Returning
// false stops the iteration early.
//
// IMPORTANT: It is the responsibility of the callback implementation to only
// use persistent Table operations (e.g. InsertPersist, DeletePersist,
// UpdatePersist, ...). Using mutating methods like Update or Delete
// inside the callback would break the iteration and may lead
// to inconsistent results.
//
// Example:
//
//	pt := t.WalkPersist(func(pt *Table[int], pfx netip.Prefix, val int) (*Table[int], bool) {
//		switch {
//		// Stop iterating if value is <0
//		case val < 0:
//			return pt, false
//
//		// Delete entries with value 0
//		case val == 0:
//			pt = pt.DeletePersist(pfx)
//
//		// Update even values by doubling them
//		case val%2 == 0:
//			pt, _ = pt.UpdatePersist(pfx, func(oldVal int, _ bool) int {
//				return oldVal * 2
//			})
//
//		// Leave odd values unchanged
//		default:
//			// no-op
//		}
//
//		// Continue iterating
//		return pt, true
//	})
func (t *Table[V]) WalkPersist(fn func(*Table[V], netip.Prefix, V) (*Table[V], bool)) *Table[V] {
	// no-op, callback is nil
	if fn == nil {
		return t
	}

	// create shallow persistent copy
	pt := &Table[V]{
		root4: t.root4,
		root6: t.root6,
		size4: t.size4,
		size6: t.size6,
	}

	var proceed bool
	for pfx, val := range t.All() {
		if pt, proceed = fn(pt, pfx, val); !proceed {
			break
		}
	}
	return pt
}
