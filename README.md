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
ard version
ardctl version --json
ard add catalog https://example.com/.well-known/ai-catalog.json
ardctl add catalog https://example.com/.well-known/ai-catalog.json
ard add mcp https://example.com/mcp/server.json
ard add mcp https://example.com/mcp/server.json --pin-source-digest
ard add a2a https://example.com/.well-known/agent-card.json
ard add skill https://example.com/skills/open-browser-use/SKILL.md
ard add openapi https://example.com/openapi.json
ard crawl
ardctl health --registry-url https://registry.example.com --json
ardctl metrics --registry-url https://registry.example.com
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
ard verify catalog ./ai-catalog.json --require-source-digests
ard verify catalog ./ai-catalog.json --attestation-digests
ard verify catalog ./ai-catalog.json --require-attestation-digests
ard verify catalog ./ai-catalog.json --provenance-digests
ard verify catalog ./ai-catalog.json --require-provenance-digests
ard --policy-file ./policy.json verify catalog ./ai-catalog.json
ard verify catalog ./ai-catalog.json --jws-trust-anchors ./trust-anchors.json
ard verify catalog ./ai-catalog.json --jws-remote-jwks https://example.com/.well-known/jwks.json
ard verify catalog ./ai-catalog.json --jws-discover-did-web
ard verify catalog ./ai-catalog.json --jws-discover-oidc
ard verify catalog ./ai-catalog.json --jws-discover-spiffe
ard verify catalog ./ai-catalog.json --jws-discover-tls-cert
ard verify catalog ./ai-catalog.json --jws-discover-tls-cert --jws-tls-spki-pin example.com=sha256:<hex> --require-jws-tls-spki-pins
ard verify catalog ./ai-catalog.json --jws-trust-anchors ./trust-anchors.json --require-jws-signatures
```

## Try It

```sh
make build
make sbom
make package
VERSION=v0.1.0 make release-dry-run
make test-integration
make test-e2e
make test-compose
make test-public-go-client
make fmt-check

bin/ard version
bin/ardctl version --json
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
bin/ard verify catalog ./ai-catalog.json --require-source-digests
bin/ard --policy-file ./policy.json verify catalog ./ai-catalog.json
bin/ard verify catalog ./ai-catalog.json --attestation-digests
bin/ard verify catalog ./ai-catalog.json --provenance-digests
bin/ard verify catalog ./ai-catalog.json --jws-trust-anchors ./trust-anchors.json
bin/ard verify catalog ./ai-catalog.json --jws-remote-jwks https://example.com/.well-known/jwks.json
bin/ard verify catalog ./ai-catalog.json --jws-discover-did-web
bin/ard verify catalog ./ai-catalog.json --jws-discover-oidc
bin/ard verify catalog ./ai-catalog.json --jws-discover-spiffe
bin/ard verify catalog ./ai-catalog.json --jws-discover-tls-cert
bin/ard verify catalog ./ai-catalog.json --jws-discover-tls-cert --jws-tls-spki-pin example.com=sha256:<hex>
bin/ard --database-url "$DATABASE_URL" crawl https://example.com/
bin/ardctl --database-url "$DATABASE_URL" export catalog -o ai-catalog.json

# terminal 1
bin/ard-server --database-url "$DATABASE_URL" --admin-token "$ARD_ADMIN_TOKEN"

# terminal 2
bin/ardctl health --registry-url http://127.0.0.1:8080 --json
bin/ardctl metrics --registry-url http://127.0.0.1:8080
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

Compatibility policy: [docs/SDK_COMPATIBILITY.md](docs/SDK_COMPATIBILITY.md).

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
artifacts can be pinned and verified with `trustManifest.sourceDigest`; ingestion policy
can require trust metadata before persistence. Detached JWS
`trustManifest.signature` values can be verified against explicit Ed25519, local JWKS,
explicit HTTPS remote JWKS, discovered `did:web` DID document keys, or discovered OIDC
`jwks_uri` keys. SPIFFE bundle JWKS and HTTPS TLS leaf certificate Ed25519 keys can also
be used as explicit verification anchors; TLS discovery supports optional SPKI SHA-256
pins. Attestation documents can be fetched and verified against pinned
`trustManifest.attestations[].digest` values, and HTTP(S) provenance sources can be verified against pinned
`trustManifest.provenance[].sourceDigest` values.
Search supports
client-followed federation referrals, bounded server-side `federation=auto` upstream
score-ranked result merging with opaque cross-registry page tokens, and `pageToken`
pagination for local search, list, review, and audit responses.
The registry also exposes request correlation, JSON access logs, and
W3C `traceparent` propagation, plus Prometheus-style `/metrics` with HTTP duration
histograms and Go runtime gauges. Optional OTLP/HTTP trace export can send server spans
to an OpenTelemetry collector. `ardctl admin --request-id` can carry one correlation ID
across remote artifact fetches and admin API calls.
It builds three entry points: `ard` for the combined toolkit, `ardctl` for CLI/client
operations, and `ard-server` for the registry server. `make package` creates Linux/macOS
release archives with embedded version metadata, an SPDX SBOM, and SHA-256 checksums.
`make release-dry-run` verifies the public surface, external Go SDK import, release
archives, checksums, and local packaged binary versions before a public tag.
`ard version`, `ardctl version`, `ard-server --version`, startup logs, and `/health`
expose the build version, commit, and build date. The repository currently ships no
GitHub Actions CI/CD workflows; maintainers run formatting checks, public surface checks,
tests, builds, packaging, Postgres integration tests, compose checks, and live E2E gates
locally as needed.
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
- [Observability](docs/OBSERVABILITY.md)
- [Product Sense](docs/PRODUCT_SENSE.md)
- [Search](docs/SEARCH.md)
- [Policy](docs/POLICY.md)
- [Trust Verification](docs/TRUST.md)
- [ARD Spec Working Notes](docs/references/ard-spec-working-notes.md)

## License

[Apache-2.0](LICENSE)
