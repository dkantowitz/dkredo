---
id: "011"
title: Activate integration test suite against compiled binary
status: To Do
priority: 4
effort: Medium
assignee: claude
created_date: 2026-03-21
labels: [feature, core]
swimlane: Core Library
phase: 4
depends_on: ["008", "009", "010"]
source_file: dk-redo-implementation.md:182
---

## Summary

Un-skip and flesh out the integration tests from ticket 002. These tests
exercise the compiled dk-redo binary end-to-end with real files on disk,
verifying correct exit codes and stamp behavior across all core commands.

## Current State

Integration test stubs exist from ticket 002 in `test/integration_test.go`
(all skipped). All three core commands are now implemented. The binary can
be built with `just build`.

## Analysis & Recommendations

Integration test cases per `dk-redo-implementation.md:182-200`:

| Test | Scenario | Expected |
|---|---|---|
| First run | no stamps, run dk-ifchange | exit 0 |
| Unchanged | stamp, no file changes, dk-ifchange | exit 1 |
| File modified | stamp, modify file, dk-ifchange | exit 0 |
| File added to glob | stamp, create new .c matching glob | exit 0 |
| File removed from glob | stamp, delete a .c file | exit 0 |
| Dir file added | stamp with dir arg, add file | exit 0 |
| Dir file removed | stamp with dir arg, remove file | exit 0 |
| Missing file sentinel | stamp nonexistent input, create it | exit 0 |
| Stamp replace | dk-stamp twice, second replaces | second wins |
| Stamp append | dk-stamp --append twice | union |
| Always | dk-always then dk-ifchange | exit 0 |
| Always --all | dk-always --all | all removed |
| Error propagation | corrupt stamp | exit 2 |
| Subcommand style | dk-redo ifchange ... | same as dk-ifchange |
| Symlink style | symlink dk-ifchange → dk-redo | same behavior |

Each test should:
1. Create a temp directory with `.stamps/` and test files
2. Run the binary via `os/exec.Command`
3. Assert exit code, stdout/stderr content
4. Verify stamp file state on disk where relevant

`TestMain` builds the binary once into `t.TempDir()` before running tests.
Symlink tests create symlinks in the temp dir pointing to the binary.

## TDD Plan

### RED

All tests from the table above should be un-skipped and fully implemented.

### GREEN

1. Implement `TestMain` with binary build step
2. Implement helper: `runBinary(t, stampsDir, args...) (stdout, stderr, exitCode)`
3. Implement each test case from the table
4. Verify all pass with `just test-integration`

### REFACTOR

- Factor out common patterns (create stamp then check) into helpers
- Add timeout to binary execution to catch hangs
