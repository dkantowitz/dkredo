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
depends_on: ["004", "007"]
source_file: dk-redo.md:269
---

## Summary

Implement the `dk-always` command — removes stamp files to force rebuilds.
This is the simplest command: delete `.stamps/<label>` for each argument.

## Current State

CLI dispatch exists from ticket 007. Label escaping (ticket 004) provides
the stamp path computation needed to map labels to filenames.

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

This command needs label escaping from ticket 004 to compute stamp paths.
It does NOT need the hasher or resolve packages.

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

func TestAlwaysLabelWithSlash(t *testing.T) {
    // dk-always output/config.json → removes .stamps/output%2Fconfig.json
}
```

### GREEN

1. Implement `runAlways(args []string)` function
2. Wire `--all` and `-v` flags
3. For each label: compute stamp path via `EscapeLabel`, attempt `os.Remove`,
   ignore `ErrNotExist`
4. For `--all`: `os.ReadDir` on stamps dir, remove each file

### REFACTOR

- Minimal — this command is intentionally simple
