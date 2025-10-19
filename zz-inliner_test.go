// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart_test

import (
	"bytes"
	"context"
	"os/exec"
	"regexp"
	"testing"
	"time"
)

// TestInlineBitSet256Functions checks if specified functions in hot path
// are inlined by the Go compiler. It compiles the package with the
// -gcflags=-m flag to get inlining debug output, then searches the output
// for evidence that the functions were inlined.
//
// The functions to check are listed in the funcs slice. If any function is
// missing the inlining message, the test fails.
func TestInlineBitSet256Functions(t *testing.T) {
	// List of functions expected to be inlined
	funcs := []string{
		"bitset.(*BitSet256).Set",
		"bitset.(*BitSet256).Clear",
		"bitset.(*BitSet256).Test",
		"bitset.(*BitSet256).IsEmpty",
		//
		"bitset.(*BitSet256).FirstSet",
		"bitset.(*BitSet256).NextSet",
		"bitset.(*BitSet256).LastSet",
		//
		"bitset.(*BitSet256).Intersects",
		"bitset.(*BitSet256).Intersection",
		"bitset.(*BitSet256).IntersectionTop",
		//
		"bitset.(*BitSet256).Rank",
		"bitset.(*BitSet256).Size",
		"bitset.(*BitSet256).Union",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	buf := new(bytes.Buffer)

	// Run 'go build' with inlining debug output enabled and capture stdout/stderr
	cmd := exec.CommandContext(ctx, "go", "build", "-gcflags=-m")
	cmd.Stdout = buf
	cmd.Stderr = buf

	err := cmd.Run()
	if err != nil {
		t.Fatalf("Build failed: %v\nCompiler output:\n%s", err, buf.String())
	}

	output := buf.String()

	// Check compiler output for each function's inlining indication
	for _, fn := range funcs {
		inlineMsgRx := regexp.MustCompile("inlining call to .*" + regexp.QuoteMeta(fn))
		if inlineMsgRx.MatchString(output) {
			continue
		}

		canInlineMsgRx := regexp.MustCompile("can inline .*" + regexp.QuoteMeta(fn))
		if canInlineMsgRx.MatchString(output) {
			continue
		}

		t.Errorf("Function %s is NOT inlined.", fn)
	}
}
