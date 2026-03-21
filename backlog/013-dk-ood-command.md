---
id: "013"
title: Implement dk-ood command (out-of-date labels)
status: To Do
priority: 5
effort: Small
assignee: claude
created_date: 2026-03-21
labels: [enhancement, core]
swimlane: Core Library
phase: 5
depends_on: ["008"]
source_file: dk-redo.md:296
---

## Summary

Implement the `dk-ood` diagnostic command — lists labels whose inputs have
changed since their last stamp. This is the multi-target dry-run: "what
needs rebuilding?"

## Current State

Core commands (ifchange, stamp, always) are implemented and tested. The stamp
reading and comparison logic in `internal/stamp/` can be reused directly.

## Analysis & Recommendations

Per `dk-redo.md:296-320`:

```
dk-ood [labels...]       # check specific labels (default: all)
```

Algorithm:
1. If labels specified: check those. If none: scan `.stamps/` for all labels.
2. For each label: read stamp, resolve current inputs from stamp's file list,
   re-check facts (same logic as ifchange's compare).
3. Print labels that are out of date.

Exit codes:
- `0` — at least one label is out of date
- `1` — all labels are up to date
- `2` — error or no stamps exist

Flags: `-v` (per-file details), `-q` (just exit code), `--json` (JSON array).

Note: `dk-ood` does NOT re-resolve inputs from command args — it reads the
file list from the stamp itself and re-checks those files. It cannot detect
new files added to a glob (that requires re-running dk-ifchange with the
glob). It answers: "have any of the files I stamped last time changed?"

## TDD Plan

### RED

```go
func TestOodFindsStaleLabel(t *testing.T) {
    // Stamp label, modify file, dk-ood → prints label, exit 0
}

func TestOodAllUpToDate(t *testing.T) {
    // Stamp label, no changes, dk-ood → exit 1
}

func TestOodNoStamps(t *testing.T) {
    // Empty .stamps/, dk-ood → exit 2
}

func TestOodSpecificLabels(t *testing.T) {
    // dk-ood label1 label2 → checks only those
}

func TestOodJsonOutput(t *testing.T) {
    // --json → valid JSON array of stale labels
}
```

### GREEN

1. Implement `runOod()` function
2. Implement stamp directory scanning (list all labels)
3. Reuse `stamp.Read` + `stamp.Compare` per label
4. Implement output formatting (plain, verbose, JSON)
5. Wire into CLI dispatch

### REFACTOR

- Share comparison logic with ifchange (don't duplicate)
