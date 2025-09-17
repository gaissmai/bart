// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"bytes"
	"strings"
	"testing"
)

func TestFast_dumpString_OnEmptyTable_ReturnsEmptyString(t *testing.T) {
	t.Parallel()
	f := &Fast[struct{}]{}

	out := f.dumpString()
	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected empty dump for empty table; got %q", out)
	}
}

func TestFast_dump_OnNilReceiver_NoPanicAndNoOutput(t *testing.T) {
	t.Parallel()
	var f *Fast[int] = nil

	var buf bytes.Buffer
	f.dump(&buf)

	got := strings.TrimSpace(buf.String())
	if got != "" {
		t.Fatalf("expected no output for nil Fast; got %q", got)
	}
}

func TestFast_dump_IPv4AndIPv6PrintedIndependently(t *testing.T) {
	t.Parallel()
	f := &Fast[int]{}
	f.size4 = 3
	f.size6 = 4

	var buf bytes.Buffer
	f.dump(&buf)
	out := buf.String()

	if !strings.Contains(out, "### IPv4: size(3)") {
		t.Fatalf("missing IPv4 header: %q", out)
	}
	if !strings.Contains(out, "### IPv6: size(4)") {
		t.Fatalf("missing IPv6 header: %q", out)
	}
}

func TestFastNode_hasType_OnEmptyNode_ReturnsNullNode(t *testing.T) {
	t.Parallel()
	n := &fastNode[struct{}]{}

	nt := hasType(n)
	if nt != nullNode {
		t.Fatalf("expected nullNode for empty node; got %v", nt)
	}
}

func TestFastNode_nodeStats_OnEmptyNode_AllZeros(t *testing.T) {
	t.Parallel()
	n := &fastNode[int]{}
	s := nodeStats(n)

	if s.pfxs != 0 || s.childs != 0 || s.nodes != 0 || s.leaves != 0 || s.fringes != 0 {
		t.Fatalf("expected zero stats for empty node; got %+v", s)
	}
}

func TestFastNode_nodeStatsRec_OnEmptyNode_NodeCountZero(t *testing.T) {
	t.Parallel()
	n := &fastNode[int]{}
	s := nodeStatsRec(n)

	if s.pfxs != 0 || s.childs != 0 || s.nodes != 0 || s.leaves != 0 || s.fringes != 0 {
		t.Fatalf("expected zero recursive stats for empty node; got %+v", s)
	}
}

func TestFastNode_dump_OnEmptyNode_PrintsHeaderOnly(t *testing.T) {
	t.Parallel()
	n := &fastNode[int]{}
	var buf bytes.Buffer

	var path stridePath
	dump(n, &buf, path, 0, true)

	out := buf.String()
	if !strings.Contains(out, "depth:  0") {
		t.Fatalf("expected depth header; got: %q", out)
	}
	if strings.Contains(out, "octets(") || strings.Contains(out, "prefxs(") {
		t.Fatalf("unexpected children or prefixes in empty dump: %q", out)
	}
}

func TestFast_dumpString_OnNonEmptySizes_PrintsHeaders(t *testing.T) {
	t.Parallel()
	f := &Fast[int]{}
	f.size4 = 1
	f.size6 = 2

	out := f.dumpString()

	if !strings.Contains(out, "### IPv4: size(1)") {
		t.Fatalf("missing IPv4 header: %q", out)
	}
	if !strings.Contains(out, "### IPv6: size(2)") {
		t.Fatalf("missing IPv6 header: %q", out)
	}
}

func TestFast_dumpString_OnNilReceiver_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	var f *Fast[int]
	out := f.dumpString()
	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected empty dump for nil Fast; got %q", out)
	}
}

func TestFast_dump_OnlyIPv4Size_PrintsIPv4Header(t *testing.T) {
	t.Parallel()
	f := &Fast[int]{}
	f.size4 = 2 // IPv6 remains zero

	var buf bytes.Buffer
	f.dump(&buf)
	out := buf.String()

	if !strings.Contains(out, "### IPv4: size(2)") {
		t.Fatalf("missing IPv4 header for v4-only dump: %q", out)
	}

	if strings.Contains(out, "### IPv6:") {
		t.Fatalf("unexpected IPv6 header for v4-only dump: %q", out)
	}
}

func TestFastNode_dump_WithNonZeroDepth_PrintsDepthHeader(t *testing.T) {
	t.Parallel()
	n := &fastNode[int]{}
	var buf bytes.Buffer
	var path stridePath

	dump(n, &buf, path, 2, false)

	out := buf.String()
	if !strings.Contains(out, "depth:  2") {
		t.Fatalf("expected depth header for depth 2; got: %q", out)
	}
}
