// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"math/rand/v2"
	"net/netip"
	"testing"
)

// ---------- helpers ----------
// isSubnetOf reports whether p is fully contained in q (same address family).
// Logic:
// - families must match (both v4 or both v6)
// - a subnet must have a prefix length >= its supernet
// - compare the network part of p at q's length against q's masked network
func isSubnetOf(p, q netip.Prefix) bool {
	if p.Addr().Is4() != q.Addr().Is4() {
		return false
	}
	if p.Bits() < q.Bits() {
		return false
	}
	return netip.PrefixFrom(p.Addr(), q.Bits()).Masked() == q.Masked()
}

// isSupernetOf reports whether r is a supernet of p (same address family).
// Logic is the inverse direction:
// - families must match
// - a supernet must have a prefix length <= that of the subnet
// - compare the network part of p at r's length against r's masked network
func isSupernetOf(r, p netip.Prefix) bool {
	if r.Addr().Is4() != p.Addr().Is4() {
		return false
	}
	if r.Bits() > p.Bits() {
		return false
	}
	return netip.PrefixFrom(p.Addr(), r.Bits()).Masked() == r.Masked()
}

func verifySortedCIDR(t *testing.T, list []netip.Prefix) {
	t.Helper()
	for i := 1; i < len(list); i++ {
		if cmpPrefix(list[i-1], list[i]) > 0 {
			t.Fatalf("order violation at %d: %v > %v", i-1, list[i-1], list[i])
		}
	}
}

// ---------- All4 / All6 (and Sorted) ----------

func TestAll4All6_Table(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 1))
	pfxs := randomPrefixes(prng, n)

	tbl := new(Table[int])
	var n4, n6 int
	for i, it := range pfxs {
		tbl.Insert(it.pfx, i)
		if it.pfx.Addr().Is4() {
			n4++
		} else {
			n6++
		}
	}

	seen4 := map[netip.Prefix]bool{}
	seen6 := map[netip.Prefix]bool{}
	got4, got6 := 0, 0

	for p := range tbl.All4() {
		if !p.Addr().Is4() {
			t.Fatalf("All4 yielded non-IPv4: %v", p)
		}
		if seen4[p] {
			t.Fatalf("duplicate in All4: %v", p)
		}
		seen4[p] = true
		got4++
	}
	for p := range tbl.All6() {
		if p.Addr().Is4() {
			t.Fatalf("All6 yielded IPv4: %v", p)
		}
		if seen6[p] {
			t.Fatalf("duplicate in All6: %v", p)
		}
		seen6[p] = true
		got6++
	}

	if got4 != n4 || got6 != n6 {
		t.Fatalf("mismatch counts: want4=%d got4=%d, want6=%d got6=%d", n4, got4, n6, got6)
	}
}

func TestAll4All6_Fast(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	prng := rand.New(rand.NewPCG(43, 1))
	pfxs := randomPrefixes(prng, n)

	f := new(Fast[int])
	var n4, n6 int
	for i, it := range pfxs {
		f.Insert(it.pfx, i)
		if it.pfx.Addr().Is4() {
			n4++
		} else {
			n6++
		}
	}

	seen4 := map[netip.Prefix]bool{}
	seen6 := map[netip.Prefix]bool{}
	got4, got6 := 0, 0

	for p := range f.All4() {
		if !p.Addr().Is4() {
			t.Fatalf("All4 yielded non-IPv4: %v", p)
		}
		if seen4[p] {
			t.Fatalf("duplicate in All4: %v", p)
		}
		seen4[p] = true
		got4++
	}
	for p := range f.All6() {
		if p.Addr().Is4() {
			t.Fatalf("All6 yielded IPv4: %v", p)
		}
		if seen6[p] {
			t.Fatalf("duplicate in All6: %v", p)
		}
		seen6[p] = true
		got6++
	}
	if got4 != n4 || got6 != n6 {
		t.Fatalf("mismatch counts: want4=%d got4=%d, want6=%d got6=%d", n4, got4, n6, got6)
	}
}

func TestAll4All6_liteTable(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	prng := rand.New(rand.NewPCG(44, 1))
	pfxs := randomPrefixes(prng, n)

	lt := new(liteTable[int])
	var n4, n6 int
	for i, it := range pfxs {
		lt.Insert(it.pfx, i)
		if it.pfx.Addr().Is4() {
			n4++
		} else {
			n6++
		}
	}

	seen4 := map[netip.Prefix]bool{}
	seen6 := map[netip.Prefix]bool{}
	got4, got6 := 0, 0

	for p := range lt.All4() {
		if !p.Addr().Is4() {
			t.Fatalf("All4 yielded non-IPv4: %v", p)
		}
		if seen4[p] {
			t.Fatalf("duplicate in All4: %v", p)
		}
		seen4[p] = true
		got4++
	}
	for p := range lt.All6() {
		if p.Addr().Is4() {
			t.Fatalf("All6 yielded IPv4: %v", p)
		}
		if seen6[p] {
			t.Fatalf("duplicate in All6: %v", p)
		}
		seen6[p] = true
		got6++
	}
	if got4 != n4 || got6 != n6 {
		t.Fatalf("mismatch counts: want4=%d got4=%d, want6=%d got6=%d", n4, got4, n6, got6)
	}
}

func TestAll4All6_Lite(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	prng := rand.New(rand.NewPCG(45, 1))
	pfxs := randomPrefixes(prng, n)

	l := new(Lite)
	var n4, n6 int
	for _, it := range pfxs {
		l.Insert(it.pfx)
		if it.pfx.Addr().Is4() {
			n4++
		} else {
			n6++
		}
	}

	seen4 := map[netip.Prefix]bool{}
	seen6 := map[netip.Prefix]bool{}
	got4, got6 := 0, 0

	for p := range l.All4() {
		if !p.Addr().Is4() {
			t.Fatalf("All4 yielded non-IPv4: %v", p)
		}
		if seen4[p] {
			t.Fatalf("duplicate in All4: %v", p)
		}
		seen4[p] = true
		got4++
	}
	for p := range l.All6() {
		if p.Addr().Is4() {
			t.Fatalf("All6 yielded IPv4: %v", p)
		}
		if seen6[p] {
			t.Fatalf("duplicate in All6: %v", p)
		}
		seen6[p] = true
		got6++
	}
	if got4 != n4 || got6 != n6 {
		t.Fatalf("mismatch counts: want4=%d got4=%d, want6=%d got6=%d", n4, got4, n6, got6)
	}
}

// Sorted variants: ensure CIDR order within each family.

func TestAllSorted4AllSorted6_Table(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	prng := rand.New(rand.NewPCG(46, 1))
	pfxs := randomPrefixes(prng, n)

	tbl := new(Table[int])
	for i, it := range pfxs {
		tbl.Insert(it.pfx, i)
	}

	var s4, s6 []netip.Prefix
	for p := range tbl.AllSorted4() {
		s4 = append(s4, p)
	}
	for p := range tbl.AllSorted6() {
		s6 = append(s6, p)
	}
	verifySortedCIDR(t, s4)
	verifySortedCIDR(t, s6)
}

func TestAllSorted4AllSorted6_Fast(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	prng := rand.New(rand.NewPCG(47, 1))
	pfxs := randomPrefixes(prng, n)

	f := new(Fast[int])
	for i, it := range pfxs {
		f.Insert(it.pfx, i)
	}

	var s4, s6 []netip.Prefix
	for p := range f.AllSorted4() {
		s4 = append(s4, p)
	}
	for p := range f.AllSorted6() {
		s6 = append(s6, p)
	}
	verifySortedCIDR(t, s4)
	verifySortedCIDR(t, s6)
}

func TestAllSorted4AllSorted6_liteTable(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	prng := rand.New(rand.NewPCG(48, 1))
	pfxs := randomPrefixes(prng, n)

	lt := new(liteTable[int])
	for i, it := range pfxs {
		lt.Insert(it.pfx, i)
	}

	var s4, s6 []netip.Prefix
	for p := range lt.AllSorted4() {
		s4 = append(s4, p)
	}
	for p := range lt.AllSorted6() {
		s6 = append(s6, p)
	}
	verifySortedCIDR(t, s4)
	verifySortedCIDR(t, s6)
}

func TestAllSorted4AllSorted6_Lite(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	prng := rand.New(rand.NewPCG(49, 1))
	pfxs := randomPrefixes(prng, n)

	l := new(Lite)
	for _, it := range pfxs {
		l.Insert(it.pfx)
	}

	var s4, s6 []netip.Prefix
	for p := range l.AllSorted4() {
		s4 = append(s4, p)
	}
	for p := range l.AllSorted6() {
		s6 = append(s6, p)
	}
	verifySortedCIDR(t, s4)
	verifySortedCIDR(t, s6)
}

// ---------- Fuzz: Subnets ----------

//nolint:gocyclo
func FuzzSubnets(f *testing.F) {
	// seed corpus
	f.Add(uint64(12345), 150, 30)
	f.Add(uint64(67890), 400, 60)
	f.Add(uint64(54321), 800, 100)

	f.Fuzz(func(t *testing.T, seed uint64, n, nq int) {
		if n < 10 || n > 5000 || nq < 1 || nq > 200 {
			t.Skip("bounds")
		}

		prng := rand.New(rand.NewPCG(seed, 13))
		pfxs := randomPrefixes(prng, n)
		queries := randomPrefixes(prng, nq) // we’ll use their pfx fields as queries

		// Table
		{
			tbl := new(Table[int])
			for i, it := range pfxs {
				tbl.Insert(it.pfx, i)
			}
			for _, q := range queries {
				want := map[netip.Prefix]bool{}
				for _, it := range pfxs {
					if isSubnetOf(it.pfx, q.pfx) {
						want[it.pfx] = true
					}
				}
				got := map[netip.Prefix]bool{}
				for p := range tbl.Subnets(q.pfx) {
					if got[p] {
						t.Fatalf("Table.Subnets duplicate: %v", p)
					}
					got[p] = true
				}
				if len(got) != len(want) {
					t.Fatalf("Table.Subnets size mismatch for %v: want %d got %d",
						q.pfx, len(want), len(got))
				}
				for p := range want {
					if !got[p] {
						t.Fatalf("Table.Subnets missing %v for %v", p, q.pfx)
					}
				}
			}
		}

		// Fast
		{
			ft := new(Fast[int])
			for i, it := range pfxs {
				ft.Insert(it.pfx, i)
			}
			for _, q := range queries {
				want := map[netip.Prefix]bool{}
				for _, it := range pfxs {
					if isSubnetOf(it.pfx, q.pfx) {
						want[it.pfx] = true
					}
				}
				got := map[netip.Prefix]bool{}
				for p := range ft.Subnets(q.pfx) {
					if got[p] {
						t.Fatalf("Fast.Subnets duplicate: %v", p)
					}
					got[p] = true
				}
				if len(got) != len(want) {
					t.Fatalf("Fast.Subnets size mismatch for %v: want %d got %d",
						q.pfx, len(want), len(got))
				}
				for p := range want {
					if !got[p] {
						t.Fatalf("Fast.Subnets missing %v for %v", p, q.pfx)
					}
				}
			}
		}

		// liteTable
		{
			lt := new(liteTable[int])
			for i, it := range pfxs {
				lt.Insert(it.pfx, i)
			}
			for _, q := range queries {
				want := map[netip.Prefix]bool{}
				for _, it := range pfxs {
					if isSubnetOf(it.pfx, q.pfx) {
						want[it.pfx] = true
					}
				}
				got := map[netip.Prefix]bool{}
				for p := range lt.Subnets(q.pfx) {
					if got[p] {
						t.Fatalf("liteTable.Subnets duplicate: %v", p)
					}
					got[p] = true
				}
				if len(got) != len(want) {
					t.Fatalf("liteTable.Subnets size mismatch for %v: want %d got %d",
						q.pfx, len(want), len(got))
				}
				for p := range want {
					if !got[p] {
						t.Fatalf("liteTable.Subnets missing %v for %v", p, q.pfx)
					}
				}
			}
		}

		// Lite (prefix-only)
		{
			l := new(Lite)
			for _, it := range pfxs {
				l.Insert(it.pfx)
			}
			for _, q := range queries {
				want := map[netip.Prefix]bool{}
				for _, it := range pfxs {
					if isSubnetOf(it.pfx, q.pfx) {
						want[it.pfx] = true
					}
				}
				got := map[netip.Prefix]bool{}
				for p := range l.Subnets(q.pfx) {
					if got[p] {
						t.Fatalf("Lite.Subnets duplicate: %v", p)
					}
					got[p] = true
				}
				if len(got) != len(want) {
					t.Fatalf("Lite.Subnets size mismatch for %v: want %d got %d",
						q.pfx, len(want), len(got))
				}
				for p := range want {
					if !got[p] {
						t.Fatalf("Lite.Subnets missing %v for %v", p, q.pfx)
					}
				}
			}
		}
	})
}

// ---------- Fuzz: Supernets ----------

//nolint:gocyclo
func FuzzSupernets(f *testing.F) {
	// seed corpus
	f.Add(uint64(222), 150, 30)
	f.Add(uint64(333), 400, 60)
	f.Add(uint64(444), 800, 100)

	f.Fuzz(func(t *testing.T, seed uint64, n, nq int) {
		if n < 10 || n > 5000 || nq < 1 || nq > 200 {
			t.Skip("bounds")
		}

		prng := rand.New(rand.NewPCG(seed, 17))
		pfxs := randomPrefixes(prng, n)
		queries := randomPrefixes(prng, nq)

		// Table
		{
			tbl := new(Table[int])
			for i, it := range pfxs {
				tbl.Insert(it.pfx, i)
			}
			for _, q := range queries {
				want := map[netip.Prefix]bool{}
				for _, it := range pfxs {
					if isSupernetOf(it.pfx, q.pfx) {
						want[it.pfx] = true
					}
				}
				got := map[netip.Prefix]bool{}
				for p := range tbl.Supernets(q.pfx) {
					if got[p] {
						t.Fatalf("Table.Supernets duplicate: %v", p)
					}
					got[p] = true
				}
				if len(got) != len(want) {
					t.Fatalf("Table.Supernets size mismatch for %v: want %d got %d",
						q.pfx, len(want), len(got))
				}
				for p := range want {
					if !got[p] {
						t.Fatalf("Table.Supernets missing %v for %v", p, q.pfx)
					}
				}
			}
		}

		// Fast
		{
			ft := new(Fast[int])
			for i, it := range pfxs {
				ft.Insert(it.pfx, i)
			}
			for _, q := range queries {
				want := map[netip.Prefix]bool{}
				for _, it := range pfxs {
					if isSupernetOf(it.pfx, q.pfx) {
						want[it.pfx] = true
					}
				}
				got := map[netip.Prefix]bool{}
				for p := range ft.Supernets(q.pfx) {
					if got[p] {
						t.Fatalf("Fast.Supernets duplicate: %v", p)
					}
					got[p] = true
				}
				if len(got) != len(want) {
					t.Fatalf("Fast.Supernets size mismatch for %v: want %d got %d",
						q.pfx, len(want), len(got))
				}
				for p := range want {
					if !got[p] {
						t.Fatalf("Fast.Supernets missing %v for %v", p, q.pfx)
					}
				}
			}
		}

		// liteTable
		{
			lt := new(liteTable[int])
			for i, it := range pfxs {
				lt.Insert(it.pfx, i)
			}
			for _, q := range queries {
				want := map[netip.Prefix]bool{}
				for _, it := range pfxs {
					if isSupernetOf(it.pfx, q.pfx) {
						want[it.pfx] = true
					}
				}
				got := map[netip.Prefix]bool{}
				for p := range lt.Supernets(q.pfx) {
					if got[p] {
						t.Fatalf("liteTable.Supernets duplicate: %v", p)
					}
					got[p] = true
				}
				if len(got) != len(want) {
					t.Fatalf("liteTable.Supernets size mismatch for %v: want %d got %d",
						q.pfx, len(want), len(got))
				}
				for p := range want {
					if !got[p] {
						t.Fatalf("liteTable.Supernets missing %v for %v", p, q.pfx)
					}
				}
			}
		}

		// Lite (prefix-only)
		{
			l := new(Lite)
			for _, it := range pfxs {
				l.Insert(it.pfx)
			}
			for _, q := range queries {
				want := map[netip.Prefix]bool{}
				for _, it := range pfxs {
					if isSupernetOf(it.pfx, q.pfx) {
						want[it.pfx] = true
					}
				}
				got := map[netip.Prefix]bool{}
				for p := range l.Supernets(q.pfx) {
					if got[p] {
						t.Fatalf("Lite.Supernets duplicate: %v", p)
					}
					got[p] = true
				}
				if len(got) != len(want) {
					t.Fatalf("Lite.Supernets size mismatch for %v: want %d got %d",
						q.pfx, len(want), len(got))
				}
				for p := range want {
					if !got[p] {
						t.Fatalf("Lite.Supernets missing %v for %v", p, q.pfx)
					}
				}
			}
		}
	})
}
