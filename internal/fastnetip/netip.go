//go:build !unsafe

package fastnetip

import "net/netip"

// Contains4 reports whether the IPv4 prefix pfx contains the IPv4 address ip4.
func Contains4(pfx *netip.Prefix, ip *netip.Addr) bool {
	return pfx.Contains(*ip)
}

// Contains6 reports whether the IPv6 prefix pfx contains the IPv6 address ip6.
func Contains6(pfx *netip.Prefix, ip *netip.Addr) bool {
	return pfx.Contains(*ip)
}
