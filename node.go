package bart

import (
	"net/netip"

	"github.com/gaissmai/bart/internal/bitset"
)

const (
	strideLen    = 8   // byte, a multibit trie with stride len 8
	maxTreeDepth = 16  // max 16 bytes for IPv6
	maxItems     = 256 // max 256 prefixes or children in node
)

var (
	basePathIP4 = netip.MustParsePrefix("0.0.0.0/0")
	basePathIP6 = netip.MustParsePrefix("::/0")
)

// node TODO
type node[V any] struct {
	prefixes [256]*V
	children [256]*node[V]

	prefixesBitSet bitset.BitSet256 // for count and fast bitset operations
	childrenBitSet bitset.BitSet256 // for count and fast bitset operations

	basePath netip.Prefix // the base path always contains all routes under this prefix
}

// newNode TODO
func newNode[V any](pfx netip.Prefix) *node[V] {
	n := new(node[V])
	div8 := pfx.Bits() >> 3
	n.basePath = netip.PrefixFrom(pfx.Addr(), div8<<3)
	return n
}

// initOnceRootPath TODO, make it really Once()
func (n *node[V]) initOnceRootPath(is4 bool) {
	if n.basePath.IsValid() {
		return
	}

	if is4 {
		n.basePath = basePathIP4
	} else {
		n.basePath = basePathIP6
	}
}

// prefixCount TODO
func (n *node[V]) prefixCount() int {
	return n.prefixesBitSet.Size()
}

// childCount TODO
func (n *node[V]) childCount() int {
	return n.childrenBitSet.Size()
}

// containsPrefix TODO
func (n *node[V]) containsPrefix(pfx netip.Prefix) bool {
	return n.basePath.Overlaps(pfx) && n.basePath.Bits() <= pfx.Bits()
}

// isEmpty returns true if node has neither prefixes nor children
func (n *node[V]) isEmpty() bool {
	return n.prefixCount() == 0 && n.childCount() == 0
}

// getChild TODO
func (n *node[V]) getChild(addr uint8) *node[V] {
	return n.children[addr]
}

// insertChild TODO
func (n *node[V]) insertChild(addr uint8, child *node[V]) (exists bool) {
	if n.children[addr] == nil {
		exists = false
		n.childrenBitSet.Set(addr)
	} else {
		exists = true
	}

	n.children[addr] = child
	return exists
}

// deleteChild TODO
func (n *node[V]) deleteChild(addr uint8) {
	if n.children[addr] != nil {
		n.childrenBitSet.Clear(addr)
	}
	n.children[addr] = nil
}

// insertPrefix adds the route addr/prefixLen to n, with value val.
func (n *node[V]) insertPrefix(idx uint8, val V) (exists bool) {
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
func (n *node[V]) getPrefix(idx uint8) (val V, exists bool) {
	if exists = n.prefixesBitSet.Test(idx); exists {
		val = *n.prefixes[idx]
	}
	return
}

// deletePrefix TODO
// func (n *artNode[V]) deletePrefix(addr uint8, prefixLen uint8) (val V, exists bool) {
func (n *node[V]) deletePrefix(idx uint8) (val V, exists bool) {
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
func (n *node[V]) contains(idx uint) (ok bool) {
	return n.prefixes[uint8(idx>>1)] != nil
}

// lookup TODO
func (n *node[V]) lookup(idx uint) (val V, ok bool) {
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
func (n *node[V]) allot(idx uint8, oldValPtr, valPtr *V) {
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
