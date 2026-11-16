// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOutputName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		prefix       string
		templateFile string
		suffix       string
		want         string
	}{
		{
			name:         "node methods",
			prefix:       "bart",
			templateFile: "commonmethods_tmpl.go",
			suffix:       "generated",
			want:         "bartmethodsgenerated.go",
		},
		{
			name:         "node tests with path",
			prefix:       "lite",
			templateFile: "internal/nodes/commontests_tmpl.go",
			suffix:       "generated_test",
			want:         "litetestsgenerated_test.go",
		},
		{
			name:         "multiple common occurrences",
			prefix:       "bart",
			templateFile: "common/dir/commonmethods_tmpl.go",
			suffix:       "generated",
			want:         "bartmethodsgenerated.go",
		},
		{
			name:         "without common prefix in filename",
			prefix:       "lite",
			templateFile: "methods_tmpl.go",
			suffix:       "generated",
			want:         "litemethodsgenerated.go",
		},
		{
			name:         "without tmpl suffix",
			prefix:       "bart",
			templateFile: "commonmethods.go",
			suffix:       "generated",
			want:         "bartmethods.go",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := outputName(tc.prefix, tc.templateFile, tc.suffix)
			if got != tc.want {
				t.Errorf("outputName(%q, %q, %q) = %q, want %q",
					tc.prefix, tc.templateFile, tc.suffix, got, tc.want)
			}
		})
	}
}

func TestRangeDeleter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		start string
		end   string
		lines []string
		want  []string
	}{
		{
			name:  "normal range deletion",
			start: "START",
			end:   "END",
			lines: []string{
				"keep 1",
				"// START",
				"delete 1",
				"delete 2",
				"// END",
				"keep 2",
			},
			want: []string{
				"keep 1",
				"keep 2",
			},
		},
		{
			name:  "multiple ranges in input",
			start: "START",
			end:   "END",
			lines: []string{
				"keep 1",
				"// START",
				"delete 1",
				"// END",
				"keep 2",
				"// START",
				"delete 2",
				"// END",
				"keep 3",
			},
			want: []string{
				"keep 1",
				"keep 2",
				"keep 3",
			},
		},
		{
			name:  "unclosed range deletes until EOF",
			start: "START",
			end:   "END",
			lines: []string{
				"keep 1",
				"// START",
				"delete 1",
				"delete 2",
			},
			want: []string{
				"keep 1",
			},
		},
		{
			name:  "end marker on same line as start does not close range immediately",
			start: "START",
			end:   "END",
			lines: []string{
				"// START END",
				"still deleted",
				"// END",
				"kept",
			},
			want: []string{
				"kept",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rd := rangeDeleter{start: tc.start, end: tc.end}
			var got []string
			for _, line := range tc.lines {
				if !rd.delete(line) {
					got = append(got, line)
				}
			}

			if len(got) != len(tc.want) {
				t.Fatalf("got %d lines %v, want %d lines %v", len(got), got, len(tc.want), tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("line %d: got %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestTransform(t *testing.T) {
	t.Parallel()

	sampleTemplate := `//go:build generate

// Copyright (c) 2026 Karl Gaissmaier

//go:generate go run ../../scripts/gen.go node-methods

package nodes

// ### GENERATE DELETE START ###
type _NODE_TYPE[T any] struct{}
// ### GENERATE DELETE END ###

func (n *_NODE_TYPE[T]) Equal(o *_NODE_TYPE[T]) bool {
	// ### GENERATE SKIP_LITE START ###
	if n.Val != o.Val {
		return false
	}
	// ### GENERATE SKIP_LITE END ###
	return true
}
`

	tests := []struct {
		name         string
		src          string
		templateFile string
		tgt          target
		skipLite     bool
		want         string
	}{
		{
			name:         "empty source returns empty string",
			src:          "",
			templateFile: "commonmethods_tmpl.go",
			tgt:          target{prefix: "bart"},
			skipLite:     true,
			want:         "",
		},
		{
			name:         "bart node with skipLite enabled keeps body and strips markers",
			src:          sampleTemplate,
			templateFile: "commonmethods_tmpl.go",
			tgt: target{
				prefix: "bart",
				lite:   false,
				repl:   []replacement{{"_NODE_TYPE", "BartNode"}},
			},
			skipLite: true,
			want: `// Code generated from file "commonmethods_tmpl.go"; DO NOT EDIT.

// Copyright (c) 2026 Karl Gaissmaier


package nodes


func (n *BartNode[T]) Equal(o *BartNode[T]) bool {
	if n.Val != o.Val {
		return false
	}
	return true
}
`,
		},
		{
			name:         "lite node with skipLite enabled strips block",
			src:          sampleTemplate,
			templateFile: "commonmethods_tmpl.go",
			tgt: target{
				prefix: "lite",
				lite:   true,
				repl:   []replacement{{"_NODE_TYPE", "LiteNode"}},
			},
			skipLite: true,
			want: `// Code generated from file "commonmethods_tmpl.go"; DO NOT EDIT.

// Copyright (c) 2026 Karl Gaissmaier


package nodes


func (n *LiteNode[T]) Equal(o *LiteNode[T]) bool {
	return true
}
`,
		},
		{
			name:         "skipLite disabled preserves SKIP_LITE markers",
			src:          sampleTemplate,
			templateFile: "commontests_tmpl.go",
			tgt: target{
				prefix: "lite",
				lite:   true,
				repl:   []replacement{{"_NODE_TYPE", "LiteNode"}},
			},
			skipLite: false,
			want: `// Code generated from file "commontests_tmpl.go"; DO NOT EDIT.

// Copyright (c) 2026 Karl Gaissmaier


package nodes


func (n *LiteNode[T]) Equal(o *LiteNode[T]) bool {
	// ### GENERATE SKIP_LITE START ###
	if n.Val != o.Val {
		return false
	}
	// ### GENERATE SKIP_LITE END ###
	return true
}
`,
		},
		{
			name:         "table methods multiple replacements",
			src:          "func (t *_TABLE_TYPE[T]) Init() { t.root = new(_NODE_TYPE[T]) }\n",
			templateFile: "commonmethods_tmpl.go",
			tgt: target{
				prefix: "lite",
				lite:   true,
				repl:   []replacement{{"_TABLE_TYPE", "liteTable"}, {"_NODE_TYPE", "LiteNode"}},
			},
			skipLite: false,
			want: "// Code generated from file \"commonmethods_tmpl.go\"; DO NOT EDIT.\n" +
				"func (t *liteTable[T]) Init() { t.root = new(LiteNode[T]) }\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := transform(tc.src, tc.templateFile, tc.tgt, tc.skipLite)
			if got != tc.want {
				t.Errorf("transform() mismatch:\ngot:\n%s\nwant:\n%s", got, tc.want)
			}
		})
	}
}

func TestEnvRunErrors(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	todoFile := filepath.Join(tmpDir, "commontodo_tmpl.go")
	if err := os.WriteFile(todoFile, []byte("package main\n// TODO: fix this\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		args    []string
		gofile  string
		wantErr string
	}{
		{
			name:    "no arguments",
			args:    nil,
			gofile:  "",
			wantErr: "usage: gen <mode> [template file]",
		},
		{
			name:    "too many arguments",
			args:    []string{"node-methods", "a.go", "b.go"},
			gofile:  "",
			wantErr: "usage: gen <mode> [template file]",
		},
		{
			name:    "unknown mode",
			args:    []string{"invalid-mode"},
			gofile:  "commonmethods_tmpl.go",
			wantErr: `unknown mode "invalid-mode"`,
		},
		{
			name:    "missing template file",
			args:    []string{"node-methods"},
			gofile:  filepath.Join(tmpDir, "nonexistent_tmpl.go"),
			wantErr: "not found",
		},
		{
			name:    "template contains TODO",
			args:    []string{"node-methods", todoFile},
			gofile:  "",
			wantErr: "found pattern 'TODO'",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var out, errOut bytes.Buffer
			e := env{out: &out, errOut: &errOut}

			err := e.run(tc.args, tc.gofile)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestEnvRunSuccess(t *testing.T) {
	// Not parallel because os.Chdir modifies process-wide working directory.
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// Isolate PATH so external tools (goimports/gofumpt) are deterministically skipped
	// and do not introduce external environment dependencies or formatting diffs.
	t.Setenv("PATH", "")

	tmplContent := `//go:build generate
package nodes

// ### GENERATE DELETE START ###
type _NODE_TYPE[T any] struct{}
// ### GENERATE DELETE END ###

func (n *_NODE_TYPE[T]) Size() int {
	// ### GENERATE SKIP_LITE START ###
	return 42
	// ### GENERATE SKIP_LITE END ###
	return 0
}
`
	tmplName := "commonmethods_tmpl.go"
	if err := os.WriteFile(tmplName, []byte(tmplContent), 0o600); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	e := env{out: &out, errOut: &errOut}

	if err := e.run([]string{"node-methods"}, tmplName); err != nil {
		t.Fatalf("run() failed unexpectedly: %v", err)
	}

	// Verify warning logged when formatters are absent from PATH
	if !strings.Contains(out.String(), "goimports not found") {
		t.Errorf("expected warning about missing goimports in stdout, got:\n%s", out.String())
	}

	// Verify bartmethodsgenerated.go
	bartBytes, err := os.ReadFile("bartmethodsgenerated.go")
	if err != nil {
		t.Fatalf("failed to read bartmethodsgenerated.go: %v", err)
	}
	bartSrc := string(bartBytes)
	if !strings.Contains(bartSrc, "func (n *BartNode[T]) Size() int") || !strings.Contains(bartSrc, "return 42") {
		t.Errorf("unexpected bartmethodsgenerated.go content:\n%s", bartSrc)
	}

	// Verify litemethodsgenerated.go
	liteBytes, err := os.ReadFile("litemethodsgenerated.go")
	if err != nil {
		t.Fatalf("failed to read litemethodsgenerated.go: %v", err)
	}
	liteSrc := string(liteBytes)
	if !strings.Contains(liteSrc, "func (n *LiteNode[T]) Size() int") || strings.Contains(liteSrc, "return 42") {
		t.Errorf("unexpected litemethodsgenerated.go content:\n%s", liteSrc)
	}
}
