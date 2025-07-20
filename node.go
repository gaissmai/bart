// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"slices"

	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/lpm"
	"github.com/gaissmai/bart/internal/sparse"
)

const (
	strideLen    = 8   // byte, a multibit trie with stride len 8
	maxTreeDepth = 16  // max 16 bytes for IPv6
	maxItems     = 256 // max 256 prefixes or children in node
)

// stridePath, max 16 octets deep
type stridePath [maxTreeDepth]uint8

// node is a trie level node in the multibit routing table.
//
// Each node contains two conceptually different arrays:
//   - prefixes: representing routes, using a complete binary tree layout
//     driven by the baseIndex() function from the ART algorithm.
//   - children: holding subtries or path-compressed leaves/fringes with
//     a branching factor of 256 (8 bits per stride).
//
// Unlike the original ART, this implementation uses popcount-compressed sparse arrays
// instead of fixed-size arrays. Array slots are not pre-allocated; insertion
// and lookup rely on fast bitset operations and precomputed rank indexes.
//
// See doc/artlookup.pdf for the mapping mechanics and prefix tree details.
type node[V any] struct {
	// prefixes stores routing entries (prefix -> value),
	// laid out as a complete binary tree using baseIndex().
	prefixes sparse.Array256[V]

	// children holds subnodes for the 256 possible next-hop paths
	// at this trie level (8-bit stride).
	//
	// Entries in children may be:
	//   - *node[V]       -> internal child node for further traversal
	//   - *leafNode[V]   -> path-comp. node (depth < maxDepth - 1)
	//   - *fringeNode[V] -> path-comp. node (depth == maxDepth - 1, stride-aligned: /8, /16, ... /128))
	//
	// Note: Both *leafNode and *fringeNode entries are only created by path compression.
	// Prefixes that match exactly at the maximum trie depth (depth == maxDepth) are
	// never stored as children, but always directly in the prefixes array at that level.
	children sparse.Array256[any]
}

// reset clears the internal state of the node by resetting both the
// prefixes and children arrays. Any stored routing entries and subnodes
// are removed, but underlying storage capacity is retained to avoid
// reallocations.
func (n *node[V]) reset() {
	// reset routing entries (prefix -> value)
	n.prefixes.Reset()

	// reset child node references (internal, leaf, fringe)
	n.children.Reset()
}

// isEmpty returns true if node has neither prefixes nor children
func (n *node[V]) isEmpty() bool {
	return n.prefixes.Len() == 0 && n.children.Len() == 0
}

// leafNode is a prefix with value, used as a path compressed child.
type leafNode[V any] struct {
	prefix netip.Prefix
	value  V
}

// fringeNode is a path-compressed leaf with value but without a prefix.
// The prefix of a fringe is solely defined by the position in the trie.
// The fringe-compressiion (no stored prefix) saves a lot of memory,
// but the algorithm is more complex.
type fringeNode[V any] struct {
	value V
}

// isFringe determines whether a prefix qualifies as a "fringe node" -
// that is, a special kind of path-compressed leaf inserted at the final
// possible trie level (depth == maxDepth - 1).
//
// Both "leaves" and "fringes" are path-compressed terminal entries;
// the distinction lies in their position within the trie:
//
//   - A leaf is inserted at any intermediate level if no further stride
//     boundary matches (depth < maxDepth - 1).
//
//   - A fringe is inserted at the last possible stride level
//     (depth == maxDepth - 1) before a prefix would otherwise land
//     as a direct prefix (depth == maxDepth).
//
// Special property:
//   - A fringe acts as a default route for all downstream bit patterns
//     extending beyond its prefix.
//
// Examples:
//
//	e.g. prefix is addr/8, or addr/16, or ... addr/128
//	depth <  maxDepth-1 : a leaf, path-compressed
//	depth == maxDepth-1 : a fringe, path-compressed
//	depth == maxDepth   : a prefix with octet/pfx == 0/0 => idx == 1, a strides default route
//
// Logic:
//   - A prefix qualifies as a fringe if:
//     depth == maxDepth - 1 &&
//     lastBits == 0 (i.e., aligned on stride boundary, /8, /16, ... /128 bits)
func isFringe(depth, bits int) bool {
	maxDepth, lastBits := maxDepthAndLastBits(bits)
	return depth == maxDepth-1 && lastBits == 0
}

// cloneOrCopy returns either a deep copy or a shallow copy of the given value v,
// depending on whether the value type V implements the Cloner[V] interface.
//
// If the provided value implements Cloner[V], its Clone method is invoked to produce
// a deep copy. Otherwise, the value is returned as-is, which yields a shallow copy.
func cloneOrCopy[V any](val V) V {
	if cloner, ok := any(val).(Cloner[V]); ok {
		return cloner.Clone()
	}
	// just a shallow copy
	return val
}

// cloneLeaf creates a copy of the current leafNode[V].
//
// If the stored value implements the Cloner[V] interface, a deep copy of the value
// is produced via cloneOrCopy. Otherwise, a shallow copy of the value is used.
//
// This function preserves the original prefix and clones the value,
// ensuring that the cloned leaf does not alias the original value if cloning is supported.
func (l *leafNode[V]) cloneLeaf() *leafNode[V] {
	return &leafNode[V]{prefix: l.prefix, value: cloneOrCopy(l.value)}
}

// cloneFringe returns a clone of the fringe
// if the value implements the Cloner interface.

// cloneFringe creates a copy of the current fringeNode[V].
//
// If the stored value implements the Cloner[V] interface, it is deep-copied
// via cloneOrCopy. Otherwise, a shallow copy is taken.
//
// Unlike leafNode, fringeNode does not store a prefix path - this method simply clones
// the held value to avoid unintended mutations on shared references.
func (l *fringeNode[V]) cloneFringe() *fringeNode[V] {
	return &fringeNode[V]{value: cloneOrCopy(l.value)}
}

// insertAtDepth inserts a network prefix and its associated value into the
// trie starting at the specified byte depth.
//
// The function walks the prefix address from the given depth and inserts the value either directly into
// the node´s prefix table or as a compressed leaf or fringe node. If a conflicting leaf or fringe exists,
// it is pushed down via a new intermediate node allocated from the pool. Existing entries with the same
// prefix are overwritten.
//
// If the Table.WithPool() was called, the provided pool is used to efficiently allocate
// new *node[V] instances, to minimize allocations during dynamic trie updates.
func (n *node[V]) insertAtDepth(p *pool[V], pfx netip.Prefix, val V, depth int) (exists bool) {
	ip := pfx.Addr() // the pfx must be in canonical form
	bits := pfx.Bits()
	octets := ip.AsSlice()
	maxDepth, lastBits := maxDepthAndLastBits(bits)

	// find the proper trie node to insert prefix
	// start with prefix octet at depth
	for ; depth < len(octets); depth++ {
		octet := octets[depth]

		// last masked octet: insert/override prefix/val into node
		if depth == maxDepth {
			return n.prefixes.InsertAt(art.PfxToIdx(octet, lastBits), val)
		}

		// reached end of trie path ...
		if !n.children.Test(octet) {
			// insert prefix path compressed as leaf or fringe
			if isFringe(depth, bits) {
				return n.children.InsertAt(octet, &fringeNode[V]{val})
			}
			return n.children.InsertAt(octet, &leafNode[V]{prefix: pfx, value: val})
		}

		// ... or decend down the trie
		kid := n.children.MustGet(octet)

		// kid is node or leaf at addr
		switch kid := kid.(type) {
		case *node[V]:
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
			newNode := p.Get()
			newNode.insertAtDepth(p, kid.prefix, kid.value, depth+1)

			n.children.InsertAt(octet, newNode)
			n = newNode

		case *fringeNode[V]:
			// reached a path compressed fringe
			// override value in slot if pfx is a fringe
			if isFringe(depth, bits) {
				kid.value = val
				// exists
				return true
			}

			// create new node
			// push the fringe down, it becomes a default route (idx=1)
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := p.Get()
			newNode.prefixes.InsertAt(1, kid.value)

			n.children.InsertAt(octet, newNode)
			n = newNode

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
}

// insertAtDepthPersist is the immutable version of insertAtDepth.
// All visited nodes are cloned during insertion.

// insertAtDepthPersist performs an immutable insertion of a network prefix and its associated value
// into the trie starting at the specified byte depth.
//
// Unlike insertAtDepth, this function preserves the original trie by cloning each node visited along
// the insertion path. Modified subtrees are allocated via the provided pool, typically backed by sync.Pool,
// to avoid redundant allocations.
//
// Nodes are shallow-cloned with deep-copied values when necessary. If a node, leaf, or fringe exists
// at the insertion point, it's replaced with a newly allocated version, ensuring that the original structure
// remains unchanged.
// The function uses cloneFlat to replicate each traversed node before continuing the insert.
//
// This allows multiple versions of the trie to coexist safely and efficiently, enabling purely functional
// route updates with structural sharing where possible.
func (n *node[V]) insertAtDepthPersist(p *pool[V], pfx netip.Prefix, val V, depth int) (exists bool) {
	ip := pfx.Addr()
	bits := pfx.Bits()
	octets := ip.AsSlice()
	maxDepth, lastBits := maxDepthAndLastBits(bits)

	// find the proper trie node to insert prefix
	// start with prefix octet at depth
	for ; depth < len(octets); depth++ {
		octet := octets[depth]

		// last masked octet: insert/override prefix/val into node
		if depth == maxDepth {
			return n.prefixes.InsertAt(art.PfxToIdx(octet, lastBits), val)
		}

		if !n.children.Test(octet) {
			// insert prefix path compressed as leaf or fringe
			if isFringe(depth, bits) {
				return n.children.InsertAt(octet, &fringeNode[V]{val})
			}
			return n.children.InsertAt(octet, &leafNode[V]{prefix: pfx, value: val})
		}
		kid := n.children.MustGet(octet)

		// kid is node or leaf at addr
		switch kid := kid.(type) {
		case *node[V]:
			// clone the traversed path

			// kid points now to cloned kid
			kid = kid.cloneFlat(p)

			// replace kid with clone
			n.children.InsertAt(octet, kid)

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
			newNode := p.Get()
			newNode.insertAtDepth(p, kid.prefix, kid.value, depth+1)

			n.children.InsertAt(octet, newNode)
			n = newNode

		case *fringeNode[V]:
			// reached a path compressed fringe
			// override value in slot if pfx is a fringe
			if isFringe(depth, bits) {
				kid.value = val
				// exists
				return true
			}

			// create new node
			// push the fringe down, it becomes a default route (idx=1)
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := p.Get()
			newNode.prefixes.InsertAt(1, kid.value)

			n.children.InsertAt(octet, newNode)
			n = newNode

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
}

// purgeAndCompress traverses the deletion path upward and removes empty or compressible nodes
// in the trie.
//
// After a route deletion, this function walks back through the recorded traversal stack
// and optimizes the trie by eliminating redundant intermediate nodes. A node is purged if it is empty,
// and compressed if it contains only a single leaf, fringe, or prefix.
//
// Compressible cases are handled by removing the node and reinserting its content (prefix or value)
// one level higher, preserving routing semantics while reducing structural depth. The child is then
// replaced in the parent, effectively flattening the trie where appropriate.
//
// The reconstruction of prefixes for fringe or prefix entries is based on
// the original `octets` traversal path and the parent´s depth.
//
// Any removed intermediate nodes are returned to the memory pool, enabling efficient re-use through
// a sync.Pool-based allocator.
func (n *node[V]) purgeAndCompress(p *pool[V], stack []*node[V], octets []uint8, is4 bool) {
	// unwind the stack
	for depth := len(stack) - 1; depth >= 0; depth-- {
		parent := stack[depth]
		octet := octets[depth]

		pfxCount := n.prefixes.Len()
		childCount := n.children.Len()

		switch {
		case n.isEmpty():
			// just delete this empty node from parent
			parent.children.DeleteAt(octet)
			p.Put(n)

		case pfxCount == 0 && childCount == 1:
			switch kid := n.children.Items[0].(type) {
			case *node[V]:
				// fast exit, we are at an intermediate path node
				// no further delete/compress upwards the stack is possible
				return
			case *leafNode[V]:
				// just one leaf, delete this node and reinsert the leaf above
				parent.children.DeleteAt(octet)

				// ... (re)insert the leaf at parents depth
				parent.insertAtDepth(p, kid.prefix, kid.value, depth)
				p.Put(n)
			case *fringeNode[V]:
				// just one fringe, delete this node and reinsert the fringe as leaf above
				parent.children.DeleteAt(octet)

				// get the last octet back, the only item is also the first item
				lastOctet, _ := n.children.FirstSet()

				// rebuild the prefix with octets, depth, ip version and addr
				// depth is the parent's depth, so add +1 here for the kid
				fringePfx := cidrForFringe(octets, depth+1, is4, lastOctet)

				// ... (re)reinsert prefix/value at parents depth
				parent.insertAtDepth(p, fringePfx, kid.value, depth)
				p.Put(n)
			}

		case pfxCount == 1 && childCount == 0:
			// just one prefix, delete this node and reinsert the idx as leaf above
			parent.children.DeleteAt(octet)

			// get prefix back from idx ...
			idx, _ := n.prefixes.FirstSet() // single idx must be first bit set
			val := n.prefixes.Items[0]      // single value must be at Items[0]

			// ... and octet path
			path := stridePath{}
			copy(path[:], octets)

			// depth is the parent's depth, so add +1 here for the kid
			pfx := cidrFromPath(path, depth+1, is4, idx)

			// ... (re)insert prefix/value at parents depth
			parent.insertAtDepth(p, pfx, val, depth)
			p.Put(n)
		}

		// climb up the stack
		n = parent
	}
}

// lpmGet performs a longest-prefix match (LPM) lookup for the given index (idx)
// within the 8-bit stride-based prefix table at this trie depth.
//
// The function returns the matched base index, associated value, and true if a
// matching prefix exists at this level; otherwise, ok is false.
//
// Internally, the prefix table is organized as a complete binary tree (CBT) indexed
// via the baseIndex function. Unlike the original ART algorithm, this implementation
// does not use an allotment-based approach. Instead, it performs CBT backtracking
// using a bitset-based operation with a precomputed backtracking pattern specific to idx.
func (n *node[V]) lpmGet(idx uint) (baseIdx uint8, val V, ok bool) {
	// top is the idx of the longest-prefix-match
	if top, ok := n.prefixes.IntersectionTop(lpm.BackTrackingBitset(idx)); ok {
		return top, n.prefixes.MustGet(top), true
	}

	// not found (on this level)
	return
}

// lpmTest returns true if an index (idx) has any matching longest-prefix
// in the current node’s prefix table.
//
// This function performs a presence check without retrieving the associated value.
// It is faster than a full lookup, as it only tests for intersection with the
// backtracking bitset for the given index.
//
// The prefix table is structured as a complete binary tree (CBT), and LPM testing
// is done via a bitset operation that maps the traversal path from the given index
// toward its possible ancestors.
func (n *node[V]) lpmTest(idx uint) bool {
	return n.prefixes.Intersects(lpm.BackTrackingBitset(idx))
}

// cloneRec performs a recursive deep copy of the node[V].
//
// The method uses a pool[V] instance for efficient memory allocation of new nodes.
// It differentiates between shallow and deep copies:
//
// If the value type V implements the Cloner[V] interface, each item is deep-copied.
func (n *node[V]) cloneRec(p *pool[V]) *node[V] {
	if n == nil {
		return nil
	}

	c := p.Get()
	if n.isEmpty() {
		return c
	}

	// shallow
	c.prefixes = *(n.prefixes.Copy())

	_, isCloner := any(*new(V)).(Cloner[V])

	// deep copy if V implements Cloner[V]
	if isCloner {
		for i, val := range c.prefixes.Items {
			c.prefixes.Items[i] = cloneOrCopy(val)
		}
	}

	// shallow
	c.children = *(n.children.Copy())

	// deep copy of nodes and leaves
	for i, kidAny := range c.children.Items {
		switch kid := kidAny.(type) {
		case *node[V]:
			// clone the child node rec-descent
			c.children.Items[i] = kid.cloneRec(p)
		case *leafNode[V]:
			// deep copy if V implements Cloner[V]
			c.children.Items[i] = kid.cloneLeaf()
		case *fringeNode[V]:
			// deep copy if V implements Cloner[V]
			c.children.Items[i] = kid.cloneFringe()

		default:
			panic("logic error, wrong node type")
		}
	}

	return c
}

// cloneFlat creates a shallow copy of the current node[V], with optional deep copies of values.
//
// This method is intended for fast, non-recursive cloning of a node structure. It copies only
// the current node and selectively performs deep copies of stored values, without recursively
// cloning child nodes.
func (n *node[V]) cloneFlat(p *pool[V]) *node[V] {
	if n == nil {
		return nil
	}

	c := p.Get()
	if n.isEmpty() {
		return c
	}

	// shallow copy
	c.prefixes = *(n.prefixes.Copy())
	c.children = *(n.children.Copy())

	if _, ok := any(*new(V)).(Cloner[V]); !ok {
		// if V doesn't implement Cloner[V], return early
		return c
	}

	// deep copy of values in prefixes
	for i, val := range c.prefixes.Items {
		c.prefixes.Items[i] = cloneOrCopy(val)
	}

	// deep copy of values in path compressed leaves
	for i, kidAny := range c.children.Items {
		switch kid := kidAny.(type) {
		case *leafNode[V]:
			c.children.Items[i] = kid.cloneLeaf()
		case *fringeNode[V]:
			c.children.Items[i] = kid.cloneFringe()
		}
	}

	return c
}

// allRec recursively traverses the trie starting at the current node,
// applying the provided yield function to every stored prefix and value.
//
// For each route entry (prefix and value), yield is invoked. If yield returns false,
// the traversal stops immediately, and false is propagated upwards,
// enabling early termination.
//
// The function handles all prefix entries in the current node, as well as any children -
// including sub-nodes, leaf nodes with full prefixes, and fringe nodes
// representing path-compressed prefixes. IP prefix reconstruction is performed on-the-fly
// from the current path and depth.
//
// The traversal order is not defined. This implementation favors simplicity
// and runtime efficiency over consistency of iteration sequence.
func (n *node[V]) allRec(path stridePath, depth int, is4 bool, yield func(netip.Prefix, V) bool) bool {
	for _, idx := range n.prefixes.AsSlice(&[256]uint8{}) {
		cidr := cidrFromPath(path, depth, is4, idx)

		// callback for this prefix and val
		if !yield(cidr, n.prefixes.MustGet(idx)) {
			// early exit
			return false
		}
	}

	// for all children (nodes and leaves) in this node do ...
	for i, addr := range n.children.AsSlice(&[256]uint8{}) {
		switch kid := n.children.Items[i].(type) {
		case *node[V]:
			// rec-descent with this node
			path[depth] = addr
			if !kid.allRec(path, depth+1, is4, yield) {
				// early exit
				return false
			}
		case *leafNode[V]:
			// callback for this leaf
			if !yield(kid.prefix, kid.value) {
				// early exit
				return false
			}
		case *fringeNode[V]:
			fringePfx := cidrForFringe(path[:], depth, is4, addr)
			// callback for this fringe
			if !yield(fringePfx, kid.value) {
				// early exit
				return false
			}

		default:
			panic("logic error, wrong node type")
		}
	}

	return true
}

// allRecSorted recursively traverses the trie in prefix-sorted order and applies
// the given yield function to each stored prefix and value.
//
// Unlike allRec, this implementation ensures that route entries are visited in
// canonical prefix sort order. To achieve this,
// both the prefixes and children of the current node are gathered, sorted,
// and then interleaved during traversal based on logical octet positioning.
//
// The function first sorts relevant entries by their prefix index and address value,
// using a comparison function that ranks prefixes according to their mask length and position.
// Then it walks the trie, always yielding child entries that fall before the current prefix,
// followed by the prefix itself. Remaining children are processed once all prefixes have been visited.
//
// Prefixes are reconstructed on-the-fly from the traversal path, and iteration includes all child types:
// inner nodes (recursive descent), leaf nodes, and fringe (compressed) prefixes.
//
// If the yield callback returns false at any point, traversal stops early and false is returned,
// allowing for efficient filtered iteration. The order is stable and predictable, making the function
// suitable for use cases like table exports, comparisons, or serialization.
func (n *node[V]) allRecSorted(path stridePath, depth int, is4 bool, yield func(netip.Prefix, V) bool) bool {
	// get slice of all child octets, sorted by addr
	allChildAddrs := n.children.AsSlice(&[256]uint8{})

	// get slice of all indexes, sorted by idx
	allIndices := n.prefixes.AsSlice(&[256]uint8{})

	// sort indices in CIDR sort order
	slices.SortFunc(allIndices, cmpIndexRank)

	childCursor := 0

	// yield indices and childs in CIDR sort order
	for _, pfxIdx := range allIndices {
		pfxOctet, _ := art.IdxToPfx(pfxIdx)

		// yield all childs before idx
		for j := childCursor; j < len(allChildAddrs); j++ {
			childAddr := allChildAddrs[j]

			if childAddr >= pfxOctet {
				break
			}

			// yield the node (rec-descent) or leaf
			switch kid := n.children.Items[j].(type) {
			case *node[V]:
				path[depth] = childAddr
				if !kid.allRecSorted(path, depth+1, is4, yield) {
					return false
				}
			case *leafNode[V]:
				if !yield(kid.prefix, kid.value) {
					return false
				}
			case *fringeNode[V]:
				fringePfx := cidrForFringe(path[:], depth, is4, childAddr)
				// callback for this fringe
				if !yield(fringePfx, kid.value) {
					// early exit
					return false
				}

			default:
				panic("logic error, wrong node type")
			}

			childCursor++
		}

		// yield the prefix for this idx
		cidr := cidrFromPath(path, depth, is4, pfxIdx)
		// n.prefixes.Items[i] not possible after sorting allIndices
		if !yield(cidr, n.prefixes.MustGet(pfxIdx)) {
			return false
		}
	}

	// yield the rest of leaves and nodes (rec-descent)
	for j := childCursor; j < len(allChildAddrs); j++ {
		addr := allChildAddrs[j]
		switch kid := n.children.Items[j].(type) {
		case *node[V]:
			path[depth] = addr
			if !kid.allRecSorted(path, depth+1, is4, yield) {
				return false
			}
		case *leafNode[V]:
			if !yield(kid.prefix, kid.value) {
				return false
			}
		case *fringeNode[V]:
			fringePfx := cidrForFringe(path[:], depth, is4, addr)
			// callback for this fringe
			if !yield(fringePfx, kid.value) {
				// early exit
				return false
			}

		default:
			panic("logic error, wrong node type")
		}
	}

	return true
}

// unionRec recursively merges another node o into the receiver node n.
//
// All prefix and child entries from o are cloned and inserted into n.
// If a prefix already exists in n, its value is overwritten by the value from o,
// and the duplicate is counted in the return value. This count can later be used
// to update size-related metadata in the parent trie.
//
// The union handles all possible combinations of child node types (node, leaf, fringe)
// between the two nodes. Structural conflicts are resolved by creating new intermediate
// *node[V] objects and pushing both children further down the trie. Leaves and fringes
// are also recursively relocated as needed to preserve prefix semantics.
//
// All allocations use the provided pool p, ensuring efficient management of intermediate
// nodes during recursive merging. The merge operation is destructive on the receiver n,
// but leaves the source node o unchanged.
//
// Returns the number of duplicate prefixes that were overwritten during merging.
func (n *node[V]) unionRec(p *pool[V], o *node[V], depth int) (duplicates int) {
	// for all prefixes in other node do ...
	for i, oIdx := range o.prefixes.AsSlice(&[256]uint8{}) {
		// clone/copy the value from other node at idx
		clonedVal := cloneOrCopy(o.prefixes.Items[i])

		// insert/overwrite cloned value from o into n
		if n.prefixes.InsertAt(oIdx, clonedVal) {
			// this prefix is duplicate in n and o
			duplicates++
		}
	}

	// for all child addrs in other node do ...
	for i, addr := range o.children.AsSlice(&[256]uint8{}) {
		//  12 possible combinations to union this child and other child
		//
		//  THIS,   OTHER: (always clone the other kid!)
		//  --------------
		//  NULL,   node    <-- insert node at addr
		//  NULL,   leaf    <-- insert leaf at addr
		//  NULL,   fringe  <-- insert fringe at addr

		//  node,   node    <-- union rec-descent with node
		//  node,   leaf    <-- insert leaf at depth+1
		//  node,   fringe  <-- insert fringe at depth+1

		//  leaf,   node    <-- insert new node, push this leaf down, union rec-descent
		//  leaf,   leaf    <-- insert new node, push both leaves down (!first check equality)
		//  leaf,   fringe  <-- insert new node, push this leaf and fringe down

		//  fringe, node    <-- insert new node, push this fringe down, union rec-descent
		//  fringe, leaf    <-- insert new node, push this fringe down, insert other leaf at depth+1
		//  fringe, fringe  <-- just overwrite value
		//
		// try to get child at same addr from n
		thisChild, thisExists := n.children.Get(addr)
		if !thisExists { // NULL, ... slot at addr is empty
			switch otherKid := o.children.Items[i].(type) {
			case *node[V]: // NULL, node
				n.children.InsertAt(addr, otherKid.cloneRec(p))
				continue

			case *leafNode[V]: // NULL, leaf
				n.children.InsertAt(addr, otherKid.cloneLeaf())
				continue

			case *fringeNode[V]: // NULL, fringe
				n.children.InsertAt(addr, otherKid.cloneFringe())
				continue

			default:
				panic("logic error, wrong node type")
			}
		}

		switch thisKid := thisChild.(type) {
		case *node[V]: // node, ...
			switch otherKid := o.children.Items[i].(type) {
			case *node[V]: // node, node
				// both childs have node at addr, call union rec-descent on child nodes
				duplicates += thisKid.unionRec(p, otherKid.cloneRec(p), depth+1)
				continue

			case *leafNode[V]: // node, leaf
				// push this cloned leaf down, count duplicate entry
				clonedLeaf := otherKid.cloneLeaf()
				if thisKid.insertAtDepth(p, clonedLeaf.prefix, clonedLeaf.value, depth+1) {
					duplicates++
				}
				continue

			case *fringeNode[V]: // node, fringe
				// push this fringe down, a fringe becomes a default route one level down
				clonedFringe := otherKid.cloneFringe()
				if thisKid.prefixes.InsertAt(1, clonedFringe.value) {
					duplicates++
				}
				continue
			}

		case *leafNode[V]: // leaf, ...
			switch otherKid := o.children.Items[i].(type) {
			case *node[V]: // leaf, node
				// create new node
				nc := p.Get()

				// push this leaf down
				nc.insertAtDepth(p, thisKid.prefix, thisKid.value, depth+1)

				// insert the new node at current addr
				n.children.InsertAt(addr, nc)

				// unionRec this new node with other kid node
				duplicates += nc.unionRec(p, otherKid.cloneRec(p), depth+1)
				continue

			case *leafNode[V]: // leaf, leaf
				// shortcut, prefixes are equal
				if thisKid.prefix == otherKid.prefix {
					thisKid.value = cloneOrCopy(otherKid.value)
					duplicates++
					continue
				}

				// create new node
				nc := p.Get()

				// push this leaf down
				nc.insertAtDepth(p, thisKid.prefix, thisKid.value, depth+1)

				// insert at depth cloned leaf, maybe duplicate
				clonedLeaf := otherKid.cloneLeaf()
				if nc.insertAtDepth(p, clonedLeaf.prefix, clonedLeaf.value, depth+1) {
					duplicates++
				}

				// insert the new node at current addr
				n.children.InsertAt(addr, nc)
				continue

			case *fringeNode[V]: // leaf, fringe
				// create new node
				nc := p.Get()

				// push this leaf down
				nc.insertAtDepth(p, thisKid.prefix, thisKid.value, depth+1)

				// push this cloned fringe down, it becomes the default route
				clonedFringe := otherKid.cloneFringe()
				if nc.prefixes.InsertAt(1, clonedFringe.value) {
					duplicates++
				}

				// insert the new node at current addr
				n.children.InsertAt(addr, nc)
				continue
			}

		case *fringeNode[V]: // fringe, ...
			switch otherKid := o.children.Items[i].(type) {
			case *node[V]: // fringe, node
				// create new node
				nc := p.Get()

				// push this fringe down, it becomes the default route
				nc.prefixes.InsertAt(1, thisKid.value)

				// insert the new node at current addr
				n.children.InsertAt(addr, nc)

				// unionRec this new node with other kid node
				duplicates += nc.unionRec(p, otherKid.cloneRec(p), depth+1)
				continue

			case *leafNode[V]: // fringe, leaf
				// create new node
				nc := p.Get()

				// push this fringe down, it becomes the default route
				nc.prefixes.InsertAt(1, thisKid.value)

				// push this cloned leaf down
				clonedLeaf := otherKid.cloneLeaf()
				if nc.insertAtDepth(p, clonedLeaf.prefix, clonedLeaf.value, depth+1) {
					duplicates++
				}

				// insert the new node at current addr
				n.children.InsertAt(addr, nc)
				continue

			case *fringeNode[V]: // fringe, fringe
				thisKid.value = otherKid.cloneFringe().value
				duplicates++
				continue
			}

		default:
			panic("logic error, wrong node type")
		}
	}

	return duplicates
}

// eachLookupPrefix does an all prefix match in the 8-bit (stride) routing table
// at this depth and calls yield() for any matching CIDR.

// eachLookupPrefix performs a hierarchical lookup of all matching prefixes
// in the current node’s 8-bit stride-based prefix table.
//
// The function walks up the trie-internal complete binary tree (CBT),
// testing each possible prefix length mask (in decreasing order of specificity),
// and invokes the yield function for every matching entry.
//
// The given idx refers to the position for this stride's prefix and is used
// to derive a backtracking path through the CBT by repeatedly halving the index.
// At each step, if a prefix exists in the table, its corresponding CIDR is
// reconstructed and yielded. If yield returns false, traversal stops early.
//
// This function is intended for internal use during supernet traversal and
// does not descend the trie further.
func (n *node[V]) eachLookupPrefix(octets []byte, depth int, is4 bool, pfxIdx uint, yield func(netip.Prefix, V) bool) (ok bool) {
	// path needed below more than once in loop
	var path stridePath
	copy(path[:], octets)

	// fast forward, it's a /8 route, too big for bitset256
	if pfxIdx > 255 {
		pfxIdx >>= 1
	}
	idx := uint8(pfxIdx) // now it fits into uint8

	for ; idx > 0; idx >>= 1 {
		if n.prefixes.Test(idx) {
			val := n.prefixes.MustGet(idx)
			cidr := cidrFromPath(path, depth, is4, idx)

			if !yield(cidr, val) {
				return false
			}
		}
	}

	return true
}

// eachSubnet yields all prefix entries and child nodes covered by a given parent prefix,
// sorted in natural CIDR order, within the current node.
//
// The function iterates through all prefixes and children from the node’s stride tables.
// Only entries that fall within the address range defined by the parent prefix index (pfxIdx)
// are included. Matching entries are buffered, sorted, and passed through to the yield function.
//
// Child entries (nodes, leaves, fringes) that fall under the covered address range
// are processed recursively via allRecSorted to ensure sorted traversal.
//
// This function is intended for internal use by Subnets(), and it assumes the
// current node is positioned at the point in the trie corresponding to the parent prefix.
func (n *node[V]) eachSubnet(octets []byte, depth int, is4 bool, pfxIdx uint8, yield func(netip.Prefix, V) bool) bool {
	// octets as array, needed below more than once
	var path stridePath
	copy(path[:], octets)

	pfxFirstAddr, pfxLastAddr := art.IdxToRange(pfxIdx)

	allCoveredIndices := make([]uint8, 0, maxItems)
	for _, idx := range n.prefixes.AsSlice(&[256]uint8{}) {
		thisFirstAddr, thisLastAddr := art.IdxToRange(idx)

		if thisFirstAddr >= pfxFirstAddr && thisLastAddr <= pfxLastAddr {
			allCoveredIndices = append(allCoveredIndices, idx)
		}
	}

	// sort indices in CIDR sort order
	slices.SortFunc(allCoveredIndices, cmpIndexRank)

	// 2. collect all covered child addrs by prefix

	allCoveredChildAddrs := make([]uint8, 0, maxItems)
	for _, addr := range n.children.AsSlice(&[256]uint8{}) {
		if addr >= pfxFirstAddr && addr <= pfxLastAddr {
			allCoveredChildAddrs = append(allCoveredChildAddrs, addr)
		}
	}

	// 3. yield covered indices, pathcomp prefixes and childs in CIDR sort order

	addrCursor := 0

	// yield indices and childs in CIDR sort order
	for _, pfxIdx := range allCoveredIndices {
		pfxOctet, _ := art.IdxToPfx(pfxIdx)

		// yield all childs before idx
		for j := addrCursor; j < len(allCoveredChildAddrs); j++ {
			addr := allCoveredChildAddrs[j]
			if addr >= pfxOctet {
				break
			}

			// yield the node or leaf?
			switch kid := n.children.MustGet(addr).(type) {
			case *node[V]:
				path[depth] = addr
				if !kid.allRecSorted(path, depth+1, is4, yield) {
					return false
				}

			case *leafNode[V]:
				if !yield(kid.prefix, kid.value) {
					return false
				}

			case *fringeNode[V]:
				fringePfx := cidrForFringe(path[:], depth, is4, addr)
				// callback for this fringe
				if !yield(fringePfx, kid.value) {
					// early exit
					return false
				}

			default:
				panic("logic error, wrong node type")
			}

			addrCursor++
		}

		// yield the prefix for this idx
		cidr := cidrFromPath(path, depth, is4, pfxIdx)
		// n.prefixes.Items[i] not possible after sorting allIndices
		if !yield(cidr, n.prefixes.MustGet(pfxIdx)) {
			return false
		}
	}

	// yield the rest of leaves and nodes (rec-descent)
	for _, addr := range allCoveredChildAddrs[addrCursor:] {
		// yield the node or leaf?
		switch kid := n.children.MustGet(addr).(type) {
		case *node[V]:
			path[depth] = addr
			if !kid.allRecSorted(path, depth+1, is4, yield) {
				return false
			}
		case *leafNode[V]:
			if !yield(kid.prefix, kid.value) {
				return false
			}
		case *fringeNode[V]:
			fringePfx := cidrForFringe(path[:], depth, is4, addr)
			// callback for this fringe
			if !yield(fringePfx, kid.value) {
				// early exit
				return false
			}

		default:
			panic("logic error, wrong node type")
		}
	}

	return true
}

// cmpIndexRank, sort indexes in prefix sort order.
func cmpIndexRank(aIdx, bIdx uint8) int {
	// convert idx [1..255] to prefix
	aOctet, aBits := art.IdxToPfx(aIdx)
	bOctet, bBits := art.IdxToPfx(bIdx)

	// cmp the prefixes, first by address and then by bits
	if aOctet == bOctet {
		if aBits <= bBits {
			return -1
		}

		return 1
	}

	if aOctet < bOctet {
		return -1
	}

	return 1
}

// cidrFromPath, helper function,
// get prefix back from stride path, depth and idx.
// The prefix is solely defined by the position in the trie and the baseIndex.
func cidrFromPath(path stridePath, depth int, is4 bool, idx uint8) netip.Prefix {
	octet, pfxLen := art.IdxToPfx(idx)

	// set masked byte in path at depth
	path[depth] = octet

	// zero/mask the bytes after prefix bits
	clear(path[depth+1:])

	// make ip addr from octets
	var ip netip.Addr
	if is4 {
		ip = netip.AddrFrom4([4]byte(path[:4]))
	} else {
		ip = netip.AddrFrom16(path)
	}

	// calc bits with pathLen and pfxLen
	bits := depth<<3 + int(pfxLen)

	// return a normalized prefix from ip/bits
	return netip.PrefixFrom(ip, bits)
}

// cidrForFringe, helper function,
// get prefix back from octets path, depth, IP version and last octet.
// The prefix of a fringe is solely defined by the position in the trie.
func cidrForFringe(octets []byte, depth int, is4 bool, lastOctet uint8) netip.Prefix {
	path := stridePath{}
	copy(path[:], octets[:depth+1])

	// replace last octet
	path[depth] = lastOctet

	// make ip addr from octets
	var ip netip.Addr
	if is4 {
		ip = netip.AddrFrom4([4]byte(path[:4]))
	} else {
		ip = netip.AddrFrom16(path)
	}

	// it's a fringe, bits are alway /8, /16, /24, ...
	bits := (depth + 1) << 3

	// return a (normalized) prefix from ip/bits
	return netip.PrefixFrom(ip, bits)
}
