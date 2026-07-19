// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Package value provides runtime utilities for working with generic type
// parameters as payload values in bart data structures.
//
// The package offers three main categories of utilities:
//
// # Zero-Sized Type (ZST) Detection
//
// IsZST[V] detects whether a type parameter V is a zero-sized type (such as
// struct{} or [0]byte). Zero-sized types carry no information in their
// values. Omitting them from dumps and prints reduces line noise and
// improves readability.
//
// # Value Equality
//
// The Equaler[V] interface and Equal function enable custom equality logic
// for payload values. When V implements Equaler[V], the Equal function uses
// that implementation, avoiding the potentially expensive reflect.DeepEqual.
//
// # Value Cloning
//
// The Cloner[V] interface and associated functions (CloneFnFactory, CloneVal,
// CopyVal) support deep copying of payload values for persistent operations.
// When V implements Cloner[V], bart methods like InsertPersist, DeletePersist,
// and UnionPersist use the Clone method to create independent copies.
//
// This is an internal package used by the bart data structure implementation.
package value

import (
	"reflect"
)

// IsZST reports whether type V is a zero-sized type (ZST).
//
// Zero-sized types such as struct{}, [0]byte, or structs/arrays with no fields
// occupy no memory. This function uses reflection to determine the size of the
// type safely without relying on the unsafe package.
func IsZST[V any]() bool {
	return reflect.TypeFor[V]().Size() == 0
}

// Equaler is a generic interface for types that can decide their own
// equality logic. It can be used to override the potentially expensive
// default comparison with [reflect.DeepEqual].
type Equaler[V any] interface {
	Equal(other V) bool
}

// Equal compares two values of type V for equality.
//
// If V implements Equaler[V], its custom equality method is used to
// avoid the potentially expensive [reflect.DeepEqual]. As a safety measure,
// if v1 is a typed nil pointer, Equal gracefully falls back to a fast,
// direct interface comparison to prevent nil receiver panics.
//
// If V does not implement Equaler[V], [reflect.DeepEqual] is used as a fallback.
func Equal[V any](v1, v2 V) bool {
	if eq, ok := any(v1).(Equaler[V]); ok {

		// Guard against typed nil pointers wrapped in the interface.
		// Calling Equal on a typed nil might panic if not handled by the receiver.
		rv := reflect.ValueOf(eq) // Avoid re-boxing v1
		if rv.Kind() == reflect.Pointer && rv.IsNil() {
			return any(v1) == any(v2) // Fast path for nil pointers
		}
		return eq.Equal(v2)
	}

	// fallback
	return reflect.DeepEqual(v1, v2)
}

// Cloner is an interface that enables deep cloning of values of type V.
// If a value implements Cloner[V], Table methods such as InsertPersist,
// ModifyPersist, DeletePersist, UnionPersist, Union and Clone will use
// its Clone method to perform deep copies.
type Cloner[V any] interface {
	Clone() V
}

// CloneFunc is a type definition for a function that takes a value of type V
// and returns the (possibly cloned) value of type V.
type CloneFunc[V any] func(V) V

// CloneFnFactory returns a CloneFunc.
// If V implements Cloner[V], the returned function should perform
// a deep copy using Clone(), otherwise it returns nil.
func CloneFnFactory[V any]() CloneFunc[V] {
	// Safely check if the type V implements Cloner[V] using modern Go reflection.
	// This avoids instantiating values or triggering strictly nil interface bugs.
	if reflect.TypeFor[V]().Implements(reflect.TypeFor[Cloner[V]]()) {
		return CloneVal[V]
	}
	return nil
}

// CloneVal returns a deep clone of val by calling its Clone method
// if val implements [Cloner].
//
// If val does not implement the interface, or if it is a typed nil pointer,
// CloneVal safely returns val unchanged to prevent potential nil receiver panics.
func CloneVal[V any](val V) V {
	if c, ok := any(val).(Cloner[V]); ok {

		// A typed nil pointer inside an interface makes the interface itself non-nil.
		// We must use reflection to safely determine if the underlying value is nil.
		rv := reflect.ValueOf(c) // Avoid re-boxing val
		if rv.Kind() == reflect.Pointer && rv.IsNil() {
			return val
		}

		return c.Clone()
	}

	// fallback
	return val
}

// CopyVal just copies the value of any type V.
func CopyVal[V any](val V) V {
	return val
}
