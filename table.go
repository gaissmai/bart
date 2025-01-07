// SPDX-License-Identifier: MIT

// package bart provides a Balanced-Routing-Table (BART).
//
// BART is balanced in terms of memory usage and lookup time
// for the longest-prefix match.
//
// BART is a multibit-trie with fixed stride length of 8 bits,
// using the _baseIndex_ function from the ART algorithm to
// build the complete-binary-tree (CBT) of prefixes for each stride.
//
// The CBT is implemented as a bit-vector, backtracking is just
// a matter of fast cache friendly bitmask operations.
//
// The routing table is implemented with popcount compressed sparse arrays
// together with path compression. This reduces storage consumption
// by almost two orders of magnitude in comparison to ART with
// similar lookup times for the longest prefix match.
package bart

import (
	"net/netip"
)

// Table is an IPv4 and IPv6 routing table with payload V.
// The zero value is ready to use.
//
// The Table is safe for concurrent readers but not for concurrent readers
// and/or writers.
type Table[V any] struct {
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

// Insert adds pfx to the tree, with given val.
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

	n := t.rootNodeByVersion(is4)

	significantIdx := (bits - 1) / strideLen
	significantBits := bits - (significantIdx * strideLen)

	octets := ipAsOctets(ip, is4)
	octets = octets[:significantIdx+1]

	// find the proper trie node to update prefix
	for depth, octet := range octets {
		// last octet from prefix, update/insert prefix into node
		if depth == significantIdx {
			newVal, exists := n.prefixes.UpdateAt(pfxToIdx(octet, significantBits), cb)
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
			n.children.InsertAt(addr, &leaf[V]{pfx, newVal})
			t.sizeUpdate(is4, 1)
			return newVal
		}

		// get node or leaf for octet
		switch k := n.children.MustGet(addr).(type) {
		case *node[V]:
			n = k
			continue
		case *leaf[V]:
			// update existing value if prefixes are equal
			if k.prefix == pfx {
				k.value = cb(k.value, true)
				return k.value
			}

			// create new node
			// push the leaf down
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			c := new(node[V])
			c.insertAtDepth(k.prefix, k.value, depth+1)

			n.children.InsertAt(addr, c)
			n = c
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

func (t *Table[V]) getAndDelete(pfx netip.Prefix) (val V, ok bool) {
	var zero V

	if !pfx.IsValid() {
		return zero, false
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()

	n := t.rootNodeByVersion(is4)

	significantIdx := (bits - 1) / strideLen
	significantBits := bits - (significantIdx * strideLen)

	octets := ipAsOctets(ip, is4)
	octets = octets[:significantIdx+1]

	// record path to deleted node
	// needed to purge and/or path compress nodes after deletion
	stack := [maxTreeDepth]*node[V]{}

	// find the trie node
LOOP:
	for depth, octet := range octets {
		// push current node on stack for path recording
		stack[depth] = n

		// try to delete prefix in trie node
		if depth == significantIdx {
			if val, ok = n.prefixes.DeleteAt(pfxToIdx(octet, significantBits)); ok {
				t.sizeUpdate(is4, -1)
				n.purgeAndCompress(stack[:depth], octets, is4)
				return val, ok
			}
		}

		addr := uint(octet)
		if !n.children.Test(addr) {
			break LOOP
		}

		// get the child: node or leaf
		switch k := n.children.MustGet(addr).(type) {
		case *node[V]:
			// descend down to next trie level
			n = k
			continue
		case *leaf[V]:
			// reached a path compressed prefix, stop traversing
			if k.prefix != pfx {
				break LOOP
			}

			// prefix is equal leaf, delete leaf
			n.children.DeleteAt(addr)

			t.sizeUpdate(is4, -1)
			n.purgeAndCompress(stack[:depth], octets, is4)

			return k.value, true
		}
	}

	return zero, false
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

	significantIdx := (bits - 1) / strideLen
	significantBits := bits - (significantIdx * strideLen)

	octets := ipAsOctets(ip, is4)
	octets = octets[:significantIdx+1]

	// find the trie node
LOOP:
	for depth, octet := range octets {
		if depth == significantIdx {
			return n.prefixes.Get(pfxToIdx(octet, significantBits))
		}

		addr := uint(octet)
		if !n.children.Test(addr) {
			break LOOP
		}

		// get the child: node or leaf
		switch k := n.children.MustGet(addr).(type) {
		case *node[V]:
			// descend down to next trie level
			n = k
			continue
		case *leaf[V]:
			// reached a path compressed prefix, stop traversing
			if k.prefix == pfx {
				return k.value, true
			}
			break LOOP
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
	if !ip.IsValid() {
		return false
	}

	is4 := ip.Is4()
	n := t.rootNodeByVersion(is4)

	octets := ipAsOctets(ip, is4)

	for _, octet := range octets {
		addr := uint(octet)

		// contains: any lpm match good enough, no backtracking needed
		if n.prefixes.Len() != 0 && n.lpmTest(hostIndex(addr)) {
			return true
		}

		if !n.children.Test(addr) {
			return false
		}

		// get node or leaf for octet
		switch k := n.children.MustGet(addr).(type) {
		case *node[V]:
			n = k
			continue
		case *leaf[V]:
			return k.prefix.Contains(ip)
		}
	}

	panic("unreachable")
}

// Lookup does a route lookup (longest prefix match) for IP and
// returns the associated value and true, or false if no route matched.
func (t *Table[V]) Lookup(ip netip.Addr) (val V, ok bool) {
	var zero V

	if !ip.IsValid() {
		return zero, false
	}

	is4 := ip.Is4()
	n := t.rootNodeByVersion(is4)

	octets := ipAsOctets(ip, is4)

	// stack of the traversed nodes for fast backtracking, if needed
	stack := [maxTreeDepth]*node[V]{}

	// run variable, used after for loop
	var depth int
	var octet byte
	var addr uint

LOOP:
	// find leaf node
	for depth, octet = range octets {
		addr = uint(octet)

		// push current node on stack for fast backtracking
		stack[depth] = n

		// go down in tight loop to last octet
		if !n.children.Test(addr) {
			// no more nodes below octet
			break LOOP
		}

		// get node or leaf for octet
		switch k := n.children.MustGet(addr).(type) {
		case *node[V]:
			// descend down to next trie level
			n = k
			continue
		case *leaf[V]:
			// reached a path compressed prefix, stop traversing
			if k.prefix.Contains(ip) {
				return k.value, true
			}
			break LOOP
		}
	}

	// start backtracking, unwind the stack
	for ; depth >= 0; depth-- {
		n = stack[depth]

		// longest prefix match, skip if node has no prefixes
		if n.prefixes.Len() != 0 {
			if _, val, ok = n.lpmGet(hostIndex(uint(octets[depth]))); ok {
				return val, ok
			}
		}
	}

	return zero, false
}

// LookupPrefix does a route lookup (longest prefix match) for pfx and
// returns the associated value and true, or false if no route matched.
func (t *Table[V]) LookupPrefix(pfx netip.Prefix) (val V, ok bool) {
	var zero V

	if !pfx.IsValid() {
		return zero, false
	}

	_, val, ok = t.lpmPrefix(pfx)

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
func (t *Table[V]) LookupPrefixLPM(pfx netip.Prefix) (lpm netip.Prefix, val V, ok bool) {
	if !pfx.IsValid() {
		return
	}

	return t.lpmPrefix(pfx)
}

// lpmPrefix, returns lpm, val and ok for a lpm match.
func (t *Table[V]) lpmPrefix(pfx netip.Prefix) (lpm netip.Prefix, val V, ok bool) {
	var zeroVal V
	var zeroPfx netip.Prefix

	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()

	n := t.rootNodeByVersion(is4)

	// see comment in insertAtDepth()
	significantIdx := (bits - 1) / strideLen
	significantBits := bits - (significantIdx * strideLen)

	octets := ipAsOctets(ip, is4)
	octets = octets[:significantIdx+1]

	// mask the prefix, pfx.Masked() is too complex and allocates
	octets[significantIdx] &= netMask(significantBits)

	// record path to leaf node
	stack := [maxTreeDepth]*node[V]{}

	var depth int
	var octet byte
	var addr uint

LOOP:
	// find the last node on the octets path in the trie
	for depth, octet = range octets {
		addr = uint(octet)

		// push current node on stack
		stack[depth] = n

		// go down in tight loop to leaf node
		if !n.children.Test(addr) {
			break LOOP
		}

		// get the child: node or leaf
		switch k := n.children.MustGet(addr).(type) {
		case *node[V]:
			// descend down to next trie level
			n = k
			continue
		case *leaf[V]:
			// reached a path compressed prefix, stop traversing
			if k.prefix.Contains(ip) && k.prefix.Bits() <= bits {
				return k.prefix, k.value, true
			}

			break LOOP
		}
	}

	// start backtracking, unwind the stack.
	for ; depth >= 0; depth-- {
		n = stack[depth]

		// longest prefix match, skip if node has no prefixes
		if n.prefixes.Len() != 0 {
			octet = octets[depth]

			// only the lastOctet may have a different prefix len
			// all others are just host routes
			var idx uint
			if depth == significantIdx {
				idx = pfxToIdx(octet, significantBits)
			} else {
				idx = hostIndex(uint(octet))
			}

			if baseIdx, val, ok := n.lpmGet(idx); ok {
				// calculate the bits from depth and idx
				bits := depth*strideLen + int(baseIdxLookupTbl[baseIdx].bits)

				// calculate the lpm from incoming ip and new mask
				lpm, _ = ip.Prefix(bits)
				return lpm, val, ok
			}
		}
	}

	return zeroPfx, zeroVal, false
}

// Supernets returns an iterator over all CIDRs covering pfx.
// The iteration is in reverse CIDR sort order, from longest-prefix-match to shortest-prefix-match.
func (t *Table[V]) Supernets(pfx netip.Prefix) func(yield func(netip.Prefix, V) bool) {
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

		n := t.rootNodeByVersion(is4)

		significantIdx := (bits - 1) / strideLen
		significantBits := bits - (significantIdx * strideLen)

		octets := ipAsOctets(ip, is4)
		octets = octets[:significantIdx+1]

		// stack of the traversed nodes for reverse ordering of supernets
		stack := [maxTreeDepth]*node[V]{}

		// run variable, used after for loop
		var depth int
		var octet byte

		// find last node along this octet path
	LOOP:
		for depth, octet = range octets {
			addr := uint(octet)

			// push current node on stack
			stack[depth] = n

			if !n.children.Test(addr) {
				break LOOP
			}

			switch k := n.children.MustGet(addr).(type) {
			case *node[V]:
				n = k
				continue LOOP
			case *leaf[V]:
				if k.prefix.Overlaps(pfx) && k.prefix.Bits() <= pfx.Bits() {
					if !yield(k.prefix, k.value) {
						// early exit
						return
					}
				}
				// end of trie along this octets path
				break LOOP
			}
		}

		// start backtracking, unwind the stack
		for ; depth >= 0; depth-- {
			n = stack[depth]

			// micro benchmarking
			if n.prefixes.Len() == 0 {
				continue
			}

			// only the lastOctet may have a different prefix len
			// all others are just host routes
			pfxLen := strideLen
			if depth == significantIdx {
				pfxLen = significantBits
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
func (t *Table[V]) Subnets(pfx netip.Prefix) func(yield func(netip.Prefix, V) bool) {
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

		n := t.rootNodeByVersion(is4)

		significantIdx := (bits - 1) / strideLen
		significantBits := bits - (significantIdx * strideLen)

		octets := ipAsOctets(ip, is4)
		octets = octets[:significantIdx+1]

		// find the trie node
		for depth, octet := range octets {
			if depth == significantIdx {
				_ = n.eachSubnet(octets, depth, is4, significantBits, yield)
				return
			}

			addr := uint(octet)
			if !n.children.Test(addr) {
				return
			}

			// node or leaf?
			switch k := n.children.MustGet(addr).(type) {
			case *node[V]:
				n = k
				continue
			case *leaf[V]:
				if pfx.Overlaps(k.prefix) && pfx.Bits() <= k.prefix.Bits() {
					_ = yield(k.prefix, k.value)
				}
				return
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

// Cloner, if implemented by payload of type V the values are deeply copied
// during [Table.Clone] and [Table.Union].
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
func (t *Table[V]) All() func(yield func(pfx netip.Prefix, val V) bool) {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root4.allRec(zeroPath, 0, true, yield) && t.root6.allRec(zeroPath, 0, false, yield)
	}
}

// All4, like [Table.All] but only for the v4 routing table.
func (t *Table[V]) All4() func(yield func(pfx netip.Prefix, val V) bool) {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root4.allRec(zeroPath, 0, true, yield)
	}
}

// All6, like [Table.All] but only for the v6 routing table.
func (t *Table[V]) All6() func(yield func(pfx netip.Prefix, val V) bool) {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root6.allRec(zeroPath, 0, false, yield)
	}
}

// AllSorted returns an iterator over key-value pairs from Table2 in natural CIDR sort order.
func (t *Table[V]) AllSorted() func(yield func(pfx netip.Prefix, val V) bool) {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root4.allRecSorted(zeroPath, 0, true, yield) &&
			t.root6.allRecSorted(zeroPath, 0, false, yield)
	}
}

// AllSorted4, like [Table.AllSorted] but only for the v4 routing table.
func (t *Table[V]) AllSorted4() func(yield func(pfx netip.Prefix, val V) bool) {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root4.allRecSorted(zeroPath, 0, true, yield)
	}
}

// AllSorted6, like [Table.AllSorted] but only for the v6 routing table.
func (t *Table[V]) AllSorted6() func(yield func(pfx netip.Prefix, val V) bool) {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root6.allRecSorted(zeroPath, 0, false, yield)
	}
}
