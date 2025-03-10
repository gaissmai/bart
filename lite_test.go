// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"testing"
)

func TestLiteInsert(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		ins  []netip.Prefix
		del  []netip.Prefix
		ip   netip.Addr
		want bool
	}{
		{
			name: "invalid IP",
			ins:  []netip.Prefix{mpp("0.0.0.0/0"), mpp("::/0")},
			ip:   netip.Addr{},
			want: false,
		},
		{
			name: "zero",
			ip:   randomAddr(),
			want: false,
		},
		{
			name: "ins/del/zero",
			ins:  []netip.Prefix{mpp("0.0.0.0/0"), mpp("::/0")},
			del:  []netip.Prefix{mpp("0.0.0.0/0"), mpp("::/0")},
			ip:   randomAddr(),
			want: false,
		},
		{
			name: "default route",
			ins:  []netip.Prefix{mpp("0.0.0.0/0"), mpp("::/0")},
			ip:   randomAddr(),
			want: true,
		},
		{
			name: "indentity v4",
			ins:  []netip.Prefix{mpp("10.20.30.40/32")},
			ip:   mpa("10.20.30.40"),
			want: true,
		},
		{
			name: "indentity v6",
			ins:  []netip.Prefix{mpp("2001:db8::1/128")},
			ip:   mpa("2001:db8::1"),
			want: true,
		},
	}
	for _, tc := range testCases {
		lt := new(Lite)
		for _, p := range tc.ins {
			lt.Insert(p)
		}
		for _, p := range tc.del {
			lt.Delete(p)
		}
		got := lt.Contains(tc.ip)
		if got != tc.want {
			t.Errorf("%s: got: %v, want: %v", tc.name, got, tc.want)
		}
	}
}

func TestLiteInsertDelete(t *testing.T) {
	t.Parallel()

	lt := new(Lite)

	pfxs := randomRealWorldPrefixes(100_000)
	for _, pfx := range pfxs {
		lt.Insert(pfx)
	}
	// delete all prefixes
	for _, pfx := range pfxs {
		lt.Delete(pfx)
	}

	root4 := lt.rootNodeByVersion(true)
	if !root4.prefixes.IsEmpty() || root4.children.Len() != 0 {
		t.Errorf("Insert -> Delete not idempotent")
	}

	root6 := lt.rootNodeByVersion(false)
	if !root6.prefixes.IsEmpty() || root6.children.Len() != 0 {
		t.Errorf("Insert -> Delete not idempotent")
	}
}

func TestLiteContains(t *testing.T) {
	t.Parallel()

	const must = 1_000

	var match4, match6 int
	var miss4, miss6 int

	lt := new(Lite)
	tb := new(Table[any])

	for _, pfx := range randomRealWorldPrefixes(100_000) {
		lt.Insert(pfx)
		tb.Insert(pfx, nil)
	}

	for {
		ip := randomAddr()
		got1 := lt.Contains(ip)
		got2 := tb.Contains(ip)
		if got1 != got2 {
			t.Errorf("compare Contains(%q), Lite: %v, Table: %v", ip, got1, got2)
		}
		switch {
		case ip.Is4() && got1:
			match4++
		case ip.Is4() && !got1:
			miss4++
		case !ip.Is4() && got1:
			match6++
		case !ip.Is4() && !got1:
			miss6++
		default:
			panic("unreachable")
		}

		if match4 > must &&
			match6 > must &&
			miss4 > must &&
			miss6 > must {
			break
		}
	}
}

func TestFringeToCIDR(t *testing.T) {
	t.Parallel()
	var ip netip.Addr

	for range 10_000 {
		ip = randomIP4()
		for i := range 5 {
			pfx := netip.PrefixFrom(ip, i*8).Masked()
			octets := ip.AsSlice()
			path := stridePath{}
			copy(path[:], octets)

			got := cidrFromFringe(path, i, true)
			if pfx != got {
				t.Errorf("fringeToCidr: octets: %v, depth: %d, is4: %v, want: %s, got: %s", octets, i, true, pfx, got)
			}
		}

		ip = randomIP6()
		for i := range 17 {
			pfx := netip.PrefixFrom(ip, i*8).Masked()
			octets := ip.AsSlice()
			path := stridePath{}
			copy(path[:], octets)

			got := cidrFromFringe(path, i, false)
			if pfx != got {
				t.Errorf("fringeToCidr: octets: %v, depth: %d, is4: %v, want: %s, got: %s", octets, i, false, pfx, got)
			}
		}
	}
}
