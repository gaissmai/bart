// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bitset

import (
	"context"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// inlineBudget is the maximum cost of a function that the Go compiler still
// considers for inlining.
const inlineBudget = 80

// canInlineRe matches the compiler diagnostics of -gcflags=-m=2, for example:
//
//	./bitset256.go:65:6: can inline (*BitSet256).Set with cost 12 as: ...
var canInlineRe = regexp.MustCompile(`can inline (.+?) with cost (\d+)`)

// TestInlineBitSet256Functions verifies that functions stay inlinable.
// Most of them are on the hot path of every trie level during lookups,
// a missing inlining costs a real function call per level.
//
// The test compiles this package with -gcflags=-m=2 in a subprocess and
// evaluates the inliner diagnostics. Use "go test -v -run TestInlineBitSet256Functions"
// to see the current costs and the remaining headroom.
func TestInlineBitSet256Functions(t *testing.T) {
	// The costs depend on math/bits functions being an intrinsic.
	switch runtime.GOARCH {
	case "amd64", "arm64":
	default:
		t.Skipf("inline costs are only verified on amd64 and arm64, not on %s", runtime.GOARCH)
	}

	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go tool not found in PATH")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, goTool, "build", "-gcflags=-m=2", ".")

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s failed: %v\n%s", cmd, err, out)
	}

	report := string(out)

	// function name -> inline cost, from the "can inline" lines
	costs := make(map[string]int)
	for line := range strings.SplitSeq(report, "\n") {
		m := canInlineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}

		cost, err := strconv.Atoi(m[2])
		if err != nil {
			t.Fatalf("cannot parse cost in %q: %v", line, err)
		}
		costs[m[1]] = cost
	}

	if len(costs) == 0 {
		t.Fatalf("no inliner diagnostics found, did the compiler output change?\n%s", report)
	}

	mustInline := []string{
		"(*BitSet256).Set",
		"(*BitSet256).Clear",
		"(*BitSet256).Test",
		"(*BitSet256).FirstSet",
		"(*BitSet256).NextSet",
		"(*BitSet256).LastSet",
		"(*BitSet256).All",
		"(*BitSet256).All.func1",
		"(*BitSet256).AllEnumerate",
		"(*BitSet256).AllEnumerate.func1",
		"(*BitSet256).AppendBits",
		"(*BitSet256).OnesCount",
		"(*BitSet256).Bits",
		"(*BitSet256).AndTop",
		"(*BitSet256).AlignedPairs",
		"(*BitSet256).IsEmpty",
		"(*BitSet256).Overlaps",
		"(*BitSet256).And",
		"(*BitSet256).Xor",
		"(*BitSet256).Or",
	}

	for _, fn := range mustInline {
		cost, ok := costs[fn]
		if !ok {
			t.Errorf("%s is not inlinable: %s", fn, whyNot(report, fn))
			continue
		}

		t.Logf("%-45s cost %2d, headroom %2d", fn, cost, inlineBudget-cost)
	}
}

// whyNot returns the compiler's reason why fn is not inlinable.
func whyNot(report, fn string) string {
	for line := range strings.SplitSeq(report, "\n") {
		if strings.Contains(line, "cannot inline "+fn+":") {
			_, reason, _ := strings.Cut(line, "cannot inline ")
			return reason
		}
	}

	return "no inliner diagnostic found, is the function instantiated in the test files?"
}
