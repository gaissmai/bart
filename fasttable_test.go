// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"fmt"
	"math/rand/v2"
	"net/netip"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/gaissmai/bart/internal/nodes"
)

func TestFastCloneFlat(t *testing.T) {
	t.Parallel()

	cloneFn := nodes.CopyVal[int] // just copy

	tests := []struct {
		name    string
		prepare func() *nodes.FastNode[int]
		check   func(t *testing.T, got, orig *nodes.FastNode[int])
	}{
		{
			name: "nil node returns nil",
			prepare: func() *nodes.FastNode[int] {
				return nil
			},
			check: func(t *testing.T, got, orig *nodes.FastNode[int]) {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
			},
		},
		{
			name: "empty node",
			prepare: func() *nodes.FastNode[int] {
				return &nodes.FastNode[int]{}
			},
			check: func(t *testing.T, got, orig *nodes.FastNode[int]) {
				if got == nil {
					t.Fatal("got is nil")
				}
				if got.PrefixCount() != 0 || got.ChildCount() != 0 {
					t.Errorf("expected empty clone, got %+v", got)
				}
			},
		},
		{
			name: "node with prefix",
			prepare: func() *nodes.FastNode[int] {
				n := &nodes.FastNode[int]{}
				pfx := mpp("8.0.0.0/6")
				val := 42
				n.Insert(pfx, val, 0)
				return n
			},
			check: func(t *testing.T, got, orig *nodes.FastNode[int]) {
				gotBuf := &strings.Builder{}
				origBuf := &strings.Builder{}

				nodes.DumpRec(got, gotBuf, stridePath{}, 0, true, nodes.ShouldPrintValues[int]())
				nodes.DumpRec(orig, origBuf, stridePath{}, 0, true, nodes.ShouldPrintValues[int]())

				if gotBuf.String() != origBuf.String() {
					t.Errorf("dump is different\norig:%sgot:%s", origBuf.String(), gotBuf.String())
				}
			},
		},
		{
			name: "node with prefixes",
			prepare: func() *nodes.FastNode[int] {
				n := &nodes.FastNode[int]{}
				pfx := mpp("8.0.0.0/6")
				val := 6
				n.Insert(pfx, val, 0)

				pfx = mpp("8.0.0.0/8")
				val = 8
				n.Insert(pfx, val, 0)

				pfx = mpp("16.0.0.0/27")
				val = 27
				n.Insert(pfx, val, 0)

				return n
			},
			check: func(t *testing.T, got, orig *nodes.FastNode[int]) {
				gotBuf := &strings.Builder{}
				origBuf := &strings.Builder{}

				nodes.DumpRec(got, gotBuf, stridePath{}, 0, true, nodes.ShouldPrintValues[int]())
				nodes.DumpRec(orig, origBuf, stridePath{}, 0, true, nodes.ShouldPrintValues[int]())

				if gotBuf.String() != origBuf.String() {
					t.Errorf("dump is different\norig:%sgot:%s", origBuf.String(), gotBuf.String())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orig := tt.prepare()
			got := orig.CloneFlat(cloneFn)
			tt.check(t, got, orig)
		})
	}
}

func TestFastInvalid(t *testing.T) {
	t.Parallel()

	tbl1 := new(Fast[any])
	tbl2 := new(Fast[any])
	var zeroPfx netip.Prefix
	var zeroIP netip.Addr

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

func TestFastInsert(t *testing.T) {
	t.Parallel()

	tbl := new(Fast[int])

	// Create a new leaf strideTable, with compressed path
	tbl.Insert(mpp("192.168.0.1/32"), 1)
	checkFastNumNodes(t, tbl, 1)
	checkFastRoutes(t, tbl, []tableTest{
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
	checkFastNumNodes(t, tbl, 4)
	checkFastRoutes(t, tbl, []tableTest{
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
	checkFastNumNodes(t, tbl, 4)
	checkFastRoutes(t, tbl, []tableTest{
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
	checkFastNumNodes(t, tbl, 4)
	checkFastRoutes(t, tbl, []tableTest{
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
	checkFastNumNodes(t, tbl, 4)
	checkFastRoutes(t, tbl, []tableTest{
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
	checkFastNumNodes(t, tbl, 4)
	checkFastRoutes(t, tbl, []tableTest{
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
	checkFastNumNodes(t, tbl, 4)
	checkFastRoutes(t, tbl, []tableTest{
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
	checkFastNumNodes(t, tbl, 5)
	checkFastRoutes(t, tbl, []tableTest{
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
	checkFastNumNodes(t, tbl, 5)
	checkFastRoutes(t, tbl, []tableTest{
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
	checkFastNumNodes(t, tbl, 6)
	checkFastRoutes(t, tbl, []tableTest{
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
	checkFastNumNodes(t, tbl, 21)
	checkFastRoutes(t, tbl, []tableTest{
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
	checkFastNumNodes(t, tbl, 21)
	checkFastRoutes(t, tbl, []tableTest{
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
	checkFastNumNodes(t, tbl, 21)
	checkFastRoutes(t, tbl, []tableTest{
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
	checkFastNumNodes(t, tbl, 21)
	checkFastRoutes(t, tbl, []tableTest{
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
	checkFastNumNodes(t, tbl, 23)
	checkFastRoutes(t, tbl, []tableTest{
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
	checkFastNumNodes(t, tbl, 23)
	checkFastRoutes(t, tbl, []tableTest{
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
	checkFastNumNodes(t, tbl, 25)
	checkFastRoutes(t, tbl, []tableTest{
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
	checkFastNumNodes(t, tbl, 25)
	checkFastRoutes(t, tbl, []tableTest{
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

func TestFastDeleteEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("table_is_empty", func(t *testing.T) {
		t.Parallel()
		prng := rand.New(rand.NewPCG(42, 42))
		// must not panic
		tbl := new(Fast[int])
		checkFastNumNodes(t, tbl, 0)
		tbl.Delete(randomPrefix(prng))
		checkFastNumNodes(t, tbl, 0)
	})

	t.Run("prefix_in_root", func(t *testing.T) {
		t.Parallel()
		// Add/remove prefix from root table.
		tbl := new(Fast[int])
		checkFastNumNodes(t, tbl, 0)

		tbl.Insert(mpp("10.0.0.0/8"), 1)
		checkFastNumNodes(t, tbl, 1)
		checkFastRoutes(t, tbl, []tableTest{
			{"10.0.0.1", 1},
			{"255.255.255.255", -1},
		})
		tbl.Delete(mpp("10.0.0.0/8"))
		checkFastNumNodes(t, tbl, 0)
		checkFastRoutes(t, tbl, []tableTest{
			{"10.0.0.1", -1},
			{"255.255.255.255", -1},
		})
	})

	t.Run("prefix_in_leaf", func(t *testing.T) {
		t.Parallel()
		// Create, then delete a single leaf table.
		tbl := new(Fast[int])
		checkFastNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		checkFastNumNodes(t, tbl, 1)
		checkFastRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"255.255.255.255", -1},
		})

		tbl.Delete(mpp("192.168.0.1/32"))
		checkFastNumNodes(t, tbl, 0)
		checkFastRoutes(t, tbl, []tableTest{
			{"192.168.0.1", -1},
			{"255.255.255.255", -1},
		})
	})

	t.Run("intermediate_no_routes", func(t *testing.T) {
		t.Parallel()
		// Create an intermediate with 2 leaves, then delete one leaf.
		tbl := new(Fast[int])
		checkFastNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		tbl.Insert(mpp("192.180.0.1/32"), 2)
		checkFastNumNodes(t, tbl, 2)
		checkFastRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", 2},
			{"192.40.0.1", -1},
		})

		tbl.Delete(mpp("192.180.0.1/32"))
		checkFastNumNodes(t, tbl, 1)
		checkFastRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", -1},
			{"192.40.0.1", -1},
		})
	})

	t.Run("intermediate_with_route", func(t *testing.T) {
		t.Parallel()
		// Same, but the intermediate carries a route as well.
		tbl := new(Fast[int])
		checkFastNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		tbl.Insert(mpp("192.180.0.1/32"), 2)
		tbl.Insert(mpp("192.0.0.0/10"), 3)

		checkFastNumNodes(t, tbl, 2)
		checkFastRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", 2},
			{"192.40.0.1", 3},
			{"192.255.0.1", -1},
		})

		tbl.Delete(mpp("192.180.0.1/32"))
		checkFastNumNodes(t, tbl, 2)
		checkFastRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", -1},
			{"192.40.0.1", 3},
			{"192.255.0.1", -1},
		})
	})

	t.Run("intermediate_many_leaves", func(t *testing.T) {
		t.Parallel()
		// Intermediate with 3 leaves, then delete one leaf.
		tbl := new(Fast[int])
		checkFastNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		tbl.Insert(mpp("192.180.0.1/32"), 2)
		tbl.Insert(mpp("192.200.0.1/32"), 3)

		checkFastNumNodes(t, tbl, 2)
		checkFastRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", 2},
			{"192.200.0.1", 3},
			{"192.255.0.1", -1},
		})

		tbl.Delete(mpp("192.180.0.1/32"))
		checkFastNumNodes(t, tbl, 2)
		checkFastRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", -1},
			{"192.200.0.1", 3},
			{"192.255.0.1", -1},
		})
	})

	t.Run("nosuchprefix_missing_child", func(t *testing.T) {
		t.Parallel()
		// Delete non-existent prefix
		tbl := new(Fast[int])
		checkFastNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		checkFastNumNodes(t, tbl, 1)
		checkFastRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.255.0.1", -1},
		})

		tbl.Delete(mpp("200.0.0.0/32"))
		checkFastNumNodes(t, tbl, 1)
		checkFastRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.255.0.1", -1},
		})
	})

	t.Run("intermediate_with_deleted_route", func(t *testing.T) {
		t.Parallel()
		// Intermediate node loses its last route and becomes
		// compactable.
		tbl := new(Fast[int])
		checkFastNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		tbl.Insert(mpp("192.168.0.0/22"), 2)
		checkFastNumNodes(t, tbl, 3)
		checkFastRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.168.0.2", 2},
			{"192.255.0.1", -1},
		})

		tbl.Delete(mpp("192.168.0.0/22"))
		checkFastNumNodes(t, tbl, 1)
		checkFastRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.168.0.2", -1},
			{"192.255.0.1", -1},
		})
	})

	t.Run("default_route", func(t *testing.T) {
		t.Parallel()
		tbl := new(Fast[int])
		checkFastNumNodes(t, tbl, 0)

		tbl.Insert(mpp("0.0.0.0/0"), 1)
		tbl.Insert(mpp("::/0"), 1)
		tbl.Delete(mpp("0.0.0.0/0"))

		checkFastNumNodes(t, tbl, 1)
		checkFastRoutes(t, tbl, []tableTest{
			{"1.2.3.4", -1},
			{"::1", 1},
		})
	})

	t.Run("path compressed purge", func(t *testing.T) {
		t.Parallel()
		tbl := new(Fast[int])
		checkFastNumNodes(t, tbl, 0)

		tbl.Insert(mpp("10.10.0.0/17"), 1)
		tbl.Insert(mpp("10.20.0.0/17"), 2)
		checkFastNumNodes(t, tbl, 2)

		tbl.Delete(mpp("10.20.0.0/17"))
		checkFastNumNodes(t, tbl, 1)

		tbl.Delete(mpp("10.10.0.0/17"))
		checkFastNumNodes(t, tbl, 0)
	})
}

// TestFastModifySemantics
func TestFastModifySemantics(t *testing.T) {
	t.Parallel()

	type args struct {
		pfx netip.Prefix
		cb  func(val int, found bool) (_ int, del bool)
	}

	tests := []struct {
		name      string
		prepare   map[netip.Prefix]int // entries to pre-populate the table
		args      args
		want      int
		finalData map[netip.Prefix]int // expected table contents after the operation
	}{
		{
			name:    "Delete existing entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42, mpp("2001:db8::/32"): 4242},
			args: args{
				pfx: mpp("10.0.0.0/8"),
				cb:  func(val int, found bool) (_ int, del bool) { return 0, true },
			},
			want:      42,
			finalData: map[netip.Prefix]int{mpp("2001:db8::/32"): 4242},
		},

		{
			name:    "Insert new entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42},
			args: args{
				pfx: mpp("2001:db8::/32"),
				cb:  func(val int, found bool) (_ int, del bool) { return 4242, false },
			},
			want:      4242,
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42, mpp("2001:db8::/32"): 4242},
		},

		{
			// For update, the callback gets oldVal, returns newVal, but Modify returns oldVal
			name:    "Update existing entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42, mpp("2001:db8::/32"): 4242},
			args: args{
				pfx: mpp("10.0.0.0/8"),
				cb:  func(val int, found bool) (_ int, del bool) { return -1, false },
			},
			want:      42,
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): -1, mpp("2001:db8::/32"): 4242},
		},

		{
			name:    "No-op on missing entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42},
			args: args{
				pfx: mpp("2001:db8::/32"),
				cb:  func(val int, found bool) (_ int, del bool) { return 0, true },
			},
			want:      0,
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42},
		},
	}

	for _, tt := range tests {
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
	}
}

func TestFastUpdateCompare(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, n)
	fast := new(Fast[int])

	gold := new(goldTable[int])
	gold.insertMany(pfxs)

	// Update as insert
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

func TestFastContainsCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, n)

	gold := new(goldTable[int])
	gold.insertMany(pfxs)

	fast := new(Fast[int])
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

func TestFastLookupCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, n)

	gold := new(goldTable[int])
	gold.insertMany(pfxs)

	fast := new(Fast[int])
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

func TestFastInsertShuffled(t *testing.T) {
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

		rt1 := new(Fast[int])
		rt2 := new(Fast[int])

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

func TestFastDeleteCompare(t *testing.T) {
	// Create large route tables repeatedly, delete half of their
	// prefixes, and compare Table's behavior to a naive and slow but
	// correct implementation.
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))

	n := workLoadN()

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

	fast := new(Fast[int])
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

func TestFastDeleteShuffled(t *testing.T) {
	// The order in which you delete prefixes from a route table
	// should not matter, as long as you're deleting the same set of
	// routes.
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))

	n := workLoadN()

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

		rt1 := new(Fast[int])

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

		rt2 := new(Fast[int])

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

		if rt1.dumpString() != rt2.dumpString() {
			t.Fatal("shuffled table has different dumpString representation")
		}
	}
}

func TestFastDeleteIsReverseOfInsert(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))
	// Insert N prefixes, then delete those same prefixes in reverse
	// order. Each deletion should exactly undo the internal structure
	// changes that each insert did.
	const N = 10_000

	tbl := new(Fast[int])
	want := tbl.dumpString()

	prefixes := randomPrefixes(prng, N)

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

func TestFastDeleteButOne(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))
	// Insert N prefixes, then delete all but one
	const N = 100

	for range 1_000 {

		tbl := new(Fast[int])
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

		stats4 := nodes.StatsRec(&tbl.root4)
		stats6 := nodes.StatsRec(&tbl.root6)

		if nodes := stats4.Nodes + stats6.Nodes; nodes != 1 {
			t.Fatalf("delete but one, want nodes: 1, got: %d\n%s", nodes, tbl.dumpString())
		}

		sum := stats4.Pfxs + stats4.Leaves + stats4.Fringes +
			stats6.Pfxs + stats6.Leaves + stats6.Fringes

		if sum != 1 {
			t.Fatalf("delete but one, only one item must be left, but: %d\n%s", sum, tbl.dumpString())
		}
	}
}

func TestFastGet(t *testing.T) {
	t.Parallel()

	t.Run("empty table", func(t *testing.T) {
		t.Parallel()
		prng := rand.New(rand.NewPCG(42, 42))

		rt := new(Fast[int])
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

	rt := new(Fast[int])
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

func TestFastGetCompare(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, n)

	gold := new(goldTable[int])
	gold.insertMany(pfxs)

	fast := new(Fast[int])
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

func TestFastCloneEdgeCases(t *testing.T) {
	t.Parallel()

	tbl := new(Fast[int])
	clone := tbl.Clone()
	if tbl.dumpString() != clone.dumpString() {
		t.Errorf("empty Clone: got:\n%swant:\n%s", clone.dumpString(), tbl.dumpString())
	}

	tbl.Insert(mpp("10.0.0.1/32"), 1)
	tbl.Insert(mpp("::1/128"), 1)
	clone = tbl.Clone()
	if tbl.dumpString() != clone.dumpString() {
		t.Errorf("Clone: got:\n%swant:\n%s", clone.dumpString(), tbl.dumpString())
	}

	// overwrite value
	tbl.Insert(mpp("::1/128"), 2)
	if tbl.dumpString() == clone.dumpString() {
		t.Errorf("overwrite, clone must be different: clone:\n%sorig:\n%s", clone.dumpString(), tbl.dumpString())
	}

	tbl.Delete(mpp("10.0.0.1/32"))
	if tbl.dumpString() == clone.dumpString() {
		t.Errorf("delete, clone must be different: clone:\n%sorig:\n%s", clone.dumpString(), tbl.dumpString())
	}
}

func TestFastClone(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))

	pfxs := randomPrefixes(prng, 2)

	golden := new(Fast[int])
	tbl := new(Fast[int])

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

func TestFastCloneShallow(t *testing.T) {
	t.Parallel()

	tbl := new(Fast[*int])
	clone := tbl.Clone()
	if tbl.dumpString() != clone.dumpString() {
		t.Errorf("empty Clone: got:\n%swant:\n%s", clone.dumpString(), tbl.dumpString())
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

func TestFastCloneDeep(t *testing.T) {
	t.Parallel()

	tbl := new(Fast[*MyInt])
	clone := tbl.Clone()
	if tbl.dumpString() != clone.dumpString() {
		t.Errorf("empty Clone: got:\n%swant:\n%s", clone.dumpString(), tbl.dumpString())
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

// ############ benchmarks ################################

func BenchmarkFastTableDelete(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, n := range []int{1_000, 10_000, 100_000, 1_000_000} {
		pfxs := randomPrefixes(prng, n)

		b.Run(fmt.Sprintf("mutable from_%d", n), func(b *testing.B) {
			for b.Loop() {
				b.StopTimer()
				rt := new(Fast[*MyInt])

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
			b.Skip("Fast.DeletePersist not yet implemented")

			/*
				for b.Loop() {
					b.StopTimer()
					rt := new(Fast[*MyInt])

					for i, route := range pfxs {
						myInt := MyInt(i)
						rt.Insert(route.pfx, &myInt)
					}
					b.StartTimer()

					for _, route := range pfxs {
						rt, _, _ = rt.DeletePersist(route.pfx)
					}
				}
				b.ReportMetric(float64(b.Elapsed())/float64(b.N)/float64(len(pfxs)), "ns/route")
				b.ReportMetric(0, "ns/op")
			*/
		})
	}
}

func BenchmarkFastTableGet(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		for _, nroutes := range benchRouteCount {
			rt := new(Fast[int])
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

func BenchmarkFastTableLPM(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		for _, nroutes := range benchRouteCount {
			rt := new(Fast[int])
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
		}
	}
}

func BenchmarkFastMemIP4(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, k := range []int{1_000, 10_000, 100_000, 1_000_000} {
		var startMem, endMem runtime.MemStats

		runtime.GC()
		runtime.ReadMemStats(&startMem)

		b.Run(strconv.Itoa(k), func(b *testing.B) {
			rt := new(Fast[any])
			for _, pfx := range randomRealWorldPrefixes4(prng, k) {
				rt.Insert(pfx, nil)
			}

			runtime.GC()
			runtime.ReadMemStats(&endMem)

			stats := nodes.StatsRec(&rt.root4)

			bytes := float64(endMem.HeapAlloc - startMem.HeapAlloc)
			b.ReportMetric(roundFloat64(bytes/float64(rt.Size())), "bytes/route")

			b.ReportMetric(float64(stats.Nodes), "node")
			b.ReportMetric(float64(stats.Pfxs), "pfxs")
			b.ReportMetric(float64(stats.Leaves), "leaf")
			b.ReportMetric(float64(stats.Fringes), "fringe")
			b.ReportMetric(0, "ns/op")
		})
	}
}

func BenchmarkFastMemIP6(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, k := range []int{1_000, 10_000, 100_000, 1_000_000} {
		var startMem, endMem runtime.MemStats

		runtime.GC()
		runtime.ReadMemStats(&startMem)

		b.Run(strconv.Itoa(k), func(b *testing.B) {
			rt := new(Fast[any])
			for _, pfx := range randomRealWorldPrefixes6(prng, k) {
				rt.Insert(pfx, nil)
			}

			runtime.GC()
			runtime.ReadMemStats(&endMem)

			stats := nodes.StatsRec(&rt.root6)

			bytes := float64(endMem.HeapAlloc - startMem.HeapAlloc)
			b.ReportMetric(roundFloat64(bytes/float64(rt.Size())), "bytes/route")

			b.ReportMetric(float64(stats.Nodes), "node")
			b.ReportMetric(float64(stats.Pfxs), "pfxs")
			b.ReportMetric(float64(stats.Leaves), "leaf")
			b.ReportMetric(float64(stats.Fringes), "fringe")
			b.ReportMetric(0, "ns/op")
		})
	}
}

func BenchmarkFastMem(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, k := range []int{1_000, 10_000, 100_000, 1_000_000} {
		var startMem, endMem runtime.MemStats

		runtime.GC()
		runtime.ReadMemStats(&startMem)

		b.Run(strconv.Itoa(k), func(b *testing.B) {
			rt := new(Fast[any])
			for _, pfx := range randomRealWorldPrefixes(prng, k) {
				rt.Insert(pfx, nil)
			}

			runtime.GC()
			runtime.ReadMemStats(&endMem)

			s4 := nodes.StatsRec(&rt.root4)
			s6 := nodes.StatsRec(&rt.root6)
			stats := nodes.StatsT{
				Pfxs:    s4.Pfxs + s6.Pfxs,
				Childs:  s4.Childs + s6.Childs,
				Nodes:   s4.Nodes + s6.Nodes,
				Leaves:  s4.Leaves + s6.Leaves,
				Fringes: s4.Fringes + s6.Fringes,
			}

			bytes := float64(endMem.HeapAlloc - startMem.HeapAlloc)
			b.ReportMetric(roundFloat64(bytes/float64(rt.Size())), "bytes/route")

			b.ReportMetric(float64(stats.Nodes), "node")
			b.ReportMetric(float64(stats.Pfxs), "pfxs")
			b.ReportMetric(float64(stats.Leaves), "leaf")
			b.ReportMetric(float64(stats.Fringes), "fringe")
			b.ReportMetric(0, "ns/op")
		})
	}
}

func BenchmarkFastFullTableMemory4(b *testing.B) {
	var startMem, endMem runtime.MemStats
	nRoutes := len(routes4)

	runtime.GC()
	runtime.ReadMemStats(&startMem)

	b.Run(fmt.Sprintf("Table[]: %d", nRoutes), func(b *testing.B) {
		rt := new(Fast[any])
		for _, route := range routes4 {
			rt.Insert(route.CIDR, nil)
		}

		runtime.GC()
		runtime.ReadMemStats(&endMem)

		stats := nodes.StatsRec(&rt.root4)

		bytes := float64(endMem.HeapAlloc - startMem.HeapAlloc)
		b.ReportMetric(roundFloat64(bytes/float64(rt.Size())), "bytes/route")

		b.ReportMetric(float64(stats.Pfxs), "pfxs")
		b.ReportMetric(float64(stats.Nodes), "nodes")
		b.ReportMetric(float64(stats.Leaves), "leaves")
		b.ReportMetric(float64(stats.Fringes), "fringes")
		b.ReportMetric(0, "ns/op")
	})
}

func BenchmarkFastFullTableMemory6(b *testing.B) {
	var startMem, endMem runtime.MemStats

	runtime.GC()
	runtime.ReadMemStats(&startMem)

	nRoutes := len(routes6)

	b.Run(fmt.Sprintf("Table[]: %d", nRoutes), func(b *testing.B) {
		rt := new(Fast[any])
		for _, route := range routes6 {
			rt.Insert(route.CIDR, nil)
		}

		runtime.GC()
		runtime.ReadMemStats(&endMem)

		stats := nodes.StatsRec(&rt.root6)

		bytes := float64(endMem.HeapAlloc - startMem.HeapAlloc)
		b.ReportMetric(roundFloat64(bytes/float64(rt.Size())), "bytes/route")

		b.ReportMetric(float64(stats.Pfxs), "pfxs")
		b.ReportMetric(float64(stats.Nodes), "nodes")
		b.ReportMetric(float64(stats.Leaves), "leaves")
		b.ReportMetric(float64(stats.Fringes), "fringes")
		b.ReportMetric(0, "ns/op")
	})
}

func BenchmarkFastFullTableMemory(b *testing.B) {
	var startMem, endMem runtime.MemStats

	runtime.GC()
	runtime.ReadMemStats(&startMem)

	nRoutes := len(routes)

	b.Run(fmt.Sprintf("Table[]: %d", nRoutes), func(b *testing.B) {
		rt := new(Fast[any])
		for _, route := range routes {
			rt.Insert(route.CIDR, nil)
		}

		runtime.GC()
		runtime.ReadMemStats(&endMem)

		s4 := nodes.StatsRec(&rt.root4)
		s6 := nodes.StatsRec(&rt.root6)
		stats := nodes.StatsT{
			Pfxs:    s4.Pfxs + s6.Pfxs,
			Childs:  s4.Childs + s6.Childs,
			Nodes:   s4.Nodes + s6.Nodes,
			Leaves:  s4.Leaves + s6.Leaves,
			Fringes: s4.Fringes + s6.Fringes,
		}

		bytes := float64(endMem.HeapAlloc - startMem.HeapAlloc)
		b.ReportMetric(roundFloat64(bytes/float64(rt.Size())), "bytes/route")

		b.ReportMetric(float64(stats.Pfxs), "pfxs")
		b.ReportMetric(float64(stats.Nodes), "nodes")
		b.ReportMetric(float64(stats.Leaves), "leaves")
		b.ReportMetric(float64(stats.Fringes), "fringes")
		b.ReportMetric(0, "ns/op")
	})
}

func BenchmarkFastFullMatch4(b *testing.B) {
	rt := new(Fast[any])

	for _, route := range routes {
		rt.Insert(route.CIDR, nil)
	}

	b.Run("Contains", func(b *testing.B) {
		for b.Loop() {
			rt.Contains(matchIP4)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		for b.Loop() {
			rt.Lookup(matchIP4)
		}
	})
}

func BenchmarkFastFullMatch6(b *testing.B) {
	rt := new(Fast[any])

	for _, route := range routes {
		rt.Insert(route.CIDR, nil)
	}

	b.Run("Contains", func(b *testing.B) {
		for b.Loop() {
			rt.Contains(matchIP6)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		for b.Loop() {
			rt.Lookup(matchIP6)
		}
	})
}

func BenchmarkFastFullMiss4(b *testing.B) {
	rt := new(Fast[any])

	for _, route := range routes {
		rt.Insert(route.CIDR, nil)
	}

	b.Run("Contains", func(b *testing.B) {
		for b.Loop() {
			rt.Contains(missIP4)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		for b.Loop() {
			rt.Lookup(missIP4)
		}
	})
}

func BenchmarkFastFullMiss6(b *testing.B) {
	rt := new(Fast[any])

	for _, route := range routes {
		rt.Insert(route.CIDR, nil)
	}

	b.Run("Contains", func(b *testing.B) {
		for b.Loop() {
			rt.Contains(missIP6)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		for b.Loop() {
			rt.Lookup(missIP6)
		}
	})
}

func checkFastNumNodes(t *testing.T, tbl *Fast[int], want int) {
	t.Helper()

	s4 := nodes.StatsRec(&tbl.root4)
	s6 := nodes.StatsRec(&tbl.root6)
	nodes := s4.Nodes + s6.Nodes

	if got := nodes; got != want {
		t.Errorf("wrong table dump, got %d nodes want %d", got, want)
		t.Error(tbl.dumpString())
	}
}

func checkFastRoutes(t *testing.T, tbl *Fast[int], tt []tableTest) {
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
