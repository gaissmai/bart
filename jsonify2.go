package bart

// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

import (
	"encoding/json"
	"slices"
)

// MarshalJSON dumps the table into two sorted lists: for ipv4 and ipv6.
// Every root and subnet is an array, not a map, because the order matters.
func (t *Table2[V]) MarshalJSON() ([]byte, error) {
	if t == nil {
		return nil, nil
	}

	result := struct {
		Ipv4 []DumpListNode[V] `json:"ipv4,omitempty"`
		Ipv6 []DumpListNode[V] `json:"ipv6,omitempty"`
	}{
		Ipv4: t.DumpList4(),
		Ipv6: t.DumpList6(),
	}

	buf, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// DumpList4 dumps the ipv4 tree into a list of roots and their subnets.
// It can be used to analyze the tree or build custom json representation.
func (t *Table2[V]) DumpList4() []DumpListNode[V] {
	if t == nil {
		return nil
	}
	return t.root4.dumpListRec(0, zeroPath, 0, true)
}

// DumpList6 dumps the ipv6 tree into a list of roots and their subnets.
// It can be used to analyze the tree or build custom json representation.
func (t *Table2[V]) DumpList6() []DumpListNode[V] {
	if t == nil {
		return nil
	}
	return t.root6.dumpListRec(0, zeroPath, 0, false)
}

func (n *node2[V]) dumpListRec(parentIdx uint, path [16]byte, depth int, is4 bool) []DumpListNode[V] {
	// recursion stop condition
	if n == nil {
		return nil
	}

	directKids := n.getKidsRec(parentIdx, path, depth, is4)
	slices.SortFunc(directKids, cmpKidByPrefix2[V])

	nodes := make([]DumpListNode[V], 0, len(directKids))
	for _, kid := range directKids {
		nodes = append(nodes, DumpListNode[V]{
			CIDR:    kid.cidr,
			Value:   kid.val,
			Subnets: kid.n.dumpListRec(kid.idx, kid.path, kid.depth, is4),
		})
	}

	return nodes
}