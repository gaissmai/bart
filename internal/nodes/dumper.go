// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package nodes

import (
	"fmt"
	"strconv"
	"strings"
)

// StatsT, only used for dump, tests and benchmarks
type StatsT struct {
	Pfxs    int
	Childs  int
	Nodes   int
	Leaves  int
	Fringes int
}

type nodeType byte

const (
	nullNode nodeType = iota // empty node
	fullNode                 // prefixes and children or path-compressed prefixes
	halfNode                 // no prefixes, only children and path-compressed prefixes
	pathNode                 // only children, no prefix nor path-compressed prefixes
	stopNode                 // no children, only prefixes or path-compressed prefixes
)

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
func ipStridePath(path StridePath, depth int, is4 bool) string {
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

/*

// Stats returns immediate statistics for n: counts of prefixes and children,
// and a classification of each child into nodes, leaves, or fringes.
// It inspects only the direct children of n (not the whole subtree).
// Panics if a child has an unexpected concrete type.
func Stats[V any](n NodeReader[V]) StatsT {
	var s StatsT

	s.Pfxs = n.PrefixCount()
	s.Childs = n.ChildCount()

	for _, child := range n.AllChildren() {
		switch child.(type) {
		case NodeReader[V]:
			s.Nodes++

		case *FringeNode[V]:
			s.Fringes++

		case *LeafNode[V]:
			s.Leaves++

		default:
			panic("logic error, wrong node type")
		}
	}

	return s
}

// StatsRec returns aggregated statistics for the subtree rooted at n.
//
// It walks the node tree recursively and sums immediate counts (prefixes and
// child slots) plus the number of nodes, leaves, and fringe nodes in the
// subtree. If n is nil or empty, a zeroed stats is returned. The returned
// stats.nodes includes the current node. The function will panic if a child
// has an unexpected concrete type.
func StatsRec[V any](n NodeReader[V]) StatsT {
	var s StatsT
	if n == nil || n.IsEmpty() {
		return s
	}

	s.Pfxs = n.PrefixCount()
	s.Childs = n.ChildCount()
	s.Nodes = 1 // this node
	s.Leaves = 0
	s.Fringes = 0

	for _, child := range n.AllChildren() {
		switch kid := child.(type) {
		case NodeReader[V]:
			// rec-descent
			rs := StatsRec[V](kid)

			s.Pfxs += rs.Pfxs
			s.Childs += rs.Childs
			s.Nodes += rs.Nodes
			s.Leaves += rs.Leaves
			s.Fringes += rs.Fringes

		case *FringeNode[V]:
			s.Fringes++

		case *LeafNode[V]:
			s.Leaves++

		default:
			panic("logic error, wrong node type")
		}
	}

	return s
}

// DumpRec recursively descends the trie rooted at n and writes a human-readable
// representation of each visited node to w.
//
// It returns immediately if n is nil or empty. For each visited internal node
// it calls dump to write the node's representation, then iterates its child
// addresses and recurses into children that implement nodeDumper[V] (internal
// subnodes). The path slice and depth together represent the byte-wise path
// from the root to the current node; depth is incremented for each recursion.
// The is4 flag controls IPv4/IPv6 formatting used by dump.
func DumpRec[V any](n NodeReader[V], w io.Writer, path StridePath, depth int, is4 bool, printVals bool) {
	if n == nil || n.IsEmpty() {
		return
	}

	// dump this node
	Dump(n, w, path, depth, is4, printVals)

	// node may have children, rec-descent down
	for addr, child := range n.AllChildren() {
		if kid, ok := child.(NodeReader[V]); ok {
			path[depth] = addr
			DumpRec(kid, w, path, depth+1, is4, printVals)
		}
	}
}

// Dump writes a human-readable representation of the node `n` to `w`.
// It prints the node type, depth, formatted path (IPv4 vs IPv6 controlled by `is4`),
// and bit count, followed by any stored prefixes (and their values when applicable),
// the set of child octets, and any path-compressed leaves or fringe entries.
// `path` and `depth` determine how prefixes and fringe CIDRs are rendered.
func Dump[V any](n NodeReader[V], w io.Writer, path StridePath, depth int, is4 bool, printVals bool) {
	bits := depth * strideLen
	indent := strings.Repeat(".", depth)

	// node type with depth and octet path and bits.
	fmt.Fprintf(w, "\n%s[%s] depth:  %d path: [%s] / %d\n",
		indent, hasType(n), depth, ipStridePath(path, depth, is4), bits)

	if nPfxCount := n.PrefixCount(); nPfxCount != 0 {
		var buf [256]uint8
		allIndices := n.GetIndices(&buf)

		// print the baseIndices for this node.
		fmt.Fprintf(w, "%sindexs(#%d): %v\n", indent, nPfxCount, allIndices)

		// print the prefixes for this node
		fmt.Fprintf(w, "%sprefxs(#%d):", indent, nPfxCount)

		for _, idx := range allIndices {
			pfx := CidrFromPath(path, depth, is4, idx)
			fmt.Fprintf(w, " %s", pfx)
		}

		fmt.Fprintln(w)

		// skip values, maybe the payload is the empty struct
		if printVals {

			// print the values for this node
			fmt.Fprintf(w, "%svalues(#%d):", indent, nPfxCount)

			for _, idx := range allIndices {
				val := n.MustGetPrefix(idx)
				fmt.Fprintf(w, " %#v", val)
			}

			fmt.Fprintln(w)
		}
	}

	if n.ChildCount() != 0 {
		allAddrs := make([]uint8, 0, MaxItems)
		childAddrs := make([]uint8, 0, MaxItems)
		leafAddrs := make([]uint8, 0, MaxItems)
		fringeAddrs := make([]uint8, 0, MaxItems)

		// the node has recursive child nodes or path-compressed leaves
		for addr, child := range n.AllChildren() {
			allAddrs = append(allAddrs, addr)

			switch child.(type) {
			case NodeReader[V]:
				childAddrs = append(childAddrs, addr)
				continue

			case *FringeNode[V]:
				fringeAddrs = append(fringeAddrs, addr)

			case *LeafNode[V]:
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
				kid := n.MustGetChild(addr).(*LeafNode[V])
				if printVals {
					fmt.Fprintf(w, " %s:{%s, %v}", addrFmt(addr, is4), kid.Prefix, kid.Value)
				} else {
					fmt.Fprintf(w, " %s:{%s}", addrFmt(addr, is4), kid.Prefix)
				}
			}

			fmt.Fprintln(w)
		}

		if fringeCount := len(fringeAddrs); fringeCount > 0 {
			// print the pathcomp prefixes for this node
			fmt.Fprintf(w, "%sfringe(#%d):", indent, fringeCount)

			for _, addr := range fringeAddrs {
				fringePfx := CidrForFringe(path[:], depth, is4, addr)

				kid := n.MustGetChild(addr).(*FringeNode[V])
				if printVals {
					fmt.Fprintf(w, " %s:{%s, %v}", addrFmt(addr, is4), fringePfx, kid.Value)
				} else {
					fmt.Fprintf(w, " %s:{%s}", addrFmt(addr, is4), fringePfx)
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
func hasType[V any](n NodeReader[V]) nodeType {
	s := Stats[V](n)

	// the order is important
	switch {
	case s.Pfxs == 0 && s.Childs == 0:
		return nullNode
	case s.Nodes == 0:
		return stopNode
	case (s.Leaves > 0 || s.Fringes > 0) && s.Nodes > 0 && s.Pfxs == 0:
		return halfNode
	case (s.Pfxs > 0 || s.Leaves > 0 || s.Fringes > 0) && s.Nodes > 0:
		return fullNode
	case (s.Pfxs == 0 && s.Leaves == 0 && s.Fringes == 0) && s.Nodes > 0:
		return pathNode
	default:
		panic(fmt.Sprintf("UNREACHABLE: pfx: %d, chld: %d, node: %d, leaf: %d, fringe: %d",
			s.Pfxs, s.Childs, s.Nodes, s.Leaves, s.Fringes))
	}
}

*/
