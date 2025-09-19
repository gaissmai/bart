// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"math/rand/v2"
	"net/netip"
	"testing"
)

func TestSlim_Insert(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		prefixes  []string
		wantSize  int
		wantSize4 int
		wantSize6 int
	}{
		{
			name:      "empty table",
			prefixes:  []string{},
			wantSize:  0,
			wantSize4: 0,
			wantSize6: 0,
		},
		{
			name:      "single IPv4 prefix",
			prefixes:  []string{"192.168.1.0/24"},
			wantSize:  1,
			wantSize4: 1,
			wantSize6: 0,
		},
		{
			name:      "single IPv6 prefix",
			prefixes:  []string{"2001:db8::/32"},
			wantSize:  1,
			wantSize4: 0,
			wantSize6: 1,
		},
		{
			name:      "mixed IPv4 and IPv6",
			prefixes:  []string{"10.0.0.0/8", "172.16.0.0/12", "2001:db8::/32", "fe80::/10"},
			wantSize:  4,
			wantSize4: 2,
			wantSize6: 2,
		},
		{
			name:      "duplicate prefixes",
			prefixes:  []string{"192.168.1.0/24", "192.168.1.0/24"},
			wantSize:  1,
			wantSize4: 1,
			wantSize6: 0,
		},
		{
			name:      "overlapping prefixes",
			prefixes:  []string{"10.0.0.0/8", "10.1.0.0/16", "10.1.1.0/24"},
			wantSize:  3,
			wantSize4: 3,
			wantSize6: 0,
		},
		{
			name:      "host routes",
			prefixes:  []string{"192.168.1.1/32", "2001:db8::1/128"},
			wantSize:  2,
			wantSize4: 1,
			wantSize6: 1,
		},
		{
			name:      "default routes",
			prefixes:  []string{"0.0.0.0/0", "::/0"},
			wantSize:  2,
			wantSize4: 1,
			wantSize6: 1,
		},
		{
			name:      "comprehensive IPv4 set",
			prefixes:  []string{"0.0.0.0/0", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "224.0.0.0/4", "240.0.0.0/4"},
			wantSize:  6,
			wantSize4: 6,
			wantSize6: 0,
		},
		{
			name:      "comprehensive IPv6 set",
			prefixes:  []string{"::/0", "2001:db8::/32", "fe80::/10", "ff00::/8", "fc00::/7"},
			wantSize:  5,
			wantSize4: 0,
			wantSize6: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := &Slim{}

			for _, prefixStr := range tt.prefixes {
				s.Insert(mpp(prefixStr))
			}

			if got := s.Size(); got != tt.wantSize {
				t.Errorf("Size() = %d, want %d", got, tt.wantSize)
			}
			if got := s.Size4(); got != tt.wantSize4 {
				t.Errorf("Size4() = %d, want %d", got, tt.wantSize4)
			}
			if got := s.Size6(); got != tt.wantSize6 {
				t.Errorf("Size6() = %d, want %d", got, tt.wantSize6)
			}
		})
	}
}

func TestSlim_Insert_InvalidPrefix(t *testing.T) {
	t.Parallel()
	s := &Slim{}

	// Insert invalid prefix should be no-op
	invalid := netip.Prefix{} // not valid; IsValid() == false
	s.Insert(invalid)

	if got := s.Size(); got != 0 {
		t.Errorf("Size() after inserting invalid prefix = %d, want 0", got)
	}
}

func TestSlim_Insert_AutoCanonicalizes(t *testing.T) {
	t.Parallel()
	s := &Slim{}

	// Insert prefix with host bits set; should be canonicalized
	pfx1 := netip.MustParsePrefix("192.168.1.123/24")
	s.Insert(pfx1)

	// Insert the properly masked version
	pfx2 := mpp("192.168.1.0/24")
	s.Insert(pfx2)

	// Should only have one prefix, as they canonicalize to the same thing
	if got := s.Size(); got != 1 {
		t.Errorf("Size() after inserting canonicalized duplicates = %d, want 1", got)
	}
}

func TestSlim_Delete(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		insertPrefixes []string
		deletePrefixes []string
		wantFound      []bool
		wantSize       int
		wantSize4      int
		wantSize6      int
	}{
		{
			name:           "delete from empty table",
			insertPrefixes: []string{},
			deletePrefixes: []string{"192.168.1.0/24"},
			wantFound:      []bool{false},
			wantSize:       0,
			wantSize4:      0,
			wantSize6:      0,
		},
		{
			name:           "delete existing prefix",
			insertPrefixes: []string{"192.168.1.0/24", "10.0.0.0/8"},
			deletePrefixes: []string{"192.168.1.0/24"},
			wantFound:      []bool{true},
			wantSize:       1,
			wantSize4:      1,
			wantSize6:      0,
		},
		{
			name:           "delete non-existing prefix",
			insertPrefixes: []string{"192.168.1.0/24"},
			deletePrefixes: []string{"192.168.2.0/24"},
			wantFound:      []bool{false},
			wantSize:       1,
			wantSize4:      1,
			wantSize6:      0,
		},
		{
			name:           "delete all prefixes",
			insertPrefixes: []string{"10.0.0.0/8", "2001:db8::/32"},
			deletePrefixes: []string{"10.0.0.0/8", "2001:db8::/32"},
			wantFound:      []bool{true, true},
			wantSize:       0,
			wantSize4:      0,
			wantSize6:      0,
		},
		{
			name:           "delete IPv6 prefix",
			insertPrefixes: []string{"2001:db8::/32", "fe80::/10"},
			deletePrefixes: []string{"2001:db8::/32"},
			wantFound:      []bool{true},
			wantSize:       1,
			wantSize4:      0,
			wantSize6:      1,
		},
		{
			name:           "delete multiple prefixes",
			insertPrefixes: []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "2001:db8::/32"},
			deletePrefixes: []string{"172.16.0.0/12", "2001:db8::/32"},
			wantFound:      []bool{true, true},
			wantSize:       2,
			wantSize4:      2,
			wantSize6:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := &Slim{}

			// Insert test prefixes
			for _, prefixStr := range tt.insertPrefixes {
				s.Insert(mpp(prefixStr))
			}

			// Delete test prefixes and verify results
			for i, prefixStr := range tt.deletePrefixes {
				got := s.Delete(mpp(prefixStr))
				if got != tt.wantFound[i] {
					t.Errorf("Delete(%q) = %t, want %t", prefixStr, got, tt.wantFound[i])
				}
			}

			// Verify final sizes
			if got := s.Size(); got != tt.wantSize {
				t.Errorf("Size() = %d, want %d", got, tt.wantSize)
			}
			if got := s.Size4(); got != tt.wantSize4 {
				t.Errorf("Size4() = %d, want %d", got, tt.wantSize4)
			}
			if got := s.Size6(); got != tt.wantSize6 {
				t.Errorf("Size6() = %d, want %d", got, tt.wantSize6)
			}
		})
	}
}

func TestSlim_Delete_InvalidPrefix(t *testing.T) {
	t.Parallel()
	s := &Slim{}
	s.Insert(mpp("192.168.1.0/24"))

	// Delete invalid prefix should return false
	invalid := netip.Prefix{} // not valid; IsValid() == false
	if got := s.Delete(invalid); got != false {
		t.Errorf("Delete(invalid prefix) = %t, want false", got)
	}

	// Size should be unchanged
	if got := s.Size(); got != 1 {
		t.Errorf("Size() after deleting invalid prefix = %d, want 1", got)
	}
}

func TestSlim_Contains(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		prefixes []string
		testIPs  []string
		want     []bool
	}{
		{
			name:     "empty table",
			prefixes: []string{},
			testIPs:  []string{"192.168.1.1", "2001:db8::1"},
			want:     []bool{false, false},
		},
		{
			name:     "IPv4 prefix matching",
			prefixes: []string{"192.168.1.0/24"},
			testIPs:  []string{"192.168.1.1", "192.168.1.255", "192.168.2.1", "10.0.0.1"},
			want:     []bool{true, true, false, false},
		},
		{
			name:     "IPv6 prefix matching",
			prefixes: []string{"2001:db8::/32"},
			testIPs:  []string{"2001:db8::1", "2001:db8:1::1", "2001:db9::1", "fe80::1"},
			want:     []bool{true, true, false, false},
		},
		{
			name:     "mixed IPv4 and IPv6",
			prefixes: []string{"10.0.0.0/8", "2001:db8::/32"},
			testIPs:  []string{"10.1.1.1", "192.168.1.1", "2001:db8::1", "fe80::1"},
			want:     []bool{true, false, true, false},
		},
		{
			name:     "overlapping prefixes - longest match",
			prefixes: []string{"10.0.0.0/8", "10.1.0.0/16", "10.1.1.0/24"},
			testIPs:  []string{"10.1.1.1", "10.1.2.1", "10.2.1.1", "192.168.1.1"},
			want:     []bool{true, true, true, false},
		},
		{
			name:     "host routes",
			prefixes: []string{"192.168.1.1/32", "2001:db8::1/128"},
			testIPs:  []string{"192.168.1.1", "192.168.1.2", "2001:db8::1", "2001:db8::2"},
			want:     []bool{true, false, true, false},
		},
		{
			name:     "default routes",
			prefixes: []string{"0.0.0.0/0", "::/0"},
			testIPs:  []string{"8.8.8.8", "192.168.1.1", "2001:db8::1", "fe80::1"},
			want:     []bool{true, true, true, true},
		},
		{
			name:     "boundary conditions IPv4",
			prefixes: []string{"128.0.0.0/1"},
			testIPs:  []string{"127.255.255.255", "128.0.0.0", "255.255.255.255"},
			want:     []bool{false, true, true},
		},
		{
			name:     "boundary conditions IPv6",
			prefixes: []string{"8000::/1"},
			testIPs:  []string{"7fff:ffff:ffff:ffff:ffff:ffff:ffff:ffff", "8000::", "ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"},
			want:     []bool{false, true, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := &Slim{}

			// Insert test prefixes
			for _, prefixStr := range tt.prefixes {
				s.Insert(mpp(prefixStr))
			}

			// Test IP lookups
			for i, ipStr := range tt.testIPs {
				got := s.Contains(mpa(ipStr))
				if got != tt.want[i] {
					t.Errorf("Contains(%q) = %t, want %t", ipStr, got, tt.want[i])
				}
			}
		})
	}
}

func TestSlim_Contains_InvalidIP(t *testing.T) {
	t.Parallel()
	s := &Slim{}
	s.Insert(mpp("192.168.1.0/24"))

	// Test with invalid IP (zero value)
	invalidIP := netip.Addr{}
	got := s.Contains(invalidIP)
	if got != false {
		t.Errorf("Contains(invalid IP) = %t, want false", got)
	}
}

func TestSlim_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("zero value Slim", func(t *testing.T) {
		t.Parallel()
		var s Slim

		if got := s.Size(); got != 0 {
			t.Errorf("Size() = %d, want 0", got)
		}
		if got := s.Size4(); got != 0 {
			t.Errorf("Size4() = %d, want 0", got)
		}
		if got := s.Size6(); got != 0 {
			t.Errorf("Size6() = %d, want 0", got)
		}

		ip := mpa("192.168.1.1")
		if got := s.Contains(ip); got != false {
			t.Errorf("Contains(%v) = %t, want false", ip, got)
		}

		prefix := mpp("192.168.1.0/24")
		if got := s.Delete(prefix); got != false {
			t.Errorf("Delete(%v) = %t, want false", prefix, got)
		}
	})

	t.Run("insert same prefix multiple times", func(t *testing.T) {
		t.Parallel()
		s := &Slim{}
		prefix := mpp("192.168.1.0/24")

		s.Insert(prefix)
		s.Insert(prefix)
		s.Insert(prefix)

		if got := s.Size(); got != 1 {
			t.Errorf("Size() = %d, want 1", got)
		}
	})

	t.Run("delete non-existent prefix multiple times", func(t *testing.T) {
		t.Parallel()
		s := &Slim{}
		prefix := mpp("192.168.1.0/24")

		if got := s.Delete(prefix); got != false {
			t.Errorf("Delete(%v) = %t, want false", prefix, got)
		}
		if got := s.Delete(prefix); got != false {
			t.Errorf("Delete(%v) = %t, want false", prefix, got)
		}
	})
}

func TestSlim_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}
	t.Parallel()

	s := &Slim{}
	prefixes := make([]netip.Prefix, 0, workLoadN())

	// Generate random prefixes
	prng := rand.New(rand.NewPCG(42, 42))
	for i := 0; i < workLoadN(); i++ {
		if prng.Float64() < 0.5 {
			// IPv4
			ip := netip.AddrFrom4([4]byte{
				byte(prng.Uint32()), byte(prng.Uint32()),
				byte(prng.Uint32()), byte(prng.Uint32()),
			})
			bits := prng.IntN(33)
			prefix := netip.PrefixFrom(ip, bits).Masked()
			prefixes = append(prefixes, prefix)
		} else {
			// IPv6
			ip := netip.AddrFrom16([16]byte{
				byte(prng.Uint32()), byte(prng.Uint32()), byte(prng.Uint32()), byte(prng.Uint32()),
				byte(prng.Uint32()), byte(prng.Uint32()), byte(prng.Uint32()), byte(prng.Uint32()),
				byte(prng.Uint32()), byte(prng.Uint32()), byte(prng.Uint32()), byte(prng.Uint32()),
				byte(prng.Uint32()), byte(prng.Uint32()), byte(prng.Uint32()), byte(prng.Uint32()),
			})
			bits := prng.IntN(129)
			prefix := netip.PrefixFrom(ip, bits).Masked()
			prefixes = append(prefixes, prefix)
		}
	}

	// Insert all prefixes
	for _, prefix := range prefixes {
		s.Insert(prefix)
	}

	originalSize := s.Size()
	t.Logf("Inserted %d unique prefixes out of %d total", originalSize, len(prefixes))

	// Test some lookups
	for i := 0; i < 1000; i++ {
		if prng.Float64() < 0.5 {
			ip := netip.AddrFrom4([4]byte{
				byte(prng.Uint32()), byte(prng.Uint32()),
				byte(prng.Uint32()), byte(prng.Uint32()),
			})
			s.Contains(ip) // Don't care about result, just that it doesn't crash
		} else {
			ip := netip.AddrFrom16([16]byte{
				byte(prng.Uint32()), byte(prng.Uint32()), byte(prng.Uint32()), byte(prng.Uint32()),
				byte(prng.Uint32()), byte(prng.Uint32()), byte(prng.Uint32()), byte(prng.Uint32()),
				byte(prng.Uint32()), byte(prng.Uint32()), byte(prng.Uint32()), byte(prng.Uint32()),
				byte(prng.Uint32()), byte(prng.Uint32()), byte(prng.Uint32()), byte(prng.Uint32()),
			})
			s.Contains(ip) // Don't care about result, just that it doesn't crash
		}
	}

	// Delete half the prefixes
	deleted := 0
	for i := 0; i < len(prefixes); i += 2 {
		if s.Delete(prefixes[i]) {
			deleted++
		}
	}

	expectedSize := originalSize - deleted
	if got := s.Size(); got != expectedSize {
		t.Errorf("Size() after deletions = %d, want %d", got, expectedSize)
	}
}
