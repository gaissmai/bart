//go:build !unsafe

package fastnetip

import "net/netip"

// Contains4 reports whether pfx contains ip.
//
// Both pfx and ip must be valid IPv4 instances. When compiled with unsafe
// optimizations, family, validity, and zone checks are bypassed for speed.
func Contains4(pfx *netip.Prefix, ip *netip.Addr) bool {
	return pfx.Contains(*ip)
}

// Contains6 reports whether pfx contains ip.
//
// Both pfx and ip must be valid IPv6 instances without scoping zones.
// When compiled with unsafe optimizations, family, validity, and zone checks
// are bypassed for speed.
func Contains6(pfx *netip.Prefix, ip *netip.Addr) bool {
	return pfx.Contains(*ip)
}
