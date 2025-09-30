// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Usage: go generate -tags=ignore ./...
//go:generate ./scripts/generate-table-methods.sh
//go:build ignore

package bart

// ### GENERATE DELETE START ###

// stub code for generator types and methods
// useful for gopls during development, deleted during go generate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/netip"
	"strings"
)

type _NODE_TYPE[V any] struct{}

type _TABLE_TYPE[V any] struct {
	root4 _NODE_TYPE[V]
	root6 _NODE_TYPE[V]
	size4 int
	size6 int
}

func (n *_NODE_TYPE[V]) isEmpty() (_ bool)                                                 { return }
func (n *_NODE_TYPE[V]) prefixCount() (_ int)                                              { return }
func (n *_NODE_TYPE[V]) childCount() (_ int)                                               { return }
func (n *_NODE_TYPE[V]) getPrefix(uint8) (_ V, _ bool)                                     { return }
func (n *_NODE_TYPE[V]) getChild(uint8) (_ any, _ bool)                                    { return }
func (n *_NODE_TYPE[V]) mustGetPrefix(uint8) (_ V)                                         { return }
func (n *_NODE_TYPE[V]) mustGetChild(uint8) (_ any)                                        { return }
func (n *_NODE_TYPE[V]) insert(netip.Prefix, V, int) (_ bool)                              { return }
func (n *_NODE_TYPE[V]) delete(netip.Prefix) (_ V, _ bool)                                 { return }
func (n *_NODE_TYPE[V]) insertPersist(cloneFunc[V], netip.Prefix, V, int) (_ bool)         { return }
func (n *_NODE_TYPE[V]) deletePersist(cloneFunc[V], netip.Prefix) (_ V, _ bool)            { return }
func (n *_NODE_TYPE[V]) get(netip.Prefix) (_ V, _ bool)                                    { return }
func (n *_NODE_TYPE[V]) overlapsPrefixAtDepth(netip.Prefix, int) (_ bool)                  { return }
func (n *_NODE_TYPE[V]) overlaps(*_NODE_TYPE[V], int) (_ bool)                             { return }
func (n *_NODE_TYPE[V]) unionRec(cloneFunc[V], *_NODE_TYPE[V], int) (_ int)                { return }
func (n *_NODE_TYPE[V]) unionRecPersist(cloneFunc[V], *_NODE_TYPE[V], int) (_ int)         { return }
func (n *_NODE_TYPE[V]) equalRec(*_NODE_TYPE[V]) (_ bool)                                  { return }
func (n *_NODE_TYPE[V]) cloneRec(cloneFunc[V]) (_ *_NODE_TYPE[V])                          { return }
func (n *_NODE_TYPE[V]) cloneFlat(cloneFunc[V]) (_ *_NODE_TYPE[V])                         { return }
func (n *_NODE_TYPE[V]) getChildAddrs(*[256]uint8) (_ []uint8)                             { return }
func (n *_NODE_TYPE[V]) getIndices(*[256]uint8) (_ []uint8)                                { return }
func (n *_NODE_TYPE[V]) allChildren() (_ iter.Seq2[uint8, any])                            { return }
func (n *_NODE_TYPE[V]) allIndices() (_ iter.Seq2[uint8, V])                               { return }
func (n *_NODE_TYPE[V]) contains(uint8) (_ bool)                                           { return }
func (n *_NODE_TYPE[V]) lookup(uint8) (_ V, _ bool)                                        { return }
func (n *_NODE_TYPE[V]) lookupIdx(uint8) (_ uint8, _ V, _ bool)                            { return }
func (n *_NODE_TYPE[V]) supernets(netip.Prefix, func(netip.Prefix, V) bool)                { return }
func (n *_NODE_TYPE[V]) subnets(netip.Prefix, func(netip.Prefix, V) bool)                  { return }
func (n *_NODE_TYPE[V]) allRec(stridePath, int, bool, func(netip.Prefix, V) bool) (_ bool) { return }
func (n *_NODE_TYPE[V]) allRecSorted(stridePath, int, bool, func(netip.Prefix, V) bool) (_ bool) {
	return
}
func (n *_NODE_TYPE[V]) modify(netip.Prefix, func(V, bool) (V, bool)) (_ int, _ V, _ bool) { return }
func (n *_NODE_TYPE[V]) modifyPersist(cloneFunc[V], netip.Prefix, func(V, bool) (V, bool)) (_ int, _ V, _ bool) {
	return
}

func (t *_TABLE_TYPE[V]) rootNodeByVersion(is4 bool) (n *_NODE_TYPE[V]) { return }

// ### GENERATE DELETE END ###

func (t *_TABLE_TYPE[V]) sizeUpdate(is4 bool, delta int) {
	if is4 {
		t.size4 += delta
		return
	}
	t.size6 += delta
}

// Insert adds or updates a prefix-value pair in the routing table.
// If the prefix already exists, its value is updated; otherwise a new entry is created.
// Invalid prefixes are silently ignored.
//
// The prefix is automatically canonicalized using pfx.Masked() to ensure
// consistent behavior regardless of host bits in the input.
func (t *_TABLE_TYPE[V]) Insert(pfx netip.Prefix, val V) {
	if !pfx.IsValid() {
		return
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	is4 := pfx.Addr().Is4()
	n := t.rootNodeByVersion(is4)

	if exists := n.insert(pfx, val, 0); exists {
		return
	}

	// true insert, update size
	t.sizeUpdate(is4, 1)
}

// InsertPersist is similar to Insert but the receiver isn't modified.
//
// All nodes touched during insert are cloned and a new _TABLE_TYPE is returned.
// This is not a full [_TABLE_TYPE.Clone], all untouched nodes are still referenced
// from both Tables.
//
// If the payload type V contains pointers or needs deep copying,
// it must implement the [bart.Cloner] interface to support correct cloning.
//
// This is orders of magnitude slower than Insert,
// typically taking μsec instead of nsec.
//
// The bulk table load could be done with [_TABLE_TYPE.Insert] and then you can
// use [_TABLE_TYPE.InsertPersist], [_TABLE_TYPE.ModifyPersist] and
// [_TABLE_TYPE.DeletePersist] for further lock-free ops.
func (t *_TABLE_TYPE[V]) InsertPersist(pfx netip.Prefix, val V) *_TABLE_TYPE[V] {
	if !pfx.IsValid() {
		return t
	}

	// canonicalize prefix
	pfx = pfx.Masked()
	is4 := pfx.Addr().Is4()

	// share size counters; root nodes cloned selectively.
	pt := &_TABLE_TYPE[V]{
		size4: t.size4,
		size6: t.size6,
	}

	// Create a cloning function for deep copying values;
	// returns nil if V does not implement the Cloner interface.
	cloneFn := cloneFnFactory[V]()

	// Clone root node corresponding to the IP version, for copy-on-write.
	n := &pt.root4

	if is4 {
		pt.root4 = *t.root4.cloneFlat(cloneFn)
		pt.root6 = t.root6
	} else {
		pt.root4 = t.root4
		pt.root6 = *t.root6.cloneFlat(cloneFn)

		n = &pt.root6
	}

	if !n.insertPersist(cloneFn, pfx, val, 0) {
		pt.sizeUpdate(is4, 1)
	}

	return pt
}

// Delete removes the exact prefix pfx from the table.
//
// This is an exact-match operation (no LPM). If pfx exists, the entry is
// removed and the previous value is returned with ok=true. If pfx does not
// exist or pfx is invalid, the table is left unchanged and the
// zero value of V and ok=false are returned.
//
// The prefix is canonicalized (Masked) before lookup.
func (t *_TABLE_TYPE[V]) Delete(pfx netip.Prefix) (val V, exists bool) {
	if !pfx.IsValid() {
		return val, exists
	}

	// canonicalize prefix
	pfx = pfx.Masked()
	is4 := pfx.Addr().Is4()

	n := t.rootNodeByVersion(is4)
	val, exists = n.delete(pfx)

	if exists {
		t.sizeUpdate(is4, -1)
	}
	return val, exists
}

// Get performs an exact-prefix lookup and returns whether the exact
// prefix exists. The prefix is canonicalized (Masked) before lookup.
//
// This is an exact-match operation (no LPM). The prefix must match exactly
// in both address and prefix length to be found. If pfx exists, the
// associated value (zero value for Lite) and found=true is returned.
// If pfx does not exist or pfx is invalid, the zero value for V and
// found=false is returned.
//
// For longest-prefix-match (LPM) lookups, use Contains(ip), Lookup(ip),
// LookupPrefix(pfx) or LookupPrefixLPM(pfx) instead.
func (t *_TABLE_TYPE[V]) Get(pfx netip.Prefix) (val V, exists bool) {
	if !pfx.IsValid() {
		return val, exists
	}
	// canonicalize prefix
	pfx = pfx.Masked()

	is4 := pfx.Addr().Is4()
	n := t.rootNodeByVersion(is4)

	return n.get(pfx)
}

// DeletePersist is similar to Delete but does not modify the receiver.
//
// It performs a copy-on-write delete operation, cloning all nodes touched during
// deletion and returning a new _TABLE_TYPE reflecting the change.
//
// If the payload type V contains pointers or requires deep copying,
// it must implement the [bart.Cloner] interface for correct cloning.
//
// Due to cloning overhead, DeletePersist is significantly slower than Delete,
// typically taking μsec instead of nsec.
func (t *_TABLE_TYPE[V]) DeletePersist(pfx netip.Prefix) (pt *_TABLE_TYPE[V], val V, found bool) {
	if !pfx.IsValid() {
		return t, val, false
	}

	// canonicalize prefix
	pfx = pfx.Masked()
	is4 := pfx.Addr().Is4()

	// Preflight check: avoid cloning if prefix doesn't exist
	node := t.rootNodeByVersion(is4)
	val, found = node.get(pfx)
	if !found {
		return t, val, false
	}

	// share size counters; root nodes cloned selectively.
	pt = &_TABLE_TYPE[V]{
		size4: t.size4,
		size6: t.size6,
	}

	// Create a cloning function for deep copying values;
	// returns nil if V does not implement the Cloner interface.
	cloneFn := cloneFnFactory[V]()

	// Clone root node corresponding to the IP version, for copy-on-write.
	n := &pt.root4
	if is4 {
		pt.root4 = *t.root4.cloneFlat(cloneFn)
		pt.root6 = t.root6
	} else {
		pt.root4 = t.root4
		pt.root6 = *t.root6.cloneFlat(cloneFn)

		n = &pt.root6
	}

	_, exists := n.deletePersist(cloneFn, pfx)
	if exists {
		pt.sizeUpdate(is4, -1)
	}

	return pt, val, exists
}

// WalkPersist traverses all prefix/value pairs in the table and calls the
// provided callback function for each entry. The callback receives the
// current persistent table, the prefix, and the associated value.
//
// The callback must return a (potentially updated) persistent table and a
// boolean flag indicating whether traversal should continue. Returning
// false stops the iteration early.
//
// IMPORTANT: It is the responsibility of the callback implementation to only
// use persistent Table operations (e.g. InsertPersist, DeletePersist,
// ModifyPersist, ...). Using mutating methods like Modify or Delete
// inside the callback would break the iteration and may lead
// to inconsistent results.
//
// Example:
//
//	pt := t.WalkPersist(func(pt *_TABLE_TYPE[int], pfx netip.Prefix, val int) (*_TABLE_TYPE[int], bool) {
//		switch {
//		// Stop iterating if value is <0
//		case val < 0:
//			return pt, false
//
//		// Delete entries with value 0
//		case val == 0:
//			pt, _, _ = pt.DeletePersist(pfx)
//
//		// modify even values by doubling them
//		case val%2 == 0:
//			pt, _, _ = pt.ModifyPersist(pfx, func(oldVal int, _ bool) (int, bool) {
//				return oldVal * 2, false
//			})
//
//		// Leave odd values unchanged
//		default:
//			// no-op
//		}
//
//		// Continue iterating
//		return pt, true
//	})
func (t *_TABLE_TYPE[V]) WalkPersist(fn func(*_TABLE_TYPE[V], netip.Prefix, V) (*_TABLE_TYPE[V], bool)) *_TABLE_TYPE[V] {
	// no-op, callback is nil
	if fn == nil {
		return t
	}

	pt := t
	var proceed bool
	for pfx, val := range t.All() {
		if pt, proceed = fn(pt, pfx, val); !proceed {
			break
		}
	}
	return pt
}

// Modify applies an insert, update, or delete operation for the value
// associated with the given prefix. The supplied callback decides the
// operation: it is called with the current value (or zero if not found)
// and a boolean indicating whether the prefix exists. The callback must
// return a new value and a delete flag: del == false inserts or updates,
// del == true deletes the entry if it exists (otherwise no-op).
//
// Modify returns the resulting value and a boolean indicating whether the
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
func (t *_TABLE_TYPE[V]) Modify(pfx netip.Prefix, cb func(_ V, ok bool) (_ V, del bool)) (_ V, deleted bool) {
	if !pfx.IsValid() {
		return
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	is4 := pfx.Addr().Is4()

	n := t.rootNodeByVersion(is4)

	delta, val, deleted := n.modify(pfx, cb)
	t.sizeUpdate(is4, delta)

	return val, deleted
}

// ModifyPersist is similar to Modify but the receiver isn't modified.
func (t *_TABLE_TYPE[V]) ModifyPersist(pfx netip.Prefix, cb func(_ V, ok bool) (_ V, del bool)) (pt *_TABLE_TYPE[V], _ V, deleted bool) {
	var zero V
	if !pfx.IsValid() {
		return t, zero, false
	}

	// make a cheap test in front of expensive operation
	oldVal, ok := t.Get(pfx)
	val := oldVal

	// to clone or not to clone ...
	cloneFn := cloneFnFactory[V]()
	if cloneFn != nil && ok {
		val = cloneFn(oldVal)
	}

	newVal, del := cb(val, ok)

	switch {
	case !ok && del: // no-op
		return t, zero, false

	case !ok && !del: // insert
		return t.InsertPersist(pfx.Masked(), newVal), newVal, false

	case ok && !del: // update
		return t.InsertPersist(pfx.Masked(), newVal), oldVal, false

	case ok && del: // delete
		pt, _, _ := t.DeletePersist(pfx.Masked())
		return pt, oldVal, true
	}

	panic("unreachable")
}

// Supernets returns an iterator over all supernet routes that cover the given prefix pfx.
//
// The traversal searches both exact-length and shorter (less specific) prefixes that
// include pfx. Starting from the most specific position in the trie,
// it walks upward through parent nodes and yields any matching entries found at each level.
//
// The iteration order is reverse-CIDR: from longest prefix match (LPM) towards
// least-specific routes.
//
// This can be used to enumerate all covering supernet routes in routing-based
// policy engines, diagnostics tools, or fallback resolution logic.
//
// Example:
//
//	for supernet, val := range table.Supernets(netip.MustParsePrefix("192.0.2.128/25")) {
//	    fmt.Println("Covered by:", supernet, "->", val)
//	}
//
// The iteration can be stopped early by breaking from the range loop.
// Returns an empty iterator if the prefix is invalid.
func (t *_TABLE_TYPE[V]) Supernets(pfx netip.Prefix) iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		if !pfx.IsValid() {
			return
		}

		// canonicalize the prefix
		pfx = pfx.Masked()

		is4 := pfx.Addr().Is4()
		n := t.rootNodeByVersion(is4)

		n.supernets(pfx, yield)
	}
}

// Subnets returns an iterator over all subnets of the given prefix
// in natural CIDR sort order. This includes prefixes of the same length
// (exact match) and longer (more specific) prefixes that are contained
// within the given prefix.
//
// Example:
//
//	for sub, val := range table.Subnets(netip.MustParsePrefix("10.0.0.0/8")) {
//	    fmt.Println("Covered:", sub, "->", val)
//	}
//
// The iteration can be stopped early by breaking from the range loop.
// Returns an empty iterator if the prefix is invalid.
func (t *_TABLE_TYPE[V]) Subnets(pfx netip.Prefix) iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		if !pfx.IsValid() {
			return
		}

		pfx = pfx.Masked()
		is4 := pfx.Addr().Is4()

		n := t.rootNodeByVersion(is4)
		n.subnets(pfx, yield)
	}
}

// OverlapsPrefix reports whether any prefix in the routing table overlaps with
// the given prefix. Two prefixes overlap if they share any IP addresses.
//
// The check is bidirectional: it returns true if the input prefix is covered by an existing
// route, or if any stored route is itself contained within the input prefix.
//
// Internally, the function normalizes the prefix and descends the relevant trie branch,
// using stride-based logic to identify overlap without performing a full lookup.
//
// This is useful for containment tests, route validation, or policy checks using prefix
// semantics without retrieving exact matches.
func (t *_TABLE_TYPE[V]) OverlapsPrefix(pfx netip.Prefix) bool {
	if !pfx.IsValid() {
		return false
	}

	// canonicalize the prefix
	pfx = pfx.Masked()

	is4 := pfx.Addr().Is4()
	n := t.rootNodeByVersion(is4)

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
func (t *_TABLE_TYPE[V]) Overlaps(o *_TABLE_TYPE[V]) bool {
	if o == nil {
		return false
	}
	return t.Overlaps4(o) || t.Overlaps6(o)
}

// Overlaps4 is like [_TABLE_TYPE.Overlaps] but for the v4 routing table only.
func (t *_TABLE_TYPE[V]) Overlaps4(o *_TABLE_TYPE[V]) bool {
	if o == nil || t.size4 == 0 || o.size4 == 0 {
		return false
	}
	return t.root4.overlaps(&o.root4, 0)
}

// Overlaps6 is like [_TABLE_TYPE.Overlaps] but for the v6 routing table only.
func (t *_TABLE_TYPE[V]) Overlaps6(o *_TABLE_TYPE[V]) bool {
	if o == nil || t.size6 == 0 || o.size6 == 0 {
		return false
	}
	return t.root6.overlaps(&o.root6, 0)
}

// Union merges another routing table into the receiver table, modifying it in-place.
//
// All prefixes and values from the other table (o) are inserted into the receiver.
// If a duplicate prefix exists in both tables, the value from o replaces the existing entry.
// This duplicate is shallow-copied by default, but if the value type V implements the
// Cloner interface, the value is deeply cloned before insertion. See also _TABLE_TYPE.Clone.
func (t *_TABLE_TYPE[V]) Union(o *_TABLE_TYPE[V]) {
	if o == nil || o == t || (o.size4 == 0 && o.size6 == 0) {
		return
	}

	// Create a cloning function for deep copying values;
	// returns nil if V does not implement the Cloner interface.
	cloneFn := cloneFnFactory[V]()
	if cloneFn == nil {
		cloneFn = copyVal
	}

	dup4 := t.root4.unionRec(cloneFn, &o.root4, 0)
	dup6 := t.root6.unionRec(cloneFn, &o.root6, 0)

	t.size4 += o.size4 - dup4
	t.size6 += o.size6 - dup6
}

// UnionPersist is similar to [Union] but the receiver isn't modified.
//
// All nodes touched during union are cloned and a new *_TABLE_TYPE is returned.
// If o is nil or empty, no nodes are touched and the receiver may be
// returned unchanged.
func (t *_TABLE_TYPE[V]) UnionPersist(o *_TABLE_TYPE[V]) *_TABLE_TYPE[V] {
	if o == nil || o == t || (o.size4 == 0 && o.size6 == 0) {
		return t
	}

	// Create a cloning function for deep copying values;
	// returns nil if V does not implement the Cloner interface.
	cloneFn := cloneFnFactory[V]()

	// new _TABLE_TYPE with root nodes just copied.
	pt := &_TABLE_TYPE[V]{
		root4: t.root4,
		root6: t.root6,
		//
		size4: t.size4,
		size6: t.size6,
	}

	// only clone the root node if there is something to union
	if o.size4 != 0 {
		pt.root4 = *t.root4.cloneFlat(cloneFn)
	}
	if o.size6 != 0 {
		pt.root6 = *t.root6.cloneFlat(cloneFn)
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

// Equal checks whether two tables are structurally and semantically equal.
// It ensures both trees (IPv4-based and IPv6-based) have the same sizes and
// recursively compares their root nodes.
func (t *_TABLE_TYPE[V]) Equal(o *_TABLE_TYPE[V]) bool {
	if o == nil || t.size4 != o.size4 || t.size6 != o.size6 {
		return false
	}
	if o == t {
		return true
	}

	return t.root4.equalRec(&o.root4) && t.root6.equalRec(&o.root6)
}

// Clone returns a copy of the routing table.
// The payload of type V is shallow copied, but if type V implements the [Cloner] interface,
// the values are cloned.
func (t *_TABLE_TYPE[V]) Clone() *_TABLE_TYPE[V] {
	if t == nil {
		return nil
	}

	c := new(_TABLE_TYPE[V])

	cloneFn := cloneFnFactory[V]()

	c.root4 = *t.root4.cloneRec(cloneFn)
	c.root6 = *t.root6.cloneRec(cloneFn)

	c.size4 = t.size4
	c.size6 = t.size6

	return c
}

// Size returns the prefix count.
func (t *_TABLE_TYPE[V]) Size() int {
	return t.size4 + t.size6
}

// Size4 returns the IPv4 prefix count.
func (t *_TABLE_TYPE[V]) Size4() int {
	return t.size4
}

// Size6 returns the IPv6 prefix count.
func (t *_TABLE_TYPE[V]) Size6() int {
	return t.size6
}

// All returns an iterator over all prefix–value pairs in the table.
//
// The entries from both IPv4 and IPv6 subtries are yielded using an internal recursive traversal.
// The iteration order is unspecified and may vary between calls; for a stable order, use AllSorted.
//
// You can use All directly in a for-range loop without providing a yield function.
// The Go compiler automatically synthesizes the yield callback for you:
//
//	for prefix, value := range t.All() {
//	    fmt.Println(prefix, value)
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
// use [_TABLE_TYPE.WalkPersist] instead.
func (t *_TABLE_TYPE[V]) All() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root4.allRec(stridePath{}, 0, true, yield) && t.root6.allRec(stridePath{}, 0, false, yield)
	}
}

// All4 is like [_TABLE_TYPE.All] but only for the v4 routing table.
func (t *_TABLE_TYPE[V]) All4() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root4.allRec(stridePath{}, 0, true, yield)
	}
}

// All6 is like [_TABLE_TYPE.All] but only for the v6 routing table.
func (t *_TABLE_TYPE[V]) All6() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root6.allRec(stridePath{}, 0, false, yield)
	}
}

// AllSorted returns an iterator over all prefix–value pairs in the table,
// ordered in canonical CIDR prefix sort order.
//
// This can be used directly with a for-range loop;
// the Go compiler provides the yield function implicitly:
//
//	for prefix, value := range t.AllSorted() {
//	    fmt.Println(prefix, value)
//	}
//
// The traversal is stable and predictable across calls.
// Iteration stops early if you break out of the loop.
//
// Modifying the table during iteration may produce undefined results.
func (t *_TABLE_TYPE[V]) AllSorted() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root4.allRecSorted(stridePath{}, 0, true, yield) &&
			t.root6.allRecSorted(stridePath{}, 0, false, yield)
	}
}

// AllSorted4 is like [_TABLE_TYPE.AllSorted] but only for the v4 routing table.
func (t *_TABLE_TYPE[V]) AllSorted4() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root4.allRecSorted(stridePath{}, 0, true, yield)
	}
}

// AllSorted6 is like [_TABLE_TYPE.AllSorted] but only for the v6 routing table.
func (t *_TABLE_TYPE[V]) AllSorted6() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root6.allRecSorted(stridePath{}, 0, false, yield)
	}
}

// String returns a hierarchical tree diagram of the ordered CIDRs
// as string, just a wrapper for [_TABLE_TYPE.Fprint].
// If Fprint returns an error, String panics.
func (t *_TABLE_TYPE[V]) String() string {
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
func (t *_TABLE_TYPE[V]) Fprint(w io.Writer) error {
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
func (t *_TABLE_TYPE[V]) fprint(w io.Writer, is4 bool) error {
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
// just a wrapper for [_TABLE_TYPE.Fprint].
func (t *_TABLE_TYPE[V]) MarshalText() ([]byte, error) {
	w := new(bytes.Buffer)
	if err := t.Fprint(w); err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

// MarshalJSON dumps the table into two sorted lists: for ipv4 and ipv6.
// Every root and subnet is an array, not a map, because the order matters.
func (t *_TABLE_TYPE[V]) MarshalJSON() ([]byte, error) {
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
func (t *_TABLE_TYPE[V]) DumpList4() []DumpListNode[V] {
	if t == nil {
		return nil
	}
	return dumpListRec(&t.root4, 0, stridePath{}, 0, true)
}

// DumpList6 dumps the ipv6 tree into a list of roots and their subnets.
// It can be used to analyze the tree or build custom json representation.
func (t *_TABLE_TYPE[V]) DumpList6() []DumpListNode[V] {
	if t == nil {
		return nil
	}
	return dumpListRec(&t.root6, 0, stridePath{}, 0, false)
}

// dumpString is just a wrapper for dump.
func (t *_TABLE_TYPE[V]) dumpString() string {
	w := new(strings.Builder)
	t.dump(w)

	return w.String()
}

// dump the table structure and all the nodes to w.
func (t *_TABLE_TYPE[V]) dump(w io.Writer) {
	if t == nil {
		return
	}

	if t.size4 > 0 {
		stats := nodeStatsRec(&t.root4)
		fmt.Fprintln(w)
		fmt.Fprintf(w, "### IPv4: size(%d), nodes(%d), pfxs(%d), leaves(%d), fringes(%d)",
			t.size4, stats.nodes, stats.pfxs, stats.leaves, stats.fringes)

		dumpRec(&t.root4, w, stridePath{}, 0, true, shouldPrintValues[V]())
	}

	if t.size6 > 0 {
		stats := nodeStatsRec(&t.root6)
		fmt.Fprintln(w)
		fmt.Fprintf(w, "### IPv6: size(%d), nodes(%d), pfxs(%d), leaves(%d), fringes(%d)",
			t.size6, stats.nodes, stats.pfxs, stats.leaves, stats.fringes)

		dumpRec(&t.root6, w, stridePath{}, 0, false, shouldPrintValues[V]())
	}
}
