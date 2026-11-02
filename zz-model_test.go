// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"errors"
	"math/rand/v2"
	"net/netip"
	"regexp"
	"testing"

	"github.com/gaissmai/bart/internal/tests/golden"
	"github.com/gaissmai/bart/internal/tests/random"
)

// #####################################################
//  helpers and test data for the model tests in the
//  generated tests, see the model tests in commontests_tmpl.go.
//
//  The helpers are not generated, the generated tests exist
//  for every table type and must not define them again.
// #####################################################

// modelSeeds is the number of random cases per configuration.
func modelSeeds() int {
	if testing.Short() {
		return 30
	}

	return 150
}

// modelConfig is a configuration for the random tests.
type modelConfig struct {
	name string
	is4  bool
	pool []byte // the bytes the addresses are made of
	maxN int    // maximum number of prefixes
}

var modelConfigs = []modelConfig{
	{"IPv4 narrow", true, []byte{0, 1, 2, 3}, 30},
	{"IPv4 wide", true, []byte{0, 1, 2, 3, 127, 128, 254, 255}, 40},
	{"IPv6 narrow", false, []byte{0, 1, 2}, 30},
	{"IPv6 wide", false, []byte{0, 1, 0x20, 0x80, 0xfe, 0xff}, 40},
}

// modelOverlapPools are the pools for the random Overlaps tests, tables with
// disjoint, partly equal and equal pools.
var modelOverlapPools = [][]byte{{0, 1}, {2, 3}, {0, 2}, {0, 1, 2, 3}, {0, 1, 2, 3, 128, 255}}

// modelGold returns a golden table with the prefixes, the values are base,
// base+1, ... in the order of the prefixes.
func modelGold(pfxs []netip.Prefix, base int) golden.Table[int] {
	gold := make(golden.Table[int], len(pfxs))
	for i, pfx := range pfxs {
		gold.Insert(pfx, base+i)
	}

	return gold
}

// modelUnion returns the union of a and b, the values of b win.
func modelUnion(a, b golden.Table[int]) golden.Table[int] {
	u := make(golden.Table[int], len(a)+len(b))
	u.Union(a)
	u.Union(b)

	return u
}

// modelRelated returns prefixes that share about 40% with a, plus some new ones.
func modelRelated(prng *rand.Rand, cfg modelConfig, a []netip.Prefix) []netip.Prefix {
	var b []netip.Prefix

	for _, pfx := range a {
		if prng.IntN(5) < 2 {
			b = append(b, pfx)
		}
	}

	return append(b, random.NestedPrefixes(prng, cfg.is4, cfg.pool, prng.IntN(12), false)...)
}

// payloadRe matches the payload of a leaf or a fringe in a dump,
// the ", 100" in "10:{10.2.0.0/15, 100}".
var payloadRe = regexp.MustCompile(`(\{[^{},]+), [^{}]*\}`)

// modelStripPayload removes the payload of the leaves and fringes from a dump.
func modelStripPayload(dump string) string {
	return payloadRe.ReplaceAllString(dump, "${1}}")
}

// modelSpec describes a prefix by the first two bytes of the address, all other
// bytes are zero, and the prefix length. The same spec is used for IPv4 and for
// IPv6, so that every explicit case runs for both address families.
type modelSpec struct {
	a, b byte
	bits int
}

func modelPfx(is4 bool, a, b byte, bits int) netip.Prefix {
	if is4 {
		return netip.PrefixFrom(netip.AddrFrom4([4]byte{a, b}), bits).Masked()
	}

	var raw [16]byte
	raw[0], raw[1] = a, b

	return netip.PrefixFrom(netip.AddrFrom16(raw), bits).Masked()
}

func (s modelSpec) pfx(is4 bool) netip.Prefix {
	return modelPfx(is4, s.a, s.b, s.bits)
}

// modelPrefixes returns the prefixes for the specs, in the order of the specs.
func modelPrefixes(is4 bool, specs []modelSpec) []netip.Prefix {
	var pfxs []netip.Prefix
	for _, s := range specs {
		pfxs = append(pfxs, s.pfx(is4))
	}

	return pfxs
}

func modelName(name string, is4 bool) string {
	if is4 {
		return name + "/IPv4"
	}

	return name + "/IPv6"
}

// modelStridePrefixes returns the 255 prefixes /0 .. /7 of the first stride,
// stored in the prefix table of the root node, and the 255 prefixes /8 .. /15
// below the octet 10, stored as a fringe and as leaves.
func modelStridePrefixes() (inRoot, belowTen []netip.Prefix) {
	for l := range 8 {
		for k := range 1 << l {
			octet := byte(k << (8 - l))

			inRoot = append(inRoot, modelPfx(true, octet, 0, l))
			belowTen = append(belowTen, modelPfx(true, 10, octet, 8+l))
		}
	}

	return inRoot, belowTen
}

// #####################################################
//  explicit test data
// #####################################################

// the kinds of entries below the slot 10 of the root node: a child node, a leaf
// (path-compressed prefix) or a fringe (path-compressed prefix on a stride border).
// A node with two fringes below
var (
	modelNode1 = []modelSpec{{10, 1, 16}, {10, 2, 16}}
	modelNode2 = []modelSpec{{10, 3, 16}, {10, 4, 16}}
	// shares 10.2.0.0/16 with modelNode1
	modelNode1x = []modelSpec{{10, 2, 16}, {10, 9, 16}}
	// a leaf, 10.2.0.0/15 covers 10.2 and 10.3
	modelLeaf1 = []modelSpec{{10, 2, 15}}
	modelLeaf2 = []modelSpec{{10, 6, 15}}
	// a leaf inside of modelLeaf1
	modelLeaf3 = []modelSpec{{10, 3, 17}}
	// a fringe, 10.0.0.0/8
	modelFringe1 = []modelSpec{{10, 0, 8}}
	// nodes that store the prefix of a leaf or of a fringe in their prefix table
	modelNodeWithLeaf1   = []modelSpec{{10, 2, 15}, {10, 6, 15}}
	modelNodeWithFringe1 = []modelSpec{{10, 0, 8}, {10, 1, 16}}
	// prefixes in the prefix table of the root node, they cover the first octets
	// 8..15, 16..23, 0..31 and 32..63
	modelStrideA = []modelSpec{{8, 0, 5}}
	modelStrideB = []modelSpec{{16, 0, 5}}
	modelStrideC = []modelSpec{{0, 0, 3}}
	modelStrideD = []modelSpec{{32, 0, 3}}
	// a table with nested prefixes down to a host route
	modelStored1 = []modelSpec{{0, 0, 0}, {10, 0, 8}, {10, 1, 16}, {10, 2, 15}, {10, 2, 24}, {10, 2, 32}}
)

// modelUnionCases are the 12 combinations of the children of a node that Union
// handles, the entry of this table and the entry of the other table: nothing,
// node, leaf or fringe. Where a prefix is in both tables, dups is 1.
var modelUnionCases = []struct {
	name string
	a, b []modelSpec
	dups int // the number of prefixes that are in both tables
}{
	{"null+node", nil, modelNode1, 0},
	{"null+leaf", nil, modelLeaf1, 0},
	{"null+fringe", nil, modelFringe1, 0},
	{"node+node", modelNode1, modelNode2, 0},
	{"node+node, one duplicate", modelNode1, modelNode1x, 1},
	{"node+leaf", modelNode1, modelLeaf1, 0},
	{"node+leaf, duplicate", modelNodeWithLeaf1, modelLeaf1, 1},
	{"node+fringe", modelNode1, modelFringe1, 0},
	{"node+fringe, duplicate", modelNodeWithFringe1, modelFringe1, 1},
	{"leaf+node", modelLeaf1, modelNode1, 0},
	{"leaf+node, duplicate", modelLeaf1, modelNodeWithLeaf1, 1},
	{"leaf+leaf", modelLeaf1, modelLeaf2, 0},
	{"leaf+leaf, same prefix", modelLeaf1, modelLeaf1, 1},
	{"leaf+fringe", modelLeaf1, modelFringe1, 0},
	{"fringe+node", modelFringe1, modelNode1, 0},
	{"fringe+node, duplicate", modelFringe1, modelNodeWithFringe1, 1},
	{"fringe+leaf", modelFringe1, modelLeaf1, 0},
	{"fringe+fringe", modelFringe1, modelFringe1, 1},
}

// modelOverlapCases are the combinations of node, leaf, fringe and prefix
// table entries for Overlaps.
var modelOverlapCases = []struct {
	name string
	a, b []modelSpec
	want bool
}{
	{"node,node overlapping", modelNode1, modelNode1x, true},
	{"node,node disjoint", modelNode1, modelNode2, false},
	{"node,leaf overlapping", modelNode1, modelLeaf1, true},
	{"node,leaf disjoint", modelNode1, modelLeaf2, false},
	{"node,fringe", modelNode1, modelFringe1, true},
	{"leaf,node overlapping", modelLeaf1, modelNode1, true},
	{"leaf,node disjoint", modelLeaf2, modelNode1, false},
	{"leaf,leaf overlapping", modelLeaf1, modelLeaf3, true},
	{"leaf,leaf disjoint", modelLeaf1, modelLeaf2, false},
	{"leaf,fringe", modelLeaf1, modelFringe1, true},
	{"fringe,node", modelFringe1, modelNode1, true},
	{"fringe,leaf", modelFringe1, modelLeaf1, true},
	{"fringe,fringe", modelFringe1, modelFringe1, true},
	{"prefix contains prefix", modelStrideC, modelStrideA, true},
	{"prefix inside prefix", modelStrideA, modelStrideC, true},
	{"prefixes disjoint", modelStrideC, modelStrideD, false},
	{"identical prefixes", modelStrideA, modelStrideA, true},
	{"prefix covers fringe", modelStrideA, modelFringe1, true},
	{"prefix beside fringe", modelStrideB, modelFringe1, false},
	{"prefix covers leaf", modelStrideA, modelLeaf1, true},
	{"prefix beside leaf", modelStrideB, modelLeaf1, false},
	{"prefix covers node", modelStrideA, modelNode1, true},
	{"prefix beside node", modelStrideB, modelNode1, false},
	{"node with prefixes, leaf inside a prefix", modelNodeWithLeaf1, []modelSpec{{10, 3, 24}}, true},
	{"node with prefixes, leaf outside", modelNodeWithLeaf1, []modelSpec{{10, 8, 24}}, false},
	{"default route against node", []modelSpec{{0, 0, 0}}, modelNode1, true},
	{"default route against empty", []modelSpec{{0, 0, 0}}, nil, false},
	{"empty against empty", nil, nil, false},
}

// modelEdgeCases are the edge cases for Subnets and Supernets with the
// expected result in the order of the iterators: the subnets in canonical order,
// the supernets the most specific first.
var modelEdgeCases = []struct {
	name               string
	stored             []modelSpec
	probe              modelSpec
	subnets, supernets []modelSpec
}{
	{
		"default route table, probe the default route", modelStored1,
		modelSpec{0, 0, 0},
		[]modelSpec{{0, 0, 0}, {10, 0, 8}, {10, 1, 16}, {10, 2, 15}, {10, 2, 24}, {10, 2, 32}},
		[]modelSpec{{0, 0, 0}},
	},
	{
		"probe is stored, shorter supernets exist", modelStored1,
		modelSpec{10, 0, 8},
		[]modelSpec{{10, 0, 8}, {10, 1, 16}, {10, 2, 15}, {10, 2, 24}, {10, 2, 32}},
		[]modelSpec{{10, 0, 8}, {0, 0, 0}},
	},
	{
		"probe is not stored, covers stored prefixes", modelStored1,
		modelSpec{10, 2, 16},
		[]modelSpec{{10, 2, 24}, {10, 2, 32}},
		[]modelSpec{{10, 2, 15}, {10, 0, 8}, {0, 0, 0}},
	},
	{
		"probe is a host route that is stored", modelStored1,
		modelSpec{10, 2, 32},
		[]modelSpec{{10, 2, 32}},
		[]modelSpec{{10, 2, 32}, {10, 2, 24}, {10, 2, 15}, {10, 0, 8}, {0, 0, 0}},
	},
	{
		"probe is disjoint from all but the default route", modelStored1,
		modelSpec{11, 0, 8},
		nil,
		[]modelSpec{{0, 0, 0}},
	},
	{
		"probe is a host route that is not stored", modelStored1,
		modelSpec{10, 3, 32},
		nil,
		[]modelSpec{{10, 2, 15}, {10, 0, 8}, {0, 0, 0}},
	},
	{
		"probe is wider than every stored prefix",
		[]modelSpec{{10, 1, 16}, {10, 2, 16}},
		modelSpec{10, 0, 8},
		[]modelSpec{{10, 1, 16}, {10, 2, 16}},
		nil,
	},
	{
		"probe is wider and no default route exists",
		[]modelSpec{{10, 1, 16}, {10, 2, 16}},
		modelSpec{0, 0, 0},
		[]modelSpec{{10, 1, 16}, {10, 2, 16}},
		nil,
	},
	{
		"nested /31 and /32",
		[]modelSpec{{10, 1, 31}, {10, 1, 32}},
		modelSpec{10, 1, 30},
		[]modelSpec{{10, 1, 31}, {10, 1, 32}},
		nil,
	},
	{
		"nested /31 and /32, probe /32",
		[]modelSpec{{10, 1, 31}, {10, 1, 32}},
		modelSpec{10, 1, 32},
		[]modelSpec{{10, 1, 32}},
		[]modelSpec{{10, 1, 32}, {10, 1, 31}},
	},
	{
		"nested /31 and /32, probe /31",
		[]modelSpec{{10, 1, 31}, {10, 1, 32}},
		modelSpec{10, 1, 31},
		[]modelSpec{{10, 1, 31}, {10, 1, 32}},
		[]modelSpec{{10, 1, 31}},
	},
	{
		"empty table", nil,
		modelSpec{0, 0, 0},
		nil,
		nil,
	},
}

// modelEqualCases are pairs of tables with the same number of prefixes, that
// differ in the structure: in the prefixes of a node, in the child slots and in
// the kind of the children. The size check of Equal does not hide the difference.
var modelEqualCases = []struct {
	name string
	a, b []modelSpec
}{
	{"leaf and node swapped between two slots", []modelSpec{{10, 2, 15}, {11, 1, 16}, {11, 2, 16}}, []modelSpec{{10, 1, 16}, {10, 2, 16}, {11, 2, 15}}},
	{"fringe and node swapped between two slots", []modelSpec{{10, 0, 8}, {11, 1, 16}, {11, 2, 16}}, []modelSpec{{10, 1, 16}, {10, 2, 16}, {11, 0, 8}}},
	{"leaf against fringe", []modelSpec{{10, 2, 15}}, []modelSpec{{10, 0, 8}}},
	{"leaves with different prefixes", []modelSpec{{10, 2, 15}}, []modelSpec{{10, 4, 15}}},
	{"nodes with different prefix tables", []modelSpec{{10, 2, 15}, {10, 6, 15}}, []modelSpec{{10, 2, 15}, {10, 4, 15}}},
	{"node with default route against node without", []modelSpec{{10, 0, 8}, {10, 1, 16}}, []modelSpec{{10, 1, 16}, {10, 2, 16}}},
	{"different child slots", []modelSpec{{10, 0, 8}}, []modelSpec{{11, 0, 8}}},
	{"different prefix tables in the root", []modelSpec{{8, 0, 5}}, []modelSpec{{16, 0, 5}}},
}

// modelValueCases are the kinds of entries for the comparison of the values
// by Equal: the prefix table of a node, a node, a leaf and a fringe.
var modelValueCases = []struct {
	name  string
	specs []modelSpec
}{
	{"prefix table", modelStrideA},
	{"fringe", modelFringe1},
	{"leaf", modelLeaf1},
	{"node with fringes", modelNode1},
	{"node with the default route", modelNodeWithFringe1},
	{"node with prefixes", modelNodeWithLeaf1},
}

// errModelWrite is the error of the modelFailWriter.
var errModelWrite = errors.New("model: write failed")

// modelFailWriter is an io.Writer that fails at the write number failAt,
// counted from 0. It never fails for a negative failAt. It counts the writes.
type modelFailWriter struct {
	failAt int
	writes int
}

func (w *modelFailWriter) Write(p []byte) (int, error) {
	n := w.writes
	w.writes++

	if n == w.failAt {
		return 0, errModelWrite
	}

	return len(p), nil
}
