// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"iter"
	"net/netip"
	"sync"

	"github.com/gaissmai/bart/internal/art"
)

// Fast follows the original ART design by Knuth in using fixed
// 256-slot arrays at each level.
// In contrast to the original, this variant introduces a new form of path
// compression. This keeps memory usage within a reasonable range while
// preserving the high lookup speed of the pure array-based ART algorithm.
//
// Both [bart.Fast] and [bart.Table] use the same path compression, but they
// differ in how levels are represented:
//
//   - [bart.Fast]:   uncompressed  fixed level arrays + path compression
//   - [bart.Table]: popcount-compressed level arrays + path compression
//
// As a result:
//   - [bart.Fast] sacrifices memory efficiency to achieve 2x higher speed
//   - [bart.Table] minimizes memory consumption as much as possible
//
// Which variant is preferable depends on the use case: [bart.Fast] is most
// beneficial when maximum speed for longest-prefix-match is the top priority,
// for example in a Forwarding Information Base (FIB).
//
// For the full Internet routing table, the [bart.Fast] structure alone requires
// about 250 MB of memory, with additional space needed for payload such as
// next hop, interface, and further attributes.
//
// Performance note: Do not pass IPv4-in-IPv6 addresses (e.g., ::ffff:192.0.2.1)
// as input. The methods do not perform automatic unmapping to avoid unnecessary
// overhead for the common case where native addresses are used.
// Users should unmap IPv4-in-IPv6 addresses to their native IPv4 form
// (e.g., 192.0.2.1) before calling these methods.
type Fast[V any] struct {
	// used by -copylocks checker from `go vet`.
	_ [0]sync.Mutex

	// the root nodes are fast nodes with fixed size arrays
	root4 fastNode[V]
	root6 fastNode[V]

	// the number of prefixes in the routing table
	size4 int
	size6 int
}

// rootNodeByVersion, root node getter for ip version and trie levels.
func (f *Fast[V]) rootNodeByVersion(is4 bool) *fastNode[V] {
	if is4 {
		return &f.root4
	}
	return &f.root6
}

// Modify applies an insert, update, or delete operation for the value
// associated with the given prefix. The supplied callback decides the
// operation.
// It receives the current value (if the prefix exists) and a boolean indicating
// existence, then returns the new value and a deletion flag.
//
// Returns the previous value (for updates/deletes) or new value (for inserts),
// and a boolean indicating whether a deletion occurred.
//
// If the prefix doesn't exist and the callback returns del=true, no operation is performed.
// The prefix is automatically canonicalized using pfx.Masked().
//
// Its value semantics are identical to [Table.Modify].
func (f *Fast[V]) Modify(pfx netip.Prefix, cb func(val V, found bool) (_ V, del bool)) (_ V, deleted bool) {
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

	n := f.rootNodeByVersion(is4)

	// record the nodes on the path to the deleted node, needed to purge
	// and/or path compress nodes after the deletion of a prefix
	stack := [maxTreeDepth]*fastNode[V]{}

	// find the trie node
	for depth, octet := range octets {
		depth = depth & depthMask // BCE

		// push current node on stack for path recording
		stack[depth] = n

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
				f.sizeUpdate(is4, -1)
				n.purgeAndCompress(stack[:depth], octets, is4)
				return oldVal, true

			case !existed: // insert
				n.insertPrefix(idx, newVal)
				f.sizeUpdate(is4, 1)
				return newVal, false

			case existed: // update
				n.insertPrefix(idx, newVal)
				return oldVal, false

			default:
				panic("unreachable")
			}
		}

		kidAny, exists := n.getChild(octet)
		if !exists {
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

			f.sizeUpdate(is4, 1)
			return newVal, false
		}

		// kid is node or leaf or fringe at octet
		switch kid := kidAny.(type) {
		case *fastNode[V]:
			n = kid // descend down to next trie level

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

				f.sizeUpdate(is4, -1)
				n.purgeAndCompress(stack[:depth], octets, is4)

				return oldVal, true // delete
			}

			// create new node
			// push the fringe down, it becomes a default route (idx=1)
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(fastNode[V])
			_ = newNode.insertPrefix(1, kid.value)
			_ = n.insertChild(octet, newNode)
			n = newNode

		case *leafNode[V]:
			oldVal := kid.value

			// update existing value if prefixes are equal
			if kid.prefix == pfx {
				newVal, del := cb(kid.value, true)
				if !del {
					kid.value = newVal
					return oldVal, false // update
				}

				// delete
				n.deleteChild(octet)

				f.sizeUpdate(is4, -1)
				n.purgeAndCompress(stack[:depth], octets, is4)

				return oldVal, true // delete
			}

			// create new node
			// push the leaf down
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(fastNode[V])
			_ = newNode.insert(kid.prefix, kid.value, depth+1)
			_ = n.insertChild(octet, newNode)
			n = newNode

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
// in the routing table contains the IP address, regardless of the associated value.
//
// Its semantics are identical to [Table.Contains].
func (f *Fast[V]) Contains(ip netip.Addr) bool {
	// speed is top priority: no explicit test for ip.Isvalid
	// if ip is invalid, AsSlice() returns nil, Contains returns false.
	is4 := ip.Is4()
	n := f.rootNodeByVersion(is4)

	for _, octet := range ip.AsSlice() {
		if n.contains(art.OctetToIdx(octet)) {
			return true
		}

		kidAny, exists := n.getChild(octet)
		if !exists {
			// no next node
			return false
		}

		// kid is node or leaf or fringe at octet
		switch kid := kidAny.(type) {
		case *fastNode[V]:
			n = kid // continue

		case *fringeNode[V]:
			// fringe is the default-route for all possible octets below
			return true

		case *leafNode[V]:
			// due to path compression, the octet path between
			// leaf and prefix may diverge
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
//
// Its semantics are identical to [Table.Lookup].
func (f *Fast[V]) Lookup(ip netip.Addr) (val V, ok bool) {
	if !ip.IsValid() {
		return val, ok
	}

	is4 := ip.Is4()
	n := f.rootNodeByVersion(is4)

	for _, octet := range ip.AsSlice() {
		// save the current best LPM val, lookup is cheap for fastNode
		if bestLPM, ok2 := n.lookup(art.OctetToIdx(octet)); ok2 {
			val = bestLPM
			ok = ok2
		}

		kidAny, exists := n.getChild(octet)
		if !exists {
			// no next node
			return val, ok
		}

		// next kid is fast, fringe or leaf node.
		switch kid := kidAny.(type) {
		case *fastNode[V]:
			n = kid

		case *fringeNode[V]:
			// fringe is the default-route for all possible nodes below
			return kid.value, true

		case *leafNode[V]:
			// due to path compression, the octet path between
			// leaf and prefix may diverge
			if kid.prefix.Contains(ip) {
				return kid.value, true
			}
			// maybe there is a current best value from upper levels
			return val, ok

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
}

// LookupPrefix does a route lookup (longest prefix match) for pfx and
// returns the associated value and true, or false if no route matched.
func (f *Fast[V]) LookupPrefix(pfx netip.Prefix) (val V, ok bool) {
	_, val, ok = f.lookupPrefixLPM(pfx, false)
	return val, ok
}

// LookupPrefixLPM is similar to [Fast.LookupPrefix],
// but it returns the lpm prefix in addition to value,ok.
//
// This method is about 20-30% slower than LookupPrefix and should only
// be used if the matching lpm entry is also required for other reasons.
//
// If LookupPrefixLPM is to be used for IP address lookups,
// they must be converted to /32 or /128 prefixes.
func (f *Fast[V]) LookupPrefixLPM(pfx netip.Prefix) (lpmPfx netip.Prefix, val V, ok bool) {
	return f.lookupPrefixLPM(pfx, true)
}

func (f *Fast[V]) lookupPrefixLPM(pfx netip.Prefix, withLPM bool) (lpmPfx netip.Prefix, val V, ok bool) {
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

	n := f.rootNodeByVersion(is4)

	// record path to leaf node
	stack := [maxTreeDepth]*fastNode[V]{}

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
		kidAny, exists := n.getChild(octet)
		if !exists {
			break LOOP
		}

		// kid is node or leaf or fringe at octet
		switch kid := kidAny.(type) {
		case *fastNode[V]:
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
		if n.pfxCount == 0 {
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

		switch withLPM {
		case false: // LookupPrefix
			if val, ok = n.lookup(idx); ok {
				return netip.Prefix{}, val, ok
			}

		case true: // LookupPrefixLPM
			if lpmIdx, val2, ok2 := n.lookupIdx(idx); ok2 {
				// get the bits from depth and lpmIdx
				pfxBits := int(art.PfxBits(depth, lpmIdx))

				// calculate the lpmPfx from incoming ip and new mask
				// netip.Addr.Prefix already canonicalize the prefix
				lpmPfx, _ = ip.Prefix(pfxBits)
				return lpmPfx, val2, ok2
			}
		}
		// continue rewinding the stack
	}

	return lpmPfx, val, ok
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
func (f *Fast[V]) Subnets(pfx netip.Prefix) iter.Seq2[netip.Prefix, V] {
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

		n := f.rootNodeByVersion(is4)

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
			case *fastNode[V]:
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
