// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"strings"
	"testing"
)

func TestDumperPanic2(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Dump(nil) did not panic")
		}
	}()

	tbl := new(Table2[any])
	tbl.Insert(mpp("1.2.3.4/32"), nil)
	tbl.dump(nil)
}

func TestDumperEmpty2(t *testing.T) {
	t.Parallel()
	tbl := new(Table2[any])
	checkDump2(t, tbl, dumpTest{
		cidrs: []netip.Prefix{},
		want: `### IPv4:
[NULL] path: [] bits: +0 depth: 0
### IPv6:
[NULL] path: [] bits: +0 depth: 0
`,
	})
}

func TestDumpDefaultV4Route2(t *testing.T) {
	t.Parallel()
	tbl := new(Table2[any])
	checkDump2(t, tbl, dumpTest{
		cidrs: []netip.Prefix{
			mpp("0.0.0.0/0"),
		},
		want: `### IPv4:
[LEAF] path: [] bits: +0 depth: 0
indexs(#1): [1]
prefxs(#1): 0/0
values(#1): <nil>
### IPv6:
[NULL] path: [] bits: +0 depth: 0
`,
	})
}

func TestDumpDefaultV6Route2(t *testing.T) {
	t.Parallel()
	tbl := new(Table2[any])
	checkDump2(t, tbl, dumpTest{
		cidrs: []netip.Prefix{
			mpp("::/0"),
		},
		want: `### IPv4:
[NULL] path: [] bits: +0 depth: 0
### IPv6:
[LEAF] path: [] bits: +0 depth: 0
indexs(#1): [1]
prefxs(#1): 0x00/0
values(#1): <nil>
`,
	})
}

func TestDumpV4Sample2(t *testing.T) {
	t.Parallel()
	tbl := new(Table2[any])
	checkDump2(t, tbl, dumpTest{
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
[FULL] path: [] bits: +0 depth: 0
indexs(#2): [266 383]
prefxs(#2): 10/8 127/8
values(#2): <nil> <nil>
childs(#5): 10 127 169 172 192 

.[LEAF] path: [10.0] bits: +16 depth: 1
.indexs(#2): [256 257]
.prefxs(#2): 0/8 1/8
.values(#2): <nil> <nil>

.[LEAF] path: [127.0.0] bits: +24 depth: 1
.indexs(#1): [257]
.prefxs(#1): 1/8
.values(#1): <nil>

.[LEAF] path: [169] bits: +8 depth: 1
.indexs(#1): [510]
.prefxs(#1): 254/8
.values(#1): <nil>

.[LEAF] path: [172] bits: +8 depth: 1
.indexs(#1): [17]
.prefxs(#1): 16/4
.values(#1): <nil>

.[FULL] path: [192] bits: +8 depth: 1
.indexs(#1): [424]
.prefxs(#1): 168/8
.values(#1): <nil>
.childs(#1): 168 

..[LEAF] path: [192.168] bits: +16 depth: 2
..indexs(#1): [257]
..prefxs(#1): 1/8
..values(#1): <nil>
### IPv6:
[NULL] path: [] bits: +0 depth: 0
`,
	})
}

func TestDumpV6Sample2(t *testing.T) {
	t.Parallel()
	tbl := new(Table2[any])
	checkDump2(t, tbl, dumpTest{
		cidrs: []netip.Prefix{
			mpp("fe80::/10"),
			mpp("2000::/3"),
			mpp("2001:db8::/32"),
		},
		want: `### IPv4:
[NULL] path: [] bits: +0 depth: 0
### IPv6:
[FULL] path: [] bits: +0 depth: 0
indexs(#1): [9]
prefxs(#1): 0x20/3
values(#1): <nil>
childs(#2): 0x20 0xfe 

.[LEAF] path: [2001:0d] bits: +24 depth: 1
.indexs(#1): [440]
.prefxs(#1): 0xb8/8
.values(#1): <nil>

.[LEAF] path: [fe] bits: +8 depth: 1
.indexs(#1): [6]
.prefxs(#1): 0x80/2
.values(#1): <nil>
`,
	})
}

func TestDumpSample2(t *testing.T) {
	t.Parallel()
	tbl := new(Table2[any])
	checkDump2(t, tbl, dumpTest{
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
[FULL] path: [] bits: +0 depth: 0
indexs(#2): [266 383]
prefxs(#2): 10/8 127/8
values(#2): <nil> <nil>
childs(#5): 10 127 169 172 192 

.[LEAF] path: [10.0] bits: +16 depth: 1
.indexs(#2): [256 257]
.prefxs(#2): 0/8 1/8
.values(#2): <nil> <nil>

.[LEAF] path: [127.0.0] bits: +24 depth: 1
.indexs(#1): [257]
.prefxs(#1): 1/8
.values(#1): <nil>

.[LEAF] path: [169] bits: +8 depth: 1
.indexs(#1): [510]
.prefxs(#1): 254/8
.values(#1): <nil>

.[LEAF] path: [172] bits: +8 depth: 1
.indexs(#1): [17]
.prefxs(#1): 16/4
.values(#1): <nil>

.[FULL] path: [192] bits: +8 depth: 1
.indexs(#1): [424]
.prefxs(#1): 168/8
.values(#1): <nil>
.childs(#1): 168 

..[LEAF] path: [192.168] bits: +16 depth: 2
..indexs(#1): [257]
..prefxs(#1): 1/8
..values(#1): <nil>
### IPv6:
[FULL] path: [] bits: +0 depth: 0
indexs(#2): [1 9]
prefxs(#2): 0x00/0 0x20/3
values(#2): <nil> <nil>
childs(#2): 0x20 0xfe 

.[LEAF] path: [2001:0d] bits: +24 depth: 1
.indexs(#1): [440]
.prefxs(#1): 0xb8/8
.values(#1): <nil>

.[LEAF] path: [fe] bits: +8 depth: 1
.indexs(#1): [6]
.prefxs(#1): 0x80/2
.values(#1): <nil>
`,
	})
}

func checkDump2(t *testing.T, tbl *Table2[any], tt dumpTest) {
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
