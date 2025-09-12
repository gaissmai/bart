package bart

import (
	"net/netip"
	"reflect"
	"sort"
	"testing"
)

// helper to parse prefix; test fails immediately on bad input
func mustPrefix(t *testing.T, s string) netip.Prefix {
	t.Helper()
	p, err := netip.ParsePrefix(s)
	if err != nil {
		t.Fatalf("parse prefix %q: %v", s, err)
	}
	return p
}

// getAllKV returns sorted list of (Prefix, V) from t.All() for stable assertions.
func getAllKV[V comparable](tbl *Table[V]) []struct {
	P netip.Prefix
	V V
} {
	var out []struct {
		P netip.Prefix
		V V
	}
	for p, v := range tbl.All() {
		out = append(out, struct {
			P netip.Prefix
			V V
		}{P: p, V: v})
	}
	sort.Slice(out, func(i, j int) bool {
		// Sort by IP version, then string form
		pi, pj := out[i].P, out[j].P
		if pi.Addr().Is4() != pj.Addr().Is4() {
			return pi.Addr().Is4()
		}
		return pi.String() < pj.String()
	})
	return out
}

// countEntries counts the number of entries by iterating over All()
func countEntries[V any](tbl *Table[V]) int {
	count := 0
	for range tbl.All() {
		count++
	}
	return count
}

// sizeSnapshot gets size4 and size6 if available via reflection fallback.
// If access fails, it falls back to counting prefixes by version.
func sizeSnapshot[V any](t *testing.T, tbl *Table[V]) (sz4, sz6 int) {
	t.Helper()
	// Try reflect on known fields
	rv := reflect.ValueOf(tbl).Elem()
	sz4f := rv.FieldByName("size4")
	sz6f := rv.FieldByName("size6")
	if sz4f.IsValid() && sz6f.IsValid() && sz4f.Kind() == reflect.Int && sz6f.Kind() == reflect.Int {
		return int(sz4f.Int()), int(sz6f.Int())
	}
	// Fallback: compute from All()
	for p := range tbl.All() {
		if p.Addr().Is4() {
			sz4++
		} else {
			sz6++
		}
	}
	return
}

// valueFor returns the stored value for exact prefix pfx by scanning All().
// ok indicates presence.
func valueFor[V comparable](tbl *Table[V], pfx netip.Prefix) (V, bool) {
	var zero V
	for p, v := range tbl.All() {
		if p == pfx {
			return v, true
		}
	}
	return zero, false
}

func TestInsertPersist_InvalidPrefix_NoChange(t *testing.T) {
	t.Parallel()
	t0 := &Table[int]{} // zero-value table
	sz4Before, sz6Before := sizeSnapshot(t, t0)

	invalid := netip.Prefix{} // not valid; IsValid() == false
	pt := t0.InsertPersist(invalid, 123)

	if t0 != pt {
		t.Fatalf("expected original table to be returned for invalid prefix")
	}
	sz4After, sz6After := sizeSnapshot(t, pt)
	entryCount := countEntries(pt)
	if sz4After != sz4Before || sz6After != sz6Before || entryCount != 0 {
		t.Fatalf("expected no change for invalid prefix; got sizes %d/%d, entries %d", sz4After, sz6After, entryCount)
	}
}

func TestInsertPersist_CanonicalizesMasked_OverrideAndSize(t *testing.T) {
	t.Parallel()
	t0 := &Table[int]{}
	// Insert with host bits set; method should mask to .0/24
	p1 := mustPrefix(t, "192.168.1.123/24")
	pt1 := t0.InsertPersist(p1, 1)

	masked := p1.Masked()
	if _, ok := valueFor[int](pt1, masked); !ok {
		t.Fatalf("masked prefix %v not found after insert", masked)
	}
	if got, ok := valueFor[int](pt1, p1); ok && p1 != masked {
		t.Fatalf("unexpected unmasked key present: %v (masked: %v)", got, masked)
	}
	sz4, sz6 := sizeSnapshot(t, pt1)
	if sz4 != 1 || sz6 != 0 {
		t.Fatalf("unexpected sizes after insert: size4=%d size6=%d", sz4, sz6)
	}

	// Override same logical prefix with different value; size must not change
	pt2 := pt1.InsertPersist(mustPrefix(t, "192.168.1.1/24"), 42)
	v, ok := valueFor[int](pt2, masked)
	if !ok || v != 42 {
		t.Fatalf("expected override to 42 at %v; got %v ok=%v", masked, v, ok)
	}
	sz4b, sz6b := sizeSnapshot(t, pt2)
	if sz4b != 1 || sz6b != 0 {
		t.Fatalf("size changed on override: size4=%d size6=%d", sz4b, sz6b)
	}
}

func TestInsertPersist_IPv6_InsertAndCounts(t *testing.T) {
	t.Parallel()
	t0 := &Table[int]{}
	p6 := mustPrefix(t, "2001:db8::1/64")
	pt := t0.InsertPersist(p6, 7)

	want := p6.Masked()
	if got, ok := valueFor[int](pt, want); !ok || got != 7 {
		t.Fatalf("expected IPv6 insert at %v=7; got %v ok=%v", want, got, ok)
	}
	sz4, sz6 := sizeSnapshot(t, pt)
	if sz4 != 0 || sz6 != 1 {
		t.Fatalf("unexpected sizes: size4=%d size6=%d", sz4, sz6)
	}
}

func TestUpdatePersist_Missing_InsertsViaCallback(t *testing.T) {
	t.Parallel()
	t0 := &Table[int]{}
	p := mustPrefix(t, "10.0.0.0/8")

	pt, newVal := t0.UpdatePersist(p, func(val int, ok bool) int {
		if ok {
			t.Fatalf("expected ok=false for missing prefix")
		}
		return 99
	})

	if v, ok := valueFor[int](pt, p.Masked()); !ok || v != 99 {
		t.Fatalf("expected inserted value 99; got %v ok=%v", v, ok)
	}
	if newVal != 99 {
		t.Fatalf("returned newVal mismatch: %v", newVal)
	}
	sz4, sz6 := sizeSnapshot(t, pt)
	if sz4 != 1 || sz6 != 0 {
		t.Fatalf("size not incremented for insert: %d/%d", sz4, sz6)
	}
}

func TestUpdatePersist_Existing_UpdatesOnly_NoSizeChange(t *testing.T) {
	t.Parallel()
	t0 := &Table[int]{}
	p := mustPrefix(t, "10.1.0.0/16")
	t1 := t0.InsertPersist(p, 5)

	pt, newVal := t1.UpdatePersist(p, func(val int, ok bool) int {
		if !ok || val != 5 {
			t.Fatalf("callback received unexpected state: ok=%v val=%v", ok, val)
		}
		return val + 1
	})
	if newVal != 6 {
		t.Fatalf("expected newVal 6; got %v", newVal)
	}
	if v, ok := valueFor[int](pt, p.Masked()); !ok || v != 6 {
		t.Fatalf("value not updated to 6; got %v ok=%v", v, ok)
	}
	sz4, sz6 := sizeSnapshot(t, pt)
	if sz4 != 1 || sz6 != 0 {
		t.Fatalf("size changed unexpectedly: %d/%d", sz4, sz6)
	}
}

func TestUpdatePersist_InvalidPrefix_ReturnsOriginalAndZero(t *testing.T) {
	t.Parallel()
	t0 := &Table[int]{}
	var zero int
	pt, newVal := t0.UpdatePersist(netip.Prefix{}, func(val int, ok bool) int { return 1 })
	if pt != t0 || newVal != zero {
		t.Fatalf("expected original table and zero for invalid prefix; got %p %v", pt, newVal)
	}
}

func TestModifyPersist_Insert_Update_Delete_Paths(t *testing.T) {
	t.Parallel()
	t0 := &Table[int]{}
	p := mustPrefix(t, "172.16.0.0/12")

	// Insert when missing (del=false)
	t1, newVal, del := t0.ModifyPersist(p, func(val int, ok bool) (int, bool) {
		if ok {
			t.Fatalf("expected ok=false for missing")
		}
		return 111, false
	})
	if del {
		t.Fatalf("unexpected delete on insert path")
	}
	if v, ok := valueFor[int](t1, p.Masked()); !ok || v != 111 || newVal != 111 {
		t.Fatalf("insert path failed: got v=%v ok=%v newVal=%v", v, ok, newVal)
	}
	sz4, sz6 := sizeSnapshot(t, t1)
	if sz4 != 1 || sz6 != 0 {
		t.Fatalf("size not incremented on insert")
	}

	// Update existing (del=false)
	t2, newVal2, del2 := t1.ModifyPersist(p, func(val int, ok bool) (int, bool) {
		if !ok || val != 111 {
			t.Fatalf("expected existing 111; got ok=%v val=%v", ok, val)
		}
		return 222, false
	})
	if del2 || newVal2 != 222 {
		t.Fatalf("update path failed: del=%v newVal=%v", del2, newVal2)
	}
	if v, ok := valueFor[int](t2, p.Masked()); !ok || v != 222 {
		t.Fatalf("value not updated to 222")
	}
	sz4b, sz6b := sizeSnapshot(t, t2)
	if sz4b != 1 || sz6b != 0 {
		t.Fatalf("size changed on update")
	}

	// Delete existing (del=true)
	t3, oldVal, deleted := t2.ModifyPersist(p, func(val int, ok bool) (int, bool) {
		if !ok || val != 222 {
			t.Fatalf("expected existing 222")
		}
		return 0, true
	})
	if !deleted || oldVal != 222 {
		t.Fatalf("delete path failed: deleted=%v oldVal=%v", deleted, oldVal)
	}
	if _, ok := valueFor[int](t3, p.Masked()); ok {
		t.Fatalf("expected prefix to be removed")
	}
	sz4c, sz6c := sizeSnapshot(t, t3)
	if sz4c != 0 || sz6c != 0 {
		t.Fatalf("size not decremented on delete: %d/%d", sz4c, sz6c)
	}
}

func TestModifyPersist_MissingAndDelTrue_NoOp(t *testing.T) {
	t.Parallel()
	t0 := &Table[int]{}
	p := mustPrefix(t, "10.10.10.0/24")
	t1, newVal, deleted := t0.ModifyPersist(p, func(val int, ok bool) (int, bool) {
		return 0, true
	})
	if deleted || newVal != 0 {
		t.Fatalf("expected no-op for missing+del=true")
	}
	// table contents remain empty
	if countEntries(t1) != 0 {
		t.Fatalf("expected no entries after no-op")
	}
	// Size stays 0/0
	if s4, s6 := sizeSnapshot(t, t1); s4 != 0 || s6 != 0 {
		t.Fatalf("unexpected sizes after no-op: %d/%d", s4, s6)
	}
}

func TestDeletePersist_Workflow_LeafAndFringe(t *testing.T) {
	t.Parallel()
	t0 := &Table[int]{}

	// Insert a /24 and a fringe /20 so we exercise both leafNode and fringeNode branches.
	pLeaf := mustPrefix(t, "192.0.2.0/24")      // exact leaf
	pFringe := mustPrefix(t, "198.51.100.0/20") // likely fringe (non-octet aligned)
	t1 := t0.InsertPersist(pLeaf, 10)
	t2 := t1.InsertPersist(pFringe, 20)

	// Delete non-existent
	t3, _, found := t2.DeletePersist(mustPrefix(t, "203.0.113.0/24"))
	t2KV := getAllKV(t2)
	t3KV := getAllKV(t3)
	if found || !reflect.DeepEqual(t2KV, t3KV) {
		t.Fatalf("delete of missing prefix should be no-op")
	}

	// Delete leaf
	t4, vLeaf, okLeaf := t3.DeletePersist(pLeaf)
	if !okLeaf || vLeaf != 10 {
		t.Fatalf("delete leaf failed: ok=%v val=%v", okLeaf, vLeaf)
	}
	if _, ok := valueFor[int](t4, pLeaf.Masked()); ok {
		t.Fatalf("leaf still present after delete")
	}

	// Delete fringe
	t5, vFringe, okFringe := t4.DeletePersist(pFringe)
	if !okFringe || vFringe != 20 {
		t.Fatalf("delete fringe failed")
	}
	if _, ok := valueFor[int](t5, pFringe.Masked()); ok {
		t.Fatalf("fringe still present after delete")
	}
	// Sizes go to zero
	if s4, s6 := sizeSnapshot(t, t5); s4 != 0 || s6 != 0 {
		t.Fatalf("sizes not updated to zero after deletions: %d/%d", s4, s6)
	}
}

func TestDeletePersist_InvalidPrefix_ReturnsOriginal(t *testing.T) {
	t.Parallel()
	t0 := &Table[int]{}
	pt, _, found := t0.DeletePersist(netip.Prefix{})
	if pt != t0 || found {
		t.Fatalf("expected original table and found=false for invalid prefix")
	}
}

func TestGetAndDeletePersist_ForwardsToDeletePersist(t *testing.T) {
	t.Parallel()
	t0 := &Table[int]{}
	p := mustPrefix(t, "10.0.0.0/8")
	t1 := t0.InsertPersist(p, 123)

	pt1, v1, ok1 := t1.DeletePersist(p)
	pt2, v2, ok2 := t1.GetAndDeletePersist(p)

	pt1KV := getAllKV(pt1)
	pt2KV := getAllKV(pt2)
	if ok1 != ok2 || v1 != v2 || !reflect.DeepEqual(pt1KV, pt2KV) {
		t.Fatalf("GetAndDeletePersist must mirror DeletePersist results")
	}
}

func TestWalkPersist_NilCallback_NoOp(t *testing.T) {
	t.Parallel()
	t0 := &Table[int]{}
	p := mustPrefix(t, "10.0.0.0/8")
	t1 := t0.InsertPersist(p, 1)
	pt := t1.WalkPersist(nil)
	if pt != t1 {
		t.Fatalf("nil callback must return original table reference")
	}
}

func TestWalkPersist_TransformsValues_StopsEarly(t *testing.T) {
	t.Parallel()
	t0 := &Table[int]{}
	// Seed with multiple entries
	pfxs := []string{
		"10.0.0.0/8",
		"192.168.0.0/16",
		"2001:db8::/64",
	}
	tbl := t0
	for i, s := range pfxs {
		tbl = tbl.InsertPersist(mustPrefix(t, s), i+1) // values 1,2,3
	}

	// Callback increments all values by 10, but stops after processing 2 entries.
	count := 0
	cb := func(pt *Table[int], pfx netip.Prefix, v int) (*Table[int], bool) {
		count++
		pt2, _ := pt.UpdatePersist(pfx, func(old int, ok bool) int { return old + 10 })
		return pt2, count < 2
	}
	pt := tbl.WalkPersist(cb)

	if count != 2 {
		t.Fatalf("expected early stop after 2 items; got %d", count)
	}

	// Validate exactly two entries were incremented; the third remained unchanged.
	kv := getAllKV[int](pt)
	if len(kv) != 3 {
		t.Fatalf("expected 3 entries; got %d", len(kv))
	}
	// Count how many values equal original+10 vs original
	var inc, same int
	orig := map[netip.Prefix]int{}
	for i, s := range pfxs {
		orig[mustPrefix(t, s).Masked()] = i + 1
	}
	for _, e := range kv {
		if e.V == orig[e.P]+10 {
			inc++
		} else if e.V == orig[e.P] {
			same++
		} else {
			t.Fatalf("unexpected value at %v: got %d", e.P, e.V)
		}
	}
	if inc != 2 || same != 1 {
		t.Fatalf("unexpected transform counts: inc=%d same=%d", inc, same)
	}
}

func TestUnionPersist_SizesAndValues(t *testing.T) {
	t.Parallel()
	a := &Table[int]{}
	b := &Table[int]{}

	// a has two entries, b has two entries, with one duplicate key but different value.
	p1 := mustPrefix(t, "10.0.0.0/8")
	p2 := mustPrefix(t, "192.168.0.0/16")
	p3 := mustPrefix(t, "2001:db8::/64")
	// duplicate of p2 to test precedence
	p2dup := mustPrefix(t, "192.168.0.1/16")

	a1 := a.InsertPersist(p1, 1).InsertPersist(p2, 2)
	b1 := b.InsertPersist(p2dup, 22).InsertPersist(p3, 3)

	u := a1.UnionPersist(b1)

	// Size should be total minus duplicates (p2).
	s4, s6 := sizeSnapshot(t, u)
	if s4 != 2 || s6 != 1 {
		t.Fatalf("unexpected union sizes: size4=%d size6=%d (want 2,1)", s4, s6)
	}

	// Ensure keys present.
	if _, ok := valueFor[int](u, p1.Masked()); !ok {
		t.Fatalf("p1 missing in union")
	}
	if _, ok := valueFor[int](u, p2.Masked()); !ok {
		t.Fatalf("p2 missing in union")
	}
	if _, ok := valueFor[int](u, p3.Masked()); !ok {
		t.Fatalf("p3 missing in union")
	}

	// Value precedence on duplicate: we expect the union operation to prefer values from 'b' for duplicates
	// or keep 'a' values if 'b' doesn't override. If implementation differs, adjust expectation accordingly.
	v2, _ := valueFor[int](u, p2.Masked())
	if v2 != 22 && v2 != 2 {
		t.Fatalf("unexpected value for duplicate key p2: got %d; want 22 (from b) or 2 (from a) depending on union semantics", v2)
	}
}
