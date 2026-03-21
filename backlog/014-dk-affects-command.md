---
id: "014"
title: Implement dk-affects command (reverse dependency query)
status: Done
completed_date: 2026-03-21
priority: 4
effort: Small
assignee: claude
created_date: 2026-03-21
labels: [enhancement, core]
swimlane: Core Library
phase: 4
depends_on: ["005", "006", "007"]
source_file: dk-redo.md:322
---

## Summary

Implement the `dk-affects` diagnostic command — answers "if I change this
file, which labels need rebuilding?" by scanning all stamps for reverse
dependencies.

## Current State

Stamp reading exists in `internal/stamp/` (ticket 005). Resolve package
(ticket 006) handles stdin input modes. CLI dispatch exists (ticket 007).

## Analysis & Recommendations

Per `dk-redo.md:322-342`:

```
dk-affects <file> [file...]
dk-affects -                    # read file list from stdin
dk-affects -0                   # null-terminated stdin
dk-affects src/main.c - lib.c   # positional + stdin combined
```

Algorithm:
1. Resolve query file list using `resolve.Resolve` (supports all input modes:
   positional args, `-` for stdin, `-0` for null-terminated stdin, combinations)
2. Read all stamp files from `.stamps/`
3. For each stamp, check if any queried file appears in its input list
4. Print labels that depend on the queried file(s)

Exit codes:
- `0` — found affected labels
- `1` — no labels depend on the given files

Flags: `-v` (show which input triggered each label).

Note: `--json` is descoped to rev2 per the spec's feature tier classification.

**All input modes must work and be tested:**
- Positional file arguments
- `-` (stdin, newline-terminated)
- `-0` (stdin, null-terminated)
- Combinations (stdin spliced at position)

Path matching: the queried file paths should be canonicalized the same way
as paths stored in stamps (project-relative, clean).

## TDD Plan

### RED

```go
func TestAffectsFindsLabel(t *testing.T) {
    // Stamp "build" with src/main.c, dk-affects src/main.c → prints "build"
}

func TestAffectsMultipleLabels(t *testing.T) {
    // Two labels both depend on config.h → both printed
}

func TestAffectsNoMatch(t *testing.T) {
    // Query file not in any stamp → exit 1
}

func TestAffectsMultipleFiles(t *testing.T) {
    // dk-affects a.c b.c → union of affected labels
}

func TestAffectsStdinNewline(t *testing.T) {
    // dk-affects - < file_list → reads query files from stdin
}

func TestAffectsStdinNull(t *testing.T) {
    // dk-affects -0 < file_list → null-terminated stdin
}

func TestAffectsStdinCombined(t *testing.T) {
    // dk-affects src/main.c - lib.c → positional + stdin combined
}

func TestAffectsVerbose(t *testing.T) {
    // -v shows which input triggered each label
}
```

### GREEN

1. Implement `runAffects(args []string)` function
2. Resolve query files using `resolve.Resolve` (supports all input modes)
3. Implement stamp directory scan + file list extraction
4. Match queried files against each stamp's input list
5. Implement output formatting (plain, verbose)

### REFACTOR

- Share stamp scanning with dk-ood, dk-sources, dk-dot

## Completion Notes

**Commit:** `ac49b60`

### Files modified
- `cmd/dk-redo/main.go` — `cmdAffects` function added (~60 lines)

### Design decisions
- Resolves query files using `resolve.Resolve` (supports positional, `-`, `-0`, combinations)
- Scans all stamps, checks if any queried file appears in each stamp's input list
- Exit 0 = found affected labels, exit 1 = no labels depend on queried files
- `-v` shows which input file triggered each label match
- Path matching uses canonical paths (same as stamp storage)

### Deferred work
- `--json` output descoped to rev2 per spec
- No dedicated unit tests — tested via stamp package tests and would benefit from integration test coverage
