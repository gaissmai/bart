// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package sparse

import (
	"context"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Force the compiler to instantiate Array256[any], the instantiation used for
// the children of all trie nodes. The compiler reports inlining decisions for
// generic methods only for instantiations that exist in the compiled package.
var (
	_ = (*Array256[any]).Get
	_ = (*Array256[any]).MustGet
)

// inlineBudget is the maximum cost of a function that the Go compiler still
// considers for inlining.
const inlineBudget = 80

// canInlineRe matches the compiler diagnostics of -gcflags=-m=2, for example:
//
//	./array256.go:142:6: can inline (*Array256[int]).Get with cost 80 as: ...
var canInlineRe = regexp.MustCompile(`can inline (.+?) with cost (\d+)`)

// TestInlineArray256Functions verifies that Get and MustGet of Array256[any]
// stay inlinable. They are on the hot path of every trie level during lookups,
// a missing inlining costs a real function call per level.
//
// Generic code is compiled per GC shape. For Array256[any] the compiler
// creates the shape implementation (*Array256[go.shape.interface {}]) and a
// thin wrapper for the concrete type (*Array256[interface {}]) that passes
// the dictionary. Callers use the wrapper, which calls the shape
// implementation, so both levels must be inlinable.
//
// The test compiles this package with -gcflags=-m=2 in a subprocess and
// evaluates the inliner diagnostics. Use "go test -v -run TestInlineArray256Functions"
// to see the current costs and the remaining headroom.
func TestInlineArray256Functions(t *testing.T) {
	// The costs depend on math/bits.OnesCount64 being an intrinsic.
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

	// "go test -c" compiles the package including the test files, which
	// instantiate Array256[any] above, but does not run any test.
	cmd := exec.CommandContext(ctx, goTool, "test", "-c",
		"-o", filepath.Join(t.TempDir(), "sparse.test"),
		"-gcflags=-m=2", ".")

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
		// shape implementation, called by the wrappers below
		"(*Array256[go.shape.interface {}]).rank0",
		"(*Array256[go.shape.interface {}]).Get",
		"(*Array256[go.shape.interface {}]).MustGet",

		// wrappers for the concrete type, called from the trie nodes
		"(*Array256[interface {}]).Get",
		"(*Array256[interface {}]).MustGet",
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
