// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"strings"
	"testing"
)

func TestLiteDumperPanic(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Dump(nil) did not panic")
		}
	}()

	tbl := new(Lite)
	tbl.Insert(mpp("1.2.3.4/32"))
	tbl.dump(nil)
}

func TestLiteDumperEmpty(t *testing.T) {
	t.Parallel()
	tbl := new(Lite)
	checkLiteDump(t, tbl, dumpTest{
		cidrs: nil,
		want:  "",
	})
}

func TestLiteDumpDefaultRouteV4(t *testing.T) {
	t.Parallel()
	tbl := new(Lite)
	checkLiteDump(t, tbl, dumpTest{
		cidrs: []netip.Prefix{
			mpp("0.0.0.0/0"),
		},
		want: `
### IPv4: nodes(1), pfxs(1), leaves(0), fringes(0),
[LEAF] depth:  0 path: [] / 0
indexs(#1): [1]
prefxs(#1): 0/0
`,
	})
}

func TestLiteDumpDefaultRouteV6(t *testing.T) {
	t.Parallel()
	tbl := new(Lite)
	checkLiteDump(t, tbl, dumpTest{
		cidrs: []netip.Prefix{
			mpp("::/0"),
		},
		want: `
### IPv6: nodes(1), pfxs(1), leaves(0), fringes(0),
[LEAF] depth:  0 path: [] / 0
indexs(#1): [1]
prefxs(#1): 0x00/0
`,
	})
}

func TestLiteDumpSampleV4(t *testing.T) {
	t.Parallel()
	tbl := new(Lite)

	checkLiteDump(t, tbl, dumpTest{
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
		want: `
### IPv4: nodes(6), pfxs(3), leaves(3), fringes(3),
[FULL] depth:  0 path: [] / 0
childs(#3): 10 127 192
leaves(#2): 169:{169.254.0.0/16} 172:{172.16.0.0/12}

.[FULL] depth:  1 path: [10] / 8
.indexs(#1): [1]
.prefxs(#1): 0/0
.childs(#1): 0

..[LEAF] depth:  2 path: [10.0] / 16
..fringe(#2): 0:{10.0.0.0/24} 1:{10.0.1.0/24}

.[LEAF] depth:  1 path: [127] / 8
.indexs(#1): [1]
.prefxs(#1): 0/0
.leaves(#1): 0:{127.0.0.1/32}

.[IMED] depth:  1 path: [192] / 8
.childs(#1): 168

..[LEAF] depth:  2 path: [192.168] / 16
..indexs(#1): [1]
..prefxs(#1): 0/0
..fringe(#1): 1:{192.168.1.0/24}
`,
	})
}

func TestLiteDumpSampleV6(t *testing.T) {
	t.Parallel()
	tbl := new(Lite)
	checkLiteDump(t, tbl, dumpTest{
		cidrs: []netip.Prefix{
			mpp("::/0"),
			mpp("fe80::/10"),
			mpp("2000::/3"),
			mpp("2001:db8::/32"),
		},
		want: `
### IPv6: nodes(1), pfxs(2), leaves(2), fringes(0),
[LEAF] depth:  0 path: [] / 0
indexs(#2): [1 9]
prefxs(#2): 0x00/0 0x20/3
leaves(#2): 0x20:{2001:db8::/32} 0xfe:{fe80::/10}
`,
	})
}

func TestLiteDumpSample(t *testing.T) {
	t.Parallel()
	tbl := new(Lite)

	checkLiteDump(t, tbl, dumpTest{
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
		want: `
### IPv4: nodes(6), pfxs(3), leaves(3), fringes(3),
[FULL] depth:  0 path: [] / 0
childs(#3): 10 127 192
leaves(#2): 169:{169.254.0.0/16} 172:{172.16.0.0/12}

.[FULL] depth:  1 path: [10] / 8
.indexs(#1): [1]
.prefxs(#1): 0/0
.childs(#1): 0

..[LEAF] depth:  2 path: [10.0] / 16
..fringe(#2): 0:{10.0.0.0/24} 1:{10.0.1.0/24}

.[LEAF] depth:  1 path: [127] / 8
.indexs(#1): [1]
.prefxs(#1): 0/0
.leaves(#1): 0:{127.0.0.1/32}

.[IMED] depth:  1 path: [192] / 8
.childs(#1): 168

..[LEAF] depth:  2 path: [192.168] / 16
..indexs(#1): [1]
..prefxs(#1): 0/0
..fringe(#1): 1:{192.168.1.0/24}

### IPv6: nodes(1), pfxs(2), leaves(2), fringes(0),
[LEAF] depth:  0 path: [] / 0
indexs(#2): [1 9]
prefxs(#2): 0x00/0 0x20/3
leaves(#2): 0x20:{2001:db8::/32} 0xfe:{fe80::/10}
`,
	})
}

func checkLiteDump(t *testing.T, tbl *Lite, tt dumpTest) {
	t.Helper()
	for _, cidr := range tt.cidrs {
		tbl.Insert(cidr)
	}
	w := new(strings.Builder)
	tbl.dump(w)
	got := w.String()
	if tt.want != got {
		t.Errorf("Dump got:\n%swant:\n%s", got, tt.want)
	}
}
