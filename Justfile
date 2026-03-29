default: build

build:
    CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)" -o dkredo ./cmd/dkredo

test:
    go vet ./...
    go test -race ./...

cover:
    go test -coverprofile=coverage.out -covermode=atomic ./internal/...
    go tool cover -func=coverage.out

cover-html:
    go test -coverprofile=coverage.out -covermode=atomic ./internal/...
    go tool cover -html=coverage.out -o coverage.html

test-integration: build
    just -f test/justfile test-all

test-all: test test-integration

cover-check:
    go test -coverprofile=coverage.out -covermode=atomic ./internal/...
    @go tool cover -func=coverage.out | grep ^total | awk '{print $$3}' | \
        awk -F. '{if ($$1 < 80) {print "FAIL: coverage " $$1 "% < 80%"; exit 1} else {print "OK: coverage " $$1 "%"}}'
