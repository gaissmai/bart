// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package nodes

import (
	"iter"
)

// compile time check
var (
	_ NodeReadWriter[any] = (*BartNode[any])(nil)
	_ NodeReadWriter[any] = (*FastNode[any])(nil)
	_ NodeReadWriter[any] = (*LiteNode[any])(nil)
)

// NodeReadWriter is a generic interface that abstracts tree node operations
// for testing, dumping and traversal.
// Note: Implementations like liteNode do not store V; value-returning
// methods will yield the zero value while still reporting presence.
type NodeReadWriter[V any] interface {
	NodeReader[V]
	NodeWriter[V]
}

type NodeWriter[V any] interface {
	InsertChild(uint8, any) bool
	InsertPrefix(uint8, V) bool

	DeleteChild(uint8) bool
	DeletePrefix(uint8) bool
}

type NodeReader[V any] interface {
	IsEmpty() bool

	ChildCount() int
	PrefixCount() int

	GetChild(uint8) (any, bool)
	GetPrefix(uint8) (V, bool)

	MustGetChild(uint8) any
	MustGetPrefix(uint8) V

	GetChildAddrs(*[256]uint8) []uint8
	GetIndices(*[256]uint8) []uint8

	AllChildren() iter.Seq2[uint8, any]
	AllIndices() iter.Seq2[uint8, V]

	Contains(uint8) bool
	Lookup(uint8) (V, bool)
	LookupIdx(uint8) (uint8, V, bool)
}
