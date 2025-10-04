// Code generated from file "nodebasics_tmpl.go"; DO NOT EDIT.

// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package nodes

import (
	"fmt"
	"io"
	"net/netip"
	"slices"
	"strings"

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
func (n *BartNode[V]) Insert(pfx netip.Prefix, val V, depth int) (exists bool) {
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
		case *BartNode[V]:
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
			newNode := new(BartNode[V])
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
			newNode := new(BartNode[V])
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
func (n *BartNode[V]) InsertPersist(cloneFn CloneFunc[V], pfx netip.Prefix, val V, depth int) (exists bool) {
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
		case *BartNode[V]:
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
			newNode := new(BartNode[V])
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
			newNode := new(BartNode[V])
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
func (n *BartNode[V]) PurgeAndCompress(stack []*BartNode[V], octets []uint8, is4 bool) {
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
			case *BartNode[V]:
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
func (n *BartNode[V]) Delete(pfx netip.Prefix) (exists bool) {
	// invariant, prefix must be masked

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := LastOctetPlusOneAndLastBits(pfx)

	// record the nodes on the path to the deleted node, needed to purge
	// and/or path compress nodes after the deletion of a prefix
	stack := [MaxTreeDepth]*BartNode[V]{}

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
		case *BartNode[V]:
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
func (n *BartNode[V]) DeletePersist(cloneFn CloneFunc[V], pfx netip.Prefix) (exists bool) {
	ip := pfx.Addr() // the pfx must be in canonical form
	is4 := ip.Is4()
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := LastOctetPlusOneAndLastBits(pfx)

	// Stack to keep track of cloned nodes along the path,
	// needed for purge and path compression after delete.
	stack := [MaxTreeDepth]*BartNode[V]{}

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
		case *BartNode[V]:
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
func (n *BartNode[V]) Get(pfx netip.Prefix) (val V, exists bool) {
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
		case *BartNode[V]:
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
func (n *BartNode[V]) Modify(pfx netip.Prefix, cb func(val V, found bool) (_ V, del bool)) (delta int) {
	var zero V

	ip := pfx.Addr()
	is4 := ip.Is4()
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := LastOctetPlusOneAndLastBits(pfx)

	// record the nodes on the path to the deleted node, needed to purge
	// and/or path compress nodes after the deletion of a prefix
	stack := [MaxTreeDepth]*BartNode[V]{}

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
		case *BartNode[V]:
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
			newNode := new(BartNode[V])
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
			newNode := new(BartNode[V])
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
func (n *BartNode[V]) EqualRec(o *BartNode[V]) bool {
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
		case *BartNode[V]:
			// oKid must also be a node
			oKid, ok := oKid.(*BartNode[V])
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

// DumpRec recursively descends the trie rooted at n and writes a human-readable
// representation of each visited node to w.
//
// It returns immediately if n is nil or empty. For each visited internal node
// it calls dump to write the node's representation, then iterates its child
// addresses and recurses into children that implement nodeDumper[V] (internal
// subnodes). The path slice and depth together represent the byte-wise path
// from the root to the current node; depth is incremented for each recursion.
// The is4 flag controls IPv4/IPv6 formatting used by dump.
func (n *BartNode[V]) DumpRec(w io.Writer, path StridePath, depth int, is4 bool, printVals bool) {
	if n == nil || n.IsEmpty() {
		return
	}

	// dump this node
	n.Dump(w, path, depth, is4, printVals)

	// node may have children, rec-descent down
	for addr, child := range n.AllChildren() {
		if kid, ok := child.(*BartNode[V]); ok {
			path[depth] = addr
			kid.DumpRec(w, path, depth+1, is4, printVals)
		}
	}
}

// Dump writes a human-readable representation of the node to `w`.
// It prints the node type, depth, formatted path (IPv4 vs IPv6 controlled by `is4`),
// and bit count, followed by any stored prefixes (and their values when applicable),
// the set of child octets, and any path-compressed leaves or fringe entries.
func (n *BartNode[V]) Dump(w io.Writer, path StridePath, depth int, is4 bool, printVals bool) {
	bits := depth * strideLen
	indent := strings.Repeat(".", depth)

	// node type with depth and octet path and bits.
	fmt.Fprintf(w, "\n%s[%s] depth:  %d path: [%s] / %d\n",
		indent, n.hasType(), depth, ipStridePath(path, depth, is4), bits)

	if nPfxCount := n.PrefixCount(); nPfxCount != 0 {
		var buf [256]uint8
		allIndices := n.GetIndices(&buf)

		// print the baseIndices for this node.
		fmt.Fprintf(w, "%sindexs(#%d): %v\n", indent, nPfxCount, allIndices)

		// print the prefixes for this node
		fmt.Fprintf(w, "%sprefxs(#%d):", indent, nPfxCount)

		for _, idx := range allIndices {
			pfx := CidrFromPath(path, depth, is4, idx)
			fmt.Fprintf(w, " %s", pfx)
		}

		fmt.Fprintln(w)

		// skip values, maybe the payload is the empty struct
		if printVals {

			// print the values for this node
			fmt.Fprintf(w, "%svalues(#%d):", indent, nPfxCount)

			for _, idx := range allIndices {
				val := n.MustGetPrefix(idx)
				fmt.Fprintf(w, " %#v", val)
			}

			fmt.Fprintln(w)
		}
	}

	if n.ChildCount() != 0 {
		allAddrs := make([]uint8, 0, MaxItems)
		childAddrs := make([]uint8, 0, MaxItems)
		leafAddrs := make([]uint8, 0, MaxItems)
		fringeAddrs := make([]uint8, 0, MaxItems)

		// the node has recursive child nodes or path-compressed leaves
		for addr, child := range n.AllChildren() {
			allAddrs = append(allAddrs, addr)

			switch child.(type) {
			case *BartNode[V]:
				childAddrs = append(childAddrs, addr)
				continue

			case *FringeNode[V]:
				fringeAddrs = append(fringeAddrs, addr)

			case *LeafNode[V]:
				leafAddrs = append(leafAddrs, addr)

			default:
				panic("logic error, wrong node type")
			}
		}

		// print the children for this node.
		fmt.Fprintf(w, "%soctets(#%d): %v\n", indent, len(allAddrs), allAddrs)

		if leafCount := len(leafAddrs); leafCount > 0 {
			// print the pathcomp prefixes for this node
			fmt.Fprintf(w, "%sleaves(#%d):", indent, leafCount)

			for _, addr := range leafAddrs {
				kid := n.MustGetChild(addr).(*LeafNode[V])
				if printVals {
					fmt.Fprintf(w, " %s:{%s, %v}", addrFmt(addr, is4), kid.Prefix, kid.Value)
				} else {
					fmt.Fprintf(w, " %s:{%s}", addrFmt(addr, is4), kid.Prefix)
				}
			}

			fmt.Fprintln(w)
		}

		if fringeCount := len(fringeAddrs); fringeCount > 0 {
			// print the pathcomp prefixes for this node
			fmt.Fprintf(w, "%sfringe(#%d):", indent, fringeCount)

			for _, addr := range fringeAddrs {
				fringePfx := CidrForFringe(path[:], depth, is4, addr)

				kid := n.MustGetChild(addr).(*FringeNode[V])
				if printVals {
					fmt.Fprintf(w, " %s:{%s, %v}", addrFmt(addr, is4), fringePfx, kid.Value)
				} else {
					fmt.Fprintf(w, " %s:{%s}", addrFmt(addr, is4), fringePfx)
				}
			}

			fmt.Fprintln(w)
		}

		if childCount := len(childAddrs); childCount > 0 {
			// print the next child
			fmt.Fprintf(w, "%schilds(#%d):", indent, childCount)

			for _, addr := range childAddrs {
				fmt.Fprintf(w, " %s", addrFmt(addr, is4))
			}

			fmt.Fprintln(w)
		}

	}
}

// hasType classifies the given node into one of the nodeType values.
//
// It inspects immediate statistics (prefix count, child count, node, leaf and
// fringe counts) for the node and returns:
//   - nullNode: no prefixes and no children
//   - stopNode: has children but no subnodes (nodes == 0)
//   - halfNode: contains at least one leaf or fringe and also has subnodes, but
//     no prefixes
//   - fullNode: has prefixes or leaves/fringes and also has subnodes
//   - pathNode: has subnodes only (no prefixes, leaves, or fringes)
//
// The order of these checks is significant to ensure the correct classification.
func (n *BartNode[V]) hasType() nodeType {
	s := n.Stats()

	// the order is important
	switch {
	case s.Pfxs == 0 && s.Childs == 0:
		return nullNode
	case s.Nodes == 0:
		return stopNode
	case (s.Leaves > 0 || s.Fringes > 0) && s.Nodes > 0 && s.Pfxs == 0:
		return halfNode
	case (s.Pfxs > 0 || s.Leaves > 0 || s.Fringes > 0) && s.Nodes > 0:
		return fullNode
	case (s.Pfxs == 0 && s.Leaves == 0 && s.Fringes == 0) && s.Nodes > 0:
		return pathNode
	default:
		panic(fmt.Sprintf("UNREACHABLE: pfx: %d, chld: %d, node: %d, leaf: %d, fringe: %d",
			s.Pfxs, s.Childs, s.Nodes, s.Leaves, s.Fringes))
	}
}

// Stats returns immediate statistics for n: counts of prefixes and children,
// and a classification of each child into nodes, leaves, or fringes.
// It inspects only the direct children of n (not the whole subtree).
// Panics if a child has an unexpected concrete type.
func (n *BartNode[V]) Stats() (s StatsT) {
	s.Pfxs = n.PrefixCount()
	s.Childs = n.ChildCount()

	for _, child := range n.AllChildren() {
		switch child.(type) {
		case *BartNode[V]:
			s.Nodes++

		case *FringeNode[V]:
			s.Fringes++

		case *LeafNode[V]:
			s.Leaves++

		default:
			panic("logic error, wrong node type")
		}
	}

	return s
}

// StatsRec returns aggregated statistics for the subtree rooted at n.
//
// It walks the node tree recursively and sums immediate counts (prefixes and
// child slots) plus the number of nodes, leaves, and fringe nodes in the
// subtree. If n is nil or empty, a zeroed stats is returned. The returned
// stats.nodes includes the current node. The function will panic if a child
// has an unexpected concrete type.
func (n *BartNode[V]) StatsRec() (s StatsT) {
	if n == nil || n.IsEmpty() {
		return s
	}

	s.Pfxs = n.PrefixCount()
	s.Childs = n.ChildCount()
	s.Nodes = 1 // this node
	s.Leaves = 0
	s.Fringes = 0

	for _, child := range n.AllChildren() {
		switch kid := child.(type) {
		case *BartNode[V]:
			// rec-descent
			rs := kid.StatsRec()

			s.Pfxs += rs.Pfxs
			s.Childs += rs.Childs
			s.Nodes += rs.Nodes
			s.Leaves += rs.Leaves
			s.Fringes += rs.Fringes

		case *FringeNode[V]:
			s.Fringes++

		case *LeafNode[V]:
			s.Leaves++

		default:
			panic("logic error, wrong node type")
		}
	}

	return s
}

// FprintRec recursively prints a hierarchical CIDR tree representation
// starting from this node to the provided writer. The output shows the
// routing table structure in human-readable format for debugging and analysis.
func (n *BartNode[V]) FprintRec(w io.Writer, parent TrieItem[V], pad string, printVals bool) error {
	// recursion stop condition
	if n == nil || n.IsEmpty() {
		return nil
	}

	// get direct covered childs for this parent ...
	directItems := n.DirectItemsRec(parent.Idx, parent.Path, parent.Depth, parent.Is4)

	// sort them by netip.Prefix, not by baseIndex
	slices.SortFunc(directItems, func(a, b TrieItem[V]) int {
		return CmpPrefix(a.Cidr, b.Cidr)
	})

	// for all direct item under this node ...
	for i, item := range directItems {
		// symbols used in tree
		glyph := "├─ "
		space := "│  "

		// ... treat last kid special
		if i == len(directItems)-1 {
			glyph = "└─ "
			space = "   "
		}

		var err error
		// val is the empty struct, don't print it
		switch {
		case !printVals:
			_, err = fmt.Fprintf(w, "%s%s\n", pad+glyph, item.Cidr)
		default:
			_, err = fmt.Fprintf(w, "%s%s (%v)\n", pad+glyph, item.Cidr, item.Val)
		}

		if err != nil {
			return err
		}

		// rec-descent with this item as parent
		nextNode, _ := item.Node.(*BartNode[V])
		if err = nextNode.FprintRec(w, item, pad+space, printVals); err != nil {
			return err
		}
	}

	return nil
}

// DirectItemsRec, returns the direct covered items by parent.
// It's a complex recursive function, you have to know the data structure
// by heart to understand this function!
//
// See the  artlookup.pdf paper in the doc folder, the baseIndex function is the key.
func (n *BartNode[V]) DirectItemsRec(parentIdx uint8, path StridePath, depth int, is4 bool) (directItems []TrieItem[V]) {
	// recursion stop condition
	if n == nil || n.IsEmpty() {
		return nil
	}

	// prefixes:
	// for all idx's (prefixes mapped by baseIndex) in this node
	// do a longest-prefix-match
	for idx, val := range n.AllIndices() {
		// tricky part, skip self
		// test with next possible lpm (idx>>1), it's a complete binary tree
		nextIdx := idx >> 1

		// fast skip, lpm not possible
		if nextIdx < parentIdx {
			continue
		}

		// do a longest-prefix-match
		lpm, _, _ := n.LookupIdx(nextIdx)

		// be aware, 0 is here a possible value for parentIdx and lpm (if not found)
		if lpm == parentIdx {
			// prefix is directly covered by parent

			item := TrieItem[V]{
				Node:  n,
				Is4:   is4,
				Path:  path,
				Depth: depth,
				Idx:   idx,
				// get the prefix back from trie
				Cidr: CidrFromPath(path, depth, is4, idx),
				Val:  val,
			}

			directItems = append(directItems, item)
		}
	}

	// children:
	for addr, child := range n.AllChildren() {
		hostIdx := art.OctetToIdx(addr)

		// do a longest-prefix-match
		lpm, _, _ := n.LookupIdx(hostIdx)

		// be aware, 0 is here a possible value for parentIdx and lpm (if not found)
		if lpm == parentIdx {
			// child is directly covered by parent
			switch kid := child.(type) {
			case *BartNode[V]: // traverse rec-descent, call with next child node,
				// next trie level, set parentIdx to 0, adjust path and depth
				path[depth] = addr
				directItems = append(directItems, kid.DirectItemsRec(0, path, depth+1, is4)...)

			case *LeafNode[V]: // path-compressed child, stop's recursion for this child
				item := TrieItem[V]{
					Node: nil,
					Is4:  is4,
					Cidr: kid.Prefix,
					Val:  kid.Value,
				}
				directItems = append(directItems, item)

			case *FringeNode[V]: // path-compressed fringe, stop's recursion for this child
				item := TrieItem[V]{
					Node: nil,
					Is4:  is4,
					// get the prefix back from trie
					Cidr: CidrForFringe(path[:], depth, is4, addr),
					Val:  kid.Value,
				}
				directItems = append(directItems, item)

			default:
				panic("logic error, wrong node type")
			}
		}
	}

	return directItems
}
