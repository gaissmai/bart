// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Command gen generates the monomorphized methods and tests for the node types
// and the table types from the template files. It replaces the bash scripts
// generate-node-methods.sh, generate-node-tests.sh, generate-table-methods.sh
// and generate-table-tests.sh.
//
// It is started by go generate from the directive in the template file, for
// example in internal/nodes/commonmethods_tmpl.go:
//
//	//go:generate go run ../../scripts/gen node-methods
//
// The template file is taken from the environment variable GOFILE, which go
// generate sets, or from the optional second argument. The generated files are
// written to the current directory, the package directory of the template.
//
// The modes and the files they generate:
//
//	node-methods   commonmethods_tmpl.go -> bartmethodsgenerated.go, litemethodsgenerated.go
//	node-tests     commontests_tmpl.go   -> barttestsgenerated_test.go, litetestsgenerated_test.go
//	table-methods  commonmethods_tmpl.go -> bartmethodsgenerated.go, litemethodsgenerated.go
//	table-tests    commontests_tmpl.go   -> barttestsgenerated_test.go, litetestsgenerated_test.go
//
// A template is a valid Go file for development with gopls. The generator
// removes what must not be generated:
//
//   - the build constraint "//go:build generate" and the "//go:generate" directives
//   - the stubs between the markers GENERATE DELETE START and GENERATE DELETE END
//   - for the node-methods mode only: the code between the markers
//     GENERATE SKIP_LITE START and GENERATE SKIP_LITE END is removed for the
//     LiteNode and kept for the BartNode, the marker lines are always removed
//
// and replaces the placeholders _NODE_TYPE and _TABLE_TYPE by the type names.
// Afterwards goimports and gofumpt are run on the generated files, if they are
// installed.
package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
)

const (
	blue = "\x1b[34m"
	red  = "\x1b[31m"
	off  = "\x1b[0m"
)

// replacement substitutes a placeholder of the template by a type name.
type replacement struct {
	placeholder string
	name        string
}

// target is a generated file.
type target struct {
	prefix string        // prefix of the file name: bart or lite
	lite   bool          // the payload-less variant, see GENERATE SKIP_LITE
	repl   []replacement // the substitutions of the placeholders
	note   string        // appended to the message about the generated file
}

// mode is one of the four generators.
type mode struct {
	desc     string // for the start message
	suffix   string // replaces _tmpl in the name of the template
	skipLite bool   // handle the markers GENERATE SKIP_LITE
	targets  []target
}

//nolint:gochecknoglobals
var modes = map[string]mode{
	"node-methods": {
		desc:     "monomorphized node methods",
		suffix:   "generated",
		skipLite: true,
		targets: []target{
			{prefix: "bart", repl: []replacement{{"_NODE_TYPE", "BartNode"}}},
			{prefix: "lite", lite: true, repl: []replacement{{"_NODE_TYPE", "LiteNode"}}},
		},
	},
	"node-tests": {
		desc:   "node tests",
		suffix: "generated_test",
		targets: []target{
			{prefix: "bart", repl: []replacement{{"_NODE_TYPE", "BartNode"}}},
			{prefix: "lite", lite: true, repl: []replacement{{"_NODE_TYPE", "LiteNode"}}},
		},
	},
	"table-methods": {
		desc:   "monomorphized table methods",
		suffix: "generated",
		targets: []target{
			{
				prefix: "bart", note: " (Table -> BartNode)",
				repl: []replacement{{"_TABLE_TYPE", "Table"}, {"_NODE_TYPE", "BartNode"}},
			},
			{
				prefix: "lite", lite: true, note: " (liteTable -> LiteNode)",
				repl: []replacement{{"_TABLE_TYPE", "liteTable"}, {"_NODE_TYPE", "LiteNode"}},
			},
		},
	},
	"table-tests": {
		desc:   "table tests",
		suffix: "generated_test",
		targets: []target{
			{prefix: "bart", repl: []replacement{{"_TABLE_TYPE", "Table"}}},
			{prefix: "lite", lite: true, repl: []replacement{{"_TABLE_TYPE", "liteTable"}}},
		},
	},
}

// the tools that are run on the generated files, in this order.
//
//nolint:gochecknoglobals
var tools = []struct {
	name     string
	skipping string // what is skipped, if the tool is not installed
}{
	{"goimports", "imports"},
	{"gofumpt", "formatting"},
}

var (
	buildConstraint   = regexp.MustCompile(`^//go:build generate\b`)
	generateDirective = regexp.MustCompile(`^//go:generate\b`)
)

// env is the output of the generator, replaceable for tests.
type env struct {
	out    io.Writer
	errOut io.Writer
}

func main() {
	e := env{out: os.Stdout, errOut: os.Stderr}

	if err := e.run(os.Args[1:], os.Getenv("GOFILE")); err != nil {
		fmt.Fprintf(os.Stderr, "%sERROR:%s %v\n", red, off, err)
		os.Exit(1)
	}
}

func (e env) info(format string, args ...any) {
	fmt.Fprintf(e.out, "%sINFO:%s %s\n", blue, off, fmt.Sprintf(format, args...))
}

func (e env) warn(format string, args ...any) {
	fmt.Fprintf(e.out, "%sWARN:%s %s\n", red, off, fmt.Sprintf(format, args...))
}

func modeNames() string {
	return strings.Join(slices.Sorted(maps.Keys(modes)), ", ")
}

// run generates the files of the mode args[0] from the template, which is
// gofile or the optional second argument.
func (e env) run(args []string, gofile string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: gen <mode> [template file], the modes are: %s", modeNames())
	}

	m, ok := modes[args[0]]
	if !ok {
		return fmt.Errorf("unknown mode %q, the modes are: %s", args[0], modeNames())
	}

	templateFile := gofile
	if len(args) == 2 {
		templateFile = args[1]
	}

	//nolint:gosec // G703: Path traversal via taint analysis
	raw, err := os.ReadFile(templateFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("template file '%s' not found, GOFILE=%s", templateFile, gofile)
		}

		return err
	}

	src := string(raw)

	e.info("START: Generating %s from template '%s' ...", m.desc, templateFile)

	if strings.Contains(src, "TODO") {
		return fmt.Errorf("✗ Aborting, found pattern 'TODO' in %s", templateFile)
	}

	var generated []string

	for _, tgt := range m.targets {
		name := outputName(tgt.prefix, templateFile, m.suffix)

		//nolint:gosec // G306: the generated files are source files, not secrets
		if err := os.WriteFile(name, []byte(transform(src, templateFile, tgt, m.skipLite)), 0o644); err != nil {
			return fmt.Errorf("✗ Failed to generate %s: %w", name, err)
		}

		e.info("✓ Generated %s%s", name, tgt.note)
		generated = append(generated, name)
	}

	fmt.Fprintln(e.out)

	for _, tool := range tools {
		if err := e.runTool(tool.name, tool.skipping, generated); err != nil {
			return err
		}
	}

	e.info("END: Template generation complete!")
	fmt.Fprintln(e.out)

	return nil
}

// runTool runs the tool with -w on the files, if it is installed.
func (e env) runTool(name, skipping string, files []string) error {
	path, err := exec.LookPath(name)
	if err != nil {
		e.warn("⚠ %s not found, skipping %s", name, skipping)
		return nil
	}

	e.info("Running %s on generated files...", name)

	//nolint:gosec // G204: the tools are looked up by fixed names
	cmd := exec.Command(path, append([]string{"-w"}, files...)...)
	cmd.Stdout, cmd.Stderr = e.out, e.errOut

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}

	e.info("✓ %s completed", name)

	return nil
}

// outputName returns the name of the generated file: the template
// commonmethods_tmpl.go with the prefix bart and the suffix generated gives
// bartmethodsgenerated.go. Everything up to the last "common" is cut off from
// the name of the template, the first "_tmpl" is replaced by the suffix.
func outputName(prefix, templateFile, suffix string) string {
	base := templateFile

	if i := strings.LastIndex(base, "common"); i >= 0 {
		base = base[i+len("common"):]
	}

	return prefix + strings.Replace(base, "_tmpl", suffix, 1)
}

// rangeDeleter deletes the lines from the line with the start marker to the
// line with the end marker, both included, like the sed address range
// /start/,/end/d: the end marker is searched from the line behind the start
// marker, a range without end marker reaches to the end of the file.
type rangeDeleter struct {
	start, end string
	inside     bool
}

// delete reports whether the line is deleted.
func (r *rangeDeleter) delete(text string) bool {
	if r.inside {
		if strings.Contains(text, r.end) {
			r.inside = false
		}

		return true
	}

	if strings.Contains(text, r.start) {
		r.inside = true
		return true
	}

	return false
}

// transform returns the source of a generated file made from the template.
//
// The lines are processed in the same order as by the sed script of the former
// bash generators, a line that is deleted by a rule is not seen by the next
// rules:
//
//  1. the header is the first line, even if the first line of the template is deleted
//  2. the build constraint and the go:generate directives are deleted
//  3. the lines between the markers GENERATE DELETE are deleted
//  4. with skipLite: for the lite variant the lines between the markers
//     GENERATE SKIP_LITE are deleted, for the others only the marker lines
//  5. the placeholders are replaced
//
// The result for an empty template is empty, there is no header.
func transform(src, templateFile string, tgt target, skipLite bool) string {
	if src == "" {
		return ""
	}

	var out strings.Builder

	fmt.Fprintf(&out, "// Code generated from file \"%s\"; DO NOT EDIT.\n", templateFile)

	deleteRange := rangeDeleter{start: "GENERATE DELETE START", end: "GENERATE DELETE END"}
	liteRange := rangeDeleter{start: "GENERATE SKIP_LITE START", end: "GENERATE SKIP_LITE END"}

	for _, line := range strings.SplitAfter(src, "\n") {
		// SplitAfter returns an empty string behind the last newline
		if line == "" {
			continue
		}

		text := strings.TrimSuffix(line, "\n")

		if buildConstraint.MatchString(text) || generateDirective.MatchString(text) {
			continue
		}

		if deleteRange.delete(text) {
			continue
		}

		if skipLite {
			if tgt.lite {
				if liteRange.delete(text) {
					continue
				}
			} else if strings.Contains(text, "GENERATE SKIP_LITE") {
				continue
			}
		}

		for _, r := range tgt.repl {
			line = strings.ReplaceAll(line, r.placeholder, r.name)
		}

		out.WriteString(line)
	}

	return out.String()
}
