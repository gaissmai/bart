//go:build generate

//go:generate go run $GOFILE

// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Package main generates the lookup table for longest-prefix-match (LPM)
// operations in the BART routing table.
//
// The generator creates a precomputed table (LookupTbl) that replaces the
// iterative backtracking in the prefix tree of a node by a single bitset
// operation.
//
// The core idea:
//
// The prefixes of a node form a complete binary tree. A prefix matches an
// address if the prefix itself or any of its ancestors (the less specific
// prefixes of the same bit string) is stored in the node. Instead of walking
// up the ancestors at runtime (idx >>= 1 in a loop), this table precomputes
// the ancestors of every index.
//
// Binary tree structure:
//   - Each index i has its parent at i>>1 (integer division by 2)
//   - Index 1 is the root, the default route /0 of the stride
//   - The children of index i are 2*i and 2*i+1, one bit more specific
//   - Indices 1..255 are the prefixes /0 up to /7, a tree with 8 levels
//   - An ancestor always has a smaller index than its descendants
//
// Example: Index 13 (binary 0b1101, the prefix 101/3) has itself and these
// ancestors:
//   - 13 itself (0b1101), the prefix 101/3
//   - 6  = 13>>1 (0b0110), the parent, the prefix 10/2
//   - 3  = 6>>1  (0b0011), the grandparent, the prefix 1/1
//   - 1  = 3>>1  (0b0001), the root, the prefix /0
//
// With this table, checking if the index or any of its ancestors exists in the
// prefix set of a node becomes:
//
//	node.Prefixes.Overlaps(&LookupTbl[idx])
//
// instead of a loop with multiple Test() calls. Because an ancestor always has
// a smaller index than its descendants, the highest bit of the intersection
// is the longest matching prefix:
//
//	node.Prefixes.AndTop(&LookupTbl[idx])
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/gaissmai/bart/internal/bitset"
)

var (
	thisFname = os.Getenv("GOFILE")
	outFname  = "lookuptblgenerated.go"
)

// data holds the template variables for code generation.
// The LookupTbl field is initialized at program start with the generated
// table string.
var data = struct {
	File      string
	LookupTbl string
}{
	File:      thisFname,
	LookupTbl: genLookupTbl(),
}

// colored prefix for informational and error messages
const (
	INFO = "\x1b[34mINFO:\x1b[0m"
	DIE  = "\x1b[31mERROR:\x1b[0m"
)

func main() {
	outFile, err := os.Create(outFname)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", DIE, err)
		os.Exit(1)
	}

	// Parse the template ...
	t := template.Must(template.New("foo").Parse(codeTemplate))

	// ... and execute it with our data
	err = t.Execute(outFile, data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", DIE, err)
		_ = outFile.Close()
		os.Exit(1)
	}

	// Ensure the file is properly closed before formatting
	if err := outFile.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", DIE, err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "%s ✓ Generated %s\n", INFO, outFname)

	// Run goimports to organize imports and format the code
	goimports := exec.Command("goimports", "-w", outFname)
	fmt.Fprintf(os.Stdout, "%s Running goimports on %s\n", INFO, outFname)
	if out, err := goimports.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "%s %s\n", DIE, string(out))
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "%s ✓ goimports completed\n", INFO)

	// Run gofumpt for additional formatting consistency
	gofumpt := exec.Command("gofumpt", "-w", outFname)
	fmt.Fprintf(os.Stdout, "%s Running gofumpt on %s\n", INFO, outFname)
	if out, err := gofumpt.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "%s %s\n", DIE, string(out))
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "%s ✓ gofumpt completed\n", INFO)

	fmt.Println()
}

// genLookupTbl generates the lookup table as a formatted string.
//
// For each index i (1..255), the table stores a BitSet256 containing:
//   - i itself (the current node)
//   - all binary ancestors of i (obtained by repeatedly right-shifting: i>>1, i>>2, ...)
//
// The algorithm walks up the binary tree from each index to the root (index 1).
// Each right-shift operation (i >>= 1) moves from a child to its parent node:
//   - Even indices (LSB=0) are left children: parent = i/2
//   - Odd indices (LSB=1) are right children: parent = (i-1)/2
//   - Both cases are handled by integer division via right-shift
//
// Example walkthrough for index 13 (binary 0b00001101):
//
//	Iteration 1: i=13 (0b00001101) => Set bit 13
//	Iteration 2: i=6  (0b00000110) => Set bit 6  (parent of 13)
//	Iteration 3: i=3  (0b00000011) => Set bit 3  (parent of 6)
//	Iteration 4: i=1  (0b00000001) => Set bit 1  (root)
//	Then i=0, the loop condition i > 0 fails and the loop terminates
//
// Result: BitSet256 with bits {1, 3, 6, 13} set. An index at depth d, counted
// from the root as depth 0, has d+1 bits set, so at most 8 bits.
//
// This allows checking for any match (contains) with a single bitset intersection:
//
//	// bitset intersection with the precomputed ancestors
//	return node.Prefixes.Overlaps(&LookupTbl[idx])
//
// instead of iterative backtracking:
//
//	// a loop with up to 8 single bit tests, each with a data dependent branch
//	for ; idx > 0; idx >>= 1 {
//	    if node.Prefixes.Test(idx) {
//	        return true
//	    }
//	}
//
// Entry 0 is empty and marked as "invalid", no prefix maps to base index 0.
//
// The generated string is formatted for embedding into Go source code.
func genLookupTbl() string {
	// Initialize the lookup table with 256 entries (one per uint8 value)
	lookupTbl := [256]bitset.BitSet256{}

	for idx := 1; idx <= 255; idx++ {
		// Walk up the binary tree by repeatedly right-shifting
		// Each iteration moves to the parent node (idx >> 1)
		for i := idx; i > 0; i >>= 1 {
			lookupTbl[idx].Set(uint8(i))
		}
	}

	// Convert the lookup table to a formatted string for the generated code
	builder := strings.Builder{}

	// Index 0 is invalid (no valid prefix maps to 0)
	fmt.Fprintf(&builder, "  /* idx: %3d */ %#v, // %s\n", 0, lookupTbl[0], "invalid")

	// Format each valid index with its bitset and the list of set bits.
	// The printed type name is shortened by the formatting step later, e.g.
	// /* idx:  13 */ {0x204a, 0x0, 0x0, 0x0}, // [1 3 6 13]
	for idx := 1; idx < 256; idx++ {
		fmt.Fprintf(&builder, "  /* idx: %3d */ %#v, // %v\n", idx, lookupTbl[idx], lookupTbl[idx].Bits())
	}

	return builder.String()
}

// codeTemplate is the Go source template for the generated file
// lookuptblgenerated.go. It embeds the precomputed lookup table and documents
// its structure and usage.
//
// The template defines:
//   - the package documentation explaining the LPM optimization
//   - the LookupTbl variable with all 256 precomputed bitsets
//   - usage examples for the single bitset operations and the loop they replace
//   - the warning not to mutate the read-only table
const codeTemplate = `// Code generated by {{.File}}; DO NOT EDIT.

// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Package lpm (longest-prefix-match) contains the lookup table with which
// the backtracking for the lpm in the complete binary tree of the prefixes
// of a node is replaced by a fast bitset operation.
package lpm

// LookupTbl is a precomputed read-only table used in hot paths.
//
// Each entry idx is the bitset of idx itself and all its binary ancestors
// (idx, idx>>1, idx>>2, ..., 1). In the complete binary tree of the prefixes
// of a node these are the prefix idx and all less specific prefixes that
// cover it.
//
// idx must be a base index produced by art.OctetToIdx or art.PfxToIdx.
// Index 0 is not a valid base index, its entry is empty and never matches.
//
// Usage:
//
// Is the prefix idx or any of its ancestors stored in the node?
// One bitset intersection:
//
//	func (n *BartNode[T]) Contains(idx uint8) bool {
//		return n.Prefixes.Overlaps(&lpm.LookupTbl[idx])
//	}
//
// Which is the longest matching prefix? An ancestor always has a smaller
// index than its descendants, so the highest bit of the intersection is the
// longest match:
//
//	top, ok := n.Prefixes.AndTop(&lpm.LookupTbl[idx])
//
// Both replace a loop of single bit tests, each with a data dependent branch:
//
//	for ; idx > 0; idx >>= 1 {
//		if n.Prefixes.Test(idx) {
//			return true
//		}
//	}
//	return false
//
// DO NOT MUTATE the table.
//
//nolint:gochecknoglobals // Precomputed read-only table used in hot paths.
var LookupTbl = [256]bitset.BitSet256{
{{.LookupTbl}} }`
