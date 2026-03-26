# demo-justfile-primitives.just
#
# Approach: No dk-ifchange / dk-stamp / dk-always commands.
# Everything composes dkredo's +primitives through parameterized just recipes.

set guards

# ---------------------------------------------------------------------------
# Primitive recipes — thin wrappers over +operations
# ---------------------------------------------------------------------------

# Guard: add files to stamp, check if anything changed (replaces dk-ifchange)
[no-exit-message]
_guard label +files:
    dkredo {{label}} +add-names {{files}} +check

# Guard with sync semantics: exact file list, prune stale entries
[no-exit-message]
_guard-sync label +files:
    dkredo {{label}} +sync-names {{files}} +check

# Record facts after a build (replaces dk-stamp)
_record label +files:
    dkredo {{label}} +stamp {{files}}

# Record facts from a gcc/clang depfile
_record-depfile label depfile:
    dkredo {{label}} +stamp -M {{depfile}}

# Record facts from both explicit files and a depfile
_record-both label depfile +files:
    dkredo {{label}} +stamp {{files}} -M {{depfile}}

# Force rebuild next time (replaces dk-always)
_invalidate label:
    dkredo {{label}} +clear

# Query file names from the stamp (replaces dk-fnames)
_names label filter='':
    dkredo {{label}} +names {{filter}}

# Dump facts for debugging
_facts label *files:
    dkredo {{label}} +facts {{files}}

# ---------------------------------------------------------------------------
# Build recipes — compose the primitives
# ---------------------------------------------------------------------------

# C compilation with gcc dependency discovery
compile:
    just _guard out.bin $(find src -name '*.c')
    gcc -o out.bin -MD -MF .deps/out.d $(just _names out.bin .c)
    just _record-depfile out.bin .deps/out.d

# Same thing using sync (prunes deleted .c files proactively)
compile-strict:
    just _guard-sync out.bin $(find src -name '*.c')
    gcc -o out.bin -MD -MF .deps/out.d $(just _names out.bin .c)
    just _record-depfile out.bin .deps/out.d

# Multi-target firmware build
firmware:
    just _guard firmware.bin src/*.c include/*.h libs/*.a
    gcc -MD -MF .deps/firmware.d -c src/*.c -Iinclude/
    ld -o firmware.bin *.o libs/*.a
    just _record-both firmware.bin .deps/firmware.d src/*.c include/*.h libs/*.a

# Deploy only when source or config changed
deploy-staging:
    just _guard deploy-staging src/*.py config/staging.yaml
    kubectl apply -f k8s/staging/
    just _record deploy-staging src/*.py config/staging.yaml

# Lint only when source changed
lint:
    just _guard lint-check src/*.py
    ruff check src/
    just _record lint-check src/*.py

# Docker image rebuild only when Dockerfile or source changed
docker-build tag='latest':
    just _guard docker-image Dockerfile src/**/*
    docker build -t myapp:{{tag}} .
    just _record docker-image Dockerfile src/**/*

# Test only when source or test files changed
test:
    just _guard test-check src/*.py tests/*.py
    pytest tests/
    just _record test-check src/*.py tests/*.py

# ---------------------------------------------------------------------------
# Maintenance
# ---------------------------------------------------------------------------

# Force-rebuild specific targets
clean *targets:
    #!/usr/bin/env bash
    for t in {{targets}}; do
        dkredo "$t" +clear
    done

# Force-rebuild everything
clean-all:
    just clean out.bin firmware.bin deploy-staging lint-check docker-image test-check

# Show what a target is tracking
inspect label:
    @echo "=== Names ==="
    just _names {{label}}
    @echo "=== Facts ==="
    just _facts {{label}}
