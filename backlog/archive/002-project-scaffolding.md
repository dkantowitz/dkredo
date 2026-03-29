---
id: 002
title: Create project scaffolding with go.mod, directory structure, and StampState type
status: Done
priority: 1
effort: Small
assignee: claude
created_date: 2026-03-27
labels: [feature, core]
swimlane: Core
dependencies: [001]
---

## Summary

Create the Go module, directory structure, core `StampState` type, and Justfile
with build/test/coverage targets. This is the shared foundation that all other
tickets build on. The `StampState` struct must be defined here so parallel
agents can code against it.

## Current State

No Go code exists. Directory structure from implementation spec:

```
cmd/dkredo/             — CLI dispatch + operation execution
internal/ops/           — individual operations
internal/hasher/        — BLAKE3 file hashing
internal/resolve/       — input argument resolution
internal/stamp/         — stamp I/O, state management
```

## Analysis & Recommendations

Define the `StampState` struct in `internal/stamp/state.go` as the shared type
all operations work with:

```go
// StampState holds the in-memory representation of a label's stamp file.
type StampState struct {
    Label    string
    Entries  []Entry  // sorted by path
    Modified bool     // true if any operation changed state
}

// Entry represents one file in the stamp.
type Entry struct {
    Path  string
    Facts string // raw fact string, empty if no facts computed
}
```

The Justfile should include targets from the implementation spec: `build`,
`test`, `cover`, `cover-html`, `cover-check`.

Build must produce a statically-linked binary:
```
CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(git describe --tags --always --dirty)" ./cmd/dkredo
```

## TDD Plan

### RED

```go
// internal/stamp/state_test.go
func TestNewStampState(t *testing.T) {
    s := NewStampState("my-label")
    if s.Label != "my-label" { t.Fatal("label mismatch") }
    if len(s.Entries) != 0 { t.Fatal("expected empty entries") }
    if s.Modified { t.Fatal("new state should not be modified") }
}

func TestStampStateAddEntry(t *testing.T) {
    s := NewStampState("test")
    s.AddEntry("src/main.c", "")
    if len(s.Entries) != 1 { t.Fatal("expected 1 entry") }
    if !s.Modified { t.Fatal("should be modified after add") }
}
```

### GREEN

1. Create `go.mod` with module path `dkredo` (or chosen path)
2. Create directory structure: `cmd/dkredo/`, `internal/stamp/`, `internal/ops/`, `internal/hasher/`, `internal/resolve/`
3. Create `cmd/dkredo/main.go` — minimal main that prints "dkredo dev" and exits
4. Create `internal/stamp/state.go` — `StampState` struct, `NewStampState()`, `AddEntry()`, `FindEntry()`, `RemoveEntry()` methods
5. Create `internal/stamp/state_test.go` — tests above
6. Create `Justfile` with build, test, cover, cover-html, cover-check targets
7. Verify: `just build` produces `./dkredo` binary
8. Verify: `just test` passes
9. Verify: `./dkredo` prints "dkredo dev"

### REFACTOR

1. Ensure `Entries` stays sorted by path after mutations.
2. Verify `go vet ./...` and `go test -race ./...` pass.

## Results

### Files Created
- `go.mod` — Go module definition
- `Justfile` — build/test/cover targets
- `cmd/dkredo/main.go` — CLI entry point
- `internal/stamp/state.go` — StampState, Entry types, NewStampState/AddEntry/FindEntry/RemoveEntry
- `internal/stamp/state_test.go` — unit tests

### Deviations
None. All GREEN and REFACTOR steps completed as planned.
