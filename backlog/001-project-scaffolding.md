---
id: "001"
title: Scaffold Go project with justfile and directory structure
status: Done
completed_date: 2026-03-21
priority: 1
effort: Small
assignee: claude
created_date: 2026-03-21
labels: [feature, core]
swimlane: Tooling/Claude Environment
phase: 1
depends_on: []
source_file: dk-redo-implementation.md:27
---

## Summary

Initialize the Go module, create the package directory structure, and write
a justfile with `build`, `test`, `test-unit`, `test-integration`, and `clean`
targets. This is the foundation everything else depends on.

## Current State

No Go code exists yet. The repo contains only design docs and the backlog.

## Analysis & Recommendations

Directory layout per `dk-redo-implementation.md:27`:

```
cmd/dk-redo/main.go      — entry point (placeholder)
internal/stamp/           — stamp read/write/compare
internal/hasher/          — BLAKE3 file/dir hashing
internal/resolve/         — input argument resolution
```

The justfile replaces any Makefile. Targets:

| Target             | Command                                                              |
| ------------------ | -------------------------------------------------------------------- |
| `build`            | `CGO_ENABLED=0 go build -ldflags="-s -w" -o dk-redo ./cmd/dk-redo`  |
| `test`             | `just test-unit && just test-integration`                            |
| `test-unit`        | `go test ./internal/...`                                             |
| `test-integration` | `go test ./test/...` (or `./cmd/dk-redo/...` integration tests)      |
| `clean`            | `rm -f dk-redo && rm -rf .stamps/`                                   |

Use `CGO_ENABLED=0` for all builds to ensure static linking. The `go.mod`
module name should be `github.com/dkantowitz/dk-redo`.

Dependencies to add:
- `github.com/zeebo/blake3` — BLAKE3 hashing

## TDD Plan

### RED

```go
// cmd/dk-redo/main.go — placeholder that proves the binary compiles
package main

import "fmt"

func main() {
    fmt.Println("dk-redo: not yet implemented")
}
```

### GREEN

1. `go mod init github.com/dkantowitz/dk-redo`
2. `go get github.com/zeebo/blake3`
3. Create `cmd/dk-redo/main.go` placeholder
4. Create empty package files: `internal/stamp/stamp.go`, `internal/hasher/hasher.go`, `internal/resolve/resolve.go`
5. Write `justfile` with all targets
6. Add `.stamps/` to `.gitignore`
7. Verify `just build` produces a static binary
8. Verify `just test` passes (trivially, no tests yet)

### REFACTOR

- Ensure `just build` output goes to project root as `./dk-redo`
- Confirm binary is static: `file dk-redo` should show "statically linked"

## Completion Notes

**Commit:** `20189e4`

### Files created
- `go.mod`, `go.sum` — Go module (`github.com/dkantowitz/dk-redo`)
- `cmd/dk-redo/main.go` — entry point placeholder
- `internal/stamp/stamp.go`, `internal/hasher/hasher.go`, `internal/resolve/resolve.go` — package placeholders
- `internal/testutil/testutil.go` — test helpers (WriteTempFile, WriteTempDir, RunBinary)
- `justfile` — build targets (build, test, test-unit, test-integration, clean)
- `.gitignore` — dk-redo binary, .stamps/, dist/, coverage files

### Outcome
- `just build` produces a 1.9MB static binary at `./dk-redo`
- `CGO_ENABLED=0` ensures static linking
- Module depends on `github.com/zeebo/blake3`
- `.stamps/` added to `.gitignore`

### Deferred work
- None
