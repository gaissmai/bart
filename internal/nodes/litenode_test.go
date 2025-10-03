package nodes

import (
	"bytes"
	"net/netip"
	"slices"
	"strings"
	"testing"
)

func TestLiteNode_EmptyState(t *testing.T) {
	t.Parallel()
	n := &LiteNode[int]{}

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
	var nilNode *LiteNode[int]
	if !nilNode.IsEmpty() {
		t.Error("nil node should be empty")
	}
}

func TestLiteNode_PrefixCRUD(t *testing.T) {
	t.Parallel()
	n := &LiteNode[int]{}

	// Insert first time
	if exists := n.InsertPrefix(32, 0); exists {
		t.Error("InsertPrefix first time returned exists=true")
	}
	if n.PrefixCount() != 1 {
		t.Errorf("PrefixCount()=%d after insert, want 1", n.PrefixCount())
	}

	// Insert overwrite (LiteNode doesn't store values but reports exists)
	if exists := n.InsertPrefix(32, 0); !exists {
		t.Error("InsertPrefix overwrite returned exists=false")
	}
	if n.PrefixCount() != 1 {
		t.Errorf("PrefixCount()=%d after overwrite, want 1", n.PrefixCount())
	}

	// GetPrefix
	if _, exists := n.GetPrefix(32); !exists {
		t.Error("GetPrefix(32) returned exists=false")
	}
	if _, exists := n.GetPrefix(64); exists {
		t.Error("GetPrefix(64) returned exists=true for non-existent")
	}

	// Delete
	if exists := n.DeletePrefix(32); !exists {
		t.Error("DeletePrefix returned exists=false")
	}
	if n.PrefixCount() != 0 {
		t.Errorf("PrefixCount()=%d after delete, want 0", n.PrefixCount())
	}

	// Delete non-existent
	if exists := n.DeletePrefix(77); exists {
		t.Error("DeletePrefix non-existent returned exists=true")
	}
}

func TestLiteNode_Contains_ART_Coverage(t *testing.T) {
	t.Parallel()
	n := &LiteNode[int]{}

	// Insert at index 32
	n.InsertPrefix(32, 0)

	// Allotment table for 32 in uint8 range: {32, 64, 65, 128, 129, 130, 131}
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
		{2, false},
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

func TestLiteNode_LookupAndLookupIdx(t *testing.T) {
	t.Parallel()
	n := &LiteNode[int]{}

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

func TestLiteNode_ChildrenCRUD(t *testing.T) {
	t.Parallel()
	n := &LiteNode[int]{}

	child := &LiteNode[int]{}
	child.InsertPrefix(1, 0)

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

func TestLiteNode_MustGetChild_Panics(t *testing.T) {
	t.Parallel()
	n := &LiteNode[int]{}

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustGetChild should panic on missing child")
		}
	}()
	n.MustGetChild(42)
}

func TestLiteNode_Iterators(t *testing.T) {
	t.Parallel()
	n := &LiteNode[int]{}

	indices := []uint8{1, 32, 64, 128}
	for _, idx := range indices {
		n.InsertPrefix(idx, 0)
	}

	// AllIndices (LiteNode yields zero values)
	count := 0
	for _, val := range n.AllIndices() {
		count++
		if val != 0 {
			t.Errorf("AllIndices yielded non-zero value: %d", val)
		}
	}
	if count != len(indices) {
		t.Errorf("AllIndices count=%d, want %d", count, len(indices))
	}

	// Children
	addrs := []uint8{10, 20, 30}
	for _, addr := range addrs {
		n.InsertChild(addr, &LiteNode[int]{})
	}

	// AllChildren
	childCount := 0
	for _, child := range n.AllChildren() {
		childCount++
		if child == nil {
			t.Error("AllChildren yielded nil child")
		}
	}
	if childCount != len(addrs) {
		t.Errorf("AllChildren count=%d, want %d", childCount, len(addrs))
	}
}

func TestLiteNode_CloneFlat(t *testing.T) {
	t.Parallel()
	n := &LiteNode[int]{}

	n.InsertPrefix(32, 0)
	n.InsertPrefix(64, 0)

	child := &LiteNode[int]{}
	n.InsertChild(10, child)

	leaf := NewLeafNode[int](netip.Prefix{}, 0)
	n.InsertChild(20, leaf)

	clone := n.CloneFlat(nil)

	// Structure counts copied
	if clone.PrefixCount() != n.PrefixCount() {
		t.Errorf("clone.PrefixCount()=%d, want %d", clone.PrefixCount(), n.PrefixCount())
	}
	if clone.ChildCount() != n.ChildCount() {
		t.Errorf("clone.ChildCount()=%d, want %d", clone.ChildCount(), n.ChildCount())
	}

	// Prefix sets independent
	clone.InsertPrefix(128, 0)
	if n.PrefixCount() == clone.PrefixCount() {
		t.Error("clone modification affected original")
	}

	// Children shallow-copied (including leaf/fringe)
	if c, _ := clone.GetChild(10); c != child {
		t.Error("child should be same instance for CloneFlat(nil)")
	}
	if c, _ := clone.GetChild(20); c != leaf {
		t.Error("leaf should be same instance for CloneFlat(nil)")
	}
}

func TestLiteNode_CloneRec(t *testing.T) {
	t.Parallel()
	n := &LiteNode[int]{}

	n.InsertPrefix(32, 0)

	child := &LiteNode[int]{}
	child.InsertPrefix(64, 0)
	n.InsertChild(10, child)

	grand := &LiteNode[int]{}
	grand.InsertPrefix(128, 0)
	child.InsertChild(20, grand)

	clone := n.CloneRec(nil)

	// Deep copy: child/grand are new instances
	cloneChild, _ := clone.GetChild(10)
	if cloneChild == child {
		t.Error("CloneRec should deep copy child")
	}
	cloneGrand, _ := cloneChild.(*LiteNode[int]).GetChild(20)
	if cloneGrand == grand {
		t.Error("CloneRec should deep copy grandchild")
	}

	// Mutating clone should not affect original
	cloneChild.(*LiteNode[int]).InsertPrefix(255, 0)
	if child.PrefixCount() != 1 {
		t.Error("modifying clone affected original child")
	}
}

func TestLiteNode_Basics_Insert_Get_Delete(t *testing.T) {
	t.Parallel()
	n := &LiteNode[int]{}

	// IPv4
	p1 := mpp("10.0.0.0/8")
	p2 := mpp("10.1.0.0/16")

	if exists := n.Insert(p1, 111, 0); exists {
		t.Errorf("Insert(%v) first time exists=true, want false", p1)
	}
	if exists := n.Insert(p2, 222, 0); exists {
		t.Errorf("Insert(%v) first time exists=true, want false", p2)
	}

	// LiteNode stores no values, only presence
	if _, ok := n.Get(p1); !ok {
		t.Errorf("Get(%v) ok=false, want true", p1)
	}
	if _, ok := n.Get(p2); !ok {
		t.Errorf("Get(%v) ok=false, want true", p2)
	}

	if exists := n.Delete(p1); !exists {
		t.Errorf("Delete(%v) exists=false, want true", p1)
	}
	if _, ok := n.Get(p1); ok {
		t.Errorf("Get(%v) ok=true after delete, want false", p1)
	}
}

func TestLiteNode_Persist_InsertPersist_DeletePersist_CopyOnWrite(t *testing.T) {
	t.Parallel()
	// Build base
	base := &LiteNode[int]{}
	p1 := mpp("10.0.0.0/8")
	p2 := mpp("10.1.0.0/16")
	base.Insert(p1, 0, 0)

	// Create an alias via shallow clone of containers
	alias := base.CloneFlat(nil)

	// Mutate base with COW insert
	if exists := base.InsertPersist(nil, p2, 0, 0); exists {
		t.Errorf("InsertPersist(%v) exists=true on first insert, want false", p2)
	}
	// alias must be unaffected
	if _, ok := alias.Get(p2); ok {
		t.Errorf("alias Get(%v)=true, want false (COW)", p2)
	}

	// Mutate base with COW delete
	if exists := base.DeletePersist(nil, p1); !exists {
		t.Errorf("DeletePersist(%v) exists=false, want true", p1)
	}
	// alias must still have p1
	if _, ok := alias.Get(p1); !ok {
		t.Errorf("alias lost %v after DeletePersist on base", p1)
	}
}

func TestLiteNode_Modify_Lifecycle(t *testing.T) {
	t.Parallel()
	n := &LiteNode[int]{}
	p := mpp("192.168.0.0/16")

	// insert via Modify
	d := n.Modify(p, func(_ int, found bool) (int, bool) {
		if found {
			t.Fatal("found=true on first Modify insert")
		}
		return 0, false
	})
	if d != 1 {
		t.Errorf("Modify insert delta=%d, want 1", d)
	}

	// update via Modify (Lite ignores values)
	d = n.Modify(p, func(_ int, found bool) (int, bool) {
		if !found {
			t.Fatal("found=false on Modify update")
		}
		return 0, false
	})
	if d != 0 {
		t.Errorf("Modify update delta=%d, want 0", d)
	}

	// delete via Modify
	d = n.Modify(p, func(_ int, found bool) (int, bool) {
		if !found {
			t.Fatal("found=false on Modify delete")
		}
		return 0, true
	})
	if d != -1 {
		t.Errorf("Modify delete delta=%d, want -1", d)
	}
}

func TestLiteNode_EqualRec(t *testing.T) {
	t.Parallel()
	a := &LiteNode[int]{}
	b := &LiteNode[int]{}

	p1 := mpp("10.0.0.0/8")
	p2 := mpp("2001:db8::/32")

	a.Insert(p1, 0, 0)
	a.Insert(p2, 0, 0)
	b.Insert(p1, 0, 0)
	b.Insert(p2, 0, 0)

	if !a.EqualRec(b) {
		t.Fatal("EqualRec: identical tries reported as not equal")
	}

	// diverge
	a.Delete(p2)
	if a.EqualRec(b) {
		t.Fatal("EqualRec: different tries reported as equal")
	}
}

func TestLiteNode_Stats_Dump_Fprint_DirectItems(t *testing.T) {
	t.Parallel()
	n := &LiteNode[int]{}

	pfx := []netip.Prefix{
		mpp("10.0.0.0/8"),
		mpp("10.1.0.0/16"),
		mpp("2001:db8::/32"),
	}
	for _, p := range pfx {
		n.Insert(p, 0, 0)
	}

	// Stats/StatsRec
	sr := n.StatsRec()
	if sum := sr.Pfxs + sr.Leaves + sr.Fringes; sum != len(pfx) {
		t.Fatalf("StatsRec Pfxs*Leaves+Fringes=%d, want %d", sum, len(pfx))
	}

	// DumpRec (non-brittle: just ensure prefixes appear)
	var dump bytes.Buffer
	n.DumpRec(&dump, StridePath{}, 0, true, false)
	out := dump.String()
	if !strings.Contains(out, "10.0.0.0/8") {
		t.Errorf("DumpRec missing 10.0.0.0/8: %s", out)
	}

	// FprintRec (hierarchical)
	var tree bytes.Buffer
	start := TrieItem[int]{Node: n, Path: StridePath{}, Idx: 0, Is4: true}
	if err := n.FprintRec(&tree, start, "", false); err != nil {
		t.Fatalf("FprintRec error: %v", err)
	}
	treeOut := tree.String()
	if !strings.Contains(treeOut, "10.1.0.0/16") {
		t.Errorf("Fprint output missing 10.1.0.0/16: %s", treeOut)
	}

	// DirectItemsRec (basic sanity)
	items := n.DirectItemsRec(0, StridePath{}, 0, true)
	if len(items) == 0 {
		t.Errorf("DirectItemsRec returned no items")
	}
}

func TestLiteNode_AllRec_and_AllRecSorted(t *testing.T) {
	t.Parallel()
	n := &LiteNode[int]{}
	pfxs := []netip.Prefix{
		mpp("10.1.0.0/16"),
		mpp("10.0.0.0/8"),
		mpp("192.168.0.0/16"),
	}
	for _, p := range pfxs {
		n.Insert(p, 0, 0)
	}

	// AllRec: collect without order guarantee
	var got []netip.Prefix
	n.AllRec(StridePath{}, 0, true, func(p netip.Prefix, _ int) bool {
		got = append(got, p)
		return true
	})
	if len(got) != len(pfxs) {
		t.Fatalf("AllRec len=%d, want %d", len(got), len(pfxs))
	}

	// AllRecSorted: verify order is sorted by CmpPrefix
	var sorted []netip.Prefix
	n.AllRecSorted(StridePath{}, 0, true, func(p netip.Prefix, _ int) bool {
		sorted = append(sorted, p)
		return true
	})
	if !slices.IsSortedFunc(sorted, CmpPrefix) {
		t.Fatalf("AllRecSorted not sorted: %v", sorted)
	}
}

func TestLiteNode_Supernets_and_Subnets(t *testing.T) {
	t.Parallel()
	n := &LiteNode[int]{}
	// tree: 10/8 -> 10.1/16 -> 10.1.1/24
	n.Insert(mpp("10.0.0.0/8"), 0, 0)
	n.Insert(mpp("10.1.0.0/16"), 0, 0)
	n.Insert(mpp("10.1.1.0/24"), 0, 0)
	n.Insert(mpp("192.168.0.0/16"), 0, 0)

	// Supernets(10.1.1.0/24)
	var supers []netip.Prefix
	n.Supernets(mpp("10.1.1.0/24"), func(p netip.Prefix, _ int) bool {
		supers = append(supers, p)
		return true
	})
	if len(supers) != 3 {
		t.Errorf("Supernets count=%d, want 3", len(supers))
	}

	// Subnets(10/8)
	var subs []netip.Prefix
	n.Subnets(mpp("10.0.0.0/8"), func(p netip.Prefix, _ int) bool {
		subs = append(subs, p)
		return true
	})
	if len(subs) != 3 {
		t.Errorf("Subnets count=%d, want 3", len(subs))
	}
}

func TestLiteNode_Overlaps_Basic_and_PrefixAtDepth(t *testing.T) {
	t.Parallel()
	a := &LiteNode[int]{}
	b := &LiteNode[int]{}

	a.Insert(mpp("10.0.0.0/8"), 0, 0)
	a.Insert(mpp("192.168.0.0/16"), 0, 0)

	// non-overlap
	b.Insert(mpp("172.16.0.0/12"), 0, 0)
	if a.Overlaps(b, 0) {
		t.Fatal("expected no overlap")
	}

	// add overlapping
	b.Insert(mpp("10.1.0.0/16"), 0, 0)
	if !a.Overlaps(b, 0) {
		t.Fatal("expected overlap")
	}

	// OverlapsPrefixAtDepth
	if !a.OverlapsPrefixAtDepth(mpp("10.1.1.0/24"), 0) {
		t.Fatal("OverlapsPrefixAtDepth should be true")
	}
	if a.OverlapsPrefixAtDepth(mpp("11.0.0.0/8"), 0) {
		t.Fatal("OverlapsPrefixAtDepth should be false")
	}
}

func TestLiteNode_UnionRec_and_UnionRecPersist(t *testing.T) {
	t.Parallel()
	n1 := &LiteNode[int]{}
	n2 := &LiteNode[int]{}

	n1.Insert(mpp("10.0.0.0/8"), 0, 0)
	n2.Insert(mpp("10.1.0.0/16"), 0, 0)
	n2.Insert(mpp("172.16.0.0/12"), 0, 0)

	dups := n1.UnionRec(nil, n2, 0)
	if dups != 0 {
		t.Fatalf("UnionRec duplicates=%d, want 0", dups)
	}
	// now all should be present in n1
	for _, p := range []string{"10.0.0.0/8", "10.1.0.0/16", "172.16.0.0/12"} {
		if _, ok := n1.Get(mpp(p)); !ok {
			t.Fatalf("after UnionRec missing %s", p)
		}
	}

	// UnionRecPersist: ensure COW (n1 remains unchanged)
	base := &LiteNode[int]{}
	alias := base.CloneFlat(nil)
	other := &LiteNode[int]{}
	other.Insert(mpp("2001:db8::/32"), 0, 0)
	_ = base.UnionRecPersist(nil, other, 0)
	if _, ok := alias.Get(mpp("2001:db8::/32")); ok {
		t.Fatalf("alias changed after UnionRecPersist, want unchanged")
	}
}

func TestLiteNode_FprintRec_and_DirectItemsRec_Smoke(t *testing.T) {
	t.Parallel()
	n := &LiteNode[int]{}
	n.Insert(mpp("10.0.0.0/8"), 0, 0)
	n.Insert(mpp("10.1.0.0/16"), 0, 0)

	var buf bytes.Buffer
	start := TrieItem[int]{Node: n, Path: StridePath{}, Idx: 0, Is4: true}
	if err := n.FprintRec(&buf, start, "", false); err != nil {
		t.Fatalf("FprintRec error: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("FprintRec empty output")
	}

	items := n.DirectItemsRec(0, StridePath{}, 0, true)
	if len(items) == 0 {
		t.Fatal("DirectItemsRec should return items")
	}
}
