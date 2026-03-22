# dk-redo

Content-hash change detection for [just](https://github.com/casey/just) recipes.

## Why?

`just` runs recipes but doesn't track whether inputs changed.
`make` tracks changes but uses timestamps, which break on `git checkout`, CI caches, and clock skew.

dk-redo gives just recipes **content-hash guards** — a recipe runs only when its
inputs actually change. Two commands, one pattern:

```just
set guards

build:
    ?dk-ifchange build.out src/main.c src/util.c
    gcc -o build.out src/main.c src/util.c
    dk-stamp build.out src/main.c src/util.c

clean:
    dk-always build.out
```

- `?dk-ifchange` — skip the recipe if nothing changed (exit 1 + `?` sigil = silent skip)
- `dk-stamp` — record the current state after a successful build
- `dk-always` — delete the stamp so the next run rebuilds

That's the whole idea. No `.do` scripts, no build orchestrator — your justfile
_is_ the build description.

## Install

```bash
dk-redo install /usr/local/bin
```

This copies the binary and creates symlinks (`dk-ifchange`, `dk-stamp`, etc.).
Both invocation styles work:

```bash
dk-redo ifchange label files...   # subcommand
dk-ifchange label files...        # symlink (argv[0] dispatch)
```

## How It Works

Every recipe that uses dk-redo follows the same **guard / build / stamp** pattern:

```just
set guards

# ── Generate a config file from a template ──────────────────────
generate-config:
    ?dk-ifchange output/config.json config.yaml templates/*.j2
    render-config config.yaml -o output/config.json
    dk-stamp output/config.json config.yaml templates/*.j2

# ── Compile firmware ─────────────────────────────────────────────
firmware:
    ?dk-ifchange firmware.bin src/*.c include/*.h
    arm-none-eabi-gcc -o firmware.bin src/*.c -Iinclude/
    dk-stamp firmware.bin src/*.c include/*.h

# ── Deploy only when source or config changed ───────────────────
deploy-staging:
    ?dk-ifchange deploy-staging src/*.py config/staging.yaml
    kubectl apply -f k8s/staging/
    dk-stamp deploy-staging src/*.py config/staging.yaml

# ── Force-rebuild any label ──────────────────────────────────────
clean:
    dk-always firmware.bin output/config.json deploy-staging
```

### The label

The first argument is always the **label** — a key that names this recipe's
stamp file (`.stamps/<label>`). It's usually the output filename, but any
unique string works for side-effect recipes like `deploy-staging`.

### The guard (`?`)

`?` is a just v1.47+ feature (`set guards` enables it). When a line prefixed
with `?` exits 1, just stops the recipe without error. dk-ifchange exits 1
when nothing changed — so the build step is skipped cleanly.

### What gets hashed

dk-redo uses BLAKE3 content hashes and file sizes. It detects actual content
changes, not just `mtime` bumps. Directories are hashed recursively.
Missing files are tracked too — so a recipe re-runs when a previously absent
file appears.

## Tutorial

See [`tutorial/`](tutorial/) for a hands-on walkthrough you can run locally.

## Reference

See [`dk-redo.md`](dk-redo.md) for the full specification.
