// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"encoding/json"
	"net/netip"
	"slices"
)

type ListElement[V any] struct {
	Cidr    netip.Prefix     `json:"cidr"`
	Value   V                `json:"value"`
	Subnets []ListElement[V] `json:"subnets,omitempty"`
}

// MarshalJSON dumps table into two lists: for ipv4 and ipv6
// every root and subnets are array, not map (cidr -> {value,subnets}), because the order matters
func (t *Table[V]) MarshalJSON() ([]byte, error) {
	t.init()

	result := struct {
		Ipv4 []ListElement[V] `json:"ipv4,omitempty"`
		Ipv6 []ListElement[V] `json:"ipv6,omitempty"`
	}{}

	var err error
	result.Ipv4, err = t.DumpList(true)
	if err != nil {
		return nil, err
	}
	result.Ipv6, err = t.DumpList(false)
	if err != nil {
		return nil, err
	}

	buf, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// DumpList dumps ipv4 od ipv6 tree into list of roots and their children
func (t *Table[V]) DumpList(is4 bool) ([]ListElement[V], error) {
	t.init()
	rootNode := t.rootNodeByVersion(is4)
	if rootNode.isEmpty() {
		return nil, nil
	}

	elements, err := rootNode.dumpList(0, nil, is4)
	if err != nil {
		return nil, err
	}

	return elements, nil
}

func (n *node[V]) dumpList(parentIdx uint, path []byte, is4 bool) ([]ListElement[V], error) {
	directKids := n.getKidsRec(parentIdx, path, is4)
	slices.SortFunc(directKids, sortPrefix[V])

	elements := make([]ListElement[V], 0, len(directKids))
	for _, kid := range directKids {
		element := ListElement[V]{
			Cidr:  kid.cidr,
			Value: kid.val,
		}

		subnetList, err := kid.n.dumpList(kid.idx, kid.path, is4)
		if err != nil {
			return nil, err
		}
		element.Subnets = subnetList

		elements = append(elements, element)
	}

	return elements, nil
}
