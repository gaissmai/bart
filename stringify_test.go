// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"testing"
)

type stringTest struct {
	cidrs []netip.Prefix
	want  string
}

func TestStringPanic(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Fprint(nil) did not panic")
		}
	}()

	p := netip.MustParsePrefix
	tbl := new(Table[any])
	tbl.Insert(p("1.2.3.4/32"), nil)
	tbl.Fprint(nil)
}

func TestStringEmpty(t *testing.T) {
	t.Parallel()
	tbl := new(Table[any])
	checkString(t, tbl, stringTest{
		cidrs: []netip.Prefix{},
		want:  "",
	})
}

func TestStringDefaultRouteV4(t *testing.T) {
	t.Parallel()
	p := netip.MustParsePrefix
	tbl := new(Table[any])
	checkString(t, tbl, stringTest{
		cidrs: []netip.Prefix{
			p("0.0.0.0/0"),
		},
		want: `▼
└─ 0.0.0.0/0 (<nil>)
`,
	})
}

func TestStringDefaultRouteV6(t *testing.T) {
	t.Parallel()
	p := netip.MustParsePrefix
	tbl := new(Table[any])
	checkString(t, tbl, stringTest{
		cidrs: []netip.Prefix{
			p("::/0"),
		},
		want: `▼
└─ ::/0 (<nil>)
`,
	})
}

func TestStringSampleV4(t *testing.T) {
	t.Parallel()
	p := netip.MustParsePrefix
	tbl := new(Table[any])
	checkString(t, tbl, stringTest{
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
		want: `▼
├─ 10.0.0.0/8 (<nil>)
│  ├─ 10.0.0.0/24 (<nil>)
│  └─ 10.0.1.0/24 (<nil>)
├─ 127.0.0.0/8 (<nil>)
│  └─ 127.0.0.1/32 (<nil>)
├─ 169.254.0.0/16 (<nil>)
├─ 172.16.0.0/12 (<nil>)
└─ 192.168.0.0/16 (<nil>)
   └─ 192.168.1.0/24 (<nil>)
`,
	})
}

func TestStringSampleV6(t *testing.T) {
	t.Parallel()
	p := netip.MustParsePrefix
	tbl := new(Table[any])
	checkString(t, tbl, stringTest{
		cidrs: []netip.Prefix{
			p("fe80::/10"),
			p("::1/128"),
			p("2000::/3"),
			p("2001:db8::/32"),
		},
		want: `▼
├─ ::1/128 (<nil>)
├─ 2000::/3 (<nil>)
│  └─ 2001:db8::/32 (<nil>)
└─ fe80::/10 (<nil>)
`,
	})
}

func TestStringSample(t *testing.T) {
	t.Parallel()
	p := netip.MustParsePrefix
	tbl := new(Table[any])
	checkString(t, tbl, stringTest{
		cidrs: []netip.Prefix{
			p("fe80::/10"),
			p("172.16.0.0/12"),
			p("10.0.0.0/24"),
			p("::1/128"),
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
		want: `▼
├─ 10.0.0.0/8 (<nil>)
│  ├─ 10.0.0.0/24 (<nil>)
│  └─ 10.0.1.0/24 (<nil>)
├─ 127.0.0.0/8 (<nil>)
│  └─ 127.0.0.1/32 (<nil>)
├─ 169.254.0.0/16 (<nil>)
├─ 172.16.0.0/12 (<nil>)
└─ 192.168.0.0/16 (<nil>)
   └─ 192.168.1.0/24 (<nil>)
▼
└─ ::/0 (<nil>)
   ├─ ::1/128 (<nil>)
   ├─ 2000::/3 (<nil>)
   │  └─ 2001:db8::/32 (<nil>)
   └─ fe80::/10 (<nil>)
`,
	})
}

func checkString(t *testing.T, tbl *Table[any], tt stringTest) {
	t.Helper()
	for _, cidr := range tt.cidrs {
		tbl.Insert(cidr, nil)
	}
	got := tbl.String()
	if tt.want != got {
		t.Errorf("String got:\n%swant:\n%s", got, tt.want)
	}

	gotBytes, err := tbl.MarshalText()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if tt.want != string(gotBytes) {
		t.Errorf("MarshalText got:\n%swant:\n%s", gotBytes, tt.want)
	}
}
