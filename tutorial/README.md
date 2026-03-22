# dk-redo Tutorial

A hands-on example that builds a small report from text files.
No compilers needed — the "build tools" are `echo`, `date`, and `cat`.

## What's in this directory

```
tutorial/
├── justfile            # three recipes: header → report → final
├── config.txt          # a one-line config the header depends on
└── data/
    ├── part1.txt       # report body, part 1
    └── part2.txt       # report body, part 2
```

## The pipeline

```
config.txt ──→ header ──→ report ──→ final
                           ↑
              data/part1.txt
              data/part2.txt
```

1. **header** reads `config.txt` and writes `build/header.txt` with a timestamp.
2. **report** concatenates the header and both data files into `build/report.txt`.
3. **final** copies the report and appends a "built-at" footer → `build/final.txt`.

## Walkthrough

### 1 — First build (everything runs)

```console
$ just final
```

All three recipes run because no stamps exist yet. Inspect the output:

```console
$ cat build/final.txt
```

### 2 — Immediate re-run (nothing runs)

```console
$ just final
```

Every `?dk-ifchange` guard exits 1 — inputs haven't changed — so just
skips each recipe silently. Zero work done.

### 3 — Edit an input (only affected recipes re-run)

```console
$ echo "release-v3" > config.txt
$ just final
```

`config.txt` changed → **header** rebuilds → its output `build/header.txt`
changed → **report** rebuilds → **final** rebuilds. The data files didn't
change, but report still re-runs because one of its inputs (the header) did.

### 4 — Edit a data file (header is skipped)

```console
$ echo "Updated results." >> data/part2.txt
$ just final
```

`config.txt` is unchanged → **header is skipped**. But `data/part2.txt`
changed → **report** rebuilds → **final** rebuilds.

### 5 — Force-rebuild

```console
$ just clean
$ just final
```

`dk-always` deletes all stamps. The next `just final` rebuilds everything
from scratch, same as step 1.

## How each recipe works

Take the **header** recipe:

```just
header:
    ?dk-ifchange build/header.txt config.txt
    mkdir -p build
    echo "# Generated $(date ...) ..." > build/header.txt
    dk-stamp build/header.txt config.txt
```

| Line | What it does |
|------|-------------|
| `?dk-ifchange build/header.txt config.txt` | Guard: compare `config.txt` against the stamp for label `build/header.txt`. If unchanged → exit 1 → `?` stops the recipe (no error). If changed (or first run) → exit 0 → recipe continues. |
| `mkdir -p build` | Ensure the output directory exists. |
| `echo ... > build/header.txt` | The actual "build step" — writes the output file. |
| `dk-stamp build/header.txt config.txt` | Record: snapshot the current hash of `config.txt` into `.stamps/build%2Fheader.txt`. Next run's guard will compare against this. |

The label `build/header.txt` ties the guard and stamp together. It's stored
as `.stamps/build%2Fheader.txt` (slashes are percent-encoded).

## Key takeaways

- **guard / build / stamp** — every recipe follows this three-line pattern.
- The **label** (first arg) is just a name — usually the output path.
- `?` (just v1.47+, `set guards`) makes exit-1 a clean skip, not an error.
- dk-redo hashes **content**, not timestamps — `touch` won't cause a rebuild,
  but changing a single byte will.
