---
id: 020
title: Implement scoped flags framework (global and per-operation)
status: To Do
priority: 2
effort: Medium
assignee: claude
created_date: 2026-03-28
labels: [enhancement, core]
swimlane: Core
dependencies: []
source_file: cmd/dkredo/parse.go
---

## Summary

Introduce a flags framework where flags like `-v` and `--stamps-dir` can appear
in two positions: before the label (global scope) or immediately after a
`+operation` (operation scope). Global flags apply to all operations.
Operation-scoped flags override or augment the global defaults for that
operation only.

## Current State

`cmd/dkredo/parse.go` defines `Config` with `Verbose` and `StampsDir` fields,
parsed only from global position (before the label). Operations receive
`verbose bool` and `stampsParent string` as flat arguments. There is no
mechanism for per-operation flag overrides.

## Analysis & Recommendations

### Flags struct

Replace the flat `Config` with a structured `Flags` type that holds all
recognized flags. A global instance is populated from pre-label args. Each
operation receives a copy that can be updated with operation-local flags.

```go
// Flags holds all flags that can appear globally or per-operation.
type Flags struct {
    Verbose   bool
    StampsDir string
}
```

### Copy-on-write semantics

The global `Flags` is populated once. For each operation, create a **copy**
of the global flags, then apply any operation-local flags on top. This way
operation-local flags don't leak to subsequent operations.

```go
globalFlags := parseGlobalFlags(args)

for _, op := range operations {
    opFlags := globalFlags          // copy (value type, not pointer)
    op.Args = extractFlags(&opFlags, op.Args)
    // opFlags now has operation-local overrides
    runOp(op, state, opFlags, ...)
}
```

### Shared flag extraction

A single utility function parses recognized flags from an arg slice and
updates a `Flags` struct. Used in two places:

1. **Global position:** parse flags before the label
2. **Operation position:** parse flags from each operation's args

```go
// ExtractFlags removes recognized flags from args, applies them to flags,
// and returns the remaining args.
func ExtractFlags(flags *Flags, args []string) []string {
    var remaining []string
    i := 0
    for i < len(args) {
        switch args[i] {
        case "-v":
            flags.Verbose = true
            i++
        case "--stamps-dir":
            i++
            if i < len(args) {
                flags.StampsDir = args[i]
                i++
            }
        default:
            remaining = append(remaining, args[i])
            i++
        }
    }
    return remaining
}
```

This is the single source of truth for which flags exist and how they're
parsed — adding a new flag means adding one `case` branch.

### Operation signature change

Operations currently receive individual flag values:

```go
func AddNames(state *stamp.StampState, args []string, stdin io.Reader, stampsParent string, verbose bool) error
```

Change to receive the `Flags` struct:

```go
func AddNames(state *stamp.StampState, args []string, stdin io.Reader, flags Flags) error
```

The `stampsParent` is derived from `flags.StampsDir` by the executor (since
it requires directory resolution logic). Pass it alongside or compute inside.
Two options:

**Option A:** `Flags` includes a resolved `StampsParent` field set by the executor.

**Option B:** Operations receive `(flags Flags, stampsParent string)`.

Option A is cleaner — the executor resolves `StampsDir` → `StampsParent` once
and stores it in the flags struct before dispatching.

```go
type Flags struct {
    Verbose      bool
    StampsDir    string
    StampsParent string  // resolved by executor, read-only for operations
}
```

### Parsing flow

```
argv:  dkredo -v --stamps-dir /tmp/s label +add-names a.c +check -v .c

       ├── global flags ──┤  label  ├─ op1 args ─┤ ├── op2 args ──┤

global:  Verbose=true, StampsDir="/tmp/s"
op1:     inherits global (no local flags)
op2:     Verbose=true (redundant), filter args=[".c"]
```

### Migration

This is a refactor of internal signatures — no user-visible behavior changes
except that `-v` and `--stamps-dir` now also work after `+operation`. All
existing tests should continue to pass since global-only usage is unchanged.

## TDD Plan

### RED

```go
func TestExtractFlagsVerbose(t *testing.T) {
    f := Flags{}
    remaining := ExtractFlags(&f, []string{"-v", "a.c", "b.c"})
    assert(f.Verbose == true)
    assert(remaining == ["a.c", "b.c"])
}

func TestExtractFlagsStampsDir(t *testing.T) {
    f := Flags{}
    remaining := ExtractFlags(&f, []string{"--stamps-dir", "/tmp/s", "a.c"})
    assert(f.StampsDir == "/tmp/s")
    assert(remaining == ["a.c"])
}

func TestExtractFlagsNoFlags(t *testing.T) {
    f := Flags{Verbose: true}
    remaining := ExtractFlags(&f, []string{"a.c", ".c"})
    assert(f.Verbose == true)  // unchanged
    assert(remaining == ["a.c", ".c"])
}

func TestOperationLocalVerbose(t *testing.T) {
    // Parse: label +add-names a.c +check -v
    // op1 (add-names): verbose=false (global default)
    // op2 (check): verbose=true (operation-local)
}

func TestOperationLocalDoesNotLeakToNext(t *testing.T) {
    // Parse: label +check -v +stamp-facts
    // op1 (check): verbose=true
    // op2 (stamp-facts): verbose=false (global default, not leaked)
}

func TestGlobalPlusOperationLocal(t *testing.T) {
    // Parse: -v label +check -v +stamp-facts
    // All operations verbose (global), check also has local (redundant)
}
```

### GREEN

1. Create `Flags` struct and `ExtractFlags` function
2. Refactor `Parse()` to use `ExtractFlags` for global flags
3. Refactor executor to copy global flags per-operation and call `ExtractFlags`
   on each operation's args
4. Update all operation signatures to accept `Flags` instead of individual args
5. Update all operation tests
6. Verify all existing tests pass

### REFACTOR

1. Remove old `Config` struct (replaced by `Flags`).
2. Ensure `ExtractFlags` is the single point for adding new flags.
3. Run with `-race`.
