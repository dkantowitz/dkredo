---
id: 013
title: Implement alias system with --cmd and argv[0] dispatch
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

Implement the alias system that maps legacy command names (`dkr-ifchange`,
`dkr-stamp`, `dkr-always`, `dkr-fnames`) to operation sequences. Aliases
are invoked via symlinks (argv[0] dispatch) or `--cmd`. Both routes expand
through the generic CLI parser — no separate code path.

## Current State

After ticket 012, the CLI parser and executor handle `+operation` syntax.
This ticket adds the alias expansion layer on top.

## Analysis & Recommendations

### Alias table

Per spec:

| Alias | Expansion |
|-------|-----------|
| `ifchange <files>` | `+add-names <files> +check` |
| `ifchange` (no files) | `+check` |
| `stamp <files>` | `+remove-names +add-names <files> +stamp-facts` |
| `stamp --append <files>` | `+add-names <files> +stamp-facts` |
| `always` | `+clear-facts` |
| `fnames [filter]` | `+names -e [filter]` |

### argv[0] dispatch

When `os.Args[0]` basename is not `dkredo`:
1. Strip `dkr-` prefix → alias name
2. Look up alias template
3. Expand with remaining args (label + files)
4. Route through generic CLI parser

### --cmd dispatch

`--cmd` appears after the label:
```
dkredo <label> --cmd <alias> [args...]
```

1. Parser encounters `--cmd`
2. Next arg is alias name
3. Remaining args are the alias's arguments
4. Expand template, replace `--cmd` invocation with expanded ops
5. Route through generic CLI parser

### Constraints

- **Single --cmd per invocation.** Multiple `--cmd` → error (exit 2)
- **--cmd and +ops cannot mix.** `dkredo label --cmd ifchange +names` → error
- **Unknown alias → exit 2** with error listing valid aliases

### dkr-stamp --append

The `--append` flag is specific to the `stamp` alias. It changes the expansion
from `+remove-names +add-names <files> +stamp-facts` to `+add-names <files> +stamp-facts`.

### dkr-stamp -M

`-M` is passed through to the operation args:
- `dkr-stamp label -M file.d` → `+remove-names +add-names -M file.d +stamp-facts`
- `dkr-stamp --append label -M file.d` → `+add-names -M file.d +stamp-facts`

## TDD Plan

### RED

```go
// cmd/dkredo/alias_test.go
func TestExpandIfchangeWithFiles(t *testing.T) {
    ops := ExpandAlias("ifchange", []string{"a.c", "b.c"})
    assert(ops == ["+add-names", "a.c", "b.c", "+check"])
}

func TestExpandIfchangeNoFiles(t *testing.T) {
    ops := ExpandAlias("ifchange", []string{})
    assert(ops == ["+check"])
}

func TestExpandStamp(t *testing.T) {
    ops := ExpandAlias("stamp", []string{"a.c", "b.c"})
    assert(ops == ["+remove-names", "+add-names", "a.c", "b.c", "+stamp-facts"])
}

func TestExpandStampAppend(t *testing.T) {
    ops := ExpandAlias("stamp", []string{"--append", "a.c", "b.c"})
    assert(ops == ["+add-names", "a.c", "b.c", "+stamp-facts"])
}

func TestExpandStampWithDepfile(t *testing.T) {
    ops := ExpandAlias("stamp", []string{"-M", "out.d"})
    assert(ops == ["+remove-names", "+add-names", "-M", "out.d", "+stamp-facts"])
}

func TestExpandStampAppendWithDepfile(t *testing.T) {
    ops := ExpandAlias("stamp", []string{"--append", "-M", "out.d"})
    assert(ops == ["+add-names", "-M", "out.d", "+stamp-facts"])
}

func TestExpandAlways(t *testing.T) {
    ops := ExpandAlias("always", []string{})
    assert(ops == ["+clear-facts"])
}

func TestExpandFnames(t *testing.T) {
    ops := ExpandAlias("fnames", []string{".c"})
    assert(ops == ["+names", "-e", ".c"])
}

func TestExpandFnamesNoFilter(t *testing.T) {
    ops := ExpandAlias("fnames", []string{})
    assert(ops == ["+names", "-e"])
}

func TestExpandUnknownAlias(t *testing.T) {
    _, err := ExpandAlias("bogus", []string{})
    assert(err != nil)
}

// argv[0] dispatch tests
func TestArgv0Dispatch(t *testing.T) {
    // Simulate argv[0] = "dkr-ifchange"
    args := DetectAlias("dkr-ifchange", []string{"label", "a.c"})
    // Should produce same as: dkredo label +add-names a.c +check
}

func TestArgv0UnknownAlias(t *testing.T) {
    _, err := DetectAlias("dkr-bogus", []string{"label"})
    assert(err != nil)  // exit 2 with usage error
}

// --cmd dispatch tests
func TestCmdFlag(t *testing.T) {
    args := []string{"label", "--cmd", "ifchange", "a.c"}
    // Expands to: label +add-names a.c +check
}

func TestCmdInvalidAlias(t *testing.T) {
    args := []string{"label", "--cmd", "bogus"}
    _, _, err := Parse(args)
    assert(err != nil)  // exit 2
}

func TestCmdNoAliasName(t *testing.T) {
    args := []string{"label", "--cmd"}
    _, _, err := Parse(args)
    assert(err != nil)  // exit 2
}

func TestCmdMixedWithOps(t *testing.T) {
    args := []string{"label", "--cmd", "ifchange", "+names"}
    _, _, err := Parse(args)
    assert(err != nil)  // error: --cmd and +operations cannot be mixed
}

func TestMultipleCmd(t *testing.T) {
    args := []string{"label", "--cmd", "ifchange", "--cmd", "stamp"}
    _, _, err := Parse(args)
    assert(err != nil)  // error: multiple --cmd
}
```

### GREEN

1. Create `cmd/dkredo/alias.go` — alias table + `ExpandAlias()` function
2. Add argv[0] detection in `main.go` — check basename, strip `dkr-` prefix
3. Add `--cmd` parsing in `parse.go` — detect flag, expand, replace
4. Wire both paths to route through generic parser
5. Handle `--append` flag within stamp alias expansion

### REFACTOR

1. Verify all alias expansions match the spec table exactly.
2. Error messages for unknown aliases should list valid options.
3. Run with `-race`.

### CLI Integration Test

```bash
# --cmd equivalence
echo "hello" > a.c
dkredo test1 --cmd ifchange a.c
dkredo test2 +add-names a.c +check
# Verify .stamps/test1 and .stamps/test2 have identical content

# --cmd stamp
dkredo test3 --cmd stamp a.c
dkredo test4 +remove-names +add-names a.c +stamp-facts
# Verify identical stamps

# --cmd stamp --append
dkredo test5 +add-names old.c +stamp-facts
dkredo test5 --cmd stamp --append a.c
# Verify old.c still in stamp (not removed)

# --cmd always
dkredo test6 +add-names a.c +stamp-facts
dkredo test6 --cmd always
dkredo test6 +check
echo $?  # 0 (changed — facts cleared)

# --cmd fnames
dkredo test7 +add-names a.c b.h +stamp-facts
dkredo test7 --cmd fnames .c
# Output: a.c

# Error cases
dkredo test --cmd bogus       # exit 2
dkredo test --cmd             # exit 2
dkredo test --cmd ifchange +names  # exit 2 (mixed)

# Symlink dispatch (if symlinks created)
# dkr-ifchange test8 a.c     # same as --cmd ifchange
# dkr-stamp test8 a.c        # same as --cmd stamp
# dkr-always test8            # same as --cmd always
# dkr-bogus test8             # exit 2
```

## Results

### Files Created
- `cmd/dkredo/alias.go` — ExpandAlias, alias table, expansion functions
- `cmd/dkredo/alias_test.go` — 16 tests covering all aliases, --cmd parsing, and error cases

### Deviations
argv[0] dispatch implemented in main.go rather than a separate dispatch.go file. Both --cmd and argv[0] paths expand through ExpandAlias then parseOps, as specified.
