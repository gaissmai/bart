// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

// Fast is an alias for [Table].
//
// Deprecated: Fast was removed as a separate table type. The rank cache now
// lives in the sparse arrays of every node, so [Table] has the identical API
// and comparable lookup speed. Use [Table] instead. With Go 1.26 or later,
// "go fix ./..." rewrites the uses automatically.
//
//go:fix inline
type Fast[T any] = Table[T]
