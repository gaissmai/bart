package main

import (
	"net/netip"

	"github.com/gaissmai/bart"
)

func main() {
	a := new(bart.Table[bool])
	b := new(bart.Table[bool])
	p := netip.MustParsePrefix("10.0.0.0/24")
	a.Insert(p, true)
	b.Insert(p, true)
	a.Overlaps(b)
}
