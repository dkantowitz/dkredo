# dk-redo — File-Dependency Guards for Justfiles

`dk-redo` brings redo-style content-hash change detection into `just` recipes
using the `?` guard sigil (just v1.47+). One system, no separate build tool.

## Philosophy

`just` is the command runner.

`dk-redo` is the minimalist dependency-tracking.

Together they make a simple makefile replacement that can be mixed into a justfile.

**Not a fork of redo.** dk-redo borrows redo's core insight (content hashing
beats timestamps) and its naming conventions, but does not implement .do
scripts, automatic transitive rebuilds, or a build orchestrator. The justfile
_is_ your build description.

**Not a fork of just.** dk-redo is used with just, but the only integration point is the '?' sigil.

## Installation

dk-redo is a single binary. Symlinks or subcommands provide the short command names.

```bash
# install the main binary
cp dk-redo /usr/local/bin/dk-redo
chmod +x /usr/local/bin/dk-redo

# create symlinks for argv[0] dispatch
cd /usr/local/bin
for cmd in dk-ifchange dk-stamp dk-always \
           dk-ood dk-affects dk-dot dk-sources; do
    ln -sf dk-redo "$cmd"
done
```

Both invocation styles work identically:

```bash
dk-redo ifchange firmware.bin src/*.c    # subcommand style
dk-ifchange firmware.bin src/*.c         # symlink style (argv[0] dispatch)
```

## Quick Start

```just
set guards

firmware:
    ?dk-ifchange firmware.bin src/*.c include/*.h
    arm-none-eabi-gcc -o firmware.bin src/*.c -Iinclude/
    dk-stamp firmware.bin src/*.c include/*.h

init-db:
    ?dk-ifchange data/app.db schema.sql
    sqlite3 data/app.db < schema.sql
    dk-stamp data/app.db schema.sql

clean:
    dk-always firmware.bin data/app.db
```

How it reads:

- `?dk-ifchange firmware.bin ...` — "if inputs to label `firmware.bin` haven't
  changed, skip this recipe"
- `dk-stamp firmware.bin ...` — "record current input state for `firmware.bin`"
- The `?` sigil (just v1.47+) stops the recipe cleanly on exit 1 — no error,
  other recipes continue

> Note: `set guards` is required in the justfile for the `?` sigil to work.
> All justfile examples in this document assume it is set.

## The Label

The first argument to `dk-ifchange`, `dk-stamp`, and `dk-always`
is the **label** — a unique key that identifies this set of dependencies. It
maps directly to a stamp file: `.stamps/<label>`.

dk-redo does not interpret the label. It never checks whether the label refers
to an existing file, never hashes it, never builds it. It is purely a namespace
key for the stamp.

### Label examples

**Output file path** — the natural default. Unique per recipe, makes
diagnostic output (`dk-ood`, `dk-affects`) self-explanatory:

```just
firmware:
    ?dk-ifchange firmware.bin src/*.c include/*.h
    gcc -o firmware.bin src/*.c -Iinclude/
    dk-stamp firmware.bin src/*.c include/*.h
```

**Output with path** — labels can include directories. `/` is escaped to `%`
in the stamp filename:

```just
generate-config:
    ?dk-ifchange output/config.json config.yaml templates/*.j2
    render-config config.yaml -o output/config.json
    dk-stamp output/config.json config.yaml templates/*.j2
```

Stamp file: `.stamps/output%2Fconfig.json`

**Descriptive label** — for recipes that produce side effects, not files.
Any unique string works:

```just
deploy-staging:
    ?dk-ifchange deploy-staging src/*.py config/staging.yaml
    kubectl apply -f k8s/staging/
    dk-stamp deploy-staging src/*.py config/staging.yaml

run-migrations:
    ?dk-ifchange run-migrations migrations/*.sql
    psql -f migrations/apply.sh
    dk-stamp run-migrations migrations/*.sql
```

**Parameterized label** — incorporates the recipe argument so each
invocation gets its own stamp:

```just
compile target:
    ?dk-ifchange {{target}} src/{{target}}.c include/*.h
    gcc -o build/{{target}} src/{{target}}.c
    dk-stamp {{target}} src/{{target}}.c include/*.h
```

`just compile foo` stamps `.stamps/foo`, `just compile bar` stamps
`.stamps/bar` — independent tracking.

**Directory as label** — when the output is a directory:

```just
assets:
    ?dk-ifchange dist/assets static/images/ static/fonts/
    optimize-assets -o dist/assets/
    dk-stamp dist/assets static/images/ static/fonts/
```

Stamp file: `.stamps/dist%2Fassets`

**Why a label is required:** Without it, there is no way to namespace stamps
per recipe. If two recipes depend on the same file (e.g., `config.json`),
whichever recipe runs first would stamp the file, and the second recipe would
see "unchanged" and skip — even though it never built against the current
content. The label isolates each recipe's "last seen" state.

## Naming

Why `dk-` instead of `redo-`? Two reasons:

1. **No collision** with actual redo installations
2. **The parent binary needs a name** that isn't one of the subcommands —
   `dk-redo` is that name, and `dk-*` is the consistent prefix

The subcommand names after the prefix (`ifchange`, `stamp`, `always`,
`ood`, `affects`, `dot`, `sources`) match redo conventions exactly.
Anyone who knows redo will recognize them.

## Commands Reference

### Core Commands

#### `dk-ifchange` — guard: skip recipe if inputs unchanged

```
dk-ifchange <label> [inputs...]
dk-ifchange <label> -              # read input list from stdin (\n terminated)
dk-ifchange <label> -0             # read input list from stdin (\0 terminated)
dk-ifchange <label> src/*.c -      # positional args + stdin combined
```

The first argument is the **label** — a unique key for this recipe's stamp
(see [The Label](#the-label)). Remaining arguments are input files or
directories.

`dk-ifchange` compares the current state of input files against the stored
stamp's per-file facts. Because globs are expanded by the shell before dk-redo
sees them, adding or removing files that match the glob changes the argument
list, which triggers a rebuild. Each file's facts (size, blake3 hash,
existence) are checked — if any fact no longer holds, the inputs are
considered changed. Size is checked first as a fast path to avoid hashing.

**Exit codes:**

- `0` — inputs changed (or first run) — recipe continues
- `1` — inputs unchanged — `?` sigil stops recipe silently
- `2` — error (corrupt stamp, I/O error, etc.) — recipe stops with error

The `?` sigil only intercepts exit code 1. Exit code 2 propagates as a
recipe failure, which is the correct behavior — errors should not be silent.

**Flags:**

| Flag | Description                                              |
| ---- | -------------------------------------------------------- |
| `-v` | Verbose: print which files changed                       |
| `-n` | Dry run: report changed/unchanged without updating state |
| `-q` | Quiet: suppress "up to date" message                     |

#### `dk-stamp` — record current input state

```
dk-stamp [--append] <label> [inputs...]
dk-stamp [--append] <label> -|-0                # stdin, same modes as dk-ifchange
dk-stamp [--append] <label> src/*.c -      # positional args + stdin combined
```

Records per-file facts (hash, size, existence) for each input under `<label>`.
Called after a successful build so the next `dk-ifchange` can detect changes.

**By default, `dk-stamp` replaces the entire stamp** — the previous input list
and facts are discarded. This is the correct behavior when `dk-stamp` is
called once at the end of a recipe with the complete input list.

**With `--append`, `dk-stamp` merges into the existing stamp:**

- New files are added to the input list
- Files already in the stamp have their facts updated (new hash, size)
- Files not mentioned in the current call are preserved

This supports multi-phase builds where inputs are discovered incrementally:

```just
firmware:
    ?dk-ifchange firmware.bin src/*.c include/*.h libs/*.a
    gcc -MD -MF .deps/firmware.d -c src/*.c -Iinclude/
    ld -o firmware.bin *.o libs/*.a
    dk-stamp firmware.bin src/*.c include/*.h libs/*.a
    dk-stamp --append firmware.bin - < .deps/firmware.d
```

By using `--append`, the final stamp contains the union: gcc's
discovered headers from the dep file AND the explicit source/lib inputs.

**Exit codes:**

- `0` — stamp written
- `2` — error (missing files, permission denied, etc.)

**Flags:**

| Flag       | Description                                      |
| ---------- | ------------------------------------------------ |
| `-v`       | Verbose: print stamp path and per-file facts      |
| `-q`       | Quiet: no output on success                      |
| `--append` | Merge into existing stamp instead of replacing it |

> **Design note: why separate ifchange and stamp?**
>
> In redo, the .do script wraps the build — redo records state automatically.
> In dk-redo, the `?` sigil stops the recipe _before_ the build runs, so
> state must be recorded _after_ the build succeeds. If the build fails,
> no stamp is written, and the next run correctly rebuilds.

**Non-existent input files:** If an input file listed in `dk-stamp` does not
exist at stamp time, dk-stamp records `missing:true` for that file (no hash or
size). On the next `dk-ifchange`, if the file still doesn't exist, the fact
holds and no change is detected. If the file has been created, `missing:true`
is no longer true, triggering a rebuild. This is how the bootstrapping pattern
works (see [Bootstrapping](#bootstrapping-file-doesnt-exist-yet)).

#### `dk-always` — invalidate stamps (force rebuild)

```
dk-always <label> [label...]
```

Removes the stamp file(s), so the next `dk-ifchange` will always proceed.
This is the escape hatch / force-rebuild mechanism.

**Exit codes:**

- `0` — always (even if stamp didn't exist)

**Flags:**

| Flag    | Description                  |
| ------- | ---------------------------- |
| `--all` | Remove all stamps            |
| `-v`    | Verbose: list removed stamps |

### Diagnostic Commands (rev1.x)

These commands are planned for rev1.x. They operate on the stamp data
written by the core commands.

#### `dk-ood` — list out-of-date labels

```
dk-ood [labels...]       # check specific labels (default: all)
```

For each stamped label, re-checks per-file facts (size, then hash) against
current file state. Prints labels whose inputs have changed.

**Exit codes:**

- `0` — at least one label is out of date
- `1` — all specified labels are up to date
- `2` — error, or no stamps exist

Exit codes follow the same convention as `dk-ifchange`: 0 means "something
needs action," 1 means "nothing to do." Exit code 2 distinguishes "no stamps
at all" from "stamps exist and are all current."

**Flags:**

| Flag     | Description                 |
| -------- | --------------------------- |
| `-v`     | Show per-file fact details  |
| `-q`     | Just exit code, no output   |
| `--json` | Output as JSON array        |

#### `dk-affects` — reverse dependency query

```
dk-affects <file> [file...]
```

Scans all stamp files to find which labels list the given file(s) as
inputs. Answers: "if I change `src/uart.c`, what needs rebuilding?"

**Exit codes:**

- `0` — found affected labels (printed to stdout)
- `1` — no labels depend on the given files

**Flags:**

| Flag     | Description                           |
| -------- | ------------------------------------- |
| `-v`     | Show which input triggered each label |
| `--json` | Output as JSON                        |

#### `dk-dot` — dependency graph in Graphviz DOT format

```
dk-dot [labels...]       # specific labels (default: all)
```

Emits a DOT-format directed graph of label-to-input dependencies. Pipe to
`dot -Tsvg` or `dot -Tpng` for visualization.

**Exit codes:**

- `0` — graph emitted
- `2` — error (no stamps, I/O error)

**Flags:**

| Flag   | Description                              |
| ------ | ---------------------------------------- |
| `--lr` | Left-to-right layout (default: top-down) |

#### `dk-sources` — list all tracked input files

```
dk-sources
```

Union of all input lists across all stamp files. Answers: "what files
does dk-redo know about?"

**Exit codes:**

- `0` — sources listed (or no stamps exist — prints nothing)
- `2` — error (corrupt stamp, I/O error)

**Flags:**

| Flag | Description                       |
| ---- | --------------------------------- |
| `-v` | Show which label tracks each file |

## Input Modes

All commands that accept input file arguments (`dk-ifchange`, `dk-stamp`)
support three input modes. They can be combined freely. `dk-affects` accepts
the same file/directory/stdin arguments for its query files.

### 1. Positional arguments (shell-expanded)

```bash
dk-ifchange firmware.bin src/*.c include/*.h
#           ^^^^^^^^^^^^ ^^^^^^^ ^^^^^^^^^^^
#           label        shell expands these before dk-redo sees them
```

The shell expands globs. dk-redo receives concrete file paths.

### 2. Directory arguments

```bash
dk-ifchange dist/assets static/images/ static/fonts/
```

When an argument is a directory (trailing `/` optional — dk-redo checks with
`test -d`), dk-redo hashes the entire tree: finds all files recursively,
sorts for determinism, hashes contents. Detects added, removed, and modified
files within the directory.

### 3. Stdin — newline or null-terminated

```bash
# newline-terminated (default stdin mode)
find src -name '*.c' -newer baseline | dk-ifchange firmware.bin -

# null-terminated (for filenames with spaces/newlines)
fd -0 -e h include | dk-ifchange firmware.bin -0

# combine with positional args
fd -e j2 templates | dk-ifchange output-config config.yaml -
```

`-` reads newline-terminated lines from stdin. `-0` reads null-terminated.
These appear as positional arguments and can be mixed with file/directory args.

## Stamp Storage

All state lives in a single `.stamps/` directory at the project root
(same level as the justfile). dk-redo creates `.stamps/` automatically on
first use if it does not exist.

```
.stamps/
  firmware.bin           # stamp for label "firmware.bin"
  deploy-staging         # stamp for label "deploy-staging"
  output%2Fconfig.json   # stamp for label "output/config.json" (/ escaped as %2F)
```

Labels are escaped for use as flat filenames using **percent-encoding**
(the same scheme as URL encoding):

| Character | Encoded | Why |
| --------- | ------- | --- |
| `/`       | `%2F`   | Cannot appear in filenames |
| `%`       | `%25`   | Escape character itself |

All other characters are passed through verbatim. This keeps stamps in a
flat directory while remaining unambiguous and reversible.

### File Format

Each stamp is a single file containing per-file facts. Each line is a
file path and its facts, separated by a **tab character**:

```
src/main.c	blake3:9f2a... size:1234
src/uart.c	blake3:d41e... size:567
include/config.h	blake3:7bc1... size:89
assets/large-blob.bin	blake3:c8f0... size:52428800
generated/version.h	missing:true
```

The tab delimiter means paths with spaces are handled correctly. Standard
tools (`sort`, `cut -f1`, `grep`) work naturally on this format.

Paths inside the stamp are **percent-encoded** for characters that would
break line parsing:

| Character | Encoded | Why |
| --------- | ------- | --- |
| `\t` (tab)| `%09`   | Tab is the path/facts delimiter |
| `\n` (newline) | `%0A` | Newline is the line delimiter |
| `%`       | `%25`   | Escape character itself |

All other characters (including spaces) are stored verbatim. In practice
this encoding almost never activates — tabs and newlines in filenames are
vanishingly rare — but when it does, errors are clear rather than mysterious.

Lines are sorted by path. Facts are space-separated `key:value` pairs after
the tab. Defined facts:

| Fact | Value | When recorded |
| ---- | ----- | ------------- |
| `blake3` | hex digest | Always (for existing files) |
| `size` | decimal byte count | Always (for existing files) |
| `missing` | `true` | File did not exist at stamp time |

A file is **changed** if any recorded fact is no longer true. `size` is
checked first as a fast path — if size differs, the hash is not recomputed
(a `stat()` call is far cheaper than reading + hashing the file).

A missing file records only `missing:true` (no hash or size). When the
file is later created, the `missing:true` fact becomes false, triggering
a rebuild.

**Forward compatibility:** readers should ignore unknown fact keys. This
allows future versions to add new facts without breaking older readers.

**Why one file, not two?** Atomicity. The stamp is written atomically
(write to temp, rename into place). With two files (.hash + .deps), a crash
between writes leaves inconsistent state. One file = one atomic unit.

Add `.stamps/` to `.gitignore` — these are local build state, not
version-controlled artifacts.

### Hashing Specification

dk-redo uses **BLAKE3** for all content hashes in rev1. BLAKE3 is chosen
for speed and collision resistance, not cryptographic security — we need
fast, deterministic, unique-enough digests for change detection.

- Per-file digest: BLAKE3 over raw file bytes, 256-bit (64 hex chars)
- Per-file facts: `blake3:<hex> size:<bytes>` (always both), or `missing:true`
- Change detection: any fact that no longer holds means the file changed

The goal is deterministic results across machines for the same workspace
content and input set.

```text
# Inputs:
#   raw_args: input arguments after <label> (files, dirs, -, -0)
#   stdin_mode: none | newline | nul
#   stdin_paths: parsed from stdin if mode is newline/nul

function resolve_inputs(raw_args, stdin_paths):
    items = []

    # 1) Build ordered item stream: positional args, with '-' or '-0'
    # replaced by stdin paths at that position.
    for arg in raw_args:
        if arg == '-' or arg == '-0':
            items.extend(stdin_paths)
        else:
            items.append(arg)

    # 2) Expand directories recursively to files.
    expanded = []
    for item in items:
        if is_directory(item):
            # Walk recursively, include files only, lexical sort by path.
            files = walk_files_recursive(item)
            files.sort()
            expanded.extend(files)
        else:
            expanded.append(item)

    # 3) Canonicalize paths (project-relative, '/' separators) and sort.
    canon = [canonical_relpath(p) for p in expanded]
    canon.sort()

    # 4) De-duplicate exact path repeats.
    return unique_preserving_order(canon)


function file_facts(path):
    if not exists(path):
        return "missing:true"
    sz = file_size(path)          # stat(), not read
    data = read_all_bytes(path)
    h = blake3(data).hex()
    return "blake3:" + h + " size:" + str(sz)


function encode_path(path):
    # Percent-encode only the characters that break parsing.
    path = path.replace("%", "%25")    # escape char first
    path = path.replace("\t", "%09")
    path = path.replace("\n", "%0A")
    return path

function stamp_line(path):
    return encode_path(path) + "\t" + file_facts(path)


function is_changed(stamp_lines, current_paths):
    # Different file list means changed.
    if set(stamp_paths(stamp_lines)) != set(current_paths):
        return true
    # Check each file's recorded facts against reality.
    for line in stamp_lines:
        path, facts = parse_line(line)    # split on tab, decode_path(path)
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


# Stamp content (one line per input, tab-delimited, sorted by path):
#   <path>\t blake3:<hex> size:<bytes>
#   <path>\tmissing:true
```

## Dry Run

```bash
dk-ifchange -n firmware.bin src/*.c include/*.h
```

With `-n`, dk-ifchange reports what _would_ happen without updating any state:

```
firmware.bin: CHANGED (3 files modified: src/main.c, src/uart.c, include/config.h)
```

or:

```
firmware.bin: up to date
```

This is simpler than redo's dry-run problem. Redo discovers dependencies
_during_ .do script execution, so dry-run requires stale graph data. dk-redo's
dependencies are explicit in the justfile — the stamp file records exactly
what was checked last time, and the current file list is known from the
command arguments. No speculation needed.

`dk-ood` is the multi-target dry-run: it checks all stamps and lists which
are out of date.

## Justfile Patterns

### Basic: file dependencies

```just
set guards

firmware:
    ?dk-ifchange firmware.bin src/*.c include/*.h
    arm-none-eabi-gcc -o firmware.bin src/*.c -Iinclude/
    dk-stamp firmware.bin src/*.c include/*.h
```

### Directory-level dependencies

```just
assets:
    ?dk-ifchange dist/assets static/images/ static/fonts/
    optimize-assets -o dist/assets/
    dk-stamp dist/assets static/images/ static/fonts/
```

### Complex deps via fd/find

```just
[script('bash')]
engine:
    rc=0
    fd -e rs src | dk-ifchange target/release/engine - || rc=$?
    if [ "$rc" -eq 1 ]; then exit 0; fi    # unchanged — skip
    if [ "$rc" -ne 0 ]; then exit "$rc"; fi # error — propagate
    cargo build --release
    fd -e rs src | dk-stamp target/release/engine -
```

> Note: stdin mode requires a `[script]` recipe (single shell process) so
> the pipe works. The explicit exit-code check replaces the `?` sigil
> (which operates on individual lines, not piped commands) and correctly
> distinguishes "unchanged" (exit 1) from errors (exit 2+).

### Bootstrapping (file doesn't exist yet)

```just
init-db:
    ?dk-ifchange data/app.db schema.sql
    sqlite3 data/app.db < schema.sql
    dk-stamp data/app.db schema.sql
```

On first run, no stamp file exists at `.stamps/data%2Fapp.db`, so `dk-ifchange`
returns 0 (changed) and the recipe runs. `dk-stamp` then records the current
hash of `schema.sql`. Subsequent runs skip unless `schema.sql` changes.

### Side-effect recipes (no output file)

```just
deploy-staging:
    ?dk-ifchange deploy-staging src/*.py config/staging.yaml
    kubectl apply -f k8s/staging/
    dk-stamp deploy-staging src/*.py config/staging.yaml

run-migrations:
    ?dk-ifchange run-migrations migrations/*.sql
    psql -f migrations/apply.sh
    dk-stamp run-migrations migrations/*.sql
```

### Detecting recipe or compiler flag changes

Build outputs depend on more than source files — compiler version, flags, and
the recipe itself are implicit inputs. Two approaches:

**Coarse: track the justfile itself.** Any edit to any recipe triggers all
guarded recipes. Simple, zero overhead, good for small projects:

```just
firmware:
    ?dk-ifchange firmware.bin justfile src/*.c include/*.h
    {{CC}} {{CFLAGS}} -o firmware.bin src/*.c -Iinclude/
    dk-stamp firmware.bin justfile src/*.c include/*.h
```

**Precise: capture flags in a file, track it as an input.** A dedicated
recipe writes the current compiler identity and flags to a file. Build
recipes depend on it via just's recipe dependencies — the flags file is
always written (cheap), and `dk-ifchange` detects when the content changes.
This works because dk-redo uses content hashing, not timestamps — rewriting
a file with identical content produces the same hash.

```just
CC := "arm-none-eabi-gcc"
CFLAGS := "-O2 -DNDEBUG -Iinclude/"

cc_cflags:
    @mkdir -p .deps
    @echo '{{CC}} {{CFLAGS}}' > .deps/cc_cflags

firmware: cc_cflags
    ?dk-ifchange firmware.bin .deps/cc_cflags src/*.c include/*.h
    {{CC}} {{CFLAGS}} -o firmware.bin src/*.c
    dk-stamp firmware.bin .deps/cc_cflags src/*.c include/*.h

release: cc_cflags firmware
    ?dk-ifchange release.tar.gz .deps/cc_cflags firmware.bin config.json
    package-release firmware.bin config.json -o release.tar.gz
    dk-stamp release.tar.gz .deps/cc_cflags firmware.bin config.json
```

`cc_cflags` runs unconditionally (no `?` guard) — it's the truth source
for the current toolchain. Multiple recipes can depend on it. Change the
compiler or flip an optimization flag, and every downstream recipe rebuilds.

The pattern generalizes to any tool: `python_version`, `node_env`,
`docker_tag`, etc.

### Using gcc -MD dependency output

Combine gcc's discovered headers with directory hashing for full coverage.
On first run, the dep file doesn't exist yet — use `touch` to create an
empty one so the stdin redirect doesn't fail:

```just
firmware: cc_cflags
    @mkdir -p .deps && touch .deps/firmware.d
    ?dk-ifchange firmware.bin .deps/cc_cflags include/ - < .deps/firmware.d
    gcc -MD -MF .deps/firmware.d -o firmware.bin src/*.c -Iinclude/
    dk-stamp firmware.bin .deps/cc_cflags include/ - < .deps/firmware.d
```

The `include/` directory argument catches new files appearing in search paths
(the negative-dependency gap that gcc's dep output misses — see
[Design Decisions](#no-negative-dependencies-and-why-thats-ok)). The dep file
provides precise per-header tracking including system headers.

### Force-rebuild and diagnostics

```just
# force-rebuild escape hatch
rebuild label:
    dk-always {{label}}
    just firmware

# see what's stale
status:
    dk-ood

# visualize dependency graph
deps:
    dk-dot | dot -Tsvg > deps.svg

# "what breaks if I change this file?"
what-breaks file:
    dk-affects {{file}}
```

### Recipe that chains outputs

```just
all: firmware release

firmware:
    ?dk-ifchange firmware.bin src/*.c include/*.h
    gcc -o firmware.bin src/*.c -Iinclude/
    dk-stamp firmware.bin src/*.c include/*.h

release: firmware
    ?dk-ifchange release.tar.gz firmware.bin config.json
    package-release firmware.bin config.json -o release.tar.gz
    dk-stamp release.tar.gz firmware.bin config.json
```

Just's recipe dependencies handle ordering. `release` runs after `firmware`.
Each recipe independently decides whether to skip via its own guard.

### Parameterized recipes

```just
compile target:
    ?dk-ifchange {{target}} src/{{target}}.c include/*.h
    gcc -o build/{{target}} src/{{target}}.c
    dk-stamp {{target}} src/{{target}}.c include/*.h
```

The label incorporates the parameter, so `just compile foo` and
`just compile bar` get independent stamps.

## Feature Tiers

Based on analysis of apenwarr/redo, goredo, and redo-c argument interfaces:

### Minimum Viable (rev1)

| Feature                                | Source                  |
| -------------------------------------- | ----------------------- |
| Content-hash change detection          | Core redo concept       |
| `dk-ifchange`, `dk-stamp`, `dk-always` | redo core commands      |
| File, directory, and stdin input modes | dk-redo design          |
| `-v` (verbose), `-q` (quiet) flags     | Universal in redo impls |
| Atomic stamp writes                    | redo-c/goredo practice  |

### Good to Have (rev1.x)

| Feature                           | Source                              |
| --------------------------------- | ----------------------------------- |
| `-n` dry run on ifchange          | goredo (only impl with `--dry-run`) |
| `--append` on dk-stamp            | dk-redo design                      |
| `dk-ood` (out-of-date query)      | apenwarr/redo, goredo               |
| `dk-affects` (reverse deps)       | goredo-only feature                 |
| `dk-sources` (list tracked files) | apenwarr/redo, goredo, redo-c       |
| `dk-dot` (Graphviz output)        | goredo-only feature                 |
| `--all` flag on dk-always         | dk-redo convenience                 |

### Fancy (rev2)

| Feature                                    | Source                                  |
| ------------------------------------------ | --------------------------------------- |
| Transitive dependency tracking             | Core redo feature — see below           |
| `--json` output on diagnostic commands     | Modern CLI practice                     |
| `dk-log` (build log capture)               | apenwarr/redo, goredo                   |
| Parallel stamp checking                    | goredo `-j`                             |


## Transitive Dependency Tracking (rev2 Roadmap)

> **Not implemented.** This section describes planned rev2 behavior.

Redo's signature feature: if label A depends on an input that is also a
tracked label B, and B's inputs change, then A is also out of date —
automatically.

### What it requires

The stamp format already stores inputs. The missing piece: **labels can appear
as inputs of other labels.**

```
# .stamps/firmware.bin
src/main.c blake3:abc123...
src/uart.c blake3:fed987...

# .stamps/release.tar.gz — depends on firmware.bin (also a label)
config.json blake3:ccc111...
firmware.bin blake3:def456...
```

For rev2, `dk-ifchange` would:

1. Resolve the input list as now
2. For each input, check if a stamp exists with that name (i.e., it's also a
   dk-redo label, not just a source file)
3. If so, recursively check whether _that_ label is out of date
4. A label is out of date if its own inputs changed OR any of its
   label-deps are out of date

**Graph walk, not re-execution.** Unlike full redo, dk-redo would not
automatically rebuild transitive deps — it would report the full out-of-date
chain and let just's recipe dependencies handle ordering. The justfile already
declares the execution order:

```just
all: firmware release

firmware:
    ?dk-ifchange firmware.bin src/*.c include/*.h
    gcc -o firmware.bin src/*.c
    dk-stamp firmware.bin src/*.c include/*.h

release: firmware
    ?dk-ifchange release.tar.gz firmware.bin config.json
    package-release firmware.bin config.json -o release.tar.gz
    dk-stamp release.tar.gz firmware.bin config.json
```

With transitive checking, `dk-ifchange release.tar.gz ...` would detect that
`firmware.bin` is a tracked label, check firmware.bin's stamp, and propagate
staleness upward. Without it (rev1), the `release: firmware` dependency
in the justfile ensures firmware rebuilds first, and the new `firmware.bin`
has a different hash, so `dk-ifchange release.tar.gz` proceeds anyway.

**Rev1 works correctly without transitive tracking** — just's recipe deps
provide ordering, and content hashing catches the changes. Transitive tracking
adds efficiency (skip the hash computation early) and enables `dk-ood` and
`dk-affects` to report the full chain.

### What it does NOT require

- No .do scripts
- No stamp format changes (inputs already recorded)
- No new commands — `dk-ifchange` gains the graph walk internally
- No change to justfile patterns

## Argument Summary

### Shared flags (all commands)

| Flag                     | Description                                    |
| ------------------------ | ---------------------------------------------- |
| `-v`                     | Verbose output                                 |
| `-q`                     | Quiet (suppress informational output)          |
| `--color` / `--no-color` | ANSI color control (default: auto-detect tty)  |
| `--stamps-dir <path>`    | Override stamp directory (default: `.stamps/`)  |
| `--help`                 | Show usage                                     |
| `--version`              | Show version                                   |

### Arguments (ifchange, stamp)

| Argument  | Type               | Description                                             |
| --------- | ------------------ | ------------------------------------------------------- |
| `<label>` | positional (first) | Unique key for this stamp (see [The Label](#the-label)) |
| `<file>`  | positional         | Input file path                                         |
| `<dir>`   | positional         | Input directory — hashed recursively                    |
| `-`       | positional         | Read input list from stdin, newline-terminated           |
| `-0`      | positional         | Read input list from stdin, null-terminated              |

### Arguments (always)

| Argument  | Type       | Description                             |
| --------- | ---------- | --------------------------------------- |
| `<label>` | positional | Label(s) whose stamps should be removed |

### Arguments (ood, dot)

| Argument  | Type                  | Description                      |
| --------- | --------------------- | -------------------------------- |
| `<label>` | positional (optional) | Label(s) to check (default: all) |

### Arguments (affects)

| Argument | Type       | Description            |
| -------- | ---------- | ---------------------- |
| `<file>` | positional | Input file(s) to query |

## Comparison with redo

|                        | redo                              | dk-redo                              |
| ---------------------- | --------------------------------- | ------------------------------------ |
| Build description      | `.do` shell scripts               | justfile recipes                     |
| Dependency declaration | `redo-ifchange` inside .do        | `dk-ifchange` guard line             |
| Change detection       | Content hash (SHA1/SHA256/BLAKE2) | Content hash (BLAKE3)                |
| Transitive rebuilds    | Automatic                         | Via just recipe deps (rev2: checked) |
| Parallel builds        | Built-in (`-j`)                   | Via just `[parallel]` attribute      |
| Task listing           | Limited (`redo-targets`)          | `just --list`                        |
| Dry run                | goredo only                       | `dk-ifchange -n` / `dk-ood`         |
| Stamp storage          | `.redo/` (SQLite or recfiles)     | `.stamps/` (plain text)             |
| Learning curve         | Moderate (new build paradigm)     | Low (just + one convention)          |

## Design Decisions

### No negative dependencies (and why that's OK)

redo has `redo-ifcreate` — a "negative dependency" that triggers a rebuild
when a currently-absent file comes into existence. The classic case: include
path search order. If `#include <config.h>` resolves to `/usr/include/config.h`
because `./include/config.h` doesn't exist, the build implicitly depends on
that file _staying absent_. If someone creates `./include/config.h`, it
shadows the system header and the build should re-run.

dk-redo does not have negative dependencies. Here's why, and what to do
instead.

**`gcc -M` / `-MD` does not catch the shadow case.** Compiler-generated dep
files record resolved paths of files that _were_ included — not paths that
were searched and missed. If `./include/config.h` appears and would shadow
`/usr/include/config.h`, the old dep file still lists the system header
(unchanged), and `dk-ifchange` skips. The shadow is invisible.

**Directory hashing catches it.** Depend on the include directories, not just
the files within them:

```just
firmware:
    ?dk-ifchange firmware.bin include/ src/*.c
    gcc -o firmware.bin src/*.c -Iinclude/
    dk-stamp firmware.bin include/ src/*.c
```

A new file in `include/` changes the directory hash. Rebuild triggers.

See [Using gcc -MD dependency output](#using-gcc--md-dependency-output) for
the full hybrid pattern combining directory hashing with per-header tracking.
