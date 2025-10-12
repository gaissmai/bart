// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Package bart provides high-performance Balanced Routing Tables (BART)
// for fastest IP-to-CIDR lookups on IPv4 and IPv6 addresses.
//
// BART offers three table variants optimized for different use cases:
//
//   - Lite:  Memory-optimized with popcount-compressed sparse arrays
//   - Table: Full-featured with popcount-compressed sparse arrays
//   - Fast:  Speed-optimized with fixed-size 256-element arrays
//
// The implementation is based on Knuth's ART algorithm with novel
// optimizations for memory efficiency and lookup speed.
//
// `Table` and `Lite` use popcount compression for memory efficiency, while
// `Fast` trades memory for maximum lookup speed with uncompressed arrays.
//
// BART excels at efficient set operations on routing tables including Union,
// Overlaps, Equal, Subnets, and Supernets with optimal complexity, making it
// ideal for large-scale IP prefix matching in ACLs, RIBs, FIBs, firewalls,
// and routers.
//
// All variants also support copy-on-write persistence.
package bart

import (
	"net/netip"

	"github.com/gaissmai/bart/internal/nodes"
)

// These types, constants, and functions are required in the bart package
// and in internal/nodes. To prevent drift in implementation or values,
// they are either aliased, copied or wrapped.

type stridePath = nodes.StridePath

const (
	maxItems     = nodes.MaxItems
	maxTreeDepth = nodes.MaxTreeDepth
	depthMask    = nodes.DepthMask
)

func lastOctetPlusOneAndLastBits(pfx netip.Prefix) (lastOctetPlusOne int, lastBits uint8) {
	return nodes.LastOctetPlusOneAndLastBits(pfx)
}

// cmpPrefix compares two netip.Prefix values and reports their ordering.
func cmpPrefix(a, b netip.Prefix) int {
	return nodes.CmpPrefix(a, b)
}

// shouldPrintValues reports whether values of type V should be printed.
// It returns true if V is not the empty struct, false otherwise.
func shouldPrintValues[V any]() bool {
	var zero V

	_, isEmptyStruct := any(zero).(struct{})
	return !isEmptyStruct
}

// Equaler is a generic interface for types that can decide their own
// equality logic. It can be used to override the potentially expensive
// default comparison with [reflect.DeepEqual].
type Equaler[V any] interface {
	Equal(other V) bool
}

// Cloner is an interface that enables deep cloning of values of type V.
// If a value implements Cloner[V], Table methods such as InsertPersist,
// ModifyPersist, DeletePersist, UnionPersist, Union and Clone will use
// its Clone method to perform deep copies.
type Cloner[V any] interface {
	Clone() V
}

// DumpListNode contains CIDR, Value and Subnets, representing the trie
// in a sorted, recursive representation, especially useful for serialization.
type DumpListNode[V any] struct {
	CIDR    netip.Prefix      `json:"cidr"`
	Value   V                 `json:"value"`
	Subnets []DumpListNode[V] `json:"subnets,omitempty"`
}

// cloneFnFactory returns a CloneFunc.
// If V implements Cloner[V], the returned function should perform
// a deep copy using Clone(), otherwise it returns nil.
func cloneFnFactory[V any]() nodes.CloneFunc[V] {
	var zero V
	// you can't assert directly on a type parameter
	if _, ok := any(zero).(Cloner[V]); ok {
		return cloneVal[V]
	}
	return nil
}

// cloneVal returns a deep clone of val by calling Clone when
// val implements Cloner[V]. If val does not implement
// Cloner[V] or the Cloner receiver is nil (val is a nil pointer),
// cloneVal returns val unchanged.
func cloneVal[V any](val V) V {
	// you can't assert directly on a type parameter
	c, ok := any(val).(Cloner[V])
	if !ok || c == nil {
		return val
	}
	return c.Clone()
}
