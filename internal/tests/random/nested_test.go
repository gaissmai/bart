// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package random

import (
	"math/rand/v2"
	"net/netip"
	"slices"
	"testing"
)

func TestNestedPrefixes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		is4  bool
		pool []byte
		n    int
	}{
		{"IPv4 narrow pool", true, []byte{0, 1, 2, 3}, 40},
		{"IPv4 wide pool", true, []byte{0, 127, 128, 255}, 40},
		{"IPv4 single byte", true, []byte{10}, 40},
		{"IPv6 narrow pool", false, []byte{0, 1, 2}, 40},
		{"IPv6 wide pool", false, []byte{0, 1, 0x20, 0x80, 0xff}, 40},
		{"nothing", true, []byte{0, 1}, 0},
	}

	for _, tt := range tests {
		for _, nest := range []bool{false, true} {
			name := tt.name
			if nest {
				name += "/nested"
			}

			t.Run(name, func(t *testing.T) {
				t.Parallel()

				base := NestedPrefixes(rand.New(rand.NewPCG(1, 2)), tt.is4, tt.pool, tt.n, false)
				got := NestedPrefixes(rand.New(rand.NewPCG(1, 2)), tt.is4, tt.pool, tt.n, nest)

				// deterministic
				if again := NestedPrefixes(rand.New(rand.NewPCG(1, 2)), tt.is4, tt.pool, tt.n, nest); !slices.Equal(got, again) {
					t.Fatal("the same seed gives different prefixes")
				}

				// the number of prefixes: n at most, 3*n with the halves
				maxLen := tt.n
				if nest {
					maxLen = 3 * tt.n
				}

				if len(got) > maxLen {
					t.Fatalf("%d prefixes, want at most %d", len(got), maxLen)
				}

				// valid, canonical, of the right family and distinct
				seen := make(map[netip.Prefix]bool, len(got))

				for _, pfx := range got {
					if !pfx.IsValid() || pfx != pfx.Masked() {
						t.Fatalf("%s is not a canonical prefix", pfx)
					}

					if pfx.Addr().Is4() != tt.is4 {
						t.Fatalf("%s has the wrong address family", pfx)
					}

					if seen[pfx] {
						t.Fatalf("%s is generated twice", pfx)
					}

					seen[pfx] = true
				}

				// the first prefixes are the same with and without nest, the
				// others are halves of them
				if len(got) < len(base) || !slices.Equal(got[:len(base)], base) {
					t.Fatal("the generated prefixes differ with nest")
				}

				for _, extra := range got[len(base):] {
					if !slices.ContainsFunc(base, func(parent netip.Prefix) bool {
						return extra.Bits() == parent.Bits()+1 && parent.Contains(extra.Addr())
					}) {
						t.Fatalf("%s is no half of a generated prefix", extra)
					}
				}

				if !nest && len(got) != len(base) {
					t.Fatal("prefixes added without nest")
				}
			})
		}
	}
}

// TestNestedPrefixesCollide verifies the purpose of the generator: with a
// small pool the prefixes overlap and nest, which prefixes of
// [RealWorldPrefixes] hardly do.
func TestNestedPrefixesCollide(t *testing.T) {
	t.Parallel()

	prng := rand.New(rand.NewPCG(7, 7))
	pfxs := NestedPrefixes(prng, true, []byte{0, 1, 2, 3}, 40, true)

	var nested int

	for _, a := range pfxs {
		for _, b := range pfxs {
			if a != b && a.Bits() < b.Bits() && a.Contains(b.Addr()) {
				nested++
			}
		}
	}

	if nested < len(pfxs) {
		t.Errorf("only %d nested pairs in %d prefixes, the generator should make them collide", nested, len(pfxs))
	}
}
