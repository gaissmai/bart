package bart

import (
	"math/rand/v2"
	"testing"
)

// testVal as a simple value type.
type testVal struct {
	Data int
}

// Clone ensures deep copying for use with ...Persist.
//
// We use *testVal as the generic payload V,
// which is a pointer type, so it must implement bart.Cloner[V]
func (v *testVal) Clone() *testVal {
	if v == nil {
		return nil
	}
	return &testVal{Data: v.Data}
}

// ########## Table[V]  ##############

func TestInsertPersistTable(t *testing.T) {
	t.Parallel()

	// setup
	const n = 10_000

	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	orig := new(Table[*testVal])
	for _, pfx := range pfxs {
		orig.Insert(pfx, &testVal{Data: 1})
	}

	clone := orig
	for _, pfx := range pfxs {
		clone := clone.InsertPersist(pfx, &testVal{Data: 2})

		// mutate clone's value to ensure it's not aliased
		v2, _ := clone.Get(pfx)
		v2.Data = 3

		// original must be unchanged
		v1, _ := orig.Get(pfx)
		if v1.Data != 1 {
			t.Errorf("InsertPersist: original table modified for prefix %s: want %q, got %q", pfx, 1, v1.Data)
		}

		// cloned table should have the mutated value
		if v2.Data != 3 {
			t.Errorf("InsertPersist: mutated value not reflected for prefix %s", pfx)
		}

		// ensure no aliasing
		if v1 == v2 {
			t.Errorf("InsertPersist: pointer aliasing detected for prefix %s", pfx)
		}
	}
}

func TestUpdatePersistTable(t *testing.T) {
	t.Parallel()

	// setup
	const n = 10_000

	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	orig := new(Table[*testVal])
	for _, pfx := range pfxs {
		orig.Insert(pfx, &testVal{Data: 1})
	}

	var newVal *testVal
	clone := orig
	for _, pfx := range pfxs {
		clone, newVal = clone.UpdatePersist(pfx, func(val *testVal, ok bool) *testVal {
			if !ok {
				t.Fatalf("UpdatePersist: prefix %s not present", pfx)
			}
			return &testVal{Data: 2}
		})

		// Mutate newVal to test for aliasing
		newVal.Data = 3

		v1, _ := orig.Get(pfx)
		v2, _ := clone.Get(pfx)

		if v1.Data != 1 {
			t.Errorf("UpdatePersist: original modified for %s: got=%q want=%q", pfx, v1.Data, 1)
		}

		if v2.Data != 3 {
			t.Errorf("UpdatePersist: clone not correctly updated for %s: got=%q want=%q", pfx, v2.Data, 3)
		}

		if v1 == v2 {
			t.Errorf("UpdatePersist: aliasing detected for %s", pfx)
		}
	}
}

func TestDeletePersistTable(t *testing.T) {
	t.Parallel()

	// setup
	const n = 10_000

	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	orig := new(Table[*testVal])
	for _, pfx := range pfxs {
		orig.Insert(pfx, &testVal{Data: 1})
	}

	clone := orig
	for _, pfx := range pfxs {
		clone, _, _ = clone.DeletePersist(pfx)

		// Deleted prefix should be absent in clone
		_, ok := clone.Get(pfx)
		if ok {
			t.Errorf("DeletePersist: prefix %s should've been deleted in clone, but it's still there", pfx)
		}

		// Original table must be unchanged
		if v, ok := orig.Get(pfx); !ok || v.Data != 1 {
			t.Errorf("DeletePersist: original affected for %s", pfx)
		}
	}
}

// ########## Lite ##############

func TestInsertPersistLite(t *testing.T) {
	t.Parallel()

	// setup
	const n = 10_000
	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	orig := new(Lite)
	for _, pfx := range pfxs {
		orig.Insert(pfx)
	}

	clone := orig
	for _, pfx := range pfxs {
		clone := clone.InsertPersist(pfx)

		// both tables must have the pfx
		ok1 := orig.Exists(pfx)
		ok2 := clone.Exists(pfx)

		if !ok1 {
			t.Errorf("InsertPersist: original table missing prefix %s", pfx)
		}

		if !ok2 {
			t.Errorf("InsertPersist: cloned table missing prefix %s", pfx)
		}

		size1 := orig.Size()
		size2 := clone.Size()

		if size1 != n {
			t.Errorf("InsertPersist: original table has unexptected size, want %d, got %d", n, size1)
		}

		if size2 != n {
			t.Errorf("InsertPersist: cloned table has unexptected size, want %d, got %d", n, size2)
		}
	}
}

func TestDeletePersistLite(t *testing.T) {
	t.Parallel()

	const n = 10_000

	// setup
	//nolint:gosec
	prng := rand.New(rand.NewPCG(42, 42))
	pfxs := randomRealWorldPrefixes(prng, n)

	orig := new(Lite)
	for _, pfx := range pfxs {
		orig.Insert(pfx)
	}

	clone := orig
	for i, pfx := range pfxs {
		clone, _ = clone.DeletePersist(pfx)

		// test for existence
		if ok := orig.Exists(pfx); !ok {
			t.Errorf("DeletePersist: original table missing prefix %s", pfx)
		}

		// test for absence
		if ok := clone.Exists(pfx); ok {
			t.Errorf("DeletePersist: prefix %s not deleted in cloned table", pfx)
		}

		size1 := orig.Size()
		size2 := clone.Size()

		if size1 != n {
			t.Errorf("InsertPersist: original table has unexptected size, want %d, got %d", n, size1)
		}

		if size2 != n-i-1 {
			t.Errorf("InsertPersist: cloned table has unexptected size, want %d, got %d", n-i-1, size2)
		}
	}
}
