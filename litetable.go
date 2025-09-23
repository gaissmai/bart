// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/netip"
	"strings"
	"sync"

	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/lpm"
)

// Lite is a prefix-only table that embeds a private liteTable[struct{}].
//
// It is intentionally not nil-receiver safe: calling methods on a nil *Lite
// will panic by design.
type Lite struct {
	liteTable[struct{}]
}

// Insert adds a pfx to the tree.
// If pfx is already present in the tree, it's a no-op.
func (l *Lite) Insert(pfx netip.Prefix) {
	l.liteTable.Insert(pfx, struct{}{})
}

// Overlaps reports whether any route in the receiver table overlaps
// with a route in the other table, in either direction.
//
// The overlap check is bidirectional: it returns true if any IP prefix
// in the receiver is covered by the other table, or vice versa.
// This includes partial overlaps, exact matches, and supernet/subnet relationships.
//
// Both IPv4 and IPv6 route trees are compared independently. If either
// tree has overlapping routes, the function returns true.
//
// This is useful for conflict detection, policy enforcement,
// or validating mutually exclusive routing domains.
//
// It is intentionally not nil-receiver safe: calling with a nil *Lite
// will panic by design.
func (l *Lite) Overlaps(o *Lite) bool {
	if o == nil {
		return false
	}
	return l.liteTable.Overlaps(&o.liteTable)
}

// Union merges another routing table into the receiver table, modifying it in-place.
//
// All prefixes from the other table (o) are inserted into the receiver.
func (l *Lite) Union(o *Lite) {
	if o == nil {
		return
	}
	l.liteTable.Union(&o.liteTable)
}

// UnionPersist is similar to [Union] but the receiver isn't modified.
//
// All nodes touched during union are cloned and a new *Lite is returned.
// If o is nil or empty, no nodes are touched and the receiver may be
// returned unchanged.
func (l *Lite) UnionPersist(o *Lite) *Lite {
	if o == nil || (o.size4 == 0 && o.size6 == 0) {
		return l
	}
	tbl := l.liteTable.UnionPersist(&o.liteTable)
	//nolint:govet // copy of *tbl is here by intention
	return &Lite{*tbl}
}

// Equal checks whether two tables are structurally and semantically equal.
// It ensures both trees (IPv4-based and IPv6-based) have the same sizes and
// recursively compares their root nodes.
func (l *Lite) Equal(o *Lite) bool {
	if o == nil {
		return false
	}
	return l.liteTable.Equal(&o.liteTable)
}

// Clone returns a copy of the routing table.
func (l *Lite) Clone() *Lite {
	return &Lite{*l.liteTable.Clone()}
}

// liteTable follows the BART design but with no payload.
// It is ideal for simple IP ACLs (access-control-lists) with plain
// true/false results with the smallest memory consumption.
//
// Performance note: Do not pass IPv4-in-IPv6 addresses (e.g., ::ffff:192.0.2.1)
// as input. The methods do not perform automatic unmapping to avoid unnecessary
// overhead for the common case where native addresses are used.
// Users should unmap IPv4-in-IPv6 addresses to their native IPv4 form
// (e.g., 192.0.2.1) before calling these methods.
type liteTable[V any] struct {
	// used by -copylocks checker from `go vet`.
	_ [0]sync.Mutex

	root4 liteNode[V]
	root6 liteNode[V]

	// the number of prefixes in the routing table
	size4 int
	size6 int
}

// rootNodeByVersion, root node getter for ip version.
func (l *liteTable[V]) rootNodeByVersion(is4 bool) *liteNode[V] {
	if is4 {
		return &l.root4
	}
	return &l.root6
}

// insert adds a prefix to the table (idempotent).
// If the prefix already exists, the operation is a no-op.
func (l *liteTable[V]) Insert(pfx netip.Prefix, _ V) {
	var zero V

	if !pfx.IsValid() {
		return
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	is4 := pfx.Addr().Is4()
	n := l.rootNodeByVersion(is4)

	if exists := n.insertAtDepth(pfx, zero, 0); exists {
		return
	}

	// true insert, update size
	l.sizeUpdate(is4, 1)
}

// Delete removes the prefix and returns true if it was present, false otherwise.
func (l *liteTable[V]) Delete(pfx netip.Prefix) (_ V, found bool) {
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

	n := l.rootNodeByVersion(is4)

	// record the nodes on the path to the deleted node, needed to purge
	// and/or path compress nodes after the deletion of a prefix
	stack := [maxTreeDepth]*liteNode[V]{}

	// find the trie node
	for depth, octet := range octets {
		depth = depth & depthMask // BCE

		// push current node on stack for path recording
		stack[depth] = n

		// Last “octet” from prefix, update/insert prefix into node.
		// Note: For /32 and /128, depth never reaches lastOctetPlusOne (4/16),
		// so those are handled below via the fringe/leaf path.
		if depth == lastOctetPlusOne {
			// try to delete prefix in trie node
			_, found = n.deletePrefix(art.PfxToIdx(octet, lastBits))
			if !found {
				return
			}

			l.sizeUpdate(is4, -1)
			// remove now-empty nodes and re-path-compress upwards
			n.purgeAndCompress(stack[:depth], octets, is4)
			return zero, true
		}

		if !n.children.Test(octet) {
			return
		}
		kid := n.mustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *liteNode[V]:
			n = kid // descend down to next trie level

		case *fringeNode[V]:
			// if pfx is no fringe at this depth, fast exit
			if !isFringe(depth, pfx) {
				return
			}

			// pfx is fringe at depth, delete fringe
			n.deleteChild(octet)

			l.sizeUpdate(is4, -1)
			// remove now-empty nodes and re-path-compress upwards
			n.purgeAndCompress(stack[:depth], octets, is4)

			return zero, true

		case *leafNode[V]:
			// Attention: pfx must be masked to be comparable!
			if kid.prefix != pfx {
				return
			}

			// prefix is equal leaf, delete leaf
			n.deleteChild(octet)

			l.sizeUpdate(is4, -1)
			// remove now-empty nodes and re-path-compress upwards
			n.purgeAndCompress(stack[:depth], octets, is4)

			return zero, true

		default:
			panic("logic error, wrong node type")
		}
	}

	return
}

// Get returns the associated payload for prefix and true, or false if
// prefix is not set in the routing table.
//
//nolint:unparam
func (l *liteTable[V]) Get(pfx netip.Prefix) (_ V, ok bool) {
	var zero V

	if !pfx.IsValid() {
		return
	}

	// canonicalize the prefix
	pfx = pfx.Masked()

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()

	n := l.rootNodeByVersion(is4)

	lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)

	octets := ip.AsSlice()

	// find the trie node
	for depth, octet := range octets {
		// Last “octet” from prefix, update/insert prefix into node.
		// Note: For /32 and /128, depth never reaches lastOctetPlusOne (4/16),
		// so those are handled below via the fringe/leaf path.
		if depth == lastOctetPlusOne {
			return n.getPrefix(art.PfxToIdx(octet, lastBits))
		}

		if !n.children.Test(octet) {
			return
		}
		kid := n.mustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *liteNode[V]:
			n = kid // descend down to next trie level

		case *fringeNode[V]:
			// reached a path compressed fringe, stop traversing
			if isFringe(depth, pfx) {
				return zero, true
			}
			return

		case *leafNode[V]:
			// reached a path compressed prefix, stop traversing
			if kid.prefix == pfx {
				return zero, true
			}
			return

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
}

// Modify applies an insert or delete operation for the given prefix.
// The supplied callback decides the operation: it is called with the
// zero value and a boolean indicating whether the prefix exists.
// The callback must return a delete flag: del == false inserts or updates,
// del == true deletes the entry if it exists (otherwise no-op). Modify
// returns a boolean indicating whether the entry was actually deleted.
//
// The operation is determined by the callback function, which is called with:
//
//	val:   always the zero value
//	found: true if the prefix currently exists, false otherwise
//
// The callback returns:
//
//	val: any value returned is ignored
//	del: true to delete the entry, false to insert or no-op
//
// Modify returns:
//
//	val:     always the zero value
//	deleted: true if the entry was deleted, false otherwise
//
// Summary:
//
//	Operation | cb-input      | cb-return  | Modify-return
//	------------------------------------------------------
//	No-op:    | (zero, false) | (_, true)  | (zero, false)
//	Insert:   | (zero, false) | (_, false) | (zero, false)
//	Update:   | (zero, true)  | (_, false) | (zero, false)
//	Delete:   | (zero, true)  | (_, true)  | (zero, true)
func (l *liteTable[V]) Modify(pfx netip.Prefix, cb func(zero V, found bool) (_ V, del bool)) (zero V, deleted bool) {
	if !pfx.IsValid() {
		return zero, deleted
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	octets := ip.AsSlice()
	lastOctetPlusOne, lastBits := lastOctetPlusOneAndLastBits(pfx)

	n := l.rootNodeByVersion(is4)

	// record the nodes on the path to the deleted node, needed to purge
	// and/or path compress nodes after the deletion of a prefix
	stack := [maxTreeDepth]*liteNode[V]{}

	// find the proper trie node to update prefix
	for depth, octet := range octets {
		// push current node on stack for path recording
		stack[depth] = n

		// Last “octet” from prefix, update/insert prefix into node.
		// Note: For /32 and /128, depth never reaches lastOctetPlusOne (4/16),
		// so those are handled below via the fringe/leaf path.
		if depth == lastOctetPlusOne {
			idx := art.PfxToIdx(octet, lastBits)

			_, existed := n.getPrefix(idx)
			_, del := cb(zero, existed)

			// update size if necessary
			switch {
			case !existed && del: // no-op
				return zero, false

			case existed && del: // delete
				n.deletePrefix(idx)
				l.sizeUpdate(is4, -1)
				// remove now-empty nodes and re-path-compress upwards
				n.purgeAndCompress(stack[:depth], octets, is4)
				return zero, true

			case !existed: // insert
				n.insertPrefix(idx, zero)
				l.sizeUpdate(is4, 1)
				return zero, false

			case existed: // no-op
				return zero, false

			default:
				panic("unreachable")
			}

		}

		// go down in tight loop to last octet
		if !n.children.Test(octet) {
			// insert prefix path compressed

			_, del := cb(zero, false)
			if del {
				return zero, false // no-op
			}

			// insert
			if isFringe(depth, pfx) {
				n.insertChild(octet, newFringeNode(zero))
			} else {
				n.insertChild(octet, newLeafNode(pfx, zero))
			}

			l.sizeUpdate(is4, 1)
			return zero, false
		}

		kid := n.mustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *liteNode[V]:
			n = kid // descend down to next trie level

		case *leafNode[V]:
			if kid.prefix == pfx {
				_, del := cb(zero, true)

				if !del {
					return zero, false // no-op
				}

				// delete
				n.deleteChild(octet)

				l.sizeUpdate(is4, -1)
				// remove now-empty nodes and re-path-compress upwards
				n.purgeAndCompress(stack[:depth], octets, is4)

				return zero, true
			}

			// create new node
			// push the leaf down
			// insert new child at current leaf position (octet)
			// descend down, replace n with new child
			newNode := new(liteNode[V])
			newNode.insertAtDepth(kid.prefix, zero, depth+1)

			n.insertChild(octet, newNode)
			n = newNode

		case *fringeNode[V]:
			// update existing value if prefix is fringe
			if isFringe(depth, pfx) {
				_, del := cb(zero, true)
				if !del {
					return zero, false // no-op
				}

				// delete
				n.deleteChild(octet)

				l.sizeUpdate(is4, -1)
				// remove now-empty nodes and re-path-compress upwards
				n.purgeAndCompress(stack[:depth], octets, is4)

				return zero, true
			}

			// create new node
			// push the fringe down, it becomes a default route (idx=1)
			// insert new child at current leaf position (octet)
			// descend down, replace n with new child
			newNode := new(liteNode[V])
			newNode.insertPrefix(1, zero)

			n.insertChild(octet, newNode)
			n = newNode

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
// in the routing table contains the IP address.
func (l *liteTable[V]) Contains(ip netip.Addr) bool {
	// speed is top priority: no explicit test for ip.Isvalid
	// if ip is invalid, AsSlice() returns nil, Contains returns false.
	is4 := ip.Is4()
	n := l.rootNodeByVersion(is4)

	for _, octet := range ip.AsSlice() {
		// for contains, any lpm match is good enough, no backtracking needed
		if n.prefixes.count != 0 && n.contains(art.OctetToIdx(octet)) {
			return true
		}

		// stop traversing?
		if !n.children.Test(octet) {
			return false
		}
		kid := n.mustGetChild(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *liteNode[V]:
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

// Lookup, only for interface satisfaction.
//
//nolint:unparam
func (l *liteTable[V]) Lookup(ip netip.Addr) (_ V, ok bool) {
	var zero V
	return zero, l.Contains(ip)
}

// LookupPrefix does a route lookup (longest prefix match) for pfx and
// returns true, or false if no route matched.
//
//nolint:unparam
func (l *liteTable[V]) LookupPrefix(pfx netip.Prefix) (_ V, ok bool) {
	_, _, ok = l.lookupPrefixLPM(pfx, false)
	return
}

// LookupPrefixLPM is similar to [Lite.LookupPrefix],
// but it returns the lpm prefix in addition to value,ok.
//
// This method is about 20-30% slower than LookupPrefix and should only
// be used if the matching lpm entry is also required for other reasons.
//
// If LookupPrefixLPM is to be used for IP address lookups,
// they must be converted to /32 or /128 prefixes.
//
//nolint:unparam
func (l *liteTable[V]) LookupPrefixLPM(pfx netip.Prefix) (lpmPfx netip.Prefix, _ V, ok bool) {
	return l.lookupPrefixLPM(pfx, true)
}

//nolint:unparam
func (l *liteTable[V]) lookupPrefixLPM(pfx netip.Prefix, withLPM bool) (lpmPfx netip.Prefix, _ V, ok bool) {
	var zero V

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

	n := l.rootNodeByVersion(is4)

	// record path to leaf node
	stack := [maxTreeDepth]*liteNode[V]{}

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
		case *liteNode[V]:
			n = kid
			continue LOOP // descend down to next trie level

		case *leafNode[V]:
			// reached a path compressed prefix, stop traversing
			if kid.prefix.Bits() > bits || !kid.prefix.Contains(ip) {
				break LOOP
			}
			return kid.prefix, zero, true

		case *fringeNode[V]:
			// the bits of the fringe are defined by the depth
			// maybe the LPM isn't needed, saves some cycles
			fringeBits := (depth + 1) << 3
			if fringeBits > bits {
				break LOOP
			}

			// the LPM isn't needed, saves some cycles
			if !withLPM {
				return netip.Prefix{}, zero, true
			}

			// sic, get the LPM prefix back, it costs some cycles!
			fringePfx := cidrForFringe(octets, depth, is4, octet)
			return fringePfx, zero, true

		default:
			panic("logic error, wrong node type")
		}
	}

	// start backtracking, unwind the stack
	for ; depth >= 0; depth-- {
		depth = depth & depthMask // BCE

		n = stack[depth]

		// longest prefix match, skip if node has no prefixes
		if n.prefixes.count == 0 {
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
		if topIdx, ok := n.prefixes.IntersectionTop(&lpm.LookupTbl[idx]); ok {
			// called from LookupPrefix
			if !withLPM {
				return netip.Prefix{}, zero, ok
			}

			// called from LookupPrefixLPM

			// get the bits from depth and top idx
			pfxBits := int(art.PfxBits(depth, topIdx))

			// calculate the lpmPfx from incoming ip and new mask
			lpmPfx, _ = ip.Prefix(pfxBits)
			return lpmPfx, zero, ok
		}
	}

	return
}

// OverlapsPrefix reports whether any route in the table overlaps with the given pfx or vice versa.
//
// The check is bidirectional: it returns true if the input prefix is covered by an existing
// route, or if any stored route is itself contained within the input prefix.
//
// Internally, the function normalizes the prefix and descends the relevant trie branch,
// using stride-based logic to identify overlap without performing a full lookup.
//
// This is useful for containment tests, route validation, or policy checks using prefix
// semantics without retrieving exact matches.
func (l *liteTable[V]) OverlapsPrefix(pfx netip.Prefix) bool {
	if !pfx.IsValid() {
		return false
	}

	// canonicalize the prefix
	pfx = pfx.Masked()

	is4 := pfx.Addr().Is4()
	n := l.rootNodeByVersion(is4)

	return n.overlapsPrefixAtDepth(pfx, 0)
}

// Overlaps reports whether any route in the receiver table overlaps
// with a route in the other table, in either direction.
//
// The overlap check is bidirectional: it returns true if any IP prefix
// in the receiver is covered by the other table, or vice versa.
// This includes partial overlaps, exact matches, and supernet/subnet relationships.
//
// Both IPv4 and IPv6 route trees are compared independently. If either
// tree has overlapping routes, the function returns true.
//
// This is useful for conflict detection, policy enforcement,
// or validating mutually exclusive routing domains.
func (l *liteTable[V]) Overlaps(o *liteTable[V]) bool {
	if o == nil {
		return false
	}
	return l.Overlaps4(o) || l.Overlaps6(o)
}

// Overlaps4 is like [Lite.Overlaps] but for the v4 routing table only.
func (l *liteTable[V]) Overlaps4(o *liteTable[V]) bool {
	if o == nil || l.size4 == 0 || o.size4 == 0 {
		return false
	}
	return l.root4.overlaps(&o.root4, 0)
}

// Overlaps6 is like [Lite.Overlaps] but for the v6 routing table only.
func (l *liteTable[V]) Overlaps6(o *liteTable[V]) bool {
	if o == nil || l.size6 == 0 || o.size6 == 0 {
		return false
	}
	return l.root6.overlaps(&o.root6, 0)
}

// Union merges another routing table into the receiver table, modifying it in-place.
//
// All prefixes and values from the other table (o) are inserted into the receiver.
// If a duplicate prefix exists in both tables, the value from o replaces the existing entry.
// This duplicate is shallow-copied by default, but if the value type V implements the
// Cloner interface, the value is deeply cloned before insertion. See also Lite.Clone.
func (l *liteTable[V]) Union(o *liteTable[V]) {
	if o == nil || (o.size4 == 0 && o.size6 == 0) {
		return
	}

	// Create a cloning function for deep copying values;
	// returns nil if V does not implement the Cloner interface.
	cloneFn := cloneFnFactory[V]()
	if cloneFn == nil {
		cloneFn = copyVal
	}

	dup4 := l.root4.unionRec(cloneFn, &o.root4, 0)
	dup6 := l.root6.unionRec(cloneFn, &o.root6, 0)

	l.size4 += o.size4 - dup4
	l.size6 += o.size6 - dup6
}

// UnionPersist is similar to [Union] but the receiver isn't modified.
//
// All nodes touched during union are cloned and a new liteTable is returned.
// If o is nil or empty, no nodes are touched and the receiver may be
// returned unchanged.
func (l *liteTable[V]) UnionPersist(o *liteTable[V]) *liteTable[V] {
	if o == nil || (o.size4 == 0 && o.size6 == 0) {
		return l
	}

	// Create a cloning function for deep copying values;
	// returns nil if V does not implement the Cloner interface.
	cloneFn := cloneFnFactory[V]()

	// new liteTable with root nodes just copied.
	pt := &liteTable[V]{
		root4: l.root4,
		root6: l.root6,
		//
		size4: l.size4,
		size6: l.size6,
	}

	// only clone the root node if there is something to union
	if o.size4 != 0 {
		pt.root4 = *l.root4.cloneFlat(cloneFn)
	}
	if o.size6 != 0 {
		pt.root6 = *l.root6.cloneFlat(cloneFn)
	}

	if cloneFn == nil {
		cloneFn = copyVal
	}

	dup4 := pt.root4.unionRecPersist(cloneFn, &o.root4, 0)
	dup6 := pt.root6.unionRecPersist(cloneFn, &o.root6, 0)

	pt.size4 += o.size4 - dup4
	pt.size6 += o.size6 - dup6

	return pt
}

// Clone returns a copy of the routing table.
func (l *liteTable[V]) Clone() *liteTable[V] {
	if l == nil {
		return nil
	}

	c := new(liteTable[V])

	c.root4 = *l.root4.cloneRec(nil)
	c.root6 = *l.root6.cloneRec(nil)

	c.size4 = l.size4
	c.size6 = l.size6

	return c
}

// Equal checks whether two tables are structurally and semantically equal.
// It ensures both trees (IPv4-based and IPv6-based) have the same sizes and
// recursively compares their root nodes.
func (l *liteTable[V]) Equal(o *liteTable[V]) bool {
	if o == nil || l.size4 != o.size4 || l.size6 != o.size6 {
		return false
	}

	return l.root4.equalRec(&o.root4) && l.root6.equalRec(&o.root6)
}

// All returns an iterator over all prefix–value pairs in the table.
//
// The entries from both IPv4 and IPv6 subtries are yielded using an internal recursive traversal.
// The iteration order is unspecified and may vary between calls; for a stable order, use AllSorted.
//
// You can use All directly in a for-range loop without providing a yield function.
// The Go compiler automatically synthesizes the yield callback for you:
//
//	for prefix, _ := range t.All() {
//	    fmt.Println(prefix)
//	}
//
// Under the hood, the loop body is passed as a yield function to the iterator.
// If you break or return from the loop, iteration stops early as expected.
//
// IMPORTANT: Modifying or deleting entries during iteration is not allowed,
// as this would interfere with the internal traversal and may corrupt or
// prematurely terminate the iteration.
//
// If mutation of the table during traversal is required,
// use [Lite.WalkPersist] instead.
func (l *liteTable[V]) All() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = l.root4.allRec(stridePath{}, 0, true, yield) && l.root6.allRec(stridePath{}, 0, false, yield)
	}
}

// All4 is like [Lite.All] but only for the v4 routing table.
func (l *liteTable[V]) All4() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = l.root4.allRec(stridePath{}, 0, true, yield)
	}
}

// All6 is like [Lite.All] but only for the v6 routing table.
func (l *liteTable[V]) All6() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = l.root6.allRec(stridePath{}, 0, false, yield)
	}
}

// AllSorted returns an iterator over all prefix–value pairs in the table,
// ordered in canonical CIDR prefix sort order.
//
// This can be used directly with a for-range loop; the Go compiler provides the yield function implicitly.
//
//	for prefix, value := range t.AllSorted() {
//	    fmt.Println(prefix, value)
//	}
//
// The traversal is stable and predictable across calls.
// Iteration stops early if you break out of the loop.
func (l *liteTable[V]) AllSorted() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = l.root4.allRecSorted(stridePath{}, 0, true, yield) &&
			l.root6.allRecSorted(stridePath{}, 0, false, yield)
	}
}

// AllSorted4 is like [Lite.AllSorted] but only for the v4 routing table.
func (l *liteTable[V]) AllSorted4() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = l.root4.allRecSorted(stridePath{}, 0, true, yield)
	}
}

// AllSorted6 is like [Lite.AllSorted] but only for the v6 routing table.
func (l *liteTable[V]) AllSorted6() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = l.root6.allRecSorted(stridePath{}, 0, false, yield)
	}
}

// Size returns the prefix count.
func (l *liteTable[V]) Size() int {
	return l.size4 + l.size6
}

// Size4 returns the IPv4 prefix count.
func (l *liteTable[V]) Size4() int {
	return l.size4
}

// Size6 returns the IPv6 prefix count.
func (l *liteTable[V]) Size6() int {
	return l.size6
}

func (l *liteTable[V]) sizeUpdate(is4 bool, delta int) {
	if is4 {
		l.size4 += delta
		return
	}
	l.size6 += delta
}

// String returns a hierarchical tree diagram of the ordered CIDRs
// as string, just a wrapper for [Lite.Fprint].
// If Fprint returns an error, String panics.
func (l *liteTable[V]) String() string {
	w := new(strings.Builder)
	if err := l.Fprint(w); err != nil {
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
func (l *liteTable[V]) Fprint(w io.Writer) error {
	if w == nil {
		return fmt.Errorf("nil writer")
	}
	if l == nil {
		return nil
	}

	// v4
	if err := l.fprint(w, true); err != nil {
		return err
	}

	// v6
	if err := l.fprint(w, false); err != nil {
		return err
	}

	return nil
}

// fprint is the version dependent adapter to fprintRec.
func (l *liteTable[V]) fprint(w io.Writer, is4 bool) error {
	n := l.rootNodeByVersion(is4)
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

	// liteTable has no payload - printVals set to false
	return fprintRec(n, w, startParent, "", false)
}

// MarshalText implements the [encoding.TextMarshaler] interface,
// just a wrapper for [Lite.Fprint].
func (l *liteTable[V]) MarshalText() ([]byte, error) {
	w := new(bytes.Buffer)
	if err := l.Fprint(w); err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

// MarshalJSON dumps the table into two sorted lists: for ipv4 and ipv6.
// Every root and subnet is an array, not a map, because the order matters.
func (l *liteTable[V]) MarshalJSON() ([]byte, error) {
	if l == nil {
		return []byte("null"), nil
	}

	result := struct {
		Ipv4 []DumpListNode[V] `json:"ipv4,omitempty"`
		Ipv6 []DumpListNode[V] `json:"ipv6,omitempty"`
	}{
		Ipv4: l.DumpList4(),
		Ipv6: l.DumpList6(),
	}

	buf, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// DumpList4 dumps the ipv4 tree into a list of roots and their subnets.
// It can be used to analyze the tree or build the text or json serialization.
func (l *liteTable[V]) DumpList4() []DumpListNode[V] {
	if l == nil {
		return nil
	}
	return dumpListRec(&l.root4, 0, stridePath{}, 0, true)
}

// DumpList6 dumps the ipv6 tree into a list of roots and their subnets.
// It can be used to analyze the tree or build custom json representation.
func (l *liteTable[V]) DumpList6() []DumpListNode[V] {
	if l == nil {
		return nil
	}
	return dumpListRec(&l.root6, 0, stridePath{}, 0, false)
}

// dumpString is just a wrapper for dump.
//
//nolint:unused
func (l *liteTable[V]) dumpString() string {
	w := new(strings.Builder)
	l.dump(w)

	return w.String()
}

// dump the table structure and all the nodes to w.
func (l *liteTable[V]) dump(w io.Writer) {
	if l == nil {
		return
	}

	if l.size4 > 0 {
		stats := nodeStatsRec(&l.root4)
		fmt.Fprintln(w)
		fmt.Fprintf(w, "### IPv4: size(%d), nodes(%d), pfxs(%d), leaves(%d), fringes(%d)",
			l.size4, stats.nodes, stats.pfxs, stats.leaves, stats.fringes)

		// liteTable has no payload - printVals set to false
		dumpRec(&l.root4, w, stridePath{}, 0, true, false)
	}

	if l.size6 > 0 {
		stats := nodeStatsRec(&l.root6)
		fmt.Fprintln(w)
		fmt.Fprintf(w, "### IPv6: size(%d), nodes(%d), pfxs(%d), leaves(%d), fringes(%d)",
			l.size6, stats.nodes, stats.pfxs, stats.leaves, stats.fringes)

		// liteTable has no payload - printVals set to false
		dumpRec(&l.root6, w, stridePath{}, 0, false, false)
	}
}
