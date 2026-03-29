# dkredo — Future Work

Planned features beyond the current Phase 1 implementation. Phase 1 (core
operations, aliases, `--install`, input modes, `-v`, `--stamps-dir`,
`DKREDO_ARGS`, scoped flags) is complete.

## Phase 2 — Diagnostics and Query Tools

### `dkr-ood` — List out-of-date labels

Scan all stamps in `.stamps/` and report which labels would trigger a
rebuild. Useful for CI ("is anything stale?") and for understanding what
`just` will do before running it.

### `dkr-affects` — Reverse dependency query

Answer "what breaks if I change this file?" by scanning all stamps for
entries that reference a given path. Enables impact analysis before editing
shared headers or config files.

### `dkr-sources` — List all tracked input files

Union of all file names across all stamps. Shows the full set of files
dkredo is watching, useful for verifying coverage and debugging missing
dependencies.

### `dkr-dot` — Graphviz dependency graph

Export label-to-file relationships in DOT format for visualization.
Renders the project's dependency structure as a directed graph.

### `-q` (quiet) flag

Suppress all non-error output. Complement to the existing `-v` (verbose)
flag. Useful in CI scripts where only the exit code matters.

### `-n` flag — Force-changed on `+check`

Force `+check` to return exit 0 (changed) without deleting the stamp.
Triggers a rebuild without losing the recorded facts — cleaner than
`+clear-facts` when the stamp should be preserved for diagnostics.

### Glob patterns in filters

Currently filters match exact paths or `.suffix` extensions. Phase 2
adds glob pattern support (e.g., `src/**/*.c`) to `+names`, `+facts`,
`+check`, `+stamp-facts`, `+remove-names`, and `+clear-facts` filter
arguments. May delegate to external tools rather than adding a glob
library.

### `+changed-names` — List files that fail their facts

An operation that prints only the dependency files whose recorded facts
no longer match the filesystem. Intended for incremental recompilation
workflows — rebuild only the changed inputs. Needs further design work
around tracking output-to-input mappings (e.g., knowing which `.o`
corresponds to which `.c`).

### Atomic writes on Windows

The current atomic write strategy (write temp file + `rename`) relies on
POSIX rename semantics. Windows rename does not atomically replace an
existing file. Investigate `MoveFileEx` with `MOVEFILE_REPLACE_EXISTING`
or similar approaches.

## Phase 3 — Directory Listing and Transitive Dependencies

### `+add-dir-listing` — Directory membership tracking

Track a directory's sorted filename list as a dependency. Detects file
additions and deletions within a directory without hashing file contents
(individual file entries handle content changes). Behaviorally equivalent
to hashing `ls | sort`.

This fills the gap where `-@ <(find ...)` scans directory contents but
cannot detect that a file was *removed* from a directory between runs
without re-running the external tool.

### Transitive dependency tracking

Redo's signature feature adapted for dkredo: if label A depends on an
input that is also a tracked label B, and B's inputs change, then A is
reported as out of date — automatically.

The stamp format already stores inputs. The missing piece: **labels can
appear as inputs of other labels.** With transitive tracking, `+check`
would:

1. Resolve the input list as now
2. For each input, check if a stamp exists with that name (i.e., it is
   also a dkredo label, not just a source file)
3. If so, recursively check whether *that* label is out of date
4. A label is out of date if its own inputs changed OR any of its
   label-deps are out of date

This is a **graph walk, not re-execution** — dkredo would report the
full out-of-date chain and let just's recipe dependencies handle build
ordering.

Phase 1 works correctly without transitive tracking because just's recipe
deps provide ordering and content hashing catches the changes. Transitive
tracking adds efficiency (skip hash computation early) and enables
`dkr-ood` and `dkr-affects` to report the full dependency chain.
