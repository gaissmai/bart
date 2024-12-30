// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"fmt"
	"io"
	"strings"
)

// ##################################################
//  useful during development, debugging and testing
// ##################################################

// dumpString is just a wrapper for dump.
func (t *Table2[V]) dumpString() string {
	w := new(strings.Builder)
	t.dump(w)

	return w.String()
}

// dump the table structure and all the nodes to w.
func (t *Table2[V]) dump(w io.Writer) {
	if t == nil {
		return
	}

	if t.Size4() > 0 {
		fmt.Fprintf(w, "### IPv4: size(%d)", t.Size4())
		t.root4.dumpRec(w, zeroPath, 0, true)
	}

	if t.Size6() > 0 {
		fmt.Fprintf(w, "### IPv6: size(%d)", t.Size6())
		t.root6.dumpRec(w, zeroPath, 0, false)
	}
}

// dumpRec, rec-descent the trie.
func (n *node2[V]) dumpRec(w io.Writer, path [16]byte, depth int, is4 bool) {
	n.dump(w, path, depth, is4)

	// no heap allocs
	allChildAddrs := n.children.AsSlice(make([]uint, 0, maxNodeChildren))

	// the node may have childs, the rec-descent monster starts
	for i, addr := range allChildAddrs {
		octet := byte(addr)
		path[depth] = octet
		switch child := n.children.Items[i].(type) {
		case *leaf[V]:
			continue
		case *node2[V]:
			child.dumpRec(w, path, depth+1, is4)
		}
	}
}

// dump the node to w.
func (n *node2[V]) dump(w io.Writer, path [16]byte, depth int, is4 bool) {
	bits := depth * strideLen
	indent := strings.Repeat(".", depth)

	// node type with depth and octet path and bits.
	fmt.Fprintf(w, "\n%s[%s] depth:  %d path: [%s] / %d\n",
		indent, n.hasType(), depth, ipStridePath(path, depth, is4), bits)

	if nPfxCount := n.prefixes.Len(); nPfxCount != 0 {
		// no heap allocs
		allIndices := n.prefixes.AsSlice(make([]uint, 0, maxNodePrefixes))

		// print the baseIndices for this node.
		fmt.Fprintf(w, "%sindexs(#%d): %v\n", indent, nPfxCount, allIndices)

		// print the prefixes for this node
		fmt.Fprintf(w, "%sprefxs(#%d):", indent, nPfxCount)

		for _, idx := range allIndices {
			octet, pfxLen := idxToPfx(idx)
			fmt.Fprintf(w, " %s/%d", octetFmt(octet, is4), pfxLen)
		}

		fmt.Fprintln(w)

		// print the values for this node
		fmt.Fprintf(w, "%svalues(#%d):", indent, nPfxCount)

		for _, val := range n.prefixes.Items {
			fmt.Fprintf(w, " %v", val)
		}

		fmt.Fprintln(w)
	}

	if n.children.Len() != 0 {

		nodeAddrs := make([]uint, 0, maxNodeChildren)
		leafAddrs := make([]uint, 0, maxNodeChildren)

		// the node has recursive child nodes or path-compressed leaves
		for i, addr := range n.children.AsSlice(make([]uint, 0, maxNodeChildren)) {
			switch n.children.Items[i].(type) {
			case *node2[V]:
				nodeAddrs = append(nodeAddrs, addr)
				continue
			case *leaf[V]:
				leafAddrs = append(leafAddrs, addr)
			}
		}

		if nodeCount := len(nodeAddrs); nodeCount > 0 {
			// print the childs for this node
			fmt.Fprintf(w, "%schilds(#%d):", indent, nodeCount)

			for _, addr := range nodeAddrs {
				octet := byte(addr)
				fmt.Fprintf(w, " %s", octetFmt(octet, is4))
			}

			fmt.Fprintln(w)
		}

		if leafCount := len(leafAddrs); leafCount > 0 {
			// print the pathcomp prefixes for this node
			fmt.Fprintf(w, "%sleaves(#%d):", indent, leafCount)

			for _, addr := range leafAddrs {
				octet := byte(addr)
				k := n.children.MustGet(addr)
				pc := k.(*leaf[V])

				fmt.Fprintf(w, " %s:{%s, %v}", octetFmt(octet, is4), pc.prefix, pc.value)
			}
			fmt.Fprintln(w)
		}
	}
}

// hasType returns the nodeType.
func (n *node2[V]) hasType() nodeType {
	prefixCount := n.prefixes.Len()
	childCount := n.children.Len()

	switch {
	case prefixCount == 0 && childCount == 0:
		return nullNode
	case prefixCount != 0 && childCount != 0:
		return fullNode
	case prefixCount == 0 && childCount != 0:
		return intermediateNode
	case prefixCount == 0 && childCount != 0:
		return intermediatePCNode
	case childCount == 0:
		return leafNode
	default:
		panic(fmt.Sprintf("UNREACHABLE: pfx: %d, chld: %d", prefixCount, childCount))
	}
}
