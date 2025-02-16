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

// kid, a node has no path information about its predecessors,
// we collect this during the recursive descent.
// The path/depth/idx is needed to get the CIDR back within the trie.
type kid[V any] struct {
	// for traversing
	n     *node[V]
	is4   bool
	path  stridePath
	depth int
	idx   uint

	// for printing
	cidr netip.Prefix
	val  V
}

// DumpListNode contains CIDR, value and list of subnets (tree childrens).
// Only used for marshalling.
type DumpListNode[V any] struct {
	CIDR    netip.Prefix      `json:"cidr"`
	Value   V                 `json:"value"`
	Subnets []DumpListNode[V] `json:"subnets,omitempty"`
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

func (n *node[V]) dumpListRec(parentIdx uint, path stridePath, depth int, is4 bool) []DumpListNode[V] {
	// recursion stop condition
	if n == nil {
		return nil
	}

	directKids := n.getKidsRec(parentIdx, path, depth, is4)
	slices.SortFunc(directKids, cmpKidByPrefix[V])

	nodes := make([]DumpListNode[V], 0, len(directKids))
	for _, kid := range directKids {
		nodes = append(nodes, DumpListNode[V]{
			CIDR:    kid.cidr,
			Value:   kid.val,
			Subnets: kid.n.dumpListRec(kid.idx, kid.path, kid.depth, is4),
		})
	}

	return nodes
}

// getKidsRec, returns the direct kids below path and parentIdx.
// It's a recursive monster together, you have to know the data structure
// by heart to understand this function!
//
// See the  artlookup.pdf paper in the doc folder, the baseIndex function is the key.
func (n *node[V]) getKidsRec(parentIdx uint, path stridePath, depth int, is4 bool) []kid[V] {
	// recursion stop condition
	if n == nil {
		return nil
	}

	var directKids []kid[V]

	for _, idx := range n.prefixes.All() {
		// parent or self, handled alreday in an upper stack frame.
		if idx <= parentIdx {
			continue
		}

		// check if lpmIdx for this idx' parent is equal to parentIdx
		lpmIdx, _, _ := n.lpmGet(idx >> 1)

		// if idx is directKid?
		if lpmIdx == parentIdx {
			cidr := cidrFromPath(path, depth, is4, idx)

			kid := kid[V]{
				n:     n,
				is4:   is4,
				path:  path,
				depth: depth,
				idx:   idx,
				cidr:  cidr,
				val:   n.prefixes.MustGet(idx),
			}

			directKids = append(directKids, kid)
		}
	}

	// the node may have childs and leaves, the rec-descent monster starts
	for i, addr := range n.children.All() {
		// do a longest-prefix-match
		lpmIdx, _, _ := n.lpmGet(art.HostIdx(addr))
		if lpmIdx == parentIdx {
			switch k := n.children.Items[i].(type) {
			case *node[V]:
				path[depth] = byte(addr)

				// traverse, rec-descent call with next child node
				directKids = append(directKids, k.getKidsRec(0, path, depth+1, is4)...)
			case *leaf[V]:
				kid := kid[V]{
					n:    nil, // path compressed item, stop recursion
					is4:  is4,
					cidr: k.prefix,
					val:  k.value,
				}

				directKids = append(directKids, kid)
			}
		}
	}

	return directKids
}

// cmpKidByPrefix, helper function, all prefixes are already normalized (Masked).
func cmpKidByPrefix[V any](a, b kid[V]) int {
	return cmpPrefix(a.cidr, b.cidr)
}

// cmpPrefix, helper function, compare func for prefix sort,
// all cidrs are already normalized
func cmpPrefix(a, b netip.Prefix) int {
	if cmp := a.Addr().Compare(b.Addr()); cmp != 0 {
		return cmp
	}

	return cmp.Compare(a.Bits(), b.Bits())
}
