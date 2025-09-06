package bart

import (
	"fmt"
	"io"
	"net/netip"
	"strconv"
	"strings"

	"github.com/gaissmai/bart/internal/art"
)

type nodeType byte

const (
	nullNode nodeType = iota // empty node
	fullNode                 // both prefixes and children
	imedNode                 // intermediate, no prefixes
	stopNode                 // only prefixes
)

// cidrFromIdx, helper function,
func (n *node[V]) cidrFromIdx(idx uint8) netip.Prefix {
	is4 := n.basePath.Addr().Is4()
	path := n.basePath.Addr().AsSlice()
	depth := n.basePath.Bits() >> 3

	octet, pfxLen := art.IdxToPfx(idx)

	// set masked byte in path at depth
	if depth < len(path) {
		path[depth] = octet

		// zero/mask the bytes after prefix bits
		clear(path[depth+1:])
	}

	// make ip addr from octets
	var ip netip.Addr
	if is4 {
		ip = netip.AddrFrom4([4]byte(path[:4]))
	} else {
		ip = netip.AddrFrom16([16]byte(path))
	}

	// calc bits with pathLen and pfxLen
	bits := depth<<3 + int(pfxLen)

	// return a normalized prefix from ip/bits
	return netip.PrefixFrom(ip, bits)
}

// addrFmt, different format strings for IPv4 and IPv6, decimal versus hex.
func addrFmt(addr byte, is4 bool) string {
	if is4 {
		return fmt.Sprintf("%d", addr)
	}
	return fmt.Sprintf("0x%02x", addr)
}

// fmtStridePath, different formats for IPv4 and IPv6, dotted decimal or hex.
//
//	127.0.0
//	2001:0d
func (n *node[V]) fmtStridePath(octetPath []byte, depth int) string {
	buf := new(strings.Builder)
	is4 := n.basePath.Addr().Is4()

	buf.WriteString("●")
	for _, b := range octetPath[:depth] {
		buf.WriteString("➔")

		if is4 {
			buf.WriteString(strconv.Itoa(int(b)))
		} else {
			fmt.Fprintf(buf, "0x%02x", b)
		}
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
	case imedNode:
		return "IMED"
	case stopNode:
		return "STOP"
	default:
		return "unreachable"
	}
}

// stats, only used for dump, tests and benchmarks
type stats struct {
	nodes  int
	pfxs   int
	childs int
}

// dump the node to w.
func (n *node[V]) dump(w io.Writer, octetPath []byte, depth int) {
	var zero V

	is4 := n.basePath.Addr().Is4()
	pcLevel := n.basePath.Bits() >> 3
	indent := strings.Repeat(".", depth)

	// node type with depth and octet path and bits.
	fmt.Fprintf(w, "\n%s[%s] depth(%d), octetPath(%s), pcLevel(%d), basePath(%s)\n",
		indent, n.hasType(), depth, n.fmtStridePath(octetPath, depth), pcLevel, n.basePath)

	allIndices := n.prefixesBitSet.AsSlice(&[256]uint8{})
	if nPfxCount := len(allIndices); nPfxCount != 0 {

		// print the baseIndices for this node.
		fmt.Fprintf(w, "%sindexs(#%d): %v\n", indent, nPfxCount, allIndices)

		// print the prefixes for this node
		fmt.Fprintf(w, "%sprefxs(#%d):", indent, nPfxCount)

		for _, idx := range allIndices {
			pfx := n.cidrFromIdx(idx)
			fmt.Fprintf(w, " %s", pfx)
		}

		fmt.Fprintln(w)

		// skip values if the payload is the empty struct
		if _, ok := any(zero).(struct{}); !ok {

			// print the values for this node
			fmt.Fprintf(w, "%svalues(#%d):", indent, nPfxCount)

			for _, idx := range allIndices {
				fmt.Fprintf(w, " %#v", *n.prefixes[idx])
			}

			fmt.Fprintln(w)
		}
	}

	allChildren := n.childrenBitSet.AsSlice(&[256]uint8{})
	if childCount := len(allChildren); childCount > 0 {
		fmt.Fprintf(w, "%schilds(#%d):", indent, childCount)

		for _, addr := range allChildren {
			fmt.Fprintf(w, " %s", addrFmt(addr, is4))
		}

		fmt.Fprintln(w)
	}
}

// hasType returns the nodeType.
func (n *node[V]) hasType() nodeType {
	s := n.nodeStats()

	switch {
	case s.pfxs == 0 && s.childs == 0:
		return nullNode
	case s.childs == 0:
		return stopNode
	case s.childs > 0 && s.pfxs > 0:
		return fullNode
	case s.pfxs == 0 && s.childs > 0:
		return imedNode
	default:
		panic(fmt.Sprintf("UNREACHABLE: pfx: %d, chld: %d", s.pfxs, s.childs))
	}
}

// node statistics for this single node
func (n *node[V]) nodeStats() (s stats) {
	s.pfxs = n.prefixCount()
	s.childs = n.childCount()
	return s
}

// dumpString is just a wrapper for dump.
func (t *Table[V]) dumpString() string {
	w := new(strings.Builder)
	t.dump(w)

	return w.String()
}

// dump the table structure and all the nodes to w.
func (t *Table[V]) dump(w io.Writer) {
	if t == nil {
		return
	}

	stats := t.root4.nodeStatsRec()
	fmt.Fprintln(w)
	fmt.Fprintf(w, "### IPv4: size(%d), nodes(%d), pfxs(%d),", t.size4, stats.nodes, stats.pfxs)
	t.root4.dumpRec(w, []byte{}, 0)

	stats = t.root6.nodeStatsRec()
	fmt.Fprintln(w)
	fmt.Fprintf(w, "### IPv6: size(%d), nodes(%d), pfxs(%d),", t.size6, stats.nodes, stats.pfxs)
	t.root6.dumpRec(w, []byte{}, 0)
}

// dumpRec, rec-descent the trie.
func (n *node[V]) dumpRec(w io.Writer, octetPath []byte, depth int) {
	// dump this node
	n.dump(w, octetPath, depth)

	// the node may have childs, rec-descent down
	for _, addr := range n.childrenBitSet.AsSlice(&[256]uint8{}) {
		kid := n.getChild(addr)
		kid.dumpRec(w, append(octetPath[:depth:depth], addr), depth+1)
	}
}

// nodeStatsRec, calculate the number of pfxs, nodes and leaves under n, rec-descent.
func (n *node[V]) nodeStatsRec() stats {
	var s stats
	if n == nil || n.isEmpty() {
		return s
	}

	s.pfxs = n.prefixCount()
	s.childs = n.childCount()
	s.nodes = 1 // this node

	for _, octet := range n.childrenBitSet.AsSlice(&[256]uint8{}) {
		kid := n.children[octet]
		// rec-descent
		rs := kid.nodeStatsRec()

		s.pfxs += rs.pfxs
		s.childs += rs.childs
		s.nodes += rs.nodes

	}

	return s
}
