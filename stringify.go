// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"slices"
	"strings"
)

// kid, a node has no path information about its predecessors,
// we collect this during the recursive descent.
// The path/depth/idx is needed to get the CIDR back.
type kid[V any] struct {
	// for traversing
	n     *node[V]
	is4   bool
	path  [16]byte
	depth int
	idx   uint

	// for printing
	cidr netip.Prefix
	val  V
}

// MarshalText implements the encoding.TextMarshaler interface,
// just a wrapper for [Table.Fprint].
func (t *Table[V]) MarshalText() ([]byte, error) {
	if !t.isInit() {
		return nil, errors.New("table not initialized")
	}

	w := new(bytes.Buffer)
	if err := t.Fprint(w); err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

// String returns a hierarchical tree diagram of the ordered CIDRs
// as string, just a wrapper for [Table.Fprint].
// If Fprint returns an error, String panics.
func (t *Table[V]) String() string {
	if !t.isInit() {
		return ""
	}

	w := new(strings.Builder)
	if err := t.Fprint(w); err != nil {
		panic(err)
	}

	return w.String()
}

// Fprint writes a hierarchical tree diagram of the ordered CIDRs
// with default formatted payload V to w. If w is nil, Fprint panics.
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
	if !t.isInit() {
		return errors.New("table not initialized")
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
func (t *Table[V]) fprint(w io.Writer, is4 bool) error {
	n := t.rootNodeByVersion(is4)
	if n.isEmpty() {
		return nil
	}

	if _, err := fmt.Fprint(w, "▼\n"); err != nil {
		return err
	}

	startKid := kid[V]{
		n:    nil,
		idx:  0,
		path: zeroPath,
		is4:  is4,
	}

	if err := n.fprintRec(w, startKid, ""); err != nil {
		return err
	}

	return nil
}

// fprintRec, the output is a hierarchical CIDR tree starting with this kid.
func (n *node[V]) fprintRec(w io.Writer, parent kid[V], pad string) error {
	// get direct childs for this kid ...
	directKids := n.getKidsRec(parent.idx, parent.path, parent.depth, parent.is4)

	// sort them by netip.Prefix, not by baseIndex
	slices.SortFunc(directKids, cmpKidByPrefix[V])

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
		if err := kid.n.fprintRec(w, kid, pad+spacer); err != nil {
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
func (n *node[V]) getKidsRec(parentIdx uint, path [16]byte, depth int, is4 bool) []kid[V] {
	directKids := []kid[V]{}

	// make backing array, no heap allocs
	idxBacking := make([]uint, maxNodePrefixes)
	for _, idx := range n.allStrideIndexes(idxBacking) {
		// parent or self, handled alreday in an upper stack frame.
		if idx <= parentIdx {
			continue
		}

		// check if lpmIdx for this idx' parent is equal to parentIdx
		lpmIdx, _, _ := n.lpm(idx >> 1)

		// if idx is directKid?
		if lpmIdx == parentIdx {
			cidr, _ := cidrFromPath(path, depth, is4, idx)

			kid := kid[V]{
				n:     n,
				is4:   is4,
				path:  path,
				depth: depth,
				idx:   idx,
				cidr:  cidr,
				val:   n.mustGetValue(idx),
			}

			directKids = append(directKids, kid)
		}
	}

	// the node may have childs, the rec-descent monster starts
	addrBacking := make([]uint, maxNodeChildren)
	for i, addr := range n.allChildAddrs(addrBacking) {
		octet := byte(addr)

		// do a longest-prefix-match
		lpmIdx, _, _ := n.lpm(hostIndex(octet))
		if lpmIdx == parentIdx {
			c := n.children[i]
			path[depth] = octet

			// traverse, rec-descent call with next child node
			directKids = append(directKids, c.getKidsRec(0, path, depth+1, is4)...)
		}
	}

	return directKids
}

// cidrFromPath, get prefix back from byte path, depth, octet and pfxLen.
func cidrFromPath(path [16]byte, depth int, is4 bool, idx uint) (netip.Prefix, error) {
	octet, pfxLen := idxToPfx(idx)

	// set (partially) masked byte in path at depth
	path[depth] = octet

	// make ip addr from octets
	var ip netip.Addr
	if is4 {
		ip = netip.AddrFrom4([4]byte(path[:4]))
	} else {
		ip = netip.AddrFrom16(path)
	}

	// calc bits with pathLen and pfxLen
	bits := depth*strideLen + pfxLen

	// make a normalized prefix from ip/bits
	return ip.Prefix(bits)
}

// cmpKidByPrefix, all prefixes are already normalized (Masked).
func cmpKidByPrefix[V any](a, b kid[V]) int {
	return cmpPrefix(a.cidr, b.cidr)
}

// cmpPrefix, compare func for prefix sort,
// all cidrs are already normalized
func cmpPrefix(a, b netip.Prefix) int {
	if cmp := a.Addr().Compare(b.Addr()); cmp != 0 {
		return cmp
	}

	return cmp.Compare(a.Bits(), b.Bits())
}
