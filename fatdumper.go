package bart

import (
	"fmt"
	"io"
	"strings"
)

// dump the node to w.
func (n *fatNode[V]) dump(w io.Writer, path stridePath, depth int, is4 bool) {
	bits := depth * strideLen
	indent := strings.Repeat(".", depth)

	// node type with depth and octet path and bits.
	fmt.Fprintf(w, "\n%s[%s] depth:  %d path: [%s] / %d\n",
		indent, n.hasType(), depth, ipStridePath(path, depth, is4), bits)

	if nPfxCount := n.prefixCount(); nPfxCount != 0 {
		allIndices := n.prefixesBitSet.AsSlice(&[256]uint8{})

		// print the baseIndices for this node.
		fmt.Fprintf(w, "%sindexs(#%d): %v\n", indent, nPfxCount, allIndices)

		// print the prefixes for this node
		fmt.Fprintf(w, "%sprefxs(#%d):", indent, nPfxCount)

		for _, idx := range allIndices {
			pfx := cidrFromPath(path, depth, is4, idx)
			fmt.Fprintf(w, " %s", pfx)
		}

		fmt.Fprintln(w)

		// skip values if the payload is the empty struct
		if shouldPrintValues[V]() {

			// print the values for this node
			fmt.Fprintf(w, "%svalues(#%d):", indent, nPfxCount)

			for _, idx := range allIndices {
				fmt.Fprintf(w, " %#v", *n.prefixes[idx])
			}

			fmt.Fprintln(w)
		}
	}

	if n.childCount() != 0 {

		childAddrs := make([]uint8, 0, maxItems)
		leafAddrs := make([]uint8, 0, maxItems)
		fringeAddrs := make([]uint8, 0, maxItems)

		for _, addr := range n.childrenBitSet.AsSlice(&[256]uint8{}) {
			// the node has recursive child nodes or path-compressed leaves
			kidAny := *n.children[addr]

			switch kidAny.(type) {
			case *fatNode[V]:
				childAddrs = append(childAddrs, addr)
				continue
			case *leafNode[V]:
				leafAddrs = append(leafAddrs, addr)
			case *fringeNode[V]:
				fringeAddrs = append(fringeAddrs, addr)

			default:
				panic("logic error, wrong node type")
			}
		}

		// print the children for this node.
		fmt.Fprintf(w, "%soctets(#%d): %v\n", indent, n.childCount(), n.childrenBitSet.AsSlice(&[256]uint8{}))

		if leafCount := len(leafAddrs); leafCount > 0 {
			// print the pathcomp prefixes for this node
			fmt.Fprintf(w, "%sleaves(#%d):", indent, leafCount)

			for _, addr := range leafAddrs {
				k := *n.children[addr]
				pc := k.(*leafNode[V])

				// val is the empty struct, don't print it
				switch {
				case !shouldPrintValues[V]():
					fmt.Fprintf(w, " %s:{%s}", addrFmt(addr, is4), pc.prefix)
				default:
					fmt.Fprintf(w, " %s:{%s, %v}", addrFmt(addr, is4), pc.prefix, pc.value)
				}
			}

			fmt.Fprintln(w)
		}

		if fringeCount := len(fringeAddrs); fringeCount > 0 {
			// print the pathcomp prefixes for this node
			fmt.Fprintf(w, "%sfringe(#%d):", indent, fringeCount)

			for _, addr := range fringeAddrs {
				fringePfx := cidrForFringe(path[:], depth, is4, addr)

				k := *n.children[addr]
				pc := k.(*fringeNode[V])

				// val is the empty struct, don't print it
				switch {
				case !shouldPrintValues[V]():
					fmt.Fprintf(w, " %s:{%s}", addrFmt(addr, is4), fringePfx)
				default:
					fmt.Fprintf(w, " %s:{%s, %v}", addrFmt(addr, is4), fringePfx, pc.value)
				}
			}

			fmt.Fprintln(w)
		}

		if childCount := len(childAddrs); childCount > 0 {
			// print the next child
			fmt.Fprintf(w, "%schilds(#%d):", indent, childCount)

			for _, addr := range childAddrs {
				fmt.Fprintf(w, " %s", addrFmt(addr, is4))
			}

			fmt.Fprintln(w)
		}
	}
}

// hasType returns the nodeType.
func (n *fatNode[V]) hasType() nodeType {
	s := n.nodeStats()

	switch {
	case s.pfxs == 0 && s.childs == 0:
		return nullNode
	case s.nodes == 0:
		return stopNode
	case (s.leaves > 0 || s.fringes > 0) && s.nodes > 0 && s.pfxs == 0:
		return halfNode
	case (s.pfxs > 0 || s.leaves > 0 || s.fringes > 0) && s.nodes > 0:
		return fullNode
	case (s.pfxs == 0 && s.leaves == 0 && s.fringes == 0) && s.nodes > 0:
		return pathNode
	default:
		panic(fmt.Sprintf("UNREACHABLE: pfx: %d, chld: %d, node: %d, leaf: %d, fringe: %d",
			s.pfxs, s.childs, s.nodes, s.leaves, s.fringes))
	}
}

// node statistics for this single node
func (n *fatNode[V]) nodeStats() stats {
	var s stats

	s.pfxs = n.prefixCount()
	s.childs = n.childCount()

	for _, addr := range n.childrenBitSet.AsSlice(&[256]uint8{}) {
		kidAny := *n.children[addr]
		switch kidAny.(type) {
		case *fatNode[V]:
			s.nodes++

		case *leafNode[V]:
			s.leaves++

		case *fringeNode[V]:
			s.fringes++

		default:
			panic("logic error, wrong node type")
		}
	}

	return s
}

// dumpString is just a wrapper for dump.
func (f *Fat[V]) dumpString() string {
	w := new(strings.Builder)
	f.dump(w)

	return w.String()
}

// dump the table structure and all the nodes to w.
func (f *Fat[V]) dump(w io.Writer) {
	if f == nil {
		return
	}

	if f.size4 > 0 {
		stats := f.root4.nodeStatsRec()
		fmt.Fprintln(w)
		fmt.Fprintf(w, "### IPv4: size(%d), nodes(%d), pfxs(%d), leaves(%d), fringes(%d)",
			f.size4, stats.nodes, stats.pfxs, stats.leaves, stats.fringes)

		f.root4.dumpRec(w, stridePath{}, 0, true)
	}

	if f.size6 > 0 {
		stats := f.root6.nodeStatsRec()
		fmt.Fprintln(w)
		fmt.Fprintf(w, "### IPv6: size(%d), nodes(%d), pfxs(%d), leaves(%d), fringes(%d)",
			f.size6, stats.nodes, stats.pfxs, stats.leaves, stats.fringes)

		f.root6.dumpRec(w, stridePath{}, 0, false)
	}
}

// dumpRec, rec-descent the trie.
func (n *fatNode[V]) dumpRec(w io.Writer, path stridePath, depth int, is4 bool) {
	// dump this node
	n.dump(w, path, depth, is4)

	// the node may have childs, rec-descent down
	for _, addr := range n.childrenBitSet.AsSlice(&[256]uint8{}) {
		if kid, ok := (*n.children[addr]).(*fatNode[V]); ok {
			path[depth] = addr
			kid.dumpRec(w, path, depth+1, is4)
		}
	}
}

// nodeStatsRec, calculate the number of pfxs, nodes and leaves under n, rec-descent.
func (n *fatNode[V]) nodeStatsRec() stats {
	var s stats
	if n == nil || n.isEmpty() {
		return s
	}

	s.pfxs = n.prefixCount()
	s.childs = n.childCount()
	s.nodes = 1 // this node
	s.leaves = 0
	s.fringes = 0

	for _, addr := range n.childrenBitSet.AsSlice(&[256]uint8{}) {
		kidAny := *n.children[addr]
		switch kid := kidAny.(type) {
		case *fatNode[V]:
			// rec-descent
			rs := kid.nodeStatsRec()

			s.pfxs += rs.pfxs
			s.childs += rs.childs
			s.nodes += rs.nodes
			s.leaves += rs.leaves
			s.fringes += rs.fringes

		case *leafNode[V]:
			s.leaves++

		case *fringeNode[V]:
			s.fringes++

		default:
			panic("logic error, wrong node type")
		}
	}

	return s
}
