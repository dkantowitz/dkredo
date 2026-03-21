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
internal/hasher/         — BLAKE3 file/directory hashing
internal/resolve/        — input argument resolution (file vs dir vs stdin, including ReadStdin)
```

### argv[0] dispatch

```go
func main() {
    cmd, args := resolveCommand(os.Args)
    switch cmd {
    case "ifchange": runIfchange(args)
    case "stamp":    runStamp(args)
    case "always":   runAlways(args)
    case "install":  runInstall(args)
    // rev1.x: ood, affects, dot, sources
    default:         usage(); os.Exit(2)
    }
}

// resolveCommand determines the subcommand and remaining args
// from argv without mutating globals.
func resolveCommand(argv []string) (string, []string) {
    cmd := filepath.Base(argv[0])
    if strings.HasPrefix(cmd, "dk-") {
        cmd = strings.TrimPrefix(cmd, "dk-")
    }
    args := argv[1:]
    // subcommand style: dk-redo ifchange ...
    if cmd == "redo" {
        if len(args) < 1 { usage(); os.Exit(2) }
        cmd = args[0]
        args = args[1:]
    }
    // "install" is only available via subcommand, not argv[0] dispatch
    if cmd == "install" && filepath.Base(argv[0]) != "dk-redo" {
        usage(); os.Exit(2)
    }
    return cmd, args
}
```

Note: command functions receive `args []string` as a parameter rather than
reading `os.Args` directly. This avoids mutating global state and makes it
easier to source arguments from other places (e.g., `DK_REDO_FLAGS`
environment variable in the future).

### Core algorithm (dk-ifchange)

```
1. Parse flags (-v, -q, -n), extract label (arg[0]) and inputs (arg[1:])
2. Resolve inputs: expand directories, read stdin if "-" or "-0"
   (stdin paths are spliced at the position of - or -0 in the arg list)
3. If -n flag: exit 0 (always report changed — force rebuild)
4. Read stamp file (.stamps/<escaped-label>)
5. If no stamp: exit 0 (first run — changed)
6. Compare file lists: if different sets of paths, exit 0 (changed)
7. For each file in stamp, check recorded facts against current state:
   - unknown fact key → warn to stderr, treat as changed (exit 0)
   - missing:true → check file still absent
   - size:<n>     → compare file size (fast path, avoids hashing)
   - blake3:<hex> → compare BLAKE3 hash
8. If any fact is false: exit 0 (changed)
9. All facts hold: exit 1 (unchanged)
10. On error: exit 2
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
// Percent-encode characters that cannot appear in filenames.
func escapeLabel(label string) string {
    label = strings.ReplaceAll(label, "%", "%25") // escape char first
    label = strings.ReplaceAll(label, "/", "%2F")
    return label
}

// Percent-encode characters that would break stamp line parsing.
func encodePath(path string) string {
    path = strings.ReplaceAll(path, "%", "%25") // escape char first
    path = strings.ReplaceAll(path, "\t", "%09")
    path = strings.ReplaceAll(path, "\n", "%0A")
    return path
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
| HashFile follows symlink | symlink to file | hash of target content |
| HashDir empty dir | empty temp dir | empty list (no files) |
| HashDir with files | dir with 3 files | sorted list, hashes change on modification |
| HashDir determinism | same files, different creation order | same hash |
| HashDir follows symlinks | dir with symlinked file | hash of target content |
| HashDir symlink loop | dir with circular symlink | error (not infinite loop) |
| HashDir size before hash | file with different size | size recorded, no hash needed for comparison |

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
| Tab-delimited roundtrip | path with spaces | parsed correctly (spaces verbatim) |
| Path with tab | "dir\tname/file" | tab encoded as %09, roundtrips |
| Path with percent | "100%/file" | percent encoded as %25, roundtrips |
| Label escaping | "output/config.json" | ".stamps/output%2Fconfig.json" |
| Label with literal % | "100%done" | ".stamps/100%25done" |
| Label with special chars | "foo bar" | handled correctly |
| Unknown facts → changed | line with extra key:val | treated as changed, warning issued |
| Corrupt stamp | garbage/malformed content | treated as changed (out of date), exit 0 |
| Adversarial stamp | binary data, very long lines | handled gracefully, treated as changed |

**resolve package:**

| Test | Input | Expected |
|---|---|---|
| File args | ["src/a.c", "src/b.c"] | two file paths |
| Dir arg | ["src/"] | all files under src/, sorted |
| Mixed | ["a.c", "src/", "b.c"] | files + expanded dir |
| Stdin newline | stdin="x.c\ny.c\n", args=["-"] | two files from stdin |
| Stdin null | stdin="x.c\0y.c\0", args=["-0"] | two files from stdin |
| Mixed with stdin | ["a.c", "-", "b.c"] + stdin="x.c\n" | a.c, x.c, b.c (stdin spliced at position) |
| Stdin position ordering | ["z.c", "-", "a.c"] + stdin="m.c\n" | final list sorted+deduped regardless |
| ReadStdin newline | "a.c\nb.c\n" reader | list of 2 files |
| ReadStdin null | "a.c\0b.c\0" reader | list of 2 files |
| ReadStdin empty | "" reader | empty list |
| Deduplication | same file from args and stdin | listed once |
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
| Label with slash | label "output/config.json" | stamp at .stamps/output%2Fconfig.json, roundtrips |
| Stdin combined with args | `dk-ifchange label a.c - b.c` with stdin | all inputs processed correctly |
| Force changed flag | dk-ifchange -n label files... | always exit 0 |
| Unknown symlink name | symlink dk-bogus -> dk-redo | exit 2 with usage |

### Performance benchmarks

Benchmarks are part of the regression test suite. They verify that dk-redo
stays within its performance budget.

| Benchmark | Setup | Target |
|---|---|---|
| BenchmarkIfchangeUnchanged10 | 10 files, stamp exists | < 10ms |
| BenchmarkIfchangeUnchanged300 | 300 files across 10 labels (30 each) | < 300ms total |
| BenchmarkStamp100 | 100 small files | < 50ms |
| BenchmarkStartupOverhead | no-op invocation (--help) | < 5ms |

The 300-dependency benchmark (300 files spread among 10 labels, total check
under 300ms) is the primary regression target. Benchmarks run as Go benchmark
tests (`go test -bench`) and can be gated in CI.

### Known test deficiencies

**Atomic write testing:** The atomic write mechanism (temp file + rename) is
inherently difficult to test for crash safety. Unit tests verify the write-
then-rename sequence but cannot simulate a crash between the two operations.
This is noted as an accepted deficiency. The mechanism is well-established
(used by redo-c, goredo, and many other tools) and does not warrant complex
crash simulation in the test suite.

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

Produces a ~3-4MB static binary. Symlinks are created by `dk-redo install <dest-path>`.

## Performance budget

| Operation | Target | Notes |
|---|---|---|
| dk-ifchange (unchanged, 10 files) | < 10ms | Read stamp + hash 10 files |
| dk-ifchange (unchanged, 1000 files) | < 200ms | I/O bound |
| dk-ifchange (unchanged, dir with 10k files) | < 2s | Walk + hash |
| dk-stamp (100 files) | < 50ms | Hash + atomic write |
| Startup overhead | < 5ms | Before any I/O |
