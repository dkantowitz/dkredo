---
id: "008"
title: Implement dk-ifchange command
status: To Do
priority: 3
effort: Medium
assignee: claude
created_date: 2026-03-21
labels: [feature, core]
swimlane: Core Library
phase: 3
depends_on: ["005", "006", "007"]
source_file: dk-redo.md:170
---

## Summary

Implement the `dk-ifchange` command — the guard that skips a recipe when
inputs are unchanged. This is the most-used command and the core of dk-redo's
value proposition.

## Current State

All internal packages are implemented (hasher, stamp, resolve) from phase 2.
CLI dispatch exists from ticket 007. The `ifchange` case in the switch
statement is a stub.

## Analysis & Recommendations

Algorithm per `dk-redo-implementation.md:59-73`:

```
1. Parse flags (-v, -q, -n), extract label (arg[0]) and inputs (arg[1:])
2. Resolve inputs via resolve.Resolve()
3. Hash all resolved files via hasher.HashFile/HashDir
4. Read stamp file via stamp.Read()
5. If no stamp exists: exit 0 (first run — changed)
6. Compare stamp vs current facts via stamp.Compare()
7. If changed: exit 0 (recipe continues)
8. If unchanged: exit 1 (? sigil stops recipe)
9. On error: exit 2
```

Exit codes per `dk-redo.md:193-197`:
- `0` — inputs changed (or first run) — recipe continues
- `1` — inputs unchanged — `?` sigil stops recipe silently
- `2` — error (corrupt stamp, I/O error)

Flags specific to ifchange:
- `-n` dry run: report but don't update state (rev1 just reports, stamp is
  only written by dk-stamp anyway, so -n here means "don't exit 1, just report")
- `-v` verbose: print which files changed
- `-q` quiet: suppress "up to date" message

## TDD Plan

### RED

```go
func TestIfchangeFirstRun(t *testing.T) {
    // No stamp exists → exit 0
}

func TestIfchangeUnchanged(t *testing.T) {
    // Stamp matches current files → exit 1
}

func TestIfchangeFileModified(t *testing.T) {
    // File content changed since stamp → exit 0
}

func TestIfchangeFileAdded(t *testing.T) {
    // New file in arg list not in stamp → exit 0
}

func TestIfchangeFileRemoved(t *testing.T) {
    // File in stamp no longer in arg list → exit 0
}

func TestIfchangeVerbose(t *testing.T) {
    // -v flag prints changed file names
}

func TestIfchangeDryRun(t *testing.T) {
    // -n flag reports without side effects
}
```

### GREEN

1. Implement `runIfchange()` function in `cmd/dk-redo/`
2. Wire flag parsing for `-v`, `-q`, `-n`
3. Call resolve → hash → read stamp → compare pipeline
4. Set exit code based on comparison result
5. Implement verbose/quiet output formatting

### REFACTOR

- Ensure error messages include the label for context
- Consider structured output for `-v` (file path + reason for change)
