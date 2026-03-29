---
id: 011
title: Implement +names and +facts query operations
status: To Do
priority: 2
effort: Small
assignee: claude
created_date: 2026-03-27
labels: [feature, core]
swimlane: Core
dependencies: [003, 005]
---

## Summary

Implement the `+names` and `+facts` query operations that print stamp
contents to stdout. These are read-only operations used for scripting
(e.g., `$(dkredo label +names -e .c)` to feed file lists to compilers)
and diagnostics.

## Current State

After tickets 003 and 005, stamp I/O and filter resolution exist.

## Analysis & Recommendations

### +names

```go
// internal/ops/names.go
func Names(state *stamp.StampState, args []string, stampsParent string, stdout io.Writer, verbose bool) error
```

Behavior:
- Print file names from stamp to stdout, one per line
- Optional filter args: `.suffix` for extension, exact path
- `-e` flag (first arg): only print names that **exist on disk**
- Does not modify state, does not affect exit code
- Output goes to stdout (not stderr, even with -v)

### +facts

```go
// internal/ops/facts.go
func Facts(state *stamp.StampState, args []string, stampsParent string, stdout io.Writer, verbose bool) error
```

Behavior:
- Print `<path>\t<facts>` for each entry (same format as stamp file)
- Optional filter args
- Diagnostic/debugging output
- Entries with no facts: print `<path>\t` (path + tab, empty facts)

## TDD Plan

### RED

```go
// internal/ops/names_test.go
func TestNamesAll(t *testing.T) {
    state := stamp.NewStampState("test")
    state.AddEntry("a.c", "blake3:abc size:100")
    state.AddEntry("b.h", "blake3:def size:200")
    var buf bytes.Buffer
    err := Names(state, []string{}, tmpDir, &buf, false)
    assert(err == nil)
    assert(buf.String() == "a.c\nb.h\n")
}

func TestNamesFilterBySuffix(t *testing.T) {
    state := stamp.NewStampState("test")
    state.AddEntry("a.c", "facts")
    state.AddEntry("b.h", "facts")
    var buf bytes.Buffer
    Names(state, []string{".c"}, tmpDir, &buf, false)
    assert(buf.String() == "a.c\n")
}

func TestNamesExistsOnly(t *testing.T) {
    // Create a.c on disk, don't create gone.c
    state := stamp.NewStampState("test")
    state.AddEntry("a.c", "facts")
    state.AddEntry("gone.c", "missing:true")
    var buf bytes.Buffer
    Names(state, []string{"-e"}, tmpDir, &buf, false)
    assert(buf.String() == "a.c\n")  // gone.c excluded
}

func TestNamesExistsWithFilter(t *testing.T) {
    // -e .c → only .c files that exist
    state := stamp.NewStampState("test")
    state.AddEntry("a.c", "facts")    // exists
    state.AddEntry("b.h", "facts")    // exists
    state.AddEntry("gone.c", "facts") // doesn't exist
    var buf bytes.Buffer
    Names(state, []string{"-e", ".c"}, tmpDir, &buf, false)
    assert(buf.String() == "a.c\n")  // only existing .c files
}

func TestNamesEmptyStamp(t *testing.T) {
    state := stamp.NewStampState("test")
    var buf bytes.Buffer
    Names(state, []string{}, tmpDir, &buf, false)
    assert(buf.String() == "")
}

// internal/ops/facts_test.go
func TestFactsAll(t *testing.T) {
    state := stamp.NewStampState("test")
    state.AddEntry("a.c", "blake3:abc size:100")
    state.AddEntry("b.h", "blake3:def size:200")
    var buf bytes.Buffer
    Facts(state, []string{}, tmpDir, &buf, false)
    assert(strings.Contains(buf.String(), "a.c\tblake3:abc size:100\n"))
    assert(strings.Contains(buf.String(), "b.h\tblake3:def size:200\n"))
}

func TestFactsWithFilter(t *testing.T) {
    state := stamp.NewStampState("test")
    state.AddEntry("a.c", "blake3:abc size:100")
    state.AddEntry("b.h", "blake3:def size:200")
    var buf bytes.Buffer
    Facts(state, []string{".c"}, tmpDir, &buf, false)
    assert(buf.String() == "a.c\tblake3:abc size:100\n")
}

func TestFactsEmptyFacts(t *testing.T) {
    state := stamp.NewStampState("test")
    state.AddEntry("a.c", "")
    var buf bytes.Buffer
    Facts(state, []string{}, tmpDir, &buf, false)
    assert(buf.String() == "a.c\t\n")  // path + tab + empty
}
```

### GREEN

1. Create `internal/ops/names.go` — filter entries, check `-e` flag, print to stdout
2. Create `internal/ops/facts.go` — filter entries, print path+tab+facts
3. Both use filter matching from resolver package

### REFACTOR

1. Ensure output is newline-terminated (each line ends with `\n`).
2. Run with `-race`.

### CLI Integration Test

```bash
# Query names for use in compiler command
echo "data" > a.c && echo "data" > b.h
dkredo test +add-names a.c b.h +stamp-facts
dkredo test +names .c        # prints: a.c
dkredo test +names           # prints: a.c\nb.h

# -e flag filters non-existent
dkredo test2 +add-names a.c gone.c +stamp-facts
dkredo test2 +names -e       # prints: a.c (gone.c excluded)

# Facts for debugging
dkredo test +facts           # prints tab-delimited path+facts
```
