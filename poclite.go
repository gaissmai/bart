// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"

	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/bitset"
	"github.com/gaissmai/bart/internal/lpmbt"
	"github.com/gaissmai/bart/internal/sparse"
)

// LitePoC is the little sister of [Table]. LitePoC is ideal for simple
// IP access-control-lists, a.k.a. longest-prefix matches
// with plain true/false results.
//
// For all other tasks the much more powerful [Table] must be used.
type LitePoC struct {
	// used by -copylocks checker from `go vet`.
	_ noCopy

	// the root nodes, implemented as popcount compressed multibit tries
	root4 liteNode
	root6 liteNode
}

// rootNodeByVersion, root node getter for ip version.
func (l *LitePoC) rootNodeByVersion(is4 bool) *liteNode {
	if is4 {
		return &l.root4
	}
	return &l.root6
}

// Insert adds pfx to the trie.
func (l *LitePoC) Insert(pfx netip.Prefix) {
	if !pfx.IsValid() {
		return
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	is4 := pfx.Addr().Is4()
	n := l.rootNodeByVersion(is4)

	n.insertAtDepth(pfx, 0)
}

// Delete removes pfx from the trie.
func (l *LitePoC) Delete(pfx netip.Prefix) {
	if !pfx.IsValid() {
		return
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()
	octets := ip.AsSlice()
	lastIdx, lastBits := lastOctetIdxAndBits(bits)

	n := l.rootNodeByVersion(is4)

	// record path to deleted node
	// needed to purge and/or path compress nodes after deletion
	stack := [maxTreeDepth]*liteNode{}

	// find the trie node
	for depth, octet := range octets {
		// push current node on stack for path recording
		stack[depth&15] = n

		// delete prefix in trie node
		if depth == lastIdx {
			n.prefixes.Clear(art.PfxToIdx(octet, lastBits))
			n.purgeAndCompress(stack[:depth], octets, is4)
			return
		}

		addr := uint(octet)
		if !n.children.Test(addr) {
			return
		}
		kid := n.children.MustGet(addr)

		// kid is node or leaf at addr
		switch kid := kid.(type) {
		case *liteNode:
			n = kid
			continue // descend down to next trie level

		case *prefixNode:
			// reached a path compressed prefix, stop traversing
			if kid.prefix != pfx {
				// nothing to delete
				return
			}

			// prefix is equal leaf, delete leaf
			n.children.DeleteAt(addr)
			n.purgeAndCompress(stack[:depth], octets, is4)
			return

		case *fringeNode:
			if !isFringe(depth, bits) {
				// this prefix is no fringe, nothing to delete
				return
			}

			// delete this fringe
			n.children.DeleteAt(addr)
			n.purgeAndCompress(stack[:depth], octets, is4)
			return

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
}

// Contains performs a longest-prefix match for the IP address
// and returns true if any route matches, otherwise false.
func (l *LitePoC) Contains(ip netip.Addr) bool {
	// if ip is invalid, Is4() returns false and AsSlice() returns nil
	is4 := ip.Is4()
	n := l.rootNodeByVersion(is4)

	for _, octet := range ip.AsSlice() {
		addr := uint(octet)

		// for contains, any lpm match is good enough, no backtracking needed
		if n.lpmTest(art.HostIdx(addr)) {
			return true
		}

		// stop traversing?
		if !n.children.Test(addr) {
			return false
		}
		kid := n.children.MustGet(addr)

		// kid is node or leaf at addr
		switch kid := kid.(type) {
		case *liteNode:
			n = kid
			continue // descend down to next trie level

		case *prefixNode:
			// kid is a path-compressed prefix
			return kid.prefix.Contains(ip)

		case *fringeNode:
			// a fringe is the default-route for all nodes below
			return true

		default:
			panic("logic error, wrong node type")
		}
	}

	// invalid IP
	return false
}

// ###################################################################

// liteNode, see the node struct, but without payload V.
// Needs less memory and insert and delete is also a bit faster.
type liteNode struct {
	prefixes bitset.BitSet256
	children sparse.Array[any] // [any] is a *liteNode or a *prefixNode
}

// prefixNode, just a path compressed prefix.
type prefixNode struct {
	prefix netip.Prefix
	// value  struct{} // would be the value slot for the node[V] in Table[V]
}

// fringeNode, a addr/0 default route for all addrs below this children slot
type fringeNode struct {
	// value  struct{} // would be the value slot for the node[V] in Table[V]
}

// isEmpty returns true if node has no prefixes and no children.
func (n *liteNode) isEmpty() bool {
	return len(n.children.Items) == 0 && n.prefixes.Size() == 0
}

// lpmTest, any longest prefix match
func (n *liteNode) lpmTest(idx uint) bool {
	return n.prefixes.IntersectsAny(lpmbt.LookupTbl[idx])
}

// insertAtDepth, see the similar method for node, but now simpler without payload V.
func (n *liteNode) insertAtDepth(pfx netip.Prefix, depth int) {
	ip := pfx.Addr()
	bits := pfx.Bits()
	octets := ip.AsSlice()
	lastIdx, lastBits := lastOctetIdxAndBits(bits)

	// find the proper trie node to insert prefix
	// start with prefix octet at depth
	for ; depth < len(octets); depth++ {
		octet := octets[depth&15]
		addr := uint(octet)

		// last significant octet: insert/override prefix into node
		if depth == lastIdx {
			// just set a bit in the CBT, Lite has no payload to insert
			n.prefixes.Set(art.PfxToIdx(octet, lastBits))
			return
		}

		// reached end of trie path ...
		// insert prefix as path-compressed prefixNode or fringeNode
		if !n.children.Test(addr) {
			if isFringe(depth, bits) {
				n.children.InsertAt(addr, &fringeNode{})
				return
			}
			n.children.InsertAt(addr, &prefixNode{prefix: pfx})
			return
		}

		// ... or decend down the trie
		kid := n.children.MustGet(addr)

		// kid is recursive node or leaf node at addr
		switch kid := kid.(type) {
		case *liteNode:
			n = kid
			continue // descend down to next trie level

		case *prefixNode:
			// reached a path-compressed leaf
			if kid.prefix == pfx {
				// already exists, nothing to do
				return
			}

			// create new node
			// push the leaf node down
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(liteNode)
			newNode.insertAtDepth(kid.prefix, depth+1)

			n.children.InsertAt(addr, newNode)
			n = newNode

		case *fringeNode:
			// reached a fringe
			if isFringe(depth, bits) {
				// already exists, nothing to do
				return
			}

			// create new node
			// convert the fringeNode to a default route one level down
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(liteNode)
			newNode.prefixes.Set(1) // a fringeNode becomes the default prefix one level down

			n.children.InsertAt(addr, newNode)
			n = newNode

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
}

// purgeAndCompress, purge empty nodes or compress nodes with single prefix or leaf.
// similar to the same helper method for node, but without payload V.
func (n *liteNode) purgeAndCompress(parentStack []*liteNode, childPath []uint8, is4 bool) {
	// unwind the stack
	for depth := len(parentStack) - 1; depth >= 0; depth-- {
		parent := parentStack[depth&15]
		addr := uint(childPath[depth&15])

		prefixCount := n.prefixes.Size()
		childCount := n.children.Len()

		switch {
		case prefixCount == 0 && childCount == 0:
			// just delete this empty node
			parent.children.DeleteAt(addr)

		case prefixCount == 0 && childCount == 1:
			// single child is a leaf or fringe node?
			// shift the prefix up one level as path-compreessed leave
			if kid, ok := n.children.Items[0].(*prefixNode); ok {
				// and override current node with this leaf
				parent.children.InsertAt(addr, &prefixNode{prefix: kid.prefix})
				break
			}

			// a fringeNode mutates to a normal prefixNode one level above
			if _, ok := n.children.Items[0].(*fringeNode); ok {
				// make a copy
				path := stridePath{}
				copy(path[:], childPath)

				// get the prefix back, sorry it's contrived
				// we have a single child, get the octet from the children bitset
				octet, _ := n.children.FirstSet()

				// replace the octet at depth+1 with this child addr
				path[depth+1] = uint8(octet)

				// generate the prefix from path, we know it's a fringe
				// has always idx==1 => 0/0 as addr/pfxLen
				pfx := cidrFromFringe(path, depth+2, is4)

				// and override current node with new leaf
				parent.children.InsertAt(addr, &prefixNode{prefix: pfx})
				break
			}

		case prefixCount == 1 && childCount == 0:
			// make prefix from idx, shift leaf one level up
			// and override current node with new leaf
			idx, _ := n.prefixes.FirstSet()

			path := stridePath{}
			copy(path[:], childPath)
			pfx := cidrFromPath(path, depth+1, is4, idx)

			// if idx == 1, this is the default route in this node
			// make a fringe node from this prefix one level above
			if idx == 1 {
				parent.children.InsertAt(addr, &fringeNode{})
			} else {
				// insert current prefix as leave one level up
				parent.children.InsertAt(addr, &prefixNode{prefix: pfx})
			}

		}

		n = parent
	}
}

// cidrFromFringe, helper function,
// get prefix back from octet path, depth and IP version.
func cidrFromFringe(path stridePath, depth int, is4 bool) netip.Prefix {
	// zero/mask the bytes after prefix bits
	clear(path[depth:])

	// make ip addr from octets
	var ip netip.Addr
	if is4 {
		ip = netip.AddrFrom4([4]byte(path[:4]))
	} else {
		ip = netip.AddrFrom16(path)
	}

	// calc bits with depth, pfxLen is always 8 for fringeNode
	bits := depth << 3

	// return a normalized prefix from ip/bits
	return netip.PrefixFrom(ip, bits)
}
