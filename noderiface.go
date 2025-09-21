// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"iter"
)

// compile time check
var (
	_ nodeReadWriter[any] = (*bartNode[any])(nil)
	_ nodeReadWriter[any] = (*fastNode[any])(nil)
	_ nodeReadWriter[any] = (*liteNode[any])(nil)
)

// nodeReadWriter is a generic interface that abstracts tree node operations
// for testing, dumping and traversal.
// Note: Implementations like liteNode do not store V; value-returning
// methods will yield the zero value while still reporting presence.
type nodeReadWriter[V any] interface {
	nodeReader[V]
	nodeWriter[V]
}

type nodeWriter[V any] interface {
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

	contains(idx uint8) bool
	lookup(idx uint8) (V, bool)
	lookupIdx(idx uint8) (uint8, V, bool)
}
