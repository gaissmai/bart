// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Usage: go generate -tags=ignore ./...
//go:generate ./scripts/generate-node-methods.sh
//go:build ignore

package bart

// ### GENERATE DELETE START ###

// stub code for generator types and methods
// useful for gopls during development, deleted during go generate

import (
	"net/netip"
	"slices"

	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/bitset"
)

type _NODE_TYPE[V any] struct {
	prefixes struct{ bitset.BitSet256 }
	children struct{ bitset.BitSet256 }
}

func (n *_NODE_TYPE[V]) prefixCount() (c int)           { return }
func (n *_NODE_TYPE[V]) childCount() (c int)            { return }
func (n *_NODE_TYPE[V]) mustGetPrefix(uint8) (val V)    { return }
func (n *_NODE_TYPE[V]) mustGetChild(uint8) (child any) { return }
func (n *_NODE_TYPE[V]) contains(uint8) (ok bool)       { return }

// ### GENERATE DELETE END ###

// allRec recursively traverses the trie starting at the current node,
// applying the provided yield function to every stored prefix and value.
//
// For each route entry (prefix and value), yield is invoked. If yield returns false,
// the traversal stops immediately, and false is propagated upwards,
// enabling early termination.
//
// The function handles all prefix entries in the current node, as well as any children -
// including sub-nodes, leaf nodes with full prefixes, and fringe nodes
// representing path-compressed prefixes. IP prefix reconstruction is performed on-the-fly
// from the current path and depth.
//
// The traversal order is not defined. This implementation favors simplicity
// and runtime efficiency over consistency of iteration sequence.
func (n *_NODE_TYPE[V]) allRec(path stridePath, depth int, is4 bool, yield func(netip.Prefix, V) bool) bool {
	var buf [256]uint8
	for _, idx := range n.prefixes.AsSlice(&buf) {
		cidr := cidrFromPath(path, depth, is4, idx)
		val := n.mustGetPrefix(idx)

		// callback for this prefix and val
		if !yield(cidr, val) {
			// early exit
			return false
		}
	}

	// for all children (nodes and leaves) in this node do ...
	for _, addr := range n.children.AsSlice(&buf) {
		anyKid := n.mustGetChild(addr)
		switch kid := anyKid.(type) {
		case *_NODE_TYPE[V]:
			// rec-descent with this node
			path[depth] = addr
			if !kid.allRec(path, depth+1, is4, yield) {
				// early exit
				return false
			}
		case *leafNode[V]:
			// callback for this leaf
			if !yield(kid.prefix, kid.value) {
				// early exit
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

// allRecSorted recursively traverses the trie in prefix-sorted order and applies
// the given yield function to each stored prefix and value.
//
// Unlike allRec, this implementation ensures that route entries are visited in
// canonical prefix sort order. To achieve this,
// both the prefixes and children of the current node are gathered, sorted,
// and then interleaved during traversal based on logical octet positioning.
//
// The function first sorts relevant entries by their prefix index and address value,
// using a comparison function that ranks prefixes according to their mask length and position.
// Then it walks the trie, always yielding child entries that fall before the current prefix,
// followed by the prefix itself. Remaining children are processed once all prefixes have been visited.
//
// Prefixes are reconstructed on-the-fly from the traversal path, and iteration includes all child types:
// inner nodes (recursive descent), leaf nodes, and fringe (compressed) prefixes.
//
// The order is stable and predictable, making the function suitable for use cases
// like table exports, comparisons or serialization.
//
// Parameters:
//   - path: the current traversal path through the trie
//   - depth: current depth in the trie (0-based)
//   - is4: true for IPv4 processing, false for IPv6
//   - yield: callback function invoked for each prefix/value pair
//
// Returns false if yield function requests early termination.
func (n *_NODE_TYPE[V]) allRecSorted(path stridePath, depth int, is4 bool, yield func(netip.Prefix, V) bool) bool {
	// get slice of all child octets, sorted by addr
	var childBuf [256]uint8
	allChildAddrs := n.children.AsSlice(&childBuf)

	// get slice of all indexes, sorted by idx
	var idxBuf [256]uint8
	allIndices := n.prefixes.AsSlice(&idxBuf)

	// sort indices in CIDR sort order
	slices.SortFunc(allIndices, cmpIndexRank)

	childCursor := 0

	// yield indices and children in CIDR sort order
	for _, pfxIdx := range allIndices {
		pfxOctet, _ := art.IdxToPfx(pfxIdx)

		// yield all children before idx
		for j := childCursor; j < len(allChildAddrs); j++ {
			childAddr := allChildAddrs[j]

			if childAddr >= pfxOctet {
				break
			}

			// yield the node (rec-descent) or leaf
			anyKid := n.mustGetChild(childAddr)
			switch kid := anyKid.(type) {
			case *_NODE_TYPE[V]:
				path[depth] = childAddr
				if !kid.allRecSorted(path, depth+1, is4, yield) {
					return false
				}
			case *leafNode[V]:
				if !yield(kid.prefix, kid.value) {
					return false
				}
			case *fringeNode[V]:
				fringePfx := cidrForFringe(path[:], depth, is4, childAddr)
				// callback for this fringe
				if !yield(fringePfx, kid.value) {
					// early exit
					return false
				}

			default:
				panic("logic error, wrong node type")
			}

			childCursor++
		}

		// yield the prefix for this idx
		cidr := cidrFromPath(path, depth, is4, pfxIdx)
		// n.prefixes.Items[i] not possible after sorting allIndices
		if !yield(cidr, n.mustGetPrefix(pfxIdx)) {
			return false
		}
	}

	// yield the rest of leaves and nodes (rec-descent)
	for j := childCursor; j < len(allChildAddrs); j++ {
		addr := allChildAddrs[j]
		anyKid := n.mustGetChild(addr)
		switch kid := anyKid.(type) {
		case *_NODE_TYPE[V]:
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
func (n *_NODE_TYPE[V]) eachLookupPrefix(octets []byte, depth int, is4 bool, pfxIdx uint8, yield func(netip.Prefix, V) bool) (ok bool) {
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
func (n *_NODE_TYPE[V]) eachSubnet(octets []byte, depth int, is4 bool, pfxIdx uint8, yield func(netip.Prefix, V) bool) bool {
	// octets as array, needed below more than once
	var path stridePath
	copy(path[:], octets)

	pfxFirstAddr, pfxLastAddr := art.IdxToRange(pfxIdx)

	allCoveredIndices := make([]uint8, 0, n.prefixCount())

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

	allCoveredChildAddrs := make([]uint8, 0, n.childCount())
	for _, addr := range n.children.AsSlice(&buf) {
		if addr >= pfxFirstAddr && addr <= pfxLastAddr {
			allCoveredChildAddrs = append(allCoveredChildAddrs, addr)
		}
	}

	// 3. yield covered indices, path-compressed prefixes
	//    and children in CIDR sort order

	addrCursor := 0

	// yield indices and children in CIDR sort order
	for _, pfxIdx := range allCoveredIndices {
		pfxOctet, _ := art.IdxToPfx(pfxIdx)

		// yield all children before idx
		for j := addrCursor; j < len(allCoveredChildAddrs); j++ {
			addr := allCoveredChildAddrs[j]
			if addr >= pfxOctet {
				break
			}

			// yield the node or leaf?
			switch kid := n.mustGetChild(addr).(type) {
			case *_NODE_TYPE[V]:
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
		case *_NODE_TYPE[V]:
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

func (n *_NODE_TYPE[V]) supernets(pfx netip.Prefix, yield func(netip.Prefix, V) bool) {
	ip := pfx.Addr()
	is4 := ip.Is4()
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)

	// stack of the traversed nodes for reverse ordering of supernets
	stack := [maxTreeDepth]*_NODE_TYPE[V]{}

	// run variable, used after for loop
	var depth int
	var octet byte

	// find last node along this octet path
LOOP:
	for depth, octet = range octets {
		// stepped one past the last stride of interest; back up to last and exit
		if depth > lastOctetPlusOne {
			depth--
			break
		}
		// push current node on stack
		stack[depth] = n

		// descend down the trie
		if !n.children.Test(octet) {
			break LOOP
		}
		kid := n.mustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *_NODE_TYPE[V]:
			n = kid
			continue LOOP // descend down to next trie level

		case *leafNode[V]:
			if kid.prefix.Bits() > pfx.Bits() {
				break LOOP
			}

			if kid.prefix.Overlaps(pfx) {
				if !yield(kid.prefix, kid.value) {
					// early exit
					return
				}
			}
			// end of trie along this octets path
			break LOOP

		case *fringeNode[V]:
			fringePfx := cidrForFringe(octets, depth, is4, octet)
			if fringePfx.Bits() > pfx.Bits() {
				break LOOP
			}

			if fringePfx.Overlaps(pfx) {
				if !yield(fringePfx, kid.value) {
					// early exit
					return
				}
			}
			// end of trie along this octets path
			break LOOP

		default:
			panic("logic error, wrong node type")
		}
	}

	// start backtracking, unwind the stack
	for ; depth >= 0; depth-- {
		n = stack[depth]

		// only the lastOctet may have a different prefix len
		// all others are just host routes
		var idx uint8
		octet = octets[depth]
		// Last “octet” from prefix, update/insert prefix into node.
		// Note: For /32 and /128, depth never reaches lastOctetPlusOne (4/16),
		// so those are handled below via the fringe/leaf path.
		if depth == lastOctetPlusOne {
			idx = art.PfxToIdx(octet, lastBits)
		} else {
			idx = art.OctetToIdx(octet)
		}

		// micro benchmarking, skip if there is no match
		if !n.contains(idx) {
			continue
		}

		// yield all the matching prefixes, not just the lpm
		if !n.eachLookupPrefix(octets, depth, is4, idx, yield) {
			// early exit
			return
		}
	}
}

func (n *_NODE_TYPE[V]) subnets(pfx netip.Prefix, yield func(netip.Prefix, V) bool) {
	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)

	// find the trie node
	for depth, octet := range octets {
		// Last “octet” from prefix, update/insert prefix into node.
		// Note: For /32 and /128, depth never reaches lastOctetPlusOne (4/16),
		// so those are handled below via the fringe/leaf path.
		if depth == lastOctetPlusOne {
			idx := art.PfxToIdx(octet, lastBits)
			n.eachSubnet(octets, depth, is4, idx, yield)
			return
		}

		if !n.children.Test(octet) {
			return
		}
		kid := n.mustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *_NODE_TYPE[V]:
			n = kid
			continue // descend down to next trie level

		case *leafNode[V]:
			if pfx.Bits() <= kid.prefix.Bits() && pfx.Overlaps(kid.prefix) {
				yield(kid.prefix, kid.value)
			}
			return // immediate return

		case *fringeNode[V]:
			fringePfx := cidrForFringe(octets, depth, is4, octet)
			if pfx.Bits() <= fringePfx.Bits() && pfx.Overlaps(fringePfx) {
				yield(fringePfx, kid.value)
			}
			return // immediate return

		default:
			panic("logic error, wrong node type")
		}
	}
}
