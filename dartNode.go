package bart

const (
	firstHostIdx = 256
	lastHostIdx  = 511
	numChildren  = 256
)

type artNode[V any] struct {
	children [numChildren]any    // 256
	prefixes [lastHostIdx + 1]*V // 512

	childCount  int16
	prefixCount int16
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

// deleteChild TODO
func (n *artNode[V]) deleteChild(addr uint8) {
	if n.children[addr] != nil {
		n.childCount--
	}
	n.children[addr] = nil
}

// setChild TODO
func (n *artNode[V]) setChild(addr uint8, child any) {
	if n.children[addr] == nil {
		n.childCount++
	}
	n.children[addr] = child
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

// deletePrefix TODO
func (n0 *artNode[V]) deletePrefix(addr uint8, prefixLen int) (exists bool) {
	idx := prefix2Index(addr, prefixLen)
	if !n0.isStartIdx(idx) {
		// Route entry doesn't exist
		return false
	}

	val := n0.prefixes[idx]
	var parentVal *V
	if parentIdx := parentIndex(idx); parentIdx != 0 {
		parentVal = n0.prefixes[parentIdx]
	}

	n0.allotRec(idx, val, parentVal)
	n0.prefixCount--

	return true
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
