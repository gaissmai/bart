package bart

import (
	"net/netip"

	"github.com/gaissmai/bart/internal/allot"
	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/bitset"
)

// overlaps returns true if any IP in the nodes n or o overlaps.
func (n *node[V]) overlaps(o *node[V], depth int) bool {
	nPfxCount := n.prefixes.Len()
	oPfxCount := o.prefixes.Len()

	nChildCount := n.children.Len()
	oChildCount := o.children.Len()

	// ##############################
	// 1. Test if any routes overlaps
	// ##############################

	// full cross check
	if nPfxCount > 0 && oPfxCount > 0 {
		if n.overlapsRoutes(o) {
			return true
		}
	}

	// ####################################
	// 2. Test if routes overlaps any child
	// ####################################

	// swap nodes to help chance on its way,
	// if the first call to expensive overlapsChildrenIn() is already true,
	// if both orders are false it doesn't help either
	if nChildCount > oChildCount {
		n, o = o, n

		nPfxCount = n.prefixes.Len()
		oPfxCount = o.prefixes.Len()

		nChildCount = n.children.Len()
		oChildCount = o.children.Len()
	}

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

	// ###########################################
	// 3. childs with same octet in nodes n and o
	// ###########################################

	// stop condition, n or o have no childs
	if nChildCount == 0 || oChildCount == 0 {
		return false
	}

	// stop condition, no child with identical octet in n and o
	if !n.children.IntersectsAny(&o.children.BitSetFringe) {
		return false
	}

	return n.overlapsSameChildren(o, depth)
}

// overlapsRoutes, test if n overlaps o prefixes and vice versa
func (n *node[V]) overlapsRoutes(o *node[V]) bool {
	// some prefixes are identical, trivial overlap
	if n.prefixes.IntersectsAny(&o.prefixes.BitSetFringe) {
		return true
	}

	// get the lowest idx (biggest prefix)
	nFirstIdx, _ := n.prefixes.FirstSet()
	oFirstIdx, _ := o.prefixes.FirstSet()

	// start with other min value, see ART algo
	nIdx := oFirstIdx
	oIdx := nFirstIdx

	// make full cross check
	nOK := true
	oOK := true

	// zip, range over n and o together to help chance on its way
	for nOK || oOK {
		if nOK {
			// does any route in o overlap this prefix from n
			if nIdx, nOK = n.prefixes.NextSet(nIdx); nOK {
				if o.lpmTest(nIdx) {
					return true
				}

				nIdx++
			}
		}

		if oOK {
			// does any route in n overlap this prefix from o
			if oIdx, oOK = o.prefixes.NextSet(oIdx); oOK {
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
	// when will we range over the children and when will we do bitset calc?
	magicNumber := 15
	doRange := childCount < magicNumber || pfxCount > magicNumber

	// do range over, not so many childs and maybe to many prefixes for other algo below
	if doRange {
		lowerBound, _ := n.prefixes.FirstSet()
		for _, addr := range o.children.AsSlice(make([]uint, 0, maxNodeChildren)) {
			idx := art.HostIdx(addr)
			if idx < lowerBound { // lpm match impossible
				continue
			}
			if n.lpmTest(idx) {
				return true
			}
		}

		return false
	}

	// do bitset intersection, alloted route table with child octets
	// maybe to many childs for range over or not so many prefixes to
	// build the alloted routing table from them

	// make allot table with prefixes as bitsets, bitsets are precalculated
	// just union the bitsets to one bitset (allot table) for all prefixes
	// in this node
	prefixRoutes := bitset.BitSetFringe{}

	allIndices := n.prefixes.AsSlice(make([]uint, 0, maxNodePrefixes))

	for _, idx := range allIndices {
		// get pre alloted bitset for idx
		prefixRoutes.InPlaceUnion(allot.LookupTblFringe[idx])
	}

	return prefixRoutes.IntersectsAny(&o.children.BitSetFringe)
}

// overlapsSameChildren, find same octets with bitset intersection.
func (n *node[V]) overlapsSameChildren(o *node[V], depth int) bool {
	nChildrenBitsetCloned := n.children.BitSetFringe

	// intersect in place the cloned child bitset from n with o
	nChildrenBitsetCloned.InPlaceIntersection(&o.children.BitSetFringe)

	allCommonChildren := nChildrenBitsetCloned.AsSlice(make([]uint, 0, maxNodeChildren))

	// range over all child addrs, common in n and o
	for _, addr := range allCommonChildren {
		nChild := n.children.MustGet(addr)
		oChild := o.children.MustGet(addr)

		if overlapsTwoChilds[V](nChild, oChild, depth+1) {
			return true
		}
	}

	return false
}

// overlapsTwoChilds, childs can be node or leaf.
func overlapsTwoChilds[V any](nChild, oChild any, depth int) bool {
	//  4 possible different combinations for n and o
	//
	//  node, node  --> overlapsRec
	//  node, leaf  --> overlapsPrefixAtDepth
	//  leaf, node  --> overlapsPrefixAtDepth
	//  leaf, leaf  --> netip.Prefix.Overlaps
	//
	switch nKind := nChild.(type) {
	case *node[V]:
		switch oKind := oChild.(type) {
		case *node[V]: // node, node
			return nKind.overlaps(oKind, depth)
		case *leaf[V]: // node, leaf
			return nKind.overlapsPrefixAtDepth(oKind.prefix, depth)
		}

	case *leaf[V]:
		switch oKind := oChild.(type) {
		case *node[V]: // leaf, node
			return oKind.overlapsPrefixAtDepth(nKind.prefix, depth)
		case *leaf[V]: // leaf, leaf
			return oKind.prefix.Overlaps(nKind.prefix)
		}

	default:
		panic("logic error, wrong node type")
	}

	return false
}

// overlapsPrefixAtDepth, returns true if node overlaps with prefix
// starting with prefix octet at depth.
//
// Needed for path compressed prefix some level down in the node trie.
func (n *node[V]) overlapsPrefixAtDepth(pfx netip.Prefix, depth int) bool {
	ip := pfx.Addr()
	bits := pfx.Bits()
	octets := ip.AsSlice()
	lastIdx, lastBits := lastOctetIdxAndBits(bits)

	for ; depth < len(octets); depth++ {
		octet := octets[depth&0xf]
		addr := uint(octet)

		// full octet path in node trie, check overlap with last prefix octet
		if depth == lastIdx {
			return n.overlapsIdx(art.PfxToIdx(octet, lastBits))
		}

		// test if any route overlaps prefix´ so far
		// no best match needed, forward tests without backtracking
		if n.prefixes.Len() != 0 && n.lpmTest(art.HostIdx(addr)) {
			return true
		}

		if !n.children.Test(addr) {
			return false
		}

		// next child, node or leaf
		switch kid := n.children.MustGet(addr).(type) {
		case *node[V]:
			n = kid
			continue
		case *leaf[V]:
			return kid.prefix.Overlaps(pfx)

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable: " + pfx.String())
}

// overlapsIdx returns true if node overlaps with prefix.
func (n *node[V]) overlapsIdx(idx uint) bool {
	// 1. Test if any route in this node overlaps prefix?
	if n.lpmTest(idx) {
		return true
	}

	// 2. Test if prefix overlaps any route in this node

	// use bitset intersections instead of range loops
	// shallow copy pre alloted bitset for idx
	allotedPrefixRoutes := allot.LookupTblFringe[idx]
	if allotedPrefixRoutes.IntersectsAny(&n.prefixes.BitSetFringe) {
		return true
	}

	// 3. Test if prefix overlaps any child in this node

	// use bitsets intersection instead of range loops
	return allotedPrefixRoutes.IntersectsAny(&n.children.BitSetFringe)
}
