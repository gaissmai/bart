// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"bytes"
	"cmp"
	"fmt"
	"io"
	"net/netip"
	"slices"
	"strings"
)

// container for the direct kids, needed for hierarchical tree print,
// but see below.
type kidT2[V any] struct {
	// for traversing
	n   *node2[V]
	idx uint
	// for printing
	cidr netip.Prefix
	val  V
}

// MarshalText implements the encoding.TextMarshaler interface,
// just a wrapper for [Table.Fprint].
func (t *Table2[V]) MarshalText() ([]byte, error) {
	t.init()
	w := new(bytes.Buffer)
	if err := t.Fprint(w); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

// String returns a hierarchical tree diagram of the ordered CIDRs
// as string, just a wrapper for [Table.Fprint].
// If Fprint returns an error, String panics.
func (t *Table2[V]) String() string {
	t.init()
	w := new(strings.Builder)
	if err := t.Fprint(w); err != nil {
		panic(err)
	}
	return w.String()
}

// Fprint writes a hierarchical tree diagram of the ordered CIDRs to w.
// If w is nil, Fprint panics.
//
// The order from top to bottom is in ascending order of the prefix address
// and the subtree structure is determined by the CIDRs coverage.
//
//	▼
//	├─ 10.0.0.0/8 (9.9.9.9)
//	│  ├─ 10.0.0.0/24 (8.8.8.8)
//	│  └─ 10.0.1.0/24 (10.0.0.0)
//	├─ 127.0.0.0/8 (127.0.0.1)
//	│  └─ 127.0.0.1/32 (127.0.0.1)
//	├─ 169.254.0.0/16 (10.0.0.0)
//	├─ 172.16.0.0/12 (8.8.8.8)
//	└─ 192.168.0.0/16 (9.9.9.9)
//	   └─ 192.168.1.0/24 (127.0.0.1)
//	▼
//	└─ ::/0 (2001:db8::1)
//	   ├─ ::1/128 (::1%eth0)
//	   ├─ 2000::/3 (2001:db8::1)
//	   │  └─ 2001:db8::/32 (2001:db8::1)
//	   └─ fe80::/10 (::1%lo)
func (t *Table2[V]) Fprint(w io.Writer) error {
	t.init()

	if err := t.fprint(w, true); err != nil {
		return err
	}

	if err := t.fprint(w, false); err != nil {
		return err
	}
	return nil
}

// fprint is the version dependent adapter to fprintRec.
func (t *Table2[V]) fprint(w io.Writer, is4 bool) error {
	rootNode := t.rootNodeByVersion(is4)
	if rootNode.isEmpty() {
		return nil
	}

	if _, err := fmt.Fprint(w, "▼\n"); err != nil {
		return err
	}
	if err := rootNode.fprintRec(w, 0, is4, ""); err != nil {
		return err
	}
	return nil
}

// fprintRec, the output is a hierarchical CIDR tree starting with parentIdx and byte path.
func (n *node2[V]) fprintRec(w io.Writer, parentIdx uint, is4 bool, pad string) error {
	// get direct childs for this parentIdx ...
	directKids := n.getKidsRec(parentIdx, is4)

	// sort them by netip.Prefix, not by baseIndex
	slices.SortFunc(directKids, sortKidsByPrefix2[V])

	// symbols used in tree
	glyphe := "├─ "
	spacer := "│  "

	// for all direct kids under this node ...
	for i, kid := range directKids {
		// ... treat last kid special
		if i == len(directKids)-1 {
			glyphe = "└─ "
			spacer = "   "
		}

		// print prefix and val, padded with glyphe
		if _, err := fmt.Fprintf(w, "%s%s (%v)\n", pad+glyphe, kid.cidr, kid.val); err != nil {
			return err
		}

		// rec-descent with this prefix as parentIdx.
		// hierarchical nested tree view, two rec-descent functions
		// work together to spoil the reader.
		if err := kid.n.fprintRec(w, kid.idx, is4, pad+spacer); err != nil {
			return err
		}
	}

	return nil
}

// getKidsRec, returns the direct kids below path and parentIdx.
// It's a recursive monster together with printRec,
// you have to know the data structure by heart to understand this function!
//
// See the  artlookup.pdf paper in the doc folder,
// the baseIndex function is the key.
func (n *node2[V]) getKidsRec(parentIdx uint, is4 bool) []kidT2[V] {
	directKids := []kidT2[V]{}

	// the node may have prefixes
	for _, idx := range n.allStrideIndexes() {
		// parent or self, handled alreday in an upper stack frame.
		if idx <= parentIdx {
			continue
		}

		// check if lpmIdx for this idx' parent is equal to parentIdx
		if lpmIdx, _, _ := n.lpmByIndex(idx >> 1); lpmIdx == parentIdx {
			val, _ := n.getValByIndex(idx)
			cidr := n.cidrFromIndex(idx, is4)
			directKids = append(directKids, kidT2[V]{n, idx, cidr, val})
		}
	}

	// the node may have childs, the rec-descent monster starts
	for _, octet := range n.allChildAddrs() {
		// do a longest-prefix-match
		if lpmIdx, _, _ := n.lpmByOctet(byte(octet)); lpmIdx == parentIdx {
			// child is directKid

			// get next child node
			c := n.getChild(byte(octet))

			// traverse, rec-descent call with next child node
			directKids = append(directKids, c.getKidsRec(0, is4)...)
		}
	}

	return directKids
}

// sortKidsByPrefix, all prefixes are already normalized (Masked).
func sortKidsByPrefix2[V any](a, b kidT2[V]) int {
	if cmp := a.cidr.Addr().Compare(b.cidr.Addr()); cmp != 0 {
		return cmp
	}
	return cmp.Compare(a.cidr.Bits(), b.cidr.Bits())
}
