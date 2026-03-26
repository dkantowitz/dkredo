# dkredo — Composable Operations Design

## Motivation

The original dk-redo design uses separate commands (`dk-ifchange`, `dk-stamp`,
`dk-always`) that each bundle multiple internal steps. This works for common
cases but becomes awkward when recipes need finer control — e.g., querying
the stamp's file list for use in a build command, or adding new dependencies
without re-hashing existing ones.

This document redesigns dkredo around **primitive operations** that can be
composed via a sequence of `+operation` arguments in a single invocation. The existing
command names become built-in aliases for common operation sequences.

## Design Principles

1. **Operations are primitives.** Each `+operation` does one thing to the
   stamp's state: add names, remove names, compute hashes, compare facts,
   print file lists, read depfiles.

2. **Operations execute left to right.** The argument list is a pipeline
   of operations applied to a single label's stamp. This follows the
   ffmpeg/ImageMagick model where flag order reflects processing order.

3. **The `+` sigil marks operations.** `+` is shell-safe (no expansion in
   bash, zsh, dash, or fish), visually distinct from `-` flags, and signals
   "this is an action, not an option."

4. **Legacy commands are aliases.** `dk-ifchange label files...` is shorthand
   for `dkredo label +add-names files... +check`. No functionality is lost.

5. **Aliases use shell (or just) mechanisms.** Custom aliases beyond the
   built-ins are defined with shell aliases or just recipes. If common
   patterns emerge that can't be expressed this way, a `.dkredo` config
   file for project-defined aliases may be added later.

## Primitive Operations

### Stamp Manipulation

| Operation | Args | Description |
|-----------|------|-------------|
| `+add-names` | `file...` | Add files to stamp's name list. No facts computed. New entries are marked `new:true`. Existing entries are untouched. |
| `+remove-names` | `file...` | Remove files from stamp's name list and their facts. |
| `+sync-names` | `file...` | Replace the stamp's name list with exactly these files. Files already in the stamp keep their facts. New files get `new:true`. Files no longer listed are removed. |
| `+stamp` | `file...` | Compute and record facts (blake3, size, missing) for the given files. If no files given, re-stamp all files currently in the name list. Replaces the entire stamp. |
| `+stamp-append` | `file...` | Like `+stamp` but merges into existing stamp (add/update, preserve unmentioned). |
| `+read-depfile` | `-M file.d` | Parse makefile dep format, add extracted paths to the name list and compute their facts. |
| `+clear` | | Remove the stamp file entirely. |

### Querying

| Operation | Args | Description |
|-----------|------|-------------|
| `+names` | `[filter]` | Print file names from the stamp to stdout. Optional filter is an extension (`.c`, `.h`) or glob pattern. |
| `+facts` | `[file...]` | Print recorded facts for the given files (or all files). Diagnostic output. |

### Testing

| Operation | Args | Description |
|-----------|------|-------------|
| `+check` | | Compare stamp facts against current filesystem. Exit 0 if any fact fails (changed). Exit 1 if all facts hold (unchanged). Exit 2 on error. |
| `+check-assert` | | Like `+check` but exit 2 (error) instead of 1 when unchanged. For scripts that should never be called on an up-to-date target. |

### Input Modifiers

These modify how the *next* operation receives its file list:

| Modifier | Description |
|----------|-------------|
| `-` | Read file list from stdin (newline-terminated), pass to next operation. |
| `-0` | Read file list from stdin (null-terminated), pass to next operation. |
| `-M file.d` | Parse makefile dep format, pass extracted paths to next operation. |

## Operation Sequencing

Operations execute left to right. Each operation may read or modify the
stamp. The exit code comes from the last operation that produces one
(typically `+check`).

```
dkredo <label> [+operation [args...]]...
```

Arguments between `+operation` markers belong to that operation. The parser
consumes arguments greedily until the next `+` token or end of args.

### Examples

```bash
# Add names, then check — the "ifchange" pattern
dkredo out.bin +add-names src/main.c src/util.c +check

# Sync names from find, then check
dkredo out.bin +sync-names $(find src -name '*.c') +check

# Query .c names from stamp
dkredo out.bin +names .c

# Stamp from depfile — the "post-build" pattern
dkredo out.bin +stamp -M .deps/out.d

# Stamp with explicit files plus depfile
dkredo out.bin +stamp src/*.c -M .deps/out.d

# Clear stamp — the "always" pattern
dkredo out.bin +clear

# Print facts for debugging
dkredo out.bin +facts
```

## Built-in Aliases

The legacy command names map to operation sequences. These are compiled into
the binary and dispatched via argv[0] or subcommand.

| Alias | Equivalent |
|-------|------------|
| `dk-ifchange <label> [files...]` | `dkredo <label> +add-names [files...] +check` |
| `dk-ifchange <label>` (no files) | `dkredo <label> +check` |
| `dk-stamp <label> [files...]` | `dkredo <label> +stamp [files...]` |
| `dk-stamp --append <label> [files...]` | `dkredo <label> +stamp-append [files...]` |
| `dk-stamp <label> -M file.d` | `dkredo <label> +stamp -M file.d` |
| `dk-always <label>` | `dkredo <label> +clear` |
| `dk-always --all` | removes all stamp files (special case, no label) |
| `dk-fnames <label> [filter]` | `dkredo <label> +names [filter]` |

The aliases preserve full backward compatibility. Existing justfiles using
`dk-ifchange` / `dk-stamp` / `dk-always` continue to work unchanged.

### Alias note: dk-ifchange union behavior

`dk-ifchange <label> files...` maps to `+add-names files... +check` rather
than `+sync-names files... +check`. This preserves the union-with-stamp
behavior: new files are added, but previously-discovered dependencies (e.g.,
headers from a prior `-M` depfile) remain in the stamp and are still checked.

If you want strict "only these files" semantics, use `+sync-names` directly:

```bash
dkredo out.bin +sync-names $(find src -name '*.c') +check
```

## Canonical Usage Patterns

### C compilation with gcc dependency discovery

```just
set guards

compile:
    dkredo out.bin +add-names $(find src -name '*.c') +check
    gcc -o out.bin -MD -MF .deps/out.d $(dkredo out.bin +names .c)
    dkredo out.bin +stamp -M .deps/out.d
```

**Line 1:** Add any new `.c` files to the stamp (existing entries and their
facts are preserved, new entries get `new:true`). Then check all facts.
If unchanged, `?` stops the recipe.

**Line 2:** Query the stamp for `.c` files to pass to gcc. The stamp
contains both `.c` and `.h` files (from the previous build's depfile),
but only `.c` files are needed on the command line. gcc writes its
dependency discovery to `.deps/out.d`.

**Line 3:** Replace the stamp with facts computed from the depfile. This
captures both the `.c` files and all `#include`d headers. Next run, line 1's
`+check` will detect changes to any of them.

### Using the backward-compatible aliases

```just
set guards

compile:
    find src -name '*.c' | ?dk-ifchange out.bin -
    gcc -o out.bin -MD -MF .deps/out.d $(dk-fnames out.bin .c)
    dk-stamp out.bin -M .deps/out.d
```

Identical behavior, using the alias commands.

### Multi-phase build

```just
set guards

firmware:
    dkredo firmware.bin +add-names src/*.c include/*.h libs/*.a +check
    gcc -MD -MF .deps/firmware.d -c src/*.c -Iinclude/
    ld -o firmware.bin *.o libs/*.a
    dkredo firmware.bin +stamp src/*.c include/*.h libs/*.a -M .deps/firmware.d
```

### Deploy (side-effect recipe)

```just
set guards

deploy-staging:
    ?dkredo deploy-staging +add-names src/*.py config/staging.yaml +check
    kubectl apply -f k8s/staging/
    dkredo deploy-staging +stamp src/*.py config/staging.yaml
```

### Force rebuild

```just
clean:
    dkredo firmware.bin +clear
    dkredo deploy-staging +clear
```

## Handling the Deleted File Problem

When a `.c` file is deleted:

1. `+add-names` does not remove anything — the deleted file remains in the
   stamp from the previous `+stamp -M` call.
2. `+check` detects the file is missing (fact `size:N` fails because file
   doesn't exist) — exits 0 (changed), recipe continues.
3. `+names .c` returns the stale entry — gcc receives a nonexistent file
   and fails with a clear error ("No such file or directory").
4. Because gcc fails, `+stamp -M` never runs. The stale entry persists.
5. On the next run after the user fixes the issue (removes the `#include`,
   updates the source), gcc succeeds and `+stamp -M` writes a clean stamp.

This is **noisy but self-healing**. The gcc error message is accurate and
actionable. No silent incorrect behavior occurs.

**Alternative:** Use `+sync-names` instead of `+add-names` to proactively
remove stale entries:

```just
compile:
    dkredo out.bin +sync-names $(find src -name '*.c') +check
    gcc -o out.bin -MD -MF .deps/out.d $(dkredo out.bin +names .c)
    dkredo out.bin +stamp -M .deps/out.d
```

`+sync-names` replaces the `.c` entries with exactly what `find` returns,
removing the deleted file. The tradeoff: previously-discovered headers are
also removed from the name list (since they aren't in the `find` output).
However, `+check` would still detect changes because the stamp's *facts*
for those headers are gone (treated as changed). And `+stamp -M` restores
them after the build.

## Custom Aliases

For now, project-specific aliases use shell or just mechanisms:

```bash
# shell alias
alias dk-compile='dkredo out.bin +add-names $(find src -name "*.c") +check'

# shell function for parameterized use
dk-gcc-guard() {
    dkredo "$1" +add-names $(find "$2" -name '*.c') +check
}
```

```just
# just recipe as alias
_guard-compile label *srcs:
    dkredo {{label}} +add-names {{srcs}} +check
```

If recurring patterns emerge that can't be cleanly expressed with shell/just
aliases — for instance, project-wide defaults for filter patterns or depfile
paths — a `.dkredo` config file may be introduced:

```ini
# .dkredo (hypothetical, not yet implemented)
[alias]
guard-c = +add-names $(find src -name '*.c') +check
post-build = +stamp -M .deps/${LABEL}.d
```

This is deferred until there's evidence it's needed. Shell aliases and just
recipes cover the known use cases.

## Architecture

### Parsing

```
dkredo <label> [+op [args...]] [+op [args...]] ...
```

1. Extract label from arg[0].
2. Split remaining args on `+` boundaries. Each segment is one operation:
   the first token (after `+`) is the operation name, the rest are its args.
3. Execute operations sequentially, threading the stamp state through.

```go
type Operation struct {
    Name string
    Args []string
}

func parseOps(args []string) (string, []Operation) {
    label := args[0]
    var ops []Operation
    var current *Operation
    for _, arg := range args[1:] {
        if strings.HasPrefix(arg, "+") {
            if current != nil {
                ops = append(ops, *current)
            }
            current = &Operation{Name: strings.TrimPrefix(arg, "+")}
        } else if current != nil {
            current.Args = append(current.Args, arg)
        } else {
            // args before first +op: implicit file list for aliases
        }
    }
    if current != nil {
        ops = append(ops, *current)
    }
    return label, ops
}
```

### Execution

Each operation receives a `*StampState` and may read/modify it:

```go
type StampState struct {
    Label    string
    Names    []string            // current file list
    Facts    map[string][]Fact   // per-file facts (may be nil if not yet computed)
    Modified bool                // stamp needs writing
}

func execute(label string, ops []Operation) (exitCode int) {
    state := loadOrInitState(label)
    for _, op := range ops {
        exitCode = dispatch(op, state)
        if exitCode != 0 {
            break  // stop on first non-zero exit
        }
    }
    if state.Modified {
        writeStamp(state)
    }
    return exitCode
}
```

Operations that produce an exit code (`+check`) stop the pipeline if
non-zero. Operations that write to stdout (`+names`, `+facts`) do not
affect the exit code.

**Special case for +check:** `+check` returning exit 1 (unchanged) stops
the pipeline but does NOT prevent writing pending stamp modifications.
The `+add-names` from earlier in the pipeline must persist so that the
next run's `+check` includes the new entries.

### Binary dispatch

```go
func main() {
    cmd, args := resolveCommand(os.Args)
    switch cmd {
    case "redo":
        // native +operation mode
        runOps(args)
    case "ifchange":
        // alias: translate to +add-names ... +check
        runIfchangeAlias(args)
    case "stamp":
        // alias: translate to +stamp ...
        runStampAlias(args)
    case "always":
        runAlwaysAlias(args)
    case "fnames":
        runFnamesAlias(args)
    case "install":
        runInstall(args)
    default:
        usage(); os.Exit(2)
    }
}
```

Alias functions translate their arguments into operation sequences and
call `execute()`. This keeps one code path for all behavior.

## Relationship to dk-redo-implementation.md

This document supersedes the command-level algorithms in
`dk-redo-implementation.md`. The internal packages remain the same:

| Package | Role | Changes |
|---------|------|---------|
| `internal/stamp/` | Stamp read/write/compare | Add `Names()` accessor; `AddNames()`, `SyncNames()`, `RemoveNames()` methods |
| `internal/hasher/` | BLAKE3 file/directory hashing | No changes |
| `internal/resolve/` | Input resolution (files, dirs, stdin, depfile) | No changes |
| `cmd/dk-redo/` | CLI dispatch + operation execution | Rewritten: operation parser, executor, alias translation |

The test plan from `dk-redo-implementation.md` applies to the individual
operations. Integration tests should cover both the `+operation` syntax
and the alias commands to verify they produce identical results.

## Exit Codes

Unchanged from the original design:

- `0` — action taken / changed detected
- `1` — no action needed / unchanged (only from `+check`)
- `2` — error

Operations that don't produce a meaningful exit code (e.g., `+add-names`,
`+stamp`, `+names`) exit 0 on success and 2 on error.

## Performance Considerations

Each `dkredo` invocation is a single process. Composing operations within
one invocation avoids repeated process startup, stamp file reads, and
stamp file writes. For example:

```bash
# One process, one stamp read, one stamp write
dkredo out.bin +add-names src/*.c +check

# vs. two processes, two stamp reads, two stamp writes
dk-fnames --add out.bin src/*.c
dk-ifchange out.bin
```

The performance budget from `dk-redo-implementation.md` applies per
invocation, not per operation.
