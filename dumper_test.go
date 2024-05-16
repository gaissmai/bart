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
		cidrs: []netip.Prefix{},
		want: `### IPv4:
[NULL] depth:  0 path: [] / 0
### IPv6:
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
		want: `### IPv4:
[LEAF] depth:  0 path: [] / 0
indexs(#1): [1]
prefxs(#1): 0/0
values(#1): <nil>
### IPv6:
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
		want: `### IPv4:
[NULL] depth:  0 path: [] / 0
### IPv6:
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
		want: `### IPv4:
[FULL] depth:  0 path: [] / 0
indexs(#2): [266 383]
prefxs(#2): 10/8 127/8
values(#2): <nil> <nil>
childs(#5): 10 127 169 172 192

.[IMED] depth:  1 path: [10] / 8
.childs(#1): 0

..[LEAF] depth:  2 path: [10.0] / 16
..indexs(#2): [256 257]
..prefxs(#2): 0/8 1/8
..values(#2): <nil> <nil>

.[IMED] depth:  1 path: [127] / 8
.childs(#1): 0

..[IMED] depth:  2 path: [127.0] / 16
..childs(#1): 0

...[LEAF] depth:  3 path: [127.0.0] / 24
...indexs(#1): [257]
...prefxs(#1): 1/8
...values(#1): <nil>

.[LEAF] depth:  1 path: [169] / 8
.indexs(#1): [510]
.prefxs(#1): 254/8
.values(#1): <nil>

.[LEAF] depth:  1 path: [172] / 8
.indexs(#1): [17]
.prefxs(#1): 16/4
.values(#1): <nil>

.[FULL] depth:  1 path: [192] / 8
.indexs(#1): [424]
.prefxs(#1): 168/8
.values(#1): <nil>
.childs(#1): 168

..[LEAF] depth:  2 path: [192.168] / 16
..indexs(#1): [257]
..prefxs(#1): 1/8
..values(#1): <nil>
### IPv6:
[NULL] depth:  0 path: [] / 0
`,
	})
}

func TestDumpSampleV6(t *testing.T) {
	t.Parallel()
	tbl := new(Table[any])
	checkDump(t, tbl, dumpTest{
		cidrs: []netip.Prefix{
			mpp("fe80::/10"),
			mpp("2000::/3"),
			mpp("2001:db8::/32"),
		},
		want: `### IPv4:
[NULL] depth:  0 path: [] / 0
### IPv6:
[FULL] depth:  0 path: [] / 0
indexs(#1): [9]
prefxs(#1): 0x20/3
values(#1): <nil>
childs(#2): 0x20 0xfe

.[IMED] depth:  1 path: [20] / 8
.childs(#1): 0x01

..[IMED] depth:  2 path: [2001] / 16
..childs(#1): 0x0d

...[LEAF] depth:  3 path: [2001:0d] / 24
...indexs(#1): [440]
...prefxs(#1): 0xb8/8
...values(#1): <nil>

.[LEAF] depth:  1 path: [fe] / 8
.indexs(#1): [6]
.prefxs(#1): 0x80/2
.values(#1): <nil>
`,
	})
}

func TestDumpSample(t *testing.T) {
	t.Parallel()
	tbl := new(Table[any])
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
		want: `### IPv4:
[FULL] depth:  0 path: [] / 0
indexs(#2): [266 383]
prefxs(#2): 10/8 127/8
values(#2): <nil> <nil>
childs(#5): 10 127 169 172 192

.[IMED] depth:  1 path: [10] / 8
.childs(#1): 0

..[LEAF] depth:  2 path: [10.0] / 16
..indexs(#2): [256 257]
..prefxs(#2): 0/8 1/8
..values(#2): <nil> <nil>

.[IMED] depth:  1 path: [127] / 8
.childs(#1): 0

..[IMED] depth:  2 path: [127.0] / 16
..childs(#1): 0

...[LEAF] depth:  3 path: [127.0.0] / 24
...indexs(#1): [257]
...prefxs(#1): 1/8
...values(#1): <nil>

.[LEAF] depth:  1 path: [169] / 8
.indexs(#1): [510]
.prefxs(#1): 254/8
.values(#1): <nil>

.[LEAF] depth:  1 path: [172] / 8
.indexs(#1): [17]
.prefxs(#1): 16/4
.values(#1): <nil>

.[FULL] depth:  1 path: [192] / 8
.indexs(#1): [424]
.prefxs(#1): 168/8
.values(#1): <nil>
.childs(#1): 168

..[LEAF] depth:  2 path: [192.168] / 16
..indexs(#1): [257]
..prefxs(#1): 1/8
..values(#1): <nil>
### IPv6:
[FULL] depth:  0 path: [] / 0
indexs(#2): [1 9]
prefxs(#2): 0x00/0 0x20/3
values(#2): <nil> <nil>
childs(#2): 0x20 0xfe

.[IMED] depth:  1 path: [20] / 8
.childs(#1): 0x01

..[IMED] depth:  2 path: [2001] / 16
..childs(#1): 0x0d

...[LEAF] depth:  3 path: [2001:0d] / 24
...indexs(#1): [440]
...prefxs(#1): 0xb8/8
...values(#1): <nil>

.[LEAF] depth:  1 path: [fe] / 8
.indexs(#1): [6]
.prefxs(#1): 0x80/2
.values(#1): <nil>
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
