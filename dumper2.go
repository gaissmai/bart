// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"fmt"
	"io"
	"strings"
)

// ###################################################
// Useful during development or debugging and testing.
// ###################################################

// dumpString is just a wrapper for dump.
func (t *Table2[V]) dumpString() string {
	w := new(strings.Builder)
	if err := t.dump(w); err != nil {
		panic(err)
	}
	return w.String()
}

// dump the table structure and all the nodes to w.
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
func (t *Table2[V]) dump(w io.Writer) error {
	t.init()

	if _, err := fmt.Fprint(w, "IPv4:\n"); err != nil {
		return err
	}
	t.rootV4.dumpRec(w, 0, true)

	if _, err := fmt.Fprint(w, "IPv6:\n"); err != nil {
		return err
	}
	t.rootV6.dumpRec(w, 0, false)

	return nil
}

// dumpRec, rec-descent the trie.
func (n *node2[V]) dumpRec(w io.Writer, depth int, is4 bool) {
	n.dump(w, depth, is4)

	for _, child := range n.children {
		child.dumpRec(w, depth+1, is4)
	}
}

// dump the node to w.
func (n *node2[V]) dump(w io.Writer, depth int, is4 bool) {
	must := func(_ int, err error) {
		if err != nil {
			panic(err)
		}
	}

	indent := strings.Repeat(".", depth)

	// node type with depth and octet path and bits.
	must(fmt.Fprintf(w, "\n%s[%s] depth:  %d path: [%v]\n",
		indent, n.hasType(), depth, ancestors(n.path, is4)))

	if len(n.prefixes) != 0 {
		indices := n.allStrideIndexes()
		// print the baseIndices for this node.
		must(fmt.Fprintf(w, "%sindexs(#%d): %v\n", indent, len(n.prefixes), indices))

		// print the prefixes for this node
		must(fmt.Fprintf(w, "%sprefxs(#%d): ", indent, len(n.prefixes)))

		for _, idx := range indices {
			octet, bits := baseIndexToPrefix(idx)
			must(fmt.Fprintf(w, "%s/%d ", octetFmt(octet, is4), bits))
		}
		must(fmt.Fprintln(w))
	}

	if len(n.children) != 0 {
		// print the childs for this node
		must(fmt.Fprintf(w, "%schilds(#%d): ", indent, len(n.children)))

		for i := range n.children {
			octet := byte(n.childrenBitset.Select(uint(i)))
			must(fmt.Fprintf(w, "%s ", octetFmt(octet, is4)))
		}
		must(fmt.Fprintln(w))
	}
}

// hasType returns the nodeType.
func (n *node2[V]) hasType() nodeType {
	lenPefixes := len(n.prefixes)
	lenChilds := len(n.children)

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
