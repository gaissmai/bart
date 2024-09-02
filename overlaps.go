package bart

import "github.com/bits-and-blooms/bitset"

// overlapsRec returns true if any IP in the nodes n or o overlaps.
func (n *node[V]) overlapsRec(o *node[V]) bool {
	nPfxLen := len(n.prefixes)
	oPfxLen := len(o.prefixes)

	nChildLen := len(n.children)
	oChildLen := len(o.children)

	// ##############################
	// 1. Test if any routes overlaps
	// ##############################

	// special case, overlapsPrefix is faster
	if nPfxLen == 1 && nChildLen == 0 {
		// get the single prefix from n
		idx, _ := n.prefixesBitset.NextSet(0)

		return o.overlapsPrefix(idxToPfx(idx))
	}

	// special case, overlapsPrefix is faster
	if oPfxLen == 1 && oChildLen == 0 {
		// get the single prefix from o
		idx, _ := o.prefixesBitset.NextSet(0)

		return n.overlapsPrefix(idxToPfx(idx))
	}

	// full cross check
	if nPfxLen > 0 && oPfxLen > 0 {
		if n.overlapsRoutes(o) {
			return true
		}
	}

	// ####################################
	// 2. Test if routes overlaps any child
	// ####################################

	if nPfxLen > 0 && oChildLen > 0 {
		if n.overlapsChildsIn(o) {
			return true
		}
	}

	// symmetric reverse
	if oPfxLen > 0 && nChildLen > 0 {
		if o.overlapsChildsIn(n) {
			return true
		}
	}

	// ################################################################
	// 3. rec-descent call for childs with same octet in nodes n and o
	// ################################################################

	// stop condition, n or o have no childs
	if nChildLen == 0 || oChildLen == 0 {
		return false
	}

	if oChildLen == 1 {
		return n.overlapsOneChildIn(o)
	}

	if nChildLen == 1 {
		return o.overlapsOneChildIn(n)
	}

	// stop condition, no child with identical octet in n and o
	if n.childrenBitset.IntersectionCardinality(o.childrenBitset) == 0 {
		return false
	}

	return n.overlapsSameChildsRec(o)
}

// overlapsRoutes, test if n overlaps o prefixes and vice versa
func (n *node[V]) overlapsRoutes(o *node[V]) bool {
	// one node has just one prefix, use bitset algo
	if len(n.prefixes) == 1 {
		return o.overlapsOneRouteIn(n)
	}

	// one node has just one prefix, use bitset algo
	if len(o.prefixes) == 1 {
		return n.overlapsOneRouteIn(o)
	}

	// some prefixes are identical, trivial overlap
	if n.prefixesBitset.IntersectionCardinality(o.prefixesBitset) > 0 {
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
			if nIdx, nOK = n.prefixesBitset.NextSet(nIdx); nOK {
				if o.lpmTest(nIdx) {
					return true
				}

				nIdx++
			}
		}

		if oOK {
			// does any route in n overlap this prefix from o
			if oIdx, oOK = o.prefixesBitset.NextSet(oIdx); oOK {
				if n.lpmTest(oIdx) {
					return true
				}

				oIdx++
			}
		}
	}

	return false
}

// overlapsChildsIn, test if prefixes in n overlaps child octets in o.
func (n *node[V]) overlapsChildsIn(o *node[V]) bool {
	pfxLen := len(n.prefixes)
	childLen := len(o.children)

	// heuristic, compare benchmarks
	// when will re range over the children and when will we do bitset calc?
	magicNumber := 15
	doRange := childLen < magicNumber || pfxLen > magicNumber

	// do range over, not so many childs and maybe to many prefixes
	if doRange {
		var oAddr uint

		ok := true
		for ok {
			// does any route in o overlap this child from n
			if oAddr, ok = o.childrenBitset.NextSet(oAddr); ok {
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
	prefixBacking := make([]uint64, 8)
	prefixRoutes := bitset.From(prefixBacking)

	idxBacking := make([]uint, maxNodePrefixes)
	for _, idx := range n.allStrideIndexes(idxBacking) {
		a8 := allotLookupTbl[idx]
		prefixRoutes.InPlaceUnion(bitset.From(a8[:]))
	}

	// shift children bitset by firstHostIndex
	c8 := make([]uint64, 8)
	copy(c8[4:], o.childrenBitset.Bytes()) // 4*64= 256
	hostRoutes := bitset.From(c8)

	return prefixRoutes.IntersectionCardinality(hostRoutes) > 0
}

// overlapsSameChildsRec, find same octets with bitset intersection.
// rec-descent with same child octet in n an o,
func (n *node[V]) overlapsSameChildsRec(o *node[V]) bool {
	// gimmicks, clone a bitset without heap allocation
	// 4*64=256, maxNodeChildren
	a4 := make([]uint64, 4)
	copy(a4, n.childrenBitset.Bytes())
	nChildrenBitsetCloned := bitset.From(a4)

	// intersect in place the child bitsets from n and o
	nChildrenBitsetCloned.InPlaceIntersection(o.childrenBitset)

	// gimmick, don't allocate
	addrBuf := [maxNodeChildren]uint{}
	_, allCommonChilds := nChildrenBitsetCloned.NextSetMany(0, addrBuf[:])

	// range over all child addrs, common in n and o
	for _, addr := range allCommonChilds {
		oChild := o.getChild(byte(addr))
		nChild := n.getChild(byte(addr))

		// rec-descent with same child
		if nChild.overlapsRec(oChild) {
			return true
		}
	}

	return false
}

func (n *node[V]) overlapsOneChildIn(o *node[V]) bool {
	// get the single addr and child
	addr, _ := o.childrenBitset.NextSet(0)
	oChild := o.children[0]

	if nChild := n.getChild(byte(addr)); nChild != nil {
		return nChild.overlapsRec(oChild)
	}

	return false
}

func (n *node[V]) overlapsOneRouteIn(o *node[V]) bool {
	// get the single prefix from o
	idx, _ := o.prefixesBitset.NextSet(0)

	// 1. Test if any route in this node overlaps prefix?
	if n.lpmTest(idx) {
		return true
	}

	// 2. Test if prefix overlaps any route in this node
	// use bitset intersection with alloted stride table instead of range loops

	// precalculated allotment for idx (complete binary tree as bitset)
	pfxBuf := allotLookupTbl[idx]
	allotedPrefixRoutes := bitset.From(pfxBuf[:])

	// use bitset intersection instead of range loops
	return allotedPrefixRoutes.IntersectionCardinality(n.prefixesBitset) > 0
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

	// precalculated allotment for idx (complete binary tree as bitset)
	allotmentForIdx := allotLookupTbl[idx]
	allotedPrefixRoutes := bitset.From(allotmentForIdx[:])

	// use bitset intersection instead of range loops
	if allotedPrefixRoutes.IntersectionCardinality(n.prefixesBitset) != 0 {
		return true
	}

	// 3. Test if prefix overlaps any child in this node
	// use bitsets intersection instead of range loops

	// shift children bitset by firstHostIndex
	c8 := make([]uint64, 8)
	copy(c8[4:], n.childrenBitset.Bytes()) // 4*64= 256
	hostRoutes := bitset.From(c8)

	// use bitsets intersection instead of range loops
	return allotedPrefixRoutes.IntersectionCardinality(hostRoutes) != 0
}
