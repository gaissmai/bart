// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"

	"github.com/gaissmai/bart/internal/nodes"
)

type stridePath = nodes.StridePath

const (
	maxItems     = nodes.MaxItems
	maxTreeDepth = nodes.MaxTreeDepth
	depthMask    = nodes.DepthMask
)

func lastOctetPlusOneAndLastBits(pfx netip.Prefix) (lastOctetPlusOne int, lastBits uint8) {
	return nodes.LastOctetPlusOneAndLastBits(pfx)
}

func shouldPrintValues[V any]() bool {
	var zero V

	_, isEmptyStruct := any(zero).(struct{})
	return !isEmptyStruct
}

// DumpListNode contains CIDR, Value and Subnets, representing the trie
// in a sorted, recursive representation, especially useful for serialization.
type DumpListNode[V any] struct {
	CIDR    netip.Prefix      `json:"cidr"`
	Value   V                 `json:"value"`
	Subnets []DumpListNode[V] `json:"subnets,omitempty"`
}

// Cloner is an interface that enables deep cloning of values of type V.
// If a value implements Cloner[V], Table methods such as InsertPersist,
// ModifyPersist, DeletePersist, UnionPersist, Union and Clone will use
// its Clone method to perform deep copies.
type Cloner[V any] interface {
	Clone() V
}

// cloneFnFactory returns a CloneFunc.
// If V implements Cloner[V], the returned function should perform
// a deep copy using Clone(), otherwise it returns nil.
func cloneFnFactory[V any]() nodes.CloneFunc[V] {
	var zero V
	// you can't assert directly on a type parameter
	if _, ok := any(zero).(Cloner[V]); ok {
		return cloneVal[V]
	}
	return nil
}

// cloneVal returns a deep clone of val by calling its Clone method when
// val implements Cloner[V]. If val does not implement Cloner[V] or the
// asserted Cloner is nil, val is returned unchanged.
func cloneVal[V any](val V) V {
	// you can't assert directly on a type parameter
	c, ok := any(val).(Cloner[V])
	if !ok || c == nil {
		return val
	}
	return c.Clone()
}

// copyVal just copies the value.
func copyVal[V any](val V) V {
	return val
}
