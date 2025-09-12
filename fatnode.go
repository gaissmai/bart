package bart

import (
	"net/netip"

	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/bitset"
)

// fatNode is a trie level node in the multibit routing table.
//
// Each fatNode contains two conceptually different fixed sized arrays:
//   - prefixes: representing routes, using a complete binary tree layout
//     driven by the baseIndex() function from the ART algorithm.
//   - children: holding subtries or path-compressed leaves or fringes.
//
// See doc/artlookup.pdf for the mapping mechanics and prefix tree details.
type fatNode[V any] struct {
	prefixes [256]*V
	children [256]*any // **fatNode or path-compressed **leaf- or **fringeNode
	// an array of "pointers to" the empty interface,
	// and not an array of empty interfaces.
	//
	// - any  ( interface{}) takes 2 words, even if nil.
	// - *any (*interface{}) requires only 1 word when nil.
	//
	// Since many slots are nil, this reduces memory by 30%.
	// The added indirection does not have a measurable performance impact,
	// but makes the code uglier.

	prefixesBitSet bitset.BitSet256 // for count and fast bitset operations
	childrenBitSet bitset.BitSet256 // for count and fast bitset operations
}

// prefixCount returns the number of prefixes stored in this node.
func (n *fatNode[V]) prefixCount() int {
	return n.prefixesBitSet.Size()
}

// childCount returns the number of slots used in this node.
func (n *fatNode[V]) childCount() int {
	return n.childrenBitSet.Size()
}

// isEmpty returns true if node has neither prefixes nor children
func (n *fatNode[V]) isEmpty() bool {
	return (n.prefixesBitSet.Size() + n.childrenBitSet.Size()) == 0
}

// getChild TODO
func (n *fatNode[V]) getChild(addr uint8) (any, bool) {
	if anyPtr := n.children[addr]; anyPtr != nil {
		return *anyPtr, true
	}
	return nil, false
}

// insertChild TODO
func (n *fatNode[V]) insertChild(addr uint8, child any) (exists bool) {
	if n.children[addr] == nil {
		exists = false
		n.childrenBitSet.Set(addr)
	} else {
		exists = true
	}

	c := child // force clear ownership; address escapes to heap
	n.children[addr] = &c

	return exists
}

// deleteChild TODO
func (n *fatNode[V]) deleteChild(addr uint8) {
	if n.children[addr] != nil {
		n.childrenBitSet.Clear(addr)
	}
	n.children[addr] = nil
}

// insertPrefix adds the route addr/prefixLen to n, with value val.
func (n *fatNode[V]) insertPrefix(idx uint8, val V) (exists bool) {
	if exists = n.prefixesBitSet.Test(idx); !exists {
		n.prefixesBitSet.Set(idx)
	}

	// insert or overwrite

	// To ensure allot works as intended, every unique prefix in the
	// fatNode must point to a distinct value pointer, even for identical values.
	// Using new() and assignment guarantees each inserted prefix gets its own address,
	valPtr := new(V)
	*valPtr = val

	oldValPtr := n.prefixes[idx]

	// overwrite oldValPtr with valPtr
	n.allot(idx, oldValPtr, valPtr)

	return
}

// getPrefix TODO
func (n *fatNode[V]) getPrefix(idx uint8) (val V, exists bool) {
	if exists = n.prefixesBitSet.Test(idx); exists {
		val = *n.prefixes[idx]
	}
	return
}

// deletePrefix TODO
// func (n *fatNode[V]) deletePrefix(addr uint8, prefixLen uint8) (val V, exists bool) {
func (n *fatNode[V]) deletePrefix(idx uint8) (val V, exists bool) {
	if exists = n.prefixesBitSet.Test(idx); !exists {
		// Route entry doesn't exist
		return
	}

	valPtr := n.prefixes[idx]
	parentValPtr := n.prefixes[idx>>1]

	// delete -> overwrite valPtr with parentValPtr
	n.allot(idx, valPtr, parentValPtr)

	n.prefixesBitSet.Clear(idx)

	return *valPtr, true
}

// contains TODO
func (n *fatNode[V]) contains(idx uint) (ok bool) {
	//nolint:gosec  // G115: integer overflow conversion int -> uint
	return n.prefixes[uint8(idx>>1)] != nil
}

// lookup TODO
func (n *fatNode[V]) lookup(idx uint) (val V, ok bool) {
	//nolint:gosec  // G115: integer overflow conversion int -> uint
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
func (n *fatNode[V]) allot(idx uint8, oldValPtr, valPtr *V) {
	// iteration with stack instead of recursion
	stack := make([]uint8, 0, 256)

	// start idx
	stack = append(stack, idx)

	for i := 0; i < len(stack); i++ {
		idx = stack[i]

		// stop this allot path, idx already points to a more specific route.
		if n.prefixes[idx] != oldValPtr {
			continue // take next path from stack
		}

		// overwrite
		n.prefixes[idx] = valPtr

		// max idx is 255, so stop the duplication at 128 and above
		if idx >= 128 {
			continue
		}

		// child nodes, it's a complete binary tree
		// left:  idx*2
		// right: idx*2+1
		stack = append(stack, idx<<1, idx<<1+1)
	}
}

func (n *fatNode[V]) insertAtDepth(pfx netip.Prefix, val V, depth int) (exists bool) {
	ip := pfx.Addr() // the pfx must be in canonical form
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)

	// find the proper trie node to insert prefix
	// start with prefix octet at depth
	for _, octet := range octets[depth:] {
		// last masked octet: insert/override prefix/val into node
		if depth == lastOctetPlusOne {
			return n.insertPrefix(art.PfxToIdx(octet, lastBits), val)
		}

		kidAny, ok := n.getChild(octet)
		// reached end of trie path ...
		if !ok {
			// insert prefix path compressed as leaf or fringe
			if isFringe(depth, pfx) {
				return n.insertChild(octet, newFringeNode(val))
			}
			return n.insertChild(octet, newLeafNode(pfx, val))
		}

		// kid is node or leaf at addr
		switch kid := kidAny.(type) {
		case *fatNode[V]:
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
			newNode := new(fatNode[V])
			_ = newNode.insertAtDepth(kid.prefix, kid.value, depth+1)
			_ = n.insertChild(octet, newNode)
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
			newNode := new(fatNode[V])
			_ = newNode.insertPrefix(1, kid.value)
			_ = n.insertChild(octet, newNode)
			n = newNode

		default:
			panic("logic error, wrong node type")
		}

		depth++
	}

	panic("unreachable")
}

func (n *fatNode[V]) purgeAndCompress(stack []*fatNode[V], octets []uint8, is4 bool) {
	// unwind the stack
	for depth := len(stack) - 1; depth >= 0; depth-- {
		parent := stack[depth]
		octet := octets[depth]

		pfxCount := n.prefixCount()
		childCount := n.childCount()

		switch {
		case n.prefixCount() == 0 && n.childCount() == 0:
			// just delete this empty node from parent
			parent.deleteChild(octet)

		case pfxCount == 0 && childCount == 1:
			addr, _ := n.childrenBitSet.FirstSet() // single child must be first child
			kidAny := *n.children[addr]

			switch kid := kidAny.(type) {
			case *fatNode[V]:
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

				// rebuild the prefix with octets, depth, ip version and addr
				// depth is the parent's depth, so add +1 here for the kid
				fringePfx := cidrForFringe(octets, depth+1, is4, addr)

				// ... (re)reinsert prefix/value at parents depth
				parent.insertAtDepth(fringePfx, kid.value, depth)
			}

		case pfxCount == 1 && childCount == 0:
			// just one prefix, delete this node and reinsert the idx as leaf above
			parent.deleteChild(octet)

			// get prefix/val back from idx ...
			idx, _ := n.prefixesBitSet.FirstSet() // single idx must be first bit set
			val := *n.prefixes[idx]

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
