// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package random

import (
	"math/rand/v2"
	"net/netip"
	"testing"
)

func TestPrefix(t *testing.T) {
	prng := rand.New(rand.NewPCG(0, 0))

	for range 100 {
		pfx := Prefix(prng)

		// Must be valid
		if !pfx.IsValid() {
			t.Errorf("generated invalid prefix: %v", pfx)
		}

		// Must be masked
		if pfx != pfx.Masked() {
			t.Errorf("prefix not masked: %v != %v", pfx, pfx.Masked())
		}

		// Must be either IPv4 or IPv6
		if !pfx.Addr().Is4() && !pfx.Addr().Is6() {
			t.Errorf("prefix is neither IPv4 nor IPv6: %v", pfx)
		}
	}
}

func TestPrefix4(t *testing.T) {
	prng := rand.New(rand.NewPCG(0, 0))

	for range 100 {
		pfx := Prefix4(prng)

		// Must be IPv4
		if !pfx.Addr().Is4() {
			t.Errorf("Prefix4 generated non-IPv4: %v", pfx)
		}

		// Must be valid and masked
		if !pfx.IsValid() {
			t.Errorf("generated invalid prefix: %v", pfx)
		}
		if pfx != pfx.Masked() {
			t.Errorf("prefix not masked: %v != %v", pfx, pfx.Masked())
		}

		// Bits should be in range 0-32
		if pfx.Bits() < 0 || pfx.Bits() > 32 {
			t.Errorf("IPv4 prefix bits out of range: %d", pfx.Bits())
		}
	}
}

func TestPrefix6(t *testing.T) {
	prng := rand.New(rand.NewPCG(0, 0))

	for range 100 {
		pfx := Prefix6(prng)

		// Must be IPv6
		if !pfx.Addr().Is6() {
			t.Errorf("Prefix6 generated non-IPv6: %v", pfx)
		}

		// Must be valid and masked
		if !pfx.IsValid() {
			t.Errorf("generated invalid prefix: %v", pfx)
		}
		if pfx != pfx.Masked() {
			t.Errorf("prefix not masked: %v != %v", pfx, pfx.Masked())
		}

		// Bits should be in range 0-128
		if pfx.Bits() < 0 || pfx.Bits() > 128 {
			t.Errorf("IPv6 prefix bits out of range: %d", pfx.Bits())
		}
	}
}

func TestIP4(t *testing.T) {
	prng := rand.New(rand.NewPCG(0, 0))

	for range 100 {
		ip := IP4(prng)

		if !ip.Is4() {
			t.Errorf("IP4 generated non-IPv4: %v", ip)
		}

		if !ip.IsValid() {
			t.Errorf("IP4 generated invalid address: %v", ip)
		}
	}
}

func TestIP6(t *testing.T) {
	prng := rand.New(rand.NewPCG(0, 0))

	for range 100 {
		ip := IP6(prng)

		if !ip.Is6() {
			t.Errorf("IP6 generated non-IPv6: %v", ip)
		}

		if !ip.IsValid() {
			t.Errorf("IP6 generated invalid address: %v", ip)
		}
	}
}

func TestIP(t *testing.T) {
	prng := rand.New(rand.NewPCG(0, 0))

	hasIPv4 := false
	hasIPv6 := false

	for range 100 {
		ip := IP(prng)

		if !ip.IsValid() {
			t.Errorf("IP generated invalid address: %v", ip)
		}

		if ip.Is4() {
			hasIPv4 = true
		}
		if ip.Is6() {
			hasIPv6 = true
		}
	}

	// With 100 iterations, we should see both IPv4 and IPv6
	if !hasIPv4 {
		t.Error("IP never generated IPv4 address")
	}
	if !hasIPv6 {
		t.Error("IP never generated IPv6 address")
	}
}

func TestRealWorldPrefixes4(t *testing.T) {
	prng := rand.New(rand.NewPCG(0, 0))
	n := 50

	pfxs := RealWorldPrefixes4(prng, n)

	if len(pfxs) != n {
		t.Errorf("expected %d prefixes, got %d", n, len(pfxs))
	}

	seen := make(map[netip.Prefix]bool)
	for _, pfx := range pfxs {
		// Must be IPv4
		if !pfx.Addr().Is4() {
			t.Errorf("RealWorldPrefixes4 generated non-IPv4: %v", pfx)
		}

		// Must be masked
		if pfx != pfx.Masked() {
			t.Errorf("prefix not masked: %v", pfx)
		}

		// Bits should be in range 8-28
		if pfx.Bits() < 8 || pfx.Bits() > 28 {
			t.Errorf("prefix bits %d out of real-world range 8-28", pfx.Bits())
		}

		// Should not overlap with 240.0.0.0/8
		reserved := mpp("240.0.0.0/8")
		if pfx.Overlaps(reserved) {
			t.Errorf("prefix overlaps with reserved range: %v", pfx)
		}

		// Should be unique
		if seen[pfx] {
			t.Errorf("duplicate prefix generated: %v", pfx)
		}
		seen[pfx] = true
	}
}

func TestRealWorldPrefixes6(t *testing.T) {
	prng := rand.New(rand.NewPCG(0, 0))
	n := 50

	pfxs := RealWorldPrefixes6(prng, n)

	if len(pfxs) != n {
		t.Errorf("expected %d prefixes, got %d", n, len(pfxs))
	}

	globalUnicast := mpp("2000::/3")
	boundary := mpp("2c0f::/16")

	seen := make(map[netip.Prefix]bool)
	for _, pfx := range pfxs {
		// Must be IPv6
		if !pfx.Addr().Is6() {
			t.Errorf("RealWorldPrefixes6 generated non-IPv6: %v", pfx)
		}

		// Must be masked
		if pfx != pfx.Masked() {
			t.Errorf("prefix not masked: %v", pfx)
		}

		// Bits should be in range 16-56
		if pfx.Bits() < 16 || pfx.Bits() > 56 {
			t.Errorf("prefix bits %d out of real-world range 16-56", pfx.Bits())
		}

		// Should overlap with 2000::/3 (global unicast)
		if !pfx.Overlaps(globalUnicast) {
			t.Errorf("prefix does not overlap with 2000::/3: %v", pfx)
		}

		// Should be <= 2c0f::/16
		if pfx.Addr().Compare(boundary.Addr()) == 1 {
			t.Errorf("prefix address exceeds 2c0f::/16 boundary: %v", pfx)
		}

		// Should be unique
		if seen[pfx] {
			t.Errorf("duplicate prefix generated: %v", pfx)
		}
		seen[pfx] = true
	}
}

func TestRealWorldPrefixes(t *testing.T) {
	prng := rand.New(rand.NewPCG(0, 0))
	n := 100

	pfxs := RealWorldPrefixes(prng, n)

	if len(pfxs) != n {
		t.Errorf("expected %d prefixes, got %d", n, len(pfxs))
	}

	hasIPv4 := false
	hasIPv6 := false
	seen := make(map[netip.Prefix]bool)

	for _, pfx := range pfxs {
		// Must be valid and masked
		if !pfx.IsValid() {
			t.Errorf("invalid prefix: %v", pfx)
		}
		if pfx != pfx.Masked() {
			t.Errorf("prefix not masked: %v", pfx)
		}

		// Track IP versions
		if pfx.Addr().Is4() {
			hasIPv4 = true
		}
		if pfx.Addr().Is6() {
			hasIPv6 = true
		}

		// Should be unique
		if seen[pfx] {
			t.Errorf("duplicate prefix generated: %v", pfx)
		}
		seen[pfx] = true
	}

	// Should have both IPv4 and IPv6
	if !hasIPv4 {
		t.Error("RealWorldPrefixes generated no IPv4 prefixes")
	}
	if !hasIPv6 {
		t.Error("RealWorldPrefixes generated no IPv6 prefixes")
	}
}

func TestMppPanicsOnNonCanonical(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("mpp should panic on non-canonical prefix")
		}
	}()

	// This should panic because 192.168.1.5/24 is not canonical
	_ = mpp("192.168.1.5/24")
}

func TestMppAcceptsCanonical(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("mpp should not panic on canonical prefix: %v", r)
		}
	}()

	pfx := mpp("192.168.1.0/24")
	expected := mpp("192.168.1.0/24")

	if pfx != expected {
		t.Errorf("mpp returned %v, want %v", pfx, expected)
	}
}

func TestPrefixDistribution(t *testing.T) {
	prng := rand.New(rand.NewPCG(42, 42))

	// Generate many prefixes and check distribution
	ipv4Count := 0
	ipv6Count := 0

	for range 1000 {
		pfx := Prefix(prng)
		if pfx.Addr().Is4() {
			ipv4Count++
		} else {
			ipv6Count++
		}
	}

	// Should be roughly 50/50, allow 40-60% range
	if ipv4Count < 400 || ipv4Count > 600 {
		t.Errorf("IPv4 distribution out of expected range: %d/1000", ipv4Count)
	}
	if ipv6Count < 400 || ipv6Count > 600 {
		t.Errorf("IPv6 distribution out of expected range: %d/1000", ipv6Count)
	}
}

func TestRealWorldPrefixesWithSmallN(t *testing.T) {
	prng := rand.New(rand.NewPCG(0, 0))

	// Test with n=1
	pfxs := RealWorldPrefixes(prng, 1)
	if len(pfxs) != 1 {
		t.Errorf("expected 1 prefix, got %d", len(pfxs))
	}

	// Test with n=0
	pfxs = RealWorldPrefixes(prng, 0)
	if len(pfxs) != 0 {
		t.Errorf("expected 0 prefixes, got %d", len(pfxs))
	}
}

func TestDeterministicWithSameSeed(t *testing.T) {
	prng1 := rand.New(rand.NewPCG(12345, 67890))
	prng2 := rand.New(rand.NewPCG(12345, 67890))

	pfxs1 := RealWorldPrefixes(prng1, 10)
	pfxs2 := RealWorldPrefixes(prng2, 10)

	if len(pfxs1) != len(pfxs2) {
		t.Errorf("different lengths with same seed: %d vs %d", len(pfxs1), len(pfxs2))
	}

	for i := range pfxs1 {
		if pfxs1[i] != pfxs2[i] {
			t.Errorf("different prefix at index %d with same seed: %v vs %v", i, pfxs1[i], pfxs2[i])
		}
	}
}
