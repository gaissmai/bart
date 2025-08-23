package bart

import (
	"net/netip"

	"github.com/gaissmai/bart/internal/art"
)

// artNode is a trie level node in the multibit routing table.
//
// Each artNode contains two conceptually different fixed sized arrays:
//   - prefixes: representing routes, using a complete binary tree layout
//     driven by the baseIndex() function from the ART algorithm.
//   - children: holding subtries or path-compressed leaves or fringes.
//
// See doc/artlookup.pdf for the mapping mechanics and prefix tree details.
type artNode[V any] struct {
	prefixes [maxItems]*V
	children [maxItems]any // *artNode or path-compreassed *leaf- or *fringeNode

	prefixCount uint16
	childCount  uint16
}

// isEmpty returns true if node has neither prefixes nor children
func (n *artNode[V]) isEmpty() bool {
	return n.prefixCount == 0 && n.childCount == 0
}

func (n *artNode[V]) prefixesAsSlice() []uint8 {
	res := make([]uint8, 0, maxItems)
	for i := range n.prefixes {
		if n.idxIsRoot(uint8(i)) {
			res = append(res, uint8(i))
		}
	}
	return res
}

func (n *artNode[V]) mustFirstPrefixItem() (idx uint8, val V) {
	for idx, valPtr := range n.prefixes {
		if valPtr != nil {
			return uint8(idx), *valPtr
		}
	}
	panic("empty prefixes")
}

func (n *artNode[V]) childrenAsSlice() []uint8 {
	res := make([]uint8, 0, maxItems)
	for i, kid := range n.children {
		if kid != nil {
			res = append(res, uint8(i))
		}
	}
	return res
}

func (n *artNode[V]) mustFirstChildItem() (octet uint8, child any) {
	for i, child := range n.children {
		if child != nil {
			return uint8(i), child
		}
	}
	panic("empty children")
}

// getChild TODO
func (n *artNode[V]) getChild(addr uint8) any {
	return n.children[addr]
}

// insertChild TODO
func (n *artNode[V]) insertChild(addr uint8, child any) (exists bool) {
	if n.children[addr] == nil {
		exists = false
		n.childCount++
	} else {
		exists = true
	}

	n.children[addr] = child
	return exists
}

// deleteChild TODO
func (n *artNode[V]) deleteChild(addr uint8) {
	if n.children[addr] != nil {
		n.childCount--
	}
	n.children[addr] = nil
}

// insertPrefix adds the route addr/prefixLen to n, with value val.
func (n *artNode[V]) insertPrefix(addr uint8, prefixLen uint8, val V) (exists bool) {
	idx := art.PfxToIdx(addr, prefixLen)
	if n.idxIsRoot(idx) {
		exists = true
	} else {
		exists = false
		n.prefixCount++
	}

	// To ensure allot works as intended, every unique prefix in the
	// artNode must point to a distinct value pointer, even for identical values.
	// Using new() and assignment guarantees each inserted prefix gets its own address,
	// but only if V IS NOT zero-sized!
	valPtr := new(V)
	*valPtr = val

	oldValPtr := n.prefixes[idx]
	n.allot(idx, oldValPtr, valPtr)

	return
}

// getPrefix TODO
func (n *artNode[V]) getPrefix(addr uint8, prefixLen uint8) (val V, exists bool) {
	idx := art.PfxToIdx(addr, prefixLen)
	if n.idxIsRoot(idx) {
		valPtr := n.prefixes[idx]
		return *valPtr, true
	}
	// Route entry doesn't exist
	return val, false
}

// deletePrefix TODO
func (n *artNode[V]) deletePrefix(addr uint8, prefixLen uint8) (val V, exists bool) {
	idx := art.PfxToIdx(addr, prefixLen)
	if !n.idxIsRoot(idx) {
		// Route entry doesn't exist
		return val, false
	}

	valPtr := n.prefixes[idx]

	var parentValPtr *V
	if parentIdx := idx >> 1; parentIdx != 0 {
		parentValPtr = n.prefixes[parentIdx]
	}

	n.allot(idx, valPtr, parentValPtr)
	n.prefixCount--

	return *valPtr, true
}

// contains TODO
func (n *artNode[V]) contains(idx uint) (ok bool) {
	return n.prefixes[uint8(idx>>1)] != nil
}

// lookup TODO
func (n *artNode[V]) lookup(idx uint) (val V, ok bool) {
	if valPtr := n.prefixes[uint8(idx>>1)]; valPtr != nil {
		return *valPtr, true
	}
	return val, false
}

// allot updates entries whose stored valPtr matches oldValPtr, in the
// subtree rooted at idx. Matching entries have their stored oldValPtr set to
// valPtr, and their value set to val.
//
// allot is the core of the ART algorithm, enabling efficient insertion/deletion
// while preserving very fast lookups.
//
// See doc/artlookup.pdf for the mapping mechanics and prefix tree details.
//
// Example of (uninterrupted) allotment sequence:
//
//	addr/bits: 0/5 -> {0/5, 0/6, 4/6, 0/7, 2/7, 4/7, 6/7}
//	                    ╭────╮╭─────────┬────╮
//	       idx: 32 ->  32    64   65   128  129 130  131
//	                    ╰─────────╯╰─────────────┴────╯
//
// Using an iterative form ensures better inlining opportunities.
func (n *artNode[V]) allot(idx uint8, oldValPtr, valPtr *V) {
	// iteration with stack instead of recursion
	stack := make([]uint8, 0, 256)

	// start idx
	stack = append(stack, idx)

	for i := 0; i < len(stack); i++ {
		idx = stack[i]

		// stop this allot path, idx already points to a more specific route.
		if n.prefixes[idx] != oldValPtr {
			continue
		}
		n.prefixes[idx] = valPtr

		// max idx is 255, so stop the duplication at 128 and above
		if idx >= 128 {
			continue
		}

		// duplicate and duplicate+1
		left := idx << 1
		right := left + 1

		stack = append(stack, left, right)
	}
}

// idxIsRoot TODO
func (n *artNode[V]) idxIsRoot(idx uint8) bool {
	valPtr := n.prefixes[idx]
	if valPtr == nil {
		return false
	}

	parentIdx := idx >> 1
	if parentIdx == 0 {
		// [idx] is non-nil, and is at the 0/0 route position (idx == 1).
		return true
	}
	if parentValPtr := n.prefixes[parentIdx]; valPtr != parentValPtr {
		// parent node in the tree isn't the same prefix
		return true
	}
	return false
}

func (n *artNode[V]) insertAtDepth(pfx netip.Prefix, val V, depth int) (exists bool) {
	ip := pfx.Addr() // the pfx must be in canonical form
	bits := pfx.Bits()
	octets := ip.AsSlice()
	maxDepth, lastBits := maxDepthAndLastBits(bits)

	// find the proper trie node to insert prefix
	// start with prefix octet at depth
	for _, octet := range octets[depth:] {
		// last masked octet: insert/override prefix/val into node
		if depth == maxDepth {
			return n.insertPrefix(octet, lastBits, val)
		}

		kidAny := n.getChild(octet)
		// reached end of trie path ...
		if kidAny == nil {
			// insert prefix path compressed as leaf or fringe
			if isFringe(depth, bits) {
				return n.insertChild(octet, newFringeNode(val))
			}
			return n.insertChild(octet, newLeafNode(pfx, val))
		}

		// kid is node or leaf at addr
		switch kid := kidAny.(type) {
		case *artNode[V]:
			n = kid // descend down to next trie level

		case *leafNode[V]:
			// reached a path compressed prefix
			// override value in slot if prefixes are equal
			if kid.prefix == pfx {
				kid.value = val
				// update
				return true
			}

			// create new node
			// push the leaf down
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(artNode[V])
			_ = newNode.insertAtDepth(kid.prefix, kid.value, depth+1)
			_ = n.insertChild(octet, newNode)
			n = newNode

		case *fringeNode[V]:
			// reached a path compressed fringe
			// override value in slot if pfx is a fringe
			if isFringe(depth, bits) {
				kid.value = val
				// exists
				return true
			}

			// create new node ART node
			// push the fringe down, it becomes a default route (idx=1)
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(artNode[V])
			_ = newNode.insertPrefix(0, 0, kid.value)
			_ = n.insertChild(octet, newNode)
			n = newNode

		default:
			panic("logic error, wrong node type")
		}

		depth++
	}

	panic("unreachable")
}

func (n *artNode[V]) purgeAndCompress(stack []*artNode[V], octets []uint8, is4 bool) {
	// unwind the stack
	for depth := len(stack) - 1; depth >= 0; depth-- {
		parent := stack[depth]
		octet := octets[depth]

		pfxCount := n.prefixCount
		childCount := n.childCount

		switch {
		case n.isEmpty():
			// just delete this empty node from parent
			parent.deleteChild(octet)

		case pfxCount == 0 && childCount == 1:
			_, kidAny := n.mustFirstChildItem() // single child must be first child

			switch kid := kidAny.(type) {
			case *artNode[V]:
				// fast exit, we are at an intermediate path node
				// no further delete/compress upwards the stack is possible
				return
			case *leafNode[V]:
				// just one leaf, delete this node and reinsert the leaf above
				parent.deleteChild(octet)

				// ... (re)insert the leaf at parents depth
				parent.insertAtDepth(kid.prefix, kid.value, depth)
			case *fringeNode[V]:
				// just one fringe, delete this node and reinsert the fringe as leaf above
				parent.deleteChild(octet)

				// get the last fringe octet back, the only item is also the first item
				lastFringeOctet, _ := n.mustFirstChildItem()

				// rebuild the prefix with octets, depth, ip version and addr
				// depth is the parent's depth, so add +1 here for the kid
				fringePfx := cidrForFringe(octets, depth+1, is4, lastFringeOctet)

				// ... (re)reinsert prefix/value at parents depth
				parent.insertAtDepth(fringePfx, kid.value, depth)
			}

		case pfxCount == 1 && childCount == 0:
			// just one prefix, delete this node and reinsert the idx as leaf above
			parent.deleteChild(octet)

			// get prefix/val back from idx ...
			idx, val := n.mustFirstPrefixItem() // single idx must be first prefix

			// ... and octet path
			path := stridePath{}
			copy(path[:], octets)

			// depth is the parent's depth, so add +1 here for the kid
			pfx := cidrFromPath(path, depth+1, is4, idx)

			// ... (re)insert prefix/value at parents depth
			parent.insertAtDepth(pfx, val, depth)
		}

		// climb up the stack
		n = parent
	}
}
