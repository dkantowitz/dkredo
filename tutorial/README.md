# dkredo Tutorial

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

Every `?dkr-ifchange` guard exits 1 — inputs haven't changed — so just
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

`dkr-always` clears all stamp facts. The next `just final` rebuilds everything
from scratch, same as step 1.

## How each recipe works

Take the **header** recipe:

```just
header:
    ?dkr-ifchange build/header.txt config.txt
    mkdir -p build
    echo "# Generated $(date ...) ..." > build/header.txt
    dkr-stamp build/header.txt config.txt
```

| Line                                        | What it does                                                                                                                                                                                |
| ------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `?dkr-ifchange build/header.txt config.txt` | Guard: compare `config.txt` against the stamp for label `build/header.txt`. If unchanged → exit 1 → `?` stops the recipe (no error). If changed (or first run) → exit 0 → recipe continues. |
| `mkdir -p build`                            | Ensure the output directory exists.                                                                                                                                                         |
| `echo ... > build/header.txt`               | The actual "build step" — writes the output file.                                                                                                                                           |
| `dkr-stamp build/header.txt config.txt`     | Record: snapshot the current hash of `config.txt` into `.stamps/build%2Fheader.txt`. Next run's guard will compare against this.                                                            |

The label `build/header.txt` ties the guard and stamp together. It's stored
as `.stamps/build%2Fheader.txt` (slashes are percent-encoded).

## Using +operation syntax

The alias commands (`dkr-ifchange`, `dkr-stamp`, `dkr-always`) are shortcuts
for dkredo's composable `+operation` primitives. Here's the same header recipe
written with `+operations` directly:

```just
header:
    ?dkredo build/header.txt +add-names config.txt +check
    mkdir -p build
    echo "# Generated $(date ...) ..." > build/header.txt
    dkredo build/header.txt +remove-names +add-names config.txt +stamp-facts
```

| Alias style                                 | +operation style                                                                 |
| ------------------------------------------- | -------------------------------------------------------------------------------- |
| `?dkr-ifchange build/header.txt config.txt` | `?dkredo build/header.txt +add-names config.txt +check`                          |
| `dkr-stamp build/header.txt config.txt`     | `dkredo build/header.txt +remove-names +add-names config.txt +stamp-facts`       |
| `dkr-always build/header.txt`               | `dkredo build/header.txt +clear-facts`                                           |

The `+operation` style gives finer control — for example, returning only file names
that exist on disk (`-e`) _and_ in the stamp for use in a build command:

```just
compile:
    ?dkredo out.bin +add-names $(find src -name '*.c') +check
    gcc -o out.bin -MMD -MF .deps/out.d $(dkredo out.bin +names -e .c)
    dkredo out.bin +add-names -M .deps/out.d +stamp-facts
```

See [`dkredo.md`](../dkredo.md) for the full list of operations.

See [`demo-primitives.just`](demo-primitives.just) for more +operation examples
including C compilation with gcc `-MD`, multi-target builds, lint, docker, and
test recipes.

## Key takeaways

- **guard / build / stamp** — every recipe follows this three-line pattern.
- The **label** (first arg) is just a name — usually the output path.
- `?` (just v1.47+, `set guards`) makes exit-1 a clean skip, not an error.
- dkredo hashes **content**, not timestamps — `touch` won't cause a rebuild,
  but changing a single byte will.
- Use **aliases** (`dkr-ifchange`, `dkr-stamp`) for simple cases,
  **+operations** (`dkredo label +add-names ... +check`) when you need
  finer control.
