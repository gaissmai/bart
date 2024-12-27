// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"strings"
	"testing"
)

type dumpTest struct {
	cidrs []netip.Prefix
	want  string
}

func TestDumperPanic(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Dump(nil) did not panic")
		}
	}()

	tbl := new(Table[any])
	tbl.Insert(mpp("1.2.3.4/32"), nil)
	tbl.dump(nil)
}

func TestDumperEmpty(t *testing.T) {
	t.Parallel()
	tbl := new(Table[any])
	checkDump(t, tbl, dumpTest{
		cidrs: nil,
		want: `### IPv4: size(0)
[NULL] depth:  0 path: [] / 0
### IPv6: size(0)
[NULL] depth:  0 path: [] / 0
`,
	})
}

func TestDumpDefaultRouteV4(t *testing.T) {
	t.Parallel()
	tbl := new(Table[any])
	checkDump(t, tbl, dumpTest{
		cidrs: []netip.Prefix{
			mpp("0.0.0.0/0"),
		},
		want: `### IPv4: size(1)
[LEAF] depth:  0 path: [] / 0
indexs(#1): [1]
prefxs(#1): 0/0
values(#1): <nil>
### IPv6: size(0)
[NULL] depth:  0 path: [] / 0
`,
	})
}

func TestDumpDefaultRouteV6(t *testing.T) {
	t.Parallel()
	tbl := new(Table[any])
	checkDump(t, tbl, dumpTest{
		cidrs: []netip.Prefix{
			mpp("::/0"),
		},
		want: `### IPv4: size(0)
[NULL] depth:  0 path: [] / 0
### IPv6: size(1)
[LEAF] depth:  0 path: [] / 0
indexs(#1): [1]
prefxs(#1): 0x00/0
values(#1): <nil>
`,
	})
}

func TestDumpSampleV4(t *testing.T) {
	t.Parallel()
	tbl := new(Table[any])
	tbl.WithPC()

	checkDump(t, tbl, dumpTest{
		cidrs: []netip.Prefix{
			mpp("172.16.0.0/12"),
			mpp("10.0.0.0/24"),
			mpp("192.168.0.0/16"),
			mpp("10.0.0.0/8"),
			mpp("10.0.1.0/24"),
			mpp("169.254.0.0/16"),
			mpp("127.0.0.0/8"),
			mpp("127.0.0.1/32"),
			mpp("192.168.1.0/24"),
		},
		want: `### IPv4: size(9)
[FULL] depth:  0 path: [] / 0
indexs(#2): [266 383]
prefxs(#2): 10/8 127/8
values(#2): <nil> <nil>
childs(#2): 10 192
pathcp(#3): 127:[127.0.0.1/32, <nil>] 169:[169.254.0.0/16, <nil>] 172:[172.16.0.0/12, <nil>]

.[IMED] depth:  1 path: [10] / 8
.childs(#1): 0

..[LEAF] depth:  2 path: [10.0] / 16
..indexs(#2): [256 257]
..prefxs(#2): 0/8 1/8
..values(#2): <nil> <nil>

.[LEAF] depth:  1 path: [192] / 8
.indexs(#1): [424]
.prefxs(#1): 168/8
.values(#1): <nil>
.pathcp(#1): 168:[192.168.1.0/24, <nil>]
### IPv6: size(0)
[NULL] depth:  0 path: [] / 0
`,
	})
}

func TestDumpSampleV6(t *testing.T) {
	t.Parallel()
	tbl := new(Table[any])
	tbl.WithPC()
	checkDump(t, tbl, dumpTest{
		cidrs: []netip.Prefix{
			mpp("fe80::/10"),
			mpp("2000::/3"),
			mpp("2001:db8::/32"),
		},
		want: `### IPv4: size(0)
[NULL] depth:  0 path: [] / 0
### IPv6: size(3)
[LEAF] depth:  0 path: [] / 0
indexs(#1): [9]
prefxs(#1): 0x20/3
values(#1): <nil>
pathcp(#2): 32:[2001:db8::/32, <nil>] 254:[fe80::/10, <nil>]
`,
	})
}

func TestDumpSample(t *testing.T) {
	t.Parallel()
	tbl := new(Table[any])
	tbl.WithPC()

	checkDump(t, tbl, dumpTest{
		cidrs: []netip.Prefix{
			mpp("fe80::/10"),
			mpp("172.16.0.0/12"),
			mpp("10.0.0.0/24"),
			mpp("192.168.0.0/16"),
			mpp("10.0.0.0/8"),
			mpp("::/0"),
			mpp("10.0.1.0/24"),
			mpp("169.254.0.0/16"),
			mpp("2000::/3"),
			mpp("2001:db8::/32"),
			mpp("127.0.0.0/8"),
			mpp("127.0.0.1/32"),
			mpp("192.168.1.0/24"),
		},
		want: `### IPv4: size(9)
[FULL] depth:  0 path: [] / 0
indexs(#2): [266 383]
prefxs(#2): 10/8 127/8
values(#2): <nil> <nil>
childs(#2): 10 192
pathcp(#3): 127:[127.0.0.1/32, <nil>] 169:[169.254.0.0/16, <nil>] 172:[172.16.0.0/12, <nil>]

.[IMED] depth:  1 path: [10] / 8
.childs(#1): 0

..[LEAF] depth:  2 path: [10.0] / 16
..indexs(#2): [256 257]
..prefxs(#2): 0/8 1/8
..values(#2): <nil> <nil>

.[LEAF] depth:  1 path: [192] / 8
.indexs(#1): [424]
.prefxs(#1): 168/8
.values(#1): <nil>
.pathcp(#1): 168:[192.168.1.0/24, <nil>]
### IPv6: size(4)
[LEAF] depth:  0 path: [] / 0
indexs(#2): [1 9]
prefxs(#2): 0x00/0 0x20/3
values(#2): <nil> <nil>
pathcp(#2): 32:[2001:db8::/32, <nil>] 254:[fe80::/10, <nil>]
`,
	})
}

func checkDump(t *testing.T, tbl *Table[any], tt dumpTest) {
	t.Helper()
	for _, cidr := range tt.cidrs {
		tbl.Insert(cidr, nil)
	}
	w := new(strings.Builder)
	tbl.dump(w)
	got := w.String()
	if tt.want != got {
		t.Errorf("Dump got:\n%swant:\n%s", got, tt.want)
	}
}
