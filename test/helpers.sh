#!/usr/bin/env bash
# Shared test helpers — sourced by justfile test recipes

setup_tmpdir() {
    set -euo pipefail
    T=$(mktemp -d)
    trap "rm -rf $T" EXIT
    cd "$T"
}

# expect_exit EXIT_CODE COMMAND...
expect_exit() {
    local want=$1; shift
    local rc=0; "$@" || rc=$?
    [ "$rc" -eq "$want" ] || { echo "FAIL: expected exit $want, got $rc"; exit 1; }
}
