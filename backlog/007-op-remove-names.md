---
id: 007
title: Implement +remove-names operation
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

Implement the `+remove-names` operation that removes entries from a stamp's
name list along with their facts. Supports exact paths, suffix filters, and
the `-ne` flag for conditional removal of files that no longer exist on disk.

## Current State

After tickets 003 and 005, stamp state and filter matching exist.

## Analysis & Recommendations

```go
// internal/ops/remove_names.go
func RemoveNames(state *stamp.StampState, args []string, stdin io.Reader, stampsParent string, verbose bool) error
```

Two modes:

**Normal mode** (`+remove-names [filter...]`):
- Empty filter → remove ALL entries (clear the stamp name list)
- With filters → remove entries matching any filter (exact path or `.suffix`)
- Removed entries lose both their name and facts

**Conditional mode** (`+remove-names -ne [filter...]`):
- `-ne` is the first arg
- For each matching entry: remove ONLY IF the file does not exist on disk
  AND the stamp fact for that file is NOT `missing:true`
- Purpose: prune entries for deleted files that were expected to exist
- Files with `missing:true` are kept (they're intentionally tracked as absent)

## TDD Plan

### RED

```go
// internal/ops/remove_names_test.go
func TestRemoveNamesByExactPath(t *testing.T) {
    state := stamp.NewStampState("test")
    state.AddEntry("a.c", "blake3:abc size:100")
    state.AddEntry("b.c", "blake3:def size:200")
    err := RemoveNames(state, []string{"a.c"}, nil, "/project", false)
    assert(err == nil)
    assert(len(state.Entries) == 1)
    assert(state.Entries[0].Path == "b.c")
    assert(state.Modified)
}

func TestRemoveNamesBySuffixFilter(t *testing.T) {
    state := stamp.NewStampState("test")
    state.AddEntry("a.c", "blake3:abc size:100")
    state.AddEntry("b.h", "blake3:def size:200")
    err := RemoveNames(state, []string{".h"}, nil, "/project", false)
    assert(err == nil)
    assert(len(state.Entries) == 1)
    assert(state.Entries[0].Path == "a.c")
}

func TestRemoveNamesAll(t *testing.T) {
    state := stamp.NewStampState("test")
    state.AddEntry("a.c", "facts")
    state.AddEntry("b.c", "facts")
    state.AddEntry("c.h", "facts")
    err := RemoveNames(state, []string{}, nil, "/project", false)
    assert(err == nil)
    assert(len(state.Entries) == 0)
    assert(state.Modified)
}

func TestRemoveNamesNE_FileExists(t *testing.T) {
    // Create real temp file "a.c"
    state := stamp.NewStampState("test")
    state.AddEntry("a.c", "blake3:abc size:100")  // file exists on disk
    err := RemoveNames(state, []string{"-ne"}, nil, tmpDir, false)
    assert(err == nil)
    assert(len(state.Entries) == 1)  // NOT removed — file exists
}

func TestRemoveNamesNE_FileMissingFactMissing(t *testing.T) {
    state := stamp.NewStampState("test")
    state.AddEntry("gone.c", "missing:true")  // file missing, fact says missing
    err := RemoveNames(state, []string{"-ne"}, nil, tmpDir, false)
    assert(err == nil)
    assert(len(state.Entries) == 1)  // NOT removed — expected absent
}

func TestRemoveNamesNE_FileMissingFactStale(t *testing.T) {
    state := stamp.NewStampState("test")
    state.AddEntry("gone.c", "blake3:abc size:100")  // file missing, fact says it existed
    err := RemoveNames(state, []string{"-ne"}, nil, tmpDir, false)
    assert(err == nil)
    assert(len(state.Entries) == 0)  // REMOVED — file gone, was expected to exist
}

func TestRemoveNamesNonexistentName(t *testing.T) {
    state := stamp.NewStampState("test")
    state.AddEntry("a.c", "facts")
    err := RemoveNames(state, []string{"x.c"}, nil, "/project", false)
    assert(err == nil)
    assert(len(state.Entries) == 1)  // no change
}

func TestRemoveNamesNE_WithFilter(t *testing.T) {
    // -ne with .c filter → only check .c files for removal
}
```

### GREEN

1. Create `internal/ops/remove_names.go`
2. Implement normal mode: parse filters, match against entries, remove matches
3. Implement `-ne` mode: detect flag, check filesystem existence + fact type
4. Verbose output

### REFACTOR

1. Ensure `-ne` stat calls don't dominate for large stamps (they're cheap, but verify).
2. Run with `-race`.

### CLI Integration Test

```bash
# Normal removal
dkredo test +add-names a.c b.c c.h
dkredo test +remove-names .h
# Verify: only a.c, b.c remain

# Clear all
dkredo test +remove-names
# Verify: stamp empty

# -ne removal (file deleted)
dkredo test +add-names real.c gone.c +stamp-facts
rm gone.c
dkredo test +remove-names -ne
# Verify: gone.c removed, real.c stays
```
