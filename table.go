// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Package bart provides a Balanced-Routing-Table (BART).
//
// BART is balanced in terms of memory usage and lookup time
// for the longest-prefix match.
//
// BART is a multibit-trie with fixed stride length of 8 bits,
// using a fast mapping function (taken from the ART algorithm) to map
// the 256 prefixes in each level node to form a complete-binary-tree.
//
// This complete binary tree is implemented with popcount compressed
// sparse arrays together with path compression. This reduces storage
// consumption by almost two orders of magnitude in comparison to ART,
// with even better lookup times for the longest prefix match.
//
// The BART algorithm is based on bit vectors and precalculated
// lookup tables. The search is performed entirely by fast,
// cache-friendly bitmask operations, which in modern CPUs are performed
// by advanced bit manipulation instruction sets (POPCNT, LZCNT, TZCNT).
//
// The algorithm was specially developed so that it can always work with a fixed
// length of 256 bits. This means that the bitsets fit well in a cache line and
// that loops in hot paths (4x uint64 = 256) can be accelerated by loop unrolling.
package bart

import (
	"iter"
	"net/netip"

	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/lpm"
)

// Table is an IPv4 and IPv6 routing table with payload V.
// The zero value is ready to use.
//
// The Table is safe for concurrent readers but not for concurrent readers
// and/or writers. Either the update operations must be protected by an
// external lock mechanism or the various ...Persist functions must be used
// which return a modified routing table by leaving the original unchanged
//
// A Table must not be copied by value.
type Table[V any] struct {
	// used by -copylocks checker from `go vet`.
	_ noCopy

	// the root nodes, implemented as popcount compressed multibit tries
	root4 node[V]
	root6 node[V]

	// the number of prefixes in the routing table
	size4 int
	size6 int
}

// rootNodeByVersion, root node getter for ip version.
func (t *Table[V]) rootNodeByVersion(is4 bool) *node[V] {
	if is4 {
		return &t.root4
	}

	return &t.root6
}

// lastOctetIdxAndBits, get last significant octet Idx and remaining bits
// for a given netip.Prefix.
//
// Split the IP prefixes at 8bit borders, count from 0.
//
//	BitPos: [0-7],[8-15],[16-23],[24-31],[32]
//	BitPos: [0-7],[8-15],[16-23],[24-31],[32-39],[40-47],[48-55],[56-63],...,[120-127],[128]
//
//			 0.0.0.0/0         => lastIdx:  0, lastBits: 0 (default route)
//			 0.0.0.0/7         => lastIdx:  0, lastBits: 7
//			 0.0.0.0/8         => lastIdx:  1, lastBits: 0 (possible fringe)
//			10.0.0.0/8         => lastIdx:  1, lastBits: 0 (possible fringe)
//			10.0.0.0/22        => lastIdx:  2, lastBits: 6
//			10.0.0.0/29        => lastIdx:  3, lastBits: 5
//			10.0.0.0/32        => lastIdx:  4, lastBits: 0 (possible fringe)
//
//			::/0               => lastIdx:  0, lastBits: 0 (default route)
//			::1/128            => lastIdx: 17, lastBits: 0 (possible fringe)
//			2001:db8::/42      => lastIdx:  5, lastBits: 2
//			2001:db8::/56      => lastIdx:  7, lastBits: 0 (possible fringe)
//
//	 /32 and /128 are special, they never form a new node, they are always inserted
//	 as path-compressed leaf, so the max-depth of the trie is still 4 or 16 (v4/v6)
//
// We are not splitting at /8, /16, ..., because this would mean that the
// first node would have 512 prefixes, bits from [0-8]. All remaining nodes
// would then only have 256 prefixes, e.g. bits from [9-16], [17-24], ...
// but the algorithm would then require a variable bitset.
// If you can commit to a fixed size of [4]uint64, then the algorithm will
// be much faster due to modern CPUs.
//
// One could also imagine special hardware, since the actual algorithm consists
// of a few standardized bitset operations on a fixed length of 256 bits.
func lastOctetIdxAndBits(bits int) (lastIdx, lastBits int) {
	return bits >> 3, bits & 7
}

// Insert adds a pfx to the tree, with given val.
// If pfx is already present in the tree, its value is set to val.
func (t *Table[V]) Insert(pfx netip.Prefix, val V) {
	if !pfx.IsValid() {
		return
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	is4 := pfx.Addr().Is4()
	n := t.rootNodeByVersion(is4)

	if n.insertAtDepth(pfx, val, 0) {
		return
	}

	// true insert, update size
	t.sizeUpdate(is4, 1)
}

// Update or set the value at pfx with a callback function.
// The callback function is called with (value, ok) and returns a new value.
//
// If the pfx does not already exist, it is set with the new value.
func (t *Table[V]) Update(pfx netip.Prefix, cb func(val V, ok bool) V) (newVal V) {
	var zero V

	if !pfx.IsValid() {
		return zero
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()
	octets := ip.AsSlice()
	lastIdx, lastBits := lastOctetIdxAndBits(bits)

	n := t.rootNodeByVersion(is4)

	// find the proper trie node to update prefix
	for depth, octet := range octets {
		// last octet from prefix, update/insert prefix into node
		if depth == lastIdx {
			newVal, exists := n.prefixes.UpdateAt(art.PfxToIdx(octet, lastBits), cb)
			if !exists {
				t.sizeUpdate(is4, 1)
			}
			return newVal
		}

		addr := uint(octet)

		// go down in tight loop to last octet
		if !n.children.Test(addr) {
			// insert prefix path compressed
			newVal := cb(zero, false)
			if isFringe(depth, bits) {
				n.children.InsertAt(addr, &fringeNode[V]{value: newVal})
			} else {
				n.children.InsertAt(addr, &leafNode[V]{prefix: pfx, value: newVal})
			}
			t.sizeUpdate(is4, 1)
			return newVal
		}
		kid := n.children.MustGet(addr)

		// kid is node or leaf or fringe at addr
		switch kid := kid.(type) {
		case *node[V]:
			n = kid
			continue // descend down to next trie level

		case *leafNode[V]:
			// update existing value if prefixes are equal
			if kid.prefix == pfx {
				kid.value = cb(kid.value, true)
				return kid.value
			}

			// create new node
			// push the leaf down
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(node[V])
			newNode.insertAtDepth(kid.prefix, kid.value, depth+1)

			n.children.InsertAt(addr, newNode)
			n = newNode

		case *fringeNode[V]:
			// update existing value if prefix is fringe
			if isFringe(depth, bits) {
				kid.value = cb(kid.value, true)
				return kid.value
			}

			// create new node
			// push the fringe down, it becomes a default route (idx=1)
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(node[V])
			newNode.prefixes.InsertAt(1, kid.value)

			n.children.InsertAt(addr, newNode)
			n = newNode

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
}

// Delete removes pfx from the tree, pfx does not have to be present.
func (t *Table[V]) Delete(pfx netip.Prefix) {
	_, _ = t.getAndDelete(pfx)
}

// GetAndDelete deletes the prefix and returns the associated payload for prefix and true,
// or the zero value and false if prefix is not set in the routing table.
func (t *Table[V]) GetAndDelete(pfx netip.Prefix) (val V, ok bool) {
	return t.getAndDelete(pfx)
}

func (t *Table[V]) getAndDelete(pfx netip.Prefix) (val V, exists bool) {
	if !pfx.IsValid() {
		return val, false
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()
	octets := ip.AsSlice()
	lastIdx, lastBits := lastOctetIdxAndBits(bits)

	n := t.rootNodeByVersion(is4)

	// record path to deleted node
	// needed to purge and/or path compress nodes after deletion
	stack := [maxTreeDepth]*node[V]{}

	// find the trie node
	for depth, octet := range octets {
		depth = depth & 0xf // BCE

		if depth > lastIdx {
			break
		}

		// push current node on stack for path recording
		stack[depth] = n

		if depth == lastIdx {
			// try to delete prefix in trie node
			val, exists = n.prefixes.DeleteAt(art.PfxToIdx(octet, lastBits))
			if !exists {
				return val, false
			}

			t.sizeUpdate(is4, -1)
			n.purgeAndCompress(stack[:depth], octets, is4)
			return val, exists
		}

		addr := uint(octet)
		if !n.children.Test(addr) {
			return val, false
		}
		kid := n.children.MustGet(addr)

		// kid is node or leaf or fringe at addr
		switch kid := kid.(type) {
		case *node[V]:
			n = kid
			continue // descend down to next trie level

		case *fringeNode[V]:
			// not this fringe
			if !isFringe(depth, bits) {
				return val, false
			}

			// prefix is fringe, delete fringe
			n.children.DeleteAt(addr)

			t.sizeUpdate(is4, -1)
			n.purgeAndCompress(stack[:depth], octets, is4)

			return kid.value, true

		case *leafNode[V]:
			// reached a path compressed prefix, stop traversing
			if kid.prefix != pfx {
				return val, false
			}

			// prefix is equal leaf, delete leaf
			n.children.DeleteAt(addr)

			t.sizeUpdate(is4, -1)
			n.purgeAndCompress(stack[:depth], octets, is4)

			return kid.value, true

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
}

// Get returns the associated payload for prefix and true, or false if
// prefix is not set in the routing table.
func (t *Table[V]) Get(pfx netip.Prefix) (val V, ok bool) {
	var zero V

	if !pfx.IsValid() {
		return zero, false
	}

	// canonicalize the prefix
	pfx = pfx.Masked()

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()

	n := t.rootNodeByVersion(is4)

	lastIdx, lastBits := lastOctetIdxAndBits(bits)

	octets := ip.AsSlice()

	// find the trie node
LOOP:
	for depth, octet := range octets {
		if depth == lastIdx {
			return n.prefixes.Get(art.PfxToIdx(octet, lastBits))
		}

		addr := uint(octet)
		if !n.children.Test(addr) {
			break LOOP
		}
		kid := n.children.MustGet(addr)

		// kid is node or leaf or fringe at addr
		switch kid := kid.(type) {
		case *node[V]:
			n = kid
			continue // descend down to next trie level

		case *fringeNode[V]:
			// reached a path compressed fringe, stop traversing
			if isFringe(depth, bits) {
				return kid.value, true
			}
			break LOOP

		case *leafNode[V]:
			// reached a path compressed prefix, stop traversing
			if kid.prefix == pfx {
				return kid.value, true
			}
			break LOOP

		default:
			panic("logic error, wrong node type")
		}
	}

	return zero, false
}

// Contains does a route lookup for IP and
// returns true if any route matched.
//
// Contains does not return the value nor the prefix of the matching item,
// but as a test against a black- or whitelist it's often sufficient
// and even few nanoseconds faster than [Table.Lookup].
func (t *Table[V]) Contains(ip netip.Addr) bool {
	// if ip is invalid, Is4() returns false and AsSlice() returns nil
	is4 := ip.Is4()
	n := t.rootNodeByVersion(is4)

	for _, octet := range ip.AsSlice() {
		addr := uint(octet)

		// for contains, any lpm match is good enough, no backtracking needed
		if n.prefixes.Len() != 0 && n.lpmTest(art.HostIdx(addr)) {
			return true
		}

		// stop traversing?
		if !n.children.Test(addr) {
			return false
		}
		kid := n.children.MustGet(addr)

		// kid is node or leaf or fringe at addr
		switch kid := kid.(type) {
		case *node[V]:
			n = kid
			continue // descend down to next trie level

		case *fringeNode[V]:
			// fringe is the default-route for all possible octets below
			return true

		case *leafNode[V]:
			return kid.prefix.Contains(ip)

		default:
			panic("logic error, wrong node type")
		}
	}

	// invalid IP
	return false
}

// Lookup does a route lookup (longest prefix match) for IP and
// returns the associated value and true, or false if no route matched.
func (t *Table[V]) Lookup(ip netip.Addr) (val V, ok bool) {
	if !ip.IsValid() {
		return val, false
	}

	is4 := ip.Is4()
	octets := ip.AsSlice()

	n := t.rootNodeByVersion(is4)

	// stack of the traversed nodes for fast backtracking, if needed
	stack := [maxTreeDepth]*node[V]{}

	// run variable, used after for loop
	var depth int
	var octet byte
	var addr uint

LOOP:
	// find leaf node
	for depth, octet = range octets {
		depth = depth & 0xf // BCE

		addr = uint(octet)

		// push current node on stack for fast backtracking
		stack[depth] = n

		// go down in tight loop to last octet
		if !n.children.Test(addr) {
			// no more nodes below octet
			break LOOP
		}
		kid := n.children.MustGet(addr)

		// kid is node or leaf or fringe at addr
		switch kid := kid.(type) {
		case *node[V]:
			n = kid
			continue // descend down to next trie level

		case *fringeNode[V]:
			// fringe is the default-route for all possible octets below
			return kid.value, true

		case *leafNode[V]:
			// reached a path compressed prefix, stop traversing
			if kid.prefix.Contains(ip) {
				return kid.value, true
			}
			break LOOP

		default:
			panic("logic error, wrong node type")
		}
	}

	// start backtracking, unwind the stack, bounds check eliminated
	for ; depth >= 0; depth-- {
		depth = depth & 0xf // BCE

		n = stack[depth]

		// longest prefix match, skip if node has no prefixes
		if n.prefixes.Len() != 0 {
			idx := art.HostIdx(uint(octets[depth]))
			// lpmGet(idx), manually inlined
			// --------------------------------------------------------------
			if topIdx, ok := n.prefixes.IntersectionTop(lpm.BackTrackingBitset(idx)); ok {
				return n.prefixes.MustGet(topIdx), true
			}
			// --------------------------------------------------------------
		}
	}

	// invalid IP
	return val, false
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
		return lpmPfx, val, false
	}

	// canonicalize the prefix
	pfx = pfx.Masked()

	ip := pfx.Addr()
	bits := pfx.Bits()
	is4 := ip.Is4()
	octets := ip.AsSlice()
	lastIdx, lastBits := lastOctetIdxAndBits(bits)

	n := t.rootNodeByVersion(is4)

	// record path to leaf node
	stack := [maxTreeDepth]*node[V]{}

	var depth int
	var octet byte
	var addr uint

LOOP:
	// find the last node on the octets path in the trie,
	for depth, octet = range octets {
		depth = depth & 0xf // BCE

		if depth > lastIdx {
			depth--
			break
		}
		// push current node on stack
		stack[depth] = n

		addr = uint(octet)

		// go down in tight loop to leaf node
		if !n.children.Test(addr) {
			break LOOP
		}
		kid := n.children.MustGet(addr)

		// kid is node or leaf or fringe at addr
		switch kid := kid.(type) {
		case *node[V]:
			n = kid
			continue LOOP // descend down to next trie level

		case *fringeNode[V]:
			// reached a path compressed fringe, stop traversing
			fringePfx := cidrForFringe(octets, depth, is4, addr)
			if fringePfx.Bits() > bits {
				break LOOP
			}
			return fringePfx, kid.value, true

		case *leafNode[V]:
			// reached a path compressed prefix, stop traversing
			if kid.prefix.Bits() > bits {
				break LOOP
			}

			if kid.prefix.Contains(ip) {
				return kid.prefix, kid.value, true
			}

			break LOOP

		default:
			panic("logic error, wrong node type")
		}
	}

	// start backtracking, unwind the stack
	for ; depth >= 0; depth-- {
		depth = depth & 0xf // BCE

		n = stack[depth]

		// longest prefix match, skip if node has no prefixes
		if n.prefixes.Len() == 0 {
			continue
		}

		// only the lastOctet may have a different prefix len
		// all others are just host routes
		var idx uint
		octet = octets[depth]
		if depth == lastIdx {
			idx = art.PfxToIdx(octet, lastBits)
		} else {
			idx = art.HostIdx(uint(octet))
		}

		// manually inlined: lpmGet(idx)
		if topIdx, ok := n.prefixes.IntersectionTop(lpm.BackTrackingBitset(idx)); ok {
			val = n.prefixes.MustGet(topIdx)

			// called from LookupPrefix
			if !withLPM {
				return netip.Prefix{}, val, ok
			}

			// called from LookupPrefixLPM

			// get the pfxLen from depth and top idx
			pfxLen := art.PfxLen(depth, topIdx)

			// calculate the lpmPfx from incoming ip and new mask
			lpmPfx, _ = ip.Prefix(pfxLen)
			return lpmPfx, val, ok
		}
	}

	return lpmPfx, val, false
}

// Supernets returns an iterator over all CIDRs covering pfx.
// The iteration is in reverse CIDR sort order, from longest-prefix-match to shortest-prefix-match.
func (t *Table[V]) Supernets(pfx netip.Prefix) iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		if !pfx.IsValid() {
			return
		}

		// canonicalize the prefix
		pfx = pfx.Masked()

		ip := pfx.Addr()
		is4 := ip.Is4()
		bits := pfx.Bits()
		octets := ip.AsSlice()
		lastIdx, lastBits := lastOctetIdxAndBits(bits)

		n := t.rootNodeByVersion(is4)

		// stack of the traversed nodes for reverse ordering of supernets
		stack := [maxTreeDepth]*node[V]{}

		// run variable, used after for loop
		var depth int
		var octet byte

		// find last node along this octet path
	LOOP:
		for depth, octet = range octets {
			depth = depth & 0xf // BCE

			if depth > lastIdx {
				depth--
				break
			}
			// push current node on stack
			stack[depth] = n

			addr := uint(octet)

			// descend down the trie
			if !n.children.Test(addr) {
				break LOOP
			}
			kid := n.children.MustGet(addr)

			// kid is node or leaf or fringe at addr
			switch kid := kid.(type) {
			case *node[V]:
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
				fringePfx := cidrForFringe(octets, depth, is4, addr)
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
			depth = depth & 0xf // BCE

			n = stack[depth]

			// micro benchmarking
			if n.prefixes.Len() == 0 {
				continue
			}

			// only the lastOctet may have a different prefix len
			// all others are just host routes
			pfxLen := strideLen
			if depth == lastIdx {
				pfxLen = lastBits
			}

			if !n.eachLookupPrefix(octets, depth, is4, pfxLen, yield) {
				// early exit
				return
			}
		}
	}
}

// Subnets returns an iterator over all CIDRs covered by pfx.
// The iteration is in natural CIDR sort order.
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
		bits := pfx.Bits()
		octets := ip.AsSlice()
		lastIdx, lastBits := lastOctetIdxAndBits(bits)

		n := t.rootNodeByVersion(is4)

		// find the trie node
		for depth, octet := range octets {
			if depth == lastIdx {
				_ = n.eachSubnet(octets, depth, is4, lastBits, yield)
				return
			}

			addr := uint(octet)
			if !n.children.Test(addr) {
				return
			}
			kid := n.children.MustGet(addr)

			// kid is node or leaf or fringe at addr
			switch kid := kid.(type) {
			case *node[V]:
				n = kid
				continue // descend down to next trie level

			case *leafNode[V]:
				if pfx.Bits() <= kid.prefix.Bits() && pfx.Overlaps(kid.prefix) {
					_ = yield(kid.prefix, kid.value)
				}
				return

			case *fringeNode[V]:
				fringePfx := cidrForFringe(octets, depth, is4, addr)
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

// OverlapsPrefix reports whether any IP in pfx is matched by a route in the table or vice versa.
func (t *Table[V]) OverlapsPrefix(pfx netip.Prefix) bool {
	if !pfx.IsValid() {
		return false
	}

	// canonicalize the prefix
	pfx = pfx.Masked()

	is4 := pfx.Addr().Is4()
	n := t.rootNodeByVersion(is4)

	return n.overlapsPrefixAtDepth(pfx, 0)
}

// Overlaps reports whether any IP in the table is matched by a route in the
// other table or vice versa.
func (t *Table[V]) Overlaps(o *Table[V]) bool {
	return t.Overlaps4(o) || t.Overlaps6(o)
}

// Overlaps4 reports whether any IPv4 in the table matches a route in the
// other table or vice versa.
func (t *Table[V]) Overlaps4(o *Table[V]) bool {
	if t.size4 == 0 || o.size4 == 0 {
		return false
	}
	return t.root4.overlaps(&o.root4, 0)
}

// Overlaps6 reports whether any IPv6 in the table matches a route in the
// other table or vice versa.
func (t *Table[V]) Overlaps6(o *Table[V]) bool {
	if t.size6 == 0 || o.size6 == 0 {
		return false
	}
	return t.root6.overlaps(&o.root6, 0)
}

// Union combines two tables, changing the receiver table.
// If there are duplicate entries, the payload of type V is shallow copied from the other table.
// If type V implements the [Cloner] interface, the values are cloned, see also [Table.Clone].
func (t *Table[V]) Union(o *Table[V]) {
	dup4 := t.root4.unionRec(&o.root4, 0)
	dup6 := t.root6.unionRec(&o.root6, 0)

	t.size4 += o.size4 - dup4
	t.size6 += o.size6 - dup6
}

// Cloner is an interface, if implemented by payload of type V the values are deeply copied
// during [Table.UpdatePersist], [Table.DeletePersist], [Table.Clone] and [Table.Union].
type Cloner[V any] interface {
	Clone() V
}

// Clone returns a copy of the routing table.
// The payload of type V is shallow copied, but if type V implements the [Cloner] interface,
// the values are cloned.
func (t *Table[V]) Clone() *Table[V] {
	if t == nil {
		return nil
	}

	c := new(Table[V])

	c.root4 = *t.root4.cloneRec()
	c.root6 = *t.root6.cloneRec()

	c.size4 = t.size4
	c.size6 = t.size6

	return c
}

func (t *Table[V]) sizeUpdate(is4 bool, n int) {
	if is4 {
		t.size4 += n
		return
	}
	t.size6 += n
}

// Size returns the prefix count.
func (t *Table[V]) Size() int {
	return t.size4 + t.size6
}

// Size4 returns the IPv4 prefix count.
func (t *Table[V]) Size4() int {
	return t.size4
}

// Size6 returns the IPv6 prefix count.
func (t *Table[V]) Size6() int {
	return t.size6
}

// All returns an iterator over key-value pairs from Table. The iteration order
// is not specified and is not guaranteed to be the same from one call to the
// next.
func (t *Table[V]) All() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root4.allRec(stridePath{}, 0, true, yield) && t.root6.allRec(stridePath{}, 0, false, yield)
	}
}

// All4 is like [Table.All] but only for the v4 routing table.
func (t *Table[V]) All4() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root4.allRec(stridePath{}, 0, true, yield)
	}
}

// All6 is like [Table.All] but only for the v6 routing table.
func (t *Table[V]) All6() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root6.allRec(stridePath{}, 0, false, yield)
	}
}

// AllSorted returns an iterator over key-value pairs from Table2 in natural CIDR sort order.
func (t *Table[V]) AllSorted() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root4.allRecSorted(stridePath{}, 0, true, yield) &&
			t.root6.allRecSorted(stridePath{}, 0, false, yield)
	}
}

// AllSorted4 is like [Table.AllSorted] but only for the v4 routing table.
func (t *Table[V]) AllSorted4() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root4.allRecSorted(stridePath{}, 0, true, yield)
	}
}

// AllSorted6 is like [Table.AllSorted] but only for the v6 routing table.
func (t *Table[V]) AllSorted6() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root6.allRecSorted(stridePath{}, 0, false, yield)
	}
}

// noCopy may be added to structs which must not be copied
// after the first use.
//
//	type My struct {
//		_ noCopy
//		A state
//		b foo
//	}
//
// See https://golang.org/issues/8005#issuecomment-190753527
// for details.
//
// Note that it must not be embedded, due to the Lock and Unlock methods.
type noCopy struct{}

// Lock is a no-op used by -copylocks checker from `go vet`.
func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}
