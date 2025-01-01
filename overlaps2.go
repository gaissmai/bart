package bart

import "github.com/gaissmai/bart/internal/bitset"

// overlapsPrefix returns true if node overlaps with prefix.
func (n *node2[V]) overlapsPrefix(octet byte, pfxLen int) bool {
	// 1. Test if any route in this node overlaps prefix?

	idx := pfxToIdx(octet, pfxLen)
	if n.lpmTest(idx) {
		return true
	}

	// 2. Test if prefix overlaps any route in this node

	// use bitset intersections instead of range loops
	// copy pre alloted bitset for idx
	a8 := idxToAllot(idx)
	allotedPrefixRoutes := bitset.BitSet(a8[:])

	if allotedPrefixRoutes.IntersectionCardinality(n.prefixes.BitSet) != 0 {
		return true
	}

	// 3. Test if prefix overlaps any child in this node

	// shift-right children bitset by 256 (firstHostIndex)
	c8 := make([]uint64, 8)
	copy(c8[4:], n.children.BitSet) // 4*64= 256
	hostRoutes := bitset.BitSet(c8)

	// use bitsets intersection instead of range loops
	return allotedPrefixRoutes.IntersectionCardinality(hostRoutes) != 0
}
