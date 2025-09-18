// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/netip"
	"strings"
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

// Insert adds or updates a prefix-value pair in the routing table.
// If the prefix already exists, its value is updated; otherwise a new entry is created.
// Invalid prefixes are silently ignored.
//
// The prefix is automatically canonicalized using pfx.Masked() to ensure
// consistent behavior regardless of host bits in the input.
//
// Its semantics are identical to [Table.Insert].
func (f *Fast[V]) Insert(pfx netip.Prefix, val V) {
	if !pfx.IsValid() {
		return
	}
	// canonicalize prefix
	pfx = pfx.Masked()
	is4 := pfx.Addr().Is4()

	n := f.rootNodeByVersion(is4)

	// insert prefix
	if exists := n.insertAtDepth(pfx, val, 0); exists {
		return
	}

	// true insert, update size
	f.sizeUpdate(is4, 1)
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
			_ = newNode.insertAtDepth(kid.prefix, kid.value, depth+1)
			_ = n.insertChild(octet, newNode)
			n = newNode

		default:
			panic("logic error, wrong node type")
		}
	}

	return
}

// Delete removes a prefix from the routing table and returns its associated value.
// Returns the zero value of V and false if the prefix doesn't exist.
// Invalid prefixes are silently ignored.
//
// The prefix is automatically canonicalized using pfx.Masked().
//
// Its semantics are identical to [Table.Delete].
func (f *Fast[V]) Delete(pfx netip.Prefix) (val V, exists bool) {
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
			// try to delete prefix in trie node
			val, exists = n.deletePrefix(art.PfxToIdx(octet, lastBits))
			if !exists {
				return val, false
			}

			f.sizeUpdate(is4, -1)
			n.purgeAndCompress(stack[:depth], octets, is4)
			return val, true
		}

		kidAny, ok := n.getChild(octet)
		if !ok {
			return
		}

		// kid is node or leaf or fringe at octet
		switch kid := kidAny.(type) {
		case *fastNode[V]:
			n = kid // descend down to next trie level

		case *fringeNode[V]:
			// if pfx is no fringe at this depth, fast exit
			if !isFringe(depth, pfx) {
				return
			}

			// pfx is fringe at depth, delete fringe
			n.deleteChild(octet)

			f.sizeUpdate(is4, -1)
			n.purgeAndCompress(stack[:depth], octets, is4)

			return kid.value, true

		case *leafNode[V]:
			// Attention: pfx must be masked to be comparable!
			if kid.prefix != pfx {
				return
			}

			// prefix is equal leaf, delete leaf
			n.deleteChild(octet)

			f.sizeUpdate(is4, -1)
			n.purgeAndCompress(stack[:depth], octets, is4)

			return kid.value, true

		default:
			panic("logic error, wrong node type")
		}
	}

	return
}

// Get retrieves the value associated with an exact prefix match.
// Returns the zero value of V and false if the prefix doesn't exist.
// Invalid prefixes return the zero value and false.
//
// The prefix is automatically canonicalized using pfx.Masked().
//
// This performs exact prefix matching, not longest-prefix matching.
// Use Lookup for longest-prefix matching with IP addresses.
//
// Its semantics are identical to [Table.Get].
func (f *Fast[V]) Get(pfx netip.Prefix) (val V, ok bool) {
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
		if depth == lastOctetPlusOne {
			return n.getPrefix(art.PfxToIdx(octet, lastBits))
		}

		kidAny, exists := n.getChild(octet)
		if !exists {
			return
		}

		// kid is node or leaf or fringe at octet
		switch kid := kidAny.(type) {
		case *fastNode[V]:
			n = kid // descend down to next trie level

		case *fringeNode[V]:
			// reached a path compressed fringe, stop traversing
			if isFringe(depth, pfx) {
				return kid.value, true
			}
			return

		case *leafNode[V]:
			// reached a path compressed prefix, stop traversing
			if kid.prefix == pfx {
				return kid.value, true
			}
			return

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
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
		return
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
		return
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
		if n.prefixCount() == 0 {
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
			if val, ok := n.lookup(idx); ok {
				return netip.Prefix{}, val, ok
			}

		case true: // LookupPrefixLPM
			if topIdx, val, ok := n.lookupIdx(idx); ok {
				// get the bits from depth and top idx
				pfxBits := int(art.PfxBits(depth, topIdx))

				// calculate the lpmPfx from incoming ip and new mask
				lpmPfx, _ = ip.Prefix(pfxBits)
				return lpmPfx, val, ok
			}
		}
		// continue rewinding the stack
	}

	return
}

// Clone creates a deep copy of the routing table, including all prefixes and values.
// If the value type V implements the Cloner[V] interface, values are cloned using
// the Clone() method; otherwise values are copied by assignment.
//
// Returns nil if the receiver is nil.
//
// Its semantics are identical to [Table.Clone].
func (f *Fast[V]) Clone() *Fast[V] {
	if f == nil {
		return nil
	}

	c := new(Fast[V])

	cloneFn := cloneFnFactory[V]()

	c.root4 = *f.root4.cloneRec(cloneFn)
	c.root6 = *f.root6.cloneRec(cloneFn)

	c.size4 = f.size4
	c.size6 = f.size6

	return c
}

func (f *Fast[V]) sizeUpdate(is4 bool, n int) {
	if is4 {
		f.size4 += n
		return
	}
	f.size6 += n
}

// Size returns the prefix count.
func (f *Fast[V]) Size() int {
	return f.size4 + f.size6
}

// Size4 returns the IPv4 prefix count.
func (f *Fast[V]) Size4() int {
	return f.size4
}

// Size6 returns the IPv6 prefix count.
func (f *Fast[V]) Size6() int {
	return f.size6
}

// dumpString is just a wrapper for dump.
func (t *Fast[V]) dumpString() string {
	w := new(strings.Builder)
	t.dump(w)

	return w.String()
}

// dump the table structure and all the nodes to w.
func (t *Fast[V]) dump(w io.Writer) {
	if t == nil {
		return
	}

	if t.size4 > 0 {
		stats := nodeStatsRec(&t.root4)
		fmt.Fprintln(w)
		fmt.Fprintf(w, "### IPv4: size(%d), nodes(%d), pfxs(%d), leaves(%d), fringes(%d)",
			t.size4, stats.nodes, stats.pfxs, stats.leaves, stats.fringes)

		dumpRec(&t.root4, w, stridePath{}, 0, true)
	}

	if t.size6 > 0 {
		stats := nodeStatsRec(&t.root6)
		fmt.Fprintln(w)
		fmt.Fprintf(w, "### IPv6: size(%d), nodes(%d), pfxs(%d), leaves(%d), fringes(%d)",
			t.size6, stats.nodes, stats.pfxs, stats.leaves, stats.fringes)

		dumpRec(&t.root6, w, stridePath{}, 0, false)
	}
}

// String returns a hierarchical tree diagram of the ordered CIDRs
// as string, just a wrapper for [Table.Fprint].
// If Fprint returns an error, String panics.
func (t *Fast[V]) String() string {
	w := new(strings.Builder)
	if err := t.Fprint(w); err != nil {
		panic(err)
	}

	return w.String()
}

// Fprint writes a hierarchical tree diagram of the ordered CIDRs
// with default formatted payload V to w.
//
// The order from top to bottom is in ascending order of the prefix address
// and the subtree structure is determined by the CIDRs coverage.
//
//	▼
//	├─ 10.0.0.0/8 (V)
//	│  ├─ 10.0.0.0/24 (V)
//	│  └─ 10.0.1.0/24 (V)
//	├─ 127.0.0.0/8 (V)
//	│  └─ 127.0.0.1/32 (V)
//	├─ 169.254.0.0/16 (V)
//	├─ 172.16.0.0/12 (V)
//	└─ 192.168.0.0/16 (V)
//	   └─ 192.168.1.0/24 (V)
//	▼
//	└─ ::/0 (V)
//	   ├─ ::1/128 (V)
//	   ├─ 2000::/3 (V)
//	   │  └─ 2001:db8::/32 (V)
//	   └─ fe80::/10 (V)
func (t *Fast[V]) Fprint(w io.Writer) error {
	if w == nil {
		return fmt.Errorf("nil writer")
	}
	if t == nil {
		return nil
	}

	// v4
	if err := t.fprint(w, true); err != nil {
		return err
	}

	// v6
	if err := t.fprint(w, false); err != nil {
		return err
	}

	return nil
}

// fprint is the version dependent adapter to fprintRec.
func (t *Fast[V]) fprint(w io.Writer, is4 bool) error {
	n := t.rootNodeByVersion(is4)
	if n.isEmpty() {
		return nil
	}

	if _, err := fmt.Fprint(w, "▼\n"); err != nil {
		return err
	}

	startParent := trieItem[V]{
		n:    nil,
		idx:  0,
		path: stridePath{},
		is4:  is4,
	}

	return fprintRec(n, w, startParent, "", shouldPrintValues[V]())
}

// MarshalText implements the [encoding.TextMarshaler] interface,
// just a wrapper for [Table.Fprint].
func (t *Fast[V]) MarshalText() ([]byte, error) {
	w := new(bytes.Buffer)
	if err := t.Fprint(w); err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

// MarshalJSON dumps the table into two sorted lists: for ipv4 and ipv6.
// Every root and subnet is an array, not a map, because the order matters.
func (t *Fast[V]) MarshalJSON() ([]byte, error) {
	if t == nil {
		return []byte("null"), nil
	}

	result := struct {
		Ipv4 []DumpListNode[V] `json:"ipv4,omitempty"`
		Ipv6 []DumpListNode[V] `json:"ipv6,omitempty"`
	}{
		Ipv4: t.DumpList4(),
		Ipv6: t.DumpList6(),
	}

	buf, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// DumpList4 dumps the ipv4 tree into a list of roots and their subnets.
// It can be used to analyze the tree or build the text or json serialization.
func (t *Fast[V]) DumpList4() []DumpListNode[V] {
	if t == nil {
		return nil
	}
	return dumpListRec(&t.root4, 0, stridePath{}, 0, true)
}

// DumpList6 dumps the ipv6 tree into a list of roots and their subnets.
// It can be used to analyze the tree or build custom json representation.
func (t *Fast[V]) DumpList6() []DumpListNode[V] {
	if t == nil {
		return nil
	}
	return dumpListRec(&t.root6, 0, stridePath{}, 0, false)
}
