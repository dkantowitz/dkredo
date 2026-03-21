---
id: "017"
title: Line and branch coverage targets with regression staking
status: Done
completed_date: 2026-03-21
priority: 3
effort: Small
assignee: claude
created_date: 2026-03-21
labels: [feature, core]
swimlane: Tooling/Claude Environment
phase: 3
depends_on: ["002", "003", "004", "005", "006"]
source_file: dk-redo-implementation.md:126
---

## Summary

Add coverage measurement infrastructure and enforce line & branch coverage
targets. Initial target is 80% during active development to surface errors
and edge conditions. Once features are settled (after phase 4), increase
the target to 95% for regression staking.

## Current State

No coverage measurement exists. Go's built-in `go test -cover` provides
line coverage. Branch coverage requires `-covermode=atomic` or third-party
tooling.

## Analysis & Recommendations

### Phase 3 target: 80% line and branch coverage

The 80% target applies during active feature development (phases 2-3). Its
purpose is **error discovery** — finding untested code paths that may contain
bugs or edge-condition handling gaps. Coverage gaps at this stage point to
code that needs tests, not code that needs to be deleted.

### Phase 5+ target: 95% line and branch coverage

Once all features are implemented and stable (after phase 4 integration tests
pass), raise the bar to 95%. At this stage, coverage serves as **regression
staking** — high coverage means future changes are unlikely to silently break
existing behavior. The remaining 5% covers genuinely unreachable defensive
code (e.g., OS-level error paths that can't be triggered in tests).

### Implementation

1. **Justfile targets:**

```just
cover:
    go test -coverprofile=coverage.out -covermode=atomic ./internal/...
    go tool cover -func=coverage.out

cover-html:
    go test -coverprofile=coverage.out -covermode=atomic ./internal/...
    go tool cover -html=coverage.out -o coverage.html

cover-check:
    go test -coverprofile=coverage.out -covermode=atomic ./internal/...
    @go tool cover -func=coverage.out | grep ^total | awk '{print $$3}' | \
        awk -F. '{if ($$1 < 80) {print "FAIL: coverage " $$1 "% < 80%"; exit 1} else {print "OK: coverage " $$1 "%"}}'
```

2. **Per-package coverage:** Report coverage per package so weak spots are
   visible. The 80% target applies to each package individually, not just
   the aggregate — a 100% hasher and 50% stamp should not average to "good
   enough."

3. **Branch coverage:** Use `-covermode=atomic` which tracks execution
   counts per statement. For true branch coverage analysis, consider
   `go-cover-treemap` or manual review of coverage HTML output to identify
   untested conditional branches.

4. **Coverage in CI:** `just cover-check` should be part of the CI pipeline.
   Fail the build if coverage drops below the target.

5. **Exclusions:** Test files, generated code, and `cmd/dk-redo/main.go`
   (thin dispatch layer) are excluded from coverage requirements. Coverage
   targets apply to `internal/` packages.

### What 80% coverage should catch

- Untested error paths in hasher (permission denied, symlink loops)
- Missing edge cases in stamp parsing (empty lines, missing tabs, extra whitespace)
- Resolve package: untested stdin/arg combinations
- Path encoding: roundtrip failures on edge-case characters

### What 95% coverage adds for regression

- Every conditional branch in Compare (size fast path, missing→appeared, etc.)
- All error handling paths in Read/Write
- Edge cases in Resolve (empty args, all-stdin, all-dirs)
- Verbose/quiet output formatting paths

## TDD Plan

### RED

No tests needed — this ticket adds measurement infrastructure.

### GREEN

1. Add `cover`, `cover-html`, `cover-check` targets to justfile
2. Add `coverage.out` and `coverage.html` to `.gitignore`
3. Run `just cover` and identify packages below 80%
4. Document per-package coverage in a comment or CI output
5. Wire `just cover-check` into `just test` pipeline

### REFACTOR

- After phase 4: update `cover-check` threshold from 80 to 95
- Add per-package threshold checking if aggregate masking becomes a problem

## Completion Notes

**Commit:** `f2514d5`

### Files modified
- `justfile` — added `cover`, `cover-html`, `cover-check` targets

### Justfile targets added
- `cover`: generates `coverage.out` and prints per-function coverage
- `cover-html`: generates `coverage.html` for browser viewing
- `cover-check`: fails build if aggregate internal coverage < 80%

### Current coverage (internal packages only)

| Package | Coverage | Notes |
|---------|----------|-------|
| `internal/hasher` | 80.4% | Uncovered: OS-level error paths in file I/O |
| `internal/resolve` | 91.5% | Uncovered: edge cases in dedup |
| `internal/stamp` | 91.7% | Uncovered: filesystem error paths in Write |
| **Aggregate** | **89.3%** | **Exceeds 80% threshold** |

### Per-function coverage highlights
- `stamp.Compare`: 100%, `stamp.Append`: 100%, `stamp.FormatFacts`: 100%
- All encoding functions: 100%
- `stamp.Write`: 72.4% (filesystem error branches)
- `hasher.HashDir`: 76.9% (symlink edge cases)

### Design decisions
- Coverage targets apply to `internal/` packages only (not `cmd/dk-redo/` which is tested via integration tests)
- `.gitignore` updated with `coverage.out` and `coverage.html`
- `cover-check` uses awk to parse total coverage and compare against threshold

### Bug fix
- `cover-check` was failing because `internal/testutil` (0.0% — test infrastructure with no unit tests of its own) dragged the aggregate total to 77.0%. Fixed by excluding testutil from coverage measurement and switching to a shebang recipe for reliable `awk`/shell variable handling in justfile.

### Deferred work
- **Phase 5 target: raise threshold from 80% to 95%** — requires additional tests for filesystem error paths in `stamp.Write` and `hasher.HashDir`
- Per-package threshold checking not implemented — aggregate masking hasn't been a problem since all packages individually exceed 80%
