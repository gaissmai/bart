// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package nodes

import (
	"fmt"
	"net/netip"
	"testing"

	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/value"
)

// helpers

// workLoadN to adjust loops for tests with -short
func workLoadN() int {
	if testing.Short() {
		return 100
	}
	return 1_000
}

// cloneFnFactory returns a CloneFunc.
// If V implements Cloner[V], the returned function should perform
// a deep copy using Clone(), otherwise it returns nil.
func cloneFnFactory[V any]() value.CloneFunc[V] {
	var zero V
	// you can't assert directly on a type parameter
	if _, ok := any(zero).(value.Cloner[V]); ok {
		return cloneVal[V]
	}
	return nil
}

// cloneVal returns a deep clone of val by calling its Clone method when
// val implements Cloner[V]. If val does not implement Cloner[V] or the
// asserted Cloner is nil, val is returned unchanged.
func cloneVal[V any](val V) V {
	// you can't assert directly on a type parameter
	c, ok := any(val).(value.Cloner[V])
	if !ok || c == nil {
		return val
	}
	return c.Clone()
}

var mpp = func(s string) netip.Prefix {
	pfx := netip.MustParsePrefix(s)
	if pfx == pfx.Masked() {
		return pfx
	}
	panic(fmt.Sprintf("%s is not canonicalized as %s", s, pfx.Masked()))
}

func TestLastOctetPlusOneAndLastBits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		pfx       netip.Prefix
		wantDepth int
		wantBits  uint8
	}{
		{
			pfx:       mpp("0.0.0.0/0"),
			wantDepth: 0,
			wantBits:  0,
		},
		{
			pfx:       mpp("0.0.0.0/32"),
			wantDepth: 4,
			wantBits:  0,
		},
		{
			pfx:       mpp("10.0.0.0/7"),
			wantDepth: 0,
			wantBits:  7,
		},
		{
			pfx:       mpp("10.20.0.0/14"),
			wantDepth: 1,
			wantBits:  6,
		},
		{
			pfx:       mpp("10.20.30.0/24"),
			wantDepth: 3,
			wantBits:  0,
		},
		{
			pfx:       mpp("10.20.30.40/31"),
			wantDepth: 3,
			wantBits:  7,
		},
		//
		{
			pfx:       mpp("::/0"),
			wantDepth: 0,
			wantBits:  0,
		},
		{
			pfx:       mpp("::/128"),
			wantDepth: 16,
			wantBits:  0,
		},
		{
			pfx:       mpp("2001:db8::/31"),
			wantDepth: 3,
			wantBits:  7,
		},
	}

	for _, tc := range tests {
		lastOctetPlusOne, gotBits := LastOctetPlusOneAndLastBits(tc.pfx)
		if lastOctetPlusOne != tc.wantDepth {
			t.Errorf("LastOctetPlusOneAndLastBits(%d), lastOctetPlusOne got: %d, want: %d",
				tc.pfx.Bits(), lastOctetPlusOne, tc.wantDepth)
		}
		if gotBits != tc.wantBits {
			t.Errorf("LastOctetPlusOneAndLastBits(%d), lastBits got: %d, want: %d",
				tc.pfx.Bits(), gotBits, tc.wantBits)
		}
	}
}

func TestNodeType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		nt       nodeType
		expected string
	}{
		{nullNode, "NULL"},
		{fullNode, "FULL"},
		{halfNode, "HALF"},
		{pathNode, "PATH"},
		{stopNode, "STOP"},
		{nodeType(99), "unreachable"}, // Invalid type
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()
			got := tt.nt.String()
			if got != tt.expected {
				t.Errorf("nodeType(%d).String() = %q, want %q", tt.nt, got, tt.expected)
			}
		})
	}
}

func TestNodeIpStridePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path     StridePath
		depth    int
		is4      bool
		expected string
	}{
		// IPv4
		{StridePath{10, 1, 2, 3}, 4, true, "10.1.2.3"},
		{StridePath{10, 1, 2, 3}, 3, true, "10.1.2"},
		{StridePath{10, 1, 2, 3}, 1, true, "10"},
		// IPv6
		{StridePath{0x20}, 1, false, "20"},
		{
			StridePath{0x20, 0x01, 0x0d, 0xb8},
			4,
			false,
			"2001:0db8",
		},
		{
			StridePath{0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			16,
			false,
			"2001:0db8:0000:0000:0000:0000:0000:0001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()
			got := ipStridePath(tt.path, tt.depth, tt.is4)
			if got != tt.expected {
				t.Errorf("ipStridePath(%v,%d,%v) = %q, want %q", tt.path, tt.depth, tt.is4, got, tt.expected)
			}
		})
	}
}

// Test cases for cidrFromPath function
func TestCidrFromPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     StridePath
		depth    int
		is4      bool
		idx      uint8
		expected string
	}{
		// IPv4 test cases
		{
			name:     "IPv4 default route /0",
			path:     StridePath{},
			depth:    0,
			is4:      true,
			idx:      1, // art.IdxToPfx(1) = (0, 0) -> 0.0.0.0/0
			expected: "0.0.0.0/0",
		},
		{
			name:     "IPv4 /1 prefix 1xxxxxxx",
			path:     StridePath{},
			depth:    0,
			is4:      true,
			idx:      3, // art.IdxToPfx(3) = (128, 1) -> 128.0.0.0/1
			expected: "128.0.0.0/1",
		},
		{
			name:     "IPv4 at depth 1",
			path:     StridePath{192, 168, 0, 0},
			depth:    1,
			is4:      true,
			idx:      3, // art.IdxToPfx(3) = (128, 1) -> 192.128.0.0/9
			expected: "192.128.0.0/9",
		},
		{
			name:     "IPv4 at depth 2",
			path:     StridePath{10, 0, 1, 0},
			depth:    2,
			is4:      true,
			idx:      15, // art.IdxToPfx(15) = (224, 3) -> 10.0.224.0/19
			expected: "10.0.224.0/19",
		},

		// IPv6 test cases - KORRIGIERT
		{
			name:     "IPv6 default route /0",
			path:     StridePath{},
			depth:    0,
			is4:      false,
			idx:      1, // art.IdxToPfx(1) = (0, 0) -> ::/0
			expected: "::/0",
		},
		{
			name:     "IPv6 at depth 1",
			path:     StridePath{0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00},
			depth:    1,
			is4:      false,
			idx:      63, // art.IdxToPfx(63) = (248, 5) -> path[1]=0xf8 -> 20f8::/13
			expected: "20f8::/13",
		},
		{
			name:     "IPv6 at depth 7",
			path:     StridePath{0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x01},
			depth:    7,
			is4:      false,
			idx:      127, // art.IdxToPfx(127) = (252, 6) -> path[7]=0xfc -> 2001:db8:0:fc::/62
			expected: "2001:db8:0:fc::/62",
		},
		{
			name:     "IPv6 at depth 15",
			path:     StridePath{0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			depth:    15,
			is4:      false,
			idx:      255, // art.IdxToPfx(255) = (254, 7) -> path[15]=0xfe -> 2001:db8:0:1::fe/127
			expected: "2001:db8:0:1::fe/127",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := CidrFromPath(tt.path, tt.depth, tt.is4, tt.idx)
			if result.String() != tt.expected {
				octet, pfxLen := art.IdxToPfx(tt.idx)
				t.Errorf("Test %s: cidrFromPath() = %v, want %v (idx %d maps to octet=%d, pfxLen=%d)",
					tt.name, result, tt.expected, tt.idx, octet, pfxLen)
			}
		})
	}
}

// Test cases for cidrForFringe function - KORRIGIERT
func TestCidrForFringe(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		octets    []byte
		depth     int
		is4       bool
		lastOctet uint8
		expected  string
	}{
		// IPv4 test cases
		{
			name:      "IPv4 fringe /8 at depth 0",
			octets:    []byte{10, 0, 0, 0},
			depth:     0,
			is4:       true,
			lastOctet: 0,
			expected:  "0.0.0.0/8", // path[0] = lastOctet = 0
		},
		{
			name:      "IPv4 fringe /16 at depth 1",
			octets:    []byte{192, 168, 0, 0},
			depth:     1,
			is4:       true,
			lastOctet: 0,
			expected:  "192.0.0.0/16", // path[1] = lastOctet = 0
		},
		{
			name:      "IPv4 fringe with non-zero lastOctet",
			octets:    []byte{172, 16, 0, 0},
			depth:     2,
			is4:       true,
			lastOctet: 50,
			expected:  "172.16.50.0/24", // path[2] = lastOctet = 50
		},

		// IPv6 test cases - KORRIGIERT
		{
			name:      "IPv6 fringe /8 at depth 0",
			octets:    []byte{0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			depth:     0,
			is4:       false,
			lastOctet: 0,
			expected:  "::/8", // path[0] = lastOctet = 0
		},
		{
			name:      "IPv6 fringe /16 at depth 1",
			octets:    []byte{0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			depth:     1,
			is4:       false,
			lastOctet: 0,
			expected:  "2000::/16", // path[1] = lastOctet = 0
		},
		{
			name:      "IPv6 fringe /64 at depth 7",
			octets:    []byte{0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			depth:     7,
			is4:       false,
			lastOctet: 0,
			expected:  "2001:db8::/64", // path[7] = lastOctet = 0 (overwrites the 0x01)
		},
		{
			name:      "IPv6 fringe /128 at depth 15 (host route)",
			octets:    []byte{0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			depth:     15,
			is4:       false,
			lastOctet: 0,
			expected:  "2001:db8:0:1::/128", // path[15] = lastOctet = 0 (overwrites the 0x01)
		},
		{
			name:      "IPv6 fringe with non-zero lastOctet",
			octets:    []byte{0xfe, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			depth:     7,
			is4:       false,
			lastOctet: 0xff,
			expected:  "fe80:0:0:ff::/64", // path[7] = lastOctet = 0xff
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := CidrForFringe(tt.octets, tt.depth, tt.is4, tt.lastOctet)

			if result.String() != tt.expected {
				t.Errorf("Test %s: cidrForFringe() = %v, want %v", tt.name, result, tt.expected)
			}
		})
	}
}

// Test isFringe function
func TestIsFringe(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		depth    int
		prefix   string
		expected bool
	}{
		// IPv4 cases - fringe nodes at stride boundaries
		{"IPv4 /8 at depth 0", 0, "10.0.0.0/8", true},
		{"IPv4 /16 at depth 1", 1, "192.168.0.0/16", true},
		{"IPv4 /24 at depth 2", 2, "10.0.1.0/24", true},
		{"IPv4 /32 at depth 3", 3, "192.168.1.1/32", true},

		// IPv4 non-fringe cases
		{"IPv4 /9 at depth 1", 1, "192.128.0.0/9", false},
		{"IPv4 /25 at depth 3", 3, "10.0.1.0/25", false},
		{"IPv4 /16 at wrong depth 0", 0, "192.168.0.0/16", false},
		{"IPv4 /8 at wrong depth 1", 1, "10.0.0.0/8", false},

		// IPv6 cases - fringe nodes at stride boundaries
		{"IPv6 /8 at depth 0", 0, "2001::/8", true},
		{"IPv6 /16 at depth 1", 1, "2001:db8::/16", true},
		{"IPv6 /64 at depth 7", 7, "2001:db8::/64", true},
		{"IPv6 /128 at depth 15", 15, "2001:db8::1/128", true},

		// IPv6 non-fringe cases
		{"IPv6 /9 at depth 1", 1, "2000::/9", false},
		{"IPv6 /65 at depth 8", 8, "2001:db8::/65", false},
		{"IPv6 /16 at wrong depth 0", 0, "2001:db8::/16", false},
		{"IPv6 /64 at wrong depth 6", 6, "2001:db8::/64", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pfx := netip.MustParsePrefix(tt.prefix)
			result := IsFringe(tt.depth, pfx)

			if result != tt.expected {
				lastOctetPlusOne, lastBits := LastOctetPlusOneAndLastBits(pfx)
				t.Errorf("Test %s: isFringe(%d, %v) = %v, want %v (lastOctetPlusOne=%d, lastBits=%d)",
					tt.name, tt.depth, pfx, result, tt.expected, lastOctetPlusOne, lastBits)
			}
		})
	}
}

// Test cmpIndexRank function
func TestCmpIndexRank(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		aIdx     uint8
		bIdx     uint8
		expected int // -1 for <, 0 for =, 1 for >
	}{
		{"Equal indices", 1, 1, 0},
		{"Default route vs /1", 1, 2, -1},                 // (0,0) vs (0,1)
		{"Different /1 prefixes", 2, 3, -1},               // (0,1) vs (128,1)
		{"Same octet different prefix lengths", 3, 7, -1}, // (128,1) vs (192,2) - 128 < 192
		{"Different octets same length", 5, 6, -1},        // (64,2) vs (128,2) - 64 < 128
		{"Reverse comparison", 7, 3, 1},                   // (192,2) vs (128,1) - 192 > 128
		{"Full tree comparison", 255, 3, 1},               // (254,7) vs (128,1) - 254 > 128
		{"Mid range comparison", 15, 31, -1},              // (224,3) vs (240,4) - 224 < 240
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := CmpIndexRank(tt.aIdx, tt.bIdx)

			// Check the sign of the result
			var resultSign int
			if result > 0 {
				resultSign = 1
			} else if result < 0 {
				resultSign = -1
			} else {
				resultSign = 0
			}

			if resultSign != tt.expected {
				// Get the actual prefixes for better error messages
				aOctet, aBits := art.IdxToPfx(tt.aIdx)
				bOctet, bBits := art.IdxToPfx(tt.bIdx)
				t.Errorf("Test %s: cmpIndexRank(%d, %d) = %d (sign %d), want sign %d\n  aIdx=%d -> %d/%d\n  bIdx=%d -> %d/%d",
					tt.name, tt.aIdx, tt.bIdx, result, resultSign, tt.expected,
					tt.aIdx, aOctet, aBits, tt.bIdx, bOctet, bBits)
			}
		})
	}
}

// Edge case tests
func TestCidrFromPathEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("IPv4 depth masking", func(t *testing.T) {
		t.Parallel()
		// Test that depth masking works correctly (depth & depthMask)
		path := StridePath{192, 168, 1, 0}
		// depth 32 should be masked to 0 (32 & 15 = 0)
		result := CidrFromPath(path, 32, true, 3) // idx 3 = (128, 1)
		expected := "128.0.0.0/1"                 // depth masked to 0, overwrites path[0] with 128
		if result.String() != expected {
			t.Errorf("Expected %s with masked depth, got %s", expected, result.String())
		}
	})

	t.Run("IPv6 depth masking", func(t *testing.T) {
		t.Parallel()
		path := StridePath{0x20, 0x01, 0x0d, 0xb8}
		// depth 48 should be masked to 0 (48 & 15 = 0)
		result := CidrFromPath(path, 48, false, 7) // idx 7 = (192, 2)
		expected := "c000::/2"                     // depth masked to 0, overwrites path[0] with 192
		if result.String() != expected {
			t.Errorf("Expected %s with masked depth, got %s", expected, result.String())
		}
	})

	t.Run("Zero path", func(t *testing.T) {
		t.Parallel()
		var path StridePath // all zeros
		result := CidrFromPath(path, 0, true, 1)
		expected := "0.0.0.0/0"
		if result.String() != expected {
			t.Errorf("cidrFromPath() = %v, want %v", result, expected)
		}
	})

	t.Run("Path canonicalization", func(t *testing.T) {
		t.Parallel()
		// Test that bytes after depth are cleared
		path := StridePath{10, 20, 30, 40, 50, 60, 70, 80, 90}
		result := CidrFromPath(path, 2, true, 15) // depth 2, idx 15 = (224, 3)
		// Should result in 10.20.224.0/19 (depth*8 + 3 bits from idx 15)
		expected := "10.20.224.0/19"
		if result.String() != expected {
			t.Errorf("Expected canonicalized result %s, got %s", expected, result.String())
		}
	})
}

func TestCidrForFringeEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("IPv4 empty octets", func(t *testing.T) {
		t.Parallel()
		result := CidrForFringe([]byte{}, 0, true, 0)
		expected := "0.0.0.0/8"
		if result.String() != expected {
			t.Errorf("cidrForFringe() = %v, want %v", result, expected)
		}
	})

	t.Run("IPv6 empty octets", func(t *testing.T) {
		t.Parallel()
		result := CidrForFringe([]byte{}, 0, false, 0)
		expected := "::/8"
		if result.String() != expected {
			t.Errorf("cidrForFringe() = %v, want %v", result, expected)
		}
	})

	t.Run("IPv4 depth masking", func(t *testing.T) {
		t.Parallel()
		octets := []byte{10, 20, 30, 40}
		// depth 32 should be masked to 0 (32 & 15 = 0)
		result := CidrForFringe(octets, 32, true, 50)
		expected := "50.0.0.0/8" // depth masked to 0, so lastOctet goes to path[0]
		if result.String() != expected {
			t.Errorf("Expected %s with masked depth, got %s", expected, result.String())
		}
	})

	t.Run("Path canonicalization", func(t *testing.T) {
		t.Parallel()
		// Test that bytes after depth+1 are cleared
		octets := []byte{0xac, 0x10, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e}
		result := CidrForFringe(octets, 2, false, 0x63) // IPv6 depth 2
		// lastOctet 0x63 goes to path[2], bytes after are cleared
		expected := "ac10:6300::/24" // (2+1)*8 = 24 bits
		if result.String() != expected {
			t.Errorf("Expected canonicalized result %s, got %s", expected, result.String())
		}
	})
}

// Test ART index boundaries and special cases
func TestARTIndexSpecialCases(t *testing.T) {
	t.Parallel()

	// Test key ART indices based on the algorithm
	artTests := []struct {
		idx           uint8
		expectedOctet uint8
		expectedBits  uint8
	}{
		{1, 0, 0},     // default route
		{2, 0, 1},     // 0/1
		{3, 128, 1},   // 128/1
		{4, 0, 2},     // 0/2
		{5, 64, 2},    // 64/2
		{6, 128, 2},   // 128/2
		{7, 192, 2},   // 192/2
		{15, 224, 3},  // 224/3
		{31, 240, 4},  // 240/4
		{63, 248, 5},  // 248/5
		{127, 252, 6}, // 252/6
		{255, 254, 7}, // 254/7
	}

	for _, test := range artTests {
		t.Run(fmt.Sprintf("ART_idx_%d", test.idx), func(t *testing.T) {
			t.Parallel()
			octet, bits := art.IdxToPfx(test.idx)
			if octet != test.expectedOctet || bits != test.expectedBits {
				t.Errorf("art.IdxToPfx(%d) = (%d, %d), want (%d, %d)",
					test.idx, octet, bits, test.expectedOctet, test.expectedBits)
			}

			// Test in cidrFromPath
			var path StridePath
			result := CidrFromPath(path, 0, true, test.idx)
			expectedBits := int(test.expectedBits)
			if result.Bits() != expectedBits {
				t.Errorf("cidrFromPath with idx %d should have %d bits, got %d",
					test.idx, expectedBits, result.Bits())
			}
		})
	}
}

// Benchmarks for cidrFromPath
func BenchmarkCidrFromPath(b *testing.B) {
	b.Run("IPv4", func(b *testing.B) {
		path := StridePath{192, 168, 1, 100}
		for b.Loop() {
			_ = CidrFromPath(path, 3, true, 255)
		}
	})

	b.Run("IPv6", func(b *testing.B) {
		path := StridePath{0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}
		for b.Loop() {
			_ = CidrFromPath(path, 15, false, 255)
		}
	})
}

// Benchmarks for cidrForFringe
func BenchmarkCidrForFringe(b *testing.B) {
	b.Run("IPv4", func(b *testing.B) {
		octets := []byte{192, 168, 1, 100}
		for b.Loop() {
			_ = CidrForFringe(octets, 3, true, 0)
		}
	})

	b.Run("IPv6", func(b *testing.B) {
		octets := []byte{0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}
		for b.Loop() {
			_ = CidrForFringe(octets, 15, false, 0)
		}
	})
}

// Additional integration tests that verify the relationship between functions
func TestIntegration(t *testing.T) {
	t.Parallel()

	t.Run("cidrForFringe_and_isFringe_consistency", func(t *testing.T) {
		t.Parallel()
		// Test that fringes created by cidrForFringe are detected by isFringe
		testCases := []struct {
			depth int
			is4   bool
		}{
			{0, true}, {1, true}, {2, true}, {3, true}, // IPv4 depths
			{0, false}, {1, false}, {7, false}, {15, false}, // IPv6 depths
		}

		for _, tc := range testCases {
			octets := make([]byte, 16)
			for i := range octets {
				//nolint:gosec
				octets[i] = uint8(i + 1) // some non-zero pattern
			}

			fringe := CidrForFringe(octets, tc.depth, tc.is4, 0)
			if !IsFringe(tc.depth, fringe) {
				t.Errorf("Prefix created by cidrForFringe at depth %d (is4=%t) should be detected as fringe: %v",
					tc.depth, tc.is4, fringe)
			}
		}
	})

	t.Run("art_consistency", func(t *testing.T) {
		t.Parallel()
		// Test that ART index operations are consistent
		// Use int counter to avoid uint8 overflow
		for i := 1; i <= 255; i++ {
			//nolint:gosec
			idx := uint8(i)
			octet, pfxLen := art.IdxToPfx(idx)

			// Verify that we can use this in cidrFromPath
			var path StridePath
			result := CidrFromPath(path, 0, true, idx)

			// The prefix length should be pfxLen
			if result.Bits() != int(pfxLen) {
				t.Errorf("ART idx %d -> octet=%d, pfxLen=%d, but cidrFromPath gave %d bits",
					idx, octet, pfxLen, result.Bits())
			}

			// For prefixes with sufficient precision, verify the octet matching
			if pfxLen >= 1 { // Test with at least 1 bit precision
				// The first byte should match the octet when masked appropriately
				mask := uint8(0xFF << (8 - pfxLen))
				expectedMasked := octet & mask
				actualMasked := result.Addr().As4()[0] & mask
				if actualMasked != expectedMasked {
					t.Errorf("ART idx %d -> octet=%d/%d, but first byte masked is %d, expected %d",
						idx, octet, pfxLen, actualMasked, expectedMasked)
				}
			}
		}
	})
}

func TestLeafNode_CloneLeaf(t *testing.T) {
	t.Parallel()

	t.Run("nil_receiver", func(t *testing.T) {
		t.Parallel()
		var l *LeafNode[int]
		cloned := l.CloneLeaf(nil)

		if cloned != nil {
			t.Error("CloneLeaf should return nil for nil receiver")
		}
	})

	t.Run("nil_cloneFn", func(t *testing.T) {
		t.Parallel()
		pfx := netip.MustParsePrefix("10.0.0.0/24")
		l := &LeafNode[int]{Prefix: pfx, Value: 42}

		cloned := l.CloneLeaf(nil)

		if cloned == nil {
			t.Fatal("CloneLeaf should return non-nil for non-nil receiver")
		}
		if cloned.Prefix != pfx {
			t.Error("CloneLeaf should preserve Prefix")
		}
		if cloned.Value != 42 {
			t.Error("CloneLeaf should preserve Value with nil cloneFn")
		}
		if cloned == l {
			t.Error("CloneLeaf should return a new instance")
		}
	})

	t.Run("with_cloneFn", func(t *testing.T) {
		t.Parallel()
		type clonableInt struct {
			Val int
		}

		pfx := netip.MustParsePrefix("192.168.0.0/16")
		l := &LeafNode[clonableInt]{Prefix: pfx, Value: clonableInt{Val: 99}}

		cloneFn := func(v clonableInt) clonableInt {
			return clonableInt{Val: v.Val * 2}
		}

		cloned := l.CloneLeaf(cloneFn)

		if cloned == nil {
			t.Fatal("CloneLeaf should return non-nil")
		}
		if cloned.Prefix != pfx {
			t.Error("CloneLeaf should preserve Prefix")
		}
		if cloned.Value.Val != 198 {
			t.Errorf("CloneLeaf should apply cloneFn, got %d, want 198", cloned.Value.Val)
		}
	})
}

func TestFringeNode_CloneFringe(t *testing.T) {
	t.Parallel()

	t.Run("nil_receiver", func(t *testing.T) {
		t.Parallel()
		var f *FringeNode[int]
		cloned := f.CloneFringe(nil)

		if cloned != nil {
			t.Error("CloneFringe should return nil for nil receiver")
		}
	})

	t.Run("nil_cloneFn", func(t *testing.T) {
		t.Parallel()
		f := &FringeNode[int]{Value: 42}

		cloned := f.CloneFringe(nil)

		if cloned == nil {
			t.Fatal("CloneFringe should return non-nil for non-nil receiver")
		}
		if cloned.Value != 42 {
			t.Error("CloneFringe should preserve Value with nil cloneFn")
		}
		if cloned == f {
			t.Error("CloneFringe should return a new instance")
		}
	})

	t.Run("with_cloneFn", func(t *testing.T) {
		t.Parallel()
		type clonableString struct {
			Str string
		}

		f := &FringeNode[clonableString]{Value: clonableString{Str: "hello"}}

		cloneFn := func(v clonableString) clonableString {
			return clonableString{Str: v.Str + "_cloned"}
		}

		cloned := f.CloneFringe(cloneFn)

		if cloned == nil {
			t.Fatal("CloneFringe should return non-nil")
		}
		if cloned.Value.Str != "hello_cloned" {
			t.Errorf("CloneFringe should apply cloneFn, got %q, want %q", cloned.Value.Str, "hello_cloned")
		}
	})
}
