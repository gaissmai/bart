package bart

import (
	"testing"
)

// A simple type that implements Equaler for testing.
type stringVal string

func (v stringVal) Equal(other stringVal) bool {
	return v == other
}

func TestTableEqual(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		buildA    func() *Table[stringVal]
		buildB    func() *Table[stringVal]
		wantEqual bool
	}{
		{
			name:      "empty tables",
			buildA:    func() *Table[stringVal] { return new(Table[stringVal]) },
			buildB:    func() *Table[stringVal] { return new(Table[stringVal]) },
			wantEqual: true,
		},
		{
			name: "same single entry",
			buildA: func() *Table[stringVal] {
				tbl := new(Table[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "foo")
				return tbl
			},
			buildB: func() *Table[stringVal] {
				tbl := new(Table[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "foo")
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "different values for same prefix",
			buildA: func() *Table[stringVal] {
				tbl := new(Table[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "foo")
				return tbl
			},
			buildB: func() *Table[stringVal] {
				tbl := new(Table[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "bar")
				return tbl
			},
			wantEqual: false,
		},
		{
			name: "different entries",
			buildA: func() *Table[stringVal] {
				tbl := new(Table[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "foo")
				return tbl
			},
			buildB: func() *Table[stringVal] {
				tbl := new(Table[stringVal])
				tbl.Insert(mpp("2001:db8::/32"), "foo")
				return tbl
			},
			wantEqual: false,
		},
		{
			name: "same entries, different insert order",
			buildA: func() *Table[stringVal] {
				tbl := new(Table[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "foo")
				tbl.Insert(mpp("198.51.100.0/24"), "bar")
				return tbl
			},
			buildB: func() *Table[stringVal] {
				tbl := new(Table[stringVal])
				tbl.Insert(mpp("198.51.100.0/24"), "bar")
				tbl.Insert(mpp("192.0.2.0/24"), "foo")
				return tbl
			},
			wantEqual: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			a := tc.buildA()
			b := tc.buildB()

			got := a.Equal(b)
			if got != tc.wantEqual {
				t.Errorf("Equal() = %v, want %v", got, tc.wantEqual)
			}
		})
	}
}

func TestFullTableEqual(t *testing.T) {
	t.Parallel()
	at := new(Table[int])
	for i, r := range routes {
		at.Insert(r.CIDR, i)
	}

	t.Run("clone", func(t *testing.T) {
		t.Parallel()
		bt := at.Clone()
		if !at.Equal(bt) {
			t.Error("expected true, got false")
		}
	})

	t.Run("update", func(t *testing.T) {
		t.Parallel()
		ct := at.Clone()

		for i, r := range routes {
			// update value
			if i%42 == 0 {
				ct.Modify(r.CIDR, func(oldVal int, _ bool) (int, bool) { return oldVal + 1, false })
			}
		}

		if at.Equal(ct) {
			t.Error("expected false, got true")
		}
	})
}
