// Code generated from file "commontests_tmpl.go"; DO NOT EDIT.

// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package nodes

import (
	"iter"
	"math/rand/v2"
	"net/netip"
	"slices"
	"strings"
	"testing"

	"github.com/gaissmai/bart/internal/tests/golden"
	"github.com/gaissmai/bart/internal/tests/random"
)

// helpers
func (n *FastNode[V]) all4() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		if n == nil {
			return
		}
		_ = n.AllRec(StridePath{}, 0, true, yield)
	}
}

func (n *FastNode[V]) all6() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		if n == nil {
			return
		}
		_ = n.AllRec(StridePath{}, 0, false, yield)
	}
}

func (n *FastNode[V]) allSorted4() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		if n == nil {
			return
		}
		_ = n.AllRecSorted(StridePath{}, 0, true, yield)
	}
}

func (n *FastNode[V]) allSorted6() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		if n == nil {
			return
		}
		_ = n.AllRecSorted(StridePath{}, 0, false, yield)
	}
}

func TestInsertDeleteFastNode(t *testing.T) {
	t.Parallel()

	zero := 0

	testsInsertDelete := []struct {
		name        string
		pfxs        []string
		is4         bool
		wantPfxs    int
		wantLeaves  int
		wantFringes int
	}{
		{
			name:        "null",
			pfxs:        []string{},
			is4:         true,
			wantPfxs:    0,
			wantLeaves:  0,
			wantFringes: 0,
		},
		{
			name:        "one prefix in root node",
			pfxs:        []string{"0.0.0.0/0"},
			is4:         true,
			wantPfxs:    1,
			wantLeaves:  0,
			wantFringes: 0,
		},
		{
			name:        "one prefix in root node IPv6",
			pfxs:        []string{"::/0"},
			is4:         false,
			wantPfxs:    1,
			wantLeaves:  0,
			wantFringes: 0,
		},
		{
			name:        "one leaf in root node",
			pfxs:        []string{"0.0.0.0/32"},
			is4:         true,
			wantPfxs:    0,
			wantLeaves:  1,
			wantFringes: 0,
		},
		{
			name:        "one leaf in root node IPv6",
			pfxs:        []string{"::/32"},
			is4:         false,
			wantPfxs:    0,
			wantLeaves:  1,
			wantFringes: 0,
		},
		{
			name:        "one fringe in root node",
			pfxs:        []string{"0.0.0.0/8"},
			is4:         true,
			wantPfxs:    0,
			wantLeaves:  0,
			wantFringes: 1,
		},
		{
			name:        "one fringe in root node IPv6",
			pfxs:        []string{"0::/8"},
			is4:         false,
			wantPfxs:    0,
			wantLeaves:  0,
			wantFringes: 1,
		},
		{
			name:        "many pfxs in root node",
			pfxs:        []string{"0.0.0.0/0", "0.0.0.0/1", "0.0.0.0/2", "0.0.0.0/3"},
			is4:         true,
			wantPfxs:    4,
			wantLeaves:  0,
			wantFringes: 0,
		},
		{
			name:        "many pfxs in root node IPv6",
			pfxs:        []string{"::/0", "::/1", "::/2", "::/3"},
			is4:         false,
			wantPfxs:    4,
			wantLeaves:  0,
			wantFringes: 0,
		},
		{
			name: "many pfxs and leaves in root node",
			pfxs: []string{
				"0.0.0.0/0", "0.0.0.0/1", "0.0.0.0/2", "0.0.0.0/3", // pfxs
				"0.0.0.0/9", "1.0.0.0/9", "2.0.0.0/9", "3.0.0.0/9", // leaves
			},
			is4:         true,
			wantPfxs:    4,
			wantLeaves:  4,
			wantFringes: 0,
		},
		{
			name: "many pfxs and leaves in root node IPv6",
			pfxs: []string{
				"::/0", "::/1", "::/2", "::/3", // pfxs
				"::/9", "0100::/9", "0200::/9", "0300::/9", // leaves
			},
			is4:         false,
			wantPfxs:    4,
			wantLeaves:  4,
			wantFringes: 0,
		},
		{
			name: "many pfxs, leaves and fringes in root node",
			pfxs: []string{
				"0.0.0.0/0", "0.0.0.0/1", // pfxs
				"0.0.0.0/9", "1.0.0.0/19", "2.0.0.0/29", // leaves
				"4.0.0.0/8", "5.0.0.0/8", "6.0.0.0/8", "7.0.0.0/8", // fringes
			},
			is4:         true,
			wantPfxs:    2,
			wantLeaves:  3,
			wantFringes: 4,
		},
		{
			name: "many pfxs, leaves and fringes in root node IPv6",
			pfxs: []string{
				"::/0", "::/1", // pfxs
				"::/9", "0100::/19", "0200::/29", // leaves
				"0400::/8", "0500::/8", "0600::/8", "0700::/8", // fringes
			},
			is4:         false,
			wantPfxs:    2,
			wantLeaves:  3,
			wantFringes: 4,
		},
		{
			name: "many pfxs, leaves and fringes in deeper level",
			pfxs: []string{
				"0.0.0.0/9", "0.0.0.0/10", // pfxs in level 1
				"0.1.0.0/19", // leaf in level 1
				"0.2.0.0/16", // fringe in level 1
			},
			is4:         true,
			wantPfxs:    2,
			wantLeaves:  1,
			wantFringes: 1,
		},
		{
			name: "many pfxs, leaves and fringes in deeper level IPv6",
			pfxs: []string{
				"::/9", "::/10", // pfxs in level 1
				"0010::/19", // leaf in level 1
				"0020::/16", // fringe in level 1
			},
			is4:         false,
			wantPfxs:    2,
			wantLeaves:  1,
			wantFringes: 1,
		},
		{
			name: "leaves and fringes in deeper level",
			pfxs: []string{
				"0.0.0.0/12", // pfx in level 1
				"0.0.0.0/16", // fringe in level 1 -> default pfx in level 2
				"0.0.0.0/24", // fringe in level 2
			},
			is4:         true,
			wantPfxs:    2,
			wantLeaves:  0,
			wantFringes: 1,
		},
		{
			name: "leaves and fringes in deeper level IPv6",
			pfxs: []string{
				"::/12", // pfx in level 1
				"::/16", // fringe in level 1 -> default pfx in level 2
				"::/24", // fringe in level 2
			},
			is4:         false,
			wantPfxs:    2,
			wantLeaves:  0,
			wantFringes: 1,
		},
	}

	for _, tt := range testsInsertDelete {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			n := new(FastNode[int])
			for _, s := range tt.pfxs {
				n.Insert(mpp(s), zero, 0)
				n.Insert(mpp(s), zero, 0) // idempotent
			}

			stats := n.StatsRec()
			if pfxs := stats.Prefixes; pfxs != tt.wantPfxs {
				t.Errorf("after insert: got num pfxs %d, want %d", pfxs, tt.wantPfxs)
			}
			if leaves := stats.Leaves; leaves != tt.wantLeaves {
				t.Errorf("after insert: got num leaves %d, want %d", leaves, tt.wantLeaves)
			}
			if fringes := stats.Fringes; fringes != tt.wantFringes {
				t.Errorf("after insert: got num fringes %d, want %d", fringes, tt.wantFringes)
			}

			if t.Failed() {
				buf := new(strings.Builder)
				n.DumpRec(buf, StridePath{}, 0, tt.is4)
				t.Logf("%s:\n%s", tt.name, buf.String())
			}

			// delete all prefixes

			for _, s := range tt.pfxs {
				n.Delete(mpp(s))
				n.Delete(mpp(s)) // idempotent
			}

			stats = n.StatsRec()
			if num := stats.Prefixes; num != 0 {
				t.Errorf("after delete: got num pfxs %d, want 0", num)
			}
			if num := stats.Leaves; num != 0 {
				t.Errorf("after delete: got num leaves %d, want 0", num)
			}
			if num := stats.Fringes; num != 0 {
				t.Errorf("after delete: got num fringes %d, want 0", num)
			}

			if t.Failed() {
				buf := new(strings.Builder)
				n.DumpRec(buf, StridePath{}, 0, tt.is4)
				t.Logf("%s:\n%s", tt.name, buf.String())
			}
		})

		t.Run("Persist_"+tt.name, func(t *testing.T) {
			t.Parallel()

			n := new(FastNode[int])

			for _, s := range tt.pfxs {
				n.InsertPersist(nil, mpp(s), zero, 0)
				n.InsertPersist(nil, mpp(s), zero, 0) // idempotent
			}

			stats := n.StatsRec()
			if pfxs := stats.Prefixes; pfxs != tt.wantPfxs {
				t.Errorf("after insert: got num pfxs %d, want %d", pfxs, tt.wantPfxs)
			}
			if leaves := stats.Leaves; leaves != tt.wantLeaves {
				t.Errorf("after insert: got num leaves %d, want %d", leaves, tt.wantLeaves)
			}
			if fringes := stats.Fringes; fringes != tt.wantFringes {
				t.Errorf("after insert: got num fringes %d, want %d", fringes, tt.wantFringes)
			}

			if t.Failed() {
				buf := new(strings.Builder)
				n.DumpRec(buf, StridePath{}, 0, tt.is4)
				t.Logf("%s:\n%s", tt.name, buf.String())
			}

			// delete all prefixes

			for _, s := range tt.pfxs {
				n.DeletePersist(nil, mpp(s))
				n.DeletePersist(nil, mpp(s)) // idempotent
			}

			stats = n.StatsRec()
			if num := stats.Prefixes; num != 0 {
				t.Errorf("after delete: got num pfxs %d, want 0", num)
			}
			if num := stats.Leaves; num != 0 {
				t.Errorf("after delete: got num leaves %d, want 0", num)
			}
			if num := stats.Fringes; num != 0 {
				t.Errorf("after delete: got num fringes %d, want 0", num)
			}

			if t.Failed() {
				buf := new(strings.Builder)
				n.DumpRec(buf, StridePath{}, 0, tt.is4)
				t.Logf("%s:\n%s", tt.name, buf.String())
			}
		})
	}
}

func TestAllIteratorsFastNode(t *testing.T) {
	t.Parallel()
	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))

	for range 10 {
		pfxs := random.RealWorldPrefixes4(prng, n)

		node := new(FastNode[int])
		for _, p := range pfxs {
			node.Insert(p, 0, 0)
		}

		// AllRec: collect without order guarantee
		var got []netip.Prefix
		i := 0
		for p := range node.all4() {
			i++
			got = append(got, p)
			if i >= n/2 {
				break
			}
		}

		if len(got) != n/2 {
			t.Fatalf("AllRec len=%d, want %d", len(got), n/2)
		}

		got = nil
		i = 0
		for p := range node.allSorted4() {
			i++
			got = append(got, p)
			if i >= n/2 {
				break
			}
		}

		if len(got) != n/2 {
			t.Fatalf("AllRecSorted len=%d, want %d", len(got), n/2)
		}

		slices.SortFunc(pfxs, CmpPrefix)
		if !slices.Equal(pfxs[:n/2], got) {
			t.Fatal("AllRecSorted is not as expected")
		}
	}
}

func TestSupernets4FastNode(t *testing.T) {
	t.Parallel()
	n := workLoadN()
	prng := rand.New(rand.NewPCG(42, 42))

	for range 10 {
		pfxs := random.RealWorldPrefixes4(prng, n)

		node := new(FastNode[int])
		gold := new(golden.Table[int])

		for i, pfx := range pfxs {
			node.Insert(pfx, i, 0)
			gold.Insert(pfx, i)
		}

		// test with random probes
		for _, probe := range pfxs {
			goldSupernets := gold.Supernets(probe)
			nodeSupernets := []netip.Prefix{}

			node.Supernets(probe, func(p netip.Prefix, _ int) bool {
				nodeSupernets = append(nodeSupernets, p)
				return true
			})

			if !slices.Equal(goldSupernets, nodeSupernets) {
				t.Errorf("Supernets expected equal to golden implementation")
			}
		}
	}
}

func TestSupernets6FastNode(t *testing.T) {
	t.Parallel()
	n := workLoadN()
	prng := rand.New(rand.NewPCG(42, 42))

	for range 10 {
		pfxs := random.RealWorldPrefixes6(prng, n)

		node := new(FastNode[int])
		gold := new(golden.Table[int])

		for i, pfx := range pfxs {
			node.Insert(pfx, i, 0)
			gold.Insert(pfx, i)
		}

		// test with random probes
		for _, probe := range pfxs {
			goldSupernets := gold.Supernets(probe)
			nodeSupernets := []netip.Prefix{}

			node.Supernets(probe, func(p netip.Prefix, _ int) bool {
				nodeSupernets = append(nodeSupernets, p)
				return true
			})

			if !slices.Equal(goldSupernets, nodeSupernets) {
				t.Errorf("Supernets expected equal to golden implementation")
			}
		}
	}
}

func TestSubnets4FastNode(t *testing.T) {
	t.Parallel()
	n := workLoadN()
	prng := rand.New(rand.NewPCG(42, 42))

	for range 10 {
		pfxs := random.RealWorldPrefixes4(prng, n)

		node := new(FastNode[int])
		gold := new(golden.Table[int])

		for i, pfx := range pfxs {
			node.Insert(pfx, i, 0)
			gold.Insert(pfx, i)
		}

		// the default route must have all pfxs as subnet
		defaultRoute := mpp("0.0.0.0/0")
		allPfxsSorted := slices.Clone(pfxs)
		slices.SortFunc(allPfxsSorted, CmpPrefix)

		nodeSubnets := []netip.Prefix{}
		node.Subnets(defaultRoute, func(p netip.Prefix, _ int) bool {
			nodeSubnets = append(nodeSubnets, p)
			return true
		})

		if !slices.Equal(allPfxsSorted, nodeSubnets) {
			t.Errorf("Subnets(%s) not equal to all sorted prefixes", defaultRoute)
		}

		kMax := max(1, n/10)
		somePfxs := make([]netip.Prefix, 0, kMax) // allocate mem 1x

		for k := range kMax {
			somePfxs = somePfxs[:0] // reset slice

			i := 0
			node.Subnets(defaultRoute, func(p netip.Prefix, _ int) bool {
				if i >= k {
					// early-termination: stop after k
					return false
				}
				i++
				somePfxs = append(somePfxs, p)
				return true
			})

			if len(somePfxs) != k {
				t.Errorf("Subnets early-termination: got %d items, want %d", len(somePfxs), k)
			}

			if !slices.Equal(somePfxs, allPfxsSorted[:k]) {
				t.Errorf("Subnets expected equal")
			}
		}

		// test with random probes
		for _, probe := range pfxs {
			goldSubnets := gold.Subnets(probe)
			nodeSubnets := []netip.Prefix{}

			node.Subnets(probe, func(p netip.Prefix, _ int) bool {
				nodeSubnets = append(nodeSubnets, p)
				return true
			})

			if !slices.Equal(goldSubnets, nodeSubnets) {
				t.Errorf("Subnets expected equal to golden implementation")
			}
		}
	}
}

func TestSubnets6FastNode(t *testing.T) {
	t.Parallel()
	n := workLoadN()
	prng := rand.New(rand.NewPCG(42, 42))

	for range 10 {
		pfxs := random.RealWorldPrefixes6(prng, n)

		node := new(FastNode[int])
		gold := new(golden.Table[int])

		for i, pfx := range pfxs {
			node.Insert(pfx, i, 0)
			gold.Insert(pfx, i)
		}

		// the default route must have all pfxs as subnet
		defaultRoute := mpp("::/0")
		allPfxsSorted := slices.Clone(pfxs)
		slices.SortFunc(allPfxsSorted, CmpPrefix)

		nodeSubnets := []netip.Prefix{}
		node.Subnets(defaultRoute, func(p netip.Prefix, _ int) bool {
			nodeSubnets = append(nodeSubnets, p)
			return true
		})

		if !slices.Equal(allPfxsSorted, nodeSubnets) {
			t.Errorf("Subnets(%s) not equal to all sorted prefixes", defaultRoute)
		}

		kMax := max(1, n/10)
		somePfxs := make([]netip.Prefix, 0, kMax) // allocate mem 1x

		for k := range kMax {
			somePfxs = somePfxs[:0] // reset slice

			i := 0
			node.Subnets(defaultRoute, func(p netip.Prefix, _ int) bool {
				if i >= k {
					// early-termination: stop after k
					return false
				}
				i++
				somePfxs = append(somePfxs, p)
				return true
			})

			if len(somePfxs) != k {
				t.Errorf("Subnets early-termination: got %d items, want %d", len(somePfxs), k)
			}

			if !slices.Equal(somePfxs, allPfxsSorted[:k]) {
				t.Errorf("Subnets expected equal")
			}
		}

		// test with random probes
		for _, probe := range pfxs {
			goldSubnets := gold.Subnets(probe)
			nodeSubnets := []netip.Prefix{}

			node.Subnets(probe, func(p netip.Prefix, _ int) bool {
				nodeSubnets = append(nodeSubnets, p)
				return true
			})

			if !slices.Equal(goldSubnets, nodeSubnets) {
				t.Errorf("Subnets expected equal to golden implementation")
			}
		}
	}
}
