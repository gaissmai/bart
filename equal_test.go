// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

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

	t.Run("modify", func(t *testing.T) {
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

func TestFastEqual(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		buildA    func() *Fast[stringVal]
		buildB    func() *Fast[stringVal]
		wantEqual bool
	}{
		{
			name:      "second nil",
			buildA:    func() *Fast[stringVal] { return new(Fast[stringVal]) },
			buildB:    func() *Fast[stringVal] { return nil },
			wantEqual: false,
		},
		{
			name:      "empty tables",
			buildA:    func() *Fast[stringVal] { return new(Fast[stringVal]) },
			buildB:    func() *Fast[stringVal] { return new(Fast[stringVal]) },
			wantEqual: true,
		},
		{
			name: "same single entry",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "foo")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "foo")
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "same single IPv6 entry",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("2001:db8::/32"), "ipv6")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("2001:db8::/32"), "ipv6")
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "different values for same prefix",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "foo")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "bar")
				return tbl
			},
			wantEqual: false,
		},
		{
			name: "different entries",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "foo")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("2001:db8::/32"), "foo")
				return tbl
			},
			wantEqual: false,
		},
		{
			name: "different sizes - more in A",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "foo")
				tbl.Insert(mpp("192.0.3.0/24"), "bar")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "foo")
				return tbl
			},
			wantEqual: false,
		},
		{
			name: "different sizes - more in B",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "foo")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "foo")
				tbl.Insert(mpp("192.0.3.0/24"), "bar")
				return tbl
			},
			wantEqual: false,
		},
		{
			name: "same entries, different insert order",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "foo")
				tbl.Insert(mpp("198.51.100.0/24"), "bar")
				tbl.Insert(mpp("2001:db8::/32"), "ipv6")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("2001:db8::/32"), "ipv6")
				tbl.Insert(mpp("198.51.100.0/24"), "bar")
				tbl.Insert(mpp("192.0.2.0/24"), "foo")
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "complex hierarchical prefixes",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("10.0.0.0/8"), "supernet")
				tbl.Insert(mpp("10.1.0.0/16"), "subnet1")
				tbl.Insert(mpp("10.2.0.0/16"), "subnet2")
				tbl.Insert(mpp("10.1.1.0/24"), "subsubnet")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("10.1.1.0/24"), "subsubnet")
				tbl.Insert(mpp("10.2.0.0/16"), "subnet2")
				tbl.Insert(mpp("10.0.0.0/8"), "supernet")
				tbl.Insert(mpp("10.1.0.0/16"), "subnet1")
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "host routes",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.1/32"), "host4")
				tbl.Insert(mpp("2001:db8::1/128"), "host6")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.1/32"), "host4")
				tbl.Insert(mpp("2001:db8::1/128"), "host6")
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "default routes",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("0.0.0.0/0"), "default4")
				tbl.Insert(mpp("::/0"), "default6")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("0.0.0.0/0"), "default4")
				tbl.Insert(mpp("::/0"), "default6")
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
				t.Errorf("Fast.Equal() = %v, want %v", got, tc.wantEqual)
			}

			// Test symmetry (a.Equal(b) should equal b.Equal(a))
			if a != nil && b != nil {
				gotReverse := b.Equal(a)
				if got != gotReverse {
					t.Errorf("Equal() not symmetric: a.Equal(b) = %v, b.Equal(a) = %v", got, gotReverse)
				}
			}
		})
	}
}

func TestLiteEqual(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		buildA    func() *Lite
		buildB    func() *Lite
		wantEqual bool
	}{
		{
			name:      "second nil",
			buildA:    func() *Lite { return new(Lite) },
			buildB:    func() *Lite { return nil },
			wantEqual: false,
		},
		{
			name:      "empty tables",
			buildA:    func() *Lite { return new(Lite) },
			buildB:    func() *Lite { return new(Lite) },
			wantEqual: true,
		},
		{
			name: "same single entry",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.0/24"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.0/24"))
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "same single IPv6 entry",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("2001:db8::/32"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("2001:db8::/32"))
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "different entries",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.0/24"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("2001:db8::/32"))
				return tbl
			},
			wantEqual: false,
		},
		{
			name: "different sizes - more in A",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.0/24"))
				tbl.Insert(mpp("192.0.3.0/24"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.0/24"))
				return tbl
			},
			wantEqual: false,
		},
		{
			name: "different sizes - more in B",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.0/24"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.0/24"))
				tbl.Insert(mpp("192.0.3.0/24"))
				return tbl
			},
			wantEqual: false,
		},
		{
			name: "same entries, different insert order",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.0/24"))
				tbl.Insert(mpp("198.51.100.0/24"))
				tbl.Insert(mpp("2001:db8::/32"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("2001:db8::/32"))
				tbl.Insert(mpp("198.51.100.0/24"))
				tbl.Insert(mpp("192.0.2.0/24"))
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "complex hierarchical prefixes",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("10.0.0.0/8"))
				tbl.Insert(mpp("10.1.0.0/16"))
				tbl.Insert(mpp("10.2.0.0/16"))
				tbl.Insert(mpp("10.1.1.0/24"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("10.1.1.0/24"))
				tbl.Insert(mpp("10.2.0.0/16"))
				tbl.Insert(mpp("10.0.0.0/8"))
				tbl.Insert(mpp("10.1.0.0/16"))
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "host routes",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.1/32"))
				tbl.Insert(mpp("2001:db8::1/128"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.1/32"))
				tbl.Insert(mpp("2001:db8::1/128"))
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "default routes",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("0.0.0.0/0"))
				tbl.Insert(mpp("::/0"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("0.0.0.0/0"))
				tbl.Insert(mpp("::/0"))
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
				t.Errorf("Lite.Equal() = %v, want %v", got, tc.wantEqual)
			}

			// Test symmetry (a.Equal(b) should equal b.Equal(a))
			if a != nil && b != nil {
				gotReverse := b.Equal(a)
				if got != gotReverse {
					t.Errorf("Equal() not symmetric: a.Equal(b) = %v, b.Equal(a) = %v", got, gotReverse)
				}
			}
		})
	}
}

func TestFullFastEqual(t *testing.T) {
	t.Parallel()
	at := new(Fast[int])
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

	t.Run("modify", func(t *testing.T) {
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

func TestFullLiteEqual(t *testing.T) {
	t.Parallel()
	at := new(Lite)
	for _, r := range routes {
		at.Insert(r.CIDR)
	}

	t.Run("clone", func(t *testing.T) {
		t.Parallel()
		bt := at.Clone()
		if !at.Equal(bt) {
			t.Error("expected true, got false")
		}
	})

	t.Run("modify", func(t *testing.T) {
		t.Parallel()
		ct := at.Clone()

		// Delete some entries to make tables different
		for i, r := range routes {
			if i%42 == 0 {
				ct.Delete(r.CIDR)
			}
		}

		if at.Equal(ct) {
			t.Error("expected false, got true")
		}
	})
}

// Cross-type consistency tests to ensure all implementations behave similarly
func TestEqualConsistencyAcrossTypes(t *testing.T) {
	t.Parallel()

	// Test that equivalent tables across types have same Equal behavior
	prefixes := []string{
		"192.0.2.0/24",
		"198.51.100.0/24",
		"2001:db8::/32",
		"10.0.0.0/8",
		"172.16.0.0/12",
	}

	t.Run("equal_empty_tables", func(t *testing.T) {
		t.Parallel()

		table1 := new(Table[stringVal])
		table2 := new(Table[stringVal])
		fast1 := new(Fast[stringVal])
		fast2 := new(Fast[stringVal])
		lite1 := new(Lite)
		lite2 := new(Lite)

		// All empty tables should be equal to their same type
		if !table1.Equal(table2) || !fast1.Equal(fast2) || !lite1.Equal(lite2) {
			t.Error("Empty tables of same type should be equal")
		}
	})

	t.Run("equal_populated_tables", func(t *testing.T) {
		t.Parallel()

		// Build identical content across types
		table1 := new(Table[stringVal])
		table2 := new(Table[stringVal])
		fast1 := new(Fast[stringVal])
		fast2 := new(Fast[stringVal])
		lite1 := new(Lite)
		lite2 := new(Lite)

		for i, pfxStr := range prefixes {
			pfx := mpp(pfxStr)
			val := stringVal("value" + string(rune('0'+i)))

			table1.Insert(pfx, val)
			table2.Insert(pfx, val)
			fast1.Insert(pfx, val)
			fast2.Insert(pfx, val)
			lite1.Insert(pfx)
			lite2.Insert(pfx)
		}

		// All populated tables should be equal to their same type
		if !table1.Equal(table2) || !fast1.Equal(fast2) || !lite1.Equal(lite2) {
			t.Error("Identically populated tables of same type should be equal")
		}
	})
}

// Benchmark tests for Equal() performance
func BenchmarkTableEqual(b *testing.B) {
	// Build two identical tables
	tbl1 := new(Table[stringVal])
	tbl2 := new(Table[stringVal])

	for i, r := range routes {
		if i > 1000 { // Limit for reasonable benchmark
			break
		}
		val := stringVal("value")
		tbl1.Insert(r.CIDR, val)
		tbl2.Insert(r.CIDR, val)
	}

	b.ResetTimer()
	for b.Loop() {
		_ = tbl1.Equal(tbl2)
	}
}

func BenchmarkFastEqual(b *testing.B) {
	// Build two identical tables
	tbl1 := new(Fast[stringVal])
	tbl2 := new(Fast[stringVal])

	for i, r := range routes {
		if i > 1000 { // Limit for reasonable benchmark
			break
		}
		val := stringVal("value")
		tbl1.Insert(r.CIDR, val)
		tbl2.Insert(r.CIDR, val)
	}

	b.ResetTimer()
	for b.Loop() {
		_ = tbl1.Equal(tbl2)
	}
}

func BenchmarkLiteEqual(b *testing.B) {
	// Build two identical tables
	tbl1 := new(Lite)
	tbl2 := new(Lite)

	for i, r := range routes {
		if i > 1000 { // Limit for reasonable benchmark
			break
		}
		tbl1.Insert(r.CIDR)
		tbl2.Insert(r.CIDR)
	}

	b.ResetTimer()
	for b.Loop() {
		_ = tbl1.Equal(tbl2)
	}
}
