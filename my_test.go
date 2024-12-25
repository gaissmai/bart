package bart

import (
	"fmt"
	"math/rand"
	"net/netip"
	"strings"
	"testing"
)

type NULLT struct{}

var NULL NULLT

func TestNodeTreePC(t *testing.T) {
	w := new(strings.Builder)
	is4 := true
	depth := 0
	got := ""
	want := ""
	pfx := netip.Prefix{}
	n := new(node[string])

	pfx = mpp("0.0.0.0/0")
	n = n.pfxToNodeTree(pfx, "default route", 0)
	n.dumpRec(w, zeroPath, depth, is4)
	got = w.String()
	w.Reset()
	want = `
[LEAF] depth:  0 path: [] / 0
indexs(#1): [1]
prefxs(#1): 0/0
values(#1): default route
`
	if got != want {
		t.Errorf("pfxToNodeTree, %s, got:\n%s\n\nwant:\n%s", pfx, got, want)
	}

	pfx = mpp("10.11.12.13/32")
	n = n.pfxToNodeTree(pfx, "full /32", 0)
	n.dumpRec(w, zeroPath, depth, is4)
	got = w.String()
	w.Reset()
	want = `
[IMED] depth:  0 path: [] / 0
childs(#1): 10

.[IMED] depth:  1 path: [10] / 8
.childs(#1): 11

..[IMED] depth:  2 path: [10.11] / 16
..childs(#1): 12

...[LEAF] depth:  3 path: [10.11.12] / 24
...indexs(#1): [269]
...prefxs(#1): 13/8
...values(#1): full /32
`
	if got != want {
		t.Errorf("pfxToNodeTree, %s, got:\n%s\n\nwant:\n%s", pfx, got, want)
	}

	is4 = false
	pfx = mpp("::1/128")
	n = n.pfxToNodeTree(pfx, "full /128", 0)
	n.dumpRec(w, zeroPath, depth, is4)
	got = w.String()
	w.Reset()
	want = `
[IMED] depth:  0 path: [] / 0
childs(#1): 0x00

.[IMED] depth:  1 path: [00] / 8
.childs(#1): 0x00

..[IMED] depth:  2 path: [0000] / 16
..childs(#1): 0x00

...[IMED] depth:  3 path: [0000:00] / 24
...childs(#1): 0x00

....[IMED] depth:  4 path: [0000:0000] / 32
....childs(#1): 0x00

.....[IMED] depth:  5 path: [0000:0000:00] / 40
.....childs(#1): 0x00

......[IMED] depth:  6 path: [0000:0000:0000] / 48
......childs(#1): 0x00

.......[IMED] depth:  7 path: [0000:0000:0000:00] / 56
.......childs(#1): 0x00

........[IMED] depth:  8 path: [0000:0000:0000:0000] / 64
........childs(#1): 0x00

.........[IMED] depth:  9 path: [0000:0000:0000:0000:00] / 72
.........childs(#1): 0x00

..........[IMED] depth:  10 path: [0000:0000:0000:0000:0000] / 80
..........childs(#1): 0x00

...........[IMED] depth:  11 path: [0000:0000:0000:0000:0000:00] / 88
...........childs(#1): 0x00

............[IMED] depth:  12 path: [0000:0000:0000:0000:0000:0000] / 96
............childs(#1): 0x00

.............[IMED] depth:  13 path: [0000:0000:0000:0000:0000:0000:00] / 104
.............childs(#1): 0x00

..............[IMED] depth:  14 path: [0000:0000:0000:0000:0000:0000:0000] / 112
..............childs(#1): 0x00

...............[LEAF] depth:  15 path: [0000:0000:0000:0000:0000:0000:0000:00] / 120
...............indexs(#1): [257]
...............prefxs(#1): 0x01/8
...............values(#1): full /128
`

	if got != want {
		t.Errorf("pfxToNodeTree, %s, got:\n%s\n\nwant:\n%s", pfx, got, want)
	}
}

func TestOverlapsPrefixPC(t *testing.T) {
	tbl := &Table[int]{}

	// default route
	tbl.Insert(mpp("10.0.0.0/9"), 1)
	tbl.Insert(mpp("2001:db8::/32"), 2)

	pfx := mpp("0.0.0.0/0")
	got := tbl.OverlapsPrefix(pfx)

	want := true
	if got != want {
		t.Errorf("OverlapsPrefix, %s, got: %v, want: %v", pfx, got, want)
	}

	pfx = mpp("::/0")
	got = tbl.OverlapsPrefix(pfx)

	want = true
	if got != want {
		t.Errorf("OverlapsPrefix, %s, got: %v, want: %v", pfx, got, want)
	}
}

func TestRandomTablePC(t *testing.T) {
	var rt Table[NULLT]
	for _, pfx := range randomPrefixes(1_000_000) {
		rt.Insert(pfx.pfx, NULL)
	}
}

func TestFullTablePC(t *testing.T) {
	var rt Table[NULLT]
	for _, route := range routes {
		rt.Insert(route.CIDR, NULL)
	}
}

func BenchmarkTableInsertPC(b *testing.B) {
	for _, n := range []int{1, 2, 5, 10, 100, 200, 500, 1_000, 10_000, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("routes: %7d", n), func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				var rt Table[netip.Prefix]
				for _, route := range routes[:n] {
					rt.Insert(route.CIDR, route.CIDR)
				}
			}
		})
	}
}

func TestWorstCasePC(t *testing.T) {
	tbl := new(Table[string])
	for _, p := range worstCasePfxsIP4 {
		tbl.Insert(p, p.String())
	}

	want := true
	ok := tbl.Contains(worstCaseProbeIP4)
	if ok != want {
		t.Errorf("Contains, worst case match IP4, expected OK: %v, got: %v", want, ok)
	}
}

func TestDeletePC(t *testing.T) {
	t.Run("path compressed purge", func(t *testing.T) {
		rtbl := &Table[int]{}
		checkNumNodes(t, rtbl, 0) // 0

		rtbl.Insert(mpp("10.10.0.0/17"), 1)
		rtbl.Insert(mpp("10.20.0.0/17"), 2)
		checkNumNodes(t, rtbl, 2) // 1 root, 1 leaf

		checkRoutes(t, rtbl, []tableTest{
			{"10.10.127.0", 1},
			{"10.20.127.0", 2},
		})

		rtbl.Delete(mpp("10.20.0.0/17"))
		checkRoutes(t, rtbl, []tableTest{
			{"10.10.127.0", 1},
			{"10.20.127.0", -1},
		})

		rtbl.Delete(mpp("10.10.0.0/17"))
		checkRoutes(t, rtbl, []tableTest{
			{"10.10.127.0", -1},
			{"10.20.127.0", -1},
		})

		checkNumNodes(t, rtbl, 0) // 0
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
