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

Algorithm per `dk-redo-implementation.md`:

```
1. Parse flags (-v, -q, -n), extract label (arg[0]) and inputs (arg[1:])
   - inputs may be empty (label-only mode)
2. Resolve inputs via resolve.Resolve() — stdin paths spliced at position
3. If -n flag: exit 0 immediately (force changed — always rebuild)
4. Read stamp file via stamp.Read()
5. If no stamp exists: exit 0 (first run — changed)
   - This applies in label-only mode too: no stamp = out of date
6. Merge: check set = union(resolved inputs, stamp's recorded file list)
   - Label-only (no inputs): check set = stamp's file list only
   - New inputs not in stamp → changed (exit 0)
7. Hash all files in the check set via hasher.HashFile/HashDir
8. Compare stamp vs current facts via stamp.Compare()
   - Returns CompareResult with Changed, ChangedFiles, Warnings
   - Unknown fact keys → treated as changed, warning issued to stderr
   - Files in check set but not in stamp → treated as changed
9. If changed: exit 0 (recipe continues)
10. If unchanged: exit 1 (? sigil stops recipe)
11. On error: exit 2
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
- Label-only (no inputs) — deps from existing stamp only
- Positional file arguments
- Directory arguments (hashed recursively)
- `-` (stdin, newline-terminated)
- `-0` (stdin, null-terminated)
- Combinations: `dk-ifchange label blah.h - bar.h` (stdin spliced at position)
- Union behavior: args merged with existing stamp entries

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

func TestIfchangeLabelOnly(t *testing.T) {
    // dk-ifchange label (no inputs)
    // No stamp exists → exit 0 (first run)
}

func TestIfchangeLabelOnlyWithStamp(t *testing.T) {
    // dk-ifchange label (no inputs), stamp exists with files
    // Files unchanged → exit 1
    // File modified → exit 0
}

func TestIfchangeUnionWithStamp(t *testing.T) {
    // Stamp has [a.c, b.h], args have [a.c]
    // b.h modified → exit 0 (detected via stamp union)
}

func TestIfchangeNewInputNotInStamp(t *testing.T) {
    // Stamp has [a.c], args have [a.c, new.c]
    // → exit 0 (new.c is an addition)
}

func TestIfchangeStampEntryFileDeleted(t *testing.T) {
    // Stamp has [a.c, b.c], args have [a.c]
    // b.c deleted from disk → exit 0 (detected via stamp union)
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
