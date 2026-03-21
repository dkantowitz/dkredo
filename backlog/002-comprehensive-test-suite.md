---
id: "002"
title: Test infrastructure and helpers
status: Done
completed_date: 2026-03-21
priority: 1
effort: Small
assignee: claude
created_date: 2026-03-21
labels: [feature, core]
swimlane: Tooling/Claude Environment
phase: 1
depends_on: ["001"]
source_file: dk-redo-implementation.md:126
---

## Summary

Create shared test helpers and infrastructure used by all implementation
tickets. This ticket does NOT write test stubs — each implementation ticket
owns its own tests (RED phase). This ticket only provides the shared
scaffolding.

## Current State

No tests exist. The test plan is fully specified in
`dk-redo-implementation.md:126-200`.

## Analysis & Recommendations

**Why not pre-write all test stubs?** Writing test stubs before any
implementation exists means guessing at function signatures, parameter types,
and return types. Each implementation ticket already has its own TDD plan
with a RED phase that defines tests. Pre-writing stubs creates duplicate work,
merge conflicts, and signature churn as packages are implemented.

This ticket creates only the shared infrastructure:

1. **Test helpers** in `internal/testutil/testutil.go`:
   - `WriteTempFile(t, dir, name, content) string` — create a temp file, return path
   - `WriteTempDir(t, dir, name, files map[string]string) string` — create dir with files
   - `RunBinary(t, binary, stampsDir, args...) (stdout, stderr string, exitCode int)` — run compiled binary

2. **Integration test skeleton** in `test/integration_test.go`:
   - `TestMain` that builds the binary once into a temp directory
   - Build tag `//go:build integration`
   - Helper that wraps `RunBinary` with the pre-built binary path

3. **Justfile updates**:
   - `test-integration` passes `-tags integration` and builds binary first
   - `test-bench` target: `go test -bench=. -benchtime=3s ./...`

4. **Benchmark skeleton** in `test/bench_test.go`:
   - Build tag `//go:build integration`
   - Empty benchmark functions that implementation tickets will fill in
   - Primary target: 300 files across 10 labels checked in < 300ms

## TDD Plan

### RED

No tests to write in RED — this ticket creates infrastructure only.

### GREEN

1. Create `internal/testutil/testutil.go` with helper functions
2. Create `test/integration_test.go` with `TestMain` and build step
3. Create `test/bench_test.go` with benchmark skeleton
4. Add `//go:build integration` tags
5. Update justfile with `test-integration` and `test-bench` targets
6. Verify `just test-unit` passes (no unit tests yet, but no failures)
7. Verify `just test-integration` builds binary and runs (no tests yet)

### REFACTOR

- Keep helpers minimal — only add what is actually needed by multiple packages

## Completion Notes

**Commit:** `085cf78`

### Files created
- `internal/testutil/testutil.go` (90 lines) — `WriteTempFile`, `WriteTempDir`, `RunBinary` helpers
- `test/integration_test.go` — skeleton with `TestMain` binary build, `//go:build integration` tag
- `test/bench_test.go` — benchmark skeleton with `//go:build integration` tag

### Justfile targets added
- `test-integration`: builds binary then runs `go test -tags integration ./test/...`
- `test-bench`: runs benchmarks with `-bench=. -benchtime=3s`

### Design decisions
- `WriteTempFile` and `WriteTempDir` accept `testing.TB` (not `*testing.T`) so they work in both tests and benchmarks
- `RunBinary` captures stdout, stderr, and exit code via `os/exec`

### Deferred work
- None
