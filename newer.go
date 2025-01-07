//go:build go1.23

// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import "net/netip"

// netip.Addr.AsSlice for go1.23 does not allocate
func ipAsOctets(ip netip.Addr, _ bool) []byte {
	return ip.AsSlice()
}
