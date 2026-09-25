package bart

// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

import (
	"math/rand/v2"
	"net/netip"
	"slices"
	"testing"

	"github.com/gaissmai/bart/internal/tests/golden"
	"github.com/gaissmai/bart/internal/tests/random"
)

func TestFastACL_NilReceiver(t *testing.T) {
	t.Parallel()

	ip4 := mpa("127.0.0.1")

	pfx4 := mpp("127.0.0.0/8")
	pfx6 := mpp("::1/128")

	var tbl1 *FastACL = nil

	t.Run("mustPanic", func(t *testing.T) {
		t.Parallel()

		mustPanic(t, "Size", func() { tbl1.Size() })
		mustPanic(t, "Size4", func() { tbl1.Size4() })
		mustPanic(t, "Size6", func() { tbl1.Size6() })

		mustPanic(t, "Get", func() { tbl1.Get(pfx4) })
		mustPanic(t, "Insert", func() { tbl1.Insert(pfx4) })
		mustPanic(t, "Delete", func() { tbl1.Delete(pfx4) })
		mustPanic(t, "Contains", func() { tbl1.Contains(ip4) })
		// TODO mustPanic(t, "LookupPrefix", func() { tbl1.LookupPrefix(pfx4) })
		// TODO mustPanic(t, "LookupPrefixLPM", func() { tbl1.LookupPrefixLPM(pfx4) })
		mustPanic(t, "Aggregate", func() { tbl1.Aggregate() })
		mustPanic(t, "Clone", func() { tbl1.Clone() })
		mustPanic(t, "Overlaps", func() { tbl1.Overlaps(nil) })
		mustPanic(t, "Overlaps4", func() { tbl1.Overlaps4(nil) })
		mustPanic(t, "Overlaps6", func() { tbl1.Overlaps6(nil) })
		mustPanic(t, "OverlapsPrefix", func() { tbl1.OverlapsPrefix(pfx4) })
		mustPanic(t, "OverlapsPrefix", func() { tbl1.OverlapsPrefix(pfx6) })
		mustPanic(t, "Equal", func() { tbl1.Equal(nil) })
		mustPanic(t, "Fprint", func() { tbl1.Fprint(nil) })

		mustPanicRangeOverFunc[any](t, "All", tbl1.All)
		mustPanicRangeOverFunc[any](t, "All4", tbl1.All4)
		mustPanicRangeOverFunc[any](t, "All6", tbl1.All6)
		mustPanicRangeOverFunc[any](t, "AllSorted", tbl1.AllSorted)
		mustPanicRangeOverFunc[any](t, "AllSorted4", tbl1.AllSorted4)
		mustPanicRangeOverFunc[any](t, "AllSorted6", tbl1.AllSorted6)
		// TODO mustPanicRangeOverFunc[any](t, "Subnets", tbl1.Subnets)
		mustPanicRangeOverFunc[any](t, "Supernets", tbl1.Supernets)
	})
}

func TestFastACL_Invalid(t *testing.T) {
	t.Parallel()

	tbl1 := new(FastACL)
	tbl2 := new(FastACL)

	var zeroIP netip.Addr
	var zeroPfx netip.Prefix

	noPanic(t, "All", func() { tbl1.All() })
	noPanic(t, "All4", func() { tbl1.All4() })
	noPanic(t, "All6", func() { tbl1.All6() })
	noPanic(t, "AllSorted", func() { tbl1.AllSorted() })
	noPanic(t, "AllSorted4", func() { tbl1.AllSorted4() })
	noPanic(t, "AllSorted6", func() { tbl1.AllSorted6() })
	noPanic(t, "Contains", func() { tbl1.Contains(zeroIP) })
	noPanic(t, "Delete", func() { tbl1.Delete(zeroPfx) })
	noPanic(t, "Equal", func() { tbl1.Equal(tbl2) })
	noPanic(t, "Fprint", func() { tbl1.Fprint(nil) })
	noPanic(t, "Get", func() { tbl1.Get(zeroPfx) })
	noPanic(t, "Insert", func() { tbl1.Insert(zeroPfx) })
	// TODO noPanic(t, "LookupPrefix", func() { tbl1.LookupPrefix(zeroPfx) })
	// TODO noPanic(t, "LookupPrefixLPM", func() { tbl1.LookupPrefixLPM(zeroPfx) })
	noPanic(t, "Overlaps", func() { tbl1.Overlaps(tbl2) })
	noPanic(t, "Overlaps4", func() { tbl1.Overlaps4(tbl2) })
	noPanic(t, "Overlaps6", func() { tbl1.Overlaps6(tbl2) })
	noPanic(t, "OverlapsPrefix", func() { tbl1.OverlapsPrefix(zeroPfx) })
	noPanic(t, "Size", func() { tbl1.Size() })
	noPanic(t, "Size4", func() { tbl1.Size4() })
	noPanic(t, "Size6", func() { tbl1.Size6() })
	// TODO noPanic(t, "Subnets", func() { tbl1.Subnets(zeroPfx) })
	noPanic(t, "Supernets", func() { tbl1.Supernets(zeroPfx) })
}

func TestFastACL_ContainsCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := random.RealWorldPrefixes(prng, n)

	gold := new(golden.Table[int])
	tbl := new(FastACL)

	for i, p := range pfxs {
		gold.Insert(p, i)
		tbl.Insert(p)
	}

	for range n {
		ip := random.IP(prng)

		_, goldOK := gold.Lookup(ip)
		tblOK := tbl.Contains(ip)

		if goldOK != tblOK {
			t.Fatalf("Contains(%q) = %v, want %v", ip, tblOK, goldOK)
		}
	}
}

func TestFastACL_ZonedContains(t *testing.T) {
	t.Parallel()

	check := func(t *testing.T, table *FastACL, ip netip.Addr, wantOK bool) {
		t.Helper()

		for _, probe := range []struct {
			name string
			ip   netip.Addr
		}{
			{name: "plain", ip: ip},
			{name: "zoned", ip: ip.WithZone("eth0")},
		} {

			t.Run(probe.name, func(t *testing.T) {
				if got := table.Contains(probe.ip); got != wantOK {
					t.Fatalf("Contains(%q) = %v, want %v", probe.ip, got, wantOK)
				}
			})
		}
	}

	type route struct {
		cidr  string
		value string
	}

	tests := []struct {
		name   string
		pfxs   []string
		probe  string
		wantOK bool
	}{
		{
			name:   "compressed leaf hit",
			pfxs:   []string{"2001:db8:1:2::/65"},
			probe:  "2001:db8:1:2::1",
			wantOK: true,
		},
		{
			name:   "leaf miss",
			pfxs:   []string{"2001:db8:1:2::/65"},
			probe:  "2001:db8:1:2:8000::1",
			wantOK: false,
		},
		{
			name: "leaf miss falls back to less specific route",
			pfxs: []string{
				"2001:db8:1::/48",
				"2001:db8:1:2::/65",
			},
			probe:  "2001:db8:1:2:8000::1",
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			table := new(FastACL)
			for _, pfxStr := range tt.pfxs {
				table.Insert(mpp(pfxStr))
			}

			check(t, table, mpa(tt.probe), tt.wantOK)
		})
	}
}

func TestFastACL_InsertShuffled(t *testing.T) {
	// The order in which you insert prefixes into a route table
	// should not matter, as long as you're inserting the same set of
	// routes.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := random.RealWorldPrefixes(prng, n)

	for range 10 {
		pfxs2 := slices.Clone(pfxs)
		prng.Shuffle(len(pfxs2), func(i, j int) { pfxs2[i], pfxs2[j] = pfxs2[j], pfxs2[i] })

		tbl1 := new(FastACL)
		tbl2 := new(FastACL)

		for _, pfx := range pfxs {
			tbl1.Insert(pfx)
			tbl1.Insert(pfx) // idempotent
		}
		for _, pfx := range pfxs2 {
			tbl2.Insert(pfx) // idempotent
		}

		if tbl1.dumpString() != tbl2.dumpString() {
			t.Fatal("tbl1 and tbl2 have different dumpString representation")
		}
		if !tbl1.Equal(tbl2) {
			t.Fatal("expected Equal")
		}
	}
}

// TestFastACL_All verifies iterator traversal behavior for All(), All4(), and All6(),
// covering empty tables, combined dual-stack iteration, and early termination.
func TestFastACL_All(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		initial   []string
		testMode  string // "all", "all4", "all6"
		stopAfter int    // stop yield after N elements (-1 for no early stop)
		want      []string
	}{
		{
			name:      "empty table iteration produces no prefixes",
			initial:   nil,
			testMode:  "all",
			stopAfter: -1,
			want:      nil,
		},
		{
			name: "All4 iterates only IPv4 prefixes",
			initial: []string{
				"10.0.0.0/8",
				"192.168.1.0/24",
				"2001:db8::/32",
			},
			testMode:  "all4",
			stopAfter: -1,
			want: []string{
				"10.0.0.0/8",
				"192.168.1.0/24",
			},
		},
		{
			name: "All6 iterates only IPv6 prefixes",
			initial: []string{
				"10.0.0.0/8",
				"2001:db8::/32",
				"fe80::/10",
			},
			testMode:  "all6",
			stopAfter: -1,
			want: []string{
				"2001:db8::/32",
				"fe80::/10",
			},
		},
		{
			name: "All iterates both IPv4 and IPv6 prefixes",
			initial: []string{
				"10.0.0.0/8",
				"192.168.1.0/24",
				"2001:db8::/32",
				"fe80::/10",
			},
			testMode:  "all",
			stopAfter: -1,
			want: []string{
				"10.0.0.0/8",
				"192.168.1.0/24",
				"2001:db8::/32",
				"fe80::/10",
			},
		},
		{
			name: "All early termination during IPv4 phase halts complete traversal",
			initial: []string{
				"10.0.0.0/8",
				"192.168.1.0/24",
				"2001:db8::/32",
			},
			testMode:  "all",
			stopAfter: 1, // stop after 1st element (IPv4 stage)
			want: []string{
				"10.0.0.0/8",
			},
		},
		{
			name: "All early termination during IPv6 phase stops traversal",
			initial: []string{
				"10.0.0.0/8",
				"2001:db8::/32",
				"2001:db8:1::/48",
			},
			testMode:  "all",
			stopAfter: 2, // stop after 2nd element (during IPv6 stage)
			want: []string{
				"10.0.0.0/8",
				"2001:db8::/32",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var facl FastACL
			for _, s := range tt.initial {
				facl.Insert(mpp(s))
			}

			// Ensure invalid prefixes do not affect the table or iterator.
			facl.Insert(netip.Prefix{})

			var iter func(yield func(netip.Prefix) bool)
			switch tt.testMode {
			case "all":
				iter = facl.All()
			case "all4":
				iter = facl.All4()
			case "all6":
				iter = facl.All6()
			default:
				t.Fatalf("unknown testMode: %s", tt.testMode)
			}

			var got []string
			for pfx := range iter {
				got = append(got, pfx.String())
				if tt.stopAfter > 0 && len(got) == tt.stopAfter {
					break // Triggers early termination (yield returns false)
				}
			}

			// Iteration order is explicitly unspecified in godoc, sort for deterministic comparison.
			slices.Sort(got)
			want := slices.Clone(tt.want)
			slices.Sort(want)

			if !slices.Equal(got, want) {
				t.Errorf("iterator mismatch:\ngot:  %v\nwant: %v", got, want)
			}
		})
	}
}

func TestFastACL_AllSorted(t *testing.T) {
	t.Parallel()

	// Test cases with known CIDR sort order
	testCases := []struct {
		name     string
		prefixes []string
		expected []string // Expected order after sorting
	}{
		{
			name: "Mixed IPv4 addresses and prefix lengths",
			prefixes: []string{
				"10.0.0.0/16",
				"10.0.0.0/8",
				"192.168.1.0/24",
				"10.0.0.0/24",
				"172.16.0.0/12",
			},
			expected: []string{
				"10.0.0.0/8",     // Same address, shorter prefix first
				"10.0.0.0/16",    // Same address, longer prefix
				"10.0.0.0/24",    // Same address, longest prefix
				"172.16.0.0/12",  // Next address
				"192.168.1.0/24", // Highest address
			},
		},
		{
			name: "Mixed IPv6 addresses and prefix lengths",
			prefixes: []string{
				"2001:db8::/32",
				"2001:db8::/64",
				"2000::/16",
				"2001:db8:1::/48",
			},
			expected: []string{
				"2000::/16",       // Lowest address
				"2001:db8::/32",   // Same address, shorter prefix first
				"2001:db8::/64",   // Same address, longer prefix
				"2001:db8:1::/48", // Higher address
			},
		},
		{
			name: "Mixed IPv4 and IPv6",
			prefixes: []string{
				"192.168.1.0/24",
				"2001:db8::/32",
				"10.0.0.0/8",
				"::1/128",
			},
			expected: []string{
				"10.0.0.0/8",     // IPv4 addresses come first (lower in comparison)
				"192.168.1.0/24", // Next IPv4 address
				"::1/128",        // IPv6 addresses after IPv4
				"2001:db8::/32",  // Higher IPv6 address
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tbl := new(FastACL)

			// Insert prefixes with index as value
			for _, prefixStr := range tc.prefixes {
				pfx := mpp(prefixStr)
				tbl.Insert(pfx)
			}

			// Collect sorted results
			var actualOrder []string
			for pfx := range tbl.AllSorted() {
				actualOrder = append(actualOrder, pfx.String())
			}

			// Verify the order matches expected
			if len(actualOrder) != len(tc.expected) {
				t.Fatalf("%s: Expected %d results, got %d", tc.name, len(tc.expected), len(actualOrder))
			}

			// Collect sorted 4 results
			var actual4Order []string
			for pfx := range tbl.AllSorted4() {
				actual4Order = append(actual4Order, pfx.String())
			}

			// Collect sorted 6 results
			var actual6Order []string
			for pfx := range tbl.AllSorted6() {
				actual6Order = append(actual6Order, pfx.String())
			}

			if !slices.Equal(slices.Concat(actual4Order, actual6Order), actualOrder) {
				t.Fatalf("%s: Prefixes: AllSorted4 + AllSorted6 != AllSorted", tc.name)
			}

			for i, expected := range tc.expected {
				if actualOrder[i] != expected {
					t.Errorf("%s:At position %d: expected %s, got %s", tc.name, i, expected, actualOrder[i])
					t.Errorf("%s:Full expected order: %v", tc.name, tc.expected)
					t.Errorf("%s:Full actual order:   %v", tc.name, actualOrder)
					break
				}
			}
		})
	}
}
