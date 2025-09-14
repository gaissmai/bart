// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"testing"

	"github.com/gaissmai/bart/internal/art"
)

type childObj struct {
	id   int
	name string
}

func TestNodes_MultipleChildrenLifecycle(t *testing.T) {
	t.Parallel()

	c1 := &childObj{id: 1, name: "alpha"}
	c2 := &childObj{id: 2, name: "bravo"}
	c3 := &childObj{id: 3, name: "charlie"}

	nodes := []struct {
		name string
		node noder[int]
	}{
		{name: "node", node: &node[int]{}},
		{name: "fatNode", node: &fatNode[int]{}},
	}

	for _, tt := range nodes {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Insert three distinct children
			if exists := tt.node.insertChild(1, c1); exists {
				t.Fatalf("insertChild first(1) exists=true, want false")
			}
			if exists := tt.node.insertChild(2, c2); exists {
				t.Fatalf("insertChild first(2) exists=true, want false")
			}
			if exists := tt.node.insertChild(24, c3); exists {
				t.Fatalf("insertChild first(24) exists=true, want false")
			}
			if got := tt.node.childCount(); got != 3 {
				t.Fatalf("childCount=%d, want 3 after inserts", got)
			}

			// Duplicate key should report exists and not change count
			if exists := tt.node.insertChild(2, c2); !exists {
				t.Fatalf("insertChild duplicate(2) exists=false, want true")
			}
			if got := tt.node.childCount(); got != 3 {
				t.Fatalf("childCount=%d, want 3 after duplicate insert", got)
			}

			// Retrieval of present and absent keys
			any2, ok := tt.node.getChild(2)
			if !ok {
				t.Fatalf("getChild(2) ok=false, want true")
			}
			got2, ok := any2.(*childObj)
			if !ok || got2 != c2 || got2.name != "bravo" {
				t.Fatalf("getChild(2) type/value mismatch: %T %#v", any2, any2)
			}
			if _, ok := tt.node.getChild(3); ok {
				t.Fatalf("getChild(3) ok=true for missing key, want false")
			}

			// Delete one child and verify idempotency
			tt.node.deleteChild(2)
			if got := tt.node.childCount(); got != 2 {
				t.Fatalf("childCount=%d, want 2 after delete(2)", got)
			}
			if _, ok := tt.node.getChild(2); ok {
				t.Fatalf("getChild(2) ok=true after delete, want false")
			}
			tt.node.deleteChild(2) // idempotent
			if got := tt.node.childCount(); got != 2 {
				t.Fatalf("childCount=%d, want 2 after idempotent delete", got)
			}

			// Other children remain intact
			if _, ok := tt.node.getChild(1); !ok {
				t.Fatalf("getChild(1) ok=false, want true (unaffected)")
			}
			if _, ok := tt.node.getChild(24); !ok {
				t.Fatalf("getChild(24) ok=false, want true (unaffected)")
			}
		})
	}
}

func TestNodes_NearestAncestorWins_AcrossMultipleLevels(t *testing.T) {
	t.Parallel()
	nodes := []struct {
		name string
		node noder[int]
	}{
		{name: "node", node: &node[int]{}},
		{name: "fatNode", node: &fatNode[int]{}},
	}

	for _, tt := range nodes {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Insert a chain of increasingly specific prefixes.
			if exists := tt.node.insertPrefix(1, 10); exists {
				t.Fatalf("insertPrefix(1) exists=true, want false")
			}
			tt.node.insertPrefix(2, 20)
			tt.node.insertPrefix(4, 40)
			tt.node.insertPrefix(8, 80)

			// Helper to assert lookups
			assertLookup := func(idx uint8, want int) {
				if got, ok := tt.node.lookup(uint(idx)); !ok || got != want {
					t.Fatalf("lookup(%d)=(%d,%v), want (%d,true)", idx, got, ok, want)
				}
			}

			// Most specific ancestor should be chosen
			assertLookup(16, 80) // 16->8->4->2->1
			assertLookup(9, 40)  // 9->4->2->1
			assertLookup(6, 10)  // 6->3->1 (note: 2 is not on 6's chain)
			assertLookup(3, 10)  // 3->1

			// contains should reflect ancestry presence
			if !tt.node.contains(18) { // 18->9->4->2->1
				t.Fatalf("contains(18)=false, want true")
			}
			if !tt.node.contains(5) { // 5->2->1
				t.Fatalf("contains(5)=false, want true")
			}

			// Remove an intermediate ancestor and verify fallback to next ancestor
			if v, ok := tt.node.deletePrefix(4); !ok || v != 40 {
				t.Fatalf("deletePrefix(4)=(%d,%v), want (40,true)", v, ok)
			}
			assertLookup(9, 20) // now falls back to 2
			assertLookup(16, 80)

			// Remove most specific and ensure fallback continues to next available
			if v, ok := tt.node.deletePrefix(8); !ok || v != 80 {
				t.Fatalf("deletePrefix(8)=(%d,%v), want (80,true)", v, ok)
			}
			assertLookup(16, 20) // 16->8(X)->4(X)->2

			if got := tt.node.prefixCount(); got != 2 {
				t.Fatalf("prefixCount=%d, want 2 (only 1 and 2 remain)", got)
			}
		})
	}
}

func TestNodes_Lookup_NoAncestorPath(t *testing.T) {
	t.Parallel()
	nodes := []struct {
		name string
		node noder[int]
	}{
		{name: "node", node: &node[int]{}},
		{name: "fatNode", node: &fatNode[int]{}},
	}

	for _, tt := range nodes {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Only index 4 exists; index 5 is on a different lineage (5->2->1),
			// so with no 1/2 present this should fail.
			tt.node.insertPrefix(4, 40)

			if tt.node.contains(5) {
				t.Fatalf("contains(5)=true, want false (no ancestor along 5's chain)")
			}
			if _, ok := tt.node.lookup(5); ok {
				t.Fatalf("lookup(5) ok=true, want false (no ancestor along 5's chain)")
			}

			// Direct getPrefix should also be false when not set
			if v, ok := tt.node.getPrefix(5); ok || v != 0 {
				t.Fatalf("getPrefix(5)=(%d,%v), want (0,false)", v, ok)
			}
		})
	}
}

func TestNodes_GetPrefix_And_OverwriteSemantics(t *testing.T) {
	t.Parallel()
	nodes := []struct {
		name string
		node noder[int]
	}{
		{name: "node", node: &node[int]{}},
		{name: "fatNode", node: &fatNode[int]{}},
	}

	for _, tt := range nodes {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if exists := tt.node.insertPrefix(32, 111); exists {
				t.Fatalf("insertPrefix(32) first exists=true, want false")
			}
			if v, ok := tt.node.getPrefix(32); !ok || v != 111 {
				t.Fatalf("getPrefix(32)=(%d,%v), want (111,true)", v, ok)
			}

			// Overwrite should report exists and not increase count
			if exists := tt.node.insertPrefix(32, 222); !exists {
				t.Fatalf("insertPrefix(32) overwrite exists=false, want true")
			}
			if got := tt.node.prefixCount(); got != 1 {
				t.Fatalf("prefixCount=%d, want 1 after overwrite", got)
			}

			// Deleting returns the last stored value
			if v, ok := tt.node.deletePrefix(32); !ok || v != 222 {
				t.Fatalf("deletePrefix(32)=(%d,%v), want (222,true)", v, ok)
			}
			if got := tt.node.prefixCount(); got != 0 {
				t.Fatalf("prefixCount=%d, want 0 after delete", got)
			}
		})
	}
}

func TestFatNode_IsEmpty_AfterAllDeletes(t *testing.T) {
	t.Parallel()
	nodes := []struct {
		name string
		node noder[int]
	}{
		{name: "node", node: &node[int]{}},
		{name: "fatNode", node: &fatNode[int]{}},
	}

	for _, tt := range nodes {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Add both a child and a prefix, then remove and verify empty state restored.
			tt.node.insertChild(7, &childObj{id: 7, name: "seven"})
			tt.node.insertPrefix(64, 999)

			if tt.node.isEmpty() {
				t.Fatalf("isEmpty=true after inserts, want false")
			}

			tt.node.deleteChild(7)
			if v, ok := tt.node.deletePrefix(64); !ok || v != 999 {
				t.Fatalf("deletePrefix(64)=(%d,%v), want (999,true)", v, ok)
			}

			if !tt.node.isEmpty() {
				t.Fatalf("isEmpty=false after removing all, want true")
			}
		})
	}
}

func TestNodes_LPMEmpty_NoMatch(t *testing.T) {
	t.Parallel()
	nodes := []struct {
		name string
		node noder[int]
	}{
		{name: "node", node: &node[int]{}},
		{name: "fatNode", node: &fatNode[int]{}},
	}

	for _, tt := range nodes {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for i := range 256 {
				addr := uint8(i) //nolint:gosec // G115
				_, ok := tt.node.lookup(art.OctetToIdx(addr))
				if ok {
					t.Fatalf("expected no match in empty node for addr=%d", addr)
				}
			}
		})
	}
}

func TestNodes_LPMLongestPrefixWins(t *testing.T) {
	t.Parallel()

	// Choose an address and craft overlapping prefixes: /0, /3, /5, /7
	const octet = uint8(170) // 0b1010_1010

	type p struct {
		bits uint8
		val  int
	}
	ps := []p{
		{0, 0}, // default route
		{3, 3}, // 101_____ cover
		{5, 5}, // 10101___ cover
		{7, 7}, // 1010110_ cover
	}

	nodes := []struct {
		name string
		node noder[int]
	}{
		{name: "node", node: &node[int]{}},
		{name: "fatNode", node: &fatNode[int]{}},
	}

	for _, tt := range nodes {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, e := range ps {
				// Mask octet per prefix length and insert.
				tt.node.insertPrefix(art.PfxToIdx(octet, e.bits), e.val)
			}

			got, ok := tt.node.lookup(art.OctetToIdx(octet))
			if !ok || got != 7 {
				t.Fatalf("lookup(%d) got=(%v,%v), want (7,true)", octet, got, ok)
			}

			// Remove the /7 and ensure next-longest (/5) is selected.
			tt.node.deletePrefix(art.PfxToIdx(octet, 7))
			got, ok = tt.node.lookup(art.OctetToIdx(octet))
			if !ok || got != 5 {
				t.Fatalf("after delete /7, lookup(%d) got=(%v,%v), want (5,true)", octet, got, ok)
			}
		})
	}
}

func TestNodes_DeleteNonExistent_Safe(t *testing.T) {
	t.Parallel()

	nodes := []struct {
		name string
		node noder[int]
	}{
		{name: "node", node: &node[int]{}},
		{name: "fatNode", node: &fatNode[int]{}},
	}

	for _, tt := range nodes {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Insert one prefix (/2), then attempt to delete a different, non-existent one.
			const presentOctet = byte(0b1100_0000)
			tt.node.insertPrefix(art.PfxToIdx(presentOctet, 2), 42)

			// Deleting non-existent should not panic and should not affect existing mappings.
			tt.node.deletePrefix(art.PfxToIdx(byte(0b0000_0000), 1))

			v, ok := tt.node.lookup(art.OctetToIdx(uint8(0b1101_0101)))
			if !ok || v != 42 {
				t.Fatalf("expected mapping to remain after deleting non-existent prefix; got (%v,%v)", v, ok)
			}
		})
	}
}

func TestNodes_Contains_EqualsLookupTruthiness(t *testing.T) {
	t.Parallel()

	nodes := []struct {
		name string
		node noder[int]
	}{
		{name: "node", node: &node[int]{}},
		{name: "fatNode", node: &fatNode[int]{}},
	}

	for _, tt := range nodes {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Insert a sample of prefixes.
			for _, pfx := range shuffleStridePfxs(allStridePfxs())[:64] {
				tt.node.insertPrefix(art.PfxToIdx(pfx.octet, pfx.bits), pfx.val)
			}

			for i := range 256 {
				addr := uint8(i) //nolint:gosec // G115
				_, getOK := tt.node.lookup(art.OctetToIdx(addr))
				containsOk := tt.node.contains(art.OctetToIdx(addr))
				if getOK != containsOk {
					t.Fatalf("lookup and contains disagree for %d: test=%v get=%v", addr, containsOk, getOK)
				}
			}
		})
	}
}

func TestNodes_OverlapsIdxWithRootPrefix(t *testing.T) {
	t.Parallel()
	nodes := []struct {
		name string
		node noder[int]
	}{
		{name: "node", node: &node[int]{}},
		{name: "fatNode", node: &fatNode[int]{}},
	}

	for _, tt := range nodes {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, ok := tt.node.(*fatNode[int]); ok {
				t.Skipf("overlapsIdx not implemented yet for fatNode")
			}

			// rewrite it if overlapsIdx is in the noder interface
			n := tt.node.(*node[int])

			// Insert root prefix (/0). It should overlap every possible prefix.
			n.insertPrefix(art.PfxToIdx(0, 0), 1)

			for _, ttt := range allStridePfxs() {
				if !n.overlapsIdx(art.PfxToIdx(ttt.octet, ttt.bits)) {
					t.Fatalf("root prefix should overlap %d/%d", ttt.octet, ttt.bits)
				}
			}
		})
	}
}

func TestNodes_OverlapsRoutes_Symmetric(t *testing.T) {
	t.Parallel()
	all := shuffleStridePfxs(allStridePfxs())

	nodes := []struct {
		name string
		node noder[int]
	}{
		{name: "node", node: &node[int]{}},
		{name: "fatNode", node: &fatNode[int]{}},
	}

	for _, tt := range nodes {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, ok := tt.node.(*fatNode[int]); ok {
				t.Skipf("overlapsIdx not implemented yet for fatNode")
			}

			build := func(pfxs []goldStrideItem[int]) *node[int] {
				// rewrite it if overlapsIdx is in the noder interface
				n := tt.node.(*node[int])

				for _, p := range pfxs {
					n.insertPrefix(art.PfxToIdx(p.octet, p.bits), p.val)
				}
				return n
			}

			a := build(all[:5])
			b := build(all[5:10])

			if a.overlapsRoutes(b) != b.overlapsRoutes(a) {
				t.Fatalf("overlapsRoutes should be symmetric")
			}
		})
	}
}

func TestNodes_Prefixes_AsSliceConsistency(t *testing.T) {
	t.Parallel()

	nodes := []struct {
		name string
		node noder[int]
	}{
		{name: "node", node: &node[int]{}},
		{name: "fatNode", node: &fatNode[int]{}},
	}

	for _, tt := range nodes {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Insert a deterministic set of prefix indices (avoid 0 which is not a valid prefix idx).
			toInsert := []byte{1, 2, 127, 128, 254, 255}
			for _, idx := range toInsert {
				tt.node.insertPrefix(idx, 0)
			}

			// getIndices is a wrapper for AsSlice()
			s := tt.node.getIndices()

			// Expect each inserted index to be present exactly once in the slice.
			if len(s) != len(toInsert) {
				t.Fatalf("getIndices length=%d, want %d", len(s), len(toInsert))
			}
			counts := make(map[uint8]int, len(s))
			for _, v := range s {
				counts[v]++
			}
			for _, want := range toInsert {
				if counts[want] != 1 {
					t.Fatalf("missing or duplicated index %d in prefixes slice (count=%d)", want, counts[want])
				}
			}
		})
	}
}

func TestNode_Children_AsSliceConsistency(t *testing.T) {
	t.Parallel()
	nodes := []struct {
		name string
		node noder[int]
	}{
		{name: "node", node: &node[int]{}},
		{name: "fatNode", node: &fatNode[int]{}},
	}

	for _, tt := range nodes {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			toInsert := []byte{0, 3, 5, 9, 64, 200, 255}
			for _, idx := range toInsert {
				tt.node.insertChild(idx, nil)
			}

			// getChildAddrs is a wrapper for AsSlice
			s := tt.node.getChildAddrs()
			if len(s) != len(toInsert) {
				t.Fatalf("getChildAddrs length=%d, want %d", len(s), len(toInsert))
			}
			counts := make(map[uint8]int, len(s))
			for _, v := range s {
				counts[v]++
			}
			for _, want := range toInsert {
				if counts[want] != 1 {
					t.Fatalf("missing or duplicated child %d in slice (count=%d)", want, counts[want])
				}
			}
		})
	}
}

func TestNodes_InsertDuplicatePrefix_OverwritesValue(t *testing.T) {
	t.Parallel()
	nodes := []struct {
		name string
		node noder[int]
	}{
		{name: "node", node: &node[int]{}},
		{name: "fatNode", node: &fatNode[int]{}},
	}

	for _, tt := range nodes {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			const idx = byte(42)
			tt.node.insertPrefix(idx, 1)
			tt.node.insertPrefix(idx, 2) // duplicate insert with different value should overwrite

			s := tt.node.getIndices()
			if len(s) != 1 || s[0] != idx {
				t.Fatalf("duplicate insert should result in a single set bit for %d; slice=%v", idx, s)
			}

			// Exact get should reflect the latest value.
			v, ok := tt.node.getPrefix(idx)
			if !ok || v != 2 {
				t.Fatalf("expected duplicate insert to overwrite value: got (%v,%v), want (2,true)", v, ok)
			}
		})
	}
}

func TestNodes_DeleteChild_Idempotent(t *testing.T) {
	t.Parallel()
	nodes := []struct {
		name string
		node noder[int]
	}{
		{name: "node", node: &node[int]{}},
		{name: "fatNode", node: &fatNode[int]{}},
	}

	for _, tt := range nodes {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			const c = uint8(100)

			tt.node.insertChild(c, nil)
			// First delete removes it.
			tt.node.deleteChild(c)
			// Second delete is a no-op and must be safe.
			tt.node.deleteChild(c)

			// No children should remain.
			if s := tt.node.getChildAddrs(); len(s) != 0 {
				t.Fatalf("expected no children after idempotent deletes, got %v", s)
			}
		})
	}
}
