package prefix

import (
	"net/netip"
	"testing"
)

var (
	mpp = netip.MustParsePrefix
	mpa = netip.MustParseAddr
)

func TestPrefixBoundsContains(t *testing.T) {
	tests := []struct {
		name    string
		prefix  string   // CIDR notation
		inside  []string // addresses that should match
		outside []string // addresses that should NOT match
	}{
		{
			name:   "IPv4 /24",
			prefix: "192.168.1.0/24",
			inside: []string{
				"192.168.1.0",
				"192.168.1.42",
				"192.168.1.255",
			},
			outside: []string{
				"192.168.0.255",
				"192.168.2.0",
				"10.0.0.1",
			},
		},
		{
			name:   "IPv4 /22",
			prefix: "192.168.4.0/22",
			inside: []string{
				"192.168.4.0",
				"192.168.5.100",
				"192.168.7.255",
			},
			outside: []string{
				"192.168.3.255",
				"192.168.8.0",
			},
		},
		{
			name:   "IPv6 /64",
			prefix: "2001:db8::/64",
			inside: []string{
				"2001:db8::",
				"2001:db8::1",
				"2001:db8::abcd",
			},
			outside: []string{
				"2001:db8:0:1::",
				"2001:dead::1",
			},
		},
		{
			name:   "IPv6 /126",
			prefix: "2001:db8::/126",
			inside: []string{
				"2001:db8::",
				"2001:db8::1",
				"2001:db8::2",
				"2001:db8::3",
			},
			outside: []string{
				"2001:db8::4",
				"2001:db8::ff",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix := mpp(tt.prefix)
			pb := newPrefixBounds(prefix)

			// Check all inside addresses
			for _, addrStr := range tt.inside {
				addr := mpa(addrStr)
				if got := pb.contains(addr.AsSlice()); !got {
					t.Errorf("inside address %s not matched for prefix %s", addrStr, tt.prefix)
				}
			}

			// Check all outside addresses
			for _, addrStr := range tt.outside {
				addr := mpa(addrStr)
				if got := pb.contains(addr.AsSlice()); got {
					t.Errorf("outside address %s matched for prefix %s", addrStr, tt.prefix)
				}
			}
		})
	}
}

func BenchmarkPrefixBoundsContainsIPv4(b *testing.B) {
	prefixStr := "192.168.1.0/24"
	prefix := mpp(prefixStr)
	addrInside := mpa("192.168.1.42")
	addrOutside := mpa("192.168.2.42")

	pb := newPrefixBounds(prefix)
	addrInsideBytes := addrInside.AsSlice()
	addrOutsideBytes := addrOutside.AsSlice()

	b.Run("prefixBounds_contains_inside", func(b *testing.B) {
		for range b.N {
			if !pb.contains(addrInsideBytes) {
				b.Fail()
			}
		}
	})
	b.Run("prefixBounds_contains_outside", func(b *testing.B) {
		for range b.N {
			if pb.contains(addrOutsideBytes) {
				b.Fail()
			}
		}
	})

	b.Run("netipPrefix_contains_inside", func(b *testing.B) {
		for range b.N {
			if !prefix.Contains(addrInside) {
				b.Fail()
			}
		}
	})
	b.Run("netipPrefix_contains_outside", func(b *testing.B) {
		for range b.N {
			if prefix.Contains(addrOutside) {
				b.Fail()
			}
		}
	})
}

func BenchmarkPrefixBoundsContainsIPv6(b *testing.B) {
	prefixStr := "2001:db8::/64"
	prefix := mpp(prefixStr)
	addrInside := mpa("2001:db8::abcd")
	addrOutside := mpa("2001:db8:0:1::1")

	pb := newPrefixBounds(prefix)
	addrInsideBytes := addrInside.AsSlice()
	addrOutsideBytes := addrOutside.AsSlice()

	b.Run("prefixBounds_contains_inside", func(b *testing.B) {
		for range b.N {
			if !pb.contains(addrInsideBytes) {
				b.Fail()
			}
		}
	})
	b.Run("prefixBounds_contains_outside", func(b *testing.B) {
		for range b.N {
			if pb.contains(addrOutsideBytes) {
				b.Fail()
			}
		}
	})

	b.Run("netipPrefix_contains_inside", func(b *testing.B) {
		for range b.N {
			if !prefix.Contains(addrInside) {
				b.Fail()
			}
		}
	})
	b.Run("netipPrefix_contains_outside", func(b *testing.B) {
		for range b.N {
			if prefix.Contains(addrOutside) {
				b.Fail()
			}
		}
	})
}
