// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package nodes

import (
	"cmp"
	"net/netip"
	"reflect"

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

// LeafNode represents a path-compressed routing entry that stores both prefix and value.
// Leaf nodes are used when a prefix doesn't align with trie stride boundaries
// and needs to be stored as a compressed path to save memory.
type LeafNode[V any] struct {
	Value  V
	Prefix netip.Prefix
}

// NewLeafNode creates a new leaf node with the specified prefix and value.
func NewLeafNode[V any](pfx netip.Prefix, val V) *LeafNode[V] {
	return &LeafNode[V]{Prefix: pfx, Value: val}
}

// FringeNode represents a path-compressed routing entry that stores only a value.
// The prefix is implicitly defined by the node's position in the trie.
// Fringe nodes are used for prefixes that align exactly with stride boundaries
// (/8, /16, /24, etc.) to save memory by not storing redundant prefix information.
type FringeNode[V any] struct {
	Value V
}

// NewFringeNode creates a new fringe node with the specified value.
func NewFringeNode[V any](val V) *FringeNode[V] {
	return &FringeNode[V]{Value: val}
}

// IsFringe determines whether a prefix qualifies as a "fringe node" -
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
func IsFringe(depth int, pfx netip.Prefix) bool {
	lastOctetPlusOne, lastBits := LastOctetPlusOneAndLastBits(pfx)
	return depth == lastOctetPlusOne-1 && lastBits == 0
}

// cmpIndexRank, sort indexes in prefix sort order.
func CmpIndexRank(aIdx, bIdx uint8) int {
	// convert idx [1..255] to prefix
	aOctet, aBits := art.IdxToPfx(aIdx)
	bOctet, bBits := art.IdxToPfx(bIdx)

	// cmp the prefixes, first by address and then by bits
	if aOctet == bOctet {
		return cmp.Compare(aBits, bBits)
	}
	return cmp.Compare(aOctet, bOctet)
}

// CidrFromPath reconstructs a CIDR prefix from a stride path, depth, and index.
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
func CidrFromPath(path stridePath, depth int, is4 bool, idx uint8) netip.Prefix {
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

// CidrForFringe reconstructs a CIDR prefix for a fringe node from the traversal path.
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
func CidrForFringe(octets []byte, depth int, is4 bool, lastOctet uint8) netip.Prefix {
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

// LastOctetPlusOneAndLastBits returns the count of full 8‑bit strides (bits/8)
// and the leftover bits in the final stride (bits%8) for pfx.
//
// lastOctetPlusOne is the count of full 8‑bit strides (bits/8).
// lastBits is the remaining bit count in the final stride (bits%8),
//
// ATTENTION: Split the IP prefixes at 8bit borders, count from 0.
//
//	/7, /15, /23, /31, ..., /127
//
//	BitPos: [0-7],[8-15],[16-23],[24-31],[32]
//	BitPos: [0-7],[8-15],[16-23],[24-31],[32-39],[40-47],[48-55],[56-63],...,[120-127],[128]
//
//	0.0.0.0/0      => lastOctetPlusOne:  0, lastBits: 0 (default route)
//	0.0.0.0/7      => lastOctetPlusOne:  0, lastBits: 7
//	0.0.0.0/8      => lastOctetPlusOne:  1, lastBits: 0 (possible fringe)
//	10.0.0.0/8     => lastOctetPlusOne:  1, lastBits: 0 (possible fringe)
//	10.0.0.0/22    => lastOctetPlusOne:  2, lastBits: 6
//	10.0.0.0/29    => lastOctetPlusOne:  3, lastBits: 5
//	10.0.0.0/32    => lastOctetPlusOne:  4, lastBits: 0 (possible fringe)
//
//	::/0           => lastOctetPlusOne:  0, lastBits: 0 (default route)
//	::1/128        => lastOctetPlusOne: 16, lastBits: 0 (possible fringe)
//	2001:db8::/42  => lastOctetPlusOne:  5, lastBits: 2
//	2001:db8::/56  => lastOctetPlusOne:  7, lastBits: 0 (possible fringe)
//
//	/32 and /128 prefixes are special, they never form a new node,
//	At the end of the trie (IPv4: depth 4, IPv6: depth 16) they are always
//	inserted as a path‑compressed fringe.
//
// We are not splitting at /8, /16, ..., because this would mean that the
// first node would have 512 prefixes, 9 bits from [0-8]. All remaining nodes
// would then only have 8 bits from [9-16], [17-24], [25..32], ...
// but the algorithm would then require a variable length bitset.
//
// If you can commit to a fixed size of [4]uint64, then the algorithm is
// much faster due to modern CPUs.
//
// Perhaps a future Go version that supports SIMD instructions for the [4]uint64 vectors
// will make the algorithm even faster on suitable hardware.
func LastOctetPlusOneAndLastBits(pfx netip.Prefix) (lastOctetPlusOne int, lastBits uint8) {
	// lastOctetPlusOne:  range from 0..4 or 0..16 !ATTENTION: not 0..3 or 0..15
	// lastBits:          range from 0..7
	bits := pfx.Bits()

	//nolint:gosec  // G115: narrowing conversion is safe here (bits in [0..128])
	return bits >> 3, uint8(bits & 7)
}

// Equaler is a generic interface for types that can decide their own
// equality logic. It can be used to override the potentially expensive
// default comparison with [reflect.DeepEqual].
type Equaler[V any] interface {
	Equal(other V) bool
}

// equal compares two values of type V for equality.
// If V implements Equaler[V], that custom equality method is used.
// Otherwise, [reflect.DeepEqual] is used as a fallback.
func Equal[V any](v1, v2 V) bool {
	// you can't assert directly on a type parameter
	if v1, ok := any(v1).(Equaler[V]); ok {
		return v1.Equal(v2)
	}
	// fallback
	return reflect.DeepEqual(v1, v2)
}

// Cloner is an interface that enables deep cloning of values of type V.
// If a value implements Cloner[V], Table methods such as InsertPersist,
// ModifyPersist, DeletePersist, UnionPersist, Union and Clone will use
// its Clone method to perform deep copies.
type Cloner[V any] interface {
	Clone() V
}

// CloneFunc is a type definition for a function that takes a value of type V
// and returns the (possibly cloned) value of type V.
type CloneFunc[V any] func(V) V

// CloneFnFactory returns a cloneFunc.
// If V implements Cloner[V], the returned function should perform
// a deep copy using Clone(), otherwise it returns nil.
func CloneFnFactory[V any]() CloneFunc[V] {
	var zero V
	// you can't assert directly on a type parameter
	if _, ok := any(zero).(Cloner[V]); ok {
		return CloneVal[V]
	}
	return nil
}

// CloneVal returns a deep clone of val by calling its Clone method when
// val implements Cloner[V]. If val does not implement Cloner[V] or the
// asserted Cloner is nil, val is returned unchanged.
func CloneVal[V any](val V) V {
	// you can't assert directly on a type parameter
	c, ok := any(val).(Cloner[V])
	if !ok || c == nil {
		return val
	}
	return c.Clone()
}

// CopyVal just copies the value.
func CopyVal[V any](val V) V {
	return val
}

// CloneLeaf creates and returns a copy of the leafNode receiver.
// If cloneFn is nil, the value is copied directly without modification.
// Otherwise, cloneFn is applied to the value for deep cloning.
// The prefix field is always copied as is.
func (l *LeafNode[V]) CloneLeaf(cloneFn CloneFunc[V]) *LeafNode[V] {
	if cloneFn == nil {
		return &LeafNode[V]{Prefix: l.Prefix, Value: l.Value}
	}
	return &LeafNode[V]{Prefix: l.Prefix, Value: cloneFn(l.Value)}
}

// cloneFringe creates and returns a copy of the fringeNode receiver.
// If cloneFn is nil, the value is copied directly without modification.
// Otherwise, cloneFn is applied to the value for deep cloning.
func (l *FringeNode[V]) CloneFringe(cloneFn CloneFunc[V]) *FringeNode[V] {
	if cloneFn == nil {
		return &FringeNode[V]{Value: l.Value}
	}
	return &FringeNode[V]{Value: cloneFn(l.Value)}
}
