---
id: "014"
title: Implement dk-affects command (reverse dependency query)
status: To Do
priority: 5
effort: Small
assignee: claude
created_date: 2026-03-21
labels: [enhancement, core]
swimlane: Core Library
phase: 5
depends_on: ["008"]
source_file: dk-redo.md:322
---

## Summary

Implement the `dk-affects` diagnostic command — answers "if I change this
file, which labels need rebuilding?" by scanning all stamps for reverse
dependencies.

## Current State

Core commands implemented. Stamp reading exists in `internal/stamp/`.

## Analysis & Recommendations

Per `dk-redo.md:322-342`:

```
dk-affects <file> [file...]
```

Algorithm:
1. Read all stamp files from `.stamps/`
2. For each stamp, check if any queried file appears in its input list
3. Print labels that depend on the queried file(s)

Exit codes:
- `0` — found affected labels
- `1` — no labels depend on the given files

Flags: `-v` (show which input triggered each label), `--json`.

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
```

### GREEN

1. Implement `runAffects()` function
2. Implement stamp directory scan + file list extraction
3. Match queried files against each stamp's input list
4. Implement output formatting (plain, verbose, JSON)

### REFACTOR

- Consider indexing (reversed map: file → labels) if stamp count grows large
