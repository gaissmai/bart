// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Package bart provides high-performance Balanced Routing Tables (BART)
// for fast IP-to-CIDR lookups on IPv4 and IPv6 addresses.
//
// BART offers two table variants optimized for different use cases:
//
//   - Lite:  memory-optimized for ACLs
//   - Table: full-featured routing table
//
// The implementation is based on Knuth's ART algorithm with novel
// optimizations for memory efficiency and lookup speed.
//
// BART excels also for efficient set operations on routing tables including Union,
// Overlaps, Equal, Subnets, and Supernets with optimal complexity, making it
// ideal for large-scale IP prefix matching in ACLs, RIBs, FIBs, firewalls,
// and routers.
//
// All variants also support copy-on-write persistence.
//
// For complex or pointer value types, you must provide custom deep cloning
// using Go's structural typing, implementing the following method on your
// value type V:
//
//	func (v V) Clone() V
//
// For custom Equality implement the following method on your value type V:
//
//	func (v V) Equal(other V) bool
//
// The bart package detects the interface satisfaction automatically at
// runtime using type assertions.
//
// Input Preconditions:
//
// Zoned IP addresses have no effect on lookup results; zone identifiers
// are unconditionally stripped during lookup.
package bart

import (
	"net/netip"

	"github.com/gaissmai/bart/internal/nodes"

	// inlining hint, see also the TestInlineBitSet256Functions.
	// without this silent import the BitSet256 functions are not inlined
	_ "github.com/gaissmai/bart/internal/bitset"
)

// stridePath is required in many places in the bart package and in
// internal/nodes. Aliased to keep the code readable.
type stridePath = nodes.StridePath

// pathContext is required in many places in the bart package and in
// internal/nodes. Aliased to keep the code readable.
type pathContext = nodes.PathContext

// DumpListNode contains CIDR, Value and Subnets, representing the trie
// in a sorted, recursive representation, especially useful for serialization.
type DumpListNode[V any] struct {
	CIDR    netip.Prefix      `json:"cidr"`
	Value   V                 `json:"value"`
	Subnets []DumpListNode[V] `json:"subnets,omitempty"`
}
