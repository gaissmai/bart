package bart

import (
	"net/netip"
	"testing"
)

// Test types for cloning behavior - defined at package level

// nonClonerStruct for testing pointer behavior without cloning
type nonClonerStruct struct {
	value int
}

// clonerStruct for testing pointer behavior with cloning
type clonerStruct struct {
	value int
}

func (c *clonerStruct) Clone() *clonerStruct {
	return &clonerStruct{value: c.value + 1000}
}

// Basic persistence tests

func TestInsertPersist_InvalidPrefix_NoChange(t *testing.T) {
	t.Parallel()
	t0 := &Table[int]{}

	invalid := netip.Prefix{} // not valid; IsValid() == false
	pt := t0.InsertPersist(invalid, 123)

	if t0 != pt {
		t.Fatalf("expected original table to be returned for invalid prefix")
	}

	if pt.Size() != 0 {
		t.Fatalf("expected empty table after invalid insert, got size %d", pt.Size())
	}
}

func TestInsertPersist_CanonicalizesMasked_OverrideAndSize(t *testing.T) {
	t.Parallel()
	t0 := &Table[int]{}

	// Insert with host bits set; method should mask to .0/24
	p1 := netip.MustParsePrefix("192.168.1.123/24")
	pt1 := t0.InsertPersist(p1, 1)

	masked := p1.Masked()
	if v, ok := pt1.Get(masked); !ok || v != 1 {
		t.Fatalf("expected masked prefix %v with value 1; got %v ok=%v", masked, v, ok)
	}

	// Override same logical prefix with different value
	pt2 := pt1.InsertPersist(netip.MustParsePrefix("192.168.1.1/24"), 42)
	if v, ok := pt2.Get(masked); !ok || v != 42 {
		t.Fatalf("expected override to 42 at %v; got %v ok=%v", masked, v, ok)
	}

	if pt2.Size() != 1 {
		t.Fatalf("expected size 1 after override, got %d", pt2.Size())
	}
}

func TestInsertPersist_IPv6(t *testing.T) {
	t.Parallel()
	t0 := &Table[int]{}
	p6 := netip.MustParsePrefix("2001:db8::1/64")
	pt := t0.InsertPersist(p6, 7)

	want := p6.Masked()
	if v, ok := pt.Get(want); !ok || v != 7 {
		t.Fatalf("expected IPv6 insert at %v=7; got %v ok=%v", want, v, ok)
	}
}

// Minimal test for deprecated UpdatePersist
func TestUpdatePersist_BasicFunctionality(t *testing.T) {
	t.Parallel()
	t0 := &Table[int]{}
	p := netip.MustParsePrefix("10.0.0.0/8")

	pt, newVal := t0.UpdatePersist(p, func(val int, ok bool) int {
		if ok {
			t.Fatalf("expected ok=false for missing prefix")
		}
		return 99
	})

	if newVal != 99 {
		t.Fatalf("returned newVal mismatch: %v", newVal)
	}
	if v, ok := pt.Get(p.Masked()); !ok || v != 99 {
		t.Fatalf("expected inserted value 99; got %v ok=%v", v, ok)
	}
}

// Comprehensive tests for ModifyPersist
func TestModifyPersist_Insert_Update_Delete_Paths(t *testing.T) {
	t.Parallel()
	t0 := &Table[int]{}
	p := netip.MustParsePrefix("172.16.0.0/12")

	// Insert when missing (del=false) - returns (newVal, false)
	t1, newVal, del := t0.ModifyPersist(p, func(val int, ok bool) (int, bool) {
		if ok {
			t.Fatalf("expected ok=false for missing")
		}
		return 111, false
	})
	if del {
		t.Fatalf("unexpected delete on insert path")
	}
	if newVal != 111 {
		t.Fatalf("expected newVal 111 for insert, got %v", newVal)
	}
	if v, ok := t1.Get(p.Masked()); !ok || v != 111 {
		t.Fatalf("insert path failed: got v=%v ok=%v", v, ok)
	}
	if t1.Size() != 1 {
		t.Fatalf("expected size 1 after insert, got %d", t1.Size())
	}

	// Update existing (del=false) - returns (oldVal, false)
	t2, oldVal2, del2 := t1.ModifyPersist(p, func(val int, ok bool) (int, bool) {
		if !ok || val != 111 {
			t.Fatalf("expected existing 111; got ok=%v val=%v", ok, val)
		}
		return 222, false
	})
	if del2 {
		t.Fatalf("unexpected delete on update path")
	}
	if oldVal2 != 111 { // ModifyPersist returns OLD value for updates!
		t.Fatalf("update should return old value 111, got %v", oldVal2)
	}
	if v, ok := t2.Get(p.Masked()); !ok || v != 222 { // Table contains NEW value
		t.Fatalf("value not updated to 222")
	}
	if t2.Size() != 1 {
		t.Fatalf("expected size 1 after update, got %d", t2.Size())
	}

	// Delete existing (del=true) - returns (oldVal, true)
	t3, oldVal3, deleted := t2.ModifyPersist(p, func(val int, ok bool) (int, bool) {
		if !ok || val != 222 {
			t.Fatalf("expected existing 222")
		}
		return 0, true
	})
	if !deleted || oldVal3 != 222 {
		t.Fatalf("delete path failed: deleted=%v oldVal=%v", deleted, oldVal3)
	}
	if _, ok := t3.Get(p.Masked()); ok {
		t.Fatalf("expected prefix to be removed")
	}
	if t3.Size() != 0 {
		t.Fatalf("expected empty table after delete, got size %d", t3.Size())
	}
}

func TestModifyPersist_MissingAndDelTrue_NoOp(t *testing.T) {
	t.Parallel()
	t0 := &Table[int]{}
	p := netip.MustParsePrefix("10.10.10.0/24")
	t1, val, deleted := t0.ModifyPersist(p, func(val int, ok bool) (int, bool) {
		return 0, true
	})
	if deleted || val != 0 {
		t.Fatalf("expected no-op for missing+del=true (zero, false)")
	}
	if t1.Size() != 0 {
		t.Fatalf("expected no entries after no-op, got size %d", t1.Size())
	}
}

func TestModifyPersist_InvalidPrefix_ReturnsOriginalAndZero(t *testing.T) {
	t.Parallel()
	t0 := &Table[int]{}
	var zero int
	pt, val, deleted := t0.ModifyPersist(netip.Prefix{}, func(val int, ok bool) (int, bool) {
		return 1, false
	})
	if pt != t0 || val != zero || deleted {
		t.Fatalf("expected original table, zero value and deleted=false for invalid prefix")
	}
}

func TestDeletePersist_Workflow(t *testing.T) {
	t.Parallel()
	t0 := &Table[int]{}

	pLeaf := netip.MustParsePrefix("192.0.2.0/24")
	pFringe := netip.MustParsePrefix("198.51.100.0/20")
	t1 := t0.InsertPersist(pLeaf, 10)
	t2 := t1.InsertPersist(pFringe, 20)

	if t2.Size() != 2 {
		t.Fatalf("expected size 2 after inserts, got %d", t2.Size())
	}

	// Delete non-existent should be no-op
	t3, _, found := t2.DeletePersist(netip.MustParsePrefix("203.0.113.0/24"))
	if found {
		t.Fatalf("delete of missing prefix should return found=false")
	}
	if t3.Size() != 2 {
		t.Fatalf("delete of missing prefix should be no-op")
	}

	// Delete leaf
	t4, vLeaf, okLeaf := t3.DeletePersist(pLeaf)
	if !okLeaf || vLeaf != 10 {
		t.Fatalf("delete leaf failed: ok=%v val=%v", okLeaf, vLeaf)
	}
	if _, ok := t4.Get(pLeaf.Masked()); ok {
		t.Fatalf("leaf still present after delete")
	}
	if t4.Size() != 1 {
		t.Fatalf("expected size 1 after first delete, got %d", t4.Size())
	}

	// Delete fringe
	t5, vFringe, okFringe := t4.DeletePersist(pFringe)
	if !okFringe || vFringe != 20 {
		t.Fatalf("delete fringe failed: ok=%v val=%v", okFringe, vFringe)
	}
	if _, ok := t5.Get(pFringe.Masked()); ok {
		t.Fatalf("fringe still present after delete")
	}
	if t5.Size() != 0 {
		t.Fatalf("expected empty table after all deletes, got size %d", t5.Size())
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
	p := netip.MustParsePrefix("10.0.0.0/8")
	t1 := t0.InsertPersist(p, 123)

	pt1, v1, ok1 := t1.DeletePersist(p)
	pt2, v2, ok2 := t1.GetAndDeletePersist(p)

	if ok1 != ok2 || v1 != v2 || pt1.Size() != pt2.Size() {
		t.Fatalf("GetAndDeletePersist must mirror DeletePersist results")
	}
}

func TestWalkPersist_NilCallback_NoOp(t *testing.T) {
	t.Parallel()
	t0 := &Table[int]{}
	p := netip.MustParsePrefix("10.0.0.0/8")
	t1 := t0.InsertPersist(p, 1)
	pt := t1.WalkPersist(nil)
	if pt != t1 {
		t.Fatalf("nil callback must return original table reference")
	}
}

func TestWalkPersist_TransformsValues_StopsEarly(t *testing.T) {
	t.Parallel()
	t0 := &Table[int]{}

	p1 := netip.MustParsePrefix("10.0.0.0/8")
	p2 := netip.MustParsePrefix("192.168.0.0/16")
	p3 := netip.MustParsePrefix("2001:db8::/64")

	tbl := t0.InsertPersist(p1, 1).InsertPersist(p2, 2).InsertPersist(p3, 3)

	// Callback increments all values by 10, but stops after processing 2 entries
	count := 0
	cb := func(pt *Table[int], pfx netip.Prefix, v int) (*Table[int], bool) {
		count++
		pt2, _, _ := pt.ModifyPersist(pfx, func(old int, ok bool) (int, bool) {
			return old + 10, false
		})
		return pt2, count < 2
	}
	pt := tbl.WalkPersist(cb)

	if count != 2 {
		t.Fatalf("expected early stop after 2 items; got %d", count)
	}
	if pt.Size() != 3 {
		t.Fatalf("expected 3 entries after walk; got %d", pt.Size())
	}

	// Verify that exactly 2 values were incremented
	var incremented, original int
	if v, ok := pt.Get(p1.Masked()); ok {
		switch v {
		case 11:
			incremented++
		case 1:
			original++
		}
	}
	if v, ok := pt.Get(p2.Masked()); ok {
		switch v {
		case 12:
			incremented++
		case 2:
			original++
		}
	}
	if v, ok := pt.Get(p3.Masked()); ok {
		switch v {
		case 13:
			incremented++
		case 3:
			original++
		}
	}

	if incremented != 2 || original != 1 {
		t.Fatalf("expected 2 incremented and 1 original value; got %d incremented, %d original", incremented, original)
	}
}

func TestUnionPersist_SizesAndValues(t *testing.T) {
	t.Parallel()
	a := &Table[int]{}
	b := &Table[int]{}

	p1 := netip.MustParsePrefix("10.0.0.0/8")
	p2 := netip.MustParsePrefix("192.168.0.0/16")
	p3 := netip.MustParsePrefix("2001:db8::/64")
	p2dup := netip.MustParsePrefix("192.168.0.1/16") // same masked prefix as p2

	a1 := a.InsertPersist(p1, 1).InsertPersist(p2, 2)
	b1 := b.InsertPersist(p2dup, 22).InsertPersist(p3, 3)

	u := a1.UnionPersist(b1)

	if u.Size() != 3 {
		t.Fatalf("expected size 3 in union; got %d", u.Size())
	}

	// Verify all expected prefixes are present
	if v, ok := u.Get(p1.Masked()); !ok || v != 1 {
		t.Fatalf("p1 missing or wrong value in union: got %v ok=%v", v, ok)
	}
	if _, ok := u.Get(p2.Masked()); !ok {
		t.Fatalf("p2 missing in union")
	}
	if v, ok := u.Get(p3.Masked()); !ok || v != 3 {
		t.Fatalf("p3 missing or wrong value in union: got %v ok=%v", v, ok)
	}

	// Check value precedence on duplicate
	v2, _ := u.Get(p2.Masked())
	if v2 != 2 && v2 != 22 {
		t.Fatalf("unexpected value for duplicate key p2: got %d; expected 2 or 22", v2)
	}
}

// Cloning and isolation tests

func TestInsertPersist_ClonesValues(t *testing.T) {
	t.Parallel()
	t0 := &Table[clonerInt]{}
	p := netip.MustParsePrefix("10.0.0.0/8")

	// First insert: no clone yet
	t1 := t0.InsertPersist(p, clonerInt(100))
	if v, ok := t1.Get(p); !ok || v != clonerInt(100) {
		t.Fatalf("expected un-cloned value 100 after first insert; got %v ok=%v", v, ok)
	}

	// Second persist op duplicates structure and clones existing values into the new table
	q := netip.MustParsePrefix("192.168.0.0/16")
	t2 := t1.InsertPersist(q, clonerInt(1))
	if v, ok := t2.Get(p); !ok || v != clonerInt(1100) {
		t.Fatalf("expected cloned value 1100 in new table; got %v ok=%v", v, ok)
	}
	// Original table remains with original (uncloned) value
	if v, ok := t1.Get(p); !ok || v != clonerInt(100) {
		t.Fatalf("original table changed unexpectedly; got %v ok=%v", v, ok)
	}
}

// Insert returns newVal (uncloned). Update returns oldVal.
// Cloning occurs when values are carried to a new persistent table.
// TODO
func TestModifyPersist_ClonesValues(t *testing.T) {
	t.Parallel()
	t0 := &Table[clonerInt]{}
	p := netip.MustParsePrefix("172.16.0.0/12")

	// Insert via ModifyPersist -> returns (newVal, false), but stored value is un-cloned
	t1, newVal, deleted := t0.ModifyPersist(p, func(val clonerInt, ok bool) (clonerInt, bool) {
		if ok {
			t.Fatalf("expected missing prefix")
		}
		return clonerInt(300), false
	})
	if deleted || newVal != clonerInt(300) {
		t.Fatalf("insert path should return newVal=300, deleted=false; got %v, %v", newVal, deleted)
	}
	if v, ok := t1.Get(p); !ok || v != clonerInt(300) {
		t.Fatalf("stored value should be 300 after insert; got %v ok=%v", v, ok)
	}

	// Next persist operation clones existing values into the new table
	q := netip.MustParsePrefix("10.0.0.0/8")
	t2 := t1.InsertPersist(q, clonerInt(1))
	if v, ok := t2.Get(q); !ok || v != clonerInt(1) {
		t.Fatalf("stored value should be 1 after insert; got %v ok=%v", v, ok)
	}
	if v, ok := t2.Get(p); !ok || v != clonerInt(1300) {
		t.Fatalf("expected cloned value 1300 in new table; got %v ok=%v", v, ok)
	}

	// Update in-place: ModifyPersist returns oldVal, table gets new value (cloned on future persists)
	t3, oldVal, del2 := t2.ModifyPersist(p, func(val clonerInt, ok bool) (clonerInt, bool) {
		if !ok || val != clonerInt(2300) {
			t.Fatalf("expected existing 2300; got %v ok=%v", val, ok)
		}
		return clonerInt(400), false
	})
	if del2 || oldVal != clonerInt(2300) {
		t.Fatalf("update should return oldVal=2300, deleted=false; got %v, %v", oldVal, del2)
	}
	if v, ok := t3.Get(p); !ok || v != clonerInt(400) {
		t.Fatalf("after update, stored value should be 400; got %v ok=%v", v, ok)
	}
	if v, ok := t3.Get(q); !ok || v != clonerInt(1001) {
		t.Fatalf("expected cloned value 1001 in new table; got %v ok=%v", v, ok)
	}
}

// After walk-modify, a subsequent persist causes cloning into the new table.
func TestWalkPersist_ClonesModifiedValues(t *testing.T) {
	t.Parallel()
	t0 := &Table[clonerInt]{}

	p1 := netip.MustParsePrefix("10.0.0.0/8")
	p2 := netip.MustParsePrefix("192.168.0.0/16")

	// Build via two persists: after the 2nd insert, p1 has already been cloned: 10 -> 1010; p2 is 20.
	t1 := t0.InsertPersist(p1, clonerInt(10)).InsertPersist(p2, clonerInt(20))

	// Walk enumerates current table values; so v is 1010 (p1) or 20 (p2).
	t2 := t1.WalkPersist(func(pt *Table[clonerInt], pfx netip.Prefix, v clonerInt) (*Table[clonerInt], bool) {
		if v != clonerInt(1010) && v != clonerInt(20) {
			t.Fatalf("unexpected value in walk: %v", v)
		}
		// ModifyPersist operates on a persistent copy (values cloned before cb);
		// we store v+100 (no clone at insert boundary).
		pt2, _, _ := pt.ModifyPersist(pfx, func(old clonerInt, ok bool) (clonerInt, bool) {
			return v + 100, false
		})
		return pt2, true
	})

	if v, ok := t2.Get(p1); !ok || v != clonerInt(2110) {
		t.Fatalf("expected 2110 after walk; got %v ok=%v", v, ok)
	}
	if v, ok := t2.Get(p2); !ok || v != clonerInt(120) {
		t.Fatalf("expected 120 after walk; got %v ok=%v", v, ok)
	}

	q := netip.MustParsePrefix("2001:db8::/64")
	t3 := t2.InsertPersist(q, clonerInt(0))

	if v, ok := t3.Get(p1); !ok || v != clonerInt(2110) {
		t.Fatalf("expected 2110 after extra persist; got %v ok=%v", v, ok)
	}
	if v, ok := t3.Get(p2); !ok || v != clonerInt(120) {
		t.Fatalf("expected 120 after extra persist; got %v ok=%v", v, ok)
	}
}

// Pointer types with Cloner are not cloned on insertion; cloning happens
// when carrying values into a new persistent table.
func TestPersist_ClonerValues_CreatesNewInstances(t *testing.T) {
	t.Parallel()
	t0 := &Table[*clonerStruct]{}
	p := netip.MustParsePrefix("10.0.0.0/8")

	orig := &clonerStruct{value: 42}
	t1 := t0.InsertPersist(p, orig)

	// No clone on initial insertion: same pointer
	if v, ok := t1.Get(p); !ok || v != orig {
		t.Fatalf("expected same pointer after first insert; got %p ok=%v", v, ok)
	}

	// Next persist clones existing values into the new table
	q := netip.MustParsePrefix("192.168.0.0/16")
	t2 := t1.InsertPersist(q, &clonerStruct{value: 7})
	v2, ok := t2.Get(p)
	if !ok {
		t.Fatalf("expected value present in new table")
	}
	if v2 == orig {
		t.Fatalf("expected different pointer after cloning into new table, got same")
	}
	if v2.value != 1042 {
		t.Fatalf("expected cloned value 1042 in new table; got %v", v2.value)
	}

	// Changing original must not affect the cloned copy in t2
	orig.value = 999
	if v2.value == 999 {
		t.Fatalf("cloned value in new table should be isolated from original")
	}
}

// Test that non-cloner types don't get cloned - pointer identity is preserved
func TestPersist_NonClonerValues_PointerIdentityPreserved(t *testing.T) {
	t.Parallel()

	t0 := &Table[*nonClonerStruct]{}
	p := netip.MustParsePrefix("10.0.0.0/8")

	originalPtr := &nonClonerStruct{value: 42}

	t1 := t0.InsertPersist(p, originalPtr)

	// Should be the exact same pointer (no cloning)
	if v, ok := t1.Get(p); !ok || v != originalPtr {
		t.Fatalf("expected same pointer, got different pointer")
	}

	// Modify through the original pointer
	originalPtr.value = 100

	// Change should be visible in the table (proves no isolation)
	if v, ok := t1.Get(p); !ok || v.value != 100 {
		t.Fatalf("expected value 100 after modification, got %v", v.value)
	}

	// Create another table with ModifyPersist
	t2, returnedPtr, _ := t1.ModifyPersist(p, func(val *nonClonerStruct, ok bool) (*nonClonerStruct, bool) {
		if !ok || val != originalPtr {
			t.Fatalf("expected original pointer in callback")
		}
		return originalPtr, false // Return same pointer
	})

	// For update, ModifyPersist returns old value (which is the same pointer)
	if returnedPtr != originalPtr {
		t.Fatalf("ModifyPersist should return original pointer for update")
	}

	// Both tables should have the same pointer
	v1, _ := t1.Get(p)
	v2, _ := t2.Get(p)
	if v1 != v2 || v1 != originalPtr {
		t.Fatalf("all tables should reference the same pointer")
	}

	// Modification affects all tables (no isolation)
	originalPtr.value = 200
	if v1.value != 200 || v2.value != 200 {
		t.Fatalf("modification should affect all tables")
	}
}
