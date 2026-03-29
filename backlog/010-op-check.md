---
id: 010
title: Implement +check and +check-assert operations
status: To Do
priority: 2
effort: Medium
assignee: claude
created_date: 2026-03-27
labels: [feature, core]
swimlane: Core
dependencies: [003, 004, 005]
---

## Summary

Implement the `+check` and `+check-assert` operations — the core of dkredo's
change detection. `+check` compares recorded stamp facts against the current
filesystem and returns exit codes that drive the `?` guard sigil in justfiles.

## Current State

After tickets 003, 004, 005: stamp I/O, hasher (with `CheckFact`), and
resolver exist.

## Analysis & Recommendations

### +check

```go
// internal/ops/check.go
func Check(state *stamp.StampState, args []string, stdin io.Reader, stampsParent string, verbose bool) (int, error)
```

Returns exit code:
- `0` — changed (at least one fact fails, or entry has no facts, or entry has unreadable/unknown facts)
- `1` — unchanged (all facts hold)
- `2` — error

Decision logic per spec:
1. Resolve filter args (empty = all entries)
2. If no entries match filter → exit 1 (empty stamp passes)
3. For each matching entry:
   - No facts (`entry.Facts == ""`) → changed (exit 0)
   - Unreadable fact line → changed (exit 0) + warning to stderr
   - Unknown fact key → changed (exit 0) + warning to stderr
   - `missing:true` + file now exists → changed
   - Has blake3/size + file now missing → changed
   - Size differs → changed (fast path, skip hash)
   - Hash differs → changed
4. If ALL entries pass → exit 1 (unchanged)

**Special pipeline behavior:** `+check` returning exit 1 stops the operation
pipeline but does NOT prevent writing pending stamp modifications. The executor
handles this — `+check` just returns its exit code.

### +check-assert

```go
func CheckAssert(state *stamp.StampState, args []string, stdin io.Reader, stampsParent string, verbose bool) (int, error)
```

Same as `+check` but exit 2 instead of exit 1 when unchanged.

## TDD Plan

### RED

```go
// internal/ops/check_test.go
func TestCheckNoStamp(t *testing.T) {
    // First run — no stamp file exists
    state := stamp.NewStampState("test")  // empty state
    code, err := Check(state, []string{}, nil, tmpDir, false)
    assert(err == nil)
    assert(code == 1)  // empty stamp → unchanged (nothing to check)
}

func TestCheckAllFactsMatch(t *testing.T) {
    // Create file, stamp it, check → exit 1 (unchanged)
    writeFile("a.c", "hello")
    state := stamp.NewStampState("test")
    state.AddEntry("a.c", computeFacts("a.c"))
    code, _ := Check(state, []string{}, nil, tmpDir, false)
    assert(code == 1)
}

func TestCheckContentChanged(t *testing.T) {
    // Create file, stamp, modify content, check → exit 0
    writeFile("a.c", "hello")
    state := stamp.NewStampState("test")
    state.AddEntry("a.c", computeFacts("a.c"))
    writeFile("a.c", "world")
    code, _ := Check(state, []string{}, nil, tmpDir, false)
    assert(code == 0)
}

func TestCheckSizeChangedFastPath(t *testing.T) {
    // Stamp file, change size → exit 0 (hash NOT computed)
    writeFile("a.c", "hi")
    state := stamp.NewStampState("test")
    state.AddEntry("a.c", computeFacts("a.c"))
    writeFile("a.c", "hello world")  // different size
    code, _ := Check(state, []string{}, nil, tmpDir, false)
    assert(code == 0)
}

func TestCheckFileAppeared(t *testing.T) {
    // missing:true → file now exists → exit 0
    state := stamp.NewStampState("test")
    state.AddEntry("a.c", "missing:true")
    writeFile("a.c", "hello")
    code, _ := Check(state, []string{}, nil, tmpDir, false)
    assert(code == 0)
}

func TestCheckFileDisappeared(t *testing.T) {
    // blake3+size facts → file deleted → exit 0
    writeFile("a.c", "hello")
    state := stamp.NewStampState("test")
    state.AddEntry("a.c", computeFacts("a.c"))
    os.Remove("a.c")
    code, _ := Check(state, []string{}, nil, tmpDir, false)
    assert(code == 0)
}

func TestCheckUnknownFactKey(t *testing.T) {
    // facts with "future:xyz" → exit 0 + warning on stderr
    state := stamp.NewStampState("test")
    state.AddEntry("a.c", "blake3:abc size:5 future:xyz")
    code, _ := Check(state, []string{}, nil, tmpDir, false)
    assert(code == 0)
    // Verify warning emitted to stderr
}

func TestCheckUnreadableFactLine(t *testing.T) {
    // Garbage fact string → exit 0 + warning
    state := stamp.NewStampState("test")
    state.AddEntry("a.c", "not-a-valid-fact")
    code, _ := Check(state, []string{}, nil, tmpDir, false)
    assert(code == 0)
}

func TestCheckNoFacts(t *testing.T) {
    // Entry with empty facts → exit 0 (changed)
    state := stamp.NewStampState("test")
    state.AddEntry("a.c", "")
    code, _ := Check(state, []string{}, nil, tmpDir, false)
    assert(code == 0)
}

func TestCheckWithFilter(t *testing.T) {
    // stamp has a.c (changed), b.h (unchanged); +check .h → exit 1
    // Only b.h is checked
}

func TestCheckEmptyStamp(t *testing.T) {
    // Stamp exists but has no entries → exit 1 (nothing to check)
    state := stamp.NewStampState("test")
    code, _ := Check(state, []string{}, nil, tmpDir, false)
    assert(code == 1)
}

// +check-assert tests
func TestCheckAssertChanged(t *testing.T) {
    // Same as check changed → exit 0
}

func TestCheckAssertUnchanged(t *testing.T) {
    // All facts match → exit 2 (error — should not be called when up to date)
    state := stamp.NewStampState("test")
    state.AddEntry("a.c", computeFacts("a.c"))
    code, _ := CheckAssert(state, []string{}, nil, tmpDir, false)
    assert(code == 2)
}
```

### GREEN

1. Create `internal/ops/check.go`
2. Implement `Check()`:
   - Filter entries
   - Empty matches → return 1
   - For each: delegate to `hasher.CheckFact()`
   - Handle no-facts, unknown keys, unreadable facts
   - First changed entry → return 0
   - All pass → return 1
3. Implement `CheckAssert()` — wrap Check, map exit 1 → exit 2
4. Verbose output with reason for change

### REFACTOR

1. Ensure the size fast path is actually taken (no hash computation when size differs).
2. Verify warning messages go to stderr with format: `warning: <label>: <message>`.
3. Run with `-race`.

### CLI Integration Test

```bash
# Full guard/build/stamp cycle
echo "hello" > a.c
dkredo test +add-names a.c +stamp-facts
dkredo test +check        # exit 1 (unchanged)
echo "world" > a.c
dkredo test +check        # exit 0 (changed)

# +check-assert
dkredo test +stamp-facts
dkredo test +check-assert  # exit 2 (unchanged = error for assert)
```
