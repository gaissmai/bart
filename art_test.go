package bart

import (
	"fmt"
	"math/rand/v2"
	"net/netip"
	"runtime"
	"strconv"
	"testing"
)

func TestArtInvalid(t *testing.T) {
	t.Parallel()

	tbl := new(ArtTable[any])
	var zeroPfx netip.Prefix
	var zeroIP netip.Addr
	var testname string

	testname = "Insert"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid prefix input", testname)
			}
		}(testname)

		tbl.Insert(zeroPfx, nil)
	})

	testname = "Delete"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid prefix input", testname)
			}
		}(testname)

		tbl.Delete(zeroPfx)
	})

	testname = "Get"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid prefix input", testname)
			}
		}(testname)

		_, _ = tbl.Get(zeroPfx)
	})

	testname = "GetAndDelete"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid prefix input", testname)
			}
		}(testname)

		_, _ = tbl.GetAndDelete(zeroPfx)
	})

	testname = "Contains"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid IP input", testname)
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
				t.Fatalf("%s panics on invalid IP input", testname)
			}
		}(testname)

		_, got := tbl.Lookup(zeroIP)
		if got != false {
			t.Errorf("%s returns true on invalid IP input, expected false", testname)
		}
	})

	testname = "Contains"
	t.Run(testname, func(t *testing.T) {
		t.Parallel()
		defer func(testname string) {
			if r := recover(); r != nil {
				t.Fatalf("%s panics on invalid ip input", testname)
			}
		}(testname)

		tbl.Contains(zeroIP)
	})
}

func TestArtInsert(t *testing.T) {
	t.Parallel()

	tbl := new(ArtTable[int])

	// Create a new leaf strideTable, with compressed path
	tbl.Insert(mpp("192.168.0.1/32"), 1)
	checkArtNumNodes(t, tbl, 1)
	checkArtRoutes(t, tbl, []tableTest{
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
	checkArtNumNodes(t, tbl, 4)
	checkArtRoutes(t, tbl, []tableTest{
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
	checkArtNumNodes(t, tbl, 4)
	checkArtRoutes(t, tbl, []tableTest{
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
	checkArtNumNodes(t, tbl, 4)
	checkArtRoutes(t, tbl, []tableTest{
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
	checkArtNumNodes(t, tbl, 4)
	checkArtRoutes(t, tbl, []tableTest{
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
	checkArtNumNodes(t, tbl, 4)
	checkArtRoutes(t, tbl, []tableTest{
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
	checkArtNumNodes(t, tbl, 4)
	checkArtRoutes(t, tbl, []tableTest{
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
	checkArtNumNodes(t, tbl, 5)
	checkArtRoutes(t, tbl, []tableTest{
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
	checkArtNumNodes(t, tbl, 5)
	checkArtRoutes(t, tbl, []tableTest{
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
	checkArtNumNodes(t, tbl, 6)
	checkArtRoutes(t, tbl, []tableTest{
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
	checkArtNumNodes(t, tbl, 21)
	checkArtRoutes(t, tbl, []tableTest{
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
	checkArtNumNodes(t, tbl, 21)
	checkArtRoutes(t, tbl, []tableTest{
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
	checkArtNumNodes(t, tbl, 21)
	checkArtRoutes(t, tbl, []tableTest{
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
	checkArtNumNodes(t, tbl, 21)
	checkArtRoutes(t, tbl, []tableTest{
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
	checkArtNumNodes(t, tbl, 23)
	checkArtRoutes(t, tbl, []tableTest{
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
	checkArtNumNodes(t, tbl, 23)
	checkArtRoutes(t, tbl, []tableTest{
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
	checkArtNumNodes(t, tbl, 25)
	checkArtRoutes(t, tbl, []tableTest{
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
	checkArtNumNodes(t, tbl, 25)
	checkArtRoutes(t, tbl, []tableTest{
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

func TestArtDelete(t *testing.T) {
	t.Parallel()

	t.Run("table_is_empty", func(t *testing.T) {
		t.Parallel()
		prng := rand.New(rand.NewPCG(42, 42))
		// must not panic
		tbl := new(ArtTable[int])
		checkArtNumNodes(t, tbl, 0)
		tbl.Delete(randomPrefix(prng))
		checkArtNumNodes(t, tbl, 0)
	})

	t.Run("prefix_in_root", func(t *testing.T) {
		t.Parallel()
		// Add/remove prefix from root table.
		tbl := new(ArtTable[int])
		checkArtNumNodes(t, tbl, 0)

		tbl.Insert(mpp("10.0.0.0/8"), 1)
		checkArtNumNodes(t, tbl, 1)
		checkArtRoutes(t, tbl, []tableTest{
			{"10.0.0.1", 1},
			{"255.255.255.255", -1},
		})
		tbl.Delete(mpp("10.0.0.0/8"))
		checkArtNumNodes(t, tbl, 0)
		checkArtRoutes(t, tbl, []tableTest{
			{"10.0.0.1", -1},
			{"255.255.255.255", -1},
		})
	})

	t.Run("prefix_in_leaf", func(t *testing.T) {
		t.Parallel()
		// Create, then delete a single leaf table.
		tbl := new(ArtTable[int])
		checkArtNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		checkArtNumNodes(t, tbl, 1)
		checkArtRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"255.255.255.255", -1},
		})

		tbl.Delete(mpp("192.168.0.1/32"))
		checkArtNumNodes(t, tbl, 0)
		checkArtRoutes(t, tbl, []tableTest{
			{"192.168.0.1", -1},
			{"255.255.255.255", -1},
		})
	})

	t.Run("intermediate_no_routes", func(t *testing.T) {
		t.Parallel()
		// Create an intermediate with 2 leaves, then delete one leaf.
		tbl := new(ArtTable[int])
		checkArtNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		tbl.Insert(mpp("192.180.0.1/32"), 2)
		checkArtNumNodes(t, tbl, 2)
		checkArtRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", 2},
			{"192.40.0.1", -1},
		})

		tbl.Delete(mpp("192.180.0.1/32"))
		checkArtNumNodes(t, tbl, 1)
		checkArtRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", -1},
			{"192.40.0.1", -1},
		})
	})

	t.Run("intermediate_with_route", func(t *testing.T) {
		t.Parallel()
		// Same, but the intermediate carries a route as well.
		tbl := new(ArtTable[int])
		checkArtNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		tbl.Insert(mpp("192.180.0.1/32"), 2)
		tbl.Insert(mpp("192.0.0.0/10"), 3)

		checkArtNumNodes(t, tbl, 2)
		checkArtRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", 2},
			{"192.40.0.1", 3},
			{"192.255.0.1", -1},
		})

		tbl.Delete(mpp("192.180.0.1/32"))
		checkArtNumNodes(t, tbl, 2)
		checkArtRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", -1},
			{"192.40.0.1", 3},
			{"192.255.0.1", -1},
		})
	})

	t.Run("intermediate_many_leaves", func(t *testing.T) {
		t.Parallel()
		// Intermediate with 3 leaves, then delete one leaf.
		tbl := new(ArtTable[int])
		checkArtNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		tbl.Insert(mpp("192.180.0.1/32"), 2)
		tbl.Insert(mpp("192.200.0.1/32"), 3)

		checkArtNumNodes(t, tbl, 2)
		checkArtRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", 2},
			{"192.200.0.1", 3},
			{"192.255.0.1", -1},
		})

		tbl.Delete(mpp("192.180.0.1/32"))
		checkArtNumNodes(t, tbl, 2)
		checkArtRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.180.0.1", -1},
			{"192.200.0.1", 3},
			{"192.255.0.1", -1},
		})
	})

	t.Run("nosuchprefix_missing_child", func(t *testing.T) {
		t.Parallel()
		// Delete non-existent prefix
		tbl := new(ArtTable[int])
		checkArtNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		checkArtNumNodes(t, tbl, 1)
		checkArtRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.255.0.1", -1},
		})

		tbl.Delete(mpp("200.0.0.0/32"))
		checkArtNumNodes(t, tbl, 1)
		checkArtRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.255.0.1", -1},
		})
	})

	t.Run("intermediate_with_deleted_route", func(t *testing.T) {
		t.Parallel()
		// Intermediate node loses its last route and becomes
		// compactable.
		tbl := new(ArtTable[int])
		checkArtNumNodes(t, tbl, 0)

		tbl.Insert(mpp("192.168.0.1/32"), 1)
		tbl.Insert(mpp("192.168.0.0/22"), 2)
		checkArtNumNodes(t, tbl, 3)
		checkArtRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.168.0.2", 2},
			{"192.255.0.1", -1},
		})

		tbl.Delete(mpp("192.168.0.0/22"))
		checkArtNumNodes(t, tbl, 1)
		checkArtRoutes(t, tbl, []tableTest{
			{"192.168.0.1", 1},
			{"192.168.0.2", -1},
			{"192.255.0.1", -1},
		})
	})

	t.Run("default_route", func(t *testing.T) {
		t.Parallel()
		tbl := new(ArtTable[int])
		checkArtNumNodes(t, tbl, 0)

		tbl.Insert(mpp("0.0.0.0/0"), 1)
		tbl.Insert(mpp("::/0"), 1)
		tbl.Delete(mpp("0.0.0.0/0"))

		checkArtNumNodes(t, tbl, 1)
		checkArtRoutes(t, tbl, []tableTest{
			{"1.2.3.4", -1},
			{"::1", 1},
		})
	})

	t.Run("path compressed purge", func(t *testing.T) {
		t.Parallel()
		tbl := new(ArtTable[int])
		checkArtNumNodes(t, tbl, 0)

		tbl.Insert(mpp("10.10.0.0/17"), 1)
		tbl.Insert(mpp("10.20.0.0/17"), 2)
		checkArtNumNodes(t, tbl, 2)

		tbl.Delete(mpp("10.20.0.0/17"))
		checkArtNumNodes(t, tbl, 1)

		tbl.Delete(mpp("10.10.0.0/17"))
		checkArtNumNodes(t, tbl, 0)
	})
}

func TestArtContainsCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, 10_000)

	gold := new(goldTable[int]).insertMany(pfxs)
	fast := new(ArtTable[int])

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

func TestArtLookupCompare(t *testing.T) {
	// Create large route tables repeatedly, and compare Table's
	// behavior to a naive and slow but correct implementation.
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, 10_000)

	fast := new(ArtTable[int])
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

func TestArtInsertShuffled(t *testing.T) {
	// The order in which you insert prefixes into a route table
	// should not matter, as long as you're inserting the same set of
	// routes.
	t.Parallel()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomPrefixes(prng, 1000)

	for range 10 {
		pfxs2 := append([]goldTableItem[int](nil), pfxs...)
		rand.Shuffle(len(pfxs2), func(i, j int) { pfxs2[i], pfxs2[j] = pfxs2[j], pfxs2[i] })

		addrs := make([]netip.Addr, 0, 10_000)
		for range 10_000 {
			addrs = append(addrs, randomAddr(prng))
		}

		rt1 := new(ArtTable[int])
		rt2 := new(ArtTable[int])

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

func TestArtDeleteCompare(t *testing.T) {
	// Create large route tables repeatedly, delete half of their
	// prefixes, and compare Table's behavior to a naive and slow but
	// correct implementation.
	t.Parallel()
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

	fast := new(ArtTable[int])
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

func TestArtDeleteShuffled(t *testing.T) {
	// The order in which you delete prefixes from a route table
	// should not matter, as long as you're deleting the same set of
	// routes.
	t.Parallel()
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

		rt1 := new(ArtTable[int])

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

		rt2 := new(ArtTable[int])

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

func TestArtDeleteIsReverseOfInsert(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))
	// Insert N prefixes, then delete those same prefixes in reverse
	// order. Each deletion should exactly undo the internal structure
	// changes that each insert did.
	const N = 10_000

	tbl := new(ArtTable[int])
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

func TestArtDeleteButOne(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))
	// Insert N prefixes, then delete all but one
	const N = 100

	for range 1_000 {

		tbl := new(ArtTable[int])
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

func TestArtGetAndDelete(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))
	// Insert N prefixes, then delete those same prefixes in shuffled
	// order.
	const N = 10_000

	tbl := new(ArtTable[int])
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

func TestArtGet(t *testing.T) {
	t.Parallel()

	t.Run("empty table", func(t *testing.T) {
		t.Parallel()
		prng := rand.New(rand.NewPCG(42, 42))

		rt := new(ArtTable[int])
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

	rt := new(ArtTable[int])
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

func TestArtGetCompare(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))

	pfxs := randomPrefixes(prng, 10_000)
	fast := new(ArtTable[int])
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

// ############ benchmarks ################################

func BenchmarkArtTableInsertRandom(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, n := range []int{10_000, 100_000, 1_000_000, 2_000_000} {
		randomPfxs := randomRealWorldPrefixes(prng, n)

		rt := new(ArtTable[*MyInt])
		for i, pfx := range randomPfxs {
			myInt := MyInt(i)
			rt.Insert(pfx, &myInt)
		}

		probe := randomPrefix(prng)
		myInt := MyInt(42)

		b.ResetTimer()
		b.Run(fmt.Sprintf("mutable into %d", n), func(b *testing.B) {
			for range b.N {
				rt.Insert(probe, &myInt)
			}

			s4 := rt.root4.nodeStatsRec()
			s6 := rt.root6.nodeStatsRec()
			stats := stats{
				s4.pfxs + s6.pfxs,
				s4.childs + s6.childs,
				s4.nodes + s6.nodes,
				s4.leaves + s6.leaves,
				s4.fringes + s6.fringes,
			}

			b.ReportMetric(float64(rt.Size())/float64(stats.nodes), "Prefix/Node")
		})
	}
}

func BenchmarkArtTableDelete(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, n := range benchRouteCount {
		rt := new(ArtTable[*MyInt])
		for i, route := range randomPrefixes(prng, n) {
			myInt := MyInt(i)
			rt.Insert(route.pfx, &myInt)
		}

		probe := randomPrefix(prng)

		b.ResetTimer()
		b.Run(fmt.Sprintf("mutable from_%d", n), func(b *testing.B) {
			for range b.N {
				rt.Delete(probe)
			}
		})
	}
}

func BenchmarkArtTableGet(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		for _, nroutes := range benchRouteCount {
			rt := new(ArtTable[int])
			for _, route := range rng(prng, nroutes) {
				rt.Insert(route.pfx, route.val)
			}

			probe := rng(prng, 1)[0]

			b.ResetTimer()
			b.Run(fmt.Sprintf("%s/From_%d", fam, nroutes), func(b *testing.B) {
				for range b.N {
					_, boolSink = rt.Get(probe.pfx)
				}
			})
		}
	}
}

func BenchmarkArtTableLPM(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, fam := range []string{"ipv4", "ipv6"} {
		rng := randomPrefixes4
		if fam == "ipv6" {
			rng = randomPrefixes6
		}

		for _, nroutes := range benchRouteCount {
			rt := new(ArtTable[int])
			for _, route := range rng(prng, nroutes) {
				rt.Insert(route.pfx, route.val)
			}

			probe := rng(prng, 1)[0]

			b.ResetTimer()
			b.Run(fmt.Sprintf("%s/In_%6d/%s", fam, nroutes, "Contains"), func(b *testing.B) {
				for range b.N {
					boolSink = rt.Contains(probe.pfx.Addr())
				}
			})

			b.ResetTimer()
			b.Run(fmt.Sprintf("%s/In_%6d/%s", fam, nroutes, "Lookup"), func(b *testing.B) {
				for range b.N {
					_, boolSink = rt.Lookup(probe.pfx.Addr())
				}
			})
		}
	}
}

func BenchmarkArtMemIP4(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, k := range []int{1_000, 10_000, 100_000, 1_000_000} {
		var startMem, endMem runtime.MemStats

		runtime.GC()
		runtime.ReadMemStats(&startMem)

		b.Run(strconv.Itoa(k), func(b *testing.B) {
			rt := new(ArtTable[struct{}])
			for range b.N {
				rt = new(ArtTable[struct{}])
				for _, pfx := range randomRealWorldPrefixes4(prng, k) {
					rt.Insert(pfx, struct{}{})
				}
			}

			runtime.GC()
			runtime.ReadMemStats(&endMem)

			stats := rt.root4.nodeStatsRec()
			b.ReportMetric(float64(int(endMem.HeapAlloc-startMem.HeapAlloc)/k), "bytes/pfx")
			b.ReportMetric(float64(stats.nodes), "node")
			b.ReportMetric(float64(stats.pfxs), "pfxs")
			b.ReportMetric(float64(stats.leaves), "leaf")
			b.ReportMetric(float64(stats.fringes), "fringe")
			b.ReportMetric(0, "ns/op")
		})
	}
}

func BenchmarkArtMemIP6(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, k := range []int{1_000, 10_000, 100_000, 1_000_000} {
		var startMem, endMem runtime.MemStats

		runtime.GC()
		runtime.ReadMemStats(&startMem)

		b.Run(strconv.Itoa(k), func(b *testing.B) {
			rt := new(ArtTable[struct{}])
			for range b.N {
				rt = new(ArtTable[struct{}])
				for _, pfx := range randomRealWorldPrefixes6(prng, k) {
					rt.Insert(pfx, struct{}{})
				}
			}

			runtime.GC()
			runtime.ReadMemStats(&endMem)

			stats := rt.root6.nodeStatsRec()
			b.ReportMetric(float64(int(endMem.HeapAlloc-startMem.HeapAlloc)/k), "bytes/pfx")
			b.ReportMetric(float64(stats.nodes), "node")
			b.ReportMetric(float64(stats.pfxs), "pfxs")
			b.ReportMetric(float64(stats.leaves), "leaf")
			b.ReportMetric(float64(stats.fringes), "fringe")
			b.ReportMetric(0, "ns/op")
		})
	}
}

func BenchmarkArtMem(b *testing.B) {
	prng := rand.New(rand.NewPCG(42, 42))
	for _, k := range []int{1_000, 10_000, 100_000, 1_000_000} {
		var startMem, endMem runtime.MemStats

		runtime.GC()
		runtime.ReadMemStats(&startMem)

		b.Run(strconv.Itoa(k), func(b *testing.B) {
			rt := new(ArtTable[struct{}])
			for range b.N {
				rt = new(ArtTable[struct{}])
				for _, pfx := range randomRealWorldPrefixes(prng, k) {
					rt.Insert(pfx, struct{}{})
				}
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

			b.ReportMetric(float64(int(endMem.HeapAlloc-startMem.HeapAlloc)/k), "bytes/pfx")
			b.ReportMetric(float64(stats.nodes), "node")
			b.ReportMetric(float64(stats.pfxs), "pfxs")
			b.ReportMetric(float64(stats.leaves), "leaf")
			b.ReportMetric(float64(stats.fringes), "fringe")
			b.ReportMetric(0, "ns/op")
		})
	}
}

func BenchmarkArtFullTableMemory4(b *testing.B) {
	var startMem, endMem runtime.MemStats

	rt := new(ArtTable[struct{}])
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	b.Run(fmt.Sprintf("Table[]: %d", len(routes4)), func(b *testing.B) {
		for range b.N {
			for _, route := range routes4 {
				rt.Insert(route.CIDR, struct{}{})
			}
		}

		runtime.GC()
		runtime.ReadMemStats(&endMem)

		stats := rt.root4.nodeStatsRec()
		b.ReportMetric(float64(int(endMem.HeapAlloc-startMem.HeapAlloc)/stats.pfxs), "bytes/pfx")
		b.ReportMetric(float64(stats.pfxs), "pfxs")
		b.ReportMetric(float64(stats.nodes), "nodes")
		b.ReportMetric(float64(stats.leaves), "leaves")
		b.ReportMetric(float64(stats.fringes), "fringes")
		b.ReportMetric(0, "ns/op")
	})
}

func BenchmarkArtFullTableMemory6(b *testing.B) {
	var startMem, endMem runtime.MemStats

	rt := new(ArtTable[struct{}])
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	b.Run(fmt.Sprintf("Table[]: %d", len(routes6)), func(b *testing.B) {
		for range b.N {
			for _, route := range routes6 {
				rt.Insert(route.CIDR, struct{}{})
			}
		}

		runtime.GC()
		runtime.ReadMemStats(&endMem)

		stats := rt.root6.nodeStatsRec()
		b.ReportMetric(float64(int(endMem.HeapAlloc-startMem.HeapAlloc)/stats.pfxs), "bytes/pfx")
		b.ReportMetric(float64(stats.pfxs), "pfxs")
		b.ReportMetric(float64(stats.nodes), "nodes")
		b.ReportMetric(float64(stats.leaves), "leaves")
		b.ReportMetric(float64(stats.fringes), "fringes")
		b.ReportMetric(0, "ns/op")
	})
}

func BenchmarkArtFullTableMemory(b *testing.B) {
	var startMem, endMem runtime.MemStats

	rt := new(ArtTable[struct{}])
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	b.Run(fmt.Sprintf("Table[]: %d", len(routes)), func(b *testing.B) {
		for range b.N {
			for _, route := range routes {
				rt.Insert(route.CIDR, struct{}{})
			}
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

		b.ReportMetric(float64(int(endMem.HeapAlloc-startMem.HeapAlloc)/stats.pfxs), "bytes/pfx")
		b.ReportMetric(float64(stats.pfxs), "pfxs")
		b.ReportMetric(float64(stats.nodes), "nodes")
		b.ReportMetric(float64(stats.leaves), "leaves")
		b.ReportMetric(float64(stats.fringes), "fringes")
		b.ReportMetric(0, "ns/op")
	})
}

func checkArtNumNodes(t *testing.T, tbl *ArtTable[int], want int) {
	t.Helper()

	s4 := tbl.root4.nodeStatsRec()
	s6 := tbl.root6.nodeStatsRec()
	nodes := s4.nodes + s6.nodes

	if got := nodes; got != want {
		t.Errorf("wrong table dump, got %d nodes want %d", got, want)
		t.Error(tbl.dumpString())
	}
}

func checkArtRoutes(t *testing.T, tbl *ArtTable[int], tt []tableTest) {
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
