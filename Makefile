PROJECT ?=
SLUG ?=

.PHONY: init new-history new-plan fmt test test-integration build

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

test:
	go test ./...

test-integration:
	./scripts/test-integration.sh

build:
	go build -o bin/ard ./cmd/ard
