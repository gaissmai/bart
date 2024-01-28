// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"sync"
)

// Table is an IPv4 and IPv6 routing table with payload V.
// The zero value is ready to use.
type Table[V any] struct {
	rootV4 *node[V]
	rootV6 *node[V]

	// simple API, no constructor needed
	initOnce sync.Once
}

// init once, so no constructor is needed.
// BitSets have to be initialized.
func (t *Table[V]) init() {
	t.initOnce.Do(func() {
		t.rootV4 = newNode[V]()
		t.rootV6 = newNode[V]()
	})
}

// rootNodeByVersion, select root node for ip version.
func (t *Table[V]) rootNodeByVersion(is4 bool) *node[V] {
	if is4 {
		return t.rootV4
	}
	return t.rootV6
}

// Insert adds pfx to the tree, with value val.
// If pfx is already present in the tree, its value is set to val.
func (t *Table[V]) Insert(pfx netip.Prefix, val V) {
	t.init()

	// always normalize the prefix
	pfx = pfx.Masked()

	// some needed values, see below
	bits := pfx.Bits()
	ip := pfx.Addr()
	is4 := ip.Is4()

	// get the root node of the routing table
	n := t.rootNodeByVersion(is4)

	// insert default route, easy peasy
	if bits == 0 {
		n.prefixes.insert(0, 0, val)
		return
	}

	// the ip is chunked in bytes, the multibit stride is 8
	bs := ip.AsSlice()

	// depth index for the child trie
	depth := 0
	for {
		addr := uint(bs[depth]) // stride = 8!

		// loop stop condition:
		// last non-masked addr chunk of prefix, insert the
		// byte and bits into the prefixHeap on this depth
		//
		// 8.0.0.0/5 ->       depth 0, addr byte  8,  bits 5
		// 10.0.0.0/8 ->      depth 0, addr byte  10, bits 8
		// 192.168.0.0/16  -> depth 1, addr byte 168, bits 0, (16-1*8 = 8)
		// 192.168.20.0/19 -> depth 2, addr byte  20, bits 3, (19-2*8 = 3)
		// 172.16.19.12/32 -> depth 3, addr byte  12, bits 8, (32-3*8 = 8)
		//
		if bits <= stride {
			n.prefixes.insert(addr, bits, val)
			return
		}

		// descend down to next child level
		child := n.children.get(addr)

		// create and insert missing intermediate child, no path compression!
		if child == nil {
			child = newNode[V]()
			n.children.insert(addr, child)
		}

		// go down
		depth++
		n = child
		bits -= stride
	}
}

// Delete removes pfx from the tree, pfx does not have to be present.
func (t *Table[V]) Delete(pfx netip.Prefix) {
	t.init()

	// always normalize the prefix
	pfx = pfx.Masked()

	bits := pfx.Bits()
	ip := pfx.Addr()
	is4 := ip.Is4()

	n := t.rootNodeByVersion(is4)

	// delete default route, easy peasy
	if bits == 0 {
		n.prefixes.delete(0, 0)
		return
	}

	// stack of the traversed child path in order to
	// purge dangling paths after deletion
	pathStack := [maxTreeDepth]*node[V]{}

	bs := ip.AsSlice()
	depth := 0
	for {
		addr := uint(bs[depth]) // stride = 8!

		// push current node on stack for path recording
		pathStack[depth] = n

		// last non-masked byte
		if bits <= stride {
			// found a child on proper depth ...
			if !n.prefixes.delete(addr, bits) {
				// ... but prefix not in tree, nothing deleted
				return
			}

			// purge dangling path, if needed
			break
		}

		// descend down to next level, no path compression
		child := n.children.get(addr)
		if child == nil {
			// no child, nothing to delete
			return
		}

		// go down
		depth++
		bits -= stride
		n = child
	}

	// check for dangling path
	for {

		// loop stop condition
		if depth == 0 {
			break
		}

		// is this an empty node?
		if len(n.prefixes.values) == 0 && len(n.children.nodes) == 0 {

			// purge this node from parents childs
			parent := pathStack[depth-1]
			parent.children.delete(uint(bs[depth-1]))
		}

		// go up
		depth--
		n = pathStack[depth]
	}
}

// Get does a route lookup for addr and returns the associated value and true, or false if
// no route matched.
func (t *Table[V]) Get(ip netip.Addr) (val V, ok bool) {
	t.init()
	_, _, val, ok = t.lpmByIP(ip)
	return
}

// Lookup does a route lookup for addr and returns the longest prefix,
// the associated value and true for success, or false otherwise if
// no route matched.
//
// Lookup is a bit slower than Get, so if you only need the payload V
// and not the matching longest-prefix back, you should use just Get.
func (t *Table[V]) Lookup(ip netip.Addr) (lpm netip.Prefix, val V, ok bool) {
	t.init()
	if depth, baseIdx, val, ok := t.lpmByIP(ip); ok {

		// add the bits from higher levels in child trie to pfxLen
		bits := depth*stride + baseIndexToPrefixLen(baseIdx)

		// mask prefix from lookup ip, masked with longest prefix bits.
		lpm = netip.PrefixFrom(ip, bits).Masked()

		return lpm, val, ok
	}
	return
}

// lpmByIP does a route lookup for IP with longest prefix match.
// Returns also depth and baseIdx for Lookup to retrieve the
// lpm prefix out of the prefix tree.
func (t *Table[V]) lpmByIP(ip netip.Addr) (depth int, baseIdx uint, val V, ok bool) {
	is4 := ip.Is4()
	n := t.rootNodeByVersion(is4)

	// stack of the traversed nodes for fast backtracking, if needed
	pathStack := [maxTreeDepth]*node[V]{}

	// keep the lpm alloc free, don't use ip.AsSlice here
	a16 := ip.As16()
	bs := a16[:]
	if is4 {
		bs = bs[12:]
	}

	depth = 0
	addr := uint(bs[depth]) // bytewise, stride = 8
	// find leaf tree
	for {

		// push current node on stack for fast backtracking
		pathStack[depth] = n

		// go down in tight loop to leaf tree
		if child := n.children.get(addr); child != nil {
			depth++
			addr = uint(bs[depth])
			n = child
			continue
		}

		break
	}

	// start backtracking at leaf node in tight loop
	for {
		// longest prefix match?
		if baseIdx, val, ok := n.prefixes.lpmByIndex(addrToBaseIndex(addr)); ok {
			// return also baseIdx and the depth, needed to
			// calculate the lpm prefix by the Lookup method.
			return depth, baseIdx, val, true
		}

		// end condition, stack is exhausted
		if depth == 0 {
			return
		}

		// go up, backtracking
		depth--
		addr = uint(bs[depth])
		n = pathStack[depth]
	}
}
