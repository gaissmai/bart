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
// The CBT is implemented as a bitvector, backtracking is just
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

// Table2 is an IPv4 and IPv6 routing table with payload V.
// The zero value is ready to use.
//
// The Table is safe for concurrent readers but not for concurrent readers
// and/or writers.
type Table2[V any] struct {
	// the root nodes, implemented as popcount compressed multibit tries
	root4 node2[V]
	root6 node2[V]

	// the number of prefixes in the routing table
	size4 int
	size6 int
}

// rootNodeByVersion, root node getter for ip version.
func (t *Table2[V]) rootNodeByVersion(is4 bool) *node2[V] {
	if is4 {
		return &t.root4
	}

	return &t.root6
}

// Insert adds pfx to the tree, with given val.
// If pfx is already present in the tree, its value is set to val.
//
// This is the path compressed version of Insert.
func (t *Table2[V]) Insert(pfx netip.Prefix, val V) {
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

	// true insert, no override
	t.sizeUpdate(is4, 1)
}

// Update or set the value at pfx with a callback function.
// The callback function is called with (value, ok) and returns a new value.
//
// If the pfx does not already exist, it is set with the new value.
func (t *Table2[V]) Update(pfx netip.Prefix, cb func(val V, ok bool) V) (newVal V) {
	var zero V

	if !pfx.IsValid() {
		return zero
	}

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()

	n := t.rootNodeByVersion(is4)

	octets := ip.AsSlice()
	sigOctetIdx := (bits - 1) / strideLen
	sigOctet := octets[sigOctetIdx]
	sigOctetBits := bits - (sigOctetIdx * strideLen)

	// mask the prefix
	sigOctet &= netMask(sigOctetBits)
	octets[sigOctetIdx] = sigOctet

	// find the proper trie node to update prefix
	for depth, octet := range octets[:sigOctetIdx] {
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
		case *node2[V]:
			// go next level down
			n = k
		case *leaf[V]:
			// update existing value if prefixes are equal
			if k.prefix == pfx {
				k.value = cb(k.value, true)
				return k.value
			}

			// create new node
			// push the leaf down
			// insert new child at cureent leaf position (addr)
			// descend down, replace n with new child
			c := new(node2[V])
			c.insertAtDepth(k.prefix, k.value, depth+1)

			n.children.InsertAt(addr, c)
			n = c
			// continue to next octet
		}
	}

	// update/insert prefix into node
	newVal, exists := n.prefixes.UpdateAt(pfxToIdx(sigOctet, sigOctetBits), cb)
	if !exists {
		t.sizeUpdate(is4, 1)
	}

	return newVal
}

// Delete removes pfx from the tree, pfx does not have to be present.
func (t *Table2[V]) Delete(pfx netip.Prefix) {
	_, _ = t.getAndDelete(pfx)
}

// GetAndDelete deletes the prefix and returns the associated payload for prefix and true,
// or the zero vlaue and false if prefix is not set in the routing table.
func (t *Table2[V]) GetAndDelete(pfx netip.Prefix) (val V, ok bool) {
	return t.getAndDelete(pfx)
}

func (t *Table2[V]) getAndDelete(pfx netip.Prefix) (val V, ok bool) {
	var zero V

	if !pfx.IsValid() {
		return zero, false
	}

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()

	n := t.rootNodeByVersion(is4)

	octets := ip.AsSlice()
	sigOctetIdx := (bits - 1) / strideLen
	sigOctet := octets[sigOctetIdx]
	sigOctetBits := bits - (sigOctetIdx * strideLen)

	// mask the prefix
	sigOctet &= netMask(sigOctetBits)
	octets[sigOctetIdx] = sigOctet

	// record path to deleted node
	stack := [maxTreeDepth]*node2[V]{}

	// find the trie node
	for i, octet := range octets[:sigOctetIdx] {
		// push current node on stack for path recording
		// needed fur purging nodes after deletion
		stack[i] = n
		addr := uint(octet)

		if !n.children.Test(addr) {
			return zero, false
		}

		// get the child: node or leaf
		switch k := n.children.MustGet(addr).(type) {
		case *node2[V]:
			// descend down to next trie level
			n = k
		case *leaf[V]:
			// reached a path compressed prefix, stop traversing
			if k.prefix != pfx {
				return zero, false
			}

			// prefix is equal leaf, delete leaf
			n.children.DeleteAt(addr)

			t.sizeUpdate(is4, -1)
			n.purgeParents(stack[:i], octets)

			return k.value, true
		}
	}

	// try to delete prefix in trie node
	if val, ok = n.prefixes.DeleteAt(pfxToIdx(sigOctet, sigOctetBits)); ok {
		t.sizeUpdate(is4, -1)
		n.purgeParents(stack[:sigOctetIdx], octets)

		return val, ok
	}

	return zero, false
}

// Get returns the associated payload for prefix and true, or false if
// prefix is not set in the routing table.
func (t *Table2[V]) Get(pfx netip.Prefix) (val V, ok bool) {
	var zero V

	if !pfx.IsValid() {
		return zero, false
	}

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()

	n := t.rootNodeByVersion(is4)

	octets := ip.AsSlice()
	sigOctetIdx := (bits - 1) / strideLen
	sigOctet := octets[sigOctetIdx]
	sigOctetBits := bits - (sigOctetIdx * strideLen)

	// mask the prefix
	sigOctet &= netMask(sigOctetBits)
	octets[sigOctetIdx] = sigOctet

	// find the trie node
	for _, octet := range octets[:sigOctetIdx] {
		addr := uint(octet)

		if !n.children.Test(addr) {
			return zero, false
		}

		// get the child: node or leaf
		switch k := n.children.MustGet(addr).(type) {
		case *node2[V]:
			// descend down to next trie level
			n = k
		case *leaf[V]:
			// reached a path compressed prefix, stop traversing
			if k.prefix == pfx {
				return k.value, true
			}
			return zero, false
		}
	}

	return n.prefixes.Get(pfxToIdx(sigOctet, sigOctetBits))
}

// Contains does a route lookup (longest prefix match, lpm) for IP and
// returns true if any route matched, or false if not.
//
// Contains does not return the value nor the prefix of the matching item,
// but as a test against a black- or whitelist it's often sufficient
// and even few nanoseconds faster than [Table2.Lookup].
func (t *Table2[V]) Contains(ip netip.Addr) bool {
	if !ip.IsValid() {
		return false
	}

	is4 := ip.Is4()
	n := t.rootNodeByVersion(is4)

	a16 := ip.As16()
	octets := a16[:]
	if is4 {
		octets = octets[12:]
	}

	// stack of the traversed nodes for fast backtracking, if needed
	stack := [maxTreeDepth]*node2[V]{}

	// run variable, used after for loop
	var i int
	var octet byte

LOOP:
	// find leaf node for octet path
	for i, octet = range octets {
		addr := uint(octet)

		// push current node on stack for fast backtracking
		stack[i] = n

		// go down in tight loop to last octet
		if !n.children.Test(addr) {
			// no more nodes below octet
			break LOOP
		}

		// get node or leaf for octet
		switch k := n.children.MustGet(addr).(type) {
		case *node2[V]:
			// go next level down
			n = k
		case *leaf[V]:
			// reached a path compressed prefix, stop traversing
			if k.prefix.Contains(ip) {
				return true
			}
			break LOOP
		}

	}

	// start backtracking, unwind the stack
	for depth := i; depth >= 0; depth-- {
		n := stack[depth]

		// longest prefix match, skip if node has no prefixes
		if n.prefixes.Len() != 0 && n.lpmTest(hostIndex(uint(octets[depth]))) {
			return true
		}
	}

	return false
}

// Lookup does a route lookup (longest prefix match) for IP and
// returns the associated value and true, or false if no route matched.
func (t *Table2[V]) Lookup(ip netip.Addr) (val V, ok bool) {
	var zero V

	if !ip.IsValid() {
		return zero, false
	}

	is4 := ip.Is4()
	n := t.rootNodeByVersion(is4)

	// do not allocate
	a16 := ip.As16()
	octets := a16[:]
	if is4 {
		octets = octets[12:]
	}

	// stack of the traversed nodes for fast backtracking, if needed
	stack := [maxTreeDepth]*node2[V]{}

	// run variable, used after for loop
	var i int
	var octet byte

LOOP:
	// find leaf node
	for i, octet = range octets {
		addr := uint(octet)

		// push current node on stack for fast backtracking
		stack[i] = n

		// go down in tight loop to last octet
		if !n.children.Test(addr) {
			// no more nodes below octet
			break LOOP
		}

		// get node or leaf for octet
		switch k := n.children.MustGet(addr).(type) {
		case *node2[V]:
			// descend down to next trie level
			n = k
		case *leaf[V]:
			// reached a path compressed prefix, stop traversing
			if k.prefix.Contains(ip) {
				return k.value, true
			}
			break LOOP
		}
	}

	// start backtracking, unwind the stack
	for depth := i; depth >= 0; depth-- {
		n = stack[depth]

		// longest prefix match, skip if node has no prefixes
		if n.prefixes.Len() != 0 {
			if _, val, ok = n.lpm(hostIndex(uint(octets[depth]))); ok {
				return val, ok
			}
		}
	}

	return zero, false
}

// LookupPrefix does a route lookup (longest prefix match) for pfx and
// returns the associated value and true, or false if no route matched.
func (t *Table2[V]) LookupPrefix(pfx netip.Prefix) (val V, ok bool) {
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
// If LookupPrefixLPM is to be used for IP addresses,
// they must be converted to /32 or /128 prefixes.
func (t *Table2[V]) LookupPrefixLPM(pfx netip.Prefix) (lpm netip.Prefix, val V, ok bool) {
	if !pfx.IsValid() {
		return
	}

	return t.lpmPrefix(pfx)
}

// lpmPrefix, returns depth, baseIdx, val and ok for a lpm match.
func (t *Table2[V]) lpmPrefix(pfx netip.Prefix) (lpm netip.Prefix, val V, ok bool) {
	var zeroVal V
	var zeroPfx netip.Prefix

	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()

	n := t.rootNodeByVersion(is4)

	a16 := ip.As16()
	octets := a16[:]
	if is4 {
		octets = octets[12:]
	}

	// see comment in Insert()
	sigOctetIdx := (bits - 1) / strideLen
	sigOctet := octets[sigOctetIdx]
	sigOctetBits := bits - (sigOctetIdx * strideLen)

	// mask the prefix
	sigOctet &= netMask(sigOctetBits)
	octets[sigOctetIdx] = sigOctet

	var i int
	var octet byte

	// record path to leaf node
	stack := [maxTreeDepth]*node2[V]{}

LOOP:
	// find the node
	for i, octet = range octets[:sigOctetIdx+1] {
		addr := uint(octet)

		// push current node on stack
		stack[i] = n

		// go down in tight loop to leaf node
		if !n.children.Test(addr) {
			break LOOP
		}

		// get the child: node or leaf
		switch k := n.children.MustGet(addr).(type) {
		case *node2[V]:
			// descend down to next trie level
			n = k
		case *leaf[V]:
			// reached a path compressed prefix, stop traversing
			if k.prefix.Contains(ip) && k.prefix.Bits() <= bits {
				return k.prefix, k.value, true
			}

			break LOOP
		}
	}

	// start backtracking, unwind the stack
	for depth := i; depth >= 0; depth-- {
		n = stack[depth]

		// longest prefix match, skip if node has no prefixes
		if n.prefixes.Len() != 0 {
			// only the lastOctet may have a different prefix len
			// all others are just host routes
			var idx uint
			if depth == sigOctetIdx {
				idx = pfxToIdx(octet, sigOctetBits)
			} else {
				idx = hostIndex(uint(octets[depth]))
			}

			if baseIdx, val, ok := n.lpm(idx); ok {
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

/*

// OverlapsPrefix reports whether any IP in pfx is matched by a route in the table or vice versa.
func (t *Table2[V]) OverlapsPrefix(pfx netip.Prefix) bool {
	if !pfx.IsValid() {
		return false
	}

	// canonicalize the prefix
	pfx = pfx.Masked()

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()

	// get the root node of the routing table
	n := t.rootNodeByVersion(is4)

	// do not allocate
	a16 := ip.As16()

	octets := a16[:]
	if is4 {
		octets = octets[12:]
	}

	// see comment in Insert()
	lastOctetIdx := (bits - 1) / strideLen
	lastOctet := octets[lastOctetIdx]
	lastOctetBits := bits - (lastOctetIdx * strideLen)

	var addr uint
	for _, octet := range octets[:lastOctetIdx] {
		addr = uint(octet)

		// test if any route overlaps prefix´ so far
		if n.prefixes.Len() != 0 && n.lpmTest(hostIndex(addr)) {
			return true
		}

		// no overlap so far, go down to next child
		if n.children.Test(addr) {
			n = n.children.MustGet(addr)
			continue
		}

		// no child so far, look for path compressed item
		if t.pathCompressed && n.pathcomp.Test(addr) {
			pc := n.pathcomp.MustGet(addr)
			return pfx.Overlaps(pc.prefix)
		}

		// nope, nothing
		return false
	}
	return n.overlapsPrefix(lastOctet, lastOctetBits)
}

// Overlaps reports whether any IP in the table is matched by a route in the
// other table or vice versa.
//
// panic's if the path compression of the two tables does not match.
// TODO, path compression not yet implemented for this method.
func (t *Table2[V]) Overlaps(o *Table2[V]) bool {
	if t.pathCompressed || o.pathCompressed {
		panic("TODO, path compression not yet implemented for this method")
	}

	if t.pathCompressed != o.pathCompressed {
		panic("tables MUST NOT differ in path compressions")
	}
	return t.Overlaps4(o) || t.Overlaps6(o)
}

// Overlaps4 reports whether any IPv4 in the table matches a route in the
// other table or vice versa.
//
// panic's if the path compression of the two tables does not match.
func (t *Table2[V]) Overlaps4(o *Table2[V]) bool {
	if t.pathCompressed != o.pathCompressed {
		panic("tables MUST NOT differ in path compressions")
	}
	if t.size4 == 0 || o.size4 == 0 {
		return false
	}
	return t.root4.overlapsRec(&o.root4)
}

// Overlaps6 reports whether any IPv6 in the table matches a route in the
// other table or vice versa.
//
// panic's if the path compression of the two tables does not match.
func (t *Table2[V]) Overlaps6(o *Table2[V]) bool {
	if t.pathCompressed != o.pathCompressed {
		panic("tables MUST NOT differ in path compressions")
	}
	if t.size6 == 0 || o.size6 == 0 {
		return false
	}
	return t.root6.overlapsRec(&o.root6)
}

// Union combines two tables, changing the receiver table.
// If there are duplicate entries, the payload of type V is shallow copied from the other table.
// If type V implements the [Cloner] interface, the values are cloned, see also [Table.Clone].
//
// panic's if the path compression of the two tables does not match.
func (t *Table2[V]) Union(o *Table2[V]) {
	if t.pathCompressed != o.pathCompressed {
		panic("tables MUST NOT differ in path compressions")
	}

	dup4 := t.root4.unionRec(&o.root4, 0)
	dup6 := t.root6.unionRec(&o.root6, 0)

	t.size4 += o.size4 - dup4
	t.size6 += o.size6 - dup6
}

*/

// Cloner, if implemented by payload of type V the values are deeply copied
// during [Table.Clone] and [Table.Union].
type Cloner[V any] interface {
	Clone() V
}

func cloneValue[V any](v V) V {
	if k, ok := any(v).(Cloner[V]); ok {
		return k.Clone()
	}
	return v
}

// Clone returns a copy of the routing table.
// The payload of type V is shallow copied, but if type V implements the [Cloner] interface,
// the values are cloned.
func (t *Table2[V]) Clone() *Table2[V] {
	if t == nil {
		return nil
	}

	c := new(Table2[V])

	c.root4 = *t.root4.cloneRec()
	c.root6 = *t.root6.cloneRec()

	c.size4 = t.size4
	c.size6 = t.size6

	return c
}

func (t *Table2[V]) sizeUpdate(is4 bool, n int) {
	if is4 {
		t.size4 += n
		return
	}
	t.size6 += n
}

// Size returns the prefix count.
func (t *Table2[V]) Size() int {
	return t.size4 + t.size6
}

// Size4 returns the IPv4 prefix count.
func (t *Table2[V]) Size4() int {
	return t.size4
}

// Size6 returns the IPv6 prefix count.
func (t *Table2[V]) Size6() int {
	return t.size6
}

// count the nodes and leaves
func (t *Table2[V]) nodeAndLeafCount() (int, int) {
	n4, l4 := t.root4.nodeAndLeafCountRec()
	n6, l6 := t.root6.nodeAndLeafCountRec()
	return n4 + n6, l4 + l6
}

// nodes, count the nodes
func (t *Table2[V]) nodes() int {
	n4, _ := t.root4.nodeAndLeafCountRec()
	n6, _ := t.root6.nodeAndLeafCountRec()
	return n4 + n6
}

// nodes4
func (t *Table2[V]) nodes4() int {
	n4, _ := t.root4.nodeAndLeafCountRec()
	return n4
}

// nodes6
func (t *Table2[V]) nodes6() int {
	n6, _ := t.root6.nodeAndLeafCountRec()
	return n6
}
