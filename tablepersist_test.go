package bart

import (
	"net/netip"
	"testing"
)

// testVal is a sample value type.
// We use *testVal as the generic payload V, which is a pointer type,
// so it must implement Cloner[*testVal].
type testVal struct {
	Data string
}

// Clone ensures deep copying for use with ...Persist.
func (v *testVal) Clone() *testVal {
	if v == nil {
		return nil
	}
	return &testVal{Data: v.Data}
}

func TestInsertPersistNotAliased(t *testing.T) {
	t.Parallel()

	tests := []struct {
		pfx      netip.Prefix
		initial  string
		override string // new value to insert via persist
	}{
		{mpp("10.0.0.0/8"), "A", "AA"},
		{mpp("192.168.0.0/16"), "B", "BB"},
		{mpp("2001:db8::/32"), "C", "CC"},
		{mpp("fd00::/8"), "D", "DD"},
	}

	// setup
	orig := new(Table[*testVal])
	for _, tc := range tests {
		orig.Insert(tc.pfx, &testVal{Data: tc.initial})
	}

	clone := orig
	for _, tc := range tests {
		clone := clone.InsertPersist(tc.pfx, &testVal{Data: tc.override})

		// mutate clone's value to ensure it's not aliased
		v2, _ := clone.Get(tc.pfx)
		v2.Data = "MUTATED"

		// original must be unchanged
		v1, _ := orig.Get(tc.pfx)
		if v1.Data != tc.initial {
			t.Errorf("InsertPersist: original table modified for prefix %s: want %q, got %q", tc.pfx, tc.initial, v1.Data)
		}

		// cloned table should have the mutated value
		if v2.Data != "MUTATED" {
			t.Errorf("InsertPersist: mutated value not reflected for prefix %s", tc.pfx)
		}

		// ensure no aliasing
		if v1 == v2 {
			t.Errorf("InsertPersist: pointer aliasing detected for prefix %s", tc.pfx)
		}
	}
}

func TestUpdatePersistNotAliased(t *testing.T) {
	t.Parallel()

	tests := []struct {
		pfx     netip.Prefix
		initial string
		mutated string
	}{
		{mpp("10.0.0.0/8"), "A", "AA"},
		{mpp("192.168.0.0/16"), "B", "BB"},
		{mpp("2001:db8::/32"), "C", "CC"},
		{mpp("fd00::/8"), "D", "DD"},
	}

	// setup
	orig := new(Table[*testVal])
	for _, tc := range tests {
		orig.Insert(tc.pfx, &testVal{Data: tc.initial})
	}

	var newVal *testVal
	clone := orig
	for _, tc := range tests {
		clone, newVal = clone.UpdatePersist(tc.pfx, func(val *testVal, ok bool) *testVal {
			if !ok {
				t.Fatalf("UpdatePersist: prefix %s not present", tc.pfx)
			}
			return &testVal{Data: tc.mutated}
		})

		// Mutate newVal to test for aliasing
		newVal.Data = "changed"

		vOrig, _ := orig.Get(tc.pfx)
		vClone, _ := clone.Get(tc.pfx)
		if vOrig.Data != tc.initial {
			t.Errorf("UpdatePersist: original modified for %s: got=%q want=%q", tc.pfx, vOrig.Data, tc.initial)
		}
		if vClone.Data != "changed" {
			t.Errorf("UpdatePersist: clone not correctly updated for %s: got=%q want=%q", tc.pfx, vClone.Data, "changed")
		}
		if vOrig == vClone {
			t.Errorf("UpdatePersist: aliasing detected for %s", tc.pfx)
		}
	}
}

func TestDeletePersistNotAliased(t *testing.T) {
	t.Parallel()

	tests := []struct {
		pfx netip.Prefix
	}{
		{mpp("10.0.0.0/8")},
		{mpp("192.168.42.0/24")},
		{mpp("2001:db8::/32")},
		{mpp("fd00::/8")},
	}

	// setup
	orig := new(Table[*testVal])
	for _, tc := range tests {
		orig.Insert(tc.pfx, &testVal{Data: "payload"})
	}

	clone := orig
	for _, tc := range tests {
		clone = clone.DeletePersist(tc.pfx)

		// Deleted prefix should be absent in clone
		_, ok := clone.Get(tc.pfx)
		if ok {
			t.Errorf("DeletePersist: prefix %s should've been deleted in clone, but it's still there", tc.pfx)
		}

		// Original table must be unchanged
		if v, ok := orig.Get(tc.pfx); !ok || v.Data != "payload" {
			t.Errorf("DeletePersist: original affected for %s", tc.pfx)
		}
	}
}
