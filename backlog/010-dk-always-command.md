---
id: "010"
title: Implement dk-always command
status: To Do
priority: 3
effort: Trivial
assignee: claude
created_date: 2026-03-21
labels: [feature, core]
swimlane: Core Library
phase: 3
depends_on: ["007"]
source_file: dk-redo.md:269
---

## Summary

Implement the `dk-always` command — removes stamp files to force rebuilds.
This is the simplest command: delete `.stamps/<label>` for each argument.

## Current State

CLI dispatch exists. The `always` case is a stub. Stamp path logic
(label escaping) exists from ticket 004.

## Analysis & Recommendations

Per `dk-redo.md:269-286`:

```
dk-always <label> [label...]
dk-always --all
```

- Remove `.stamps/<escaped-label>` for each label argument
- `--all`: remove all files in `.stamps/`
- Always exits 0, even if stamp didn't exist
- `-v`: list removed stamps

This command doesn't need the hasher or resolve packages — it only needs
label escaping and file deletion.

## TDD Plan

### RED

```go
func TestAlwaysRemovesStamp(t *testing.T) {
    // Create stamp, dk-always <label>, verify stamp gone
}

func TestAlwaysNonexistentStamp(t *testing.T) {
    // dk-always on label with no stamp → exit 0 (no error)
}

func TestAlwaysMultipleLabels(t *testing.T) {
    // dk-always label1 label2 → both stamps removed
}

func TestAlwaysAll(t *testing.T) {
    // dk-always --all → all stamps removed
}

func TestAlwaysVerbose(t *testing.T) {
    // -v prints removed stamp paths
}
```

### GREEN

1. Implement `runAlways()` function
2. Wire `--all` and `-v` flags
3. For each label: compute stamp path, attempt `os.Remove`, ignore `ErrNotExist`
4. For `--all`: `os.ReadDir` on stamps dir, remove each file

### REFACTOR

- Minimal — this command is intentionally simple
