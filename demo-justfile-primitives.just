# demo-justfile-primitives.just
#
# Approach: No dk-ifchange / dk-stamp / dk-always commands.
# Everything composes dkredo's +primitives directly or via thin just recipes.
#
# Three helper recipes are kept for the most common patterns:
#   _guard  — add names + check (the "should I rebuild?" test)
#   _record — stamp facts after a build
#   _names  — query file names from a stamp
#
# Everything else calls dkredo directly.

set guards

# ---------------------------------------------------------------------------
# Helper recipes — kept because they appear in nearly every build recipe
# ---------------------------------------------------------------------------

# Guard: add files to stamp, check if anything changed
[no-exit-message]
_guard label +files:
    dkredo {{label}} +add-names {{files}} +check

# Record facts after a build
_record label +files:
    dkredo {{label}} +stamp {{files}}

# Query file names from the stamp
_names label filter='':
    dkredo {{label}} +names {{filter}}

# ---------------------------------------------------------------------------
# Build recipes — helper style (uses _guard / _record / _names)
# ---------------------------------------------------------------------------

# C compilation with gcc dependency discovery
compile:
    just _guard out.bin $(find src -name '*.c')
    gcc -o out.bin -MD -MF .deps/out.d $(just _names out.bin .c)
    just _record out.bin -M .deps/out.d

# Deploy only when source or config changed
deploy-staging:
    just _guard deploy-staging src/*.py config/staging.yaml
    kubectl apply -f k8s/staging/
    just _record deploy-staging src/*.py config/staging.yaml

# ---------------------------------------------------------------------------
# Build recipes — direct dkredo style (no helper recipes)
# ---------------------------------------------------------------------------

# C compilation — same as 'compile' but calling dkredo directly
compile-direct:
    dkredo out.bin +add-names $(find src -name '*.c') +check
    gcc -o out.bin -MD -MF .deps/out.d $(dkredo out.bin +names .c)
    dkredo out.bin +stamp -M .deps/out.d

# Strict guard using +sync-names to prune deleted files
compile-strict:
    dkredo out.bin +sync-names $(find src -name '*.c') +check
    gcc -o out.bin -MD -MF .deps/out.d $(dkredo out.bin +names .c)
    dkredo out.bin +stamp -M .deps/out.d

# Multi-target firmware build
firmware:
    dkredo firmware.bin +add-names src/*.c include/*.h libs/*.a +check
    gcc -MD -MF .deps/firmware.d -c src/*.c -Iinclude/
    ld -o firmware.bin *.o libs/*.a
    dkredo firmware.bin +stamp src/*.c include/*.h libs/*.a -M .deps/firmware.d

# Lint only when source changed
lint:
    dkredo lint-check +add-names src/*.py +check
    ruff check src/
    dkredo lint-check +stamp src/*.py

# Docker image rebuild only when Dockerfile or source changed
docker-build tag='latest':
    dkredo docker-image +add-names Dockerfile src/**/* +check
    docker build -t myapp:{{tag}} .
    dkredo docker-image +stamp Dockerfile src/**/*

# Test only when source or test files changed
test:
    dkredo test-check +add-names src/*.py tests/*.py +check
    pytest tests/
    dkredo test-check +stamp src/*.py tests/*.py

# ---------------------------------------------------------------------------
# Maintenance — always direct dkredo calls
# ---------------------------------------------------------------------------

# Force-rebuild specific targets
clean *targets:
    #!/usr/bin/env bash
    for t in {{targets}}; do
        dkredo "$t" +clear
    done

# Force-rebuild everything
clean-all:
    #!/usr/bin/env bash
    for t in out.bin firmware.bin deploy-staging lint-check docker-image test-check; do
        dkredo "$t" +clear
    done

# Show what a target is tracking
inspect label:
    @echo "=== Names ==="
    dkredo {{label}} +names
    @echo "=== Facts ==="
    dkredo {{label}} +facts
