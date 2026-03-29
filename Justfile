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

# Get the latest version tag, defaulting to v0.0.0
_latest-version:
    @git tag -l 'v*' --sort=-v:refname | head -1 | grep . || echo "v0.0.0"

# Tag and push an arbitrary version (e.g. just release 0.2-beta1)
release version: test
    #!/usr/bin/env bash
    set -euo pipefail
    tag="v{{version}}"
    echo "Releasing $tag"
    git tag -a "$tag" -m "Release $tag"
    git push origin "$tag"
    echo "Pushed tag $tag — release workflow will build and publish"

# Bump minor version and push tag (v0.1.0 → v0.2.0)
release-minor: test
    #!/usr/bin/env bash
    set -euo pipefail
    current=$(just _latest-version)
    major=$(echo "$current" | sed 's/v//' | cut -d. -f1)
    minor=$(echo "$current" | sed 's/v//' | cut -d. -f2)
    next="v${major}.$((minor + 1)).0"
    echo "Releasing $current → $next"
    git tag -a "$next" -m "Release $next"
    git push origin "$next"
    echo "Pushed tag $next — release workflow will build and publish"

# Bump major version and push tag (v0.2.0 → v1.0.0)
release-major: test
    #!/usr/bin/env bash
    set -euo pipefail
    current=$(just _latest-version)
    major=$(echo "$current" | sed 's/v//' | cut -d. -f1)
    next="v$((major + 1)).0.0"
    echo "Releasing $current → $next"
    git tag -a "$next" -m "Release $next"
    git push origin "$next"
    echo "Pushed tag $next — release workflow will build and publish"

cover-check:
    #!/usr/bin/env bash
    set -euo pipefail
    go test -coverprofile=coverage.out -covermode=atomic ./internal/...
    pct=$(go tool cover -func=coverage.out | grep ^total | awk '{print $NF}' | tr -d '%')
    int=${pct%%.*}
    if [ "$int" -lt 80 ]; then
        echo "FAIL: coverage ${int}% < 80%"
        exit 1
    else
        echo "OK: coverage ${int}%"
    fi
