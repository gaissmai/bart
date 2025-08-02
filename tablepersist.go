// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"

	"github.com/gaissmai/bart/internal/art"
)

func (t *Table[V]) InsertSync(pfx netip.Prefix, val V) {
	t.init()

	t.mu.Lock()
	defer t.mu.Unlock()

	if !pfx.IsValid() {
		return
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	is4 := pfx.Addr().Is4()
	n := t.rootNodeByVersion(is4)

	if exists := n.insertAtDepthSync(pfx, val, 0); exists {
		return
	}

	// true insert, update size
	t.sizeUpdate(is4, 1)
}

// UpdateSync TODO
func (t *Table[V]) UpdateSync(pfx netip.Prefix, cb func(val V, ok bool) V) (newVal V) {
	panic("UpdateSync() not yet implemented !!!")
	/*
		t.init()

		t.mu.Lock()
		defer t.mu.Unlock()

		var zero V // zero value of V for default initialization

		if !pfx.IsValid() {
			return zero
		}

		// Normalize prefix by masking host bits.
		pfx = pfx.Masked()

		// Extract address, version info and prefix length.
		ip := pfx.Addr()
		is4 := ip.Is4()
		bits := pfx.Bits()

		// Prepare traversal info.
		maxDepth, lastBits := maxDepthAndLastBits(bits)
		octets := ip.AsSlice()

		// Select the root node to operate on.
		n := t.rootNodeByVersion(is4)

		// Traverse the trie by octets to find the node to update.
		for depth, octet := range octets {
			// If at the last relevant octet, update or insert the prefix in this node.
			if depth == maxDepth {
				// ##########################################
				// TODO
				// ##########################################
				newVal, exists := n.prefixes.Load().UpdateAt(art.PfxToIdx(octet, lastBits), cb)
				// If prefix did not previously exist, increment size counter.
				if !exists {
					t.sizeUpdate(is4, 1)
				}
				return newVal
			}

			addr := octet

			// If child node for this address does not exist, insert new leaf or fringe.
			if !n.children.Load().Test(addr) {
				newVal := cb(zero, false)
				if isFringe(depth, bits) {
					n.children.Load().InsertAt(addr, newFringeNode[V](newVal))
				} else {
					n.children.Load().InsertAt(addr, newLeafNode[V](pfx, newVal))
				}

				// New prefix addition updates size.
				t.sizeUpdate(is4, 1)
				return newVal
			}

			// Child exists - retrieve it.
			kid := n.children.Load().MustGet(addr)

			// kid is node or leaf at addr
			switch kid := kid.(type) {
			case *node[V]:
				// Clone the node along the traversed path to respect copy-on-write.
				kid = kid.cloneFlat()

				// Replace original child with the cloned child.
				n.children.Load().InsertAt(addr, kid)

				// Descend into cloned child for further traversal.
				n = kid
				continue

			case *leafNode[V]:
				// If the leaf's prefix matches, update the value using callback.
				if kid.prefix == pfx {
					newVal = cb(kid.value, true)

					// Replace the existing leaf with an updated one.
					n.children.Load().InsertAt(addr, newLeafNode[V](pfx, newVal))

					return newVal
				}

				// Prefixes differ - need to push existing leaf down the trie,
				// create a new internal node, and insert the original leaf under it.
				newNode := newNode[V]()
				newNode.insertAtDepthSync(kid.prefix, kid.value, depth+1)

				// Replace leaf with new node and descend.
				n.children.Load().InsertAt(addr, newNode)
				n = newNode

			case *fringeNode[V]:
				// If current node corresponds to a fringe prefix, update its value.
				if isFringe(depth, bits) {
					newVal = cb(kid.value, true)
					// Replace fringe node with updated value.
					n.children.Load().InsertAt(addr, newFringeNode[V](newVal))
					return newVal
				}

				// Else convert fringe node into an internal node with fringe value
				// pushed down as default route (idx=1).
				newNode := newNode[V]()
				newNode.prefixes.Load().InsertAt(1, kid.value)

				// Replace fringe with newly created internal node and descend.
				n.children.Load().InsertAt(addr, newNode)
				n = newNode

			default:
				// Unexpected node type indicates logic error.
				panic("logic error, wrong node type")
			}
		}

		// Should never reach here: the loop should always return or panic.
		panic("unreachable")
	*/
}

// DeleteSync TODO
func (t *Table[V]) DeleteSync(pfx netip.Prefix) {
	_, _ = t.getAndDeleteSync(pfx)
}

// GetAndDeleteSync TODO
func (t *Table[V]) GetAndDeleteSync(pfx netip.Prefix) (val V, ok bool) {
	return t.getAndDeleteSync(pfx)
}

// getAndDeletePersist TODO
func (t *Table[V]) getAndDeleteSync(pfx netip.Prefix) (val V, exists bool) {
	t.init()

	t.mu.Lock()
	defer t.mu.Unlock()

	if !pfx.IsValid() {
		return
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()
	octets := ip.AsSlice()
	maxDepth, lastBits := maxDepthAndLastBits(bits)

	n := t.rootNodeByVersion(is4)

	// record the nodes on the path to the deleted node, needed to purge
	// and/or path compress nodes after the deletion of a prefix
	stack := [maxTreeDepth]*node[V]{}

	// find the trie node
	for depth, octet := range octets {
		depth = depth & 0xf // BCE, Delete must be fast

		// push current node on stack for path recording
		stack[depth] = n

		if depth == maxDepth {
			// try to delete prefix in trie node
			clonedPfxs := n.prefixes.Load().Clone(cloneOrCopy)
			val, exists = clonedPfxs.DeleteAt(art.PfxToIdx(octet, lastBits))
			n.prefixes.Store(clonedPfxs)

			if !exists {
				return
			}

			t.sizeUpdate(is4, -1)
			n.purgeAndCompressSync(stack[:depth], octets, is4)
			return val, true
		}

		if !n.children.Load().Test(octet) {
			return
		}

		kid, idx := n.children.Load().MustGet2(octet)
		if kid, ok := kid.(*node[V]); ok {
			n = kid
			continue // descend down to next trie level
		}

		// TODO
		clonedKids := n.children.Load().Clone(cloneLeafOrFringe[V])
		kid = clonedKids.Items[idx]

		switch kid := kid.(type) {
		case *fringeNode[V]:
			// if pfx is no fringe at this depth, fast exit
			if !isFringe(depth, bits) {
				return
			}

			// pfx is fringe at depth, delete fringe
			clonedKids.DeleteAt(octet)
			n.children.Store(clonedKids)

			t.sizeUpdate(is4, -1)
			n.purgeAndCompressSync(stack[:depth], octets, is4)

			return kid.value, true

		case *leafNode[V]:
			// Attention: pfx must be masked to be comparable!
			if kid.prefix != pfx {
				return
			}

			// prefix is equal leaf, delete leaf
			clonedKids.DeleteAt(octet)
			n.children.Store(clonedKids)

			t.sizeUpdate(is4, -1)
			n.purgeAndCompressSync(stack[:depth], octets, is4)

			return kid.value, true

		default:
			panic("logic error, wrong node type")
		}
	}

	return
}
