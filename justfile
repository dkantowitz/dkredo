# dk-redo justfile

# Version from git
version := `git describe --tags --always --dirty 2>/dev/null || echo dev`

# Build a static binary
build:
    CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)" -o dk-redo ./cmd/dk-redo

# Run all tests
test: test-unit test-integration

# Run unit tests
test-unit:
    go test ./internal/...

# Run integration tests (builds binary, then tests)
test-integration: build
    go test -tags integration ./test/...

# Run benchmarks
test-bench: build
    go test -tags integration -bench=. -benchtime=3s -run='^$' ./test/...

# Run coverage analysis
cover:
    go test -coverprofile=coverage.out -covermode=atomic ./internal/hasher ./internal/resolve ./internal/stamp
    go tool cover -func=coverage.out

# Generate HTML coverage report
cover-html:
    go test -coverprofile=coverage.out -covermode=atomic ./internal/hasher ./internal/resolve ./internal/stamp
    go tool cover -html=coverage.out -o coverage.html

# Check coverage meets threshold (80%)
cover-check:
    #!/usr/bin/env bash
    set -euo pipefail
    go test -coverprofile=coverage.out -covermode=atomic ./internal/hasher ./internal/resolve ./internal/stamp
    pct=$(go tool cover -func=coverage.out | grep ^total | awk '{print $3}' | tr -d '%')
    int=${pct%.*}
    if [ "$int" -lt 80 ]; then
        echo "FAIL: coverage ${pct}% < 80%"
        exit 1
    else
        echo "OK: coverage ${pct}%"
    fi

# Build release binaries for linux and windows
release:
    mkdir -p dist
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version={{version}}" -o dist/dk-redo-linux-amd64 ./cmd/dk-redo
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X main.version={{version}}" -o dist/dk-redo-windows-amd64.exe ./cmd/dk-redo

# Build release binaries for macOS (optional)
release-macos:
    mkdir -p dist
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.version={{version}}" -o dist/dk-redo-darwin-amd64 ./cmd/dk-redo
    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.version={{version}}" -o dist/dk-redo-darwin-arm64 ./cmd/dk-redo

# Clean build artifacts
clean:
    rm -f dk-redo
    rm -rf .stamps/
    rm -rf dist/
