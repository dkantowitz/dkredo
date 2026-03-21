---
id: "013"
title: Implement dk-ood command (out-of-date labels)
status: To Do
priority: 4
effort: Small
assignee: claude
created_date: 2026-03-21
labels: [enhancement, core]
swimlane: Core Library
phase: 4
depends_on: ["005", "007"]
source_file: dk-redo.md:296
---

## Summary

Implement the `dk-ood` diagnostic command — lists labels whose inputs have
changed since their last stamp. This is the multi-target dry-run: "what
needs rebuilding?"

## Current State

The stamp reading and comparison logic in `internal/stamp/` (ticket 005) and
CLI dispatch (ticket 007) are the actual dependencies. dk-ood does not need
dk-ifchange — it only needs to read stamps and re-check facts.

## Analysis & Recommendations

Per `dk-redo.md:296-320`:

```
dk-ood [labels...]       # check specific labels (default: all)
```

Algorithm:
1. If labels specified: check those. If none: scan `.stamps/` for all labels.
2. For each label: read stamp, extract file paths from stamp's file list,
   re-hash those files using `hasher.HashFile`, then call `stamp.Compare`
   with the current facts.
3. Print labels that are out of date.

**Re-checking workflow:** dk-ood does NOT re-resolve inputs from command
args — it reads the file list from the stamp itself and re-checks those
files against disk. It uses `stamp.Compare` which returns a `CompareResult`
with structured change information. For each file in the stamp, dk-ood
must re-hash it (via `hasher.HashFile`) and provide the current facts to
`stamp.Compare`.

Exit codes:
- `0` — at least one label is out of date
- `1` — all labels are up to date
- `2` — error or no stamps exist

Flags: `-v` (per-file details from `CompareResult.ChangedFiles`), `-q` (just exit code).

Note: `--json` is descoped to rev2 per the spec's feature tier classification.

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

func TestOodVerbose(t *testing.T) {
    // -v → shows per-file change details from CompareResult
}

func TestOodUnknownFacts(t *testing.T) {
    // Stamp with unknown fact keys → label reported as out of date, warning
}
```

### GREEN

1. Implement `runOod(args []string)` function
2. Implement stamp directory scanning (list all labels via `UnescapeLabel`)
3. For each label: read stamp, re-hash files from stamp's file list,
   call `stamp.Compare` with current facts
4. Implement output formatting (plain, verbose)
5. Wire into CLI dispatch

### REFACTOR

- Share stamp scanning logic with dk-affects, dk-sources, dk-dot
  (all read all stamps)
