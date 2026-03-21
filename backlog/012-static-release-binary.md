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
depends_on: ["008", "009", "010"]
source_file: dk-redo-implementation.md:215
---

## Summary

Add a `just release` target that produces statically linked binaries for
linux/amd64 and windows/amd64. macOS builds are optional.

## Current State

`just build` produces a local binary. Cross-compilation and release packaging
are not yet set up.

## Analysis & Recommendations

Justfile targets to add:

```just
release:
    mkdir -p dist
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version={{version}}" -o dist/dk-redo-linux-amd64 ./cmd/dk-redo
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X main.version={{version}}" -o dist/dk-redo-windows-amd64.exe ./cmd/dk-redo

release-macos:
    mkdir -p dist
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.version={{version}}" -o dist/dk-redo-darwin-amd64 ./cmd/dk-redo
    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.version={{version}}" -o dist/dk-redo-darwin-arm64 ./cmd/dk-redo
```

Version should come from git: `git describe --tags --always --dirty`.

**No install.sh.** Installation is handled by the `dk-redo install <dest-path>`
subcommand (ticket 007), which copies the binary and creates all symlinks.
A separate shell script is unnecessary — the binary installs itself.

Add `dist/` to `.gitignore`.

Verify linux binary is truly static: `file dist/dk-redo-linux-amd64` should
show "statically linked". Expected size ~3-4MB per implementation doc.

macOS builds are in a separate `release-macos` target since they are optional
and not all build environments support darwin cross-compilation.

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

func TestInstallSubcommand(t *testing.T) {
    // Run: dk-redo install <temp-dir>
    // Assert: binary copied, all symlinks created
    // Assert: symlinks point to dk-redo
}
```

### GREEN

1. Add `version` variable to justfile (from `git describe`)
2. Add `release` target with linux/windows cross-compilation
3. Add `release-macos` target (optional)
4. Add `dist/` to `.gitignore`
5. Verify `just release` produces working binaries

### REFACTOR

- Consider checksums file (sha256sums.txt) in dist/ for download verification
