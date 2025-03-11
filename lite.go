// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import "net/netip"

// Lite is the little sister of [Table]. Lite is ideal for simple
// IP access-control-lists, a.k.a. longest-prefix matches
// with plain true/false results.
//
// For all other tasks the much more powerful [Table] must be used.
type Lite struct {
	tbl Table[struct{}]
}

// Insert adds pfx to the tree.
func (l *Lite) Insert(pfx netip.Prefix) {
	l.tbl.Insert(pfx, struct{}{})
}

// Delete removes pfx from the tree, pfx does not have to be present.
func (l *Lite) Delete(pfx netip.Prefix) {
	l.tbl.Delete(pfx)
}

// Contains performs a longest-prefix match for the IP address
// and returns true if any route matches, otherwise false.
func (l *Lite) Contains(ip netip.Addr) bool {
	return l.tbl.Contains(ip)
}

// String returns a hierarchical tree diagram of the ordered CIDRs as string.
func (l *Lite) String() string {
	return l.tbl.String()
}

// dumpString is just a wrapper for dump.
func (l *Lite) dumpString() string {
	return l.tbl.dumpString()
}
