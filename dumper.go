// Copyright (c) 2024 Karl Gaissmaier
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

// nodeDumper is a generic interface that abstracts tree node operations
// for dumping and traversal.
//
// It provides a unified API for accessing both regular nodes and fat nodes
// in the routing table structures. The interface supports prefix and child
// management operations needed for tree traversal, statistics collection,
// and structural dumping.
//
// Type parameter V represents the value type stored at prefixes in the tree.
//
// The interface combines read operations (get*, count*, isEmpty)
// to support inspection during tree operations. This abstraction enables the
// dumper functionality to work uniformly across different node implementations
// (node[V] and fatNode[V]).
type nodeDumper[V any] interface {
	isEmpty() bool

	childCount() int
	prefixCount() int

	getChild(uint8) (any, bool)
	getPrefix(idx uint8) (V, bool)

	mustGetChild(uint8) any
	mustGetPrefix(idx uint8) V

	getChildAddrs() []uint8
	getIndices() []uint8
}

// dumpRec recursively descends the trie rooted at n and writes a human-readable
// representation of each visited node to w.
//
// It returns immediately if n is nil or empty. For each visited internal node
// it calls dump to write the node's representation, then iterates its child
// addresses and recurses into children that implement nodeDumper[V] (internal
// subnodes). The path slice and depth together represent the byte-wise path
// from the root to the current node; depth is incremented for each recursion.
// dumpRec recursively writes a human-readable dump of the trie subtree rooted at n to w.
// It returns immediately if n is nil or empty. The function dumps the current node and
// then descends depth-first into child entries obtained from n.getChildAddrs(), recursing
// only into children that themselves implement nodeDumper (leaves and fringes are not
// traversed). The provided path and depth are used to build per-node prefixes; the is4
// flag selects IPv4 vs IPv6 formatting.
func dumpRec[V any](n nodeDumper[V], w io.Writer, path stridePath, depth int, is4 bool) {
	if n == nil || n.isEmpty() {
		return
	}

	// dump this node
	dump(n, w, path, depth, is4)

	// node may have children, rec-descent down
	for _, addr := range n.getChildAddrs() {
		anyKid := n.mustGetChild(addr)
		if kid, ok := anyKid.(nodeDumper[V]); ok {
			path[depth] = addr
			dumpRec(kid, w, path, depth+1, is4)
		}
	}
}

// dump writes a human-readable representation of the node `n` to `w`.
// It prints the node type, depth, formatted path (IPv4 vs IPv6 controlled by `is4`),
// and bit count, followed by any stored prefixes (and their values when applicable),
// the set of child octets, and any path-compressed leaves or fringe entries.
// dump writes a human-readable representation of the node n to w.
// 
// It prints the node's type, depth, path (derived from `path` and `depth`), bit count,
// any stored prefixes (and their values when the generic payload type is non-empty),
// and a breakdown of children into internal nodes, path-compressed leaves, and fringes.
// The `path` and `depth` parameters determine how prefixes and fringe CIDRs are rendered;
// `is4` controls IPv4 vs IPv6 formatting. The function writes directly to w and may panic
// if a child has an unexpected concrete type.
func dump[V any](n nodeDumper[V], w io.Writer, path stridePath, depth int, is4 bool) {
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
		if shouldPrintValues[V]() {

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
		allChildAddrs := n.getChildAddrs()

		childAddrs := make([]uint8, 0, maxItems)
		leafAddrs := make([]uint8, 0, maxItems)
		fringeAddrs := make([]uint8, 0, maxItems)

		// the node has recursive child nodes or path-compressed leaves
		for _, addr := range allChildAddrs {
			anyKid := n.mustGetChild(addr)
			switch anyKid.(type) {
			case nodeDumper[V]:
				childAddrs = append(childAddrs, addr)
				continue

			case *fringeNode[V]:
				fringeAddrs = append(fringeAddrs, addr)

			case *leafNode[V]:
				leafAddrs = append(leafAddrs, addr)

			default:
				panic("logic error, wrong node type")
			}
		}

		// print the children for this node.
		fmt.Fprintf(w, "%soctets(#%d): %v\n", indent, n.childCount(), allChildAddrs)

		if leafCount := len(leafAddrs); leafCount > 0 {
			// print the pathcomp prefixes for this node
			fmt.Fprintf(w, "%sleaves(#%d):", indent, leafCount)

			for _, addr := range leafAddrs {
				k := n.mustGetChild(addr)
				pc := k.(*leafNode[V])

				// Lite: val is the empty struct, don't print it
				switch any(pc.value).(type) {
				case struct{}:
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

				k := n.mustGetChild(addr)
				pc := k.(*fringeNode[V])

				// Lite: val is the empty struct, don't print it
				switch any(pc.value).(type) {
				case struct{}:
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
// Panics if the node's immediate statistics do not match any expected case.
func hasType[V any](n nodeDumper[V]) nodeType {
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
// nodeStats computes immediate statistics for node n: counts of prefixes (pfxs),
// direct children (childs), child nodes that are internal nodes (nodes), fringe
// children (fringes), and leaf children (leaves).
//
// The function inspects each child address returned by n.getChildAddrs() and
// classifies the concrete child type. It panics if any child has an unexpected
// concrete type.
func nodeStats[V any](n nodeDumper[V]) stats {
	var s stats

	s.pfxs = n.prefixCount()
	s.childs = n.childCount()

	for _, addr := range n.getChildAddrs() {
		switch n.mustGetChild(addr).(type) {
		case nodeDumper[V]:
			s.nodes++

		case *fringeNode[V]:
			s.fringes++

		case *leafNode[V]:
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
// nodeStatsRec recursively computes aggregated immediate statistics for the subtree
// rooted at n and returns a stats struct summarizing that subtree.
//
// If n is nil or n.isEmpty(), a zero-valued stats is returned. The returned stats
// includes the current node (nodes starts at 1 for a non-empty n), the total
// number of prefixes (pfxs), direct child slots (childs), and counts of leaf and
// fringe children found anywhere in the subtree. For each child address returned
// by n.getChildAddrs(), the function inspects the concrete child type:
//  - nodeDumper[V]: recurses into that child and accumulates its stats,
//  - *fringeNode[V]: increments the fringes count,
//  - *leafNode[V]: increments the leaves count.
// The function panics if a child has an unexpected concrete type.
func nodeStatsRec[V any](n nodeDumper[V]) stats {
	var s stats
	if n == nil || n.isEmpty() {
		return s
	}

	s.pfxs = n.prefixCount()
	s.childs = n.childCount()
	s.nodes = 1 // this node
	s.leaves = 0
	s.fringes = 0

	for _, addr := range n.getChildAddrs() {
		switch kid := n.mustGetChild(addr).(type) {
		case nodeDumper[V]:
			// rec-descent
			rs := nodeStatsRec[V](kid)

			s.pfxs += rs.pfxs
			s.childs += rs.childs
			s.nodes += rs.nodes
			s.leaves += rs.leaves
			s.fringes += rs.fringes

		case *fringeNode[V]:
			s.fringes++

		case *leafNode[V]:
			s.leaves++

		default:
			panic("logic error, wrong node type")
		}
	}

	return s
}
