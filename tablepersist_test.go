// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"testing"
)

// defined in cloner_test.go
// ---- Test helper types ----

// // routeEntry represents a realistic routing table entry for testing persist operations
// type routeEntry struct {
// 	nextHop    netip.Addr
// 	exitIF     string
// 	attributes map[string]int
// }
//
// // Clone implements Cloner[*routeEntry] for deep cloning of routing entries
// func (r *routeEntry) Clone() *routeEntry {
// 	if r == nil {
// 		return nil
// 	}
//
// 	clone := &routeEntry{
// 		nextHop:    r.nextHop,
// 		exitIF:     r.exitIF,
// 		attributes: make(map[string]int, len(r.attributes)),
// 	}
//
// 	// Deep clone the attributes map
// 	for k, v := range r.attributes {
// 		clone.attributes[k] = v
// 	}
//
// 	return clone
// }
//
// // routeEntryNonCloner is the same struct but without Clone method for testing non-cloner behavior
// type routeEntryNonCloner struct {
// 	nextHop    netip.Addr
// 	exitIF     string
// 	attributes map[string]int
// }

// ---- Test data helpers ----

func newRoute(nextHop, exitIF string, metric int) *routeEntry {
	return &routeEntry{
		nextHop:    netip.MustParseAddr(nextHop),
		exitIF:     exitIF,
		attributes: map[string]int{"metric": metric, "preference": 100},
	}
}

func newRouteNonCloner(nextHop, exitIF string, metric int) *routeEntryNonCloner {
	return &routeEntryNonCloner{
		nextHop:    netip.MustParseAddr(nextHop),
		exitIF:     exitIF,
		attributes: map[string]int{"metric": metric, "preference": 100},
	}
}

// ---- Basic persistence tests ----

func TestInsertPersist_InvalidPrefix_NoChange(t *testing.T) {
	t.Parallel()
	t0 := &Table[*routeEntry]{}

	invalid := netip.Prefix{} // not valid; IsValid() == false
	route := newRoute("10.0.0.1", "eth0", 100)
	pt := t0.InsertPersist(invalid, route)

	if t0 != pt {
		t.Fatalf("expected original table to be returned for invalid prefix")
	}

	if pt.Size() != 0 {
		t.Fatalf("expected empty table after invalid insert, got size %d", pt.Size())
	}
}

func TestInsertPersist_CanonicalizesMasked_OverrideAndSize(t *testing.T) {
	t.Parallel()
	t0 := &Table[*routeEntry]{}

	// Insert with host bits set; method should mask to .0/24
	p1 := netip.MustParsePrefix("192.168.1.123/24")
	route1 := newRoute("192.168.1.1", "eth0", 100)
	pt1 := t0.InsertPersist(p1, route1)

	masked := p1.Masked()
	if v, ok := pt1.Get(masked); !ok {
		t.Fatalf("expected route at masked prefix %v", masked)
	} else if v.nextHop != route1.nextHop || v.exitIF != route1.exitIF {
		t.Fatalf("route values should match inserted route")
	}

	// Override same logical prefix with different route
	route2 := newRoute("192.168.1.2", "eth1", 200)
	pt2 := pt1.InsertPersist(netip.MustParsePrefix("192.168.1.1/24"), route2)
	if v, ok := pt2.Get(masked); !ok {
		t.Fatalf("expected route at %v after override", masked)
	} else if v.nextHop != route2.nextHop || v.exitIF != route2.exitIF {
		t.Fatalf("route should be overridden with new values")
	}

	if pt2.Size() != 1 {
		t.Fatalf("expected size 1 after override, got %d", pt2.Size())
	}
}

func TestInsertPersist_IPv6(t *testing.T) {
	t.Parallel()
	t0 := &Table[*routeEntry]{}
	p6 := netip.MustParsePrefix("2001:db8::1/64")
	route := &routeEntry{
		nextHop:    netip.MustParseAddr("2001:db8::1"),
		exitIF:     "eth0",
		attributes: map[string]int{"metric": 100, "preference": 50},
	}
	pt := t0.InsertPersist(p6, route)

	want := p6.Masked()
	if v, ok := pt.Get(want); !ok {
		t.Fatalf("expected IPv6 route at %v", want)
	} else if !v.nextHop.Is6() || v.nextHop != route.nextHop {
		t.Fatalf("IPv6 route nextHop should match")
	}
}

// ---- Comprehensive tests for ModifyPersist ----

func TestModifyPersist_Insert_Update_Delete_Paths(t *testing.T) {
	t.Parallel()
	t0 := &Table[*routeEntry]{}
	p := netip.MustParsePrefix("172.16.0.0/12")

	// Insert when missing (del=false) - returns (newVal, false)
	route1 := newRoute("172.16.0.1", "eth0", 111)
	t1 := t0.ModifyPersist(p, func(val *routeEntry, ok bool) (*routeEntry, bool) {
		if ok {
			t.Fatalf("expected ok=false for missing")
		}
		return route1, false
	})
	if v, ok := t1.Get(p.Masked()); !ok {
		t.Fatalf("insert path failed")
	} else if v.attributes["metric"] != 111 {
		t.Fatalf("inserted route metric should be 111")
	}
	if t1.Size() != 1 {
		t.Fatalf("expected size 1 after insert, got %d", t1.Size())
	}

	// Update existing (del=false) - returns (oldVal, false)
	route2 := newRoute("172.16.0.2", "eth1", 222)
	t2 := t1.ModifyPersist(p, func(val *routeEntry, ok bool) (*routeEntry, bool) {
		if !ok {
			t.Fatalf("expected existing route")
		}
		if val.attributes["metric"] != 111 {
			t.Fatalf("expected existing route with metric 111, got %d", val.attributes["metric"])
		}
		return route2, false
	})
	if v, ok := t2.Get(p.Masked()); !ok {
		t.Fatalf("route should exist after update")
	} else if v.attributes["metric"] != 222 { // Table contains NEW value
		t.Fatalf("route not updated to metric 222, got %d", v.attributes["metric"])
	}
	if t2.Size() != 1 {
		t.Fatalf("expected size 1 after update, got %d", t2.Size())
	}

	// Delete existing (del=true) - returns (oldVal, true)
	t3 := t2.ModifyPersist(p, func(val *routeEntry, ok bool) (*routeEntry, bool) {
		if !ok {
			t.Fatalf("expected existing route")
		}
		if val.attributes["metric"] != 222 {
			t.Fatalf("expected existing route with metric 222")
		}
		return val, true
	})
	if _, ok := t3.Get(p.Masked()); ok {
		t.Fatalf("expected prefix to be removed")
	}
	if t3.Size() != 0 {
		t.Fatalf("expected empty table after delete, got size %d", t3.Size())
	}
}

func TestModifyPersist_MissingAndDelTrue_NoOp(t *testing.T) {
	t.Parallel()
	t0 := &Table[*routeEntry]{}
	p := netip.MustParsePrefix("10.10.10.0/24")
	t1 := t0.ModifyPersist(p, func(val *routeEntry, ok bool) (*routeEntry, bool) {
		return nil, true
	})
	if t1.Size() != 0 {
		t.Fatalf("expected no entries after no-op, got size %d", t1.Size())
	}
}

func TestModifyPersist_InvalidPrefix_ReturnsOriginal(t *testing.T) {
	t.Parallel()
	t0 := &Table[*routeEntry]{}
	pt := t0.ModifyPersist(netip.Prefix{}, func(val *routeEntry, ok bool) (*routeEntry, bool) {
		return newRoute("10.0.0.1", "eth0", 100), false
	})
	if pt != t0 {
		t.Fatalf("expected original table for invalid prefix")
	}
}

// ---- DeletePersist ----

func TestDeletePersist_Workflow(t *testing.T) {
	t.Parallel()
	t0 := &Table[*routeEntry]{}

	pLeaf := netip.MustParsePrefix("192.0.2.0/24")
	pFringe := netip.MustParsePrefix("198.51.100.0/20")
	routeLeaf := newRoute("192.0.2.1", "eth0", 10)
	routeFringe := newRoute("198.51.100.1", "ppp0", 20)

	t1 := t0.InsertPersist(pLeaf, routeLeaf)
	t2 := t1.InsertPersist(pFringe, routeFringe)

	if t2.Size() != 2 {
		t.Fatalf("expected size 2 after inserts, got %d", t2.Size())
	}

	// Delete non-existent should be no-op
	t3 := t2.DeletePersist(netip.MustParsePrefix("203.0.113.0/24"))
	if t3.Size() != 2 {
		t.Fatalf("delete of missing prefix should be no-op")
	}

	// Delete leaf
	t4 := t3.DeletePersist(pLeaf)
	if _, ok := t4.Get(pLeaf.Masked()); ok {
		t.Fatalf("leaf still present after delete")
	}
	if t4.Size() != 1 {
		t.Fatalf("expected size 1 after first delete, got %d", t4.Size())
	}

	// Delete fringe
	t5 := t4.DeletePersist(pFringe)
	if _, ok := t5.Get(pFringe.Masked()); ok {
		t.Fatalf("fringe still present after delete")
	}
	if t5.Size() != 0 {
		t.Fatalf("expected empty table after all deletes, got size %d", t5.Size())
	}
}

func TestDeletePersist_InvalidPrefix_ReturnsOriginal(t *testing.T) {
	t.Parallel()
	t0 := &Table[*routeEntry]{}
	pt := t0.DeletePersist(netip.Prefix{})
	if pt != t0 {
		t.Fatalf("expected original table for invalid prefix")
	}
}

// ---- UnionPersist ----

func TestUnionPersist_SizesAndValues(t *testing.T) {
	t.Parallel()
	a := &Table[*routeEntry]{}
	b := &Table[*routeEntry]{}

	p1 := netip.MustParsePrefix("10.0.0.0/8")
	p2 := netip.MustParsePrefix("192.168.0.0/16")
	p3 := netip.MustParsePrefix("2001:db8::/64")
	p2dup := netip.MustParsePrefix("192.168.0.1/16") // same masked prefix as p2

	route1 := newRoute("10.0.0.1", "eth0", 1)
	route2 := newRoute("192.168.0.1", "eth1", 2)
	route2dup := newRoute("192.168.0.2", "eth2", 22) // different route for same prefix
	route3 := &routeEntry{
		nextHop:    netip.MustParseAddr("2001:db8::1"),
		exitIF:     "eth3",
		attributes: map[string]int{"metric": 3, "preference": 100},
	}

	a1 := a.InsertPersist(p1, route1).InsertPersist(p2, route2)
	b1 := b.InsertPersist(p2dup, route2dup).InsertPersist(p3, route3)

	u := a1.UnionPersist(b1)

	if u.Size() != 3 {
		t.Fatalf("expected size 3 in union; got %d", u.Size())
	}

	// Verify all expected prefixes are present
	if v, ok := u.Get(p1.Masked()); !ok {
		t.Fatalf("p1 missing in union")
	} else if v.attributes["metric"] != 1 {
		t.Fatalf("p1 should have metric 1, got %d", v.attributes["metric"])
	}

	if v, ok := u.Get(p2.Masked()); !ok {
		t.Fatalf("p2 missing in union")
	} else {
		// UnionPersist has right-side precedence: b1 wins over a1 on duplicates
		if v.attributes["metric"] != 22 {
			t.Fatalf("duplicate key should keep right value (metric 22), got %d", v.attributes["metric"])
		}
	}

	if v, ok := u.Get(p3.Masked()); !ok {
		t.Fatalf("p3 missing in union")
	} else if v.attributes["metric"] != 3 {
		t.Fatalf("p3 should have metric 3, got %d", v.attributes["metric"])
	}
}

// ---- Cloning and isolation tests ----

func TestInsertPersist_ClonesValues(t *testing.T) {
	t.Parallel()
	t0 := &Table[*routeEntry]{}
	p := netip.MustParsePrefix("10.0.0.0/8")

	// First insert: no clone yet
	route1 := newRoute("10.0.0.1", "eth0", 100)
	t1 := t0.InsertPersist(p, route1)
	if v, ok := t1.Get(p); !ok {
		t.Fatalf("expected route after first insert")
	} else if v.attributes["metric"] != 100 {
		t.Fatalf("expected un-cloned metric 100 after first insert; got %d", v.attributes["metric"])
	}

	// Second persist op duplicates structure and clones existing values into the new table
	q := netip.MustParsePrefix("192.168.0.0/16")
	route2 := newRoute("192.168.0.1", "eth1", 1)
	t2 := t1.InsertPersist(q, route2)
	if v, ok := t2.Get(p); !ok {
		t.Fatalf("expected cloned route in new table")
	} else if v.attributes["metric"] != 100 {
		t.Fatalf("expected cloned metric 100 in new table; got %d", v.attributes["metric"])
	}

	// Verify the routes are different instances (cloned)
	orig, _ := t1.Get(p)
	cloned, _ := t2.Get(p)
	if orig == cloned {
		t.Fatalf("routes should be different instances after cloning")
	}

	// Modify cloned route attributes - should not affect original
	cloned.attributes["metric"] = 999
	if orig.attributes["metric"] != 100 {
		t.Fatalf("modifying cloned route affected original")
	}
}

func TestModifyPersist_ClonesValues(t *testing.T) {
	t.Parallel()
	t0 := &Table[*routeEntry]{}
	p := netip.MustParsePrefix("172.16.0.0/12")

	// Insert via ModifyPersist -> returns (newVal, false), but stored value is un-cloned
	route1 := newRoute("172.16.0.1", "eth0", 300)
	t1 := t0.ModifyPersist(p, func(val *routeEntry, ok bool) (*routeEntry, bool) {
		if ok {
			t.Fatalf("expected missing prefix")
		}
		return route1, false
	})
	if v, ok := t1.Get(p); !ok {
		t.Fatalf("stored route should exist after insert")
	} else if v.attributes["metric"] != 300 {
		t.Fatalf("stored route should have metric 300 after insert; got %d", v.attributes["metric"])
	}

	// Next persist operation clones existing values into the new table
	q := netip.MustParsePrefix("10.0.0.0/8")
	route2 := newRoute("10.0.0.1", "eth1", 1)
	t2 := t1.InsertPersist(q, route2)
	if v, ok := t2.Get(q); !ok {
		t.Fatalf("new route should exist")
	} else if v.attributes["metric"] != 1 {
		t.Fatalf("stored value should be 1 after insert; got %d", v.attributes["metric"])
	}

	// Original route should be cloned
	if v, ok := t2.Get(p); !ok {
		t.Fatalf("expected cloned route in new table")
	} else if v.attributes["metric"] != 300 {
		t.Fatalf("expected cloned metric 300 in new table; got %d", v.attributes["metric"])
	}

	// Update in-place: ModifyPersist returns oldVal, table gets new value (cloned on future persists)
	route3 := newRoute("172.16.0.2", "eth2", 400)
	t3 := t2.ModifyPersist(p, func(val *routeEntry, ok bool) (*routeEntry, bool) {
		if !ok {
			t.Fatalf("expected existing route")
		}
		if val.attributes["metric"] != 300 {
			t.Fatalf("expected existing metric 300; got %d", val.attributes["metric"])
		}
		return route3, false
	})
	if v, ok := t3.Get(p); !ok {
		t.Fatalf("updated route should exist")
	} else if v.attributes["metric"] != 400 {
		t.Fatalf("after update, stored route should have metric 400; got %d", v.attributes["metric"])
	}
	if v, ok := t3.Get(q); !ok {
		t.Fatalf("other route should still exist")
	} else if v.attributes["metric"] != 1 {
		t.Fatalf("other route should have metric 1; got %d", v.attributes["metric"])
	}
}

// ---- Cloner vs Non-cloner behavior ----

func TestPersist_ClonerValues_CreatesNewInstances(t *testing.T) {
	t.Parallel()
	t0 := &Table[*routeEntry]{}
	p := netip.MustParsePrefix("10.0.0.0/8")

	orig := newRoute("10.0.0.1", "eth0", 42)
	t1 := t0.InsertPersist(p, orig)

	// No clone on initial insertion: same pointer
	if v, ok := t1.Get(p); !ok || v != orig {
		t.Fatalf("expected same pointer after first insert")
	}

	// Next persist clones existing values into the new table
	q := netip.MustParsePrefix("192.168.0.0/16")
	newRouteVal := newRoute("192.168.0.1", "eth1", 7)
	t2 := t1.InsertPersist(q, newRouteVal)
	v2, ok := t2.Get(p)
	if !ok {
		t.Fatalf("expected route present in new table")
	}
	if v2 == orig {
		t.Fatalf("expected different pointer after cloning into new table")
	}
	if v2.attributes["metric"] != 42 {
		t.Fatalf("expected cloned metric 42 in new table; got %d", v2.attributes["metric"])
	}

	// Changing original must not affect the cloned copy in t2
	orig.attributes["metric"] = 999
	if v2.attributes["metric"] == 999 {
		t.Fatalf("cloned value in new table should be isolated from original")
	}
}

// Additional comprehensive non-cloner tests
func TestNonClonerInsertPersist_MultipleRoutes(t *testing.T) {
	t.Parallel()

	t0 := &Table[*routeEntryNonCloner]{}
	p1 := netip.MustParsePrefix("172.16.0.0/12")
	p2 := netip.MustParsePrefix("192.168.0.0/16")

	route1 := newRouteNonCloner("10.0.0.1", "eth0", 100)
	route2 := newRouteNonCloner("10.0.0.2", "eth1", 200)

	// Insert first route
	t1 := t0.InsertPersist(p1, route1)

	// Insert second route - should preserve pointer identity for first route
	t2 := t1.InsertPersist(p2, route2)

	// Get routes from new table
	gotRoute1, ok1 := t2.Get(p1)
	gotRoute2, ok2 := t2.Get(p2)

	if !ok1 || !ok2 {
		t.Fatal("both routes should exist in new table")
	}

	// Should be same instances (no cloning for non-cloner)
	if gotRoute1 != route1 || gotRoute2 != route2 {
		t.Error("non-cloner routes should preserve pointer identity")
	}

	// Verify shared state - modifications affect all references
	route1.attributes["metric"] = 999
	if gotRoute1.attributes["metric"] != 999 {
		t.Error("modification should be visible through all references")
	}
}

func TestNonClonerModifyPersist_PointerPreservation(t *testing.T) {
	t.Parallel()

	t0 := &Table[*routeEntryNonCloner]{}
	p := netip.MustParsePrefix("203.0.113.0/24")

	// Insert with non-cloner helper
	original := newRouteNonCloner("203.0.113.1", "wan0", 500)
	t1 := t0.InsertPersist(p, original)

	// Modify without changing the route instance
	t2 := t1.ModifyPersist(p, func(old *routeEntryNonCloner, found bool) (*routeEntryNonCloner, bool) {
		if !found || old != original {
			t.Error("should receive original pointer in callback")
		}
		// Return same instance with modified attributes
		old.attributes["preference"] = 200
		return old, false
	})

	// Both tables should reference the same instance
	val1, _ := t1.Get(p)
	val2, _ := t2.Get(p)

	if val1 != original || val2 != original {
		t.Error("all table references should point to same instance")
	}

	// Modifications are visible everywhere due to shared references
	if val1.attributes["preference"] != 200 || val2.attributes["preference"] != 200 {
		t.Error("modification should be visible in all table references")
	}
}

func TestNonClonerUnionPersist_SharedReferences(t *testing.T) {
	t.Parallel()

	t1 := &Table[*routeEntryNonCloner]{}
	t2 := &Table[*routeEntryNonCloner]{}

	p1 := netip.MustParsePrefix("10.1.0.0/16")
	p2 := netip.MustParsePrefix("10.2.0.0/16")
	p3 := netip.MustParsePrefix("10.2.0.1/16") // same masked prefix as p2

	route1 := newRouteNonCloner("10.1.0.1", "eth0", 1)
	route2 := newRouteNonCloner("10.2.0.1", "eth1", 2)
	route3 := newRouteNonCloner("10.2.0.2", "eth2", 3) // different route for same prefix

	t1 = t1.InsertPersist(p1, route1).InsertPersist(p2, route2)
	t2 = t2.InsertPersist(p3, route3)

	union := t1.UnionPersist(t2)

	// Verify union preserves pointer identity for non-cloners
	if v, ok := union.Get(p1); !ok || v != route1 {
		t.Error("union should preserve pointer identity for route1")
	}

	// Right-side should win for duplicates, preserving pointer identity
	if v, ok := union.Get(p2.Masked()); !ok || v != route3 {
		t.Error("union should use right-side route (route3) for duplicate prefix")
	}

	// Modifications should be visible through union references
	route1.attributes["metric"] = 999
	if v, ok := union.Get(p1); !ok || v.attributes["metric"] != 999 {
		t.Error("modification should be visible in union table")
	}
}
