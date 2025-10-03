// Code generated from file "nodeoverlaps_tmpl.go"; DO NOT EDIT.

// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package nodes

import (
	"net/netip"

	"github.com/gaissmai/bart/internal/allot"
	"github.com/gaissmai/bart/internal/art"
)

// overlaps recursively compares two trie nodes and returns true
// if any of their prefixes or descendants overlap.
//
// The implementation checks for:
// 1. Direct overlapping prefixes on this node level
// 2. Prefixes in one node overlapping with children in the other
// 3. Matching child addresses in both nodes, which are recursively compared
//
// All 12 possible type combinations for child entries (node, leaf, fringe) are supported.
//
// The function is optimized for early exit on first match and uses heuristics to
// choose between set-based and loop-based matching for performance.
func (n *BartNode[V]) Overlaps(o *BartNode[V], depth int) bool {
	nPfxCount := n.PrefixCount()
	oPfxCount := o.PrefixCount()

	nChildCount := n.ChildCount()
	oChildCount := o.ChildCount()

	// ##############################
	// 1. Test if any routes overlaps
	// ##############################

	// full cross check
	if nPfxCount > 0 && oPfxCount > 0 {
		if n.OverlapsRoutes(o) {
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

		nPfxCount = n.PrefixCount()
		oPfxCount = o.PrefixCount()

		nChildCount = n.ChildCount()
		oChildCount = o.ChildCount()
	}

	if nPfxCount > 0 && oChildCount > 0 {
		if n.OverlapsChildrenIn(o) {
			return true
		}
	}

	// symmetric reverse
	if oPfxCount > 0 && nChildCount > 0 {
		if o.OverlapsChildrenIn(n) {
			return true
		}
	}

	// ############################################
	// 3. children with same octet in nodes n and o
	// ############################################

	// stop condition, n or o have no children
	if nChildCount == 0 || oChildCount == 0 {
		return false
	}

	// stop condition, no child with identical octet in n and o
	if !n.Children.Intersects(&o.Children.BitSet256) {
		return false
	}

	return n.OverlapsSameChildren(o, depth)
}

// overlapsRoutes compares the prefix sets of two nodes (n and o).
//
// It first checks for direct bitset intersection (identical indices),
// then walks both prefix sets using lpmTest to detect if any
// of the n-prefixes is contained in o, or vice versa.
func (n *BartNode[V]) OverlapsRoutes(o *BartNode[V]) bool {
	// some prefixes are identical, trivial overlap
	if n.Prefixes.Intersects(&o.Prefixes.BitSet256) {
		return true
	}

	// get the lowest idx (biggest prefix)
	nFirstIdx, _ := n.Prefixes.FirstSet()
	oFirstIdx, _ := o.Prefixes.FirstSet()

	// start with other min value
	nIdx := oFirstIdx
	oIdx := nFirstIdx

	nOK := true
	oOK := true

	// zip, range over n and o together to help chance on its way
	for nOK || oOK {
		if nOK {
			// does any route in o overlap this prefix from n
			if nIdx, nOK = n.Prefixes.NextSet(nIdx); nOK {
				if o.Contains(nIdx) {
					return true
				}

				if nIdx == 255 {
					// stop, don't overflow uint8!
					nOK = false
				} else {
					nIdx++
				}
			}
		}

		if oOK {
			// does any route in n overlap this prefix from o
			if oIdx, oOK = o.Prefixes.NextSet(oIdx); oOK {
				if n.Contains(oIdx) {
					return true
				}

				if oIdx == 255 {
					// stop, don't overflow uint8!
					oOK = false
				} else {
					oIdx++
				}
			}
		}
	}

	return false
}

// overlapsChildrenIn checks whether the prefixes in node n
// overlap with any children (by address range) in node o.
//
// Uses bitset intersection or manual iteration heuristically,
// depending on prefix and child count.
//
// Bitset-based matching uses precomputed coverage tables
// to avoid per-address looping. This is critical for high fan-out nodes.
func (n *BartNode[V]) OverlapsChildrenIn(o *BartNode[V]) bool {
	pfxCount := n.PrefixCount()
	childCount := o.ChildCount()

	// heuristic: 15 is the crossover point where bitset operations become
	// more efficient than iteration, determined by micro benchmarks on typical
	// routing table distributions
	const overlapsRangeCutoff = 15

	doRange := childCount < overlapsRangeCutoff || pfxCount > overlapsRangeCutoff

	// do range over, not so many children and maybe too many prefixes for other algo below
	var buf [256]uint8
	if doRange {
		for _, addr := range o.Children.AsSlice(&buf) {
			if n.Contains(art.OctetToIdx(addr)) {
				return true
			}
		}
		return false
	}

	// do bitset intersection, alloted route table with child octets
	// maybe too many children for range-over or not so many prefixes to
	// build the alloted routing table from them

	// use allot table with prefixes as bitsets, bitsets are precalculated.
	for _, idx := range n.Prefixes.AsSlice(&buf) {
		if o.Children.Intersects(&allot.FringeRoutesLookupTbl[idx]) {
			return true
		}
	}

	return false
}

// overlapsSameChildren compares all matching child addresses (octets)
// between node n and node o recursively.
//
// For each shared address, the corresponding child nodes (of any type)
// are compared using BartNodeOverlapsTwoChildren, which handles all
// node/leaf/fringe combinations.
func (n *BartNode[V]) OverlapsSameChildren(o *BartNode[V], depth int) bool {
	// intersect the child bitsets from n with o
	commonChildren := n.Children.Intersection(&o.Children.BitSet256)

	for addr, ok := commonChildren.NextSet(0); ok; {
		nChild := n.MustGetChild(addr)
		oChild := o.MustGetChild(addr)

		if n.OverlapsTwoChildren(nChild, oChild, depth+1) {
			return true
		}

		if addr == 255 {
			break // Prevent uint8 overflow
		}

		addr, ok = commonChildren.NextSet(addr + 1)
	}
	return false
}

// overlapsPrefixAtDepth returns true if any route in the subtree rooted at this node
// overlaps with the given pfx, starting the comparison at the specified depth.
//
// This function supports structural overlap detection even in compressed or sparse
// paths within the trie, including fringe and leaf nodes. Matching is directional:
// it returns true if a route fully covers pfx, or if pfx covers an existing route.
//
// At each step, it checks for visible prefixes and children that may intersect the
// target prefix via stride-based longest-prefix test. The walk terminates early as
// soon as a structural overlap is found.
//
// This function underlies the top-level OverlapsPrefix behavior and handles details of
// trie traversal across varying prefix lengths and compression levels.
func (n *BartNode[V]) OverlapsPrefixAtDepth(pfx netip.Prefix, depth int) bool {
	ip := pfx.Addr()
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := LastOctetPlusOneAndLastBits(pfx)

	for ; depth < len(octets); depth++ {
		if depth > lastOctetPlusOne {
			break
		}

		octet := octets[depth]

		// full octet path in node trie, check overlap with last prefix octet
		if depth == lastOctetPlusOne {
			return n.OverlapsIdx(art.PfxToIdx(octet, lastBits))
		}

		// test if any route overlaps prefix´ so far
		// no best match needed, forward tests without backtracking
		if n.PrefixCount() != 0 && n.Contains(art.OctetToIdx(octet)) {
			return true
		}

		if !n.Children.Test(octet) {
			return false
		}

		// next child, node or leaf
		switch kid := n.MustGetChild(octet).(type) {
		case *BartNode[V]:
			n = kid
			continue

		case *LeafNode[V]:
			return kid.Prefix.Overlaps(pfx)

		case *FringeNode[V]:
			return true

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable: " + pfx.String())
}

// overlapsIdx returns true if the given prefix index overlaps with any entry in this node.
//
// The overlap detection considers three categories:
//
//  1. Whether any stored prefix in this node covers the requested prefix (LPM test)
//  2. Whether the requested prefix covers any stored route in the node
//  3. Whether the requested prefix overlaps with any fringe or child entry
//
// Internally, it leverages precomputed bitsets from the allotment model,
// using fast bitwise set intersections instead of explicit range comparisons.
// This enables high-performance overlap checks on a single stride level
// without descending further into the trie.
func (n *BartNode[V]) OverlapsIdx(idx uint8) bool {
	// 1. Test if any route in this node overlaps prefix?
	if n.Contains(idx) {
		return true
	}

	// 2. Test if prefix overlaps any route in this node
	if n.Prefixes.Intersects(&allot.PfxRoutesLookupTbl[idx]) {
		return true
	}

	// 3. Test if prefix overlaps any child in this node
	return n.Children.Intersects(&allot.FringeRoutesLookupTbl[idx])
}

// overlapsTwoChildren handles all 3x3 combinations of
// node kinds (node, leaf, fringe).
//
//	3x3 possible different combinations for n and o
//
//	node, node    --> overlaps rec descent
//	node, leaf    --> overlapsPrefixAtDepth
//	node, fringe  --> true
//
//	leaf, node    --> overlapsPrefixAtDepth
//	leaf, leaf    --> netip.Prefix.Overlaps
//	leaf, fringe  --> true
//
//	fringe, node    --> true
//	fringe, leaf    --> true
//	fringe, fringe  --> true
func (n *BartNode[V]) OverlapsTwoChildren(nChild, oChild any, depth int) bool {
	// child type detection
	nNode, nIsNode := nChild.(*BartNode[V])
	nLeaf, nIsLeaf := nChild.(*LeafNode[V])
	_, nIsFringe := nChild.(*FringeNode[V])

	oNode, oIsNode := oChild.(*BartNode[V])
	oLeaf, oIsLeaf := oChild.(*LeafNode[V])
	_, oIsFringe := oChild.(*FringeNode[V])

	// Handle all 9 combinations with a single expression
	switch {
	// NODE cases
	case nIsNode && oIsNode:
		return nNode.Overlaps(oNode, depth)
	case nIsNode && oIsLeaf:
		return nNode.OverlapsPrefixAtDepth(oLeaf.Prefix, depth)
	case nIsNode && oIsFringe:
		return true

	// LEAF cases
	case nIsLeaf && oIsNode:
		return oNode.OverlapsPrefixAtDepth(nLeaf.Prefix, depth)
	case nIsLeaf && oIsLeaf:
		return oLeaf.Prefix.Overlaps(nLeaf.Prefix)
	case nIsLeaf && oIsFringe:
		return true

	// FRINGE cases
	case nIsFringe:
		return true // fringe overlaps with everything

	default:
		panic("logic error, wrong node type combination")
	}
}
