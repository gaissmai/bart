// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"fmt"
	"io"
	"net/netip"
	"strings"
	"testing"
)

// Test unified dumper functionality across node types
//
//nolint:tparallel
func TestUnifiedDumper_NodeTypes(t *testing.T) {
	t.Parallel()

	nodeBuilder := map[string]func() nodeReadWriter[any]{
		"node":     func() nodeReadWriter[any] { return &bartNode[any]{} },
		"fastNode": func() nodeReadWriter[any] { return &fastNode[any]{} },
		"liteNode": func() nodeReadWriter[any] { return &liteNode[any]{} },
	}

	for nodeTypeName, nodeBuilder := range nodeBuilder {
		t.Run(nodeTypeName+"_EmptyNodeStats", func(t *testing.T) {
			// Test nodeStats
			n := nodeBuilder()
			stats := nodeStats(n)
			// For empty nodes, stats should have reasonable values
			if stats.nodes < 0 || stats.pfxs < 0 {
				t.Errorf("Invalid stats for empty %s: %+v", nodeTypeName, stats)
			}
		})

		t.Run(nodeTypeName+"_WithPrefix", func(t *testing.T) {
			n := nodeBuilder()
			n.insertPrefix(128, "test-value")

			// Test that we can get stats after insertion
			stats := nodeStats(n)
			if stats.pfxs == 0 {
				t.Errorf("Expected non-zero prefixes after insertion in %s", nodeTypeName)
			}
		})

		t.Run(nodeTypeName+"_DumpToWriter", func(t *testing.T) {
			var buf strings.Builder
			path := stridePath{}

			n := nodeBuilder()
			n.insertPrefix(64, "dump-test")

			// Use the dump function that takes io.Writer
			dump(n, &buf, path, 0, false, shouldPrintValues[any]())

			output := buf.String()
			// Just check that it produces some output without panicking
			if output == "" {
				t.Errorf("Expected non-empty dump output for %s with data", nodeTypeName)
			}
		})
	}
}

// Test typed nil handling across node types
func TestUnifiedDumper_TypedNilHandling(t *testing.T) {
	t.Parallel()

	nodeBuilder := map[string]func() nodeReader[any]{
		"node":     func() nodeReader[any] { return (*bartNode[any])(nil) },
		"fastNode": func() nodeReader[any] { return (*fastNode[any])(nil) },
		"liteNode": func() nodeReader[any] { return (*liteNode[any])(nil) },
	}

	for nodeTypeName, createNilNode := range nodeBuilder {
		t.Run("TypedNil_"+nodeTypeName, func(t *testing.T) {
			t.Parallel()
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("dumpRec: panic with typed-nil %s: %v", nodeTypeName, r)
				}
			}()

			pn := createNilNode()
			w := new(strings.Builder)
			path := stridePath{}
			dumpRec(pn, w, path, 0, true, shouldPrintValues[any]())
		})
	}
}

// Test unified tables dumping using zero values
//
//nolint:tparallel
func TestUnifiedDumper_TableTypes(t *testing.T) {
	t.Parallel()

	type tabler interface {
		dump(io.Writer)
		dumpString() string
		String() string
	}

	pfxs := []netip.Prefix{
		mpp("192.168.1.0/24"),
		mpp("2001:db8::/32"),
	}

	testCases := []struct {
		name      string
		table     tabler
		setupData func(tabler)
	}{
		{
			name:  "Table",
			table: &Table[struct{}]{},
			setupData: func(tbl tabler) {
				foo := tbl.(*Table[struct{}])
				for _, pfx := range pfxs {
					foo.Insert(pfx, struct{}{})
				}
			},
		},
		{
			name:  "Fast",
			table: &Fast[struct{}]{},
			setupData: func(tbl tabler) {
				foo := tbl.(*Fast[struct{}])
				for _, pfx := range pfxs {
					foo.Insert(pfx, struct{}{})
				}
			},
		},
		{
			name:  "Lite",
			table: &Lite{},
			setupData: func(tbl tabler) {
				foo := tbl.(*Lite)
				for _, pfx := range pfxs {
					foo.Insert(pfx)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name+"_EmptyTableString", func(t *testing.T) {
			output := tc.table.String()
			if output != "" {
				t.Errorf("Expected empty string for empty %s table, got: %s", tc.name, output)
			}
		})

		t.Run(tc.name+"_WithDataString", func(t *testing.T) {
			tc.setupData(tc.table)

			output := tc.table.String()
			if !strings.Contains(output, pfxs[0].String()) || !strings.Contains(output, pfxs[1].String()) {
				t.Errorf("Expected %s table output to contain inserted routes, got: %s", tc.name, output)
			}
		})

		t.Run(tc.name+"_EmptyTableDump", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s table dump panicked: %v", tc.name, r)
				}
			}()

			var buf strings.Builder
			tc.table.dump(&buf)
		})

		t.Run(tc.name+"_WithDataDump", func(t *testing.T) {
			tc.setupData(tc.table)

			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s table dump panicked: %v", tc.name, r)
				}
			}()

			var buf strings.Builder
			tc.table.dump(&buf)
		})

		t.Run(tc.name+"_EmptyTableDumpString", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s table dump panicked: %v", tc.name, r)
				}
			}()

			_ = tc.table.dumpString()
		})

		t.Run(tc.name+"_IPv4AndIPv6Headers", func(t *testing.T) {
			tc.setupData(tc.table)

			var buf strings.Builder
			tc.table.dump(&buf)

			out := buf.String()
			if !strings.Contains(out, "### IPv4: size(1)") {
				t.Fatalf("%s table dump, missing IPv4 header: %q", tc.name, out)
			}
			if !strings.Contains(out, "### IPv6: size(1)") {
				t.Fatalf("%s table dump, missing IPv6 header: %q", tc.name, out)
			}
		})
	}
}

func TestUnifiedDumper_DefaultRouteV4(t *testing.T) {
	pfxs := []netip.Prefix{
		mpp("0.0.0.0/0"),
	}

	want := `
### IPv4: size(1), nodes(1), pfxs(1), leaves(0), fringes(0)
[STOP] depth:  0 path: [] / 0
indexs(#1): [1]
prefxs(#1): 0.0.0.0/0
`
	type zeroT = struct{}
	var zero zeroT

	type tabler[V zeroT] interface {
		Insert(netip.Prefix, V)
		dump(io.Writer)
	}

	type testCase[V zeroT] struct {
		name      string
		table     tabler[V]
		setupData func(tabler[V])
	}

	testCases := []testCase[zeroT]{
		{
			name:  "Table",
			table: &Table[zeroT]{},
			setupData: func(tbl tabler[zeroT]) {
				for _, pfx := range pfxs {
					tbl.Insert(pfx, zero)
				}
			},
		},
		{
			name:  "Fast",
			table: &Fast[zeroT]{},
			setupData: func(tbl tabler[zeroT]) {
				for _, pfx := range pfxs {
					tbl.Insert(pfx, zero)
				}
			},
		},
		{
			name:  "liteTable",
			table: &liteTable[zeroT]{},
			setupData: func(tbl tabler[zeroT]) {
				for _, pfx := range pfxs {
					tbl.Insert(pfx, zero)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupData(tc.table)
			if err := checkDump(tc.table, want); err != nil {
				t.Errorf("%T table dump: %v", tc.table, err)
			}
		})
	}
}

func TestUnifiedDumper_DefaultRouteV6(t *testing.T) {
	pfxs := []netip.Prefix{
		mpp("::/0"),
	}

	want := `
### IPv6: size(1), nodes(1), pfxs(1), leaves(0), fringes(0)
[STOP] depth:  0 path: [] / 0
indexs(#1): [1]
prefxs(#1): ::/0
`
	type zeroT = struct{}
	var zero zeroT

	type tabler[V zeroT] interface {
		Insert(netip.Prefix, V)
		dump(io.Writer)
	}

	type testCase[V zeroT] struct {
		name      string
		table     tabler[V]
		setupData func(tabler[V])
	}

	testCases := []testCase[zeroT]{
		{
			name:  "Table",
			table: &Table[zeroT]{},
			setupData: func(tbl tabler[zeroT]) {
				for _, pfx := range pfxs {
					tbl.Insert(pfx, zero)
				}
			},
		},
		{
			name:  "Fast",
			table: &Fast[zeroT]{},
			setupData: func(tbl tabler[zeroT]) {
				for _, pfx := range pfxs {
					tbl.Insert(pfx, zero)
				}
			},
		},
		{
			name:  "liteTable",
			table: &liteTable[zeroT]{},
			setupData: func(tbl tabler[zeroT]) {
				for _, pfx := range pfxs {
					tbl.Insert(pfx, zero)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupData(tc.table)
			if err := checkDump(tc.table, want); err != nil {
				t.Errorf("%T table dump: %v", tc.table, err)
			}
		})
	}
}

func TestUnifiedDumper_SampleV4(t *testing.T) {
	pfxs := []netip.Prefix{
		mpp("172.16.0.0/12"),
		mpp("10.0.0.0/24"),
		mpp("192.168.0.0/16"),
		mpp("10.0.0.0/8"),
		mpp("10.0.1.0/24"),
		mpp("169.254.0.0/16"),
		mpp("127.0.0.0/8"),
		mpp("127.0.0.1/32"),
		mpp("192.168.1.0/24"),
	}

	want := `
### IPv4: size(9), nodes(6), pfxs(3), leaves(3), fringes(3)
[HALF] depth:  0 path: [] / 0
octets(#5): [10 127 169 172 192]
leaves(#2): 169:{169.254.0.0/16} 172:{172.16.0.0/12}
childs(#3): 10 127 192

.[FULL] depth:  1 path: [10] / 8
.indexs(#1): [1]
.prefxs(#1): 10.0.0.0/8
.octets(#1): [0]
.childs(#1): 0

..[STOP] depth:  2 path: [10.0] / 16
..octets(#2): [0 1]
..fringe(#2): 0:{10.0.0.0/24} 1:{10.0.1.0/24}

.[STOP] depth:  1 path: [127] / 8
.indexs(#1): [1]
.prefxs(#1): 127.0.0.0/8
.octets(#1): [0]
.leaves(#1): 0:{127.0.0.1/32}

.[PATH] depth:  1 path: [192] / 8
.octets(#1): [168]
.childs(#1): 168

..[STOP] depth:  2 path: [192.168] / 16
..indexs(#1): [1]
..prefxs(#1): 192.168.0.0/16
..octets(#1): [1]
..fringe(#1): 1:{192.168.1.0/24}
`

	type zeroT = struct{}
	var zero zeroT

	type tabler[V zeroT] interface {
		Insert(netip.Prefix, V)
		dump(io.Writer)
	}

	type testCase[V zeroT] struct {
		name      string
		table     tabler[V]
		setupData func(tabler[V])
	}

	testCases := []testCase[zeroT]{
		{
			name:  "Table",
			table: &Table[zeroT]{},
			setupData: func(tbl tabler[zeroT]) {
				for _, pfx := range pfxs {
					tbl.Insert(pfx, zero)
				}
			},
		},
		{
			name:  "Fast",
			table: &Fast[zeroT]{},
			setupData: func(tbl tabler[zeroT]) {
				for _, pfx := range pfxs {
					tbl.Insert(pfx, zero)
				}
			},
		},
		{
			name:  "liteTable",
			table: &liteTable[zeroT]{},
			setupData: func(tbl tabler[zeroT]) {
				for _, pfx := range pfxs {
					tbl.Insert(pfx, zero)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupData(tc.table)
			if err := checkDump(tc.table, want); err != nil {
				t.Errorf("%T table dump: %v", tc.table, err)
			}
		})
	}
}

func TestUnifiedDumper_SampleV6(t *testing.T) {
	pfxs := []netip.Prefix{
		mpp("::/0"),
		mpp("fe80::/10"),
		mpp("2000::/3"),
		mpp("2001:db8::/32"),
	}

	want := `
### IPv6: size(4), nodes(1), pfxs(2), leaves(2), fringes(0)
[STOP] depth:  0 path: [] / 0
indexs(#2): [1 9]
prefxs(#2): ::/0 2000::/3
octets(#2): [32 254]
leaves(#2): 0x20:{2001:db8::/32} 0xfe:{fe80::/10}
`

	type zeroT = struct{}
	var zero zeroT

	type tabler[V zeroT] interface {
		Insert(netip.Prefix, V)
		dump(io.Writer)
	}

	type testCase[V zeroT] struct {
		name      string
		table     tabler[V]
		setupData func(tabler[V])
	}

	testCases := []testCase[zeroT]{
		{
			name:  "Table",
			table: &Table[zeroT]{},
			setupData: func(tbl tabler[zeroT]) {
				for _, pfx := range pfxs {
					tbl.Insert(pfx, zero)
				}
			},
		},
		{
			name:  "Fast",
			table: &Fast[zeroT]{},
			setupData: func(tbl tabler[zeroT]) {
				for _, pfx := range pfxs {
					tbl.Insert(pfx, zero)
				}
			},
		},
		{
			name:  "liteTable",
			table: &liteTable[zeroT]{},
			setupData: func(tbl tabler[zeroT]) {
				for _, pfx := range pfxs {
					tbl.Insert(pfx, zero)
				}
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupData(tc.table)
			if err := checkDump(tc.table, want); err != nil {
				t.Errorf("%T table dump: %v", tc.table, err)
			}
		})
	}
}

func TestUnifiedDumper_Sample(t *testing.T) {
	pfxs := []netip.Prefix{
		mpp("fe80::/10"),
		mpp("172.16.0.0/12"),
		mpp("10.0.0.0/24"),
		mpp("192.168.0.0/16"),
		mpp("10.0.0.0/8"),
		mpp("::/0"),
		mpp("10.0.1.0/24"),
		mpp("169.254.0.0/16"),
		mpp("2000::/3"),
		mpp("2001:db8::/32"),
		mpp("127.0.0.0/8"),
		mpp("127.0.0.1/32"),
		mpp("192.168.1.0/24"),
	}

	want := `
### IPv4: size(9), nodes(6), pfxs(3), leaves(3), fringes(3)
[HALF] depth:  0 path: [] / 0
octets(#5): [10 127 169 172 192]
leaves(#2): 169:{169.254.0.0/16} 172:{172.16.0.0/12}
childs(#3): 10 127 192

.[FULL] depth:  1 path: [10] / 8
.indexs(#1): [1]
.prefxs(#1): 10.0.0.0/8
.octets(#1): [0]
.childs(#1): 0

..[STOP] depth:  2 path: [10.0] / 16
..octets(#2): [0 1]
..fringe(#2): 0:{10.0.0.0/24} 1:{10.0.1.0/24}

.[STOP] depth:  1 path: [127] / 8
.indexs(#1): [1]
.prefxs(#1): 127.0.0.0/8
.octets(#1): [0]
.leaves(#1): 0:{127.0.0.1/32}

.[PATH] depth:  1 path: [192] / 8
.octets(#1): [168]
.childs(#1): 168

..[STOP] depth:  2 path: [192.168] / 16
..indexs(#1): [1]
..prefxs(#1): 192.168.0.0/16
..octets(#1): [1]
..fringe(#1): 1:{192.168.1.0/24}

### IPv6: size(4), nodes(1), pfxs(2), leaves(2), fringes(0)
[STOP] depth:  0 path: [] / 0
indexs(#2): [1 9]
prefxs(#2): ::/0 2000::/3
octets(#2): [32 254]
leaves(#2): 0x20:{2001:db8::/32} 0xfe:{fe80::/10}
`

	type zeroT = struct{}
	var zero zeroT

	type tabler[V zeroT] interface {
		Insert(netip.Prefix, V)
		dump(io.Writer)
	}

	type testCase[V zeroT] struct {
		name      string
		table     tabler[V]
		setupData func(tabler[V])
	}

	testCases := []testCase[zeroT]{
		{
			name:  "Table",
			table: &Table[zeroT]{},
			setupData: func(tbl tabler[zeroT]) {
				for _, pfx := range pfxs {
					tbl.Insert(pfx, zero)
				}
			},
		},
		{
			name:  "Fast",
			table: &Fast[zeroT]{},
			setupData: func(tbl tabler[zeroT]) {
				for _, pfx := range pfxs {
					tbl.Insert(pfx, zero)
				}
			},
		},
		{
			name:  "liteTable",
			table: &liteTable[zeroT]{},
			setupData: func(tbl tabler[zeroT]) {
				for _, pfx := range pfxs {
					tbl.Insert(pfx, zero)
				}
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupData(tc.table)
			if err := checkDump(tc.table, want); err != nil {
				t.Errorf("%T table dump: %v", tc.table, err)
			}
		})
	}
}

func checkDump(tbl interface{ dump(io.Writer) }, want string) error {
	w := new(strings.Builder)
	tbl.dump(w)

	got := w.String()
	if got == want {
		return nil
	}
	return fmt.Errorf("Dump got:\n%swant:\n%s", got, want)
}
