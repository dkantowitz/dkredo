---
id: 016
title: Implement justfile-based integration tests for all documented use patterns
status: To Do
priority: 2
effort: Large
assignee: claude
created_date: 2026-03-27
labels: [feature, testing]
swimlane: Testing
dependencies: [012, 013, 014, 015]
---

## Summary

Create a comprehensive justfile-based integration test suite that exercises
the real `dkredo` binary against real files. Tests cover all documented use
patterns from `dkredo.md` and `dkredo-implementation.md`, including operation
pipelines, alias commands, symlink dispatch, edge cases, and the canonical
usage patterns (C compilation, deploy, bootstrapping, etc.).

Individual operation tickets include CLI integration tests for their specific
functionality. This ticket adds the cross-cutting tests that verify complete
workflows and documented patterns as a cohesive suite.

## Current State

After tickets 012-015, the full CLI is functional. Individual operations have
unit tests. This ticket adds end-to-end integration tests.

## Analysis & Recommendations

Implement as a `test/justfile` (or `test/integration.just`) with recipes that:
1. Create temp directories with test fixtures
2. Run `dkredo` (and alias symlinks) against real files
3. Assert exit codes, stamp file contents, and stdout output
4. Clean up

Each test recipe should be independent (own temp dir) so tests can run in
any order. A top-level `test-integration` recipe runs them all.

### Test categories from the implementation spec

**Operation pipeline tests** (spec lines 786-794):
- Guard/build/stamp cycle
- File change triggers rebuild
- Name addition persists across +check (exit 1)
- +remove-names + +stamp-facts pipeline
- +clear-facts forces re-check
- Missing file bootstrapping
- Depfile integration

**Alias (--cmd) tests** (spec lines 796-804):
- --cmd ifchange ≡ +add-names +check
- --cmd stamp ≡ +remove-names +add-names +stamp-facts
- --cmd stamp --append ≡ +add-names +stamp-facts
- --cmd always ≡ +clear-facts
- --cmd fnames ≡ +names -e

**Symlink dispatch tests** (spec lines 806-813):
- dkr-ifchange via symlink
- dkr-stamp via symlink
- dkr-always via symlink
- Unknown symlink → exit 2

**Edge cases** (spec lines 815-823):
- Label with slash → correct stamp filename
- Label with spaces
- Corrupt stamp recovery
- Empty stamp
- Concurrent access (different labels)

**.stamps/ directory location tests** (spec lines 825-835):
- .stamps/ in cwd
- .stamps/ in parent
- .stamps/ in grandparent
- No .stamps/ anywhere → created in cwd
- Nested project (child .stamps/ wins)
- Paths are project-relative

**Documented canonical patterns** (from dkredo.md):
- C compilation with gcc -MD dependency discovery
- Basic file dependencies
- Side-effect recipes (no output file)
- Multi-phase build
- Force rebuild (clean)
- Recipe chaining
- Parameterized recipes
- Bootstrapping (file doesn't exist yet)
- Deploy patterns
- Detecting recipe/compiler flag changes

## TDD Plan

### RED

Write all test recipes first. They should all fail initially (before
verifying the binary works end-to-end).

```just
# test/justfile

# Helper: assert exit code
# Usage: just _assert_exit cmd expected_code
_assert_exit cmd expected:
    #!/usr/bin/env bash
    set -uo pipefail
    eval "{{cmd}}" ; rc=$?
    if [ "$rc" -ne "{{expected}}" ]; then
        echo "FAIL: expected exit {{expected}}, got $rc"
        echo "  cmd: {{cmd}}"
        exit 1
    fi

# --- Operation Pipeline Tests ---

test-guard-build-stamp-cycle:
    #!/usr/bin/env bash
    set -euo pipefail
    T=$(mktemp -d)
    trap "rm -rf $T" EXIT
    cd "$T" && mkdir -p .stamps
    echo "hello" > a.c
    dkredo test +add-names a.c +stamp-facts
    dkredo test +check ; rc=$? ; [ "$rc" -eq 1 ] || exit 1  # unchanged
    echo "modified" > a.c
    dkredo test +check ; rc=$? ; [ "$rc" -eq 0 ] || exit 1  # changed
    echo "PASS: guard-build-stamp cycle"

test-name-addition-persists-across-check:
    #!/usr/bin/env bash
    set -euo pipefail
    T=$(mktemp -d)
    trap "rm -rf $T" EXIT
    cd "$T" && mkdir -p .stamps
    echo "data" > a.c
    dkredo test +add-names a.c +stamp-facts
    # Now add b.c (no facts) and check — check returns 0 (changed, b.c has no facts)
    echo "data" > b.c
    dkredo test +add-names b.c +check ; rc=$? ; [ "$rc" -eq 0 ] || exit 1
    # Verify b.c is in the stamp (persisted despite check stopping pipeline)
    grep -q "b.c" .stamps/test || { echo "FAIL: b.c not in stamp"; exit 1; }
    echo "PASS: name addition persists across +check"

test-clear-facts-forces-recheck:
    #!/usr/bin/env bash
    set -euo pipefail
    T=$(mktemp -d)
    trap "rm -rf $T" EXIT
    cd "$T" && mkdir -p .stamps
    echo "data" > a.c
    dkredo test +add-names a.c +stamp-facts
    dkredo test +check ; rc=$? ; [ "$rc" -eq 1 ] || exit 1  # unchanged
    dkredo test +clear-facts
    dkredo test +check ; rc=$? ; [ "$rc" -eq 0 ] || exit 1  # changed (no facts)
    echo "PASS: clear-facts forces recheck"

test-missing-file-bootstrapping:
    #!/usr/bin/env bash
    set -euo pipefail
    T=$(mktemp -d)
    trap "rm -rf $T" EXIT
    cd "$T" && mkdir -p .stamps
    dkredo test +add-names nonexistent.c +stamp-facts
    grep -q "missing:true" .stamps/test || exit 1
    echo "now exists" > nonexistent.c
    dkredo test +check ; rc=$? ; [ "$rc" -eq 0 ] || exit 1  # changed (file appeared)
    echo "PASS: missing file bootstrapping"

# --- Alias Equivalence Tests ---

test-cmd-ifchange-equivalence:
    #!/usr/bin/env bash
    set -euo pipefail
    T=$(mktemp -d)
    trap "rm -rf $T" EXIT
    cd "$T" && mkdir -p .stamps
    echo "data" > a.c
    dkredo test1 --cmd ifchange a.c
    dkredo test2 +add-names a.c +check
    diff .stamps/test1 .stamps/test2 || { echo "FAIL: stamps differ"; exit 1; }
    echo "PASS: --cmd ifchange equivalence"

test-cmd-stamp-equivalence:
    #!/usr/bin/env bash
    set -euo pipefail
    T=$(mktemp -d)
    trap "rm -rf $T" EXIT
    cd "$T" && mkdir -p .stamps
    echo "data" > a.c
    dkredo test1 --cmd stamp a.c
    dkredo test2 +remove-names +add-names a.c +stamp-facts
    diff .stamps/test1 .stamps/test2 || { echo "FAIL: stamps differ"; exit 1; }
    echo "PASS: --cmd stamp equivalence"

test-cmd-stamp-append:
    #!/usr/bin/env bash
    set -euo pipefail
    T=$(mktemp -d)
    trap "rm -rf $T" EXIT
    cd "$T" && mkdir -p .stamps
    echo "old" > old.c && echo "new" > new.c
    dkredo test +add-names old.c +stamp-facts
    dkredo test --cmd stamp --append new.c
    grep -q "old.c" .stamps/test || { echo "FAIL: old.c missing"; exit 1; }
    grep -q "new.c" .stamps/test || { echo "FAIL: new.c missing"; exit 1; }
    echo "PASS: --cmd stamp --append"

# --- Edge Cases ---

test-label-with-slash:
    #!/usr/bin/env bash
    set -euo pipefail
    T=$(mktemp -d)
    trap "rm -rf $T" EXIT
    cd "$T" && mkdir -p .stamps
    echo "data" > a.c
    dkredo "output/config.json" +add-names a.c +stamp-facts
    [ -f ".stamps/output%2Fconfig.json" ] || { echo "FAIL: escaped label file missing"; exit 1; }
    echo "PASS: label with slash"

test-stamps-dir-parent-search:
    #!/usr/bin/env bash
    set -euo pipefail
    T=$(mktemp -d)
    trap "rm -rf $T" EXIT
    mkdir -p "$T/.stamps" "$T/subdir"
    echo "data" > "$T/subdir/a.c"
    cd "$T/subdir"
    dkredo test +add-names a.c +stamp-facts
    [ -f "$T/.stamps/test" ] || { echo "FAIL: stamp not in parent .stamps/"; exit 1; }
    echo "PASS: stamps dir parent search"

test-empty-stamp-check:
    #!/usr/bin/env bash
    set -euo pipefail
    T=$(mktemp -d)
    trap "rm -rf $T" EXIT
    cd "$T" && mkdir -p .stamps
    # Add names with no files, then check
    dkredo test +add-names +check ; rc=$? ; [ "$rc" -eq 1 ] || exit 1  # empty → unchanged
    echo "PASS: empty stamp check"

# --- Documented Canonical Patterns (simplified) ---

test-pattern-basic-file-deps:
    #!/usr/bin/env bash
    set -euo pipefail
    T=$(mktemp -d)
    trap "rm -rf $T" EXIT
    cd "$T" && mkdir -p .stamps
    echo "v1" > src.c && echo "v1" > hdr.h
    # ifchange pattern
    dkredo out.bin +add-names src.c hdr.h +check ; rc=$?
    [ "$rc" -eq 0 ] || exit 1  # first run: changed
    dkredo out.bin +remove-names +add-names src.c hdr.h +stamp-facts
    dkredo out.bin +add-names src.c hdr.h +check ; rc=$?
    [ "$rc" -eq 1 ] || exit 1  # second run: unchanged
    echo "v2" > hdr.h
    dkredo out.bin +add-names src.c hdr.h +check ; rc=$?
    [ "$rc" -eq 0 ] || exit 1  # header changed: rebuild
    echo "PASS: basic file deps pattern"

test-pattern-force-rebuild:
    #!/usr/bin/env bash
    set -euo pipefail
    T=$(mktemp -d)
    trap "rm -rf $T" EXIT
    cd "$T" && mkdir -p .stamps
    echo "data" > a.c
    dkredo out.bin +add-names a.c +stamp-facts
    dkredo out.bin +check ; rc=$? ; [ "$rc" -eq 1 ] || exit 1
    dkredo out.bin +clear-facts  # the "always" / "clean" pattern
    dkredo out.bin +check ; rc=$? ; [ "$rc" -eq 0 ] || exit 1
    echo "PASS: force rebuild pattern"

test-pattern-side-effect-recipe:
    #!/usr/bin/env bash
    set -euo pipefail
    T=$(mktemp -d)
    trap "rm -rf $T" EXIT
    cd "$T" && mkdir -p .stamps
    echo "v1" > config.yaml
    # Check (first run, no stamp) → changed
    dkredo deploy-staging +add-names config.yaml +check ; rc=$?
    [ "$rc" -eq 0 ] || exit 1
    # Stamp after "deploy"
    dkredo deploy-staging +stamp-facts
    # Check again → unchanged
    dkredo deploy-staging +check ; rc=$?
    [ "$rc" -eq 1 ] || exit 1
    # Modify config → changed
    echo "v2" > config.yaml
    dkredo deploy-staging +check ; rc=$?
    [ "$rc" -eq 0 ] || exit 1
    echo "PASS: side-effect recipe pattern"

# --- Run All Tests ---

test-all: test-guard-build-stamp-cycle test-name-addition-persists-across-check test-clear-facts-forces-recheck test-missing-file-bootstrapping test-cmd-ifchange-equivalence test-cmd-stamp-equivalence test-cmd-stamp-append test-label-with-slash test-stamps-dir-parent-search test-empty-stamp-check test-pattern-basic-file-deps test-pattern-force-rebuild test-pattern-side-effect-recipe
    @echo "ALL INTEGRATION TESTS PASSED"
```

### GREEN

1. Build the `dkredo` binary (`just build`)
2. Ensure binary is on PATH for test recipes
3. Create `test/justfile` with all test recipes above
4. Add `test-integration` target to root Justfile that runs `just -f test/justfile test-all`
5. Run full suite, fix any failures

### REFACTOR

1. Add any missing test cases discovered during implementation.
2. Ensure each test is fully isolated (own temp dir, no shared state).
3. Add depfile integration test (requires creating a mock .d file).
4. Add -@ file input integration test.
5. Consider adding timing assertions for performance budget (optional).
