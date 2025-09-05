// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"testing"
)

// #########################################################

func TestArtStringEmpty(t *testing.T) {
	t.Parallel()
	tbl := new(ArtTable[any])
	want := ""
	got := tbl.String()
	if got != want {
		t.Errorf("table is nil, expected %q, got %q", want, got)
	}
}

func TestArtStringDefaultRouteV4(t *testing.T) {
	t.Parallel()

	tt := stringTest{
		cidrs: []netip.Prefix{
			mpp("0.0.0.0/0"),
		},
		want: `▼
└─ 0.0.0.0/0 (<nil>)
`,
	}

	tbl := new(ArtTable[any])
	checkArtString(t, tbl, tt)
}

func TestArtStringDefaultRouteV6(t *testing.T) {
	t.Parallel()

	tt := stringTest{
		cidrs: []netip.Prefix{
			mpp("::/0"),
		},
		want: `▼
└─ ::/0 (<nil>)
`,
	}

	tbl := new(ArtTable[any])
	checkArtString(t, tbl, tt)
}

func TestArtStringSampleV4(t *testing.T) {
	t.Parallel()

	tt := stringTest{
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
	}

	tbl := new(ArtTable[any])
	checkArtString(t, tbl, tt)
}

func TestArtStringSampleV6(t *testing.T) {
	t.Parallel()
	tt := stringTest{
		cidrs: []netip.Prefix{
			mpp("fe80::/10"),
			mpp("::1/128"),
			mpp("2000::/3"),
			mpp("2001:db8::/32"),
		},
		want: `▼
├─ ::1/128 (<nil>)
├─ 2000::/3 (<nil>)
│  └─ 2001:db8::/32 (<nil>)
└─ fe80::/10 (<nil>)
`,
	}

	tbl := new(ArtTable[any])
	checkArtString(t, tbl, tt)
}

func TestArtStringSample(t *testing.T) {
	t.Parallel()

	tt := stringTest{
		cidrs: []netip.Prefix{
			mpp("fe80::/10"),
			mpp("172.16.0.0/12"),
			mpp("10.0.0.0/24"),
			mpp("::1/128"),
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
	}

	tbl := new(ArtTable[any])
	checkArtString(t, tbl, tt)
}

func checkArtString(t *testing.T, tbl *ArtTable[any], tt stringTest) {
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
