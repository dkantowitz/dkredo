# dk-redo Implementation Guide

## Language Choice: Go

**Decision:** Go with `CGO_ENABLED=0` for a static binary.

**Rationale:**

| Concern | Go | C (cosmocc) | Rust | Bash |
|---|---|---|---|---|
| Startup time | ~3-5ms | ~1ms | ~2-4ms | ~5-10ms |
| Maintenance cost | Low | High (manual string/IO) | Medium | High past ~200 lines |
| BLAKE3 library | zeebo/blake3 | need lib | blake3 crate | No (shell out) |
| File walking, stdin | stdlib | Manual | stdlib | Fragile |
| Testing | `go test`, table-driven | Poor story | Good | Very poor |
| Cross-compile | Trivial | cosmocc handles it | Needs targets | N/A |
| Concurrency (rev2) | Goroutines | pthreads | tokio/rayon | Not practical |

The ~2ms startup penalty vs C is irrelevant — hashing file contents dominates.
dk-ifchange on the "unchanged" fast path (read stamp, compare hash) is I/O
bound, not CPU bound.

## Architecture

Single binary, argv[0] dispatch (like busybox/redo-c/goredo):

```
cmd/dk-redo/main.go     — argv[0] dispatch + subcommand parsing
internal/stamp/          — stamp read/write/compare
internal/hasher/         — file/directory/stdin hashing
internal/resolve/        — input argument resolution (file vs dir vs stdin)
```

### argv[0] dispatch

```go
func main() {
    cmd := filepath.Base(os.Args[0])
    // strip dk- prefix for symlink style
    if strings.HasPrefix(cmd, "dk-") {
        cmd = strings.TrimPrefix(cmd, "dk-")
    }
    // or use first arg for subcommand style (dk-redo ifchange ...)
    if cmd == "dk-redo" || cmd == "redo" {
        if len(os.Args) < 2 { usage(); os.Exit(2) }
        cmd = os.Args[1]
        os.Args = os.Args[1:] // shift
    }
    switch cmd {
    case "ifchange": runIfchange()
    case "stamp":    runStamp()
    case "always":   runAlways()
    // rev1.x: ood, affects, dot, sources
    default:         usage(); os.Exit(2)
    }
}
```

### Core algorithm (dk-ifchange)

```
1. Parse flags, extract label (arg[0]) and inputs (arg[1:])
2. Resolve inputs: expand directories, read stdin if "-" or "-0"
3. Read stamp file (.stamps/<escaped-label>)
4. Compare file lists: if different sets of paths, exit 0 (changed)
5. For each file in stamp, check recorded facts against current state:
   - missing:true → check file still absent
   - size:<n>     → compare file size (fast path, avoids hashing)
   - blake3:<hex> → compare BLAKE3 hash
6. If any fact is false: exit 0 (changed)
7. All facts hold: exit 1 (unchanged)
8. On error: exit 2
```

### Core algorithm (dk-stamp)

```
1. Parse flags (including --append), extract label and inputs
2. Resolve inputs (same as ifchange)
3. For each resolved file: compute facts (blake3 hash + size via stat; missing:true if absent)
4. If --append: read existing stamp, merge file lists, replace facts for updated files
5. Write stamp lines (one per file, sorted by path, tab-delimited): <path>\t<facts...>
6. Write atomically: temp file + rename
```

### Atomic writes

```go
func writeStamp(path string, content []byte) error {
    tmp := path + ".tmp." + strconv.Itoa(os.Getpid())
    if err := os.WriteFile(tmp, content, 0644); err != nil {
        return err
    }
    return os.Rename(tmp, path)
}
```

### Facts for non-existent files

When a file does not exist (`os.ErrNotExist`), record `missing:true` as
its only fact — no hash or size. This enables the bootstrapping pattern:
the fact becomes false when the file is created, triggering a rebuild.

### Label escaping

```go
func escapeLabel(label string) string {
    return strings.ReplaceAll(label, "/", "%")
}
```

## Test Plan

### Unit tests (internal packages)

**hasher package:**

| Test | Input | Expected |
|---|---|---|
| HashFile with content | temp file "hello" | deterministic BLAKE3 + size |
| HashFile empty file | temp file "" | BLAKE3 of empty + size:0 |
| HashFile missing file | nonexistent path | `missing:true` fact |
| HashFile permission denied | unreadable file | error |
| HashDir empty dir | empty temp dir | deterministic hash |
| HashDir with files | dir with 3 files | hash changes when file added/removed/modified |
| HashDir determinism | same files, different creation order | same hash |
| HashDir symlink loop | dir with circular symlink | error (not infinite loop) |
| HashStdin newline | "a.c\nb.c\n" | list of 2 files |
| HashStdin null | "a.c\0b.c\0" | list of 2 files |
| HashStdin empty | "" | empty list |

**stamp package:**

| Test | Input | Expected |
|---|---|---|
| Write then read | label + file list + facts | roundtrip matches |
| Write creates .stamps/ | nonexistent .stamps dir | auto-created |
| Write is atomic | kill mid-write | no partial stamp |
| Read missing stamp | nonexistent label | "no stamp" (not error) |
| Read corrupt stamp | garbage content | error (exit 2) |
| Compare unchanged | same hash + same files | match (exit 1) |
| Compare changed hash | different blake3 fact | no match (exit 0) |
| Compare changed filelist | same facts but different files | no match (exit 0) |
| Compare size fast path | size fact differs | no match without hashing (exit 0, no file read) |
| Compare missing appeared | missing:true but file exists | no match (exit 0) |
| Append merges | existing stamp + new files | union of files |
| Append updates facts | existing file with new content | facts updated |
| Append preserves | files not in new call | preserved in stamp |
| Tab-delimited roundtrip | path with spaces | parsed correctly |
| Label escaping | "output/config.json" | ".stamps/output%config.json" |
| Label with special chars | "foo bar" | handled correctly |
| Unknown facts ignored | line with extra key:val | no error, unknown facts skipped |

**resolve package:**

| Test | Input | Expected |
|---|---|---|
| File args | ["src/a.c", "src/b.c"] | two file paths |
| Dir arg | ["src/"] | all files under src/, sorted |
| Mixed | ["a.c", "src/", "b.c"] | files + expanded dir |
| Stdin newline | stdin="x.c\ny.c\n", args=["-"] | two files from stdin |
| Stdin null | stdin="x.c\0y.c\0", args=["-0"] | two files from stdin |
| Mixed with stdin | ["a.c", "-", "b.c"] | positional + stdin merged |
| No stdin when tty | ["-"] but stdin is tty | error |

### Integration tests (full binary)

These test the actual binary with real files on disk.

| Test | Scenario | Expected |
|---|---|---|
| First run | no stamps, run dk-ifchange | exit 0 |
| Unchanged | stamp, no file changes, run dk-ifchange | exit 1 |
| File modified | stamp, modify a file, run dk-ifchange | exit 0 |
| File added to glob | stamp, create new .c file matching glob | exit 0 |
| File removed from glob | stamp, delete a .c file | exit 0 |
| Dir file added | stamp with dir arg, add file to dir | exit 0 |
| Dir file removed | stamp with dir arg, remove file from dir | exit 0 |
| Missing file sentinel | stamp with nonexistent input, create it | exit 0 |
| Stamp replace | dk-stamp twice, second replaces first | second stamp wins |
| Stamp append | dk-stamp --append twice | union of inputs |
| Always | dk-always then dk-ifchange | exit 0 |
| Always --all | dk-always --all | all stamps removed |
| Error propagation | corrupt stamp file | exit 2 |
| Subcommand style | `dk-redo ifchange ...` | same as `dk-ifchange ...` |
| Symlink style | symlink dk-ifchange -> dk-redo | same behavior |

### Test sources to study

- **redo-c** (`github.com/leahneukirchen/redo-c`): minimal test suite in the
  repo, but the single-file C implementation is easy to read for edge cases
- **apenwarr/redo** (`github.com/apenwarr/redo`): comprehensive test suite
  in `t/` directory using the `sharness` test framework — best source for
  behavioral edge cases and expected semantics
- **goredo**: test patterns for the Go implementation specifically

The apenwarr/redo test suite is the most valuable. Many tests won't apply
(dk-redo has no .do scripts), but the dependency-tracking and change-detection
tests encode years of discovered edge cases.

## Build

```bash
CGO_ENABLED=0 go build -ldflags="-s -w" -o dk-redo ./cmd/dk-redo
```

Produces a ~3-4MB static binary. Symlinks created at install time.

## Performance budget

| Operation | Target | Notes |
|---|---|---|
| dk-ifchange (unchanged, 10 files) | < 10ms | Read stamp + hash 10 files |
| dk-ifchange (unchanged, 1000 files) | < 200ms | I/O bound |
| dk-ifchange (unchanged, dir with 10k files) | < 2s | Walk + hash |
| dk-stamp (100 files) | < 50ms | Hash + atomic write |
| Startup overhead | < 5ms | Before any I/O |
