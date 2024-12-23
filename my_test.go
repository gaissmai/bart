package bart

import (
	"math/rand"
	"net/netip"
	"testing"
)

func TestDeletePC(t *testing.T) {
	t.Parallel()

	t.Run("table_is_empty", func(t *testing.T) {
		t.Parallel()
		// must not panic
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
		checkNumNodes(t, tbl, 2) // 1 root4, 1 imed with 2 pc
		tbl.Delete(mpp("192.180.0.1/32"))
		checkRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", -1},
			{"192.40.0.1", -1},
		})
		checkNumNodes(t, tbl, 2) // 1 root4, 1 imed with 1 pc
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

		checkNumNodes(t, rtbl, 2) // 1 root4, 1 intermediates with 2 pc
		rtbl.Delete(mpp("192.180.0.1/32"))
		checkRoutes(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", -1},
			{"192.40.0.1", 3},
			{"192.255.0.1", -1},
		})
		checkNumNodes(t, rtbl, 2) // 1 root4, 1 intermediate, with 1 pc
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
		checkNumNodes(t, rtbl, 2) // 1 root4, 1 intermediate with 3 pc
		rtbl.Delete(mpp("192.180.0.1/32"))
		checkRoutes(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", -1},
			{"192.200.0.1", 3},
			{"192.255.0.1", -1},
		})
		checkNumNodes(t, rtbl, 2) // 1 root4, 1 intermediate with 2 pc
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
		checkNumNodes(t, rtbl, 1)        // 1 root4 with 1 pc
		rtbl.Delete(mpp("200.0.0.0/32")) // lookup miss in root
		checkRoutes(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.255.0.1", -1},
		})
		checkNumNodes(t, rtbl, 1) // 1 root4 with 1 pc
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
		checkNumNodes(t, rtbl, 1)          // 1 root4, path compressed
		rtbl.Delete(mpp("192.168.0.5/32")) // right leaf, no route
		checkRoutes(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.255.0.1", -1},
		})
		checkNumNodes(t, rtbl, 1) // 1 root4, path compressed
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
		checkNumNodes(t, rtbl, 3) // 1 root4, 2 imed, 2 path-compressed
		rtbl.Delete(mpp("192.168.0.0/22"))
		checkRoutes(t, rtbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.168.0.2", -1},
			{"192.255.0.1", -1},
		})
		checkNumNodes(t, rtbl, 3) // 1 root4, 2 imed, 1 pc
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

func TestGetAndDeletePC(t *testing.T) {
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

func TestInsertPC(t *testing.T) {
	tcs := []struct {
		name      string
		pfxs      []netip.Prefix
		wantNodes int
		wantSize  int
	}{
		{
			name:      "nil",
			pfxs:      nil,
			wantNodes: 0,
			wantSize:  0,
		},
		{
			name:      "single prefix",
			pfxs:      []netip.Prefix{mpp("10.10.10.10/32")},
			wantNodes: 1,
			wantSize:  1,
		},
		{
			name: "override single prefix",
			pfxs: []netip.Prefix{
				mpp("10.10.10.10/32"),
				mpp("10.10.10.10/32"),
			},
			wantNodes: 1,
			wantSize:  1,
		},
		{
			name: "two pc prefix",
			pfxs: []netip.Prefix{
				mpp("10.10.10.10/32"),
				mpp("20.20.20.20/32"),
			},
			wantNodes: 1,
			wantSize:  2,
		},
		{
			name: "two prefix",
			pfxs: []netip.Prefix{
				mpp("10.10.10.10/32"),
				mpp("10.10.10.11/32"),
			},
			wantNodes: 4,
			wantSize:  2,
		},
		{
			name: "two prefix, one pc",
			pfxs: []netip.Prefix{
				mpp("10.10.10.10/32"),
				mpp("10.10.10.11/32"),
				mpp("10.20.20.20/32"),
			},
			wantNodes: 4,
			wantSize:  3,
		},
		{
			name: "two prefix, two pc",
			pfxs: []netip.Prefix{
				mpp("10.10.10.10/32"),
				mpp("10.10.10.11/32"),
				mpp("10.20.20.20/32"),
				mpp("10.20.30.30/32"),
			},
			wantNodes: 5,
			wantSize:  4,
		},
	}

	for _, tc := range tcs {
		tbl := new(Table[string])
		for _, pfx := range tc.pfxs {
			tbl.Insert(pfx, pfx.String())
		}

		gotNodes := tbl.nodes()
		if gotNodes != tc.wantNodes {
			t.Errorf("InsertPC, %s, nodes: got: %d, want: %d", tc.name, gotNodes, tc.wantNodes)
		}

		gotSize := tbl.Size()
		if gotSize != tc.wantSize {
			t.Errorf("InsertPC, %s, size: got: %d, want: %d", tc.name, gotSize, tc.wantSize)
		}

		// t.Log(tbl.dumpString())
	}
}
