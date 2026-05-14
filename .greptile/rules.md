# Language

- ALWAYS respond in English, regardless of the language used in questions or comments.

# AI Assistant Rules

When generating or modifying code:

- Do not use emojis or typographic quotes
- NEVER use Unicode characters, ALWAYS use plain ASCII.
- Do not flag issues in PR reviews that are already covered by linters.
- Only comment on logic, correctness, architecture, security, and issues that linters would not catch.
- Whenever you conduct a PR review, ALWAYS post a message at the end, even if you didn't find anything.

# Module Overview Files
When reviewing any PR in this repository, ALWAYS consult the following files for context:
- `README.md` — public API and usage documentation
- `NODETYPES.md` — describes the different node types: BartNode, FastNode, LiteNode
- `bart.go` — core Table implementation with BartNode
- `fast.go` — Fast implementation with FastNode
- `lite.go` — Lite implementation with LiteNode
- `internal/nodes/bart.go`
- `internal/nodes/fast.go`
- `internal/nodes/lite.go`

These files describe the module's architecture and design intent.

# Generated code and shared templates

- If you are analyzing files that contain the following first line:
  // Code generated from file “...”; DO NOT EDIT.
  then look at the source file from which this file was generated,
  and NEVER make any changes or suggestions to generated files.

- e.g., Files like `bartmethodsgenerated.go`, `litemethodsgenerated.go`, and `fastmethodsgenerated.go`
  are all generated from `commonmethods_tmpl.go` and share dependencies (e.g. `internal/allot`).
  Before calling a package obsolete, verify it is unused across ALL generated files, not just the
  one being changed in the PR.

- When reviewing Go code that uses `//go:build generate` build constraints on
  template/generator files: `go generate` automatically sets the `generate` build tag when it runs
  directives.

- Do not suggest `-tags generate`
