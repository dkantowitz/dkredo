---
id: "005"
title: Implement stamp package for read/write/compare/append
status: Done
completed_date: 2026-03-21
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

Package exists as a placeholder. Path encoding functions exist from ticket 004.
Hasher package exists from ticket 003.

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

// CompareResult holds structured comparison information for -v support.
type CompareResult struct {
    Changed      bool
    ChangedFiles []ChangedFile  // which files changed and why
    Warnings     []string       // e.g., unknown fact keys encountered
}

type ChangedFile struct {
    Path   string
    Reason string  // "modified", "added", "removed", "appeared", "disappeared", "unknown_facts"
}

// Read loads a stamp from disk. Returns (nil, nil) if stamp doesn't exist.
func Read(stampsDir, label string) (*Stamp, error)

// Write atomically writes a stamp to disk. Creates stampsDir if needed.
func Write(stampsDir string, s *Stamp) error

// Compare checks if current file facts match the stamp.
// Returns a structured CompareResult (not just bool) to support -v output
// and dk-ood's re-check workflow.
func Compare(s *Stamp, currentFacts []hasher.FileFacts) CompareResult

// Append merges new facts into an existing stamp.
func Append(existing *Stamp, newFacts []hasher.FileFacts) *Stamp
```

**`Compare` returns `CompareResult`, not `bool`.** This is intentional:
- `dk-ifchange -v` needs to report which files changed and why
- `dk-ood` needs the same structured info for its output
- The `Warnings` field reports unknown fact keys encountered

**Unknown fact keys:** If a stamp contains fact keys not recognized by the
current version (e.g., `mtime:123`), `Compare` treats the file as changed
and adds a warning. The reasoning: we cannot verify facts we don't
understand, so we must assume the file may have changed. When the label is
rebuilt, `dk-stamp` writes a fresh stamp with only known facts, clearing
the unknown keys.

**Adversarial input handling:** Stamp files are external input. `Read` must
handle corrupt, malformed, or adversarial stamp files gracefully:
- Binary data, very long lines, missing tab delimiters → treat stamp as
  invalid, return as "changed" (out of date) so the label gets rebuilt
- Do not panic on malformed input

Atomic write per `dk-redo-implementation.md:88-96`:
write to `<path>.tmp.<pid>`, then `os.Rename`.

Compare algorithm per `dk-redo.md:574-594`:
1. Different file lists → changed
2. For each file: check for unknown facts (→ changed with warning),
   then missing, then size (fast path), then blake3

**Size is checked before hash.** This is a fast path: `stat()` is far cheaper
than reading + hashing. If size differs, the file is changed without computing
the hash.

## TDD Plan

### RED

```go
func TestWriteThenRead(t *testing.T)       // roundtrip
func TestWriteCreatesDir(t *testing.T)     // auto-create .stamps/
func TestWriteAtomic(t *testing.T)         // no partial stamp on error
func TestReadMissing(t *testing.T)         // returns nil, nil
func TestReadCorrupt(t *testing.T)         // returns error or treats as changed
func TestReadAdversarial(t *testing.T)     // binary data, very long lines → graceful
func TestCompareUnchanged(t *testing.T)    // CompareResult.Changed == false
func TestCompareChangedHash(t *testing.T)  // Changed == true, ChangedFiles populated
func TestCompareChangedFileList(t *testing.T) // Changed == true (different files)
func TestCompareSizeFastPath(t *testing.T) // size differs → changed without hashing
func TestCompareMissingAppeared(t *testing.T) // missing:true but file exists
func TestCompareUnknownFacts(t *testing.T) // unknown key → Changed, Warnings populated
func TestAppendMerges(t *testing.T)        // new files added
func TestAppendUpdatesFacts(t *testing.T)  // existing file facts updated
func TestAppendPreserves(t *testing.T)     // unmentioned files kept
func TestTabDelimitedRoundtrip(t *testing.T)  // path with spaces handled correctly
func TestPathWithTab(t *testing.T)         // tab encoded as %09, roundtrips
func TestPathWithPercent(t *testing.T)     // percent encoded as %25, roundtrips
func TestLabelEscaping(t *testing.T)       // "output/config.json" → ".stamps/output%2Fconfig.json"
```

### GREEN

1. Implement `Read` — parse tab-delimited lines, decode paths, handle
   corrupt/adversarial input gracefully
2. Implement `Write` — encode paths, sort, atomic write
3. Implement `Compare` — file list diff, unknown fact detection, then
   per-file fact checks (size before hash)
4. Implement `Append` — merge file lists, update facts
5. Verify all tests pass

### REFACTOR

- Ensure error messages include the label for context
- Verify that `Compare` result provides enough info for both `-v` output
  and `dk-ood` usage

## Completion Notes

**Commit:** `12e7de7`

### Files modified
- `internal/stamp/stamp.go` (324 lines) — `Read`, `Write`, `Compare`, `Append`, `FormatFacts`, `parseFacts`
- `internal/stamp/stamp_test.go` (635 lines) — 30 top-level tests (49 including subtests)

### Test inventory (30 top-level tests, 49 with subtests)
- Read/Write: `TestWriteThenReadRoundtrip`, `TestWriteCreatesStampsDir`, `TestWriteIsAtomic`, `TestReadMissingStamp`, `TestReadCorruptStamp`, `TestReadAdversarialBinaryInput`, `TestReadAdversarialLongLine`, `TestReadEmptyStamp`
- Compare: `TestCompareUnchanged`, `TestCompareChangedHash`, `TestCompareChangedFileList`, `TestCompareSizeFastPath`, `TestCompareMissingAppeared`, `TestCompareFileDisappeared`, `TestCompareMissingStillMissing`, `TestCompareUnknownFacts`
- Append: `TestAppendMergesNewFiles`, `TestAppendUpdatesExistingFacts`, `TestAppendPreservesUnmentionedFiles`, `TestAppendWithMissingFile`, `TestAppendPreservesLabel`
- Paths: `TestRoundtripWithSpacesInPaths`, `TestPathWithTabEncoded`, `TestPathWithPercentEncoded`, `TestLabelWithSlash`
- Format: `TestFormatFacts` (2 subtests)

### Coverage
- **91.7%** statement coverage
- `Read`: 92.6%, `Write`: 72.4%, `Compare`: 100%, `Append`: 100%, `FormatFacts`: 100%, `parseFacts`: 85.7%

### Design decisions
- `Compare` returns structured `CompareResult` (not bool) — used by ifchange `-v` and dk-ood
- Atomic write via temp file + `os.Rename`
- Corrupt/adversarial stamps return error (exit 2), empty stamps treated as valid (no files)
- Unknown fact keys cause `Changed=true` with warning

### Deferred work
- `Write` coverage at 72.4% — the uncovered paths are error branches from `os.MkdirAll`, `os.CreateTemp`, and `os.Rename` that require simulating filesystem failures
