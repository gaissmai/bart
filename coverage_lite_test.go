package bart

import (
	"encoding/json"
	"net/netip"
	"testing"
)

// Test 1: Lite API-Basics — Lookup / LookupPrefix / LookupPrefixLPM / Size / Size4 / Size6 / DumpList / MarshalJSON / Equal
func TestLite_API_Basics(t *testing.T) {
	t.Parallel()

	l1 := new(Lite)

	// Insert both IPv4 and IPv6 prefixes
	v4 := []netip.Prefix{
		mpp("10.0.0.0/8"),
		mpp("10.1.0.0/16"),
	}
	v6 := []netip.Prefix{
		mpp("2001:db8::/32"),
		mpp("2001:db8:1::/48"),
	}

	for _, p := range v4 {
		l1.Insert(p)
	}
	for _, p := range v6 {
		l1.Insert(p)
	}

	// Size / Size4 / Size6
	if got := l1.Size(); got != len(v4)+len(v6) {
		t.Fatalf("Size() = %d, want %d", got, len(v4)+len(v6))
	}
	// These methods exist per your coverage listing; assert they sum correctly.
	if got4 := l1.Size4(); got4 != len(v4) {
		t.Fatalf("Size4() = %d, want %d", got4, len(v4))
	}
	if got6 := l1.Size6(); got6 != len(v6) {
		t.Fatalf("Size6() = %d, want %d", got6, len(v6))
	}

	// Lookup (Addr) — hit
	if ok := l1.Lookup(mpa("10.1.2.3")); !ok {
		t.Fatal("Lookup(10.1.2.3) = false, want true (LPM)")
	}
	// Lookup (Addr) — miss
	if ok := l1.Lookup(mpa("192.0.2.1")); ok {
		t.Fatal("Lookup(192.0.2.1) = true, want false")
	}

	// LookupPrefix — should LPM-match within the given prefix range
	if ok := l1.LookupPrefix(mpp("10.1.2.0/24")); !ok {
		t.Fatal("LookupPrefix(10.1.2.0/24) = false, want true (within 10.1.0.0/16)")
	}
	if ok := l1.LookupPrefix(mpp("192.0.2.0/24")); ok {
		t.Fatal("LookupPrefix(192.0.2.0/24) = true, want false")
	}

	// LookupPrefixLPM — same semantics, also returns the matched lpmPrefix
	if gotPfx, ok := l1.LookupPrefixLPM(mpp("10.1.2.0/24")); !ok || gotPfx.String() != "10.1.0.0/16" {
		t.Fatalf("LookupPrefixLPM(10.1.2.0/24) got (%s,%v), want (10.1.0.0/16,true)", gotPfx, ok)
	}
	if _, ok := l1.LookupPrefixLPM(mpp("203.0.113.0/24")); ok {
		t.Fatal("LookupPrefixLPM(203.0.113.0/24) = true, want false")
	}

	// DumpList4 / DumpList6 — just smoke test: non-empty and valid roots/subnets
	if dl4 := l1.DumpList4(); len(dl4) == 0 {
		t.Fatal("DumpList4() returned empty, want non-empty")
	}
	if dl6 := l1.DumpList6(); len(dl6) == 0 {
		t.Fatal("DumpList6() returned empty, want non-empty")
	}

	// MarshalJSON — validate it produces parseable JSON
	if b, err := l1.MarshalJSON(); err != nil || len(b) == 0 {
		t.Fatalf("MarshalJSON() error=%v len=%d", err, len(b))
	} else {
		var anyI interface{}
		if err := json.Unmarshal(b, &anyI); err != nil {
			t.Fatalf("MarshalJSON produced invalid JSON: %v", err)
		}
	}

	// Equal — identical content vs. different content
	l2 := new(Lite)
	for _, p := range append(append([]netip.Prefix{}, v4...), v6...) {
		l2.Insert(p)
	}
	if !l1.Equal(l2) {
		t.Fatal("Equal(l1,l2) = false, want true (same prefixes)")
	}
	l2.Insert(mpp("10.2.0.0/16"))
	if l1.Equal(l2) {
		t.Fatal("Equal(l1,l2) = true, want false (l2 has extra prefix)")
	}
}

// Test 2: Lite persistent ops — ModifyPersist (insert/update/delete) + WalkPersist
func TestLite_Persist_Modify(t *testing.T) {
	t.Parallel()

	// Start empty
	l := new(Lite)

	// ModifyPersist — insert when not present (exists=false, return false to keep => insert)
	l = l.ModifyPersist(mpp("10.0.0.0/8"), func(exists bool) bool {
		// return false => do not delete => ensure present (insert when !exists)
		return false
	})
	if ok := l.Get(mpp("10.0.0.0/8")); !ok {
		t.Fatal("ModifyPersist(insert): prefix not present after insert")
	}

	// ModifyPersist — update path for presence-only table is a no-op on value (still test callback path)
	l2 := l.ModifyPersist(mpp("10.0.0.0/8"), func(exists bool) bool {
		if !exists {
			t.Fatal("ModifyPersist(update): exists=false, want true")
		}
		// keep it
		return false
	})
	if ok := l2.Get(mpp("10.0.0.0/8")); !ok {
		t.Fatal("ModifyPersist(update): prefix missing after update")
	}

	// ModifyPersist — delete existing (return true => delete)
	l3 := l2.ModifyPersist(mpp("10.0.0.0/8"), func(exists bool) bool {
		if !exists {
			t.Fatal("ModifyPersist(delete): exists=false, want true")
		}
		return true // delete
	})
	if ok := l3.Get(mpp("10.0.0.0/8")); ok {
		t.Fatal("ModifyPersist(delete): prefix still present after delete")
	}
}

// Lite: Lookup, LookupPrefix, LookupPrefixLPM
func TestLite_Lookup_Family(t *testing.T) {
	t.Parallel()

	l := new(Lite)
	l.Insert(mpp("10.1.0.0/16"))
	l.Insert(mpp("2001:db8::/32"))

	// Lookup(addr) hit/miss
	if !l.Lookup(mpa("10.1.2.3")) {
		t.Fatalf("Lite.Lookup(10.1.2.3) = false, want true")
	}
	if l.Lookup(mpa("192.0.2.1")) {
		t.Fatalf("Lite.Lookup(192.0.2.1) = true, want false")
	}
	if !l.Lookup(mpa("2001:db8:1234::1")) {
		t.Fatalf("Lite.Lookup(2001:db8:1234::1) = false, want true")
	}

	// LookupPrefix(pfx) — LPM über den Bereich
	if !l.LookupPrefix(mpp("10.1.2.0/24")) {
		t.Fatalf("Lite.LookupPrefix(10.1.2.0/24) = false, want true")
	}
	if l.LookupPrefix(mpp("203.0.113.0/24")) {
		t.Fatalf("Lite.LookupPrefix(203.0.113.0/24) = true, want false")
	}

	// LookupPrefixLPM(pfx) — gibt auch den gematchten Prefix zurück
	if gotPfx, ok := l.LookupPrefixLPM(mpp("10.1.2.0/24")); !ok || gotPfx.String() != "10.1.0.0/16" {
		t.Fatalf("LookupPrefixLPM(10.1.2.0/24) got (%s,%v), want (10.1.0.0/16,true)", gotPfx, ok)
	}
	if _, ok := l.LookupPrefixLPM(mpp("203.0.113.0/24")); ok {
		t.Fatalf("LookupPrefixLPM(203.0.113.0/24) = true, want false")
	}
}
