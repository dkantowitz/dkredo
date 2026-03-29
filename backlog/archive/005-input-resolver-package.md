---
id: 005
title: Implement input resolver for file args, stdin, file-input, depfile, and filters
status: Done
priority: 1
effort: Medium
assignee: claude
created_date: 2026-03-27
labels: [feature, core]
swimlane: Core
dependencies: [002]
---

## Summary

Implement the `internal/resolve/` package that resolves operation arguments
into concrete file paths. Handles positional file args, stdin (`-`/`-0`),
file-input (`-@`/`-@0`), makefile depfile parsing (`-M`), path canonicalization,
deduplication, and suffix/exact filter matching.

## Current State

No code exists. Per the implementation spec, input resolution is shared across
all operations that accept file or filter arguments.

## Analysis & Recommendations

Two distinct modes of argument resolution:

### 1. File resolution (for `+add-names` and similar)

Resolves `file...` arguments including all input modes:

```go
// ResolveFiles resolves raw args into canonical, deduplicated file paths.
// stampsParent is the .stamps/ parent dir for making paths relative.
func ResolveFiles(args []string, stdin io.Reader, stampsParent string) ([]string, error)
```

Input modes mixed freely in args:
- Plain paths: `src/main.c` — used as-is
- `-` — read newline-terminated paths from stdin
- `-0` — read null-terminated paths from stdin
- `-@ <file>` — read newline-terminated paths from named file
- `-@0 <file>` — read null-terminated paths from named file
- `-M <file.d>` — parse makefile dep format, extract dependency paths

After gathering: canonicalize (relative to stampsParent), sort, deduplicate.

### 2. Filter resolution (for `+remove-names`, `+check`, etc.)

Filters are a superset of files. Additionally:
- `.suffix` — matches all entries with that extension
- Plain path — exact match

```go
// MatchesFilter returns true if path matches the filter.
func MatchesFilter(path string, filter string) bool

// ResolveFilters resolves filter args. Filters can include all input modes
// (-, -0, -@, -@0, -M) plus suffix patterns (.c, .h).
func ResolveFilters(args []string, stdin io.Reader, stampsParent string) ([]string, error)
```

### 3. Makefile dep parsing

Parse gcc `-MD`/`-MMD` output format:

```
target.o: src/main.c src/util.h \
  include/config.h
```

Extract dependency paths (everything after the first `:`), handle `\` line
continuations, handle escaped spaces in paths.

```go
func ParseDepfile(path string) ([]string, error)
```

## TDD Plan

### RED

```go
// internal/resolve/resolve_test.go
func TestResolveFilesPositional(t *testing.T) {
    // args: ["src/a.c", "src/b.c"] → two paths
}

func TestResolveFilesStdinNewline(t *testing.T) {
    // args: ["-"], stdin: "x.c\ny.c\n" → two paths
}

func TestResolveFilesStdinNull(t *testing.T) {
    // args: ["-0"], stdin: "x.c\0y.c\0" → two paths
}

func TestResolveFilesFileInput(t *testing.T) {
    // write temp file with paths, args: ["-@", tempfile] → paths from file
}

func TestResolveFilesFileInputNull(t *testing.T) {
    // null-terminated temp file, args: ["-@0", tempfile] → paths
}

func TestResolveFilesDepfile(t *testing.T) {
    // args: ["-M", "out.d"] → paths extracted from depfile
}

func TestResolveFilesMixed(t *testing.T) {
    // args: ["a.c", "-M", "out.d", "-@", "list.txt"] → union of all
}

func TestResolveFilesSplicing(t *testing.T) {
    // args: ["a.c", "-", "b.c"], stdin: "x.c\n"
    // gathered order: a.c, x.c, b.c (but final is sorted+deduped)
}

func TestResolveFilesDedup(t *testing.T) {
    // same file from args and stdin → listed once
}

func TestResolveFilesCanonical(t *testing.T) {
    // "./src/main.c" and "src/main.c" → same entry
}

func TestResolveFilesStdinOnTty(t *testing.T) {
    // args: ["-"] but stdin has no data / is a tty → error
}

func TestResolveFilesEmptyArgs(t *testing.T) {
    // no args → empty list, no error
}

// internal/resolve/filter_test.go
func TestMatchesFilterExactPath(t *testing.T) {
    // "src/main.c" matches "src/main.c"
}

func TestMatchesFilterSuffix(t *testing.T) {
    // "src/main.c" matches ".c", does not match ".h"
}

func TestMatchesFilterSuffixMultiple(t *testing.T) {
    // stamp has a.c, b.h — filter ".c" matches only a.c
}

// internal/resolve/depfile_test.go
func TestParseDepfileSimple(t *testing.T) {
    // "out.o: src/main.c src/util.h" → ["src/main.c", "src/util.h"]
}

func TestParseDepfileMultiline(t *testing.T) {
    // "out.o: a.c \\\n  b.c c.c" → ["a.c", "b.c", "c.c"]
}

func TestParseDepfileMultipleTargets(t *testing.T) {
    // "out.o out.d: a.c b.c" → ["a.c", "b.c"] (targets ignored)
}

func TestParseDepfileEscapedSpaces(t *testing.T) {
    // paths with escaped spaces → correctly parsed
}

func TestParseDepfileEmpty(t *testing.T) {
    // empty file → no paths, no error
}

func TestParseDepfileMissing(t *testing.T) {
    // nonexistent file → error (exit 2)
}

func TestParseDepfileMalformed(t *testing.T) {
    // garbage content → error (exit 2)
}
```

### GREEN

1. Implement `ParseDepfile()` in `internal/resolve/depfile.go`
2. Implement `ResolveFiles()` in `internal/resolve/resolve.go`
   - Iterate args, handle `-`, `-0`, `-@`, `-@0`, `-M` modes
   - Canonicalize paths relative to stampsParent
   - Sort and deduplicate
3. Implement `MatchesFilter()` in `internal/resolve/filter.go`
4. Implement `ResolveFilters()` — same as ResolveFiles but filters can also be `.suffix` patterns

### REFACTOR

1. Extract stdin reading into helper to avoid duplication between `-` and `-0`.
2. Ensure error messages identify which input mode failed (e.g., "reading from -@ list.txt: ...").
3. Run with `-race`.

## Results

### Files Created
- `internal/resolve/resolve.go` — ResolveFiles, ResolveFilters, MatchesFilter, canonicalize, dedup, readLines, readNullTerminated
- `internal/resolve/resolve_test.go` — 16 tests for all input modes
- `internal/resolve/depfile.go` — ParseDepfile, depfile parsing with continuation lines and escaped spaces
- `internal/resolve/depfile_test.go` — 7 tests for depfile edge cases
- `internal/resolve/filter.go` — FilterEntries helper

### Coverage
`internal/resolve`: 60.1% of statements (ResolveFilters exercised indirectly via ops tests)

### Deviations
None. All input modes implemented as specified.
