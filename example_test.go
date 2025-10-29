// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart_test

import (
	"fmt"
	"maps"
	"net/netip"
	"os"

	"github.com/gaissmai/bart"
)

var (
	mpa = netip.MustParseAddr
	mpp = netip.MustParsePrefix
)

var examplePrefixes = []netip.Prefix{
	mpp("192.168.0.0/16"),
	mpp("192.168.1.0/24"),
	mpp("2001:7c0:3100::/40"),
	mpp("2001:7c0:3100:1::/64"),
	mpp("fc00::/7"),
}

// some example IP addresses for black/whitelist containment
var exampleIPs = []netip.Addr{
	mpa("192.168.1.100"),      // must match
	mpa("192.168.2.1"),        // must match
	mpa("2001:7c0:3100:1::1"), // must match
	mpa("2001:7c0:3100:2::1"), // must match
	mpa("fc00::1"),            // must match
	//
	mpa("172.16.0.1"),        // must NOT match
	mpa("2003:dead:beef::1"), // must NOT match
}

func ExampleLite_contains() {
	lite := new(bart.Lite)

	for _, pfx := range examplePrefixes {
		lite.Insert(pfx)
	}

	for _, ip := range exampleIPs {
		ok := lite.Contains(ip)
		fmt.Printf("%-20s is contained: %t\n", ip, ok)
	}

	// Output:
	// 192.168.1.100        is contained: true
	// 192.168.2.1          is contained: true
	// 2001:7c0:3100:1::1   is contained: true
	// 2001:7c0:3100:2::1   is contained: true
	// fc00::1              is contained: true
	// 172.16.0.1           is contained: false
	// 2003:dead:beef::1    is contained: false
}

var input = []struct {
	cidr    netip.Prefix
	nextHop netip.Addr
}{
	{mpp("fe80::/10"), mpa("::1%eth0")},
	{mpp("172.16.0.0/12"), mpa("8.8.8.8")},
	{mpp("10.0.0.0/24"), mpa("8.8.8.8")},
	{mpp("::1/128"), mpa("::1%lo")},
	{mpp("192.168.0.0/16"), mpa("9.9.9.9")},
	{mpp("10.0.0.0/8"), mpa("9.9.9.9")},
	{mpp("::/0"), mpa("2001:db8::1")},
	{mpp("10.0.1.0/24"), mpa("10.0.0.0")},
	{mpp("169.254.0.0/16"), mpa("10.0.0.0")},
	{mpp("2000::/3"), mpa("2000::")},
	{mpp("2001:db8::/32"), mpa("2001:db8::1")},
	{mpp("127.0.0.0/8"), mpa("127.0.0.1")},
	{mpp("127.0.0.1/32"), mpa("127.0.0.1")},
	{mpp("192.168.1.0/24"), mpa("127.0.0.1")},
}

func ExampleTable_Lookup() {
	rtbl := new(bart.Table[netip.Addr])
	for _, item := range input {
		rtbl.Insert(item.cidr, item.nextHop)
	}
	rtbl.Fprint(os.Stdout)

	fmt.Println()

	ip := mpa("42.0.0.0")
	value, ok := rtbl.Lookup(ip)
	fmt.Printf("Lookup: %-20v next-hop: %11v, ok: %v\n", ip, value, ok)

	ip = mpa("10.0.1.17")
	value, ok = rtbl.Lookup(ip)
	fmt.Printf("Lookup: %-20v next-hop: %11v, ok: %v\n", ip, value, ok)

	ip = mpa("2001:7c0:3100:1::111")
	value, ok = rtbl.Lookup(ip)
	fmt.Printf("Lookup: %-20v next-hop: %11v, ok: %v\n", ip, value, ok)

	// Output:
	// ▼
	// ├─ 10.0.0.0/8 (9.9.9.9)
	// │  ├─ 10.0.0.0/24 (8.8.8.8)
	// │  └─ 10.0.1.0/24 (10.0.0.0)
	// ├─ 127.0.0.0/8 (127.0.0.1)
	// │  └─ 127.0.0.1/32 (127.0.0.1)
	// ├─ 169.254.0.0/16 (10.0.0.0)
	// ├─ 172.16.0.0/12 (8.8.8.8)
	// └─ 192.168.0.0/16 (9.9.9.9)
	//    └─ 192.168.1.0/24 (127.0.0.1)
	// ▼
	// └─ ::/0 (2001:db8::1)
	//    ├─ ::1/128 (::1%lo)
	//    ├─ 2000::/3 (2000::)
	//    │  └─ 2001:db8::/32 (2001:db8::1)
	//    └─ fe80::/10 (::1%eth0)
	//
	// Lookup: 42.0.0.0             next-hop:  invalid IP, ok: false
	// Lookup: 10.0.1.17            next-hop:    10.0.0.0, ok: true
	// Lookup: 2001:7c0:3100:1::111 next-hop:      2000::, ok: true
}

func ExampleTable_AllSorted4_rangeoverfunc() {
	rtbl := new(bart.Table[netip.Addr])
	for _, item := range input {
		rtbl.Insert(item.cidr, item.nextHop)
	}

	for pfx, val := range rtbl.AllSorted4() {
		fmt.Printf("%v\t%v\n", pfx, val)
	}

	// Output:
	// 10.0.0.0/8	9.9.9.9
	// 10.0.0.0/24	8.8.8.8
	// 10.0.1.0/24	10.0.0.0
	// 127.0.0.0/8	127.0.0.1
	// 127.0.0.1/32	127.0.0.1
	// 169.254.0.0/16	10.0.0.0
	// 172.16.0.0/12	8.8.8.8
	// 192.168.0.0/16	9.9.9.9
	// 192.168.1.0/24	127.0.0.1
}

func ExampleTable_Subnets_rangeoverfunc() {
	rtbl := new(bart.Table[netip.Addr])
	for _, item := range input {
		rtbl.Insert(item.cidr, item.nextHop)
	}

	cidr := netip.MustParsePrefix("0.0.0.0/1")

	for pfx := range rtbl.Subnets(cidr) {
		fmt.Printf("%v\n", pfx)
	}

	// Output:
	// 10.0.0.0/8
	// 10.0.0.0/24
	// 10.0.1.0/24
	// 127.0.0.0/8
	// 127.0.0.1/32
}

func ExampleTable_Supernets_rangeoverfunc() {
	rtbl := new(bart.Table[netip.Addr])
	for _, item := range input {
		rtbl.Insert(item.cidr, item.nextHop)
	}

	cidr := netip.MustParsePrefix("2001:db8::/32")

	counter := 0
	for pfx := range rtbl.Supernets(cidr) {
		fmt.Printf("%v\n", pfx)
		counter++
		if counter >= 2 {
			break
		}
	}

	// Output:
	// 2001:db8::/32
	// 2000::/3
}

type route struct {
	ASN   int
	Attrs map[string]string
}

func (r route) Equal(other route) bool {
	return r.ASN == other.ASN && maps.Equal(r.Attrs, other.Attrs)
}

func (r route) Clone() route {
	return route{
		ASN:   r.ASN,
		Attrs: maps.Clone(r.Attrs),
	}
}

// Example of a custom value type with both equality and cloning
func ExampleTable_customValue() {
	table := new(bart.Table[route])
	table = table.InsertPersist(mpp("10.0.0.0/8"), route{ASN: 64512, Attrs: map[string]string{"foo": "bar"}})
	clone := table.Clone()
	fmt.Printf("Cloned tables are equal: %v\n", table.Equal(clone))

	// Output:
	// Cloned tables are equal: true
}
