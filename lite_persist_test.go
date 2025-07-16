package bart

import (
	"net/netip"
	"testing"
)

func TestInsertPersistLite(t *testing.T) {
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
	orig := new(Lite)
	for _, tc := range tests {
		orig.Insert(tc.pfx)
	}

	clone := orig
	for _, tc := range tests {
		clone := clone.InsertPersist(tc.pfx)

		// both tables must have the pfx
		ok1 := orig.Exists(tc.pfx)
		ok2 := clone.Exists(tc.pfx)

		if !ok1 {
			t.Errorf("InsertPersist: original table missing prefix %s", tc.pfx)
		}

		if !ok2 {
			t.Errorf("InsertPersist: cloned table missing prefix %s", tc.pfx)
		}
	}
}

func TestDeletePersistLite(t *testing.T) {
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
	orig := new(Lite)
	for _, tc := range tests {
		orig.Insert(tc.pfx)
	}

	clone := orig
	for _, tc := range tests {
		clone = clone.DeletePersist(tc.pfx)

		// test for existence
		if ok := orig.Exists(tc.pfx); !ok {
			t.Errorf("DeletePersist: original table missing prefix %s", tc.pfx)
		}

		// test for absence
		if ok := clone.Exists(tc.pfx); ok {
			t.Errorf("DeletePersist: prefix %s not deleted in cloned table", tc.pfx)
		}
	}
}
