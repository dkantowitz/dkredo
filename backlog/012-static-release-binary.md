---
id: "012"
title: Add release target for static cross-compiled binaries
status: To Do
priority: 4
effort: Small
assignee: claude
created_date: 2026-03-21
labels: [feature, core]
swimlane: Tooling/Claude Environment
phase: 4
depends_on: ["011"]
source_file: dk-redo-implementation.md:215
---

## Summary

Add a `just release` target that produces statically linked binaries for
linux/amd64 and windows/amd64. Include symlink creation in an install script.

## Current State

`just build` produces a local binary. Cross-compilation and release packaging
are not yet set up. Build command per `dk-redo-implementation.md:215-219`:
```bash
CGO_ENABLED=0 go build -ldflags="-s -w" -o dk-redo ./cmd/dk-redo
```

## Analysis & Recommendations

Justfile targets to add:

```just
release:
    mkdir -p dist
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version={{version}}" -o dist/dk-redo-linux-amd64 ./cmd/dk-redo
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X main.version={{version}}" -o dist/dk-redo-windows-amd64.exe ./cmd/dk-redo
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.version={{version}}" -o dist/dk-redo-darwin-amd64 ./cmd/dk-redo
    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.version={{version}}" -o dist/dk-redo-darwin-arm64 ./cmd/dk-redo
```

Version should come from git: `git describe --tags --always --dirty`.

Include an `install.sh` script that:
1. Copies the binary to a user-specified prefix (default `/usr/local/bin`)
2. Creates symlinks: dk-ifchange, dk-stamp, dk-always, dk-ood, dk-affects,
   dk-dot, dk-sources

Add `dist/` to `.gitignore`.

Verify linux binary is truly static: `file dist/dk-redo-linux-amd64` should
show "statically linked". Expected size ~3-4MB per implementation doc.

## TDD Plan

### RED

```go
func TestBinaryIsStatic(t *testing.T) {
    // Run: file dk-redo-linux-amd64
    // Assert: contains "statically linked"
}

func TestVersionFlag(t *testing.T) {
    // Run: dk-redo --version
    // Assert: prints version string
}

func TestSymlinkDispatch(t *testing.T) {
    // Create symlinks, invoke via symlink, verify correct command runs
}
```

### GREEN

1. Add `version` variable to justfile (from `git describe`)
2. Add `release` target with cross-compilation
3. Write `install.sh` with symlink creation
4. Add `dist/` to `.gitignore`
5. Verify `just release` produces working binaries

### REFACTOR

- Consider checksums file (sha256sums.txt) in dist/ for download verification
