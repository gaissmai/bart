// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause
//
// some regression tests modified from github.com/tailscale/art
// for this implementation by:
//
// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"fmt"
	"math/rand"
	"net/netip"
	"runtime"
	"testing"
)

var mpa = netip.MustParseAddr

var mpp = netip.MustParsePrefix

// nodes, calculates the nodes.
func (t *Table[V]) nodes() int {
	return t.root4.numNodesRec() + t.root6.numNodesRec()
}

// ############ tests ################################

func TestInvalidPrefix(t *testing.T) {
	t.Parallel()

	tbl := new(Table[any])
	var zero netip.Prefix
	var testname string

	testname = "Insert"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid prefix input", testname)
			}
		}(testname)

		tbl.Insert(zero, nil)
	})

	testname = "Delete"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid prefix input", testname)
			}
		}(testname)

		tbl.Delete(zero)
	})

	testname = "Update"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid prefix input", testname)
			}
		}(testname)

		tbl.Update(zero, func(v any, _ bool) any { return v })
	})

	testname = "Get"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid prefix input", testname)
			}
		}(testname)

		tbl.Get(zero)
	})

	testname = "LookupPrefix"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid prefix input", testname)
			}
		}(testname)

		tbl.LookupPrefix(zero)
	})

	testname = "LookupPrefixLPM"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid prefix input", testname)
			}
		}(testname)

		tbl.LookupPrefixLPM(zero)
	})

	testname = "OverlapsPrefix"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid prefix input", testname)
			}
		}(testname)

		tbl.OverlapsPrefix(zero)
	})
}

func TestInsert(t *testing.T) {
	t.Parallel()

	tbl := &Table[int]{}

	// Create a new leaf strideTable, with compressed path
	tbl.Insert(mpp("192.168.0.1/32"), 1)
	checkRoutes(t, tbl, []tableTest{
		{"192.168.0.1", 1},
		{"192.168.0.2", -1},
		{"192.168.0.3", -1},
		{"192.168.0.255", -1},
		{"192.168.1.1", -1},
		{"192.170.1.1", -1},
		{"192.180.0.1", -1},
		{"192.180.3.5", -1},
		{"10.0.0.5", -1},
		{"10.0.0.15", -1},
	})

	// Insert into previous leaf, no tree changes
	tbl.Insert(mpp("192.168.0.2/32"), 2)
	checkRoutes(t, tbl, []tableTest{
		{"192.168.0.1", 1},
		{"192.168.0.2", 2},
		{"192.168.0.3", -1},
		{"192.168.0.255", -1},
		{"192.168.1.1", -1},
		{"192.170.1.1", -1},
		{"192.180.0.1", -1},
		{"192.180.3.5", -1},
		{"10.0.0.5", -1},
		{"10.0.0.15", -1},
	})

	// Insert into previous leaf, unaligned prefix covering the /32s
	tbl.Insert(mpp("192.168.0.0/26"), 7)
	checkRoutes(t, tbl, []tableTest{
		{"192.168.0.1", 1},
		{"192.168.0.2", 2},
		{"192.168.0.3", 7},
		{"192.168.0.255", -1},
		{"192.168.1.1", -1},
		{"192.170.1.1", -1},
		{"192.180.0.1", -1},
		{"192.180.3.5", -1},
		{"10.0.0.5", -1},
		{"10.0.0.15", -1},
	})

	// Create a different leaf elsewhere
	tbl.Insert(mpp("10.0.0.0/27"), 3)
	checkRoutes(t, tbl, []tableTest{
		{"192.168.0.1", 1},
		{"192.168.0.2", 2},
		{"192.168.0.3", 7},
		{"192.168.0.255", -1},
		{"192.168.1.1", -1},
		{"192.170.1.1", -1},
		{"192.180.0.1", -1},
		{"192.180.3.5", -1},
		{"10.0.0.5", 3},
		{"10.0.0.15", 3},
	})

	// Insert that creates a new intermediate table and a new child
	tbl.Insert(mpp("192.168.1.1/32"), 4)
	checkRoutes(t, tbl, []tableTest{
		{"192.168.0.1", 1},
		{"192.168.0.2", 2},
		{"192.168.0.3", 7},
		{"192.168.0.255", -1},
		{"192.168.1.1", 4},
		{"192.170.1.1", -1},
		{"192.180.0.1", -1},
		{"192.180.3.5", -1},
		{"10.0.0.5", 3},
		{"10.0.0.15", 3},
	})

	// Insert that creates a new intermediate table but no new child
	tbl.Insert(mpp("192.170.0.0/16"), 5)
	checkRoutes(t, tbl, []tableTest{
		{"192.168.0.1", 1},
		{"192.168.0.2", 2},
		{"192.168.0.3", 7},
		{"192.168.0.255", -1},
		{"192.168.1.1", 4},
		{"192.170.1.1", 5},
		{"192.180.0.1", -1},
		{"192.180.3.5", -1},
		{"10.0.0.5", 3},
		{"10.0.0.15", 3},
	})

	// New leaf in a different subtree, so the next insert can test a
	// variant of decompression.
	tbl.Insert(mpp("192.180.0.1/32"), 8)
	checkRoutes(t, tbl, []tableTest{
		{"192.168.0.1", 1},
		{"192.168.0.2", 2},
		{"192.168.0.3", 7},
		{"192.168.0.255", -1},
		{"192.168.1.1", 4},
		{"192.170.1.1", 5},
		{"192.180.0.1", 8},
		{"192.180.3.5", -1},
		{"10.0.0.5", 3},
		{"10.0.0.15", 3},
	})

	// Insert that creates a new intermediate table but no new child,
	// with an unaligned intermediate
	tbl.Insert(mpp("192.180.0.0/21"), 9)
	checkRoutes(t, tbl, []tableTest{
		{"192.168.0.1", 1},
		{"192.168.0.2", 2},
		{"192.168.0.3", 7},
		{"192.168.0.255", -1},
		{"192.168.1.1", 4},
		{"192.170.1.1", 5},
		{"192.180.0.1", 8},
		{"192.180.3.5", 9},
		{"10.0.0.5", 3},
		{"10.0.0.15", 3},
	})

	// Insert a default route, those have their own codepath.
	tbl.Insert(mpp("0.0.0.0/0"), 6)
	checkRoutes(t, tbl, []tableTest{
		{"192.168.0.1", 1},
		{"192.168.0.2", 2},
		{"192.168.0.3", 7},
		{"192.168.0.255", 6},
		{"192.168.1.1", 4},
		{"192.170.1.1", 5},
		{"192.180.0.1", 8},
		{"192.180.3.5", 9},
		{"10.0.0.5", 3},
		{"10.0.0.15", 3},
	})

	// Now all of the above again, but for IPv6.

	// Create a new leaf strideTable, with compressed path
	tbl.Insert(mpp("ff:aaaa::1/128"), 1)
	checkRoutes(t, tbl, []tableTest{
		{"ff:aaaa::1", 1},
		{"ff:aaaa::2", -1},
		{"ff:aaaa::3", -1},
		{"ff:aaaa::255", -1},
		{"ff:aaaa:aaaa::1", -1},
		{"ff:aaaa:aaaa:bbbb::1", -1},
		{"ff:cccc::1", -1},
		{"ff:cccc::ff", -1},
		{"ffff:bbbb::5", -1},
		{"ffff:bbbb::15", -1},
	})

	// Insert into previous leaf, no tree changes
	tbl.Insert(mpp("ff:aaaa::2/128"), 2)
	checkRoutes(t, tbl, []tableTest{
		{"ff:aaaa::1", 1},
		{"ff:aaaa::2", 2},
		{"ff:aaaa::3", -1},
		{"ff:aaaa::255", -1},
		{"ff:aaaa:aaaa::1", -1},
		{"ff:aaaa:aaaa:bbbb::1", -1},
		{"ff:cccc::1", -1},
		{"ff:cccc::ff", -1},
		{"ffff:bbbb::5", -1},
		{"ffff:bbbb::15", -1},
	})

	// Insert into previous leaf, unaligned prefix covering the /128s
	tbl.Insert(mpp("ff:aaaa::/125"), 7)
	checkRoutes(t, tbl, []tableTest{
		{"ff:aaaa::1", 1},
		{"ff:aaaa::2", 2},
		{"ff:aaaa::3", 7},
		{"ff:aaaa::255", -1},
		{"ff:aaaa:aaaa::1", -1},
		{"ff:aaaa:aaaa:bbbb::1", -1},
		{"ff:cccc::1", -1},
		{"ff:cccc::ff", -1},
		{"ffff:bbbb::5", -1},
		{"ffff:bbbb::15", -1},
	})

	// Create a different leaf elsewhere
	tbl.Insert(mpp("ffff:bbbb::/120"), 3)
	checkRoutes(t, tbl, []tableTest{
		{"ff:aaaa::1", 1},
		{"ff:aaaa::2", 2},
		{"ff:aaaa::3", 7},
		{"ff:aaaa::255", -1},
		{"ff:aaaa:aaaa::1", -1},
		{"ff:aaaa:aaaa:bbbb::1", -1},
		{"ff:cccc::1", -1},
		{"ff:cccc::ff", -1},
		{"ffff:bbbb::5", 3},
		{"ffff:bbbb::15", 3},
	})

	// Insert that creates a new intermediate table and a new child
	tbl.Insert(mpp("ff:aaaa:aaaa::1/128"), 4)
	checkRoutes(t, tbl, []tableTest{
		{"ff:aaaa::1", 1},
		{"ff:aaaa::2", 2},
		{"ff:aaaa::3", 7},
		{"ff:aaaa::255", -1},
		{"ff:aaaa:aaaa::1", 4},
		{"ff:aaaa:aaaa:bbbb::1", -1},
		{"ff:cccc::1", -1},
		{"ff:cccc::ff", -1},
		{"ffff:bbbb::5", 3},
		{"ffff:bbbb::15", 3},
	})

	// Insert that creates a new intermediate table but no new child
	tbl.Insert(mpp("ff:aaaa:aaaa:bb00::/56"), 5)
	checkRoutes(t, tbl, []tableTest{
		{"ff:aaaa::1", 1},
		{"ff:aaaa::2", 2},
		{"ff:aaaa::3", 7},
		{"ff:aaaa::255", -1},
		{"ff:aaaa:aaaa::1", 4},
		{"ff:aaaa:aaaa:bbbb::1", 5},
		{"ff:cccc::1", -1},
		{"ff:cccc::ff", -1},
		{"ffff:bbbb::5", 3},
		{"ffff:bbbb::15", 3},
	})

	// New leaf in a different subtree, so the next insert can test a
	// variant of decompression.
	tbl.Insert(mpp("ff:cccc::1/128"), 8)
	checkRoutes(t, tbl, []tableTest{
		{"ff:aaaa::1", 1},
		{"ff:aaaa::2", 2},
		{"ff:aaaa::3", 7},
		{"ff:aaaa::255", -1},
		{"ff:aaaa:aaaa::1", 4},
		{"ff:aaaa:aaaa:bbbb::1", 5},
		{"ff:cccc::1", 8},
		{"ff:cccc::ff", -1},
		{"ffff:bbbb::5", 3},
		{"ffff:bbbb::15", 3},
	})

	// Insert that creates a new intermediate table but no new child,
	// with an unaligned intermediate
	tbl.Insert(mpp("ff:cccc::/37"), 9)
	checkRoutes(t, tbl, []tableTest{
		{"ff:aaaa::1", 1},
		{"ff:aaaa::2", 2},
		{"ff:aaaa::3", 7},
		{"ff:aaaa::255", -1},
		{"ff:aaaa:aaaa::1", 4},
		{"ff:aaaa:aaaa:bbbb::1", 5},
		{"ff:cccc::1", 8},
		{"ff:cccc::ff", 9},
		{"ffff:bbbb::5", 3},
		{"ffff:bbbb::15", 3},
	})

	// Insert a default route, those have their own codepath.
	tbl.Insert(mpp("::/0"), 6)
	checkRoutes(t, tbl, []tableTest{
		{"ff:aaaa::1", 1},
		{"ff:aaaa::2", 2},
		{"ff:aaaa::3", 7},
		{"ff:aaaa::255", 6},
		{"ff:aaaa:aaaa::1", 4},
		{"ff:aaaa:aaaa:bbbb::1", 5},
		{"ff:cccc::1", 8},
		{"ff:cccc::ff", 9},
		{"ffff:bbbb::5", 3},
		{"ffff:bbbb::15", 3},
	})
}

func TestDelete(t *testing.T) {
	t.Parallel()

	t.Run("table_is_empty", func(t *testing.T) {
		// must not panic
		t.Parallel()
		rtbl := &Table[int]{}
		rtbl.Delete(randomPrefix())
	})

	t.Run("prefix_in_root", func(t *testing.T) {
		t.Parallel()
		// Add/remove prefix from root table.
		rtbl := &Table[int]{}
		checkNumNodes(t, rtbl, 0)

		rtbl.Insert(mpp("10.0.0.0/8"), 1)
		checkRoutes(t, rtbl, []tableTest{
			{"10.0.0.1", 1},
			{"255.255.255.255", -1},
		})
		checkNumNodes(t, rtbl, 1)
		rtbl.Delete(mpp("10.0.0.0/8"))
		checkRoutes(t, rtbl, []tableTest{
			{"10.0.0.1", -1},
			{"255.255.255.255", -1},
		})
		checkNumNodes(t, rtbl, 0)
	})

	t.Run("prefix_in_leaf", func(t *testing.T) {
		t.Parallel()
		// Create, then delete a single leaf table.
		rtbl := &Table[int]{}
		checkNumNodes(t, rtbl, 0)

		rtbl.Insert(mpp("192.168.0.1/32"), 1)
		checkRoutes(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"255.255.255.255", -1},
		})
		rtbl.Delete(mpp("192.168.0.1/32"))
		checkRoutes(t, rtbl, []tableTest{
			{"192.168.0.1", -1},
			{"255.255.255.255", -1},
		})
		checkNumNodes(t, rtbl, 0)
	})

	t.Run("intermediate_no_routes", func(t *testing.T) {
		t.Parallel()
		// Create an intermediate with 2 children, then delete one leaf.
		tbl := &Table[int]{}
		checkNumNodes(t, tbl, 0)
		tbl.Insert(mpp("192.168.0.1/32"), 1)
		tbl.Insert(mpp("192.180.0.1/32"), 2)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", 2},
			{"192.40.0.1", -1},
		})
		checkNumNodes(t, tbl, 6) // 1 root4, 3 intermediate, 2 leaves
		tbl.Delete(mpp("192.180.0.1/32"))
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", -1},
			{"192.40.0.1", -1},
		})
		checkNumNodes(t, tbl, 4) // 1 root4, 2 intermediates, 1 leaf
	})

	t.Run("intermediate_with_route", func(t *testing.T) {
		t.Parallel()
		// Same, but the intermediate carries a route as well.
		rtbl := &Table[int]{}
		checkNumNodes(t, rtbl, 0)
		rtbl.Insert(mpp("192.168.0.1/32"), 1)
		rtbl.Insert(mpp("192.180.0.1/32"), 2)
		rtbl.Insert(mpp("192.0.0.0/10"), 3)
		checkRoutes(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", 2},
			{"192.40.0.1", 3},
			{"192.255.0.1", -1},
		})
		checkNumNodes(t, rtbl, 6) // 1 root4, 2 intermediates, 2 leaves
		rtbl.Delete(mpp("192.180.0.1/32"))
		checkRoutes(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", -1},
			{"192.40.0.1", 3},
			{"192.255.0.1", -1},
		})
		checkNumNodes(t, rtbl, 4) // 1 root4, 1 full, 1 intermediate, 1 leaf
	})

	t.Run("intermediate_many_leaves", func(t *testing.T) {
		t.Parallel()
		// Intermediate with 3 leaves, then delete one leaf.
		rtbl := &Table[int]{}
		checkNumNodes(t, rtbl, 0)
		rtbl.Insert(mpp("192.168.0.1/32"), 1)
		rtbl.Insert(mpp("192.180.0.1/32"), 2)
		rtbl.Insert(mpp("192.200.0.1/32"), 3)
		checkRoutes(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", 2},
			{"192.200.0.1", 3},
			{"192.255.0.1", -1},
		})
		checkNumNodes(t, rtbl, 8) // 1 root4, 4 intermediate, 3 leaves
		rtbl.Delete(mpp("192.180.0.1/32"))
		checkRoutes(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", -1},
			{"192.200.0.1", 3},
			{"192.255.0.1", -1},
		})
		checkNumNodes(t, rtbl, 6) // 1 root4, 3 intermediate, 2 leaves
	})

	t.Run("nosuchprefix_missing_child", func(t *testing.T) {
		t.Parallel()
		// Delete non-existent prefix, missing strideTable path.
		rtbl := &Table[int]{}
		checkNumNodes(t, rtbl, 0)
		rtbl.Insert(mpp("192.168.0.1/32"), 1)
		checkRoutes(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.255.0.1", -1},
		})
		checkNumNodes(t, rtbl, 4)        // 1 root4, 2 intermediate, 1 leaf
		rtbl.Delete(mpp("200.0.0.0/32")) // lookup miss in root
		checkRoutes(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.255.0.1", -1},
		})
		checkNumNodes(t, rtbl, 4) // 1 root4, 2 intermediate, 1 leaf
	})

	t.Run("nosuchprefix_not_in_leaf", func(t *testing.T) {
		t.Parallel()
		// Delete non-existent prefix, strideTable path exists but
		// leaf doesn't contain route.
		rtbl := &Table[int]{}
		checkNumNodes(t, rtbl, 0)
		rtbl.Insert(mpp("192.168.0.1/32"), 1)
		checkRoutes(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.255.0.1", -1},
		})
		checkNumNodes(t, rtbl, 4)          // 1 root4, 2 intermediate, 1 leaf
		rtbl.Delete(mpp("192.168.0.5/32")) // right leaf, no route
		checkRoutes(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.255.0.1", -1},
		})
		checkNumNodes(t, rtbl, 4) // 1 root4, 2 intermediate, 1 leaf
	})

	t.Run("intermediate_with_deleted_route", func(t *testing.T) {
		t.Parallel()
		// Intermediate table loses its last route and becomes
		// compactable.
		rtbl := &Table[int]{}
		checkNumNodes(t, rtbl, 0)
		rtbl.Insert(mpp("192.168.0.1/32"), 1)
		rtbl.Insert(mpp("192.168.0.0/22"), 2)
		checkRoutes(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.168.0.2", 2},
			{"192.255.0.1", -1},
		})
		checkNumNodes(t, rtbl, 4) // 1 root4, 1 intermediate, 1 full, 1 leaf
		rtbl.Delete(mpp("192.168.0.0/22"))
		checkRoutes(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.168.0.2", -1},
			{"192.255.0.1", -1},
		})
		checkNumNodes(t, rtbl, 4) // 1 root4, 2 intermediate, 1 leaf
	})

	t.Run("default_route", func(t *testing.T) {
		t.Parallel()
		// Default routes have a special case in the code.
		rtbl := &Table[int]{}

		rtbl.Insert(mpp("0.0.0.0/0"), 1)
		rtbl.Insert(mpp("::/0"), 1)
		rtbl.Delete(mpp("0.0.0.0/0"))

		checkRoutes(t, rtbl, []tableTest{
			{"1.2.3.4", -1},
			{"::1", 1},
		})
		checkNumNodes(t, rtbl, 1) // 1 root6
	})
}

func TestContainsCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()
	pfxs := randomPrefixes(10_000)

	gold := goldTable[int](pfxs)
	fast := Table[int]{}

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	for range 10_000 {
		a := randomAddr()

		_, goldOK := gold.lookup(a)
		fastOK := fast.Contains(a)

		if goldOK != fastOK {
			t.Fatalf("Contains(%q) = %v, want %v", a, fastOK, goldOK)
		}
	}
}

func TestLookupCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()
	pfxs := randomPrefixes(10_000)

	gold := goldTable[int](pfxs)
	fast := Table[int]{}

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	seenVals4 := map[int]bool{}
	seenVals6 := map[int]bool{}

	for range 10_000 {
		a := randomAddr()

		goldVal, goldOK := gold.lookup(a)
		fastVal, fastOK := fast.Lookup(a)

		if !getsEqual(goldVal, goldOK, fastVal, fastOK) {
			t.Fatalf("Lookup(%q) = (%v, %v), want (%v, %v)", a, fastVal, fastOK, goldVal, goldOK)
		}

		if a.Is6() {
			seenVals6[fastVal] = true
		} else {
			seenVals4[fastVal] = true
		}
	}

	// Empirically, 10k probes into 5k v4 prefixes and 5k v6 prefixes results in
	// ~1k distinct values for v4 and ~300 for v6. distinct routes. This sanity
	// check that we didn't just return a single route for everything should be
	// very generous indeed.
	if cnt := len(seenVals4); cnt < 10 {
		t.Fatalf("saw %d distinct v4 route results, statistically expected ~1000", cnt)
	}
	if cnt := len(seenVals6); cnt < 10 {
		t.Fatalf("saw %d distinct v6 route results, statistically expected ~300", cnt)
	}
}

func TestLookupPrefixCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()
	pfxs := randomPrefixes(10_000)

	fast := Table[int]{}
	gold := goldTable[int](pfxs)

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	seenVals4 := map[int]bool{}
	seenVals6 := map[int]bool{}

	for range 10_000 {
		pfx := randomPrefix()

		goldVal, goldOK := gold.lookupPfx(pfx)
		fastVal, fastOK := fast.LookupPrefix(pfx)

		if !getsEqual(goldVal, goldOK, fastVal, fastOK) {
			t.Fatalf("LookupPrefix(%q) = (%v, %v), want (%v, %v)", pfx, fastVal, fastOK, goldVal, goldOK)
		}

		if pfx.Addr().Is6() {
			seenVals6[fastVal] = true
		} else {
			seenVals4[fastVal] = true
		}
	}

	// Empirically, 10k probes into 5k v4 prefixes and 5k v6 prefixes results in
	// ~1k distinct values for v4 and ~300 for v6. distinct routes. This sanity
	// check that we didn't just return a single route for everything should be
	// very generous indeed.
	if cnt := len(seenVals4); cnt < 10 {
		t.Fatalf("saw %d distinct v4 route results, statistically expected ~1000", cnt)
	}
	if cnt := len(seenVals6); cnt < 10 {
		t.Fatalf("saw %d distinct v6 route results, statistically expected ~300", cnt)
	}
}

func TestLookupPrefixLPMCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()
	pfxs := randomPrefixes(10_000)

	fast := Table[int]{}
	gold := goldTable[int](pfxs)

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	seenVals4 := map[int]bool{}
	seenVals6 := map[int]bool{}

	for range 10_000 {
		pfx := randomPrefix()

		goldLPM, goldVal, goldOK := gold.lookupPfxLPM(pfx)
		fastLPM, fastVal, fastOK := fast.LookupPrefixLPM(pfx)

		if !getsEqual(goldVal, goldOK, fastVal, fastOK) {
			t.Fatalf("LookupPrefixLPM(%q) = (%v, %v), want (%v, %v)", pfx, fastVal, fastOK, goldVal, goldOK)
		}

		if !getsEqual(goldLPM, goldOK, fastLPM, fastOK) {
			t.Fatalf("LookupPrefixLPM(%q) = (%v, %v), want (%v, %v)", pfx, fastLPM, fastOK, goldLPM, goldOK)
		}

		if pfx.Addr().Is6() {
			seenVals6[fastVal] = true
		} else {
			seenVals4[fastVal] = true
		}
	}

	// Empirically, 10k probes into 5k v4 prefixes and 5k v6 prefixes results in
	// ~1k distinct values for v4 and ~300 for v6. distinct routes. This sanity
	// check that we didn't just return a single route for everything should be
	// very generous indeed.
	if cnt := len(seenVals4); cnt < 10 {
		t.Fatalf("saw %d distinct v4 route results, statistically expected ~1000", cnt)
	}
	if cnt := len(seenVals6); cnt < 10 {
		t.Fatalf("saw %d distinct v6 route results, statistically expected ~300", cnt)
	}
}

func TestInsertShuffled(t *testing.T) {
	// The order in which you insert prefixes into a route table
	// should not matter, as long as you're inserting the same set of
	// routes.
	t.Parallel()

	pfxs := randomPrefixes(1000)

	for range 10 {
		pfxs2 := append([]goldTableItem[int](nil), pfxs...)
		rand.Shuffle(len(pfxs2), func(i, j int) { pfxs2[i], pfxs2[j] = pfxs2[j], pfxs2[i] })

		addrs := make([]netip.Addr, 0, 10_000)
		for range 10_000 {
			addrs = append(addrs, randomAddr())
		}

		rt1 := Table[int]{}
		rt2 := Table[int]{}

		for _, pfx := range pfxs {
			rt1.Insert(pfx.pfx, pfx.val)
		}
		for _, pfx := range pfxs2 {
			rt2.Insert(pfx.pfx, pfx.val)
		}

		for _, a := range addrs {
			val1, ok1 := rt1.Lookup(a)
			val2, ok2 := rt2.Lookup(a)

			if !getsEqual(val1, ok1, val2, ok2) {
				t.Fatalf("Lookup(%q) = (%v, %v), want (%v, %v)", a, val2, ok2, val1, ok1)
			}
		}
	}
}

func TestDeleteCompare(t *testing.T) {
	// Create large route tables repeatedly, delete half of their
	// prefixes, and compare Table's behavior to a naive and slow but
	// correct implementation.
	t.Parallel()

	const (
		numPrefixes  = 10_000 // total prefixes to insert (test deletes 50% of them)
		numPerFamily = numPrefixes / 2
		deleteCut    = numPerFamily / 2
		numProbes    = 10_000 // random addr lookups to do
	)

	// We have to do this little dance instead of just using allPrefixes,
	// because we want pfxs and toDelete to be non-overlapping sets.
	all4, all6 := randomPrefixes4(numPerFamily), randomPrefixes6(numPerFamily)

	pfxs := append([]goldTableItem[int](nil), all4[:deleteCut]...)
	pfxs = append(pfxs, all6[:deleteCut]...)

	toDelete := append([]goldTableItem[int](nil), all4[deleteCut:]...)
	toDelete = append(toDelete, all6[deleteCut:]...)

	fast := Table[int]{}
	gold := goldTable[int](pfxs)

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	for _, pfx := range toDelete {
		fast.Insert(pfx.pfx, pfx.val)
	}
	for _, pfx := range toDelete {
		fast.Delete(pfx.pfx)
	}

	seenVals4 := map[int]bool{}
	seenVals6 := map[int]bool{}

	for range numProbes {
		a := randomAddr()

		goldVal, goldOK := gold.lookup(a)
		fastVal, fastOK := fast.Lookup(a)

		if !getsEqual(goldVal, goldOK, fastVal, fastOK) {
			t.Fatalf("Lookup(%q) = (%v, %v), want (%v, %v)", a, fastVal, fastOK, goldVal, goldOK)
		}

		if a.Is6() {
			seenVals6[fastVal] = true
		} else {
			seenVals4[fastVal] = true
		}
	}
	// Empirically, 10k probes into 5k v4 prefixes and 5k v6 prefixes results in
	// ~1k distinct values for v4 and ~300 for v6. distinct routes. This sanity
	// check that we didn't just return a single route for everything should be
	// very generous indeed.
	if cnt := len(seenVals4); cnt < 10 {
		t.Fatalf("saw %d distinct v4 route results, statistically expected ~1000", cnt)
	}
	if cnt := len(seenVals6); cnt < 10 {
		t.Fatalf("saw %d distinct v6 route results, statistically expected ~300", cnt)
	}
}

func TestDeleteShuffled(t *testing.T) {
	// The order in which you delete prefixes from a route table
	// should not matter, as long as you're deleting the same set of
	// routes.
	t.Parallel()

	const (
		numPrefixes  = 10_000 // prefixes to insert (test deletes 50% of them)
		numPerFamily = numPrefixes / 2
		deleteCut    = numPerFamily / 2
		numProbes    = 10_000 // random addr lookups to do
	)

	// We have to do this little dance instead of just using allPrefixes,
	// because we want pfxs and toDelete to be non-overlapping sets.
	all4, all6 := randomPrefixes4(numPerFamily), randomPrefixes6(numPerFamily)

	pfxs := append([]goldTableItem[int](nil), all4[:deleteCut]...)
	pfxs = append(pfxs, all6[:deleteCut]...)

	toDelete := append([]goldTableItem[int](nil), all4[deleteCut:]...)
	toDelete = append(toDelete, all6[deleteCut:]...)

	rt1 := Table[int]{}
	for _, pfx := range pfxs {
		rt1.Insert(pfx.pfx, pfx.val)
	}
	for _, pfx := range toDelete {
		rt1.Insert(pfx.pfx, pfx.val)
	}
	for _, pfx := range toDelete {
		rt1.Delete(pfx.pfx)
	}

	for range 10 {
		pfxs2 := append([]goldTableItem[int](nil), pfxs...)
		toDelete2 := append([]goldTableItem[int](nil), toDelete...)
		rand.Shuffle(len(toDelete2), func(i, j int) { toDelete2[i], toDelete2[j] = toDelete2[j], toDelete2[i] })
		rt2 := Table[int]{}
		for _, pfx := range pfxs2 {
			rt2.Insert(pfx.pfx, pfx.val)
		}
		for _, pfx := range toDelete2 {
			rt2.Insert(pfx.pfx, pfx.val)
		}
		for _, pfx := range toDelete2 {
			rt2.Delete(pfx.pfx)
		}

		// Diffing a deep tree of tables gives cmp.Diff a nervous breakdown, so
		// test for equivalence statistically with random probes instead.
		for range numProbes {
			a := randomAddr()
			val1, ok1 := rt1.Lookup(a)
			val2, ok2 := rt2.Lookup(a)
			if !getsEqual(val1, ok1, val2, ok2) {
				t.Errorf("Lookup(%q) = (%v, %v), want (%v, %v)", a, val2, ok2, val1, ok1)
			}
		}
	}
}

func TestDeleteIsReverseOfInsert(t *testing.T) {
	t.Parallel()
	// Insert N prefixes, then delete those same prefixes in reverse
	// order. Each deletion should exactly undo the internal structure
	// changes that each insert did.
	const N = 10_000

	tbl := new(Table[int])
	want := tbl.dumpString()

	prefixes := randomPrefixes(N)

	defer func() {
		if t.Failed() {
			t.Logf("the prefixes that fail the test: %v\n", prefixes)
		}
	}()

	for _, p := range prefixes {
		tbl.Insert(p.pfx, p.val)
	}

	for i := len(prefixes) - 1; i >= 0; i-- {
		tbl.Delete(prefixes[i].pfx)
	}
	if got := tbl.dumpString(); got != want {
		t.Fatalf("after delete, mismatch:\n\n got: %s\n\nwant: %s", got, want)
	}
}

func TestGetAndDelete(t *testing.T) {
	t.Parallel()
	// Insert N prefixes, then delete those same prefixes in shuffled
	// order.
	const N = 10_000

	tbl := new(Table[int])
	prefixes := randomPrefixes(N)

	// insert the prefixes
	for _, p := range prefixes {
		tbl.Insert(p.pfx, p.val)
	}

	// shuffle the prefixes
	rand.Shuffle(N, func(i, j int) {
		prefixes[i], prefixes[j] = prefixes[j], prefixes[i]
	})

	for _, p := range prefixes {
		want, _ := tbl.Get(p.pfx)
		val, ok := tbl.GetAndDelete(p.pfx)

		if !ok {
			t.Errorf("GetAndDelete, expected true, got %v", ok)
		}

		if val != want {
			t.Errorf("GetAndDelete, expected %v, got %v", want, val)
		}

		val, ok = tbl.GetAndDelete(p.pfx)
		if ok {
			t.Errorf("GetAndDelete, expected false, got (%v, %v)", val, ok)
		}
	}
}

func TestGet(t *testing.T) {
	t.Parallel()

	t.Run("empty table", func(t *testing.T) {
		t.Parallel()

		rt := new(Table[int])
		pfx := randomPrefix()
		_, ok := rt.Get(pfx)

		if ok {
			t.Errorf("empty table: Get(%v), ok=%v, expected: %v", pfx, ok, false)
		}
	})

	tests := []struct {
		name string
		pfx  netip.Prefix
		val  int
	}{
		{
			name: "default route v4",
			pfx:  mpp("0.0.0.0/0"),
			val:  0,
		},
		{
			name: "default route v6",
			pfx:  mpp("::/0"),
			val:  0,
		},
		{
			name: "set v4",
			pfx:  mpp("1.2.3.4/32"),
			val:  1234,
		},
		{
			name: "set v6",
			pfx:  mpp("2001:db8::/32"),
			val:  2001,
		},
	}

	rt := new(Table[int])
	for _, tt := range tests {
		rt.Insert(tt.pfx, tt.val)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := rt.Get(tt.pfx)

			if !ok {
				t.Errorf("%s: ok=%v, expected: %v", tt.name, ok, true)
			}

			if got != tt.val {
				t.Errorf("%s: val=%v, expected: %v", tt.name, got, tt.val)
			}
		})
	}
}

func TestGetCompare(t *testing.T) {
	t.Parallel()

	pfxs := randomPrefixes(10_000)
	fast := Table[int]{}
	gold := goldTable[int](pfxs)

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	for _, pfx := range pfxs {
		goldVal, goldOK := gold.get(pfx.pfx)
		fastVal, fastOK := fast.Get(pfx.pfx)

		if !getsEqual(goldVal, goldOK, fastVal, fastOK) {
			t.Fatalf("Get(%q) = (%v, %v), want (%v, %v)", pfx.pfx, fastVal, fastOK, goldVal, goldOK)
		}
	}
}

func TestUpdateCompare(t *testing.T) {
	t.Parallel()

	pfxs := randomPrefixes(10_000)
	fast := Table[int]{}
	gold := goldTable[int](pfxs)

	// Update as insert
	for _, pfx := range pfxs {
		fast.Update(pfx.pfx, func(int, bool) int { return pfx.val })
	}

	for _, pfx := range pfxs {
		goldVal, goldOK := gold.get(pfx.pfx)
		fastVal, fastOK := fast.Get(pfx.pfx)

		if !getsEqual(goldVal, goldOK, fastVal, fastOK) {
			t.Fatalf("Get(%q) = (%v, %v), want (%v, %v)", pfx.pfx, fastVal, fastOK, goldVal, goldOK)
		}
	}

	cb := func(val int, _ bool) int { return val + 1 }

	// Update as update
	for _, pfx := range pfxs[:len(pfxs)/2] {
		gold.update(pfx.pfx, cb)
		fast.Update(pfx.pfx, cb)
	}

	for _, pfx := range pfxs {
		goldVal, goldOK := gold.get(pfx.pfx)
		fastVal, fastOK := fast.Get(pfx.pfx)

		if !getsEqual(goldVal, goldOK, fastVal, fastOK) {
			t.Fatalf("Get(%q) = (%v, %v), want (%v, %v)", pfx.pfx, fastVal, fastOK, goldVal, goldOK)
		}
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		pfx  netip.Prefix
	}{
		{
			name: "default route v4",
			pfx:  mpp("0.0.0.0/0"),
		},
		{
			name: "default route v6",
			pfx:  mpp("::/0"),
		},
		{
			name: "set v4",
			pfx:  mpp("1.2.3.4/32"),
		},
		{
			name: "set v6",
			pfx:  mpp("2001:db8::/32"),
		},
	}

	rt := new(Table[int])

	// just increment val
	cb := func(val int, ok bool) int {
		if ok {
			return val + 1
		}
		return 0
	}

	// update as insert
	for _, tt := range tests {
		t.Run(fmt.Sprintf("insert: %s", tt.name), func(t *testing.T) {
			val := rt.Update(tt.pfx, cb)
			got, ok := rt.Get(tt.pfx)

			if !ok {
				t.Errorf("%s: ok=%v, expected: %v", tt.name, ok, true)
			}

			if got != 0 || got != val {
				t.Errorf("%s: got=%v, expected: %v", tt.name, got, 0)
			}
		})
	}

	// update as update
	for _, tt := range tests {
		t.Run(fmt.Sprintf("update: %s", tt.name), func(t *testing.T) {
			val := rt.Update(tt.pfx, cb)
			got, ok := rt.Get(tt.pfx)

			if !ok {
				t.Errorf("%s: ok=%v, expected: %v", tt.name, ok, true)
			}

			if got != 1 || got != val {
				t.Errorf("%s: got=%v, expected: %v", tt.name, got, 1)
			}
		})
	}
}

func TestUnionEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		aTbl := &Table[int]{}
		bTbl := &Table[int]{}

		// union empty tables
		aTbl.Union(bTbl)

		want := ""
		got := aTbl.String()
		if got != want {
			t.Fatalf("got:\n%v\nwant:\n%v", got, want)
		}
	})

	t.Run("other empty", func(t *testing.T) {
		t.Parallel()
		aTbl := &Table[int]{}
		bTbl := &Table[int]{}

		// one empty table, b
		aTbl.Insert(mpp("0.0.0.0/0"), 0)

		aTbl.Union(bTbl)
		want := `▼
└─ 0.0.0.0/0 (0)
`
		got := aTbl.String()
		if got != want {
			t.Fatalf("got:\n%v\nwant:\n%v", got, want)
		}
	})

	t.Run("other empty", func(t *testing.T) {
		t.Parallel()
		aTbl := &Table[int]{}
		bTbl := &Table[int]{}

		// one empty table, a
		bTbl.Insert(mpp("0.0.0.0/0"), 0)

		aTbl.Union(bTbl)
		want := `▼
└─ 0.0.0.0/0 (0)
`
		got := aTbl.String()
		if got != want {
			t.Fatalf("got:\n%v\nwant:\n%v", got, want)
		}
	})

	t.Run("duplicate prefix", func(t *testing.T) {
		t.Parallel()
		aTbl := &Table[string]{}
		bTbl := &Table[string]{}

		// one empty table
		aTbl.Insert(mpp("::/0"), "orig value")
		bTbl.Insert(mpp("::/0"), "overwrite")

		aTbl.Union(bTbl)
		want := `▼
└─ ::/0 (overwrite)
`
		got := aTbl.String()
		if got != want {
			t.Fatalf("got:\n%v\nwant:\n%v", got, want)
		}
	})

	t.Run("different IP versions", func(t *testing.T) {
		t.Parallel()
		aTbl := &Table[int]{}
		bTbl := &Table[int]{}

		// one empty table
		aTbl.Insert(mpp("0.0.0.0/0"), 1)
		bTbl.Insert(mpp("::/0"), 2)

		aTbl.Union(bTbl)
		want := `▼
└─ 0.0.0.0/0 (1)
▼
└─ ::/0 (2)
`
		got := aTbl.String()
		if got != want {
			t.Fatalf("got:\n%v\nwant:\n%v", got, want)
		}
	})

	t.Run("same children", func(t *testing.T) {
		t.Parallel()
		aTbl := &Table[int]{}
		bTbl := &Table[int]{}

		aTbl.Insert(mpp("127.0.0.1/32"), 1)
		aTbl.Insert(mpp("::1/128"), 1)

		bTbl.Insert(mpp("127.0.0.2/32"), 2)
		bTbl.Insert(mpp("::2/128"), 2)

		aTbl.Union(bTbl)
		want := `▼
├─ 127.0.0.1/32 (1)
└─ 127.0.0.2/32 (2)
▼
├─ ::1/128 (1)
└─ ::2/128 (2)
`
		got := aTbl.String()
		if got != want {
			t.Fatalf("got:\n%v\nwant:\n%v", got, want)
		}
	})
}

// TestUnionMemoryAliasing tests that the Union method does not alias memory
// between the two tables.
func TestUnionMemoryAliasing(t *testing.T) {
	t.Parallel()

	newTable := func(pfx ...string) *Table[struct{}] {
		var t Table[struct{}]
		for _, s := range pfx {
			t.Insert(mpp(s), struct{}{})
		}
		return &t
	}
	// First create two tables with disjoint prefixes.
	stable := newTable("0.0.0.0/24")
	temp := newTable("100.69.1.0/24")

	// Verify that the tables are disjoint.
	if stable.Overlaps(temp) {
		t.Error("stable should not overlap temp")
	}

	// Now union them.
	temp.Union(stable)

	// Add a new prefix to temp.
	temp.Insert(mpp("0.0.1.0/24"), struct{}{})

	// Ensure that stable is unchanged.
	_, ok := stable.Lookup(mpa("0.0.1.1"))
	if ok {
		t.Error("stable should not contain 0.0.1.1")
	}
	if stable.OverlapsPrefix(mpp("0.0.1.1/32")) {
		t.Error("stable should not overlap 0.0.1.1/32")
	}
}

func TestUnionCompare(t *testing.T) {
	t.Parallel()

	const numEntries = 200

	for range 100 {
		pfxs := randomPrefixes(numEntries)
		fast := Table[int]{}
		gold := goldTable[int](pfxs)

		for _, pfx := range pfxs {
			fast.Insert(pfx.pfx, pfx.val)
		}

		pfxs2 := randomPrefixes(numEntries)
		gold2 := goldTable[int](pfxs2)
		fast2 := Table[int]{}
		for _, pfx := range pfxs2 {
			fast2.Insert(pfx.pfx, pfx.val)
		}

		gold.union(&gold2)
		fast.Union(&fast2)

		// dump as slow table for comparison
		fastAsGoldenTbl := fast.dumpAsGoldTable()

		// sort for comparison
		gold.sort()
		fastAsGoldenTbl.sort()

		for i := range gold {
			goldItem := gold[i]
			fastItem := fastAsGoldenTbl[i]
			if goldItem != fastItem {
				t.Fatalf("Union(...): items[%d] differ slow(%v) != fast(%v)", i, goldItem, fastItem)
			}
		}

		// check the size
		if fast.Size() != len(gold) {
			t.Errorf("sizes differ, got: %d, want: %d", fast.Size(), len(gold))
		}
	}
}

func TestCloneEdgeCases(t *testing.T) {
	t.Parallel()

	tbl := new(Table[int])
	clone := tbl.Clone()
	if tbl.String() != clone.String() {
		t.Errorf("empty Clone: got:\n%swant:\n%s", clone.String(), tbl.String())
	}

	tbl.Insert(mpp("10.0.0.1/32"), 1)
	tbl.Insert(mpp("::1/128"), 1)
	clone = tbl.Clone()
	if tbl.String() != clone.String() {
		t.Errorf("Clone: got:\n%swant:\n%s", clone.String(), tbl.String())
	}

	// overwrite value
	tbl.Insert(mpp("::1/128"), 2)
	if tbl.String() == clone.String() {
		t.Errorf("overwrite, clone must be different: clone:\n%sorig:\n%s", clone.String(), tbl.String())
	}

	tbl.Delete(mpp("10.0.0.1/32"))
	if tbl.String() == clone.String() {
		t.Errorf("delete, clone must be different: clone:\n%sorig:\n%s", clone.String(), tbl.String())
	}
}

func TestClone(t *testing.T) {
	t.Parallel()

	pfxs := randomPrefixes(1_000)

	golden := new(Table[int])
	tbl := new(Table[int])
	for _, pfx := range pfxs {
		golden.Insert(pfx.pfx, pfx.val)
		tbl.Insert(pfx.pfx, pfx.val)
	}
	clone := tbl.Clone()

	if tbl.dumpString() != clone.dumpString() {
		t.Errorf("Clone: got:\n%swant:\n%s", clone.dumpString(), tbl.dumpString())
	}
}

func TestCloneShallow(t *testing.T) {
	t.Parallel()

	tbl := new(Table[*int])
	clone := tbl.Clone()
	if tbl.dumpString() != clone.dumpString() {
		t.Errorf("empty Clone: got:\n%swant:\n%s", clone.String(), tbl.String())
	}

	val := 1
	pfx := mpp("10.0.0.1/32")
	tbl.Insert(pfx, &val)

	clone = tbl.Clone()
	want, _ := tbl.Get(pfx)
	got, _ := clone.Get(pfx)

	if !(*got == *want && got == want) {
		t.Errorf("shallow copy, values and pointers must be equal:\nvalues(%d, %d)\n(ptr(%v, %v)", *got, *want, got, want)
	}

	// update value, shallow copy of values, clone must be equal
	val = 2
	want, _ = tbl.Get(pfx)
	got, _ = clone.Get(pfx)

	if *got != *want {
		t.Errorf("memory aliasing after shallow copy, values must be equal:\nvalues(%d, %d)", *got, *want)
	}
}

// tests for deep copies with Cloner interface
type MyInt int

// implement the Cloner interface
func (i *MyInt) Clone() *MyInt {
	a := *i
	return &a
}

func TestCloneDeep(t *testing.T) {
	t.Parallel()

	tbl := new(Table[*MyInt])
	clone := tbl.Clone()
	if tbl.String() != clone.String() {
		t.Errorf("empty Clone: got:\n%swant:\n%s", clone.String(), tbl.String())
	}

	val := MyInt(1)
	pfx := mpp("10.0.0.1/32")
	tbl.Insert(pfx, &val)

	clone = tbl.Clone()
	want, _ := tbl.Get(pfx)
	got, _ := clone.Get(pfx)

	if *got != *want || got == want {
		t.Errorf("deep copy, values must be equal, pointers must be different:\nvalues(%d, %d)\n(ptr(%v, %v)", *got, *want, got, want)
	}

	// update value, deep copy of values, cloned value must now be different
	val = 2
	want, _ = tbl.Get(pfx)
	got, _ = clone.Get(pfx)

	if *got == *want {
		t.Errorf("memory aliasing after deep copy, values must be different:\nvalues(%d, %d)", *got, *want)
	}
}

// test some edge cases
func TestOverlapsPrefixEdgeCases(t *testing.T) {
	t.Parallel()

	tbl := &Table[int]{}

	// empty table
	checkOverlaps(t, tbl, []tableOverlapsTest{
		{"0.0.0.0/0", false},
		{"::/0", false},
	})

	// default route
	tbl.Insert(mpp("10.0.0.0/8"), 0)
	tbl.Insert(mpp("2001:db8::/32"), 0)
	checkOverlaps(t, tbl, []tableOverlapsTest{
		{"0.0.0.0/0", true},
		{"::/0", true},
	})

	// default route
	tbl = &Table[int]{}
	tbl.Insert(mpp("0.0.0.0/0"), 0)
	tbl.Insert(mpp("::/0"), 0)
	checkOverlaps(t, tbl, []tableOverlapsTest{
		{"10.0.0.0/8", true},
		{"2001:db8::/32", true},
	})

	// single IP
	tbl = &Table[int]{}
	tbl.Insert(mpp("10.0.0.0/7"), 0)
	tbl.Insert(mpp("2001::/16"), 0)
	checkOverlaps(t, tbl, []tableOverlapsTest{
		{"10.1.2.3/32", true},
		{"2001:db8:affe::cafe/128", true},
	})

	// single IPv
	tbl = &Table[int]{}
	tbl.Insert(mpp("10.1.2.3/32"), 0)
	tbl.Insert(mpp("2001:db8:affe::cafe/128"), 0)
	checkOverlaps(t, tbl, []tableOverlapsTest{
		{"10.0.0.0/7", true},
		{"2001::/16", true},
	})

	// same IPv
	tbl = &Table[int]{}
	tbl.Insert(mpp("10.1.2.3/32"), 0)
	tbl.Insert(mpp("2001:db8:affe::cafe/128"), 0)
	checkOverlaps(t, tbl, []tableOverlapsTest{
		{"10.1.2.3/32", true},
		{"2001:db8:affe::cafe/128", true},
	})
}

func TestSize(t *testing.T) {
	t.Parallel()

	rtbl := new(Table[any])
	if rtbl.Size() != 0 {
		t.Errorf("empty Table: want: 0, got: %d", rtbl.Size())
	}

	if rtbl.Size4() != 0 {
		t.Errorf("empty Table: want: 0, got: %d", rtbl.Size4())
	}

	if rtbl.Size6() != 0 {
		t.Errorf("empty Table: want: 0, got: %d", rtbl.Size6())
	}

	pfxs1 := randomPrefixes(10_000)
	pfxs2 := randomPrefixes(10_000)

	for _, pfx := range pfxs1 {
		rtbl.Insert(pfx.pfx, nil)
	}

	for _, pfx := range pfxs2 {
		rtbl.Update(pfx.pfx, func(any, bool) any { return nil })
	}

	pfxs1 = append(pfxs1, pfxs2...)

	for _, pfx := range pfxs1[:1_000] {
		rtbl.Update(pfx.pfx, func(any, bool) any { return nil })
	}

	for _, pfx := range randomPrefixes(20_000) {
		rtbl.Delete(pfx.pfx)
	}

	var golden4 int
	var golden6 int

	rtbl.AllSorted4()(func(netip.Prefix, any) bool {
		golden4++
		return true
	})

	rtbl.AllSorted6()(func(netip.Prefix, any) bool {
		golden6++
		return true
	})

	if golden4 != rtbl.Size4() {
		t.Errorf("Size4: want: %d, got: %d", golden4, rtbl.Size4())
	}

	if golden6 != rtbl.Size6() {
		t.Errorf("Size6: want: %d, got: %d", golden6, rtbl.Size6())
	}
}

// ############ benchmarks ################################

var benchRouteCount = []int{1, 2, 5, 10, 100, 1000, 10_000, 100_000, 1_000_000}

func BenchmarkTableInsert(b *testing.B) {
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		for _, nroutes := range benchRouteCount {
			var rt Table[struct{}]
			for _, route := range rng(nroutes) {
				rt.Insert(route.pfx, struct{}{})
			}

			probe := rng(1)[0]

			b.ResetTimer()
			b.Run(fmt.Sprintf("%s/Into_%d", fam, nroutes), func(b *testing.B) {
				for range b.N {
					rt.Insert(probe.pfx, struct{}{})
				}
			})
		}
	}
}

func BenchmarkTableDelete(b *testing.B) {
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		for _, nroutes := range benchRouteCount {
			var rt Table[int]
			for _, route := range rng(nroutes) {
				rt.Insert(route.pfx, route.val)
			}

			probe := rng(1)[0]

			b.ResetTimer()
			b.Run(fmt.Sprintf("%s/From_%d", fam, nroutes), func(b *testing.B) {
				for range b.N {
					rt.Delete(probe.pfx)
				}
			})
		}
	}
}

func BenchmarkTableGet(b *testing.B) {
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		for _, nroutes := range benchRouteCount {
			var rt Table[int]
			for _, route := range rng(nroutes) {
				rt.Insert(route.pfx, route.val)
			}

			probe := rng(1)[0]

			b.ResetTimer()
			b.Run(fmt.Sprintf("%s/From_%d", fam, nroutes), func(b *testing.B) {
				for range b.N {
					writeSink, _ = rt.Get(probe.pfx)
				}
			})
		}
	}
}

func BenchmarkTableLookup(b *testing.B) {
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		for _, nroutes := range benchRouteCount {
			var rt Table[int]
			for _, route := range rng(nroutes) {
				rt.Insert(route.pfx, route.val)
			}

			probe := rng(1)[0]

			b.ResetTimer()
			b.Run(fmt.Sprintf("%s/In_%6d/%s", fam, nroutes, "Contains"), func(b *testing.B) {
				for range b.N {
					okSink = rt.Contains(probe.pfx.Addr())
				}
			})

			b.ResetTimer()
			b.Run(fmt.Sprintf("%s/In_%6d/%s", fam, nroutes, "Lookup"), func(b *testing.B) {
				for range b.N {
					writeSink, _ = rt.Lookup(probe.pfx.Addr())
				}
			})

			b.ResetTimer()
			b.Run(fmt.Sprintf("%s/In_%6d/%s", fam, nroutes, "Prefix"), func(b *testing.B) {
				for range b.N {
					writeSink, _ = rt.LookupPrefix(probe.pfx)
				}
			})

			b.ResetTimer()
			b.Run(fmt.Sprintf("%s/In_%6d/%s", fam, nroutes, "PrefixLPM"), func(b *testing.B) {
				for range b.N {
					_, writeSink, _ = rt.LookupPrefixLPM(probe.pfx)
				}
			})
		}
	}
}

func BenchmarkTableOverlapsPrefix(b *testing.B) {
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		for _, nroutes := range benchRouteCount {
			var rt Table[int]
			for _, route := range rng(nroutes) {
				rt.Insert(route.pfx, route.val)
			}

			probe := rng(1)[0]

			b.ResetTimer()
			b.Run(fmt.Sprintf("%s/With_%d", fam, nroutes), func(b *testing.B) {
				for range b.N {
					boolSink = rt.OverlapsPrefix(probe.pfx)
				}
			})
		}
	}
}

func BenchmarkTableOverlaps(b *testing.B) {
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		for _, nroutes := range benchRouteCount {
			var rt Table[int]
			for _, route := range rng(nroutes) {
				rt.Insert(route.pfx, route.val)
			}

			const (
				intersectSize = 8
				numIntersects = 1_000
			)

			intersects := make([]*Table[int], numIntersects)
			for i := range intersects {
				inter := &Table[int]{}
				for _, route := range rng(intersectSize) {
					inter.Insert(route.pfx, route.val)
				}
				intersects[i] = inter
			}

			b.ResetTimer()
			b.Run(fmt.Sprintf("%s/%d_with_%d", fam, nroutes, intersectSize), func(b *testing.B) {
				for i := range b.N {
					boolSink = rt.Overlaps(intersects[i%numIntersects])
				}
			})
		}
	}
}

func BenchmarkTableClone(b *testing.B) {
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		for _, nroutes := range benchRouteCount {
			var rt Table[int]
			for _, route := range rng(nroutes) {
				rt.Insert(route.pfx, route.val)
			}

			b.ResetTimer()
			b.Run(fmt.Sprintf("%s/%d", fam, nroutes), func(b *testing.B) {
				for range b.N {
					rt.Clone()
				}
			})
		}
	}
}

func BenchmarkMemory(b *testing.B) {
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		var startMem, endMem runtime.MemStats
		for _, nroutes := range benchRouteCount {
			rt := new(Table[any])

			b.Run(fmt.Sprintf("%s/random/%d", fam, nroutes), func(b *testing.B) {
				b.ResetTimer()

				for range b.N {
					rt = new(Table[any])
					runtime.GC()
					runtime.ReadMemStats(&startMem)

					for _, route := range rng(nroutes) {
						rt.Insert(route.pfx, struct{}{})
					}

					runtime.GC()
					runtime.ReadMemStats(&endMem)

					b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc), "Bytes")
					b.ReportMetric(float64(nroutes)/float64(rt.nodes()), "Prefix/Node")
					b.ReportMetric(0, "ns/op") // silence
				}
			})
		}
	}
}

// ##################### helpers ############################

type tableOverlapsTest struct {
	prefix string
	want   bool
}

// checkOverlaps verifies that the overlaps lookups in tt return the
// expected results on tbl.
func checkOverlaps(t *testing.T, tbl *Table[int], tests []tableOverlapsTest) {
	t.Helper()
	for _, tt := range tests {
		got := tbl.OverlapsPrefix(mpp(tt.prefix))
		if got != tt.want {
			t.Log(tbl.String())
			t.Errorf("OverlapsPrefix(%v) = %v, want %v", mpp(tt.prefix), got, tt.want)
		}
	}
}

type tableTest struct {
	// addr is an IP address string to look up in a route table.
	addr string
	// want is the expected >=0 value associated with the route, or -1
	// if we expect a lookup miss.
	want int
}

// checkRoutes verifies that the route lookups in tt return the
// expected results on tbl.
func checkRoutes(t *testing.T, tbl *Table[int], tt []tableTest) {
	t.Helper()
	for _, tc := range tt {
		v, ok := tbl.Lookup(mpa(tc.addr))

		if !ok && tc.want != -1 {
			t.Errorf("Lookup %q got (%v, %v), want (%v, false)", tc.addr, v, ok, tc.want)
		}
		if ok && v != tc.want {
			t.Errorf("Lookup %q got (%v, %v), want (%v, true)", tc.addr, v, ok, tc.want)
		}
	}
}

func checkNumNodes(t *testing.T, tbl *Table[int], want int) {
	t.Helper()
	if got := tbl.nodes(); got != want {
		t.Errorf("wrong table size, got %d strides want %d", got, want)
	}
}

// dumpAsGoldTable, just a helper to compare with golden table.
func (t *Table[V]) dumpAsGoldTable() goldTable[V] {
	var tbl goldTable[V]

	t.AllSorted()(func(pfx netip.Prefix, val V) bool {
		tbl = append(tbl, goldTableItem[V]{pfx: pfx, val: val})
		return true
	})

	return tbl
}
