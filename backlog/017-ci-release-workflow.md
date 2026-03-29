---
id: 017
title: Add GitHub Actions CI workflow and release automation
status: Done
priority: 2
effort: Small
assignee: claude
created_date: 2026-03-28
labels: [feature, infrastructure]
swimlane: Infrastructure
dependencies: []
---

## Summary

Add GitHub Actions workflows for continuous integration (test on push/PR) and
automated release (cross-compile and publish on version tag). Add Justfile
recipes for creating major and minor releases with automatic version bumping.

## Current State

No CI or release automation exists. The project builds and tests locally via
`just build`, `just test`, `just test-integration`, and `just cover-check`.

## Analysis & Recommendations

### CI Workflow (`.github/workflows/ci.yml`)

Triggers on push to `main` and pull requests against `main`. Steps run in
fast-fail order (cheapest checks first):

1. **Vet + unit tests** (`just test`) — `go vet` catches static errors, then
   `go test -race` runs all unit tests. Fastest to fail (~2s).
2. **Build** (`just build`) — confirms the binary compiles. Catches import
   errors or linker issues not caught by tests.
3. **Integration tests** (`just test-integration`) — builds binary then runs
   31 bash test recipes against real files. Slower (~10s).
4. **Coverage check** (`just cover-check`) — verifies ≥80% line coverage per
   package. Catches regressions in test coverage.

Each step is a shell command. If it exits non-zero, the GitHub Actions step
turns red and the job aborts (remaining steps are skipped). `just` propagates
the exit code of the failing recipe, so `go test` exit 1 → `just test` exit 1
→ CI step fails → job stops.

Requires `just` installed in the runner. Use `extractions/setup-just@v2`.

### Release Workflow (`.github/workflows/release.yml`)

Triggers on tag push matching `v*`. Cross-compiles for two platforms:

| GOOS    | GOARCH | Binary name              |
|---------|--------|--------------------------|
| linux   | amd64  | `dkredo-linux-amd64`     |
| windows | amd64  | `dkredo-windows-amd64.exe` |

Version is embedded via `-ldflags -X main.version=<tag>`. Binaries are
uploaded as GitHub Release assets with auto-generated release notes.

### Justfile Release Recipes

```
just release-minor    # v0.1.0 → v0.2.0
just release-major    # v0.2.0 → v1.0.0
```

Each recipe:
1. Reads the latest `v*` tag (or defaults to `v0.0.0`)
2. Bumps the appropriate version component
3. Creates an annotated git tag
4. Pushes the tag to origin (triggers release workflow)

## TDD Plan

### RED

No automated tests for CI workflows — verified manually by pushing a tag
and observing the GitHub Actions run.

### GREEN

1. Create `.github/workflows/ci.yml`
2. Create `.github/workflows/release.yml`
3. Add `release-minor` and `release-major` recipes to `Justfile`
4. Push and verify CI runs on the commit
5. Create a test tag `v0.1.0` and verify release workflow produces binaries

### REFACTOR

1. Pin action versions to specific SHAs if desired for supply-chain security.
2. Add caching for Go modules (`actions/cache`) if CI is slow.

## Results

### Files Created
- `.github/workflows/ci.yml` — CI workflow (test/build/integration/coverage on push and PR)
- `.github/workflows/release.yml` — Release workflow (cross-compile linux-amd64 + windows-amd64, publish on tag)
- `Justfile` — added `release-minor`, `release-major`, `_latest-version` recipes

### Deviations
Justfile release recipes were added before the workflow files (during ticket creation). Windows binary gets `.exe` suffix via matrix include.
