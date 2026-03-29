# dkredo — File-Dependency Guards for Justfiles

`dkredo` brings redo-style content-hash change detection into `just` recipes
using the `?` guard sigil (just v1.47+). One system, no separate build tool.

## Philosophy

`just` is the command runner.

`dkredo` is the minimalist dependency-tracking.

Together they make a simple makefile replacement that can be mixed into a justfile.

**Not a fork of redo.** dkredo borrows redo's core insight (content hashing
beats timestamps) and its naming conventions, but does not implement .do
scripts, automatic transitive rebuilds, or a build orchestrator. The justfile
_is_ your build description.

**Not a fork of just.** dkredo is used with just, but the only integration point is the '?' sigil.

## Quick Start

```just
set guards

firmware:
    ?dkr-ifchange firmware.bin src/*.c include/*.h
    arm-none-eabi-gcc -o firmware.bin src/*.c -Iinclude/
    dkr-stamp firmware.bin src/*.c include/*.h

clean:
    dkr-always firmware.bin
```

`set guards` is required in the justfile for the `?` sigil to work.
All justfile examples in this document assume it is set.

How it reads:

- `?dkr-ifchange firmware.bin ...` — "if inputs to label `firmware.bin` haven't
  changed, skip this recipe"
- `dkr-stamp firmware.bin ...` — "record current input state for `firmware.bin`"
- The `?` sigil (just v1.47+) stops the recipe cleanly on exit 1 — no error,
  other recipes continue

### operation style

The commands like `dkr-ifchange` are strung together from smaller operations to create a style reminiscent of the `redo` build system. These internal operations can be used very conveniently with the `+operation` style arguments.

```
init-db:
    ?dkredo app.db +add-names schema.sql +check
    sqlite3 app.db < schema.sql
    dkredo app.db +stamp-facts

clean:
    dkredo firmware.bin +clear-facts
    dkredo app.db +clear-facts
```

## Installation

dkredo is a single binary. Symlinks provide the short command names, or use
`--cmd` to invoke them without symlinks.

```bash
# install binary and create symlinks
dkredo --install /usr/local/bin

# or manually:
cp dkredo /usr/local/bin/dkredo
chmod +x /usr/local/bin/dkredo
cd /usr/local/bin
for cmd in dkr-ifchange dkr-stamp dkr-always dkr-fnames; do
    ln -sf dkredo "$cmd"
done
```

`dkredo --install` copies the binary to the destination directory and
creates all symlinks.

Three equivalent invocation styles:

```bash
dkr-ifchange firmware.bin src/*.c               # symlink (argv[0] dispatch)
dkredo firmware.bin --cmd ifchange src/*.c      # no symlink needed
dkredo firmware.bin +add-names src/*.c +check   # explicit operations
```

`--cmd` expands a named alias into its operation sequence, so
`--cmd ifchange` becomes `+add-names ... +check`. This avoids
needing symlinks while keeping the familiar command names.

## The Label

The label is always the first positional argument — no ambiguity with operation
names.

The **label** is a unique key that identifies a set of dependencies. It
maps directly to a stamp file: `.stamps/<label>`.

dkredo does not interpret the label. It never checks whether the label refers
to an existing file, never hashes it, never builds it. It is purely a namespace
key for the stamp.

### Label examples

**Output file path** — the natural default. Unique per recipe, makes
diagnostic output self-explanatory:

```just
firmware:
    ?dkr-ifchange firmware.bin src/*.c include/*.h
    gcc -o firmware.bin src/*.c -Iinclude/
    dkr-stamp firmware.bin src/*.c include/*.h
```

**Output with path** — labels can include directories. `/` is escaped to `%2F`
in the stamp filename:

```just
generate-config:
    ?dkr-ifchange output/config.json config.yaml templates/*.j2
    render-config config.yaml -o output/config.json
    dkr-stamp output/config.json config.yaml templates/*.j2
```

Stamp file: `.stamps/output%2Fconfig.json`

**Descriptive label** — for recipes that produce side effects, not files.
Any unique string works:

```just
deploy-staging:
    ?dkr-ifchange deploy-staging src/*.py config/staging.yaml
    kubectl apply -f k8s/staging/
    dkr-stamp deploy-staging src/*.py config/staging.yaml

run-migrations:
    ?dkr-ifchange run-migrations migrations/*.sql
    psql -f migrations/apply.sh
    dkr-stamp run-migrations migrations/*.sql
```

**Parameterized label** — incorporates the recipe argument so each
invocation gets its own stamp:

```just
compile target:
    ?dkr-ifchange {{target}} src/{{target}}.c include/*.h
    gcc -o build/{{target}} src/{{target}}.c
    dkr-stamp {{target}} src/{{target}}.c include/*.h
```

`just compile foo` stamps `.stamps/foo`, `just compile bar` stamps
`.stamps/bar` — independent tracking.

**Directory as label** — when the output is a directory. Use `-@` with
process substitution to list the input files:

```just
assets:
    ?dkredo dist/assets +add-names -@ <(fd -t f static/images static/fonts) +check
    optimize-assets -o dist/assets/
    dkredo dist/assets +remove-names +add-names -@ <(fd -t f static/images static/fonts) +stamp-facts
```

Stamp file: `.stamps/dist%2Fassets`

**Why a label is required:** Without it, there is no way to namespace stamps
per recipe. If two recipes depend on the same file (e.g., `config.json`),
whichever recipe runs first would stamp the file, and the second recipe would
see "unchanged" and skip — even though it never built against the current
content. The label isolates each recipe's "last seen" state.

## Operational Model

dkredo is built around **primitive operations** that compose via `+operation`
arguments in a single invocation. The familiar command names (`dkr-ifchange`,
`dkr-stamp`, `dkr-always`) are built-in aliases for common operation sequences.

```
dkredo <label> [+operation [args...]]...
```

Operations execute left to right, threading state through a single stamp file.
Arguments between `+operation` markers belong to the operation on the left.

### Why composable operations?

The alias commands (`dkr-ifchange`, `dkr-stamp`) work for common cases but
become awkward when recipes need finer control — e.g., querying the stamp's
file list for use in a build command, or adding new dependencies without
re-hashing existing ones. Primitive operations provide that control while
keeping the simple aliases for everyday use.

## Commands Reference

### Primitive Operations

#### Stamp Manipulation

| Operation       | Args              | Description                                                                                                                                                         |
| --------------- | ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `+add-names`    | `file...`         | Add files to stamp's name list. No facts computed. New entries have an empty fact list. Existing or duplicate entries are untouched.                                |
| `+add-names`    | `-M file...`      | Parse makefile dep format, add extracted paths to the name list.                                                                                                    |
| `+remove-names` | `[filter...]`     | Remove files from stamp's name list along with their facts. Empty filter matches every entry.                                                                       |
| `+remove-names` | `-ne [filter...]` | Iff the filename does not exist and the stamp fact for that file is not `missing:true`, remove it from stamp's name list along with its facts.                                                                                   |
| `+stamp-facts`  | `[filter...]`     | Compute and record facts (blake3, size, missing) for the selected file names. If empty filter, re-calculate facts for all files currently in the stamp's name list. Does not add names — use `+add-names` first. |
| `+clear-facts`  | `[filter...]`     | Remove facts from filtered file names, but leave the filename in the stamp.                                                                                         |

#### Querying

| Operation | Args             | Description                                                                                              |
| --------- | ---------------- | -------------------------------------------------------------------------------------------------------- |
| `+names`  | `[filter...]`    | Print file names from the stamp to stdout. Optional filter is an extension (`.c`, `.h`) or glob pattern. |
| `+names`  | `-e [filter...]` | Print only file names that exist on disk.                                                                |
| `+facts`  | `[filter...]`    | Print recorded facts for the given files (or all files). Diagnostic output.                              |

#### Testing

| Operation       | Args          | Description                                                                                                                                |
| --------------- | ------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| `+check`        | `[filter...]` | Compare stamp facts against current filesystem. Exit 0 if any fact fails (changed) or if any entry has no facts. Exit 1 if all facts hold (unchanged). An empty stamp (no entries) passes. Exit 2 on error. |
| `+check-assert` | `[filter...]` | Like `+check` but exit 2 (error) instead of 1 when unchanged. For scripts that should never be called on an up-to-date target.             |

#### Files and Filters

| Modifier       | Description                                                                            |
| -------------- | -------------------------------------------------------------------------------------- |
| `file/path.x`  | Matches the exact path.                                                                |
| `.suffix`      | Filter matches all files with that suffix. Only available where argument is `filters`. |
| `-`            | Read file list from stdin (newline-terminated), pass to current operation.             |
| `-0`           | Read file list from stdin (null-terminated), pass to current operation.                |
| `-@ file`      | Read file names from `file` (newline-terminated). Works with process substitution: `-@ <(fd -e c src)`. |
| `-@0 file`     | Read file names from `file` (null-terminated).                                         |
| `-M file.d`    | Parse makefile dep format (gcc `-MD` output), extract paths. An input mode available on any operation that accepts file arguments (like `-`, `-0`, `-@`, `-@0`). |

Note: `filter...` is strictly a superset of `file...`

### Operation Examples

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

# Clear facts — the "always" pattern
dkredo out.bin +clear-facts

# Add names from a file listing (process substitution)
dkredo out.bin +add-names -@ <(fd -e c src) +check

# Add names from a file listing (null-terminated)
dkredo out.bin +add-names -@0 <(fd -0 -e c src) +check

# Print facts for debugging
dkredo out.bin +facts

# Use an alias without symlinks (--cmd)
dkredo out.bin --cmd ifchange src/main.c src/util.c
dkredo out.bin --cmd stamp -M .deps/out.d
```

### Operation Sequencing

Operations execute left to right. Each operation may read or modify the
stamp. The exit code comes from the last operation that produces one
(typically `+check`).

Operations that produce an exit code (`+check`) stop the pipeline if
non-zero. Operations that write to stdout (`+names`, `+facts`) do not
affect the exit code.

**Special case for +check:** `+check` returning exit 1 (unchanged) stops
the pipeline but does NOT prevent writing pending stamp modifications.
The `+add-names` from earlier in the pipeline must persist so that the
next run's `+check` includes the new entries.

### Built-in Aliases

The familiar command names map to operation sequences. These are compiled into
the binary and dispatched via argv[0] (symlinks) or `--cmd`.

| Alias                                   | Equivalent                                    |
| --------------------------------------- | --------------------------------------------- |
| `dkr-ifchange <label> [files...]`       | `dkredo <label> +add-names [files...] +check` |
| `dkr-ifchange <label>` (no files)       | `dkredo <label> +check`                       |
| `dkr-stamp <label> [files...]`          | `dkredo <label> +remove-names +add-names [files...] +stamp-facts` |
| `dkr-stamp --append <label> [files...]` | `dkredo <label> +add-names [files...] +stamp-facts` |
| `dkr-stamp --append <label> -M file.d` | `dkredo <label> +add-names -M file.d +stamp-facts` |
| `dkr-stamp <label> -M file.d`           | `dkredo <label> +remove-names +add-names -M file.d +stamp-facts` |
| `dkr-always <label>`                    | `dkredo <label> +clear-facts`                 |
| `dkr-fnames <label> [filter]`           | `dkredo <label> +names -e [filter]`           |

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

#### Alias note: dkr-ifchange union behavior

`dkr-ifchange <label> files...` maps to `+add-names files... +check` rather
than a sync operation. This preserves the union-with-stamp behavior: new files
are added, but previously-discovered dependencies (e.g., headers from a prior
`-M` depfile) remain in the stamp and are still checked.

If you want strict "only these files" semantics, add `+remove-names -ne`:

```bash
dkredo out.bin +add-names $(find src -name '*.c') +remove-names -ne +check
```

### Exit Codes

- `0` — action taken / change detected
- `1` — no action needed / unchanged (only from `+check`)
- `2` — error

The `?` sigil only intercepts exit code 1. Exit code 2 propagates as a
recipe failure, which is the correct behavior — errors should not be silent.

Operations that don't produce a meaningful exit code (e.g., `+add-names`,
`+stamp-facts`, `+names`) exit 0 on success and 2 on error.

## Input Modes

All operations that accept file arguments support multiple input modes.
They can be combined freely.

### 1. Positional arguments (shell-expanded)

```bash
dkr-ifchange firmware.bin src/*.c include/*.h
#             ^^^^^^^^^^^^ ^^^^^^^ ^^^^^^^^^^^
#             label        shell expands these before dkredo sees them
```

The shell expands globs. dkredo receives concrete file paths.

### 2. File input (`-@`, `-@0`)

`-@` and `-@0` read file names from a named file rather than stdin. They
work with process substitution, making them the recommended way to feed
external tool output into dkredo:

```bash
# newline-terminated file input (works with process substitution)
dkredo dist/assets +add-names -@ <(fd -e png -e jpg static/images) +check

# null-terminated file input
dkredo dist/assets +add-names -@0 <(fd -0 -e png static/images) +check

# read from an actual file
dkredo lint-check +add-names -@ filelist.txt +check
```

**Why `-@` over stdin pipes?** With a stdin pipe (`fd ... | dkredo ... -`),
if `fd` fails, dkredo sees empty stdin and silently proceeds with zero files.
With `-@ <(fd ...)`, a failed process substitution produces a read error
that dkredo can detect and report.

### 3. Stdin — newline or null-terminated

```bash
# newline-terminated (default stdin mode)
find src -name '*.c' -newer baseline | dkr-ifchange firmware.bin -

# null-terminated (for filenames with spaces/newlines)
fd -0 -e h include | dkr-ifchange firmware.bin -0

# combine with positional args
fd -e j2 templates | dkr-ifchange output-config config.yaml -
```

`-` reads newline-terminated lines from stdin. `-0` reads null-terminated.
These appear as positional arguments and can be mixed with file args. **Stdin paths are spliced into the argument list at the position where
`-` or `-0` appears.** For example:

```bash
dkr-ifchange label blah.h - bar.h
```

This processes `blah.h`, then all paths read from stdin (in order), then
`bar.h`. The final list is sorted and deduplicated, so the positional
ordering affects only how inputs are gathered, not the stamp content.

## Stamp Storage

All state lives in a single `.stamps/` directory. dkredo locates it by
searching upward from the current working directory. If no `.stamps/`
directory is found, dkredo creates one in the current working directory
on the first stamp write.

All file paths in stamps are stored relative to the `.stamps/` directory's
parent (the project root). This means the same file referenced from
different working directories resolves to the same stamp entry.

```
.stamps/
  firmware.bin           # stamp for label "firmware.bin"
  deploy-staging         # stamp for label "deploy-staging"
  output%2Fconfig.json   # stamp for label "output/config.json" (/ escaped as %2F)
```

Labels are escaped for use as flat filenames using **percent-encoding**
(the same scheme as URL encoding):

| Character | Encoded | Why                        |
| --------- | ------- | -------------------------- |
| `/`       | `%2F`   | Cannot appear in filenames |
| `%`       | `%25`   | Escape character itself    |

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

| Character      | Encoded | Why                             |
| -------------- | ------- | ------------------------------- |
| `\t` (tab)     | `%09`   | Tab is the path/facts delimiter |
| `\n` (newline) | `%0A`   | Newline is the line delimiter   |
| `%`            | `%25`   | Escape character itself         |

All other characters (including spaces) are stored verbatim. In practice
this encoding almost never activates — tabs and newlines in filenames are
vanishingly rare — but when it does, errors are clear rather than mysterious.

Lines are sorted by path. Facts are space-separated `key:value` pairs after
the tab. Defined facts:

| Fact      | Value              | When recorded                    |
| --------- | ------------------ | -------------------------------- |
| `blake3`  | hex digest         | Always (for existing files)      |
| `size`    | decimal byte count | Always (for existing files)      |
| `missing` | `true`             | File did not exist at stamp time |

A file is **changed** if any recorded fact is no longer true. `size` is
checked first as a fast path — if size differs, the hash is not recomputed
(a `stat()` call is far cheaper than reading + hashing the file).

A missing file records only `missing:true` (no hash or size). When the
file is later created, the `missing:true` fact becomes false, triggering
a rebuild.

**Unknown fact keys:** if a stamp contains fact keys not recognized by the
current version of dkredo, the file is treated as **changed** (not a match).
The reasoning: if we cannot verify all recorded facts, we cannot confirm the
file is unchanged. When the label is rebuilt, `+stamp-facts` writes a new stamp
with only the facts known to the current version, eliminating the unknown
keys. A warning is issued to stderr when unknown facts are encountered:
`warning: <label>: unknown fact key "<key>" in stamp — treating as changed`.

**Why one file, not two?** Atomicity. The stamp is written atomically
(write to temp, rename into place). With two files (.hash + .deps), a crash
between writes leaves inconsistent state. One file = one atomic unit.

Add `.stamps/` to `.gitignore` — these are local build state, not
version-controlled artifacts.

### Hashing Specification

dkredo uses **BLAKE3** for all content hashes. BLAKE3 is chosen for speed
and collision resistance, not cryptographic security — we need fast,
deterministic, unique-enough digests for change detection.

- Per-file digest: BLAKE3 over raw file bytes, 256-bit (64 hex chars)
- Per-file facts: `blake3:<hex> size:<bytes>` (always both), or `missing:true`
- Change detection: any fact that no longer holds means the file changed

The goal is deterministic results across machines for the same workspace
content and input set. See [`dkredo-implementation.md`](dkredo-implementation.md)
for pseudocode of the input resolution, fact computation, and change
detection algorithms.

## Canonical Usage Patterns

### C compilation with gcc dependency discovery

Using `+operation` syntax:

```just
set guards

compile:
    ?dkredo out.bin +add-names $(find src -name '*.c') +check
    gcc -o out.bin -MMD -MF .deps/out.d $(dkredo out.bin +names -e .c)
    dkredo out.bin +add-names -M .deps/out.d +stamp-facts
```

Using backward-compatible aliases:

```just
set guards

compile:
    ?dkr-ifchange out.bin $(find src -name '*.c')
    gcc -o out.bin -MMD -MF .deps/out.d $(dkr-fnames out.bin .c)
    dkr-stamp out.bin -M .deps/out.d
```

**Line 1:** Add any new `.c` files to the stamp (existing entries and their
facts are preserved, new entries have no facts and fail the +check). Then check
all facts. If all facts match, `?` stops the recipe.

**Line 2:** Query the stamp for `.c` files to pass to gcc. The stamp
contains both `.c` and `.h` files (from the previous build's depfile),
but only `.c` files are needed on the command line. Deleted .c files are
weeded out by the `-e` option. gcc writes its dependency discovery to
`.deps/out.d`.

**Line 3:** Add names from the depfile (both `.c` files and all `#include`d
headers), then stamp facts for all names in the stamp. Next run, the `+check`
will detect changes to any of them.

### Basic: file dependencies

```just
set guards

firmware:
    ?dkr-ifchange firmware.bin src/*.c include/*.h
    arm-none-eabi-gcc -o firmware.bin src/*.c -Iinclude/
    dkr-stamp firmware.bin src/*.c include/*.h
```

Or with `+operation` syntax:

```just
firmware:
    ?dkredo firmware.bin +add-names src/*.c include/*.h +check
    arm-none-eabi-gcc -o firmware.bin src/*.c -Iinclude/
    dkredo firmware.bin +remove-names +add-names src/*.c include/*.h +stamp-facts
```

### Directory-level dependencies via external tools

dkredo delegates directory scanning to external tools. Use `-@` with process
substitution to feed file lists from `find`, `fd`, or `git ls-files`:

```just
assets:
    ?dkredo dist/assets +add-names -@ <(find static/images static/fonts -type f) +check
    optimize-assets -o dist/assets/
    dkredo dist/assets +remove-names +add-names -@ <(find static/images static/fonts -type f) +stamp-facts
```

### Complex deps via fd/find

```just
engine:
    ?dkredo target/release/engine +add-names -@ <(fd -e rs src) +check
    cargo build --release
    dkredo target/release/engine +remove-names +add-names -@ <(fd -e rs src) +stamp-facts
```

The `-@` approach avoids the `[script]` recipe requirement that stdin pipes
need, and provides better error propagation: if `fd` fails, the process
substitution produces a read error rather than silently providing empty input.

Stdin pipes still work when preferred:

```just
[script('bash')]
engine:
    rc=0
    fd -e rs src | dkr-ifchange target/release/engine - || rc=$?
    if [ "$rc" -eq 1 ]; then exit 0; fi    # unchanged — skip
    if [ "$rc" -ne 0 ]; then exit "$rc"; fi # error — propagate
    cargo build --release
    fd -e rs src | dkr-stamp target/release/engine -
```

> Note: stdin mode requires a `[script]` recipe (single shell process) so
> the pipe works. The explicit exit-code check replaces the `?` sigil
> (which operates on individual lines, not piped commands) and correctly
> distinguishes "unchanged" (exit 1) from errors (exit 2+).

### Bootstrapping (file doesn't exist yet)

```just
init-db:
    ?dkr-ifchange data/app.db schema.sql
    sqlite3 data/app.db < schema.sql
    dkr-stamp data/app.db schema.sql
```

On first run, no stamp file exists at `.stamps/data%2Fapp.db`, so `+check`
returns 0 (changed) and the recipe runs. `+stamp-facts` then records the current
hash of `schema.sql`. Subsequent runs skip unless `schema.sql` changes.

### Side-effect recipes (no output file)

```just
deploy-staging:
    ?dkr-ifchange deploy-staging src/*.py config/staging.yaml
    kubectl apply -f k8s/staging/
    dkr-stamp deploy-staging src/*.py config/staging.yaml

run-migrations:
    ?dkr-ifchange run-migrations migrations/*.sql
    psql -f migrations/apply.sh
    dkr-stamp run-migrations migrations/*.sql
```

### Multi-phase build

```just
firmware:
    ?dkredo firmware.bin +add-names src/*.c include/*.h libs/*.a +check
    gcc -MMD -MF .deps/firmware.d -c src/*.c -Iinclude/
    ld -o firmware.bin *.o libs/*.a
    dkredo firmware.bin +add-names libs/*.a -M .deps/firmware.d +stamp-facts
```

### Deploy (side-effect recipe with +operations)

```just
deploy-staging:
    ?dkredo deploy-staging +add-names src/*.py config/staging.yaml +remove-names -ne +check
    kubectl apply -f k8s/staging/
    dkredo deploy-staging +stamp-facts
```

### Detecting recipe or compiler flag changes

Build outputs depend on more than source files — compiler version, flags, and
the recipe itself are implicit inputs. Two approaches:

**Coarse: track the justfile itself.** Any edit to any recipe triggers all
guarded recipes. Simple, zero overhead, good for small projects:

```just
firmware:
    ?dkr-ifchange firmware.bin justfile src/*.c include/*.h
    {{CC}} {{CFLAGS}} -o firmware.bin src/*.c -Iinclude/
    dkr-stamp firmware.bin justfile src/*.c include/*.h
```

**Precise: capture flags in a file, track it as an input.** A dedicated
recipe writes the current compiler identity and flags to a file. Build
recipes depend on it via just's recipe dependencies — the flags file is
always written (cheap), and `+check` detects when the content changes.
This works because dkredo uses content hashing, not timestamps — rewriting
a file with identical content produces the same hash.

```just
CC := "arm-none-eabi-gcc"
CFLAGS := "-O2 -DNDEBUG -Iinclude/"

cc_cflags:
    @mkdir -p .deps
    @echo '{{CC}} {{CFLAGS}}' > .deps/cc_cflags

firmware: cc_cflags
    ?dkr-ifchange firmware.bin .deps/cc_cflags src/*.c include/*.h
    {{CC}} {{CFLAGS}} -o firmware.bin src/*.c
    dkr-stamp firmware.bin .deps/cc_cflags src/*.c include/*.h

release: cc_cflags firmware
    ?dkr-ifchange release.tar.gz .deps/cc_cflags firmware.bin config.json
    package-release firmware.bin config.json -o release.tar.gz
    dkr-stamp release.tar.gz .deps/cc_cflags firmware.bin config.json
```

`cc_cflags` runs unconditionally (no `?` guard) — it's the truth source
for the current toolchain. Multiple recipes can depend on it. Change the
compiler or flip an optimization flag, and every downstream recipe rebuilds.

The pattern generalizes to any tool: `python_version`, `node_env`,
`docker_tag`, etc.

### Using gcc -MD dependency output

Combine gcc's discovered headers with explicit file tracking for full coverage.
On first run, the dep file doesn't exist yet — use `touch` to create an
empty one so the stdin redirect doesn't fail:

```just
firmware: cc_cflags
    @mkdir -p .deps && touch .deps/firmware.d
    ?dkr-ifchange firmware.bin .deps/cc_cflags - < .deps/firmware.d
    gcc -MD -MF .deps/firmware.d -o firmware.bin src/*.c -Iinclude/
    dkr-stamp firmware.bin .deps/cc_cflags - < .deps/firmware.d
```

The dep file provides precise per-header tracking including system headers.

To also catch new files appearing in include directories (the negative-dependency
gap that gcc's dep output misses — see
[Design Decisions](#no-negative-dependencies-and-why-thats-ok)), use `-@`
with process substitution to scan the include path:

```just
firmware: cc_cflags
    @mkdir -p .deps && touch .deps/firmware.d
    ?dkredo firmware.bin +add-names .deps/cc_cflags -M .deps/firmware.d -@ <(find include -type f) +check
    gcc -MD -MF .deps/firmware.d -o firmware.bin src/*.c -Iinclude/
    dkredo firmware.bin +remove-names +add-names .deps/cc_cflags -M .deps/firmware.d -@ <(find include -type f) +stamp-facts
```

### Force-rebuild

Both styles are equivalent; use whichever matches your justfile:

```just
# using aliases (concise, handles multiple labels)
clean:
    dkr-always firmware.bin output/config.json deploy-staging

# using +operations (explicit)
clean:
    dkredo firmware.bin +clear-facts
    dkredo deploy-staging +clear-facts
```

### Recipe that chains outputs

```just
all: firmware release

firmware:
    ?dkr-ifchange firmware.bin src/*.c include/*.h
    gcc -o firmware.bin src/*.c -Iinclude/
    dkr-stamp firmware.bin src/*.c include/*.h

release: firmware
    ?dkr-ifchange release.tar.gz firmware.bin config.json
    package-release firmware.bin config.json -o release.tar.gz
    dkr-stamp release.tar.gz firmware.bin config.json
```

Just's recipe dependencies handle ordering. `release` runs after `firmware`.
Each recipe independently decides whether to skip via its own guard.

### Parameterized recipes

```just
compile target:
    ?dkr-ifchange {{target}} src/{{target}}.c include/*.h
    gcc -o build/{{target}} src/{{target}}.c
    dkr-stamp {{target}} src/{{target}}.c include/*.h
```

The label incorporates the parameter, so `just compile foo` and
`just compile bar` get independent stamps.

## Implementation Phases

### Phase 1 — Core (current)

| Feature                                                                 | Status |
| ----------------------------------------------------------------------- | ------ |
| Content-hash change detection (BLAKE3)                                  | Core   |
| `+add-names`, `+remove-names`, `+stamp-facts`, `+clear-facts`            | Core   |
| `+names`, `+facts`, `+check`, `+check-assert`                           | Core   |
| Alias commands: `dkr-ifchange`, `dkr-stamp`, `dkr-always`, `dkr-fnames` | Core   |
| `--cmd` alias dispatch (symlink-free)                                   | Core   |
| `--install` (binary + symlink setup)                                    | Core   |
| File, stdin, and file-input (`-@`, `-@0`) modes                        | Core   |
| Makefile depfile parsing (`-M`)                                         | Core   |
| Atomic stamp writes                                                     | Core   |

### Phase 2 — Diagnostic Commands

| Feature                            | Description                                                     |
| ---------------------------------- | --------------------------------------------------------------- |
| `dkr-ood`                          | List out-of-date labels                                         |
| `dkr-affects`                      | Reverse dependency query ("what breaks if I change this file?") |
| `dkr-sources`                      | List all tracked input files                                    |
| `dkr-dot`                          | Dependency graph in Graphviz DOT format                         |
| `-v` (verbose), `-q` (quiet) flags | Universal on all commands                                       |
| `-n` force-changed on +check       | Force rebuild without deleting stamp                            |
| `--stamps-dir` override            | Custom stamp directory location                                 |

### Phase 3 — Directory Listing and Transitive Dependencies

| Feature | Description |
| ------- | ----------- |
| `+add-dir-listing` (tentative) | Track a directory's sorted filename list as a dependency. Detects file additions and deletions within a directory (not content changes -- individual file entries handle that). Behaviorally equivalent to hashing `ls \| sort`. |

### Phase 3 — Transitive Dependency Tracking

Redo's signature feature: if label A depends on an input that is also a
tracked label B, and B's inputs change, then A is also out of date —
automatically.

The stamp format already stores inputs. The missing piece: **labels can appear
as inputs of other labels.** For phase 3, `+check` would:

1. Resolve the input list as now
2. For each input, check if a stamp exists with that name (i.e., it's also a
   dkredo label, not just a source file)
3. If so, recursively check whether _that_ label is out of date
4. A label is out of date if its own inputs changed OR any of its
   label-deps are out of date

**Graph walk, not re-execution.** Unlike full redo, dkredo would not
automatically rebuild transitive deps — it would report the full out-of-date
chain and let just's recipe dependencies handle ordering.

**Phase 1 works correctly without transitive tracking** — just's recipe deps
provide ordering, and content hashing catches the changes. Transitive tracking
adds efficiency (skip the hash computation early) and enables `dkr-ood` and
`dkr-affects` to report the full chain.

## Argument Summary

### Arguments (aliases)

| Argument   | Type               | Description                                             |
| ---------- | ------------------ | ------------------------------------------------------- |
| `<label>`  | positional (first) | Unique key for this stamp (see [The Label](#the-label)) |
| `<file>`   | positional         | Input file path                                         |
| `-`        | positional         | Read input list from stdin, newline-terminated          |
| `-0`       | positional         | Read input list from stdin, null-terminated             |
| `-@ file`  | input mode         | Read file names from `file` (newline-terminated). Works with process substitution. |
| `-@0 file` | input mode         | Read file names from `file` (null-terminated).          |
| `-M`       | input mode         | Parse makefile dep format — available on any operation that accepts file arguments |
| `--append` | flag               | Merge into existing stamp (dkr-stamp)                   |
| `--help`   | flag               | Print full help and exit                                |
| `-h`       | flag               | Print short help and exit                               |
| `--cmd`    | flag               | Expand a named alias (see [Built-in Aliases](#built-in-aliases)) |
| `--install`| flag               | Install binary + symlinks to a directory                |

### Arguments (+operations)

Invoking `dkredo <label>` with no operations and no `--cmd` is an error —
dkredo prints help and exits with code 2.

```
dkredo <label> [+operation [args...]]...
```

Arguments between `+operation` markers belong to the operation on the left.
The parser consumes arguments greedily until the next `+` token or end of args.

## Comparison with redo

|                        | redo                              | dkredo                                  |
| ---------------------- | --------------------------------- | --------------------------------------- |
| Build description      | `.do` shell scripts               | justfile recipes                        |
| Dependency declaration | `redo-ifchange` inside .do        | `dkr-ifchange` guard line               |
| Change detection       | Content hash (SHA1/SHA256/BLAKE2) | Content hash (BLAKE3)                   |
| Transitive rebuilds    | Automatic                         | Via just recipe deps (phase 3: checked) |
| Parallel builds        | Built-in (`-j`)                   | Via just `[parallel]` attribute         |
| Task listing           | Limited (`redo-targets`)          | `just --list`                           |
| Stamp storage          | `.redo/` (SQLite or recfiles)     | `.stamps/` (plain text)                 |
| Learning curve         | Moderate (new build paradigm)     | Low (just + one convention)             |

## Design Decisions

### No negative dependencies (and why that's OK)

redo has `redo-ifcreate` — a "negative dependency" that triggers a rebuild
when a currently-absent file comes into existence. The classic case: include
path search order. If `#include <config.h>` resolves to `/usr/include/config.h`
because `./include/config.h` doesn't exist, the build implicitly depends on
that file _staying absent_. If someone creates `./include/config.h`, it
shadows the system header and the build should re-run.

dkredo does not have negative dependencies. Here's why, and what to do
instead.

**`gcc -M` / `-MD` does not catch the shadow case.** Compiler-generated dep
files record resolved paths of files that _were_ included — not paths that
were searched and missed. If `./include/config.h` appears and would shadow
`/usr/include/config.h`, the old dep file still lists the system header
(unchanged), and `+check` skips. The shadow is invisible.

**Scanning include directories catches it.** Use `-@` with `find` or `fd`
to depend on all files in include directories. A new file appearing will be
picked up on the next run:

```just
firmware:
    ?dkredo firmware.bin +add-names src/*.c -@ <(find include -type f) +check
    gcc -o firmware.bin src/*.c -Iinclude/
    dkredo firmware.bin +remove-names +add-names src/*.c -@ <(find include -type f) +stamp-facts
```

A new file in `include/` will be picked up by `find` on the next run. This
does not detect new files mid-build, but it does detect them on the next
invocation.

See [Using gcc -MD dependency output](#using-gcc--md-dependency-output) for
the full hybrid pattern combining directory scanning with per-header tracking.

### Why no directory arguments?

dkredo does not accept directory paths as dependency inputs. Scanning
subdirectories is inherently complex: depth limits, symlink following,
.gitignore awareness, dotfile handling, and cross-platform differences all
require policy decisions that a minimal tool should not bake in.

Instead, dkredo delegates directory scanning to specialized external tools
(`find`, `fd`, `git ls-files`) and reads their output via `-@` with
process substitution:

```bash
# Recommended: -@ with process substitution
dkredo dist/assets +add-names -@ <(fd -e c src) +check
dkredo lint-check +add-names -@ <(git ls-files '*.py') +check
```

**Why `-@` over stdin pipes?** With a stdin pipe (`fd ... | dkredo ... -`),
if `fd` fails, dkredo sees empty stdin and silently proceeds with zero files.
With `-@ <(fd ...)`, a failed process substitution produces a read error
that dkredo can detect and report.

### Why separate guard and stamp?

In redo, the .do script wraps the build — redo records state automatically.
In dkredo, the `?` sigil stops the recipe _before_ the build runs, so
state must be recorded _after_ the build succeeds. If the build fails,
no stamp is written, and the next run correctly rebuilds.
