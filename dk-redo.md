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

Stamp file: `.stamps/output%config.json`

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

Stamp file: `.stamps/dist%assets`

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

`dk-ifchange` hashes whatever input files it receives as arguments, computes
a combined hash, and compares it to the stored stamp. Because globs are
expanded by the shell before dk-redo sees them, adding or removing files that
match the glob changes the argument list, which changes the combined hash,
which triggers a rebuild. The stamp also records the input file list, so a
change in the set of files (not just their contents) is detected.

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

Records the combined content hash plus input file list under `<label>`.
Called after a successful build so the next `dk-ifchange` can detect changes.

**By default, `dk-stamp` replaces the entire stamp** — the previous input list
and hashes are discarded. This is the correct behavior when `dk-stamp` is
called once at the end of a recipe with the complete input list.

**With `--append`, `dk-stamp` merges into the existing stamp:**

- New files are added to the input list
- Files already in the stamp have their hashes updated
- Files not mentioned in the current call are preserved
- The combined hash is recomputed over the merged file list

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
| `-v`       | Verbose: print stamp path and hash               |
| `-q`       | Quiet: no output on success                      |
| `--append` | Merge into existing stamp instead of replacing it |

> **Design note: why separate ifchange and stamp?**
>
> In redo, the .do script wraps the build — redo records state automatically.
> In dk-redo, the `?` sigil stops the recipe _before_ the build runs, so
> state must be recorded _after_ the build succeeds. If the build fails,
> no stamp is written, and the next run correctly rebuilds.

**Non-existent input files:** If an input file listed in `dk-stamp` does not
exist at stamp time, dk-stamp records a sentinel hash for that file (distinct
from any real content hash). On the next `dk-ifchange`, if the file still
doesn't exist, the sentinel matches and no change is detected. If the file
has been created, its real hash won't match the sentinel, triggering a rebuild.
This is how the bootstrapping pattern works (see
[Bootstrapping](#bootstrapping-file-doesnt-exist-yet)).

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

For each stamped label, re-hashes its inputs and compares to stored hash.
Prints labels whose inputs have changed.

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
| `-v`     | Show hash details per label |
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
  output%config.json     # stamp for label "output/config.json" (/ escaped as %)
```

Labels containing `/` are escaped: `/` becomes `%`. This keeps all
stamps in a flat directory (same convention as systemd unit escaping).
Labels should not contain literal `%` characters to avoid collisions.

### File Format

Each stamp is a single file — hash and input list combined:

```
sha256:a1b2c3d4e5f6...
src/main.c
src/uart.c
include/config.h
```

Line 1 is the content hash (prefixed with algorithm). Remaining lines are the
input file list, one per line, sorted. Directories are expanded to their
constituent files.

**Why one file, not two?** Atomicity. The stamp is written atomically
(write to temp, rename into place). With two files (.hash + .deps), a crash
between writes leaves inconsistent state. One file = one atomic unit.

Add `.stamps/` to `.gitignore` — these are local build state, not
version-controlled artifacts.

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

On first run, no stamp file exists at `.stamps/data%app.db`, so `dk-ifchange`
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
| Hash algorithm selection (`--algo blake3`) | redo-c uses SHA256, goredo uses BLAKE2b |

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
sha256:abc123...
src/main.c
src/uart.c

# .stamps/release.tar.gz — depends on firmware.bin (also a label)
sha256:def456...
firmware.bin
config.json
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
| Change detection       | Content hash (SHA1/SHA256/BLAKE2) | Content hash (SHA256)                |
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
