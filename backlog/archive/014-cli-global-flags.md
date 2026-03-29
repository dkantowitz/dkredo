---
id: 014
title: Implement global CLI flags (--version, --help, --stamps-dir, -v, DKREDO_ARGS)
status: Done
priority: 2
effort: Medium
assignee: claude
created_date: 2026-03-27
labels: [feature, core]
swimlane: Core
dependencies: [012]
---

## Summary

Implement the global CLI flags that appear before the label: `--version`,
`--help`/`-h`, `--stamps-dir`, `-v` (verbose), and the `DKREDO_ARGS`
environment variable. These affect early CLI processing before any operations
execute.

## Current State

After ticket 012, the CLI parser handles `label +op` syntax. This ticket
adds the global flag layer.

## Analysis & Recommendations

### Flag processing order

1. Read `DKREDO_ARGS` env var, shell-split, prepend to argv[1:]
2. Parse global flags from the front of the arg list
3. Remaining args go to label + operation parsing

### --version

`dkredo --version` prints version string and exits 0.
- Version embedded at build time via `-ldflags -X main.version=...`
- If no version embedded: print `dkredo dev`
- Works regardless of other args (early exit)

### --help / -h

- `--help` → full help text, exit 0
- `-h` → short help text, exit 0

### --stamps-dir `<path>`

- Overrides the upward `.stamps/` search
- Must appear before the label
- Directory created on first write if it doesn't exist
- Paths in stamps stored relative to `--stamps-dir` parent
- Works with all invocation styles (+ops, --cmd, argv[0])

### -v (verbose)

- Must appear before the label
- Threaded through executor to all operations
- Operations emit diagnostic messages to stderr
- Does not affect stdout output from +names/+facts

### DKREDO_ARGS

- If set, value is POSIX shell-split and inserted between argv[0] and argv[1]
- Respects single/double quotes and backslash escapes (for paths with spaces)
- Empty or unset → no effect
- Applies to all invocation styles

```go
// Effective args: [argv[0]] + split(DKREDO_ARGS) + argv[1:]
```

## TDD Plan

### RED

```go
// cmd/dkredo/flags_test.go
func TestVersionFlag(t *testing.T) {
    // --version → prints version, exits 0
}

func TestVersionDev(t *testing.T) {
    // No version embedded → prints "dkredo dev"
}

func TestHelpFlag(t *testing.T) {
    // --help → prints help, exit 0
}

func TestShortHelpFlag(t *testing.T) {
    // -h → prints short help, exit 0
}

func TestVerboseFlag(t *testing.T) {
    args := []string{"-v", "label", "+check"}
    cfg, _, _, _ := ParseFull(args)
    assert(cfg.Verbose == true)
}

func TestStampsDirFlag(t *testing.T) {
    args := []string{"--stamps-dir", "/tmp/s", "label", "+check"}
    cfg, _, _, _ := ParseFull(args)
    assert(cfg.StampsDir == "/tmp/s")
}

func TestStampsDirCreatedOnWrite(t *testing.T) {
    // --stamps-dir to nonexistent dir + +stamp-facts → dir created
}

func TestStampsDirNotCreatedOnRead(t *testing.T) {
    // --stamps-dir to nonexistent dir + +check → no dir created
}

func TestStampsDirPathsRelative(t *testing.T) {
    // --stamps-dir /tmp/s, stamp src/main.c
    // Entry should be relative to /tmp (parent of /tmp/s)
}

func TestStampsDirWithAliases(t *testing.T) {
    // dkr-ifchange --stamps-dir /tmp/s label files → uses /tmp/s
}

// DKREDO_ARGS tests
func TestDkredoArgsStampsDir(t *testing.T) {
    // DKREDO_ARGS="--stamps-dir /tmp/s"
    // dkredo label +stamp-facts → stamp written to /tmp/s/label
}

func TestDkredoArgsVerbose(t *testing.T) {
    // DKREDO_ARGS="-v" → verbose output
}

func TestDkredoArgsCombinedWithCli(t *testing.T) {
    // DKREDO_ARGS="-v", CLI: --stamps-dir /tmp/s
    // Both active
}

func TestDkredoArgsQuotedPath(t *testing.T) {
    // DKREDO_ARGS='--stamps-dir "/tmp/my stamps"'
    // Correctly parsed path with spaces
}

func TestDkredoArgsEmpty(t *testing.T) {
    // DKREDO_ARGS="" → no effect
}

func TestDkredoArgsUnset(t *testing.T) {
    // Not set → no effect
}

func TestDkredoArgsWithArgv0(t *testing.T) {
    // DKREDO_ARGS="--stamps-dir /tmp/s" + argv[0]=dkr-ifchange
    // Uses /tmp/s
}

func TestNoOperationsError(t *testing.T) {
    // dkredo label (no ops, no --cmd) → exit 2 with help
}
```

### GREEN

1. Implement POSIX shell-splitting for `DKREDO_ARGS` (handle quotes, escapes)
2. Add `DKREDO_ARGS` prepend logic in early main() processing
3. Add `--version` handler — check `main.version` var, print, exit 0
4. Add `--help`/`-h` handlers — print help text, exit 0
5. Add `--stamps-dir` flag parsing — store in config, thread to executor
6. Add `-v` flag parsing — store in config, thread to executor
7. Update build ldflags in Justfile for version embedding

### REFACTOR

1. Verify flag order doesn't matter (e.g., `-v --stamps-dir` and `--stamps-dir -v` both work).
2. Ensure help text lists all operations and aliases.
3. Run with `-race`.

### CLI Integration Test

```bash
# --version
dkredo --version           # prints version string
echo $?                    # 0

# --help
dkredo --help              # full help
dkredo -h                  # short help

# --stamps-dir
dkredo --stamps-dir /tmp/test-stamps test +add-names a.c +stamp-facts
ls /tmp/test-stamps/test   # stamp file exists

# -v verbose
dkredo -v test +add-names a.c +stamp-facts 2>&1 | grep "add-names"
# Verify verbose output on stderr

# DKREDO_ARGS
DKREDO_ARGS="--stamps-dir /tmp/env-stamps" dkredo test +add-names a.c +stamp-facts
ls /tmp/env-stamps/test    # stamp file exists

# DKREDO_ARGS with quotes
DKREDO_ARGS='--stamps-dir "/tmp/my stamps"' dkredo test +stamp-facts
ls "/tmp/my stamps/test"   # stamp file exists
```

## Results

### Files Created
- `cmd/dkredo/shellsplit.go` — POSIX shell splitting for DKREDO_ARGS
- `cmd/dkredo/shellsplit_test.go` — 6 tests for quoting and escaping

### Deviations
--version, --help, -h, --stamps-dir, and -v were partially implemented during ticket 012 (parser/executor). This ticket completed DKREDO_ARGS support and shellsplit. All flags verified working with both +operation and alias invocation styles.
