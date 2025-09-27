// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"cmp"
	"net/netip"

	"github.com/gaissmai/bart/internal/art"
)

// strideLen represents the byte stride length for the multibit trie.
// Each stride processes 8 bits (1 byte) at a time.
const strideLen = 8

// maxItems defines the maximum number of prefixes or children that can be stored in a single node.
// This corresponds to 256 possible values for an 8-bit stride.
const maxItems = 256

// maxTreeDepth represents the maximum depth of the trie structure.
// For IPv6 addresses, this allows up to 16 bytes of depth.
const maxTreeDepth = 16

// depthMask is used for bounds check elimination (BCE) when accessing depth-indexed arrays.
const depthMask = maxTreeDepth - 1

// stridePath represents a path through the trie, with a maximum depth of 16 octets for IPv6.
type stridePath [maxTreeDepth]uint8

// leafNode represents a path-compressed routing entry that stores both prefix and value.
// Leaf nodes are used when a prefix doesn't align with trie stride boundaries
// and needs to be stored as a compressed path to save memory.
type leafNode[V any] struct {
	value  V
	prefix netip.Prefix
}

// newLeafNode creates a new leaf node with the specified prefix and value.
func newLeafNode[V any](pfx netip.Prefix, val V) *leafNode[V] {
	return &leafNode[V]{prefix: pfx, value: val}
}

// fringeNode represents a path-compressed routing entry that stores only a value.
// The prefix is implicitly defined by the node's position in the trie.
// Fringe nodes are used for prefixes that align exactly with stride boundaries
// (/8, /16, /24, etc.) to save memory by not storing redundant prefix information.
type fringeNode[V any] struct {
	value V
}

// newFringeNode creates a new fringe node with the specified value.
func newFringeNode[V any](val V) *fringeNode[V] {
	return &fringeNode[V]{value: val}
}

// isFringe determines whether a prefix qualifies as a "fringe node" -
// that is, a special kind of path-compressed leaf inserted at the final
// possible trie level (depth == lastOctet).
//
// Both "leaves" and "fringes" are path-compressed terminal entries;
// the distinction lies in their position within the trie:
//
//   - A leaf is inserted at any intermediate level if no further stride
//     boundary matches (depth < lastOctet).
//
//   - A fringe is inserted at the last possible stride level
//     (depth == lastOctet) before a prefix would otherwise land
//     as a direct prefix (depth == lastOctet+1).
//
// Special property:
//   - A fringe acts as a default route for all downstream bit patterns
//     extending beyond its prefix.
//
// Examples:
//
//	e.g. prefix is addr/8, or addr/16, or ... addr/128
//	depth <  lastOctet :  a leaf, path-compressed
//	depth == lastOctet :  a fringe, path-compressed
//	depth == lastOctet+1: a prefix with octet/pfx == 0/0 => idx == 1, a strides default route
//
// Logic:
//   - A prefix qualifies as a fringe if:
//     depth == lastOctet && lastBits == 0
//     (i.e., aligned on stride boundary, /8, /16, ... /128 bits)
func isFringe(depth int, pfx netip.Prefix) bool {
	lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)
	return depth == lastOctetPlusOne-1 && lastBits == 0
}

// cmpIndexRank, sort indexes in prefix sort order.
func cmpIndexRank(aIdx, bIdx uint8) int {
	// convert idx [1..255] to prefix
	aOctet, aBits := art.IdxToPfx(aIdx)
	bOctet, bBits := art.IdxToPfx(bIdx)

	// cmp the prefixes, first by address and then by bits
	if aOctet == bOctet {
		return cmp.Compare(aBits, bBits)
	}
	return cmp.Compare(aOctet, bOctet)
}

// cidrFromPath reconstructs a CIDR prefix from a stride path, depth, and index.
// The prefix is determined by the node's position in the trie and the base index
// from the ART algorithm's complete binary tree representation.
//
// Parameters:
//   - path: The stride path through the trie
//   - depth: Current depth in the trie
//   - is4: True for IPv4 processing, false for IPv6
//   - idx: The base index from the prefix table
//
// Returns the reconstructed netip.Prefix.
func cidrFromPath(path stridePath, depth int, is4 bool, idx uint8) netip.Prefix {
	depth = depth & depthMask // BCE

	// retrieve the last octet and pfxLen
	octet, pfxLen := art.IdxToPfx(idx)

	// set byte in path at depth with last octet
	path[depth] = octet

	// canonicalize
	clear(path[depth+1:])

	// make ip addr from octets
	var ip netip.Addr
	if is4 {
		ip = netip.AddrFrom4([4]byte(path[:4]))
	} else {
		ip = netip.AddrFrom16(path)
	}

	// calc bits with pathLen and pfxLen
	bits := depth<<3 + int(pfxLen)

	// PrefixFrom does not allocate and does not mask off the host bits of ip.
	// With the clear(), the non-canonical bytes have already been removed.
	return netip.PrefixFrom(ip, bits)
}

// cidrForFringe reconstructs a CIDR prefix for a fringe node from the traversal path.
// Since fringe nodes don't store their prefix explicitly, it's derived entirely
// from the node's position in the trie.
//
// Parameters:
//   - octets: The path of octets leading to the fringe
//   - depth: Current depth in the trie
//   - is4: True for IPv4 processing, false for IPv6
//   - lastOctet: The final octet where the fringe is located
//
// Returns the reconstructed netip.Prefix for the fringe.
func cidrForFringe(octets []byte, depth int, is4 bool, lastOctet uint8) netip.Prefix {
	depth = depth & depthMask // BCE

	var path stridePath
	copy(path[:], octets)
	path[depth] = lastOctet

	// canonicalize, fringe bit boundaries are always a multiple of a byte
	clear(path[depth+1:])

	// make ip addr from octets
	var ip netip.Addr
	if is4 {
		ip = netip.AddrFrom4([4]byte(path[:4]))
	} else {
		ip = netip.AddrFrom16(path)
	}

	// it's a fringe, bits are always /8, /16, /24, ...
	bits := (depth + 1) << 3

	// PrefixFrom does not allocate and does not mask off the host bits of ip.
	// With the clear(), the non-canonical bytes have already been removed.
	return netip.PrefixFrom(ip, bits)
}
