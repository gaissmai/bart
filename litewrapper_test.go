// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"slices"
	"testing"
)

// The tests in this file cover the wrapper methods of Lite that adapt the
// signatures of the generated liteTable methods: the callback of Modify and
// ModifyPersist, and the early return of UnionPersist. The generated tests run
// against liteTable, not against these wrappers.

func TestLiteWrapperModify(t *testing.T) {
	t.Parallel()

	pfx := mpp("10.0.0.0/8")

	tests := []struct {
		name     string
		inTable  bool // the prefix is in the table before
		del      bool // the return value of the callback
		wantIn   bool // the prefix is in the table afterwards
		wantSize int
	}{
		{"no-op: not in the table, callback returns true", false, true, false, 0},
		{"insert: not in the table, callback returns false", false, false, true, 1},
		{"update: in the table, callback returns false", true, false, true, 1},
		{"delete: in the table, callback returns true", true, true, false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			newTable := func() *Lite {
				l := new(Lite)
				if tt.inTable {
					l.Insert(pfx)
				}

				return l
			}

			// the callback is called once, with the information if the prefix is in the table
			var calls []bool

			cb := func(exists bool) (del bool) {
				calls = append(calls, exists)
				return tt.del
			}

			wantCalls := []bool{tt.inTable}

			// Modify
			l := newTable()
			l.Modify(pfx, cb)

			if !slices.Equal(calls, wantCalls) {
				t.Errorf("Modify: callback calls = %v, want %v", calls, wantCalls)
			}

			if got := l.Get(pfx); got != tt.wantIn {
				t.Errorf("Modify: Get(%s) = %v, want %v", pfx, got, tt.wantIn)
			}

			if got := l.Size(); got != tt.wantSize {
				t.Errorf("Modify: Size() = %d, want %d", got, tt.wantSize)
			}

			// ModifyPersist, the result is the same and the receiver is unchanged
			calls = nil
			orig := newTable()
			res := orig.ModifyPersist(pfx, cb)

			if !slices.Equal(calls, wantCalls) {
				t.Errorf("ModifyPersist: callback calls = %v, want %v", calls, wantCalls)
			}

			if got := res.Get(pfx); got != tt.wantIn {
				t.Errorf("ModifyPersist: Get(%s) = %v, want %v", pfx, got, tt.wantIn)
			}

			if got := res.Size(); got != tt.wantSize {
				t.Errorf("ModifyPersist: Size() = %d, want %d", got, tt.wantSize)
			}

			wantOrigSize := 0
			if tt.inTable {
				wantOrigSize = 1
			}

			if got := orig.Get(pfx); got != tt.inTable || orig.Size() != wantOrigSize {
				t.Errorf("ModifyPersist modified the receiver: Get(%s) = %v, Size() = %d", pfx, got, orig.Size())
			}
		})
	}
}

func TestLiteWrapperModifyInvalidPrefix(t *testing.T) {
	t.Parallel()

	cb := func(bool) bool {
		t.Error("the callback must not be called for an invalid prefix")
		return false
	}

	l := new(Lite)
	l.Insert(mpp("10.0.0.0/8"))

	l.Modify(netip.Prefix{}, cb)

	if l.Size() != 1 {
		t.Errorf("Modify with an invalid prefix changed the table, Size() = %d", l.Size())
	}

	if res := l.ModifyPersist(netip.Prefix{}, cb); res.Size() != 1 {
		t.Errorf("ModifyPersist with an invalid prefix changed the table, Size() = %d", res.Size())
	}
}

// TestLiteWrapperModifyCanonicalizes tests that the prefix is masked before
// the operation, 10.1.2.3/8 is the prefix 10.0.0.0/8.
func TestLiteWrapperModifyCanonicalizes(t *testing.T) {
	t.Parallel()

	unmasked := netip.MustParsePrefix("10.1.2.3/8")
	insert := func(bool) bool { return false }

	l := new(Lite)
	l.Modify(unmasked, insert)

	if !l.Get(mpp("10.0.0.0/8")) {
		t.Error("Modify did not insert the canonical prefix 10.0.0.0/8")
	}

	if res := new(Lite).ModifyPersist(unmasked, insert); !res.Get(mpp("10.0.0.0/8")) {
		t.Error("ModifyPersist did not insert the canonical prefix 10.0.0.0/8")
	}
}

// TestLiteWrapperUnionPersist tests UnionPersist with an empty table, with the
// table itself and with a table with prefixes. The receiver stays unchanged.
func TestLiteWrapperUnionPersist(t *testing.T) {
	t.Parallel()

	recv := []netip.Prefix{mpp("10.0.0.0/8"), mpp("2001:db8::/32")}

	tests := []struct {
		name  string
		other func(a *Lite) *Lite
		want  []netip.Prefix
	}{
		{"empty other table", func(*Lite) *Lite { return new(Lite) }, recv},
		{"the receiver itself", func(a *Lite) *Lite { return a }, recv},
		{
			"other table with prefixes",
			func(*Lite) *Lite {
				o := new(Lite)
				o.Insert(mpp("192.168.0.0/16"))

				return o
			},
			[]netip.Prefix{mpp("10.0.0.0/8"), mpp("192.168.0.0/16"), mpp("2001:db8::/32")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := new(Lite)
			for _, pfx := range recv {
				a.Insert(pfx)
			}

			res := a.UnionPersist(tt.other(a))

			if got := slices.Collect(res.AllSorted()); !slices.Equal(got, tt.want) {
				t.Errorf("UnionPersist: prefixes = %v, want %v", got, tt.want)
			}

			if got := slices.Collect(a.AllSorted()); !slices.Equal(got, recv) {
				t.Errorf("UnionPersist modified the receiver: prefixes = %v, want %v", got, recv)
			}
		})
	}
}
