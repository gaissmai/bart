// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Package bart provides a high-performance Balanced Routing Table (BART).
//
// BART is balanced in terms of memory usage and lookup time
// for longest-prefix match (LPM) queries on IPv4 and IPv6 addresses.
//
// Internally, BART is implemented as a multibit trie with a fixed stride of 8 bits.
// Each level node uses a fast mapping function (adapted from D. E. Knuth's ART algorithm)
// to arrange all 256 possible prefixes in a complete binary tree structure.
//
// Instead of allocating full arrays, BART uses popcount-compressed sparse arrays
// and aggressive path compression. This results in up to 100x less memory usage
// than ART, while maintaining or even improving lookup speed.
//
// Lookup operations are entirely bit-vector based and rely on precomputed
// lookup tables. Because the data fits within 256-bit blocks, it allows
// for extremely efficient, cacheline-aligned access and is accelerated by
// CPU instructions such as POPCNT, LZCNT, and TZCNT.
//
// The fixed 256-bit representation (4x uint64) permits loop unrolling in hot paths,
// ensuring predictable and fast performance even under high routing load.
package bart

import (
	"iter"
	"net/netip"
	"sync"

	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/lpm"
)

// Table represents a thread-safe IPv4 and IPv6 routing table with payload V.
//
// The zero value is ready to use.
//
// The Table is safe for concurrent reads, but concurrent reads and writes
// must be externally synchronized. Mutation via Insert/Delete requires locks,
// or alternatively, use ...Persist methods which return a modified copy
// without altering the original table (copy-on-write).
//
// A Table must not be copied by value; always pass by pointer.
//
// Performance note: Do not pass IPv4-in-IPv6 addresses (e.g., ::ffff:192.0.2.1)
// as input. The methods do not perform automatic unmapping to avoid unnecessary
// overhead for the common case where native addresses are used.
// Users should unmap IPv4-in-IPv6 addresses to their native IPv4 form
// (e.g., 192.0.2.1) before calling these methods.
type Table[V any] struct {
	// used by -copylocks checker from `go vet`.
	_ [0]sync.Mutex

	// the root nodes, implemented as popcount compressed multibit tries
	root4 bartNode[V]
	root6 bartNode[V]

	// the number of prefixes in the routing table
	size4 int
	size6 int
}

// rootNodeByVersion, root node getter for ip version.
func (t *Table[V]) rootNodeByVersion(is4 bool) *bartNode[V] {
	if is4 {
		return &t.root4
	}
	return &t.root6
}

// lastOctetPlusOneAndLastBits returns the count of full 8‑bit strides (bits/8)
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
func lastOctetPlusOneAndLastBits(pfx netip.Prefix) (lastOctetPlusOne int, lastBits uint8) {
	// lastOctetPlusOne:  range from 0..4 or 0..16 !ATTENTION: not 0..3 or 0..15
	// lastBits:          range from 0..7
	bits := pfx.Bits()

	//nolint:gosec  // G115: narrowing conversion is safe here (bits in [0..128])
	return bits >> 3, uint8(bits & 7)
}

// Deprecated: use [Table.Modify] instead.
//
// Update or set the value at pfx with a callback function.
// The callback function is called with (value, found) and returns a new value.
//
// If the pfx does not already exist, it is set with the new value.
func (t *Table[V]) Update(pfx netip.Prefix, cb func(val V, found bool) V) (newVal V) {
	var zero V

	if !pfx.IsValid() {
		return newVal
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)

	n := t.rootNodeByVersion(is4)

	// find the proper trie node to update prefix
	for depth, octet := range octets {
		// Last “octet” from prefix, update/insert prefix into node.
		// Note: For /32 and /128, depth never reaches lastOctetPlusOne (4/16),
		// so those are handled below via the fringe/leaf path.
		if depth == lastOctetPlusOne {
			idx := art.PfxToIdx(octet, lastBits)

			oldVal, existed := n.getPrefix(idx)
			newVal := cb(oldVal, existed)
			n.insertPrefix(idx, newVal)

			if !existed {
				t.sizeUpdate(is4, 1)
			}
			return newVal
		}

		// go down in tight loop to last octet
		if !n.children.Test(octet) {
			// insert prefix path compressed
			newVal := cb(zero, false)
			if isFringe(depth, pfx) {
				n.insertChild(octet, newFringeNode(newVal))
			} else {
				n.insertChild(octet, newLeafNode(pfx, newVal))
			}
			t.sizeUpdate(is4, 1)
			return newVal
		}
		kid := n.mustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *bartNode[V]:
			n = kid // descend down to next trie level

		case *leafNode[V]:
			// update existing value if prefixes are equal
			if kid.prefix == pfx {
				kid.value = cb(kid.value, true)
				return kid.value
			}

			// create new node
			// push the leaf down
			// insert new child at current leaf position (octet)
			// descend down, replace n with new child
			newNode := new(bartNode[V])
			newNode.insert(kid.prefix, kid.value, depth+1)

			n.insertChild(octet, newNode)
			n = newNode

		case *fringeNode[V]:
			// update existing value if prefix is fringe
			if isFringe(depth, pfx) {
				kid.value = cb(kid.value, true)
				return kid.value
			}

			// create new node
			// push the fringe down, it becomes a default route (idx=1)
			// insert new child at current leaf position (octet)
			// descend down, replace n with new child
			newNode := new(bartNode[V])
			newNode.insertPrefix(1, kid.value)

			n.insertChild(octet, newNode)
			n = newNode

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
}

// Modify applies an insert, update, or delete operation for the value
// associated with the given prefix. The supplied callback decides the
// operation: it is called with the current value (or zero if not found)
// and a boolean indicating whether the prefix exists. The callback must
// return a new value and a delete flag: del == false inserts or updates,
// del == true deletes the entry if it exists (otherwise no-op). Modify
// returns the resulting value and a boolean indicating whether the
// entry was actually deleted.
//
// The operation is determined by the callback function, which is called with:
//
//	val:   the current value (or zero value if not found)
//	found: true if the prefix currently exists, false otherwise
//
// The callback returns:
//
//	val: the new value to insert or update (ignored if del == true)
//	del: true to delete the entry, false to insert or update
//
// Modify returns:
//
//	val:     the zero, old, or new value depending on the operation (see table)
//	deleted: true if the entry was deleted, false otherwise
//
// Summary:
//
//	Operation | cb-input        | cb-return       | Modify-return
//	---------------------------------------------------------------
//	No-op:    | (zero,   false) | (_,      true)  | (zero,   false)
//	Insert:   | (zero,   false) | (newVal, false) | (newVal, false)
//	Update:   | (oldVal, true)  | (newVal, false) | (oldVal, false)
//	Delete:   | (oldVal, true)  | (_,      true)  | (oldVal, true)
func (t *Table[V]) Modify(pfx netip.Prefix, cb func(val V, found bool) (_ V, del bool)) (_ V, deleted bool) {
	var zero V

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

	n := t.rootNodeByVersion(is4)

	// record the nodes on the path to the deleted node, needed to purge
	// and/or path compress nodes after the deletion of a prefix
	stack := [maxTreeDepth]*bartNode[V]{}

	// find the proper trie node to update prefix
	for depth, octet := range octets {
		// push current node on stack for path recording
		stack[depth] = n

		// Last “octet” from prefix, update/insert prefix into node.
		// Note: For /32 and /128, depth never reaches lastOctetPlusOne (4/16),
		// so those are handled below via the fringe/leaf path.
		if depth == lastOctetPlusOne {
			idx := art.PfxToIdx(octet, lastBits)

			oldVal, existed := n.getPrefix(idx)
			newVal, del := cb(oldVal, existed)

			// update size if necessary
			switch {
			case !existed && del: // no-op
				return zero, false

			case existed && del: // delete
				n.deletePrefix(idx)
				t.sizeUpdate(is4, -1)
				// remove now-empty nodes and re-path-compress upwards
				n.purgeAndCompress(stack[:depth], octets, is4)
				return oldVal, true

			case !existed: // insert
				n.insertPrefix(idx, newVal)
				t.sizeUpdate(is4, 1)
				return newVal, false

			case existed: // update
				n.insertPrefix(idx, newVal)
				return oldVal, false

			default:
				panic("unreachable")
			}

		}

		// go down in tight loop to last octet
		if !n.children.Test(octet) {
			// insert prefix path compressed

			newVal, del := cb(zero, false)
			if del {
				return zero, false // no-op
			}

			// insert
			if isFringe(depth, pfx) {
				n.insertChild(octet, newFringeNode(newVal))
			} else {
				n.insertChild(octet, newLeafNode(pfx, newVal))
			}

			t.sizeUpdate(is4, 1)
			return newVal, false
		}

		kid := n.mustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *bartNode[V]:
			n = kid // descend down to next trie level

		case *leafNode[V]:
			oldVal := kid.value

			// update existing value if prefixes are equal
			if kid.prefix == pfx {
				newVal, del := cb(oldVal, true)

				if !del {
					kid.value = newVal
					return oldVal, false // update
				}

				// delete
				n.deleteChild(octet)

				t.sizeUpdate(is4, -1)
				// remove now-empty nodes and re-path-compress upwards
				n.purgeAndCompress(stack[:depth], octets, is4)

				return oldVal, true
			}

			// create new node
			// push the leaf down
			// insert new child at current leaf position (octet)
			// descend down, replace n with new child
			newNode := new(bartNode[V])
			newNode.insert(kid.prefix, kid.value, depth+1)

			n.insertChild(octet, newNode)
			n = newNode

		case *fringeNode[V]:
			oldVal := kid.value

			// update existing value if prefix is fringe
			if isFringe(depth, pfx) {
				newVal, del := cb(kid.value, true)
				if !del {
					kid.value = newVal
					return oldVal, false // update
				}

				// delete
				n.deleteChild(octet)

				t.sizeUpdate(is4, -1)
				// remove now-empty nodes and re-path-compress upwards
				n.purgeAndCompress(stack[:depth], octets, is4)

				return oldVal, true
			}

			// create new node
			// push the fringe down, it becomes a default route (idx=1)
			// insert new child at current leaf position (octet)
			// descend down, replace n with new child
			newNode := new(bartNode[V])
			newNode.insertPrefix(1, kid.value)

			n.insertChild(octet, newNode)
			n = newNode

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
}

// Deprecated: use [Table.Delete] instead.
func (t *Table[V]) GetAndDelete(pfx netip.Prefix) (val V, found bool) {
	return t.Delete(pfx)
}

// Contains reports whether any stored prefix covers the given IP address.
// Returns false for invalid IP addresses.
//
// This performs longest-prefix matching and returns true if any prefix
// in the routing table contains the IP address, regardless of the associated value.
//
// It does not return the value nor the prefix of the matching item,
// but as a test against an allow-/deny-list it's often sufficient
// and even few nanoseconds faster than [Table.Lookup].
func (t *Table[V]) Contains(ip netip.Addr) bool {
	// speed is top priority: no explicit test for ip.Isvalid
	// if ip is invalid, AsSlice() returns nil, Contains returns false.
	is4 := ip.Is4()
	n := t.rootNodeByVersion(is4)

	for _, octet := range ip.AsSlice() {
		// for contains, any lpm match is good enough, no backtracking needed
		if n.prefixCount() != 0 && n.contains(art.OctetToIdx(octet)) {
			return true
		}

		// stop traversing?
		if !n.children.Test(octet) {
			return false
		}
		kid := n.mustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *bartNode[V]:
			n = kid // descend down to next trie level

		case *fringeNode[V]:
			// fringe is the default-route for all possible octets below
			return true

		case *leafNode[V]:
			return kid.prefix.Contains(ip)

		default:
			panic("logic error, wrong node type")
		}
	}

	return false
}

// Lookup performs longest-prefix matching for the given IP address and returns
// the associated value of the most specific matching prefix.
// Returns the zero value of V and false if no prefix matches.
// Returns false for invalid IP addresses.
//
// This is the core routing table operation used for packet forwarding decisions.
func (t *Table[V]) Lookup(ip netip.Addr) (val V, ok bool) {
	if !ip.IsValid() {
		return val, ok
	}

	is4 := ip.Is4()
	octets := ip.AsSlice()

	n := t.rootNodeByVersion(is4)

	// stack of the traversed nodes for fast backtracking, if needed
	stack := [maxTreeDepth]*bartNode[V]{}

	// run variable, used after for loop
	var depth int
	var octet byte

LOOP:
	// find leaf node
	for depth, octet = range octets {
		depth = depth & depthMask // BCE, Lookup must be fast

		// push current node on stack for fast backtracking
		stack[depth] = n

		// go down in tight loop to last octet
		if !n.children.Test(octet) {
			// no more nodes below octet
			break LOOP
		}
		kid := n.mustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *bartNode[V]:
			n = kid
			continue LOOP // descend down to next trie level

		case *fringeNode[V]:
			// fringe is the default-route for all possible nodes below
			return kid.value, true

		case *leafNode[V]:
			if kid.prefix.Contains(ip) {
				return kid.value, true
			}
			// reached a path compressed prefix, stop traversing
			break LOOP

		default:
			panic("logic error, wrong node type")
		}
	}

	// start backtracking, unwind the stack, bounds check eliminated
	for ; depth >= 0; depth-- {
		depth = depth & depthMask // BCE

		n = stack[depth]

		// longest prefix match, skip if node has no prefixes
		if n.prefixCount() != 0 {
			idx := art.OctetToIdx(octets[depth])
			// lookupIdx() manually inlined
			if lpmIdx, ok2 := n.prefixes.IntersectionTop(&lpm.LookupTbl[idx]); ok2 {
				return n.mustGetPrefix(lpmIdx), ok2
			}
		}
	}

	return val, ok
}

// LookupPrefix does a route lookup (longest prefix match) for pfx and
// returns the associated value and true, or false if no route matched.
func (t *Table[V]) LookupPrefix(pfx netip.Prefix) (val V, ok bool) {
	_, val, ok = t.lookupPrefixLPM(pfx, false)
	return val, ok
}

// LookupPrefixLPM is similar to [Table.LookupPrefix],
// but it returns the lpm prefix in addition to value,ok.
//
// This method is about 20-30% slower than LookupPrefix and should only
// be used if the matching lpm entry is also required for other reasons.
//
// If LookupPrefixLPM is to be used for IP address lookups,
// they must be converted to /32 or /128 prefixes.
func (t *Table[V]) LookupPrefixLPM(pfx netip.Prefix) (lpmPfx netip.Prefix, val V, ok bool) {
	return t.lookupPrefixLPM(pfx, true)
}

func (t *Table[V]) lookupPrefixLPM(pfx netip.Prefix, withLPM bool) (lpmPfx netip.Prefix, val V, ok bool) {
	if !pfx.IsValid() {
		return lpmPfx, val, ok
	}

	// canonicalize the prefix
	pfx = pfx.Masked()

	ip := pfx.Addr()
	bits := pfx.Bits()
	is4 := ip.Is4()
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)

	n := t.rootNodeByVersion(is4)

	// record path to leaf node
	stack := [maxTreeDepth]*bartNode[V]{}

	var depth int
	var octet byte

LOOP:
	// find the last node on the octets path in the trie,
	for depth, octet = range octets {
		depth = depth & depthMask // BCE

		// stepped one past the last stride of interest; back up to last and break
		if depth > lastOctetPlusOne {
			depth--
			break
		}
		// push current node on stack
		stack[depth] = n

		// go down in tight loop to leaf node
		if !n.children.Test(octet) {
			break LOOP
		}
		kid := n.mustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *bartNode[V]:
			n = kid
			continue LOOP // descend down to next trie level

		case *leafNode[V]:
			// reached a path compressed prefix, stop traversing
			if kid.prefix.Bits() > bits || !kid.prefix.Contains(ip) {
				break LOOP
			}
			return kid.prefix, kid.value, true

		case *fringeNode[V]:
			// the bits of the fringe are defined by the depth
			// maybe the LPM isn't needed, saves some cycles
			fringeBits := (depth + 1) << 3
			if fringeBits > bits {
				break LOOP
			}

			// the LPM isn't needed, saves some cycles
			if !withLPM {
				return netip.Prefix{}, kid.value, true
			}

			// sic, get the LPM prefix back, it costs some cycles!
			fringePfx := cidrForFringe(octets, depth, is4, octet)
			return fringePfx, kid.value, true

		default:
			panic("logic error, wrong node type")
		}
	}

	// start backtracking, unwind the stack
	for ; depth >= 0; depth-- {
		depth = depth & depthMask // BCE

		n = stack[depth]

		// longest prefix match, skip if node has no prefixes
		if n.prefixes.Len() == 0 {
			continue
		}

		// only the lastOctet may have a different prefix len
		// all others are just host routes
		var idx uint8
		octet = octets[depth]
		// Last “octet” from prefix, update/insert prefix into node.
		// Note: For /32 and /128, depth never reaches lastOctetPlusOne (4 or 16),
		// so those are handled below via the fringe/leaf path.
		if depth == lastOctetPlusOne {
			idx = art.PfxToIdx(octet, lastBits)
		} else {
			idx = art.OctetToIdx(octet)
		}

		// manually inlined: lookupIdx(idx)
		if topIdx, ok2 := n.prefixes.IntersectionTop(&lpm.LookupTbl[idx]); ok2 {
			val = n.mustGetPrefix(topIdx)

			// called from LookupPrefix
			if !withLPM {
				return netip.Prefix{}, val, ok2
			}

			// called from LookupPrefixLPM

			// get the bits from depth and top idx
			pfxBits := int(art.PfxBits(depth, topIdx))

			// calculate the lpmPfx from incoming ip and new mask
			lpmPfx, _ = ip.Prefix(pfxBits)
			return lpmPfx, val, ok2
		}
	}

	return lpmPfx, val, ok
}

// Supernets returns an iterator over all supernet routes that cover the given prefix pfx.
//
// The traversal searches both exact-length and shorter (less specific) prefixes that
// overlap or include pfx. Starting from the most specific position in the trie,
// it walks upward through parent nodes and yields any matching entries found at each level.
//
// The iteration order is reverse-CIDR: from longest prefix match (LPM) towards
// least-specific routes.
//
// The search is protocol-specific (IPv4 or IPv6) and stops immediately if the yield
// function returns false. If pfx is invalid, the function silently returns.
//
// This can be used to enumerate all covering supernet routes in routing-based
// policy engines, diagnostics tools, or fallback resolution logic.
//
// Example:
//
//	for supernet, val := range table.Supernets(netip.MustParsePrefix("192.0.2.128/25")) {
//	    fmt.Println("Matched covering route:", supernet, "->", val)
//	}
func (t *Table[V]) Supernets(pfx netip.Prefix) iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		if !pfx.IsValid() {
			return
		}

		// canonicalize the prefix
		pfx = pfx.Masked()

		ip := pfx.Addr()
		is4 := ip.Is4()
		octets := ip.AsSlice()
		lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)

		n := t.rootNodeByVersion(is4)

		// stack of the traversed nodes for reverse ordering of supernets
		stack := [maxTreeDepth]*bartNode[V]{}

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
			case *bartNode[V]:
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
}

// Subnets returns an iterator over all prefix–value pairs in the routing table
// that are fully contained within the given prefix pfx.
//
// Entries are returned in CIDR sort order.
//
// Example:
//
//	for sub, val := range table.Subnets(netip.MustParsePrefix("10.0.0.0/8")) {
//	    fmt.Println("Covered:", sub, "->", val)
//	}
func (t *Table[V]) Subnets(pfx netip.Prefix) iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		if !pfx.IsValid() {
			return
		}

		// canonicalize the prefix
		pfx = pfx.Masked()

		// values derived from pfx
		ip := pfx.Addr()
		is4 := ip.Is4()
		octets := ip.AsSlice()
		lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)

		n := t.rootNodeByVersion(is4)

		// find the trie node
		for depth, octet := range octets {
			// Last “octet” from prefix, update/insert prefix into node.
			// Note: For /32 and /128, depth never reaches lastOctetPlusOne (4/16),
			// so those are handled below via the fringe/leaf path.
			if depth == lastOctetPlusOne {
				idx := art.PfxToIdx(octet, lastBits)
				_ = n.eachSubnet(octets, depth, is4, idx, yield)
				return
			}

			if !n.children.Test(octet) {
				return
			}
			kid := n.mustGetChild(octet)

			// kid is node or leaf or fringe at octet
			switch kid := kid.(type) {
			case *bartNode[V]:
				n = kid
				continue // descend down to next trie level

			case *leafNode[V]:
				if pfx.Bits() <= kid.prefix.Bits() && pfx.Overlaps(kid.prefix) {
					_ = yield(kid.prefix, kid.value)
				}
				return

			case *fringeNode[V]:
				fringePfx := cidrForFringe(octets, depth, is4, octet)
				if pfx.Bits() <= fringePfx.Bits() && pfx.Overlaps(fringePfx) {
					_ = yield(fringePfx, kid.value)
				}
				return

			default:
				panic("logic error, wrong node type")
			}
		}
	}
}
