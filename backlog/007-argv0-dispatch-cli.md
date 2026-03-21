---
id: "007"
title: Implement argv[0] dispatch and CLI flag parsing
status: To Do
priority: 3
effort: Small
assignee: claude
created_date: 2026-03-21
labels: [feature, core]
swimlane: Core Library
phase: 3
depends_on: ["001"]
source_file: dk-redo-implementation.md:36
---

## Summary

Implement the main entry point with argv[0]-based command dispatch (busybox
style) and shared CLI flag parsing (`-v`, `-q`, `--color`, `--stamps-dir`,
`--help`, `--version`). This is the CLI skeleton that command implementations
plug into.

## Current State

`cmd/dk-redo/main.go` exists as a placeholder from ticket 001. The dispatch
pattern is specified in `dk-redo-implementation.md:36-57`.

## Analysis & Recommendations

Two invocation styles must work identically:
- **Symlink style:** `dk-ifchange firmware.bin src/*.c` (argv[0] = "dk-ifchange")
- **Subcommand style:** `dk-redo ifchange firmware.bin src/*.c` (argv[0] = "dk-redo")

Dispatch logic per `dk-redo-implementation.md:37-57`:

```go
func main() {
    cmd := filepath.Base(os.Args[0])
    if strings.HasPrefix(cmd, "dk-") {
        cmd = strings.TrimPrefix(cmd, "dk-")
    }
    if cmd == "redo" {
        if len(os.Args) < 2 { usage(); os.Exit(2) }
        cmd = os.Args[1]
        os.Args = os.Args[1:]
    }
    // dispatch to command functions
}
```

Shared flags per `dk-redo.md:922-931`:
- `-v` verbose, `-q` quiet, `--color`/`--no-color`, `--stamps-dir <path>`
- `--help`, `--version`

Use Go's `flag` package or a thin wrapper. Keep it simple — no heavy CLI
framework needed for this scope.

The `--version` flag should embed the version at build time via `-ldflags`:
```
-ldflags="-s -w -X main.version=..."
```

Update the justfile `build` target to pass the version.

## TDD Plan

### RED

```go
func TestDispatchSymlinkStyle(t *testing.T) {
    // Simulate argv[0] = "dk-ifchange"
    // Verify correct command is resolved
}

func TestDispatchSubcommandStyle(t *testing.T) {
    // Simulate argv = ["dk-redo", "ifchange", ...]
    // Verify correct command is resolved, args shifted
}

func TestDispatchUnknownCommand(t *testing.T) {
    // Unknown command → exit 2
}

func TestSharedFlags(t *testing.T) {
    // --stamps-dir, -v, -q parsed before command dispatch
}
```

### GREEN

1. Implement `resolveCommand() string` — argv[0] dispatch logic
2. Implement shared flag parsing with `flag.FlagSet`
3. Implement `usage()` function listing all commands
4. Wire dispatch to stub command functions (return exit code 2 "not implemented")
5. Add version embedding via ldflags
6. Update justfile build target with version injection

### REFACTOR

- Extract command registry into a map for cleaner dispatch
- Ensure `--help` output is consistent across all commands
