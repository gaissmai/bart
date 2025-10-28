// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Package value provides utilities for working with generic type parameters
// as payload at runtime.
//
// The primary functionality is zero-sized type (ZST) detection via IsZST[V].
// This is critical for runtime validation: Fast[V] cannot work correctly with
// zero-sized types (like struct{} or [0]byte). IsZST enables a safety check
// that panics during [Fast.Insert] and [Fast.InsertPersist] operations when
// a zero-sized type is detected.
//
// Additionally, ZST detection improves the clarity of debug output. Since
// zero-sized types carry no information in their values, omitting them from
// dumps and prints reduces line noise and makes the output more readable.
//
// This is an internal package used by the bart data structure implementation.
package value

// IsZST reports whether type V is a zero-sized type (ZST).
//
// Zero-sized types such as struct{}, [0]byte, or structs/arrays with no fields
// occupy no memory. The Go runtime optimizes allocations of ZSTs by returning
// pointers to the same memory address (typically runtime.zerobase).
//
// This function exploits that optimization: it allocates two instances of V
// and compares their addresses. If the addresses are equal, V must be a ZST,
// since distinct non-zero-sized allocations would have different addresses.
//
// The helper escapeToHeap ensures both allocations reach the heap and prevents
// the compiler from proving address equality at compile time, which would
// invalidate the runtime check.
func IsZST[V any]() bool {
	a, b := escapeToHeap[V]()
	return a == b
}

// escapeToHeap forces two allocations of type V to escape to the heap.
//
// The go:noinline directive is critical: it prevents the compiler from inlining
// this function and optimizing away the allocations or proving that a == b at
// compile time. Without it, the compiler could elide one allocation or determine
// the result statically, breaking the ZST detection heuristic.
//
//go:noinline
func escapeToHeap[V any]() (*V, *V) {
	return new(V), new(V)
}
