package bart

import (
	"net/netip"
	"testing"
)

// Fast: Size4/Size6, ModifyPersist, WalkPersist
func TestFast_Size_ModifyPersist_WalkPersist(t *testing.T) {
	t.Parallel()

	// v4 + v6 anlegen
	f := new(Fast[int])
	f.Insert(mpp("10.0.0.0/8"), 8)
	f.Insert(mpp("10.1.0.0/16"), 16)
	f.Insert(mpp("2001:db8::/32"), 32)

	// Size4 / Size6
	if got := f.Size4(); got != 2 {
		t.Fatalf("Fast.Size4() = %d, want 2", got)
	}
	if got := f.Size6(); got != 1 {
		t.Fatalf("Fast.Size6() = %d, want 1", got)
	}

	// ModifyPersist: update, returns oldVal
	f2, v2, del2 := f.ModifyPersist(mpp("10.1.0.0/16"), func(v int, ok bool) (int, bool) {
		if !ok || v != 16 {
			t.Fatalf("update path: ok=%v v=%d, want ok=true v=16", ok, v)
		}
		return v * 10, false
	})
	if del2 || v2 != 16 { // old value, updated value, not new value!
		t.Fatalf("update result: del=%v v=%d, want del=false v=160", del2, v2)
	}
	if f2 == f {
		t.Fatalf("ModifyPersist(update) must return a new instance")
	}

	// ModifyPersist: insert
	f3, v3, del3 := f2.ModifyPersist(mpp("172.16.0.0/12"), func(v int, ok bool) (int, bool) {
		if ok {
			t.Fatal("insert path: unexpectedly exists")
		}
		return 777, false
	})
	if del3 || v3 != 777 {
		t.Fatalf("insert result: del=%v v=%d, want del=false v=777", del3, v3)
	}

	// ModifyPersist: delete
	f4, _, del4 := f3.ModifyPersist(mpp("10.0.0.0/8"), func(v int, ok bool) (int, bool) {
		if !ok || v != 8 {
			t.Fatalf("delete path: ok=%v v=%d, want ok=true v=8", ok, v)
		}
		return 0, true
	})
	if !del4 {
		t.Fatalf("delete result: del=false, want true")
	}

	// WalkPersist: no-change (early exit) => equal trie but different instance
	visits := 0
	f5 := f4.WalkPersist(func(cur *Fast[int], pfx netip.Prefix, v int) (*Fast[int], bool) {
		visits++
		return cur, visits < 2
	})
	if f5 != f4 {
		t.Fatalf("WalkPersist(no-change) must return same instance")
	}
	if visits < 1 {
		t.Fatalf("WalkPersist(no-change) visits=%d, want >=1", visits)
	}

	// WalkPersist: mit Änderung => neue Instanz
	changed := false
	f6 := f4.WalkPersist(func(cur *Fast[int], pfx netip.Prefix, v int) (*Fast[int], bool) {
		if !changed && pfx.String() == "10.1.0.0/16" {
			cur = cur.InsertPersist(pfx, v+1)
			changed = true
		}
		return cur, true
	})
	if !changed {
		t.Fatalf("WalkPersist(change) did not hit modification branch")
	}
	if f6 == f4 {
		t.Fatalf("WalkPersist(change) must return a new instance")
	}
}
