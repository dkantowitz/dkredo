---
id: "007"
title: Implement argv[0] dispatch and CLI flag parsing
status: To Do
priority: 2
effort: Small
assignee: claude
created_date: 2026-03-21
labels: [feature, core]
swimlane: Core Library
phase: 2
depends_on: ["001"]
source_file: dk-redo-implementation.md:36
---

## Summary

Implement the main entry point with argv[0]-based command dispatch (busybox
style) and shared CLI flag parsing (`-v`, `-q`, `--stamps-dir`,
`--help`, `--version`). This is the CLI skeleton that command implementations
plug into.

## Current State

`cmd/dk-redo/main.go` exists as a placeholder from ticket 001. The dispatch
pattern is specified in `dk-redo-implementation.md:36-57`.

## Analysis & Recommendations

Two invocation styles must work identically:
- **Symlink style:** `dk-ifchange firmware.bin src/*.c` (argv[0] = "dk-ifchange")
- **Subcommand style:** `dk-redo ifchange firmware.bin src/*.c` (argv[0] = "dk-redo")

**Pass args as arguments, do not mutate globals.** Command functions receive
`args []string` as a parameter rather than reading `os.Args`. This avoids
mutating global state and makes it easier to source arguments from other
places (e.g., `DK_REDO_FLAGS` environment variable in the future).

Dispatch logic per `dk-redo-implementation.md:36-57`:

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

func resolveCommand(argv []string) (string, []string) {
    cmd := filepath.Base(argv[0])
    if strings.HasPrefix(cmd, "dk-") {
        cmd = strings.TrimPrefix(cmd, "dk-")
    }
    args := argv[1:]
    if cmd == "redo" {
        if len(args) < 1 { usage(); os.Exit(2) }
        cmd = args[0]
        args = args[1:]
    }
    // "install" only via subcommand, not argv[0] dispatch
    if cmd == "install" && filepath.Base(argv[0]) != "dk-redo" {
        usage(); os.Exit(2)
    }
    return cmd, args
}
```

Shared flags per `dk-redo.md:922-931`:
- `-v` verbose, `-q` quiet, `--stamps-dir <path>`
- `--help`, `--version`

Note: `--color`/`--no-color` are intentionally omitted — not needed.

Use Go's `flag` package or a thin wrapper. Keep it simple — no heavy CLI
framework needed for this scope.

The `--version` flag should embed the version at build time via `-ldflags`:
```
-ldflags="-s -w -X main.version=..."
```

**Unknown command handling:** If invoked via an unrecognized symlink name
(e.g., `dk-bogus`), exit 2 with a usage message.

**`install` subcommand:** The `install` command copies the binary to a
destination directory and creates all symlinks. It is only available via
`dk-redo install <dest-path>`, NOT via argv[0] dispatch (a symlink named
`dk-install` would not trigger it).

**`--stamps-dir` plumbing:** The `--stamps-dir` flag is parsed in this
module and threaded through to all command functions. Every command that
reads or writes stamps must accept the stamps directory as a parameter
(not hardcoded to `.stamps/`).

## TDD Plan

### RED

```go
func TestResolveCommandSymlinkStyle(t *testing.T) {
    // argv = ["dk-ifchange", "label", "file.c"]
    // → cmd="ifchange", args=["label", "file.c"]
}

func TestResolveCommandSubcommandStyle(t *testing.T) {
    // argv = ["dk-redo", "ifchange", "label", "file.c"]
    // → cmd="ifchange", args=["label", "file.c"]
}

func TestResolveCommandUnknown(t *testing.T) {
    // argv = ["dk-bogus"] → exit 2
}

func TestResolveCommandInstallNotViaSymlink(t *testing.T) {
    // argv = ["dk-install"] → exit 2 (install only via subcommand)
    // argv = ["dk-redo", "install", "/usr/local/bin"] → cmd="install"
}

func TestSharedFlags(t *testing.T) {
    // --stamps-dir parsed and available to command functions
    // -v, -q parsed correctly
}

func TestNoSubcommand(t *testing.T) {
    // argv = ["dk-redo"] with no subcommand → exit 2 with usage
}
```

### GREEN

1. Implement `resolveCommand(argv) (string, []string)` — argv[0] dispatch logic
2. Implement shared flag parsing with `flag.FlagSet`
3. Implement `usage()` function listing all commands
4. Wire dispatch to stub command functions that accept `args []string`
5. Add version embedding via ldflags
6. Update justfile build target with version injection
7. Implement `--stamps-dir` flag, default to `.stamps/`

### REFACTOR

- Extract command registry into a map for cleaner dispatch
- Ensure `--help` output is consistent across all commands
