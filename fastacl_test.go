package bart

// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

import (
	"iter"
	"math/rand/v2"
	"net/netip"
	"slices"
	"testing"

	"github.com/gaissmai/bart/internal/tests/golden"
	"github.com/gaissmai/bart/internal/tests/random"
)

// buildFastACL is a test helper that constructs and populates a FastACL from CIDR strings.
func buildFastACL(t *testing.T, cidrs []string) *FastACL {
	t.Helper()

	f := new(FastACL)
	for _, cidr := range cidrs {
		pfx, err := netip.ParsePrefix(cidr)
		if err != nil {
			t.Fatalf("failed to parse CIDR %q: %v", cidr, err)
		}
		f.Insert(pfx)
	}

	return f
}

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

		mustPanic(t, "Insert", func() { tbl1.Insert(pfx4) })
		mustPanic(t, "Delete", func() { tbl1.Delete(pfx4) })
		mustPanic(t, "Contains", func() { tbl1.Contains(ip4) })
		mustPanic(t, "LookupPrefix", func() { tbl1.ContainsPrefix(pfx4) })
		mustPanic(t, "LookupPrefixLPM", func() { tbl1.LookupPrefixLPM(pfx4) })
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
		mustPanicRangeOverFunc[any](t, "Subnets", tbl1.Subnets)
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
	noPanic(t, "Insert", func() { tbl1.Insert(zeroPfx) })
	noPanic(t, "LookupPrefix", func() { tbl1.ContainsPrefix(zeroPfx) })
	noPanic(t, "LookupPrefixLPM", func() { tbl1.LookupPrefixLPM(zeroPfx) })
	noPanic(t, "Overlaps", func() { tbl1.Overlaps(tbl2) })
	noPanic(t, "Overlaps4", func() { tbl1.Overlaps4(tbl2) })
	noPanic(t, "Overlaps6", func() { tbl1.Overlaps6(tbl2) })
	noPanic(t, "OverlapsPrefix", func() { tbl1.OverlapsPrefix(zeroPfx) })
	noPanic(t, "Size", func() { tbl1.Size() })
	noPanic(t, "Size4", func() { tbl1.Size4() })
	noPanic(t, "Size6", func() { tbl1.Size6() })
	noPanic(t, "Subnets", func() { tbl1.Subnets(zeroPfx) })
	noPanic(t, "Supernets", func() { tbl1.Supernets(zeroPfx) })
}

func TestFastACL_Contains_Compare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := random.RealWorldPrefixes(prng, n)

	gold := new(golden.Table[int])
	facl := new(FastACL)

	for i, p := range pfxs {
		gold.Insert(p, i)
		facl.Insert(p)
	}

	for range n {
		ip := random.IP(prng)

		_, goldOK := gold.Lookup(ip)
		faclOK := facl.Contains(ip)

		if goldOK != faclOK {
			t.Fatalf("Contains(%q) = %v, want %v", ip, faclOK, goldOK)
		}
	}
}

func TestFastACL_Contains_Zoned(t *testing.T) {
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

			facl := new(FastACL)
			for _, pfxStr := range tt.pfxs {
				facl.Insert(mpp(pfxStr))
			}

			check(t, facl, mpa(tt.probe), tt.wantOK)
		})
	}
}

func TestFastACL_LookupPrefix_Unmasked(t *testing.T) {
	// test that the pfx must not be masked on input for LookupPrefix
	t.Parallel()

	facl := new(FastACL)
	facl.Insert(mpp("10.20.30.0/24"))
	facl.Insert(mpp("2001:db8::/32"))

	// not normalized pfxs
	tests := []struct {
		probe   netip.Prefix
		wantLPM netip.Prefix
		wantOk  bool
	}{
		{
			probe:   netip.MustParsePrefix("10.20.30.40/0"),
			wantLPM: netip.Prefix{},
			wantOk:  false,
		},
		{
			probe:   netip.MustParsePrefix("10.20.30.40/23"),
			wantLPM: netip.Prefix{},
			wantOk:  false,
		},
		{
			probe:   netip.MustParsePrefix("10.20.30.40/24"),
			wantLPM: mpp("10.20.30.0/24"),
			wantOk:  true,
		},
		{
			probe:   netip.MustParsePrefix("10.20.30.40/25"),
			wantLPM: mpp("10.20.30.0/24"),
			wantOk:  true,
		},
		{
			probe:   mpp("10.20.30.40/32"),
			wantLPM: mpp("10.20.30.0/24"),
			wantOk:  true,
		},
		// IPv6 counterparts
		{
			probe:   netip.MustParsePrefix("2001:db8::1/0"),
			wantLPM: netip.Prefix{},
			wantOk:  false,
		},
		{
			probe:   netip.MustParsePrefix("2001:db8::1/31"),
			wantLPM: netip.Prefix{},
			wantOk:  false,
		},
		{
			probe:   netip.MustParsePrefix("2001:db8::1/32"),
			wantLPM: mpp("2001:db8::/32"),
			wantOk:  true,
		},
		{
			probe:   netip.MustParsePrefix("2001:db8::1/64"),
			wantLPM: mpp("2001:db8::/32"),
			wantOk:  true,
		},
	}

	for _, tc := range tests {
		got := facl.ContainsPrefix(tc.probe)
		if got != tc.wantOk {
			t.Errorf("LookupPrefix non canonical prefix (%s), got: %v, want: %v", tc.probe, got, tc.wantOk)
		}

		lpm, got := facl.LookupPrefixLPM(tc.probe)
		if got != tc.wantOk {
			t.Errorf("LookupPrefixLPM non canonical prefix (%s), got: %v, want: %v", tc.probe, got, tc.wantOk)
		}
		if lpm != tc.wantLPM {
			t.Errorf("LookupPrefixLPM non canonical prefix (%s), got: %v, want: %v", tc.probe, lpm, tc.wantLPM)
		}
	}
}

func TestFastACL_ContainsPrefix_Compare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := random.RealWorldPrefixes(prng, n)

	gold := new(golden.Table[any])
	facl := new(FastACL)
	for _, pfx := range pfxs {
		gold.Insert(pfx, nil)
		facl.Insert(pfx)
	}

	for range n {
		pfx := random.Prefix(prng)

		_, goldOK := gold.LookupPrefix(pfx)
		faclOK := facl.ContainsPrefix(pfx)

		if goldOK != faclOK {
			t.Fatalf("ContainsPrefix(%q) = %v, want %v", pfx, faclOK, goldOK)
		}
	}
}

func TestFastACL_LookupPrefixLPM_Compare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := random.RealWorldPrefixes(prng, n)

	gold := new(golden.Table[any])
	facl := new(FastACL)
	for _, pfx := range pfxs {
		gold.Insert(pfx, nil)
		facl.Insert(pfx)
	}

	for range n {
		pfx := random.Prefix(prng)

		goldLPM, _, goldOK := gold.LookupPrefixLPM(pfx)
		tblLPM, tblOK := facl.LookupPrefixLPM(pfx)

		if goldOK != tblOK {
			t.Fatalf("LookupPrefixLPM(%q) = (_, %v), want (_, %v)", pfx, tblOK, goldOK)
		}

		if goldLPM != tblLPM {
			t.Fatalf("LookupPrefixLPM(%q) = ( %v, _), want ( %v, _)", pfx, tblLPM, goldLPM)
		}

	}
}

func TestFastACL_Insert_Shuffled(t *testing.T) {
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

		facl1 := new(FastACL)
		facl2 := new(FastACL)

		for _, pfx := range pfxs {
			facl1.Insert(pfx)
			facl1.Insert(pfx) // idempotent
		}
		for _, pfx := range pfxs2 {
			facl2.Insert(pfx) // idempotent
		}

		if facl1.dumpString() != facl2.dumpString() {
			t.Fatal("tbl1 and tbl2 have different dumpString representation")
		}
		if !facl1.Equal(facl2) {
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

// TestFastACL_AllSorted verifies the public iterator methods AllSorted, AllSorted4,
// and AllSorted6 for canonical CIDR prefix-sorted traversal and early termination handling.
func TestFastACL_AllSorted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		initial   []netip.Prefix
		mode      string // "all", "v4", "v6"
		stopAfter int    // if > 0, stop after yielding stopAfter items
		want      []netip.Prefix
	}{
		{
			name:      "Empty FastACL produces empty sequence for AllSorted",
			initial:   nil,
			mode:      "all",
			stopAfter: 0,
			want:      nil,
		},
		{
			name: "AllSorted4 returns IPv4 prefixes in strictly sorted CIDR order",
			initial: []netip.Prefix{
				mpp("192.168.1.0/24"),
				mpp("10.0.0.0/8"),
				mpp("0.0.0.0/0"),
				mpp("10.0.0.0/16"),
				mpp("2001:db8::/32"), // IPv6 ignored by AllSorted4
			},
			mode: "v4",
			want: []netip.Prefix{
				mpp("0.0.0.0/0"),
				mpp("10.0.0.0/8"),
				mpp("10.0.0.0/16"),
				mpp("192.168.1.0/24"),
			},
		},
		{
			name: "AllSorted6 returns IPv6 prefixes in strictly sorted CIDR order",
			initial: []netip.Prefix{
				mpp("fe80::/10"),
				mpp("2001:db8:1::/48"),
				mpp("2001:db8::/32"),
				mpp("10.0.0.0/8"), // IPv4 ignored by AllSorted6
			},
			mode: "v6",
			want: []netip.Prefix{
				mpp("2001:db8::/32"),
				mpp("2001:db8:1::/48"),
				mpp("fe80::/10"),
			},
		},
		{
			name: "AllSorted yields all IPv4 prefixes followed by all IPv6 prefixes",
			initial: []netip.Prefix{
				mpp("2001:db8::/32"),
				mpp("192.168.0.0/16"),
				mpp("10.0.0.0/8"),
				mpp("fe80::/10"),
			},
			mode: "all",
			want: []netip.Prefix{
				mpp("10.0.0.0/8"),
				mpp("192.168.0.0/16"),
				mpp("2001:db8::/32"),
				mpp("fe80::/10"),
			},
		},
		{
			name: "AllSorted early termination during IPv4 phase halts entire iteration",
			initial: []netip.Prefix{
				mpp("10.0.0.0/8"),
				mpp("192.168.0.0/16"),
				mpp("2001:db8::/32"),
			},
			mode:      "all",
			stopAfter: 1,
			want: []netip.Prefix{
				mpp("10.0.0.0/8"),
			},
		},
		{
			name: "AllSorted early termination during IPv6 phase halts iteration",
			initial: []netip.Prefix{
				mpp("10.0.0.0/8"),
				mpp("2001:db8::/32"),
				mpp("fe80::/10"),
			},
			mode:      "all",
			stopAfter: 2,
			want: []netip.Prefix{
				mpp("10.0.0.0/8"),
				mpp("2001:db8::/32"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var facl FastACL
			for _, pfx := range tt.initial {
				facl.Insert(pfx)
			}

			var seq iter.Seq[netip.Prefix]
			switch tt.mode {
			case "all":
				seq = facl.AllSorted()
			case "v4":
				seq = facl.AllSorted4()
			case "v6":
				seq = facl.AllSorted6()
			default:
				t.Fatalf("unsupported mode: %s", tt.mode)
			}

			var got []netip.Prefix
			count := 0
			seq(func(pfx netip.Prefix) bool {
				got = append(got, pfx)
				count++
				if tt.stopAfter > 0 && count >= tt.stopAfter {
					return false
				}
				return true
			})

			if !slices.Equal(got, tt.want) {
				t.Errorf("AllSorted sequence mismatch:\ngot:  %v\nwant: %v", got, tt.want)
			}
		})
	}
}

func TestFastACL_AllSorted_Compare(t *testing.T) {
	t.Parallel()

	n := workLoadN()
	prng := rand.New(rand.NewPCG(42, 42))

	for range 3 {
		pfxs := random.RealWorldPrefixes(prng, n)

		gold := new(golden.Table[any])
		facl := new(FastACL)

		for _, pfx := range pfxs {
			gold.Insert(pfx, nil)
			facl.Insert(pfx)
		}

		goldFlat := gold.FlatSorted()
		faclSorted := slices.Collect(facl.AllSorted())

		if !slices.Equal(goldFlat.SortKeys(), faclSorted) {
			t.Fatal("expected Equal")
		}
	}
}

func TestFastACL_Aggregate(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name: "empty",
		},
		{
			name:  "duplicate canonical prefix",
			input: []string{"192.0.2.0/24", "192.0.2.0/24"},
			want:  []string{"192.0.2.0/24"},
		},
		{
			name:  "prefix subsumption",
			input: []string{"10.1.2.0/24", "10.1.0.0/16", "10.0.0.0/8"},
			want:  []string{"10.0.0.0/8"},
		},
		{
			name:  "child subsumption",
			input: []string{"192.0.2.1/32", "192.0.2.128/25", "192.0.2.0/24"},
			want:  []string{"192.0.2.0/24"},
		},
		{
			name:  "ipv4 adjacent prefixes",
			input: []string{"192.0.2.0/25", "192.0.2.128/25"},
			want:  []string{"192.0.2.0/24"},
		},
		{
			name: "ipv4 recursive leaf merging",
			input: []string{
				"192.0.2.0/32",
				"192.0.2.1/32",
				"192.0.2.2/32",
				"192.0.2.3/32",
			},
			want: []string{"192.0.2.0/30"},
		},
		{
			name: "Promote each child after recursive aggregation",
			input: []string{
				"10.0.0.0/25",
				"10.0.0.128/25",
				"10.0.1.0/24",
			},
			want: []string{"10.0.0.0/23"},
		},
		{
			name: "ipv4 recursive fringe merging",
			input: []string{
				"8.0.0.0/8",
				"9.0.0.0/8",
				"10.0.0.0/8",
				"11.0.0.0/8",
			},
			want: []string{"8.0.0.0/6"},
		},
		{
			name:  "non siblings remain separate",
			input: []string{"10.0.0.0/8", "12.0.0.0/8"},
			want:  []string{"10.0.0.0/8", "12.0.0.0/8"},
		},
		{
			name:  "different prefix lengths remain separate",
			input: []string{"192.0.2.0/25", "192.0.2.128/26"},
			want:  []string{"192.0.2.0/25", "192.0.2.128/26"},
		},
		{
			name:  "ipv6 adjacent prefixes",
			input: []string{"2001:db8::/65", "2001:db8:0:0:8000::/65"},
			want:  []string{"2001:db8::/64"},
		},
		{
			name: "ipv6 recursive leaf merging",
			input: []string{
				"2001:db8::/128",
				"2001:db8::1/128",
				"2001:db8::2/128",
				"2001:db8::3/128",
			},
			want: []string{"2001:db8::/126"},
		},
		{
			name:  "default routes subsume their family",
			input: []string{"0.0.0.0/0", "10.0.0.0/8", "128.0.0.0/1", "::/0", "2001:db8::/32"},
			want:  []string{"0.0.0.0/0", "::/0"},
		},
		{
			name:  "both address families",
			input: []string{"192.0.2.0/25", "192.0.2.128/25", "2001:db8::/65", "2001:db8:0:0:8000::/65"},
			want:  []string{"192.0.2.0/24", "2001:db8::/64"},
		},

		// more corner cases
		{
			name:  "adjacent leaf siblings ipv4",
			input: []string{"10.0.0.0/24", "10.0.1.0/24"},
			want:  []string{"10.0.0.0/23"},
		},
		{
			name:  "adjacent non-byte-aligned siblings ipv4",
			input: []string{"10.0.0.0/25", "10.0.0.128/25"},
			want:  []string{"10.0.0.0/24"},
		},
		{
			name:  "adjacent siblings ipv6",
			input: []string{"2001:db8::/53", "2001:db8:0:0800::/53"},
			want:  []string{"2001:db8::/52"},
		},
		{
			name:  "zero address fringe pair ipv4",
			input: []string{"0.0.0.0/8", "1.0.0.0/8"},
			want:  []string{"0.0.0.0/7"},
		},
		{
			name:  "zero address fringe pair ipv6",
			input: []string{"::/8", "100::/8"},
			want:  []string{"::/7"},
		},
		{
			name: "covered child with multiple routes",
			input: []string{
				"10.0.0.0/8",
				"10.1.0.0/17",
				"10.1.128.0/17",
				"10.2.0.0/17",
				"10.2.128.0/17",
			},
			want: []string{"10.0.0.0/8"},
		},
		{
			name: "cascading prefix merges",
			input: []string{
				"0.0.0.0/5",
				"8.0.0.0/5",
				"16.0.0.0/5",
				"24.0.0.0/5",
			},
			want: []string{"0.0.0.0/3"},
		},
		{
			name:  "partial sibling set",
			input: []string{"0.0.0.0/8", "1.0.0.0/8", "2.0.0.0/8"},
			want:  []string{"0.0.0.0/7", "2.0.0.0/8"},
		},
		{
			name:  "non-adjacent prefixes remain separate",
			input: []string{"10.0.0.0/24", "10.0.2.0/24"},
			want:  []string{"10.0.0.0/24", "10.0.2.0/24"},
		},
		{
			name:  "single prefix",
			input: []string{"192.0.2.1/32"},
			want:  []string{"192.0.2.1/32"},
		},
		{
			name:  "default route per family",
			input: []string{"0.0.0.0/0", "10.0.0.0/8", "::/0", "2001:db8::/32"},
			want:  []string{"0.0.0.0/0", "::/0"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			facl := new(FastACL)
			for _, prefix := range parseAndCollect(test.input) {
				facl.Insert(prefix)
			}

			facl.Aggregate()

			want := parseAndCollect(test.want)
			got := slices.Collect(facl.AllSorted())
			if !slices.Equal(got, want) {
				t.Fatalf("%s: Aggregate(), got: %v, want: %v", test.name, got, want)
			}

			want4, want6 := 0, 0
			for _, prefix := range want {
				if prefix.Addr().Is4() {
					want4++
				} else {
					want6++
				}
			}
			if facl.Size() != len(want) || facl.Size4() != want4 || facl.Size6() != want6 {
				t.Fatalf("%s: sizes, got: (%d, %d, %d), want: (%d, %d, %d)",
					test.name, facl.Size(), facl.Size4(), facl.Size6(), len(want), want4, want6)
			}

			first := slices.Collect(facl.AllSorted())
			facl.Aggregate()
			second := slices.Collect(facl.AllSorted())
			if !slices.Equal(second, first) {
				t.Fatalf("%s: Aggregate() is not idempotent: first %v, second %v", test.name, first, second)
			}
		})
	}
}

func TestFastACL_Aggregate_PreservesMembership(t *testing.T) {
	tests := []struct {
		name   string
		input  []string
		probes []string
	}{
		{
			name:  "ipv4 boundaries",
			input: []string{"192.0.2.0/25", "192.0.2.128/25"},
			probes: []string{
				"192.0.1.255",
				"192.0.2.0",
				"192.0.2.127",
				"192.0.2.128",
				"192.0.2.255",
				"192.0.3.0",
			},
		},
		{
			name:  "ipv6 boundaries",
			input: []string{"2001:db8::/65", "2001:db8:0:0:8000::/65"},
			probes: []string{
				"2001:db7:ffff:ffff:ffff:ffff:ffff:ffff",
				"2001:db8::",
				"2001:db8:0:0:ffff:ffff:ffff:ffff",
				"2001:db9::",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := new(FastACL)
			for _, prefix := range parseAndCollect(test.input) {
				before.Insert(prefix)
			}

			after := before.Clone()
			after.Aggregate()

			for _, address := range test.probes {
				ip := netip.MustParseAddr(address)

				beforeContains := before.Contains(ip)
				afterContains := after.Contains(ip)

				if afterContains != beforeContains {
					t.Errorf("Contains(%s) changed from %v to %v", address, beforeContains, afterContains)
				}
			}
		})
	}
}

func TestFastACL_Aggregate_Compare(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	for i := range n {
		t.Run("subtest", func(t *testing.T) {
			t.Parallel()

			prng := rand.New(rand.NewPCG(uint64(n), uint64(i)))
			pfxs := random.RealWorldPrefixes(prng, n)

			gold := new(golden.Table[any])
			facl := new(FastACL)

			for _, pfx := range pfxs {
				gold.Insert(pfx, nil)
				facl.Insert(pfx)
			}

			gold.Aggregate()
			facl.Aggregate()

			goldSorted := gold.FlatSorted().SortKeys()
			faclSorted := slices.Collect(facl.AllSorted())

			if !slices.Equal(goldSorted, faclSorted) {
				t.Fatal("Aggregate(): tables are different!")
			}
		})
	}
}

func TestFastACL_Aggregate_StructuralCompare(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	for i := range 50 {
		t.Run("subtest", func(t *testing.T) {
			t.Parallel()

			prng := rand.New(rand.NewPCG(uint64(n), uint64(i)))
			pfxs := random.RealWorldPrefixes(prng, n)

			gold := new(golden.Table[any])
			facl1 := new(FastACL)
			facl2 := new(FastACL)

			for _, pfx := range pfxs {
				gold.Insert(pfx, nil)
				facl1.Insert(pfx)
			}

			gold.Aggregate()
			facl1.Aggregate()

			// build facl2 with aggregated prefixes
			for pfx := range gold.All() {
				facl2.Insert(pfx)
			}

			facl1Sorted := slices.Collect(facl1.AllSorted())
			facl2Sorted := slices.Collect(facl2.AllSorted())

			if !slices.Equal(facl1Sorted, facl2Sorted) {
				t.Fatal("Aggregate(): the tables have different prefixes!")
			}

			if facl1.dumpString() != facl2.dumpString() {
				t.Fatal("Aggregate(): the tables have mismatched internal structures!")
			}
		})
	}
}

func TestFastACL_Supernets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		query    string
		inserts  []netip.Prefix
		want     []netip.Prefix
		wantExit bool // Test early break in yield callback via range break
	}{
		{
			name:    "invalid prefix query",
			query:   "invalid",
			inserts: []netip.Prefix{mpp("10.0.0.0/8")},
			want:    nil,
		},
		{
			name:    "empty table",
			query:   "10.0.0.0/16",
			inserts: nil,
			want:    nil,
		},
		{
			name:    "exact match only",
			query:   "10.1.2.0/24",
			inserts: []netip.Prefix{mpp("10.1.2.0/24")},
			want:    []netip.Prefix{mpp("10.1.2.0/24")},
		},
		{
			name:    "uncanonical query normalization",
			query:   "10.1.2.255/24", // Host bits set
			inserts: []netip.Prefix{mpp("10.0.0.0/8"), mpp("10.1.2.0/24")},
			want:    []netip.Prefix{mpp("10.1.2.0/24"), mpp("10.0.0.0/8")},
		},
		{
			name:  "exact match and ascending supernets",
			query: "10.1.2.128/25",
			inserts: []netip.Prefix{
				mpp("0.0.0.0/0"),
				mpp("10.0.0.0/8"),
				mpp("10.1.0.0/16"),
				mpp("10.1.2.0/24"),
				mpp("10.1.2.128/25"),
			},
			want: []netip.Prefix{
				mpp("10.1.2.128/25"),
				mpp("10.1.2.0/24"),
				mpp("10.1.0.0/16"),
				mpp("10.0.0.0/8"),
				mpp("0.0.0.0/0"),
			},
		},
		{
			name:  "disjoint branches - skip non-matching siblings",
			query: "10.1.2.0/24",
			inserts: []netip.Prefix{
				mpp("10.0.0.0/8"),
				mpp("10.2.0.0/16"),    // Divergent stride
				mpp("10.1.2.0/24"),    // Target match
				mpp("10.1.3.0/24"),    // Divergent sibling
				mpp("192.168.0.0/16"), // Completely separate tree
			},
			want: []netip.Prefix{
				mpp("10.1.2.0/24"),
				mpp("10.0.0.0/8"),
			},
		},
		{
			name:  "stride boundary fringes (/8, /16, /24)",
			query: "10.1.2.4/30",
			inserts: []netip.Prefix{
				mpp("10.0.0.0/8"),  // Stride 1 fringe
				mpp("10.1.0.0/16"), // Stride 2 fringe
				mpp("10.1.2.0/24"), // Stride 3 fringe
			},
			want: []netip.Prefix{
				mpp("10.1.2.0/24"),
				mpp("10.1.0.0/16"),
				mpp("10.0.0.0/8"),
			},
		},
		{
			name:  "cbt internal node prefixes within same stride",
			query: "10.1.2.0/27",
			inserts: []netip.Prefix{
				mpp("10.1.2.0/24"),
				mpp("10.1.2.0/25"),
				mpp("10.1.2.0/26"),
				mpp("10.1.2.0/27"),
			},
			want: []netip.Prefix{
				mpp("10.1.2.0/27"),
				mpp("10.1.2.0/26"),
				mpp("10.1.2.0/25"),
				mpp("10.1.2.0/24"),
			},
		},
		{
			name:  "path-compressed cidr leaf match first",
			query: "10.1.2.128/28",
			inserts: []netip.Prefix{
				mpp("10.0.0.0/8"),
				mpp("10.1.2.128/28"), // Unbranched path compression -> CIDRLeaf
			},
			want: []netip.Prefix{
				mpp("10.1.2.128/28"), // Leaf yielded first in Phase 1
				mpp("10.0.0.0/8"),    // Root supernet yielded during Phase 2 stack unwinding
			},
		},
		{
			name:  "path-compressed leaf skipped if query is broader than leaf",
			query: "10.1.2.0/24", // Query is broader (/24) than leaf (/28)
			inserts: []netip.Prefix{
				mpp("10.0.0.0/8"),
				mpp("10.1.2.128/28"), // Specific leaf in trie
			},
			want: []netip.Prefix{
				mpp("10.0.0.0/8"), // Should not yield /28 because it's a subnet, not a supernet
			},
		},
		{
			name:  "ipv6 deep nested supernets",
			query: "2001:db8:85a3:8a2e:370::/80",
			inserts: []netip.Prefix{
				mpp("::/0"),
				mpp("2001:db8::/32"),
				mpp("2001:db8:85a3::/48"),
				mpp("2001:db8:85a3:8a2e::/64"),
				mpp("2001:db8:85a3:8a2e:370::/80"),
			},
			want: []netip.Prefix{
				mpp("2001:db8:85a3:8a2e:370::/80"),
				mpp("2001:db8:85a3:8a2e::/64"),
				mpp("2001:db8:85a3::/48"),
				mpp("2001:db8::/32"),
				mpp("::/0"),
			},
		},
		{
			name:  "early exit on yield false",
			query: "10.1.2.0/24",
			inserts: []netip.Prefix{
				mpp("10.0.0.0/8"),
				mpp("10.1.0.0/16"),
				mpp("10.1.2.0/24"),
			},
			wantExit: true,
			want: []netip.Prefix{
				mpp("10.1.2.0/24"), // Should stop after first yield via break
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tbl := new(FastACL)
			for _, pfx := range tc.inserts {
				tbl.Insert(pfx)
			}

			queryPfx, err := netip.ParsePrefix(tc.query)
			if err != nil && tc.query != "invalid" {
				t.Fatalf("unexpected test setup parse error for query %q: %v", tc.query, err)
			}

			var got []netip.Prefix

			if tc.wantExit {
				// Verify early termination semantics using native range-over-func with break.
				for pfx := range tbl.Supernets(queryPfx) {
					got = append(got, pfx)
					break // Stop iteration immediately after receiving the first result.
				}
			} else {
				// Collect all yielded netip.Prefix values directly from the iterator.
				got = slices.Collect(tbl.Supernets(queryPfx))
			}

			if !slices.Equal(got, tc.want) {
				t.Errorf("Supernets(%q) mismatch:\ngot:  %v\nwant: %v", tc.query, got, tc.want)
			}
		})
	}
}

func TestFastACL_Supernets_Compare(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))

	pfxs := random.RealWorldPrefixes(prng, n)

	gold := new(golden.Table[any])
	facl := new(FastACL)

	for _, pfx := range pfxs {
		gold.Insert(pfx, nil)
		facl.Insert(pfx)
	}

	for _, pfx := range random.RealWorldPrefixes(prng, n) {
		t.Run("subtest", func(t *testing.T) {
			t.Parallel()
			goldGot := gold.Supernets(pfx)
			faclGot := slices.Collect(facl.Supernets(pfx))

			if !slices.Equal(goldGot, faclGot) {
				t.Fatalf("Supernets(%q) = %v, want %v", pfx, faclGot, goldGot)
			}
		})
	}
}

func TestFastACL_Subnets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		routes   []string // Prefixes to insert into FastACL
		query    string   // Target prefix passed to Subnets()
		want     []string // Expected subnets in natural CIDR sort order
		maxYield int      // If > 0, stop range iteration early after N items
	}{
		{
			name:   "invalid target prefix yields nothing",
			routes: []string{"10.0.0.0/8"},
			query:  "invalid-pfx",
			want:   nil,
		},
		{
			name:   "empty ACL yields nothing",
			routes: nil,
			query:  "10.0.0.0/8",
			want:   nil,
		},
		{
			name:   "exact match single prefix",
			routes: []string{"10.0.0.0/8"},
			query:  "10.0.0.0/8",
			want:   []string{"10.0.0.0/8"},
		},
		{
			name:   "unmasked prefix query is normalized automatically",
			routes: []string{"10.1.2.0/24"},
			query:  "10.1.2.255/24", // Normalized to 10.1.2.0/24
			want:   []string{"10.1.2.0/24"},
		},
		{
			name: "IPv4 nested subnets in canonical CIDR order",
			routes: []string{
				"10.0.0.0/8",
				"10.1.0.0/16",
				"10.1.1.0/24",
				"10.1.2.0/24",
				"10.2.0.0/16",
				"192.168.0.0/16",
			},
			query: "10.0.0.0/8",
			want: []string{
				"10.0.0.0/8",
				"10.1.0.0/16",
				"10.1.1.0/24",
				"10.1.2.0/24",
				"10.2.0.0/16",
			},
		},
		{
			name:   "pruning non-matching sibling subtrees",
			routes: []string{"10.1.0.0/16", "10.1.1.0/24", "10.2.0.0/16"},
			query:  "10.1.0.0/16",
			want: []string{
				"10.1.0.0/16",
				"10.1.1.0/24",
			},
		},
		{
			name: "IPv6 subnets across stride boundaries",
			routes: []string{
				"2001:db8::/32",
				"2001:db8:1000::/36",
				"2001:db8:1000::/48",
				"2001:db8:2000::/36",
				"2001:dc0::/32",
			},
			query: "2001:db8::/32",
			want: []string{
				"2001:db8::/32",
				"2001:db8:1000::/36",
				"2001:db8:1000::/48",
				"2001:db8:2000::/36",
			},
		},
		{
			name:     "early iteration break",
			routes:   []string{"10.0.0.0/8", "10.1.0.0/16", "10.1.1.0/24", "10.2.0.0/16"},
			query:    "10.0.0.0/8",
			want:     []string{"10.0.0.0/8", "10.1.0.0/16"},
			maxYield: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			facl := new(FastACL)
			for _, r := range tt.routes {
				pfx := mpp(r)
				facl.Insert(pfx)
			}

			var target netip.Prefix
			if tt.query != "invalid-pfx" {
				target = netip.MustParsePrefix(tt.query)
			}

			var got []string
			for pfx := range facl.Subnets(target) {
				got = append(got, pfx.String())
				if tt.maxYield > 0 && len(got) == tt.maxYield {
					break
				}
			}

			if !slices.Equal(got, tt.want) {
				t.Errorf("Subnets(%q) mismatch:\n  got:  %v\n  want: %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestFastACL_Subnets_Compare(t *testing.T) {
	t.Parallel()
	n := workLoadN()
	prng := rand.New(rand.NewPCG(42, 42))

	pfxs := random.RealWorldPrefixes(prng, n)

	gold := new(golden.Table[any])
	facl := new(FastACL)

	for _, pfx := range pfxs {
		gold.Insert(pfx, nil)
		facl.Insert(pfx)
	}

	for _, pfx := range random.RealWorldPrefixes(prng, n) {
		t.Run("subtest", func(t *testing.T) {
			t.Parallel()

			goldGot := gold.Subnets(pfx)
			faclGot := slices.Collect(facl.Subnets(pfx))

			if !slices.Equal(goldGot, faclGot) {
				t.Fatalf("Subnets(%q) = %v, want %v", pfx, faclGot, goldGot)
			}
		})
	}
}

func TestFastACL_Union(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		base      []string
		other     []string
		wantSize4 int
		wantSize6 int
	}{
		{
			name:      "union disjoint IPv4 prefixes",
			base:      []string{"192.168.1.0/24", "10.0.0.0/8"},
			other:     []string{"172.16.0.0/12", "192.168.2.0/24"},
			wantSize4: 4,
			wantSize6: 0,
		},
		{
			name:      "union overlapping and duplicate IPv4 prefixes",
			base:      []string{"192.168.0.0/16", "10.0.0.0/24"},
			other:     []string{"192.168.1.0/24", "10.0.0.0/24"}, // 10.0.0.0/24 is duplicate
			wantSize4: 3,                                         // /16, /24, /24
			wantSize6: 0,
		},
		{
			name:      "union disjoint IPv6 prefixes",
			base:      []string{"2001:db8::/32"},
			other:     []string{"fe80::/10", "2001:db8:1::/48"},
			wantSize4: 0,
			wantSize6: 3,
		},
		{
			name:      "union dual-stack mixed prefixes",
			base:      []string{"192.168.1.0/24", "2001:db8::/32"},
			other:     []string{"10.0.0.0/8", "fe80::/10"},
			wantSize4: 2,
			wantSize6: 2,
		},
		{
			name:      "union empty other table into non-empty base",
			base:      []string{"192.168.1.0/24"},
			other:     []string{},
			wantSize4: 1,
			wantSize6: 0,
		},
		{
			name:      "union empty base with non empty other table",
			base:      []string{},
			other:     []string{"192.168.1.0/24"},
			wantSize4: 1,
			wantSize6: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Initialize base FastACL table
			facl := new(FastACL)
			for _, pfxStr := range tt.base {
				facl.Insert(mpp(pfxStr))
			}

			// Initialize other FastACL table
			other := new(FastACL)
			for _, pfxStr := range tt.other {
				other.Insert(mpp(pfxStr))
			}

			// Execute Union operation
			facl.Union(other)

			// Verify final sizes match expectations
			if got := facl.Size4(); got != tt.wantSize4 {
				t.Errorf("FastACL.Union() Size4 = %v, want %v", got, tt.wantSize4)
			}
			if got := facl.Size6(); got != tt.wantSize6 {
				t.Errorf("FastACL.Union() Size6 = %v, want %v", got, tt.wantSize6)
			}
		})
	}
}

func TestFastACL_Union_Compare(t *testing.T) {
	t.Parallel()
	n := workLoadN()
	prng := rand.New(rand.NewPCG(42, 42))

	for range 3 {
		pfxs := random.RealWorldPrefixes(prng, n)

		gold := new(golden.Table[any])
		facl := new(FastACL)

		for _, pfx := range pfxs {
			gold.Insert(pfx, nil)
			facl.Insert(pfx)
		}

		pfxs2 := random.RealWorldPrefixes(prng, n)

		gold2 := new(golden.Table[any])
		facl2 := new(FastACL)

		for _, pfx := range pfxs2 {
			gold2.Insert(pfx, nil)
			facl2.Insert(pfx)
		}

		gold.Union(*gold2)
		facl.Union(facl2)

		goldFlat := gold.FlatSorted()
		faclSorted := slices.Collect(facl.AllSorted())

		if !slices.Equal(goldFlat.SortKeys(), faclSorted) {
			t.Fatal("expected Equal")
		}
	}
}

func TestFastACL_OverlapsPrefix(t *testing.T) {
	t.Parallel()

	type probe struct {
		pfx  netip.Prefix
		want bool
	}

	type probes []probe
	type pfxs []netip.Prefix

	type test struct {
		name   string
		insert pfxs
		probes probes
	}

	tests := []test{
		{
			name:   "empty table",
			insert: nil,
			probes: probes{{mpp("0.0.0.0/0"), false}, {mpp("::/0"), false}},
		},
		{
			name:   "default route I",
			insert: pfxs{mpp("10.0.0.0/9"), mpp("2001:db8::/32")},
			probes: probes{{mpp("0.0.0.0/0"), true}, {mpp("::/0"), true}},
		},
		{
			name:   "default route II",
			insert: pfxs{mpp("0.0.0.0/0"), mpp("::/0")},
			probes: probes{{mpp("10.0.0.0/9"), true}, {mpp("2001:db8::/32"), true}},
		},
		{
			name:   "single IP I",
			insert: pfxs{mpp("10.0.0.0/7"), mpp("2001::/16")},
			probes: probes{{mpp("10.1.2.3/32"), true}, {mpp("2001:db8:affe::cafe/128"), true}},
		},
		{
			name:   "single IP II",
			insert: pfxs{mpp("10.1.2.3/32"), mpp("2001:db8:affe::cafe/128")},
			probes: probes{{mpp("10.0.0.0/7"), true}, {mpp("2001::/16"), true}},
		},
		{
			name:   "same IP",
			insert: pfxs{mpp("10.1.2.3/32"), mpp("2001:db8:affe::cafe/128")},
			probes: probes{{mpp("10.1.2.3/32"), true}, {mpp("2001:db8:affe::cafe/128"), true}},
		},
		{
			name:   "full expanded path",
			insert: pfxs{mpp("10.1.2.3/32"), mpp("10.1.2.4/32")},
			probes: probes{{mpp("10.1.2.3/32"), true}, {mpp("10.1.2.5/32"), false}},
		},
	}

	for _, tt := range tests {
		facl := new(FastACL)
		for _, pfx := range tt.insert {
			facl.Insert(pfx)
		}

		for _, probe := range tt.probes {
			got := facl.OverlapsPrefix(probe.pfx)
			if got != probe.want {
				t.Errorf("[%s] OverlapsPrefix(%v) = %v, want %v", tt.name, probe.pfx, got, probe.want)
			}
		}
	}
}

func TestFastACL_OverlapsPrefix_Compare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := random.RealWorldPrefixes(prng, n)

	gold := new(golden.Table[any])
	facl := new(FastACL)
	for _, pfx := range pfxs {
		gold.Insert(pfx, nil)
		facl.Insert(pfx)
	}

	for range n {
		pfx := random.Prefix(prng)

		goldOK := gold.OverlapsPrefix(pfx)
		faclOK := facl.OverlapsPrefix(pfx)

		if goldOK != faclOK {
			t.Fatalf("OverlapsPrefix(%q) = %v, want %v", pfx, faclOK, goldOK)
		}
	}
}

func TestFastACL_Overlaps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		routes1   []string
		routes2   []string
		wantTotal bool
		wantV4    bool
		wantV6    bool
	}{
		{
			name:      "both empty tables",
			routes1:   nil,
			routes2:   nil,
			wantTotal: false,
			wantV4:    false,
			wantV6:    false,
		},
		{
			name:      "receiver empty",
			routes1:   nil,
			routes2:   []string{"10.0.0.0/8", "2001:db8::/32"},
			wantTotal: false,
			wantV4:    false,
			wantV6:    false,
		},
		{
			name:      "other empty",
			routes1:   []string{"10.0.0.0/8", "2001:db8::/32"},
			routes2:   nil,
			wantTotal: false,
			wantV4:    false,
			wantV6:    false,
		},
		{
			name:      "disjoint IPv4 and IPv6",
			routes1:   []string{"10.0.0.0/16", "2001:db8:1::/48"},
			routes2:   []string{"10.1.0.0/16", "2001:db8:2::/48"},
			wantTotal: false,
			wantV4:    false,
			wantV6:    false,
		},
		{
			name:      "exact match IPv4",
			routes1:   []string{"192.168.1.0/24"},
			routes2:   []string{"192.168.1.0/24"},
			wantTotal: true,
			wantV4:    true,
			wantV6:    false,
		},
		{
			name:      "exact match IPv6",
			routes1:   []string{"fe80::/10"},
			routes2:   []string{"fe80::/10"},
			wantTotal: true,
			wantV4:    false,
			wantV6:    true,
		},
		{
			name:      "IPv4 subnet overlap (receiver contains other)",
			routes1:   []string{"10.0.0.0/8"},
			routes2:   []string{"10.1.2.0/24"},
			wantTotal: true,
			wantV4:    true,
			wantV6:    false,
		},
		{
			name:      "IPv4 supernet overlap (other contains receiver)",
			routes1:   []string{"172.16.10.0/24"},
			routes2:   []string{"172.16.0.0/12"},
			wantTotal: true,
			wantV4:    true,
			wantV6:    false,
		},
		{
			name:      "IPv6 subnet overlap (receiver contains other)",
			routes1:   []string{"2001:db8::/32"},
			routes2:   []string{"2001:db8:abcd::/48"},
			wantTotal: true,
			wantV4:    false,
			wantV6:    true,
		},
		{
			name:      "IPv6 supernet overlap (other contains receiver)",
			routes1:   []string{"2001:db8:ffff::/48"},
			routes2:   []string{"2001:db8::/32"},
			wantTotal: true,
			wantV4:    false,
			wantV6:    true,
		},
		{
			name:      "mixed dual-stack: only IPv4 overlaps",
			routes1:   []string{"10.0.0.0/16", "2001:db8:1::/48"},
			routes2:   []string{"10.0.1.0/24", "2001:db8:2::/48"},
			wantTotal: true,
			wantV4:    true,
			wantV6:    false,
		},
		{
			name:      "mixed dual-stack: only IPv6 overlaps",
			routes1:   []string{"10.0.0.0/16", "2001:db8::/32"},
			routes2:   []string{"10.1.0.0/16", "2001:db8:1234::/48"},
			wantTotal: true,
			wantV4:    false,
			wantV6:    true,
		},
		{
			name:      "mixed dual-stack: both IPv4 and IPv6 overlap",
			routes1:   []string{"10.0.0.0/8", "2001:db8::/32"},
			routes2:   []string{"10.1.0.0/16", "2001:db8:abcd::/48"},
			wantTotal: true,
			wantV4:    true,
			wantV6:    true,
		},
		{
			name:      "default route IPv4 (/0) overlap",
			routes1:   []string{"0.0.0.0/0"},
			routes2:   []string{"192.168.0.1/32"},
			wantTotal: true,
			wantV4:    true,
			wantV6:    false,
		},
		{
			name:      "default route IPv6 (/0) overlap",
			routes1:   []string{"::/0"},
			routes2:   []string{"2001:db8::1/128"},
			wantTotal: true,
			wantV4:    false,
			wantV6:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			facl1 := buildFastACL(t, tt.routes1)
			facl2 := buildFastACL(t, tt.routes2)

			if got := facl1.Overlaps(facl2); got != tt.wantTotal {
				t.Errorf("FastACL.Overlaps() = %v, want %v", got, tt.wantTotal)
			}

			if got := facl1.Overlaps4(facl2); got != tt.wantV4 {
				t.Errorf("FastACL.Overlaps4() = %v, want %v", got, tt.wantV4)
			}

			if got := facl1.Overlaps6(facl2); got != tt.wantV6 {
				t.Errorf("FastACL.Overlaps6() = %v, want %v", got, tt.wantV6)
			}

			// Verify bidirectionality (facl2.Overlaps(facl1) must yield the exact same result)
			if got := facl2.Overlaps(facl1); got != tt.wantTotal {
				t.Errorf("FastACL.Overlaps() bidirectional = %v, want %v", got, tt.wantTotal)
			}

			if got := facl2.Overlaps4(facl1); got != tt.wantV4 {
				t.Errorf("FastACL.Overlaps4() bidirectional = %v, want %v", got, tt.wantV4)
			}

			if got := facl2.Overlaps6(facl1); got != tt.wantV6 {
				t.Errorf("FastACL.Overlaps6() bidirectional = %v, want %v", got, tt.wantV6)
			}
		})
	}
}

func TestFastACL_Compare(t *testing.T) {
	t.Parallel()

	n := workLoadN()
	prng := rand.New(rand.NewPCG(42, 42))

	for range 10 {
		t.Run("subtest", func(t *testing.T) {
			t.Parallel()

			pfxs := random.RealWorldPrefixes(prng, n)

			gold1 := new(golden.Table[any])
			facl1 := new(FastACL)

			for _, pfx := range pfxs {
				gold1.Insert(pfx, nil)
				facl1.Insert(pfx)
			}

			pfxs2 := random.RealWorldPrefixes(prng, n)

			gold2 := new(golden.Table[any])
			facl2 := new(FastACL)

			for _, pfx := range pfxs2 {
				gold2.Insert(pfx, nil)
				facl2.Insert(pfx)
			}

			goldGot := gold1.Overlaps(*gold2)
			faclGot := facl1.Overlaps(facl2)

			if goldGot != faclGot {
				t.Fatal("Overlaps is different")
			}
		})
	}
}
