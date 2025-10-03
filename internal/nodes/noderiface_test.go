// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package nodes

import (
	"maps"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/gaissmai/bart/internal/art"
)

func isLiteNode[V any](n NodeReadWriter[V]) bool {
	_, ok := n.(*LiteNode[V])
	return ok
}

func TestZeroValueState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		nodeBuilder func() NodeReader[string]
	}{
		{"bartNode", func() NodeReader[string] { return &BartNode[string]{} }},
		{"fastNode", func() NodeReader[string] { return &FastNode[string]{} }},
		{"liteNode", func() NodeReader[string] { return &LiteNode[string]{} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			n := tt.nodeBuilder()

			if !n.IsEmpty() {
				t.Error("Zero value node should be empty")
			}

			if n.ChildCount() != 0 {
				t.Errorf("Zero value node childCount should be 0, got: %d", n.ChildCount())
			}

			if n.PrefixCount() != 0 {
				t.Errorf("Zero value node prefixCount should be 0, got: %d", n.PrefixCount())
			}

			// Test that getIndices returns empty slice
			indices := n.GetIndices(&[256]uint8{})
			if len(indices) != 0 {
				t.Errorf("Zero value node getIndices() should be empty, got length %d", len(indices))
			}

			// Test that getChildAddrs returns empty slice
			addrs := n.GetChildAddrs(&[256]uint8{})
			if len(addrs) != 0 {
				t.Errorf("Zero value node getChildAddrs() should be empty, got length %d", len(addrs))
			}
		})
	}
}

func TestEmptyNodeIterators(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		nodeBuilder func() NodeReader[string]
	}{
		{"bartNode", func() NodeReader[string] { return &BartNode[string]{} }},
		{"fastNode", func() NodeReader[string] { return &FastNode[string]{} }},
		{"liteNode", func() NodeReader[string] { return &LiteNode[string]{} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			n := tt.nodeBuilder()

			// Empty node should have no iterations
			var indices []uint8
			for idx := range n.AllIndices() {
				indices = append(indices, idx)
			}
			if len(indices) != 0 {
				t.Errorf("Empty node allIndices should have 0 iterations, got %d", len(indices))
			}

			var addrs []uint8
			for addr := range n.AllChildren() {
				addrs = append(addrs, addr)
			}
			if len(addrs) != 0 {
				t.Errorf("Empty node allChildren should have 0 iterations, got %d", len(addrs))
			}
		})
	}
}

func TestAllIndices(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		nodeBuilder func() NodeReadWriter[string]
	}{
		{"bartNode", func() NodeReadWriter[string] { return &BartNode[string]{} }},
		{"fastNode", func() NodeReadWriter[string] { return &FastNode[string]{} }},
		{"liteNode", func() NodeReadWriter[string] { return &LiteNode[string]{} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			n := tt.nodeBuilder()

			// Insert test data with specific indices and values
			expectedData := map[uint8]string{
				1:  "default",
				8:  "net8",
				16: "net16",
				24: "net24",
			}

			var expectedIndices []uint8
			var expectedValues []string

			for idx := range maps.Keys(expectedData) {
				expectedIndices = append(expectedIndices, idx)
			}

			slices.Sort(expectedIndices)

			for _, idx := range expectedIndices {
				expectedValues = append(expectedValues, expectedData[idx])
			}

			// Insert in non-sorted order to test sorting
			n.InsertPrefix(24, "net24")
			n.InsertPrefix(1, "default") // default route uses index 1
			n.InsertPrefix(16, "net16")
			n.InsertPrefix(8, "net8")

			var indices []uint8
			var values []string

			for idx, val := range n.AllIndices() {
				indices = append(indices, idx)
				values = append(values, val)
			}

			if !slices.Equal(indices, expectedIndices) {
				t.Errorf("Expected indices, got %v, want %v", indices, expectedIndices)
			}

			// liteNode has no real payload, return early
			if _, ok := n.(*LiteNode[string]); ok {
				return
			}

			if !slices.Equal(values, expectedValues) {
				t.Errorf("Expected values, got %v, want %v", values, expectedValues)
			}
		})
	}
}

func TestAllChildren(t *testing.T) {
	t.Parallel()

	childAddrs := []uint8{0, 17, 42, 64, 128, 192, 199, 255}

	tests := []struct {
		name        string
		nodeBuilder func() NodeReadWriter[string]
	}{
		{
			name:        "bartNode",
			nodeBuilder: func() NodeReadWriter[string] { return &BartNode[string]{} },
		},
		{
			name:        "fastNode",
			nodeBuilder: func() NodeReadWriter[string] { return &FastNode[string]{} },
		},
		{
			name:        "liteNode",
			nodeBuilder: func() NodeReadWriter[string] { return &LiteNode[string]{} },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			n := tt.nodeBuilder()

			// Create expected children data
			expectedChildren := make(map[uint8]any)
			for _, addr := range childAddrs {
				child := tt.nodeBuilder()
				child.InsertPrefix(1, "child_val")
				expectedChildren[addr] = child
			}

			var expectedAddrs []uint8
			for addr := range maps.Keys(expectedChildren) {
				expectedAddrs = append(expectedAddrs, addr)
			}
			slices.Sort(expectedAddrs)

			// Insert children in non-sorted order to test sorting
			for i := len(childAddrs) - 1; i >= 0; i-- {
				addr := childAddrs[i]
				n.InsertChild(addr, expectedChildren[addr])
			}

			var addrs []uint8
			var children []any

			for addr, child := range n.AllChildren() {
				addrs = append(addrs, addr)
				children = append(children, child)
			}

			if !slices.Equal(addrs, expectedAddrs) {
				t.Errorf("Expected addresses, got %v, want %v", addrs, expectedAddrs)
			}

			// Check exact children match expected
			for i, addr := range addrs {
				expectedChild := expectedChildren[addr]
				if children[i] != expectedChild {
					t.Errorf("Address %d: child pointer mismatch", addr)
				}
			}
		})
	}
}

func TestImplementsNoder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		nodeBuilder func() NodeReadWriter[string]
	}{
		{"bartNode", func() NodeReadWriter[string] { return &BartNode[string]{} }},
		{"fastNode", func() NodeReadWriter[string] { return &FastNode[string]{} }},
		{"liteNode", func() NodeReadWriter[string] { return &LiteNode[string]{} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			n := tt.nodeBuilder()

			// Test initial state
			if n.PrefixCount() != 0 {
				t.Errorf("Initial prefixCount should be 0, got %d", n.PrefixCount())
			}

			// Test insertPrefix with specific values
			testData := map[uint8]string{
				1:  "default",
				8:  "net8",
				16: "net16",
			}

			for idx, value := range testData {
				exists := n.InsertPrefix(idx, value)
				if exists {
					t.Errorf("insertPrefix(%d): should return false for new index", idx)
				}
			}

			// Verify final count
			if n.PrefixCount() != len(testData) {
				t.Errorf("Expected prefixCount %d after insertions, got %d", len(testData), n.PrefixCount())
			}

			// Test duplicate insertion
			exists := n.InsertPrefix(8, "duplicate")
			if !exists {
				t.Error("insertPrefix(8): should return true for existing index")
			}

			// Count should remain the same
			if n.PrefixCount() != len(testData) {
				t.Errorf("prefixCount should remain %d after duplicate insertion, got %d", len(testData), n.PrefixCount())
			}

			// Test deletePrefix with exact expected values
			expectedAfterDuplicate := maps.Clone(testData)
			expectedAfterDuplicate[8] = "duplicate" // was overwritten

			for idx := range testData {
				n.DeletePrefix(idx)
			}

			// Verify final count after deletions
			if n.PrefixCount() != 0 {
				t.Errorf("Expected prefixCount 0 after deletions, got %d", n.PrefixCount())
			}
		})
	}
}

func TestIteratorConsistency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		nodeBuilder func() NodeReader[string]
	}{
		{"bartNode", func() NodeReader[string] { return &BartNode[string]{} }},
		{"fastNode", func() NodeReader[string] { return &FastNode[string]{} }},
		{"liteNode", func() NodeReader[string] { return &LiteNode[string]{} }},
	}

	// Define expected test data
	expectedData := map[uint8]string{
		1:  "default", // default route uses index 1
		8:  "net8",
		24: "net24",
	}

	var expectedIndices []uint8
	var expectedValues []string

	for idx := range maps.Keys(expectedData) {
		expectedIndices = append(expectedIndices, idx)
	}
	slices.Sort(expectedIndices)

	for _, idx := range expectedIndices {
		expectedValues = append(expectedValues, expectedData[idx])
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			n := tt.nodeBuilder()

			// Cast to noder to insert data
			noder := n.(NodeReadWriter[string])
			for idx, val := range expectedData {
				noder.InsertPrefix(idx, val)
			}

			// Test that allIndices and getIndices are consistent
			directIndices := n.GetIndices(&[256]uint8{})

			var iterIndices []uint8
			var iterValues []string
			for idx, val := range n.AllIndices() {
				iterIndices = append(iterIndices, idx)
				iterValues = append(iterValues, val)
			}

			// Check indices match exactly
			if !slices.Equal(directIndices, expectedIndices) {
				t.Errorf("Direct indices, got %v, want %v", directIndices, expectedIndices)
			}

			if !slices.Equal(iterIndices, expectedIndices) {
				t.Errorf("Iterator indices, got %v, want %v", iterIndices, expectedIndices)
			}

			if !isLiteNode(noder) {
				if !slices.Equal(iterValues, expectedValues) {
					t.Errorf("Iterator values, got %v, want %v", iterValues, expectedValues)
				}
			}
		})
	}
}

// additional node tests

type childObj struct {
	id   int
	name string
}

func TestNodes_MultipleChildrenLifecycle(t *testing.T) {
	t.Parallel()

	c1 := &childObj{id: 1, name: "alpha"}
	c2 := &childObj{id: 2, name: "bravo"}
	c3 := &childObj{id: 3, name: "charlie"}

	tests := []struct {
		name        string
		nodeBuilder func() NodeReadWriter[string]
	}{
		{"bartNode", func() NodeReadWriter[string] { return &BartNode[string]{} }},
		{"fastNode", func() NodeReadWriter[string] { return &FastNode[string]{} }},
		{"liteNode", func() NodeReadWriter[string] { return &LiteNode[string]{} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			n := tt.nodeBuilder()

			// Insert three distinct children
			if exists := n.InsertChild(1, c1); exists {
				t.Fatalf("insertChild first(1) exists=true, want false")
			}
			if exists := n.InsertChild(2, c2); exists {
				t.Fatalf("insertChild first(2) exists=true, want false")
			}
			if exists := n.InsertChild(24, c3); exists {
				t.Fatalf("insertChild first(24) exists=true, want false")
			}
			if got := n.ChildCount(); got != 3 {
				t.Fatalf("childCount=%d, want 3 after inserts", got)
			}

			// Duplicate key should report exists and not change count
			if exists := n.InsertChild(2, c2); !exists {
				t.Fatalf("insertChild duplicate(2) exists=false, want true")
			}
			if got := n.ChildCount(); got != 3 {
				t.Fatalf("childCount=%d, want 3 after duplicate insert", got)
			}

			// Retrieval of present and absent keys
			any2, ok := n.GetChild(2)
			if !ok {
				t.Fatalf("getChild(2) ok=false, want true")
			}
			got2, ok := any2.(*childObj)
			if !ok || got2 != c2 || got2.name != "bravo" {
				t.Fatalf("getChild(2) type/value mismatch: %T %#v", any2, any2)
			}
			if _, ok := n.GetChild(3); ok {
				t.Fatalf("getChild(3) ok=true for missing key, want false")
			}

			// Delete one child and verify idempotency
			n.DeleteChild(2)
			if got := n.ChildCount(); got != 2 {
				t.Fatalf("childCount=%d, want 2 after delete(2)", got)
			}
			if _, ok := n.GetChild(2); ok {
				t.Fatalf("getChild(2) ok=true after delete, want false")
			}
			n.DeleteChild(2) // idempotent
			if got := n.ChildCount(); got != 2 {
				t.Fatalf("childCount=%d, want 2 after idempotent delete", got)
			}

			// Other children remain intact
			if _, ok := n.GetChild(1); !ok {
				t.Fatalf("getChild(1) ok=false, want true (unaffected)")
			}
			if _, ok := n.GetChild(24); !ok {
				t.Fatalf("getChild(24) ok=false, want true (unaffected)")
			}
		})
	}
}

func TestNodes_NearestAncestorWins_AcrossMultipleLevels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		nodeBuilder func() NodeReadWriter[int]
	}{
		{"bartNode", func() NodeReadWriter[int] { return &BartNode[int]{} }},
		{"fastNode", func() NodeReadWriter[int] { return &FastNode[int]{} }},
		{"liteNode", func() NodeReadWriter[int] { return &LiteNode[int]{} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			n := tt.nodeBuilder()

			// Insert a chain of increasingly specific prefixes.
			if exists := n.InsertPrefix(1, 10); exists {
				t.Fatalf("insertPrefix(1) exists=true, want false")
			}
			n.InsertPrefix(2, 20)
			n.InsertPrefix(4, 40)
			n.InsertPrefix(8, 80)

			// Helper to assert lookups
			assertLookup := func(idx uint8, want int) {
				if got, ok := n.Lookup(idx); !ok || got != want {
					t.Fatalf("lookup(%d)=(%d,%v), want (%d,true)", idx, got, ok, want)
				}
			}

			// Most specific ancestor should be chosen
			if !isLiteNode(n) {
				assertLookup(16, 80) // 16->8->4->2->1
				assertLookup(9, 40)  // 9->4->2->1
				assertLookup(6, 10)  // 6->3->1 (note: 2 is not on 6's chain)
				assertLookup(3, 10)  // 3->1
			}

			// contains should reflect ancestry presence
			if !n.Contains(18) { // 18->9->4->2->1
				t.Fatalf("contains(18)=false, want true")
			}
			if !n.Contains(5) { // 5->2->1
				t.Fatalf("contains(5)=false, want true")
			}
		})
	}
}

func TestNodes_Lookup_NoAncestorPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		nodeBuilder func() NodeReadWriter[int]
	}{
		{"bartNode", func() NodeReadWriter[int] { return &BartNode[int]{} }},
		{"fastNode", func() NodeReadWriter[int] { return &FastNode[int]{} }},
		{"liteNode", func() NodeReadWriter[int] { return &LiteNode[int]{} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			n := tt.nodeBuilder()

			// Only index 4 exists; index 5 is on a different lineage (5->2->1),
			// so with no 1/2 present this should fail.
			n.InsertPrefix(4, 40)

			if n.Contains(5) {
				t.Fatalf("contains(5)=true, want false (no ancestor along 5's chain)")
			}
			if !isLiteNode(n) {
				if _, ok := n.Lookup(5); ok {
					t.Fatalf("lookup(5) ok=true, want false (no ancestor along 5's chain)")
				}

				// Direct getPrefix should also be false when not set
				if v, ok := n.GetPrefix(5); ok || v != 0 {
					t.Fatalf("getPrefix(5)=(%d,%v), want (0,false)", v, ok)
				}
			}
		})
	}
}

func TestNodes_GetPrefix_And_OverwriteSemantics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		nodeBuilder func() NodeReadWriter[int]
	}{
		{"bartNode", func() NodeReadWriter[int] { return &BartNode[int]{} }},
		{"fastNode", func() NodeReadWriter[int] { return &FastNode[int]{} }},
		{"liteNode", func() NodeReadWriter[int] { return &LiteNode[int]{} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			n := tt.nodeBuilder()

			if exists := n.InsertPrefix(32, 111); exists {
				t.Fatalf("insertPrefix(32) first exists=true, want false")
			}
			if !isLiteNode(n) {
				if v, ok := n.GetPrefix(32); !ok || v != 111 {
					t.Fatalf("getPrefix(32)=(%d,%v), want (111,true)", v, ok)
				}
			}

			// Overwrite should report exists and not increase count
			if exists := n.InsertPrefix(32, 222); !exists {
				t.Fatalf("insertPrefix(32) overwrite exists=false, want true")
			}
			if got := n.PrefixCount(); got != 1 {
				t.Fatalf("prefixCount=%d, want 1 after overwrite", got)
			}

			n.DeletePrefix(32)
			if got := n.PrefixCount(); got != 0 {
				t.Fatalf("prefixCount=%d, want 0 after delete", got)
			}
		})
	}
}

func TestNode_IsEmpty_AfterAllDeletes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		nodeBuilder func() NodeReadWriter[int]
	}{
		{"bartNode", func() NodeReadWriter[int] { return &BartNode[int]{} }},
		{"fastNode", func() NodeReadWriter[int] { return &FastNode[int]{} }},
		{"liteNode", func() NodeReadWriter[int] { return &LiteNode[int]{} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			n := tt.nodeBuilder()

			// Add both a child and a prefix, then remove and verify empty state restored.
			n.InsertChild(7, &childObj{id: 7, name: "seven"})
			n.InsertPrefix(64, 999)

			if n.IsEmpty() {
				t.Fatalf("isEmpty=true after inserts, want false")
			}

			n.DeleteChild(7)
			n.DeletePrefix(64)
			if !n.IsEmpty() {
				t.Fatalf("isEmpty=false after removing all, want true")
			}
		})
	}
}

func TestNodes_LPMEmpty_NoMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		nodeBuilder func() NodeReadWriter[int]
	}{
		{"bartNode", func() NodeReadWriter[int] { return &BartNode[int]{} }},
		{"fastNode", func() NodeReadWriter[int] { return &FastNode[int]{} }},
		{"liteNode", func() NodeReadWriter[int] { return &LiteNode[int]{} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			n := tt.nodeBuilder()

			for i := range 256 {
				addr := uint8(i) //nolint:gosec // G115
				_, ok := n.Lookup(art.OctetToIdx(addr))
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

	// octet = 0b1010_1010
	ps := []p{
		{0, 0}, // default route
		{3, 3}, // 101_____ cover
		{5, 5}, // 10101___ cover
		{7, 7}, // 1010101_ cover
	}

	tests := []struct {
		name        string
		nodeBuilder func() NodeReadWriter[int]
	}{
		{"bartNode", func() NodeReadWriter[int] { return &BartNode[int]{} }},
		{"fastNode", func() NodeReadWriter[int] { return &FastNode[int]{} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			n := tt.nodeBuilder()

			for _, e := range ps {
				n.InsertPrefix(art.PfxToIdx(octet, e.bits), e.val)
			}

			got, ok := n.Lookup(art.OctetToIdx(octet))
			if !ok || got != 7 {
				t.Fatalf("lookup(%d) got=(%v,%v), want (7,true)", art.OctetToIdx(octet), got, ok)
			}

			// Remove the /7 and ensure next-longest (/5) is selected.
			n.DeletePrefix(art.PfxToIdx(octet, 7))
			got, ok = n.Lookup(art.OctetToIdx(octet))
			if !ok || got != 5 {
				t.Fatalf("after delete /7, lookup(%d) got=(%v,%v), want (5,true)", art.OctetToIdx(octet), got, ok)
			}
		})
	}
}

func TestNodes_DeleteNonExistent_Safe(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		nodeBuilder func() NodeReadWriter[int]
	}{
		{"bartNode", func() NodeReadWriter[int] { return &BartNode[int]{} }},
		{"fastNode", func() NodeReadWriter[int] { return &FastNode[int]{} }},
		{"liteNode", func() NodeReadWriter[int] { return &LiteNode[int]{} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			n := tt.nodeBuilder()

			// Insert one prefix (/2), then attempt to delete a different, non-existent one.
			const presentOctet = byte(0b1100_0000)
			n.InsertPrefix(art.PfxToIdx(presentOctet, 2), 42)

			// Deleting non-existent should not panic and should not affect existing mappings.
			n.DeletePrefix(art.PfxToIdx(byte(0b0000_0000), 1))

			_, ok := n.Lookup(art.OctetToIdx(uint8(0b1101_0101)))
			if !ok {
				t.Fatalf("expected mapping to remain after deleting non-existent prefix; got (_,%v)", ok)
			}
		})
	}
}

func TestNodes_Contains_EqualsLookupTruthiness(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		nodeBuilder func() NodeReadWriter[int]
	}{
		{"bartNode", func() NodeReadWriter[int] { return &BartNode[int]{} }},
		{"fastNode", func() NodeReadWriter[int] { return &FastNode[int]{} }},
		{"liteNode", func() NodeReadWriter[int] { return &LiteNode[int]{} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			n := tt.nodeBuilder()

			prng := rand.New(rand.NewPCG(42, 42))

			// Insert a sample of prefixes.
			for _, pfx := range shuffleNodePfxs(prng, allNodePfxs())[:64] {
				n.InsertPrefix(art.PfxToIdx(pfx.octet, pfx.bits), pfx.val)
			}

			for i := range 256 {
				addr := uint8(i) //nolint:gosec // G115
				_, getOK := n.Lookup(art.OctetToIdx(addr))
				containsOk := n.Contains(art.OctetToIdx(addr))
				if getOK != containsOk {
					t.Fatalf("lookup and contains disagree for %d: test=%v get=%v", addr, containsOk, getOK)
				}
			}
		})
	}
}

func TestNodes_Prefixes_AsSliceConsistency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		nodeBuilder func() NodeReadWriter[int]
	}{
		{"bartNode", func() NodeReadWriter[int] { return &BartNode[int]{} }},
		{"fastNode", func() NodeReadWriter[int] { return &FastNode[int]{} }},
		{"liteNode", func() NodeReadWriter[int] { return &LiteNode[int]{} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			n := tt.nodeBuilder()

			// Insert a deterministic set of prefix indices (avoid 0 which is not a valid prefix idx).
			toInsert := []byte{1, 2, 127, 128, 254, 255}
			for _, idx := range toInsert {
				n.InsertPrefix(idx, 0)
			}

			// getIndices is a wrapper for AsSlice()
			s := n.GetIndices(&[256]uint8{})

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

	tests := []struct {
		name        string
		nodeBuilder func() NodeReadWriter[int]
	}{
		{"bartNode", func() NodeReadWriter[int] { return &BartNode[int]{} }},
		{"fastNode", func() NodeReadWriter[int] { return &FastNode[int]{} }},
		{"liteNode", func() NodeReadWriter[int] { return &LiteNode[int]{} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			n := tt.nodeBuilder()

			toInsert := []byte{0, 3, 5, 9, 64, 200, 255}
			for _, idx := range toInsert {
				n.InsertChild(idx, nil)
			}

			// getChildAddrs is a wrapper for AsSlice
			s := n.GetChildAddrs(&[256]uint8{})
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

	tests := []struct {
		name        string
		nodeBuilder func() NodeReadWriter[int]
	}{
		{"bartNode", func() NodeReadWriter[int] { return &BartNode[int]{} }},
		{"fastNode", func() NodeReadWriter[int] { return &FastNode[int]{} }},
		{"liteNode", func() NodeReadWriter[int] { return &LiteNode[int]{} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			n := tt.nodeBuilder()

			const idx = byte(42)
			n.InsertPrefix(idx, 1)
			n.InsertPrefix(idx, 2) // duplicate insert with different value should overwrite

			s := n.GetIndices(&[256]uint8{})
			if len(s) != 1 || s[0] != idx {
				t.Fatalf("duplicate insert should result in a single set bit for %d; slice=%v", idx, s)
			}

			// Exact get should reflect the latest value.
			v, ok := n.GetPrefix(idx)
			if !ok {
				t.Fatalf("expected duplicate insert to keep prefix: got (_,%v), want (_,true)", ok)
			}
			if !isLiteNode(n) {
				if !ok || v != 2 {
					t.Fatalf("expected duplicate insert to overwrite value: got (%v,%v), want (2,true)", v, ok)
				}
			}
		})
	}
}

func TestNodes_DeleteChild_Idempotent(t *testing.T) {
	t.Parallel()

	const c = uint8(100)

	tests := []struct {
		name        string
		nodeBuilder func() NodeReadWriter[int]
	}{
		{"bartNode", func() NodeReadWriter[int] { return &BartNode[int]{} }},
		{"fastNode", func() NodeReadWriter[int] { return &FastNode[int]{} }},
		{"liteNode", func() NodeReadWriter[int] { return &LiteNode[int]{} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			n := tt.nodeBuilder()

			n.InsertChild(c, nil)
			// First delete removes it.
			n.DeleteChild(c)
			// Second delete is a no-op and must be safe.
			n.DeleteChild(c)

			// No children should remain.
			if s := n.GetChildAddrs(&[256]uint8{}); len(s) != 0 {
				t.Fatalf("expected no children after idempotent deletes, got %v", s)
			}
		})
	}
}
