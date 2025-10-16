// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart_test

import (
	"net/netip"
	"testing"

	"github.com/gaissmai/bart"
)

var (
	a = netip.MustParseAddr
	p = netip.MustParsePrefix

	worstCaseProbeIP4 = a("255.255.255.255")

	worstCasePfxsIP4 = []netip.Prefix{
		p("0.0.0.0/1"),
		p("254.0.0.0/8"),
		p("255.0.0.0/9"),
		p("255.254.0.0/16"),
		p("255.255.0.0/17"),
		p("255.255.254.0/24"),
		p("255.255.255.0/25"),
		p("255.255.255.255/32"), // matching prefix
	}
)

func BenchmarkMyLite(b *testing.B) {
	tbl := new(bart.Lite)
	for _, p := range worstCasePfxsIP4 {
		tbl.Insert(p)
	}

	for b.Loop() {
		_ = tbl.Contains(worstCaseProbeIP4)
	}
}

func BenchmarkMyBart(b *testing.B) {
	tbl := new(bart.Table[string])
	for _, p := range worstCasePfxsIP4 {
		tbl.Insert(p, p.String())
	}

	for b.Loop() {
		_ = tbl.Contains(worstCaseProbeIP4)
	}
}

func BenchmarkMyFast(b *testing.B) {
	tbl := new(bart.Fast[string])
	for _, p := range worstCasePfxsIP4 {
		tbl.Insert(p, p.String())
	}

	for b.Loop() {
		_ = tbl.Contains(worstCaseProbeIP4)
	}
}
