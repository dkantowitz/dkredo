---
id: 003
title: Implement stamp file I/O with atomic writes, label escaping, and directory search
status: To Do
priority: 1
effort: Medium
assignee: claude
created_date: 2026-03-27
labels: [feature, core]
swimlane: Core
dependencies: [002]
---

## Summary

Implement the `internal/stamp/` package's file I/O: reading and writing stamp
files, atomic writes (temp+rename), label-to-filename escaping, path encoding
within stamps, and the upward `.stamps/` directory search algorithm. This is a
critical foundation — every operation reads or writes stamps through this package.

## Current State

After ticket 002, `internal/stamp/state.go` will have the `StampState` struct.
This ticket adds persistence.

## Analysis & Recommendations

### Label escaping (label → filename)

Per spec: `/ → %2F`, `% → %25`. All other characters verbatim.

```go
func EscapeLabel(label string) string
func UnescapeLabel(filename string) string
```

### Path encoding (path → stamp line)

Per spec: `\t → %09`, `\n → %0A`, `% → %25`. Spaces stored verbatim.

```go
func EncodePath(path string) string
func DecodePath(encoded string) string
```

### Stamp file format

Each line: `<encoded-path>\t<facts>`. Lines sorted by path. Facts are
space-separated `key:value` pairs. A file with no facts has path + tab + empty string.

### Atomic write

Write to `<path>.tmp.<pid>`, then `os.Rename()` into place.

### .stamps/ directory search

Walk upward from cwd. If not found, create in cwd on first write (not on read).

```go
func FindStampsDir() (string, error)      // search upward, return "" if not found
func StampsDir() (string, error)          // find or create
func StampPath(stampsDir, label string) string  // stampsDir + "/" + EscapeLabel(label)
```

### Read/Write

```go
func ReadStamp(stampsDir, label string) (*StampState, error)   // returns empty state if no file
func WriteStamp(stampsDir string, state *StampState) error     // atomic write
```

## TDD Plan

### RED

```go
// internal/stamp/escape_test.go
func TestEscapeLabel(t *testing.T) {
    tests := []struct{ in, want string }{
        {"firmware.bin", "firmware.bin"},
        {"output/config.json", "output%2Fconfig.json"},
        {"100%done", "100%25done"},
        {"output/100%/file", "output%2F100%25%2Ffile"},
    }
    for _, tt := range tests {
        got := EscapeLabel(tt.in)
        if got != tt.want { t.Errorf("EscapeLabel(%q) = %q, want %q", tt.in, got, tt.want) }
        back := UnescapeLabel(got)
        if back != tt.in { t.Errorf("roundtrip failed: %q -> %q -> %q", tt.in, got, back) }
    }
}

func TestEncodePath(t *testing.T) {
    tests := []struct{ in, want string }{
        {"src/main.c", "src/main.c"},          // slashes NOT encoded in paths
        {"my file.c", "my file.c"},            // spaces verbatim
        {"dir\tname/file", "dir%09name/file"}, // tab encoded
        {"a\nb", "a%0Ab"},                     // newline encoded
        {"100%/file", "100%25/file"},           // percent encoded
    }
    // ... similar pattern with roundtrip
}

// internal/stamp/io_test.go
func TestWriteReadRoundtrip(t *testing.T) {
    // Create state with entries + facts, write, read back, compare
}

func TestWriteCreatesStampsDir(t *testing.T) {
    // Write to nonexistent .stamps/, verify dir auto-created
}

func TestReadMissingStamp(t *testing.T) {
    // Read nonexistent label → empty state, no error
}

func TestAtomicWrite(t *testing.T) {
    // Verify no .tmp file remains after write
}

func TestLabelWithSlash(t *testing.T) {
    // label "output/config.json" → file .stamps/output%2Fconfig.json
}

func TestPathWithTab(t *testing.T) {
    // entry with tab in path roundtrips correctly
}

func TestPathWithPercent(t *testing.T) {
    // "100%/file" roundtrips through encode/decode
}

func TestPathWithSpaces(t *testing.T) {
    // "my file.c" stored verbatim, roundtrips
}

// internal/stamp/find_test.go
func TestFindStampsDirInCwd(t *testing.T) {
    // .stamps/ exists in cwd → returns it
}

func TestFindStampsDirInParent(t *testing.T) {
    // .stamps/ in parent, cd to child → finds parent's
}

func TestFindStampsDirInGrandparent(t *testing.T) {
    // .stamps/ two levels up → found
}

func TestFindStampsDirNotFound(t *testing.T) {
    // fresh temp dir → returns ""
}

func TestNestedProject(t *testing.T) {
    // parent has .stamps/, child has own .stamps/ → child's wins
}

func TestPathsAreProjectRelative(t *testing.T) {
    // from subdir, stamp src/main.c → entry relative to .stamps/ parent
}
```

### GREEN

1. Implement `EscapeLabel` / `UnescapeLabel` in `internal/stamp/escape.go`
2. Implement `EncodePath` / `DecodePath` in `internal/stamp/encode.go`
3. Implement `FindStampsDir` in `internal/stamp/find.go` — upward walk from cwd
4. Implement `ReadStamp` — read file, parse lines, populate `StampState`
5. Implement `WriteStamp` — serialize entries, atomic temp+rename write
6. Implement `StampPath` — join stamps dir + escaped label

### REFACTOR

1. Ensure consistent error wrapping with `fmt.Errorf("stamp: %w", err)`.
2. Verify file permissions (0644 for stamp files, 0755 for .stamps/ dir).
3. Run tests with `-race` flag.

### CLI Integration Test

```bash
# After CLI exists, verify:
# 1. dkredo creates .stamps/ on first stamp write
# 2. Label with "/" creates correctly-escaped filename
# 3. Stamp content matches expected format (tab-separated, sorted)
```
