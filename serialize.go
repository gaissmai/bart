// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"net/netip"
	"slices"
	"strings"

	"github.com/gaissmai/bart/internal/art"
)

// DumpListNode contains CIDR, Value and Subnets, representing the trie
// in a sorted, recursive representation, especially useful for serialization.
type DumpListNode[V any] struct {
	CIDR    netip.Prefix      `json:"cidr"`
	Value   V                 `json:"value"`
	Subnets []DumpListNode[V] `json:"subnets,omitempty"`
}

// trieItem, a node has no path information about its predecessors,
// we collect this during the recursive descent.
type trieItem[V any] struct {
	// for traversing, path/depth/idx is needed to get the CIDR back from the trie.
	n     *node[V]
	is4   bool
	path  stridePath
	depth int
	idx   uint

	// for printing
	cidr netip.Prefix
	val  V
}

// String returns a hierarchical tree diagram of the ordered CIDRs
// as string, just a wrapper for [Table.Fprint].
// If Fprint returns an error, String panics.
func (t *Table[V]) String() string {
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
func (t *Table[V]) Fprint(w io.Writer) error {
	if t == nil || w == nil {
		return nil
	}

	if t.size4 != 0 {
		if _, err := fmt.Fprint(w, "▼\n"); err != nil {
			return err
		}

		start4 := DumpListNode[V]{Subnets: t.DumpList4()}
		if err := start4.printNodeRec(w, ""); err != nil {
			return err
		}
	}

	if t.size6 != 0 {
		if _, err := fmt.Fprint(w, "▼\n"); err != nil {
			return err
		}

		start6 := DumpListNode[V]{Subnets: t.DumpList6()}
		if err := start6.printNodeRec(w, ""); err != nil {
			return err
		}
	}

	return nil
}

// MarshalText implements the [encoding.TextMarshaler] interface,
// just a wrapper for [Table.Fprint].
func (t *Table[V]) MarshalText() ([]byte, error) {
	w := new(bytes.Buffer)
	if err := t.Fprint(w); err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

// MarshalJSON dumps the table into two sorted lists: for ipv4 and ipv6.
// Every root and subnet is an array, not a map, because the order matters.
func (t *Table[V]) MarshalJSON() ([]byte, error) {
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
func (t *Table[V]) DumpList4() []DumpListNode[V] {
	if t == nil {
		return nil
	}
	return t.root4.dumpListRec(0, stridePath{}, 0, true)
}

// DumpList6 dumps the ipv6 tree into a list of roots and their subnets.
// It can be used to analyze the tree or build custom json representation.
func (t *Table[V]) DumpList6() []DumpListNode[V] {
	if t == nil {
		return nil
	}
	return t.root6.dumpListRec(0, stridePath{}, 0, false)
}

// dumpListRec, build the data structure rec-descent with the help
// of getDirectCoveredEntries()
func (n *node[V]) dumpListRec(parentIdx uint, path stridePath, depth int, is4 bool) []DumpListNode[V] {
	// recursion stop condition
	if n == nil {
		return nil
	}

	directItems := n.directItemsRec(parentIdx, path, depth, is4)

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
func (n *node[V]) directItemsRec(parentIdx uint, path stridePath, depth int, is4 bool) (directItems []trieItem[V]) {
	// recursion stop condition
	if n == nil {
		return nil
	}

	// prefixes:
	// for all idx's (prefixes mapped by baseIndex) in this node
	// do a longest-prefix-match
	for i, idx := range n.prefixes.AsSlice(make([]uint, 0, maxNodePrefixes)) {
		// tricky part, skip self, test with next possible lpm (idx>>1), it's a complete binary tree
		nextIdx := idx >> 1

		// fast skip, lpm not possible
		if nextIdx < parentIdx {
			continue
		}

		// do a longest-prefix-match
		lpm, _, _ := n.lpmGet(nextIdx)

		// be aware, 0 is here a possible value for parentIdx and lpm (if not found)
		if lpm == parentIdx {
			// prefix is directly covered by parent

			item := trieItem[V]{
				n:     n,
				is4:   is4,
				path:  path,
				depth: depth,
				idx:   idx,
				//
				cidr: cidrFromPath(path, depth, is4, idx),
				val:  n.prefixes.Items[i],
			}

			directItems = append(directItems, item)
		}
	}

	// children:
	for i, addr := range n.children.AsSlice(make([]uint, 0, maxNodeChildren)) {
		hostIdx := art.HostIdx(addr)

		// fast skip, lpm not possible
		if hostIdx < parentIdx {
			continue
		}

		// do a longest-prefix-match
		lpm, _, _ := n.lpmGet(hostIdx)

		// be aware, 0 is here a possible value for parentIdx and lpm (if not found)
		if lpm == parentIdx {
			// child is directly covered by parent
			switch kid := n.children.Items[i].(type) {
			case *node[V]: // traverse rec-descent, call with next child node,
				// next trie level, set parentIdx to 0, adjust path and depth
				path[depth] = byte(addr)
				directItems = append(directItems, kid.directItemsRec(0, path, depth+1, is4)...)

			case *leaf[V]: // path-compressed child, stop's recursion for this child
				item := trieItem[V]{
					n:    nil,
					is4:  is4,
					cidr: kid.prefix,
					val:  kid.value,
				}

				directItems = append(directItems, item)
			}
		}
	}

	return directItems
}

// printNodeRec, the rec-descent tree printer.
func (dln *DumpListNode[V]) printNodeRec(w io.Writer, pad string) error {
	// symbols used in tree
	glyphe := "├─ "
	spacer := "│  "

	// range over all dumplist-nodes on this level
	for i, dumpListNode := range dln.Subnets {
		// ... treat last kid special
		if i == len(dln.Subnets)-1 {
			glyphe = "└─ "
			spacer = "   "
		}

		// print prefix and val, padded with glyphe
		if _, err := fmt.Fprintf(w, "%s%s (%v)\n", pad+glyphe, dumpListNode.CIDR, dumpListNode.Value); err != nil {
			return err
		}

		// rec-descent with this node
		if err := dumpListNode.printNodeRec(w, pad+spacer); err != nil {
			return err
		}
	}

	return nil
}

// cmpPrefix, helper function, compare func for prefix sort,
// all cidrs are already normalized
func cmpPrefix(a, b netip.Prefix) int {
	if cmp := a.Addr().Compare(b.Addr()); cmp != 0 {
		return cmp
	}

	return cmp.Compare(a.Bits(), b.Bits())
}
