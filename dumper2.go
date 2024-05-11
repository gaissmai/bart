// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ###################################################
// Useful during development or debugging and testing.
// ###################################################

// dumpString2 is just a wrapper for dump.
func (t *Table2[V]) dumpString2() string {
	w := new(strings.Builder)
	if err := t.dump(w); err != nil {
		panic(err)
	}
	return w.String()
}

// dump the table structure and all the nodes to w.
//
//	Output:
//
// IPv4:
//
// [FULL] path: [] bits: +0 depth: 0
// indexs(#2): [266 383]
// prefxs(#2): 10/8 127/8
// values(#2): <nil> <nil>
// childs(#5): 10 127 169 172 192
//
// .[LEAF] path: [10.0] bits: +16 depth: 1
// .indexs(#2): [256 257]
// .prefxs(#2): 0/8 1/8
// .values(#2): <nil> <nil>
//
// .[LEAF] path: [127.0.0] bits: +24 depth: 1
// .indexs(#1): [257]
// .prefxs(#1): 1/8
// .values(#1): <nil>
//
// .[LEAF] path: [169] bits: +8 depth: 1
// .indexs(#1): [510]
// .prefxs(#1): 254/8
// .values(#1): <nil>
//
// .[LEAF] path: [172] bits: +8 depth: 1
// .indexs(#1): [17]
// .prefxs(#1): 16/4
// .values(#1): <nil>
//
// .[FULL] path: [192] bits: +8 depth: 1
// .indexs(#1): [424]
// .prefxs(#1): 168/8
// .values(#1): <nil>
// .childs(#1): 168
//
// ..[LEAF] path: [192.168] bits: +16 depth: 2
// ..indexs(#1): [257]
// ..prefxs(#1): 1/8
// ..values(#1): <nil>
// IPv6:
//
// [FULL] path: [] bits: +0 depth: 0
// indexs(#2): [1 9]
// prefxs(#2): 0x00/0 0x20/3
// values(#2): <nil> <nil>
// childs(#2): 0x20 0xfe
//
// .[LEAF] path: [2001:0d] bits: +24 depth: 1
// .indexs(#1): [440]
// .prefxs(#1): 0xb8/8
// .values(#1): <nil>
//
// .[LEAF] path: [fe] bits: +8 depth: 1
// .indexs(#1): [6]
// .prefxs(#1): 0x80/2
// .values(#1): <nil>
func (t *Table2[V]) dump(w io.Writer) error {
	t.init()

	if _, err := fmt.Fprint(w, "IPv4:\n"); err != nil {
		return err
	}
	t.rootV4.dumpRec2(w, 0)

	if _, err := fmt.Fprint(w, "IPv6:\n"); err != nil {
		return err
	}
	t.rootV6.dumpRec2(w, 0)

	return nil
}

// dumpRec2, rec-descent the trie.
func (n *node2[V]) dumpRec2(w io.Writer, depth int) {
	n.dump2(w, depth)

	for _, child := range n.children {
		child.dumpRec2(w, depth+1)
	}
}

// dump2 the node to w.
func (n *node2[V]) dump2(w io.Writer, depth int) {
	must := func(_ int, err error) {
		if err != nil {
			panic(err)
		}
	}

	bits := n.pathLen() * strideLen
	indent := strings.Repeat(".", depth)

	// node type with depth and octet path and bits.
	must(fmt.Fprintf(w, "\n%s[%s] path: [%s] bits: +%d depth: %d\n",
		indent, n.hasType2(), n.pathAsString(), bits, depth))

	if len(n.prefixes) != 0 {
		indices := n.allStrideIndexes()
		// print the baseIndices for this node.
		must(fmt.Fprintf(w, "%sindexs(#%d): %v\n", indent, len(n.prefixes), indices))

		// print the prefixes for this node
		must(fmt.Fprintf(w, "%sprefxs(#%d):", indent, len(n.prefixes)))

		for _, idx := range indices {
			octet, bits := baseIndexToPrefix(idx)
			must(fmt.Fprintf(w, " %s/%d", octetFmt(octet, n.is4), bits))
		}
		must(fmt.Fprintln(w))

		// print the values for this node
		must(fmt.Fprintf(w, "%svalues(#%d):", indent, len(n.prefixes)))

		for _, val := range n.prefixes {
			must(fmt.Fprintf(w, " %v", val))
		}
		must(fmt.Fprintln(w))
	}

	if len(n.children) != 0 {
		// print the childs for this node
		must(fmt.Fprintf(w, "%schilds(#%d): ", indent, len(n.children)))

		for i := range n.children {
			octet := byte(n.childrenBitset.Select(uint(i)))
			must(fmt.Fprintf(w, "%s ", octetFmt(octet, n.is4)))
		}
		must(fmt.Fprintln(w))
	}
}

// pathAsString, stride path, different formats for IPv4 and IPv6, dotted decimal or hex.
func (n *node2[V]) pathAsString() string {
	buf := new(strings.Builder)

	if n.is4 {
		for i, b := range n.pathAsSlice() {
			if i != 0 {
				buf.WriteString(".")
			}
			buf.WriteString(strconv.Itoa(int(b)))
		}
		return buf.String()
	}

	for i, b := range n.pathAsSlice() {
		if i != 0 && i%2 == 0 {
			buf.WriteString(":")
		}
		buf.WriteString(fmt.Sprintf("%02x", b))
	}
	return buf.String()
}

// hasType2 returns the nodeType.
func (n *node2[V]) hasType2() nodeType {
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
