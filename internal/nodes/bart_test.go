// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package nodes

import (
	"bytes"
	"maps"
	"net/netip"
	"slices"
	"strings"
	"testing"

	"github.com/gaissmai/bart/internal/art"
)

func TestBartNode_EmptyState(t *testing.T) {
	t.Parallel()
	n := &BartNode[int]{}

	if got := n.PrefixCount(); got != 0 {
		t.Errorf("PrefixCount()=%d, want 0", got)
	}
	if got := n.ChildCount(); got != 0 {
		t.Errorf("ChildCount()=%d, want 0", got)
	}
	if !n.IsEmpty() {
		t.Error("IsEmpty()=false, want true")
	}

	// Nil node should be empty
	var nilNode *BartNode[int]
	if !nilNode.IsEmpty() {
		t.Error("nil node should be empty")
	}
}

func TestBartNode_PrefixCRUD(t *testing.T) {
	t.Parallel()
	n := &BartNode[int]{}

	// Insert first time
	if exists := n.InsertPrefix(32, 100); exists {
		t.Error("InsertPrefix first time returned exists=true")
	}
	if n.PrefixCount() != 1 {
		t.Errorf("PrefixCount()=%d after insert, want 1", n.PrefixCount())
	}

	// Insert overwrite
	if exists := n.InsertPrefix(32, 111); !exists {
		t.Error("InsertPrefix overwrite returned exists=false")
	}
	if n.PrefixCount() != 1 {
		t.Errorf("PrefixCount()=%d after overwrite, want 1", n.PrefixCount())
	}
	if v, ok := n.GetPrefix(32); !ok || v != 111 {
		t.Errorf("GetPrefix(32)=(%d,%v), want (111,true)", v, ok)
	}

	// Delete
	if exists := n.DeletePrefix(32); !exists {
		t.Error("DeletePrefix returned exists=false")
	}
	if n.PrefixCount() != 0 {
		t.Errorf("PrefixCount()=%d after delete, want 0", n.PrefixCount())
	}
	if _, ok := n.GetPrefix(32); ok {
		t.Error("GetPrefix(32) after delete returned ok=true")
	}

	// Delete non-existent
	if exists := n.DeletePrefix(77); exists {
		t.Error("DeletePrefix non-existent returned exists=true")
	}
}

func TestBartNode_Contains_ART_Coverage(t *testing.T) {
	t.Parallel()
	n := &BartNode[int]{}

	// Insert at index 32 (allot() populates covered slots)
	n.InsertPrefix(32, 100)

	// Allotment set for 32 (uint8 range): {32,64,65,128,129,130,131}
	testCases := []struct {
		idx  uint8
		want bool
	}{
		{32, true},
		{64, true},
		{65, true},
		{128, true},
		{129, true},
		{130, true},
		{131, true},
		{1, false},
		{16, false},
		{33, false},
		{63, false},
		{127, false},
		{132, false},
		{255, false},
	}

	for _, tc := range testCases {
		if got := n.Contains(tc.idx); got != tc.want {
			t.Errorf("Contains(%d)=%v, want %v", tc.idx, got, tc.want)
		}
	}
}

func TestBartNode_LookupAndLookupIdx(t *testing.T) {
	t.Parallel()
	n := &BartNode[int]{}

	n.InsertPrefix(32, 0)
	n.InsertPrefix(64, 0)

	// Lookup returns zero value and existence for LiteNode
	if _, ok := n.Lookup(128); !ok {
		t.Error("Lookup(128) should succeed (covered by 64 and 32)")
	}

	// LookupIdx returns most specific covering index
	if top, _, ok := n.LookupIdx(128); !ok || top != 64 {
		t.Errorf("LookupIdx(128)=(top=%d, ok=%v), want (64, true)", top, ok)
	}

	// No coverage
	if _, ok := n.Lookup(127); ok {
		t.Error("Lookup(127) should fail (not covered)")
	}
}

func TestBartNode_ChildrenCRUD(t *testing.T) {
	t.Parallel()
	n := &BartNode[int]{}

	child := &BartNode[int]{}
	child.InsertPrefix(1, 10)

	// Insert
	if exists := n.InsertChild(10, child); exists {
		t.Error("InsertChild first time returned exists=true")
	}
	if n.ChildCount() != 1 {
		t.Errorf("ChildCount()=%d, want 1", n.ChildCount())
	}

	// Get
	if got, ok := n.GetChild(10); !ok {
		t.Error("GetChild(10) returned ok=false")
	} else if got != child {
		t.Error("GetChild returned wrong child")
	}

	// MustGetChild
	if got := n.MustGetChild(10); got != child {
		t.Error("MustGetChild returned wrong child")
	}

	// Delete
	if exists := n.DeleteChild(10); !exists {
		t.Error("DeleteChild returned exists=false")
	}
	if n.ChildCount() != 0 {
		t.Errorf("ChildCount()=%d after delete, want 0", n.ChildCount())
	}

	// Idempotent delete
	if exists := n.DeleteChild(10); exists {
		t.Error("DeleteChild on non-existent returned exists=true")
	}
}

func TestBartNode_MustGetChild_Panics(t *testing.T) {
	t.Parallel()
	n := &BartNode[int]{}

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustGetChild should panic on missing child")
		}
	}()
	_ = n.MustGetChild(42)
}

func TestBartNode_Iterators(t *testing.T) {
	t.Parallel()
	n := &BartNode[int]{}

	indices := []uint8{1, 32, 64, 128}
	for _, idx := range indices {
		n.InsertPrefix(idx, int(idx)*10)
	}

	// AllIndices: verify yielded values
	m := maps.Collect(n.AllIndices())
	for _, idx := range indices {
		want := int(idx) * 10
		if m[idx] != want {
			t.Errorf("AllIndices[%d]=%d, want %d", idx, m[idx], want)
		}
	}

	// Children
	addrs := []uint8{10, 20, 30}
	for _, addr := range addrs {
		n.InsertChild(addr, &BartNode[int]{})
	}

	count := 0
	for _, child := range n.AllChildren() {
		count++
		if child == nil {
			t.Error("AllChildren yielded nil child")
		}
	}
	if count != len(addrs) {
		t.Errorf("AllChildren count=%d, want %d", count, len(addrs))
	}
}

func TestBartNode_CloneFlat_ShallowAndWithCloneFn(t *testing.T) {
	t.Parallel()
	n := &BartNode[int]{}

	n.InsertPrefix(32, 100)
	n.InsertPrefix(64, 200)

	child := &BartNode[int]{}
	n.InsertChild(10, child)

	leaf := NewLeafNode[int](netip.Prefix{}, 7)
	n.InsertChild(20, leaf)

	fringe := NewFringeNode[int](70)
	n.InsertChild(30, fringe)

	// CloneFlat(nil): containers duplicated, children shallow, values not cloned
	shallow := n.CloneFlat(nil)

	if shallow.PrefixCount() != n.PrefixCount() || shallow.ChildCount() != n.ChildCount() {
		t.Fatalf("CloneFlat(nil) counts mismatch: got (p=%d,c=%d), want (p=%d,c=%d)",
			shallow.PrefixCount(), shallow.ChildCount(), n.PrefixCount(), n.ChildCount())
	}

	// Modify clone's prefixes — should not affect original
	shallow.InsertPrefix(128, 300)
	if _, ok := n.GetPrefix(128); ok {
		t.Error("modifying clone affected original prefixes")
	}

	// Children shallow-copied, leaf and fringe deep
	if c, _ := shallow.GetChild(10); c != child {
		t.Error("child should be same instance for CloneFlat(nil)")
	}
	if c, _ := shallow.GetChild(20); c == leaf {
		t.Error("leaf should not be same instance for CloneFlat(nil)")
	}
	if c, _ := shallow.GetChild(30); c == fringe {
		t.Error("fringe should not be same instance for CloneFlat(nil)")
	}

	// CloneFlat with cloneFn: values cloned (e.g., doubled)
	deepVals := n.CloneFlat(func(v int) int { return v * 2 })
	if v, ok := deepVals.GetPrefix(32); !ok || v != 200 {
		t.Errorf("CloneFlat(cloneFn) GetPrefix(32)=(%d,%v), want (200,true)", v, ok)
	}
	if v, ok := deepVals.GetPrefix(64); !ok || v != 400 {
		t.Errorf("CloneFlat(cloneFn) GetPrefix(64)=(%d,%v), want (400,true)", v, ok)
	}
	// After cloning with cloneFn, allot-derived lookups should reflect cloned values
	if v, ok := deepVals.Lookup(128); !ok || v != 400 {
		t.Errorf("CloneFlat(cloneFn) Lookup(128)=(%d,%v), want (400,true)", v, ok)
	}

	// Children shallow-copied, leaf and fringe deep
	if c, _ := deepVals.GetChild(10); c != child {
		t.Error("CloneFlat(cloneFn) child should be same instance")
	}
	if c, _ := deepVals.GetChild(20); c == leaf {
		t.Error("CloneFlat(cloneFn) leaf should NOT be same instance")
	} else if val := c.(*LeafNode[int]).Value; val != 14 {
		t.Errorf("CloneFlat(cloneFn) Leaf.Value=%v, want 14", val)
	}
	if c, _ := deepVals.GetChild(30); c == fringe {
		t.Error("CloneFlat(cloneFn) fringe should NOT be same instance")
	} else if val := c.(*FringeNode[int]).Value; val != 140 {
		t.Errorf("CloneFlat(cloneFn) Leaf.Value=%v, want 140", val)
	}
}

func TestBartNode_CloneRec(t *testing.T) {
	t.Parallel()
	n := &BartNode[int]{}

	n.InsertPrefix(32, 100)
	child := &BartNode[int]{}
	child.InsertPrefix(64, 200)
	n.InsertChild(10, child)

	grand := &BartNode[int]{}
	grand.InsertPrefix(128, 300)
	child.InsertChild(20, grand)

	clone := n.CloneRec(nil)

	// Deep copy: child/grand are new instances
	cloneChild, _ := clone.GetChild(10)
	if cloneChild == child {
		t.Error("CloneRec should deep copy child")
	}
	cloneGrand, _ := cloneChild.(*BartNode[int]).GetChild(20)
	if cloneGrand == grand {
		t.Error("CloneRec should deep copy grandchild")
	}

	// Mutating clone should not affect original
	cloneChild.(*BartNode[int]).InsertPrefix(255, 999)
	if child.PrefixCount() != 1 {
		t.Error("modifying clone affected original child")
	}
}

func TestBartNode_Basics_Insert_Get_Delete(t *testing.T) {
	t.Parallel()
	n := &BartNode[int]{}

	// IPv4 + IPv6
	p1 := netip.MustParsePrefix("10.0.0.0/8")
	p2 := netip.MustParsePrefix("10.1.0.0/16")
	p3 := netip.MustParsePrefix("2001:db8::/32")

	if exists := n.Insert(p1, 100, 0); exists {
		t.Errorf("Insert(%v) first time exists=true, want false", p1)
	}
	if exists := n.Insert(p2, 200, 0); exists {
		t.Errorf("Insert(%v) first time exists=true, want false", p2)
	}
	if exists := n.Insert(p3, 300, 0); exists {
		t.Errorf("Insert(%v) first time exists=true, want false", p3)
	}

	if v, ok := n.Get(p2); !ok || v != 200 {
		t.Errorf("Get(%v)=(%d,%v), want (200,true)", p2, v, ok)
	}
	if exists := n.Delete(p1); !exists {
		t.Errorf("Delete(%v) exists=false, want true", p1)
	}
	if _, ok := n.Get(p1); ok {
		t.Errorf("Get(%v) ok=true after delete, want false", p1)
	}
}

func TestBartNode_Persist_InsertPersist_DeletePersist_CopyOnWrite(t *testing.T) {
	t.Parallel()
	base := &BartNode[int]{}
	pBase := netip.MustParsePrefix("10.0.0.0/8")
	pNew := netip.MustParsePrefix("10.1.0.0/16")

	base.Insert(pBase, 1, 0)
	alias := base.CloneFlat(nil)

	// InsertPersist should not affect alias
	if exists := base.InsertPersist(func(v int) int { return v }, pNew, 2, 0); exists {
		t.Errorf("InsertPersist(%v) exists=true on first insert, want false", pNew)
	}
	if _, ok := alias.Get(pNew); ok {
		t.Errorf("alias Get(%v)=true, want false (COW)", pNew)
	}

	// DeletePersist should not affect alias
	if exists := base.DeletePersist(nil, pBase); !exists {
		t.Errorf("DeletePersist(%v) exists=false, want true", pBase)
	}
	if _, ok := alias.Get(pBase); !ok {
		t.Errorf("alias lost %v after DeletePersist on base", pBase)
	}
}

func TestBartNode_Modify_Lifecycle(t *testing.T) {
	t.Parallel()
	n := &BartNode[int]{}
	p := netip.MustParsePrefix("192.168.0.0/16")

	// insert via Modify
	d := n.Modify(p, func(_ int, found bool) (int, bool) {
		if found {
			t.Fatal("found=true on first Modify insert")
		}
		return 42, false
	})
	if d != 1 {
		t.Errorf("Modify insert delta=%d, want 1", d)
	}

	// update via Modify
	d = n.Modify(p, func(v int, found bool) (int, bool) {
		if !found || v != 42 {
			t.Fatalf("expected found=true and v=42, got found=%v v=%d", found, v)
		}
		return 100, false
	})
	if d != 0 {
		t.Errorf("Modify update delta=%d, want 0", d)
	}

	// delete via Modify
	d = n.Modify(p, func(v int, found bool) (int, bool) {
		if !found || v != 100 {
			t.Fatalf("expected found=true and v=100, got found=%v v=%d", found, v)
		}
		return 0, true
	})
	if d != -1 {
		t.Errorf("Modify delete delta=%d, want -1", d)
	}
}

func TestBartNode_EqualRec(t *testing.T) {
	t.Parallel()
	a := &BartNode[int]{}
	b := &BartNode[int]{}

	ps := []struct {
		p netip.Prefix
		v int
	}{
		{netip.MustParsePrefix("10.0.0.0/8"), 1},
		{netip.MustParsePrefix("10.1.0.0/16"), 2},
		{netip.MustParsePrefix("2001:db8::/32"), 3},
	}
	for _, x := range ps {
		a.Insert(x.p, x.v, 0)
		b.Insert(x.p, x.v, 0)
	}
	if !a.EqualRec(b) {
		t.Fatal("EqualRec: identical tries reported as not equal")
	}

	// diverge
	a.Insert(netip.MustParsePrefix("10.2.0.0/16"), 9, 0)
	if a.EqualRec(b) {
		t.Fatal("EqualRec: different tries reported as equal")
	}
}

func TestBartNode_Stats_Dump_Fprint_DirectItems(t *testing.T) {
	t.Parallel()
	n := &BartNode[int]{}

	pfx4 := []struct {
		p netip.Prefix
		v int
	}{
		{netip.MustParsePrefix("10.0.0.0/8"), 1},
		{netip.MustParsePrefix("10.1.0.0/16"), 2},
	}
	pfx6 := []struct {
		p netip.Prefix
		v int
	}{
		{netip.MustParsePrefix("2001:db8::/32"), 3},
	}
	for _, x := range pfx4 {
		n.Insert(x.p, x.v, 0)
	}
	for _, x := range pfx6 {
		n.Insert(x.p, x.v, 0)
	}

	// Stats
	s := n.StatsRec()
	if sum := s.Prefixes + s.Leaves + s.Fringes; sum != len(pfx4)+len(pfx6) {
		t.Fatalf("StatsRec.Prefixes+Leaves+Fringes=%d, want %d", sum, len(pfx4)+len(pfx6))
	}

	// DumpRec
	var dump bytes.Buffer
	n.DumpRec(&dump, StridePath{}, 0, true)
	if out := dump.String(); !strings.Contains(out, "10.0.0.0/8") {
		t.Errorf("DumpRec output missing 10.0.0.0/8: %s", out)
	}

	// FprintRec
	var tree bytes.Buffer
	start := TrieItem[int]{Node: n, Path: StridePath{}, Idx: 0, Is4: true}
	if err := n.FprintRec(&tree, start, ""); err != nil {
		t.Fatalf("FprintRec error: %v", err)
	}
	if out := tree.String(); !strings.Contains(out, "10.1.0.0/16") {
		t.Errorf("FprintRec output missing 10.1.0.0/16: %s", out)
	}

	// DirectItemsRec
	items := n.DirectItemsRec(0, StridePath{}, 0, true)
	if len(items) == 0 {
		t.Errorf("DirectItemsRec returned no items")
	}
}

func TestBartNode_AllRec_and_AllRecSorted(t *testing.T) {
	t.Parallel()
	n := &BartNode[int]{}
	pfxs := []struct {
		p netip.Prefix
		v int
	}{
		{mpp("10.1.0.0/16"), 2},
		{mpp("10.0.0.0/8"), 1},
		{mpp("192.168.0.0/16"), 3},
	}
	for _, x := range pfxs {
		n.Insert(x.p, x.v, 0)
	}

	// AllRec: presence
	var got []netip.Prefix
	n.AllRec(StridePath{}, 0, true, func(p netip.Prefix, _ int) bool {
		got = append(got, p)
		return true
	})
	if len(got) != len(pfxs) {
		t.Fatalf("AllRec len=%d, want %d", len(got), len(pfxs))
	}

	// AllRecSorted: verify sorted
	var sorted []netip.Prefix
	n.AllRecSorted(StridePath{}, 0, true, func(p netip.Prefix, _ int) bool {
		sorted = append(sorted, p)
		return true
	})
	if !slices.IsSortedFunc(sorted, CmpPrefix) {
		t.Fatalf("AllRecSorted not sorted: %v", sorted)
	}
}

func TestBartNode_Supernets_and_Subnets(t *testing.T) {
	t.Parallel()
	n := &BartNode[int]{}
	n.Insert(mpp("10.0.0.0/8"), 1, 0)
	n.Insert(mpp("10.1.0.0/16"), 2, 0)
	n.Insert(mpp("10.1.1.0/24"), 3, 0)
	n.Insert(mpp("192.168.0.0/16"), 9, 0)

	// Supernets(10.1.1/24)
	var supers []netip.Prefix
	n.Supernets(mpp("10.1.1.0/24"), func(p netip.Prefix, _ int) bool {
		supers = append(supers, p)
		return true
	})
	if len(supers) != 3 {
		t.Fatalf("Supernets count=%d, want 3", len(supers))
	}

	// Subnets(10/8)
	var subs []netip.Prefix
	n.Subnets(mpp("10.0.0.0/8"), func(p netip.Prefix, _ int) bool {
		subs = append(subs, p)
		return true
	})
	if len(subs) != 3 {
		t.Fatalf("Subnets count=%d, want 3", len(subs))
	}
}

func TestBartNode_Overlaps_Basic_and_PrefixAtDepth(t *testing.T) {
	t.Parallel()
	a := &BartNode[int]{}
	b := &BartNode[int]{}

	a.Insert(mpp("10.0.0.0/8"), 1, 0)
	a.Insert(mpp("192.168.0.0/16"), 2, 0)

	b.Insert(mpp("172.16.0.0/12"), 3, 0)
	if a.Overlaps(b, 0) {
		t.Fatal("expected no overlap")
	}

	b.Insert(mpp("10.1.0.0/16"), 4, 0)
	if !a.Overlaps(b, 0) {
		t.Fatal("expected overlap")
	}

	if !a.OverlapsPrefixAtDepth(mpp("10.1.1.0/24"), 0) {
		t.Fatal("OverlapsPrefixAtDepth should be true")
	}
	if a.OverlapsPrefixAtDepth(mpp("11.0.0.0/8"), 0) {
		t.Fatal("OverlapsPrefixAtDepth should be false")
	}
}

func TestBartNode_UnionRec_and_UnionRecPersist(t *testing.T) {
	t.Parallel()
	n1 := &BartNode[int]{}
	n2 := &BartNode[int]{}

	n1.Insert(mpp("10.0.0.0/8"), 1, 0)
	n2.Insert(mpp("10.1.0.0/16"), 2, 0)
	n2.Insert(mpp("172.16.0.0/12"), 3, 0)

	dups := n1.UnionRec(nil, n2, 0)
	if dups != 0 {
		t.Fatalf("UnionRec duplicates=%d, want 0", dups)
	}
	// Validate presence and values
	tests := []struct {
		p string
		v int
	}{
		{"10.0.0.0/8", 1},
		{"10.1.0.0/16", 2},
		{"172.16.0.0/12", 3},
	}
	for _, tc := range tests {
		v, ok := n1.Get(mpp(tc.p))
		if !ok || v != tc.v {
			t.Fatalf("after union Get(%s)=(%d,%v), want (%d,true)", tc.p, v, ok, tc.v)
		}
	}

	// Persist flavor must not mutate alias
	base := &BartNode[int]{}
	alias := base.CloneFlat(nil)
	other := &BartNode[int]{}
	other.Insert(mpp("2001:db8::/32"), 9, 0)
	_ = base.UnionRecPersist(nil, other, 0)
	if _, ok := alias.Get(mpp("2001:db8::/32")); ok {
		t.Fatalf("alias changed after UnionRecPersist, want unchanged")
	}
}

func TestBartNode_FprintRec_and_DirectItemsRec_Smoke(t *testing.T) {
	t.Parallel()
	n := &BartNode[int]{}
	n.Insert(mpp("10.0.0.0/8"), 10, 0)
	n.Insert(mpp("10.1.0.0/16"), 11, 0)

	var buf bytes.Buffer
	start := TrieItem[int]{Node: n, Path: StridePath{}, Idx: 0, Is4: true}
	if err := n.FprintRec(&buf, start, ""); err != nil {
		t.Fatalf("FprintRec error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "10.1.0.0/16") {
		t.Fatalf("FprintRec output missing expected prefix; got: %s", out)
	}

	items := n.DirectItemsRec(0, StridePath{}, 0, true)
	if len(items) == 0 {
		t.Fatal("DirectItemsRec returned no items")
	}
}

func TestBartNode_OverlapsRoutes_DirectIntersection(t *testing.T) {
	t.Parallel()
	a := &BartNode[int]{}
	b := &BartNode[int]{}

	// Both nodes have identical prefix at same index
	a.InsertPrefix(32, 100)
	b.InsertPrefix(32, 200)

	if !a.OverlapsRoutes(b) {
		t.Error("OverlapsRoutes should return true for identical indices")
	}
}

func TestBartNode_OverlapsRoutes_LPM_Containment(t *testing.T) {
	t.Parallel()
	a := &BartNode[int]{}
	b := &BartNode[int]{}

	// Insert prefix in a that will be contained by prefix in b
	// Index 64 is covered by index 32 in the CBT (32 is parent of 64)
	a.InsertPrefix(64, 100)
	b.InsertPrefix(32, 200)

	if !a.OverlapsRoutes(b) {
		t.Error("OverlapsRoutes should detect LPM containment (a contains b's prefix)")
	}

	// Test reverse direction
	c := &BartNode[int]{}
	d := &BartNode[int]{}
	c.InsertPrefix(32, 100)
	d.InsertPrefix(64, 200)

	if !c.OverlapsRoutes(d) {
		t.Error("OverlapsRoutes should detect LPM containment (reverse)")
	}
}

func TestBartNode_OverlapsRoutes_NoOverlap(t *testing.T) {
	t.Parallel()
	a := &BartNode[int]{}
	b := &BartNode[int]{}

	// Insert non-overlapping prefixes - use sibling indices
	// Indices 2 and 3 are siblings (children of 1), so they don't overlap
	a.InsertPrefix(2, 100)
	b.InsertPrefix(3, 200)

	if a.OverlapsRoutes(b) {
		t.Error("OverlapsRoutes should return false for non-overlapping prefixes")
	}

	// Test with more specific non-overlapping indices
	c := &BartNode[int]{}
	d := &BartNode[int]{}
	c.InsertPrefix(4, 100) // Left child of 2
	d.InsertPrefix(6, 200) // Left child of 3

	if c.OverlapsRoutes(d) {
		t.Error("OverlapsRoutes should return false for non-overlapping sibling subtrees")
	}
}

func TestBartNode_OverlapsRoutes_EmptyNodes(t *testing.T) {
	t.Parallel()
	a := &BartNode[int]{}
	b := &BartNode[int]{}

	if a.OverlapsRoutes(b) {
		t.Error("OverlapsRoutes should return false for empty nodes")
	}

	a.InsertPrefix(32, 100)
	if a.OverlapsRoutes(b) {
		t.Error("OverlapsRoutes should return false when one node is empty")
	}
}

func TestBartNode_OverlapsRoutes_MultiplePrefix_WithOverlap(t *testing.T) {
	t.Parallel()
	a := &BartNode[int]{}
	b := &BartNode[int]{}

	// Insert multiple prefixes with one overlapping pair
	a.InsertPrefix(16, 100)
	a.InsertPrefix(64, 101)
	a.InsertPrefix(128, 102)

	b.InsertPrefix(8, 200)
	b.InsertPrefix(32, 201) // This is parent of 64, so overlaps
	b.InsertPrefix(255, 202)

	if !a.OverlapsRoutes(b) {
		t.Error("OverlapsRoutes should detect overlap in multi-prefix scenario")
	}
}

func TestBartNode_OverlapsRoutes_Uint8_Boundary(t *testing.T) {
	t.Parallel()
	a := &BartNode[int]{}
	b := &BartNode[int]{}

	// Test uint8 boundary (255)
	a.InsertPrefix(255, 100)
	b.InsertPrefix(254, 200)

	// These are siblings in the CBT, should not overlap
	if a.OverlapsRoutes(b) {
		t.Error("OverlapsRoutes returned unexpected overlap for sibling indices at boundary")
	}

	// But 255 and 255 should overlap
	b.InsertPrefix(255, 300)
	if !a.OverlapsRoutes(b) {
		t.Error("OverlapsRoutes should detect overlap at index 255")
	}
}

func TestBartNode_OverlapsChildrenIn_BitsetPath(t *testing.T) {
	t.Parallel()
	n := &BartNode[int]{}
	o := &BartNode[int]{}

	// Setup: n has prefix at index 1 (root, covers everything)
	n.InsertPrefix(1, 100)

	// Insert many children in o (triggers bitset path)
	for i := range uint8(20) {
		child := &BartNode[int]{}
		child.InsertPrefix(1, int(i))
		o.InsertChild(i, child)
	}

	if !n.OverlapsChildrenIn(o) {
		t.Error("OverlapsChildrenIn should detect overlap using bitset path")
	}
}

func TestBartNode_OverlapsChildrenIn_IterationPath(t *testing.T) {
	t.Parallel()
	n := &BartNode[int]{}
	o := &BartNode[int]{}

	// Setup: n has prefix that covers left half (index 2 covers octets 0-127)
	n.InsertPrefix(2, 100)

	// Insert a child in the right half (should not overlap)
	child := &BartNode[int]{}
	child.InsertPrefix(1, 1)
	o.InsertChild(128, child) // OctetToIdx(128) = 192, not covered by index 2

	if n.OverlapsChildrenIn(o) {
		t.Error("OverlapsChildrenIn should not detect overlap for non-overlapping children")
	}

	// Add overlapping child in the left half
	child2 := &BartNode[int]{}
	child2.InsertPrefix(1, 2)
	o.InsertChild(0, child2) // OctetToIdx(0) = 128, which is covered by index 2

	if !n.OverlapsChildrenIn(o) {
		t.Error("OverlapsChildrenIn should detect overlap using iteration path")
	}
}

func TestBartNode_OverlapsChildrenIn_EmptyCases(t *testing.T) {
	t.Parallel()
	n := &BartNode[int]{}
	o := &BartNode[int]{}

	// Both empty
	if n.OverlapsChildrenIn(o) {
		t.Error("OverlapsChildrenIn should return false for empty nodes")
	}

	// n has prefixes but o has no children
	n.InsertPrefix(32, 100)
	if n.OverlapsChildrenIn(o) {
		t.Error("OverlapsChildrenIn should return false when o has no children")
	}

	// n has no prefixes but o has children
	n2 := &BartNode[int]{}
	child := &BartNode[int]{}
	o.InsertChild(10, child)
	if n2.OverlapsChildrenIn(o) {
		t.Error("OverlapsChildrenIn should return false when n has no prefixes")
	}
}

func TestBartNode_OverlapsTwoChildren_AllCombinations(t *testing.T) {
	t.Parallel()

	// Test all 9 combinations: node-node, node-leaf, node-fringe,
	// leaf-node, leaf-leaf, leaf-fringe, fringe-node, fringe-leaf, fringe-fringe

	t.Run("node-node_overlap", func(t *testing.T) {
		t.Parallel()
		n1 := &BartNode[int]{}
		n2 := &BartNode[int]{}
		n1.InsertPrefix(32, 100)
		n2.InsertPrefix(32, 200)

		parent := &BartNode[int]{}
		if !parent.OverlapsTwoChildren(n1, n2, 0) {
			t.Error("node-node should overlap when prefixes overlap")
		}
	})

	t.Run("node-node_no_overlap", func(t *testing.T) {
		t.Parallel()
		n1 := &BartNode[int]{}
		n2 := &BartNode[int]{}
		// Use sibling indices that don't overlap
		n1.InsertPrefix(2, 100)
		n2.InsertPrefix(3, 200)

		parent := &BartNode[int]{}
		if parent.OverlapsTwoChildren(n1, n2, 0) {
			t.Error("node-node should not overlap when prefixes don't overlap")
		}
	})

	t.Run("node-leaf", func(t *testing.T) {
		t.Parallel()
		node := &BartNode[int]{}
		leaf := &LeafNode[int]{
			Prefix: mpp("10.0.0.0/8"),
			Value:  100,
		}

		// Insert prefix that overlaps with leaf
		node.Insert(mpp("10.0.0.0/16"), 200, 0)

		parent := &BartNode[int]{}
		if !parent.OverlapsTwoChildren(node, leaf, 0) {
			t.Error("node-leaf should overlap when node contains overlapping prefix")
		}
	})

	t.Run("node-fringe_always_overlap", func(t *testing.T) {
		t.Parallel()
		node := &BartNode[int]{}
		fringe := &FringeNode[int]{
			Value: 100,
		}

		parent := &BartNode[int]{}
		if !parent.OverlapsTwoChildren(node, fringe, 0) {
			t.Error("node-fringe should always overlap")
		}
	})

	t.Run("leaf-leaf_overlap", func(t *testing.T) {
		t.Parallel()
		leaf1 := &LeafNode[int]{
			Prefix: mpp("10.0.0.0/8"),
			Value:  100,
		}
		leaf2 := &LeafNode[int]{
			Prefix: mpp("10.0.0.0/16"),
			Value:  200,
		}

		parent := &BartNode[int]{}
		if !parent.OverlapsTwoChildren(leaf1, leaf2, 0) {
			t.Error("leaf-leaf should overlap when prefixes overlap")
		}
	})

	t.Run("leaf-leaf_no_overlap", func(t *testing.T) {
		t.Parallel()
		leaf1 := &LeafNode[int]{
			Prefix: mpp("10.0.0.0/8"),
			Value:  100,
		}
		leaf2 := &LeafNode[int]{
			Prefix: mpp("192.168.0.0/16"),
			Value:  200,
		}

		parent := &BartNode[int]{}
		if parent.OverlapsTwoChildren(leaf1, leaf2, 0) {
			t.Error("leaf-leaf should not overlap when prefixes don't overlap")
		}
	})

	t.Run("leaf-fringe_always_overlap", func(t *testing.T) {
		t.Parallel()
		leaf := &LeafNode[int]{
			Prefix: mpp("10.0.0.0/8"),
			Value:  100,
		}
		fringe := &FringeNode[int]{
			Value: 200,
		}

		parent := &BartNode[int]{}
		if !parent.OverlapsTwoChildren(leaf, fringe, 0) {
			t.Error("leaf-fringe should always overlap")
		}
	})

	t.Run("fringe-fringe_always_overlap", func(t *testing.T) {
		t.Parallel()
		fringe1 := &FringeNode[int]{
			Value: 100,
		}
		fringe2 := &FringeNode[int]{
			Value: 200,
		}

		parent := &BartNode[int]{}
		if !parent.OverlapsTwoChildren(fringe1, fringe2, 0) {
			t.Error("fringe-fringe should always overlap")
		}
	})
}

func TestBartNode_Overlaps_CompleteFlow(t *testing.T) {
	t.Parallel()

	t.Run("routes_overlap", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		a.Insert(mpp("10.0.0.0/8"), 100, 0)
		b.Insert(mpp("10.0.0.0/16"), 200, 0)

		if !a.Overlaps(b, 0) {
			t.Error("Overlaps should detect route overlap")
		}
	})

	t.Run("children_overlap", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		// a has prefix that will overlap with child in b
		// Use specific addresses to ensure overlap
		a.Insert(mpp("10.0.0.0/8"), 100, 0)
		b.Insert(mpp("10.1.0.0/16"), 200, 0)

		if !a.Overlaps(b, 0) {
			t.Error("Overlaps should detect child overlap")
		}
	})

	t.Run("same_children_overlap", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		// Both have children at same address with overlapping content
		a.Insert(mpp("10.1.0.0/16"), 100, 0)
		b.Insert(mpp("10.1.0.0/24"), 200, 0)

		if !a.Overlaps(b, 0) {
			t.Error("Overlaps should detect same-children overlap")
		}
	})

	t.Run("no_overlap", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		a.Insert(mpp("10.0.0.0/8"), 100, 0)
		b.Insert(mpp("192.168.0.0/16"), 200, 0)

		if a.Overlaps(b, 0) {
			t.Error("Overlaps should return false for non-overlapping trees")
		}
	})

	t.Run("empty_nodes", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		if a.Overlaps(b, 0) {
			t.Error("Overlaps should return false for empty nodes")
		}
	})
}

func TestBartNode_OverlapsIdx(t *testing.T) {
	t.Parallel()

	t.Run("prefix_covers_idx", func(t *testing.T) {
		t.Parallel()
		n := &BartNode[int]{}
		n.InsertPrefix(1, 100) // Root covers everything

		if !n.OverlapsIdx(128) {
			t.Error("OverlapsIdx should return true when prefix covers idx")
		}
	})

	t.Run("idx_covers_routes", func(t *testing.T) {
		t.Parallel()
		n := &BartNode[int]{}
		n.InsertPrefix(64, 100) // Specific route

		// Index 32 is parent of 64, so it covers this route
		if !n.OverlapsIdx(32) {
			t.Error("OverlapsIdx should return true when idx covers routes")
		}
	})

	t.Run("idx_overlaps_child", func(t *testing.T) {
		t.Parallel()
		n := &BartNode[int]{}
		child := &BartNode[int]{}
		child.InsertPrefix(1, 100)

		// Insert child at address 10
		n.InsertChild(10, child)

		// Check if an idx that covers address 10 returns true
		// OctetToIdx(10) = 128 + 10>>1 = 133
		// We need an ancestor of 133
		idx := art.OctetToIdx(10)
		// Find parent

		found := false
		for ; idx >= 1; idx >>= 1 {
			if n.OverlapsIdx(idx) {
				// This should eventually return true for ancestors
				// But since we have no prefixes, only children, it depends on allotment tables
				found = true
				break
			}
		}
		if !found {
			t.Error("expected some ancestor idx to overlap child")
		}
	})

	t.Run("no_overlap", func(t *testing.T) {
		t.Parallel()
		n := &BartNode[int]{}
		n.InsertPrefix(2, 100) // Left subtree

		// Index 3 is right subtree, should not overlap
		if n.OverlapsIdx(3) {
			t.Error("OverlapsIdx should return false for non-overlapping idx")
		}
	})
}

//nolint:gocyclo
func TestBartNode_UnionRec_AllCombinations(t *testing.T) {
	t.Parallel()

	cloneFn := cloneFnFactory[int]()

	t.Run("null_plus_node", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		childNode := &BartNode[int]{}
		childNode.InsertPrefix(32, 100)
		b.InsertChild(10, childNode)

		duplicates := a.UnionRec(cloneFn, b, 0)
		if duplicates != 0 {
			t.Errorf("Expected 0 duplicates, got %d", duplicates)
		}

		child, exists := a.GetChild(10)
		if !exists {
			t.Error("Child should exist after union")
		}
		if childNode, ok := child.(*BartNode[int]); !ok {
			t.Error("Child should be a BartNode")
		} else if val, ok := childNode.GetPrefix(32); !ok || val != 100 {
			t.Error("Child node should have prefix 32 with value 100")
		}
	})

	t.Run("null_plus_leaf", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		leaf := &LeafNode[int]{
			Prefix: mpp("10.10.0.0/16"),
			Value:  100,
		}
		b.InsertChild(10, leaf)

		duplicates := a.UnionRec(cloneFn, b, 0)
		if duplicates != 0 {
			t.Errorf("Expected 0 duplicates, got %d", duplicates)
		}

		child, exists := a.GetChild(10)
		if !exists {
			t.Error("Leaf should exist after union")
		}
		if childLeaf, ok := child.(*LeafNode[int]); !ok {
			t.Error("Child should be a LeafNode")
		} else if childLeaf.Value != 100 {
			t.Errorf("Leaf value should be 100, got %d", childLeaf.Value)
		}
	})

	t.Run("null_plus_fringe", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		fringe := &FringeNode[int]{Value: 100}
		b.InsertChild(10, fringe)

		duplicates := a.UnionRec(cloneFn, b, 0)
		if duplicates != 0 {
			t.Errorf("Expected 0 duplicates, got %d", duplicates)
		}

		child, exists := a.GetChild(10)
		if !exists {
			t.Error("Fringe should exist after union")
		}
		if childFringe, ok := child.(*FringeNode[int]); !ok {
			t.Error("Child should be a FringeNode")
		} else if childFringe.Value != 100 {
			t.Errorf("Fringe value should be 100, got %d", childFringe.Value)
		}
	})

	t.Run("node_plus_node", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		childNodeA := &BartNode[int]{}
		childNodeA.InsertPrefix(32, 100)
		a.InsertChild(10, childNodeA)

		childNodeB := &BartNode[int]{}
		childNodeB.InsertPrefix(64, 200)
		b.InsertChild(10, childNodeB)

		duplicates := a.UnionRec(cloneFn, b, 0)
		if duplicates != 0 {
			t.Errorf("Expected 0 duplicates, got %d", duplicates)
		}

		child, _ := a.GetChild(10)
		mergedNode := child.(*BartNode[int])
		if val, ok := mergedNode.GetPrefix(32); !ok || val != 100 {
			t.Error("Should have prefix 32 with value 100")
		}
		if val, ok := mergedNode.GetPrefix(64); !ok || val != 200 {
			t.Error("Should have prefix 64 with value 200")
		}
	})

	t.Run("node_plus_node_with_duplicate", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		childNodeA := &BartNode[int]{}
		childNodeA.InsertPrefix(32, 100)
		a.InsertChild(10, childNodeA)

		childNodeB := &BartNode[int]{}
		childNodeB.InsertPrefix(32, 999)
		b.InsertChild(10, childNodeB)

		duplicates := a.UnionRec(cloneFn, b, 0)
		if duplicates != 1 {
			t.Errorf("Expected 1 duplicate, got %d", duplicates)
		}

		child, _ := a.GetChild(10)
		mergedNode := child.(*BartNode[int])
		if val, ok := mergedNode.GetPrefix(32); !ok || val != 999 {
			t.Errorf("Should have prefix 32 with value 999, got %d", val)
		}
	})

	t.Run("node_plus_leaf", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		childNodeA := &BartNode[int]{}
		childNodeA.InsertPrefix(32, 100)
		a.InsertChild(10, childNodeA)

		leaf := &LeafNode[int]{
			Prefix: mpp("10.10.1.0/24"),
			Value:  200,
		}
		b.InsertChild(10, leaf)

		duplicates := a.UnionRec(cloneFn, b, 0)
		if duplicates != 0 {
			t.Errorf("Expected 0 duplicates, got %d", duplicates)
		}

		child, _ := a.GetChild(10)
		mergedNode := child.(*BartNode[int])
		if mergedNode.PrefixCount() == 0 && mergedNode.ChildCount() == 0 {
			t.Error("Node should not be empty after inserting leaf")
		}
	})

	t.Run("node_plus_fringe", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		childNodeA := &BartNode[int]{}
		childNodeA.InsertPrefix(32, 100)
		a.InsertChild(10, childNodeA)

		fringe := &FringeNode[int]{Value: 200}
		b.InsertChild(10, fringe)

		a.UnionRec(cloneFn, b, 0)

		child, _ := a.GetChild(10)
		mergedNode := child.(*BartNode[int])
		if val, ok := mergedNode.GetPrefix(1); !ok || val != 200 {
			t.Error("Node should have fringe prefix (idx=1)")
		}
	})

	t.Run("leaf_plus_node", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		leafA := &LeafNode[int]{
			Prefix: mpp("10.10.0.0/16"),
			Value:  100,
		}
		a.InsertChild(10, leafA)

		childNodeB := &BartNode[int]{}
		childNodeB.InsertPrefix(32, 200)
		b.InsertChild(10, childNodeB)

		duplicates := a.UnionRec(cloneFn, b, 0)
		if duplicates != 0 {
			t.Errorf("Expected 0 duplicates, got %d", duplicates)
		}

		child, exists := a.GetChild(10)
		if !exists {
			t.Fatal("Child should exist")
		}
		newNode, ok := child.(*BartNode[int])
		if !ok {
			t.Fatal("Child should be a BartNode after union")
		}
		if newNode.PrefixCount() == 0 && newNode.ChildCount() == 0 {
			t.Error("New node should not be empty")
		}
	})

	t.Run("leaf_plus_leaf_same_prefix", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		prefix := mpp("10.10.0.0/16")
		leafA := &LeafNode[int]{Prefix: prefix, Value: 100}
		leafB := &LeafNode[int]{Prefix: prefix, Value: 999}

		a.InsertChild(10, leafA)
		b.InsertChild(10, leafB)

		duplicates := a.UnionRec(cloneFn, b, 0)
		if duplicates != 1 {
			t.Errorf("Expected 1 duplicate, got %d", duplicates)
		}

		child, _ := a.GetChild(10)
		leaf := child.(*LeafNode[int])
		if leaf.Value != 999 {
			t.Errorf("Leaf value should be 999, got %d", leaf.Value)
		}
	})

	t.Run("leaf_plus_leaf_different_prefix", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		leafA := &LeafNode[int]{
			Prefix: mpp("10.10.0.0/16"),
			Value:  100,
		}
		leafB := &LeafNode[int]{
			Prefix: mpp("10.10.1.0/24"),
			Value:  200,
		}

		a.InsertChild(10, leafA)
		b.InsertChild(10, leafB)

		duplicates := a.UnionRec(cloneFn, b, 0)
		if duplicates != 0 {
			t.Errorf("Expected 0 duplicates, got %d", duplicates)
		}

		child, _ := a.GetChild(10)
		if _, ok := child.(*BartNode[int]); !ok {
			t.Error("Should create new BartNode when merging different leaves")
		}
	})

	t.Run("leaf_plus_fringe", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		leafA := &LeafNode[int]{
			Prefix: mpp("10.10.0.0/16"),
			Value:  100,
		}
		fringeB := &FringeNode[int]{Value: 200}

		a.InsertChild(10, leafA)
		b.InsertChild(10, fringeB)

		a.UnionRec(cloneFn, b, 0)

		child, _ := a.GetChild(10)
		if _, ok := child.(*BartNode[int]); !ok {
			t.Error("Should create new BartNode when merging leaf + fringe")
		}
	})

	t.Run("fringe_plus_node", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		fringeA := &FringeNode[int]{Value: 100}
		childNodeB := &BartNode[int]{}
		childNodeB.InsertPrefix(32, 200)

		a.InsertChild(10, fringeA)
		b.InsertChild(10, childNodeB)

		a.UnionRec(cloneFn, b, 0)

		child, _ := a.GetChild(10)
		newNode, ok := child.(*BartNode[int])
		if !ok {
			t.Fatal("Should create new BartNode when merging fringe + node")
		}
		if newNode.PrefixCount() == 0 {
			t.Error("New node should have prefixes")
		}
	})

	t.Run("fringe_plus_leaf", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		fringeA := &FringeNode[int]{Value: 100}
		leafB := &LeafNode[int]{
			Prefix: mpp("10.10.0.0/16"),
			Value:  200,
		}

		a.InsertChild(10, fringeA)
		b.InsertChild(10, leafB)

		a.UnionRec(cloneFn, b, 0)

		child, _ := a.GetChild(10)
		if _, ok := child.(*BartNode[int]); !ok {
			t.Error("Should create new BartNode when merging fringe + leaf")
		}
	})

	t.Run("fringe_plus_fringe", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		fringeA := &FringeNode[int]{Value: 100}
		fringeB := &FringeNode[int]{Value: 999}

		a.InsertChild(10, fringeA)
		b.InsertChild(10, fringeB)

		duplicates := a.UnionRec(cloneFn, b, 0)
		if duplicates != 1 {
			t.Errorf("Expected 1 duplicate, got %d", duplicates)
		}

		child, _ := a.GetChild(10)
		fringe := child.(*FringeNode[int])
		if fringe.Value != 999 {
			t.Errorf("Fringe value should be 999, got %d", fringe.Value)
		}
	})
}

func TestBartNode_UnionRecPersist_AllCombinations(t *testing.T) {
	t.Parallel()

	cloneFn := cloneFnFactory[int]()

	t.Run("null_plus_node_persist", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		childNodeB := &BartNode[int]{}
		childNodeB.InsertPrefix(32, 100)
		b.InsertChild(10, childNodeB)

		duplicates := a.UnionRecPersist(cloneFn, b, 0)
		if duplicates != 0 {
			t.Errorf("Expected 0 duplicates, got %d", duplicates)
		}

		child, exists := a.GetChild(10)
		if !exists {
			t.Fatal("Child should exist")
		}
		childNode := child.(*BartNode[int])

		// Modify original, check clone unchanged
		childNodeB.InsertPrefix(64, 999)
		if _, ok := childNode.GetPrefix(64); ok {
			t.Error("Clone should not reflect changes to original")
		}
	})

	t.Run("null_plus_leaf_persist", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		originalLeaf := &LeafNode[int]{
			Prefix: mpp("10.10.0.0/16"),
			Value:  100,
		}
		b.InsertChild(10, originalLeaf)

		duplicates := a.UnionRecPersist(cloneFn, b, 0)
		if duplicates != 0 {
			t.Errorf("Expected 0 duplicates, got %d", duplicates)
		}

		child, _ := a.GetChild(10)
		clonedLeaf := child.(*LeafNode[int])

		// Modify original, check clone unchanged
		originalLeaf.Value = 999
		if clonedLeaf.Value == 999 {
			t.Error("Clone should not reflect changes to original")
		}
	})

	t.Run("null_plus_fringe_persist", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		fringe := &FringeNode[int]{Value: 100}
		b.InsertChild(10, fringe)

		duplicates := a.UnionRecPersist(cloneFn, b, 0)
		if duplicates != 0 {
			t.Errorf("Expected 0 duplicates, got %d", duplicates)
		}

		child, exists := a.GetChild(10)
		if !exists {
			t.Fatal("Fringe should exist")
		}
		if _, ok := child.(*FringeNode[int]); !ok {
			t.Error("Child should be a FringeNode")
		}
	})

	t.Run("node_plus_node_persist", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		childNodeA := &BartNode[int]{}
		childNodeA.InsertPrefix(32, 100)
		originalChildA := childNodeA
		a.InsertChild(10, childNodeA)

		childNodeB := &BartNode[int]{}
		childNodeB.InsertPrefix(64, 200)
		b.InsertChild(10, childNodeB)

		duplicates := a.UnionRecPersist(cloneFn, b, 0)
		if duplicates != 0 {
			t.Errorf("Expected 0 duplicates, got %d", duplicates)
		}

		child, _ := a.GetChild(10)
		mergedNode := child.(*BartNode[int])

		if mergedNode == originalChildA {
			t.Error("Child node should be cloned, not reused")
		}

		if _, ok := mergedNode.GetPrefix(32); !ok {
			t.Error("Should have prefix 32")
		}
		if _, ok := mergedNode.GetPrefix(64); !ok {
			t.Error("Should have prefix 64")
		}

		if originalChildA.PrefixCount() != 1 {
			t.Error("Original should remain unchanged")
		}
	})

	t.Run("node_plus_leaf_persist", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		childNodeA := &BartNode[int]{}
		childNodeA.InsertPrefix(32, 100)
		originalChildA := childNodeA
		a.InsertChild(10, childNodeA)

		leaf := &LeafNode[int]{
			Prefix: mpp("10.10.1.0/24"),
			Value:  200,
		}
		b.InsertChild(10, leaf)

		a.UnionRecPersist(cloneFn, b, 0)

		child, _ := a.GetChild(10)
		if child == originalChildA {
			t.Error("Child should be cloned")
		}
	})

	t.Run("node_plus_fringe_persist", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		childNodeA := &BartNode[int]{}
		childNodeA.InsertPrefix(32, 100)
		originalChildA := childNodeA
		a.InsertChild(10, childNodeA)

		fringe := &FringeNode[int]{Value: 200}
		b.InsertChild(10, fringe)

		a.UnionRecPersist(cloneFn, b, 0)

		child, _ := a.GetChild(10)
		if child == originalChildA {
			t.Error("Child should be cloned")
		}
	})

	t.Run("leaf_plus_node_persist", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		originalLeaf := &LeafNode[int]{
			Prefix: mpp("10.10.0.0/16"),
			Value:  100,
		}
		a.InsertChild(10, originalLeaf)

		childNodeB := &BartNode[int]{}
		childNodeB.InsertPrefix(32, 200)
		b.InsertChild(10, childNodeB)

		a.UnionRecPersist(cloneFn, b, 0)

		child, _ := a.GetChild(10)
		if _, ok := child.(*BartNode[int]); !ok {
			t.Error("Should create new BartNode")
		}
	})

	t.Run("leaf_plus_leaf_same_prefix_persist", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		prefix := mpp("10.10.0.0/16")
		originalLeaf := &LeafNode[int]{Prefix: prefix, Value: 100}
		leafB := &LeafNode[int]{Prefix: prefix, Value: 999}

		a.InsertChild(10, originalLeaf)
		b.InsertChild(10, leafB)

		duplicates := a.UnionRecPersist(cloneFn, b, 0)
		if duplicates != 1 {
			t.Errorf("Expected 1 duplicate, got %d", duplicates)
		}

		child, _ := a.GetChild(10)
		leaf := child.(*LeafNode[int])
		if leaf.Value != 999 {
			t.Errorf("Value should be 999, got %d", leaf.Value)
		}
	})

	t.Run("leaf_plus_leaf_different_prefix_persist", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		leafA := &LeafNode[int]{
			Prefix: mpp("10.10.0.0/16"),
			Value:  100,
		}
		leafB := &LeafNode[int]{
			Prefix: mpp("10.10.1.0/24"),
			Value:  200,
		}

		a.InsertChild(10, leafA)
		b.InsertChild(10, leafB)

		a.UnionRecPersist(cloneFn, b, 0)

		child, _ := a.GetChild(10)
		if _, ok := child.(*BartNode[int]); !ok {
			t.Error("Should create new BartNode")
		}
	})

	t.Run("leaf_plus_fringe_persist", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		leafA := &LeafNode[int]{
			Prefix: mpp("10.10.0.0/16"),
			Value:  100,
		}
		fringeB := &FringeNode[int]{Value: 200}

		a.InsertChild(10, leafA)
		b.InsertChild(10, fringeB)

		a.UnionRecPersist(cloneFn, b, 0)

		child, _ := a.GetChild(10)
		if _, ok := child.(*BartNode[int]); !ok {
			t.Error("Should create new BartNode")
		}
	})

	t.Run("fringe_plus_node_persist", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		fringeA := &FringeNode[int]{Value: 100}
		a.InsertChild(10, fringeA)

		childNodeB := &BartNode[int]{}
		childNodeB.InsertPrefix(32, 200)
		b.InsertChild(10, childNodeB)

		a.UnionRecPersist(cloneFn, b, 0)

		child, _ := a.GetChild(10)
		if _, ok := child.(*BartNode[int]); !ok {
			t.Error("Should create new BartNode")
		}
	})

	t.Run("fringe_plus_leaf_persist", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		fringeA := &FringeNode[int]{Value: 100}
		a.InsertChild(10, fringeA)

		leafB := &LeafNode[int]{
			Prefix: mpp("10.10.0.0/16"),
			Value:  200,
		}
		b.InsertChild(10, leafB)

		a.UnionRecPersist(cloneFn, b, 0)

		child, _ := a.GetChild(10)
		if _, ok := child.(*BartNode[int]); !ok {
			t.Error("Should create new BartNode")
		}
	})

	t.Run("fringe_plus_fringe_persist", func(t *testing.T) {
		t.Parallel()
		a := &BartNode[int]{}
		b := &BartNode[int]{}

		fringeA := &FringeNode[int]{Value: 100}
		fringeB := &FringeNode[int]{Value: 999}

		a.InsertChild(10, fringeA)
		b.InsertChild(10, fringeB)

		duplicates := a.UnionRecPersist(cloneFn, b, 0)
		if duplicates != 1 {
			t.Errorf("Expected 1 duplicate, got %d", duplicates)
		}

		child, _ := a.GetChild(10)
		fringe := child.(*FringeNode[int])
		if fringe.Value != 999 {
			t.Errorf("Value should be 999, got %d", fringe.Value)
		}
	})
}

//nolint:gocyclo
func TestBartNode_Modify_AllPaths(t *testing.T) {
	t.Parallel()

	t.Run("modify_at_lastOctet_delete_nonexistent", func(t *testing.T) {
		t.Parallel()
		n := &BartNode[int]{}

		// Use /0 for depth=0, lastOctetPlusOne=0, so depth == lastOctetPlusOne
		delta := n.Modify(mpp("0.0.0.0/0"), func(val int, found bool) (int, bool) {
			if found {
				t.Error("Should not find non-existent prefix")
			}
			return 0, true // request deletion
		})

		if delta != 0 {
			t.Errorf("Expected delta 0 for no-op delete, got %d", delta)
		}
	})

	t.Run("modify_at_lastOctet_delete_existing", func(t *testing.T) {
		t.Parallel()
		n := &BartNode[int]{}
		pfx := mpp("0.0.0.0/0")
		n.Insert(pfx, 100, 0)

		delta := n.Modify(pfx, func(val int, found bool) (int, bool) {
			if !found || val != 100 {
				t.Errorf("Expected found=true with val=100, got found=%v val=%d", found, val)
			}
			return 0, true // delete
		})

		if delta != -1 {
			t.Errorf("Expected delta -1 for delete, got %d", delta)
		}

		if _, ok := n.Get(pfx); ok {
			t.Error("Prefix should be deleted")
		}
	})

	t.Run("modify_at_lastOctet_insert_new", func(t *testing.T) {
		t.Parallel()
		n := &BartNode[int]{}
		pfx := mpp("0.0.0.0/0")

		delta := n.Modify(pfx, func(val int, found bool) (int, bool) {
			if found {
				t.Error("Should not find new prefix")
			}
			return 999, false // insert
		})

		if delta != 1 {
			t.Errorf("Expected delta 1 for insert, got %d", delta)
		}

		if val, ok := n.Get(pfx); !ok || val != 999 {
			t.Errorf("Expected val=999, got val=%d ok=%v", val, ok)
		}
	})

	t.Run("modify_at_lastOctet_update_existing", func(t *testing.T) {
		t.Parallel()
		n := &BartNode[int]{}
		pfx := mpp("0.0.0.0/0")
		n.Insert(pfx, 100, 0)

		delta := n.Modify(pfx, func(val int, found bool) (int, bool) {
			if !found || val != 100 {
				t.Errorf("Expected found=true with val=100")
			}
			return 999, false // update
		})

		if delta != 0 {
			t.Errorf("Expected delta 0 for update, got %d", delta)
		}

		if val, ok := n.Get(pfx); !ok || val != 999 {
			t.Errorf("Expected val=999, got val=%d ok=%v", val, ok)
		}
	})

	t.Run("modify_insert_path_compressed_fringe", func(t *testing.T) {
		t.Parallel()
		n := &BartNode[int]{}

		// Insert /8 which becomes a FringeNode at depth 0
		pfx := mpp("10.0.0.0/8")
		delta := n.Modify(pfx, func(val int, found bool) (int, bool) {
			return 100, false
		})

		if delta != 1 {
			t.Errorf("Expected delta 1, got %d", delta)
		}

		child, exists := n.GetChild(10)
		if !exists {
			t.Fatal("Child should exist")
		}

		if fringe, ok := child.(*FringeNode[int]); !ok {
			t.Error("Child should be FringeNode")
		} else if fringe.Value != 100 {
			t.Errorf("Expected value 100, got %d", fringe.Value)
		}

		// Verify we can get it back via Get
		if val, ok := n.Get(pfx); !ok || val != 100 {
			t.Errorf("Expected val=100 via Get, got val=%d ok=%v", val, ok)
		}
	})

	t.Run("modify_insert_path_compressed_leaf", func(t *testing.T) {
		t.Parallel()
		n := &BartNode[int]{}

		// Insert /16 which becomes a LeafNode directly (no intermediate BartNode)
		pfx := mpp("10.1.0.0/16")
		delta := n.Modify(pfx, func(val int, found bool) (int, bool) {
			return 200, false
		})

		if delta != 1 {
			t.Errorf("Expected delta 1, got %d", delta)
		}

		// Should have child at octets[0]=10
		child, exists := n.GetChild(10)
		if !exists {
			t.Fatal("Child should exist at octets[0]")
		}

		// This child could be a BartNode with further children, or could be a LeafNode directly
		// According to user: /16 is a leaf, no child nodes
		if leaf, ok := child.(*LeafNode[int]); !ok {
			t.Errorf("Child should be LeafNode, got %T", child)
		} else if leaf.Value != 200 {
			t.Errorf("Expected value 200, got %d", leaf.Value)
		}

		// Verify we can get it back via Get
		if val, ok := n.Get(pfx); !ok || val != 200 {
			t.Errorf("Expected val=200 via Get, got val=%d ok=%v", val, ok)
		}
	})

	t.Run("modify_delete_nonexistent_path_compressed", func(t *testing.T) {
		t.Parallel()
		n := &BartNode[int]{}

		// Try to delete where path doesn't exist
		delta := n.Modify(mpp("10.1.0.0/16"), func(val int, found bool) (int, bool) {
			return 0, true // delete
		})

		if delta != 0 {
			t.Errorf("Expected delta 0 for no-op, got %d", delta)
		}
	})

	t.Run("modify_update_leaf_same_prefix", func(t *testing.T) {
		t.Parallel()
		n := &BartNode[int]{}
		pfx := mpp("10.1.0.0/16")
		n.Insert(pfx, 100, 0)

		// Modify existing leaf
		delta := n.Modify(pfx, func(val int, found bool) (int, bool) {
			if !found || val != 100 {
				t.Errorf("Expected found=true with val=100")
			}
			return 999, false // update
		})

		if delta != 0 {
			t.Errorf("Expected delta 0 for update, got %d", delta)
		}

		if val, ok := n.Get(pfx); !ok || val != 999 {
			t.Errorf("Expected val=999, got val=%d ok=%v", val, ok)
		}
	})

	t.Run("modify_delete_leaf_same_prefix", func(t *testing.T) {
		t.Parallel()
		n := &BartNode[int]{}
		pfx := mpp("10.1.0.0/16")
		n.Insert(pfx, 100, 0)

		// Delete existing leaf
		delta := n.Modify(pfx, func(val int, found bool) (int, bool) {
			return 0, true // delete
		})

		if delta != -1 {
			t.Errorf("Expected delta -1, got %d", delta)
		}

		if n.ChildCount() != 0 {
			t.Error("Node should be empty after delete")
		}
	})

	t.Run("modify_insert_creates_node_from_leaf", func(t *testing.T) {
		t.Parallel()
		n := &BartNode[int]{}
		pfx1 := mpp("10.1.0.0/16")
		n.Insert(pfx1, 100, 0)

		// Insert different prefix that conflicts with leaf (same first two octets)
		pfx2 := mpp("10.1.1.0/24")
		delta := n.Modify(pfx2, func(val int, found bool) (int, bool) {
			if found {
				t.Error("Should not find")
			}
			return 200, false
		})

		if delta != 1 {
			t.Errorf("Expected delta 1, got %d", delta)
		}

		// Verify both prefixes exist
		if val, ok := n.Get(pfx1); !ok || val != 100 {
			t.Errorf("Original prefix should still exist with val=100, got val=%d ok=%v", val, ok)
		}
		if val, ok := n.Get(pfx2); !ok || val != 200 {
			t.Errorf("New prefix should exist with val=200, got val=%d ok=%v", val, ok)
		}
	})

	t.Run("modify_delete_noop_from_leaf_mismatch", func(t *testing.T) {
		t.Parallel()
		n := &BartNode[int]{}
		n.Insert(mpp("10.1.0.0/16"), 100, 0)

		// Try to insert but callback says delete (tests the no-op path)
		delta := n.Modify(mpp("10.1.1.0/24"), func(val int, found bool) (int, bool) {
			return 0, true // delete non-existent
		})

		if delta != 0 {
			t.Errorf("Expected delta 0, got %d", delta)
		}
	})

	t.Run("modify_update_fringe_same_prefix", func(t *testing.T) {
		t.Parallel()
		n := &BartNode[int]{}
		pfx := mpp("10.0.0.0/8")
		n.Insert(pfx, 100, 0)

		// Update fringe
		delta := n.Modify(pfx, func(val int, found bool) (int, bool) {
			if !found || val != 100 {
				t.Errorf("Expected found=true with val=100")
			}
			return 999, false
		})

		if delta != 0 {
			t.Errorf("Expected delta 0, got %d", delta)
		}

		if val, ok := n.Get(pfx); !ok || val != 999 {
			t.Errorf("Expected val=999, got val=%d ok=%v", val, ok)
		}
	})

	t.Run("modify_delete_fringe", func(t *testing.T) {
		t.Parallel()
		n := &BartNode[int]{}
		pfx := mpp("10.0.0.0/8")
		n.Insert(pfx, 100, 0)

		// Delete fringe
		delta := n.Modify(pfx, func(val int, found bool) (int, bool) {
			return 0, true
		})

		if delta != -1 {
			t.Errorf("Expected delta -1, got %d", delta)
		}

		if n.ChildCount() != 0 {
			t.Error("Node should be empty")
		}
	})

	t.Run("modify_insert_creates_node_from_fringe", func(t *testing.T) {
		t.Parallel()
		n := &BartNode[int]{}
		pfx1 := mpp("10.0.0.0/8")
		n.Insert(pfx1, 100, 0)

		// Insert non-fringe that conflicts (different path)
		pfx2 := mpp("10.1.0.0/16")
		delta := n.Modify(pfx2, func(val int, found bool) (int, bool) {
			return 200, false
		})

		if delta != 1 {
			t.Errorf("Expected delta 1, got %d", delta)
		}

		// Verify both prefixes exist
		if val, ok := n.Get(pfx1); !ok || val != 100 {
			t.Errorf("Original fringe should exist with val=100, got val=%d ok=%v", val, ok)
		}
		if val, ok := n.Get(pfx2); !ok || val != 200 {
			t.Errorf("New prefix should exist with val=200, got val=%d ok=%v", val, ok)
		}
	})

	t.Run("modify_delete_noop_from_fringe_mismatch", func(t *testing.T) {
		t.Parallel()
		n := &BartNode[int]{}
		n.Insert(mpp("10.0.0.0/8"), 100, 0)

		// Try to insert non-fringe but callback says delete
		delta := n.Modify(mpp("10.1.0.0/16"), func(val int, found bool) (int, bool) {
			return 0, true
		})

		if delta != 0 {
			t.Errorf("Expected delta 0, got %d", delta)
		}
	})
}

func TestBartNode_DumpString_IPv4_DeepSubtree(t *testing.T) {
	t.Parallel()

	var root BartNode[int]

	// Build a multi-level structure using BartNode children
	// root --(10)--> lvl1 --(1)--> lvl2
	lvl1 := &BartNode[int]{}
	lvl2 := &BartNode[int]{}

	// Populate lvl1 and lvl2 with some prefixes so the dump has content.
	// Using InsertPrefix to avoid creating Leaf/Fringe along the traversal path.
	lvl1.InsertPrefix(200, 333) // arbitrary idx with a recognizable value
	lvl2.InsertPrefix(32, 424242)
	lvl2.InsertPrefix(64, 515151)

	// Link children as BartNodes so DumpString traversal sees BartNode at each path element.
	lvl1.InsertChild(1, lvl2)
	root.InsertChild(10, lvl1)

	// 1) Dump the deeper node at path [10, 1] with values printed.
	outDeep := root.DumpString([]uint8{10, 1}, 2, true)

	if outDeep == "" {
		t.Fatalf("DumpString returned empty output")
	}
	if strings.Contains(outDeep, "ERROR:") {
		t.Fatalf("DumpString returned an error: %q", outDeep)
	}
	// Expect a depth marker and the recognizable values we inserted.
	if !strings.Contains(outDeep, "depth:") {
		t.Fatalf("dump should contain a depth marker, got: %q", outDeep)
	}
	if !strings.Contains(outDeep, "424242") || !strings.Contains(outDeep, "515151") {
		t.Fatalf("deep dump should contain lvl2 values, got: %q", outDeep)
	}

	// 2) Dump the intermediate node at path [10] with values printed.
	outLvl1 := root.DumpString([]uint8{10}, 1, true)
	if strings.Contains(outLvl1, "ERROR:") {
		t.Fatalf("DumpString (lvl1) returned an error: %q", outLvl1)
	}
	if !strings.Contains(outLvl1, "333") {
		t.Fatalf("lvl1 dump should contain value 333, got: %q", outLvl1)
	}
}

func TestBartNode_DumpString_Error_KidNotSet_AtRootStep(t *testing.T) {
	t.Parallel()

	var root BartNode[int]

	// depth=1, path[0]=10, but no kid at 10
	out := root.DumpString([]uint8{10}, 1, true)

	if out == "" || !strings.Contains(out, "ERROR:") {
		t.Fatalf("expected ERROR, got: %q", out)
	}
	if !strings.Contains(out, "NOT set in node") {
		t.Fatalf("expected 'NOT set in node' error, got: %q", out)
	}
	// Index-Hinweis und Pfad-Präsenz prüfen
	if !strings.Contains(out, "[0]") {
		t.Fatalf("expected index [0] in error, got: %q", out)
	}
}

func TestBartNode_DumpString_Error_KidNotSet_AtDeeperStep(t *testing.T) {
	t.Parallel()

	var root BartNode[int]
	lvl1 := &BartNode[int]{}

	// root --(10)--> lvl1
	root.InsertChild(10, lvl1)

	// depth=2, path[0]=10 existiert, path[1]=1 fehlt
	out := root.DumpString([]uint8{10, 1}, 2, true)

	if out == "" || !strings.Contains(out, "ERROR:") {
		t.Fatalf("expected ERROR, got: %q", out)
	}
	if !strings.Contains(out, "NOT set in node") {
		t.Fatalf("expected 'NOT set in node' error, got: %q", out)
	}
	if !strings.Contains(out, "[1]") {
		t.Fatalf("expected index [1] in error, got: %q", out)
	}
}

func TestBartNode_DumpString_Error_KidWrongType_LeafAtDeeperStep(t *testing.T) {
	t.Parallel()

	var root BartNode[int]
	lvl1 := &BartNode[int]{}
	leaf := &LeafNode[int]{Prefix: mpp("10.1.0.0/16"), Value: 42}

	// root --(10)--> lvl1 --(1)--> leaf
	lvl1.InsertChild(1, leaf)
	root.InsertChild(10, lvl1)

	out := root.DumpString([]uint8{10, 1}, 2, true)

	if out == "" || !strings.Contains(out, "ERROR:") {
		t.Fatalf("expected ERROR, got: %q", out)
	}
	if !strings.Contains(out, "NO BartNode") {
		t.Fatalf("expected 'NO BartNode' type error, got: %q", out)
	}
	if !strings.Contains(out, "[1]") {
		t.Fatalf("expected index [1] in error, got: %q", out)
	}
	// Bei Fehler sollte kein normaler Dump-Marker vorkommen
	if strings.Contains(out, "depth:") {
		t.Fatalf("unexpected normal dump output on error, got: %q", out)
	}
}

func TestBartNode_DumpString_Error_KidWrongType_FringeAtDeeperStep(t *testing.T) {
	t.Parallel()

	var root BartNode[int]
	lvl1 := &BartNode[int]{}
	fringe := &FringeNode[int]{Value: 7}

	// root --(10)--> lvl1 --(2)--> fringe
	lvl1.InsertChild(2, fringe)
	root.InsertChild(10, lvl1)

	out := root.DumpString([]uint8{10, 2}, 2, true)

	if out == "" || !strings.Contains(out, "ERROR:") {
		t.Fatalf("expected ERROR, got: %q", out)
	}
	if !strings.Contains(out, "NO BartNode") {
		t.Fatalf("expected 'NO BartNode' type error, got: %q", out)
	}
	if !strings.Contains(out, "[1]") {
		t.Fatalf("expected index [1] in error, got: %q", out)
	}
	if strings.Contains(out, "depth:") {
		t.Fatalf("unexpected normal dump output on error, got: %q", out)
	}
}
