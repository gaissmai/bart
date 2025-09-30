package bart

import (
	"testing"
)

// Fast: Size4/Size6, ModifyPersist
func TestFast_Size_ModifyPersist(t *testing.T) {
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

	f2 := f.ModifyPersist(mpp("10.1.0.0/16"), func(v int, ok bool) (int, bool) {
		if !ok || v != 16 {
			t.Fatalf("update path: ok=%v v=%d, want ok=true v=16", ok, v)
		}
		return v * 10, false
	})
	if f2 == f {
		t.Fatalf("ModifyPersist(update) must return a new instance")
	}

	// ModifyPersist: insert
	f3 := f2.ModifyPersist(mpp("172.16.0.0/12"), func(v int, ok bool) (int, bool) {
		if ok {
			t.Fatal("insert path: unexpectedly exists")
		}
		return 777, false
	})

	// ModifyPersist: delete
	f4 := f3.ModifyPersist(mpp("10.0.0.0/8"), func(v int, ok bool) (int, bool) {
		if !ok || v != 8 {
			t.Fatalf("delete path: ok=%v v=%d, want ok=true v=8", ok, v)
		}
		return 0, true
	})

	if _, ok := f4.Get(mpp("10.0.0.0/8")); ok {
		t.Fatal("ModifyPersist(delete): prefix still present after delete")
	}
}
