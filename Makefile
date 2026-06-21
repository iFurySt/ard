PROJECT ?=
SLUG ?=
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell git log -1 --format=%cI 2>/dev/null || date -u +%Y-%m-%dT%H:%M:%SZ)
BUILD_LDFLAGS := -s -w -X github.com/ifuryst/ard/internal/buildinfo.Version=$(VERSION) -X github.com/ifuryst/ard/internal/buildinfo.Commit=$(COMMIT) -X github.com/ifuryst/ard/internal/buildinfo.Date=$(BUILD_DATE)

.PHONY: init new-history new-plan fmt fmt-check check-workflows check-public-surface test test-public-go-client test-integration test-e2e test-compose build sbom package release-dry-run docker-build

init:
	@if [ -z "$(PROJECT)" ]; then echo "usage: make init PROJECT=my-project"; exit 1; fi
	./scripts/init-project.sh "$(PROJECT)"

new-history:
	@if [ -z "$(SLUG)" ]; then echo "usage: make new-history SLUG=my-change"; exit 1; fi
	./scripts/new-history.sh "$(SLUG)"

new-plan:
	@if [ -z "$(SLUG)" ]; then echo "usage: make new-plan SLUG=my-plan"; exit 1; fi
	./scripts/new-exec-plan.sh "$(SLUG)"

fmt:
	gofmt -w cmd internal

fmt-check:
	./scripts/check-fmt.sh

check-workflows:
	go run ./internal/tools/workflowcheck

check-public-surface:
	go run ./internal/tools/publicsurface

test:
	go test ./...

test-public-go-client:
	./scripts/test-public-go-client.sh

test-integration:
	./scripts/test-integration.sh

test-e2e:
	./scripts/test-e2e-artifacts.sh

test-compose: build
	./scripts/test-compose.sh

build:
	go build -trimpath -ldflags "$(BUILD_LDFLAGS)" -o bin/ard ./cmd/ard
	go build -trimpath -ldflags "$(BUILD_LDFLAGS)" -o bin/ardctl ./cmd/ardctl
	go build -trimpath -ldflags "$(BUILD_LDFLAGS)" -o bin/ard-server ./cmd/ard-server

sbom:
	@mkdir -p dist
	go run ./internal/tools/sbom -version "$${VERSION:-dev}" -created "$${CREATED:-1970-01-01T00:00:00Z}" -out dist/sbom.spdx.json

package:
	./scripts/package-release.sh

release-dry-run:
	@if [ "$(origin VERSION)" = "file" ]; then echo "usage: VERSION=v0.1.0 make release-dry-run"; exit 1; fi
	VERSION="$(VERSION)" ./scripts/release-dry-run.sh

docker-build:
	docker build --build-arg VERSION="$(VERSION)" --build-arg COMMIT="$(COMMIT)" --build-arg BUILD_DATE="$(BUILD_DATE)" -t ard:local .
