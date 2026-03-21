---
id: "015"
title: Implement dk-sources command (list all tracked files)
status: To Do
priority: 5
effort: Trivial
assignee: claude
created_date: 2026-03-21
labels: [enhancement, core]
swimlane: Core Library
phase: 5
depends_on: ["008"]
source_file: dk-redo.md:362
---

## Summary

Implement the `dk-sources` diagnostic command — lists the union of all input
files across all stamps. Answers: "what files does dk-redo know about?"

## Current State

Core commands implemented. Stamp reading exists.

## Analysis & Recommendations

Per `dk-redo.md:362-378`:

```
dk-sources
```

Algorithm:
1. Read all stamp files from `.stamps/`
2. Collect all input file paths across all stamps
3. Deduplicate and sort
4. Print one path per line

Exit codes:
- `0` — sources listed (or no stamps — prints nothing)
- `2` — error

Flag: `-v` shows which label tracks each file.

## TDD Plan

### RED

```go
func TestSourcesListsAll(t *testing.T) {
    // Two stamps with overlapping files → deduplicated union
}

func TestSourcesEmpty(t *testing.T) {
    // No stamps → empty output, exit 0
}

func TestSourcesVerbose(t *testing.T) {
    // -v shows "file.c (label1, label2)"
}
```

### GREEN

1. Implement `runSources()` function
2. Scan stamps, collect all paths into a set
3. Sort and print
4. Verbose mode: build file→labels map

### REFACTOR

- Share stamp scanning with dk-affects (both read all stamps)
