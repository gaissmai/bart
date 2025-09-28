// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"reflect"
)

// Equaler is a generic interface for types that can decide their own
// equality logic. It can be used to override the potentially expensive
// default comparison with [reflect.DeepEqual].
type Equaler[V any] interface {
	Equal(other V) bool
}

// equal compares two values of type V for equality.
// If V implements Equaler[V], that custom equality method is used.
// Otherwise, [reflect.DeepEqual] is used as a fallback.
func equal[V any](v1, v2 V) bool {
	// you can't assert directly on a type parameter
	if v1, ok := any(v1).(Equaler[V]); ok {
		return v1.Equal(v2)
	}
	// fallback
	return reflect.DeepEqual(v1, v2)
}
