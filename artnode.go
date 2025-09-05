package bart

import (
	"net/netip"

	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/bitset"
)

const (
	strideLen    = 8   // byte, a multibit trie with stride len 8
	maxTreeDepth = 16  // max 16 bytes for IPv6
	maxItems     = 256 // max 256 prefixes or children in node
)

// stridePath, max 16 octets deep
type stridePath [maxTreeDepth]uint8

// artNode is a trie level node in the multibit routing table.
//
// Each artNode contains two conceptually different fixed sized arrays:
//   - prefixes: representing routes, using a complete binary tree layout
//     driven by the baseIndex() function from the ART algorithm.
//   - children: holding subtries or path-compressed leaves or fringes.
//
// See doc/artlookup.pdf for the mapping mechanics and prefix tree details.
type artNode[V any] struct {
	prefixes [256]*V
	children [256]*any // **artNode or path-compreassed **leaf- or **fringeNode
	// is an array of pointers to the empty interface,
	// and not an array of empty interfaces.
	//
	// - any  ( interface{}) takes 2 words, even if nil.
	// - *any (*interface{}) requires only 1 word when nil.
	//
	// Since many slots are nil, this reduces memory by 30%.
	// The added indirection does not have a measurable performance impact,
	// just makes the code uglier.

	prefixesBitSet bitset.BitSet256 // for count and fast bitset operations
	childrenBitSet bitset.BitSet256 // for count and fast bitset operations
}

// leafNode is a prefix with value, used as a path compressed child.
type leafNode[V any] struct {
	prefix netip.Prefix
	value  V
}

func newLeafNode[V any](pfx netip.Prefix, val V) *leafNode[V] {
	return &leafNode[V]{prefix: pfx, value: val}
}

// fringeNode is a path-compressed leaf with value but without a prefix.
// The prefix of a fringe is solely defined by the position in the trie.
// The fringe-compressiion (no stored prefix) saves a lot of memory,
// but the algorithm is more complex.
type fringeNode[V any] struct {
	value V
}

func newFringeNode[V any](val V) *fringeNode[V] {
	return &fringeNode[V]{value: val}
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

// TODO
func (n *artNode[V]) prefixCount() int {
	return n.prefixesBitSet.Size()
}

// TODO
func (n *artNode[V]) childCount() int {
	return n.childrenBitSet.Size()
}

// isEmpty returns true if node has neither prefixes nor children
func (n *artNode[V]) isEmpty() bool {
	return n.prefixCount() == 0 && n.childCount() == 0
}

// getChild TODO
func (n *artNode[V]) getChild(addr uint8) *any {
	return n.children[addr]
}

// insertChild TODO
func (n *artNode[V]) insertChild(addr uint8, child any) (exists bool) {
	if n.children[addr] == nil {
		exists = false
		n.childrenBitSet.Set(addr)
	} else {
		exists = true
	}

	n.children[addr] = &child
	return exists
}

// deleteChild TODO
func (n *artNode[V]) deleteChild(addr uint8) {
	if n.children[addr] != nil {
		n.childrenBitSet.Clear(addr)
	}
	n.children[addr] = nil
}

// insertPrefix adds the route addr/prefixLen to n, with value val.
func (n *artNode[V]) insertPrefix(idx uint8, val V) (exists bool) {
	if exists = n.prefixesBitSet.Test(idx); !exists {
		n.prefixesBitSet.Set(idx)
	}

	// insert or overwrite

	// To ensure allot works as intended, every unique prefix in the
	// artNode must point to a distinct value pointer, even for identical values.
	// Using new() and assignment guarantees each inserted prefix gets its own address,
	valPtr := new(V)
	*valPtr = val

	oldValPtr := n.prefixes[idx]

	// overwrite oldValPtr with valPtr
	n.allot(idx, oldValPtr, valPtr)

	return
}

// getPrefix TODO
func (n *artNode[V]) getPrefix(idx uint8) (val V, exists bool) {
	if exists = n.prefixesBitSet.Test(idx); exists {
		val = *n.prefixes[idx]
	}
	return
}

// deletePrefix TODO
// func (n *artNode[V]) deletePrefix(addr uint8, prefixLen uint8) (val V, exists bool) {
func (n *artNode[V]) deletePrefix(idx uint8) (val V, exists bool) {
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
			return n.insertPrefix(art.PfxToIdx(octet, lastBits), val)
		}

		anyPtr := n.getChild(octet)
		// reached end of trie path ...
		if anyPtr == nil {
			// insert prefix path compressed as leaf or fringe
			if isFringe(depth, bits) {
				return n.insertChild(octet, newFringeNode(val))
			}
			return n.insertChild(octet, newLeafNode(pfx, val))
		}

		// kid is node or leaf at addr
		kidAny := *anyPtr
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

func (n *artNode[V]) purgeAndCompress(stack []*artNode[V], octets []uint8, is4 bool) {
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
	depth = depth & 0xf // BCE

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
	depth = depth & 0xf // BCE

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
