package bart

import "unsafe"

// rejectZeroSized enforces (at runtime) that V is not zero-sized.
//
// In Go, zero-sized types (ZSTs) do not require storage. Consequently,
// taking the address of distinct values of such types is not guaranteed
// to produce distinct pointers. In fact, &a == &b may be true for
// different zero-sized values a and b.
//
// This breaks assumptions in the allotment algorithm where a
// value’s memory address (pointer identity) must serve as a
// unique identifier for that object.
func rejectZeroSized[V any]() {
	var zero V
	if unsafe.Sizeof(zero) == 0 {
		panic("zero-sized types not supported because pointer identity is not unique")
	}
}
