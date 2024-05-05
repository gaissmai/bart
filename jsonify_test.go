// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"encoding/json"
	"net/netip"
	"testing"
)

type jsonTestNode struct {
	cidr  netip.Prefix
	value any
}

func newJsonTestNode(cidr string, value any) jsonTestNode {
	return jsonTestNode{
		cidr:  mpp(cidr),
		value: value,
	}
}

type jsonTest struct {
	nodes []jsonTestNode
	want  string
}

func TestJsonEmpty(t *testing.T) {
	t.Parallel()
	tbl := new(Table[any])
	checkJson(t, tbl, jsonTest{
		nodes: []jsonTestNode{},
		want:  "{}",
	})
}

func TestJsonDefaultRouteV4(t *testing.T) {
	t.Parallel()
	tbl := new(Table[any])
	checkJson(t, tbl, jsonTest{
		nodes: []jsonTestNode{
			newJsonTestNode("0.0.0.0/0", nil),
		},
		want: `{"ipv4":[{"cidr":"0.0.0.0/0","value":null}]}`,
	})
}

func TestJsonDefaultRouteV6(t *testing.T) {
	t.Parallel()
	tbl := new(Table[any])
	checkJson(t, tbl, jsonTest{
		nodes: []jsonTestNode{
			newJsonTestNode("::/0", 31337),
		},
		want: `{"ipv6":[{"cidr":"::/0","value":31337}]}`,
	})
}

func TestJsonSampleV4(t *testing.T) {
	t.Parallel()
	tbl := new(Table[any])
	checkJson(t, tbl, jsonTest{
		nodes: []jsonTestNode{
			newJsonTestNode("172.16.0.0/12", nil),
			newJsonTestNode("10.0.0.0/24", nil),
			newJsonTestNode("192.168.0.0/16", nil),
			newJsonTestNode("10.0.0.0/8", nil),
			newJsonTestNode("10.0.1.0/24", nil),
			newJsonTestNode("169.254.0.0/16", nil),
			newJsonTestNode("127.0.0.0/8", nil),
			newJsonTestNode("127.0.0.1/32", nil),
			newJsonTestNode("192.168.1.0/24", nil),
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
	})
}

func TestJsonSampleV6(t *testing.T) {
	t.Parallel()
	tbl := new(Table[any])
	checkJson(t, tbl, jsonTest{
		nodes: []jsonTestNode{
			newJsonTestNode("fe80::/10", nil),
			newJsonTestNode("::1/128", nil),
			newJsonTestNode("2000::/3", nil),
			newJsonTestNode("2001:db8::/32", nil),
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
	})
}

func TestJsonSample(t *testing.T) {
	t.Parallel()
	// ipv4 + ipv6 and various types of value
	tbl := new(Table[any])
	checkJson(t, tbl, jsonTest{
		nodes: []jsonTestNode{
			newJsonTestNode("fe80::/10", nil),
			newJsonTestNode("172.16.0.0/12", nil),
			newJsonTestNode("10.0.0.0/24", nil),
			newJsonTestNode("::1/128", nil),
			newJsonTestNode("10.0.0.0/8", nil),
			newJsonTestNode("::/0", nil),
			newJsonTestNode("10.0.1.0/24", nil),
			newJsonTestNode("2000::/3", nil),
			newJsonTestNode("2001:db8::/32", nil),
			// some different value types:
			newJsonTestNode("127.0.0.0/8", 31337),
			newJsonTestNode("169.254.0.0/16", 3.14),
			newJsonTestNode("127.0.0.1/32", "some string"),
			newJsonTestNode("192.168.0.0/16", []string{"a", "c", "ff"}),
			newJsonTestNode("192.168.1.0/24", "550e8400-e29b-41d4-a716-446655440000"),
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
	})
}

func checkJson(t *testing.T, tbl *Table[any], tt jsonTest) {
	t.Helper()
	for _, node := range tt.nodes {
		tbl.Insert(node.cidr, node.value)
	}

	jsonBuffer, err := json.Marshal(tbl)
	if err != nil {
		t.Errorf("Json marshal got error: %s", err)
	}

	got := string(jsonBuffer)
	if tt.want != got {
		t.Errorf("String got:\n%s\nwant:\n%s", got, tt.want)
	}
}
