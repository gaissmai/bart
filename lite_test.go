package bart

import (
	"math/rand/v2"
	"net/netip"
	"testing"
)

func TestLiteInsert(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		ins  []netip.Prefix
		del  []netip.Prefix
		ip   netip.Addr
		want bool
	}{
		{
			name: "invalid IP",
			ins:  []netip.Prefix{mpp("0.0.0.0/0"), mpp("::/0")},
			ip:   netip.Addr{},
			want: false,
		},
		{
			name: "zero",
			ip:   randomAddr(),
			want: false,
		},
		{
			name: "ins/del/zero",
			ins:  []netip.Prefix{mpp("0.0.0.0/0"), mpp("::/0")},
			del:  []netip.Prefix{mpp("0.0.0.0/0"), mpp("::/0")},
			ip:   randomAddr(),
			want: false,
		},
		{
			name: "default route",
			ins:  []netip.Prefix{mpp("0.0.0.0/0"), mpp("::/0")},
			ip:   randomAddr(),
			want: true,
		},
		{
			name: "indentity v4",
			ins:  []netip.Prefix{mpp("10.20.30.40/32")},
			ip:   mpa("10.20.30.40"),
			want: true,
		},
		{
			name: "indentity v6",
			ins:  []netip.Prefix{mpp("2001:db8::1/128")},
			ip:   mpa("2001:db8::1"),
			want: true,
		},
	}
	for _, tc := range testCases {
		lt := new(Lite)
		for _, p := range tc.ins {
			lt.Insert(p)
		}
		for _, p := range tc.del {
			lt.Delete(p)
		}
		got := lt.Contains(tc.ip)
		if got != tc.want {
			t.Errorf("%s: got: %v, want: %v", tc.name, got, tc.want)
		}
	}
}

func TestLiteInsertDelete(t *testing.T) {
	t.Parallel()

	lt := new(Lite)

	pfxs := randomRealWorldPrefixes(100_000)
	for _, pfx := range pfxs {
		lt.Insert(pfx)
	}
	// delete all prefixes
	for _, pfx := range pfxs {
		lt.Delete(pfx)
	}

	root4 := lt.rootNodeByVersion(true)
	if !root4.prefixes.IsEmpty() || root4.children.Len() != 0 {
		t.Errorf("Insert -> Delete not idempotent")
	}

	root6 := lt.rootNodeByVersion(false)
	if !root6.prefixes.IsEmpty() || root6.children.Len() != 0 {
		t.Errorf("Insert -> Delete not idempotent")
	}
}

func TestLiteContains(t *testing.T) {
	t.Parallel()

	lt := new(Lite)
	tb := new(Table[any])

	for _, route := range randomPrefixes(1_000_000) {
		lt.Insert(route.pfx)
		tb.Insert(route.pfx, nil)
	}

	for range 10_000 {
		ip := randomAddr()

		got1 := lt.Contains(ip)
		got2 := tb.Contains(ip)

		if got1 != got2 {
			t.Errorf("compare Contains(%q), Lite: %v, Table: %v", ip, got1, got2)
		}
	}
}

func TestLiteDeleteShuffled(t *testing.T) {
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

	rt1 := new(Lite)
	for _, pfx := range pfxs {
		rt1.Insert(pfx.pfx)
	}
	for _, pfx := range toDelete {
		rt1.Insert(pfx.pfx)
	}
	for _, pfx := range toDelete {
		rt1.Delete(pfx.pfx)
	}

	for range 10 {
		pfxs2 := append([]goldTableItem[int](nil), pfxs...)
		toDelete2 := append([]goldTableItem[int](nil), toDelete...)
		rand.Shuffle(len(toDelete2), func(i, j int) { toDelete2[i], toDelete2[j] = toDelete2[j], toDelete2[i] })

		rt2 := new(Lite)
		for _, pfx := range pfxs2 {
			rt2.Insert(pfx.pfx)
		}
		for _, pfx := range toDelete2 {
			rt2.Insert(pfx.pfx)
		}
		for _, pfx := range toDelete2 {
			rt2.Delete(pfx.pfx)
		}

		if rt1.dumpString() != rt2.dumpString() {
			t.Fatal("shuffled table has different dumpString representation")
		}
	}
}

func TestLiteDeleteIsReverseOfInsert(t *testing.T) {
	t.Parallel()
	// Insert N prefixes, then delete those same prefixes in reverse
	// order. Each deletion should exactly undo the internal structure
	// changes that each insert did.
	const N = 10_000

	tbl := new(Lite)
	prefixes := randomPrefixes(N)

	for _, p := range prefixes {
		tbl.Insert(p.pfx)
	}

	for i := len(prefixes) - 1; i >= 0; i-- {
		tbl.Delete(prefixes[i].pfx)
	}

	if !tbl.root4.children.IsEmpty() {
		t.Error("DeleteIsReverseOfInsert, the root4 node isn't empty, but shold")
	}
	if !tbl.root6.children.IsEmpty() {
		t.Error("DeleteIsReverseOfInsert, the root6 node isn't empty, but shold")
	}
}
