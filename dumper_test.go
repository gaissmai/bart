// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"strings"
	"testing"
)

type dumpTest struct {
	cidrs []netip.Prefix
	want  string
}

func TestDumpRec_MustNotPanic(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("dumpRec: panic occurred: %v", r)
		}
	}()

	w := new(strings.Builder)
	path := stridePath{}

	// This should not panic
	dumpRec[any](nil, w, path, 0, true)
}

func TestDumpRec_MustNotPanicWithTypedNilNode(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("dumpRec: panic with typed-nil: %v", r)
		}
	}()
	var pn *node[any]
	w := new(strings.Builder)
	path := stridePath{}
	dumpRec(nodeReader[any](pn), w, path, 0, true)
}

func TestDumpRec_MustNotPanicWithTypedNilFatNode(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("dumpRec: panic with typed-nil fatNode: %v", r)
		}
	}()
	var pn *fatNode[any]
	w := new(strings.Builder)
	path := stridePath{}
	dumpRec(nodeReader[any](pn), w, path, 0, true)
}

func TestDumperPanic(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Dump(nil) did not panic")
		}
	}()

	tbl := new(Table[any])
	tbl.Insert(mpp("1.2.3.4/32"), nil)
	tbl.dump(nil)
}

func TestDumperEmpty(t *testing.T) {
	t.Parallel()
	tbl := new(Table[any])
	checkDump(t, tbl, dumpTest{
		cidrs: nil,
		want:  "",
	})
}

func TestDumpDefaultRouteV4(t *testing.T) {
	t.Parallel()
	tbl := new(Table[any])
	checkDump(t, tbl, dumpTest{
		cidrs: []netip.Prefix{
			mpp("0.0.0.0/0"),
		},
		want: `
### IPv4: size(1), nodes(1), pfxs(1), leaves(0), fringes(0)
[STOP] depth:  0 path: [] / 0
indexs(#1): [1]
prefxs(#1): 0.0.0.0/0
values(#1): <nil>
`,
	})
}

func TestDumpDefaultRouteV6(t *testing.T) {
	t.Parallel()
	tbl := new(Table[any])
	checkDump(t, tbl, dumpTest{
		cidrs: []netip.Prefix{
			mpp("::/0"),
		},
		want: `
### IPv6: size(1), nodes(1), pfxs(1), leaves(0), fringes(0)
[STOP] depth:  0 path: [] / 0
indexs(#1): [1]
prefxs(#1): ::/0
values(#1): <nil>
`,
	})
}

func TestDumpSampleV4(t *testing.T) {
	t.Parallel()
	tbl := new(Table[any])

	checkDump(t, tbl, dumpTest{
		cidrs: []netip.Prefix{
			mpp("172.16.0.0/12"),
			mpp("10.0.0.0/24"),
			mpp("192.168.0.0/16"),
			mpp("10.0.0.0/8"),
			mpp("10.0.1.0/24"),
			mpp("169.254.0.0/16"),
			mpp("127.0.0.0/8"),
			mpp("127.0.0.1/32"),
			mpp("192.168.1.0/24"),
		},
		want: `
### IPv4: size(9), nodes(6), pfxs(3), leaves(3), fringes(3)
[HALF] depth:  0 path: [] / 0
octets(#5): [10 127 169 172 192]
leaves(#2): 169:{169.254.0.0/16, <nil>} 172:{172.16.0.0/12, <nil>}
childs(#3): 10 127 192

.[FULL] depth:  1 path: [10] / 8
.indexs(#1): [1]
.prefxs(#1): 10.0.0.0/8
.values(#1): <nil>
.octets(#1): [0]
.childs(#1): 0

..[STOP] depth:  2 path: [10.0] / 16
..octets(#2): [0 1]
..fringe(#2): 0:{10.0.0.0/24, <nil>} 1:{10.0.1.0/24, <nil>}

.[STOP] depth:  1 path: [127] / 8
.indexs(#1): [1]
.prefxs(#1): 127.0.0.0/8
.values(#1): <nil>
.octets(#1): [0]
.leaves(#1): 0:{127.0.0.1/32, <nil>}

.[PATH] depth:  1 path: [192] / 8
.octets(#1): [168]
.childs(#1): 168

..[STOP] depth:  2 path: [192.168] / 16
..indexs(#1): [1]
..prefxs(#1): 192.168.0.0/16
..values(#1): <nil>
..octets(#1): [1]
..fringe(#1): 1:{192.168.1.0/24, <nil>}
`,
	})
}

func TestDumpSampleV6(t *testing.T) {
	t.Parallel()
	tbl := new(Table[any])
	checkDump(t, tbl, dumpTest{
		cidrs: []netip.Prefix{
			mpp("::/0"),
			mpp("fe80::/10"),
			mpp("2000::/3"),
			mpp("2001:db8::/32"),
		},
		want: `
### IPv6: size(4), nodes(1), pfxs(2), leaves(2), fringes(0)
[STOP] depth:  0 path: [] / 0
indexs(#2): [1 9]
prefxs(#2): ::/0 2000::/3
values(#2): <nil> <nil>
octets(#2): [32 254]
leaves(#2): 0x20:{2001:db8::/32, <nil>} 0xfe:{fe80::/10, <nil>}
`,
	})
}

func TestDumpSample(t *testing.T) {
	t.Parallel()
	tbl := new(Table[any])

	checkDump(t, tbl, dumpTest{
		cidrs: []netip.Prefix{
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
		},
		want: `
### IPv4: size(9), nodes(6), pfxs(3), leaves(3), fringes(3)
[HALF] depth:  0 path: [] / 0
octets(#5): [10 127 169 172 192]
leaves(#2): 169:{169.254.0.0/16, <nil>} 172:{172.16.0.0/12, <nil>}
childs(#3): 10 127 192

.[FULL] depth:  1 path: [10] / 8
.indexs(#1): [1]
.prefxs(#1): 10.0.0.0/8
.values(#1): <nil>
.octets(#1): [0]
.childs(#1): 0

..[STOP] depth:  2 path: [10.0] / 16
..octets(#2): [0 1]
..fringe(#2): 0:{10.0.0.0/24, <nil>} 1:{10.0.1.0/24, <nil>}

.[STOP] depth:  1 path: [127] / 8
.indexs(#1): [1]
.prefxs(#1): 127.0.0.0/8
.values(#1): <nil>
.octets(#1): [0]
.leaves(#1): 0:{127.0.0.1/32, <nil>}

.[PATH] depth:  1 path: [192] / 8
.octets(#1): [168]
.childs(#1): 168

..[STOP] depth:  2 path: [192.168] / 16
..indexs(#1): [1]
..prefxs(#1): 192.168.0.0/16
..values(#1): <nil>
..octets(#1): [1]
..fringe(#1): 1:{192.168.1.0/24, <nil>}

### IPv6: size(4), nodes(1), pfxs(2), leaves(2), fringes(0)
[STOP] depth:  0 path: [] / 0
indexs(#2): [1 9]
prefxs(#2): ::/0 2000::/3
values(#2): <nil> <nil>
octets(#2): [32 254]
leaves(#2): 0x20:{2001:db8::/32, <nil>} 0xfe:{fe80::/10, <nil>}
`,
	})
}

func TestLiteDumpSample(t *testing.T) {
	t.Parallel()
	tbl := new(Lite)

	checkLiteDump(t, tbl, dumpTest{
		cidrs: []netip.Prefix{
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
		},
		want: `
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
`,
	})
}

func checkDump(t *testing.T, tbl *Table[any], tt dumpTest) {
	t.Helper()
	for _, cidr := range tt.cidrs {
		tbl.Insert(cidr, nil)
	}
	w := new(strings.Builder)
	tbl.dump(w)
	got := w.String()
	if tt.want != got {
		t.Errorf("Dump got:\n%swant:\n%s", got, tt.want)
	}
}

func checkLiteDump(t *testing.T, tbl *Lite, tt dumpTest) {
	t.Helper()
	for _, cidr := range tt.cidrs {
		tbl.Insert(cidr)
	}
	w := new(strings.Builder)
	tbl.dump(w)
	got := w.String()
	if tt.want != got {
		t.Errorf("Dump got:\n%swant:\n%s", got, tt.want)
	}
}

// -----------------------------------------------------------------------------
// Additional tests appended to strengthen coverage.
// Test framework: Go's standard library "testing" (no external test deps).
// -----------------------------------------------------------------------------

// Golden dumps reused to assert order-independence/determinism.

const wantV4SampleOrderInvariance = `
### IPv4: size(9), nodes(6), pfxs(3), leaves(3), fringes(3)
[HALF] depth:  0 path: [] / 0
octets(#5): [10 127 169 172 192]
leaves(#2): 169:{169.254.0.0/16, <nil>} 172:{172.16.0.0/12, <nil>}
childs(#3): 10 127 192

.[FULL] depth:  1 path: [10] / 8
.indexs(#1): [1]
.prefxs(#1): 10.0.0.0/8
.values(#1): <nil>
.octets(#1): [0]
.childs(#1): 0

..[STOP] depth:  2 path: [10.0] / 16
..octets(#2): [0 1]
..fringe(#2): 0:{10.0.0.0/24, <nil>} 1:{10.0.1.0/24, <nil>}

.[STOP] depth:  1 path: [127] / 8
.indexs(#1): [1]
.prefxs(#1): 127.0.0.0/8
.values(#1): <nil>
.octets(#1): [0]
.leaves(#1): 0:{127.0.0.1/32, <nil>}

.[PATH] depth:  1 path: [192] / 8
.octets(#1): [168]
.childs(#1): 168

..[STOP] depth:  2 path: [192.168] / 16
..indexs(#1): [1]
..prefxs(#1): 192.168.0.0/16
..values(#1): <nil>
..octets(#1): [1]
..fringe(#1): 1:{192.168.1.0/24, <nil>}
`

const wantV6SampleOrderInvariance = `
### IPv6: size(4), nodes(1), pfxs(2), leaves(2), fringes(0)
[STOP] depth:  0 path: [] / 0
indexs(#2): [1 9]
prefxs(#2): ::/0 2000::/3
values(#2): <nil> <nil>
octets(#2): [32 254]
leaves(#2): 0x20:{2001:db8::/32, <nil>} 0xfe:{fe80::/10, <nil>}
`

const wantLiteSampleOrderInvariance = `
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

// Validates that different insertion orders yield the exact same dump (deterministic formatting).
func TestDumpSampleV4_OrderIndependence(t *testing.T) {
	t.Parallel()
	tbl := new(Table[any])

	checkDump(t, tbl, dumpTest{
		cidrs: []netip.Prefix{
			mpp("127.0.0.1/32"),
			mpp("10.0.1.0/24"),
			mpp("192.168.1.0/24"),
			mpp("10.0.0.0/8"),
			mpp("172.16.0.0/12"),
			mpp("169.254.0.0/16"),
			mpp("10.0.0.0/24"),
			mpp("127.0.0.0/8"),
			mpp("192.168.0.0/16"),
		},
		want: wantV4SampleOrderInvariance,
	})
}

// Validates order-independence for IPv6 sample as well.
func TestDumpSampleV6_OrderIndependence(t *testing.T) {
	t.Parallel()
	tbl := new(Table[any])

	checkDump(t, tbl, dumpTest{
		cidrs: []netip.Prefix{
			mpp("2001:db8::/32"),
			mpp("2000::/3"),
			mpp("fe80::/10"),
			mpp("::/0"),
		},
		want: wantV6SampleOrderInvariance,
	})
}

// Ensures Lite.dump remains deterministic regardless of insertion order across mixed IPv4/IPv6.
func TestLiteDumpSample_OrderIndependence(t *testing.T) {
	t.Parallel()
	tbl := new(Lite)

	checkLiteDump(t, tbl, dumpTest{
		cidrs: []netip.Prefix{
			mpp("2001:db8::/32"),
			mpp("10.0.1.0/24"),
			mpp("127.0.0.1/32"),
			mpp("192.168.1.0/24"),
			mpp("172.16.0.0/12"),
			mpp("::/0"),
			mpp("192.168.0.0/16"),
			mpp("10.0.0.0/8"),
			mpp("169.254.0.0/16"),
			mpp("fe80::/10"),
			mpp("10.0.0.0/24"),
			mpp("127.0.0.0/8"),
			mpp("2000::/3"),
		},
		want: wantLiteSampleOrderInvariance,
	})
}

// The nil node interface must not write any output and must not panic.
func TestDumpRec_NilInterfaceProducesNoOutput(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("dumpRec: panic occurred: %v", r)
		}
	}()

	var nilInterface nodeReader[any]
	w := new(strings.Builder)
	path := stridePath{}

	dumpRec(nilInterface, w, path, 0, true)

	if w.Len() != 0 {
		t.Fatalf("dumpRec(nilInterface, ...) produced unexpected output: %q", w.String())
	}
}

// Ensure Table.dump prints a values(...) section while Lite.dump does not.
func TestDump_ValueSectionPresence(t *testing.T) {
	t.Parallel()

	// Table includes values
	{
		tbl := new(Table[any])
		tbl.Insert(mpp("0.0.0.0/0"), nil)

		w := new(strings.Builder)
		tbl.dump(w)
		got := w.String()
		if !strings.Contains(got, "values(") {
			t.Fatalf("Table.dump output missing values(...) section:\n%s", got)
		}
	}

	// Lite excludes values
	{
		tbl := new(Lite)
		tbl.Insert(mpp("0.0.0.0/0"))

		w := new(strings.Builder)
		tbl.dump(w)
		got := w.String()
		if strings.Contains(got, "values(") {
			t.Fatalf("Lite.dump output unexpectedly contains values(...) section:\n%s", got)
		}
	}
}
