---
id: "005"
title: Implement stamp package for read/write/compare/append
status: To Do
priority: 2
effort: Medium
assignee: claude
created_date: 2026-03-21
labels: [feature, core]
swimlane: Core Library
phase: 2
depends_on: ["003", "004"]
source_file: dk-redo-implementation.md:86
---

## Summary

Implement `internal/stamp/` — the package responsible for reading, writing,
comparing, and appending stamp files. This is the persistence layer that
`dk-ifchange` and `dk-stamp` commands operate on.

## Current State

Package exists as a placeholder. Test cases exist from ticket 002. Path encoding
functions exist from ticket 004. Hasher package exists from ticket 003.

## Analysis & Recommendations

Stamp file format per `dk-redo.md:449-500`:
```
<percent-encoded-path>\tblake3:<hex> size:<bytes>
<percent-encoded-path>\tmissing:true
```

Lines sorted by path. Tab-delimited. One file per label in `.stamps/` directory.

Core types and functions:

```go
type Stamp struct {
    Label string
    Files []FileFact  // sorted by path
}

type FileFact struct {
    Path  string
    Facts string  // raw "blake3:... size:..." or "missing:true"
}

// Read loads a stamp from disk. Returns (nil, nil) if stamp doesn't exist.
func Read(stampsDir, label string) (*Stamp, error)

// Write atomically writes a stamp to disk. Creates stampsDir if needed.
func Write(stampsDir string, s *Stamp) error

// Compare checks if current file facts match the stamp.
// Returns true if all facts hold (unchanged), false if any fact differs.
func Compare(s *Stamp, currentFacts []hasher.FileFacts) bool

// Append merges new facts into an existing stamp.
func Append(existing *Stamp, newFacts []hasher.FileFacts) *Stamp
```

Atomic write per `dk-redo-implementation.md:88-96`:
write to `<path>.tmp.<pid>`, then `os.Rename`.

Compare algorithm per `dk-redo.md:574-594`:
1. Different file lists → changed
2. For each file: check missing, then size (fast path), then blake3

## TDD Plan

### RED

Tests from ticket 002 in `internal/stamp/stamp_test.go`:
- `TestWriteThenRead` — roundtrip
- `TestWriteCreatesDir` — auto-create `.stamps/`
- `TestWriteAtomic` — no partial stamp on error
- `TestReadMissing` — returns nil, nil
- `TestReadCorrupt` — returns error
- `TestCompareUnchanged` — exit 1 equivalent
- `TestCompareChangedHash` — exit 0 equivalent
- `TestCompareChangedFileList` — exit 0 equivalent
- `TestCompareSizeFastPath` — size differs, no hash computed
- `TestCompareMissingAppeared` — missing:true but file exists
- `TestAppendMerges` — new files added
- `TestAppendUpdatesFacts` — existing file facts updated
- `TestAppendPreserves` — unmentioned files kept

### GREEN

1. Implement `Read` — parse tab-delimited lines, decode paths
2. Implement `Write` — encode paths, sort, atomic write
3. Implement `Compare` — file list diff, then per-file fact checks
4. Implement `Append` — merge file lists, update facts
5. Remove `t.Skip` from stamp tests, verify all pass

### REFACTOR

- Ensure `Compare` returns structured info (which files changed) for `-v` flag support
