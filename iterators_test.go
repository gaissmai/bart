// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"math/rand/v2"
	"net/netip"
	"testing"
)

// verifySortedCIDR asserts the slice is sorted in natural CIDR order.
func verifySortedCIDR(t *testing.T, list []netip.Prefix) {
	t.Helper()
	for i := 1; i < len(list); i++ {
		if cmpPrefix(list[i-1], list[i]) > 0 {
			t.Fatalf("order violation at %d: %v > %v", i-1, list[i-1], list[i])
		}
	}
}

// isSubnetOf reports whether p is fully contained in q (same address family).
// Families must match; a subnet must have a prefix length >= its supernet.
// Compares the network of p at q's length with q's canonical (Masked) network.
// Note: use netip.PrefixFrom(addr,bits) (single return) to avoid multi-value errors.
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
// Families must match; a supernet must have a prefix length <= the subnet.
// Compares the network of p at r's length with r's canonical (Masked) network.
func isSupernetOf(r, p netip.Prefix) bool {
	if r.Addr().Is4() != p.Addr().Is4() {
		return false
	}
	if r.Bits() > p.Bits() {
		return false
	}
	return netip.PrefixFrom(p.Addr(), r.Bits()).Masked() == r.Masked()
}

// ---------------- All4 / All6 (counts and family checks) ----------------

func TestAll4All6_Table(t *testing.T) {
	t.Parallel()

	prng := rand.New(rand.NewPCG(42, 1))
	pfxs := randomPrefixes(prng, 500)

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
	got4 := 0
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
	got6 := 0
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

	prng := rand.New(rand.NewPCG(43, 1))
	pfxs := randomPrefixes(prng, 500)

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
	got4 := 0
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
	got6 := 0
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

	prng := rand.New(rand.NewPCG(44, 1))
	pfxs := randomPrefixes(prng, 500)

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
	got4 := 0
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
	got6 := 0
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

	prng := rand.New(rand.NewPCG(45, 1))
	pfxs := randomPrefixes(prng, 500)

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
	got4 := 0
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
	got6 := 0
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

// ---------------- AllSorted4 / AllSorted6 (order checks) ----------------

func TestAllSorted4AllSorted6_Table(t *testing.T) {
	t.Parallel()

	prng := rand.New(rand.NewPCG(46, 1))
	pfxs := randomPrefixes(prng, 400)

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

	prng := rand.New(rand.NewPCG(47, 1))
	pfxs := randomPrefixes(prng, 400)

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

	prng := rand.New(rand.NewPCG(48, 1))
	pfxs := randomPrefixes(prng, 400)

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

	prng := rand.New(rand.NewPCG(49, 1))
	pfxs := randomPrefixes(prng, 400)

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
