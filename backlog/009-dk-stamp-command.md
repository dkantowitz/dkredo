---
id: "009"
title: Implement dk-stamp command
status: To Do
priority: 3
effort: Small
assignee: claude
created_date: 2026-03-21
labels: [feature, core]
swimlane: Core Library
phase: 3
depends_on: ["005", "006", "007"]
source_file: dk-redo.md:207
---

## Summary

Implement the `dk-stamp` command — records current input state after a
successful build so the next `dk-ifchange` can detect changes.

## Current State

Internal packages implemented. CLI dispatch exists. The `stamp` case is a stub.

## Analysis & Recommendations

Algorithm per `dk-redo-implementation.md:76-84`:

```
1. Parse flags (--append, -v, -q), extract label and inputs
2. Resolve inputs via resolve.Resolve() — stdin paths spliced at position
3. Hash all resolved files via hasher.HashFile/HashDir
4. If --append: read existing stamp, merge via stamp.Append()
5. Write stamp atomically via stamp.Write()
6. Exit 0 on success, exit 2 on error
```

Exit codes per `dk-redo.md:243-244`:
- `0` — stamp written
- `2` — error

The `--append` flag per `dk-redo.md:222-240`:
- New files added to stamp
- Existing files have facts updated
- Files not in current call are preserved

Non-existent input files per `dk-redo.md:262-267`:
record `missing:true` (no hash or size). This enables the bootstrapping pattern.

**All input modes must work and be tested:**
- Positional file arguments
- Directory arguments (hashed recursively)
- `-` (stdin, newline-terminated)
- `-0` (stdin, null-terminated)
- Combinations: `dk-stamp label src/*.c - lib.c` (stdin spliced at position)

## TDD Plan

### RED

```go
func TestStampWritesFile(t *testing.T) {
    // Creates .stamps/<label> with correct content
}

func TestStampReplace(t *testing.T) {
    // Second dk-stamp replaces first stamp entirely
}

func TestStampAppend(t *testing.T) {
    // --append merges into existing stamp
}

func TestStampMissingInput(t *testing.T) {
    // Non-existent input file records missing:true
}

func TestStampCreatesStampsDir(t *testing.T) {
    // .stamps/ auto-created if absent
}

func TestStampStdinNewline(t *testing.T) {
    // dk-stamp label - < file_list → reads from stdin
}

func TestStampStdinNull(t *testing.T) {
    // dk-stamp label -0 < file_list → reads null-terminated
}

func TestStampStdinCombined(t *testing.T) {
    // dk-stamp label a.c - b.c < stdin → stdin spliced at position
}

func TestStampDirectoryInput(t *testing.T) {
    // dk-stamp label src/ → directory hashed recursively
}

func TestStampMixedInputModes(t *testing.T) {
    // dk-stamp label file.c dir/ - < stdin → all modes combined
}

func TestStampVerbose(t *testing.T) {
    // -v prints stamp path and per-file facts
}
```

### GREEN

1. Implement `runStamp(args []string)` function in `cmd/dk-redo/`
2. Wire flag parsing for `--append`, `-v`, `-q`
3. Call resolve → hash → (optional read + append) → write pipeline
4. Implement verbose output showing stamp path and per-file facts

### REFACTOR

- Share flag definitions with ifchange where applicable
