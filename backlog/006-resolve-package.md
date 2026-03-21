---
id: "006"
title: Implement resolve package for input argument resolution
status: To Do
priority: 2
effort: Small
assignee: claude
created_date: 2026-03-21
labels: [feature, core]
swimlane: Core Library
phase: 2
depends_on: ["003"]
source_file: dk-redo-implementation.md:168
---

## Summary

Implement `internal/resolve/` — the package that takes raw command-line
arguments (files, directories, `-`, `-0`) and resolves them into a canonical,
deduplicated, sorted list of file paths with their facts.

## Current State

Package exists as a placeholder. Test cases exist from ticket 002.
The hasher package (ticket 003) provides `HashDir` and `ReadStdin`.

## Analysis & Recommendations

The resolve algorithm is specified in `dk-redo.md:524-551`:

```go
// Resolve takes the raw arguments after <label> and returns a sorted,
// deduplicated list of canonical file paths.
func Resolve(args []string, stdin io.Reader) ([]string, error)
```

Steps:
1. Iterate args: replace `-` or `-0` with paths read from stdin
2. Expand directory args to recursive file lists (sorted)
3. Canonicalize all paths (project-relative, `/` separators)
4. Sort and deduplicate

This package orchestrates hasher's `ReadStdin` and `HashDir` for path
discovery, but does NOT hash files — that's the caller's job.

Edge case: `-` or `-0` when stdin is a TTY should be an error.

## TDD Plan

### RED

Tests from ticket 002 in `internal/resolve/resolve_test.go`:
- `TestResolve/file_args` — ["src/a.c", "src/b.c"] → two paths
- `TestResolve/dir_arg` — ["src/"] → all files under src/, sorted
- `TestResolve/mixed` — ["a.c", "src/", "b.c"] → merged and sorted
- `TestResolve/stdin_newline` — args=["-"], stdin="x.c\ny.c\n" → two paths
- `TestResolve/stdin_null` — args=["-0"], stdin="x.c\0y.c\0" → two paths
- `TestResolve/mixed_with_stdin` — ["a.c", "-", "b.c"] + stdin → merged
- `TestResolve/deduplication` — same file from args and stdin → listed once

### GREEN

1. Implement stdin detection (is TTY check)
2. Implement arg iteration with `-`/`-0` replacement
3. Implement directory expansion using `filepath.WalkDir`
4. Implement path canonicalization (clean, relative to project root)
5. Implement sort + dedup
6. Remove `t.Skip` from resolve tests, verify all pass

### REFACTOR

- Consider whether resolve should return `[]string` or a richer type
  that preserves which args were directories (useful for `-v` output)
