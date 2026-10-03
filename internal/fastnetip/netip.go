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
// Both pfx and ip must be valid IPv6 instances. When compiled with unsafe
// optimizations, family, validity, and zone checks are bypassed for speed.
func Contains6(pfx *netip.Prefix, ip *netip.Addr) bool {
	// Strip IPv6 zone before netip.Prefix.Contains to prevent false returns.
	// but netip.Addr.withoutZone  is not exported :-(
	// and netip.Addr.WithZone("") is not inlinable, so we have to resort to this clever trick:
	// https://github.com/gaissmai/bart/pull/418#issuecomment-5735613506
	ipnoz := netip.PrefixFrom(*ip, 0).Addr()
	return pfx.Contains(ipnoz)
}
