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

type zeroStructT = struct{}

var zeroStruct zeroStructT

type tablerTiny interface {
	Insert(netip.Prefix, zeroStructT)
	dump(io.Writer)
	dumpString() string
}

var tables = map[string]func() tablerTiny{
	"Table":     func() tablerTiny { return new(Table[zeroStructT]) },
	"Fast":      func() tablerTiny { return new(Fast[zeroStructT]) },
	"liteTable": func() tablerTiny { return new(liteTable[zeroStructT]) },
}

func insertAll(tbl tablerTiny, pfxs []netip.Prefix) {
	for _, p := range pfxs {
		tbl.Insert(p, zeroStruct)
	}
}

// Test unified tables dumping using zero values
//
//nolint:tparallel
func TestUnifiedDumper_TableTypes(t *testing.T) {
	t.Parallel()

	pfxs := []netip.Prefix{
		mpp("192.168.1.0/24"),
		mpp("2001:db8::/32"),
	}

	for name, builder := range tables {
		t.Run(name+"_EmptyTableDump", func(t *testing.T) {
			t.Parallel()

			tbl := builder()

			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s table dump panicked: %v", name, r)
				}
			}()

			var buf strings.Builder
			tbl.dump(&buf)
		})

		t.Run(name+"_WithDataDump", func(t *testing.T) {
			t.Parallel()

			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s table dump panicked: %v", name, r)
				}
			}()

			tbl := builder()
			insertAll(tbl, pfxs)

			var buf strings.Builder
			tbl.dump(&buf)
		})

		t.Run(name+"_EmptyTableDumpString", func(t *testing.T) {
			t.Parallel()

			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s table dump panicked: %v", name, r)
				}
			}()

			tbl := builder()
			_ = tbl.dumpString()
		})

		t.Run(name+"_IPv4AndIPv6Headers", func(t *testing.T) {
			t.Parallel()

			tbl := builder()
			insertAll(tbl, pfxs)

			var buf strings.Builder
			tbl.dump(&buf)

			out := buf.String()
			if !strings.Contains(out, "### IPv4: size(1)") {
				t.Fatalf("%s table dump, missing IPv4 header: %q", name, out)
			}
			if !strings.Contains(out, "### IPv6: size(1)") {
				t.Fatalf("%s table dump, missing IPv6 header: %q", name, out)
			}
		})
	}
}

func TestUnifiedDumper_DefaultRouteV4(t *testing.T) {
	t.Parallel()

	pfxs := []netip.Prefix{
		mpp("0.0.0.0/0"),
	}

	want := `
### IPv4: size(1), nodes(1), pfxs(1), leaves(0), fringes(0)
[STOP] depth:  0 path: [] / 0
indexs(#1): [1]
prefxs(#1): 0.0.0.0/0
`
	for name, builder := range tables {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tbl := builder()
			insertAll(tbl, pfxs)

			if err := checkDump(tbl, want); err != nil {
				t.Errorf("%s table dump: %v", name, err)
			}
		})
	}
}

func TestUnifiedDumper_DefaultRouteV6(t *testing.T) {
	t.Parallel()

	pfxs := []netip.Prefix{
		mpp("::/0"),
	}

	want := `
### IPv6: size(1), nodes(1), pfxs(1), leaves(0), fringes(0)
[STOP] depth:  0 path: [] / 0
indexs(#1): [1]
prefxs(#1): ::/0
`
	for name, builder := range tables {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tbl := builder()
			insertAll(tbl, pfxs)

			if err := checkDump(tbl, want); err != nil {
				t.Errorf("%s table dump: %v", name, err)
			}
		})
	}
}

func TestUnifiedDumper_SampleV4(t *testing.T) {
	t.Parallel()

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

	for name, builder := range tables {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tbl := builder()
			insertAll(tbl, pfxs)

			if err := checkDump(tbl, want); err != nil {
				t.Errorf("%s table dump: %v", name, err)
			}
		})
	}
}

func TestUnifiedDumper_SampleV6(t *testing.T) {
	t.Parallel()

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

	for name, builder := range tables {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tbl := builder()
			insertAll(tbl, pfxs)

			if err := checkDump(tbl, want); err != nil {
				t.Errorf("%s table dump: %v", name, err)
			}
		})
	}
}

func TestUnifiedDumper_Sample(t *testing.T) {
	t.Parallel()

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

	for name, builder := range tables {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tbl := builder()
			insertAll(tbl, pfxs)

			if err := checkDump(tbl, want); err != nil {
				t.Errorf("%s table dump: %v", name, err)
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
