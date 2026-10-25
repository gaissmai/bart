## [Unreleased]

### Deprecated

- `bart.Fast` is deprecated and now a generic type alias for `bart.Table`.
  Normally, existing code keeps compiling. The alias will be removed in a future release.
  With Go 1.26 or later, `go fix ./...` replaces `bart.Fast` with `bart.Table`
  automatically, see the README section "Migrating from `bart.Fast`".

### Changed

- All tables now use a rank cache in the sparse arrays of their nodes, so that
  finding a child or a prefix in a node needs one popcount instead of four.
  This replaces the per-node cache of `bart.Fast`.
- Each trie node grows by up to 16 bytes.

### Removed

- The 256-byte child rank cache of `bart.Fast` and its separate node type.
  These were internal, no public API is affected apart from the deprecation
  above.
