package bart

import (
	"bytes"
	"strings"
	"testing"
)

func TestFat_dumpString_OnEmptyTable_ReturnsEmptyString(t *testing.T) {
	t.Parallel()
	f := &Fat[struct{}]{}

	out := f.dumpString()
	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected empty dump for empty table; got %q", out)
	}
}

func TestFat_dump_OnNilReceiver_NoPanicAndNoOutput(t *testing.T) {
	t.Parallel()
	var f *Fat[int] = nil

	var buf bytes.Buffer
	f.dump(&buf)

	got := strings.TrimSpace(buf.String())
	if got != "" {
		t.Fatalf("expected no output for nil Fat; got %q", got)
	}
}

func TestFat_dump_IPv4AndIPv6PrintedIndependently(t *testing.T) {
	t.Parallel()
	f := &Fat[int]{}
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

func TestFatNode_hasType_OnEmptyNode_ReturnsNullNode(t *testing.T) {
	t.Parallel()
	n := &fatNode[struct{}]{}

	nt := hasType[struct{}](n)
	if nt != nullNode {
		t.Fatalf("expected nullNode for empty node; got %v", nt)
	}
}

func TestFatNode_nodeStats_OnEmptyNode_AllZeros(t *testing.T) {
	t.Parallel()
	n := &fatNode[int]{}
	s := nodeStats[int](n)

	if s.pfxs != 0 || s.childs != 0 || s.nodes != 0 || s.leaves != 0 || s.fringes != 0 {
		t.Fatalf("expected zero stats for empty node; got %+v", s)
	}
}

func TestFatNode_nodeStatsRec_OnEmptyNode_NodeCountZero(t *testing.T) {
	t.Parallel()
	n := &fatNode[int]{}
	s := nodeStatsRec(n)

	if s.pfxs != 0 || s.childs != 0 || s.nodes != 0 || s.leaves != 0 || s.fringes != 0 {
		t.Fatalf("expected zero recursive stats for empty node; got %+v", s)
	}
}

func TestFatNode_dump_OnEmptyNode_PrintsHeaderOnly(t *testing.T) {
	t.Parallel()
	n := &fatNode[int]{}
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
