# ard

Neutral self-hosted registry and toolkit for Agentic Resource Discovery.

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
ard-server --addr :8080 --admin-token "$ARD_ADMIN_TOKEN"
ard add catalog https://example.com/.well-known/ai-catalog.json
ardctl add catalog https://example.com/.well-known/ai-catalog.json
ard add mcp https://example.com/mcp/server.json
ard add a2a https://example.com/.well-known/agent-card.json
ard add skill https://example.com/skills/open-browser-use/SKILL.md
ard add openapi https://example.com/openapi.json
ard crawl
ardctl list --kind mcp
ardctl remove urn:air:example.com:server:weather --yes
ardctl export catalog -o ai-catalog.json
ardctl admin list --registry-url https://registry.example.com --admin-token "$ARD_ADMIN_TOKEN"
ardctl admin add catalog ./ai-catalog.json --registry-url https://registry.example.com
ardctl admin status urn:air:example.com:server:weather disabled --registry-url https://registry.example.com
ardctl admin audit --registry-url https://registry.example.com --admin-token "$ARD_ADMIN_TOKEN"
ard search "query observability logs" --kind mcp
ard verify catalog https://example.com/.well-known/ai-catalog.json
```

## Try It

```sh
make build
make test-integration
make test-e2e
make fmt-check

bin/ard --database-url "$DATABASE_URL" add catalog ./internal/catalog/testdata/acme-ai-catalog.json
bin/ard --database-url "$DATABASE_URL" add mcp ./internal/adapters/testdata/mcp-server-card.json
bin/ard --database-url "$DATABASE_URL" add a2a ./internal/adapters/testdata/a2a-agent-card.json
bin/ard --database-url "$DATABASE_URL" add skill ./internal/adapters/testdata/open-browser-use/SKILL.md
bin/ardctl --database-url "$DATABASE_URL" list --kind mcp
bin/ard verify catalog ./internal/catalog/testdata/acme-ai-catalog.json
bin/ard --database-url "$DATABASE_URL" crawl https://example.com/
bin/ardctl --database-url "$DATABASE_URL" export catalog -o ai-catalog.json

# terminal 1
bin/ard-server --database-url "$DATABASE_URL" --admin-token "$ARD_ADMIN_TOKEN"

# terminal 2
bin/ardctl search "weather forecast" --kind mcp --json
bin/ardctl admin list --admin-token "$ARD_ADMIN_TOKEN"
```

## Status

This repository is in early implementation. Current milestones include a Go CLI,
Gin-based registry server, GORM/Postgres persistence, catalog import, well-known catalog
crawl, MCP/A2A/Skill/OpenAPI artifact onboarding, catalog verification, ARD search, browse, and
explore facets, catalog export, local listing, entry removal, and token-protected admin
API routes with an `ardctl admin` client. Admin flows can disable, reactivate, filter
entries, and inspect mutation audit events without exposing inactive resources through
public discovery. It builds three entry points: `ard` for the combined toolkit, `ardctl`
for CLI/client operations, and `ard-server` for the registry server. CI runs formatting
checks, tests, builds, and Postgres integration tests.
`make test-e2e` runs the real artifact onboarding flow with live MCP and Skill examples.

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
