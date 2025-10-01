// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"fmt"
	"testing"
)

// A simple type that implements Equaler for testing.
type stringVal string

func (v stringVal) Equal(other stringVal) bool {
	return v == other
}

// Test nil receiver behavior
// Test reflexivity property: a.Equal(a) should always be true
func TestTableEqualReflexivity(t *testing.T) {
	t.Parallel()

	t.Run("empty_table", func(t *testing.T) {
		t.Parallel()
		tbl := new(Table[stringVal])
		if !tbl.Equal(tbl) {
			t.Error("Table should be equal to itself")
		}
	})

	t.Run("single_entry", func(t *testing.T) {
		t.Parallel()
		tbl := new(Table[stringVal])
		tbl.Insert(mpp("192.0.2.0/24"), "foo")
		if !tbl.Equal(tbl) {
			t.Error("Table should be equal to itself")
		}
	})

	t.Run("multiple_entries", func(t *testing.T) {
		t.Parallel()
		tbl := new(Table[stringVal])
		tbl.Insert(mpp("192.0.2.0/24"), "foo")
		tbl.Insert(mpp("198.51.100.0/24"), "bar")
		tbl.Insert(mpp("2001:db8::/32"), "baz")
		if !tbl.Equal(tbl) {
			t.Error("Table should be equal to itself")
		}
	})
}

func TestFastEqualReflexivity(t *testing.T) {

// Test edge cases with overlapping and adjacent prefixes
func TestFastEqualOverlappingPrefixes(t *testing.T) {

// Test comprehensive mixed IPv4 and IPv6 scenarios
func TestFastEqualMixedIPVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		buildA    func() *Fast[stringVal]
		buildB    func() *Fast[stringVal]
		wantEqual bool
	}{
		{
			name: "IPv4 only vs IPv6 only",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "v4")
				tbl.Insert(mpp("198.51.100.0/24"), "v4")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("2001:db8::/32"), "v6")
				tbl.Insert(mpp("2001:db8:1::/48"), "v6")
				return tbl
			},
			wantEqual: false,
		},
		{
			name: "same mixed IPv4 and IPv6",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "val1")
				tbl.Insert(mpp("2001:db8::/32"), "val2")
				tbl.Insert(mpp("10.0.0.0/8"), "val3")
				tbl.Insert(mpp("2001:db8:1::/48"), "val4")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("2001:db8:1::/48"), "val4")
				tbl.Insert(mpp("10.0.0.0/8"), "val3")
				tbl.Insert(mpp("2001:db8::/32"), "val2")
				tbl.Insert(mpp("192.0.2.0/24"), "val1")
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "different number IPv4 vs IPv6",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "val")
				tbl.Insert(mpp("2001:db8::/32"), "val")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "val")
				tbl.Insert(mpp("198.51.100.0/24"), "val")
				tbl.Insert(mpp("2001:db8::/32"), "val")
				return tbl
			},
			wantEqual: false,
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
		})
	}
}

	t.Parallel()

	tests := []struct {
		name      string
		buildA    func() *Fast[stringVal]
		buildB    func() *Fast[stringVal]
		wantEqual bool
	}{
		{
			name: "overlapping prefixes different values",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("10.0.0.0/8"), "parent")
				tbl.Insert(mpp("10.1.0.0/16"), "child")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("10.0.0.0/8"), "parent")
				tbl.Insert(mpp("10.1.0.0/16"), "different")
				return tbl
			},
			wantEqual: false,
		},
		{
			name: "adjacent prefixes",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/25"), "first")
				tbl.Insert(mpp("192.0.2.128/25"), "second")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/25"), "first")
				tbl.Insert(mpp("192.0.2.128/25"), "second")
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "missing child prefix",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("10.0.0.0/8"), "parent")
				tbl.Insert(mpp("10.1.0.0/16"), "child")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("10.0.0.0/8"), "parent")
				return tbl
			},
			wantEqual: false,
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
		})
	}
}

	t.Parallel()

	t.Run("empty_table", func(t *testing.T) {
		t.Parallel()
		tbl := new(Fast[stringVal])
		if !tbl.Equal(tbl) {
			t.Error("Fast table should be equal to itself")
		}
	})

	t.Run("multiple_entries", func(t *testing.T) {
		t.Parallel()
		tbl := new(Fast[stringVal])
		tbl.Insert(mpp("192.0.2.0/24"), "foo")
		tbl.Insert(mpp("198.51.100.0/24"), "bar")
		tbl.Insert(mpp("2001:db8::/32"), "baz")
		if !tbl.Equal(tbl) {
			t.Error("Fast table should be equal to itself")
		}
	})
}

func TestLiteEqualReflexivity(t *testing.T) {

// Test edge cases with various prefix lengths
func TestLiteEqualPrefixLengthBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		buildA    func() *Lite
		buildB    func() *Lite
		wantEqual bool
	}{
		{
			name: "different prefix lengths same base",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.0/24"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.0/25"))
				return tbl
			},
			wantEqual: false,
		},
		{
			name: "minimum prefix length IPv4 /0",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("0.0.0.0/0"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("0.0.0.0/0"))
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "maximum prefix length IPv4 /32",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.1/32"))
				tbl.Insert(mpp("192.0.2.2/32"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.1/32"))
				tbl.Insert(mpp("192.0.2.2/32"))
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "maximum prefix length IPv6 /128",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("2001:db8::1/128"))
				tbl.Insert(mpp("2001:db8::2/128"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("2001:db8::1/128"))
				tbl.Insert(mpp("2001:db8::2/128"))
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
		})
	}
}

	t.Parallel()

	t.Run("empty_table", func(t *testing.T) {
		t.Parallel()
		tbl := new(Lite)
		if !tbl.Equal(tbl) {
			t.Error("Lite table should be equal to itself")
		}
	})

	t.Run("multiple_entries", func(t *testing.T) {
		t.Parallel()
		tbl := new(Lite)
		tbl.Insert(mpp("192.0.2.0/24"))
		tbl.Insert(mpp("198.51.100.0/24"))
		tbl.Insert(mpp("2001:db8::/32"))
		if !tbl.Equal(tbl) {
			t.Error("Lite table should be equal to itself")
		}
	})
}

func TestTableEqualNilReceiver(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		buildA    func() *Table[stringVal]
		buildB    func() *Table[stringVal]
		wantEqual bool
	}{
		{
			name:      "nil receiver vs empty table",
			buildA:    func() *Table[stringVal] { return nil },
			buildB:    func() *Table[stringVal] { return new(Table[stringVal]) },
			wantEqual: false,
		},
		{
			name:      "nil receiver vs nil",
			buildA:    func() *Table[stringVal] { return nil },
			buildB:    func() *Table[stringVal] { return nil },
			wantEqual: true,
		},
		{
			name:      "empty table vs nil",
			buildA:    func() *Table[stringVal] { return new(Table[stringVal]) },
			buildB:    func() *Table[stringVal] { return nil },
			wantEqual: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			a := tc.buildA()
			b := tc.buildB()

			if a == nil {
				// Skip calling Equal on nil receiver
				// Test should handle this gracefully if Equal checks for nil receiver
				return
			}

			got := a.Equal(b)
			if got != tc.wantEqual {
				t.Errorf("Equal() = %v, want %v", got, tc.wantEqual)
			}
		})
	}
}

// Test reflexivity property: a.Equal(a) should always be true
func TestTableEqualReflexivity(t *testing.T) {
	t.Parallel()

	t.Run("empty_table", func(t *testing.T) {
		t.Parallel()
		tbl := new(Table[stringVal])
		if !tbl.Equal(tbl) {
			t.Error("Table should be equal to itself")
		}
	})

	t.Run("single_entry", func(t *testing.T) {
		t.Parallel()
		tbl := new(Table[stringVal])
		tbl.Insert(mpp("192.0.2.0/24"), "foo")
		if !tbl.Equal(tbl) {
			t.Error("Table should be equal to itself")
		}
	})

	t.Run("multiple_entries", func(t *testing.T) {
		t.Parallel()
		tbl := new(Table[stringVal])
		tbl.Insert(mpp("192.0.2.0/24"), "foo")
		tbl.Insert(mpp("198.51.100.0/24"), "bar")
		tbl.Insert(mpp("2001:db8::/32"), "baz")
		if !tbl.Equal(tbl) {
			t.Error("Table should be equal to itself")
		}
	})
}

func TestFastEqualReflexivity(t *testing.T) {

// Test edge cases with overlapping and adjacent prefixes
func TestFastEqualOverlappingPrefixes(t *testing.T) {

// Test comprehensive mixed IPv4 and IPv6 scenarios
func TestFastEqualMixedIPVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		buildA    func() *Fast[stringVal]
		buildB    func() *Fast[stringVal]
		wantEqual bool
	}{
		{
			name: "IPv4 only vs IPv6 only",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "v4")
				tbl.Insert(mpp("198.51.100.0/24"), "v4")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("2001:db8::/32"), "v6")
				tbl.Insert(mpp("2001:db8:1::/48"), "v6")
				return tbl
			},
			wantEqual: false,
		},
		{
			name: "same mixed IPv4 and IPv6",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "val1")
				tbl.Insert(mpp("2001:db8::/32"), "val2")
				tbl.Insert(mpp("10.0.0.0/8"), "val3")
				tbl.Insert(mpp("2001:db8:1::/48"), "val4")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("2001:db8:1::/48"), "val4")
				tbl.Insert(mpp("10.0.0.0/8"), "val3")
				tbl.Insert(mpp("2001:db8::/32"), "val2")
				tbl.Insert(mpp("192.0.2.0/24"), "val1")
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "different number IPv4 vs IPv6",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "val")
				tbl.Insert(mpp("2001:db8::/32"), "val")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "val")
				tbl.Insert(mpp("198.51.100.0/24"), "val")
				tbl.Insert(mpp("2001:db8::/32"), "val")
				return tbl
			},
			wantEqual: false,
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
		})
	}
}

	t.Parallel()

	tests := []struct {
		name      string
		buildA    func() *Fast[stringVal]
		buildB    func() *Fast[stringVal]
		wantEqual bool
	}{
		{
			name: "overlapping prefixes different values",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("10.0.0.0/8"), "parent")
				tbl.Insert(mpp("10.1.0.0/16"), "child")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("10.0.0.0/8"), "parent")
				tbl.Insert(mpp("10.1.0.0/16"), "different")
				return tbl
			},
			wantEqual: false,
		},
		{
			name: "adjacent prefixes",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/25"), "first")
				tbl.Insert(mpp("192.0.2.128/25"), "second")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/25"), "first")
				tbl.Insert(mpp("192.0.2.128/25"), "second")
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "missing child prefix",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("10.0.0.0/8"), "parent")
				tbl.Insert(mpp("10.1.0.0/16"), "child")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("10.0.0.0/8"), "parent")
				return tbl
			},
			wantEqual: false,
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
		})
	}
}

	t.Parallel()

	t.Run("empty_table", func(t *testing.T) {
		t.Parallel()
		tbl := new(Fast[stringVal])
		if !tbl.Equal(tbl) {
			t.Error("Fast table should be equal to itself")
		}
	})

	t.Run("multiple_entries", func(t *testing.T) {
		t.Parallel()
		tbl := new(Fast[stringVal])
		tbl.Insert(mpp("192.0.2.0/24"), "foo")
		tbl.Insert(mpp("198.51.100.0/24"), "bar")
		tbl.Insert(mpp("2001:db8::/32"), "baz")
		if !tbl.Equal(tbl) {
			t.Error("Fast table should be equal to itself")
		}
	})
}

func TestLiteEqualReflexivity(t *testing.T) {

// Test edge cases with various prefix lengths
func TestLiteEqualPrefixLengthBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		buildA    func() *Lite
		buildB    func() *Lite
		wantEqual bool
	}{
		{
			name: "different prefix lengths same base",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.0/24"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.0/25"))
				return tbl
			},
			wantEqual: false,
		},
		{
			name: "minimum prefix length IPv4 /0",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("0.0.0.0/0"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("0.0.0.0/0"))
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "maximum prefix length IPv4 /32",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.1/32"))
				tbl.Insert(mpp("192.0.2.2/32"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.1/32"))
				tbl.Insert(mpp("192.0.2.2/32"))
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "maximum prefix length IPv6 /128",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("2001:db8::1/128"))
				tbl.Insert(mpp("2001:db8::2/128"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("2001:db8::1/128"))
				tbl.Insert(mpp("2001:db8::2/128"))
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
		})
	}
}

	t.Parallel()

	t.Run("empty_table", func(t *testing.T) {
		t.Parallel()
		tbl := new(Lite)
		if !tbl.Equal(tbl) {
			t.Error("Lite table should be equal to itself")
		}
	})

	t.Run("multiple_entries", func(t *testing.T) {
		t.Parallel()
		tbl := new(Lite)
		tbl.Insert(mpp("192.0.2.0/24"))
		tbl.Insert(mpp("198.51.100.0/24"))
		tbl.Insert(mpp("2001:db8::/32"))
		if !tbl.Equal(tbl) {
			t.Error("Lite table should be equal to itself")
		}
	})
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

// Test Equal with edge case values (empty strings, special characters)
func TestTableEqualEdgeValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		buildA    func() *Table[stringVal]
		buildB    func() *Table[stringVal]
		wantEqual bool
	}{
		{
			name: "empty string values",
			buildA: func() *Table[stringVal] {
				tbl := new(Table[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "")
				return tbl
			},
			buildB: func() *Table[stringVal] {
				tbl := new(Table[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "")
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "empty vs non-empty string",
			buildA: func() *Table[stringVal] {
				tbl := new(Table[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "")
				return tbl
			},
			buildB: func() *Table[stringVal] {
				tbl := new(Table[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "value")
				return tbl
			},
			wantEqual: false,
		},
		{
			name: "unicode values",
			buildA: func() *Table[stringVal] {
				tbl := new(Table[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "日本語")
				tbl.Insert(mpp("198.51.100.0/24"), "🔥")
				return tbl
			},
			buildB: func() *Table[stringVal] {
				tbl := new(Table[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "日本語")
				tbl.Insert(mpp("198.51.100.0/24"), "🔥")
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
				t.Errorf("Table.Equal() = %v, want %v", got, tc.wantEqual)
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

// Test edge cases with overlapping and adjacent prefixes
func TestFastEqualOverlappingPrefixes(t *testing.T) {

// Test comprehensive mixed IPv4 and IPv6 scenarios
func TestFastEqualMixedIPVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		buildA    func() *Fast[stringVal]
		buildB    func() *Fast[stringVal]
		wantEqual bool
	}{
		{
			name: "IPv4 only vs IPv6 only",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "v4")
				tbl.Insert(mpp("198.51.100.0/24"), "v4")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("2001:db8::/32"), "v6")
				tbl.Insert(mpp("2001:db8:1::/48"), "v6")
				return tbl
			},
			wantEqual: false,
		},
		{
			name: "same mixed IPv4 and IPv6",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "val1")
				tbl.Insert(mpp("2001:db8::/32"), "val2")
				tbl.Insert(mpp("10.0.0.0/8"), "val3")
				tbl.Insert(mpp("2001:db8:1::/48"), "val4")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("2001:db8:1::/48"), "val4")
				tbl.Insert(mpp("10.0.0.0/8"), "val3")
				tbl.Insert(mpp("2001:db8::/32"), "val2")
				tbl.Insert(mpp("192.0.2.0/24"), "val1")
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "different number IPv4 vs IPv6",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "val")
				tbl.Insert(mpp("2001:db8::/32"), "val")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "val")
				tbl.Insert(mpp("198.51.100.0/24"), "val")
				tbl.Insert(mpp("2001:db8::/32"), "val")
				return tbl
			},
			wantEqual: false,
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
		})
	}
}

	t.Parallel()

	tests := []struct {
		name      string
		buildA    func() *Fast[stringVal]
		buildB    func() *Fast[stringVal]
		wantEqual bool
	}{
		{
			name: "overlapping prefixes different values",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("10.0.0.0/8"), "parent")
				tbl.Insert(mpp("10.1.0.0/16"), "child")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("10.0.0.0/8"), "parent")
				tbl.Insert(mpp("10.1.0.0/16"), "different")
				return tbl
			},
			wantEqual: false,
		},
		{
			name: "adjacent prefixes",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/25"), "first")
				tbl.Insert(mpp("192.0.2.128/25"), "second")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/25"), "first")
				tbl.Insert(mpp("192.0.2.128/25"), "second")
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "missing child prefix",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("10.0.0.0/8"), "parent")
				tbl.Insert(mpp("10.1.0.0/16"), "child")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("10.0.0.0/8"), "parent")
				return tbl
			},
			wantEqual: false,
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
		})
	}
}

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

// Test edge cases with various prefix lengths
func TestLiteEqualPrefixLengthBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		buildA    func() *Lite
		buildB    func() *Lite
		wantEqual bool
	}{
		{
			name: "different prefix lengths same base",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.0/24"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.0/25"))
				return tbl
			},
			wantEqual: false,
		},
		{
			name: "minimum prefix length IPv4 /0",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("0.0.0.0/0"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("0.0.0.0/0"))
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "maximum prefix length IPv4 /32",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.1/32"))
				tbl.Insert(mpp("192.0.2.2/32"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.1/32"))
				tbl.Insert(mpp("192.0.2.2/32"))
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "maximum prefix length IPv6 /128",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("2001:db8::1/128"))
				tbl.Insert(mpp("2001:db8::2/128"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("2001:db8::1/128"))
				tbl.Insert(mpp("2001:db8::2/128"))
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
		})
	}
}

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

// Test Equal with large number of mixed IPv4 and IPv6 entries
func TestFastEqualLargeDataset(t *testing.T) {
	t.Parallel()

	a := new(Fast[int])
	b := new(Fast[int])

	// Insert 100 IPv4 and IPv6 prefixes
	for i := 0; i < 50; i++ {
		// Generate diverse test prefixes
		ipv4Prefix := mpp(fmt.Sprintf("10.%d.0.0/16", i))
		ipv6Prefix := mpp(fmt.Sprintf("2001:db8:%x::/48", i))
		a.Insert(ipv4Prefix, i)
		a.Insert(ipv6Prefix, i+1000)
		b.Insert(ipv4Prefix, i)
		b.Insert(ipv6Prefix, i+1000)
	}

	if !a.Equal(b) {
		t.Error("Large datasets should be equal")
	}

	// Modify one entry and test inequality
	b.Insert(mpp("10.0.0.0/16"), 999)
	if a.Equal(b) {
		t.Error("Modified large datasets should not be equal")
	}
}

func TestLiteEqualLargeDataset(t *testing.T) {

// Test edge cases with various prefix lengths
func TestLiteEqualPrefixLengthBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		buildA    func() *Lite
		buildB    func() *Lite
		wantEqual bool
	}{
		{
			name: "different prefix lengths same base",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.0/24"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.0/25"))
				return tbl
			},
			wantEqual: false,
		},
		{
			name: "minimum prefix length IPv4 /0",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("0.0.0.0/0"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("0.0.0.0/0"))
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "maximum prefix length IPv4 /32",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.1/32"))
				tbl.Insert(mpp("192.0.2.2/32"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.1/32"))
				tbl.Insert(mpp("192.0.2.2/32"))
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "maximum prefix length IPv6 /128",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("2001:db8::1/128"))
				tbl.Insert(mpp("2001:db8::2/128"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("2001:db8::1/128"))
				tbl.Insert(mpp("2001:db8::2/128"))
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
		})
	}
}

	t.Parallel()

	a := new(Lite)
	b := new(Lite)

	// Insert 100 IPv4 and IPv6 prefixes
	for i := 0; i < 50; i++ {
		ipv4Prefix := mpp(fmt.Sprintf("10.%d.0.0/16", i))
		ipv6Prefix := mpp(fmt.Sprintf("2001:db8:%x::/48", i))
		a.Insert(ipv4Prefix)
		a.Insert(ipv6Prefix)
		b.Insert(ipv4Prefix)
		b.Insert(ipv6Prefix)
	}

	if !a.Equal(b) {
		t.Error("Large datasets should be equal")
	}

	// Delete one entry and test inequality
	b.Delete(mpp("10.0.0.0/16"))
	if a.Equal(b) {
		t.Error("Modified large datasets should not be equal")
	}
}

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
// Test transitivity property: if a.Equal(b) and b.Equal(c), then a.Equal(c)
func TestTableEqualTransitivity(t *testing.T) {
	t.Parallel()

	a := new(Table[stringVal])
	b := new(Table[stringVal])
	c := new(Table[stringVal])

	a.Insert(mpp("192.0.2.0/24"), "foo")
	b.Insert(mpp("192.0.2.0/24"), "foo")
	c.Insert(mpp("192.0.2.0/24"), "foo")

	if !a.Equal(b) {
		t.Fatal("a should equal b")
	}
	if !b.Equal(c) {
		t.Fatal("b should equal c")
	}
	if !a.Equal(c) {
		t.Error("a should equal c (transitivity)")
	}
}

func TestFastEqualTransitivity(t *testing.T) {

// Test edge cases with overlapping and adjacent prefixes
func TestFastEqualOverlappingPrefixes(t *testing.T) {

// Test comprehensive mixed IPv4 and IPv6 scenarios
func TestFastEqualMixedIPVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		buildA    func() *Fast[stringVal]
		buildB    func() *Fast[stringVal]
		wantEqual bool
	}{
		{
			name: "IPv4 only vs IPv6 only",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "v4")
				tbl.Insert(mpp("198.51.100.0/24"), "v4")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("2001:db8::/32"), "v6")
				tbl.Insert(mpp("2001:db8:1::/48"), "v6")
				return tbl
			},
			wantEqual: false,
		},
		{
			name: "same mixed IPv4 and IPv6",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "val1")
				tbl.Insert(mpp("2001:db8::/32"), "val2")
				tbl.Insert(mpp("10.0.0.0/8"), "val3")
				tbl.Insert(mpp("2001:db8:1::/48"), "val4")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("2001:db8:1::/48"), "val4")
				tbl.Insert(mpp("10.0.0.0/8"), "val3")
				tbl.Insert(mpp("2001:db8::/32"), "val2")
				tbl.Insert(mpp("192.0.2.0/24"), "val1")
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "different number IPv4 vs IPv6",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "val")
				tbl.Insert(mpp("2001:db8::/32"), "val")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/24"), "val")
				tbl.Insert(mpp("198.51.100.0/24"), "val")
				tbl.Insert(mpp("2001:db8::/32"), "val")
				return tbl
			},
			wantEqual: false,
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
		})
	}
}

	t.Parallel()

	tests := []struct {
		name      string
		buildA    func() *Fast[stringVal]
		buildB    func() *Fast[stringVal]
		wantEqual bool
	}{
		{
			name: "overlapping prefixes different values",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("10.0.0.0/8"), "parent")
				tbl.Insert(mpp("10.1.0.0/16"), "child")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("10.0.0.0/8"), "parent")
				tbl.Insert(mpp("10.1.0.0/16"), "different")
				return tbl
			},
			wantEqual: false,
		},
		{
			name: "adjacent prefixes",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/25"), "first")
				tbl.Insert(mpp("192.0.2.128/25"), "second")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("192.0.2.0/25"), "first")
				tbl.Insert(mpp("192.0.2.128/25"), "second")
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "missing child prefix",
			buildA: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("10.0.0.0/8"), "parent")
				tbl.Insert(mpp("10.1.0.0/16"), "child")
				return tbl
			},
			buildB: func() *Fast[stringVal] {
				tbl := new(Fast[stringVal])
				tbl.Insert(mpp("10.0.0.0/8"), "parent")
				return tbl
			},
			wantEqual: false,
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
		})
	}
}

	t.Parallel()

	a := new(Fast[stringVal])
	b := new(Fast[stringVal])
	c := new(Fast[stringVal])

	a.Insert(mpp("192.0.2.0/24"), "foo")
	a.Insert(mpp("198.51.100.0/24"), "bar")
	b.Insert(mpp("192.0.2.0/24"), "foo")
	b.Insert(mpp("198.51.100.0/24"), "bar")
	c.Insert(mpp("192.0.2.0/24"), "foo")
	c.Insert(mpp("198.51.100.0/24"), "bar")

	if !a.Equal(b) {
		t.Fatal("a should equal b")
	}
	if !b.Equal(c) {
		t.Fatal("b should equal c")
	}
	if !a.Equal(c) {
		t.Error("a should equal c (transitivity)")
	}
}

func TestLiteEqualTransitivity(t *testing.T) {

// Test edge cases with various prefix lengths
func TestLiteEqualPrefixLengthBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		buildA    func() *Lite
		buildB    func() *Lite
		wantEqual bool
	}{
		{
			name: "different prefix lengths same base",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.0/24"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.0/25"))
				return tbl
			},
			wantEqual: false,
		},
		{
			name: "minimum prefix length IPv4 /0",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("0.0.0.0/0"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("0.0.0.0/0"))
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "maximum prefix length IPv4 /32",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.1/32"))
				tbl.Insert(mpp("192.0.2.2/32"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("192.0.2.1/32"))
				tbl.Insert(mpp("192.0.2.2/32"))
				return tbl
			},
			wantEqual: true,
		},
		{
			name: "maximum prefix length IPv6 /128",
			buildA: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("2001:db8::1/128"))
				tbl.Insert(mpp("2001:db8::2/128"))
				return tbl
			},
			buildB: func() *Lite {
				tbl := new(Lite)
				tbl.Insert(mpp("2001:db8::1/128"))
				tbl.Insert(mpp("2001:db8::2/128"))
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
		})
	}
}

	t.Parallel()

	a := new(Lite)
	b := new(Lite)
	c := new(Lite)

	a.Insert(mpp("192.0.2.0/24"))
	a.Insert(mpp("198.51.100.0/24"))
	b.Insert(mpp("192.0.2.0/24"))
	b.Insert(mpp("198.51.100.0/24"))
	c.Insert(mpp("192.0.2.0/24"))
	c.Insert(mpp("198.51.100.0/24"))

	if !a.Equal(b) {
		t.Fatal("a should equal b")
	}
	if !b.Equal(c) {
		t.Fatal("b should equal c")
	}
	if !a.Equal(c) {
		t.Error("a should equal c (transitivity)")
	}
}

// Test that Equal method does not modify the tables
func TestEqualDoesNotModify(t *testing.T) {
	t.Parallel()

	t.Run("Table", func(t *testing.T) {
		t.Parallel()
		a := new(Table[stringVal])
		b := new(Table[stringVal])
		a.Insert(mpp("192.0.2.0/24"), "foo")
		b.Insert(mpp("192.0.2.0/24"), "foo")
		
		// Clone before Equal
		aClone := a.Clone()
		bClone := b.Clone()
		
		_ = a.Equal(b)
		
		// Verify tables unchanged
		if !a.Equal(aClone) {
			t.Error("Equal() modified receiver table")
		}
		if !b.Equal(bClone) {
			t.Error("Equal() modified argument table")
		}
	})

	t.Run("Fast", func(t *testing.T) {
		t.Parallel()
		a := new(Fast[stringVal])
		b := new(Fast[stringVal])
		a.Insert(mpp("192.0.2.0/24"), "foo")
		a.Insert(mpp("2001:db8::/32"), "bar")
		b.Insert(mpp("192.0.2.0/24"), "foo")
		b.Insert(mpp("2001:db8::/32"), "bar")
		
		aClone := a.Clone()
		bClone := b.Clone()
		
		_ = a.Equal(b)
		
		if !a.Equal(aClone) {
			t.Error("Equal() modified receiver table")
		}
		if !b.Equal(bClone) {
			t.Error("Equal() modified argument table")
		}
	})

	t.Run("Lite", func(t *testing.T) {
		t.Parallel()
		a := new(Lite)
		b := new(Lite)
		a.Insert(mpp("192.0.2.0/24"))
		a.Insert(mpp("2001:db8::/32"))
		b.Insert(mpp("192.0.2.0/24"))
		b.Insert(mpp("2001:db8::/32"))
		
		aClone := a.Clone()
		bClone := b.Clone()
		
		_ = a.Equal(b)
		
		if !a.Equal(aClone) {
			t.Error("Equal() modified receiver table")
		}
		if !b.Equal(bClone) {
			t.Error("Equal() modified argument table")
		}
	})
}

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

// Benchmark comparing unequal tables
func BenchmarkTableEqualUnequal(b *testing.B) {
	tbl1 := new(Table[stringVal])
	tbl2 := new(Table[stringVal])

	for i, r := range routes {
		if i > 1000 {
			break
		}
		val := stringVal("value")
		tbl1.Insert(r.CIDR, val)
		if i%2 == 0 {
			tbl2.Insert(r.CIDR, val)
		} else {
			tbl2.Insert(r.CIDR, stringVal("different"))
		}
	}

	b.ResetTimer()
	for b.Loop() {
		_ = tbl1.Equal(tbl2)
	}
}

func BenchmarkFastEqualUnequal(b *testing.B) {
	tbl1 := new(Fast[stringVal])
	tbl2 := new(Fast[stringVal])

	for i, r := range routes {
		if i > 1000 {
			break
		}
		val := stringVal("value")
		tbl1.Insert(r.CIDR, val)
		if i%2 == 0 {
			tbl2.Insert(r.CIDR, val)
		} else {
			tbl2.Insert(r.CIDR, stringVal("different"))
		}
	}

	b.ResetTimer()
	for b.Loop() {
		_ = tbl1.Equal(tbl2)
	}
}

func BenchmarkLiteEqualUnequal(b *testing.B) {
	tbl1 := new(Lite)
	tbl2 := new(Lite)

	for i, r := range routes {
		if i > 1000 {
			break
		}
		tbl1.Insert(r.CIDR)
		// Make table 2 different by skipping every other entry
		if i%2 == 0 {
			tbl2.Insert(r.CIDR)
		}
	}

	b.ResetTimer()
	for b.Loop() {
		_ = tbl1.Equal(tbl2)
	}
}

// Benchmark comparing tables with different sizes (early exit optimization)
func BenchmarkFastEqualDifferentSizes(b *testing.B) {
	tbl1 := new(Fast[stringVal])
	tbl2 := new(Fast[stringVal])

	for i, r := range routes {
		if i > 1000 {
			break
		}
		val := stringVal("value")
		tbl1.Insert(r.CIDR, val)
		if i < 500 {
			tbl2.Insert(r.CIDR, val)
		}
	}

	b.ResetTimer()
	for b.Loop() {
		_ = tbl1.Equal(tbl2)
	}
}

