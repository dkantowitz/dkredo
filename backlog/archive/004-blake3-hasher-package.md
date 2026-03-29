---
id: 004
title: Implement BLAKE3 hasher package for file fact computation
status: Done
priority: 1
effort: Small
assignee: claude
created_date: 2026-03-27
labels: [feature, core]
swimlane: Core
dependencies: [002]
---

## Summary

Implement the `internal/hasher/` package that computes per-file facts using
BLAKE3 hashing. This package is used by `+stamp-facts` (to record facts) and
`+check` (to verify facts). It produces the `blake3:<hex> size:<bytes>` and
`missing:true` fact strings.

## Current State

No code exists. The Go BLAKE3 library (`github.com/zeebo/blake3`) will be
available after ticket 001.

## Analysis & Recommendations

Per spec:
- BLAKE3 over raw file bytes, 256-bit (64 hex chars)
- Facts format: `blake3:<hex> size:<bytes>` for existing files, `missing:true` for absent
- Symlinks: hash the target content, not the link itself (default `os.Open` behavior)
- Size from `stat()` before reading (used as fast-path check in `+check`)

```go
// FileFacts computes facts for a single file path.
// Returns "blake3:<hex> size:<n>" or "missing:true".
func FileFacts(path string) (string, error)

// ParseFacts parses a fact string into structured key:value pairs.
func ParseFacts(raw string) (map[string]string, error)

// CheckFact verifies a single file's facts against current filesystem state.
// Returns (changed bool, reason string, err error).
func CheckFact(path string, recordedFacts string) (bool, string, error)
```

## TDD Plan

### RED

```go
// internal/hasher/hasher_test.go
func TestFileFactsExistingFile(t *testing.T) {
    // Create temp file with known content "hello"
    // FileFacts → "blake3:<expected_hex> size:5"
    // Verify hex is 64 chars, size is correct
}

func TestFileFactsEmptyFile(t *testing.T) {
    // Empty temp file → "blake3:<empty_hash> size:0"
}

func TestFileFactsMissingFile(t *testing.T) {
    // Nonexistent path → "missing:true"
}

func TestFileFactsPermissionDenied(t *testing.T) {
    // Unreadable file → error (not missing:true)
}

func TestFileFactsFollowsSymlink(t *testing.T) {
    // Symlink to file → facts reflect target content
}

func TestFileFactsDeterministic(t *testing.T) {
    // Same file hashed twice → identical result
}

func TestParseFacts(t *testing.T) {
    facts, _ := ParseFacts("blake3:abcd1234 size:567")
    // facts["blake3"] == "abcd1234", facts["size"] == "567"
}

func TestParseFactsMissing(t *testing.T) {
    facts, _ := ParseFacts("missing:true")
    // facts["missing"] == "true"
}

func TestParseFactsUnknownKey(t *testing.T) {
    facts, _ := ParseFacts("blake3:abc size:5 future:xyz")
    // facts["future"] == "xyz" — preserved, caller decides
}

func TestCheckFactUnchanged(t *testing.T) {
    // Create file, compute facts, check → changed=false
}

func TestCheckFactContentChanged(t *testing.T) {
    // Create file, compute facts, modify content, check → changed=true
}

func TestCheckFactSizeChanged(t *testing.T) {
    // Create file, compute facts, change size, check → changed=true, reason mentions size
    // Verify hash is NOT computed (size fast path)
}

func TestCheckFactFileAppeared(t *testing.T) {
    // missing:true facts, then create file → changed=true
}

func TestCheckFactFileDisappeared(t *testing.T) {
    // blake3+size facts, then delete file → changed=true
}

func TestCheckFactUnknownKey(t *testing.T) {
    // facts with "future:xyz" → changed=true (cannot verify)
}
```

### GREEN

1. Implement `FileFacts()` — stat, read, BLAKE3 hash, format string
2. Implement `ParseFacts()` — split on space, split each on first `:`
3. Implement `CheckFact()` — parse recorded facts, compare against filesystem
   - Check `missing:true` first
   - Size fast path: if size differs, skip hash
   - Then hash comparison

### REFACTOR

1. Ensure large file handling works (don't read entire file into memory — use streaming BLAKE3).
2. Verify error messages include the file path for diagnostics.
3. Run with `-race`.

## Results

### Files Created
- `internal/hasher/hasher.go` — FileFacts, ParseFacts, CheckFact, KnownFactKeys
- `internal/hasher/hasher_test.go` — 17 tests covering all fact computation and verification paths

### Coverage
`internal/hasher`: 87.1% of statements

### Deviations
None. Streaming BLAKE3 via io.Copy as recommended in REFACTOR.
