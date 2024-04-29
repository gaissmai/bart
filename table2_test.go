// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause
//
// some tests copied from github.com/tailscale/art
// and modified for this implementation by:
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

func TestRegression2(t *testing.T) {
	t.Parallel()
	// original comment by tailscale for ART,
	//
	// These tests are specific triggers for subtle correctness issues
	// that came up during initial implementation. Even if they seem
	// arbitrary, please do not clean them up. They are checking edge
	// cases that are very easy to get wrong, and quite difficult for
	// the other statistical tests to trigger promptly.
	//
	// ... but the BART implementation is different and has other edge cases.

	t.Run("prefixes_aligned_on_stride_boundary", func(t *testing.T) {
		fast := &Table2[int]{}
		slow := slowRT[int]{}

		fast.Insert(mpp("226.205.197.0/24"), 1)
		slow.insert(mpp("226.205.197.0/24"), 1)

		fast.Insert(mpp("226.205.0.0/16"), 2)
		slow.insert(mpp("226.205.0.0/16"), 2)

		probe := mpa("226.205.121.152")
		got, gotOK := fast.Lookup(probe)
		want, wantOK := slow.lookup(probe)
		if !getsEqual(got, gotOK, want, wantOK) {
			t.Fatalf("got (%v, %v), want (%v, %v)", got, gotOK, want, wantOK)
		}
	})

	t.Run("parent_prefix_inserted_in_different_orders", func(t *testing.T) {
		t1, t2 := &Table2[int]{}, &Table2[int]{}

		t1.Insert(mpp("136.20.0.0/16"), 1)
		t1.Insert(mpp("136.20.201.62/32"), 2)

		t2.Insert(mpp("136.20.201.62/32"), 2)
		t2.Insert(mpp("136.20.0.0/16"), 1)

		a := mpa("136.20.54.139")
		got1, ok1 := t1.Lookup(a)
		got2, ok2 := t2.Lookup(a)
		if !getsEqual(got1, ok1, got2, ok2) {
			t.Errorf("Lookup(%q) is insertion order dependent: t1=(%v, %v), t2=(%v, %v)", a, got1, ok1, got2, ok2)
		}
	})
}

func TestInsert2(t *testing.T) {
	tbl := &Table2[int]{}

	// Create a new leaf strideTable, with compressed path
	tbl.Insert(mpp("192.168.0.1/32"), 1)
	checkRoutes2(t, tbl, []tableTest{
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
	checkRoutes2(t, tbl, []tableTest{
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
	checkRoutes2(t, tbl, []tableTest{
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
	checkRoutes2(t, tbl, []tableTest{
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
	checkRoutes2(t, tbl, []tableTest{
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
	checkRoutes2(t, tbl, []tableTest{
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
	checkRoutes2(t, tbl, []tableTest{
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
	checkRoutes2(t, tbl, []tableTest{
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
	checkRoutes2(t, tbl, []tableTest{
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
	checkRoutes2(t, tbl, []tableTest{
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
	checkRoutes2(t, tbl, []tableTest{
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
	checkRoutes2(t, tbl, []tableTest{
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
	checkRoutes2(t, tbl, []tableTest{
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
	checkRoutes2(t, tbl, []tableTest{
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
	checkRoutes2(t, tbl, []tableTest{
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
	checkRoutes2(t, tbl, []tableTest{
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
	checkRoutes2(t, tbl, []tableTest{
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
	checkRoutes2(t, tbl, []tableTest{
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

func TestGet2(t *testing.T) {
	t.Parallel()

	rt := new(Table2[int])
	t.Run("empty table", func(t *testing.T) {
		_, ok := rt.Get(randomPrefix4())

		if ok {
			t.Errorf("empty table: ok=%v, expected: %v", ok, false)
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

	rt = new(Table2[int])

	for _, tt := range tests {
		rt.Insert(tt.pfx, tt.val)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestUpdateCompare2(t *testing.T) {
	// use update as insert and compare with slow implementation
	t.Parallel()
	pfxs := randomPrefixes(10_000)
	slow := slowRT[int]{pfxs}
	fast := Table2[int]{}

	t.Run("update as insert", func(t *testing.T) {
		for _, pfx := range pfxs {
			// contrived insert
			fast.Update(pfx.pfx, func(_ int, _ bool) int {
				return pfx.val
			})
		}

		seenVals4 := map[int]bool{}
		seenVals6 := map[int]bool{}

		for i := 0; i < 10_000; i++ {
			a := randomAddr()

			slowVal, slowOK := slow.lookup(a)
			fastVal, fastOK := fast.Lookup(a)

			if !getsEqual(slowVal, slowOK, fastVal, fastOK) {
				t.Fatalf("get(%q) = (%v, %v), want (%v, %v)", a, fastVal, fastOK, slowVal, slowOK)
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
	})

	t.Run("update", func(t *testing.T) {
		// update half of the prefixes

		cb := func(val int, ok bool) int {
			return val + 42
		}

		for _, pfx := range pfxs[len(pfxs)/2:] {
			slow.update(pfx.pfx, cb)
			fast.Update(pfx.pfx, cb)
		}

		for i := 0; i < 10_000; i++ {
			a := randomAddr()

			slowVal, slowOK := slow.lookup(a)
			fastVal, fastOK := fast.Lookup(a)

			if !getsEqual(slowVal, slowOK, fastVal, fastOK) {
				t.Fatalf("get(%q) = (%v, %v), want (%v, %v)", a, fastVal, fastOK, slowVal, slowOK)
			}
		}
	})
}

func TestUpdate2(t *testing.T) {
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

	rt := new(Table2[int])

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

func TestLookupCompare2(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()
	pfxs := randomPrefixes(10_000)

	slow := slowRT[int]{pfxs}
	fast := Table2[int]{}

	for _, pfx := range pfxs {
		fast.Insert(pfx.pfx, pfx.val)
	}

	seenVals4 := map[int]bool{}
	seenVals6 := map[int]bool{}

	for i := 0; i < 10_000; i++ {
		a := randomAddr()

		slowVal, slowOK := slow.lookup(a)
		fastVal, fastOK := fast.Lookup(a)

		if !getsEqual(slowVal, slowOK, fastVal, fastOK) {
			t.Fatalf("get(%q) = (%v, %v), want (%v, %v)", a, fastVal, fastOK, slowVal, slowOK)
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

func TestInsertShuffled2(t *testing.T) {
	// The order in which you insert prefixes into a route table
	// should not matter, as long as you're inserting the same set of
	// routes.
	t.Parallel()

	pfxs := randomPrefixes(1000)
	/* uncomment for failure debugging
	var pfxs2 []slowRTEntry[int]
	defer func() {
		if t.Failed() {
			t.Logf("pre-shuffle: %#v", pfxs)
			t.Logf("post-shuffle: %#v", pfxs2)
		}
	}()
	*/

	for i := 0; i < 10; i++ {
		pfxs2 := append([]slowRTEntry[int](nil), pfxs...)
		rand.Shuffle(len(pfxs2), func(i, j int) { pfxs2[i], pfxs2[j] = pfxs2[j], pfxs2[i] })

		addrs := make([]netip.Addr, 0, 10_000)
		for i := 0; i < 10_000; i++ {
			addrs = append(addrs, randomAddr())
		}

		rt1 := Table2[int]{}
		rt2 := Table2[int]{}

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
				t.Fatalf("get(%q) = (%v, %v), want (%v, %v)", a, val2, ok2, val1, ok1)
			}
		}
	}
}

// ############################################################################

// checkOverlaps2 verifies that the overlaps lookups in tt return the
// expected results on tbl.
func checkOverlaps2(t *testing.T, tbl *Table2[int], tests []tableOverlapsTest) {
	for _, tt := range tests {
		got := tbl.OverlapsPrefix(mpp(tt.prefix))
		if got != tt.want {
			t.Log(tbl.String())
			t.Errorf("OverlapsPrefix(%v) = %v, want %v", mpp(tt.prefix), got, tt.want)
		}
	}
}

// dumpAsPrefixTable, just a helper to compare with slowPrefixTable
func (t *Table2[V]) dumpAsPrefixTable() slowRT[V] {
	pfxs := []slowRTEntry[V]{}
	pfxs = dumpListRec(pfxs, t.DumpList4())
	pfxs = dumpListRec(pfxs, t.DumpList6())

	ret := slowRT[V]{pfxs}
	return ret
}

/*
func dumpListRec[V any](pfxs []slowRTEntry[V], dumpList []DumpListNode[V]) []slowRTEntry[V] {
	for _, node := range dumpList {
		pfxs = append(pfxs, slowRTEntry[V]{pfx: node.CIDR, val: node.Value})
		pfxs = append(pfxs, dumpListRec[V](nil, node.Subnets)...)
	}
	return pfxs
}
*/

// #########################################################

// checkRoutes verifies that the route lookups in tt return the
// expected results on tbl.
func checkRoutes2(t *testing.T, tbl *Table2[int], tt []tableTest) {
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

func checkSize2(t *testing.T, tbl *Table2[int], want int) {
	tbl.init()
	t.Helper()
	if got := tbl.numNodes(); got != want {
		t.Errorf("wrong table size, got %d strides want %d", got, want)
	}
}

func (t *Table2[V]) numNodes() int {
	t.init()
	return t.numNodesRec(t.rootV4) + t.numNodesRec(t.rootV6)
}

func (t *Table2[V]) numNodesRec(n *node2[V]) int {
	ret := 1 // this node
	for _, c := range n.children {
		ret += t.numNodesRec(c)
	}
	return ret
}

func (t *Table2[V]) numPrefixes() int {
	t.init()
	return t.numPrefixesRec(t.rootV4) + t.numPrefixesRec(t.rootV6)
}

func (t *Table2[V]) numPrefixesRec(n *node2[V]) int {
	ret := len(n.prefixes)
	for _, c := range n.children {
		ret += t.numPrefixesRec(c)
	}
	return ret
}

func BenchmarkTableLookup2(b *testing.B) {
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		// same routes
		for _, nroutes := range []int{100, 1_000, 10_000, 100_000, 1_000_000} {
			runtime.GC()

			var rt1 Table[int]
			var rt2 Table2[int]
			for _, route := range rng(nroutes) {
				rt1.Insert(route.pfx, route.val)
				rt2.Insert(route.pfx, route.val)
			}

			// same probe
			probe := rng(1)[0]

			b.ResetTimer()
			b.Run(fmt.Sprintf("orig: %s/In_%6d/%s", fam, nroutes, "IP"), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					writeSink, _ = rt1.Lookup(probe.pfx.Addr())
				}
				b.ReportMetric(float64(rt1.numNodes()), "Nodes")
			})

			b.ResetTimer()
			b.Run(fmt.Sprintf("comp: %s/In_%6d/%s", fam, nroutes, "IP"), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					writeSink, _ = rt2.Lookup(probe.pfx.Addr())
				}
				b.ReportMetric(float64(rt2.numNodes()), "Nodes")
			})

		}
	}
}

func BenchmarkSize2(b *testing.B) {
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		var startMem, endMem runtime.MemStats
		for _, nroutes := range []int{10, 100, 1_000, 10_000, 100_000, 1_000_000} {
			rt1 := new(Table[any])
			rt2 := new(Table2[any])

			b.Run(fmt.Sprintf("orig:%7d/%s", nroutes, fam), func(b *testing.B) {
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					rt1 = new(Table[any])
					runtime.GC()
					runtime.ReadMemStats(&startMem)

					for _, route := range rng(nroutes) {
						rt1.Insert(route.pfx, struct{}{})
					}

					runtime.GC()
					runtime.ReadMemStats(&endMem)
					if npfx := rt1.numPrefixes(); npfx != nroutes {
						b.Fatalf("expect %v prefixes, got %v", nroutes, npfx)
					}

					b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc), "Bytes")
					b.ReportMetric(float64(rt1.numNodes()), "Nodes")
					b.ReportMetric(0, "ns/op") // silence
				}
			})
			b.Run(fmt.Sprintf("comp:%7d/%s", nroutes, fam), func(b *testing.B) {
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					rt2 = new(Table2[any])
					runtime.GC()
					runtime.ReadMemStats(&startMem)

					for _, route := range rng(nroutes) {
						rt2.Insert(route.pfx, struct{}{})
					}

					runtime.GC()
					runtime.ReadMemStats(&endMem)
					if npfx := rt2.numPrefixes(); npfx != nroutes {
						b.Fatalf("expect %v prefixes, got %v", nroutes, npfx)
					}

					b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc), "Bytes")
					b.ReportMetric(float64(rt2.numNodes()), "Nodes")
					b.ReportMetric(0, "ns/op") // silence
				}
			})
		}
	}
}

func BenchmarkTableInsert2(b *testing.B) {
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		for _, nroutes := range benchRouteCount {
			var rt1 Table[struct{}]
			var rt2 Table2[struct{}]
			for _, route := range rng(nroutes) {
				rt1.Insert(route.pfx, struct{}{})
				rt2.Insert(route.pfx, struct{}{})
			}

			probe := rng(1)[0]

			b.ResetTimer()
			b.Run(fmt.Sprintf("orig/%s/Into_%d", fam, nroutes), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					rt1.Insert(probe.pfx, struct{}{})
				}
			})

			b.ResetTimer()
			b.Run(fmt.Sprintf("comp/%s/Into_%d", fam, nroutes), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					rt2.Insert(probe.pfx, struct{}{})
				}
			})
		}
	}
}
