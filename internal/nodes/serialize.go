// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package nodes

import (
	"cmp"
	"fmt"
	"io"
	"net/netip"
	"slices"

	"github.com/gaissmai/bart/internal/art"
)

// TrieItem, a node has no path information about its predecessors,
// we collect this during the recursive descent.
type TrieItem[V any] struct {
	// for traversing, path/depth/idx is needed to get the CIDR back from the trie.
	Node  NodeReader[V]
	Is4   bool
	Path  StridePath
	Depth int
	Idx   uint8

	// for printing
	Cidr netip.Prefix
	Val  V
}

func ShouldPrintValues[V any]() bool {
	var zero V

	_, isEmptyStruct := any(zero).(struct{})
	return !isEmptyStruct
}

// CmpPrefix, helper function, compare func for prefix sort,
// all cidrs are already normalized
func CmpPrefix(a, b netip.Prefix) int {
	if cmpAddr := a.Addr().Compare(b.Addr()); cmpAddr != 0 {
		return cmpAddr
	}

	return cmp.Compare(a.Bits(), b.Bits())
}

// FprintRec recursively prints a hierarchical CIDR tree representation
// starting from this node to the provided writer. The output shows the
// routing table structure in human-readable format for debugging and analysis.
func FprintRec[V any](n NodeReader[V], w io.Writer, parent TrieItem[V], pad string, printVals bool) error {
	// recursion stop condition
	if n == nil || n.IsEmpty() {
		return nil
	}

	// get direct covered childs for this parent ...
	directItems := DirectItemsRec(n, parent.Idx, parent.Path, parent.Depth, parent.Is4)

	// sort them by netip.Prefix, not by baseIndex
	slices.SortFunc(directItems, func(a, b TrieItem[V]) int {
		return CmpPrefix(a.Cidr, b.Cidr)
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
			_, err = fmt.Fprintf(w, "%s%s\n", pad+glyph, item.Cidr)
		default:
			_, err = fmt.Fprintf(w, "%s%s (%v)\n", pad+glyph, item.Cidr, item.Val)
		}

		if err != nil {
			return err
		}

		// rec-descent with this item as parent
		if err = FprintRec(item.Node, w, item, pad+space, printVals); err != nil {
			return err
		}
	}

	return nil
}

// DirectItemsRec, returns the direct covered items by parent.
// It's a complex recursive function, you have to know the data structure
// by heart to understand this function!
//
// See the  artlookup.pdf paper in the doc folder, the baseIndex function is the key.
func DirectItemsRec[V any](n NodeReader[V], parentIdx uint8, path StridePath, depth int, is4 bool) (directItems []TrieItem[V]) {
	// recursion stop condition
	if n == nil || n.IsEmpty() {
		return nil
	}

	// prefixes:
	// for all idx's (prefixes mapped by baseIndex) in this node
	// do a longest-prefix-match
	for idx, val := range n.AllIndices() {
		// tricky part, skip self
		// test with next possible lpm (idx>>1), it's a complete binary tree
		nextIdx := idx >> 1

		// fast skip, lpm not possible
		if nextIdx < parentIdx {
			continue
		}

		// do a longest-prefix-match
		lpm, _, _ := n.LookupIdx(nextIdx)

		// be aware, 0 is here a possible value for parentIdx and lpm (if not found)
		if lpm == parentIdx {
			// prefix is directly covered by parent

			item := TrieItem[V]{
				Node:  n,
				Is4:   is4,
				Path:  path,
				Depth: depth,
				Idx:   idx,
				// get the prefix back from trie
				Cidr: CidrFromPath(path, depth, is4, idx),
				Val:  val,
			}

			directItems = append(directItems, item)
		}
	}

	// children:
	for addr, child := range n.AllChildren() {
		hostIdx := art.OctetToIdx(addr)

		// do a longest-prefix-match
		lpm, _, _ := n.LookupIdx(hostIdx)

		// be aware, 0 is here a possible value for parentIdx and lpm (if not found)
		if lpm == parentIdx {
			// child is directly covered by parent
			switch kid := child.(type) {
			case NodeReader[V]: // traverse rec-descent, call with next child node,
				// next trie level, set parentIdx to 0, adjust path and depth
				path[depth] = addr
				directItems = append(directItems, DirectItemsRec(kid, 0, path, depth+1, is4)...)

			case *LeafNode[V]: // path-compressed child, stop's recursion for this child
				item := TrieItem[V]{
					Node: nil,
					Is4:  is4,
					Cidr: kid.Prefix,
					Val:  kid.Value,
				}
				directItems = append(directItems, item)

			case *FringeNode[V]: // path-compressed fringe, stop's recursion for this child
				item := TrieItem[V]{
					Node: nil,
					Is4:  is4,
					// get the prefix back from trie
					Cidr: CidrForFringe(path[:], depth, is4, addr),
					Val:  kid.Value,
				}
				directItems = append(directItems, item)

			default:
				panic("logic error, wrong node type")
			}
		}
	}

	return directItems
}
