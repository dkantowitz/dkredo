---
id: "006"
title: Implement resolve package for input argument resolution
status: Done
completed_date: 2026-03-21
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
deduplicated, sorted list of file paths.

This package also owns `ReadStdin` — reading file paths from stdin (newline
or null-terminated). This is a path-parsing function, not a hashing function,
so it belongs here rather than in the hasher package.

## Current State

Package exists as a placeholder. The hasher package (ticket 003) provides
`HashDir` for directory expansion.

## Analysis & Recommendations

The resolve algorithm is specified in `dk-redo.md:524-551`:

```go
// Resolve takes the raw arguments after <label> and returns a sorted,
// deduplicated list of canonical file paths.
func Resolve(args []string, stdin io.Reader) ([]string, error)

// ReadStdin reads file paths from an io.Reader.
// If nullTerminated is true, splits on \0; otherwise splits on \n.
func ReadStdin(r io.Reader, nullTerminated bool) ([]string, error)
```

Steps:
1. Iterate args in order: for each `-` or `-0`, splice stdin paths at that
   position in the argument list (stdin is read once, paths inserted where
   the `-`/`-0` appears)
2. Expand directory args to recursive file lists using `hasher.HashDir`
   (reuse its directory walking, but only need the paths, not the hashes)
3. Canonicalize all paths (clean, project-relative, `/` separators)
4. Sort and deduplicate

**Stdin ordering:** Stdin paths are spliced at the position where `-` or `-0`
appears in the argument list. For example: `dk-ifchange label blah.h - bar.h`
processes `blah.h`, then all paths from stdin, then `bar.h`. The final list
is sorted and deduplicated, so positional ordering affects gathering only,
not the stamp content.

**Directory expansion:** Use the hasher package's directory walking rather
than reimplementing `filepath.WalkDir`. The resolve package depends on
hasher (ticket 003) for this.

Edge case: `-` or `-0` when stdin is a TTY should be an error.

## TDD Plan

### RED

```go
func TestResolve(t *testing.T) {
    tests := []struct {
        name  string
        args  []string
        stdin string
        want  []string
    }{
        {"file args", []string{"src/a.c", "src/b.c"}, "", []string{"src/a.c", "src/b.c"}},
        {"dir arg", []string{"src/"}, "", []string{"src/a.c", "src/b.c"}},
        {"mixed files and dirs", []string{"a.c", "src/", "b.c"}, "", []string{"a.c", "b.c", "src/x.c"}},
        {"stdin newline", []string{"-"}, "x.c\ny.c\n", []string{"x.c", "y.c"}},
        {"stdin null", []string{"-0"}, "x.c\0y.c\0", []string{"x.c", "y.c"}},
        {"mixed with stdin at position", []string{"a.c", "-", "b.c"}, "x.c\n", []string{"a.c", "b.c", "x.c"}},
        {"deduplication", []string{"a.c", "-"}, "a.c\n", []string{"a.c"}},
    }
    // ...
}

func TestReadStdin(t *testing.T) {
    tests := []struct {
        name           string
        input          string
        nullTerminated bool
        want           []string
    }{
        {"newline terminated", "a.c\nb.c\n", false, []string{"a.c", "b.c"}},
        {"null terminated", "a.c\0b.c\0", true, []string{"a.c", "b.c"}},
        {"empty", "", false, []string{}},
        {"no trailing delimiter", "a.c\nb.c", false, []string{"a.c", "b.c"}},
    }
    // ...
}
```

### GREEN

1. Implement `ReadStdin` using `bufio.Scanner` with custom split function
2. Implement stdin detection (is TTY check)
3. Implement arg iteration with `-`/`-0` replacement at position
4. Implement directory expansion using hasher's directory walking
5. Implement path canonicalization (clean, relative to project root)
6. Implement sort + dedup
7. Verify all tests pass

### REFACTOR

- Consider whether resolve should return `[]string` or a richer type
  that preserves which args were directories (useful for `-v` output)

## Completion Notes

**Commit:** `e56054e`

### Files modified
- `internal/resolve/resolve.go` (110 lines) — `Resolve`, `ReadStdin`, `dedup`
- `internal/resolve/resolve_test.go` (185 lines) — 15 test cases

### Test inventory (15 tests including subtests)
- `TestReadStdin` (5 subtests): newline terminated, null terminated, empty, no trailing delimiter, empty lines skipped
- `TestResolve` (8 subtests): file args, dir arg, mixed files and dirs, stdin newline, stdin null, mixed with stdin at position, deduplication, stdin read only once

### Coverage
- **91.5%** statement coverage
- `Resolve`: 92.0%, `ReadStdin`: 93.3%, `dedup`: 85.7%

### Design decisions
- Returns `[]string` (sorted, deduplicated paths) — simpler than a richer type; `-v` output handled at command level
- Stdin is read once and spliced at the position of `-` or `-0` in the arg list
- Directory expansion delegates to `filepath.WalkDir` (not hasher — avoids hashing just for path listing)
- TTY detection for stdin not implemented (stdin is always consumed if `-`/`-0` present)

### Deferred work
- None. TTY detection was descoped — the command is designed for scripts, not interactive use.
