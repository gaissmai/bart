// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

type nodeType byte

const (
	nullNode nodeType = iota // empty node
	fullNode                 // prefixes and children or path-compressed prefixes
	halfNode                 // no prefixes, only children and path-compressed prefixes
	pathNode                 // only children, no prefix nor path-compressed prefixes
	stopNode                 // no children, only prefixes or path-compressed prefixes
)

// dumpRec recursively descends the trie rooted at n and writes a human-readable
// representation of each visited node to w.
//
// It returns immediately if n is nil or empty. For each visited internal node
// it calls dump to write the node's representation, then iterates its child
// addresses and recurses into children that implement nodeDumper[V] (internal
// subnodes). The path slice and depth together represent the byte-wise path
// from the root to the current node; depth is incremented for each recursion.
// The is4 flag controls IPv4/IPv6 formatting used by dump.
func dumpRec[V any](n nodeReader[V], w io.Writer, path stridePath, depth int, is4 bool) {
	if n == nil || n.isEmpty() {
		return
	}

	// dump this node
	dump(n, w, path, depth, is4)

	// node may have children, rec-descent down
	for addr, child := range n.allChildren() {
		if kid, ok := child.(nodeReader[V]); ok {
			path[depth] = addr
			dumpRec(kid, w, path, depth+1, is4)
		}
	}
}

// dump writes a human-readable representation of the node `n` to `w`.
// It prints the node type, depth, formatted path (IPv4 vs IPv6 controlled by `is4`),
// and bit count, followed by any stored prefixes (and their values when applicable),
// the set of child octets, and any path-compressed leaves or fringe entries.
// `path` and `depth` determine how prefixes and fringe CIDRs are rendered.
func dump[V any](n nodeReader[V], w io.Writer, path stridePath, depth int, is4 bool) {
	bits := depth * strideLen
	indent := strings.Repeat(".", depth)

	// node type with depth and octet path and bits.
	fmt.Fprintf(w, "\n%s[%s] depth:  %d path: [%s] / %d\n",
		indent, hasType(n), depth, ipStridePath(path, depth, is4), bits)

	if nPfxCount := n.prefixCount(); nPfxCount != 0 {
		// no heap allocs
		allIndices := n.getIndices()

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
		if shouldPrintValues[V](n) {

			// print the values for this node
			fmt.Fprintf(w, "%svalues(#%d):", indent, nPfxCount)

			for _, idx := range allIndices {
				val := n.mustGetPrefix(idx)
				fmt.Fprintf(w, " %#v", val)
			}

			fmt.Fprintln(w)
		}
	}

	if n.childCount() != 0 {
		allAddrs := make([]uint8, 0, maxItems)
		childAddrs := make([]uint8, 0, maxItems)
		leafAddrs := make([]uint8, 0, maxItems)
		fringeAddrs := make([]uint8, 0, maxItems)

		// the node has recursive child nodes or path-compressed leaves
		for addr, child := range n.allChildren() {
			allAddrs = append(allAddrs, addr)

			switch child.(type) {
			case nodeReader[V]:
				childAddrs = append(childAddrs, addr)
				continue

			case *fringeNode[V], *liteFringeNode:
				fringeAddrs = append(fringeAddrs, addr)

			case *leafNode[V], *liteLeafNode:
				leafAddrs = append(leafAddrs, addr)

			default:
				panic("logic error, wrong node type")
			}
		}

		// print the children for this node.
		fmt.Fprintf(w, "%soctets(#%d): %v\n", indent, len(allAddrs), allAddrs)

		if leafCount := len(leafAddrs); leafCount > 0 {
			// print the pathcomp prefixes for this node
			fmt.Fprintf(w, "%sleaves(#%d):", indent, leafCount)

			for _, addr := range leafAddrs {
				anyKid := n.mustGetChild(addr)
				switch kid := anyKid.(type) {
				case *leafNode[V]:
					if shouldPrintValues[V](n) {
						fmt.Fprintf(w, " %s:{%s, %v}", addrFmt(addr, is4), kid.prefix, kid.value)
					} else {
						fmt.Fprintf(w, " %s:{%s}", addrFmt(addr, is4), kid.prefix)
					}
				case *liteLeafNode:
					fmt.Fprintf(w, " %s:{%s}", addrFmt(addr, is4), kid.prefix)
				default:
				}
			}

			fmt.Fprintln(w)
		}

		if fringeCount := len(fringeAddrs); fringeCount > 0 {
			// print the pathcomp prefixes for this node
			fmt.Fprintf(w, "%sfringe(#%d):", indent, fringeCount)

			for _, addr := range fringeAddrs {
				fringePfx := cidrForFringe(path[:], depth, is4, addr)

				anyKid := n.mustGetChild(addr)
				switch kid := anyKid.(type) {
				case *liteFringeNode:
					fmt.Fprintf(w, " %s:{%s}", addrFmt(addr, is4), fringePfx)
				case *fringeNode[V]:
					if shouldPrintValues[V](n) {
						fmt.Fprintf(w, " %s:{%s, %v}", addrFmt(addr, is4), fringePfx, kid.value)
					} else {
						fmt.Fprintf(w, " %s:{%s}", addrFmt(addr, is4), fringePfx)
					}
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

// hasType classifies the given node into one of the nodeType values.
//
// It inspects immediate statistics (prefix count, child count, node, leaf and
// fringe counts) for the node and returns:
//   - nullNode: no prefixes and no children
//   - stopNode: has children but no subnodes (nodes == 0)
//   - halfNode: contains at least one leaf or fringe and also has subnodes, but
//     no prefixes
//   - fullNode: has prefixes or leaves/fringes and also has subnodes
//   - pathNode: has subnodes only (no prefixes, leaves, or fringes)
//
// The order of these checks is significant to ensure the correct classification.
func hasType[V any](n nodeReader[V]) nodeType {
	s := nodeStats[V](n)

	// the order is important
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

// addrFmt, different format strings for IPv4 and IPv6, decimal versus hex.
func addrFmt(addr byte, is4 bool) string {
	if is4 {
		return fmt.Sprintf("%d", addr)
	}

	return fmt.Sprintf("0x%02x", addr)
}

// ip stride path, different formats for IPv4 and IPv6, dotted decimal or hex.
//
//	127.0.0
//	2001:0d
func ipStridePath(path stridePath, depth int, is4 bool) string {
	buf := new(strings.Builder)

	if is4 {
		for i, b := range path[:depth] {
			if i != 0 {
				buf.WriteString(".")
			}

			buf.WriteString(strconv.Itoa(int(b)))
		}

		return buf.String()
	}

	for i, b := range path[:depth] {
		if i != 0 && i%2 == 0 {
			buf.WriteString(":")
		}

		fmt.Fprintf(buf, "%02x", b)
	}

	return buf.String()
}

// String implements Stringer for nodeType.
func (nt nodeType) String() string {
	switch nt {
	case nullNode:
		return "NULL"
	case fullNode:
		return "FULL"
	case halfNode:
		return "HALF"
	case pathNode:
		return "PATH"
	case stopNode:
		return "STOP"
	default:
		return "unreachable"
	}
}

// stats, only used for dump, tests and benchmarks
type stats struct {
	pfxs    int
	childs  int
	nodes   int
	leaves  int
	fringes int
}

// nodeStats returns immediate statistics for n: counts of prefixes and children,
// and a classification of each child into nodes, leaves, or fringes.
// It inspects only the direct children of n (not the whole subtree).
// Panics if a child has an unexpected concrete type.
func nodeStats[V any](n nodeReader[V]) stats {
	var s stats

	s.pfxs = n.prefixCount()
	s.childs = n.childCount()

	for _, child := range n.allChildren() {
		switch child.(type) {
		case nodeReader[V]:
			s.nodes++

		case *fringeNode[V], *liteFringeNode:
			s.fringes++

		case *leafNode[V], *liteLeafNode:
			s.leaves++

		default:
			panic("logic error, wrong node type")
		}
	}

	return s
}

// nodeStatsRec returns aggregated statistics for the subtree rooted at n.
//
// It walks the node tree recursively and sums immediate counts (prefixes and
// child slots) plus the number of nodes, leaves, and fringe nodes in the
// subtree. If n is nil or empty, a zeroed stats is returned. The returned
// stats.nodes includes the current node. The function will panic if a child
// has an unexpected concrete type.
func nodeStatsRec[V any](n nodeReader[V]) stats {
	var s stats
	if n == nil || n.isEmpty() {
		return s
	}

	s.pfxs = n.prefixCount()
	s.childs = n.childCount()
	s.nodes = 1 // this node
	s.leaves = 0
	s.fringes = 0

	for _, child := range n.allChildren() {
		switch kid := child.(type) {
		case nodeReader[V]:
			// rec-descent
			rs := nodeStatsRec[V](kid)

			s.pfxs += rs.pfxs
			s.childs += rs.childs
			s.nodes += rs.nodes
			s.leaves += rs.leaves
			s.fringes += rs.fringes

		case *fringeNode[V], *liteFringeNode:
			s.fringes++

		case *leafNode[V], *liteLeafNode:
			s.leaves++

		default:
			panic("logic error, wrong node type")
		}
	}

	return s
}
