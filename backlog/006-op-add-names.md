---
id: 006
title: Implement +add-names operation
status: To Do
priority: 2
effort: Small
assignee: claude
created_date: 2026-03-27
labels: [feature, core]
swimlane: Core
dependencies: [003, 005]
---

## Summary

Implement the `+add-names` operation in `internal/ops/`. This is the primary
way files enter a stamp's name list. It resolves file arguments (positional,
stdin, -@, -@0, -M) and adds them to the stamp. Existing entries and their
facts are preserved. New entries get empty facts. Duplicates are ignored.

## Current State

After tickets 003 and 005, the stamp I/O and input resolver packages exist.
The operation needs to use both.

## Analysis & Recommendations

Signature:

```go
// internal/ops/add_names.go
func AddNames(state *stamp.StampState, args []string, stdin io.Reader, stampsParent string, verbose bool) error
```

Behavior:
1. Resolve args via `resolve.ResolveFiles(args, stdin, stampsParent)`
2. For each resolved path:
   - If already in `state.Entries` → skip (preserve existing facts)
   - If new → append `Entry{Path: path, Facts: ""}` and set `state.Modified = true`
3. Re-sort `state.Entries` by path
4. If `-v`: log to stderr `+add-names: added N new entries (M total)`

Exit: 0 on success, 2 on error. No meaningful exit code (not a check operation).

Key edge cases:
- `-M` depfile parsing integrated via resolver
- Mix of positional files + stdin + -@ in single invocation
- Empty args (no files given) → no-op, no error

## TDD Plan

### RED

```go
// internal/ops/add_names_test.go
func TestAddNamesToEmptyStamp(t *testing.T) {
    state := stamp.NewStampState("test")
    err := AddNames(state, []string{"src/a.c", "src/b.c"}, nil, "/project", false)
    assert(err == nil)
    assert(len(state.Entries) == 2)
    assert(state.Entries[0].Path == "src/a.c")
    assert(state.Entries[0].Facts == "")  // no facts
    assert(state.Modified == true)
}

func TestAddNamesDuplicateIgnored(t *testing.T) {
    state := stamp.NewStampState("test")
    state.AddEntry("src/a.c", "blake3:abc size:100")
    err := AddNames(state, []string{"src/a.c"}, nil, "/project", false)
    assert(err == nil)
    assert(len(state.Entries) == 1)
    assert(state.Entries[0].Facts == "blake3:abc size:100")  // facts preserved
}

func TestAddNamesPreservesExistingFacts(t *testing.T) {
    state := stamp.NewStampState("test")
    state.AddEntry("src/a.c", "blake3:abc size:100")
    state.Modified = false
    err := AddNames(state, []string{"src/b.c"}, nil, "/project", false)
    assert(err == nil)
    assert(len(state.Entries) == 2)
    assert(state.Entries[0].Facts == "blake3:abc size:100")  // a.c preserved
    assert(state.Entries[1].Facts == "")                     // b.c new, empty
}

func TestAddNamesFromStdin(t *testing.T) {
    state := stamp.NewStampState("test")
    stdin := strings.NewReader("x.c\ny.c\n")
    err := AddNames(state, []string{"-"}, stdin, "/project", false)
    assert(err == nil)
    assert(len(state.Entries) == 2)
}

func TestAddNamesFromFileInput(t *testing.T) {
    // Write temp file with paths, use -@ flag
    state := stamp.NewStampState("test")
    err := AddNames(state, []string{"-@", tmpFile}, nil, "/project", false)
    assert(err == nil)
    // entries match file contents
}

func TestAddNamesFromDepfile(t *testing.T) {
    // Write temp .d file: "out.o: src/main.c include/config.h"
    state := stamp.NewStampState("test")
    err := AddNames(state, []string{"-M", depFile}, nil, "/project", false)
    assert(err == nil)
    assert(len(state.Entries) == 2)  // main.c and config.h
}

func TestAddNamesMixedInputs(t *testing.T) {
    // Positional + -M + -@ in one call
    // Verify union, deduplication, sorted
}

func TestAddNamesEmptyArgs(t *testing.T) {
    state := stamp.NewStampState("test")
    err := AddNames(state, []string{}, nil, "/project", false)
    assert(err == nil)
    assert(len(state.Entries) == 0)
    assert(state.Modified == false)
}

func TestAddNamesDeduplicatesAcrossInputs(t *testing.T) {
    // Same file from positional and stdin → appears once
}

func TestAddNamesEntriesSorted(t *testing.T) {
    state := stamp.NewStampState("test")
    AddNames(state, []string{"z.c", "a.c", "m.c"}, nil, "/project", false)
    assert(state.Entries[0].Path == "a.c")
    assert(state.Entries[1].Path == "m.c")
    assert(state.Entries[2].Path == "z.c")
}
```

### GREEN

1. Create `internal/ops/add_names.go`
2. Implement `AddNames()` — resolve inputs, merge into state, preserve existing
3. Verbose output to stderr when enabled

### REFACTOR

1. Ensure no unnecessary allocations on the no-new-names path.
2. Run with `-race`.

### CLI Integration Test

```bash
# After CLI exists:
dkredo test-label +add-names src/a.c src/b.c
# Verify .stamps/test-label contains two entries with no facts

dkredo test-label +add-names src/c.c
# Verify .stamps/test-label now has 3 entries, a.c and b.c facts untouched
```
