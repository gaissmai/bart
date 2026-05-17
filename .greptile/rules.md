# Language

- ALWAYS respond in English, regardless of the language used in questions or comments.

# AI Assistant Rules

When generating or modifying code:

- NEVER use Unicode characters in comments
- Use ASCII only
- Do not use emojis or typographic quotes

## Generated code and shared templates

- Files like `bartmethodsgenerated.go`, `litemethodsgenerated.go`, and `fastmethodsgenerated.go`
  are all generated from `commonmethods_tmpl.go` and share dependencies (e.g. `internal/allot`).
  Before calling a package obsolete, verify it is unused across ALL generated files, not just the
  one being changed in the PR.
