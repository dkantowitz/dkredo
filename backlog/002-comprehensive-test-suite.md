---
id: "002"
title: Build comprehensive test suite with table-driven tests
status: To Do
priority: 1
effort: Medium
assignee: claude
created_date: 2026-03-21
labels: [feature, core]
swimlane: Core Library
phase: 1
depends_on: ["001"]
source_file: dk-redo-implementation.md:126
---

## Summary

Create the full test scaffolding with table-driven test stubs for all three
internal packages (hasher, stamp, resolve) and integration tests for the
compiled binary. Tests should be written RED first — they define the contract
that implementation tickets will make pass.

## Current State

No tests exist. The test plan is fully specified in
`dk-redo-implementation.md:126-200`.

## Analysis & Recommendations

Write tests in three locations:

1. `internal/hasher/hasher_test.go` — 11 test cases per implementation doc
2. `internal/stamp/stamp_test.go` — 17 test cases per implementation doc
3. `internal/resolve/resolve_test.go` — 7 test cases per implementation doc
4. `test/integration_test.go` — 14 integration test cases (build tag: `integration`)

Use Go table-driven test style (`[]struct{ name string; ... }`). Each test
should have a clear name matching the "Test" column in the implementation doc.

Integration tests should use `os/exec` to run the compiled `dk-redo` binary
and check exit codes. They need a `TestMain` that builds the binary once into
a temp directory before running tests.

Use `t.TempDir()` for all file system operations — no test pollution.

Helper functions to create:
- `writeTempFile(t, dir, name, content) string` — create a temp file, return path
- `writeTempDir(t, dir, name, files map[string]string) string` — create dir with files
- `runDkRedo(t, args...) (stdout, stderr string, exitCode int)` — run binary

All tests should initially fail (RED) or be marked with `t.Skip("not yet implemented")`
until implementation tickets are completed.

## TDD Plan

### RED

```go
// internal/hasher/hasher_test.go
func TestHashFile(t *testing.T) {
    tests := []struct {
        name     string
        content  *string // nil means file doesn't exist
        wantErr  bool
        wantMissing bool
    }{
        {"with content", ptr("hello"), false, false},
        {"empty file", ptr(""), false, false},
        {"missing file", nil, false, true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Skip("not yet implemented")
        })
    }
}
```

### GREEN

1. Create test files with all table entries from implementation doc
2. Write test helpers (temp file creation, binary runner)
3. Add build tag `//go:build integration` to integration tests
4. Update justfile: `test-integration` passes `-tags integration` and
   builds the binary first
5. Verify `just test` runs and all tests are skipped (not failing)

### REFACTOR

- Ensure test helper functions are in a shared `internal/testutil/` package
  if needed across packages, or keep them local if only used in one package
