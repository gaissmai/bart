package fastnetip

import (
	"net/netip"
	"testing"
)

func TestContains4(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		ip       string
		expected bool
	}{
		// Basic Subnet Tests
		{"Match /24", "192.168.1.0/24", "192.168.1.42", true},
		{"No Match /24", "192.168.1.0/24", "192.168.2.42", false},

		// Boundary / Edge Cases
		{"First IP in /24 (Network)", "10.0.0.0/24", "10.0.0.0", true},
		{"Last IP in /24 (Broadcast)", "10.0.0.0/24", "10.0.0.255", true},
		{"IP right after Broadcast", "10.0.0.0/24", "10.0.1.0", false},

		// Extreme Prefixes (/0 and /32)
		{"Match /0 (Any IP)", "0.0.0.0/0", "1.2.3.4", true},
		{"Match /32 Exact", "172.16.0.1/32", "172.16.0.1", true},
		{"No Match /32", "172.16.0.1/32", "172.16.0.2", false},

		// Single-Bit Boundaries
		{"Match /31 First", "10.0.0.0/31", "10.0.0.0", true},
		{"Match /31 Second", "10.0.0.0/31", "10.0.0.1", true},
		{"No Match /31 Third", "10.0.0.0/31", "10.0.0.2", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pfx := netip.MustParsePrefix(tt.prefix)
			ip := netip.MustParseAddr(tt.ip)

			got := Contains4(&pfx, &ip)
			if got != tt.expected {
				t.Errorf("Contains4(%s, %s) = %v; want %v", tt.prefix, tt.ip, got, tt.expected)
			}

			stdlibGot := pfx.Contains(ip)
			if got != stdlibGot {
				t.Errorf("Mismatch with stdlib for %s in %s! fastnetip=%v, stdlib=%v", tt.ip, tt.prefix, got, stdlibGot)
			}
		})
	}
}

func TestContains6(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		ip       string
		expected bool
	}{
		// Basic Subnet Tests
		{"Match /32", "2001:db8::/32", "2001:db8::1", true},
		{"No Match /32", "2001:db8::/32", "2001:db9::1", false},

		// Register-Boundary Cases (hi vs. lo Register: /64)
		{"Match /64 (hi-Register exact)", "2001:db8:1:2::/64", "2001:db8:1:2:ffff:ffff:ffff:ffff", true},
		{"No Match /64 (hi-Register diff)", "2001:db8:1:2::/64", "2001:db8:1:3::1", false},
		{"Match /65 (crosses into lo-Register)", "2001:db8::/65", "2001:db8:0:0:7fff:ffff:ffff:ffff", true},
		{"No Match /65", "2001:db8::/65", "2001:db8:0:0:8000::", false},

		// Extreme Prefixes (/0 and /128)
		{"Match /0 (Any IPv6)", "::/0", "2607:f8b0:4005:805::200e", true},
		{"Match /128 Exact", "2001:db8::1234/128", "2001:db8::1234", true},
		{"No Match /128", "2001:db8::1234/128", "2001:db8::1235", false},

		// High-Bit lo-Register Limits
		{"Match /127 First", "fe80::10/127", "fe80::10", true},
		{"Match /127 Second", "fe80::10/127", "fe80::11", true},
		{"No Match /127 Third", "fe80::10/127", "fe80::12", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pfx := netip.MustParsePrefix(tt.prefix)
			ip := netip.MustParseAddr(tt.ip)

			got := Contains6(&pfx, &ip)
			if got != tt.expected {
				t.Errorf("Contains6(%s, %s) = %v; want %v", tt.prefix, tt.ip, got, tt.expected)
			}

			stdlibGot := pfx.Contains(ip)
			if got != stdlibGot {
				t.Errorf("Mismatch with stdlib for %s in %s! fastnetip=%v, stdlib=%v", tt.ip, tt.prefix, got, stdlibGot)
			}
		})
	}
}
