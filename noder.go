// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import "iter"

// noder is a generic interface that abstracts tree node operations
// for testing, dumping and traversal.
//
// It provides a unified API for setting or accessing both regular
// nodes and fast nodes in the routing table structures.
//
// Type parameter V represents the value type stored at prefixes in the tree.
type noder[V any] interface {
	nodeReader[V]

	// + writer methods
	insertChild(uint8, any) bool
	insertPrefix(uint8, V) bool

	deleteChild(uint8) bool
	deletePrefix(uint8) (V, bool)
}

type nodeReader[V any] interface {
	isEmpty() bool

	childCount() int
	prefixCount() int

	getChild(uint8) (any, bool)
	getPrefix(idx uint8) (V, bool)

	mustGetChild(uint8) any
	mustGetPrefix(idx uint8) V

	getChildAddrs() []uint8
	getIndices() []uint8

	allChildren() iter.Seq2[uint8, any]
	allIndices() iter.Seq2[uint8, V]

	contains(idx uint) bool
	lookup(idx uint) (V, bool)
	lookupIdx(idx uint) (uint8, V, bool)
}
