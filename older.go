//go:build !go1.23

// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import "net/netip"

// netip.Addr.AsSlice for go < go1.23 allocates
func ipAsOctets(ip netip.Addr) []byte {
	a16 := ip.As16()
	octets := a16[:]
	if ip.Is4() {
		octets = octets[12:]
	}
	return octets
}
