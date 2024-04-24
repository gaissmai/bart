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
	"net/netip"
	"runtime"
	"testing"
)

func TestPathComp(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		rt2 := new(Table2[any])

		t.Log(rt2.String())

		exp := 2
		if got := rt2.numNodes(); got != exp {
			t.Log(rt2.dumpString())
			t.Errorf("numNodes: expected %v, got %v", exp, got)
		}

		exp = 0
		if got := rt2.numPrefixes(); got != exp {
			t.Log(rt2.dumpString())
			t.Errorf("numPrefixes: expected %v, got %v", exp, got)
		}
	})

	t.Run("default", func(t *testing.T) {
		rt2 := new(Table2[any])
		rt2.Insert(mpp("0.0.0.0/0"), nil)
		rt2.Insert(mpp("::/0"), nil)

		t.Log(rt2.String())

		exp := 2
		if got := rt2.numNodes(); got != exp {
			t.Log(rt2.dumpString())
			t.Errorf("numNodes: expected %v, got %v", exp, got)
		}

		exp = 2
		if got := rt2.numPrefixes(); got != exp {
			t.Log(rt2.dumpString())
			t.Errorf("numPrefixes: expected %v, got %v", exp, got)
		}
	})

	t.Run("insert sorted", func(t *testing.T) {
		rt2 := new(Table2[any])
		rt2.Insert(mpp("192.0.0.0/4"), nil)
		t.Log(rt2.dumpString4())

		rt2.Insert(mpp("192.0.0.0/12"), nil)
		t.Log(rt2.dumpString4())

		rt2.Insert(mpp("192.0.0.0/20"), nil)
		t.Log(rt2.dumpString4())

		rt2.Insert(mpp("192.0.0.0/28"), nil)
		t.Log(rt2.dumpString4())

		rt2.Insert(mpp("192.0.0.0/32"), nil)
		t.Log(rt2.dumpString4())

		t.Log(rt2.String())

		exp := 5
		if got := rt2.numNodes(); got != exp {
			t.Log(rt2.dumpString4())
			t.Errorf("numNodes: expected %v, got %v", exp, got)
		}

		exp = 5
		if got := rt2.numPrefixes(); got != exp {
			t.Log(rt2.dumpString4())
			t.Errorf("numPrefixes: expected %v, got %v", exp, got)
		}
	})

	t.Run("insert inverse", func(t *testing.T) {
		rt2 := new(Table2[any])
		rt2.Insert(mpp("192.0.0.0/32"), nil)
		t.Log(rt2.dumpString4())

		rt2.Insert(mpp("192.0.0.0/28"), nil)
		t.Log(rt2.dumpString4())

		rt2.Insert(mpp("192.0.0.0/20"), nil)
		t.Log(rt2.dumpString4())

		rt2.Insert(mpp("192.0.0.0/12"), nil)
		t.Log(rt2.dumpString4())

		rt2.Insert(mpp("192.0.0.0/4"), nil)
		t.Log(rt2.dumpString4())

		t.Log(rt2.String())

		exp := 5
		if got := rt2.numNodes(); got != exp {
			t.Log(rt2.dumpString4())
			t.Errorf("numNodes: expected %v, got %v", exp, got)
		}

		exp = 5
		if got := rt2.numPrefixes(); got != exp {
			t.Log(rt2.dumpString4())
			t.Errorf("numPrefixes: expected %v, got %v", exp, got)
		}
	})
}

func BenchmarkSize2(b *testing.B) {
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		var startMem, endMem runtime.MemStats
		for _, nroutes := range benchRouteCount {
			rt := new(Table2[any])

			b.Run(fmt.Sprintf("%d/%s", nroutes, fam), func(b *testing.B) {
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					rt = new(Table2[any])
					runtime.GC()
					runtime.ReadMemStats(&startMem)

					for _, route := range rng(nroutes) {
						rt.Insert(route.pfx, struct{}{})
					}

					runtime.GC()
					runtime.ReadMemStats(&endMem)
					if npfx := rt.numPrefixes(); npfx != nroutes {
						b.Fatalf("expect %v prefixes, got %v", nroutes, npfx)
					}

					b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc), "Bytes")
					b.ReportMetric(float64(rt.numNodes()), "Nodes")
					b.ReportMetric(float64(rt.numPrefixes()), "Prefixes")
					b.ReportMetric(0, "ns/op") // silence
				}
			})
		}
	}
}

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
			t.Errorf("Lookup %q got (%v, %v), want (_, false)", tc.addr, v, ok)
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

		for _, nroutes := range []int{100_000, 1_000_000} {
			var rt1 Table[int]
			var rt2 Table2[int]
			for _, route := range rng(nroutes) {
				rt1.Insert(route.pfx, route.val)
				rt2.Insert(route.pfx, route.val)
			}

			probe := rng(1)[0]

			b.ResetTimer()
			b.Run(fmt.Sprintf("none: %s/In_%6d/%s", fam, nroutes, "IP"), func(b *testing.B) {
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
