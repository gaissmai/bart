// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart_test

import (
	"fmt"
	"net/netip"
	"os"

	"github.com/gaissmai/bart"
)

func ExampleLite_Contains() {
	lite := new(bart.Lite)

	// Insert some prefixes
	prefixes := []string{
		"192.168.0.0/16",       // corporate
		"192.168.1.0/24",       // department
		"2001:7c0:3100::/40",   // corporate
		"2001:7c0:3100:1::/64", // department
		"fc00::/7",             // unique local
	}

	for _, s := range prefixes {
		pfx := netip.MustParsePrefix(s)
		lite.Insert(pfx)
	}

	// Test some IP addresses for black/whitelist containment
	ips := []string{
		"192.168.1.100",      // must match, department
		"192.168.2.1",        // must match, corporate
		"2001:7c0:3100:1::1", // must match, department
		"2001:7c0:3100:2::1", // must match, corporate
		"fc00::1",            // must match, unique local
		//
		"172.16.0.1",        // must NOT match
		"2003:dead:beef::1", // must NOT match
	}

	for _, s := range ips {
		ip := netip.MustParseAddr(s)
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

var (
	a = netip.MustParseAddr
	p = netip.MustParsePrefix
)

var input = []struct {
	cidr    netip.Prefix
	nextHop netip.Addr
}{
	{p("fe80::/10"), a("::1%eth0")},
	{p("172.16.0.0/12"), a("8.8.8.8")},
	{p("10.0.0.0/24"), a("8.8.8.8")},
	{p("::1/128"), a("::1%lo")},
	{p("192.168.0.0/16"), a("9.9.9.9")},
	{p("10.0.0.0/8"), a("9.9.9.9")},
	{p("::/0"), a("2001:db8::1")},
	{p("10.0.1.0/24"), a("10.0.0.0")},
	{p("169.254.0.0/16"), a("10.0.0.0")},
	{p("2000::/3"), a("2000::")},
	{p("2001:db8::/32"), a("2001:db8::1")},
	{p("127.0.0.0/8"), a("127.0.0.1")},
	{p("127.0.0.1/32"), a("127.0.0.1")},
	{p("192.168.1.0/24"), a("127.0.0.1")},
}

func ExampleTable_Lookup() {
	rtbl := new(bart.Table[netip.Addr])
	for _, item := range input {
		rtbl.Insert(item.cidr, item.nextHop)
	}
	rtbl.Fprint(os.Stdout)

	fmt.Println()

	ip := a("42.0.0.0")
	value, ok := rtbl.Lookup(ip)
	fmt.Printf("Lookup: %-20v next-hop: %11v, ok: %v\n", ip, value, ok)

	ip = a("10.0.1.17")
	value, ok = rtbl.Lookup(ip)
	fmt.Printf("Lookup: %-20v next-hop: %11v, ok: %v\n", ip, value, ok)

	ip = a("2001:7c0:3100:1::111")
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
