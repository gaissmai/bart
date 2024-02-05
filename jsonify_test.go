package bart

import (
	"encoding/json"
	"net/netip"
	"testing"
)

type jsonTestElement struct {
	cidr  netip.Prefix
	value any
}

func newJsonTestElement(cidr string, value any) jsonTestElement {
	return jsonTestElement{
		cidr:  netip.MustParsePrefix(cidr),
		value: value,
	}
}

type jsonTest struct {
	elements []jsonTestElement
	want     string
}

func TestJsonEmpty(t *testing.T) {
	tbl := new(Table[any])
	checkJson(t, tbl, jsonTest{
		elements: []jsonTestElement{},
		want:     "{}",
	})
}

func TestJsonDefaultRouteV4(t *testing.T) {
	tbl := new(Table[any])
	checkJson(t, tbl, jsonTest{
		elements: []jsonTestElement{
			newJsonTestElement("0.0.0.0/0", nil),
		},
		want: `{"ipv4":[{"cidr":"0.0.0.0/0","value":null}]}`,
	})
}

func TestJsonDefaultRouteV6(t *testing.T) {
	tbl := new(Table[any])
	checkJson(t, tbl, jsonTest{
		elements: []jsonTestElement{
			newJsonTestElement("::/0", 31337),
		},
		want: `{"ipv6":[{"cidr":"::/0","value":31337}]}`,
	})
}

func TestJsonSampleV4(t *testing.T) {
	tbl := new(Table[any])
	checkJson(t, tbl, jsonTest{
		elements: []jsonTestElement{
			newJsonTestElement("172.16.0.0/12", nil),
			newJsonTestElement("10.0.0.0/24", nil),
			newJsonTestElement("192.168.0.0/16", nil),
			newJsonTestElement("10.0.0.0/8", nil),
			newJsonTestElement("10.0.1.0/24", nil),
			newJsonTestElement("169.254.0.0/16", nil),
			newJsonTestElement("127.0.0.0/8", nil),
			newJsonTestElement("127.0.0.1/32", nil),
			newJsonTestElement("192.168.1.0/24", nil),
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
	tbl := new(Table[any])
	checkJson(t, tbl, jsonTest{
		elements: []jsonTestElement{
			newJsonTestElement("fe80::/10", nil),
			newJsonTestElement("::1/128", nil),
			newJsonTestElement("2000::/3", nil),
			newJsonTestElement("2001:db8::/32", nil),
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
	// ipv4 + ipv6 and various types of value
	tbl := new(Table[any])
	checkJson(t, tbl, jsonTest{
		elements: []jsonTestElement{
			newJsonTestElement("fe80::/10", nil),
			newJsonTestElement("172.16.0.0/12", nil),
			newJsonTestElement("10.0.0.0/24", nil),
			newJsonTestElement("::1/128", nil),
			newJsonTestElement("10.0.0.0/8", nil),
			newJsonTestElement("::/0", nil),
			newJsonTestElement("10.0.1.0/24", nil),
			newJsonTestElement("2000::/3", nil),
			newJsonTestElement("2001:db8::/32", nil),
			// some different value types:
			newJsonTestElement("127.0.0.0/8", 31337),
			newJsonTestElement("169.254.0.0/16", 3.14),
			newJsonTestElement("127.0.0.1/32", "some string"),
			newJsonTestElement("192.168.0.0/16", []string{"a", "c", "ff"}),
			newJsonTestElement("192.168.1.0/24", "550e8400-e29b-41d4-a716-446655440000"),
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
	for _, element := range tt.elements {
		tbl.Insert(element.cidr, element.value)
	}

	jsonBuffer, err := json.Marshal(tbl)
	if err != nil {
		t.Errorf("Json marshal got error: %s", err)
	}

	got := string(jsonBuffer)
	if tt.want != got {
		t.Errorf("String got:\n%s\nwant:\n%s\n%s", got, tt.want, tt.want)
	}
}
