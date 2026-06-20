PROJECT ?=
SLUG ?=

.PHONY: init new-history new-plan fmt fmt-check test test-public-go-client test-integration test-e2e test-compose build sbom package docker-build

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
	go build -o bin/ard ./cmd/ard
	go build -o bin/ardctl ./cmd/ardctl
	go build -o bin/ard-server ./cmd/ard-server

sbom:
	@mkdir -p dist
	go run ./internal/tools/sbom -version "$${VERSION:-dev}" -created "$${CREATED:-1970-01-01T00:00:00Z}" -out dist/sbom.spdx.json

package:
	./scripts/package-release.sh

docker-build:
	docker build -t ard:local .
