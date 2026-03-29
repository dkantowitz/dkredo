# dkredo — Implementation

## Motivation

The original design uses separate commands (`dkr-ifchange`, `dkr-stamp`,
`dkr-always`) that each bundle multiple internal steps. This works for common
cases but becomes awkward when recipes need finer control — e.g., querying
the stamp's file list for use in a build command, or adding new dependencies
without re-hashing existing ones.

dkredo is built around **primitive operations** that can be composed via a
sequence of `+operation` arguments in a single invocation. The existing
command names become built-in aliases for common operation sequences.

## Design Principles

1. **Operations are primitives.** Each `+operation` does one thing with a
   stamp file's state: add names, remove names, compute hashes, compare facts,
   print file lists.

2. **Separate file lists from fact maintenance** Allows establishment of working file sets that are used consistently with rest of commands in recipe.

3. **Operations execute left to right.** The argument list is a pipeline
   of operations applied to a single label's stamp file. This follows the
   ffmpeg/ImageMagick model where flag order reflects processing order.

4. **The `+` marks operation words.** `+` is shell-safe (no expansion in
   bash, zsh, dash, or fish), visually distinct from `-` flags, and signals
   "this is an action, not an option."

5. **Legacy commands are aliases.** `dkr-ifchange label files...` is shorthand
   for `dkredo label +add-names files... +check`. No functionality is lost.

6. **Aliases use shell (or just) mechanisms.** Custom aliases beyond the
   built-ins are defined with shell aliases or just recipes. If common
   patterns emerge or approaches that can't be expressed this way, a `.dkredo` config
   file for project-defined aliases may be added later.

## Primitive Operations

### Stamp Manipulation

| Operation       | Args                | Description                                                                                                                                                           |
| --------------- | ------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `+add-names`    | `file...`           | Add files to stamp's name list. No facts computed. New entries have an empty fact list. Existing or duplicate entries are untouched.                                  |
| `+add-names`    | -M `file...`        | Parse makefile dep format, add extracted paths to the name list. No facts computed. New entries have an empty fact list. Existing or duplicate entries are untouched. |
| `+remove-names` | [`filter...`]       | Remove files from stamp's name list along with their facts. Empty filter matches every entry in the stamp file.                                                       |
| `+remove-names` | `-ne` `[filter...]` | Iff the filename does not exist and the stamp fact for that file is not `missing:true`, remove it from stamp's name list along with their facts.                      |
| `+stamp-facts`  | `[filter...]`       | Compute and record facts (blake3, size, missing) for the selected file names. If empty filter, re-calculate facts for all files currently in the stamp's name list. Does not add names — use `+add-names` first. |
| `+clear-facts`  | [`filter...`]       | Remove facts from filtered file names, but leave the filename in the stamp.                                                                                           |

### Querying

| Operation | Args             | Description                                                                                                                      |
| --------- | ---------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| `+names`  | `[filter...]`    | Print file names from the stamp to stdout. Optional filter is an extension (`.c`, `.h`) or glob pattern.                         |
| `+names`  | `-e [filter...]` | Print only file names that exist on disk from the stamp to stdout. Optional filter is an extension (`.c`, `.h`) or glob pattern. |
| `+facts`  | `[filter...]`    | Print recorded facts for the given files (or all files). Diagnostic output.                                                      |

**TODO (phase 2)**: glob patterns

**TODO (phase 2)**: an operation for listing only the dependency files that fail their facts. The purpose would be to recompile only the changed files, but there's a missing check to see if the .o exists. That is, a .c file could need recompiling because either the .c (or a dependency for that .c) changed _or_ the .o for that .c is missing. Currently the dkredo system is not able to track the .o dependencies the same way make does it.

### Verifying Facts

| Operation       | Args          | Description                                                                                                                                |
| --------------- | ------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| `+check`        | `[filter...]` | Compare stamp facts against current filesystem. Exit 0 if any fact fails (changed), if any entry has no facts, or if any entry has an unreadable fact line or an unknown fact key. Exit 1 if all facts hold (unchanged). An empty stamp (no entries) passes. Exit 2 on error. |
| `+check-assert` | `[filter...]` | Like `+check` but exit 2 (error) instead of 1 when unchanged. For scripts that should never be called on an up-to-date target.             |

### Files and Filters

| Modifier      | Description                                                                                                                                                                                                                 |
| ------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `file/path.x` | Matches the exact path.                                                                                                                                                                                                     |
| `.suffix`     | Filter matches all files with that suffix. Only available where argument is `filters`.                                                                                                                                      |
| `-`           | Read file list from stdin (newline-terminated), pass to current operation.                                                                                                                                                  |
| `-0`          | Read file list from stdin (null-terminated), pass to current operation.                                                                                                                                                     |
| `-@ file`     | Read file names from `file` (newline-terminated). Works with process substitution: `-@ <(fd -e c src)`.                                                                                                                     |
| `-@0 file`    | Read file names from `file` (null-terminated).                                                                                                                                                                              |
| `-M file.d`   | Parse makefile dep format (gcc `-MD` output), extract paths. An input mode available on any operation that accepts file arguments (like `-`, `-0`, `-@`, `-@0`).                                                             |

Note: `filter...` is strictly a superset of `file...`

**TODO (phase 2)**: Support glob patterns. I'm inclined to avoid adding a glob or regex library by relying on find and reading the file list from stdin.

## Operation Sequencing

Operations execute left to right. Each operation may read or modify the
stamp. The exit code comes from the last operation that produces one
(typically `+check`).

```
dkredo <label> [+operation [args...]]...
```

Arguments between `+operation` markers belong to the operation on the left. The parser
consumes arguments greedily until the next `+` token or end of args.

### Examples

```bash
# Add names, then check — the "ifchange" pattern
dkredo out.bin +add-names src/main.c src/util.c +check

# Sync names from find, then check
dkredo out.bin +add-names $(find src -name '*.c') +remove-names -ne +check

# Query .c names from stamp
dkredo out.bin +names .c .cpp

# Add names from depfile, then stamp — the "post-build" pattern
dkredo out.bin +add-names -M .deps/out.d +stamp-facts

# Add names from explicit files and depfile, then stamp
dkredo out.bin +add-names src/*.c -M .deps/out.d +stamp-facts

# Clear stamp — the "always" pattern
dkredo out.bin +clear-facts

# Add names from a file listing (process substitution)
dkredo out.bin +add-names -@ <(fd -e c src) +check

# Print facts for debugging
dkredo out.bin +facts

# Use aliases without symlinks
dkredo out.bin --cmd ifchange src/main.c src/util.c
dkredo out.bin --cmd stamp -M .deps/out.d
dkredo out.bin --cmd always
```

## Built-in Aliases

The legacy command names map to operation sequences. These are compiled into
the binary and dispatched via argv[0] (symlinks) or `--cmd`.

| Alias                                   | Equivalent                                                        |
| --------------------------------------- | ----------------------------------------------------------------- |
| `dkr-ifchange <label> [files...]`       | `dkredo <label> +add-names [files...] +check`                     |
| `dkr-ifchange <label>` (no files)       | `dkredo <label> +check`                                           |
| `dkr-stamp <label> [files...]`          | `dkredo <label> +remove-names +add-names [files...] +stamp-facts` |
| `dkr-stamp --append <label> [files...]` | `dkredo <label> +add-names [files...] +stamp-facts`               |
| `dkr-stamp --append <label> -M file.d` | `dkredo <label> +add-names -M file.d +stamp-facts`                |
| `dkr-stamp <label> -M file.d`           | `dkredo <label> +remove-names +add-names -M file.d +stamp-facts`  |
| `dkr-always <label>`                    | `dkredo <label> +clear-facts`                                     |
| `dkr-fnames <label> [filter]`           | `dkredo <label> +names -e [filter]`                               |

Aliases can also be invoked via `--cmd` without symlinks:

```bash
dkredo firmware.bin --cmd ifchange src/*.c    # same as dkr-ifchange
dkredo firmware.bin --cmd stamp src/*.c       # same as dkr-stamp
dkredo firmware.bin --cmd always              # same as dkr-always
```

`--cmd` expands the named alias into its operation sequence and routes it
through the generic CLI parser. It is not a `+operation` — it is a CLI
flag that provides symlink-free access to aliases.

The aliases preserve backward compatibility. Existing justfiles using
`dkr-ifchange` / `dkr-stamp` / `dkr-always` continue to work unchanged.

### Alias note: dkr-ifchange union behavior

`dkr-ifchange <label> files...` maps to `+add-names files... +check` rather
than `+remove-names +add-names files... +check`. This preserves the union-with-stamp
behavior: new files are added, but previously-discovered dependencies (e.g.,
headers from a prior `-M` depfile) remain in the stamp and are still checked.

If you want strict "only these files" semantics, add `+remove-names` to start with a clear stamp file:

```bash
dkredo out.bin +remove-names +add-names $(find src -name '*.c') +check
```

## Canonical Usage Patterns

### C compilation with gcc dependency discovery

```just
set guards

compile:
    ?dkredo out.bin +add-names $(find src -name '*.c') +check
    gcc -o out.bin -MMD -MF .deps/out.d $(dkredo out.bin +names -e .c)
    dkredo out.bin +add-names -M .deps/out.d +stamp-facts
```

**Line 1:** Add any new `.c` files to the stamp (existing entries and their
facts are preserved, new entries have no facts and fail the +check). Then check all facts.
If all facts match, `?` stops the recipe.

**Line 2:** Query the stamp for `.c` files to pass to gcc. The stamp
contains both `.c` and `.h` files (from the previous build's depfile),
but only `.c` files are needed on the command line. Deleted .c files are weeded out by
the `-e` options. gcc writes its dependency discovery to `.deps/out.d`.

**Line 3:** Add names from the depfile (both `.c` files and all `#include`d
headers), then stamp facts for all names in the stamp. Next run, line 1's
`+check` will detect changes to any of them.

### Using the backward-compatible aliases

```just
set guards

compile:
    ?dkr-ifchange out.bin $(find src -name '*.c')
    gcc -o out.bin -MMD -MF .deps/out.d $(dkr-fnames out.bin .c)
    dkr-stamp out.bin -M .deps/out.d
```

Identical behavior, using the alias commands.

### Multi-phase build

```just
set guards

firmware:
    ?dkredo firmware.bin +add-names src/*.c include/*.h libs/*.a +check
    gcc -MMD -MF .deps/firmware.d -c src/*.c -Iinclude/
    ld -o firmware.bin *.o libs/*.a
    dkredo firmware.bin +add-names libs/*.a -M .deps/firmware.d +stamp-facts
```

### Deploy (side-effect recipe)

```just
set guards

deploy-staging:
    ?dkredo deploy-staging +add-names src/*.py config/staging.yaml +remove-names -ne +check
    kubectl apply -f k8s/staging/
    dkredo deploy-staging +stamp-facts
```

### Force rebuild

```just
clean:
    dkredo firmware.bin +clear-facts
    dkredo deploy-staging +clear-facts
```

## Software Design

Each `+operation` is implemented as an independent unit with its own
function, tests, and clear interface to `StampState`. Operations are
developed and tested one at a time — a working `+add-names` can ship
before `+check` exists. The executor (`execute`) ties them together
but has no operation-specific logic; it just dispatches by name.

### File Organization

```
cmd/dkredo/             — CLI dispatch + operation execution
internal/ops/           — individual operations
internal/hasher/        — BLAKE3 file hashing
internal/resolve/       — input argument resolution (file vs stdin vs file-input, depfile)
```

### Generic CLI Parsing

```
dkredo <label> [+op [args...]] [+op [args...]] ...
```

1. Extract label from arg[0] (which is os.Args[1]).
2. Split remaining args on `+` boundaries. Each segment is one operation:
   the first token (after `+`) is the operation name, the rest are its args.
3. Read the current stamp state from `.stamps/{{label}}`.
4. Execute operations sequentially, passing the stamp state from op to op.

### $0 Dispatch Parsing

When the executable name is not `dkredo`, we're in alias mode (argv[0]
dispatch via symlinks, e.g., `dkr-ifchange`):

1. Strip the `dkr-` prefix from argv[0] to get the alias name.
2. Look up the alias template (e.g., `ifchange` → `+add-names <files> +check`).
3. Expand the template with the command-line args (label + remaining args).
4. Route the expanded args through Generic CLI Parsing.

Do not build the execution array directly — always route through the generic
CLI parse. This is a deliberate design choice to make changing or adding
alias commands very easy.

### --cmd Parsing

`--cmd` follows the same path as $0 dispatch but is triggered within a
normal `dkredo` invocation:

1. Parser encounters `--cmd` as a CLI flag.
2. The first arg to `--cmd` is the alias name (e.g., `ifchange`).
3. Remaining args are the alias's arguments.
4. Look up the alias template, expand with the args.
5. Replace the `--cmd` invocation with the expanded operation sequence.
6. Continue normal execution.

This means `dkredo out.bin --cmd ifchange src/*.c` and
`dkr-ifchange out.bin src/*.c` produce identical operation sequences —
both resolve to `+add-names src/*.c +check` before execution begins.

### --version Flag

`dkredo --version` prints the version string and exits 0. The version is
embedded at build time via Go's `-ldflags`:

```
go build -ldflags="-s -w -X main.version=$(git describe --tags --always --dirty)" ./cmd/dkredo
```

If no version is embedded (development build), print `dkredo dev`.

### --stamps-dir Flag

`--stamps-dir <path>` overrides the automatic `.stamps/` directory
location. It must appear before the label on the command line:

```
dkredo --stamps-dir /tmp/my-stamps out.bin +add-names src/*.c +check
dkr-ifchange --stamps-dir /tmp/my-stamps out.bin src/*.c
```

When `--stamps-dir` is set, the upward-search algorithm in
`find_stamps_dir()` is bypassed entirely — the given path is used as-is.
The directory is created on first write if it does not exist.

All file paths in stamps are stored relative to the `--stamps-dir`
directory's parent, just as they would be for an auto-discovered `.stamps/`
directory. This keeps stamp contents portable regardless of how the
directory was located.

`--stamps-dir` is parsed early in CLI processing and threaded through the
executor to all stamp I/O functions. It works with both `+operation` style
and alias (`--cmd` / argv[0]) invocations.

### DKREDO_ARGS Environment Variable

If the `DKREDO_ARGS` environment variable is set, its value is
shell-split and inserted between argv[0] and argv[1] before any other
parsing. This allows flags like `--stamps-dir` and `-v` to be set
once per project rather than repeated in every recipe line.

```bash
# In a justfile or shell profile:
export DKREDO_ARGS="--stamps-dir .build/stamps -v"

# These two are now equivalent:
dkredo out.bin +check
dkredo --stamps-dir .build/stamps -v out.bin +check
```

The effective argument list is: `[argv[0]] + split(DKREDO_ARGS) + argv[1:]`.
Splitting follows POSIX shell quoting rules (respects single/double quotes
and backslash escapes) so that paths with spaces work:

```bash
export DKREDO_ARGS='--stamps-dir "/path with spaces/.stamps"'
```

`DKREDO_ARGS` applies to all invocation styles — `dkredo`, `--cmd`, and
argv[0] alias dispatch. Because the args are inserted before the label,
they occupy the same position as flags typed on the command line and go
through the same parsing path.

**No subcommand style.** There is no `dkredo ifchange ...` form. The first
positional argument to `dkredo` is always the label. This eliminates the
parsing ambiguity of "is this a subcommand name or a label?" — `--cmd`
provides the symlink-free alternative instead.

**Single `--cmd` per invocation.** It is an error to have multiple --cmd args. Detect this early in cli argument processing.

### Execution

Each operation receives a `*StampState` and may read/modify it:

```text
StampState:
    label:    string
    names:    list of string      # current file list
    facts:    dict[string, list]  # per-file facts (may be empty if not yet computed)
    modified: bool                # in-memory record had been modified

function execute(label, ops):
    state = load_or_init_state(label)
    exit_code = 0
    for op in ops:
        exit_code = op_dispatch(op, state)
        if exit_code != 0:
            break  # stop on first non-zero exit
    if state.modified:
        write_stamp(state)
    return exit_code
```

Operations that produce an exit code (`+check`) stop the pipeline if
non-zero. Operations that write to stdout (`+names`, `+facts`) do not
affect the exit code.

**Special case for +check:** `+check` returning exit 1 (unchanged) stops
the pipeline but does NOT prevent writing pending stamp modifications.
The `+add-names` from earlier in the pipeline must persist so that the
next run's `+check` includes the new entries.

Command functions receive args as a parameter rather than reading globals
directly. This avoids mutating global state and makes it easier to source
arguments from other places — critical for alias expansion, which routes
through the generic CLI parser.

### -v Flag (Verbose)

`-v` logs diagnostic messages to stderr showing what each operation does.
It must appear before the label on the command line:

```
dkredo -v <label> +add-names src/*.c +check
dkr-ifchange -v out.bin src/*.c
```

Each operation emits one or more lines to stderr when `-v` is active:

| Operation       | Verbose output                                                      |
| --------------- | ------------------------------------------------------------------- |
| `+add-names`    | `+add-names: added 3 new entries (5 total)` — count of new vs existing |
| `+remove-names` | `+remove-names: removed 2 entries (3 remaining)` — or `+remove-names -ne: removed gone.c (file missing, not expected absent)` |
| `+stamp-facts`  | `+stamp-facts: computed facts for 5 files` — and for each file: `  src/main.c blake3:ab12... size:1842` |
| `+clear-facts`  | `+clear-facts: cleared facts for 3 entries`                         |
| `+check`        | `+check: changed (src/main.c: size differs)` — or `+check: unchanged (5 files, all facts match)` |
| `+check-assert` | Same as `+check`                                                    |
| `+names`        | (no extra output — already writes to stdout)                        |
| `+facts`        | (no extra output — already writes to stdout)                        |
| stamp I/O       | `stamp: loaded .stamps/out.bin (5 entries)` on read, `stamp: wrote .stamps/out.bin (5 entries)` on write |

Verbose output goes to stderr so it never interferes with stdout output
from `+names` or `+facts`. The `-v` flag is threaded through the executor
to each operation function.

### Atomic Writes

```text
function write_stamp(path, content):
    tmp = path + ".tmp." + str(pid())
    write_file(tmp, content, mode=0644)
    rename(tmp, path)
```

Write to temp file + rename into place. Crash between the two operations is an
accepted deficiency — the mechanism is well-established (used by redo-c, goredo,
and many other tools).

**TODO (phase 2)**: Investigate atomic write on Windows (non-POSIX rename semantics).

### Facts for Non-Existent Files

When a file does not exist (`os.ErrNotExist`), record `missing:true` as its
only fact — no hash or size. This enables the bootstrapping pattern: the fact
becomes false when the file is created, triggering a rebuild.

### Label Escaping

```text
function escape_label(label):
    label = label.replace("%", "%25")   # escape char first
    label = label.replace("/", "%2F")
    return label
```

### Path Encoding in Stamps

```text
function encode_path(path):
    path = path.replace("%", "%25")    # escape char first
    path = path.replace("\t", "%09")
    path = path.replace("\n", "%0A")
    return path
```

## Pseudocode Sketches

Draft algorithms for core operations. These will be replaced with references
to `file:fn()` as the code is written.

### Stamps Directory Location

```text
function find_stamps_dir():
    dir = cwd()
    while dir != filesystem_root:
        if exists(dir + "/.stamps") and is_directory(dir + "/.stamps"):
            return dir + "/.stamps"
        dir = parent(dir)
    # Not found — will be created in cwd on first stamp write
    return None

function stamps_dir():
    found = find_stamps_dir()
    if found is not None:
        return found
    # Lazy create: only on write, not on read
    path = cwd() + "/.stamps"
    mkdir(path)
    return path
```

All file paths in stamps are stored relative to the `.stamps/` directory's
parent (the project root). This ensures the same file referenced from
different working directories resolves to the same stamp entry.

### Input Resolution

```text
function resolve_inputs(raw_args, stdin_paths):
    items = []

    # 1) Build ordered item stream: positional args, with '-' or '-0'
    # replaced by stdin paths, and '-@ file' or '-@0 file' replaced
    # by paths read from the named file.
    for arg in raw_args:
        if arg == '-' or arg == '-0':
            items.extend(stdin_paths)
        elif arg == '-@':
            file_arg = next_arg()
            items.extend(read_lines(file_arg))
        elif arg == '-@0':
            file_arg = next_arg()
            items.extend(read_null_terminated(file_arg))
        else:
            items.append(arg)

    # 2) Canonicalize: make paths relative to the .stamps/ parent dir
    #    (the project root). Normalize separators to '/'.
    #    ./src/main.c and src/main.c resolve to the same entry.
    root = stamps_dir().parent
    canon = [relpath(abspath(p), root) for p in items]
    canon.sort()

    # 3) De-duplicate exact path repeats.
    return unique_preserving_order(canon)
```

### File Facts

```text
function file_facts(path):
    if not exists(path):
        return "missing:true"
    sz = file_size(path)          # stat(), not read
    data = read_all_bytes(path)
    h = blake3(data).hex()
    return "blake3:" + h + " size:" + str(sz)
```

### Path Encoding

```text
function encode_path(path):
    path = path.replace("%", "%25")    # escape char first
    path = path.replace("\t", "%09")
    path = path.replace("\n", "%0A")
    return path

function stamp_line(path):
    return encode_path(path) + "\t" + file_facts(path)
```

### Change Detection

```text
function is_changed(stamp_lines, current_paths):
    # Different file list means changed.
    if set(stamp_paths(stamp_lines)) != set(current_paths):
        return true
    # Check each file's recorded facts against reality.
    for line in stamp_lines:
        path, facts = parse_line(line)
        if facts is unparseable:
            warn(stderr, "unreadable fact line for " + path)
            return true           # can't verify → treat as changed
        if has_unknown_fact_keys(facts):
            warn(stderr, "unknown fact key in " + path)
            return true           # can't verify → treat as changed
        if "missing:true" in facts:
            if exists(path):
                return true       # file appeared
        else:
            if not exists(path):
                return true       # file disappeared
            # Fast path: check size first (stat only, no read).
            recorded_size = parse_fact(facts, "size")
            if file_size(path) != recorded_size:
                return true       # size differs → changed, skip hash
            recorded_hash = parse_fact(facts, "blake3")
            if blake3(read_all_bytes(path)).hex() != recorded_hash:
                return true
    return false
```

**Unreadable or unknown facts:** If a stamp line cannot be parsed (e.g.,
binary data, missing tab delimiter, truncated line), or if it contains fact
keys not recognized by the current version (e.g., `future:xyz`), `+check`
treats the entry as changed and emits a warning to stderr. The reasoning:
we cannot verify facts we don't understand, so we must conservatively
assume the file may have changed. When the label is next stamped,
`+stamp-facts` writes a fresh stamp with only known facts, clearing the
unrecognized entries.

## Exit Codes

- `0` — action taken / change detected
- `1` — no action needed / unchanged (only from `+check`)
- `2` — error

Operations that don't produce a meaningful exit code (e.g., `+add-names`,
`+stamp-facts`, `+names`) exit 0 on success and 2 on error.

## Performance

### Performance Budget

| Operation                              | Target  | Notes                      |
| -------------------------------------- | ------- | -------------------------- |
| +check (unchanged, 10 files)           | < 10ms  | Read stamp + hash 10 files |
| +check (unchanged, 1000 files)         | < 200ms | I/O bound                  |
| +stamp-facts (100 files)               | < 50ms  | Hash + atomic write        |
| Startup overhead                       | < 5ms   | Before any I/O             |

### Benchmarks

| Benchmark                  | Setup                                | Target        |
| -------------------------- | ------------------------------------ | ------------- |
| BenchmarkCheckUnchanged10  | 10 files, stamp exists               | < 10ms        |
| BenchmarkCheckUnchanged300 | 300 files across 10 labels (30 each) | < 300ms total |
| BenchmarkStamp100          | 100 small files                      | < 50ms        |
| BenchmarkStartupOverhead   | no-op invocation (--help)            | < 5ms         |

The 300-dependency benchmark is the primary regression target. Benchmarks run
as Go benchmark tests (`go test -bench`) and can be gated in CI.

## Test Plan

Tests are organized around individual operations. Each operation has its own
unit tests (`go test`). Integration tests are implemented as justfiles that
exercise multi-operation sequences and alias commands against real files.

### Unit Tests: +add-names

| Test                         | Setup                             | Expected                                               |
| ---------------------------- | --------------------------------- | ------------------------------------------------------ |
| Add to empty stamp           | no stamp exists, add 3 files      | names list contains 3 entries, no facts, stamp written |
| Add duplicates               | stamp has a.c, add a.c again      | no change, a.c appears once                            |
| Add preserves existing facts | stamp has a.c with facts, add b.c | a.c facts untouched, b.c has empty facts               |
| Add with -M depfile          | gcc .d file listing 5 headers     | all 5 paths added to names                             |
| Add with -M and files        | files + -M depfile                | union of both                                          |
| Add with stdin (-)           | pipe 3 paths                      | 3 files added                                          |
| Add with -@ file             | file listing 3 paths              | 3 files added                                          |
| Add with -@0 file            | null-terminated file              | correctly parsed and added                             |
| Add with -@ <(process sub)   | `-@ <(fd -e c src)`              | files from process substitution added                  |
| Add with stdin (-0)          | null-terminated paths             | correctly parsed and added                             |
| Add deduplicates             | same file from args and stdin     | listed once                                            |
| Empty args                   | no files given                    | no change to stamp                                     |

### Unit Tests: +remove-names

| Test                                           | Setup                                   | Expected                               |
| ---------------------------------------------- | --------------------------------------- | -------------------------------------- |
| Remove by exact path                           | stamp has a.c, b.c; remove a.c          | only b.c remains                       |
| Remove by suffix filter                        | stamp has a.c, b.h; remove .h           | only a.c remains                       |
| Remove all (empty filter)                      | stamp has 3 files; +remove-names        | stamp empty                            |
| Remove -ne: file exists                        | stamp has a.c (exists on disk)          | a.c NOT removed                        |
| Remove -ne: file missing, fact is missing:true | stamp has gone.c with missing:true      | gone.c NOT removed (expected absent)   |
| Remove -ne: file missing, fact is blake3       | stamp has gone.c with stale blake3 fact | gone.c removed (was expected to exist) |
| Remove nonexistent name                        | remove x.c not in stamp                 | no change                              |

### Unit Tests: +stamp-facts

| Test                     | Setup                                   | Expected                                |
| ------------------------ | --------------------------------------- | --------------------------------------- |
| Stamp all (empty filter) | stamp has a.c, b.c with empty facts     | facts computed for both (blake3 + size) |
| Stamp by filter          | stamp has a.c, b.h; +stamp-facts .c     | a.c gets facts, b.h unchanged           |
| Stamp missing file       | stamp has gone.c, file doesn't exist    | fact recorded as missing:true           |
| Stamp with -M depfile    | +add-names -M .deps/out.d +stamp-facts  | names added from depfile, facts computed for all |
| -M without +add-names    | stamp has a.c only; +stamp-facts -M file.d (file.d lists b.h) | b.h NOT added to stamp; only a.c stamped |
| Facts are deterministic  | stamp same file twice                   | identical blake3 + size                 |
| Size fast path           | stamp, change file content but not size | blake3 changes, size stays              |
| Symlink target           | stamp has symlink                       | facts reflect target content, not link  |

### Unit Tests: +clear-facts

| Test                     | Setup                               | Expected                               |
| ------------------------ | ----------------------------------- | -------------------------------------- |
| Clear all (empty filter) | stamp has a.c, b.c with facts       | names preserved, all facts removed     |
| Clear by filter          | stamp has a.c, b.h; +clear-facts .h | a.c facts untouched, b.h facts cleared |
| Clear already empty      | stamp has a.c with no facts         | no change                              |

### Unit Tests: +check

| Test                   | Setup                                   | Expected                                  |
| ---------------------- | --------------------------------------- | ----------------------------------------- |
| No stamp exists        | first run                               | exit 0 (changed)                          |
| All facts match        | stamp matches filesystem                | exit 1 (unchanged)                        |
| File content changed   | modify a stamped file                   | exit 0 (changed)                          |
| File size changed      | change file size                        | exit 0 (changed, size fast path, no hash) |
| File appeared          | stamp has missing:true, file now exists | exit 0 (changed)                          |
| File disappeared       | stamp has facts, file deleted           | exit 0 (changed)                          |
| Unknown fact key       | stamp line has future:xyz fact          | exit 0 (changed), warning to stderr       |
| Unreadable fact line   | stamp line has no tab or binary garbage | exit 0 (changed), warning to stderr       |
| Corrupt stamp          | garbage content                         | exit 2 (error)                            |
| Check with filter      | stamp has a.c, b.h; +check .c           | only a.c checked                          |
| Empty stamp (no names) | stamp exists but has no entries         | exit 1 (unchanged — nothing to check)     |

### Unit Tests: +check-assert

| Test      | Setup           | Expected                                              |
| --------- | --------------- | ----------------------------------------------------- |
| Changed   | file modified   | exit 0 (same as +check)                               |
| Unchanged | all facts match | exit 2 (error — should not be called when up to date) |

### Unit Tests: +names

| Test                  | Setup                                    | Expected              |
| --------------------- | ---------------------------------------- | --------------------- |
| All names             | stamp has a.c, b.h                       | prints both to stdout |
| Filter by suffix      | stamp has a.c, b.h; +names .c            | prints a.c only       |
| With -e (exists only) | stamp has a.c (exists), gone.c (missing) | prints a.c only       |
| Empty stamp           | no entries                               | empty output          |

### Unit Tests: +facts

| Test        | Setup                        | Expected                     |
| ----------- | ---------------------------- | ---------------------------- |
| All facts   | stamp has 2 files with facts | prints path + facts for each |
| Filter      | +facts .c                    | only .c entries shown        |
| Empty facts | file in stamp with no facts  | shows name with no facts     |

### Unit Tests: internal packages

**hasher:**

| Test                   | Input                                | Expected                       |
| ---------------------- | ------------------------------------ | ------------------------------ |
| Hash file with content | temp file "hello"                    | deterministic BLAKE3 + size    |
| Hash empty file        | temp file ""                         | BLAKE3 of empty + size:0       |
| Hash missing file      | nonexistent path                     | missing:true                   |
| Hash permission denied | unreadable file                      | error                          |
| Hash follows symlink   | symlink to file                      | hash of target content         |

**stamp I/O:**

| Test                      | Input                   | Expected                           |
| ------------------------- | ----------------------- | ---------------------------------- |
| Write then read roundtrip | label + names + facts   | roundtrip matches                  |
| Write creates .stamps/    | nonexistent .stamps dir | auto-created                       |
| Write is atomic           | interrupt mid-write     | no partial stamp on disk           |
| Read missing stamp        | nonexistent label       | "no stamp" (not error)             |
| Label escaping            | "output/config.json"    | .stamps/output%2Fconfig.json       |
| Label with literal %      | "100%done"              | .stamps/100%25done                 |
| Path with tab             | "dir\tname/file"        | tab encoded as %09, roundtrips     |
| Path with percent         | "100%/file"             | percent encoded as %25, roundtrips |
| Path with spaces          | "my file.c"             | spaces stored verbatim, roundtrips |

**resolve:**

| Test              | Input                               | Expected                      |
| ----------------- | ----------------------------------- | ----------------------------- |
| File args         | ["src/a.c", "src/b.c"]              | two file paths                |
| Stdin newline     | stdin="x.c\ny.c\n", args=["-"]      | two files                     |
| Stdin null        | stdin="x.c\0y.c\0", args=["-0"]     | two files                     |
| -@ file           | file listing paths, args=["-@", "list.txt"] | paths from file           |
| -@0 file          | null-term file, args=["-@0", "list.txt"]    | paths from file           |
| -@ process sub    | args=["-@", "<(fd -e c)"]           | paths from process sub        |
| Stdin splicing    | ["a.c", "-", "b.c"] + stdin="x.c\n" | a.c, x.c, b.c                 |
| Deduplication     | same file from args and stdin       | listed once                   |
| Stdin on tty      | ["-"] but stdin is tty              | error                         |

### Integration Tests (justfile-based)

Integration tests are implemented as justfile recipes that run the real
`dkredo` binary against real files. Each test recipe creates temp files,
runs operations, and asserts exit codes and stamp contents.

**Operation pipeline tests:**

| Test                                 | Recipe                                                     | Expected                                              |
| ------------------------------------ | ---------------------------------------------------------- | ----------------------------------------------------- |
| Guard/build/stamp cycle              | +add-names + +check → build → +stamp-facts                 | first run: exit 0; second run: exit 1                 |
| File change triggers rebuild         | +stamp-facts, modify file, +check                          | exit 0                                                |
| Name addition persists across +check | +add-names a.c +check (exit 1)                             | a.c in stamp names even though check stopped pipeline |
| +remove-names + +stamp-facts         | add 3 files, remove 1, stamp                               | stamp has 2 files with facts                          |
| +clear-facts forces re-check         | +stamp-facts, +clear-facts, +check                         | exit 0 (changed — no facts to verify)                 |
| Missing file bootstrapping           | +add-names nonexistent.c +stamp-facts, create file, +check | exit 0                                                |
| Depfile integration                  | build with gcc -MD, +add-names -M .deps/out.d +stamp-facts, +names | stamp contains headers from depfile                   |

**Alias (--cmd) tests:**

| Test                 | Recipe                                       | Expected                                                         |
| -------------------- | -------------------------------------------- | ---------------------------------------------------------------- |
| --cmd ifchange       | `dkredo label --cmd ifchange files...`       | identical stamp to `+add-names files +check`                     |
| --cmd stamp          | `dkredo label --cmd stamp files...`          | identical stamp to `+remove-names +add-names files +stamp-facts` |
| --cmd stamp --append | `dkredo label --cmd stamp --append files...` | identical stamp to `+add-names files +stamp-facts`               |
| --cmd always         | `dkredo label --cmd always`                  | identical stamp to `+clear-facts`                                |
| --cmd fnames         | `dkredo label --cmd fnames .c`               | identical output to `+names -e .c`                               |

**Symlink dispatch tests:**

| Test                 | Recipe                        | Expected                 |
| -------------------- | ----------------------------- | ------------------------ |
| dkr-ifchange symlink | `dkr-ifchange label files...` | same as `--cmd ifchange` |
| dkr-stamp symlink    | `dkr-stamp label files...`    | same as `--cmd stamp`    |
| dkr-always symlink   | `dkr-always label`            | same as `--cmd always`   |
| Unknown symlink      | `dkr-bogus label`             | exit 2 with usage error  |

**Edge cases:**

| Test                   | Recipe                                     | Expected                                          |
| ---------------------- | ------------------------------------------ | ------------------------------------------------- |
| Label with slash       | label "output/config.json"                 | stamp at .stamps/output%2Fconfig.json, roundtrips |
| Label with spaces      | label "my build"                           | stamp at .stamps/my build, roundtrips             |
| Corrupt stamp recovery | write garbage to .stamps/label, run +check | exit 2; +stamp-facts overwrites with valid stamp  |
| Empty stamp            | +add-names (no files) then +check          | exit 1 (nothing to check)                         |
| Concurrent access      | two dkredo invocations on different labels | no interference                                   |

**.stamps/ directory location tests:**

| Test | Recipe | Expected |
|------|--------|----------|
| .stamps/ in cwd | create .stamps/ in cwd, run dkredo | uses existing .stamps/ |
| .stamps/ in parent | cd into subdir, run dkredo | finds .stamps/ in parent |
| .stamps/ in grandparent | cd two levels deep, run dkredo | finds .stamps/ in grandparent |
| No .stamps/ anywhere | fresh temp dir, run +stamp-facts | creates .stamps/ in cwd |
| Nested project | parent has .stamps/, child has own .stamps/ | child's .stamps/ wins (closest) |
| Paths are project-relative | cd into subdir, stamp src/main.c | stamp entry is relative to .stamps/ parent |

**--install tests:**

| Test | Recipe | Expected |
|------|--------|----------|
| Install to directory | `dkredo --install /tmp/test-bin` | binary copied, all symlinks created |
| Install creates symlinks | check /tmp/test-bin/dkr-ifchange etc. | all alias symlinks point to dkredo |
| Install over existing | run --install twice | no error, files replaced |
| Install to missing dir | `dkredo --install /nonexistent/path` | exit 2 with error message |
| Install permissions | target dir not writable | exit 2 with error message |

**--cmd parsing edge cases:**

| Test | Recipe | Expected |
|------|--------|----------|
| --cmd with invalid alias | `dkredo label --cmd bogus` | exit 2 with error listing valid aliases |
| --cmd with no alias name | `dkredo label --cmd` | exit 2 with error |
| --cmd mixed with +ops | `dkredo label --cmd ifchange +names` | error: --cmd and +operations cannot be mixed |
| --help flag | `dkredo --help` | full help text, exit 0 |
| -h flag | `dkredo -h` | short help text, exit 0 |
| No operations | `dkredo label` | exit 2 with help |
| --version flag | `dkredo --version` | prints version string, exit 0 |
| --version dev build | no ldflags version embedded | prints `dkredo dev`, exit 0 |

**--stamps-dir tests:**

| Test | Recipe | Expected |
|------|--------|----------|
| Override stamps dir | `dkredo --stamps-dir /tmp/s label +stamp-facts` | stamp written to /tmp/s/label |
| Dir created on write | `--stamps-dir` to nonexistent dir, +stamp-facts | dir created, stamp written |
| Dir not created on read | `--stamps-dir` to nonexistent dir, +check | no dir created, exit 0 (no stamp) |
| Paths relative to parent | `--stamps-dir /tmp/s`, stamp src/main.c | entry is relative to /tmp/s parent (/tmp) |
| Works with aliases | `dkr-ifchange --stamps-dir /tmp/s label files` | uses /tmp/s instead of .stamps/ |

**DKREDO_ARGS tests:**

| Test | Recipe | Expected |
|------|--------|----------|
| Stamps dir via env | `DKREDO_ARGS="--stamps-dir /tmp/s" dkredo label +stamp-facts` | stamp written to /tmp/s/label |
| Verbose via env | `DKREDO_ARGS="-v" dkredo label +check` | verbose output on stderr |
| Combined with CLI args | `DKREDO_ARGS="-v" dkredo --stamps-dir /tmp/s label +check` | both -v and --stamps-dir active |
| Quoted path with spaces | `DKREDO_ARGS='--stamps-dir "/tmp/my stamps"' dkredo label +stamp-facts` | uses path with spaces |
| Empty DKREDO_ARGS | `DKREDO_ARGS="" dkredo label +check` | no effect, normal behavior |
| Unset DKREDO_ARGS | (not set) `dkredo label +check` | no effect, normal behavior |
| Works with argv[0] alias | `DKREDO_ARGS="--stamps-dir /tmp/s" dkr-ifchange label files` | uses /tmp/s |

**-v verbose tests:**

| Test | Recipe | Expected |
|------|--------|----------|
| -v with +add-names | `dkredo -v label +add-names a.c b.c` | stderr shows count of added entries |
| -v with +check changed | `dkredo -v label +check` (file modified) | stderr shows which file and why |
| -v with +check unchanged | `dkredo -v label +check` (all match) | stderr shows "unchanged" with count |
| -v with +stamp-facts | `dkredo -v label +stamp-facts` | stderr shows per-file facts |
| -v stamp I/O | `dkredo -v label +check` | stderr shows "loaded .stamps/..." |
| -v does not pollute stdout | `dkredo -v label +names` | stdout has names only, verbose on stderr |
| No -v | `dkredo label +check` | no stderr output (unless warning) |

**-M depfile parsing edge cases:**

| Test | Recipe | Expected |
|------|--------|----------|
| Simple depfile | `out.o: src/main.c src/util.h` | 2 paths extracted |
| Multi-line continuation | `out.o: a.c \`<br>`  b.c c.c` | 3 paths extracted |
| Multiple targets | `out.o out.d: a.c b.c` | 2 paths (targets ignored) |
| Paths with spaces | depfile with escaped spaces | paths correctly parsed |
| Empty depfile | empty file | no paths, no error |
| Missing depfile | nonexistent .d file | exit 2 with error |
| Malformed depfile | garbage content | exit 2 with error |

## Design Rationale: Why No Directory Arguments?

dkredo does not accept directory paths as dependency inputs. Scanning
subdirectories is inherently complex: depth limits, symlink following,
.gitignore awareness, dotfile handling, and cross-platform differences all
require policy decisions that a minimal tool should not bake in.

Instead, dkredo delegates directory scanning to specialized external tools
(`find`, `fd`, `git ls-files`) and reads their output via `-@` with
process substitution:

```bash
# Recommended: -@ with process substitution
dkredo dist/assets +add-names -@ <(fd -e png -e jpg static/images) +check

# Also works: -@ with a file
fd -e py src > /tmp/pyfiles.txt
dkredo lint-check +add-names -@ /tmp/pyfiles.txt +check

# Stdin pipe also works, but -@ is preferred (better error propagation)
fd -e rs src | dkredo target/release/engine +add-names - +check
```

**Why `-@` over stdin pipes?** With a stdin pipe (`fd ... | dkredo ... -`),
if `fd` fails, dkredo sees empty stdin and silently proceeds with zero files.
With `-@ <(fd ...)`, a failed process substitution produces a read error
that dkredo can detect and report.

## Software Development Plan

### Code Coverage Targets

**Phase 1 target: 80% line coverage per package.** The purpose is error
discovery — finding untested code paths during active development. Coverage
gaps at this stage point to code that needs tests. The 80% target applies
to each `internal/` package individually, not just the aggregate.

**Post-stabilization target: 95% line coverage per package.** Once all
Phase 1 features are implemented and integration tests pass, raise the bar.
At this stage, high coverage serves as regression staking — future changes
are unlikely to silently break existing behavior. The remaining 5% covers
genuinely unreachable defensive code (e.g., OS-level error paths).

Justfile targets:

```just
cover:
    go test -coverprofile=coverage.out -covermode=atomic ./internal/...
    go tool cover -func=coverage.out

cover-html:
    go test -coverprofile=coverage.out -covermode=atomic ./internal/...
    go tool cover -html=coverage.out -o coverage.html

cover-check:
    go test -coverprofile=coverage.out -covermode=atomic ./internal/...
    @go tool cover -func=coverage.out | grep ^total | awk '{print $$3}' | \
        awk -F. '{if ($$1 < 80) {print "FAIL: coverage " $$1 "% < 80%"; exit 1} else {print "OK: coverage " $$1 "%"}}'
```

Exclusions: test files, generated code, and `cmd/dkredo/main.go` (thin
dispatch layer) are excluded from coverage requirements.

## Future Work (Phase 3)

| Feature | Description |
| ------- | ----------- |
| `+add-dir-listing` (tentative) | Track a directory's sorted filename list as a dependency. Detects file additions and deletions within a directory (not content changes -- individual file entries handle that). Behaviorally equivalent to hashing `ls \| sort`. |

## Test Sources to Study

- **apenwarr/redo** (`github.com/apenwarr/redo`): comprehensive test suite
  in `t/` directory using `sharness` — best source for change-detection edge cases
- **redo-c** (`github.com/leahneukirchen/redo-c`): single-file C, easy to
  read for edge cases
- **goredo**: test patterns for the Go implementation
