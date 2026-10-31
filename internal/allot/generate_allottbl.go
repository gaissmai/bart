//go:build generate

//go:generate go run $GOFILE

// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Package main generates the lookup tables for the route allotment of the BART
// routing table.
//
// The generator creates two precomputed tables, PfxRoutesLookupTbl and
// FringeRoutesLookupTbl. They answer the question "which items does the prefix
// idx cover?" with a single bitset intersection.
//
// Both tables are indexed by the base index idx (1..255) of a prefix, which is
// the position of the prefix in the complete binary tree of one 8-bit stride:
//
//   - idx 1 is the root of the tree, the default route /0 of the stride
//   - idx 2*i and 2*i+1 are the two children of idx i, one bit more specific
//   - idx 1..255 are the prefixes /0 up to /7
//   - idx 256..511 are the /8 prefixes of the stride, one for every octet
//     (octet = idx-256). They are not stored as prefixes in a node, they are the
//     child addresses and are handled as "fringe" (path compression).
//
// Every table entry is the subtree rooted at idx, i.e. idx itself and all its
// descendants. The subtree is split into two bitsets of 256 bits each:
//
//   - PfxRoutesLookupTbl[idx] holds the part of the subtree in 1..255: idx
//     itself and all more specific prefixes up to /7.
//   - FringeRoutesLookupTbl[idx] holds the part of the subtree in 256..511,
//     stored without the offset of 256. These are the octets, i.e. the child
//     addresses, covered by the prefix idx. idx itself is never a member.
//
// Example: idx 13 is the prefix 101/3, it covers the octets 160..191.
// PfxRoutesLookupTbl[13] holds the 31 indices 13, 26, 27, 52..55, 104..111 and
// 208..223. FringeRoutesLookupTbl[13] holds the 32 octets 160..191.
//
// Entry 0 of both tables is empty, 0 is not a valid base index.
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
	outFname  = "allotroutesgenerated.go"
)

var (
	// prefixAllotTbl is the content of PfxRoutesLookupTbl. For every base index
	// idx (1..255) the bitset holds idx itself and all its descendants in the
	// range 1..255 (the prefixes up to /7).
	prefixAllotTbl [256]bitset.BitSet256

	// fringeAllotTbl is the content of FringeRoutesLookupTbl. For every base index
	// idx (1..255) the bitset holds all descendants of idx in the range 256..511
	// (the /8 prefixes), stored without the offset of 256. These are the octets
	// covered by the prefix idx.
	fringeAllotTbl [256]bitset.BitSet256
)

// data holds the template variables for code generation
var data struct {
	File             string
	PrefixAllotTable string
	FringeAllotTable string
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

	// Generate both lookup tables using the work list algorithm
	prefixAllotTbl, fringeAllotTbl = genAllotTables()

	// Populate template data
	data.File = thisFname
	data.PrefixAllotTable = asString(prefixAllotTbl)
	data.FringeAllotTable = asString(fringeAllotTbl)

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

// genAllotTables generates the prefix and the fringe allotment lookup tables.
//
// For each base index idx (1..255) it collects the subtree of the complete
// binary tree rooted at idx: idx itself and all its descendants. The traversal
// is breadth-first and uses a work list instead of recursion.
//
// The algorithm:
//  1. Start with the root index idx in the work list
//  2. For each index j in the work list:
//     - If j < 256: add j to prefixAllotTbl[idx] and append the children
//     2*j and 2*j+1 to the work list
//     - If j >= 256: add (j-256) to fringeAllotTbl[idx]. The /8 level is the
//     last level of the tree, j has no children
//  3. Continue until the end of the work list is reached
//
// Hence idx itself is always a member of prefixAllotTbl[idx] and never a
// member of fringeAllotTbl[idx].
//
// Returns the prefix table (the subtrees restricted to the indices 1..255) and
// the fringe table (the subtrees restricted to the indices 256..511, stored
// as 0..255).
func genAllotTables() (prefix, fringe [256]bitset.BitSet256) {
	for idx := 1; idx < 256; idx++ {
		// allot algorithm, work list instead of recursion
		queue := make([]int, 0, 512)
		queue = append(queue, idx)

		for i := 0; i < len(queue); i++ {
			j := queue[i]
			if j < 256 {
				prefix[idx].Set(uint8(j))
			} else {
				fringe[idx].Set(uint8(j - 256))
			}

			// max j is 511, the /8 level (256 and above) is the last level
			// of the tree, so stop the descent there
			if j >= 256 {
				continue
			}

			// append the children in the complete binary tree
			// left:  j*2
			// right: (j*2)+1
			queue = append(queue, j<<1, (j<<1)+1)
		}
	}
	return prefix, fringe
}

// asString converts a lookup table to a formatted Go source string.
//
// Each entry of the table is written as one line:
//
//	/* idx: N */ bitset.BitSet256{...}, // [list of the set bits]
//
// The formatting step later shortens the type name in the generated file.
// For example, idx 13 (the prefix 101/3) of the PfxRoutesLookupTbl becomes
//
//	/* idx:  13 */ {0xf000000c002000, 0xff0000000000, 0x0, 0xffff0000}, // [13→26→27→52→53→...]
//
// A list with fewer than 10 set bits is printed completely, e.g. "[0 1]".
// A longer list is cut after the first five bits and ends with "... ]",
// to keep the generated code readable.
//
// Entry 0 is empty and marked as "invalid", since no prefix maps to base
// index 0 in the BART structure.
//
// The returned string is ready to be embedded into the generated Go file.
func asString(tbl [256]bitset.BitSet256) string {
	builder := strings.Builder{}

	// Index 0 is invalid (no valid prefix maps to 0)
	fmt.Fprintf(&builder, "  /* idx: %3d */ %#v, // %s\n", 0, tbl[0], "invalid")

	// Format each valid index with its bitset and the list of set bits, e.g.
	for idx := 1; idx < 256; idx++ {
		fmt.Fprintf(&builder, "  /* idx: %3d */ %#v, // ", idx, tbl[idx])

		bits := tbl[idx].Bits()
		if len(bits) < 10 {
			fmt1 := fmt.Sprintf("%v", bits)
			fmt2 := strings.Join(strings.Split(fmt1, " "), "→") // [1←3←6←13]
			fmt.Fprintf(&builder, "%s\n", fmt2)
		} else {
			fmt.Fprint(&builder, "[")
			for _, bit := range bits[:5] {
				fmt.Fprintf(&builder, "%d→", bit)
			}
			fmt.Fprintln(&builder, "...]")
		}
	}

	return builder.String()
}

// codeTemplate is the Go source template for the generated file
// allotroutesgenerated.go. It embeds the precomputed lookup tables and
// documents their structure and usage.
const codeTemplate = `// Code generated by {{.File}}; DO NOT EDIT.

// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package allot

// PfxRoutesLookupTbl is a lookup table of BitSet256 entries for the route allotment.
//
// It maps a baseIndex idx (1..255) to a bitset that contains idx itself and all
// its descendants in the complete binary tree of the stride, i.e. all more
// specific indices up to /7.
//
// The idea:
//
//	If prefix P has baseIndex idx, then P covers all prefixes in the subtree
//	rooted at idx, including P itself. This table enumerates and encodes
//	those indices.
//
// The subtree of idx has up to 512 bits and is split into two tables of
// 256 bits each:
//   - PfxRoutesLookupTbl holds the part with the indices 1..255, the prefixes up to /7
//   - FringeRoutesLookupTbl holds the part with the indices 256..511, the /8 prefixes
//
// This structure allows for very fast coverage queries using a simple bitwise AND,
// e.g. the prefixes of a node covered by idx are node.Prefixes AND PfxRoutesLookupTbl[idx].
//
// Index 0 is not a valid baseIndex, its entry is empty.
//
//nolint:gochecknoglobals // Precomputed read‑only table used in hot paths.
var PfxRoutesLookupTbl = [256]bitset.BitSet256{
{{.PrefixAllotTable}} }

// FringeRoutesLookupTbl is the second half of the subtree bitsets, see
// PfxRoutesLookupTbl for the first half. The 512 bits of a subtree are split
// into 2×256 bits to leverage the BitSet256 optimizations.
//
// It maps a baseIndex idx (1..255) to a bitset that contains the /8 level of
// the subtree of idx. The index 256+n is stored as bit n, so the bits are the
// octets, i.e. the child addresses, covered by the prefix idx. For example,
// the prefix 0b101/3 with idx=13 covers the octets 160..191.
//
// The idx itself is never a member of this table.
//
//nolint:gochecknoglobals // Precomputed read‑only table used in hot paths.
var FringeRoutesLookupTbl = [256]bitset.BitSet256{
{{.FringeAllotTable}} }
`
