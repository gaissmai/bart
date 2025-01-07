//go:build !go1.23

// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import "net/netip"

// netip.Addr.AsSlice for go < go1.23 allocates
func ipAsOctets(ip netip.Addr, is4 bool) []byte {
	a16 := ip.As16()
	octets := a16[:]
	if is4 {
		octets = octets[12:]
	}
	return octets
}
