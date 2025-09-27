// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Code generated from file "tablemethods_tmpl.go"; DO NOT EDIT.

package bart

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/netip"
	"strings"
)

func (t *liteTable[V]) sizeUpdate(is4 bool, delta int) {
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
func (t *liteTable[V]) Insert(pfx netip.Prefix, val V) {
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

// Delete the prefix and returns the associated payload for prefix and true if found
// or the zero value and false if prefix is not set in the routing table.
func (t *liteTable[V]) Delete(pfx netip.Prefix) (val V, exists bool) {
	if !pfx.IsValid() {
		return val, exists
	}

	// canonicalize prefix
	pfx = pfx.Masked()
	is4 := pfx.Addr().Is4()

	n := t.rootNodeByVersion(is4)
	val, exists = n.del(pfx)

	if exists {
		t.sizeUpdate(is4, -1)
	}
	return val, exists
}

// Get returns the associated payload for prefix and true, or false if
// prefix is not set in the routing table.
func (t *liteTable[V]) Get(pfx netip.Prefix) (val V, exists bool) {
	if !pfx.IsValid() {
		return val, exists
	}
	// canonicalize prefix
	pfx = pfx.Masked()

	is4 := pfx.Addr().Is4()
	n := t.rootNodeByVersion(is4)

	return n.get(pfx)
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
func (f *liteTable[V]) Supernets(pfx netip.Prefix) iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		if !pfx.IsValid() {
			return
		}

		// canonicalize the prefix
		pfx = pfx.Masked()

		is4 := pfx.Addr().Is4()
		n := f.rootNodeByVersion(is4)

		n.supernets(pfx, yield)
	}
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
func (t *liteTable[V]) OverlapsPrefix(pfx netip.Prefix) bool {
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
func (t *liteTable[V]) Overlaps(o *liteTable[V]) bool {
	if o == nil {
		return false
	}
	return t.Overlaps4(o) || t.Overlaps6(o)
}

// Overlaps4 is like [liteTable.Overlaps] but for the v4 routing table only.
func (t *liteTable[V]) Overlaps4(o *liteTable[V]) bool {
	if o == nil || t.size4 == 0 || o.size4 == 0 {
		return false
	}
	return t.root4.overlaps(&o.root4, 0)
}

// Overlaps6 is like [liteTable.Overlaps] but for the v6 routing table only.
func (t *liteTable[V]) Overlaps6(o *liteTable[V]) bool {
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
// Cloner interface, the value is deeply cloned before insertion. See also liteTable.Clone.
func (t *liteTable[V]) Union(o *liteTable[V]) {
	if o == nil || (o.size4 == 0 && o.size6 == 0) {
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
// All nodes touched during union are cloned and a new *liteTable is returned.
// If o is nil or empty, no nodes are touched and the receiver may be
// returned unchanged.
func (t *liteTable[V]) UnionPersist(o *liteTable[V]) *liteTable[V] {
	if o == nil || (o.size4 == 0 && o.size6 == 0) {
		return t
	}

	// Create a cloning function for deep copying values;
	// returns nil if V does not implement the Cloner interface.
	cloneFn := cloneFnFactory[V]()

	// new liteTable with root nodes just copied.
	pt := &liteTable[V]{
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
func (t *liteTable[V]) Equal(o *liteTable[V]) bool {
	if o == nil || t.size4 != o.size4 || t.size6 != o.size6 {
		return false
	}

	return t.root4.equalRec(&o.root4) && t.root6.equalRec(&o.root6)
}

// Clone returns a copy of the routing table.
// The payload of type V is shallow copied, but if type V implements the [Cloner] interface,
// the values are cloned.
func (t *liteTable[V]) Clone() *liteTable[V] {
	if t == nil {
		return nil
	}

	c := new(liteTable[V])

	cloneFn := cloneFnFactory[V]()

	c.root4 = *t.root4.cloneRec(cloneFn)
	c.root6 = *t.root6.cloneRec(cloneFn)

	c.size4 = t.size4
	c.size6 = t.size6

	return c
}

// Size returns the prefix count.
func (t *liteTable[V]) Size() int {
	return t.size4 + t.size6
}

// Size4 returns the IPv4 prefix count.
func (t *liteTable[V]) Size4() int {
	return t.size4
}

// Size6 returns the IPv6 prefix count.
func (t *liteTable[V]) Size6() int {
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
// use [liteTable.WalkPersist] instead.
func (t *liteTable[V]) All() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root4.allRec(stridePath{}, 0, true, yield) && t.root6.allRec(stridePath{}, 0, false, yield)
	}
}

// All4 is like [liteTable.All] but only for the v4 routing table.
func (t *liteTable[V]) All4() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root4.allRec(stridePath{}, 0, true, yield)
	}
}

// All6 is like [liteTable.All] but only for the v6 routing table.
func (t *liteTable[V]) All6() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root6.allRec(stridePath{}, 0, false, yield)
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
func (t *liteTable[V]) AllSorted() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root4.allRecSorted(stridePath{}, 0, true, yield) &&
			t.root6.allRecSorted(stridePath{}, 0, false, yield)
	}
}

// AllSorted4 is like [liteTable.AllSorted] but only for the v4 routing table.
func (t *liteTable[V]) AllSorted4() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root4.allRecSorted(stridePath{}, 0, true, yield)
	}
}

// AllSorted6 is like [liteTable.AllSorted] but only for the v6 routing table.
func (t *liteTable[V]) AllSorted6() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root6.allRecSorted(stridePath{}, 0, false, yield)
	}
}

// String returns a hierarchical tree diagram of the ordered CIDRs
// as string, just a wrapper for [liteTable.Fprint].
// If Fprint returns an error, String panics.
func (t *liteTable[V]) String() string {
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
func (t *liteTable[V]) Fprint(w io.Writer) error {
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
func (t *liteTable[V]) fprint(w io.Writer, is4 bool) error {
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
// just a wrapper for [liteTable.Fprint].
func (t *liteTable[V]) MarshalText() ([]byte, error) {
	w := new(bytes.Buffer)
	if err := t.Fprint(w); err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

// MarshalJSON dumps the table into two sorted lists: for ipv4 and ipv6.
// Every root and subnet is an array, not a map, because the order matters.
func (t *liteTable[V]) MarshalJSON() ([]byte, error) {
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
func (t *liteTable[V]) DumpList4() []DumpListNode[V] {
	if t == nil {
		return nil
	}
	return dumpListRec(&t.root4, 0, stridePath{}, 0, true)
}

// DumpList6 dumps the ipv6 tree into a list of roots and their subnets.
// It can be used to analyze the tree or build custom json representation.
func (t *liteTable[V]) DumpList6() []DumpListNode[V] {
	if t == nil {
		return nil
	}
	return dumpListRec(&t.root6, 0, stridePath{}, 0, false)
}

// dumpString is just a wrapper for dump.
func (t *liteTable[V]) dumpString() string {
	w := new(strings.Builder)
	t.dump(w)

	return w.String()
}

// dump the table structure and all the nodes to w.
func (t *liteTable[V]) dump(w io.Writer) {
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
