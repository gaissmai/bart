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

// Lite is the little sister of [Table]. Lite is ideal for simple
// IP access-control-lists, a.k.a. longest-prefix matches
// with plain true/false results.
//
// For all other tasks the much more powerful [Table] must be used.
type Lite struct {
	// used by -copylocks checker from `go vet`.
	_ noCopy

	// the root nodes, implemented as popcount compressed multibit tries
	root4 liteNode
	root6 liteNode
}

// rootNodeByVersion, root node getter for ip version.
func (l *Lite) rootNodeByVersion(is4 bool) *liteNode {
	if is4 {
		return &l.root4
	}
	return &l.root6
}

// Insert adds pfx to the trie.
func (l *Lite) Insert(pfx netip.Prefix) {
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
func (l *Lite) Delete(pfx netip.Prefix) {
	if !pfx.IsValid() {
		return
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()

	n := l.rootNodeByVersion(is4)

	// delete global default route upfront
	if bits == 0 {
		n.prefixes.Clear(1)
		return
	}

	lastIdx, lastBits := liteOctetIdxAndBits(bits)

	octets := ip.AsSlice()

	// record path to deleted node
	// needed to purge and/or path compress nodes after deletion
	stack := [maxTreeDepth]*liteNode{}

	// find the trie node
	for depth, octet := range octets {
		// push current node on stack for path recording
		stack[depth] = n

		// delete prefix in trie node
		if depth == lastIdx && !isFringe(bits) {
			n.prefixes.Clear(art.PfxToIdx(octet, lastBits))
			n.purgeAndCompress(stack[:depth], octets, is4)
			return
		}

		if depth > lastIdx && isFringe(bits) {
			n.prefixes.Clear(1)
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

		case *fringeNode:
			if depth == lastIdx && isFringe(bits) {
				n.children.DeleteAt(addr)
				n.purgeAndCompress(stack[:depth], octets, is4)
			}
			return

		case *prefixNode:
			// reached a path compressed prefix, stop traversing
			if kid.Prefix != pfx {
				// nothing to delete
				return
			}

			// prefix is equal leaf, delete leaf
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
func (l *Lite) Contains(ip netip.Addr) bool {
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

		case *fringeNode:
			// kid is a addr/8 fringe
			return true

		case *prefixNode:
			// kid is a path-compressed prefix
			return kid.Contains(ip)

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
	prefixes bitset.BitSetFringe
	// [any] is a:
	//  *liteNode, regular recursive trie node, or a
	//  *prefixNode, a path-compressed prefix, or a
	//  *fringeNode, an addr/8 prefix
	children sparse.ArrayFringe[any]
}

// prefixNode, just a path compressed prefix.
type prefixNode struct {
	netip.Prefix
}

// fringeNode, just a place holder, empty stcrut.
type fringeNode struct{}

// #############################################################

func (n *liteNode) isEmpty() bool {
	return len(n.children.Items) == 0 && n.prefixes.Size() == 0
}

// lpmTest, any longest prefix match
func (n *liteNode) lpmTest(idx uint) bool {
	return n.prefixes.IntersectsAny(lpmbt.LookupTblArray[idx])
}

// insertAtDepth, see the similar method for node, but now simpler without payload V.
func (n *liteNode) insertAtDepth(pfx netip.Prefix, depth int) {
	ip := pfx.Addr()
	bits := pfx.Bits()

	// handle default route upfront
	if bits == 0 {
		n.prefixes.Set(1)
		return
	}

	lastIdx, lastBits := liteOctetIdxAndBits(bits)

	octets := ip.AsSlice()

	// find the proper trie node to insert prefix
	// start with prefix octet at depth
	for ; depth < len(octets); depth++ {
		octet := octets[depth]
		addr := uint(octet)

		// last significant octet: insert/override prefix into node
		if depth == lastIdx && !isFringe(bits) {
			// just set a bit, no payload to insert
			n.prefixes.Set(art.PfxToIdx(octet, lastBits))
			return
		}

		if !n.children.Test(addr) {
			// insert fringe node in existing node
			if depth == lastIdx && isFringe(bits) {
				n.children.InsertAt(addr, &fringeNode{})
				return
			}

			// insert prefix as path-compressed leaf
			n.children.InsertAt(addr, &prefixNode{pfx})
			return
		}
		kid := n.children.MustGet(addr)

		// kid is node or leaf at addr
		switch kid := kid.(type) {
		case *liteNode:
			n = kid
			if depth == lastIdx && isFringe(bits) {
				// insert fringe prefix in existing node
				n.prefixes.Set(1)
				return
			}
			continue // descend down to next trie level

		case *fringeNode:
			// same fringe node, do nothing
			if depth == lastIdx && isFringe(bits) {
				return
			}

			// create new node
			// convert this fringeNode to fringe prefix in new node
			// insert new child at current child position (addr)
			// descend down, replace n with new child
			newNode := new(liteNode)
			newNode.prefixes.Set(1)

			n.children.InsertAt(addr, newNode)
			n = newNode
			continue

		case *prefixNode:
			// reached a path-compressed leaf, just a netip.Prefix
			if kid.Prefix == pfx {
				// already exists, nothing to do
				return
			}

			// create new node
			// push the leaf down
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(liteNode)
			newNode.insertAtDepth(kid.Prefix, depth+1)

			n.children.InsertAt(addr, newNode)
			n = newNode

			// insert fringe prefix addr/8
			if depth == lastIdx && lastBits == 0 {
				n.prefixes.Set(1)
				return
			}
			continue

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
		parent := parentStack[depth]
		childAddr := uint(childPath[depth])

		prefixCount := n.prefixes.Size()
		childCount := n.children.Len()

		switch {
		case prefixCount == 0 && childCount == 0:
			// just delete this empty node
			parent.children.DeleteAt(childAddr)

		case prefixCount == 0 && childCount == 1:
			// if single child is a path-compressed leaf, shift it up one level
			// and delete current node with this leaf
			if pfx, ok := n.children.Items[0].(*prefixNode); ok {
				parent.children.DeleteAt(childAddr)
				parent.insertAtDepth(pfx.Prefix, depth)
				break
			}

			// if single child is a fringeNode, shift it up one level
			// and delete current node with this fringe
			if _, ok := n.children.Items[0].(*fringeNode); ok {
				// get the fringeAddr back
				fringeAddr, _ := n.children.FirstSet()

				// build the stride path with child and fringeAddr
				path := stridePath{}
				copy(path[:], childPath)
				path[depth+1] = uint8(fringeAddr)

				// get the prefix back for this fringe path
				// a fringe has always /0, /8, /16, ... bits
				pfx := cidrFromFringe(path, depth+1, is4)

				// delete the this child slot
				parent.children.DeleteAt(childAddr)
				parent.insertAtDepth(pfx, 0)
				break
			}

		case prefixCount == 1 && childCount == 0:
			// make prefix from idx, shift leaf one level up
			// and override current node with new leaf
			idx, _ := n.prefixes.FirstSet()

			path := stridePath{}
			copy(path[:], childPath)
			pfx := cidrFromPath(path, depth+1, is4, idx)

			parent.children.DeleteAt(childAddr)
			parent.insertAtDepth(pfx, 0)
		}

		n = parent
	}
}

func liteOctetIdxAndBits(bits int) (lastIdx, lastBits int) {
	if bits == 0 {
		panic("global default route")
	}
	return (bits - 1) >> 3, bits % 8
}

func isFringe(bits int) bool {
	return bits%8 == 0
}

// cidrFromFringe TODO
func cidrFromFringe(path stridePath, depth int, is4 bool) netip.Prefix {
	// zero/mask the bytes after prefix bits
	clear(path[depth+1:])

	// make ip addr from octets
	var ip netip.Addr
	if is4 {
		ip = netip.AddrFrom4([4]byte(path[:4]))
	} else {
		ip = netip.AddrFrom16(path)
	}

	// calc bits with pathLen, pfxLen for fringe is always 0
	bits := (depth + 1) << 3

	// return a normalized prefix from ip/bits
	return netip.PrefixFrom(ip, bits)
}
