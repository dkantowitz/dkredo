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
depends_on: ["007", "008", "009", "010"]
source_file: dk-redo-implementation.md:182
---

## Summary

Un-skip and flesh out the integration tests from ticket 002. These tests
exercise the compiled dk-redo binary end-to-end with real files on disk,
verifying correct exit codes and stamp behavior across all core commands.

Also activate the performance benchmark suite to establish regression baselines.

## Current State

Integration test skeleton exists from ticket 002 in `test/integration_test.go`.
All three core commands are now implemented. The binary can be built with
`just build`.

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
| Error propagation | corrupt stamp | exit 0 (treated as changed) |
| Subcommand style | dk-redo ifchange ... | same as dk-ifchange |
| Symlink style | symlink dk-ifchange → dk-redo | same behavior |
| Label with slash | label "output/config.json" | stamp at .stamps/output%2Fconfig.json |
| Stdin combined | dk-ifchange label a.c - b.c | all inputs processed |
| Force changed | dk-ifchange -n label files | always exit 0 |
| Unknown symlink | symlink dk-bogus → dk-redo | exit 2 with usage |
| Unknown facts | stamp with unknown fact keys | exit 0, warning on stderr |
| Adversarial stamp | binary/malformed stamp | exit 0 (treated as changed) |

### Performance benchmarks

| Benchmark | Setup | Target |
|---|---|---|
| BenchmarkIfchangeUnchanged10 | 10 files, stamp exists | < 10ms |
| BenchmarkIfchangeUnchanged300 | 300 files across 10 labels (30 each) | < 300ms total |
| BenchmarkStamp100 | 100 small files | < 50ms |
| BenchmarkStartupOverhead | no-op invocation (--help) | < 5ms |

The 300-dependency benchmark is the primary regression target.

### Known test deficiencies

**Atomic write testing:** The atomic write mechanism (temp file + rename)
is inherently difficult to test for crash safety. Tests verify the write-
then-rename sequence occurs but cannot simulate a crash between the two
operations. This is an accepted deficiency — the mechanism is well-established.

Each test should:
1. Create a temp directory with `.stamps/` and test files
2. Run the binary via `os/exec.Command`
3. Assert exit code, stdout/stderr content
4. Verify stamp file state on disk where relevant

`TestMain` builds the binary once into `t.TempDir()` before running tests.
Symlink tests create symlinks in the temp dir pointing to the binary.

## TDD Plan

### RED

All tests from the table above should be implemented.

### GREEN

1. Implement `TestMain` with binary build step
2. Implement helper: `runBinary(t, stampsDir, args...) (stdout, stderr, exitCode)`
3. Implement each test case from the table
4. Implement benchmark functions
5. Verify all pass with `just test-integration`
6. Verify benchmarks pass with `just test-bench`

### REFACTOR

- Factor out common patterns (create stamp then check) into helpers
- Add timeout to binary execution to catch hangs
