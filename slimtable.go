// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"sync"

	"github.com/gaissmai/bart/internal/art"
)

// Slim follows the BART design but with no payload.
// It is ideal for simple IP ACLs (access-control-lists) with plain
// true/false results with the smallest memory consumption.
type Slim struct {
	// used by -copylocks checker from `go vet`.
	_ [0]sync.Mutex

	root4 slimNode[any]
	root6 slimNode[any]

	// the number of prefixes in the routing table
	size4 int
	size6 int
}

// rootNodeByVersion, root node getter for ip version.
func (l *Slim) rootNodeByVersion(is4 bool) *slimNode[any] {
	if is4 {
		return &l.root4
	}
	return &l.root6
}

// Insert adds a prefix to the table (idempotent).
// If the prefix already exists, the operation is a no-op.
func (l *Slim) Insert(pfx netip.Prefix) {
	if !pfx.IsValid() {
		return
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	is4 := pfx.Addr().Is4()
	n := l.rootNodeByVersion(is4)

	if exists := n.insertAtDepth(pfx, 0); exists {
		return
	}

	// true insert, update size
	l.sizeUpdate(is4, 1)
}

// Delete removes the prefix and returns true if it was present, false otherwise.
func (l *Slim) Delete(pfx netip.Prefix) (found bool) {
	if !pfx.IsValid() {
		return
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)

	n := l.rootNodeByVersion(is4)

	// record the nodes on the path to the deleted node, needed to purge
	// and/or path compress nodes after the deletion of a prefix
	stack := [maxTreeDepth]*slimNode[any]{}

	// find the trie node
	for depth, octet := range octets {
		depth = depth & depthMask // BCE, Delete must be fast

		// push current node on stack for path recording
		stack[depth] = n

		// Last “octet” from prefix, update/insert prefix into node.
		// Note: For /32 and /128, depth never reaches lastOctetPlusOne (4/16),
		// so those are handled below via the fringe/leaf path.
		if depth == lastOctetPlusOne {
			// try to delete prefix in trie node
			_, found = n.deletePrefix(art.PfxToIdx(octet, lastBits))
			if !found {
				return
			}

			l.sizeUpdate(is4, -1)
			// remove now-empty nodes and re-path-compress upwards
			n.purgeAndCompress(stack[:depth], octets, is4)
			return true
		}

		if !n.children.Test(octet) {
			return
		}
		kid := n.mustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *slimNode[any]:
			n = kid // descend down to next trie level

		case *slimFringeNode:
			// if pfx is no fringe at this depth, fast exit
			if !isFringe(depth, pfx) {
				return
			}

			// pfx is fringe at depth, delete fringe
			n.deleteChild(octet)

			l.sizeUpdate(is4, -1)
			// remove now-empty nodes and re-path-compress upwards
			n.purgeAndCompress(stack[:depth], octets, is4)

			return true

		case *slimLeafNode:
			// Attention: pfx must be masked to be comparable!
			if kid.prefix != pfx {
				return
			}

			// prefix is equal leaf, delete leaf
			n.deleteChild(octet)

			l.sizeUpdate(is4, -1)
			// remove now-empty nodes and re-path-compress upwards
			n.purgeAndCompress(stack[:depth], octets, is4)

			return true

		default:
			panic("logic error, wrong node type")
		}
	}

	return
}

// Contains reports whether any stored prefix covers the given IP address.
// Returns false for invalid IP addresses.
//
// This performs longest-prefix matching and returns true if any prefix
// in the routing table contains the IP address.
func (l *Slim) Contains(ip netip.Addr) bool {
	// speed is top priority: no explicit test for ip.Isvalid
	// if ip is invalid, AsSlice() returns nil, Contains returns false.
	is4 := ip.Is4()
	n := l.rootNodeByVersion(is4)

	for _, octet := range ip.AsSlice() {
		// for contains, any lpm match is good enough, no backtracking needed
		if n.pfxCount != 0 && n.contains(art.OctetToIdx(octet)) {
			return true
		}

		// stop traversing?
		if !n.children.Test(octet) {
			return false
		}
		kid := n.mustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *slimNode[any]:
			n = kid // descend down to next trie level

		case *slimFringeNode:
			// fringe is the default-route for all possible octets below
			return true

		case *slimLeafNode:
			return kid.prefix.Contains(ip)

		default:
			panic("logic error, wrong node type")
		}
	}

	return false
}

// Size returns the prefix count.
func (l *Slim) Size() int {
	return l.size4 + l.size6
}

// Size4 returns the IPv4 prefix count.
func (l *Slim) Size4() int {
	return l.size4
}

// Size6 returns the IPv6 prefix count.
func (l *Slim) Size6() int {
	return l.size6
}

func (l *Slim) sizeUpdate(is4 bool, delta int) {
	if is4 {
		l.size4 += delta
		return
	}
	l.size6 += delta
}
