# Plan: Build dk-redo from Backlog Tickets

## Research Summary

The repo contains 17 backlog tickets (001-017) that build a Go-based redo-style change detection tool from scratch. No Go code exists yet — only design docs and tickets. The tickets form a deep dependency DAG that must be respected.

## Dependency Graph (Layers)

```
Layer 0: 001 (scaffold — foundation for everything)
Layer 1: 002, 003, 004, 007 (each depends only on 001)
Layer 2: 005, 006 (005→003+004, 006→003)
Layer 3: 008, 009, 010, 013, 014, 015, 016, 017
         (various deps from layers 1-2)
Layer 4: 011, 012 (depend on 008+009+010)
```

## Work Units

Since tickets have deep interdependencies (each layer needs prior layers committed), I'll execute in 5 sequential layers with parallelism within each layer.

### Layer 0 (sequential — foundation)
1. **001-scaffold**: Initialize Go module, create directory structure, justfile, placeholder main.go. Files: go.mod, cmd/dk-redo/main.go, internal/*/placeholder.go, justfile, .gitignore

### Layer 1 (4 parallel agents)
2. **002-test-infra**: Create internal/testutil/ with helpers, integration test skeleton. Files: internal/testutil/, test/
3. **003-hasher**: Implement BLAKE3 file/dir hashing. Files: internal/hasher/
4. **004-encoding**: Implement label escaping and path encoding. Files: internal/stamp/encoding.go
5. **007-cli-dispatch**: Implement argv[0] dispatch and flag parsing. Files: cmd/dk-redo/main.go

### Layer 2 (2 parallel agents)
6. **005-stamp**: Implement stamp read/write/compare/append. Files: internal/stamp/
7. **006-resolve**: Implement resolve package for input args. Files: internal/resolve/

### Layer 3 (8 parallel agents)
8. **008-ifchange**: Implement dk-ifchange command. Files: cmd/dk-redo/ (ifchange.go)
9. **009-stamp-cmd**: Implement dk-stamp command. Files: cmd/dk-redo/ (stamp_cmd.go)
10. **010-always**: Implement dk-always command. Files: cmd/dk-redo/ (always.go)
11. **013-ood**: Implement dk-ood command. Files: cmd/dk-redo/ (ood.go)
12. **014-affects**: Implement dk-affects command. Files: cmd/dk-redo/ (affects.go)
13. **015-sources**: Implement dk-sources command. Files: cmd/dk-redo/ (sources.go)
14. **016-dot**: Implement dk-dot command. Files: cmd/dk-redo/ (dot.go)
15. **017-coverage**: Add coverage targets to justfile. Files: justfile

### Layer 4 (2 parallel agents)
16. **011-integration**: Activate integration test suite. Files: test/
17. **012-release**: Add release targets. Files: justfile, cmd/dk-redo/

## E2E Test Recipe

Skip e2e for individual units — each agent runs `go test ./internal/...` for unit tests. After all layers complete, I'll run the full build and test suite (`just build && just test`) as a final verification step.

## Worker Instructions (shared template)

Each worker will receive:
- The full ticket content from backlog/*.md
- The relevant design docs from dk-redo-implementation.md
- The current state of all files they need to modify
- Instructions to implement, test, commit, and push

Since worktrees can't see sibling changes, I'll execute layers sequentially — committing and merging each layer's results before starting the next. Within each layer, agents run in parallel worktrees.

**Adaptation note**: Because these tickets deeply depend on each other, I will NOT create separate PRs per unit. Instead, each layer's agents work in worktrees, I merge their changes back to the feature branch, and commit per-layer. The final result is one branch with all tickets implemented.
