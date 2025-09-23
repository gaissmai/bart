// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"cmp"
	"iter"
	"net/netip"
	"slices"

	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/lpm"
	"github.com/gaissmai/bart/internal/sparse"
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

// bartNode is a trie level bartNode in the multibit routing table.
//
// Each bartNode contains two conceptually different arrays:
//   - prefixes: representing routes, using a complete binary tree layout
//     driven by the baseIndex() function from the ART algorithm.
//   - children: holding subtries or path-compressed leaves/fringes with
//     a branching factor of 256 (8 bits per stride).
//
// Unlike the original ART, this implementation uses popcount-compressed sparse arrays
// instead of fixed-size arrays. Array slots are not pre-allocated; insertion
// and lookup rely on fast bitset operations and precomputed rank indexes.
//
// See doc/artlookup.pdf for the mapping mechanics and prefix tree details.
type bartNode[V any] struct {
	// prefixes stores routing entries (prefix -> value),
	// laid out as a complete binary tree using baseIndex().
	prefixes sparse.Array256[V]

	// children holds subnodes for the 256 possible next-hop paths
	// at this trie level (8-bit stride).
	//
	// Entries in children may be:
	//   - *bartNode[V]   -> internal child node for further traversal
	//   - *leafNode[V]   -> path-comp. node (depth < maxDepth - 1)
	//   - *fringeNode[V] -> path-comp. node (depth == maxDepth - 1, stride-aligned: /8, /16, ... /128)
	//
	// Note: Both *leafNode and *fringeNode entries are only created by path compression.
	// Prefixes that match exactly at the maximum trie depth (depth == maxDepth) are
	// never stored as children, but always directly in the prefixes array at that level.
	children sparse.Array256[any]
}

// isEmpty returns true if the node contains no routing entries (prefixes)
// and no child nodes. Empty nodes are candidates for compression or removal
// during trie optimization.
func (n *bartNode[V]) isEmpty() bool {
	if n == nil {
		return true
	}
	return n.prefixes.Len() == 0 && n.children.Len() == 0
}

// prefixCount returns the number of prefixes stored in this node.
func (n *bartNode[V]) prefixCount() int {
	return n.prefixes.Len()
}

// childCount returns the number of slots used in this node.
func (n *bartNode[V]) childCount() int {
	return n.children.Len()
}

// insertPrefix adds a routing entry at the specified index with the given value.
// It returns true if a prefix already existed at that index (indicating an update),
// false if this is a new insertion.
func (n *bartNode[V]) insertPrefix(idx uint8, val V) (exists bool) {
	return n.prefixes.InsertAt(idx, val)
}

// getPrefix retrieves the value associated with the prefix at the given index.
// Returns the value and true if found, or zero value and false if not present.
func (n *bartNode[V]) getPrefix(idx uint8) (val V, exists bool) {
	return n.prefixes.Get(idx)
}

// getIndices returns a slice of all index positions that have prefixes stored in this node.
// The indices correspond to positions in the complete binary tree representation used
// for prefix storage within the 8-bit stride.
//
//nolint:unused // used via nodeReader interface
func (n *bartNode[V]) getIndices() []uint8 {
	var buf [256]uint8
	return n.prefixes.AsSlice(&buf)
}

// allIndices returns an iterator over all prefix entries.
// Each iteration yields the prefix index (uint8) and its associated value (V).
//
//nolint:unused // used via nodeReader interface
func (n *bartNode[V]) allIndices() iter.Seq2[uint8, V] {
	return func(yield func(uint8, V) bool) {
		var buf [256]uint8
		for _, idx := range n.prefixes.AsSlice(&buf) {
			val := n.mustGetPrefix(idx)
			if !yield(idx, val) {
				return
			}
		}
	}
}

// mustGetPrefix retrieves the value at the specified index, panicking if not found.
// This method should only be used when the caller is certain the index exists.
func (n *bartNode[V]) mustGetPrefix(idx uint8) (val V) {
	return n.prefixes.MustGet(idx)
}

// deletePrefix removes the prefix at the specified index and returns its value.
// Returns the deleted value and true if the prefix existed, or zero value and false otherwise.
func (n *bartNode[V]) deletePrefix(idx uint8) (val V, exists bool) {
	return n.prefixes.DeleteAt(idx)
}

// insertChild adds a child node at the specified address (0-255).
// The child can be a *bartNode[V], *leafNode[V], or *fringeNode[V].
// Returns true if a child already existed at that address.
func (n *bartNode[V]) insertChild(addr uint8, child any) (exists bool) {
	return n.children.InsertAt(addr, child)
}

// getChild retrieves the child node at the specified address.
// Returns the child and true if found, or nil and false if not present.
func (n *bartNode[V]) getChild(addr uint8) (any, bool) {
	return n.children.Get(addr)
}

// getChildAddrs returns a slice of all addresses (0-255) that have children in this node.
// This is useful for iterating over all child nodes without checking every possible address.
//
//nolint:unused // used via nodeReader interface
func (n *bartNode[V]) getChildAddrs() []uint8 {
	var buf [256]uint8
	return n.children.AsSlice(&buf)
}

// allChildren returns an iterator over all child nodes.
// Each iteration yields the child's address (uint8) and the child node (any).
//
//nolint:unused // used via nodeReader interface
func (n *bartNode[V]) allChildren() iter.Seq2[uint8, any] {
	return func(yield func(addr uint8, child any) bool) {
		var buf [256]uint8
		addrs := n.children.AsSlice(&buf)
		for i, addr := range addrs {
			child := n.children.Items[i]
			if !yield(addr, child) {
				return
			}
		}
	}
}

// mustGetChild retrieves the child at the specified address, panicking if not found.
// This method should only be used when the caller is certain the child exists.
func (n *bartNode[V]) mustGetChild(addr uint8) any {
	return n.children.MustGet(addr)
}

// deleteChild removes the child node at the specified address.
// This operation is idempotent - removing a non-existent child is safe.
func (n *bartNode[V]) deleteChild(addr uint8) (exists bool) {
	_, exists = n.children.DeleteAt(addr)
	return exists
}

// contains returns true if an index (idx) has any matching longest-prefix
// in the current node’s prefix table.
//
// This function performs a presence check without retrieving the associated value.
// It is faster than a full lookup, as it only tests for intersection with the
// backtracking bitset for the given index.
//
// The prefix table is structured as a complete binary tree (CBT), and LPM testing
// is done via a bitset operation that maps the traversal path from the given index
// toward its possible ancestors.
func (n *bartNode[V]) contains(idx uint8) bool {
	return n.prefixes.Intersects(&lpm.LookupTbl[idx])
}

// lookupIdx performs a longest-prefix match (LPM) lookup for the given index (idx)
// within the 8-bit stride-based prefix table at this trie depth.
//
// The function returns the matched base index, associated value, and true if a
// matching prefix exists at this level; otherwise, ok is false.
//
// Internally, the prefix table is organized as a complete binary tree (CBT) indexed
// via the baseIndex function. Unlike the original ART algorithm, this implementation
// does not use an allotment-based approach. Instead, it performs CBT backtracking
// using a bitset-based operation with a precomputed backtracking pattern specific to idx.
func (n *bartNode[V]) lookupIdx(idx uint8) (top uint8, val V, ok bool) {
	// top is the idx of the longest-prefix-match
	if top, ok = n.prefixes.IntersectionTop(&lpm.LookupTbl[idx]); ok {
		return top, n.mustGetPrefix(top), true
	}
	return top, val, ok
}

// lookup is just a simple wrapper for lookupIdx.
func (n *bartNode[V]) lookup(idx uint8) (val V, ok bool) {
	_, val, ok = n.lookupIdx(idx)
	return val, ok
}

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

// eachLookupPrefix performs a hierarchical lookup of all matching prefixes
// in the current node’s 8-bit stride-based prefix table.
//
// The function walks up the trie-internal complete binary tree (CBT),
// testing each possible prefix length mask (in decreasing order of specificity),
// and invokes the yield function for every matching entry.
//
// The given idx refers to the position for this stride's prefix and is used
// to derive a backtracking path through the CBT by repeatedly halving the index.
// At each step, if a prefix exists in the table, its corresponding CIDR is
// reconstructed and yielded. If yield returns false, traversal stops early.
//
// This function is intended for internal use during supernet traversal and
// does not descend the trie further.
func (n *bartNode[V]) eachLookupPrefix(octets []byte, depth int, is4 bool, pfxIdx uint8, yield func(netip.Prefix, V) bool) (ok bool) {
	// path needed below more than once in loop
	var path stridePath
	copy(path[:], octets)

	for ; pfxIdx > 0; pfxIdx >>= 1 {
		if n.prefixes.Test(pfxIdx) {
			val := n.mustGetPrefix(pfxIdx)
			cidr := cidrFromPath(path, depth, is4, pfxIdx)

			if !yield(cidr, val) {
				return false
			}
		}
	}

	return true
}

// eachSubnet yields all prefix entries and child nodes covered by a given parent prefix,
// sorted in natural CIDR order, within the current node.
//
// The function iterates through all prefixes and children from the node’s stride tables.
// Only entries that fall within the address range defined by the parent prefix index (pfxIdx)
// are included. Matching entries are buffered, sorted, and passed through to the yield function.
//
// Child entries (nodes, leaves, fringes) that fall under the covered address range
// are processed recursively via allRecSorted to ensure sorted traversal.
//
// This function is intended for internal use by Subnets(), and it assumes the
// current node is positioned at the point in the trie corresponding to the parent prefix.
func (n *bartNode[V]) eachSubnet(octets []byte, depth int, is4 bool, pfxIdx uint8, yield func(netip.Prefix, V) bool) bool {
	// octets as array, needed below more than once
	var path stridePath
	copy(path[:], octets)

	pfxFirstAddr, pfxLastAddr := art.IdxToRange(pfxIdx)

	allCoveredIndices := make([]uint8, 0, maxItems)

	var buf [256]uint8
	for _, idx := range n.prefixes.AsSlice(&buf) {
		thisFirstAddr, thisLastAddr := art.IdxToRange(idx)

		if thisFirstAddr >= pfxFirstAddr && thisLastAddr <= pfxLastAddr {
			allCoveredIndices = append(allCoveredIndices, idx)
		}
	}

	// sort indices in CIDR sort order
	slices.SortFunc(allCoveredIndices, cmpIndexRank)

	// 2. collect all covered child addrs by prefix

	allCoveredChildAddrs := make([]uint8, 0, maxItems)
	for _, addr := range n.children.AsSlice(&buf) {
		if addr >= pfxFirstAddr && addr <= pfxLastAddr {
			allCoveredChildAddrs = append(allCoveredChildAddrs, addr)
		}
	}

	// 3. yield covered indices, pathcomp prefixes and childs in CIDR sort order

	addrCursor := 0

	// yield indices and childs in CIDR sort order
	for _, pfxIdx := range allCoveredIndices {
		pfxOctet, _ := art.IdxToPfx(pfxIdx)

		// yield all childs before idx
		for j := addrCursor; j < len(allCoveredChildAddrs); j++ {
			addr := allCoveredChildAddrs[j]
			if addr >= pfxOctet {
				break
			}

			// yield the node or leaf?
			switch kid := n.mustGetChild(addr).(type) {
			case *bartNode[V]:
				path[depth] = addr
				if !kid.allRecSorted(path, depth+1, is4, yield) {
					return false
				}

			case *leafNode[V]:
				if !yield(kid.prefix, kid.value) {
					return false
				}

			case *fringeNode[V]:
				fringePfx := cidrForFringe(path[:], depth, is4, addr)
				// callback for this fringe
				if !yield(fringePfx, kid.value) {
					// early exit
					return false
				}

			default:
				panic("logic error, wrong node type")
			}

			addrCursor++
		}

		// yield the prefix for this idx
		cidr := cidrFromPath(path, depth, is4, pfxIdx)
		// n.prefixes.Items[i] not possible after sorting allIndices
		if !yield(cidr, n.mustGetPrefix(pfxIdx)) {
			return false
		}
	}

	// yield the rest of leaves and nodes (rec-descent)
	for _, addr := range allCoveredChildAddrs[addrCursor:] {
		// yield the node or leaf?
		switch kid := n.mustGetChild(addr).(type) {
		case *bartNode[V]:
			path[depth] = addr
			if !kid.allRecSorted(path, depth+1, is4, yield) {
				return false
			}
		case *leafNode[V]:
			if !yield(kid.prefix, kid.value) {
				return false
			}
		case *fringeNode[V]:
			fringePfx := cidrForFringe(path[:], depth, is4, addr)
			// callback for this fringe
			if !yield(fringePfx, kid.value) {
				// early exit
				return false
			}

		default:
			panic("logic error, wrong node type")
		}
	}

	return true
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

	octet, pfxLen := art.IdxToPfx(idx)

	// set masked byte in path at depth
	path[depth] = octet

	// zero/mask the bytes after prefix bits
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

	// return a normalized prefix from ip/bits
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

	path := stridePath{}
	copy(path[:], octets[:depth+1])

	// replace last octet
	path[depth] = lastOctet

	// make ip addr from octets
	var ip netip.Addr
	if is4 {
		ip = netip.AddrFrom4([4]byte(path[:4]))
	} else {
		ip = netip.AddrFrom16(path)
	}

	// it's a fringe, bits are always /8, /16, /24, ...
	bits := (depth + 1) << 3

	// return a (normalized) prefix from ip/bits
	return netip.PrefixFrom(ip, bits)
}
