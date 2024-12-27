// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"slices"

	"github.com/gaissmai/bart/internal/sparse"
)

const (
	strideLen       = 8   // octet
	maxTreeDepth    = 16  // 16 for IPv6
	maxNodeChildren = 256 // 256
	maxNodePrefixes = 512 // 512
)

// a zero value, used manifold
var zeroPath [16]byte

// node is a level node in the multibit-trie.
// A node has prefixes and children, forming the multibit trie.
//
// The prefixes form a complete binary tree, see the artlookup.pdf
// paper in the doc folder to understand the data structure.
//
// In contrast to the ART algorithm, sparse arrays
// (popcount-compressed slices) are used instead of fixed-size arrays.
//
// The array slots are also not pre-allocated (alloted) as described
// in the ART algorithm, but backtracking is used for the longest-prefix-match.
//
// The lookup is then slower, but this is the intended trade-off to prevent
// memory consumption from exploding.
//
// The nodes can also be used in path compressd mode, this reduces the memory consumption
// by almost an order of magnitude, but the updates (insert/delete) are slower,
// but the search times remain comparable.
type node[V any] struct {
	// prefixes contains the routes with payload V
	prefixes sparse.Array[V]

	// children, recursively spans the trie with a branching factor of 256
	children sparse.Array[*node[V]]

	// path compressed items, just a nil pointer without path compression
	// and additional 8 bytes per node wasted without compression.
	pathcomp *sparse.Array[*pathItem[V]]
}

// newNode returns a *node.
// If n was path compressed, the new node
// is it also.
func (n node[V]) newNode() *node[V] {
	c := new(node[V])
	if n.pathcomp != nil {
		// also make n.pathcomp != nil in new node
		c.pathcomp = &sparse.Array[*pathItem[V]]{}
	}
	return c
}

// pathItem is prefix and value together
type pathItem[V any] struct {
	prefix netip.Prefix
	value  V
}

// isEmpty returns true if node has neither prefixes nor children nor path compressed items.
func (n *node[V]) isEmpty() bool {
	return n.prefixes.Len() == 0 &&
		n.children.Len() == 0 &&
		(n.pathcomp == nil || n.pathcomp.Len() == 0)
}

// insertAtDepth insert a prefix/val into a node tree at depth.
// n must not be nil, prefix must be valid and already in canonical form.
//
// Required if a path compression has to be resolved because a new value
// is added that collides with the compression. The prefix is then reinserted
// further down at depth in the tree.
func (n *node[V]) insertAtDepth(pfx netip.Prefix, val V, depth int) (exists bool) {
	ip := pfx.Addr()
	bits := pfx.Bits()
	octets := ip.AsSlice()

	lastOctetIdx := (bits - 1) / strideLen
	lastOctet := octets[lastOctetIdx]
	lastOctetBits := bits - (lastOctetIdx * strideLen)

	// take the desired octets from prefix, starting at depth
	for i := depth; i < lastOctetIdx; i++ {
		addr := uint(octets[i])

		// descend down to next trie level
		if c, ok := n.children.Get(addr); ok {
			n = c
			continue
		}

		// no child found, look for path compressed item in slot
		pc, ok := n.pathcomp.Get(addr)
		if !ok {
			// insert prefix path compressed
			return n.pathcomp.InsertAt(addr, &pathItem[V]{pfx, val})
		}

		// pathcomp slot is already occupied

		// override prefix in slot if equal
		if pc.prefix == pfx {
			pc.value = val
			return true
		}

		// free this pathcomp slot ...
		// insert new intermdiate child ...
		// shuffle down existing path-compressed prefix
		// shuffle down prefix
		n.pathcomp.DeleteAt(addr)

		c := n.newNode()
		n.children.InsertAt(addr, c)
		n = c

		// shuffle down
		_ = n.insertAtDepth(pc.prefix, pc.value, depth+1)
		return n.insertAtDepth(pfx, val, depth+1)
	}

	// insert/override flattened prefix/val into node
	return n.prefixes.InsertAt(pfxToIdx(lastOctet, lastOctetBits), val)
}

// purgeParents, dangling nodes after successful deletion
func (n *node[V]) purgeParents(parentStack []*node[V], childPath []byte) {
	for i := len(parentStack) - 1; i >= 0; i-- {
		if n.isEmpty() {
			parent := parentStack[i]
			parent.children.DeleteAt(uint(childPath[i]))
		}
		n = parentStack[i]
	}
}

// lpm does a route lookup for idx in the 8-bit (stride) routing table
// at this depth and returns (baseIdx, value, true) if a matching
// longest prefix exists, or ok=false otherwise.
//
// backtracking is fast, it's just a bitset test and, if found, one popcount.
// max steps in backtracking is the stride length.
func (n *node[V]) lpm(idx uint) (baseIdx uint, val V, ok bool) {
	// shortcut optimization
	minIdx, ok := n.prefixes.FirstSet()
	if !ok {
		return 0, val, false
	}

	// backtracking the CBT
	for baseIdx = idx; baseIdx >= minIdx; baseIdx >>= 1 {
		// practically it's get, but get is not inlined
		if n.prefixes.Test(baseIdx) {
			return baseIdx, n.prefixes.MustGet(baseIdx), true
		}
	}

	// not found (on this level)
	return 0, val, false
}

// lpmTest for faster lpm tests without value returns
func (n *node[V]) lpmTest(idx uint) bool {
	// shortcut optimization
	minIdx, ok := n.prefixes.FirstSet()
	if !ok {
		return false
	}

	// backtracking the CBT
	for idx := idx; idx >= minIdx; idx >>= 1 {
		if n.prefixes.Test(idx) {
			return true
		}
	}

	return false
}

// ### more complex functions than routing table lookups ###

// eachLookupPrefix does an all prefix match in the 8-bit (stride) routing table
// at this depth and calls yield() for any matching CIDR.
func (n *node[V]) eachLookupPrefix(
	path [16]byte,
	depth int,
	is4 bool,
	octet byte,
	pfxLen int,
	yield func(netip.Prefix, V) bool,
) bool {
	// backtracking the CBT
	for idx := pfxToIdx(octet, pfxLen); idx > 0; idx >>= 1 {
		if val, ok := n.prefixes.Get(idx); ok {
			cidr, _ := cidrFromPath(path, depth, is4, idx)

			if !yield(cidr, val) {
				// early exit
				return false
			}
		}
	}

	return true
}

// eachSubnet calls yield() for any covered CIDR by parent prefix in natural CIDR sort order.
func (n *node[V]) eachSubnet(
	path [16]byte,
	depth int,
	is4 bool,
	octet byte,
	pfxLen int,
	yield func(netip.Prefix, V) bool,
) bool {
	// ###############################################################
	// 1. collect all indices in n covered by prefix
	// ###############################################################
	pfxFirstAddr := uint(octet)
	pfxLastAddr := uint(octet | ^netMask(pfxLen))

	allCoveredIndices := make([]uint, 0, maxNodePrefixes)

	var idx uint
	var ok bool
	for {
		if idx, ok = n.prefixes.NextSet(idx); !ok {
			break
		}

		// idx is covered by prefix
		thisOctet, thisPfxLen := idxToPfx(idx)

		thisFirstAddr := uint(thisOctet)
		thisLastAddr := uint(thisOctet | ^netMask(thisPfxLen))

		if thisFirstAddr >= pfxFirstAddr && thisLastAddr <= pfxLastAddr {
			allCoveredIndices = append(allCoveredIndices, idx)
		}

		idx++
	}

	// sort indices in CIDR sort order
	slices.SortFunc(allCoveredIndices, cmpIndexRank)

	// ###############################################################
	// 2. collect all children in n covered by prefix
	// ###############################################################

	allCoveredAddrs := make([]uint, 0, maxNodeChildren)

	var addr uint

	for {
		if addr, ok = n.children.NextSet(addr); !ok {
			break
		}

		// host addrs are sorted in indexRank order
		if addr > pfxLastAddr {
			break
		}

		if addr >= pfxFirstAddr {
			allCoveredAddrs = append(allCoveredAddrs, addr)
		}

		addr++
	}

	cursor := 0

	// #####################################################
	// 3. yield indices and childs in CIDR sort order
	// #####################################################

	for _, idx := range allCoveredIndices {
		thisOctet, _ := idxToPfx(idx)

		// yield all childs before idx
		for j := cursor; j < len(allCoveredAddrs); j++ {
			addr = allCoveredAddrs[j]

			// yield prefix
			if addr >= uint(thisOctet) {
				break
			}

			// yield child

			octet = byte(addr)
			c, _ := n.children.Get(uint(octet))

			// add (set) this octet to path
			path[depth] = octet

			// all cidrs under this child are covered by pfx
			if !c.allRecSorted(path, depth+1, is4, yield) {
				// early exit
				return false
			}

			cursor++
		}

		// yield the prefix for this idx
		cidr, _ := cidrFromPath(path, depth, is4, idx)
		if !yield(cidr, n.prefixes.MustGet(idx)) {
			// early exit
			return false
		}
	}

	// ###############################################
	// 4. yield the rest of childs, if any
	// ###############################################

	for j := cursor; j < len(allCoveredAddrs); j++ {
		addr = allCoveredAddrs[j]

		octet = byte(addr)
		c, _ := n.children.Get(uint(octet))

		// add (set) this octet to path
		path[depth] = octet

		// all cidrs under this child are covered by pfx
		if !c.allRecSorted(path, depth+1, is4, yield) {
			// early exit
			return false
		}
	}

	return true
}

// unionRec combines two nodes, changing the receiver node.
// If there are duplicate entries, the value is taken from the other node.
// Count duplicate entries to adjust the t.size struct members.
func (n *node[V]) unionRec(o *node[V], depth int) (duplicates int) {
	allIndices := o.prefixes.AsSlice(make([]uint, 0, maxNodePrefixes))

	// for all prefixes in other node do ...
	for i, oIdx := range allIndices {
		// insert/overwrite prefix/value from oNode to nNode
		if exists := n.prefixes.InsertAt(oIdx, o.prefixes.Items[i]); exists {
			// this prefix is duplicate in n and o
			duplicates++
		}
	}

	if n.pathcomp != nil {
		allPathCompAddrs := o.pathcomp.AsSlice(make([]uint, 0, maxNodeChildren))
		// for all pathcomp items in other node do ...
		for i, addr := range allPathCompAddrs {
			oPCItem := o.pathcomp.Items[i]

			// get n child with same addr, if exists insert prefix at depth
			if nc, ok := n.children.Get(addr); ok {
				if nc.insertAtDepth(oPCItem.prefix, oPCItem.value, depth+1) {
					// this prefix is duplicate in n and o
					duplicates++
				}
				continue
			}

			// no child found, look for path compressed item in slot
			if nPCItem, ok := n.pathcomp.Get(addr); ok {
				if nPCItem.prefix == oPCItem.prefix {
					nPCItem.value = oPCItem.value
					// this prefix is duplicate in n and o
					duplicates++
					continue
				}

				// free this pathcomp slot ...
				// insert new intermdiate child ...
				// shuffle down existing path-compressed prefix
				// union other path-compressed prefix
				n.pathcomp.DeleteAt(addr)

				nc := n.newNode()
				n.children.InsertAt(addr, nc)

				// shuffle down
				_ = nc.insertAtDepth(nPCItem.prefix, nPCItem.value, depth+1)

				// union other
				exists := nc.insertAtDepth(oPCItem.prefix, oPCItem.value, depth+1)
				if exists {
					duplicates++
				}

				continue
			}

			// no child nor pathcomp, insert as path compressed
			n.pathcomp.InsertAt(addr, oPCItem)
		}
	}

	allChildAddrs := o.children.AsSlice(make([]uint, 0, maxNodeChildren))

	// for all children in other node do ...
	for i, addr := range allChildAddrs {
		oc := o.children.Items[i]

		if n.pathcomp != nil {
			// get n pathcomp with same addr
			if nPCItem, ok := n.pathcomp.Get(addr); ok {
				// free this pathcomp slot ...
				// insert new intermdiate child ...
				// shuffle down existing path-compressed prefix
				// union other child
				n.pathcomp.DeleteAt(addr)

				nc := n.newNode()
				n.children.InsertAt(addr, nc)

				// shuffle down
				_ = nc.insertAtDepth(nPCItem.prefix, nPCItem.value, depth+1)

				duplicates += nc.unionRec(oc, depth+1)
				continue
			}
		}

		// get n child with same addr,
		if nc, ok := n.children.Get(addr); !ok {
			// insert cloned child from oNode into nNode
			n.children.InsertAt(addr, oc.cloneRec())
		} else {
			// both nodes have child with addr, call union rec-descent
			duplicates += nc.unionRec(oc, depth+1)
		}
	}

	return duplicates
}

// cloneRec, clones the node recursive.
func (n *node[V]) cloneRec() *node[V] {
	c := n.newNode()
	if n.isEmpty() {
		return c
	}

	c.prefixes.BitSet = n.prefixes.BitSet.Clone()     // deep
	c.prefixes.Items = slices.Clone(n.prefixes.Items) // values, shallow copy

	// deep copy if V implements Cloner[V]
	for i, v := range c.prefixes.Items {
		if v, ok := any(v).(Cloner[V]); ok {
			c.prefixes.Items[i] = v.Clone()
		} else {
			break
		}
	}

	c.children.BitSet = n.children.BitSet.Clone()     // deep
	c.children.Items = slices.Clone(n.children.Items) // children, shallow copy

	// deep copy of children
	for i, child := range c.children.Items {
		c.children.Items[i] = child.cloneRec()
	}

	if n.pathcomp != nil {
		c.pathcomp.BitSet = n.pathcomp.BitSet.Clone()     // deep
		c.pathcomp.Items = slices.Clone(n.pathcomp.Items) // values, shallow copy

		// deep copy
		for i, pc := range c.pathcomp.Items {
			item := *pc

			// deep copy if V implements Cloner[V]
			if v, ok := any(item.value).(Cloner[V]); ok {
				item.value = v.Clone()
			}
			c.pathcomp.Items[i] = &item
		}
	}

	return c
}

// allRec runs recursive the trie, starting at this node and
// the yield function is called for each route entry with prefix and value.
// If the yield function returns false the recursion ends prematurely and the
// false value is propagated.
//
// The iteration order is not defined, just the simplest and fastest recursive implementation.
func (n *node[V]) allRec(
	path [16]byte,
	depth int,
	is4 bool,
	yield func(netip.Prefix, V) bool,
) bool {
	allIndices := n.prefixes.AsSlice(make([]uint, 0, maxNodePrefixes))

	// for all prefixes in this node do ...
	for _, idx := range allIndices {
		cidr, _ := cidrFromPath(path, depth, is4, idx)

		// make the callback for this prefix
		if !yield(cidr, n.prefixes.MustGet(idx)) {
			// early exit
			return false
		}
	}

	// for all path compressed items do ...
	if n.pathcomp != nil {
		for _, pc := range n.pathcomp.Items {
			// make the callback for this prefix
			if !yield(pc.prefix, pc.value) {
				// early exit
				return false
			}
		}
	}

	allChildAddrs := n.children.AsSlice(make([]uint, 0, maxNodeChildren))
	// for all children in this node do ...
	for i, addr := range allChildAddrs {
		child := n.children.Items[i]
		path[depth] = byte(addr)

		if !child.allRec(path, depth+1, is4, yield) {
			// early exit
			return false
		}
	}

	return true
}

// allRecSorted runs recursive the trie, starting at node and
// the yield function is called for each route entry with prefix and value.
// The iteration is in prefix sort order.
//
// If the yield function returns false the recursion ends prematurely and the
// false value is propagated.
func (n *node[V]) allRecSorted(
	path [16]byte,
	depth int,
	is4 bool,
	yield func(netip.Prefix, V) bool,
) bool {
	// get slice of all indexes, sorted by idx
	allIndices := n.prefixes.AsSlice(make([]uint, 0, maxNodePrefixes))

	// sort indices in CIDR sort order
	slices.SortFunc(allIndices, cmpIndexRank)

	// get all the bits in fast addressable form as a set of bool.
	allChildSet := n.children.AsSet(make([]bool, 0, maxNodeChildren))
	allPathCompSet := []bool{}
	if n.pathcomp != nil {
		allPathCompSet = n.pathcomp.AsSet(make([]bool, 0, maxNodeChildren))
	}

	// yield indices, pathcomp prefixes and childs in CIDR sort order
	var lower, upper uint

	for _, idx := range allIndices {
		pfxAddr, _ := idxToPfx(idx)
		upper = uint(pfxAddr)

		// for all pathcomp and child items < pfxAddr
		for addr := lower; addr < upper; addr++ {
			// either pathcomp or children match this addr, but not possible for both

			if n.pathcomp != nil && allPathCompSet[addr] {
				pc := n.pathcomp.MustGet(addr)
				if !yield(pc.prefix, pc.value) {
					return false
				}
			}

			if allChildSet[addr] {
				// yield this child rec-descent, if matched
				c := n.children.MustGet(addr)
				path[depth] = byte(addr)
				if !c.allRecSorted(path, depth+1, is4, yield) {
					return false
				}
			}

		}

		// yield the prefix for this idx
		cidr, _ := cidrFromPath(path, depth, is4, idx)
		if !yield(cidr, n.prefixes.MustGet(idx)) {
			return false
		}

		// forward lower bound for next round
		lower = upper
	}

	// yield the rest of pathcomp and child items, if any
	for addr := lower; addr < maxNodeChildren; addr++ {
		// either pathcomp or children match this addr, but not possible for both

		if n.pathcomp != nil && allPathCompSet[addr] {
			pc := n.pathcomp.MustGet(addr)
			if !yield(pc.prefix, pc.value) {
				return false
			}
		}

		if allChildSet[addr] {
			c := n.children.MustGet(addr)
			path[depth] = byte(addr)
			if !c.allRecSorted(path, depth+1, is4, yield) {
				return false
			}
		}
	}

	return true
}
