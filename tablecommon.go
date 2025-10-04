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
