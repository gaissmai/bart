// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause
//
// some regression tests modified from github.com/tailscale/art
// for this implementation by:
//
// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"fmt"
	"math/rand/v2"
	"net/netip"
	"runtime"
	"strconv"
	"testing"
)

// workLoadN to adjust loops for tests with -short
func workLoadN() int {
	if testing.Short() {
		return 1_000
	}
	return 10_000
}

// tabler is a test-only interface that provides a unified abstraction for
// testing multiple table implementations (Table and Fast) with the same test suite.
type tabler[V any] interface {
	Get(netip.Prefix) (V, bool)
	Insert(netip.Prefix, V)
	Delete(netip.Prefix)
	Modify(netip.Prefix, func(V, bool) (V, bool))
	Contains(netip.Addr) bool
	Size() int
}

var mpa = netip.MustParseAddr

var mpp = func(s string) netip.Prefix {
	pfx := netip.MustParsePrefix(s)
	if pfx == pfx.Masked() {
		return pfx
	}
	panic(fmt.Sprintf("%s is not canonicalized as %s", s, pfx.Masked()))
}

// tests for deep copies with Cloner interface
type MyInt int

// implement the Cloner interface
func (i *MyInt) Clone() *MyInt {
	a := *i
	return &a
}

func noPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("%s panicked: %v", name, r)
		}
	}()
	fn()
}

// ############ tests ################################

func TestInvalid(t *testing.T) {
	t.Parallel()

	tbl1 := new(Table[any])
	tbl2 := new(Table[any])

	var zeroIP netip.Addr
	var zeroPfx netip.Prefix

	noPanic(t, "Contains", func() { tbl1.Contains(zeroIP) })
	noPanic(t, "Lookup", func() { tbl1.Lookup(zeroIP) })

	noPanic(t, "LookupPrefix", func() { tbl1.LookupPrefix(zeroPfx) })
	noPanic(t, "LookupPrefixLPM", func() { tbl1.LookupPrefixLPM(zeroPfx) })

	noPanic(t, "Insert", func() { tbl1.Insert(zeroPfx, nil) })
	noPanic(t, "Get", func() { tbl1.Get(zeroPfx) })
	noPanic(t, "Delete", func() { tbl1.Delete(zeroPfx) })
	noPanic(t, "Modify", func() { tbl1.Modify(zeroPfx, nil) })

	noPanic(t, "InsertPersist", func() { tbl1.InsertPersist(zeroPfx, nil) })
	noPanic(t, "DeletePersist", func() { tbl1.DeletePersist(zeroPfx) })
	noPanic(t, "ModifyPersist", func() { tbl1.ModifyPersist(zeroPfx, nil) })

	noPanic(t, "OverlapsPrefix", func() { tbl1.OverlapsPrefix(zeroPfx) })

	noPanic(t, "Overlaps", func() { tbl1.Overlaps(tbl2) })
	noPanic(t, "Overlaps4", func() { tbl1.Overlaps4(tbl2) })
	noPanic(t, "Overlaps6", func() { tbl1.Overlaps6(tbl2) })
}

func TestInsert(t *testing.T) {
	t.Parallel()

	tbl := new(Table[int])

	// Create a new leaf strideTable, with compressed path
	tbl.Insert(mpp("192.168.0.1/32"), 1)
	checkNumNodes(t, tbl, 1)
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

	// explode path compressed
	tbl.Insert(mpp("192.168.0.2/32"), 2)
	checkNumNodes(t, tbl, 4)
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

	// Insert into existing leaf
	tbl.Insert(mpp("192.168.0.0/26"), 7)
	checkNumNodes(t, tbl, 4)
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

	// Create a different leaf at root
	tbl.Insert(mpp("10.0.0.0/27"), 3)
	checkNumNodes(t, tbl, 4)
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

	// Insert that creates a new path compressed leaf
	tbl.Insert(mpp("192.168.1.1/32"), 4)
	checkNumNodes(t, tbl, 4)
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

	// Insert that creates a new path compressed leaf
	tbl.Insert(mpp("192.170.0.0/16"), 5)
	checkNumNodes(t, tbl, 4)
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
	checkNumNodes(t, tbl, 4)
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

	// Insert that explodes the previous path compression
	tbl.Insert(mpp("192.180.0.0/21"), 9)
	checkNumNodes(t, tbl, 5)
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
	checkNumNodes(t, tbl, 5)
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

	// Create a new path compressed leaf
	tbl.Insert(mpp("ff:aaaa::1/128"), 1)
	checkNumNodes(t, tbl, 6)
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

	// Insert into previous leaf, explode v6 path compression
	tbl.Insert(mpp("ff:aaaa::2/128"), 2)
	checkNumNodes(t, tbl, 21)
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

	// Insert into previous node
	tbl.Insert(mpp("ff:aaaa::/125"), 7)
	checkNumNodes(t, tbl, 21)
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
	checkNumNodes(t, tbl, 21)
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

	// Insert that creates a new path compressed leaf
	tbl.Insert(mpp("ff:aaaa:aaaa::1/128"), 4)
	checkNumNodes(t, tbl, 21)
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

	// Insert that creates a new path in tree
	tbl.Insert(mpp("ff:aaaa:aaaa:bb00::/56"), 5)
	checkNumNodes(t, tbl, 23)
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
	checkNumNodes(t, tbl, 23)
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

	// Insert that explodes a previous path compressed leaf
	tbl.Insert(mpp("ff:cccc::/37"), 9)
	checkNumNodes(t, tbl, 25)
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
	checkNumNodes(t, tbl, 25)
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

func TestInsertPersist(t *testing.T) {
	t.Parallel()

	tbl := new(Table[int])

	// Create a new leaf strideTable, with compressed path
	tbl = tbl.InsertPersist(mpp("192.168.0.1/32"), 1)
	checkNumNodes(t, tbl, 1)
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

	// explode path compressed
	tbl = tbl.InsertPersist(mpp("192.168.0.2/32"), 2)
	checkNumNodes(t, tbl, 4)
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

	// Insert into existing leaf
	tbl = tbl.InsertPersist(mpp("192.168.0.0/26"), 7)
	checkNumNodes(t, tbl, 4)
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

	// Create a different leaf at root
	tbl = tbl.InsertPersist(mpp("10.0.0.0/27"), 3)
	checkNumNodes(t, tbl, 4)
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

	// Insert that creates a new path compressed leaf
	tbl = tbl.InsertPersist(mpp("192.168.1.1/32"), 4)
	checkNumNodes(t, tbl, 4)
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

	// Insert that creates a new path compressed leaf
	tbl = tbl.InsertPersist(mpp("192.170.0.0/16"), 5)
	checkNumNodes(t, tbl, 4)
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
	tbl = tbl.InsertPersist(mpp("192.180.0.1/32"), 8)
	checkNumNodes(t, tbl, 4)
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

	// Insert that explodes the previous path compression
	tbl = tbl.InsertPersist(mpp("192.180.0.0/21"), 9)
	checkNumNodes(t, tbl, 5)
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
	tbl = tbl.InsertPersist(mpp("0.0.0.0/0"), 6)
	checkNumNodes(t, tbl, 5)
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

	// Create a new path compressed leaf
	tbl = tbl.InsertPersist(mpp("ff:aaaa::1/128"), 1)
	checkNumNodes(t, tbl, 6)
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

	// Insert into previous leaf, explode v6 path compression
	tbl = tbl.InsertPersist(mpp("ff:aaaa::2/128"), 2)
	checkNumNodes(t, tbl, 21)
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

	// Insert into previous node
	tbl = tbl.InsertPersist(mpp("ff:aaaa::/125"), 7)
	checkNumNodes(t, tbl, 21)
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
	tbl = tbl.InsertPersist(mpp("ffff:bbbb::/120"), 3)
	checkNumNodes(t, tbl, 21)
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

	// Insert that creates a new path compressed leaf
	tbl = tbl.InsertPersist(mpp("ff:aaaa:aaaa::1/128"), 4)
	checkNumNodes(t, tbl, 21)
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

	// Insert that creates a new path in tree
	tbl = tbl.InsertPersist(mpp("ff:aaaa:aaaa:bb00::/56"), 5)
	checkNumNodes(t, tbl, 23)
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
	tbl = tbl.InsertPersist(mpp("ff:cccc::1/128"), 8)
	checkNumNodes(t, tbl, 23)
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

	// Insert that explodes a previous path compressed leaf
	tbl = tbl.InsertPersist(mpp("ff:cccc::/37"), 9)
	checkNumNodes(t, tbl, 25)
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
	tbl = tbl.InsertPersist(mpp("::/0"), 6)
	checkNumNodes(t, tbl, 25)
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
		t.Parallel()
		prng := rand.New(rand.NewPCG(42, 42))
		// must not panic
		tbl := new(Table[int])
		checkNumNodes(t, tbl, 0)
		tbl.Delete(randomPrefix(prng))
		checkNumNodes(t, tbl, 0)
	})

	t.Run("prefix_in_root", func(t *testing.T) {
		t.Parallel()
		// Add/remove prefix from root table.
		tbl := new(Table[int])
		checkNumNodes(t, tbl, 0)

		tbl.Insert(mpp("10.0.0.0/8"), 1)
		checkNumNodes(t, tbl, 1)
		checkRoutes(t, tbl, []tableTest{
			{"10.0.0.1", 1},
			{"255.255.255.255", -1},
		})
		tbl.Delete(mpp("10.0.0.0/8"))
		checkNumNodes(t, tbl, 0)
		checkRoutes(t, tbl, []tableTest{
			{"10.0.0.1", -1},
			{"255.255.255.255", -1},
		})
	})

	t.Run("prefix_in_leaf", func(t *testing.T) {
		t.Parallel()
		// Create, then delete a single leaf table.
		tbl := new(Table[int])
		checkNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		checkNumNodes(t, tbl, 1)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"255.255.255.255", -1},
		})

		tbl.Delete(mpp("192.168.0.1/32"))
		checkNumNodes(t, tbl, 0)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", -1},
			{"255.255.255.255", -1},
		})
	})

	t.Run("intermediate_no_routes", func(t *testing.T) {
		t.Parallel()
		// Create an intermediate with 2 leaves, then delete one leaf.
		tbl := new(Table[int])
		checkNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		tbl.Insert(mpp("192.180.0.1/32"), 2)
		checkNumNodes(t, tbl, 2)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", 2},
			{"192.40.0.1", -1},
		})

		tbl.Delete(mpp("192.180.0.1/32"))
		checkNumNodes(t, tbl, 1)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", -1},
			{"192.40.0.1", -1},
		})
	})

	t.Run("intermediate_with_route", func(t *testing.T) {
		t.Parallel()
		// Same, but the intermediate carries a route as well.
		tbl := new(Table[int])
		checkNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		tbl.Insert(mpp("192.180.0.1/32"), 2)
		tbl.Insert(mpp("192.0.0.0/10"), 3)

		checkNumNodes(t, tbl, 2)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", 2},
			{"192.40.0.1", 3},
			{"192.255.0.1", -1},
		})

		tbl.Delete(mpp("192.180.0.1/32"))
		checkNumNodes(t, tbl, 2)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", -1},
			{"192.40.0.1", 3},
			{"192.255.0.1", -1},
		})
	})

	t.Run("intermediate_many_leaves", func(t *testing.T) {
		t.Parallel()
		// Intermediate with 3 leaves, then delete one leaf.
		tbl := new(Table[int])
		checkNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		tbl.Insert(mpp("192.180.0.1/32"), 2)
		tbl.Insert(mpp("192.200.0.1/32"), 3)

		checkNumNodes(t, tbl, 2)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", 2},
			{"192.200.0.1", 3},
			{"192.255.0.1", -1},
		})

		tbl.Delete(mpp("192.180.0.1/32"))
		checkNumNodes(t, tbl, 2)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", -1},
			{"192.200.0.1", 3},
			{"192.255.0.1", -1},
		})
	})

	t.Run("nosuchprefix_missing_child", func(t *testing.T) {
		t.Parallel()
		// Delete non-existent prefix
		tbl := new(Table[int])
		checkNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		checkNumNodes(t, tbl, 1)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.255.0.1", -1},
		})

		tbl.Delete(mpp("200.0.0.0/32"))
		checkNumNodes(t, tbl, 1)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.255.0.1", -1},
		})
	})

	t.Run("intermediate_with_deleted_route", func(t *testing.T) {
		t.Parallel()
		// Intermediate node loses its last route and becomes
		// compactable.
		tbl := new(Table[int])
		checkNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		tbl.Insert(mpp("192.168.0.0/22"), 2)
		checkNumNodes(t, tbl, 3)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.168.0.2", 2},
			{"192.255.0.1", -1},
		})

		tbl.Delete(mpp("192.168.0.0/22"))
		checkNumNodes(t, tbl, 1)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.168.0.2", -1},
			{"192.255.0.1", -1},
		})
	})

	t.Run("default_route", func(t *testing.T) {
		t.Parallel()
		tbl := new(Table[int])
		checkNumNodes(t, tbl, 0)

		tbl.Insert(mpp("0.0.0.0/0"), 1)
		tbl.Insert(mpp("::/0"), 1)
		tbl.Delete(mpp("0.0.0.0/0"))

		checkNumNodes(t, tbl, 1)
		checkRoutes(t, tbl, []tableTest{
			{"1.2.3.4", -1},
			{"::1", 1},
		})
	})

	t.Run("path compressed purge", func(t *testing.T) {
		t.Parallel()
		tbl := new(Table[int])
		checkNumNodes(t, tbl, 0)

		tbl.Insert(mpp("10.10.0.0/17"), 1)
		tbl.Insert(mpp("10.20.0.0/17"), 2)
		checkNumNodes(t, tbl, 2)

		tbl.Delete(mpp("10.20.0.0/17"))
		checkNumNodes(t, tbl, 1)

		tbl.Delete(mpp("10.10.0.0/17"))
		checkNumNodes(t, tbl, 0)
	})
}

func TestDeletePersist(t *testing.T) {
	t.Parallel()

	t.Run("table_is_empty", func(t *testing.T) {
		t.Parallel()
		prng := rand.New(rand.NewPCG(42, 42))
		// must not panic
		tbl := new(Table[int])
		checkNumNodes(t, tbl, 0)
		tbl = tbl.DeletePersist(randomPrefix(prng))
		checkNumNodes(t, tbl, 0)
	})

	t.Run("prefix_in_root", func(t *testing.T) {
		t.Parallel()
		// Add/remove prefix from root table.
		tbl := new(Table[int])
		checkNumNodes(t, tbl, 0)

		tbl.Insert(mpp("10.0.0.0/8"), 1)
		checkNumNodes(t, tbl, 1)
		checkRoutes(t, tbl, []tableTest{
			{"10.0.0.1", 1},
			{"255.255.255.255", -1},
		})
		tbl = tbl.DeletePersist(mpp("10.0.0.0/8"))
		checkNumNodes(t, tbl, 0)
		checkRoutes(t, tbl, []tableTest{
			{"10.0.0.1", -1},
			{"255.255.255.255", -1},
		})
	})

	t.Run("prefix_in_leaf", func(t *testing.T) {
		t.Parallel()
		// Create, then delete a single leaf table.
		tbl := new(Table[int])
		checkNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		checkNumNodes(t, tbl, 1)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"255.255.255.255", -1},
		})

		tbl = tbl.DeletePersist(mpp("192.168.0.1/32"))
		checkNumNodes(t, tbl, 0)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", -1},
			{"255.255.255.255", -1},
		})
	})

	t.Run("intermediate_no_routes", func(t *testing.T) {
		t.Parallel()
		// Create an intermediate with 2 leaves, then delete one leaf.
		tbl := new(Table[int])
		checkNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		tbl.Insert(mpp("192.180.0.1/32"), 2)
		checkNumNodes(t, tbl, 2)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", 2},
			{"192.40.0.1", -1},
		})

		tbl = tbl.DeletePersist(mpp("192.180.0.1/32"))
		checkNumNodes(t, tbl, 1)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", -1},
			{"192.40.0.1", -1},
		})
	})

	t.Run("intermediate_with_route", func(t *testing.T) {
		t.Parallel()
		// Same, but the intermediate carries a route as well.
		tbl := new(Table[int])
		checkNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		tbl.Insert(mpp("192.180.0.1/32"), 2)
		tbl.Insert(mpp("192.0.0.0/10"), 3)

		checkNumNodes(t, tbl, 2)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", 2},
			{"192.40.0.1", 3},
			{"192.255.0.1", -1},
		})

		tbl = tbl.DeletePersist(mpp("192.180.0.1/32"))
		checkNumNodes(t, tbl, 2)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", -1},
			{"192.40.0.1", 3},
			{"192.255.0.1", -1},
		})
	})

	t.Run("intermediate_many_leaves", func(t *testing.T) {
		t.Parallel()
		// Intermediate with 3 leaves, then delete one leaf.
		tbl := new(Table[int])
		checkNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		tbl.Insert(mpp("192.180.0.1/32"), 2)
		tbl.Insert(mpp("192.200.0.1/32"), 3)

		checkNumNodes(t, tbl, 2)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", 2},
			{"192.200.0.1", 3},
			{"192.255.0.1", -1},
		})

		tbl = tbl.DeletePersist(mpp("192.180.0.1/32"))
		checkNumNodes(t, tbl, 2)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", -1},
			{"192.200.0.1", 3},
			{"192.255.0.1", -1},
		})
	})

	t.Run("nosuchprefix_missing_child", func(t *testing.T) {
		t.Parallel()
		// Delete non-existent prefix
		tbl := new(Table[int])
		checkNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		checkNumNodes(t, tbl, 1)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.255.0.1", -1},
		})

		tbl = tbl.DeletePersist(mpp("200.0.0.0/32"))
		checkNumNodes(t, tbl, 1)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.255.0.1", -1},
		})
	})

	t.Run("intermediate_with_deleted_route", func(t *testing.T) {
		t.Parallel()
		// Intermediate node loses its last route and becomes
		// compactable.
		tbl := new(Table[int])
		checkNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		tbl.Insert(mpp("192.168.0.0/22"), 2)
		checkNumNodes(t, tbl, 3)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.168.0.2", 2},
			{"192.255.0.1", -1},
		})

		tbl = tbl.DeletePersist(mpp("192.168.0.0/22"))
		checkNumNodes(t, tbl, 1)
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.168.0.2", -1},
			{"192.255.0.1", -1},
		})
	})

	t.Run("default_route", func(t *testing.T) {
		t.Parallel()
		tbl := new(Table[int])
		checkNumNodes(t, tbl, 0)

		tbl.Insert(mpp("0.0.0.0/0"), 1)
		tbl.Insert(mpp("::/0"), 1)
		tbl = tbl.DeletePersist(mpp("0.0.0.0/0"))

		checkNumNodes(t, tbl, 1)
		checkRoutes(t, tbl, []tableTest{
			{"1.2.3.4", -1},
			{"::1", 1},
		})
	})

	t.Run("path compressed purge", func(t *testing.T) {
		t.Parallel()
		tbl := new(Table[int])
		checkNumNodes(t, tbl, 0)

		tbl.Insert(mpp("10.10.0.0/17"), 1)
		tbl.Insert(mpp("10.20.0.0/17"), 2)
		checkNumNodes(t, tbl, 2)

		tbl = tbl.DeletePersist(mpp("10.20.0.0/17"))
		checkNumNodes(t, tbl, 1)

		tbl = tbl.DeletePersist(mpp("10.10.0.0/17"))
		checkNumNodes(t, tbl, 0)
	})
}

func TestContainsCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, n)

	gold := new(goldTable[int])
	gold.insertMany(pfxs)

	fast := new(Table[int])
	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	for range n {
		a := randomAddr(prng)

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

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, n)

	gold := new(goldTable[int])
	gold.insertMany(pfxs)

	fast := new(Table[int])
	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	for range n {
		a := randomAddr(prng)

		goldVal, goldOK := gold.lookup(a)
		fastVal, fastOK := fast.Lookup(a)

		if !getsEqual(goldVal, goldOK, fastVal, fastOK) {
			t.Fatalf("Lookup(%q) = (%v, %v), want (%v, %v)", a, fastVal, fastOK, goldVal, goldOK)
		}
	}
}

func TestLookupPrefixUnmasked(t *testing.T) {
	// test that the pfx must not be masked on input for LookupPrefix
	t.Parallel()

	rt := new(Table[any])
	rt.Insert(mpp("10.20.30.0/24"), nil)

	// not normalized pfxs
	tests := []struct {
		probe   netip.Prefix
		wantLPM netip.Prefix
		wantOk  bool
	}{
		{
			probe:   netip.MustParsePrefix("10.20.30.40/0"),
			wantLPM: netip.Prefix{},
			wantOk:  false,
		},
		{
			probe:   netip.MustParsePrefix("10.20.30.40/23"),
			wantLPM: netip.Prefix{},
			wantOk:  false,
		},
		{
			probe:   netip.MustParsePrefix("10.20.30.40/24"),
			wantLPM: mpp("10.20.30.0/24"),
			wantOk:  true,
		},
		{
			probe:   netip.MustParsePrefix("10.20.30.40/25"),
			wantLPM: mpp("10.20.30.0/24"),
			wantOk:  true,
		},
		{
			probe:   netip.MustParsePrefix("10.20.30.40/32"),
			wantLPM: mpp("10.20.30.0/24"),
			wantOk:  true,
		},
	}

	for _, tc := range tests {
		_, got := rt.LookupPrefix(tc.probe)
		if got != tc.wantOk {
			t.Errorf("LookupPrefix non canonical prefix (%s), got: %v, want: %v", tc.probe, got, tc.wantOk)
		}

		lpm, _, got := rt.LookupPrefixLPM(tc.probe)
		if got != tc.wantOk {
			t.Errorf("LookupPrefixLPM non canonical prefix (%s), got: %v, want: %v", tc.probe, got, tc.wantOk)
		}
		if lpm != tc.wantLPM {
			t.Errorf("LookupPrefixLPM non canonical prefix (%s), got: %v, want: %v", tc.probe, lpm, tc.wantLPM)
		}
	}
}

func TestLookupPrefixCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, n)

	gold := new(goldTable[int])
	gold.insertMany(pfxs)

	fast := new(Table[int])
	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	for range n {
		pfx := randomPrefix(prng)

		goldVal, goldOK := gold.lookupPfx(pfx)
		fastVal, fastOK := fast.LookupPrefix(pfx)

		if !getsEqual(goldVal, goldOK, fastVal, fastOK) {
			t.Fatalf("LookupPrefix(%q) = (%v, %v), want (%v, %v)", pfx, fastVal, fastOK, goldVal, goldOK)
		}
	}
}

func TestLookupPrefixLPMCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, n)

	gold := new(goldTable[int])
	gold.insertMany(pfxs)

	fast := new(Table[int])
	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	for range n {
		pfx := randomPrefix(prng)

		goldLPM, goldVal, goldOK := gold.lookupPfxLPM(pfx)
		fastLPM, fastVal, fastOK := fast.LookupPrefixLPM(pfx)

		if !getsEqual(goldVal, goldOK, fastVal, fastOK) {
			t.Fatalf("LookupPrefixLPM(%q) = (%v, %v), want (%v, %v)", pfx, fastVal, fastOK, goldVal, goldOK)
		}

		if !getsEqual(goldLPM, goldOK, fastLPM, fastOK) {
			t.Fatalf("LookupPrefixLPM(%q) = (%v, %v), want (%v, %v)", pfx, fastLPM, fastOK, goldLPM, goldOK)
		}
	}
}

func TestInsertShuffled(t *testing.T) {
	// The order in which you insert prefixes into a route table
	// should not matter, as long as you're inserting the same set of
	// routes.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, 1000)

	for range 10 {
		pfxs2 := append([]goldTableItem[int](nil), pfxs...)
		prng.Shuffle(len(pfxs2), func(i, j int) { pfxs2[i], pfxs2[j] = pfxs2[j], pfxs2[i] })

		addrs := make([]netip.Addr, 0, n)
		for range n {
			addrs = append(addrs, randomAddr(prng))
		}

		rt1 := new(Table[int])
		rt2 := new(Table[int])

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

func TestInsertPersistShuffled(t *testing.T) {
	// The order in which you insert prefixes into a route table
	// should not matter, as long as you're inserting the same set of
	// routes.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, 1000)

	for range 10 {
		pfxs2 := append([]goldTableItem[int](nil), pfxs...)
		prng.Shuffle(len(pfxs2), func(i, j int) { pfxs2[i], pfxs2[j] = pfxs2[j], pfxs2[i] })

		addrs := make([]netip.Addr, 0, n)
		for range n {
			addrs = append(addrs, randomAddr(prng))
		}

		rt1 := new(Table[int])
		rt2 := new(Table[int])

		// rt1 is mutable
		for _, pfx := range pfxs {
			rt1.Insert(pfx.pfx, pfx.val)
		}

		// rt2 is persistent
		for _, pfx := range pfxs2 {
			rt2 = rt2.InsertPersist(pfx.pfx, pfx.val)
		}

		if rt1.String() != rt2.String() {
			t.Fatal("mutable and immutable table have different string representation")
		}

		if rt1.dumpString() != rt2.dumpString() {
			t.Fatal("mutable and immutable table have different dumpString representation")
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

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))

	var (
		numPrefixes  = n // total prefixes to insert (test deletes 50% of them)
		numPerFamily = numPrefixes / 2
		deleteCut    = numPerFamily / 2
		numProbes    = n // random addr lookups to do
	)

	// We have to do this little dance instead of just using allPrefixes,
	// because we want pfxs and toDelete to be non-overlapping sets.
	all4, all6 := randomPrefixes4(prng, numPerFamily), randomPrefixes6(prng, numPerFamily)

	pfxs := append([]goldTableItem[int](nil), all4[:deleteCut]...)
	pfxs = append(pfxs, all6[:deleteCut]...)

	toDelete := append([]goldTableItem[int](nil), all4[deleteCut:]...)
	toDelete = append(toDelete, all6[deleteCut:]...)

	gold := new(goldTable[int])
	gold.insertMany(pfxs)

	fast := new(Table[int])
	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	for _, pfx := range toDelete {
		fast.Insert(pfx.pfx, pfx.val)
	}
	for _, pfx := range toDelete {
		fast.Delete(pfx.pfx)
	}

	for range numProbes {
		a := randomAddr(prng)

		goldVal, goldOK := gold.lookup(a)
		fastVal, fastOK := fast.Lookup(a)

		if !getsEqual(goldVal, goldOK, fastVal, fastOK) {
			t.Fatalf("Lookup(%q) = (%v, %v), want (%v, %v)", a, fastVal, fastOK, goldVal, goldOK)
		}
	}
}

func TestDeleteShuffled(t *testing.T) {
	// The order in which you delete prefixes from a route table
	// should not matter, as long as you're deleting the same set of
	// routes.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))

	var (
		numPrefixes  = n // prefixes to insert (test deletes 50% of them)
		numPerFamily = numPrefixes / 2
		deleteCut    = numPerFamily / 2
	)

	for range 10 {
		// We have to do this little dance instead of just using allPrefixes,
		// because we want pfxs and toDelete to be non-overlapping sets.
		all4, all6 := randomPrefixes4(prng, numPerFamily), randomPrefixes6(prng, numPerFamily)

		pfxs := append([]goldTableItem[int](nil), all4[:deleteCut]...)
		pfxs = append(pfxs, all6[:deleteCut]...)

		toDelete := append([]goldTableItem[int](nil), all4[deleteCut:]...)
		toDelete = append(toDelete, all6[deleteCut:]...)

		rt1 := new(Table[int])

		// insert
		for _, pfx := range pfxs {
			rt1.Insert(pfx.pfx, pfx.val)
		}
		for _, pfx := range toDelete {
			rt1.Insert(pfx.pfx, pfx.val)
		}

		// delete
		for _, pfx := range toDelete {
			rt1.Delete(pfx.pfx)
		}

		pfxs2 := append([]goldTableItem[int](nil), pfxs...)
		toDelete2 := append([]goldTableItem[int](nil), toDelete...)
		prng.Shuffle(len(toDelete2), func(i, j int) { toDelete2[i], toDelete2[j] = toDelete2[j], toDelete2[i] })

		rt2 := new(Table[int])

		// insert
		for _, pfx := range pfxs2 {
			rt2.Insert(pfx.pfx, pfx.val)
		}
		for _, pfx := range toDelete2 {
			rt2.Insert(pfx.pfx, pfx.val)
		}

		// delete
		for _, pfx := range toDelete2 {
			rt2.Delete(pfx.pfx)
		}

		if rt1.String() != rt2.String() {
			t.Fatal("shuffled table has different string representation")
		}

		if rt1.dumpString() != rt2.dumpString() {
			t.Fatal("shuffled table has different dumpString representation")
		}
	}
}

func TestDeleteIsReverseOfInsert(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	// Insert n prefixes, then delete those same prefixes in reverse
	// order. Each deletion should exactly undo the internal structure
	// changes that each insert did.

	tbl := new(Table[int])
	want := tbl.dumpString()

	prefixes := randomPrefixes(prng, n)

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

func TestDeleteButOne(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))
	// Insert N prefixes, then delete all but one
	const N = 100

	for range 1_000 {

		tbl := new(Table[int])
		prefixes := randomPrefixes(prng, N)

		for _, p := range prefixes {
			tbl.Insert(p.pfx, p.val)
		}

		// shuffle the prefixes
		prng.Shuffle(N, func(i, j int) {
			prefixes[i], prefixes[j] = prefixes[j], prefixes[i]
		})

		for i, p := range prefixes {
			// skip the first
			if i == 0 {
				continue
			}
			tbl.Delete(p.pfx)
		}

		stats4 := nodeStatsRec(&tbl.root4)
		stats6 := nodeStatsRec(&tbl.root6)

		if nodes := stats4.nodes + stats6.nodes; nodes != 1 {
			t.Fatalf("delete but one, want nodes: 1, got: %d\n%s", nodes, tbl.dumpString())
		}

		sum := stats4.pfxs + stats4.leaves + stats4.fringes +
			stats6.pfxs + stats6.leaves + stats6.fringes

		if sum != 1 {
			t.Fatalf("delete but one, only one item must be left, but: %d\n%s", sum, tbl.dumpString())
		}
	}
}

func TestGet(t *testing.T) {
	t.Parallel()

	t.Run("empty table", func(t *testing.T) {
		t.Parallel()
		prng := rand.New(rand.NewPCG(42, 42))

		rt := new(Table[int])
		pfx := randomPrefix(prng)
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

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, n)

	gold := new(goldTable[int])
	gold.insertMany(pfxs)

	fast := new(Table[int])
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

// TestModifySemantics
//
// Operation | cb-input        | cb-return
// ------------------------------------------
// No-op:    | (zero,   false) | (_,      true)
// Insert:   | (zero,   false) | (newVal, false)
// Update:   | (oldVal, true)  | (newVal, false)
// Delete:   | (oldVal, true)  | (_,      true)
func TestModifySemantics(t *testing.T) {
	t.Parallel()

	type args struct {
		pfx    netip.Prefix
		cb     func(val int, found bool) (_ int, del bool)
		cbLite func(found bool) (del bool)
	}

	type want struct {
		val     int
		deleted bool
	}

	tests := []struct {
		name      string
		prepare   map[netip.Prefix]int // entries to pre-populate the table
		args      args
		want      want
		finalData map[netip.Prefix]int // expected table contents after the operation
	}{
		{
			name:    "Delete existing entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42, mpp("2001:db8::/32"): 4242},
			args: args{
				pfx:    mpp("10.0.0.0/8"),
				cb:     func(val int, found bool) (_ int, del bool) { return 0, true },
				cbLite: func(found bool) (del bool) { return true },
			},
			want:      want{val: 42, deleted: true},
			finalData: map[netip.Prefix]int{mpp("2001:db8::/32"): 4242},
		},

		{
			name:    "Insert new entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42},
			args: args{
				pfx:    mpp("2001:db8::/32"),
				cb:     func(val int, found bool) (_ int, del bool) { return 4242, false },
				cbLite: func(found bool) (del bool) { return false },
			},
			want:      want{val: 4242, deleted: false},
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42, mpp("2001:db8::/32"): 4242},
		},

		{
			// For update, the callback gets oldVal, returns newVal, but Modify returns oldVal
			name:    "Update existing entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42, mpp("2001:db8::/32"): 4242},
			args: args{
				pfx:    mpp("10.0.0.0/8"),
				cb:     func(val int, found bool) (_ int, del bool) { return -1, false },
				cbLite: func(found bool) (del bool) { return false },
			},
			want:      want{val: 42, deleted: false},
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): -1, mpp("2001:db8::/32"): 4242},
		},

		{
			name:    "No-op on missing entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42},
			args: args{
				pfx:    mpp("2001:db8::/32"),
				cb:     func(val int, found bool) (_ int, del bool) { return 0, true },
				cbLite: func(found bool) (del bool) { return true },
			},
			want:      want{val: 0, deleted: false},
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42},
		},
	}

	for _, tt := range tests {
		// Table
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rt := new(Table[int])

			// Insert initial entries using Modify
			for pfx, v := range tt.prepare {
				rt.Modify(pfx, func(_ int, _ bool) (_ int, del bool) { return v, false })
			}

			rt.Modify(tt.args.pfx, tt.args.cb)

			// Check the final state of the table using Get, compares expected and actual table
			for pfx, wantVal := range tt.finalData {
				gotVal, ok := rt.Get(pfx)
				if !ok || gotVal != wantVal {
					t.Errorf("[%s] final table: key %v = %v (ok=%v), want %v (ok=true)", tt.name, pfx, gotVal, ok, wantVal)
				}
			}
			// Ensure there are no unexpected entries
			for pfx := range tt.prepare {
				if _, expect := tt.finalData[pfx]; !expect {
					if _, ok := rt.Get(pfx); ok {
						t.Errorf("[%s] final table: key %v should not be present", tt.name, pfx)
					}
				}
			}
		})

		// Fast
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rt := new(Fast[int])

			// Insert initial entries using Modify
			for pfx, v := range tt.prepare {
				rt.Modify(pfx, func(_ int, _ bool) (_ int, del bool) { return v, false })
			}

			rt.Modify(tt.args.pfx, tt.args.cb)

			// Check the final state of the table using Get, compares expected and actual table
			for pfx, wantVal := range tt.finalData {
				gotVal, ok := rt.Get(pfx)
				if !ok || gotVal != wantVal {
					t.Errorf("[%s] final table: key %v = %v (ok=%v), want %v (ok=true)", tt.name, pfx, gotVal, ok, wantVal)
				}
			}
			// Ensure there are no unexpected entries
			for pfx := range tt.prepare {
				if _, expect := tt.finalData[pfx]; !expect {
					if _, ok := rt.Get(pfx); ok {
						t.Errorf("[%s] final table: key %v should not be present", tt.name, pfx)
					}
				}
			}
		})

		// Lite
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rt := new(Lite)

			// Insert initial entries using Modify
			for pfx := range tt.prepare {
				rt.Modify(pfx, func(_ bool) (del bool) { return false })
			}

			rt.Modify(tt.args.pfx, tt.args.cbLite)

			// Check the final state of the table using Get, compares expected and actual table
			for pfx := range tt.finalData {
				ok := rt.Get(pfx)
				if !ok {
					t.Errorf("[%s] final table: pfx %v (ok=%v), want (ok=true)", tt.name, pfx, ok)
				}
			}

			// Ensure there are no unexpected entries
			for pfx := range rt.All() {
				if _, ok := tt.finalData[pfx]; !ok {
					t.Errorf("[%s] final table: pfx %v should not be present", tt.name, pfx)
				}
			}
		})
	}
}

// TestModifyPersistSemantics
//
// Operation | cb-input        | cb-return
// ------------------------------------------
// No-op:    | (zero,   false) | (_,      true)
// Insert:   | (zero,   false) | (newVal, false)
// Update:   | (oldVal, true)  | (newVal, false)
// Delete:   | (oldVal, true)  | (_,      true)
func TestTableModifyPersistSemantics(t *testing.T) {
	t.Parallel()

	type args struct {
		pfx    netip.Prefix
		cb     func(val int, found bool) (_ int, del bool)
		cbLite func(found bool) (del bool)
	}

	type want struct {
		val     int
		deleted bool
	}

	tests := []struct {
		name      string
		prepare   map[netip.Prefix]int // entries to pre-populate the table
		args      args
		want      want
		finalData map[netip.Prefix]int // expected table contents after the operation
	}{
		{
			name:    "Delete existing entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42, mpp("2001:db8::/32"): 4242},
			args: args{
				pfx:    mpp("10.0.0.0/8"),
				cb:     func(val int, found bool) (_ int, del bool) { return 0, true },
				cbLite: func(found bool) (del bool) { return true },
			},
			want:      want{val: 42, deleted: true},
			finalData: map[netip.Prefix]int{mpp("2001:db8::/32"): 4242},
		},

		{
			name:    "Insert new entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42},
			args: args{
				pfx:    mpp("2001:db8::/32"),
				cb:     func(val int, found bool) (_ int, del bool) { return 4242, false },
				cbLite: func(found bool) (del bool) { return false },
			},
			want:      want{val: 4242, deleted: false},
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42, mpp("2001:db8::/32"): 4242},
		},

		{
			// For update, the callback gets oldVal, returns newVal, but Modify returns oldVal
			name:    "Update existing entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42, mpp("2001:db8::/32"): 4242},
			args: args{
				pfx:    mpp("10.0.0.0/8"),
				cb:     func(val int, found bool) (_ int, del bool) { return -1, false },
				cbLite: func(found bool) (del bool) { return false },
			},
			want:      want{val: 42, deleted: false},
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): -1, mpp("2001:db8::/32"): 4242},
		},

		{
			name:    "No-op on missing entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42},
			args: args{
				pfx:    mpp("2001:db8::/32"),
				cb:     func(val int, found bool) (_ int, del bool) { return 0, true },
				cbLite: func(found bool) (del bool) { return true },
			},
			want:      want{val: 0, deleted: false},
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42},
		},
	}

	for _, tt := range tests {
		// Table
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rt := new(Table[int])

			// Insert initial entries using Modify
			for pfx, v := range tt.prepare {
				rt.Modify(pfx, func(_ int, _ bool) (_ int, del bool) { return v, false })
			}

			prt := rt.ModifyPersist(tt.args.pfx, tt.args.cb)

			// Check the final state of the table using Get, compares expected and actual table
			for pfx, wantVal := range tt.finalData {
				gotVal, ok := prt.Get(pfx)
				if !ok || gotVal != wantVal {
					t.Errorf("[%s] final table: key %v = %v (ok=%v), want %v (ok=true)", tt.name, pfx, gotVal, ok, wantVal)
				}
			}
			// Ensure there are no unexpected entries
			for pfx := range tt.prepare {
				if _, expect := tt.finalData[pfx]; !expect {
					if _, ok := prt.Get(pfx); ok {
						t.Errorf("[%s] final table: key %v should not be present", tt.name, pfx)
					}
				}
			}
		})

		// Fast
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rt := new(Fast[int])

			// Insert initial entries using Modify
			for pfx, v := range tt.prepare {
				rt.Modify(pfx, func(_ int, _ bool) (_ int, del bool) { return v, false })
			}

			prt := rt.ModifyPersist(tt.args.pfx, tt.args.cb)

			// Check the final state of the table using Get, compares expected and actual table
			for pfx, wantVal := range tt.finalData {
				gotVal, ok := prt.Get(pfx)
				if !ok || gotVal != wantVal {
					t.Errorf("[%s] final table: key %v = %v (ok=%v), want %v (ok=true)", tt.name, pfx, gotVal, ok, wantVal)
				}
			}
			// Ensure there are no unexpected entries
			for pfx := range tt.prepare {
				if _, expect := tt.finalData[pfx]; !expect {
					if _, ok := prt.Get(pfx); ok {
						t.Errorf("[%s] final table: key %v should not be present", tt.name, pfx)
					}
				}
			}
		})

		// Lite
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rt := new(Lite)

			// Insert initial entries using Modify
			for pfx := range tt.prepare {
				rt.Modify(pfx, func(_ bool) (del bool) { return false })
			}

			prt := rt.ModifyPersist(tt.args.pfx, tt.args.cbLite)

			// Check the final state of the table using Get, compares expected and actual table
			for pfx := range tt.finalData {
				ok := prt.Get(pfx)
				if !ok {
					t.Errorf("[%s] final table: pfx %v (ok=%v), want (ok=true)", tt.name, pfx, ok)
				}
			}

			// Ensure there are no unexpected entries
			for pfx := range prt.All() {
				if _, ok := tt.finalData[pfx]; !ok {
					t.Errorf("[%s] final table: pfx %v should not be present", tt.name, pfx)
				}
			}
		})
	}
}

func TestModifyCompare(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, n)

	gold := new(goldTable[int])
	gold.insertMany(pfxs)

	// Update as insert
	fast := new(Table[int])
	for _, pfx := range pfxs {
		fast.Modify(pfx.pfx, func(int, bool) (int, bool) { return pfx.val, false })
	}

	for _, pfx := range pfxs {
		goldVal, goldOK := gold.get(pfx.pfx)
		fastVal, fastOK := fast.Get(pfx.pfx)

		if !getsEqual(goldVal, goldOK, fastVal, fastOK) {
			t.Fatalf("Get(%q) = (%v, %v), want (%v, %v)", pfx.pfx, fastVal, fastOK, goldVal, goldOK)
		}
	}

	cb1 := func(val int, _ bool) int { return val + 1 }
	cb2 := func(val int, _ bool) (int, bool) { return val + 1, false }

	// Update as update
	for _, pfx := range pfxs[:len(pfxs)/2] {
		gold.update(pfx.pfx, cb1)
		fast.Modify(pfx.pfx, cb2)
	}

	for _, pfx := range pfxs {
		goldVal, goldOK := gold.get(pfx.pfx)
		fastVal, fastOK := fast.Get(pfx.pfx)

		if !getsEqual(goldVal, goldOK, fastVal, fastOK) {
			t.Fatalf("Get(%q) = (%v, %v), want (%v, %v)", pfx.pfx, fastVal, fastOK, goldVal, goldOK)
		}
	}
}

func TestModifyPersistCompare(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, n)
	imu := new(Table[int])
	mut := new(Table[int])

	// Update as insert
	for _, pfx := range pfxs {
		imu = imu.ModifyPersist(pfx.pfx, func(int, bool) (int, bool) { return pfx.val, false })
		mut.Modify(pfx.pfx, func(int, bool) (int, bool) { return pfx.val, false })
	}

	for _, pfx := range pfxs {
		imuVal, imuOk := imu.Get(pfx.pfx)
		mutVal, mutOk := mut.Get(pfx.pfx)

		if !getsEqual(mutVal, mutOk, imuVal, imuOk) {
			t.Fatalf("Get(%q) = (%v, %v), want (%v, %v)", pfx.pfx, imuVal, imuOk, mutVal, mutOk)
		}
	}

	cb := func(val int, _ bool) (int, bool) { return val + 1, false }

	// Update as update
	for _, pfx := range pfxs[:len(pfxs)/2] {
		imu = imu.ModifyPersist(pfx.pfx, cb)
		mut.Modify(pfx.pfx, cb)
	}

	for _, pfx := range pfxs {
		bartVal, bartOK := mut.Get(pfx.pfx)
		immuVal, immuOK := imu.Get(pfx.pfx)

		if !getsEqual(bartVal, bartOK, immuVal, immuOK) {
			t.Fatalf("Get(%q) = (%v, %v), want (%v, %v)", pfx.pfx, immuVal, immuOK, bartVal, bartOK)
		}
	}
}

func TestModifyShuffled(t *testing.T) {
	// The order in which you delete prefixes from a route table
	// should not matter, as long as you're deleting the same set of
	// routes.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))

	var (
		numPrefixes  = n // prefixes to insert (test deletes 50% of them)
		numPerFamily = numPrefixes / 2
		deleteCut    = numPerFamily / 2
	)

	for range 10 {
		// We have to do this little dance instead of just using allPrefixes,
		// because we want pfxs and toDelete to be non-overlapping sets.
		all4, all6 := randomPrefixes4(prng, numPerFamily), randomPrefixes6(prng, numPerFamily)

		pfxs := append([]goldTableItem[int](nil), all4[:deleteCut]...)
		pfxs = append(pfxs, all6[:deleteCut]...)

		toDelete := append([]goldTableItem[int](nil), all4[deleteCut:]...)
		toDelete = append(toDelete, all6[deleteCut:]...)

		rt1 := new(Table[int])

		// insert
		for _, pfx := range pfxs {
			rt1.Insert(pfx.pfx, pfx.val)
		}
		for _, pfx := range toDelete {
			rt1.Insert(pfx.pfx, pfx.val)
		}

		// this callback deletes unconditionally
		cb := func(int, bool) (int, bool) { return 0, true }

		// delete
		for _, pfx := range toDelete {
			rt1.Modify(pfx.pfx, cb)
		}

		pfxs2 := append([]goldTableItem[int](nil), pfxs...)
		toDelete2 := append([]goldTableItem[int](nil), toDelete...)
		prng.Shuffle(len(toDelete2), func(i, j int) { toDelete2[i], toDelete2[j] = toDelete2[j], toDelete2[i] })

		rt2 := new(Table[int])

		// insert
		for _, pfx := range pfxs2 {
			rt2.Insert(pfx.pfx, pfx.val)
		}
		for _, pfx := range toDelete2 {
			rt2.Insert(pfx.pfx, pfx.val)
		}

		// delete
		for _, pfx := range toDelete2 {
			rt2.Modify(pfx.pfx, cb)
		}

		if rt1.String() != rt2.String() {
			t.Fatal("shuffled table has different string representation")
		}

		if rt1.dumpString() != rt2.dumpString() {
			t.Fatal("shuffled table has different dumpString representation")
		}
	}
}

func TestModifyPersistShuffled(t *testing.T) {
	// The order in which you delete prefixes from a route table
	// should not matter, as long as you're deleting the same set of
	// routes.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))

	var (
		numPrefixes  = n // prefixes to insert (test deletes 50% of them)
		numPerFamily = numPrefixes / 2
		deleteCut    = numPerFamily / 2
	)

	for range 10 {
		// We have to do this little dance instead of just using allPrefixes,
		// because we want pfxs and toDelete to be non-overlapping sets.
		all4, all6 := randomPrefixes4(prng, numPerFamily), randomPrefixes6(prng, numPerFamily)

		pfxs := append([]goldTableItem[int](nil), all4[:deleteCut]...)
		pfxs = append(pfxs, all6[:deleteCut]...)

		toDelete := append([]goldTableItem[int](nil), all4[deleteCut:]...)
		toDelete = append(toDelete, all6[deleteCut:]...)

		rt1 := new(Table[int])

		// insert
		for _, pfx := range pfxs {
			rt1.Insert(pfx.pfx, pfx.val)
		}
		for _, pfx := range toDelete {
			rt1.Insert(pfx.pfx, pfx.val)
		}

		// this callback deletes unconditionally
		cb := func(int, bool) (int, bool) { return 0, true }

		// delete
		for _, pfx := range toDelete {
			rt1 = rt1.ModifyPersist(pfx.pfx, cb)
		}

		pfxs2 := append([]goldTableItem[int](nil), pfxs...)
		toDelete2 := append([]goldTableItem[int](nil), toDelete...)
		prng.Shuffle(len(toDelete2), func(i, j int) { toDelete2[i], toDelete2[j] = toDelete2[j], toDelete2[i] })

		rt2 := new(Table[int])

		// insert
		for _, pfx := range pfxs2 {
			rt2.Insert(pfx.pfx, pfx.val)
		}
		for _, pfx := range toDelete2 {
			rt2.Insert(pfx.pfx, pfx.val)
		}

		// delete
		for _, pfx := range toDelete2 {
			rt2 = rt2.ModifyPersist(pfx.pfx, cb)
		}

		if rt1.String() != rt2.String() {
			t.Fatal("shuffled table has different string representation")
		}

		if rt1.dumpString() != rt2.dumpString() {
			t.Fatal("shuffled table has different dumpString representation")
		}
	}
}

func TestUnionEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		aTbl := new(Table[int])
		bTbl := new(Table[int])

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
		aTbl := new(Table[int])
		bTbl := new(Table[int])

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

	t.Run("this empty", func(t *testing.T) {
		t.Parallel()
		aTbl := new(Table[int])
		bTbl := new(Table[int])

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
		aTbl := new(Table[string])
		bTbl := new(Table[string])

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
		aTbl := new(Table[int])
		bTbl := new(Table[int])

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
		aTbl := new(Table[int])
		bTbl := new(Table[int])

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

func TestUnionPersistEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		aTbl := new(Table[int])
		bTbl := new(Table[int])

		// union empty tables
		cTbl := aTbl.UnionPersist(bTbl)

		want := ""
		got := cTbl.String()
		if got != want {
			t.Fatalf("got:\n%v\nwant:\n%v", got, want)
		}
	})

	t.Run("other empty", func(t *testing.T) {
		t.Parallel()
		aTbl := new(Table[int])
		bTbl := new(Table[int])

		// one empty table, b
		aTbl.Insert(mpp("10.0.0.0/8"), 0)
		aTbl.Insert(mpp("10.1.0.0/24"), 0)
		aTbl.Insert(mpp("2001:db8::/64"), 0)
		aTbl.Insert(mpp("2001:db8::1/128"), 0)

		cTbl := aTbl.UnionPersist(bTbl)
		want := `▼
└─ 10.0.0.0/8 (0)
   └─ 10.1.0.0/24 (0)
▼
└─ 2001:db8::/64 (0)
   └─ 2001:db8::1/128 (0)
`
		got := cTbl.String()
		if got != want {
			t.Fatalf("got:\n%v\nwant:\n%v", got, want)
		}
	})

	t.Run("this empty", func(t *testing.T) {
		t.Parallel()
		aTbl := new(Table[int])
		bTbl := new(Table[int])

		bTbl.Insert(mpp("10.0.0.0/8"), 0)
		bTbl.Insert(mpp("10.1.0.0/24"), 0)
		bTbl.Insert(mpp("2001:db8::/64"), 0)
		bTbl.Insert(mpp("2001:db8::1/128"), 0)

		cTbl := aTbl.UnionPersist(bTbl)
		want := `▼
└─ 10.0.0.0/8 (0)
   └─ 10.1.0.0/24 (0)
▼
└─ 2001:db8::/64 (0)
   └─ 2001:db8::1/128 (0)
`
		got := cTbl.String()
		if got != want {
			t.Fatalf("got:\n%v\nwant:\n%v", got, want)
		}
	})

	t.Run("duplicate prefix", func(t *testing.T) {
		t.Parallel()
		aTbl := new(Table[string])
		bTbl := new(Table[string])

		aTbl.Insert(mpp("::/0"), "orig value")
		bTbl.Insert(mpp("::/0"), "overwrite")

		cTbl := aTbl.UnionPersist(bTbl)
		want := `▼
└─ ::/0 (overwrite)
`
		got := cTbl.String()
		if got != want {
			t.Fatalf("got:\n%v\nwant:\n%v", got, want)
		}
	})

	t.Run("same children", func(t *testing.T) {
		t.Parallel()
		aTbl := new(Table[int])
		bTbl := new(Table[int])

		aTbl.Insert(mpp("127.0.0.1/32"), 1)
		aTbl.Insert(mpp("::1/128"), 1)

		bTbl.Insert(mpp("127.0.0.2/32"), 2)
		bTbl.Insert(mpp("::2/128"), 2)

		cTbl := aTbl.UnionPersist(bTbl)
		want := `▼
├─ 127.0.0.1/32 (1)
└─ 127.0.0.2/32 (2)
▼
├─ ::1/128 (1)
└─ ::2/128 (2)
`
		got := cTbl.String()
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
		t := new(Table[struct{}])
		for _, s := range pfx {
			t.Insert(mpp(s), struct{}{})
		}
		return t
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

// TestUnionPersistMemoryAliasing tests that the Union method does not alias memory
// between the tables.
func TestUnionPersistMemoryAliasing(t *testing.T) {
	t.Parallel()

	newTable := func(pfx ...string) *Table[struct{}] {
		t := new(Table[struct{}])
		for _, s := range pfx {
			t.Insert(mpp(s), struct{}{})
		}
		return t
	}
	// First create two tables with disjoint prefixes.
	a := newTable("100.69.1.0/24")
	b := newTable("0.0.0.0/24")

	// Verify that the tables are disjoint.
	if a.Overlaps(b) {
		t.Error("this should not overlap other")
	}

	// Now union them with copy-on-write.
	pTbl := a.UnionPersist(b)

	// Add a new prefix to new union
	pTbl.Insert(mpp("0.0.1.0/24"), struct{}{})

	// Ensure that a is unchanged.
	_, ok := a.Lookup(mpa("0.0.1.1"))
	if ok {
		t.Error("a should not contain 0.0.1.1")
	}
	if a.OverlapsPrefix(mpp("0.0.1.1/32")) {
		t.Error("a should not overlap 0.0.1.1/32")
	}
}

func TestUnionCompare(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))

	const numEntries = 200

	for range 100 {
		pfxs := randomPrefixes(prng, numEntries)

		gold := new(goldTable[int])
		gold.insertMany(pfxs)

		fast := new(Table[int])
		for _, pfx := range pfxs {
			fast.Insert(pfx.pfx, pfx.val)
		}

		pfxs2 := randomPrefixes(prng, numEntries)

		gold2 := new(goldTable[int])
		gold2.insertMany(pfxs2)

		fast2 := new(Table[int])
		for _, pfx := range pfxs2 {
			fast2.Insert(pfx.pfx, pfx.val)
		}

		gold.union(gold2)
		fast.Union(fast2)

		// dump as slow table for comparison
		fastAsGoldenTbl := fast.dumpAsGoldTable()

		// sort for comparison
		gold.sort()
		fastAsGoldenTbl.sort()

		for i := range *gold {
			goldItem := (*gold)[i]
			fastItem := fastAsGoldenTbl[i]
			if goldItem != fastItem {
				t.Fatalf("Union(...): items[%d] differ slow(%v) != fast(%v)", i, goldItem, fastItem)
			}
		}

		// check the size
		if fast.Size() != len(*gold) {
			t.Errorf("sizes differ, got: %d, want: %d", fast.Size(), len(*gold))
		}
	}
}

func TestUnionPersistCompare(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))

	const numEntries = 200

	for range 100 {
		pfxs := randomPrefixes(prng, numEntries)

		gold := new(goldTable[int])
		gold.insertMany(pfxs)

		fast := new(Table[int])
		for _, pfx := range pfxs {
			fast.Insert(pfx.pfx, pfx.val)
		}

		pfxs2 := randomPrefixes(prng, numEntries)

		gold2 := new(goldTable[int])
		gold2.insertMany(pfxs2)

		fast2 := new(Table[int])
		for _, pfx := range pfxs2 {
			fast2.Insert(pfx.pfx, pfx.val)
		}

		gold.union(gold2)
		pTbl := fast.UnionPersist(fast2)

		// dump as slow table for comparison
		fastAsGoldenTbl := pTbl.dumpAsGoldTable()

		// sort for comparison
		gold.sort()
		fastAsGoldenTbl.sort()

		for i := range *gold {
			goldItem := (*gold)[i]
			fastItem := fastAsGoldenTbl[i]
			if goldItem != fastItem {
				t.Fatalf("UnionPersist(...): items[%d] differ slow(%v) != fast(%v)", i, goldItem, fastItem)
			}
		}

		// check the size
		if pTbl.Size() != len(*gold) {
			t.Errorf("sizes differ, got: %d, want: %d", pTbl.Size(), len(*gold))
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
	prng := rand.New(rand.NewPCG(42, 42))

	pfxs := randomPrefixes(prng, 100_000)

	golden := new(Table[int])
	tbl := new(Table[int])
	for _, pfx := range pfxs {
		golden.Insert(pfx.pfx, pfx.val)
		tbl.Insert(pfx.pfx, pfx.val)
	}
	clone := tbl.Clone()

	if golden.dumpString() != clone.dumpString() {
		t.Errorf("Clone: got:\n%swant:\n%s", clone.dumpString(), golden.dumpString())
	}

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

	if *got != *want || got != want {
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
		t.Errorf("value with Cloner interface, pointers must be different:\nvalues(%d, %d)\n(ptr(%v, %v)", *got, *want, got, want)
	}

	// update value, deep copy of values, cloned value must now be different
	val = 2
	want, _ = tbl.Get(pfx)
	got, _ = clone.Get(pfx)

	if *got == *want {
		t.Errorf("memory aliasing after deep copy, values must be different:\nvalues(%d, %d)", *got, *want)
	}
}

func TestUnionShallow(t *testing.T) {
	t.Parallel()

	tbl1 := new(Table[*int])
	tbl2 := new(Table[*int])

	val := 1
	pfx := mpp("10.0.0.1/32")
	tbl2.Insert(pfx, &val)

	tbl1.Union(tbl2)
	got, _ := tbl1.Get(pfx)
	want, _ := tbl2.Get(pfx)

	if *got != *want || got != want {
		t.Errorf("shallow copy, values and pointers must be equal:\nvalues(%d, %d)\n(ptr(%v, %v)", *got, *want, got, want)
	}

	// update value, shallow copy of values, union must be equal
	val = 2
	got, _ = tbl1.Get(pfx)
	want, _ = tbl2.Get(pfx)

	if *got != *want {
		t.Errorf("memory aliasing after shallow copy, values must be equal:\nvalues(%d, %d)", *got, *want)
	}
}

func TestUnionDeep(t *testing.T) {
	t.Parallel()

	tbl1 := new(Table[*MyInt])
	tbl2 := new(Table[*MyInt])

	val := MyInt(1)
	pfx := mpp("10.0.0.1/32")
	tbl2.Insert(pfx, &val)

	tbl1.Union(tbl2)
	got, _ := tbl1.Get(pfx)
	want, _ := tbl2.Get(pfx)

	if *got != *want || got == want {
		t.Errorf("value with Cloner interface, pointers must be different:\nvalues(%d, %d)\n(ptr(%v, %v)", *got, *want, got, want)
	}

	// update value, shallow copy of values, union must be equal
	val = 2
	got, _ = tbl1.Get(pfx)
	want, _ = tbl2.Get(pfx)

	if *got == *want {
		t.Errorf("memory aliasing after deep copy, values must be different:\nvalues(%d, %d)", *got, *want)
	}
}

// test some edge cases
func TestOverlapsPrefixEdgeCases(t *testing.T) {
	t.Parallel()

	tbl := new(Table[int])

	// empty table
	checkOverlapsPrefix(t, tbl, []tableOverlapsTest{
		{"0.0.0.0/0", false},
		{"::/0", false},
	})

	// default route
	tbl.Insert(mpp("10.0.0.0/9"), 0)
	tbl.Insert(mpp("2001:db8::/32"), 0)
	checkOverlapsPrefix(t, tbl, []tableOverlapsTest{
		{"0.0.0.0/0", true},
		{"::/0", true},
	})

	// default route
	tbl = new(Table[int])
	tbl.Insert(mpp("0.0.0.0/0"), 0)
	tbl.Insert(mpp("::/0"), 0)
	checkOverlapsPrefix(t, tbl, []tableOverlapsTest{
		{"10.0.0.0/9", true},
		{"2001:db8::/32", true},
	})

	// single IP
	tbl = new(Table[int])
	tbl.Insert(mpp("10.0.0.0/7"), 0)
	tbl.Insert(mpp("2001::/16"), 0)
	checkOverlapsPrefix(t, tbl, []tableOverlapsTest{
		{"10.1.2.3/32", true},
		{"2001:db8:affe::cafe/128", true},
	})

	// single IP
	tbl = new(Table[int])
	tbl.Insert(mpp("10.1.2.3/32"), 0)
	tbl.Insert(mpp("2001:db8:affe::cafe/128"), 0)
	checkOverlapsPrefix(t, tbl, []tableOverlapsTest{
		{"10.0.0.0/7", true},
		{"2001::/16", true},
	})

	// same IPv
	tbl = new(Table[int])
	tbl.Insert(mpp("10.1.2.3/32"), 0)
	tbl.Insert(mpp("2001:db8:affe::cafe/128"), 0)
	checkOverlapsPrefix(t, tbl, []tableOverlapsTest{
		{"10.1.2.3/32", true},
		{"2001:db8:affe::cafe/128", true},
	})
}

func TestSize(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))

	tbl := new(Table[any])
	if tbl.Size() != 0 {
		t.Errorf("empty Table: want: 0, got: %d", tbl.Size())
	}

	if tbl.Size4() != 0 {
		t.Errorf("empty Table: want: 0, got: %d", tbl.Size4())
	}

	if tbl.Size6() != 0 {
		t.Errorf("empty Table: want: 0, got: %d", tbl.Size6())
	}

	pfxs1 := randomPrefixes(prng, n)
	pfxs2 := randomPrefixes(prng, n)

	for _, pfx := range pfxs1 {
		tbl.Insert(pfx.pfx, nil)
	}

	for _, pfx := range pfxs2 {
		tbl.Modify(pfx.pfx, func(any, bool) (any, bool) { return nil, false })
	}

	pfxs1 = append(pfxs1, pfxs2...)

	for _, pfx := range pfxs1[:n] {
		tbl.Modify(pfx.pfx, func(any, bool) (any, bool) { return nil, false })
	}

	for _, pfx := range randomPrefixes(prng, n) {
		tbl.Delete(pfx.pfx)
	}

	var allInc4 int
	var allInc6 int

	for range tbl.AllSorted4() {
		allInc4++
	}

	for range tbl.AllSorted6() {
		allInc6++
	}

	if allInc4 != tbl.Size4() {
		t.Errorf("Size4: want: %d, got: %d", allInc4, tbl.Size4())
	}

	if allInc6 != tbl.Size6() {
		t.Errorf("Size6: want: %d, got: %d", allInc6, tbl.Size6())
	}
}

func TestLastOctetPlusOneAndLastBits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		pfx       netip.Prefix
		wantDepth int
		wantBits  uint8
	}{
		{
			pfx:       mpp("0.0.0.0/0"),
			wantDepth: 0,
			wantBits:  0,
		},
		{
			pfx:       mpp("0.0.0.0/32"),
			wantDepth: 4,
			wantBits:  0,
		},
		{
			pfx:       mpp("10.0.0.0/7"),
			wantDepth: 0,
			wantBits:  7,
		},
		{
			pfx:       mpp("10.20.0.0/14"),
			wantDepth: 1,
			wantBits:  6,
		},
		{
			pfx:       mpp("10.20.30.0/24"),
			wantDepth: 3,
			wantBits:  0,
		},
		{
			pfx:       mpp("10.20.30.40/31"),
			wantDepth: 3,
			wantBits:  7,
		},
		//
		{
			pfx:       mpp("::/0"),
			wantDepth: 0,
			wantBits:  0,
		},
		{
			pfx:       mpp("::/128"),
			wantDepth: 16,
			wantBits:  0,
		},
		{
			pfx:       mpp("2001:db8::/31"),
			wantDepth: 3,
			wantBits:  7,
		},
	}

	for _, tc := range tests {
		lastOctetPlusOne, gotBits := lastOctetPlusOneAndLastBits(tc.pfx)
		if lastOctetPlusOne != tc.wantDepth {
			t.Errorf("lastOctetPlusOneAndLastBits(%d), lastOctetPlusOne got: %d, want: %d",
				tc.pfx.Bits(), lastOctetPlusOne, tc.wantDepth)
		}
		if gotBits != tc.wantBits {
			t.Errorf("lastOctetPlusOneAndLastBits(%d), lastBits got: %d, want: %d",
				tc.pfx.Bits(), gotBits, tc.wantBits)
		}
	}
}

// ############ benchmarks ################################

var benchRouteCount = []int{1, 2, 5, 10, 100, 1000, 10_000, 100_000, 200_000}

func BenchmarkTableModifyRandom(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, n := range benchRouteCount {
		randomPfxs := randomRealWorldPrefixes(prng, n)

		rt := new(Table[int])
		for i, pfx := range randomPfxs {
			rt.Insert(pfx, i)
		}

		prt := rt

		probe := randomPfxs[prng.IntN(len(randomPfxs))]

		b.Run(fmt.Sprintf("mutable into %d", n), func(b *testing.B) {
			for b.Loop() {
				rt.Modify(probe, func(int, bool) (int, bool) { return 42, false })
			}
		})

		b.Run(fmt.Sprintf("persist into %d", n), func(b *testing.B) {
			for b.Loop() {
				prt.ModifyPersist(probe, func(int, bool) (int, bool) { return 42, false })
			}
		})

	}
}

func BenchmarkTableDelete(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, n := range []int{1_000, 10_000, 100_000, 1_000_000} {
		pfxs := randomPrefixes(prng, n)

		b.Run(fmt.Sprintf("mutable from_%d", n), func(b *testing.B) {
			for b.Loop() {
				b.StopTimer()
				rt := new(Table[*MyInt])

				for i, route := range pfxs {
					myInt := MyInt(i)
					rt.Insert(route.pfx, &myInt)
				}
				b.StartTimer()

				for _, route := range pfxs {
					rt.Delete(route.pfx)
				}
			}
			b.ReportMetric(float64(b.Elapsed())/float64(b.N)/float64(len(pfxs)), "ns/route")
			b.ReportMetric(0, "ns/op")
		})

		b.Run(fmt.Sprintf("persist from_%d", n), func(b *testing.B) {
			for b.Loop() {
				b.StopTimer()
				rt := new(Table[*MyInt])

				for i, route := range pfxs {
					myInt := MyInt(i)
					rt.Insert(route.pfx, &myInt)
				}
				b.StartTimer()

				for _, route := range pfxs {
					rt = rt.DeletePersist(route.pfx)
				}
			}
			b.ReportMetric(float64(b.Elapsed())/float64(b.N)/float64(len(pfxs)), "ns/route")
			b.ReportMetric(0, "ns/op")
		})
	}
}

func BenchmarkTableGet(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		for _, nroutes := range benchRouteCount {
			rt := new(Table[int])
			for _, route := range rng(prng, nroutes) {
				rt.Insert(route.pfx, route.val)
			}

			probe := rng(prng, 1)[0]

			b.Run(fmt.Sprintf("%s/From_%d", fam, nroutes), func(b *testing.B) {
				for b.Loop() {
					rt.Get(probe.pfx)
				}
			})
		}
	}
}

func BenchmarkTableLPM(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		for _, nroutes := range benchRouteCount {
			rt := new(Table[int])
			for _, route := range rng(prng, nroutes) {
				rt.Insert(route.pfx, route.val)
			}

			probe := rng(prng, 1)[0]

			b.Run(fmt.Sprintf("%s/In_%6d/%s", fam, nroutes, "Contains"), func(b *testing.B) {
				for b.Loop() {
					rt.Contains(probe.pfx.Addr())
				}
			})

			b.Run(fmt.Sprintf("%s/In_%6d/%s", fam, nroutes, "Lookup"), func(b *testing.B) {
				for b.Loop() {
					rt.Lookup(probe.pfx.Addr())
				}
			})

			b.Run(fmt.Sprintf("%s/In_%6d/%s", fam, nroutes, "Prefix"), func(b *testing.B) {
				for b.Loop() {
					rt.LookupPrefix(probe.pfx)
				}
			})

			b.Run(fmt.Sprintf("%s/In_%6d/%s", fam, nroutes, "PrefixLPM"), func(b *testing.B) {
				for b.Loop() {
					rt.LookupPrefixLPM(probe.pfx)
				}
			})
		}
	}
}

func BenchmarkTableOverlapsPrefix(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		for _, nroutes := range benchRouteCount {
			rt := new(Table[int])
			for _, route := range rng(prng, nroutes) {
				rt.Insert(route.pfx, route.val)
			}

			probe := rng(prng, 1)[0]

			b.Run(fmt.Sprintf("%s/With_%d", fam, nroutes), func(b *testing.B) {
				for b.Loop() {
					rt.OverlapsPrefix(probe.pfx)
				}
			})
		}
	}
}

func BenchmarkTableOverlaps(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		for _, nroutes := range benchRouteCount {
			rt := new(Table[int])
			for _, route := range rng(prng, nroutes) {
				rt.Insert(route.pfx, route.val)
			}

			inter := new(Table[int])
			for _, route := range rng(prng, nroutes) {
				inter.Insert(route.pfx, route.val)
			}

			b.Run(fmt.Sprintf("%s/%d_with_%d", fam, nroutes, nroutes), func(b *testing.B) {
				for b.Loop() {
					rt.Overlaps(inter)
				}
			})
		}
	}
}

func BenchmarkTableClone(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		for _, nroutes := range benchRouteCount {
			rt := new(Table[int])
			for _, route := range rng(prng, nroutes) {
				rt.Insert(route.pfx, route.val)
			}

			b.Run(fmt.Sprintf("%s/%d", fam, nroutes), func(b *testing.B) {
				for b.Loop() {
					rt.Clone()
				}
			})
		}
	}
}

func BenchmarkMemIP4(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, k := range []int{1_000, 10_000, 100_000, 1_000_000} {
		var startMem, endMem runtime.MemStats

		runtime.GC()
		runtime.ReadMemStats(&startMem)

		b.Run(strconv.Itoa(k), func(b *testing.B) {
			rt := new(Table[struct{}])
			for b.Loop() {
				rt = new(Table[struct{}])
				for _, pfx := range randomRealWorldPrefixes4(prng, k) {
					rt.Insert(pfx, struct{}{})
				}
			}

			runtime.GC()
			runtime.ReadMemStats(&endMem)

			stats := nodeStatsRec(&rt.root4)

			bytes := float64(endMem.HeapAlloc - startMem.HeapAlloc)
			b.ReportMetric(roundFloat64(bytes/float64(stats.pfxs)), "bytes/route")
			b.ReportMetric(float64(stats.pfxs), "pfxs")
			b.ReportMetric(float64(stats.nodes), "nodes")
			b.ReportMetric(float64(stats.leaves), "leaves")
			b.ReportMetric(float64(stats.fringes), "fringes")
			b.ReportMetric(0, "ns/op")
		})
	}
}

func BenchmarkMemIP6(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, k := range []int{1_000, 10_000, 100_000, 1_000_000} {
		var startMem, endMem runtime.MemStats

		runtime.GC()
		runtime.ReadMemStats(&startMem)

		b.Run(strconv.Itoa(k), func(b *testing.B) {
			rt := new(Table[struct{}])
			for b.Loop() {
				rt = new(Table[struct{}])
				for _, pfx := range randomRealWorldPrefixes6(prng, k) {
					rt.Insert(pfx, struct{}{})
				}
			}

			runtime.GC()
			runtime.ReadMemStats(&endMem)

			stats := nodeStatsRec(&rt.root6)

			bytes := float64(endMem.HeapAlloc - startMem.HeapAlloc)
			b.ReportMetric(roundFloat64(bytes/float64(stats.pfxs)), "bytes/route")
			b.ReportMetric(float64(stats.pfxs), "pfxs")
			b.ReportMetric(float64(stats.nodes), "nodes")
			b.ReportMetric(float64(stats.leaves), "leaves")
			b.ReportMetric(float64(stats.fringes), "fringes")
			b.ReportMetric(0, "ns/op")
		})
	}
}

func BenchmarkMem(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, k := range []int{1_000, 10_000, 100_000, 1_000_000} {
		var startMem, endMem runtime.MemStats

		runtime.GC()
		runtime.ReadMemStats(&startMem)

		b.Run(strconv.Itoa(k), func(b *testing.B) {
			rt := new(Table[struct{}])
			for b.Loop() {
				rt = new(Table[struct{}])
				for _, pfx := range randomRealWorldPrefixes(prng, k) {
					rt.Insert(pfx, struct{}{})
				}
			}

			runtime.GC()
			runtime.ReadMemStats(&endMem)

			s4 := nodeStatsRec(&rt.root4)
			s6 := nodeStatsRec(&rt.root6)
			stats := statsT{
				s4.pfxs + s6.pfxs,
				s4.childs + s6.childs,
				s4.nodes + s6.nodes,
				s4.leaves + s6.leaves,
				s4.fringes + s6.fringes,
			}

			bytes := float64(endMem.HeapAlloc - startMem.HeapAlloc)
			b.ReportMetric(roundFloat64(bytes/float64(stats.pfxs)), "bytes/route")
			b.ReportMetric(float64(stats.pfxs), "pfxs")
			b.ReportMetric(float64(stats.nodes), "nodes")
			b.ReportMetric(float64(stats.leaves), "leaves")
			b.ReportMetric(float64(stats.fringes), "fringes")
			b.ReportMetric(0, "ns/op")
		})
	}
}

// ##################### helpers ############################

type tableOverlapsTest struct {
	prefix string
	want   bool
}

// checkOverlapsPrefix verifies that the overlaps lookups in tt return the
// expected results on tbl.
func checkOverlapsPrefix(t *testing.T, tbl *Table[int], tests []tableOverlapsTest) {
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
			t.Errorf("Lookup %q got (%v, %v), want (%v, true)", tc.addr, v, ok, tc.want)
		}
		if ok && v != tc.want {
			t.Errorf("Lookup %q got (%v, %v), want (%v, true)", tc.addr, v, ok, tc.want)
		}
	}
}

func checkNumNodes(t *testing.T, tbl *Table[int], want int) {
	t.Helper()

	s4 := nodeStatsRec(&tbl.root4)
	s6 := nodeStatsRec(&tbl.root6)
	nodes := s4.nodes + s6.nodes

	if got := nodes; got != want {
		t.Errorf("wrong table dump, got %d nodes want %d", got, want)
		t.Error(tbl.dumpString())
	}
}

// dumpAsGoldTable, just a helper to compare with golden table.
func (t *Table[V]) dumpAsGoldTable() goldTable[V] {
	var tbl goldTable[V]

	for p, v := range t.AllSorted() {
		tbl = append(tbl, goldTableItem[V]{pfx: p, val: v})
	}

	return tbl
}
