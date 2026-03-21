---
id: "003"
title: Implement hasher package for BLAKE3 file/dir hashing
status: Done
completed_date: 2026-03-21
priority: 2
effort: Medium
assignee: claude
created_date: 2026-03-21
labels: [feature, core]
swimlane: Core Library
phase: 2
depends_on: ["001"]
source_file: dk-redo-implementation.md:128
---

## Summary

Implement `internal/hasher/` — the package responsible for computing BLAKE3
hashes of files and directories. This is the core change detection primitive
that stamp and ifchange depend on.

**Note:** `ReadStdin` (reading file paths from stdin) is NOT in this package.
It is a path-parsing function that belongs in the `resolve` package (ticket
006). The hasher package only deals with computing hashes of file content.

## Current State

Package exists as a placeholder from ticket 001. No implementation.

## Analysis & Recommendations

The package needs two primary functions per `dk-redo-implementation.md:128-141`:

```go
// HashFile returns per-file facts for a single file path.
// Symlinks are followed — the hash reflects the target content.
// Returns "blake3:<hex> size:<n>" for existing files, "missing:true" for absent files.
func HashFile(path string) (Facts, error)

// HashDir walks a directory recursively, hashing all files.
// Symlinks are followed. Circular symlink loops are detected and reported as errors.
// Returns a sorted list of (path, Facts) pairs.
func HashDir(dirPath string) ([]FileFacts, error)
```

Key types:

```go
type Facts struct {
    Blake3  string // hex digest, empty if missing
    Size    int64  // byte count, -1 if missing
    Missing bool   // true if file did not exist
}

type FileFacts struct {
    Path  string
    Facts Facts
}
```

BLAKE3 usage: `github.com/zeebo/blake3` — hash raw file bytes, produce
256-bit (64 hex char) digest.

Directory hashing: walk recursively with `filepath.WalkDir`, collect files
only (skip dirs), **follow symlinks**, sort lexically by path, hash each file.
Detect symlink loops by tracking visited inodes or real paths.

**Symlinks are followed.** Both `HashFile` and `HashDir` follow symbolic links
to hash the target content. This matches what the build would see — the hash
should reflect the actual data, not the link itself.

**Size is always recorded alongside the hash.** The caller (stamp comparison)
uses size for fast-path rejection: if size differs, skip the expensive hash
comparison. The hasher computes both in a single pass (stat for size, read
for hash).

## TDD Plan

### RED

```go
func TestHashFile(t *testing.T) {
    tests := []struct {
        name        string
        setup       func(t *testing.T, dir string) string // returns path
        wantErr     bool
        wantMissing bool
        wantSize    int64
    }{
        {"with content", createFile("hello"), false, false, 5},
        {"empty file", createFile(""), false, false, 0},
        {"missing file", returnPath("nonexistent"), false, true, -1},
        {"permission denied", createUnreadable(), true, false, 0},
        {"follows symlink", createSymlink("hello"), false, false, 5},
    }
    // ...
}

func TestHashDir(t *testing.T) {
    tests := []struct {
        name    string
        // ...
    }{
        {"empty dir"},
        {"with files - sorted list, hashes change on modification"},
        {"determinism - same files, different creation order, same result"},
        {"follows symlinks - symlinked file hashed by target content"},
        {"symlink loop - error, not infinite loop"},
    }
    // ...
}

func TestHashFileSizeBeforeHash(t *testing.T) {
    // Verify that Facts always contains size alongside blake3.
    // This enables the caller to do size-first comparison.
}
```

### GREEN

1. Implement `Facts` and `FileFacts` types
2. Implement `HashFile` using `zeebo/blake3`, following symlinks
3. Implement `HashDir` using `filepath.WalkDir`, following symlinks,
   detecting circular loops
4. Verify all tests pass

### REFACTOR

- Extract BLAKE3 digest computation into a small helper if used in multiple places
- Ensure consistent error wrapping with `fmt.Errorf("hasher: %w", err)`

## Completion Notes

**Commit:** `085cf78`

### Files modified
- `internal/hasher/hasher.go` (136 lines) — `HashFile`, `HashDir`, `walkDir`
- `internal/hasher/hasher_test.go` (309 lines) — 11 unit tests

### Test inventory (11 tests)
- `TestHashFileWithContent`, `TestHashFileEmpty`, `TestHashFileMissing`
- `TestHashFilePermissionDenied`, `TestHashFileFollowsSymlink`
- `TestHashDirEmpty`, `TestHashDirWithFiles`, `TestHashDirDeterminism`
- `TestHashDirFollowsSymlinks`, `TestHashDirSymlinkLoop`
- `TestFactsSizeAlongsideBlake3`

### Coverage
- **80.4%** statement coverage
- `HashFile`: 80.0%, `HashDir`: 76.9%, `walkDir`: 82.6%

### Design decisions
- Symlinks are followed in both `HashFile` and `HashDir`
- Symlink loop detection via tracking visited real paths
- `Facts.Missing` uses `Size: -1` sentinel
- Size is always recorded alongside hash for fast-path comparison

### Deferred work
- None. Coverage at 80.4% meets the 80% threshold (ticket 017). The uncovered paths are OS-level error conditions (e.g., file disappearing between stat and read) that are difficult to trigger reliably in tests.
