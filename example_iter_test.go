//go:build go1.23

// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart_test

import (
	"fmt"
	"net/netip"

	"github.com/gaissmai/bart"
)

func ExampleTable_All4_rangeoverfunc() {
	rtbl := new(bart.Table[netip.Addr])
	for _, item := range input {
		rtbl.Insert(item.cidr, item.nextHop)
	}

	for pfx, val := range rtbl.All4Sorted {
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

	for pfx, _ := range rtbl.Subnets(cidr) {
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
	for pfx, _ := range rtbl.Supernets(cidr) {
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
