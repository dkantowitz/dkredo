---
id: 012
title: Implement CLI parser and operation executor pipeline
status: Done
priority: 1
effort: Medium
assignee: claude
created_date: 2026-03-27
labels: [feature, core]
swimlane: Core
dependencies: [003, 006, 007, 008, 009, 010, 011]
---

## Summary

Implement the generic CLI parser that extracts the label, splits `+operation`
arguments, and the executor that dispatches operations sequentially, threading
`StampState` through the pipeline. This is the central dispatch layer in
`cmd/dkredo/`.

## Current State

After operation tickets (006-011), all operations exist as functions in
`internal/ops/`. They need a CLI front-end to parse arguments and call them.

## Analysis & Recommendations

### Parser

Per spec (`dkredo-implementation.md` lines 266-298):

```
dkredo <label> [+operation [args...]]...
```

1. Extract label from `os.Args[1]` (first positional arg after any global flags)
2. Split remaining args on `+` boundaries — each segment is one operation
3. First token after `+` is the operation name, rest are its args
4. Parser consumes greedily until next `+` or end of args

Global flags that appear BEFORE the label: `-v`, `--stamps-dir <path>`,
`--version`, `--help`/`-h`.

Error cases:
- `dkredo label` with no operations and no `--cmd` → exit 2, print help
- Unknown `+operation` → exit 2 with error

### Executor

Per spec (`dkredo-implementation.md` lines 376-411):

```go
func Execute(label string, ops []Operation, stampsDir string, verbose bool) int {
    state := stamp.ReadStamp(stampsDir, label)  // empty if no file
    exitCode := 0
    for _, op := range ops {
        exitCode = op.Run(state)
        if exitCode != 0 {
            break  // stop on first non-zero
        }
    }
    if state.Modified {
        stamp.WriteStamp(stampsDir, state)  // ALWAYS write, even after exit 1
    }
    return exitCode
}
```

**Critical: +check exit 1 still writes.** The pipeline stops, but pending
stamp modifications (e.g., from `+add-names` earlier) MUST be persisted.

### Operation dispatch table

```go
var opDispatch = map[string]OpFunc{
    "add-names":    ops.AddNames,
    "remove-names": ops.RemoveNames,
    "stamp-facts":  ops.StampFacts,
    "clear-facts":  ops.ClearFacts,
    "check":        ops.Check,
    "check-assert": ops.CheckAssert,
    "names":        ops.Names,
    "facts":        ops.Facts,
}
```

## TDD Plan

### RED

```go
// cmd/dkredo/parse_test.go
func TestParseLabel(t *testing.T) {
    args := []string{"my-label", "+add-names", "a.c", "b.c"}
    label, ops, err := Parse(args)
    assert(label == "my-label")
    assert(len(ops) == 1)
    assert(ops[0].Name == "add-names")
    assert(ops[0].Args == ["a.c", "b.c"])
}

func TestParseMultipleOps(t *testing.T) {
    args := []string{"label", "+add-names", "a.c", "+check"}
    label, ops, _ := Parse(args)
    assert(len(ops) == 2)
    assert(ops[0].Name == "add-names")
    assert(ops[1].Name == "check")
    assert(len(ops[1].Args) == 0)
}

func TestParseNoOps(t *testing.T) {
    args := []string{"label"}
    _, _, err := Parse(args)
    assert(err != nil)  // no operations = error
}

func TestParseUnknownOp(t *testing.T) {
    args := []string{"label", "+bogus"}
    _, _, err := Parse(args)
    assert(err != nil)
}

func TestParseGlobalFlags(t *testing.T) {
    args := []string{"-v", "label", "+check"}
    cfg, label, ops, _ := ParseFull(args)
    assert(cfg.Verbose == true)
    assert(label == "label")
}

// cmd/dkredo/execute_test.go
func TestExecutePipelineStopsOnNonZero(t *testing.T) {
    // +check returns 1 (unchanged) → pipeline stops
    // But state.Modified changes from +add-names ARE written
}

func TestExecuteWritesOnExit1(t *testing.T) {
    // +add-names a.c +check (exit 1)
    // Verify a.c is in stamp file after execution
}

func TestExecuteNoWriteIfUnmodified(t *testing.T) {
    // +check only → state not modified → no stamp write
}
```

### GREEN

1. Create `cmd/dkredo/parse.go` — argument parsing
2. Create `cmd/dkredo/execute.go` — executor pipeline
3. Create `cmd/dkredo/dispatch.go` — operation dispatch table
4. Update `cmd/dkredo/main.go` — wire parsing → execution → exit code
5. Verify: `dkredo label +add-names a.c +check` works end-to-end

### REFACTOR

1. Ensure exit codes propagate correctly to `os.Exit()`.
2. Verify error messages go to stderr.
3. Run with `-race`.

### CLI Integration Test

```bash
# Full pipeline
echo "hello" > a.c
dkredo test +add-names a.c +stamp-facts
echo $?  # 0

dkredo test +check
echo $?  # 1 (unchanged)

echo "modified" > a.c
dkredo test +check
echo $?  # 0 (changed)

# Pipeline stops on +check but writes persist
dkredo test2 +add-names a.c +check
echo $?  # 0 (no stamp yet → changed)
# Verify .stamps/test2 exists with a.c entry

# No operations = error
dkredo test3
echo $?  # 2

# Unknown operation
dkredo test3 +bogus
echo $?  # 2
```

## Results

### Files Created
- `cmd/dkredo/parse.go` — Parse function, Operation/Config types, parseOps helper
- `cmd/dkredo/parse_test.go` — 8 parser tests
- `cmd/dkredo/execute.go` — Execute function, runOp dispatcher
- `cmd/dkredo/execute_test.go` — 5 executor tests including pipeline persistence
- `cmd/dkredo/main.go` — CLI entry point with flag handling

### Deviations
None. Pipeline correctly breaks on non-zero exit but always writes modified state.
