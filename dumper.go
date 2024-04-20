// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// dumpString is just a wrapper for dump.
func (t *Table[V]) dumpString() string {
	w := new(strings.Builder)
	if err := t.dump(w); err != nil {
		panic(err)
	}
	return w.String()
}

// dump the IPv4 and IPv6 tables to w.
// Useful during development and debugging.
//
//	 Output:
//
//		[FULL] depth:  0 path: [] / 0
//		indexs(#6): 1 66 128 133 266 383
//		prefxs(#6): 0/0 8/6 0/7 10/7 10/8 127/8
//		childs(#3): 10 127 192
//
//		.[IMED] depth:  1 path: [10] / 8
//		.childs(#1): 0
//
//		..[LEAF] depth:  2 path: [10.0] / 16
//		..indexs(#2): 256 257
//		..prefxs(#2): 0/8 1/8
//
//		.[IMED] depth:  1 path: [127] / 8
//		.childs(#1): 0
//
//		..[IMED] depth:  2 path: [127.0] / 16
//		..childs(#1): 0
//
//		...[LEAF] depth:  3 path: [127.0.0] / 24
//		...indexs(#1): 257
//		...prefxs(#1): 1/8
//
// ...
func (t *Table[V]) dump(w io.Writer) error {
	t.init()

	if _, err := fmt.Fprint(w, "IPv4:\n"); err != nil {
		return err
	}
	t.rootV4.dumpRec(w, nil, true)

	if _, err := fmt.Fprint(w, "IPv6:\n"); err != nil {
		return err
	}
	t.rootV6.dumpRec(w, nil, false)

	return nil
}

// dumpRec, rec-descent the trie.
func (n *node[V]) dumpRec(w io.Writer, path []byte, is4 bool) {
	n.dump(w, path, is4)

	for i, child := range n.children.childs {
		octet := n.children.Select(uint(i))
		child.dumpRec(w, append(path, byte(octet)), is4)
	}
}

// dump the node to w.
func (n *node[V]) dump(w io.Writer, path []byte, is4 bool) {
	must := func(_ int, err error) {
		if err != nil {
			panic(err)
		}
	}

	depth := len(path)
	bits := depth * strideLen
	indent := strings.Repeat(".", depth)

	// node type with depth and octet path and bits.
	must(fmt.Fprintf(w, "\n%s[%s] depth:  %d path: [%v] / %d\n",
		indent, n.hasType(), depth, ancestors(path, is4), bits))

	if len(n.prefixes.values) != 0 {
		indices := n.prefixes.allIndexes()
		// print the baseIndices for this node.
		must(fmt.Fprintf(w, "%sindexs(#%d): %v\n", indent, len(n.prefixes.values), indices))

		// print the prefixes for this node
		must(fmt.Fprintf(w, "%sprefxs(#%d): ", indent, len(n.prefixes.values)))

		for _, idx := range indices {
			octet, bits := baseIndexToPrefix(idx)
			must(fmt.Fprintf(w, "%s/%d ", octetFmt(octet, is4), bits))
		}
		must(fmt.Fprintln(w))
	}

	if len(n.children.childs) != 0 {
		// print the childs for this node
		must(fmt.Fprintf(w, "%schilds(#%d): ", indent, len(n.children.childs)))

		for i := range n.children.childs {
			octet := n.children.Select(uint(i))
			must(fmt.Fprintf(w, "%s ", octetFmt(octet, is4)))
		}
		must(fmt.Fprintln(w))
	}
}

// octetFmt, different format strings for IPv4 and IPv6, decimal versus hex.
func octetFmt(octet uint, is4 bool) string {
	if is4 {
		return fmt.Sprintf("%d", octet)
	}
	return fmt.Sprintf("0x%02x", octet)
}

// IP stride path, different formats for IPv4 and IPv6, dotted decimal or hex.
func ancestors(path []byte, is4 bool) string {
	buf := new(strings.Builder)

	if is4 {
		for i, b := range path {
			if i != 0 {
				buf.WriteString(".")
			}
			buf.WriteString(strconv.Itoa(int(b)))
		}
		return buf.String()
	}

	for i, b := range path {
		if i != 0 && i%2 == 0 {
			buf.WriteString(":")
		}
		buf.WriteString(fmt.Sprintf("%02x", b))
	}
	return buf.String()
}

// String implements Stringer for nodeType.
func (nt nodeType) String() string {
	switch nt {
	case nullNode:
		return "ROOT"
	case fullNode:
		return "FULL"
	case leafNode:
		return "LEAF"
	case intermediateNode:
		return "IMED"
	}
	panic("unreachable")
}

// hasType returns the nodeType.
func (n *node[V]) hasType() nodeType {
	lenPefixes := len(n.prefixes.values)
	lenChilds := len(n.children.childs)

	if lenPefixes == 0 && lenChilds != 0 {
		return intermediateNode
	}

	if lenPefixes == 0 && lenChilds == 0 {
		return nullNode
	}

	if lenPefixes != 0 && lenChilds == 0 {
		return leafNode
	}

	if lenPefixes != 0 && lenChilds != 0 {
		return fullNode
	}
	panic("unreachable")
}
