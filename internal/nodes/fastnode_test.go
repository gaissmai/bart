// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package nodes

import (
	"bytes"
	"net/netip"
	"slices"
	"strings"
	"testing"
)

func TestFastNode_EmptyState(t *testing.T) {
	t.Parallel()
	n := &FastNode[int]{}

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
	var nilNode *FastNode[int]
	if !nilNode.IsEmpty() {
		t.Error("nil node should be empty")
	}
}

func TestFastNode_PrefixCRUD(t *testing.T) {
	t.Parallel()
	n := &FastNode[int]{}

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

func TestFastNode_Contains_ART_Coverage(t *testing.T) {
	t.Parallel()
	n := &FastNode[int]{}

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

func TestFastNode_LookupAndLookupIdx(t *testing.T) {
	t.Parallel()
	n := &FastNode[int]{}

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

func TestFastNode_ChildrenCRUD(t *testing.T) {
	t.Parallel()
	n := &FastNode[int]{}

	child := &FastNode[int]{}
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

func TestFastNode_MustGetChild_Panics(t *testing.T) {
	t.Parallel()
	n := &FastNode[int]{}

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustGetChild should panic on missing child")
		}
	}()
	_ = n.MustGetChild(42)
}

func TestFastNode_Iterators(t *testing.T) {
	t.Parallel()
	n := &FastNode[int]{}

	indices := []uint8{1, 32, 64, 128}
	for _, idx := range indices {
		n.InsertPrefix(idx, int(idx)*10)
	}

	// AllIndices: verify yielded values
	m := map[uint8]int{}
	for idx, val := range n.AllIndices() {
		m[idx] = val
	}
	for _, idx := range indices {
		want := int(idx) * 10
		if m[idx] != want {
			t.Errorf("AllIndices[%d]=%d, want %d", idx, m[idx], want)
		}
	}

	// Children
	addrs := []uint8{10, 20, 30}
	for _, addr := range addrs {
		n.InsertChild(addr, &FastNode[int]{})
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

func TestFastNode_CloneFlat_ShallowAndWithCloneFn(t *testing.T) {
	t.Parallel()
	n := &FastNode[int]{}

	n.InsertPrefix(32, 100)
	n.InsertPrefix(64, 200)

	child := &FastNode[int]{}
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

func TestFastNode_CloneRec(t *testing.T) {
	t.Parallel()
	n := &FastNode[int]{}

	n.InsertPrefix(32, 100)
	child := &FastNode[int]{}
	child.InsertPrefix(64, 200)
	n.InsertChild(10, child)

	grand := &FastNode[int]{}
	grand.InsertPrefix(128, 300)
	child.InsertChild(20, grand)

	clone := n.CloneRec(nil)

	// Deep copy: child/grand are new instances
	cloneChild, _ := clone.GetChild(10)
	if cloneChild == child {
		t.Error("CloneRec should deep copy child")
	}
	cloneGrand, _ := cloneChild.(*FastNode[int]).GetChild(20)
	if cloneGrand == grand {
		t.Error("CloneRec should deep copy grandchild")
	}

	// Mutating clone should not affect original
	cloneChild.(*FastNode[int]).InsertPrefix(255, 999)
	if child.PrefixCount() != 1 {
		t.Error("modifying clone affected original child")
	}
}

func TestFastNode_Basics_Insert_Get_Delete(t *testing.T) {
	t.Parallel()
	n := &FastNode[int]{}

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

func TestFastNode_Persist_InsertPersist_DeletePersist_CopyOnWrite(t *testing.T) {
	t.Parallel()
	base := &FastNode[int]{}
	pBase := netip.MustParsePrefix("10.0.0.0/8")
	pNew := netip.MustParsePrefix("10.1.0.0/16")

	base.Insert(pBase, 1, 0)
	alias := base.CloneFlat(nil)

	// InsertPersist must not affect alias
	if exists := base.InsertPersist(func(v int) int { return v }, pNew, 2, 0); exists {
		t.Errorf("InsertPersist(%v) exists=true on first insert, want false", pNew)
	}
	if _, ok := alias.Get(pNew); ok {
		t.Errorf("alias Get(%v)=true, want false (COW)", pNew)
	}

	// DeletePersist must not affect alias
	if exists := base.DeletePersist(nil, pBase); !exists {
		t.Errorf("DeletePersist(%v) exists=false, want true", pBase)
	}
	if _, ok := alias.Get(pBase); !ok {
		t.Errorf("alias lost %v after DeletePersist on base", pBase)
	}
}

func TestFastNode_Modify_Lifecycle(t *testing.T) {
	t.Parallel()
	n := &FastNode[int]{}
	p := netip.MustParsePrefix("192.168.0.0/16")

	// insert
	d := n.Modify(p, func(_ int, found bool) (int, bool) {
		if found {
			t.Fatal("found=true on first Modify insert")
		}
		return 42, false
	})
	if d != 1 {
		t.Errorf("Modify insert delta=%d, want 1", d)
	}

	// update
	d = n.Modify(p, func(v int, found bool) (int, bool) {
		if !found || v != 42 {
			t.Fatalf("expected found=true and v=42, got found=%v v=%d", found, v)
		}
		return 100, false
	})
	if d != 0 {
		t.Errorf("Modify update delta=%d, want 0", d)
	}

	// delete
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

func TestFastNode_EqualRec(t *testing.T) {
	t.Parallel()
	a := &FastNode[int]{}
	b := &FastNode[int]{}

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

func TestFastNode_Stats_Dump_Fprint_DirectItems(t *testing.T) {
	t.Parallel()
	n := &FastNode[int]{}

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
	if sum := s.Pfxs + s.Leaves + s.Fringes; sum != len(pfx4)+len(pfx6) {
		t.Fatalf("StatsRec.Pfxs+s.Leaves+s.Fimges=%d, want %d", sum, len(pfx4)+len(pfx6))
	}

	// DumpRec (ensure contains a known prefix)
	var dump bytes.Buffer
	n.DumpRec(&dump, StridePath{}, 0, true, true)
	if out := dump.String(); !strings.Contains(out, "10.0.0.0/8") {
		t.Errorf("DumpRec output missing 10.0.0.0/8: %s", out)
	}

	// FprintRec
	var tree bytes.Buffer
	start := TrieItem[int]{Node: n, Path: StridePath{}, Idx: 0, Is4: true}
	if err := n.FprintRec(&tree, start, "", true); err != nil {
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

func TestFastNode_AllRec_and_AllRecSorted(t *testing.T) {
	t.Parallel()
	n := &FastNode[int]{}
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

	var got []netip.Prefix
	n.AllRec(StridePath{}, 0, true, func(p netip.Prefix, _ int) bool {
		got = append(got, p)
		return true
	})
	if len(got) != len(pfxs) {
		t.Fatalf("AllRec len=%d, want %d", len(got), len(pfxs))
	}

	var sorted []netip.Prefix
	n.AllRecSorted(StridePath{}, 0, true, func(p netip.Prefix, _ int) bool {
		sorted = append(sorted, p)
		return true
	})
	if !slices.IsSortedFunc(sorted, CmpPrefix) {
		t.Fatalf("AllRecSorted not sorted: %v", sorted)
	}
}

func TestFastNode_Supernets_and_Subnets(t *testing.T) {
	t.Parallel()
	n := &FastNode[int]{}
	n.Insert(mpp("10.0.0.0/8"), 1, 0)
	n.Insert(mpp("10.1.0.0/16"), 2, 0)
	n.Insert(mpp("10.1.1.0/24"), 3, 0)

	var supers []netip.Prefix
	n.Supernets(mpp("10.1.1.0/24"), func(p netip.Prefix, _ int) bool {
		supers = append(supers, p)
		return true
	})
	if len(supers) != 3 {
		t.Fatalf("Supernets count=%d, want 3", len(supers))
	}

	var subs []netip.Prefix
	n.Subnets(mpp("10.0.0.0/8"), func(p netip.Prefix, _ int) bool {
		subs = append(subs, p)
		return true
	})
	if len(subs) != 3 {
		t.Fatalf("Subnets count=%d, want 3", len(subs))
	}
}

func TestFastNode_Overlaps_Basic_and_PrefixAtDepth(t *testing.T) {
	t.Parallel()
	a := &FastNode[int]{}
	b := &FastNode[int]{}

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

func TestFastNode_UnionRec_and_UnionRecPersist(t *testing.T) {
	t.Parallel()
	n1 := &FastNode[int]{}
	n2 := &FastNode[int]{}

	n1.Insert(mpp("10.0.0.0/8"), 1, 0)
	n2.Insert(mpp("10.1.0.0/16"), 2, 0)
	n2.Insert(mpp("172.16.0.0/12"), 3, 0)

	dups := n1.UnionRec(nil, n2, 0)
	if dups != 0 {
		t.Fatalf("UnionRec duplicates=%d, want 0", dups)
	}

	for _, tc := range []struct {
		p string
		v int
	}{
		{"10.0.0.0/8", 1},
		{"10.1.0.0/16", 2},
		{"172.16.0.0/12", 3},
	} {
		v, ok := n1.Get(mpp(tc.p))
		if !ok || v != tc.v {
			t.Fatalf("after union Get(%s)=(%d,%v), want (%d,true)", tc.p, v, ok, tc.v)
		}
	}

	// Persist: base unchanged relative to alias
	base := &FastNode[int]{}
	alias := base.CloneFlat(nil)
	other := &FastNode[int]{}
	other.Insert(mpp("2001:db8::/32"), 9, 0)
	_ = base.UnionRecPersist(nil, other, 0)
	if _, ok := alias.Get(mpp("2001:db8::/32")); ok {
		t.Fatalf("alias changed after UnionRecPersist, want unchanged")
	}
}

func TestFastNode_FprintRec_and_DirectItemsRec_Smoke(t *testing.T) {
	t.Parallel()
	n := &FastNode[int]{}
	n.Insert(mpp("10.0.0.0/8"), 10, 0)
	n.Insert(mpp("10.1.0.0/16"), 11, 0)

	var buf bytes.Buffer
	start := TrieItem[int]{Node: n, Path: StridePath{}, Idx: 0, Is4: true}
	if err := n.FprintRec(&buf, start, "", true); err != nil {
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
