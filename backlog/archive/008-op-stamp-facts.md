---
id: 008
title: Implement +stamp-facts operation
status: Done
priority: 2
effort: Small
assignee: claude
created_date: 2026-03-27
labels: [feature, core]
swimlane: Core
dependencies: [003, 004, 005]
---

## Summary

Implement the `+stamp-facts` operation that computes and records BLAKE3 hash
and size facts for files already in the stamp's name list. This is the "record
current state" step — used after a successful build to snapshot dependency state.

## Current State

After tickets 003, 004, 005: stamp I/O, hasher, and resolver packages exist.

## Analysis & Recommendations

```go
// internal/ops/stamp_facts.go
func StampFacts(state *stamp.StampState, args []string, stdin io.Reader, stampsParent string, verbose bool) error
```

Behavior:
1. Resolve filter args (empty filter = all entries in stamp)
2. For each matching entry in `state.Entries`:
   - Call `hasher.FileFacts(fullPath)` to compute blake3+size or missing:true
   - Update `entry.Facts` with the result
   - Set `state.Modified = true`
3. Does NOT add names — operates only on names already in the stamp
4. If `-v`: log per-file facts to stderr

**Critical:** `+stamp-facts` with a `-M` filter does NOT add names from the
depfile. `-M` as a filter on `+stamp-facts` would filter existing entries by
paths extracted from the depfile — but this is unusual. The normal pattern is
`+add-names -M file.d +stamp-facts` (two operations). The implementation
spec's test case confirms: `-M` without `+add-names` does not add entries.

## TDD Plan

### RED

```go
// internal/ops/stamp_facts_test.go
func TestStampFactsAll(t *testing.T) {
    // Create temp files a.c, b.c
    state := stamp.NewStampState("test")
    state.AddEntry("a.c", "")
    state.AddEntry("b.c", "")
    err := StampFacts(state, []string{}, nil, tmpDir, false)
    assert(err == nil)
    assert(strings.HasPrefix(state.Entries[0].Facts, "blake3:"))
    assert(strings.Contains(state.Entries[0].Facts, "size:"))
    assert(strings.HasPrefix(state.Entries[1].Facts, "blake3:"))
    assert(state.Modified)
}

func TestStampFactsByFilter(t *testing.T) {
    // stamp has a.c, b.h; +stamp-facts .c → only a.c gets facts
    state := stamp.NewStampState("test")
    state.AddEntry("a.c", "")
    state.AddEntry("b.h", "")
    err := StampFacts(state, []string{".c"}, nil, tmpDir, false)
    assert(state.Entries[0].Facts != "")  // a.c stamped
    assert(state.Entries[1].Facts == "")  // b.h unchanged
}

func TestStampFactsMissingFile(t *testing.T) {
    // stamp has gone.c, file doesn't exist → facts = "missing:true"
    state := stamp.NewStampState("test")
    state.AddEntry("gone.c", "")
    err := StampFacts(state, []string{}, nil, tmpDir, false)
    assert(state.Entries[0].Facts == "missing:true")
}

func TestStampFactsDeterministic(t *testing.T) {
    // Stamp same file twice → identical facts
}

func TestStampFactsUpdatesExisting(t *testing.T) {
    // Entry has old facts, re-stamp → facts updated
    state := stamp.NewStampState("test")
    state.AddEntry("a.c", "blake3:old size:1")
    // modify a.c content
    StampFacts(state, []string{}, nil, tmpDir, false)
    assert(state.Entries[0].Facts != "blake3:old size:1")
}

func TestStampFactsSymlink(t *testing.T) {
    // stamp has symlink → facts reflect target content
}

func TestStampFactsDoesNotAddNames(t *testing.T) {
    // stamp has only a.c; +stamp-facts with no filter
    // Even if b.c exists on disk, it's NOT added
    state := stamp.NewStampState("test")
    state.AddEntry("a.c", "")
    StampFacts(state, []string{}, nil, tmpDir, false)
    assert(len(state.Entries) == 1)  // still just a.c
}
```

### GREEN

1. Create `internal/ops/stamp_facts.go`
2. Implement filter resolution (empty = all entries)
3. For each matching entry, compute facts via hasher
4. Update entry facts, set modified flag

### REFACTOR

1. Consider parallel hashing for large file counts (future optimization, not required now).
2. Run with `-race`.

### CLI Integration Test

```bash
# Create files and stamp
echo "hello" > a.c
echo "world" > b.c
dkredo test +add-names a.c b.c +stamp-facts
# Verify .stamps/test has both entries with blake3+size facts

# Missing file stamps as missing:true
dkredo test2 +add-names nonexistent.c +stamp-facts
# Verify entry has "missing:true"
```

## Results

### Files Created
- `internal/ops/stamp_facts.go` — StampFacts operation
- `internal/ops/stamp_facts_test.go` — 5 tests

### Deviations
None.
