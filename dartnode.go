package bart

import (
	"fmt"
	"io"
	"net/netip"
	"strings"
)

const (
	firstHostIdx = 256
	lastHostIdx  = 511
	numChildren  = 256
)

// artNode is a trie level node in the multibit routing table.
//
// Each artNode contains two conceptually different fixed sized arrays:
//   - prefixes: representing routes, using a complete binary tree layout
//     driven by the baseIndex() function from the ART algorithm.
//   - children: holding subtries a branching factor of 256.
//
// See doc/artlookup.pdf for the mapping mechanics and prefix tree details.
type artNode[V any] struct {
	prefixes [lastHostIdx + 1]*V // 512
	children [numChildren]any    // 256

	prefixCount int16
	childCount  int16
}

// isEmpty returns true if node has neither prefixes nor children
func (n *artNode[V]) isEmpty() bool {
	return n.prefixCount == 0 && n.childCount == 0
}

func (n *artNode[V]) prefixesAsSlice() []uint8 {
	res := make([]uint8, 0, 512)
	for i, pv := range n.prefixes {
		if pv != nil {
			res = append(res, uint8(i))
		}
	}
	return res
}

func (n *artNode[V]) childrenAsSlice() []uint8 {
	res := make([]uint8, 0, 256)
	for i, kid := range n.children {
		if kid != nil {
			res = append(res, uint8(i))
		}
	}
	return res
}

// TODO
func prefix2Index(addr uint8, prefixLen int) int {
	return (int(addr) >> (8 - prefixLen)) + (1 << prefixLen)
}

// TODO
func hostIndex(addr uint8) int {
	return int(addr) + firstHostIdx
}

// TODO
func parentIndex(idx int) int {
	return idx >> 1
}

// getChild TODO
func (n *artNode[V]) getChild(addr uint8) any {
	return n.children[addr]
}

func (n *artNode[V]) getOrCreateChild(addr uint8) any {
	c := n.children[addr]
	if c == nil {
		c = &artNode[V]{}
		n.children[addr] = c
		n.childCount++
	}
	return c
}

// insertChild TODO
func (n *artNode[V]) insertChild(addr uint8, child any) {
	if n.children[addr] == nil {
		n.childCount++
	}
	n.children[addr] = child
}

// deleteChild TODO
func (n *artNode[V]) deleteChild(addr uint8) {
	if n.children[addr] != nil {
		n.childCount--
	}
	n.children[addr] = nil
}

// insertPrefix adds the route addr/prefixLen to n, with value val.
func (n *artNode[V]) insertPrefix(addr uint8, prefixLen int, val V) (exists bool) {
	exists = true

	idx := prefix2Index(addr, prefixLen)
	if !n.isStartIdx(idx) {
		// new prefix
		exists = false
		n.prefixCount++
	}

	// To ensure allot works as intended, every unique prefix in the
	// artNode must point to a distinct value pointer, even for identical values.
	// Using new() and assignment guarantees each inserted prefix gets its own address.
	p := new(V)
	*p = val

	old := n.prefixes[idx]
	n.allotRec(idx, old, p)

	return
}

func (n *artNode[V]) insertAtDepth(pfx netip.Prefix, val V, depth int) (exists bool) {
	ip := pfx.Addr() // the pfx must be in canonical form
	bits := pfx.Bits()
	octets := ip.AsSlice()
	maxDepth, lastBits := finalArt(bits)

	// find the proper trie node to insert prefix
	// start with prefix octet at depth
	for _, octet := range octets[depth:] {
		// last masked octet: insert/override prefix/val into node
		if depth == maxDepth {
			return n.insertPrefix(octet, lastBits, val)
		}

		// maybe nil
		kidAny := n.getChild(octet)

		// insert leafNode path compressed
		if kidAny == nil {
			n.insertChild(octet, newLeafNode(pfx, val))
			return false
		}

		panic("TODO: insertAt fro ART nodes")
	}

	panic("unreachable")
}

// getPrefix TODO
func (n *artNode[V]) getPrefix(addr uint8, prefixLen int) (val V, exists bool) {
	idx := prefix2Index(addr, prefixLen)
	if n.isStartIdx(idx) {
		pv := n.prefixes[idx]
		return *pv, true
	}
	// Route entry doesn't exist
	return val, false
}

// deletePrefix TODO
func (n *artNode[V]) deletePrefix(addr uint8, prefixLen int) (val V, exists bool) {
	idx := prefix2Index(addr, prefixLen)
	if !n.isStartIdx(idx) {
		// Route entry doesn't exist
		return val, false
	}

	pv := n.prefixes[idx]
	var parentVal *V
	if parentIdx := parentIndex(idx); parentIdx != 0 {
		parentVal = n.prefixes[parentIdx]
	}

	n.allotRec(idx, pv, parentVal)
	n.prefixCount--

	return *pv, true
}

// contains TODO
func (n *artNode[V]) contains(addr uint8) (ok bool) {
	return n.prefixes[hostIndex(addr)] != nil
}

// lookup TODO
func (n *artNode[V]) lookup(addr uint8) (ret V, ok bool) {
	if val := n.prefixes[hostIndex(addr)]; val != nil {
		return *val, true
	}
	return ret, false
}

// allotRec updates entries in the subtree rooted at idx whose stored prefix pointer equals old.
// For each matching entry, it updates the prefix pointer to val.
//
// This function is central to the ART algorithm, efficiently supporting fast lookups.
func (n *artNode[V]) allotRec(idx int, old, val *V) {
	if n.prefixes[idx] != old {
		// This index doesn't match what we're looking for-likely a recursive call
		// has reached a child node with a more specific route already in place.
		// Don't modify this branch.
		return
	}
	n.prefixes[idx] = val
	if idx >= firstHostIdx {
		// updated a host route, we've reached a leaf node in the binary tree.
		return
	}

	// Continue updating in both child subtrees.
	leftChildIdx := idx << 1
	n.allotRec(leftChildIdx, old, val)

	rightChildIdx := leftChildIdx + 1
	n.allotRec(rightChildIdx, old, val)
}

// isStartIdx TODO
func (n *artNode[V]) isStartIdx(idx int) bool {
	val := n.prefixes[idx]
	if val == nil {
		return false
	}

	parentIdx := parentIndex(idx)
	if parentIdx == 0 {
		// idx is non-nil, and is at the 0/0 route position.
		return true
	}
	if parentVal := n.prefixes[parentIdx]; val != parentVal {
		// parent node in the tree isn't the same prefix, so idx must
		// be a startIdx
		return true
	}
	return false
}

// nodeStatsRec, calculate the number of pfxs, nodes and leaves under n, rec-descent.
func (n *artNode[V]) nodeStatsRec() stats {
	var s stats
	if n == nil || n.isEmpty() {
		return s
	}

	s.pfxs = int(n.prefixCount)
	s.childs = int(n.childCount)
	s.nodes = 1 // this node
	s.leaves = 0
	s.fringes = 0

	for _, kidAny := range n.children {
		if kidAny == nil {
			continue
		}

		switch kid := kidAny.(type) {
		case *artNode[V]:
			// rec-descent
			rs := kid.nodeStatsRec()

			s.pfxs += rs.pfxs
			s.childs += rs.childs
			s.nodes += rs.nodes
			s.leaves += rs.leaves
			s.fringes += rs.fringes

		case *bartNode[V]:
			// rec-descent
			rs := kid.nodeStatsRec()

			s.pfxs += rs.pfxs
			s.childs += rs.childs
			s.nodes += rs.nodes
			s.leaves += rs.leaves
			s.fringes += rs.fringes

		case *leafNode[V]:
			s.leaves++

		default:
			panic("logic error, wrong node type")
		}
	}

	return s
}

// dump the node to w.
func (n *artNode[V]) dump(w io.Writer, path stridePath, depth int, is4 bool) {
	var zero V

	bits := depth * strideLen
	indent := strings.Repeat(".", depth)

	// node type with depth and octet path and bits.
	fmt.Fprintf(w, "\n%s[%s: ART] depth:  %d path: [%s] / %d\n",
		indent, n.hasType(), depth, ipStridePath(path, depth, is4), bits)

	if nPfxCount := n.prefixCount; nPfxCount != 0 {
		// no heap allocs
		allIndices := n.prefixesAsSlice()

		// print the baseIndices for this node.
		fmt.Fprintf(w, "%sindexs(#%d): %v\n", indent, nPfxCount, allIndices)

		// print the prefixes for this node
		fmt.Fprintf(w, "%sprefxs(#%d):", indent, nPfxCount)

		for _, idx := range allIndices {
			// pfx := cidrFromPath(path, depth, is4, idx)
			_ = idx
			fmt.Fprintf(w, " %s", "TODO")
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

	if n.childCount != 0 {

		childAddrs := make([]uint8, 0, maxItems)
		leafAddrs := make([]uint8, 0, maxItems)

		// the node has recursive child nodes or path-compressed leaves
		for i, kid := range n.children {
			if kid == nil {
				continue
			}
			addr := uint8(i)

			switch kid.(type) {
			case *artNode[V]:
				childAddrs = append(childAddrs, addr)
				continue
			case *bartNode[V]:
				childAddrs = append(childAddrs, addr)
				continue
			case *leafNode[V]:
				leafAddrs = append(leafAddrs, addr)

			default:
				panic("logic error, wrong node type")
			}
		}

		// print the children for this node.
		fmt.Fprintf(w, "%soctets(#%d): %v\n", indent, n.childCount, n.childrenAsSlice())

		if leafCount := len(leafAddrs); leafCount > 0 {
			// print the pathcomp prefixes for this node
			fmt.Fprintf(w, "%sleaves(#%d):", indent, leafCount)

			for _, addr := range leafAddrs {
				k := n.children[addr]
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
func (n *artNode[V]) hasType() nodeType {
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
func (n *artNode[V]) nodeStats() stats {
	var s stats

	s.pfxs = int(n.prefixCount)
	s.childs = int(n.childCount)

	for _, kid := range n.children {
		if kid == nil {
			continue
		}

		switch kid.(type) {
		case *artNode[V]:
			s.nodes++

		case *bartNode[V]:
			s.nodes++

		case *leafNode[V]:
			s.leaves++

		default:
			panic("logic error, wrong node type")
		}
	}

	return s
}

// dumpString is just a wrapper for dump.
func (d *Dart[V]) dumpString() string {
	w := new(strings.Builder)
	d.dump(w)

	return w.String()
}

// dump the table structure and all the nodes to w.
func (d *Dart[V]) dump(w io.Writer) {
	if d == nil {
		return
	}

	if d.size4 > 0 {
		stats := d.root4.nodeStatsRec()
		fmt.Fprintln(w)
		fmt.Fprintf(w, "### IPv4: size(%d), nodes(%d), pfxs(%d), leaves(%d), fringes(%d),",
			d.size4, stats.nodes, stats.pfxs, stats.leaves, stats.fringes)
		d.root4.dumpRec(w, stridePath{}, 0, true)
	}

	if d.size6 > 0 {
		stats := d.root6.nodeStatsRec()
		fmt.Fprintln(w)
		fmt.Fprintf(w, "### IPv6: size(%d), nodes(%d), pfxs(%d), leaves(%d), fringes(%d),",
			d.size6, stats.nodes, stats.pfxs, stats.leaves, stats.fringes)
		d.root6.dumpRec(w, stridePath{}, 0, false)
	}
}

// dumpRec, rec-descent the trie.
func (n *artNode[V]) dumpRec(w io.Writer, path stridePath, depth int, is4 bool) {
	// dump this node
	n.dump(w, path, depth, is4)

	// the node may have childs, rec-descent down
	for addr, kidAny := range n.children {
		if kidAny == nil {
			continue
		}

		path[depth&15] = uint8(addr)

		switch kid := kidAny.(type) {
		case *artNode[V]:
			kid.dumpRec(w, path, depth+1, is4)
		case *bartNode[V]:
			kid.dumpRec(w, path, depth+1, is4)
		}
	}
}
