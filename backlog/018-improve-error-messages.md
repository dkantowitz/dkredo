---
id: 018
title: Improve error messages
status: Done
priority: 3
effort: Trivial
assignee: claude
created_date: 2026-03-28
labels: [enhancement, core]
swimlane: Core
dependencies: []
source_file: cmd/dkredo/parse.go
---

## Summary

When the user forgets the label, the error message is misleading. For example:

```
$ ./dkredo +add-names -@ <(find . -iname *.go)
error: expected +operation, got "-@"
```

The real problem is that `+add-names` was parsed as the label, then `-@` is
unexpected because it's not a `+operation`. The user sees an error about `-@`
when the actual mistake is the missing label.

## Current State

`cmd/dkredo/parse.go` treats the first positional arg after global flags as
the label unconditionally. If that arg starts with `+`, it's consumed as the
label and the remaining args fail to parse.

## Analysis & Recommendations

After extracting the label, check if it starts with `+`. If so, emit a
targeted error:

```
error: missing label — first argument "+add-names" looks like an operation, not a label
usage: dkredo <label> [+operation [args...]]...
```

This is a simple string prefix check on the label before continuing to parse
operations. No ambiguity — a valid label never starts with `+` (the `+` prefix
is reserved for operations by design).

```go
// In Parse(), after extracting label:
if strings.HasPrefix(label, "+") {
    return cfg, "", nil, fmt.Errorf(
        "missing label — first argument %q looks like an operation, not a label\n"+
        "usage: dkredo <label> [+operation [args...]]...", label)
}
```

## TDD Plan

### RED

```go
func TestParseMissingLabelWithOperation(t *testing.T) {
    _, _, _, err := Parse([]string{"+add-names", "a.c"})
    if err == nil {
        t.Fatal("expected error")
    }
    if !strings.Contains(err.Error(), "missing label") {
        t.Fatalf("expected 'missing label' error, got: %v", err)
    }
}
```

### GREEN

Add the `strings.HasPrefix(label, "+")` check in `Parse()` after extracting
the label, before parsing operations.

### REFACTOR

Verify the error message is clear when combined with argv[0] alias dispatch
(e.g., `dkr-ifchange +add-names` should also produce a helpful message).

## Results

Implemented as planned with no deviations from the TDD plan.

- Added `strings.HasPrefix(label, "+")` check in `Parse()` after extracting the label (line 62 of `cmd/dkredo/parse.go`)
- When triggered, returns error: `missing label — first argument "+add-names" looks like an operation, not a label`
- Added `TestParseMissingLabelWithOperation` test in `cmd/dkredo/parse_test.go`
- All existing tests continue to pass (`go test ./...` green across all packages)

### Files modified

- `cmd/dkredo/parse.go` — Added label prefix check
- `cmd/dkredo/parse_test.go` — Added test and `strings` import
- `backlog/018-improve-error-messages.md` — Updated status and added results

### Review notes

Reviewed 2026-03-28. Implementation matches the ticket spec exactly:
- `strings.HasPrefix(label, "+")` check is in the correct location (after label extraction, before operation parsing)
- Error message satisfies all three requirements: contains "missing label", shows the offending argument via `%q`, and includes a usage hint
- `TestParseMissingLabelWithOperation` test correctly validates the error path
- All tests pass (`go test ./...` green across all 5 packages)
- No issues found; no changes needed beyond setting status to Done
