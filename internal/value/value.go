// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Package value provides runtime utilities for working with generic type
// parameters as payload values in bart data structures.
//
// The package offers three main categories of utilities:
//
// # T as type struct{} Detection
//
// IsEmptyStruct[T] detects whether a type parameter T is a `struct{}`. These
// types carry no information in their values. Omitting them from dumps
// and prints reduces line noise and improves readability.
//
// # Value Equality
//
// The [Equal] function enables custom equality logic for payload values.
// When T implements an `Equal(T) bool` method, [Equal] uses that implementation.
//
// # Value Cloning
//
// The [CloneFnFactory] supports copying of payload values for persistent
// operations. When T implements a `Clone() T` method,
// bart methods like InsertPersist, DeletePersist, and UnionPersist use the
// generated clone function to create independent copies.
//
// This is an internal package used by the bart data structure implementation.
package value

// IsEmptyStruct reports whether type T is the unnamed empty struct type `struct{}`.
func IsEmptyStruct[T any]() bool {
	var zero T
	_, ok := any(zero).(struct{})
	return ok
}

// Equal compares two values of type T for equality.
//
// If T implements an `Equal(T) bool` method, its custom equality logic is used.
// Otherwise, values are compared directly using the == operator.
//
// If T is not comparable at runtime (such as a slice or map without an Equal
// method), a runtime panic will occur.
//
// Note: If T implements Equal(T) bool with a pointer receiver, the Equal
// method should handle nil receivers gracefully.
func Equal[T any](v1, v2 T) bool {
	if eq, ok := any(v1).(interface{ Equal(T) bool }); ok {
		return eq.Equal(v2)
	}

	return any(v1) == any(v2)
}

// CloneFnFactory returns a function that takes a value of type T and returns
// a copy by calling its `Clone` method.
//
// If T does not implement a `Clone() T` method, it returns nil.
//
// Note: If T implements `Clone() T` with a pointer receiver, the `Clone`
// method should handle nil receivers gracefully.
func CloneFnFactory[T any]() func(T) T {
	var zero T

	// Safely check if T implements the clone method using an inline interface.
	if _, ok := any(zero).(interface{ Clone() T }); ok {
		// Return an anonymous closure directly.
		// Since we already proved T implements the interface above,
		// the direct type assertion here is guaranteed to succeed.
		return func(val T) T {
			return any(val).(interface{ Clone() T }).Clone()
		}
	}

	return nil
}
