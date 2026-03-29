# dkredo

Content-hash change detection for [just](https://github.com/casey/just) recipes.

## Why?

`just` runs recipes but doesn't track whether inputs changed.
`make` is fifty years old and awkward for most scripting tasks.
`redo` is simple, but it replaces the familiar makefile-style workflow.

dkredo adds redo-inspired **file content-hash guards** to justfile recipes: make-like syntax, redo-like simplicity.

```just
set guards

build:
    ?dkr-ifchange build.out src/main.c src/util.c
    gcc -o build.out src/main.c src/util.c
    dkr-stamp build.out

clean:
    dkr-always build.out
```

- `?dkr-ifchange` — skip the recipe if nothing changed (exit 1 + `?` sigil = silent skip)
- `dkr-stamp` — record the current state of inputs after a successful build
- `dkr-always` — clear the stamp's facts so the next run rebuilds

That's the whole idea. No `.do` scripts, no build orchestrator — justfile recipes _are_ the makefile.

### Composable operations

The alias commands above (`dkr-ifchange`, `dkr-stamp`, `dkr-always`) are
shortcuts for dkredo's `+operation` primitives. When you need finer control,
compose operations directly:

```just
build:
    ?dkredo build.out +add-names src/main.c src/util.c +check
    gcc -o build.out src/main.c src/util.c
    dkredo build.out +stamp-facts
```

See [`dkredo.md`](docs/dkredo.md) for the full list of operations.

## Install

```bash
dkredo --install /usr/local/bin
```

This copies the binary and creates symlinks (`dkr-ifchange`, `dkr-stamp`, etc.).
Three invocation styles:

```bash
dkr-ifchange firmware.bin src/*.c               # symlink (argv[0] dispatch)
dkredo firmware.bin --cmd ifchange src/*.c      # no symlink needed
dkredo firmware.bin +add-names src/*.c +check   # explicit operations
```

## How It Works

Every recipe that uses dkredo follows the same **guard / build / stamp** pattern:

```just
set guards

# ── Generate a config file from a template ──────────────────────
generate-config:
    ?dkr-ifchange output/config.json config.yaml templates/*.j2
    render-config config.yaml -o output/config.json
    dkr-stamp output/config.json

# ── Compile firmware (operations syntax) ───────────────────────
firmware:
    ?dkredo firmware.bin +add-names src/*.c include/*.h
    arm-none-eabi-gcc -o firmware.bin $(dkredo +names -e .c) -Iinclude/
    dkr-stamp firmware.bin

# ── Deploy only when source or config changed ───────────────────
deploy-staging:
    ?dkr-ifchange deploy-staging src/*.py config/staging.yaml
    kubectl apply -f k8s/staging/
    dkr-stamp deploy-staging

# ── Force-rebuild any label ──────────────────────────────────────
clean:
    dkr-always firmware.bin
    dkr-always output/config.json
    dkr-always deploy-staging
```

### The label

The first argument is always the **label** — a key that names this recipe's
stamp file (`.stamps/<label>`). It's usually the output filename, but any
unique string works for side-effect recipes like `deploy-staging`.

### The guard (`?`)

`?` is a just v1.47+ feature (`set guards` enables it). When a line prefixed
with `?` exits 1, just stops the recipe without error. `+check` exits 1
when nothing changed — so the build step is skipped cleanly.

### What gets hashed

dkredo uses BLAKE3 content hashes and file sizes. It detects actual content
changes, not just `mtime` bumps. Missing files are tracked too — so a recipe
re-runs when a previously absent file appears.

## Tutorial

See [`tutorial/`](docs/tutorial/) for a hands-on walkthrough you can run locally.

## Reference

See [`dkredo.md`](docs/dkredo.md) for the full specification.

## Future Work

See [`future-work.md`](docs/future-work.md) for planned Phase 2 and Phase 3 features, including diagnostic query tools (`dkr-ood`, `dkr-affects`), glob filters, and transitive dependency tracking.
