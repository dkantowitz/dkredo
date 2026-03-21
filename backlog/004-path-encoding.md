---
id: "004"
title: Implement label escaping and path percent-encoding
status: To Do
priority: 2
effort: Trivial
assignee: claude
created_date: 2026-03-21
labels: [feature, core]
swimlane: Core Library
phase: 2
depends_on: ["001", "002"]
source_file: dk-redo-implementation.md:106
---

## Summary

Implement the percent-encoding functions for label-to-filename escaping and
path-in-stamp-file encoding. These are small, pure functions but critical for
correctness — stamp files must roundtrip paths with special characters.

## Current State

Functions specified in `dk-redo-implementation.md:106-121`. No implementation exists.

## Analysis & Recommendations

Two encoding contexts with different character sets:

**Label escaping** (label → stamp filename):
```go
func EscapeLabel(label string) string {
    // %25 first (escape char), then %2F (slash)
    label = strings.ReplaceAll(label, "%", "%25")
    label = strings.ReplaceAll(label, "/", "%2F")
    return label
}

func UnescapeLabel(escaped string) string {
    escaped = strings.ReplaceAll(escaped, "%2F", "/")
    escaped = strings.ReplaceAll(escaped, "%25", "%")
    return escaped
}
```

**Path encoding** (path → stamp line):
```go
func EncodePath(path string) string {
    // %25 first, then %09 (tab), then %0A (newline)
    path = strings.ReplaceAll(path, "%", "%25")
    path = strings.ReplaceAll(path, "\t", "%09")
    path = strings.ReplaceAll(path, "\n", "%0A")
    return path
}

func DecodePath(encoded string) string {
    encoded = strings.ReplaceAll(encoded, "%09", "\t")
    encoded = strings.ReplaceAll(encoded, "%0A", "\n")
    encoded = strings.ReplaceAll(encoded, "%25", "%")
    return encoded
}
```

These can live in `internal/stamp/encoding.go` since they're only used by
the stamp package.

## TDD Plan

### RED

```go
func TestEscapeLabel(t *testing.T) {
    tests := []struct {
        input, want string
    }{
        {"firmware.bin", "firmware.bin"},
        {"output/config.json", "output%2Fconfig.json"},
        {"100%done", "100%25done"},
        {"a/b%c/d", "a%2Fb%25c%2Fd"},
    }
    // ...
}

func TestEncodePath(t *testing.T) {
    tests := []struct {
        input, want string
    }{
        {"src/main.c", "src/main.c"},           // slashes are fine in paths
        {"dir\tname/file", "dir%09name/file"},   // tab encoded
        {"100%/file", "100%25/file"},            // percent encoded
    }
    // ...
}
```

### GREEN

1. Implement `EscapeLabel` / `UnescapeLabel` in `internal/stamp/encoding.go`
2. Implement `EncodePath` / `DecodePath` in the same file
3. Verify roundtrip: `Unescape(Escape(x)) == x` for all test cases

### REFACTOR

- Property-based roundtrip test if `testing/quick` is worth it here
