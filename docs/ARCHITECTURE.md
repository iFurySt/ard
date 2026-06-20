# Architecture

This file is the top-level map for `ard`. The project is now a Go implementation using
Cobra, Gin, GORM, and Postgres.

## Product Surfaces

- Registry server: self-hosted ARD registry exposing discovery, search, health, and
  catalog endpoints through Gin, plus optional token-protected admin routes.
- CLI: Cobra operational entry point for `serve`, `add catalog`, `add mcp`, `add a2a`,
  `add skill`, `add openapi`, `admin`, `crawl`, `export catalog`, `list`, `remove`,
  `verify catalog`, and `search` today.
- Entrypoints: `cmd/ard` ships the combined toolkit, `cmd/ardctl` ships client and
  management operations without server startup, and `cmd/ard-server` ships a dedicated
  registry server binary.
- Container distribution: the root `Dockerfile` builds all three binaries and defaults
  to the dedicated `ard-server` runtime entrypoint. `infra/compose.yaml` runs the
  registry with Postgres for local self-hosted trials.
- Client flow: `ard search` sends spec-shaped `SearchRequest` bodies to a registry.
- Pagination: `POST /search`, `GET /agents`, and admin list/review/audit endpoints
  return opaque offset page tokens when additional local results are available.
- Federation referrals: `POST /search` supports `federation=referrals` by returning
  active `application/ai-registry+json` entries in `SearchResponse.referrals` for
  client-followed federation.
- Federation auto merge: `POST /search` supports `federation=auto` by querying active
  registry referrals, forcing upstream requests to `federation=none`, and merging
  upstream results with local results by descending semantic `score`. When upstream
  results are merged, the response does not return a local-only `pageToken` as a
  federated cursor. Upstream requests propagate `X-Request-ID` for log correlation.
- Catalog ingestion: `ard add catalog` loads local or remote `ai-catalog.json` files,
  validates them, and persists entries.
- Catalog export: `ardctl export catalog` writes persisted registry entries as a
  spec-shaped `ai-catalog.json` for backup, migration, or well-known publication.
- Local registry management: `ardctl list` and `ardctl remove` inspect and prune
  persisted catalog entries.
- Admin API: when `ARD_ADMIN_TOKEN`, `--admin-token`, `ARD_ADMIN_TOKENS_FILE`, or
  `--admin-tokens-file` is configured, Gin exposes protected `/admin/*` routes for entry
  listing, entry upsert, catalog upsert, catalog export, lifecycle status changes, audit
  event listing, and deletion. `ardctl admin` is the first client for those remote routes.
- Admin authorization: a single legacy admin token still grants full access. Optional
  role-scoped token files split admin access into `reader`, `publisher`, `reviewer`,
  `operator`, and `admin` permissions and are reloaded when the file changes.
- Lifecycle governance: persisted entries have an implementation-owned lifecycle status
  of `active`, `pending`, or `disabled`. Public discovery, search, explore, and catalog
  export only expose `active` entries; admin list can include and filter all statuses.
- Ingestion policy: an optional `ARD_POLICY_FILE` / `--policy-file` JSON policy can deny
  entries or require review by moving new or updated entries to `pending` based on
  publisher or media type.
- Review workflow: pending entries can be listed through `/admin/reviews` and approved or
  rejected through dedicated review routes and `ardctl admin review`. Review decisions
  can carry an optional reason that is recorded on the audit event, not on the ARD
  catalog entry.
- Audit log: admin mutations append persisted events for upsert, status changes, and
  deletion with action, identifier, status, optional review reason, source, remote
  address, request ID, timestamp, previous hash, and event hash. `/admin/audit/verify`
  checks the persisted hash chain.
- Request correlation: Gin middleware preserves or generates `X-Request-ID`, returns it
  on every HTTP response, emits JSON access logs, and attaches request IDs to admin audit
  events. Shared request-ID context propagation also covers outbound catalog/artifact
  fetches and source digest verification.
- Trace context: Gin middleware accepts or generates W3C `traceparent`, returns the
  current service span on each HTTP response, adds trace IDs and span IDs to JSON access
  logs, and propagates trace context to outbound federation, catalog, artifact, source
  digest, and admin client requests.
- Metrics: Gin exposes public Prometheus-style `/metrics` with process uptime,
  in-flight requests, request totals, HTTP duration histograms by method, route, and
  status, plus low-cardinality Go runtime gauges for goroutines, heap, and GC state.
- Artifact onboarding: `ard add mcp`, `ard add a2a`, `ard add skill`, and
  `ard add openapi` translate real MCP server cards, A2A agent cards, Skill markdown
  files, and OpenAPI documents into ARD catalog entries.
- Verification engine: schema-level checks cover `urn:air:`, required fields,
  `url`/`data` exclusivity, URL syntax, representative query count, and minimal
  `trustManifest` structure, including `identityType` enum validation,
  attestation/provenance structure validation, and URL identity host alignment with the
  `urn:air:` publisher. URL artifacts can be pinned and verified with
  `trustManifest.sourceDigest`.

## Intended Repository Shape

- `cmd/ard/`: combined CLI and server binary entry point.
- `cmd/ardctl/`: CLI/client-only binary entry point.
- `cmd/ard-server/`: server-only binary entry point.
- `internal/cli/`: Cobra command tree.
- `internal/httpapi/`: Gin router and HTTP handlers.
- `internal/ard/`: ARD models, media type constants, filters, and validation.
- `internal/adapters/`: artifact-to-catalog-entry adapters for MCP, A2A, Skills, and
  OpenAPI.
- `internal/catalog/`: local and HTTP catalog loading.
- `internal/store/`: GORM/Postgres persistence and search.
- `internal/config/`: environment and CLI config helpers.
- `packages/`: reserved for future public SDK packages once the internal API stabilizes.
- `apps/registry/`: reserved for a separate deployable server only if the single binary
  becomes limiting.
- `infra/`: Docker Compose, deployment, and environment definitions.
- `scripts/`: repository automation that agents can run directly.
- `docs/`: repository knowledge base and system of record.

Keep the internal boundaries visible. Only promote packages out of `internal/` when there
is a stable public SDK contract.

## Runtime Topology

The smallest useful deployment is one registry process with embedded persistence:

```text
catalog URLs / local artifacts
        |
        v
crawler + adapter layer
        |
        v
validation + verification
        |
        v
metadata store + search index
        |
        v
ARD /search API + CLI client
```

The first storage target is Postgres through GORM. Search is currently simple
case-insensitive text recall over persisted `search_text`, with score computed as
semantic relevance approximation. More advanced ranking can replace this behind the store
boundary without changing HTTP contracts.

## Core Data Flow

1. A user adds, lists, removes, exports, or searches catalog entries with the CLI or API.
2. The crawler fetches `/.well-known/ai-catalog.json` or a direct artifact URL.
3. The adapter layer normalizes supported artifacts into ARD catalog entries.
4. The verification layer validates schema, media type, `url`/`data` exclusivity,
   domain-anchored `urn:air:` identifiers, publisher domains, trust metadata, and
   optional URL source digests.
5. The index layer stores normalized entries and searchable fields.
6. `POST /search` accepts an ARD `SearchRequest` and returns a ranked `SearchResponse`.
7. Clients fetch the selected artifact and execute it through its native protocol.

## Boundary Rules

- Keep ARD models and protocol handling independent from transport, storage, and CLI
  code.
- Keep verification pure and reusable; both server-side ingestion and CLI validation
  should call the same logic.
- Keep adapters narrow. MCP, Skills, A2A, and OpenAPI adapters should translate metadata,
  not execute tools.
- Search and ranking should consume normalized catalog entries, not protocol-specific
  objects.
- Page tokens are opaque implementation details. Do not expose raw database cursors or
  require clients to parse token contents.
- Federation traversal must stay bounded by depth, registry count, response size, and
  timeout controls. Auto federation currently queries at most three upstream registry
  referrals, uses non-recursive upstream search requests, limits response bodies, and
  returns a score-ranked merged result set with local entries winning duplicate
  identifiers. Local page tokens are not forwarded to upstream registries, and local
  next-page tokens are suppressed when upstream results participate in the merged
  response because the implementation does not yet expose a cross-registry cursor.
  Request IDs are forwarded for correlation; admin tokens are not.
- Outbound catalog and artifact fetches should propagate request IDs when the initiating
  context carries one. `ardctl admin` generates an operation-level request ID by default
  and accepts `--request-id` / `ARD_REQUEST_ID` when operators want to set it explicitly.
- Inbound and outbound HTTP trace context uses W3C `traceparent`. The registry should
  preserve the incoming trace ID, create a local span ID, and propagate that context
  downstream. This is context propagation only, not a trace exporter.
- Secrets and tokens may be used during request scope only; they must not be stored or
  emitted in plain text.
- Admin API routes must remain disabled by default and require an authorized
  `Authorization: Bearer` token when enabled.
- Role-scoped token file reloads must preserve the last valid token set if a changed file
  is invalid, so a partial write does not lock operators out.
- Audit hash chains are tamper-evident integrity metadata, not a replacement for external
  immutable storage, signatures, or database access control.
- Inactive lifecycle states are implementation metadata, not ARD catalog schema fields.
  Do not export disabled or pending entries through public catalog/search surfaces.
- Policy evaluation must happen before persistence for local add/crawl and remote admin
  imports. Denied entries must not be persisted.
- Specification behavior should be derived from `ards-project/ard-spec`, especially
  `spec/ard.md`, `spec/schemas/`, ADRs, and `conformance/`.

## API Targets

- `GET /.well-known/ai-catalog.json`: advertise this registry and any configured catalog
  entries. Implemented.
- `POST /search`: ARD search endpoint with root-level `pageSize` and `pageToken`.
  Implemented.
- `POST /explore`: optional; implemented for local facet aggregation.
- `GET /agents`: optional deterministic browse endpoint with `pageSize` and `pageToken`.
  Implemented for basic listing.
- `GET /health`: deployment health. Implemented.
- `GET /metrics`: Prometheus-style operational metrics. Implemented.
- `/admin/*`: implementation-specific management routes; disabled unless an admin token
  is configured. Implemented, including entry lifecycle status management and paginated
  audit event listing plus audit hash-chain verification.
- CLI equivalents: `serve`, `add catalog`, `add mcp`, `add a2a`, `add skill`,
  `add openapi`, `crawl`, `admin`, `export catalog`, `list`, `remove`, `verify catalog`,
  and `search` are implemented. `ardctl admin status` manages remote entry lifecycle
  state, `ardctl admin review --reason` handles pending review decisions with optional
  audit reasons, and `ardctl admin audit` lists and verifies admin mutation events.
  `ard-server` runs the same server without exposing management subcommands.

## Specification Alignment

The upstream specification source is:

- Repository: `https://github.com/ards-project/ard-spec`
- Rendered spec: `https://agenticresourcediscovery.org/spec/`
- Current observed draft: v0.9
- Current observed local checkout commit during planning: `a78be70`

Implementation decisions should prefer the upstream main spec, schemas, ADRs, and
conformance tool over older reference implementations. In particular:

- Use `urn:air:` identifiers, not the older `urn:ai:` form.
- Treat `application/mcp-server-card+json` as the MCP discovery media type.
- Treat OpenAPI artifact onboarding as an implementation extension using
  `application/openapi+json` until upstream ARD standardizes an OpenAPI discovery media
  type.
- Treat `trustManifest.sourceDigest` as source artifact integrity metadata. It verifies
  bytes fetched from the entry URL; it is not a signature or identity proof.
- Treat HTTP(S) `trustManifest.identity` host matching as catalog metadata consistency,
  not as proof of publisher ownership.
- Keep `score` strictly as semantic relevance, not a trust or safety signal.
- Support web ingestion of `ai-catalog.json` catalogs as a required registry capability.
- Keep `/explore` local-only and optional; if unsupported, return `501`.
- Keep federation controlled by root-level `SearchRequest.federation`. `referrals` mode
  returns registry entries in `SearchResponse.referrals`; `auto` mode performs a bounded
  server-side upstream merge. Upstream auto requests are sent with `federation=none` to
  avoid recursive traversal and without the local registry's page token. They include the
  inbound `X-Request-ID` when one is available.

Do not vendor or fork the upstream spec content casually. If the implementation needs
schemas or conformance tools in-repo, add a pinned, documented copy under a clearly named
third-party or generated directory and record the source commit.

## Open Decisions

- Ranking strategy for the first release.
- Trust manifest verification depth for MVP.
- Whether to add an embedded non-Postgres development mode.
- Whether to vendor selected upstream spec artifacts, use a git submodule, or fetch pinned
  artifacts during development.
- Whether to replace the MVP JSON ingestion policy with a richer policy engine.

When these decisions are made, update this file in the same task as the code.
