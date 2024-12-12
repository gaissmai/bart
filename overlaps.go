package bart

import "github.com/gaissmai/bart/internal/bitset"

// overlapsRec returns true if any IP in the nodes n or o overlaps.
func (n *node[V]) overlapsRec(o *node[V]) bool {
	nPfxCount := n.prefixes.Len()
	oPfxCount := o.prefixes.Len()

	nChildCount := n.children.Len()
	oChildCount := o.children.Len()

	// ##############################
	// 1. Test if any routes overlaps
	// ##############################

	// special case, overlapsPrefix is faster
	if nPfxCount == 1 && nChildCount == 0 {
		// get the single prefix from n
		idx, _ := n.prefixes.BitSet.NextSet(0)

		return o.overlapsPrefix(idxToPfx(idx))
	}

	// special case, overlapsPrefix is faster
	if oPfxCount == 1 && oChildCount == 0 {
		// get the single prefix from o
		idx, _ := o.prefixes.BitSet.NextSet(0)

		return n.overlapsPrefix(idxToPfx(idx))
	}

	// full cross check
	if nPfxCount > 0 && oPfxCount > 0 {
		if n.overlapsRoutes(o) {
			return true
		}
	}

	// ####################################
	// 2. Test if routes overlaps any child
	// ####################################

	if nPfxCount > 0 && oChildCount > 0 {
		if n.overlapsChildrenIn(o) {
			return true
		}
	}

	// symmetric reverse
	if oPfxCount > 0 && nChildCount > 0 {
		if o.overlapsChildrenIn(n) {
			return true
		}
	}

	// ################################################################
	// 3. rec-descent call for childs with same octet in nodes n and o
	// ################################################################

	// stop condition, n or o have no childs
	if nChildCount == 0 || oChildCount == 0 {
		return false
	}

	if oChildCount == 1 {
		return n.overlapsOneChildIn(o)
	}

	if nChildCount == 1 {
		return o.overlapsOneChildIn(n)
	}

	// stop condition, no child with identical octet in n and o
	if n.children.BitSet.IntersectionCardinality(o.children.BitSet) == 0 {
		return false
	}

	return n.overlapsSameChildrenRec(o)
}

// overlapsRoutes, test if n overlaps o prefixes and vice versa
func (n *node[V]) overlapsRoutes(o *node[V]) bool {
	// one node has just one prefix, use bitset algo
	if n.prefixes.Len() == 1 {
		return o.overlapsOneRouteIn(n)
	}

	// one node has just one prefix, use bitset algo
	if o.prefixes.Len() == 1 {
		return n.overlapsOneRouteIn(o)
	}

	// some prefixes are identical, trivial overlap
	if n.prefixes.BitSet.IntersectionCardinality(o.prefixes.BitSet) > 0 {
		return true
	}

	// make full cross check
	nOK := true
	oOK := true

	var nIdx, oIdx uint

	// zip, range over n and o together to help chance on its way
	for nOK || oOK {
		if nOK {
			// does any route in o overlap this prefix from n
			if nIdx, nOK = n.prefixes.BitSet.NextSet(nIdx); nOK {
				if o.lpmTest(nIdx) {
					return true
				}

				nIdx++
			}
		}

		if oOK {
			// does any route in n overlap this prefix from o
			if oIdx, oOK = o.prefixes.BitSet.NextSet(oIdx); oOK {
				if n.lpmTest(oIdx) {
					return true
				}

				oIdx++
			}
		}
	}

	return false
}

// overlapsChildrenIn, test if prefixes in n overlaps child octets in o.
func (n *node[V]) overlapsChildrenIn(o *node[V]) bool {
	pfxCount := n.prefixes.Len()
	childCount := o.children.Len()

	// heuristic, compare benchmarks
	// when will re range over the children and when will we do bitset calc?
	magicNumber := 15
	doRange := childCount < magicNumber || pfxCount > magicNumber

	// do range over, not so many childs and maybe to many prefixes
	if doRange {
		var oAddr uint

		ok := true
		for ok {
			// does any route in o overlap this child from n
			if oAddr, ok = o.children.BitSet.NextSet(oAddr); ok {
				if n.lpmTest(hostIndex(byte(oAddr))) {
					return true
				}

				oAddr++
			}
		}

		return false
	}

	// do bitset intersection, alloted route table with child octets
	// maybe to many childs ro range over or not so many prefixes to
	// build the alloted routing table from them

	// make allot table with prefixes as bitsets, bitsets are precalculated
	// just union the bitsets to one bitset (allot table) for all prefixes
	// in this node

	// gimmick, don't allocate, can't use bitset.New()
	prefixRoutes := bitset.BitSet(make([]uint64, 8))

	_, allIndices := n.prefixes.BitSet.NextSetMany(0, make([]uint, maxNodePrefixes))

	for _, idx := range allIndices {
		// get pre alloted bitset for idx
		a8 := allotLookupTbl[idx]
		prefixRoutes.InPlaceUnion(bitset.BitSet(a8[:]))
	}

	// shift-right children bitset by 256 (firstHostIndex)
	c8 := make([]uint64, 8)
	copy(c8[4:], o.children.BitSet) // 4*64= 256
	hostRoutes := bitset.BitSet(c8)

	return prefixRoutes.IntersectionCardinality(hostRoutes) > 0
}

// overlapsSameChildrenRec, find same octets with bitset intersection.
// rec-descent with same child octet in n an o,
func (n *node[V]) overlapsSameChildrenRec(o *node[V]) bool {
	// gimmicks, clone a bitset without heap allocation
	// 4*64=256, maxNodeChildren
	a4 := make([]uint64, 4)
	copy(a4, n.children.BitSet)
	nChildrenBitsetCloned := bitset.BitSet(a4)

	// intersect in place the child bitsets from n and o
	nChildrenBitsetCloned.InPlaceIntersection(o.children.BitSet)

	_, allCommonChildren := nChildrenBitsetCloned.NextSetMany(0, make([]uint, maxNodeChildren))

	// range over all child addrs, common in n and o
	for _, addr := range allCommonChildren {
		oChild, _ := o.children.Get(addr)
		nChild, _ := n.children.Get(addr)

		// rec-descent with same child
		if nChild.overlapsRec(oChild) {
			return true
		}
	}

	return false
}

func (n *node[V]) overlapsOneChildIn(o *node[V]) bool {
	// get the single addr and child
	addr, _ := o.children.BitSet.NextSet(0)
	oChild := o.children.Items[0]

	if nChild, ok := n.children.Get(addr); ok {
		return nChild.overlapsRec(oChild)
	}

	return false
}

func (n *node[V]) overlapsOneRouteIn(o *node[V]) bool {
	// get the single prefix from o
	idx, _ := o.prefixes.BitSet.NextSet(0)

	// 1. Test if any route in this node overlaps prefix?
	if n.lpmTest(idx) {
		return true
	}

	// 2. Test if prefix overlaps any route in this node
	// use bitset intersection with alloted stride table instead of range loops

	// copy pre alloted bitset for idx
	a8 := allotLookupTbl[idx]
	allotedPrefixRoutes := bitset.BitSet(a8[:])

	// use bitset intersection instead of range loops
	return allotedPrefixRoutes.IntersectionCardinality(n.prefixes.BitSet) > 0
}

// overlapsPrefix returns true if node overlaps with prefix.
func (n *node[V]) overlapsPrefix(octet byte, pfxLen int) bool {
	// 1. Test if any route in this node overlaps prefix?
	idx := pfxToIdx(octet, pfxLen)
	if n.lpmTest(idx) {
		return true
	}

	// 2. Test if prefix overlaps any route in this node
	// use bitset intersection with alloted stride table instead of range loops

	// copy pre alloted bitset for idx
	a8 := allotLookupTbl[idx]
	allotedPrefixRoutes := bitset.BitSet(a8[:])

	// use bitset intersection instead of range loops
	if allotedPrefixRoutes.IntersectionCardinality(n.prefixes.BitSet) != 0 {
		return true
	}

	// 3. Test if prefix overlaps any child in this node
	// use bitsets intersection instead of range loops

	// shift-right children bitset by 256 (firstHostIndex)
	c8 := make([]uint64, 8)
	copy(c8[4:], n.children.BitSet) // 4*64= 256
	hostRoutes := bitset.BitSet(c8)

	// use bitsets intersection instead of range loops
	return allotedPrefixRoutes.IntersectionCardinality(hostRoutes) != 0
}
