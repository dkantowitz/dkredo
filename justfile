# dk-redo justfile

# Build static binary
build:
    CGO_ENABLED=0 go build -ldflags="-s -w" -o dk-redo ./cmd/dk-redo

# Run all tests
test: test-unit test-integration

# Run unit tests
test-unit:
    go test ./internal/...

# Run integration tests
test-integration:
    go test -tags integration ./test/...

# Run benchmarks
test-bench:
    go test -bench=. -benchtime=3s ./...

# Clean build artifacts
clean:
    rm -f dk-redo && rm -rf .stamps/
