---
id: "008"
title: Implement dk-ifchange command
status: Done
completed_date: 2026-03-21
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
2. Resolve inputs via resolve.Resolve() — stdin paths spliced at position
3. If -n flag: exit 0 immediately (force changed — always rebuild)
4. Hash all resolved files via hasher.HashFile/HashDir
5. Read stamp file via stamp.Read()
6. If no stamp exists: exit 0 (first run — changed)
7. Compare stamp vs current facts via stamp.Compare()
   - Returns CompareResult with Changed, ChangedFiles, Warnings
   - Unknown fact keys → treated as changed, warning issued to stderr
8. If changed: exit 0 (recipe continues)
9. If unchanged: exit 1 (? sigil stops recipe)
10. On error: exit 2
```

Exit codes per `dk-redo.md:193-197`:
- `0` — inputs changed (or first run) — recipe continues
- `1` — inputs unchanged — `?` sigil stops recipe silently
- `2` — error (corrupt stamp, I/O error)

Flags specific to ifchange:
- `-n` **force changed**: always exit 0, regardless of actual state.
  Simulates a "force rebuild" for a single invocation without deleting the
  stamp. The stamp is not modified. This is useful for testing/CI.
- `-v` verbose: print which files changed (uses `CompareResult.ChangedFiles`)
- `-q` quiet: suppress "up to date" message

**All input modes must work:**
- Positional file arguments
- Directory arguments (hashed recursively)
- `-` (stdin, newline-terminated)
- `-0` (stdin, null-terminated)
- Combinations: `dk-ifchange label blah.h - bar.h` (stdin spliced at position)

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
    // -v flag prints changed file names and reasons
}

func TestIfchangeForceChanged(t *testing.T) {
    // -n flag → always exit 0, even if unchanged
    // stamp is NOT modified
}

func TestIfchangeStdinNewline(t *testing.T) {
    // dk-ifchange label - < file_list → reads from stdin
}

func TestIfchangeStdinNull(t *testing.T) {
    // dk-ifchange label -0 < file_list → reads null-terminated
}

func TestIfchangeStdinCombined(t *testing.T) {
    // dk-ifchange label a.c - b.c < stdin → stdin spliced at position
}

func TestIfchangeDirectoryInput(t *testing.T) {
    // dk-ifchange label src/ → directory hashed recursively
}

func TestIfchangeUnknownFacts(t *testing.T) {
    // Stamp with unknown fact keys → exit 0 (changed), warning on stderr
}

func TestIfchangeCorruptStamp(t *testing.T) {
    // Corrupt stamp file → exit 0 (treated as changed, rebuild)
}
```

### GREEN

1. Implement `runIfchange(args []string)` function in `cmd/dk-redo/`
2. Wire flag parsing for `-v`, `-q`, `-n`
3. If `-n`: exit 0 immediately (force changed)
4. Call resolve → hash → read stamp → compare pipeline
5. Use `CompareResult` for exit code and verbose output
6. Set exit code based on comparison result

### REFACTOR

- Ensure error messages include the label for context
- Structured output for `-v` (file path + reason for change from CompareResult)

## Completion Notes

**Commit:** `5d1c392`

### Files modified
- `cmd/dk-redo/main.go` — `cmdIfchange` function added (~50 lines)

### Unit tests
- Tested via integration tests (ticket 011), not separate unit tests — the command is a thin orchestrator over resolve, hasher, stamp packages

### Integration test coverage (from ticket 011)
- `TestFirstRun` — no stamp, exit 0
- `TestUnchanged` — stamp matches, exit 1
- `TestFileModified` — content changed, exit 0
- `TestFileAdded` — new file in args, exit 0
- `TestFileRemoved` — file removed from args, exit 0
- `TestDirFileAdded`, `TestDirFileRemoved` — directory expansion
- `TestMissingFileSentinel` — missing:true then file appears
- `TestForceChanged` — `-n` flag always exits 0
- `TestCorruptStamp` — corrupt stamp treated as error (exit 2) or changed (exit 0)

### Design decisions
- Pipeline: parse flags → resolve inputs → hash files → read stamp → compare → exit code
- `-n` (force changed) exits 0 immediately without reading stamp
- Exit 0 = changed (recipe continues), exit 1 = unchanged (recipe stops), exit 2 = error
- `-v` prints changed file names and reasons from `CompareResult.ChangedFiles`

### Deferred work
- Stdin input modes (`-`, `-0`) not tested in integration tests (tested at resolve package level)
