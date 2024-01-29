package bart_test

import (
	"fmt"
	"net/netip"
	"os"

	"github.com/gaissmai/bart"
)

func mustAddr(s string) netip.Addr {
	return netip.MustParseAddr(s)
}

func mustPfx(s string) netip.Prefix {
	return netip.MustParsePrefix(s)
}

var input = []struct {
	cidr    netip.Prefix
	nextHop netip.Addr
}{
	{mustPfx("fe80::/10"), mustAddr("::1%lo")},
	{mustPfx("172.16.0.0/12"), mustAddr("8.8.8.8")},
	{mustPfx("10.0.0.0/24"), mustAddr("8.8.8.8")},
	{mustPfx("::1/128"), mustAddr("::1%eth0")},
	{mustPfx("192.168.0.0/16"), mustAddr("9.9.9.9")},
	{mustPfx("10.0.0.0/8"), mustAddr("9.9.9.9")},
	{mustPfx("::/0"), mustAddr("2001:db8::1")},
	{mustPfx("10.0.1.0/24"), mustAddr("10.0.0.0")},
	{mustPfx("169.254.0.0/16"), mustAddr("10.0.0.0")},
	{mustPfx("2000::/3"), mustAddr("2001:db8::1")},
	{mustPfx("2001:db8::/32"), mustAddr("2001:db8::1")},
	{mustPfx("127.0.0.0/8"), mustAddr("127.0.0.1")},
	{mustPfx("127.0.0.1/32"), mustAddr("127.0.0.1")},
	{mustPfx("192.168.1.0/24"), mustAddr("127.0.0.1")},
}

func ExampleTable_Lookup() {
	rtbl := new(bart.Table[netip.Addr])
	for _, item := range input {
		rtbl.Insert(item.cidr, item.nextHop)
	}
	rtbl.Fprint(os.Stdout)

	fmt.Println()

	ip := mustAddr("42.0.0.0")
	lpm, value, ok := rtbl.Lookup(ip)
	fmt.Printf("Lookup: %-20v lpm: %-15v value: %11v, ok: %v\n", ip, lpm, value, ok)

	ip = mustAddr("10.0.1.17")
	lpm, value, ok = rtbl.Lookup(ip)
	fmt.Printf("Lookup: %-20v lpm: %-15v value: %11v, ok: %v\n", ip, lpm, value, ok)

	ip = mustAddr("2001:7c0:3100:1::111")
	lpm, value, ok = rtbl.Lookup(ip)
	fmt.Printf("Lookup: %-20v lpm: %-15v value: %11v, ok: %v\n", ip, lpm, value, ok)

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
	//    ├─ ::1/128 (::1%eth0)
	//    ├─ 2000::/3 (2001:db8::1)
	//    │  └─ 2001:db8::/32 (2001:db8::1)
	//    └─ fe80::/10 (::1%lo)
	//
	// Lookup: 42.0.0.0             lpm: invalid Prefix  value:  invalid IP, ok: false
	// Lookup: 10.0.1.17            lpm: 10.0.1.0/24     value:    10.0.0.0, ok: true
	// Lookup: 2001:7c0:3100:1::111 lpm: 2000::/3        value: 2001:db8::1, ok: true
}
