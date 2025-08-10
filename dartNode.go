package bart

const (
	firstHostIdx = 256
	lastHostIdx  = 511
	numChildren  = 256
)

// artNode is a trie level node in the multibit routing table.
//
// Each artNode contains two conceptually different fixed sized arrays:
//   - prefixes: representing routes, using a complete binary tree layout
//     driven by the baseIndex() function from the ART algorithm.
//   - children: holding subtries a branching factor of 256.
//
// See doc/artlookup.pdf for the mapping mechanics and prefix tree details.
type artNode[V any] struct {
	prefixes [lastHostIdx + 1]*V // 512
	children [numChildren]any    // 256

	prefixCount int16
	childCount  int16
}

// isEmpty returns true if node has neither prefixes nor children
func (n *artNode[V]) isEmpty() bool {
	return n.prefixCount == 0 && n.childCount == 0
}

// TODO
func prefix2Index(addr uint8, prefixLen int) int {
	return (int(addr) >> (8 - prefixLen)) + (1 << prefixLen)
}

// TODO
func hostIndex(addr uint8) int {
	return int(addr) + firstHostIdx
}

// TODO
func parentIndex(idx int) int {
	return idx >> 1
}

// getChild TODO
func (n *artNode[V]) getChild(addr uint8) any {
	return n.children[addr]
}

func (n *artNode[V]) getOrCreateChild(addr uint8) any {
	c := n.children[addr]
	if c == nil {
		c = &artNode[V]{}
		n.children[addr] = c
		n.childCount++
	}
	return c
}

// setChild TODO
func (n *artNode[V]) setChild(addr uint8, child any) {
	if n.children[addr] == nil {
		n.childCount++
	}
	n.children[addr] = child
}

// deleteChild TODO
func (n *artNode[V]) deleteChild(addr uint8) {
	if n.children[addr] != nil {
		n.childCount--
	}
	n.children[addr] = nil
}

// insertPrefix adds the route addr/prefixLen to n, with value val.
func (n *artNode[V]) insertPrefix(addr uint8, prefixLen int, val V) (exists bool) {
	exists = true

	idx := prefix2Index(addr, prefixLen)
	if !n.isStartIdx(idx) {
		// new prefix
		exists = false
		n.prefixCount++
	}

	// To ensure allot works as intended, every unique prefix in the
	// artNode must point to a distinct value pointer, even for identical values.
	// Using new() and assignment guarantees each inserted prefix gets its own address.
	p := new(V)
	*p = val

	old := n.prefixes[idx]
	n.allotRec(idx, old, p)

	return
}

// getPrefix TODO
func (n *artNode[V]) getPrefix(addr uint8, prefixLen int) (val V, exists bool) {
	idx := prefix2Index(addr, prefixLen)
	if n.isStartIdx(idx) {
		pv := n.prefixes[idx]
		return *pv, true
	}
	// Route entry doesn't exist
	return val, false
}

// deletePrefix TODO
func (n *artNode[V]) deletePrefix(addr uint8, prefixLen int) (val V, exists bool) {
	idx := prefix2Index(addr, prefixLen)
	if !n.isStartIdx(idx) {
		// Route entry doesn't exist
		return val, false
	}

	pv := n.prefixes[idx]
	var parentVal *V
	if parentIdx := parentIndex(idx); parentIdx != 0 {
		parentVal = n.prefixes[parentIdx]
	}

	n.allotRec(idx, pv, parentVal)
	n.prefixCount--

	return *pv, true
}

// contains TODO
func (n *artNode[V]) contains(addr uint8) (ok bool) {
	return n.prefixes[hostIndex(addr)] != nil
}

// lookup TODO
func (n *artNode[V]) lookup(addr uint8) (ret V, ok bool) {
	if val := n.prefixes[hostIndex(addr)]; val != nil {
		return *val, true
	}
	return ret, false
}

// allotRec updates entries in the subtree rooted at idx whose stored prefix pointer equals old.
// For each matching entry, it updates the prefix pointer to val.
//
// This function is central to the ART algorithm, efficiently supporting fast lookups.
func (n *artNode[V]) allotRec(idx int, old, val *V) {
	if n.prefixes[idx] != old {
		// This index doesn't match what we're looking for-likely a recursive call
		// has reached a child node with a more specific route already in place.
		// Don't modify this branch.
		return
	}
	n.prefixes[idx] = val
	if idx >= firstHostIdx {
		// updated a host route, we've reached a leaf node in the binary tree.
		return
	}

	// Continue updating in both child subtrees.
	leftChildIdx := idx << 1
	n.allotRec(leftChildIdx, old, val)

	rightChildIdx := leftChildIdx + 1
	n.allotRec(rightChildIdx, old, val)
}

// isStartIdx TODO
func (n *artNode[V]) isStartIdx(idx int) bool {
	val := n.prefixes[idx]
	if val == nil {
		return false
	}

	parentIdx := parentIndex(idx)
	if parentIdx == 0 {
		// idx is non-nil, and is at the 0/0 route position.
		return true
	}
	if parentVal := n.prefixes[parentIdx]; val != parentVal {
		// parent node in the tree isn't the same prefix, so idx must
		// be a startIdx
		return true
	}
	return false
}

// nodeStatsRec, calculate the number of pfxs, nodes and leaves under n, rec-descent.
func (n *artNode[V]) nodeStatsRec() stats {
	var s stats
	if n == nil || n.isEmpty() {
		return s
	}

	s.pfxs = int(n.prefixCount)
	s.childs = int(n.childCount)
	s.nodes = 1 // this node
	s.leaves = 0
	s.fringes = 0

	for _, kidAny := range n.children {
		if kidAny == nil {
			continue
		}

		switch kid := kidAny.(type) {
		case *artNode[V]:
			// rec-descent
			rs := kid.nodeStatsRec()

			s.pfxs += rs.pfxs
			s.childs += rs.childs
			s.nodes += rs.nodes
			s.leaves += rs.leaves
			s.fringes += rs.fringes

		case *bartNode[V]:
			// rec-descent
			rs := kid.nodeStatsRec()

			s.pfxs += rs.pfxs
			s.childs += rs.childs
			s.nodes += rs.nodes
			s.leaves += rs.leaves
			s.fringes += rs.fringes

		default:
			panic("logic error, wrong node type")
		}
	}

	return s
}
