package bart

import (
	"net/netip"
	"testing"
)

func TestLiteModifySemantics(t *testing.T) {
	t.Parallel()

	type args struct {
		pfx netip.Prefix
		cb  func(exists bool) (del bool)
	}

	type want struct {
		deleted bool
		present bool // whether prefix should exist after operation
	}

	tests := []struct {
		name     string
		prepare  []string // prefixes to pre-populate
		args     args
		want     want
		finalSet []string // expected final set of prefixes
	}{
		{
			name:    "Insert new IPv4 prefix",
			prepare: []string{"10.0.0.0/8"},
			args: args{
				pfx: mpp("192.168.1.0/24"),
				cb: func(found bool) (del bool) {
					if found {
						t.Error("should not be found for new prefix")
					}
					return false // insert
				},
			},
			want:     want{deleted: false, present: true},
			finalSet: []string{"10.0.0.0/8", "192.168.1.0/24"},
		},

		{
			name:    "Insert new IPv6 prefix",
			prepare: []string{"2001:db8::/32"},
			args: args{
				pfx: mpp("fe80::/64"),
				cb: func(found bool) (del bool) {
					if found {
						t.Error("should not be found for new prefix")
					}
					return false // insert
				},
			},
			want:     want{deleted: false, present: true},
			finalSet: []string{"2001:db8::/32", "fe80::/64"},
		},

		{
			name:    "Delete existing IPv4 prefix",
			prepare: []string{"192.168.1.0/24", "10.0.0.0/8"},
			args: args{
				pfx: mpp("192.168.1.0/24"),
				cb: func(found bool) (del bool) {
					if !found {
						t.Error("should be found for existing prefix")
					}
					return true // delete
				},
			},
			want:     want{deleted: true, present: false},
			finalSet: []string{"10.0.0.0/8"},
		},

		{
			name:    "Delete existing IPv6 prefix",
			prepare: []string{"2001:db8::/32", "fe80::/64"},
			args: args{
				pfx: mpp("2001:db8::/32"),
				cb: func(found bool) (del bool) {
					if !found {
						t.Error("should be found for existing prefix")
					}
					return true // delete
				},
			},
			want:     want{deleted: true, present: false},
			finalSet: []string{"fe80::/64"},
		},

		{
			name:    "No-op on existing prefix",
			prepare: []string{"192.168.1.0/24"},
			args: args{
				pfx: mpp("192.168.1.0/24"),
				cb: func(found bool) (del bool) {
					if !found {
						t.Error("should be found")
					}
					return false // keep existing
				},
			},
			want:     want{deleted: false, present: true},
			finalSet: []string{"192.168.1.0/24"},
		},

		{
			name:    "No-op on non-existing prefix",
			prepare: []string{"10.0.0.0/8"},
			args: args{
				pfx: mpp("172.16.0.0/12"),
				cb: func(found bool) (del bool) {
					if found {
						t.Error("should not be found")
					}
					return true // no insert (del=true for no-op)
				},
			},
			want:     want{deleted: false, present: false},
			finalSet: []string{"10.0.0.0/8"},
		},

		{
			name:    "Delete non-existing prefix",
			prepare: []string{"10.0.0.0/8"},
			args: args{
				pfx: mpp("172.16.0.0/12"),
				cb: func(found bool) (del bool) {
					if found {
						t.Error("should not be found")
					}
					return true // attempt delete (no-op)
				},
			},
			want:     want{deleted: false, present: false},
			finalSet: []string{"10.0.0.0/8"},
		},

		// Edge cases
		{
			name:    "Insert IPv4 root",
			prepare: []string{},
			args: args{
				pfx: mpp("0.0.0.0/0"),
				cb: func(found bool) (del bool) {
					return false // insert
				},
			},
			want:     want{deleted: false, present: true},
			finalSet: []string{"0.0.0.0/0"},
		},

		{
			name:    "Insert IPv6 root",
			prepare: []string{},
			args: args{
				pfx: mpp("::/0"),
				cb: func(found bool) (del bool) {
					return false // insert
				},
			},
			want:     want{deleted: false, present: true},
			finalSet: []string{"::/0"},
		},

		{
			name:    "Insert IPv4 host route",
			prepare: []string{},
			args: args{
				pfx: mpp("192.168.1.1/32"),
				cb: func(found bool) (del bool) {
					return false // insert
				},
			},
			want:     want{deleted: false, present: true},
			finalSet: []string{"192.168.1.1/32"},
		},

		{
			name:    "Insert IPv6 host route",
			prepare: []string{},
			args: args{
				pfx: mpp("2001:db8::1/128"),
				cb: func(found bool) (del bool) {
					return false // insert
				},
			},
			want:     want{deleted: false, present: true},
			finalSet: []string{"2001:db8::1/128"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lite := &Lite{}

			// Setup: Insert initial prefixes using Modify
			for _, pfxStr := range tt.prepare {
				pfx := mpp(pfxStr)
				lite.Modify(pfx, func(exists bool) (del bool) {
					return false // insert
				})
			}

			// Execute the test operation
			lite.Modify(tt.args.pfx, tt.args.cb)

			// Verify final table state
			finalPrefixSet := make(map[netip.Prefix]bool)
			for _, pfxStr := range tt.finalSet {
				finalPrefixSet[mpp(pfxStr)] = true
			}

			// Check expected prefixes exist
			for pfxStr := range finalPrefixSet {
				if found := lite.Get(pfxStr); !found {
					t.Errorf("Expected prefix %v not found in table", pfxStr)
				}
			}

			// Check target prefix presence
			targetFound := lite.Get(tt.args.pfx)
			if targetFound != tt.want.present {
				t.Errorf("Target prefix %v presence = %v, want %v",
					tt.args.pfx, targetFound, tt.want.present)
			}

			// Verify table size
			expectedSize := len(tt.finalSet)
			if lite.Size() != expectedSize {
				t.Errorf("Size() = %v, want %v", lite.Size(), expectedSize)
			}
		})
	}
}

func TestLiteModifyInvalidPrefix(t *testing.T) {
	t.Parallel()

	lite := &Lite{}

	// Test with invalid prefix - should be no-op, callback not called
	invalidPrefix := netip.Prefix{} // zero value is invalid
	callbackInvoked := false

	lite.Modify(invalidPrefix, func(found bool) bool {
		callbackInvoked = true
		return false
	})

	if callbackInvoked {
		t.Error("callback should not be invoked for invalid prefix")
	}
}

// Test edge cases specific to Lite
func TestLiteModifyEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("callback_panic_recovery", func(t *testing.T) {
		t.Parallel()

		lite := &Lite{}
		lite.Modify(mpp("192.168.1.0/24"), func(bool) bool {
			return false
		})

		// Test that panicking callback doesn't corrupt the table
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic from callback")
			}

			// Verify table is still consistent after panic
			if found := lite.Get(mpp("192.168.1.0/24")); !found {
				t.Error("table corrupted after callback panic")
			}
		}()

		lite.Modify(mpp("192.168.1.0/24"), func(bool) bool {
			panic("intentional panic for testing")
		})
	})

	t.Run("overlapping_prefixes", func(t *testing.T) {
		t.Parallel()

		lite := &Lite{}

		// Insert overlapping prefixes
		prefixes := []string{
			"192.168.0.0/16",
			"192.168.1.0/24",
			"192.168.1.1/32",
		}

		for _, pfxStr := range prefixes {
			lite.Modify(mpp(pfxStr), func(bool) bool {
				return false // insert
			})
		}

		// Delete the middle prefix
		lite.Modify(mpp("192.168.1.0/24"), func(found bool) bool {
			if !found {
				t.Error("expected to find middle prefix")
			}
			return true // delete
		})

		// Verify other prefixes still exist
		expected := []string{"192.168.0.0/16", "192.168.1.1/32"}
		for _, pfxStr := range expected {
			if found := lite.Get(mpp(pfxStr)); !found {
				t.Errorf("prefix %s should still exist", pfxStr)
			}
		}

		// Verify deleted prefix is gone
		if found := lite.Get(mpp("192.168.1.0/24")); found {
			t.Error("deleted prefix should not exist")
		}
	})

	t.Run("empty_table_operations", func(t *testing.T) {
		t.Parallel()

		lite := &Lite{}

		// Try to delete from empty table
		lite.Modify(mpp("10.0.0.0/8"), func(found bool) bool {
			if found {
				t.Error("should not find anything in empty table")
			}
			return true // attempt delete
		})

		if lite.Size() != 0 {
			t.Error("table should remain empty")
		}
	})
}

func TestTablesModifySemantics(t *testing.T) {
	t.Parallel()

	type args struct {
		pfx netip.Prefix
		cb  func(val int, found bool) (_ int, del bool)
	}

	type want struct {
		val     int
		deleted bool
	}

	tests := []struct {
		name      string
		prepare   map[netip.Prefix]int // entries to pre-populate the table
		args      args
		want      want
		finalData map[netip.Prefix]int // expected table contents after the operation
	}{
		{
			name:    "Delete existing IPv4 entry",
			prepare: map[netip.Prefix]int{mpp("192.168.1.0/24"): 100, mpp("10.0.0.0/8"): 200},
			args: args{
				pfx: mpp("192.168.1.0/24"),
				cb:  func(val int, found bool) (_ int, del bool) { return 0, true },
			},
			want:      want{val: 100, deleted: true},
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): 200},
		},

		{
			name:    "Delete existing IPv6 entry",
			prepare: map[netip.Prefix]int{mpp("2001:db8::/32"): 300, mpp("fe80::/64"): 400},
			args: args{
				pfx: mpp("2001:db8::/32"),
				cb:  func(val int, found bool) (_ int, del bool) { return 0, true },
			},
			want:      want{val: 300, deleted: true},
			finalData: map[netip.Prefix]int{mpp("fe80::/64"): 400},
		},

		{
			name:    "Insert new IPv4 entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 100},
			args: args{
				pfx: mpp("192.168.0.0/16"),
				cb:  func(val int, found bool) (_ int, del bool) { return 500, false },
			},
			want:      want{val: 500, deleted: false},
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): 100, mpp("192.168.0.0/16"): 500},
		},

		{
			name:    "Insert new IPv6 entry",
			prepare: map[netip.Prefix]int{mpp("2001:db8::/32"): 300},
			args: args{
				pfx: mpp("2001:db8:1::/48"),
				cb:  func(val int, found bool) (_ int, del bool) { return 600, false },
			},
			want:      want{val: 600, deleted: false},
			finalData: map[netip.Prefix]int{mpp("2001:db8::/32"): 300, mpp("2001:db8:1::/48"): 600},
		},

		{
			// For update, the callback gets oldVal, returns newVal, but Modify returns oldVal
			name:    "Update existing IPv4 entry",
			prepare: map[netip.Prefix]int{mpp("192.168.1.0/24"): 100, mpp("10.0.0.0/8"): 200},
			args: args{
				pfx: mpp("192.168.1.0/24"),
				cb:  func(val int, found bool) (_ int, del bool) { return 999, false },
			},
			want:      want{val: 100, deleted: false},                                           // Returns OLD value!
			finalData: map[netip.Prefix]int{mpp("192.168.1.0/24"): 999, mpp("10.0.0.0/8"): 200}, // But stores NEW value
		},

		{
			name:    "Update existing IPv6 entry",
			prepare: map[netip.Prefix]int{mpp("2001:db8::/32"): 300, mpp("fe80::/64"): 400},
			args: args{
				pfx: mpp("2001:db8::/32"),
				cb:  func(val int, found bool) (_ int, del bool) { return 777, false },
			},
			want:      want{val: 300, deleted: false},                                         // Returns OLD value
			finalData: map[netip.Prefix]int{mpp("2001:db8::/32"): 777, mpp("fe80::/64"): 400}, // Stores NEW value
		},

		{
			name:    "No-op on missing IPv4 entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 200},
			args: args{
				pfx: mpp("172.16.0.0/12"),
				cb:  func(val int, found bool) (_ int, del bool) { return 0, true },
			},
			want:      want{val: 0, deleted: false}, // Cannot delete what doesn't exist
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): 200},
		},

		{
			name:    "No-op on missing IPv6 entry",
			prepare: map[netip.Prefix]int{mpp("2001:db8::/32"): 300},
			args: args{
				pfx: mpp("2001:db8:1::/48"),
				cb:  func(val int, found bool) (_ int, del bool) { return 0, true },
			},
			want:      want{val: 0, deleted: false},
			finalData: map[netip.Prefix]int{mpp("2001:db8::/32"): 300},
		},

		{
			name:    "No-op existing entry (return same value)",
			prepare: map[netip.Prefix]int{mpp("192.168.1.0/24"): 100},
			args: args{
				pfx: mpp("192.168.1.0/24"),
				cb:  func(val int, found bool) (_ int, del bool) { return val, false },
			},
			want:      want{val: 100, deleted: false},
			finalData: map[netip.Prefix]int{mpp("192.168.1.0/24"): 100},
		},

		{
			name:    "No-op non-existing entry (return zero, don't insert)",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 200},
			args: args{
				pfx: mpp("172.16.0.0/12"),
				cb:  func(val int, found bool) (_ int, del bool) { return 0, true },
			},
			want:      want{val: 0, deleted: false},
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): 200},
		},

		// Edge cases: root prefixes
		{
			name:    "Insert IPv4 root prefix",
			prepare: map[netip.Prefix]int{},
			args: args{
				pfx: mpp("0.0.0.0/0"),
				cb:  func(val int, found bool) (_ int, del bool) { return 1000, false },
			},
			want:      want{val: 1000, deleted: false},
			finalData: map[netip.Prefix]int{mpp("0.0.0.0/0"): 1000},
		},

		{
			name:    "Insert IPv6 root prefix",
			prepare: map[netip.Prefix]int{},
			args: args{
				pfx: mpp("::/0"),
				cb:  func(val int, found bool) (_ int, del bool) { return 2000, false },
			},
			want:      want{val: 2000, deleted: false},
			finalData: map[netip.Prefix]int{mpp("::/0"): 2000},
		},

		// Edge cases: host routes
		{
			name:    "Insert IPv4 host route",
			prepare: map[netip.Prefix]int{},
			args: args{
				pfx: mpp("192.168.1.1/32"),
				cb:  func(val int, found bool) (_ int, del bool) { return 3000, false },
			},
			want:      want{val: 3000, deleted: false},
			finalData: map[netip.Prefix]int{mpp("192.168.1.1/32"): 3000},
		},

		{
			name:    "Insert IPv6 host route",
			prepare: map[netip.Prefix]int{},
			args: args{
				pfx: mpp("2001:db8::1/128"),
				cb:  func(val int, found bool) (_ int, del bool) { return 4000, false },
			},
			want:      want{val: 4000, deleted: false},
			finalData: map[netip.Prefix]int{mpp("2001:db8::1/128"): 4000},
		},

		// Zero value tests
		{
			name:    "Insert zero value",
			prepare: map[netip.Prefix]int{},
			args: args{
				pfx: mpp("192.168.1.0/24"),
				cb:  func(val int, found bool) (_ int, del bool) { return 0, false },
			},
			want:      want{val: 0, deleted: false},
			finalData: map[netip.Prefix]int{mpp("192.168.1.0/24"): 0},
		},

		{
			name:    "Update to zero value",
			prepare: map[netip.Prefix]int{mpp("192.168.1.0/24"): 100},
			args: args{
				pfx: mpp("192.168.1.0/24"),
				cb:  func(val int, found bool) (_ int, del bool) { return 0, false },
			},
			want:      want{val: 100, deleted: false},                 // Returns old value
			finalData: map[netip.Prefix]int{mpp("192.168.1.0/24"): 0}, // Stores new (zero) value
		},
		{
			name:    "Delete existing entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42, mpp("2001:db8::/32"): 4242},
			args: args{
				pfx: mpp("10.0.0.0/8"),
				cb:  func(val int, found bool) (_ int, del bool) { return 0, true },
			},
			want:      want{val: 42, deleted: true},
			finalData: map[netip.Prefix]int{mpp("2001:db8::/32"): 4242},
		},

		{
			name:    "Insert new entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42},
			args: args{
				pfx: mpp("2001:db8::/32"),
				cb:  func(val int, found bool) (_ int, del bool) { return 4242, false },
			},
			want:      want{val: 4242, deleted: false},
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42, mpp("2001:db8::/32"): 4242},
		},

		{
			// For update, the callback gets oldVal, returns newVal, but Modify returns oldVal
			name:    "Update existing entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42, mpp("2001:db8::/32"): 4242},
			args: args{
				pfx: mpp("10.0.0.0/8"),
				cb:  func(val int, found bool) (_ int, del bool) { return -1, false },
			},
			want:      want{val: 42, deleted: false},
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): -1, mpp("2001:db8::/32"): 4242},
		},

		{
			name:    "No-op on missing entry",
			prepare: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42},
			args: args{
				pfx: mpp("2001:db8::/32"),
				cb:  func(val int, found bool) (_ int, del bool) { return 0, true },
			},
			want:      want{val: 0, deleted: false},
			finalData: map[netip.Prefix]int{mpp("10.0.0.0/8"): 42},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test both Table and Fast (following the existing pattern)
			tableTypes := []struct {
				name    string
				builder func() interface {
					Modify(netip.Prefix, func(int, bool) (int, bool))
					Get(netip.Prefix) (int, bool)
				}
			}{
				{"Table", func() interface {
					Modify(netip.Prefix, func(int, bool) (int, bool))
					Get(netip.Prefix) (int, bool)
				} {
					return &Table[int]{}
				}},
				{"Fast", func() interface {
					Modify(netip.Prefix, func(int, bool) (int, bool))
					Get(netip.Prefix) (int, bool)
				} {
					return &Fast[int]{}
				}},
			}

			for _, tableType := range tableTypes {
				t.Run(tableType.name, func(t *testing.T) {
					t.Parallel()

					rt := tableType.builder()

					// Insert initial entries using Modify (following existing pattern)
					for pfx, v := range tt.prepare {
						rt.Modify(pfx, func(_ int, _ bool) (_ int, del bool) { return v, false })
					}

					// Execute the test operation
					rt.Modify(tt.args.pfx, tt.args.cb)

					// Check the final state of the table using Get (following existing pattern)
					for pfx, wantVal := range tt.finalData {
						gotVal, ok := rt.Get(pfx)
						if !ok || gotVal != wantVal {
							t.Errorf("[%s] final table: key %v = %v (ok=%v), want %v (ok=true)",
								tt.name, pfx, gotVal, ok, wantVal)
						}
					}

					// Ensure there are no unexpected entries (following existing pattern)
					for pfx := range tt.prepare {
						if _, expect := tt.finalData[pfx]; !expect {
							if _, ok := rt.Get(pfx); ok {
								t.Errorf("[%s] final table: key %v should not be present", tt.name, pfx)
							}
						}
					}

					// Also ensure the target prefix stays absent in no-op scenarios
					if _, expect := tt.finalData[tt.args.pfx]; !expect {
						if _, ok := rt.Get(tt.args.pfx); ok {
							t.Errorf("[%s] final table: key %v should not be present", tt.name, tt.args.pfx)
						}
					}
				})
			}
		})
	}
}

func TestTableModifyInvalidPrefix(t *testing.T) {
	t.Parallel()

	tableTypes := []struct {
		name    string
		builder func() interface {
			Modify(netip.Prefix, func(int, bool) (int, bool))
		}
	}{
		{"Table", func() interface {
			Modify(netip.Prefix, func(int, bool) (int, bool))
		} {
			return &Table[int]{}
		}},
		{"Fast", func() interface {
			Modify(netip.Prefix, func(int, bool) (int, bool))
		} {
			return &Fast[int]{}
		}},
	}

	for _, tt := range tableTypes {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			table := tt.builder()

			// Test with invalid prefix (should be no-op, callback not called)
			invalidPrefix := netip.Prefix{} // zero value is invalid
			callbackInvoked := false

			table.Modify(invalidPrefix, func(v int, found bool) (int, bool) {
				callbackInvoked = true
				return 42, false
			})

			if callbackInvoked {
				t.Error("callback should not be invoked for invalid prefix")
			}
		})
	}
}

// Test edge cases and error conditions
func TestModifyEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("callback_panic_recovery", func(t *testing.T) {
		t.Parallel()

		table := &Table[int]{}
		table.Modify(mpp("192.168.1.0/24"), func(_ int, _ bool) (int, bool) { return 100, false })

		// Test that panicking callback doesn't corrupt the table
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic from callback")
			}

			// Verify table is still consistent after panic
			if val, found := table.Get(mpp("192.168.1.0/24")); !found || val != 100 {
				t.Error("table corrupted after callback panic")
			}
		}()

		table.Modify(mpp("192.168.1.0/24"), func(val int, found bool) (int, bool) {
			panic("intentional panic for testing")
		})
	})

	t.Run("zero_values", func(t *testing.T) {
		t.Parallel()

		table := &Table[int]{}

		// Insert zero value
		table.Modify(mpp("192.168.1.0/24"), func(val int, found bool) (int, bool) {
			return 0, false // insert zero
		})

		// Verify zero value is stored and retrievable
		if lookupVal, found := table.Get(mpp("192.168.1.0/24")); !found || lookupVal != 0 {
			t.Errorf("zero value not stored correctly: got (%v, %v), want (0, true)", lookupVal, found)
		}
	})

	t.Run("large_values", func(t *testing.T) {
		t.Parallel()

		table := &Fast[int]{}
		largeValue := 1<<30 - 1 // Large but valid int

		table.Modify(mpp("10.0.0.0/8"), func(val int, found bool) (int, bool) {
			return largeValue, false
		})
	})

	t.Run("overlapping_prefixes", func(t *testing.T) {
		t.Parallel()

		table := &Table[int]{}

		// Insert overlapping prefixes and modify them
		prefixes := []struct {
			pfx string
			val int
		}{
			{"192.168.0.0/16", 1},
			{"192.168.1.0/24", 2},
			{"192.168.1.1/32", 3},
		}

		for _, p := range prefixes {
			table.Modify(mpp(p.pfx), func(_ int, _ bool) (int, bool) {
				return p.val, false
			})
		}

		// Update the middle prefix
		table.Modify(mpp("192.168.1.0/24"), func(val int, found bool) (int, bool) {
			if !found || val != 2 {
				t.Errorf("expected found=true, val=2, got found=%v, val=%v", found, val)
			}
			return 20, false // update
		})

		// Verify all prefixes still exist with correct values
		expected := map[string]int{
			"192.168.0.0/16": 1,
			"192.168.1.0/24": 20, // updated
			"192.168.1.1/32": 3,
		}

		for pfxStr, expectedVal := range expected {
			if gotVal, found := table.Get(mpp(pfxStr)); !found || gotVal != expectedVal {
				t.Errorf("prefix %s: got (%v, %v), want (%v, true)", pfxStr, gotVal, found, expectedVal)
			}
		}
	})
}

// Test that demonstrates Lite vs regular Table behavior
func TestLiteTableVsTableComparison(t *testing.T) {
	t.Parallel()

	prefix := mpp("192.168.1.0/24")

	// Test with regular Table
	regularTable := &Table[int]{}
	regularTable.Modify(prefix, func(val int, found bool) (int, bool) {
		return 42, false // insert with value
	})

	if val, found := regularTable.Get(prefix); !found || val != 42 {
		t.Errorf("Regular table should have value 42, got (%v, %v)", val, found)
	}

	// Test with Lite - no meaningful payload
	lite := &Lite{}
	lite.Modify(prefix, func(found bool) bool {
		return false // insert (no meaningful value)
	})

	if found := lite.Get(prefix); !found {
		t.Error("Lite table should have prefix present")
	}

	// Both tables should report the prefix as existing, but only regular table has a value
	if regularTable.Size() != 1 || lite.Size() != 1 {
		t.Error("Both tables should have size 1")
	}
}
