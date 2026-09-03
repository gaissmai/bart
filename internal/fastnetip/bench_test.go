package fastnetip

import (
	"net/netip"
	"testing"
)

var (
	// Test-Set IPv4
	pfx4 = netip.MustParsePrefix("10.0.0.0/16")
	ips4 = []netip.Addr{
		netip.MustParseAddr("10.0.1.1"),    // Match
		netip.MustParseAddr("10.0.255.2"),  // Match
		netip.MustParseAddr("10.1.0.1"),    // No Match
		netip.MustParseAddr("192.168.1.1"), // No Match
	}

	// Test-Set IPv6
	pfx6 = netip.MustParsePrefix("2001:db8:1234::/48")
	ips6 = []netip.Addr{
		netip.MustParseAddr("2001:db8:1234::1"),         // Match (hi-Register)
		netip.MustParseAddr("2001:db8:1234:ffff::ffff"), // Match (hi-Register boundary)
		netip.MustParseAddr("2001:db8:4321::1"),         // No Match
		netip.MustParseAddr("fe80::1"),                  // No Match
	}
)

// -----------------------------------------------------------------------------
// IPv4 Benchmarks
// -----------------------------------------------------------------------------

func BenchmarkContains4_Fast(b *testing.B) {
	i := 0
	for b.Loop() {
		ip := ips4[i&3]
		_ = Contains4(&pfx4, &ip)
		i++
	}
}

func BenchmarkContains4_Stdlib(b *testing.B) {
	i := 0
	for b.Loop() {
		ip := ips4[i&3]
		_ = pfx4.Contains(ip)
		i++
	}
}

// -----------------------------------------------------------------------------
// IPv6 Benchmarks
// -----------------------------------------------------------------------------

func BenchmarkContains6_Fast(b *testing.B) {
	i := 0
	for b.Loop() {
		ip := ips6[i&3]
		_ = Contains6(&pfx6, &ip)
		i++
	}
}

func BenchmarkContains6_Stdlib(b *testing.B) {
	i := 0
	for b.Loop() {
		ip := ips6[i&3]
		_ = pfx6.Contains(ip)
		i++
	}
}
