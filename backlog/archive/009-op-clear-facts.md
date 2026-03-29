---
id: 009
title: Implement +clear-facts operation
status: Done
priority: 2
effort: Trivial
assignee: claude
created_date: 2026-03-27
labels: [feature, core]
swimlane: Core
dependencies: [003, 005]
---

## Summary

Implement the `+clear-facts` operation that removes facts from stamp entries
while preserving the file names. This is the mechanism behind `dkr-always` —
clearing facts forces the next `+check` to report "changed" since entries
with no facts always fail verification.

## Current State

After tickets 003 and 005, stamp state and filter resolution exist.

## Analysis & Recommendations

```go
// internal/ops/clear_facts.go
func ClearFacts(state *stamp.StampState, args []string, stdin io.Reader, stampsParent string, verbose bool) error
```

Behavior:
1. Resolve filter args (empty filter = all entries)
2. For each matching entry: set `entry.Facts = ""` and `state.Modified = true`
3. Names remain in the stamp
4. If `-v`: log `+clear-facts: cleared facts for N entries`

This is intentionally the simplest operation.

## TDD Plan

### RED

```go
// internal/ops/clear_facts_test.go
func TestClearFactsAll(t *testing.T) {
    state := stamp.NewStampState("test")
    state.AddEntry("a.c", "blake3:abc size:100")
    state.AddEntry("b.c", "blake3:def size:200")
    err := ClearFacts(state, []string{}, nil, "/project", false)
    assert(err == nil)
    assert(state.Entries[0].Facts == "")
    assert(state.Entries[1].Facts == "")
    assert(state.Entries[0].Path == "a.c")  // names preserved
    assert(state.Entries[1].Path == "b.c")
    assert(state.Modified)
}

func TestClearFactsByFilter(t *testing.T) {
    state := stamp.NewStampState("test")
    state.AddEntry("a.c", "blake3:abc size:100")
    state.AddEntry("b.h", "blake3:def size:200")
    err := ClearFacts(state, []string{".h"}, nil, "/project", false)
    assert(state.Entries[0].Facts == "blake3:abc size:100")  // a.c untouched
    assert(state.Entries[1].Facts == "")                     // b.h cleared
}

func TestClearFactsAlreadyEmpty(t *testing.T) {
    state := stamp.NewStampState("test")
    state.AddEntry("a.c", "")
    state.Modified = false
    err := ClearFacts(state, []string{}, nil, "/project", false)
    assert(err == nil)
    // Modified may or may not be set — clearing "" to "" is idempotent
}
```

### GREEN

1. Create `internal/ops/clear_facts.go`
2. Implement filter matching + fact clearing
3. Verbose output

### REFACTOR

1. Minimal — this is a trivial operation.
2. Run with `-race`.

### CLI Integration Test

```bash
# The "always" pattern
echo "data" > a.c
dkredo test +add-names a.c +stamp-facts
dkredo test +check        # exit 1 (unchanged)
dkredo test +clear-facts
dkredo test +check        # exit 0 (changed — no facts to verify)
```

## Results

### Files Created
- `internal/ops/clear_facts.go` — ClearFacts operation
- `internal/ops/clear_facts_test.go` — 3 tests

### Deviations
None. Idempotent clearing (empty→empty does not set Modified).
