// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

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
