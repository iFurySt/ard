# ard

Neutral, self-hosted Agentic Resource Discovery registry and management toolkit.

`ard` is an independent open-source implementation of the ARD ecosystem. It is being
designed for enterprises and agent platforms that need to discover, verify, govern, and
manage agentic resources across MCP, A2A, Skills, OpenAPI, and future protocols.

MCP, A2A, Skills, and APIs define how capabilities are used. ARD defines how they are
found.

## What It Will Provide

- A self-hosted ARD registry server.
- A standards-aligned ARD client and CLI.
- Catalog crawling for `/.well-known/ai-catalog.json`.
- Resource onboarding for MCP, A2A, Skills, OpenAPI, and URL artifacts.
- Validation, verification, policy, and federation workflows.

## Target CLI

```sh
ard serve
ard add catalog https://example.com/.well-known/ai-catalog.json
ard add mcp https://example.com/mcp/server.json
ard crawl
ard search "query observability logs" --kind mcp
ard verify https://example.com/.well-known/ai-catalog.json
```

## Try It

```sh
make build
make test-integration

bin/ard --database-url "$DATABASE_URL" add catalog ./internal/catalog/testdata/acme-ai-catalog.json
bin/ard verify catalog ./internal/catalog/testdata/acme-ai-catalog.json
bin/ard --database-url "$DATABASE_URL" crawl https://example.com/
bin/ard --database-url "$DATABASE_URL" serve
bin/ard search "weather forecast" --kind mcp --json
```

## Status

This repository is in early implementation. Current milestones include a Go CLI,
Gin-based registry server, GORM/Postgres persistence, catalog import, well-known catalog
crawl, catalog verification, ARD search, browse, and explore facets.

Implementation should track the upstream
[`ards-project/ard-spec`](https://github.com/ards-project/ard-spec) closely, including
`urn:air:` identifiers, `application/mcp-server-card+json`, and the official conformance
tools.

See:

- [Architecture](docs/ARCHITECTURE.md)
- [Product Sense](docs/PRODUCT_SENSE.md)
- [ARD Spec Working Notes](docs/references/ard-spec-working-notes.md)

## License

[Apache-2.0](LICENSE)
