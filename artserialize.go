// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/netip"
	"slices"
	"strings"

	"github.com/gaissmai/bart/internal/art"
)

// artTrieItem, a node has no path information about its predecessors,
// we collect this during the recursive descent.
type artTrieItem[V any] struct {
	// for traversing, path/depth/idx is needed to get the CIDR back from the trie.
	n     *artNode[V]
	is4   bool
	path  stridePath
	depth int
	idx   uint8

	// for printing
	cidr netip.Prefix
	val  V
}

// String returns a hierarchical tree diagram of the ordered CIDRs
// as string, just a wrapper for [Table.Fprint].
// If Fprint returns an error, String panics.
func (t *ArtTable[V]) String() string {
	w := new(strings.Builder)
	if err := t.Fprint(w); err != nil {
		panic(err)
	}

	return w.String()
}

// Fprint writes a hierarchical tree diagram of the ordered CIDRs
// with default formatted payload V to w.
//
// The order from top to bottom is in ascending order of the prefix address
// and the subtree structure is determined by the CIDRs coverage.
//
//	▼
//	├─ 10.0.0.0/8 (V)
//	│  ├─ 10.0.0.0/24 (V)
//	│  └─ 10.0.1.0/24 (V)
//	├─ 127.0.0.0/8 (V)
//	│  └─ 127.0.0.1/32 (V)
//	├─ 169.254.0.0/16 (V)
//	├─ 172.16.0.0/12 (V)
//	└─ 192.168.0.0/16 (V)
//	   └─ 192.168.1.0/24 (V)
//	▼
//	└─ ::/0 (V)
//	   ├─ ::1/128 (V)
//	   ├─ 2000::/3 (V)
//	   │  └─ 2001:db8::/32 (V)
//	   └─ fe80::/10 (V)
func (t *ArtTable[V]) Fprint(w io.Writer) error {
	if t == nil || w == nil {
		return nil
	}

	// v4
	if err := t.fprint(w, true); err != nil {
		return err
	}

	// v6
	if err := t.fprint(w, false); err != nil {
		return err
	}

	return nil
}

// fprint is the version dependent adapter to fprintRec.
func (t *ArtTable[V]) fprint(w io.Writer, is4 bool) error {
	n := t.rootNodeByVersion(is4)
	if n.isEmpty() {
		return nil
	}

	if _, err := fmt.Fprint(w, "▼\n"); err != nil {
		return err
	}

	startParent := artTrieItem[V]{
		n:    nil,
		idx:  0,
		path: stridePath{},
		is4:  is4,
	}

	return n.fprintRec(w, startParent, "")
}

// fprintRec, the output is a hierarchical CIDR tree covered starting with this node
func (n *artNode[V]) fprintRec(w io.Writer, parent artTrieItem[V], pad string) error {
	// recursion stop condition
	if n == nil {
		return nil
	}

	// get direct covered childs for this parent ...
	directItems := n.directItemsRec(parent.idx, parent.path, parent.depth, parent.is4)

	// sort them by netip.Prefix, not by baseIndex
	slices.SortFunc(directItems, func(a, b artTrieItem[V]) int {
		return cmpPrefix(a.cidr, b.cidr)
	})

	// symbols used in tree
	glyphe := "├─ "
	spacer := "│  "

	// for all direct item under this node ...
	for i, item := range directItems {
		// ... treat last kid special
		if i == len(directItems)-1 {
			glyphe = "└─ "
			spacer = "   "
		}

		_, err := fmt.Fprintf(w, "%s%s (%v)\n", pad+glyphe, item.cidr, item.val)
		if err != nil {
			return err
		}

		// rec-descent with this item as parent
		if err := item.n.fprintRec(w, item, pad+spacer); err != nil {
			return err
		}
	}

	return nil
}

// MarshalText implements the [encoding.TextMarshaler] interface,
// just a wrapper for [Table.Fprint].
func (t *ArtTable[V]) MarshalText() ([]byte, error) {
	w := new(bytes.Buffer)
	if err := t.Fprint(w); err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

// MarshalJSON dumps the table into two sorted lists: for ipv4 and ipv6.
// Every root and subnet is an array, not a map, because the order matters.
func (t *ArtTable[V]) MarshalJSON() ([]byte, error) {
	if t == nil {
		return nil, nil
	}

	result := struct {
		Ipv4 []DumpListNode[V] `json:"ipv4,omitempty"`
		Ipv6 []DumpListNode[V] `json:"ipv6,omitempty"`
	}{
		Ipv4: t.DumpList4(),
		Ipv6: t.DumpList6(),
	}

	buf, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// DumpList4 dumps the ipv4 tree into a list of roots and their subnets.
// It can be used to analyze the tree or build the text or json serialization.
func (t *ArtTable[V]) DumpList4() []DumpListNode[V] {
	if t == nil {
		return nil
	}
	return t.root4.dumpListRec(0, stridePath{}, 0, true)
}

// DumpList6 dumps the ipv6 tree into a list of roots and their subnets.
// It can be used to analyze the tree or build custom json representation.
func (t *ArtTable[V]) DumpList6() []DumpListNode[V] {
	if t == nil {
		return nil
	}
	return t.root6.dumpListRec(0, stridePath{}, 0, false)
}

// dumpListRec, build the data structure rec-descent with the help
// of directItemsRec.
func (n *artNode[V]) dumpListRec(parentIdx uint8, path stridePath, depth int, is4 bool) []DumpListNode[V] {
	// recursion stop condition
	if n == nil {
		return nil
	}

	directItems := n.directItemsRec(parentIdx, path, depth, is4)

	// sort the items by prefix
	slices.SortFunc(directItems, func(a, b artTrieItem[V]) int {
		return cmpPrefix(a.cidr, b.cidr)
	})

	nodes := make([]DumpListNode[V], 0, len(directItems))

	for _, item := range directItems {
		nodes = append(nodes, DumpListNode[V]{
			CIDR:  item.cidr,
			Value: item.val,
			// build it rec-descent
			Subnets: item.n.dumpListRec(item.idx, item.path, item.depth, is4),
		})
	}

	return nodes
}

// directItemsRec, returns the direct covered items by parent.
// It's a complex recursive function, you have to know the data structure
// by heart to understand this function!
//
// See the  artlookup.pdf paper in the doc folder, the baseIndex function is the key.
func (n *artNode[V]) directItemsRec(parentIdx uint8, path stridePath, depth int, is4 bool) (directItems []artTrieItem[V]) {
	// recursion stop condition
	if n == nil {
		return nil
	}

	// used to compare the LPM match, maybe nil for parentIdx == 0
	parentValPtr := n.prefixes[parentIdx]

	for _, idx := range n.prefixesBitSet.AsSlice(&[256]uint8{}) {
		// tricky part, skip self, test with next possible lpm (idx>>1), it's a complete binary tree
		nextIdx := idx >> 1

		// fast skip, lpm not possible
		if nextIdx < parentIdx {
			continue
		}

		// if prefix is directly covered by parentIdx ...
		valPtr := n.prefixes[nextIdx]

		// both maybe nil
		if valPtr == parentValPtr {

			item := artTrieItem[V]{
				n:     n,
				is4:   is4,
				path:  path,
				depth: depth,
				idx:   idx,
				// get the prefix back from trie
				cidr: cidrFromPath(path, depth, is4, idx),
				val:  *n.prefixes[idx],
			}

			directItems = append(directItems, item)
		}
	}

	// children:
	for _, octet := range n.childrenBitSet.AsSlice(&[256]uint8{}) {
		hostIdx := art.OctetToIdx(octet) >> 1

		// fast skip, lpm not possible
		if hostIdx < uint(parentIdx) {
			continue
		}

		// lookup
		valPtr := n.prefixes[hostIdx] // maybe nil
		if valPtr == parentValPtr {
			kidAny := *n.children[octet]

			switch kid := kidAny.(type) {
			case *artNode[V]:
				// traverse rec-descent, call with next child node,
				// next trie level, set parentIdx to 0, adjust path and depth
				path[depth] = octet
				directItems = append(directItems, kid.directItemsRec(0, path, depth+1, is4)...)

			case *leafNode[V]:
				// path-compressed child, stop's recursion for this child
				item := artTrieItem[V]{
					n:    nil,
					is4:  is4,
					cidr: kid.prefix,
					val:  kid.value,
				}
				directItems = append(directItems, item)

			case *fringeNode[V]:
				// path-compressed fringe, stop's recursion for this child
				item := artTrieItem[V]{
					n:   nil,
					is4: is4,
					// get the prefix back from trie
					cidr: cidrForFringe(path[:], depth, is4, octet),
					val:  kid.value,
				}
				directItems = append(directItems, item)
			}
		}
	}

	return directItems
}
