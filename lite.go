// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"sync"

	"github.com/gaissmai/bart/internal/nodes"
)

// liteTable follows the BART design but with no payload.
// It is ideal for simple IP ACLs (access-control-lists) with plain
// true/false results with the smallest memory consumption.
//
// Performance note: Do not pass IPv4-in-IPv6 addresses (e.g., ::ffff:192.0.2.1)
// as input. The methods do not perform automatic unmapping to avoid unnecessary
// overhead for the common case where native addresses are used.
// Users should unmap IPv4-in-IPv6 addresses to their native IPv4 form
// (e.g., 192.0.2.1) before calling these methods.
type liteTable[V any] struct {
	// used by -copylocks checker from `go vet`.
	_ [0]sync.Mutex

	root4 nodes.LiteNode[V]
	root6 nodes.LiteNode[V]

	// the number of prefixes in the routing table
	size4 int
	size6 int
}
