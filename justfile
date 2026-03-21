# dk-redo justfile

# Build a static binary
build:
    CGO_ENABLED=0 go build -ldflags="-s -w" -o dk-redo ./cmd/dk-redo

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

# Clean build artifacts
clean:
    rm -f dk-redo
    rm -rf .stamps/
