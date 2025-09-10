package bart

import (
	"fmt"
	"math/rand/v2"
	"net/netip"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestFatCloneFlat(t *testing.T) {
	t.Parallel()

	cloneFn := copyVal[int] // just copy

	tests := []struct {
		name    string
		prepare func() *fatNode[int]
		check   func(t *testing.T, got, orig *fatNode[int])
	}{
		{
			name: "nil node returns nil",
			prepare: func() *fatNode[int] {
				return nil
			},
			check: func(t *testing.T, got, orig *fatNode[int]) {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
			},
		},
		{
			name: "empty node",
			prepare: func() *fatNode[int] {
				return &fatNode[int]{}
			},
			check: func(t *testing.T, got, orig *fatNode[int]) {
				if got == nil {
					t.Fatal("got is nil")
				}
				if got.prefixCount() != 0 || got.childCount() != 0 {
					t.Errorf("expected empty clone, got %+v", got)
				}
			},
		},
		{
			name: "node with prefix",
			prepare: func() *fatNode[int] {
				n := &fatNode[int]{}
				pfx := mpp("8.0.0.0/6")
				val := 42
				n.insertAtDepth(pfx, val, 0)
				return n
			},
			check: func(t *testing.T, got, orig *fatNode[int]) {
				gotBuf := &strings.Builder{}
				origBuf := &strings.Builder{}

				got.dumpRec(gotBuf, stridePath{}, 0, true)
				orig.dumpRec(origBuf, stridePath{}, 0, true)

				if gotBuf.String() != origBuf.String() {
					t.Errorf("dump is different\norig:%sgot:%s", origBuf.String(), gotBuf.String())
				}
			},
		},
		{
			name: "node with prefixes",
			prepare: func() *fatNode[int] {
				n := &fatNode[int]{}
				pfx := mpp("8.0.0.0/6")
				val := 6
				n.insertAtDepth(pfx, val, 0)

				pfx = mpp("8.0.0.0/8")
				val = 8
				n.insertAtDepth(pfx, val, 0)

				pfx = mpp("16.0.0.0/27")
				val = 27
				n.insertAtDepth(pfx, val, 0)

				return n
			},
			check: func(t *testing.T, got, orig *fatNode[int]) {
				gotBuf := &strings.Builder{}
				origBuf := &strings.Builder{}

				got.dumpRec(gotBuf, stridePath{}, 0, true)
				orig.dumpRec(origBuf, stridePath{}, 0, true)

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
			got := orig.cloneFlat(cloneFn)
			tt.check(t, got, orig)
		})
	}
}

func TestFatInvalid(t *testing.T) {
	t.Parallel()

	tbl := new(Fat[any])
	var zeroPfx netip.Prefix
	var zeroIP netip.Addr
	var testname string

	testname = "Insert"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s should not panic on invalid prefix input", testname)
			}
		}(testname)

		tbl.Insert(zeroPfx, nil)
	})

	testname = "Delete"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s should not panic on invalid prefix input", testname)
			}
		}(testname)

		_, _ = tbl.Delete(zeroPfx)
	})

	testname = "Get"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s should not panic on invalid prefix input", testname)
			}
		}(testname)

		_, _ = tbl.Get(zeroPfx)
	})

	testname = "Contains"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s should not panic on invalid IP input", testname)
			}
		}(testname)

		if tbl.Contains(zeroIP) != false {
			t.Errorf("%s returns true on invalid IP input, expected false", testname)
		}
	})

	testname = "Lookup"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s should not panic on invalid IP input", testname)
			}
		}(testname)

		_, got := tbl.Lookup(zeroIP)
		if got != false {
			t.Errorf("%s returns true on invalid IP input, expected false", testname)
		}
	})
}

func TestFatInsert(t *testing.T) {
	t.Parallel()

	tbl := new(Fat[int])

	// Create a new leaf strideTable, with compressed path
	tbl.Insert(mpp("192.168.0.1/32"), 1)
	checkFatNumNodes(t, tbl, 1)
	checkFatRoutes(t, tbl, []tableTest{
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
	checkFatNumNodes(t, tbl, 4)
	checkFatRoutes(t, tbl, []tableTest{
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
	checkFatNumNodes(t, tbl, 4)
	checkFatRoutes(t, tbl, []tableTest{
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
	checkFatNumNodes(t, tbl, 4)
	checkFatRoutes(t, tbl, []tableTest{
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
	checkFatNumNodes(t, tbl, 4)
	checkFatRoutes(t, tbl, []tableTest{
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
	checkFatNumNodes(t, tbl, 4)
	checkFatRoutes(t, tbl, []tableTest{
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
	checkFatNumNodes(t, tbl, 4)
	checkFatRoutes(t, tbl, []tableTest{
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
	checkFatNumNodes(t, tbl, 5)
	checkFatRoutes(t, tbl, []tableTest{
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
	checkFatNumNodes(t, tbl, 5)
	checkFatRoutes(t, tbl, []tableTest{
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
	checkFatNumNodes(t, tbl, 6)
	checkFatRoutes(t, tbl, []tableTest{
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
	checkFatNumNodes(t, tbl, 21)
	checkFatRoutes(t, tbl, []tableTest{
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
	checkFatNumNodes(t, tbl, 21)
	checkFatRoutes(t, tbl, []tableTest{
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
	checkFatNumNodes(t, tbl, 21)
	checkFatRoutes(t, tbl, []tableTest{
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
	checkFatNumNodes(t, tbl, 21)
	checkFatRoutes(t, tbl, []tableTest{
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
	checkFatNumNodes(t, tbl, 23)
	checkFatRoutes(t, tbl, []tableTest{
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
	checkFatNumNodes(t, tbl, 23)
	checkFatRoutes(t, tbl, []tableTest{
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
	checkFatNumNodes(t, tbl, 25)
	checkFatRoutes(t, tbl, []tableTest{
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
	checkFatNumNodes(t, tbl, 25)
	checkFatRoutes(t, tbl, []tableTest{
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

func TestFatDeleteEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("table_is_empty", func(t *testing.T) {
		t.Parallel()
		//nolint:gosec
		prng := rand.New(rand.NewPCG(42, 42))
		// must not panic
		tbl := new(Fat[int])
		checkFatNumNodes(t, tbl, 0)
		tbl.Delete(randomPrefix(prng))
		checkFatNumNodes(t, tbl, 0)
	})

	t.Run("prefix_in_root", func(t *testing.T) {
		t.Parallel()
		// Add/remove prefix from root table.
		tbl := new(Fat[int])
		checkFatNumNodes(t, tbl, 0)

		tbl.Insert(mpp("10.0.0.0/8"), 1)
		checkFatNumNodes(t, tbl, 1)
		checkFatRoutes(t, tbl, []tableTest{
			{"10.0.0.1", 1},
			{"255.255.255.255", -1},
		})
		tbl.Delete(mpp("10.0.0.0/8"))
		checkFatNumNodes(t, tbl, 0)
		checkFatRoutes(t, tbl, []tableTest{
			{"10.0.0.1", -1},
			{"255.255.255.255", -1},
		})
	})

	t.Run("prefix_in_leaf", func(t *testing.T) {
		t.Parallel()
		// Create, then delete a single leaf table.
		tbl := new(Fat[int])
		checkFatNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		checkFatNumNodes(t, tbl, 1)
		checkFatRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"255.255.255.255", -1},
		})

		tbl.Delete(mpp("192.168.0.1/32"))
		checkFatNumNodes(t, tbl, 0)
		checkFatRoutes(t, tbl, []tableTest{
			{"192.168.0.1", -1},
			{"255.255.255.255", -1},
		})
	})

	t.Run("intermediate_no_routes", func(t *testing.T) {
		t.Parallel()
		// Create an intermediate with 2 leaves, then delete one leaf.
		tbl := new(Fat[int])
		checkFatNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		tbl.Insert(mpp("192.180.0.1/32"), 2)
		checkFatNumNodes(t, tbl, 2)
		checkFatRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", 2},
			{"192.40.0.1", -1},
		})

		tbl.Delete(mpp("192.180.0.1/32"))
		checkFatNumNodes(t, tbl, 1)
		checkFatRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", -1},
			{"192.40.0.1", -1},
		})
	})

	t.Run("intermediate_with_route", func(t *testing.T) {
		t.Parallel()
		// Same, but the intermediate carries a route as well.
		tbl := new(Fat[int])
		checkFatNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		tbl.Insert(mpp("192.180.0.1/32"), 2)
		tbl.Insert(mpp("192.0.0.0/10"), 3)

		checkFatNumNodes(t, tbl, 2)
		checkFatRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", 2},
			{"192.40.0.1", 3},
			{"192.255.0.1", -1},
		})

		tbl.Delete(mpp("192.180.0.1/32"))
		checkFatNumNodes(t, tbl, 2)
		checkFatRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", -1},
			{"192.40.0.1", 3},
			{"192.255.0.1", -1},
		})
	})

	t.Run("intermediate_many_leaves", func(t *testing.T) {
		t.Parallel()
		// Intermediate with 3 leaves, then delete one leaf.
		tbl := new(Fat[int])
		checkFatNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		tbl.Insert(mpp("192.180.0.1/32"), 2)
		tbl.Insert(mpp("192.200.0.1/32"), 3)

		checkFatNumNodes(t, tbl, 2)
		checkFatRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", 2},
			{"192.200.0.1", 3},
			{"192.255.0.1", -1},
		})

		tbl.Delete(mpp("192.180.0.1/32"))
		checkFatNumNodes(t, tbl, 2)
		checkFatRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", -1},
			{"192.200.0.1", 3},
			{"192.255.0.1", -1},
		})
	})

	t.Run("nosuchprefix_missing_child", func(t *testing.T) {
		t.Parallel()
		// Delete non-existent prefix
		tbl := new(Fat[int])
		checkFatNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		checkFatNumNodes(t, tbl, 1)
		checkFatRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.255.0.1", -1},
		})

		tbl.Delete(mpp("200.0.0.0/32"))
		checkFatNumNodes(t, tbl, 1)
		checkFatRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.255.0.1", -1},
		})
	})

	t.Run("intermediate_with_deleted_route", func(t *testing.T) {
		t.Parallel()
		// Intermediate node loses its last route and becomes
		// compactable.
		tbl := new(Fat[int])
		checkFatNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		tbl.Insert(mpp("192.168.0.0/22"), 2)
		checkFatNumNodes(t, tbl, 3)
		checkFatRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.168.0.2", 2},
			{"192.255.0.1", -1},
		})

		tbl.Delete(mpp("192.168.0.0/22"))
		checkFatNumNodes(t, tbl, 1)
		checkFatRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.168.0.2", -1},
			{"192.255.0.1", -1},
		})
	})

	t.Run("default_route", func(t *testing.T) {
		t.Parallel()
		tbl := new(Fat[int])
		checkFatNumNodes(t, tbl, 0)

		tbl.Insert(mpp("0.0.0.0/0"), 1)
		tbl.Insert(mpp("::/0"), 1)
		tbl.Delete(mpp("0.0.0.0/0"))

		checkFatNumNodes(t, tbl, 1)
		checkFatRoutes(t, tbl, []tableTest{
			{"1.2.3.4", -1},
			{"::1", 1},
		})
	})

	t.Run("path compressed purge", func(t *testing.T) {
		t.Parallel()
		tbl := new(Fat[int])
		checkFatNumNodes(t, tbl, 0)

		tbl.Insert(mpp("10.10.0.0/17"), 1)
		tbl.Insert(mpp("10.20.0.0/17"), 2)
		checkFatNumNodes(t, tbl, 2)

		tbl.Delete(mpp("10.20.0.0/17"))
		checkFatNumNodes(t, tbl, 1)

		tbl.Delete(mpp("10.10.0.0/17"))
		checkFatNumNodes(t, tbl, 0)
	})
}

// TestFatModifySemantics
//
// Operation | cb-input        | cb-return       | Modify-return
// ---------------------------------------------------------------
// No-op:    | (zero,   false) | (_,      true)  | (zero,   false)
// Insert:   | (zero,   false) | (newVal, false) | (newVal, false)
// Update:   | (oldVal, true)  | (newVal, false) | (oldVal, false)
// Delete:   | (oldVal, true)  | (_,      true)  | (oldVal, true)
func TestFatModifySemantics(t *testing.T) {
	t.Parallel()

	type args struct {
		pfx netip.Prefix
		cb  func(val int, found bool) (_ int, del bool)
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
				pfx: mpp("10.0.0.0/8"),
				cb:  func(val int, found bool) (_ int, del bool) { return 0, true },
			},
			want:      want{val: 42, deleted: true},
			finalData: map[netip.Prefix]int{mpp("2001:db8::/32"): 4242},
		},

		{
			name:    "Insert new entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42},
			args: args{
				pfx: mpp("2001:db8::/32"),
				cb:  func(val int, found bool) (_ int, del bool) { return 4242, false },
			},
			want:      want{val: 4242, deleted: false},
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
			want:      want{val: 42, deleted: false},
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): -1, mpp("2001:db8::/32"): 4242},
		},

		{
			name:    "No-op on missing entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42},
			args: args{
				pfx: mpp("2001:db8::/32"),
				cb:  func(val int, found bool) (_ int, del bool) { return 0, true },
			},
			want:      want{val: 0, deleted: false},
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rt := new(Fat[int])

			// Insert initial entries using Modify
			for pfx, v := range tt.prepare {
				rt.Modify(pfx, func(_ int, _ bool) (_ int, del bool) { return v, false })
			}

			got, deleted := rt.Modify(tt.args.pfx, tt.args.cb)
			if got != tt.want.val || deleted != tt.want.deleted {
				t.Errorf("[%s] Modify() = (%v, %v), want (%v, %v)", tt.name, got, deleted, tt.want.val, tt.want.deleted)
			}

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

func TestFatUpdateCompare(t *testing.T) {
	t.Parallel()

	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, 10_000)
	fast := new(Fat[int])
	gold := new(goldTable[int]).insertMany(pfxs)

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

func TestFatContainsCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()
	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, 10_000)

	gold := new(goldTable[int]).insertMany(pfxs)
	fast := new(Fat[int])

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	for range 10_000 {
		a := randomAddr(prng)

		_, goldOK := gold.lookup(a)
		fastOK := fast.Contains(a)

		if goldOK != fastOK {
			t.Fatalf("Contains(%q) = %v, want %v", a, fastOK, goldOK)
		}
	}
}

func TestFatLookupCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()
	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, 10_000)

	fast := new(Fat[int])
	gold := new(goldTable[int]).insertMany(pfxs)

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	seenVals4 := map[int]bool{}
	seenVals6 := map[int]bool{}

	for range 10_000 {
		a := randomAddr(prng)

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

func TestFatInsertShuffled(t *testing.T) {
	// The order in which you insert prefixes into a route table
	// should not matter, as long as you're inserting the same set of
	// routes.
	t.Parallel()

	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, 1000)

	for range 10 {
		pfxs2 := append([]goldTableItem[int](nil), pfxs...)
		rand.Shuffle(len(pfxs2), func(i, j int) { pfxs2[i], pfxs2[j] = pfxs2[j], pfxs2[i] })

		addrs := make([]netip.Addr, 0, 10_000)
		for range 10_000 {
			addrs = append(addrs, randomAddr(prng))
		}

		rt1 := new(Fat[int])
		rt2 := new(Fat[int])

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

func TestFatDeleteCompare(t *testing.T) {
	// Create large route tables repeatedly, delete half of their
	// prefixes, and compare Table's behavior to a naive and slow but
	// correct implementation.
	t.Parallel()
	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))

	const (
		numPrefixes  = 10_000 // total prefixes to insert (test deletes 50% of them)
		numPerFamily = numPrefixes / 2
		deleteCut    = numPerFamily / 2
		numProbes    = 10_000 // random addr lookups to do
	)

	// We have to do this little dance instead of just using allPrefixes,
	// because we want pfxs and toDelete to be non-overlapping sets.
	all4, all6 := randomPrefixes4(prng, numPerFamily), randomPrefixes6(prng, numPerFamily)

	pfxs := append([]goldTableItem[int](nil), all4[:deleteCut]...)
	pfxs = append(pfxs, all6[:deleteCut]...)

	toDelete := append([]goldTableItem[int](nil), all4[deleteCut:]...)
	toDelete = append(toDelete, all6[deleteCut:]...)

	fast := new(Fat[int])
	gold := new(goldTable[int]).insertMany(pfxs)

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
		a := randomAddr(prng)

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

func TestFatDeleteShuffled(t *testing.T) {
	// The order in which you delete prefixes from a route table
	// should not matter, as long as you're deleting the same set of
	// routes.
	t.Parallel()
	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))

	const (
		numPrefixes  = 10_000 // prefixes to insert (test deletes 50% of them)
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

		rt1 := new(Fat[int])

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
		rand.Shuffle(len(toDelete2), func(i, j int) { toDelete2[i], toDelete2[j] = toDelete2[j], toDelete2[i] })

		rt2 := new(Fat[int])

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

func TestFatDeleteIsReverseOfInsert(t *testing.T) {
	t.Parallel()
	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))
	// Insert N prefixes, then delete those same prefixes in reverse
	// order. Each deletion should exactly undo the internal structure
	// changes that each insert did.
	const N = 10_000

	tbl := new(Fat[int])
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

func TestFatDeleteButOne(t *testing.T) {
	t.Parallel()
	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))
	// Insert N prefixes, then delete all but one
	const N = 100

	for range 1_000 {

		tbl := new(Fat[int])
		prefixes := randomPrefixes(prng, N)

		for _, p := range prefixes {
			tbl.Insert(p.pfx, p.val)
		}

		// shuffle the prefixes
		rand.Shuffle(N, func(i, j int) {
			prefixes[i], prefixes[j] = prefixes[j], prefixes[i]
		})

		for i, p := range prefixes {
			// skip the first
			if i == 0 {
				continue
			}
			tbl.Delete(p.pfx)
		}

		stats4 := tbl.root4.nodeStatsRec()
		stats6 := tbl.root6.nodeStatsRec()

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

func TestFatDelete(t *testing.T) {
	t.Parallel()
	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))
	// Insert N prefixes, then delete those same prefixes in shuffled
	// order.
	const N = 10_000

	tbl := new(Fat[int])
	prefixes := randomPrefixes(prng, N)

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
		val, ok := tbl.Delete(p.pfx)

		if !ok {
			t.Errorf("Delete, expected true, got %v", ok)
		}

		if val != want {
			t.Errorf("Delete, expected %v, got %v", want, val)
		}

		val, ok = tbl.Delete(p.pfx)
		if ok {
			t.Errorf("Delete, expected false, got (%v, %v)", val, ok)
		}
	}
}

func TestFatGet(t *testing.T) {
	t.Parallel()

	t.Run("empty table", func(t *testing.T) {
		t.Parallel()
		//nolint:gosec
		prng := rand.New(rand.NewPCG(42, 42))

		rt := new(Fat[int])
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

	rt := new(Fat[int])
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

func TestFatGetCompare(t *testing.T) {
	t.Parallel()
	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))

	pfxs := randomPrefixes(prng, 10_000)
	fast := new(Fat[int])
	gold := new(goldTable[int]).insertMany(pfxs)

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

func TestFatCloneEdgeCases(t *testing.T) {
	t.Parallel()

	tbl := new(Fat[int])
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

func TestFatClone(t *testing.T) {
	t.Parallel()
	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))

	pfxs := randomPrefixes(prng, 2)

	golden := new(Fat[int])
	tbl := new(Fat[int])

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

func TestFatCloneShallow(t *testing.T) {
	t.Parallel()

	tbl := new(Fat[*int])
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

func TestFatCloneDeep(t *testing.T) {
	t.Parallel()

	tbl := new(Fat[*MyInt])
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

func BenchmarkFatTableDelete(b *testing.B) {
	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))

	for _, n := range benchRouteCount {
		rt := new(Fat[*MyInt])

		for i, route := range randomPrefixes(prng, n) {
			myInt := MyInt(i)
			rt.Insert(route.pfx, &myInt)
		}

		probe := randomPrefix(prng)

		b.Run(fmt.Sprintf("mutable from_%d", n), func(b *testing.B) {
			for b.Loop() {
				rt.Delete(probe)
			}
		})
	}
}

func BenchmarkFatTableGet(b *testing.B) {
	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		for _, nroutes := range benchRouteCount {
			rt := new(Fat[int])
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

func BenchmarkFatTableLPM(b *testing.B) {
	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		for _, nroutes := range benchRouteCount {
			rt := new(Fat[int])
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

func BenchmarkFatMemIP4(b *testing.B) {
	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))
	for _, k := range []int{1_000, 10_000, 100_000, 1_000_000} {
		var startMem, endMem runtime.MemStats

		runtime.GC()
		runtime.ReadMemStats(&startMem)

		b.Run(strconv.Itoa(k), func(b *testing.B) {
			rt := new(Fat[any])
			for _, pfx := range randomRealWorldPrefixes4(prng, k) {
				rt.Insert(pfx, nil)
			}

			runtime.GC()
			runtime.ReadMemStats(&endMem)

			stats := rt.root4.nodeStatsRec()
			//nolint:gosec
			b.ReportMetric(float64(int(endMem.HeapAlloc-startMem.HeapAlloc)/k), "bytes/route")
			b.ReportMetric(float64(stats.nodes), "node")
			b.ReportMetric(float64(stats.pfxs), "pfxs")
			b.ReportMetric(float64(stats.leaves), "leaf")
			b.ReportMetric(float64(stats.fringes), "fringe")
			b.ReportMetric(0, "ns/op")
		})
	}
}

func BenchmarkFatMemIP6(b *testing.B) {
	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))
	for _, k := range []int{1_000, 10_000, 100_000, 1_000_000} {
		var startMem, endMem runtime.MemStats

		runtime.GC()
		runtime.ReadMemStats(&startMem)

		b.Run(strconv.Itoa(k), func(b *testing.B) {
			rt := new(Fat[any])
			for _, pfx := range randomRealWorldPrefixes6(prng, k) {
				rt.Insert(pfx, nil)
			}

			runtime.GC()
			runtime.ReadMemStats(&endMem)

			stats := rt.root6.nodeStatsRec()
			//nolint:gosec
			b.ReportMetric(float64(int(endMem.HeapAlloc-startMem.HeapAlloc)/k), "bytes/route")
			b.ReportMetric(float64(stats.nodes), "node")
			b.ReportMetric(float64(stats.pfxs), "pfxs")
			b.ReportMetric(float64(stats.leaves), "leaf")
			b.ReportMetric(float64(stats.fringes), "fringe")
			b.ReportMetric(0, "ns/op")
		})
	}
}

func BenchmarkFatMem(b *testing.B) {
	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))
	for _, k := range []int{1_000, 10_000, 100_000, 1_000_000} {
		var startMem, endMem runtime.MemStats

		runtime.GC()
		runtime.ReadMemStats(&startMem)

		b.Run(strconv.Itoa(k), func(b *testing.B) {
			rt := new(Fat[any])
			for _, pfx := range randomRealWorldPrefixes(prng, k) {
				rt.Insert(pfx, nil)
			}

			runtime.GC()
			runtime.ReadMemStats(&endMem)

			s4 := rt.root4.nodeStatsRec()
			s6 := rt.root6.nodeStatsRec()
			stats := stats{
				s4.pfxs + s6.pfxs,
				s4.childs + s6.childs,
				s4.nodes + s6.nodes,
				s4.leaves + s6.leaves,
				s4.fringes + s6.fringes,
			}

			//nolint:gosec
			b.ReportMetric(float64(int(endMem.HeapAlloc-startMem.HeapAlloc)/k), "bytes/route")
			b.ReportMetric(float64(stats.nodes), "node")
			b.ReportMetric(float64(stats.pfxs), "pfxs")
			b.ReportMetric(float64(stats.leaves), "leaf")
			b.ReportMetric(float64(stats.fringes), "fringe")
			b.ReportMetric(0, "ns/op")
		})
	}
}

func BenchmarkFatFullTableMemory4(b *testing.B) {
	var startMem, endMem runtime.MemStats
	nRoutes := len(routes4)

	b.Run(fmt.Sprintf("Table[]: %d", nRoutes), func(b *testing.B) {
		rt := new(Fat[any])
		runtime.GC()
		runtime.ReadMemStats(&startMem)

		for _, route := range routes4 {
			rt.Insert(route.CIDR, nil)
		}

		runtime.GC()
		runtime.ReadMemStats(&endMem)

		stats := rt.root4.nodeStatsRec()
		//nolint:gosec
		b.ReportMetric(float64(int(endMem.HeapAlloc-startMem.HeapAlloc)/nRoutes), "bytes/route")
		b.ReportMetric(float64(stats.pfxs), "pfxs")
		b.ReportMetric(float64(stats.nodes), "nodes")
		b.ReportMetric(float64(stats.leaves), "leaves")
		b.ReportMetric(float64(stats.fringes), "fringes")
		b.ReportMetric(0, "ns/op")
	})
}

func BenchmarkFatFullTableMemory6(b *testing.B) {
	var startMem, endMem runtime.MemStats

	rt := new(Fat[any])
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	nRoutes := len(routes6)

	b.Run(fmt.Sprintf("Table[]: %d", nRoutes), func(b *testing.B) {
		for _, route := range routes6 {
			rt.Insert(route.CIDR, nil)
		}

		runtime.GC()
		runtime.ReadMemStats(&endMem)

		stats := rt.root6.nodeStatsRec()
		//nolint:gosec
		b.ReportMetric(float64(int(endMem.HeapAlloc-startMem.HeapAlloc)/nRoutes), "bytes/route")
		b.ReportMetric(float64(stats.pfxs), "pfxs")
		b.ReportMetric(float64(stats.nodes), "nodes")
		b.ReportMetric(float64(stats.leaves), "leaves")
		b.ReportMetric(float64(stats.fringes), "fringes")
		b.ReportMetric(0, "ns/op")
	})
}

func BenchmarkFatFullTableMemory(b *testing.B) {
	var startMem, endMem runtime.MemStats

	rt := new(Fat[any])
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	nRoutes := len(routes)

	b.Run(fmt.Sprintf("Table[]: %d", nRoutes), func(b *testing.B) {
		for _, route := range routes {
			rt.Insert(route.CIDR, nil)
		}

		runtime.GC()
		runtime.ReadMemStats(&endMem)

		s4 := rt.root4.nodeStatsRec()
		s6 := rt.root6.nodeStatsRec()
		stats := stats{
			pfxs:    s4.pfxs + s6.pfxs,
			childs:  s4.childs + s6.childs,
			nodes:   s4.nodes + s6.nodes,
			leaves:  s4.leaves + s6.leaves,
			fringes: s4.fringes + s6.fringes,
		}

		//nolint:gosec
		b.ReportMetric(float64(int(endMem.HeapAlloc-startMem.HeapAlloc)/nRoutes), "bytes/route")
		b.ReportMetric(float64(stats.pfxs), "pfxs")
		b.ReportMetric(float64(stats.nodes), "nodes")
		b.ReportMetric(float64(stats.leaves), "leaves")
		b.ReportMetric(float64(stats.fringes), "fringes")
		b.ReportMetric(0, "ns/op")
	})
}

func BenchmarkFatFullMatch4(b *testing.B) {
	rt := new(Fat[any])

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

func BenchmarkFatFullMatch6(b *testing.B) {
	rt := new(Fat[any])

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

func BenchmarkFatFullMiss4(b *testing.B) {
	rt := new(Fat[any])

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

func BenchmarkFatFullMiss6(b *testing.B) {
	rt := new(Fat[any])

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

func checkFatNumNodes(t *testing.T, tbl *Fat[int], want int) {
	t.Helper()

	s4 := tbl.root4.nodeStatsRec()
	s6 := tbl.root6.nodeStatsRec()
	nodes := s4.nodes + s6.nodes

	if got := nodes; got != want {
		t.Errorf("wrong table dump, got %d nodes want %d", got, want)
		t.Error(tbl.dumpString())
	}
}

func checkFatRoutes(t *testing.T, tbl *Fat[int], tt []tableTest) {
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
