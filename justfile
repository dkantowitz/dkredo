# dk-redo justfile

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
    go test -coverprofile=coverage.out -covermode=atomic ./internal/...
    go tool cover -func=coverage.out

# Generate HTML coverage report
cover-html:
    go test -coverprofile=coverage.out -covermode=atomic ./internal/...
    go tool cover -html=coverage.out -o coverage.html

# Check coverage meets threshold (80%)
cover-check:
    go test -coverprofile=coverage.out -covermode=atomic ./internal/...
    @go tool cover -func=coverage.out | grep ^total | awk '{print $$3}' | awk -F. '{if ($$1 < 80) {print "FAIL: coverage " $$1 "% < 80%"; exit 1} else {print "OK: coverage " $$1 "%"}}'

# Clean build artifacts
clean:
    rm -f dk-redo
    rm -rf .stamps/
