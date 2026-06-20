# ard

Neutral self-hosted registry and toolkit for Agentic Resource Discovery.

`ard` is an independent open-source implementation of the ARD ecosystem. It is being
designed for enterprises and agent platforms that need to discover, verify, govern, and
manage agentic resources across MCP, A2A, Skills, OpenAPI, and future protocols.

MCP, A2A, Skills, and APIs define how capabilities are used. ARD defines how they are
found.

## What It Will Provide

- A self-hosted ARD registry server.
- A standards-aligned ARD Go client and CLI.
- Catalog crawling for `/.well-known/ai-catalog.json`.
- Resource onboarding for MCP, A2A, Skills, OpenAPI, and URL artifacts.
- Validation, verification, policy, and federation workflows.

## Target CLI

```sh
ard serve
ard-server --addr :8080 --admin-token "$ARD_ADMIN_TOKEN"
ard-server --admin-tokens-file ./admin-tokens.json
ard-server --policy-file ./policy.json --admin-token "$ARD_ADMIN_TOKEN"
ard add catalog https://example.com/.well-known/ai-catalog.json
ardctl add catalog https://example.com/.well-known/ai-catalog.json
ard add mcp https://example.com/mcp/server.json
ard add mcp https://example.com/mcp/server.json --pin-source-digest
ard add a2a https://example.com/.well-known/agent-card.json
ard add skill https://example.com/skills/open-browser-use/SKILL.md
ard add openapi https://example.com/openapi.json
ard crawl
ardctl list --kind mcp
ardctl list --filter "publisherId = 'github.com'" --order-by "displayName DESC"
ardctl list --filter "tags = 'skill' AND metadata.adapter = 'skill'"
ardctl list --filter "displayName contains 'browser' AND type != 'application/mcp-server-card+json'"
ardctl list --filter "type = 'application/openapi+json' OR (tags = 'skill' AND metadata.adapter = 'skill')"
ardctl browse --registry-url https://registry.example.com --filter "publisherId = 'github.com'" --json
ardctl remove urn:air:example.com:server:weather --yes
ardctl export catalog -o ai-catalog.json
ardctl admin list --registry-url https://registry.example.com --admin-token "$ARD_ADMIN_TOKEN"
ardctl admin add catalog ./ai-catalog.json --registry-url https://registry.example.com
ardctl admin status urn:air:example.com:server:weather disabled --registry-url https://registry.example.com
ardctl admin review approve urn:air:example.com:server:weather --registry-url https://registry.example.com
ardctl admin audit --registry-url https://registry.example.com --admin-token "$ARD_ADMIN_TOKEN"
ardctl admin audit --limit 10 --page-token "$PAGE_TOKEN" --registry-url https://registry.example.com
ardctl admin audit --verify-chain --registry-url https://registry.example.com
ard search "query observability logs" --kind mcp
ard search "query observability logs" --limit 10 --page-token "$PAGE_TOKEN" --json
ard search "query observability logs" --federation referrals --json
ard search "query observability logs" --federation auto --json
ard verify catalog https://example.com/.well-known/ai-catalog.json
ard verify catalog ./ai-catalog.json --source-digests
```

## Try It

```sh
make build
make test-integration
make test-e2e
make test-compose
make test-public-go-client
make fmt-check

bin/ard --database-url "$DATABASE_URL" add catalog ./internal/catalog/testdata/acme-ai-catalog.json
bin/ard --database-url "$DATABASE_URL" add mcp ./internal/adapters/testdata/mcp-server-card.json
bin/ard --database-url "$DATABASE_URL" add a2a ./internal/adapters/testdata/a2a-agent-card.json
bin/ard --database-url "$DATABASE_URL" add skill ./internal/adapters/testdata/open-browser-use/SKILL.md
bin/ardctl --database-url "$DATABASE_URL" list --kind mcp
bin/ardctl --database-url "$DATABASE_URL" list --filter "type = 'application/mcp-server-card+json'" --order-by "displayName DESC"
bin/ardctl --database-url "$DATABASE_URL" list --filter "capabilities = 'ForecastTool'"
bin/ardctl --database-url "$DATABASE_URL" list --filter "tags contains 'weath' AND capabilities != 'BlockedTool'"
bin/ardctl --database-url "$DATABASE_URL" list --filter "type = 'application/openapi+json' OR (tags = 'skill' AND metadata.adapter = 'skill')"
bin/ard verify catalog ./internal/catalog/testdata/acme-ai-catalog.json
bin/ard --database-url "$DATABASE_URL" crawl https://example.com/
bin/ardctl --database-url "$DATABASE_URL" export catalog -o ai-catalog.json

# terminal 1
bin/ard-server --database-url "$DATABASE_URL" --admin-token "$ARD_ADMIN_TOKEN"

# terminal 2
bin/ardctl browse --registry-url http://127.0.0.1:8080 --filter "type = 'application/mcp-server-card+json'" --json
bin/ardctl search "weather forecast" --kind mcp --json
bin/ardctl admin list --admin-token "$ARD_ADMIN_TOKEN"
```

## Go SDK

```go
registry, _ := client.New("https://registry.example.com")
results, _ := registry.Search(ctx, ard.SearchRequest{
	Query: ard.SearchQuery{Text: "weather"},
})
_ = results

admin, _ := client.New("https://registry.example.com", client.WithAdminToken(adminToken))
entries, _ := admin.AdminList(ctx, client.AdminListOptions{Kind: "mcp"})
_ = entries
```

Import paths:

- `github.com/ifuryst/ard/pkg/ard`
- `github.com/ifuryst/ard/pkg/client`

## Status

This repository is in early implementation. Current milestones include a Go CLI,
Gin-based registry server, GORM/Postgres persistence, catalog import, well-known
catalog crawl and publication, MCP/A2A/Skill/OpenAPI artifact onboarding, catalog
verification, ARD search, browse, and explore facets, a public Go SDK, catalog export,
field-filtered local listing, remote public browsing, entry removal, and token-protected admin
API routes with `ardctl admin` and Go SDK clients. Admin flows can disable, reactivate, filter
entries, apply ingestion policy, review pending entries with decision reasons, and
inspect mutation audit events without exposing inactive resources through public
discovery. Audit events are hash-chained and can be verified through
`ardctl admin audit --verify-chain`. Server
deployments can use a single admin token or reloadable role-scoped admin token files. URL
artifacts can be pinned and verified with `trustManifest.sourceDigest`. Search supports
client-followed federation referrals, bounded server-side `federation=auto` upstream
score-ranked result merging without exposing local-only page tokens as federated cursors,
and opaque `pageToken` pagination for local search, list, review, and audit responses.
The registry also exposes request correlation, JSON access logs, and
W3C `traceparent` propagation, plus Prometheus-style `/metrics` with HTTP duration
histograms and Go runtime gauges. `ardctl admin --request-id` can carry one correlation
ID across remote artifact fetches and admin API calls.
It builds three entry points: `ard` for the combined toolkit, `ardctl` for CLI/client
operations, and `ard-server` for the registry server. CI runs formatting checks, tests,
public Go client import checks, builds, and Postgres integration tests.
`make test-e2e` runs the real artifact onboarding flow with live MCP, Skill, OpenAPI,
policy-gate examples, a local upstream registry for auto federation, and an external
Go admin SDK check against the live registry.
`make test-compose` builds the container image and verifies a compose-backed registry
against Postgres.

Implementation should track the upstream
[`ards-project/ard-spec`](https://github.com/ards-project/ard-spec) closely, including
`urn:air:` identifiers, `application/mcp-server-card+json`, and the official conformance
tools.

See:

- [Architecture](docs/ARCHITECTURE.md)
- [Admin Authorization](docs/ADMIN_AUTH.md)
- [Deployment](docs/DEPLOYMENT.md)
- [Product Sense](docs/PRODUCT_SENSE.md)
- [Policy](docs/POLICY.md)
- [Trust Verification](docs/TRUST.md)
- [ARD Spec Working Notes](docs/references/ard-spec-working-notes.md)

## License

[Apache-2.0](LICENSE)
