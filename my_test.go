package bart

import (
	"fmt"
	"math/rand"
	"net/netip"
	"testing"
)

type NULLT struct{}

var NULL NULLT

func TestOverlapsPrefixPC(t *testing.T) {
	tbl := &Table[int]{}

	// default route
	tbl.InsertPC(mpp("10.0.0.0/9"), 1)
	tbl.InsertPC(mpp("2001:db8::/32"), 2)

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
		rt.InsertPC(pfx.pfx, NULL)
	}
}

func TestFullTablePC(t *testing.T) {
	var rt Table[NULLT]
	for _, route := range routes {
		rt.InsertPC(route.CIDR, NULL)
	}
}

func BenchmarkTableInsertPC(b *testing.B) {
	for _, n := range []int{1, 2, 5, 10, 100, 200, 500, 1_000, 10_000, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("routes: %7d", n), func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				var rt Table[netip.Prefix]
				for _, route := range routes[:n] {
					rt.InsertPC(route.CIDR, route.CIDR)
				}
			}
		})
	}
}

func TestDeletePC(t *testing.T) {
	t.Run("path compressed purge", func(t *testing.T) {
		rtbl := &Table[int]{}
		checkNumNodes(t, rtbl, 0) // 0

		rtbl.InsertPC(mpp("10.10.0.0/17"), 1)
		rtbl.InsertPC(mpp("10.20.0.0/17"), 2)
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
		tbl.InsertPC(p.pfx, p.val)
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
		{
			name: "three prefix, one pc",
			pfxs: []netip.Prefix{
				mpp("73.77.182.0/24"),
				mpp("197.160.0.0/21"),
				mpp("197.224.0.0/12"),
			},
			wantNodes: 2,
			wantSize:  3,
		},
	}

	for _, tc := range tcs {
		tbl := new(Table[string])
		for _, pfx := range tc.pfxs {
			tbl.InsertPC(pfx, pfx.String())
		}

		gotNodes := tbl.nodes()
		if gotNodes != tc.wantNodes {
			t.Errorf("InsertPC, %s, nodes: got: %d, want: %d", tc.name, gotNodes, tc.wantNodes)
		}

		gotSize := tbl.Size()
		if gotSize != tc.wantSize {
			t.Errorf("InsertPC, %s, size: got: %d, want: %d", tc.name, gotSize, tc.wantSize)
		}
	}
}
