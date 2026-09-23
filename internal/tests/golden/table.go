// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package golden

import (
	"cmp"
	"iter"
	"maps"
	"net/netip"
	"slices"
)

// cmpPrefix, helper function, compare func for prefix sort,
// all cidrs are already normalized
func cmpPrefix(a, b netip.Prefix) int {
	if cmpAddr := a.Addr().Compare(b.Addr()); cmpAddr != 0 {
		return cmpAddr
	}
	return cmp.Compare(a.Bits(), b.Bits())
}

// Table is a un-optimized routing table implemented as a map of
// prefix-value pairs. It serves as a simple, easy-to-verify golden reference
// for testing complex routing table implementations (like BART).
type Table[V any] map[netip.Prefix]V

// Item represents a key-value pair stored in a Table.
type Item[V any] struct {
	Pfx netip.Prefix
	Val V
}

// TableSlice represents an ordered or unordered sequence of prefix-value items.
type TableSlice[V any] []Item[V]

// FlatSorted returns all key-value pairs in the table as a slice,
// sorted in ascending order by their IP prefix key.
func (t Table[V]) FlatSorted() TableSlice[V] {
	if len(t) == 0 {
		return nil
	}

	items := make([]Item[V], 0, len(t))
	for pfx, val := range t {
		items = append(items, Item[V]{Pfx: pfx, Val: val})
	}

	slices.SortFunc(items, func(a, b Item[V]) int {
		return cmpPrefix(a.Pfx, b.Pfx)
	})

	return TableSlice[V](items)
}

// SortKeys extracts all IP prefixes from the slice and returns them as a new
// slice sorted in ascending order.
func (t TableSlice[V]) SortKeys() []netip.Prefix {
	if len(t) == 0 {
		return nil
	}

	result := make([]netip.Prefix, 0, len(t))
	for _, item := range t {
		result = append(result, item.Pfx)
	}

	slices.SortFunc(result, cmpPrefix)
	return result
}

// Equal reports whether ta and tb contain the exact same set of key-value pairs.
//
// If V implements an `Equal(V) bool` method, its custom equality logic is used.
// Otherwise, values are compared directly using the == operator.
//
// If V is not comparable at runtime (such as a slice or map without an Equal
// method), a runtime panic will occur.
//
// Note: If V implements Equal(V) bool with a pointer receiver, the Equal
// method should handle nil receivers gracefully.
func (ta Table[V]) Equal(tb Table[V]) bool {
	return maps.EqualFunc(ta, tb, func(v1, v2 V) bool {
		if eq1, ok := any(v1).(interface{ Equal(V) bool }); ok {
			return eq1.Equal(v2)
		}
		return any(v1) == any(v2)
	})
}

// Insert adds or updates a prefix-value mapping in the table.
// The prefix is normalized (masked) before insertion.
// If the table pointer or underlying map is nil, a new map is allocated.
func (t *Table[V]) Insert(pfx netip.Prefix, val V) {
	if t == nil {
		return
	}
	if *t == nil {
		*t = make(Table[V])
	}
	(*t)[pfx.Masked()] = val
}

// Delete removes the specified prefix from the table.
// Returns true if the prefix was present and removed, false otherwise.
func (t Table[V]) Delete(pfx netip.Prefix) (exists bool) {
	pfx = pfx.Masked()
	if _, ok := t[pfx]; ok {
		delete(t, pfx)
		return true
	}
	return
}

// All returns an iterator of all prefix, value pairs currently present
// in the table as iterator.
func (t Table[V]) All() iter.Seq2[netip.Prefix, V] {
	return func(yield func(pfx netip.Prefix, val V) bool) {
		for pfx, val := range t {
			if !yield(pfx, val) {
				return
			}
		}
	}
}

// AllKeys returns an iterator over all IP prefixes in the table.
// The iteration order is non-deterministic, following map iteration semantics.
func (t Table[V]) AllKeys() iter.Seq[netip.Prefix] {
	return func(yield func(pfx netip.Prefix) bool) {
		for pfx := range t {
			if !yield(pfx) {
				return
			}
		}
	}
}

// Get performs an exact match search for the given prefix.
func (t Table[V]) Get(pfx netip.Prefix) (val V, ok bool) {
	val, ok = t[pfx.Masked()]
	return
}

// Update modifies an existing prefix or inserts a new one using a callback function.
// The callback receives the current value (or zero value) and a boolean indicating
// whether the prefix was found. Returns the newly set value.
func (t *Table[V]) Update(pfx netip.Prefix, cb func(V, bool) V) (val V) {
	if t == nil {
		return
	}
	if *t == nil {
		*t = make(Table[V])
	}

	pfx = pfx.Masked()

	oldVal, ok := (*t)[pfx]
	newVal := cb(oldVal, ok)

	(*t)[pfx] = newVal
	return newVal
}

// Union merges entries from tb into ta. Entries in tb override matching prefixes in ta.
func (ta *Table[V]) Union(tb Table[V]) {
	if ta == nil {
		return
	}
	if *ta == nil {
		*ta = make(Table[V])
	}

	maps.Copy(*ta, tb)
}

// Lookup performs a Longest Prefix Match (LPM) for the given IP address.
func (t Table[V]) Lookup(addr netip.Addr) (val V, ok bool) {
	bestLen := -1

	for pfx, v := range t {
		if pfx.Bits() > bestLen && pfx.Contains(addr) {
			val = v
			ok = true
			bestLen = pfx.Bits()
		}
	}
	return val, ok
}

// LookupPrefix performs a Longest Prefix Match (LPM) for a covering prefix.
// It returns the value associated with the most specific prefix that covers pfx.
func (t Table[V]) LookupPrefix(pfx netip.Prefix) (val V, ok bool) {
	_, val, ok = t.LookupPrefixLPM(pfx)
	return val, ok
}

// LookupPrefixLPM performs a Longest Prefix Match (LPM) for a covering prefix.
// It returns the matched covering prefix, its associated value, and a boolean status.
func (t Table[V]) LookupPrefixLPM(searchPfx netip.Prefix) (lpm netip.Prefix, val V, ok bool) {
	searchPfx = searchPfx.Masked()
	bestLen := -1

	for pfx, v := range t {
		// A prefix covers searchPfx if they overlap and pfx is equal to or shorter (broader) than searchPfx.
		if pfx.Bits() <= searchPfx.Bits() && pfx.Bits() > bestLen && pfx.Overlaps(searchPfx) {
			val = v
			lpm = pfx
			ok = true
			bestLen = pfx.Bits()
		}
	}
	return lpm, val, ok
}

// Subnets returns all prefixes in the table that are subnets of (covered by) pfx,
// sorted in canonical prefix order.
func (t Table[V]) Subnets(searchPfx netip.Prefix) []netip.Prefix {
	searchPfx = searchPfx.Masked()
	var result []netip.Prefix

	for pfx := range t {
		if searchPfx.Bits() <= pfx.Bits() && searchPfx.Overlaps(pfx) {
			result = append(result, pfx)
		}
	}
	slices.SortFunc(result, cmpPrefix)
	return result
}

// Supernets returns all covering prefixes (supernets) for pfx contained in the table,
// ordered from most-specific to least-specific (longest to shortest mask length).
func (t Table[V]) Supernets(searchPfx netip.Prefix) []netip.Prefix {
	searchPfx = searchPfx.Masked()
	var result []netip.Prefix

	for pfx := range t {
		if pfx.Bits() <= searchPfx.Bits() && pfx.Overlaps(searchPfx) {
			result = append(result, pfx)
		}
	}
	slices.SortFunc(result, cmpPrefix)
	slices.Reverse(result)
	return result
}

// OverlapsPrefix reports whether any prefix in the table overlaps with pfx.
func (t Table[V]) OverlapsPrefix(searchPfx netip.Prefix) bool {
	searchPfx = searchPfx.Masked()
	for pfx := range t {
		if pfx.Overlaps(searchPfx) {
			return true
		}
	}
	return false
}

// Overlaps reports whether any prefix in ta overlaps with any prefix in tb.
func (ta Table[V]) Overlaps(tb Table[V]) bool {
	for aPfx := range ta {
		for bPfx := range tb {
			if aPfx.Overlaps(bPfx) {
				return true
			}
		}
	}
	return false
}

// Aggregate compresses the Table in-place by merging overlapping and
// adjacent IP prefixes into their minimal covering CIDR blocks.
// Values are zeoed out.
func (t Table[V]) Aggregate() {
	if len(t) == 0 {
		return
	}

	var zero V

	currentPfxs := make([]netip.Prefix, 0, len(t))

	// sort netip.Prefixes from t
	for pfx := range t {
		currentPfxs = append(currentPfxs, pfx)
	}
	slices.SortFunc(currentPfxs, cmpPrefix)

	// Pre-allocate output buffer for iterative ping-pong aggregation passes.
	aggregatedPfxs := make([]netip.Prefix, 0, len(t))

	// as long as there was a merge in the last run ...
	for merged := true; merged; {
		merged = false

		// reset output buffer
		aggregatedPfxs = aggregatedPfxs[:0]

		for _, pfx := range currentPfxs {
			// first pfx
			if len(aggregatedPfxs) == 0 {
				aggregatedPfxs = append(aggregatedPfxs, pfx)
				continue
			}

			lastIdx := len(aggregatedPfxs) - 1
			prevPfx := aggregatedPfxs[lastIdx]

			// Rule 1: Address family mismatch (IPv4 vs IPv6) -> cannot merge.
			if prevPfx.Addr().Is4() != pfx.Addr().Is4() {
				aggregatedPfxs = append(aggregatedPfxs, pfx)
				continue
			}

			// Rule 2: Overlapping / Containment.
			// Since current is sorted, broader prefixes appear first for identical base addresses.
			if prevPfx.Contains(pfx.Addr()) {
				// Drop the contained sub-prefix.
				merged = true
				continue
			}

			// Rule 3: Adjacency (merging sibling prefixes).
			// Sibling prefixes have identical bit length and fit under a shared parent prefix of (bits - 1).
			bits := prevPfx.Bits()
			if bits == pfx.Bits() && bits > 0 {
				parent, err := prevPfx.Addr().Prefix(bits - 1)
				if err == nil && parent.Contains(pfx.Addr()) {
					// Merge into parent block, keeping the value of the primary prefix.
					aggregatedPfxs[lastIdx] = parent
					merged = true
					continue
				}
			}

			aggregatedPfxs = append(aggregatedPfxs, pfx)
		}

		// Swap slices for the next iteration step.
		currentPfxs = aggregatedPfxs
	}

	// clear table
	clear(t)

	// re-fill table with aggregated prefixes
	for _, pfx := range currentPfxs {
		t[pfx] = zero
	}
}
