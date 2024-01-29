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
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Dump(nil) did not panic")
		}
	}()

	p := netip.MustParsePrefix
	tbl := new(Table[any])
	tbl.Insert(p("1.2.3.4/32"), nil)
	_ = tbl.Dump(nil)
}

func TestDumperEmpty(t *testing.T) {
	tbl := new(Table[any])
	checkDump(t, tbl, dumpTest{
		cidrs: []netip.Prefix{},
		want: `IPv4:

[ROOT] depth:  0 path: [] / 0
IPv6:

[ROOT] depth:  0 path: [] / 0
`,
	})
}

func TestDumpDefaultRouteV4(t *testing.T) {
	p := netip.MustParsePrefix
	tbl := new(Table[any])
	checkDump(t, tbl, dumpTest{
		cidrs: []netip.Prefix{
			p("0.0.0.0/0"),
		},
		want: `IPv4:

[LEAF] depth:  0 path: [] / 0
indexs(#1): [1]
prefxs(#1): 0/0 
IPv6:

[ROOT] depth:  0 path: [] / 0
`,
	})
}

func TestDumpDefaultRouteV6(t *testing.T) {
	p := netip.MustParsePrefix
	tbl := new(Table[any])
	checkDump(t, tbl, dumpTest{
		cidrs: []netip.Prefix{
			p("::/0"),
		},
		want: `IPv4:

[ROOT] depth:  0 path: [] / 0
IPv6:

[LEAF] depth:  0 path: [] / 0
indexs(#1): [1]
prefxs(#1): 0x00/0 
`,
	})
}

func TestDumpSampleV4(t *testing.T) {
	p := netip.MustParsePrefix
	tbl := new(Table[any])
	checkDump(t, tbl, dumpTest{
		cidrs: []netip.Prefix{
			p("172.16.0.0/12"),
			p("10.0.0.0/24"),
			p("192.168.0.0/16"),
			p("10.0.0.0/8"),
			p("10.0.1.0/24"),
			p("169.254.0.0/16"),
			p("127.0.0.0/8"),
			p("127.0.0.1/32"),
			p("192.168.1.0/24"),
		},
		want: `IPv4:

[FULL] depth:  0 path: [] / 0
indexs(#2): [266 383]
prefxs(#2): 10/8 127/8 
childs(#5): 10 127 169 172 192 

.[IMED] depth:  1 path: [10] / 8
.childs(#1): 0 

..[LEAF] depth:  2 path: [10.0] / 16
..indexs(#2): [256 257]
..prefxs(#2): 0/8 1/8 

.[IMED] depth:  1 path: [127] / 8
.childs(#1): 0 

..[IMED] depth:  2 path: [127.0] / 16
..childs(#1): 0 

...[LEAF] depth:  3 path: [127.0.0] / 24
...indexs(#1): [257]
...prefxs(#1): 1/8 

.[LEAF] depth:  1 path: [169] / 8
.indexs(#1): [510]
.prefxs(#1): 254/8 

.[LEAF] depth:  1 path: [172] / 8
.indexs(#1): [17]
.prefxs(#1): 16/4 

.[FULL] depth:  1 path: [192] / 8
.indexs(#1): [424]
.prefxs(#1): 168/8 
.childs(#1): 168 

..[LEAF] depth:  2 path: [192.168] / 16
..indexs(#1): [257]
..prefxs(#1): 1/8 
IPv6:

[ROOT] depth:  0 path: [] / 0
`,
	})
}

func TestDumpSampleV6(t *testing.T) {
	p := netip.MustParsePrefix
	tbl := new(Table[any])
	checkDump(t, tbl, dumpTest{
		cidrs: []netip.Prefix{
			p("fe80::/10"),
			p("2000::/3"),
			p("2001:db8::/32"),
		},
		want: `IPv4:

[ROOT] depth:  0 path: [] / 0
IPv6:

[FULL] depth:  0 path: [] / 0
indexs(#1): [9]
prefxs(#1): 0x20/3 
childs(#2): 0x20 0xfe 

.[IMED] depth:  1 path: [20] / 8
.childs(#1): 0x01 

..[IMED] depth:  2 path: [2001] / 16
..childs(#1): 0x0d 

...[LEAF] depth:  3 path: [2001:0d] / 24
...indexs(#1): [440]
...prefxs(#1): 0xb8/8 

.[LEAF] depth:  1 path: [fe] / 8
.indexs(#1): [6]
.prefxs(#1): 0x80/2 
`,
	})
}

func TestDumpSample(t *testing.T) {
	p := netip.MustParsePrefix
	tbl := new(Table[any])
	checkDump(t, tbl, dumpTest{
		cidrs: []netip.Prefix{
			p("fe80::/10"),
			p("172.16.0.0/12"),
			p("10.0.0.0/24"),
			p("192.168.0.0/16"),
			p("10.0.0.0/8"),
			p("::/0"),
			p("10.0.1.0/24"),
			p("169.254.0.0/16"),
			p("2000::/3"),
			p("2001:db8::/32"),
			p("127.0.0.0/8"),
			p("127.0.0.1/32"),
			p("192.168.1.0/24"),
		},
		want: `IPv4:

[FULL] depth:  0 path: [] / 0
indexs(#2): [266 383]
prefxs(#2): 10/8 127/8 
childs(#5): 10 127 169 172 192 

.[IMED] depth:  1 path: [10] / 8
.childs(#1): 0 

..[LEAF] depth:  2 path: [10.0] / 16
..indexs(#2): [256 257]
..prefxs(#2): 0/8 1/8 

.[IMED] depth:  1 path: [127] / 8
.childs(#1): 0 

..[IMED] depth:  2 path: [127.0] / 16
..childs(#1): 0 

...[LEAF] depth:  3 path: [127.0.0] / 24
...indexs(#1): [257]
...prefxs(#1): 1/8 

.[LEAF] depth:  1 path: [169] / 8
.indexs(#1): [510]
.prefxs(#1): 254/8 

.[LEAF] depth:  1 path: [172] / 8
.indexs(#1): [17]
.prefxs(#1): 16/4 

.[FULL] depth:  1 path: [192] / 8
.indexs(#1): [424]
.prefxs(#1): 168/8 
.childs(#1): 168 

..[LEAF] depth:  2 path: [192.168] / 16
..indexs(#1): [257]
..prefxs(#1): 1/8 
IPv6:

[FULL] depth:  0 path: [] / 0
indexs(#2): [1 9]
prefxs(#2): 0x00/0 0x20/3 
childs(#2): 0x20 0xfe 

.[IMED] depth:  1 path: [20] / 8
.childs(#1): 0x01 

..[IMED] depth:  2 path: [2001] / 16
..childs(#1): 0x0d 

...[LEAF] depth:  3 path: [2001:0d] / 24
...indexs(#1): [440]
...prefxs(#1): 0xb8/8 

.[LEAF] depth:  1 path: [fe] / 8
.indexs(#1): [6]
.prefxs(#1): 0x80/2 
`,
	})
}

func checkDump(t *testing.T, tbl *Table[any], tt dumpTest) {
	t.Helper()
	for _, cidr := range tt.cidrs {
		tbl.Insert(cidr, nil)
	}
	w := new(strings.Builder)
	if err := tbl.Dump(w); err != nil {
		t.Errorf("Dump() unexpected err: %v", err)
	}
	got := w.String()
	if tt.want != got {
		t.Errorf("Dump got:\n%swant:\n%s", got, tt.want)
	}
}
