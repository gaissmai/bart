// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"encoding/json"
	"net/netip"
	"testing"
)

// #########################################################

func TestFatStringEmpty(t *testing.T) {
	t.Parallel()
	tbl := new(Fat[any])
	want := ""
	got := tbl.String()
	if got != want {
		t.Errorf("empty table, expected %q, got %q", want, got)
	}
}

func TestFatStringDefaultRouteV4(t *testing.T) {
	t.Parallel()

	tt := stringTest{
		cidrs: []netip.Prefix{
			mpp("0.0.0.0/0"),
		},
		want: `▼
└─ 0.0.0.0/0 (<nil>)
`,
	}

	tbl := new(Fat[any])
	checkFatString(t, tbl, tt)
}

func TestFatStringDefaultRouteV6(t *testing.T) {
	t.Parallel()

	tt := stringTest{
		cidrs: []netip.Prefix{
			mpp("::/0"),
		},
		want: `▼
└─ ::/0 (<nil>)
`,
	}

	tbl := new(Fat[any])
	checkFatString(t, tbl, tt)
}

func TestFatStringSampleV4(t *testing.T) {
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

	tbl := new(Fat[any])
	checkFatString(t, tbl, tt)
}

func TestFatStringSampleV6(t *testing.T) {
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

	tbl := new(Fat[any])
	checkFatString(t, tbl, tt)
}

func TestFatStringSample(t *testing.T) {
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

	tbl := new(Fat[any])
	checkFatString(t, tbl, tt)
}

func checkFatString(t *testing.T, tbl *Fat[any], tt stringTest) {
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
		t.Fatalf("unexpected error: %v", err)
	}
	if tt.want != string(gotBytes) {
		t.Errorf("MarshalText got:\n%swant:\n%s", gotBytes, tt.want)
	}
}

func TestFatJSONTableIsNil(t *testing.T) {
	t.Parallel()
	tt := jsonTest{
		want: "null",
	}

	var tbl *Fat[any]
	checkFatJSON(t, tbl, tt)
}

func TestFatJSONEmpty(t *testing.T) {
	t.Parallel()

	tt := jsonTest{
		want: "{}",
	}

	tbl := new(Fat[any])
	checkFatJSON(t, tbl, tt)
}

func TestFatJSONDefaultRouteV4(t *testing.T) {
	t.Parallel()

	tt := jsonTest{
		nodes: []jsonTestNode{
			newJSONTestNode("0.0.0.0/0", nil),
		},
		want: `{"ipv4":[{"cidr":"0.0.0.0/0","value":null}]}`,
	}

	tbl := new(Fat[any])
	checkFatJSON(t, tbl, tt)
}

func TestFatJSONDefaultRouteV6(t *testing.T) {
	t.Parallel()

	tt := jsonTest{
		nodes: []jsonTestNode{
			newJSONTestNode("::/0", 31337),
		},
		want: `{"ipv6":[{"cidr":"::/0","value":31337}]}`,
	}

	tbl := new(Fat[any])
	checkFatJSON(t, tbl, tt)
}

func TestFatJSONSampleV4(t *testing.T) {
	t.Parallel()

	tt := jsonTest{
		nodes: []jsonTestNode{
			newJSONTestNode("172.16.0.0/12", nil),
			newJSONTestNode("10.0.0.0/24", nil),
			newJSONTestNode("192.168.0.0/16", nil),
			newJSONTestNode("10.0.0.0/8", nil),
			newJSONTestNode("10.0.1.0/24", nil),
			newJSONTestNode("169.254.0.0/16", nil),
			newJSONTestNode("127.0.0.0/8", nil),
			newJSONTestNode("127.0.0.1/32", nil),
			newJSONTestNode("192.168.1.0/24", nil),
		},
		/*
		   {
		     "ipv4": [
		       {
		         "cidr": "10.0.0.0/8",
		         "value": null,
		         "subnets": [
		           { "cidr": "10.0.0.0/24", "value": null },
		           { "cidr": "10.0.1.0/24", "value": null }
		         ]
		       },
		       {
		         "cidr": "127.0.0.0/8",
		         "value": null,
		         "subnets": [{ "cidr": "127.0.0.1/32", "value": null }]
		       },
		       { "cidr": "169.254.0.0/16", "value": null },
		       { "cidr": "172.16.0.0/12", "value": null },
		       {
		         "cidr": "192.168.0.0/16",
		         "value": null,
		         "subnets": [{ "cidr": "192.168.1.0/24", "value": null }]
		       }
		     ]
		   }
		*/
		want: `{"ipv4":[{"cidr":"10.0.0.0/8","value":null,"subnets":[{"cidr":"10.0.0.0/24","value":null},{"cidr":"10.0.1.0/24","value":null}]},{"cidr":"127.0.0.0/8","value":null,"subnets":[{"cidr":"127.0.0.1/32","value":null}]},{"cidr":"169.254.0.0/16","value":null},{"cidr":"172.16.0.0/12","value":null},{"cidr":"192.168.0.0/16","value":null,"subnets":[{"cidr":"192.168.1.0/24","value":null}]}]}`,
	}

	tbl := new(Fat[any])
	checkFatJSON(t, tbl, tt)
}

func TestFatJSONSampleV6(t *testing.T) {
	t.Parallel()

	tt := jsonTest{
		nodes: []jsonTestNode{
			newJSONTestNode("fe80::/10", nil),
			newJSONTestNode("::1/128", nil),
			newJSONTestNode("2000::/3", nil),
			newJSONTestNode("2001:db8::/32", nil),
		},
		/*
			{
			  "ipv6": [
			    { "cidr": "::1/128", "value": null },
			    {
			      "cidr": "2000::/3",
			      "value": null,
			      "subnets": [{ "cidr": "2001:db8::/32", "value": null }]
			    },
			    { "cidr": "fe80::/10", "value": null }
			  ]
			}
		*/
		want: `{"ipv6":[{"cidr":"::1/128","value":null},{"cidr":"2000::/3","value":null,"subnets":[{"cidr":"2001:db8::/32","value":null}]},{"cidr":"fe80::/10","value":null}]}`,
	}

	tbl := new(Fat[any])
	checkFatJSON(t, tbl, tt)
}

func TestFatJSONSample(t *testing.T) {
	t.Parallel()

	// ipv4 + ipv6 and various types of value
	tt := jsonTest{
		nodes: []jsonTestNode{
			newJSONTestNode("fe80::/10", nil),
			newJSONTestNode("172.16.0.0/12", nil),
			newJSONTestNode("10.0.0.0/24", nil),
			newJSONTestNode("::1/128", nil),
			newJSONTestNode("10.0.0.0/8", nil),
			newJSONTestNode("::/0", nil),
			newJSONTestNode("10.0.1.0/24", nil),
			newJSONTestNode("2000::/3", nil),
			newJSONTestNode("2001:db8::/32", nil),
			// some different value types:
			newJSONTestNode("127.0.0.0/8", 31337),
			newJSONTestNode("169.254.0.0/16", 3.14),
			newJSONTestNode("127.0.0.1/32", "some string"),
			newJSONTestNode("192.168.0.0/16", []string{"a", "c", "ff"}),
			newJSONTestNode("192.168.1.0/24", "550e8400-e29b-41d4-a716-446655440000"),
		},
		/*
			{
			  "ipv4": [
			    {
			      "cidr": "10.0.0.0/8",
			      "value": null,
			      "subnets": [
			        { "cidr": "10.0.0.0/24", "value": null },
			        { "cidr": "10.0.1.0/24", "value": null }
			      ]
			    },
			    {
			      "cidr": "127.0.0.0/8",
			      "value": 31337,
			      "subnets": [{ "cidr": "127.0.0.1/32", "value": "some string" }]
			    },
			    { "cidr": "169.254.0.0/16", "value": 3.14 },
			    { "cidr": "172.16.0.0/12", "value": null },
			    {
			      "cidr": "192.168.0.0/16",
			      "value": ["a", "c", "ff"],
			      "subnets": [
			        {
			          "cidr": "192.168.1.0/24",
			          "value": "550e8400-e29b-41d4-a716-446655440000"
			        }
			      ]
			    }
			  ],
			  "ipv6": [
			    {
			      "cidr": "::/0",
			      "value": null,
			      "subnets": [
			        { "cidr": "::1/128", "value": null },
			        {
			          "cidr": "2000::/3",
			          "value": null,
			          "subnets": [{ "cidr": "2001:db8::/32", "value": null }]
			        },
			        { "cidr": "fe80::/10", "value": null }
			      ]
			    }
			  ]
			}

		*/
		want: `{"ipv4":[{"cidr":"10.0.0.0/8","value":null,"subnets":[{"cidr":"10.0.0.0/24","value":null},{"cidr":"10.0.1.0/24","value":null}]},{"cidr":"127.0.0.0/8","value":31337,"subnets":[{"cidr":"127.0.0.1/32","value":"some string"}]},{"cidr":"169.254.0.0/16","value":3.14},{"cidr":"172.16.0.0/12","value":null},{"cidr":"192.168.0.0/16","value":["a","c","ff"],"subnets":[{"cidr":"192.168.1.0/24","value":"550e8400-e29b-41d4-a716-446655440000"}]}],"ipv6":[{"cidr":"::/0","value":null,"subnets":[{"cidr":"::1/128","value":null},{"cidr":"2000::/3","value":null,"subnets":[{"cidr":"2001:db8::/32","value":null}]},{"cidr":"fe80::/10","value":null}]}]}`,
	}

	tbl := new(Fat[any])
	checkFatJSON(t, tbl, tt)
}

func checkFatJSON(t *testing.T, tbl *Fat[any], tt jsonTest) {
	t.Helper()
	for _, node := range tt.nodes {
		tbl.Insert(node.cidr, node.value)
	}

	jsonBuffer, err := json.Marshal(tbl)
	if err != nil {
		t.Fatalf("JSON marshal got error: %s", err)
	}

	got := string(jsonBuffer)
	if tt.want != got {
		t.Errorf("String got:\n%s\nwant:\n%s", got, tt.want)
	}
}
