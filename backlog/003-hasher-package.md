---
id: "003"
title: Implement hasher package for BLAKE3 file/dir/stdin hashing
status: To Do
priority: 2
effort: Medium
assignee: claude
created_date: 2026-03-21
labels: [feature, core]
swimlane: Core Library
phase: 2
depends_on: ["001", "002"]
source_file: dk-redo-implementation.md:128
---

## Summary

Implement `internal/hasher/` — the package responsible for computing BLAKE3
hashes of files, directories, and stdin streams. This is the core change
detection primitive that stamp and ifchange depend on.

## Current State

Package exists as a placeholder from ticket 001. Test cases exist from ticket 002
(currently skipped). No implementation.

## Analysis & Recommendations

The package needs three primary functions per `dk-redo-implementation.md:128-141`:

```go
// HashFile returns per-file facts for a single file path.
// Returns "blake3:<hex> size:<n>" for existing files, "missing:true" for absent files.
func HashFile(path string) (Facts, error)

// HashDir walks a directory recursively, hashing all files.
// Returns a sorted list of (path, Facts) pairs.
func HashDir(dirPath string) ([]FileFacts, error)

// ReadStdin reads file paths from stdin (newline or null-terminated).
func ReadStdin(r io.Reader, nullTerminated bool) ([]string, error)
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
only (skip dirs/symlinks to dirs), sort lexically by path, hash each file.
Detect symlink loops by checking for `fs.ErrPermission` or stat errors.

Size fast path: `Facts` always includes both hash and size. The caller
(stamp comparison) uses size for fast-path rejection before comparing hashes.

## TDD Plan

### RED

Tests from ticket 002 in `internal/hasher/hasher_test.go`:
- `TestHashFile/with_content` — deterministic BLAKE3 + correct size
- `TestHashFile/empty_file` — BLAKE3 of empty + size:0
- `TestHashFile/missing_file` — `Facts{Missing: true}`
- `TestHashFile/permission_denied` — returns error
- `TestHashDir/empty_dir` — returns empty list
- `TestHashDir/with_files` — sorted list, hashes change on modification
- `TestHashDir/determinism` — same files in different creation order → same result
- `TestHashDir/symlink_loop` — error, not infinite loop
- `TestReadStdin/newline` — parses "a.c\nb.c\n" → ["a.c", "b.c"]
- `TestReadStdin/null` — parses "a.c\0b.c\0" → ["a.c", "b.c"]
- `TestReadStdin/empty` — returns empty list

### GREEN

1. Implement `Facts` and `FileFacts` types
2. Implement `HashFile` using `zeebo/blake3`
3. Implement `HashDir` using `filepath.WalkDir`
4. Implement `ReadStdin` using `bufio.Scanner` with custom split
5. Remove `t.Skip` from hasher tests, verify all pass

### REFACTOR

- Extract BLAKE3 digest computation into a small helper if used in multiple places
- Ensure consistent error wrapping with `fmt.Errorf("hasher: %w", err)`
