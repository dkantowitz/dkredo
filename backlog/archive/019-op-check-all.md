---
id: 019
title: Implement +check-all operation that verifies all entries before returning
status: Done
priority: 3
effort: Small
assignee: claude
created_date: 2026-03-28
labels: [feature, core]
swimlane: Core
dependencies: []
source_file: internal/ops/check.go
---

## Summary

Add a `+check-all` operation that verifies facts for every entry in the stamp
(or matching a filter) before returning a result. Unlike `+check` which exits
on the first changed entry, `+check-all` evaluates all entries and reports
all changes.

## Current State

`+check` in `internal/ops/check.go` returns exit 0 on the **first** entry
whose facts don't match. This is correct for its purpose (fast guard for
build systems — any change is enough to trigger a rebuild). But it provides
no visibility into which other files also changed.

## Analysis & Recommendations

`+check-all` uses the same exit code semantics as `+check`:
- `0` — changed (at least one fact fails)
- `1` — unchanged (all facts hold)
- `2` — error

The difference: `+check-all` checks **every** matching entry before returning,
and with `-v` reports each changed file and reason. Without `-v`, the exit
code is the only output (same as `+check`).

```go
// internal/ops/check.go
func CheckAll(state *stamp.StampState, args []string, stdin io.Reader, stampsParent string, verbose bool) (int, error)
```

Verbose output example:
```
+check-all: src/main.c: hash differs
+check-all: include/config.h: file disappeared
+check-all: changed (2 of 5 entries)
```

vs unchanged:
```
+check-all: unchanged (5 files, all facts match)
```

### Integration points

- Add `"check-all"` to `ValidOps` in `cmd/dkredo/parse.go`
- Add `case "check-all"` to `runOp` in `cmd/dkredo/execute.go`
- Same pipeline behavior as `+check`: exit 1 stops pipeline but writes persist

## TDD Plan

### RED

```go
func TestCheckAllReportsAllChanges(t *testing.T) {
    // Create 3 files, stamp all, modify 2, run CheckAll
    // exit 0 (changed), verbose output mentions both changed files
}

func TestCheckAllUnchanged(t *testing.T) {
    // All facts match → exit 1
}

func TestCheckAllEmpty(t *testing.T) {
    // Empty stamp → exit 1
}

func TestCheckAllWithFilter(t *testing.T) {
    // Filter .c — only checks .c files, reports all changed .c files
}
```

### GREEN

1. Implement `CheckAll()` — same structure as `Check()` but accumulate
   changed entries in a slice instead of returning on first hit
2. Register in dispatch table and parser
3. Add verbose summary line with count

### REFACTOR

1. Extract shared filter/setup logic between `Check` and `CheckAll` if
   duplication is significant.
2. Run with `-race`.

## Results

Implementation followed the TDD plan exactly, no deviations.

### Files modified
- `internal/ops/check.go` — Added `CheckAll()` function
- `internal/ops/check_test.go` — Added 4 tests: `TestCheckAllReportsAllChanges`, `TestCheckAllUnchanged`, `TestCheckAllEmpty`, `TestCheckAllWithFilter`
- `cmd/dkredo/parse.go` — Added `"check-all"` to `ValidOps`
- `cmd/dkredo/execute.go` — Added `case "check-all"` to `runOp` dispatch

### Test results
All tests pass (`go test ./...`), including 4 new CheckAll tests.

### Review notes
Reviewed 2026-03-28. Core implementation in `internal/ops/check.go` is correct:
exit code semantics (0/1/2) are right, all entries are checked without
short-circuit, verbose output reports each changed file plus a summary count.
Four tests cover the key scenarios (multiple changes, unchanged, empty, filter).

The `cmd/dkredo/parse.go` and `cmd/dkredo/execute.go` changes use the old
`Config` type (pre-018/020 merge). During merge to main, `Config` was replaced
with `Flags` and `Parse`/`Execute` signatures changed. The two additive lines
(`"check-all"` in `ValidOps` and `case "check-all"` in `runOp`) will need to be
applied to the new main versions. The `runOp` internal signature is unchanged,
so this is a straightforward merge conflict resolution.
