package nodes

import (
	"bytes"
	"math/rand/v2"
	"net/netip"
	"slices"
	"strings"
	"testing"

	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/tests/golden"
	"github.com/gaissmai/bart/internal/tests/random"
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
	if sum := sr.Prefixes + sr.Leaves + sr.Fringes; sum != len(pfx) {
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

func TestLiteNode_OverlapsRoutes_DirectIntersection(t *testing.T) {
	t.Parallel()
	a := &LiteNode[int]{}
	b := &LiteNode[int]{}

	a.InsertPrefix(32, 0)
	b.InsertPrefix(32, 0)

	if !a.OverlapsRoutes(b) {
		t.Error("OverlapsRoutes should return true for identical indices")
	}
}

func TestLiteNode_OverlapsRoutes_LPM_Containment(t *testing.T) {
	t.Parallel()
	a := &LiteNode[int]{}
	b := &LiteNode[int]{}

	a.InsertPrefix(64, 0)
	b.InsertPrefix(32, 0)

	if !a.OverlapsRoutes(b) {
		t.Error("OverlapsRoutes should detect LPM containment")
	}

	c := &LiteNode[int]{}
	d := &LiteNode[int]{}
	c.InsertPrefix(32, 0)
	d.InsertPrefix(64, 0)

	if !c.OverlapsRoutes(d) {
		t.Error("OverlapsRoutes should detect LPM containment (reverse)")
	}
}

func TestLiteNode_OverlapsRoutes_NoOverlap(t *testing.T) {
	t.Parallel()
	a := &LiteNode[int]{}
	b := &LiteNode[int]{}

	a.InsertPrefix(2, 0)
	b.InsertPrefix(3, 0)

	if a.OverlapsRoutes(b) {
		t.Error("OverlapsRoutes should return false for non-overlapping prefixes")
	}

	c := &LiteNode[int]{}
	d := &LiteNode[int]{}
	c.InsertPrefix(4, 0)
	d.InsertPrefix(6, 0)

	if c.OverlapsRoutes(d) {
		t.Error("OverlapsRoutes should return false for non-overlapping sibling subtrees")
	}
}

func TestLiteNode_OverlapsRoutes_EmptyNodes(t *testing.T) {
	t.Parallel()
	a := &LiteNode[int]{}
	b := &LiteNode[int]{}

	if a.OverlapsRoutes(b) {
		t.Error("OverlapsRoutes should return false for empty nodes")
	}

	a.InsertPrefix(32, 0)
	if a.OverlapsRoutes(b) {
		t.Error("OverlapsRoutes should return false when one node is empty")
	}
}

func TestLiteNode_OverlapsRoutes_MultiplePrefix_WithOverlap(t *testing.T) {
	t.Parallel()
	a := &LiteNode[int]{}
	b := &LiteNode[int]{}

	a.InsertPrefix(16, 0)
	a.InsertPrefix(64, 0)
	a.InsertPrefix(128, 0)

	b.InsertPrefix(8, 0)
	b.InsertPrefix(32, 0)
	b.InsertPrefix(255, 0)

	if !a.OverlapsRoutes(b) {
		t.Error("OverlapsRoutes should detect overlap in multi-prefix scenario")
	}
}

func TestLiteNode_OverlapsRoutes_Uint8_Boundary(t *testing.T) {
	t.Parallel()
	a := &LiteNode[int]{}
	b := &LiteNode[int]{}

	a.InsertPrefix(255, 0)
	b.InsertPrefix(254, 0)

	if a.OverlapsRoutes(b) {
		t.Error("OverlapsRoutes returned unexpected overlap for sibling indices at boundary")
	}

	b.InsertPrefix(255, 0)
	if !a.OverlapsRoutes(b) {
		t.Error("OverlapsRoutes should detect overlap at index 255")
	}
}

func TestLiteNode_OverlapsChildrenIn_BitsetPath(t *testing.T) {
	t.Parallel()
	n := &LiteNode[int]{}
	o := &LiteNode[int]{}

	n.InsertPrefix(1, 0)

	for i := range uint8(20) {
		child := &LiteNode[int]{}
		child.InsertPrefix(1, 0)
		o.InsertChild(i, child)
	}

	if !n.OverlapsChildrenIn(o) {
		t.Error("OverlapsChildrenIn should detect overlap using bitset path")
	}
}

func TestLiteNode_OverlapsChildrenIn_IterationPath(t *testing.T) {
	t.Parallel()
	n := &LiteNode[int]{}
	o := &LiteNode[int]{}

	n.InsertPrefix(2, 0)

	child := &LiteNode[int]{}
	child.InsertPrefix(1, 0)
	o.InsertChild(128, child)

	if n.OverlapsChildrenIn(o) {
		t.Error("OverlapsChildrenIn should not detect overlap for non-overlapping children")
	}

	child2 := &LiteNode[int]{}
	child2.InsertPrefix(1, 0)
	o.InsertChild(0, child2)

	if !n.OverlapsChildrenIn(o) {
		t.Error("OverlapsChildrenIn should detect overlap using iteration path")
	}
}

func TestLiteNode_OverlapsChildrenIn_EmptyCases(t *testing.T) {
	t.Parallel()
	n := &LiteNode[int]{}
	o := &LiteNode[int]{}

	if n.OverlapsChildrenIn(o) {
		t.Error("OverlapsChildrenIn should return false for empty nodes")
	}

	n.InsertPrefix(32, 0)
	if n.OverlapsChildrenIn(o) {
		t.Error("OverlapsChildrenIn should return false when o has no children")
	}

	n2 := &LiteNode[int]{}
	child := &LiteNode[int]{}
	o.InsertChild(10, child)
	if n2.OverlapsChildrenIn(o) {
		t.Error("OverlapsChildrenIn should return false when n has no prefixes")
	}
}

func TestLiteNode_OverlapsTwoChildren_AllCombinations(t *testing.T) {
	t.Parallel()

	t.Run("node-node_overlap", func(t *testing.T) {
		t.Parallel()
		n1 := &LiteNode[int]{}
		n2 := &LiteNode[int]{}
		n1.InsertPrefix(32, 0)
		n2.InsertPrefix(32, 0)

		parent := &LiteNode[int]{}
		if !parent.OverlapsTwoChildren(n1, n2, 0) {
			t.Error("node-node should overlap when prefixes overlap")
		}
	})

	t.Run("node-node_no_overlap", func(t *testing.T) {
		t.Parallel()
		n1 := &LiteNode[int]{}
		n2 := &LiteNode[int]{}
		n1.InsertPrefix(2, 0)
		n2.InsertPrefix(3, 0)

		parent := &LiteNode[int]{}
		if parent.OverlapsTwoChildren(n1, n2, 0) {
			t.Error("node-node should not overlap when prefixes don't overlap")
		}
	})

	t.Run("node-leaf", func(t *testing.T) {
		t.Parallel()
		node := &LiteNode[int]{}
		leaf := &LeafNode[int]{
			Prefix: mpp("10.0.0.0/8"),
			Value:  0,
		}

		node.Insert(mpp("10.0.0.0/16"), 0, 0)

		parent := &LiteNode[int]{}
		if !parent.OverlapsTwoChildren(node, leaf, 0) {
			t.Error("node-leaf should overlap when node contains overlapping prefix")
		}
	})

	t.Run("node-fringe_always_overlap", func(t *testing.T) {
		t.Parallel()
		node := &LiteNode[int]{}
		fringe := &FringeNode[int]{
			Value: 0,
		}

		parent := &LiteNode[int]{}
		if !parent.OverlapsTwoChildren(node, fringe, 0) {
			t.Error("node-fringe should always overlap")
		}
	})

	t.Run("leaf-leaf_overlap", func(t *testing.T) {
		t.Parallel()
		leaf1 := &LeafNode[int]{
			Prefix: mpp("10.0.0.0/8"),
			Value:  0,
		}
		leaf2 := &LeafNode[int]{
			Prefix: mpp("10.0.0.0/16"),
			Value:  0,
		}

		parent := &LiteNode[int]{}
		if !parent.OverlapsTwoChildren(leaf1, leaf2, 0) {
			t.Error("leaf-leaf should overlap when prefixes overlap")
		}
	})

	t.Run("leaf-leaf_no_overlap", func(t *testing.T) {
		t.Parallel()
		leaf1 := &LeafNode[int]{
			Prefix: mpp("10.0.0.0/8"),
			Value:  0,
		}
		leaf2 := &LeafNode[int]{
			Prefix: mpp("192.168.0.0/16"),
			Value:  0,
		}

		parent := &LiteNode[int]{}
		if parent.OverlapsTwoChildren(leaf1, leaf2, 0) {
			t.Error("leaf-leaf should not overlap when prefixes don't overlap")
		}
	})

	t.Run("leaf-fringe_always_overlap", func(t *testing.T) {
		t.Parallel()
		leaf := &LeafNode[int]{
			Prefix: mpp("10.0.0.0/8"),
			Value:  0,
		}
		fringe := &FringeNode[int]{
			Value: 0,
		}

		parent := &LiteNode[int]{}
		if !parent.OverlapsTwoChildren(leaf, fringe, 0) {
			t.Error("leaf-fringe should always overlap")
		}
	})

	t.Run("fringe-fringe_always_overlap", func(t *testing.T) {
		t.Parallel()
		fringe1 := &FringeNode[int]{
			Value: 0,
		}
		fringe2 := &FringeNode[int]{
			Value: 0,
		}

		parent := &LiteNode[int]{}
		if !parent.OverlapsTwoChildren(fringe1, fringe2, 0) {
			t.Error("fringe-fringe should always overlap")
		}
	})
}

func TestLiteNode_Overlaps_CompleteFlow(t *testing.T) {
	t.Parallel()

	t.Run("routes_overlap", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		a.Insert(mpp("10.0.0.0/8"), 0, 0)
		b.Insert(mpp("10.0.0.0/16"), 0, 0)

		if !a.Overlaps(b, 0) {
			t.Error("Overlaps should detect route overlap")
		}
	})

	t.Run("children_overlap", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		a.Insert(mpp("10.0.0.0/8"), 0, 0)
		b.Insert(mpp("10.1.0.0/16"), 0, 0)

		if !a.Overlaps(b, 0) {
			t.Error("Overlaps should detect child overlap")
		}
	})

	t.Run("same_children_overlap", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		a.Insert(mpp("10.1.0.0/16"), 0, 0)
		b.Insert(mpp("10.1.0.0/24"), 0, 0)

		if !a.Overlaps(b, 0) {
			t.Error("Overlaps should detect same-children overlap")
		}
	})

	t.Run("no_overlap", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		a.Insert(mpp("10.0.0.0/8"), 0, 0)
		b.Insert(mpp("192.168.0.0/16"), 0, 0)

		if a.Overlaps(b, 0) {
			t.Error("Overlaps should return false for non-overlapping trees")
		}
	})

	t.Run("empty_nodes", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		if a.Overlaps(b, 0) {
			t.Error("Overlaps should return false for empty nodes")
		}
	})
}

func TestLiteNode_OverlapsIdx(t *testing.T) {
	t.Parallel()

	t.Run("prefix_covers_idx", func(t *testing.T) {
		t.Parallel()
		n := &LiteNode[int]{}
		n.InsertPrefix(1, 0)

		if !n.OverlapsIdx(128) {
			t.Error("OverlapsIdx should return true when prefix covers idx")
		}
	})

	t.Run("idx_covers_routes", func(t *testing.T) {
		t.Parallel()
		n := &LiteNode[int]{}
		n.InsertPrefix(64, 0)

		if !n.OverlapsIdx(32) {
			t.Error("OverlapsIdx should return true when idx covers routes")
		}
	})

	t.Run("idx_overlaps_child", func(t *testing.T) {
		t.Parallel()
		n := &LiteNode[int]{}
		child := &LiteNode[int]{}
		child.InsertPrefix(1, 0)

		n.InsertChild(10, child)

		idx := art.OctetToIdx(10)
		for idx > 1 {
			idx = idx / 2
			if n.OverlapsIdx(idx) {
				break
			}
		}
	})

	t.Run("no_overlap", func(t *testing.T) {
		t.Parallel()
		n := &LiteNode[int]{}
		n.InsertPrefix(2, 0)

		if n.OverlapsIdx(3) {
			t.Error("OverlapsIdx should return false for non-overlapping idx")
		}
	})
}

//nolint:gocyclo
func TestLiteNode_UnionRec_AllCombinations(t *testing.T) {
	t.Parallel()

	cloneFn := cloneFnFactory[int]()

	t.Run("null_plus_node", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		childNode := &LiteNode[int]{}
		childNode.InsertPrefix(32, 0)
		b.InsertChild(10, childNode)

		duplicates := a.UnionRec(cloneFn, b, 0)
		if duplicates != 0 {
			t.Errorf("Expected 0 duplicates, got %d", duplicates)
		}

		child, exists := a.GetChild(10)
		if !exists {
			t.Error("Child should exist after union")
		}
		if childNode, ok := child.(*LiteNode[int]); !ok {
			t.Error("Child should be a LiteNode")
		} else if _, ok := childNode.GetPrefix(32); !ok {
			t.Error("Child node should have prefix 32")
		}
	})

	t.Run("null_plus_leaf", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		leaf := &LeafNode[int]{
			Prefix: mpp("10.10.0.0/16"),
			Value:  0,
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
		if _, ok := child.(*LeafNode[int]); !ok {
			t.Error("Child should be a LeafNode")
		}
	})

	t.Run("null_plus_fringe", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		fringe := &FringeNode[int]{Value: 0}
		b.InsertChild(10, fringe)

		duplicates := a.UnionRec(cloneFn, b, 0)
		if duplicates != 0 {
			t.Errorf("Expected 0 duplicates, got %d", duplicates)
		}

		child, exists := a.GetChild(10)
		if !exists {
			t.Error("Fringe should exist after union")
		}
		if _, ok := child.(*FringeNode[int]); !ok {
			t.Error("Child should be a FringeNode")
		}
	})

	t.Run("node_plus_node", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		childNodeA := &LiteNode[int]{}
		childNodeA.InsertPrefix(32, 0)
		a.InsertChild(10, childNodeA)

		childNodeB := &LiteNode[int]{}
		childNodeB.InsertPrefix(64, 0)
		b.InsertChild(10, childNodeB)

		duplicates := a.UnionRec(cloneFn, b, 0)
		if duplicates != 0 {
			t.Errorf("Expected 0 duplicates, got %d", duplicates)
		}

		child, _ := a.GetChild(10)
		mergedNode := child.(*LiteNode[int])
		if _, ok := mergedNode.GetPrefix(32); !ok {
			t.Error("Should have prefix 32")
		}
		if _, ok := mergedNode.GetPrefix(64); !ok {
			t.Error("Should have prefix 64")
		}
	})

	t.Run("node_plus_node_with_duplicate", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		childNodeA := &LiteNode[int]{}
		childNodeA.InsertPrefix(32, 0)
		a.InsertChild(10, childNodeA)

		childNodeB := &LiteNode[int]{}
		childNodeB.InsertPrefix(32, 0)
		b.InsertChild(10, childNodeB)

		duplicates := a.UnionRec(cloneFn, b, 0)
		if duplicates != 1 {
			t.Errorf("Expected 1 duplicate, got %d", duplicates)
		}

		child, _ := a.GetChild(10)
		mergedNode := child.(*LiteNode[int])
		if _, ok := mergedNode.GetPrefix(32); !ok {
			t.Error("Should have prefix 32")
		}
	})

	t.Run("node_plus_leaf", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		childNodeA := &LiteNode[int]{}
		childNodeA.InsertPrefix(32, 0)
		a.InsertChild(10, childNodeA)

		leaf := &LeafNode[int]{
			Prefix: mpp("10.10.1.0/24"),
			Value:  0,
		}
		b.InsertChild(10, leaf)

		duplicates := a.UnionRec(cloneFn, b, 0)
		if duplicates != 0 {
			t.Errorf("Expected 0 duplicates, got %d", duplicates)
		}

		child, _ := a.GetChild(10)
		mergedNode := child.(*LiteNode[int])
		if mergedNode.PrefixCount() == 0 && mergedNode.ChildCount() == 0 {
			t.Error("Node should not be empty after inserting leaf")
		}
	})

	t.Run("node_plus_fringe", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		childNodeA := &LiteNode[int]{}
		childNodeA.InsertPrefix(32, 0)
		a.InsertChild(10, childNodeA)

		fringe := &FringeNode[int]{Value: 0}
		b.InsertChild(10, fringe)

		a.UnionRec(cloneFn, b, 0)

		child, _ := a.GetChild(10)
		mergedNode := child.(*LiteNode[int])
		if _, ok := mergedNode.GetPrefix(1); !ok {
			t.Error("Node should have fringe prefix (idx=1)")
		}
	})

	t.Run("leaf_plus_node", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		leafA := &LeafNode[int]{
			Prefix: mpp("10.10.0.0/16"),
			Value:  0,
		}
		a.InsertChild(10, leafA)

		childNodeB := &LiteNode[int]{}
		childNodeB.InsertPrefix(32, 0)
		b.InsertChild(10, childNodeB)

		duplicates := a.UnionRec(cloneFn, b, 0)
		if duplicates != 0 {
			t.Errorf("Expected 0 duplicates, got %d", duplicates)
		}

		child, exists := a.GetChild(10)
		if !exists {
			t.Fatal("Child should exist")
		}
		newNode, ok := child.(*LiteNode[int])
		if !ok {
			t.Fatal("Child should be a LiteNode after union")
		}
		if newNode.PrefixCount() == 0 && newNode.ChildCount() == 0 {
			t.Error("New node should not be empty")
		}
	})

	t.Run("leaf_plus_leaf_same_prefix", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		prefix := mpp("10.10.0.0/16")
		leafA := &LeafNode[int]{Prefix: prefix, Value: 0}
		leafB := &LeafNode[int]{Prefix: prefix, Value: 0}

		a.InsertChild(10, leafA)
		b.InsertChild(10, leafB)

		duplicates := a.UnionRec(cloneFn, b, 0)
		if duplicates != 1 {
			t.Errorf("Expected 1 duplicate, got %d", duplicates)
		}

		child, _ := a.GetChild(10)
		if _, ok := child.(*LeafNode[int]); !ok {
			t.Error("Child should remain a LeafNode")
		}
	})

	t.Run("leaf_plus_leaf_different_prefix", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		leafA := &LeafNode[int]{
			Prefix: mpp("10.10.0.0/16"),
			Value:  0,
		}
		leafB := &LeafNode[int]{
			Prefix: mpp("10.10.1.0/24"),
			Value:  0,
		}

		a.InsertChild(10, leafA)
		b.InsertChild(10, leafB)

		duplicates := a.UnionRec(cloneFn, b, 0)
		if duplicates != 0 {
			t.Errorf("Expected 0 duplicates, got %d", duplicates)
		}

		child, _ := a.GetChild(10)
		if _, ok := child.(*LiteNode[int]); !ok {
			t.Error("Should create new LiteNode when merging different leaves")
		}
	})

	t.Run("leaf_plus_fringe", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		leafA := &LeafNode[int]{
			Prefix: mpp("10.10.0.0/16"),
			Value:  0,
		}
		fringeB := &FringeNode[int]{Value: 0}

		a.InsertChild(10, leafA)
		b.InsertChild(10, fringeB)

		a.UnionRec(cloneFn, b, 0)

		child, _ := a.GetChild(10)
		if _, ok := child.(*LiteNode[int]); !ok {
			t.Error("Should create new LiteNode when merging leaf + fringe")
		}
	})

	t.Run("fringe_plus_node", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		fringeA := &FringeNode[int]{Value: 0}
		childNodeB := &LiteNode[int]{}
		childNodeB.InsertPrefix(32, 0)

		a.InsertChild(10, fringeA)
		b.InsertChild(10, childNodeB)

		a.UnionRec(cloneFn, b, 0)

		child, _ := a.GetChild(10)
		newNode, ok := child.(*LiteNode[int])
		if !ok {
			t.Fatal("Should create new LiteNode when merging fringe + node")
		}
		if newNode.PrefixCount() == 0 {
			t.Error("New node should have prefixes")
		}
	})

	t.Run("fringe_plus_leaf", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		fringeA := &FringeNode[int]{Value: 0}
		leafB := &LeafNode[int]{
			Prefix: mpp("10.10.0.0/16"),
			Value:  0,
		}

		a.InsertChild(10, fringeA)
		b.InsertChild(10, leafB)

		a.UnionRec(cloneFn, b, 0)

		child, _ := a.GetChild(10)
		if _, ok := child.(*LiteNode[int]); !ok {
			t.Error("Should create new LiteNode when merging fringe + leaf")
		}
	})

	t.Run("fringe_plus_fringe", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		fringeA := &FringeNode[int]{Value: 0}
		fringeB := &FringeNode[int]{Value: 0}

		a.InsertChild(10, fringeA)
		b.InsertChild(10, fringeB)

		duplicates := a.UnionRec(cloneFn, b, 0)
		if duplicates != 1 {
			t.Errorf("Expected 1 duplicate, got %d", duplicates)
		}

		child, _ := a.GetChild(10)
		if _, ok := child.(*FringeNode[int]); !ok {
			t.Error("Child should remain a FringeNode")
		}
	})
}

func TestLiteNode_UnionRecPersist_AllCombinations(t *testing.T) {
	t.Parallel()

	cloneFn := cloneFnFactory[int]()

	t.Run("null_plus_node_persist", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		childNodeB := &LiteNode[int]{}
		childNodeB.InsertPrefix(32, 0)
		b.InsertChild(10, childNodeB)

		duplicates := a.UnionRecPersist(cloneFn, b, 0)
		if duplicates != 0 {
			t.Errorf("Expected 0 duplicates, got %d", duplicates)
		}

		child, exists := a.GetChild(10)
		if !exists {
			t.Fatal("Child should exist")
		}
		childNode := child.(*LiteNode[int])

		// Modify original, check clone unchanged
		childNodeB.InsertPrefix(64, 0)
		if _, ok := childNode.GetPrefix(64); ok {
			t.Error("Clone should not reflect changes to original")
		}
	})

	t.Run("null_plus_leaf_persist", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		originalLeaf := &LeafNode[int]{
			Prefix: mpp("10.10.0.0/16"),
			Value:  0,
		}
		b.InsertChild(10, originalLeaf)

		duplicates := a.UnionRecPersist(cloneFn, b, 0)
		if duplicates != 0 {
			t.Errorf("Expected 0 duplicates, got %d", duplicates)
		}

		child, _ := a.GetChild(10)
		clonedLeaf := child.(*LeafNode[int])

		// Modify original prefix (create new one)
		originalLeaf.Prefix = mpp("10.10.1.0/24")
		if clonedLeaf.Prefix.String() == "10.10.1.0/24" {
			t.Error("Clone should not reflect changes to original")
		}
	})

	t.Run("null_plus_fringe_persist", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		fringe := &FringeNode[int]{Value: 0}
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
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		childNodeA := &LiteNode[int]{}
		childNodeA.InsertPrefix(32, 0)
		originalChildA := childNodeA
		a.InsertChild(10, childNodeA)

		childNodeB := &LiteNode[int]{}
		childNodeB.InsertPrefix(64, 0)
		b.InsertChild(10, childNodeB)

		duplicates := a.UnionRecPersist(cloneFn, b, 0)
		if duplicates != 0 {
			t.Errorf("Expected 0 duplicates, got %d", duplicates)
		}

		child, _ := a.GetChild(10)
		mergedNode := child.(*LiteNode[int])

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
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		childNodeA := &LiteNode[int]{}
		childNodeA.InsertPrefix(32, 0)
		originalChildA := childNodeA
		a.InsertChild(10, childNodeA)

		leaf := &LeafNode[int]{
			Prefix: mpp("10.10.1.0/24"),
			Value:  0,
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
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		childNodeA := &LiteNode[int]{}
		childNodeA.InsertPrefix(32, 0)
		originalChildA := childNodeA
		a.InsertChild(10, childNodeA)

		fringe := &FringeNode[int]{Value: 0}
		b.InsertChild(10, fringe)

		a.UnionRecPersist(cloneFn, b, 0)

		child, _ := a.GetChild(10)
		if child == originalChildA {
			t.Error("Child should be cloned")
		}
	})

	t.Run("leaf_plus_node_persist", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		originalLeaf := &LeafNode[int]{
			Prefix: mpp("10.10.0.0/16"),
			Value:  0,
		}
		a.InsertChild(10, originalLeaf)

		childNodeB := &LiteNode[int]{}
		childNodeB.InsertPrefix(32, 0)
		b.InsertChild(10, childNodeB)

		a.UnionRecPersist(cloneFn, b, 0)

		child, _ := a.GetChild(10)
		if _, ok := child.(*LiteNode[int]); !ok {
			t.Error("Should create new LiteNode")
		}
	})

	t.Run("leaf_plus_leaf_same_prefix_persist", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		prefix := mpp("10.10.0.0/16")
		leafA := &LeafNode[int]{Prefix: prefix, Value: 0}
		leafB := &LeafNode[int]{Prefix: prefix, Value: 0}

		a.InsertChild(10, leafA)
		b.InsertChild(10, leafB)

		duplicates := a.UnionRecPersist(cloneFn, b, 0)
		if duplicates != 1 {
			t.Errorf("Expected 1 duplicate, got %d", duplicates)
		}

		child, _ := a.GetChild(10)
		if _, ok := child.(*LeafNode[int]); !ok {
			t.Error("Child should remain a LeafNode")
		}
	})

	t.Run("leaf_plus_leaf_different_prefix_persist", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		leafA := &LeafNode[int]{
			Prefix: mpp("10.10.0.0/16"),
			Value:  0,
		}
		leafB := &LeafNode[int]{
			Prefix: mpp("10.10.1.0/24"),
			Value:  0,
		}

		a.InsertChild(10, leafA)
		b.InsertChild(10, leafB)

		a.UnionRecPersist(cloneFn, b, 0)

		child, _ := a.GetChild(10)
		if _, ok := child.(*LiteNode[int]); !ok {
			t.Error("Should create new LiteNode")
		}
	})

	t.Run("leaf_plus_fringe_persist", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		leafA := &LeafNode[int]{
			Prefix: mpp("10.10.0.0/16"),
			Value:  0,
		}
		fringeB := &FringeNode[int]{Value: 0}

		a.InsertChild(10, leafA)
		b.InsertChild(10, fringeB)

		a.UnionRecPersist(cloneFn, b, 0)

		child, _ := a.GetChild(10)
		if _, ok := child.(*LiteNode[int]); !ok {
			t.Error("Should create new LiteNode")
		}
	})

	t.Run("fringe_plus_node_persist", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		fringeA := &FringeNode[int]{Value: 0}
		a.InsertChild(10, fringeA)

		childNodeB := &LiteNode[int]{}
		childNodeB.InsertPrefix(32, 0)
		b.InsertChild(10, childNodeB)

		a.UnionRecPersist(cloneFn, b, 0)

		child, _ := a.GetChild(10)
		if _, ok := child.(*LiteNode[int]); !ok {
			t.Error("Should create new LiteNode")
		}
	})

	t.Run("fringe_plus_leaf_persist", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		fringeA := &FringeNode[int]{Value: 0}
		a.InsertChild(10, fringeA)

		leafB := &LeafNode[int]{
			Prefix: mpp("10.10.0.0/16"),
			Value:  0,
		}
		b.InsertChild(10, leafB)

		a.UnionRecPersist(cloneFn, b, 0)

		child, _ := a.GetChild(10)
		if _, ok := child.(*LiteNode[int]); !ok {
			t.Error("Should create new LiteNode")
		}
	})

	t.Run("fringe_plus_fringe_persist", func(t *testing.T) {
		t.Parallel()
		a := &LiteNode[int]{}
		b := &LiteNode[int]{}

		fringeA := &FringeNode[int]{Value: 0}
		fringeB := &FringeNode[int]{Value: 0}

		a.InsertChild(10, fringeA)
		b.InsertChild(10, fringeB)

		duplicates := a.UnionRecPersist(cloneFn, b, 0)
		if duplicates != 1 {
			t.Errorf("Expected 1 duplicate, got %d", duplicates)
		}

		child, _ := a.GetChild(10)
		if _, ok := child.(*FringeNode[int]); !ok {
			t.Error("Child should remain a FringeNode")
		}
	})
}

//nolint:gocyclo
func TestLiteNode_Modify_AllPaths(t *testing.T) {
	t.Parallel()

	t.Run("modify_at_lastOctet_delete_nonexistent", func(t *testing.T) {
		t.Parallel()
		n := &LiteNode[int]{}

		delta := n.Modify(mpp("0.0.0.0/0"), func(_ int, found bool) (int, bool) {
			if found {
				t.Error("Should not find non-existent prefix")
			}
			return 0, true
		})
		if delta != 0 {
			t.Errorf("Expected delta 0 for no-op delete, got %d", delta)
		}
	})

	t.Run("modify_at_lastOctet_delete_existing", func(t *testing.T) {
		t.Parallel()
		n := &LiteNode[int]{}
		pfx := mpp("0.0.0.0/0")
		n.Insert(pfx, 0, 0)

		delta := n.Modify(pfx, func(_ int, found bool) (int, bool) {
			if !found {
				t.Errorf("Expected found=true")
			}
			return 0, true
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
		n := &LiteNode[int]{}
		pfx := mpp("0.0.0.0/0")

		delta := n.Modify(pfx, func(_ int, found bool) (int, bool) {
			if found {
				t.Error("Should not find new prefix")
			}
			return 0, false
		})
		if delta != 1 {
			t.Errorf("Expected delta 1 for insert, got %d", delta)
		}
		if _, ok := n.Get(pfx); !ok {
			t.Errorf("Expected prefix to exist")
		}
	})

	t.Run("modify_at_lastOctet_update_existing", func(t *testing.T) {
		t.Parallel()
		n := &LiteNode[int]{}
		pfx := mpp("0.0.0.0/0")
		n.Insert(pfx, 0, 0)

		delta := n.Modify(pfx, func(_ int, found bool) (int, bool) {
			if !found {
				t.Errorf("Expected found=true")
			}
			return 0, false
		})
		if delta != 0 {
			t.Errorf("Expected delta 0 for update, got %d", delta)
		}
		if _, ok := n.Get(pfx); !ok {
			t.Errorf("Expected prefix to still exist")
		}
	})

	t.Run("modify_insert_path_compressed_fringe", func(t *testing.T) {
		t.Parallel()
		n := &LiteNode[int]{}
		pfx := mpp("10.0.0.0/8")

		delta := n.Modify(pfx, func(_ int, _ bool) (int, bool) { return 0, false })
		if delta != 1 {
			t.Errorf("Expected delta 1, got %d", delta)
		}
		// Verify via Get and by child type
		if _, ok := n.Get(pfx); !ok {
			t.Errorf("Expected fringe prefix to exist via Get")
		}
		if ch, ok := n.GetChild(10); !ok {
			t.Fatal("Child should exist")
		} else if _, isFr := ch.(*FringeNode[int]); !isFr {
			t.Errorf("Child should be FringeNode, got %T", ch)
		}
	})

	t.Run("modify_insert_path_compressed_leaf", func(t *testing.T) {
		t.Parallel()
		n := &LiteNode[int]{}
		pfx := mpp("10.1.0.0/16")

		delta := n.Modify(pfx, func(_ int, _ bool) (int, bool) { return 0, false })
		if delta != 1 {
			t.Errorf("Expected delta 1, got %d", delta)
		}
		// Verify via Get and by child type
		if _, ok := n.Get(pfx); !ok {
			t.Errorf("Expected leaf prefix to exist via Get")
		}
		if ch, ok := n.GetChild(10); !ok {
			t.Fatal("Child should exist at octet 10")
		} else if _, isLeaf := ch.(*LeafNode[int]); !isLeaf {
			t.Errorf("Child should be LeafNode, got %T", ch)
		}
	})

	t.Run("modify_delete_nonexistent_path_compressed", func(t *testing.T) {
		t.Parallel()
		n := &LiteNode[int]{}
		delta := n.Modify(mpp("10.1.0.0/16"), func(_ int, _ bool) (int, bool) { return 0, true })
		if delta != 0 {
			t.Errorf("Expected delta 0 for no-op, got %d", delta)
		}
	})

	t.Run("modify_update_leaf_same_prefix", func(t *testing.T) {
		t.Parallel()
		n := &LiteNode[int]{}
		pfx := mpp("10.1.0.0/16")
		n.Insert(pfx, 0, 0)

		delta := n.Modify(pfx, func(_ int, found bool) (int, bool) {
			if !found {
				t.Errorf("Expected found=true")
			}
			return 0, false
		})
		if delta != 0 {
			t.Errorf("Expected delta 0 for update, got %d", delta)
		}
		if _, ok := n.Get(pfx); !ok {
			t.Errorf("Expected prefix to still exist")
		}
	})

	t.Run("modify_delete_leaf_same_prefix", func(t *testing.T) {
		t.Parallel()
		n := &LiteNode[int]{}
		pfx := mpp("10.1.0.0/16")
		n.Insert(pfx, 0, 0)

		delta := n.Modify(pfx, func(_ int, _ bool) (int, bool) { return 0, true })
		if delta != -1 {
			t.Errorf("Expected delta -1, got %d", delta)
		}
		if !n.IsEmpty() {
			t.Error("Node should be empty after delete and purge")
		}
	})

	t.Run("modify_insert_creates_node_from_leaf", func(t *testing.T) {
		t.Parallel()
		n := &LiteNode[int]{}
		p1 := mpp("10.1.0.0/16")
		p2 := mpp("10.1.1.0/24")
		n.Insert(p1, 0, 0)

		delta := n.Modify(p2, func(_ int, found bool) (int, bool) {
			if found {
				t.Error("Should not find")
			}
			return 0, false
		})
		if delta != 1 {
			t.Errorf("Expected delta 1, got %d", delta)
		}
		if _, ok := n.Get(p1); !ok {
			t.Errorf("Expected p1 to exist")
		}
		if _, ok := n.Get(p2); !ok {
			t.Errorf("Expected p2 to exist")
		}
	})

	t.Run("modify_delete_noop_from_leaf_mismatch", func(t *testing.T) {
		t.Parallel()
		n := &LiteNode[int]{}
		n.Insert(mpp("10.1.0.0/16"), 0, 0)

		delta := n.Modify(mpp("10.1.1.0/24"), func(_ int, _ bool) (int, bool) { return 0, true })
		if delta != 0 {
			t.Errorf("Expected delta 0, got %d", delta)
		}
	})

	t.Run("modify_update_fringe_same_prefix", func(t *testing.T) {
		t.Parallel()
		n := &LiteNode[int]{}
		pfx := mpp("10.0.0.0/8")
		n.Insert(pfx, 0, 0)

		delta := n.Modify(pfx, func(_ int, found bool) (int, bool) {
			if !found {
				t.Errorf("Expected found=true")
			}
			return 0, false
		})
		if delta != 0 {
			t.Errorf("Expected delta 0, got %d", delta)
		}
		if _, ok := n.Get(pfx); !ok {
			t.Errorf("Expected fringe prefix to still exist")
		}
	})

	t.Run("modify_delete_fringe", func(t *testing.T) {
		t.Parallel()
		n := &LiteNode[int]{}
		pfx := mpp("10.0.0.0/8")
		n.Insert(pfx, 0, 0)

		delta := n.Modify(pfx, func(_ int, _ bool) (int, bool) { return 0, true })
		if delta != -1 {
			t.Errorf("Expected delta -1, got %d", delta)
		}
		if !n.IsEmpty() {
			t.Error("Node should be empty")
		}
	})

	t.Run("modify_insert_creates_node_from_fringe", func(t *testing.T) {
		t.Parallel()
		n := &LiteNode[int]{}
		pFr := mpp("10.0.0.0/8")
		pLeaf := mpp("10.1.0.0/16")
		n.Insert(pFr, 0, 0)

		delta := n.Modify(pLeaf, func(_ int, _ bool) (int, bool) { return 0, false })
		if delta != 1 {
			t.Errorf("Expected delta 1, got %d", delta)
		}
		if _, ok := n.Get(pFr); !ok {
			t.Errorf("Fringe should still exist")
		}
		if _, ok := n.Get(pLeaf); !ok {
			t.Errorf("Leaf should exist")
		}
	})

	t.Run("modify_delete_noop_from_fringe_mismatch", func(t *testing.T) {
		t.Parallel()
		n := &LiteNode[int]{}
		n.Insert(mpp("10.0.0.0/8"), 0, 0)

		delta := n.Modify(mpp("10.1.0.0/16"), func(_ int, _ bool) (int, bool) { return 0, true })
		if delta != 0 {
			t.Errorf("Expected delta 0, got %d", delta)
		}
	})

	t.Run("modify_noop_update", func(t *testing.T) {
		t.Parallel()
		n := &LiteNode[int]{}
		pfx := mpp("0.0.0.0/0")
		n.Insert(pfx, 0, 0)

		delta := n.Modify(pfx, func(_ int, found bool) (int, bool) {
			if !found {
				t.Fatalf("Expected found")
			}
			return 0, false // same presence
		})
		if delta != 0 {
			t.Errorf("Expected delta 0, got %d", delta)
		}
	})
}

func TestLiteNode_DumpString_IPv4_DeepSubtree(t *testing.T) {
	t.Parallel()

	var root LiteNode[int]

	// root --(10)--> lvl1 --(1)--> lvl2
	lvl1 := &LiteNode[int]{}
	lvl2 := &LiteNode[int]{}

	// only presence, no values
	lvl1.InsertPrefix(200, 0)
	lvl2.InsertPrefix(32, 0)
	lvl2.InsertPrefix(64, 0)

	lvl1.InsertChild(1, lvl2)
	root.InsertChild(10, lvl1)

	// Deep dump [10,1]
	outDeep := root.DumpString([]uint8{10, 1}, 2, true, true)
	if outDeep == "" || strings.Contains(outDeep, "ERROR:") {
		t.Fatalf("unexpected dump: %q", outDeep)
	}
	if !strings.Contains(outDeep, "depth:") {
		t.Fatalf("missing depth marker in deep dump: %q", outDeep)
	}

	// Intermediate dump [10]
	outLvl1 := root.DumpString([]uint8{10}, 1, true, true)
	if strings.Contains(outLvl1, "ERROR:") {
		t.Fatalf("lvl1 dump error: %q", outLvl1)
	}

	// Deep dump without values
	outDeepNoVals := root.DumpString([]uint8{10, 1}, 2, true, false)
	if outDeepNoVals == "" || strings.Contains(outDeepNoVals, "ERROR:") {
		t.Fatalf("unexpected dump (no vals): %q", outDeepNoVals)
	}
}

func TestLiteNode_DumpString_Error_KidNotSet_AtRootStep(t *testing.T) {
	t.Parallel()
	var root LiteNode[int]

	out := root.DumpString([]uint8{10}, 1, true, true)
	if out == "" || !strings.Contains(out, "ERROR:") || !strings.Contains(out, "NOT set in node") || !strings.Contains(out, "[0]") {
		t.Fatalf("expected missing-kid error with [0], got: %q", out)
	}
}

func TestLiteNode_DumpString_Error_KidNotSet_AtDeeperStep(t *testing.T) {
	t.Parallel()
	var root LiteNode[int]
	lvl1 := &LiteNode[int]{}
	root.InsertChild(10, lvl1)

	out := root.DumpString([]uint8{10, 1}, 2, true, true)
	if out == "" || !strings.Contains(out, "ERROR:") || !strings.Contains(out, "NOT set in node") || !strings.Contains(out, "[1]") {
		t.Fatalf("expected missing-kid error with [1], got: %q", out)
	}
}

func TestLiteNode_DumpString_Error_KidWrongType_LeafAtDeeperStep(t *testing.T) {
	t.Parallel()
	var root LiteNode[int]
	lvl1 := &LiteNode[int]{}
	leaf := &LeafNode[int]{Prefix: mpp("10.1.0.0/16")}
	lvl1.InsertChild(1, leaf)
	root.InsertChild(10, lvl1)

	out := root.DumpString([]uint8{10, 1}, 2, true, true)
	if out == "" || !strings.Contains(out, "ERROR:") || !strings.Contains(out, "NO LiteNode") || !strings.Contains(out, "[1]") {
		t.Fatalf("expected wrong-type error (Leaf), got: %q", out)
	}
	if strings.Contains(out, "depth:") {
		t.Fatalf("unexpected normal dump on error, got: %q", out)
	}
}

func TestLiteNode_DumpString_Error_KidWrongType_FringeAtDeeperStep(t *testing.T) {
	t.Parallel()
	var root LiteNode[int]
	lvl1 := &LiteNode[int]{}
	fringe := &FringeNode[int]{}
	lvl1.InsertChild(2, fringe)
	root.InsertChild(10, lvl1)

	out := root.DumpString([]uint8{10, 2}, 2, true, true)
	if out == "" || !strings.Contains(out, "ERROR:") || !strings.Contains(out, "NO LiteNode") || !strings.Contains(out, "[1]") {
		t.Fatalf("expected wrong-type error (Fringe), got: %q", out)
	}
	if strings.Contains(out, "depth:") {
		t.Fatalf("unexpected normal dump on error, got: %q", out)
	}
}

func TestLiteInsertShuffled(t *testing.T) {
	// The order in which you insert prefixes into a route table
	// should not matter, as long as you're inserting the same set of
	// routes.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := random.RealWorldPrefixes(prng, n)

	for range 10 {
		pfxs2 := slices.Clone(pfxs)
		prng.Shuffle(len(pfxs2), func(i, j int) { pfxs2[i], pfxs2[j] = pfxs2[j], pfxs2[i] })

		lite1 := new(LiteNode[struct{}])
		lite2 := new(LiteNode[struct{}])

		for _, pfx := range pfxs {
			lite1.Insert(pfx, struct{}{}, 0)
			lite1.Insert(pfx, struct{}{}, 0) // idempotent
		}
		for _, pfx := range pfxs2 {
			lite2.Insert(pfx, struct{}{}, 0)
			lite2.Insert(pfx, struct{}{}, 0) // idempotent
		}

		if !lite1.EqualRec(lite2) {
			t.Fatal("expected Equal")
		}
	}
}

func TestLiteInsertPersistShuffled(t *testing.T) {
	// The order in which you insert prefixes into a route table
	// should not matter, as long as you're inserting the same set of
	// routes.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := random.RealWorldPrefixes(prng, n)

	for range 10 {
		pfxs2 := slices.Clone(pfxs)
		prng.Shuffle(len(pfxs2), func(i, j int) { pfxs2[i], pfxs2[j] = pfxs2[j], pfxs2[i] })

		lite1 := new(LiteNode[struct{}])
		lite2 := new(LiteNode[struct{}])

		// lite1 is mutable
		for _, pfx := range pfxs {
			lite1.Insert(pfx, struct{}{}, 0)
			lite1.Insert(pfx, struct{}{}, 0) // idempotent
		}

		// lite2 is persistent
		for _, pfx := range pfxs2 {
			lite2.InsertPersist(nil, pfx, struct{}{}, 0)
			lite2.InsertPersist(nil, pfx, struct{}{}, 0) // idempotent
		}

		if !lite1.EqualRec(lite2) {
			t.Fatal("expected Equal")
		}
	}
}

func TestLiteDeleteCompare4(t *testing.T) {
	t.Parallel()

	var (
		n         = workLoadN()
		prng      = rand.New(rand.NewPCG(42, 42))
		deleteCut = n / 2
		probes    = 3
	)

	for range probes {
		all4 := random.RealWorldPrefixes4(prng, n)

		// pfxs and toDelete should be non-overlapping sets
		pfxs := all4[:deleteCut]
		toDelete := all4[deleteCut:]

		gold := new(golden.Table[struct{}])
		lite := new(LiteNode[struct{}])

		for _, pfx := range pfxs {
			gold.Insert(pfx, struct{}{})
			lite.Insert(pfx, struct{}{}, 0)
		}

		for _, pfx := range toDelete {
			gold.Delete(pfx)
			lite.Delete(pfx)
		}

		collect := []netip.Prefix{}
		for pfx := range lite.allSorted4() {
			collect = append(collect, pfx)
		}

		if !slices.Equal(gold.AllSorted(), collect) {
			t.Fatal("expected Equal")
		}
	}
}

func TestLiteDeleteCompare6(t *testing.T) {
	t.Parallel()

	var (
		n         = workLoadN()
		prng      = rand.New(rand.NewPCG(42, 42))
		deleteCut = n / 2
		probes    = 3
	)

	for range probes {
		all6 := random.RealWorldPrefixes6(prng, n)

		// pfxs and toDelete should be non-overlapping sets
		pfxs := all6[:deleteCut]
		toDelete := all6[deleteCut:]

		gold := new(golden.Table[struct{}])
		lite := new(LiteNode[struct{}])

		for _, pfx := range pfxs {
			gold.Insert(pfx, struct{}{})
			lite.Insert(pfx, struct{}{}, 0)
		}

		for _, pfx := range toDelete {
			gold.Delete(pfx)
			lite.Delete(pfx)
		}

		collect := []netip.Prefix{}
		for pfx := range lite.allSorted6() {
			collect = append(collect, pfx)
		}

		if !slices.Equal(gold.AllSorted(), collect) {
			t.Fatal("expected Equal")
		}
	}
}

func TestLiteDeleteShuffled4(t *testing.T) {
	// The order in which you delete prefixes from a route table
	// should not matter, as long as you're deleting the same set of
	// routes.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))

	deleteCut := n / 2

	for range 10 {
		all4 := random.RealWorldPrefixes4(prng, n)

		// pfxs and toDelete should be non-overlapping sets
		toDelete := all4[deleteCut:]

		lite := new(LiteNode[struct{}])

		// insert
		for _, pfx := range all4 {
			lite.Insert(pfx, struct{}{}, 0)
		}

		// delete
		for _, pfx := range toDelete {
			lite.Delete(pfx)
		}

		toDelete2 := slices.Clone(toDelete)
		prng.Shuffle(len(toDelete2), func(i, j int) { toDelete2[i], toDelete2[j] = toDelete2[j], toDelete2[i] })

		lite2 := new(LiteNode[struct{}])

		// insert
		for _, pfx := range all4 {
			lite2.Insert(pfx, struct{}{}, 0)
		}

		// delete
		for _, pfx := range toDelete2 {
			lite2.Delete(pfx)
		}

		if !lite.EqualRec(lite2) {
			t.Fatal("expect equal")
		}
	}
}

func TestLiteDeleteShuffled6(t *testing.T) {
	// The order in which you delete prefixes from a route table
	// should not matter, as long as you're deleting the same set of
	// routes.
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))

	deleteCut := n / 2

	for range 10 {
		all6 := random.RealWorldPrefixes6(prng, n)

		// pfxs and toDelete should be non-overlapping sets
		toDelete := all6[deleteCut:]

		lite := new(LiteNode[struct{}])

		// insert
		for _, pfx := range all6 {
			lite.Insert(pfx, struct{}{}, 0)
		}

		// delete
		for _, pfx := range toDelete {
			lite.Delete(pfx)
		}

		toDelete2 := slices.Clone(toDelete)
		prng.Shuffle(len(toDelete2), func(i, j int) { toDelete2[i], toDelete2[j] = toDelete2[j], toDelete2[i] })

		lite2 := new(LiteNode[struct{}])

		// insert
		for _, pfx := range all6 {
			lite2.Insert(pfx, struct{}{}, 0)
		}

		// delete
		for _, pfx := range toDelete2 {
			lite2.Delete(pfx)
		}

		if !lite.EqualRec(lite2) {
			t.Fatal("expect equal")
		}
	}
}

func TestLiteDeleteIsReverseOfInsert4(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	// Insert n prefixes, then delete those same prefixes in reverse
	// order. Each deletion should exactly undo the internal structure
	// changes that each insert did.

	pfxs := random.RealWorldPrefixes4(prng, n)

	lite := new(LiteNode[struct{}])
	for _, pfx := range pfxs {
		lite.Insert(pfx, struct{}{}, 0)
	}

	for i := len(pfxs) - 1; i >= 0; i-- {
		lite.Delete(pfxs[i])
	}

	if !lite.IsEmpty() {
		t.Errorf("after delete, expected empty LiteNode")
	}
	if nt := lite.hasType(); nt != nullNode {
		t.Fatalf("after delete, expected NULL node, but got %s", nt)
	}
}

func TestLiteDeleteIsReverseOfInsert6(t *testing.T) {
	t.Parallel()

	n := workLoadN()

	prng := rand.New(rand.NewPCG(42, 42))
	// Insert n prefixes, then delete those same prefixes in reverse
	// order. Each deletion should exactly undo the internal structure
	// changes that each insert did.

	pfxs := random.RealWorldPrefixes6(prng, n)

	lite := new(LiteNode[struct{}])
	for _, pfx := range pfxs {
		lite.Insert(pfx, struct{}{}, 0)
	}

	for i := len(pfxs) - 1; i >= 0; i-- {
		lite.Delete(pfxs[i])
	}

	if !lite.IsEmpty() {
		t.Errorf("after delete, expected empty LiteNode")
	}
	if nt := lite.hasType(); nt != nullNode {
		t.Fatalf("after delete, expected NULL node, but got %s", nt)
	}
}

func TestLiteDeleteButOne4(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))
	// Insert n prefixes, then delete all but one
	n := workLoadN()

	for range 10 {

		lite := new(LiteNode[struct{}])
		prefixes := random.RealWorldPrefixes4(prng, n)

		for _, pfx := range prefixes {
			lite.Insert(pfx, struct{}{}, 0)
		}

		// shuffle the prefixes
		prng.Shuffle(n, func(i, j int) {
			prefixes[i], prefixes[j] = prefixes[j], prefixes[i]
		})

		// skip the first
		for i := 1; i < len(prefixes); i++ {
			lite.Delete(prefixes[i])
		}

		if nt := lite.hasType(); nt != stopNode {
			t.Fatalf("after delete but one, expected STOP node, but got %s", nt)
		}
		if sum := lite.PrefixCount() + lite.ChildCount(); sum != 1 {
			t.Fatalf("after delete but one, sum of prefixes and children must be 1, got %d", sum)
		}
	}
}

func TestLiteDeleteButOne6(t *testing.T) {
	t.Parallel()
	prng := rand.New(rand.NewPCG(42, 42))
	// Insert n prefixes, then delete all but one
	n := workLoadN()

	for range 10 {

		lite := new(LiteNode[struct{}])
		prefixes := random.RealWorldPrefixes6(prng, n)

		for _, pfx := range prefixes {
			lite.Insert(pfx, struct{}{}, 0)
		}

		// shuffle the prefixes
		prng.Shuffle(n, func(i, j int) {
			prefixes[i], prefixes[j] = prefixes[j], prefixes[i]
		})

		// skip the first
		for i := 1; i < len(prefixes); i++ {
			lite.Delete(prefixes[i])
		}

		if nt := lite.hasType(); nt != stopNode {
			t.Fatalf("after delete but one, expected STOP node, but got %s", nt)
		}
		if sum := lite.PrefixCount() + lite.ChildCount(); sum != 1 {
			t.Fatalf("after delete but one, sum of prefixes and children must be 1, got %d", sum)
		}
	}
}
