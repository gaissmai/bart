package bart

import (
	"net/netip"
	"slices"
	"testing"
)

func aggregatePrefixes(values []string) []netip.Prefix {
	prefixes := make([]netip.Prefix, len(values))
	for i, value := range values {
		prefixes[i] = netip.MustParsePrefix(value)
	}
	return prefixes
}

func collectedPrefixes(table *Lite) []netip.Prefix {
	prefixes := make([]netip.Prefix, 0, table.Size())
	for prefix := range table.AllSorted() {
		prefixes = append(prefixes, prefix)
	}
	return prefixes
}

func TestLiteAggregate(t *testing.T) {
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
			input: []string{"192.0.2.1/24", "192.0.2.0/24"},
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			table := new(Lite)
			for _, prefix := range aggregatePrefixes(test.input) {
				table.Insert(prefix)
			}

			table.Aggregate()

			want := aggregatePrefixes(test.want)
			got := collectedPrefixes(table)
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
			if table.Size() != len(want) || table.Size4() != want4 || table.Size6() != want6 {
				t.Fatalf("%s: sizes, got: (%d, %d, %d), want: (%d, %d, %d)", test.name, table.Size(), table.Size4(), table.Size6(), len(want), want4, want6)
			}

			first := collectedPrefixes(table)
			table.Aggregate()
			second := collectedPrefixes(table)
			if !slices.Equal(second, first) {
				t.Fatalf("%s: Aggregate() is not idempotent: first %v, second %v", test.name, first, second)
			}
		})
	}
}

func TestLiteAggregatePreservesMembership(t *testing.T) {
	tests := []struct {
		name   string
		input  []string
		probes []string
	}{
		{
			name:   "ipv4 boundaries",
			input:  []string{"192.0.2.0/25", "192.0.2.128/25"},
			probes: []string{"192.0.1.255", "192.0.2.0", "192.0.2.127", "192.0.2.128", "192.0.2.255", "192.0.3.0"},
		},
		{
			name:   "ipv6 boundaries",
			input:  []string{"2001:db8::/65", "2001:db8:0:0:8000::/65"},
			probes: []string{"2001:db7:ffff:ffff:ffff:ffff:ffff:ffff", "2001:db8::", "2001:db8:0:0:ffff:ffff:ffff:ffff", "2001:db9::"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := new(Lite)
			for _, prefix := range aggregatePrefixes(test.input) {
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

func TestLiteAggregateInvalidAndNil(t *testing.T) {
	table := new(Lite)
	table.Insert(netip.Prefix{})
	table.Aggregate()
	if table.Size() != 0 {
		t.Fatalf("Aggregate() inserted an invalid prefix, size = %d", table.Size())
	}

	var nilTable *Lite
	mustPanic(t, "Aggregate", func() { nilTable.Aggregate() })
}
