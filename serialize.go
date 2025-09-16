// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"cmp"
	"fmt"
	"io"
	"net/netip"
	"slices"

	"github.com/gaissmai/bart/internal/art"
)

// trieItem, a node has no path information about its predecessors,
// we collect this during the recursive descent.
type trieItem[V any] struct {
	// for traversing, path/depth/idx is needed to get the CIDR back from the trie.
	n     nodeReader[V]
	is4   bool
	path  stridePath
	depth int
	idx   uint8

	// for printing
	cidr netip.Prefix
	val  V
}

// DumpListNode contains CIDR, Value and Subnets, representing the trie
// in a sorted, recursive representation, especially useful for serialization.
type DumpListNode[V any] struct {
	CIDR    netip.Prefix      `json:"cidr"`
	Value   V                 `json:"value"`
	Subnets []DumpListNode[V] `json:"subnets,omitempty"`
}

func shouldPrintValues[V any]() bool {
	var zero V
	_, isEmptyStruct := any(zero).(struct{})
	return !isEmptyStruct
}

// cmpPrefix, helper function, compare func for prefix sort,
// all cidrs are already normalized
func cmpPrefix(a, b netip.Prefix) int {
	if cmpAddr := a.Addr().Compare(b.Addr()); cmpAddr != 0 {
		return cmpAddr
	}

	return cmp.Compare(a.Bits(), b.Bits())
}

// fprintRec recursively prints a hierarchical CIDR tree representation
// starting from this node to the provided writer. The output shows the
// routing table structure in human-readable format for debugging and analysis.
func fprintRec[V any](n nodeReader[V], w io.Writer, parent trieItem[V], pad string, printVals bool) error {
	// recursion stop condition
	if n == nil || n.isEmpty() {
		return nil
	}

	// get direct covered childs for this parent ...
	directItems := directItemsRec(n, parent.idx, parent.path, parent.depth, parent.is4)

	// sort them by netip.Prefix, not by baseIndex
	slices.SortFunc(directItems, func(a, b trieItem[V]) int {
		return cmpPrefix(a.cidr, b.cidr)
	})

	// for all direct item under this node ...
	for i, item := range directItems {
		// symbols used in tree
		glyph := "├─ "
		space := "│  "

		// ... treat last kid special
		if i == len(directItems)-1 {
			glyph = "└─ "
			space = "   "
		}

		var err error
		// val is the empty struct, don't print it
		switch {
		case !printVals:
			_, err = fmt.Fprintf(w, "%s%s\n", pad+glyph, item.cidr)
		default:
			_, err = fmt.Fprintf(w, "%s%s (%v)\n", pad+glyph, item.cidr, item.val)
		}

		if err != nil {
			return err
		}

		// rec-descent with this item as parent
		if err = fprintRec(item.n, w, item, pad+space, printVals); err != nil {
			return err
		}
	}

	return nil
}

// dumpListRec, build the data structure rec-descent with the help
// of directItemsRec.
func dumpListRec[V any](n nodeReader[V], parentIdx uint8, path stridePath, depth int, is4 bool) []DumpListNode[V] {
	// recursion stop condition
	if n == nil {
		return nil
	}

	directItems := directItemsRec(n, parentIdx, path, depth, is4)

	// sort the items by prefix
	slices.SortFunc(directItems, func(a, b trieItem[V]) int {
		return cmpPrefix(a.cidr, b.cidr)
	})

	nodes := make([]DumpListNode[V], 0, len(directItems))

	for _, item := range directItems {
		nodes = append(nodes, DumpListNode[V]{
			CIDR:  item.cidr,
			Value: item.val,
			// build it rec-descent
			Subnets: dumpListRec(item.n, item.idx, item.path, item.depth, is4),
		})
	}

	return nodes
}

// directItemsRec, returns the direct covered items by parent.
// It's a complex recursive function, you have to know the data structure
// by heart to understand this function!
//
// See the  artlookup.pdf paper in the doc folder, the baseIndex function is the key.
func directItemsRec[V any](n nodeReader[V], parentIdx uint8, path stridePath, depth int, is4 bool) (directItems []trieItem[V]) {
	// recursion stop condition
	if n == nil || n.isEmpty() {
		return nil
	}

	// prefixes:
	// for all idx's (prefixes mapped by baseIndex) in this node
	// do a longest-prefix-match
	for _, idx := range n.getIndices() {
		// tricky part, skip self
		// test with next possible lpm (idx>>1), it's a complete binary tree
		nextIdx := idx >> 1

		// fast skip, lpm not possible
		if nextIdx < parentIdx {
			continue
		}

		// do a longest-prefix-match
		lpm, _, _ := n.lookupIdx(uint(nextIdx))

		// be aware, 0 is here a possible value for parentIdx and lpm (if not found)
		if lpm == parentIdx {
			// prefix is directly covered by parent

			item := trieItem[V]{
				n:     n,
				is4:   is4,
				path:  path,
				depth: depth,
				idx:   idx,
				// get the prefix back from trie
				cidr: cidrFromPath(path, depth, is4, idx),
				val:  n.mustGetPrefix(idx),
			}

			directItems = append(directItems, item)
		}
	}

	// children:
	for _, addr := range n.getChildAddrs() {
		hostIdx := art.OctetToIdx(addr)

		// do a longest-prefix-match
		lpm, _, _ := n.lookupIdx(hostIdx)

		// be aware, 0 is here a possible value for parentIdx and lpm (if not found)
		if lpm == parentIdx {
			// child is directly covered by parent
			switch kid := n.mustGetChild(addr).(type) {
			case nodeReader[V]: // traverse rec-descent, call with next child node,
				// next trie level, set parentIdx to 0, adjust path and depth
				path[depth] = addr
				directItems = append(directItems, directItemsRec(kid, 0, path, depth+1, is4)...)

			case *leafNode[V]: // path-compressed child, stop's recursion for this child
				item := trieItem[V]{
					n:    nil,
					is4:  is4,
					cidr: kid.prefix,
					val:  kid.value,
				}
				directItems = append(directItems, item)

			case *fringeNode[V]: // path-compressed fringe, stop's recursion for this child
				item := trieItem[V]{
					n:   nil,
					is4: is4,
					// get the prefix back from trie
					cidr: cidrForFringe(path[:], depth, is4, addr),
					val:  kid.value,
				}
				directItems = append(directItems, item)

			default:
				panic("logic error, wrong node type")
			}
		}
	}

	return directItems
}
