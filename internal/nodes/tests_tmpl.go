// REPLACE with generate hint

// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

//go:generate ../../scripts/generate-node-tests.sh
//go:build generate

package nodes

// ### GENERATE DELETE START ###

// stub code for generator types and methods
// useful for gopls during development, deleted during go generate

import (
	"io"
	"net/netip"
	"strings"
	"testing"
)

var mpp = netip.MustParsePrefix

type _NODE_TYPE[V any] struct{}

func (n *_NODE_TYPE[V]) StatsRec() (_ StatsT)                                      { return }
func (n *_NODE_TYPE[V]) DumpRec(io.Writer, StridePath, int, bool, bool)            { return }
func (n *_NODE_TYPE[V]) Insert(netip.Prefix, V, int) (_ bool)                      { return }
func (n *_NODE_TYPE[V]) Delete(netip.Prefix) (_ bool)                              { return }
func (n *_NODE_TYPE[V]) InsertPersist(CloneFunc[V], netip.Prefix, V, int) (_ bool) { return }
func (n *_NODE_TYPE[V]) DeletePersist(CloneFunc[V], netip.Prefix) (_ bool)         { return }

// ### GENERATE DELETE END ###

func TestInsertDelete_NODE_TYPE(t *testing.T) {
	t.Parallel()

	zero := 0

	testsInsertDelete := []struct {
		name        string
		pfxs        []string
		is4         bool
		wantSize    int
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
			name:        "one leave in root node",
			pfxs:        []string{"0.0.0.0/32"},
			is4:         true,
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
			name:        "many pfxs in root node",
			pfxs:        []string{"0.0.0.0/0", "0.0.0.0/1", "0.0.0.0/2", "0.0.0.0/3"},
			is4:         true,
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
			name: "many pfxs, leaves and fringes in deeper level",
			pfxs: []string{
				"0.0.0.0/9", "0.0.0.0/10", // pfxs in level 1
				"0.1.0.0/19", // leave in level 1
				"0.2.0.0/16", // fringe in level 1
			},
			is4:         true,
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
	}

	for _, tt := range testsInsertDelete {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			n := new(_NODE_TYPE[int])
			for _, s := range tt.pfxs {
				n.Insert(mpp(s), zero, 0)
				n.Insert(mpp(s), zero, 0) // idempotent
			}

			stats := n.StatsRec()
			if pfxs := stats.Pfxs; pfxs != tt.wantPfxs {
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
				n.DumpRec(buf, StridePath{}, 0, tt.is4, false)
				t.Logf("%s:\n%s", tt.name, buf.String())
			}

			// delete all prefixes

			for _, s := range tt.pfxs {
				n.Delete(mpp(s))
				n.Delete(mpp(s)) // idempotent
			}

			stats = n.StatsRec()
			if num := stats.Pfxs; num != 0 {
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
				n.DumpRec(buf, StridePath{}, 0, tt.is4, false)
				t.Logf("%s:\n%s", tt.name, buf.String())
			}
		})

		t.Run("Persist_"+tt.name, func(t *testing.T) {
			t.Parallel()

			n := new(_NODE_TYPE[int])

			for _, s := range tt.pfxs {
				n.InsertPersist(nil, mpp(s), zero, 0)
				n.InsertPersist(nil, mpp(s), zero, 0) // idempotent
			}

			stats := n.StatsRec()
			if pfxs := stats.Pfxs; pfxs != tt.wantPfxs {
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
				n.DumpRec(buf, StridePath{}, 0, tt.is4, false)
				t.Logf("%s:\n%s", tt.name, buf.String())
			}

			// delete all prefixes

			for _, s := range tt.pfxs {
				n.DeletePersist(nil, mpp(s))
				n.DeletePersist(nil, mpp(s)) // idempotent
			}

			stats = n.StatsRec()
			if num := stats.Pfxs; num != 0 {
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
				n.DumpRec(buf, StridePath{}, 0, tt.is4, false)
				t.Logf("%s:\n%s", tt.name, buf.String())
			}
		})
	}
}
