// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// package bart provides a Balanced-Routing-Table (BART).
//
// BART is balanced in terms of memory consumption versus
// lookup time.
//
// The lookup time is by a factor of ~2 slower on average as the
// routing algorithms ART, SMART, CPE, ... but reduces the memory
// consumption by an order of magnitude in comparison.
//
// BART is a multibit-trie with fixed stride length of 8 bits,
// using the _baseIndex_ function from the ART algorithm to
// build the complete-binary-tree (CBT) of prefixes for each stride.
//
// The second key factor is popcount array compression at each stride level
// of the CBT prefix tree and backtracking along the CBT in O(k).
//
// The CBT is implemented as a bitvector, backtracking is just
// a matter of fast cache friendly bitmask operations.
//
// The child array at each stride level is also popcount compressed.
package bart

import (
	"net/netip"
)

// Table is an IPv4 and IPv6 routing table with payload V.
// The zero value is ready to use.
//
// The Table is safe for concurrent readers but not for
// concurrent readers and/or writers.
type Table[V any] struct {
	root4 *node[V]
	root6 *node[V]

	size4 int
	size6 int
}

// isInit reports if the table is already initialized.
func (t *Table[V]) isInit() bool {
	// could also test t.root6, no hidden magic meaning
	return t.root4 != nil
}

// init BitSets once, so no constructor is needed and the zero value is ready to use.
// Not using sync.Once, the table is not safe for concurrent writers anyway
func (t *Table[V]) init() {
	// could also test t.root6, no hidden magic meaning
	if !t.isInit() {
		t.root4 = newNode[V]()
		t.root6 = newNode[V]()
	}
}

// rootNodeByVersion, select root node for ip version.
func (t *Table[V]) rootNodeByVersion(is4 bool) *node[V] {
	if is4 {
		return t.root4
	}
	return t.root6
}

// Cloner, if implemented by payload of type V the values are deeply copied during [Table.Clone] and [Table.Union].
type Cloner[V any] interface {
	Clone() V
}

// Insert adds pfx to the tree, with value val.
// If pfx is already present in the tree, its value is set to val.
func (t *Table[V]) Insert(pfx netip.Prefix, val V) {
	if !pfx.IsValid() {
		return
	}

	t.init()

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()

	// get the root node of the routing table
	n := t.rootNodeByVersion(is4)

	// Do not allocate!
	// As16() is inlined, the preffered AsSlice() is too complex for inlining.
	// starting with go1.23 we can use AsSlice(),
	// see https://github.com/golang/go/issues/56136

	a16 := ip.As16()
	octets := a16[:]
	if is4 {
		octets = octets[12:]
	}

	// 10.0.0.0/8    -> 0
	// 10.12.0.0/15  -> 1
	// 10.12.0.0/16  -> 1
	// 10.12.10.9/32 -> 3
	lastOctetIdx := (bits - 1) / strideLen

	// 10.0.0.0/8    -> 10
	// 10.12.0.0/15  -> 12
	// 10.12.0.0/16  -> 12
	// 10.12.10.9/32 -> 9
	lastOctet := octets[lastOctetIdx]

	// 10.0.0.0/8    -> 8
	// 10.12.0.0/15  -> 7
	// 10.12.0.0/16  -> 8
	// 10.12.10.9/32 -> 8
	lastOctetBits := bits - (lastOctetIdx * strideLen)

	// mask the prefix, this is faster than netip.Prefix.Masked()
	lastOctet &= netMask(lastOctetBits)

	// find the proper trie node to insert prefix
	for _, octet := range octets[:lastOctetIdx] {
		// descend down to next trie level
		c := n.getChild(octet)
		if c == nil {
			// create and insert missing intermediate child
			c = newNode[V]()
			n.insertChild(octet, c)
		}

		// proceed with next level
		n = c
	}

	// insert prefix/val into node
	if n.insertPrefix(prefixToBaseIndex(lastOctet, lastOctetBits), val) {
		t.sizeUpdate(is4, 1)
	}
}

// Update or set the value at pfx with a callback function.
// The callback function is called with (value, ok) and returns a new value..
//
// If the pfx does not already exist, it is set with the new value.
func (t *Table[V]) Update(pfx netip.Prefix, cb func(val V, ok bool) V) (newVal V) {
	if !pfx.IsValid() {
		var zero V
		return zero
	}

	t.init()

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()

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

	// mask the prefix
	lastOctet &= netMask(lastOctetBits)

	// find the proper trie node to update prefix
	for _, octet := range octets[:lastOctetIdx] {
		// descend down to next trie level
		c := n.getChild(octet)
		if c == nil {
			// create and insert missing intermediate child
			c = newNode[V]()
			n.insertChild(octet, c)
		}

		// proceed with next level
		n = c
	}

	// update/insert prefix into node
	var wasPresent bool
	newVal, wasPresent = n.updatePrefix(lastOctet, lastOctetBits, cb)
	if !wasPresent {
		t.sizeUpdate(is4, 1)
	}

	return newVal
}

// Get returns the associated payload for prefix and true, or false if
// prefix is not set in the routing table.
func (t *Table[V]) Get(pfx netip.Prefix) (val V, ok bool) {
	if !pfx.IsValid() || !t.isInit() {
		return
	}

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()

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

	// mask the prefix
	lastOctet &= netMask(lastOctetBits)

	// find the proper trie node
	for _, octet := range octets[:lastOctetIdx] {
		c := n.getChild(octet)
		if c == nil {
			// not found
			return
		}
		n = c
	}
	return n.getValueOK(prefixToBaseIndex(lastOctet, lastOctetBits))
}

// Delete removes pfx from the tree, pfx does not have to be present.
func (t *Table[V]) Delete(pfx netip.Prefix) {
	if !pfx.IsValid() || !t.isInit() {
		return
	}

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()

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

	// mask the prefix
	lastOctet &= netMask(lastOctetBits)
	octets[lastOctetIdx] = lastOctet

	// record path to deleted node
	stack := [maxTreeDepth]*node[V]{}

	// run variable as stackPointer, see below
	var i int

	// find the trie node
	for i = range octets {
		// push current node on stack for path recording
		stack[i] = n

		if i == lastOctetIdx {
			break
		}

		// descend down to next level
		c := n.getChild(octets[i])
		if c == nil {
			return
		}
		n = c
	}

	// try to delete prefix in trie node
	if !n.deletePrefix(lastOctet, lastOctetBits) {
		// nothing deleted
		return
	}
	t.sizeUpdate(is4, -1)

	// purge dangling nodes after successful deletion
	for i > 0 {
		if n.isEmpty() {
			// purge empty node from parents children
			parent := stack[i-1]
			parent.deleteChild(octets[i-1])
		}

		// unwind the stack
		i--
		n = stack[i]
	}
}

// Lookup does a route lookup (longest prefix match) for IP and
// returns the associated value and true, or false if no route matched.
func (t *Table[V]) Lookup(ip netip.Addr) (val V, ok bool) {
	if !ip.IsValid() || !t.isInit() {
		return
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
	stack := [maxTreeDepth]*node[V]{}

	// run variable, used after for loop
	var i int
	var octet byte

	// find leaf node
	for i, octet = range octets {
		// push current node on stack for fast backtracking
		stack[i] = n

		// go down in tight loop to leaf node
		c := n.getChild(octet)
		if c == nil {
			break
		}
		n = c
	}

	// start backtracking, unwind the stack
	for depth := i; depth >= 0; depth-- {
		n = stack[depth]
		octet = octets[depth]

		// longest prefix match
		// micro benchmarking: skip if node has no prefixes
		if len(n.prefixes) != 0 {
			if _, val, ok = n.lpm(octetToBaseIndex(octet)); ok {
				return val, true
			}
		}
	}
	return
}

// LookupPrefix does a route lookup (longest prefix match) for pfx and
// returns the associated value and true, or false if no route matched.
func (t *Table[V]) LookupPrefix(pfx netip.Prefix) (val V, ok bool) {
	if !pfx.IsValid() || !t.isInit() {
		return
	}

	_, _, val, ok = t.lpmPrefix(pfx)
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
func (t *Table[V]) LookupPrefixLPM(pfx netip.Prefix) (lpm netip.Prefix, val V, ok bool) {
	if !pfx.IsValid() || !t.isInit() {
		return
	}

	depth, baseIdx, val, ok := t.lpmPrefix(pfx)

	if ok {
		// calculate the mask from baseIdx and depth
		mask := baseIndexToPrefixLen(baseIdx, depth)

		// calculate the lpm from ip and mask
		lpm, _ = pfx.Addr().Prefix(mask)
	}

	return lpm, val, ok
}

func (t *Table[V]) lpmPrefix(pfx netip.Prefix) (depth int, baseIdx uint, val V, ok bool) {
	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()

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

	// mask the prefix
	lastOctet &= netMask(lastOctetBits)
	octets[lastOctetIdx] = lastOctet

	var i int
	var octet byte

	// record path to leaf node
	stack := [maxTreeDepth]*node[V]{}

	// find the node
	for i, octet = range octets[:lastOctetIdx+1] {
		// push current node on stack
		stack[i] = n

		// go down in tight loop
		c := n.getChild(octet)
		if c == nil {
			break
		}
		n = c
	}

	// start backtracking, unwind the stack
	for depth = i; depth >= 0; depth-- {
		n = stack[depth]
		octet = octets[depth]

		// longest prefix match
		// micro benchmarking: skip if node has no prefixes
		if len(n.prefixes) != 0 {

			// only the lastOctet may have a different prefix len
			// all others are just host routes
			var idx uint
			if depth == lastOctetIdx {
				idx = prefixToBaseIndex(octet, lastOctetBits)
			} else {
				idx = octetToBaseIndex(octet)
			}

			baseIdx, val, ok = n.lpm(idx)
			if ok {
				return depth, baseIdx, val, ok
			}
		}
	}
	return
}

// OverlapsPrefix reports whether any IP in pfx is matched by a route in the table or vice versa.
func (t *Table[V]) OverlapsPrefix(pfx netip.Prefix) bool {
	if !pfx.IsValid() || !t.isInit() {
		return false
	}

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

	// mask the prefix
	lastOctet &= netMask(lastOctetBits)

	for _, octet := range octets[:lastOctetIdx] {
		// test if any route overlaps prefix´ so far
		if n.lpmTest(octetToBaseIndex(octet)) {
			return true
		}

		// no overlap so far, go down to next c
		c := n.getChild(octet)
		if c == nil {
			return false
		}
		n = c
	}

	return n.overlapsPrefix(lastOctet, lastOctetBits)
}

// Overlaps reports whether any IP in the table is matched by a route in the
// other table or vice versa.
func (t *Table[V]) Overlaps(o *Table[V]) bool {
	if t.Size() == 0 || o.Size() == 0 {
		return false
	}

	// t and o are already intialized (size != 0)

	// at least one v4 is empty
	if t.size4 == 0 || o.size4 == 0 {
		return t.Overlaps6(o)
	}

	// at least one v6 is empty
	if t.size6 == 0 || o.size6 == 0 {
		return t.Overlaps4(o)
	}

	return t.Overlaps4(o) || t.Overlaps6(o)
}

// Overlaps4 reports whether any IPv4 in the table matches a route in the
// other table or vice versa.
func (t *Table[V]) Overlaps4(o *Table[V]) bool {
	if t.size4 == 0 || o.size4 == 0 {
		return false
	}

	// t and o are already intialized (size != 0)
	return t.root4.overlapsRec(o.root4)
}

// Overlaps6 reports whether any IPv6 in the table matches a route in the
// other table or vice versa.
func (t *Table[V]) Overlaps6(o *Table[V]) bool {
	if t.size6 == 0 || o.size6 == 0 {
		return false
	}

	// t and o are already intialized (size != 0)
	return t.root6.overlapsRec(o.root6)
}

// Union combines two tables, changing the receiver table.
// If there are duplicate entries, the payload of type V is shallow copied from the other table.
// If type V implements the [Cloner] interface, the values are cloned, see also [Table.Clone].
func (t *Table[V]) Union(o *Table[V]) {
	// nothing to do
	if !o.isInit() {
		return
	}

	t.init()

	dup4 := t.root4.unionRec(o.root4)
	dup6 := t.root6.unionRec(o.root6)

	t.size4 += o.size4 - dup4
	t.size6 += o.size6 - dup6
}

// Clone returns a copy of the routing table.
// The payload of type V is shallow copied, but if type V implements the [Cloner] interface, the values are cloned.
func (t *Table[V]) Clone() *Table[V] {
	c := new(Table[V])
	if !t.isInit() {
		return c
	}

	c.root4 = t.root4.cloneRec()
	c.root6 = t.root6.cloneRec()

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

// nodes, calculates the IPv4 and IPv6 nodes and returns the sum.
func (t *Table[V]) nodes() int {
	if !t.isInit() {
		return 0
	}

	return t.root4.numNodesRec() + t.root6.numNodesRec()
}
