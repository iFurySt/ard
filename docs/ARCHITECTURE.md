# Architecture

This file is the top-level map for `ard`. The project is now a Go implementation using
Cobra, Gin, GORM, and Postgres.

## Product Surfaces

- Registry server: self-hosted ARD registry exposing discovery, search, health, and
  catalog endpoints through Gin, plus optional token-protected admin routes.
- CLI: Cobra operational entry point for `serve`, `add catalog`, `add mcp`, `add a2a`,
  `add skill`, `add openapi`, `admin`, `browse`, `crawl`, `export catalog`, `health`,
  `list`, `metrics`, `remove`, `verify catalog`, `version`, and `search` today.
- Entrypoints: `cmd/ard` ships the combined toolkit, `cmd/ardctl` ships client and
  management operations without server startup, and `cmd/ard-server` ships a dedicated
  registry server binary.
- Public Go SDK: `pkg/ard` exposes spec-shaped ARD model aliases and validation helpers,
  while `pkg/client` provides an embeddable HTTP client for public discovery and
  token-protected admin registry surfaces. `docs/SDK_COMPATIBILITY.md` defines the
  public import paths, pre-1.0 compatibility expectations, and unstable boundaries.
- Container distribution: the root `Dockerfile` builds all three binaries and defaults
  to the dedicated `ard-server` runtime entrypoint. `infra/compose.yaml` runs the
  registry with Postgres for local self-hosted trials.
- Binary distribution: `make package` produces versioned Linux/macOS amd64/arm64
  archives for `ard`, `ardctl`, and `ard-server`, plus an SPDX SBOM and SHA-256 checksum
  manifest. Build metadata is embedded into all packaged binaries and exposed through
  CLI version commands, server startup logs, and `/health`. `make release-dry-run`
  exercises the same packaging path with pre-tag public surface, workflow, SDK,
  checksum, archive-content, and packaged-binary version checks.
- Release publishing: pushing a `v*` tag runs the GitHub Actions release workflow, checks
  package checksums, generates signed GitHub artifact attestations for release
  provenance and SBOM, and publishes the `dist/` artifacts to a GitHub release.
- E2E automation: `.github/workflows/e2e.yml` runs `make test-e2e` manually and weekly
  so live MCP, Skill, OpenAPI, A2A, policy, federation, and SDK drift is visible without
  making every pull request depend on external services.
- Client flow: `ard search` and the public Go client send spec-shaped `SearchRequest`
  bodies to a registry. The registry rejects unknown request/query fields, missing
  `query.text`, and unsupported `federation` values instead of silently normalizing
  invalid request modes.
- Explore flow: `POST /explore` accepts spec-shaped `ExploreRequest` bodies for local
  facet aggregation. The registry rejects unknown request/query/facet fields and invalid
  facet requests instead of silently ignoring malformed introspection options.
- Pagination: `POST /search`, `GET /agents`, and admin list/review/audit endpoints
  return opaque offset page tokens when additional local results are available.
- Browse flow: `GET /agents` validates public query parameters and supports deterministic
  filtering plus whitelisted ordering instead of ignoring malformed pagination or browse
  options. `ardctl browse` calls this public endpoint without admin credentials and
  exposes filter, order, limit, and page-token flags for remote registry inventory.
  Filters support `AND`/`OR`, parenthesized groups, equality, exclusion, substring
  containment, and timestamp boundary operators across common spec fields, tags,
  capabilities, and metadata keys.
- Federation referrals: `POST /search` supports `federation=referrals` by returning
  active `application/ai-registry+json` entries in `SearchResponse.referrals` for
  client-followed federation.
- Federation auto merge: `POST /search` supports `federation=auto` by querying active
  registry referrals, forcing upstream requests to `federation=none`, and merging
  upstream results with local results by descending semantic `score`. Auto-federated
  responses return an opaque composite `pageToken` that carries local cursors, upstream
  cursors, and already-fetched candidates that were not returned on the current page, so
  clients can continue cross-registry pagination without skipping lower-ranked results.
  Upstream requests propagate `X-Request-ID` for log correlation.
- Catalog ingestion: `ard add catalog` loads local or remote `ai-catalog.json` files,
  validates them, and persists entries.
- Catalog export: `ardctl export catalog` writes persisted registry entries as a
  spec-shaped `ai-catalog.json` for backup, migration, or well-known publication.
- Local registry management: `ardctl list` and `ardctl remove` inspect and prune
  persisted catalog entries. Local list shares the public browse filter/order parser for
  deterministic inventory workflows.
- Admin API: when `ARD_ADMIN_TOKEN`, `--admin-token`, `ARD_ADMIN_TOKENS_FILE`, or
  `--admin-tokens-file` is configured, Gin exposes protected `/admin/*` routes for entry
  listing, entry upsert, catalog upsert, catalog export, lifecycle status changes, review
  decisions, audit event listing, audit hash-chain verification, and deletion. `ardctl
  admin` and `pkg/client` can call those remote routes.
- Admin authorization: a single legacy admin token still grants full access. Optional
  role-scoped token files split admin access into `reader`, `publisher`, `reviewer`,
  `operator`, and `admin` permissions and are reloaded when the file changes.
- Lifecycle governance: persisted entries have an implementation-owned lifecycle status
  of `active`, `pending`, or `disabled`. Public discovery, search, explore, and catalog
  export only expose `active` entries; admin list can include and filter all statuses.
- Ingestion policy: an optional `ARD_POLICY_FILE` / `--policy-file` JSON policy can deny
  entries, require trust metadata fields before persistence, or require review by moving
  new or updated entries to `pending` based on publisher or media type.
  `ard verify catalog` can also evaluate the same policy without persistence for CI or
  preflight use.
- Review workflow: pending entries can be listed through `/admin/reviews` and approved or
  rejected through dedicated review routes and `ardctl admin review`. Policy can require
  multiple distinct reviewer approvals before a pending entry becomes active. Review
  decisions can carry an optional reason that is recorded on the audit event, not on the
  ARD catalog entry.
- Audit log: admin mutations append persisted events for upsert, status changes, and
  deletion with action, identifier, status, optional review reason, source, remote
  address, request ID, timestamp, previous hash, and event hash. `/admin/audit/verify`
  checks the persisted hash chain.
- Request correlation: Gin middleware preserves or generates `X-Request-ID`, returns it
  on every HTTP response, emits JSON access logs, and attaches request IDs to admin audit
  events. Shared request-ID context propagation also covers outbound catalog/artifact
  fetches, source digest verification, attestation digest verification, and provenance
  digest verification.
- Trace context: Gin middleware accepts or generates W3C `traceparent`, returns the
  current service span on each HTTP response, adds trace IDs and span IDs to JSON access
  logs, optionally exports server spans to an OTLP/HTTP trace endpoint, and propagates
  trace context to outbound federation, catalog, artifact, source digest, attestation
  digest, provenance digest, and admin client requests.
- Metrics: Gin exposes public Prometheus-style `/metrics` with process uptime,
  in-flight requests, request totals, HTTP duration histograms by method, route, and
  status, plus low-cardinality Go runtime gauges for goroutines, heap, and GC state.
- Artifact onboarding: `ard add mcp`, `ard add a2a`, `ard add skill`, and
  `ard add openapi` translate real MCP server cards, A2A agent cards, Skill markdown
  files, and OpenAPI documents into ARD catalog entries.
- Verification engine: schema-level checks cover `urn:air:`, required fields,
  schema-aligned catalog root and host fields, media type syntax, `url`/`data`
  exclusivity, absolute HTTP(S) URL syntax, `updatedAt` date-time format, scalar
  metadata values, representative query count, duplicate catalog identifiers, and minimal
  catalog host metadata plus `trustManifest` structure, including `identityType` enum
  validation, identity shape checks for `https`, `spiffe`, and `did`, schema-aligned
  known-field enforcement, `trustSchema`/signature shape validation,
  attestation/provenance structure validation, and HTTP(S)/SPIFFE/`did:web` identity
  trust-domain alignment with the `urn:air:` publisher. URL artifacts can be pinned and
  verified with `trustManifest.sourceDigest`, and strict verification can require all
  URL-delivered entries to carry pinned source digests. Attestation documents can be
  fetched and verified against `trustManifest.attestations[].digest`, and strict
  verification can require every attestation to carry a pinned digest. HTTP(S)
  provenance sources can be fetched and verified against
  `trustManifest.provenance[].sourceDigest`, and strict verification can require every
  HTTP(S) provenance `sourceId` to carry a pinned source digest. Detached compact JWS
  `trustManifest.signature` values can be verified against explicit Ed25519 trust
  anchors supplied in ard's native format, local JWKS OKP/Ed25519 format, or explicit
  HTTPS remote JWKS URLs whose host matches the entry trust domain, and strict
  verification can require every catalog entry to carry a verifiable signature.

## Intended Repository Shape

- `cmd/ard/`: combined CLI and server binary entry point.
- `cmd/ardctl/`: CLI/client-only binary entry point.
- `cmd/ard-server/`: server-only binary entry point.
- `internal/cli/`: Cobra command tree.
- `internal/httpapi/`: Gin router and HTTP handlers.
- `internal/ard/`: ARD models, media type constants, filters, and validation.
- `internal/adapters/`: artifact-to-catalog-entry adapters for MCP, A2A, Skills, and
  OpenAPI.
- `internal/buildinfo/`: linked build version, commit, and build date metadata shared by
  binaries, HTTP health, and package checks.
- `internal/catalog/`: local and HTTP catalog loading.
- `internal/store/`: GORM/Postgres persistence and search.
- `internal/config/`: environment and CLI config helpers.
- `internal/tools/sbom/`: repository-native SPDX SBOM generator used by release
  packaging.
- `internal/tools/publicsurface/`: repository-native public API and CLI surface checker
  for pre-release compatibility gates.
- `internal/tools/workflowcheck/`: repository-native GitHub Actions workflow guard for
  CI, E2E, and release automation invariants.
- `pkg/ard/`: public ARD model aliases and validation helpers for Go consumers.
- `pkg/client/`: public HTTP client for unauthenticated registry discovery surfaces and
  token-protected admin management routes.
- `packages/`: reserved for future non-Go SDK packages or generated artifacts.
- `apps/registry/`: reserved for a separate deployable server only if the single binary
  becomes limiting.
- `infra/`: Docker Compose, deployment, and environment definitions.
- `scripts/`: repository automation that agents can run directly.
- `docs/`: repository knowledge base and system of record.

Keep the internal boundaries visible. Public packages under `pkg/` should remain small,
spec-shaped, and stable enough for external import.

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
case-insensitive text recall over persisted `search_text`, with score computed as a
deterministic semantic relevance approximation. Local results are ordered by descending
score and stable catalog fields. `docs/SEARCH.md` records the first-release search
contract. More advanced ranking can replace this behind the store boundary without
changing HTTP contracts.

## Core Data Flow

1. A user adds, lists, removes, exports, or searches catalog entries with the CLI or API.
2. The crawler fetches `/.well-known/ai-catalog.json` or a direct artifact URL.
3. The adapter layer normalizes supported artifacts into ARD catalog entries.
4. The verification layer validates schema, catalog root/host known fields, media type
   syntax, `url`/`data` exclusivity, domain-anchored `urn:air:` identifiers, duplicate
   identifiers within a catalog, publisher domains, trust metadata, and optional URL
   source digests.
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
  identifiers. Auto federation page tokens are implementation-owned opaque cursors; do
  not parse them or forward local cursor internals to upstream registries. Request IDs
  are forwarded for correlation; admin tokens are not.
- Outbound catalog and artifact fetches should propagate request IDs when the initiating
  context carries one. `ardctl admin` generates an operation-level request ID by default
  and accepts `--request-id` / `ARD_REQUEST_ID` when operators want to set it explicitly.
- Inbound and outbound HTTP trace context uses W3C `traceparent`. The registry should
  preserve the incoming trace ID, create a local span ID, optionally export the completed
  server span to OTLP/HTTP, and propagate that context downstream.
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
- Policy preflight through `ard verify catalog --policy-file` must evaluate the same
  policy rules without opening the database or mutating registry state.
- Policy trust metadata requirements are field-presence gates only. Cryptographic,
  digest, key, and identity proof must stay in explicit verification workflows.
- Specification behavior should be derived from `ards-project/ard-spec`, especially
  `spec/ard.md`, `spec/schemas/`, ADRs, and `conformance/`.

## API Targets

- `GET /.well-known/ai-catalog.json`: advertises this registry plus active configured
  catalog entries. Pending and disabled entries are not published. Implemented.
- `POST /search`: ARD search endpoint with root/query known-field validation plus
  root-level `pageSize`, `pageToken`, and `federation` enum validation. Implemented.
- `POST /explore`: optional; implemented for local facet aggregation with request,
  query, and facet known-field validation.
- `GET /agents`: optional deterministic browse endpoint with validated `pageSize` and
  `pageToken`, EBNF-like `filter` support for `displayName`, `type`, `publisherId`,
  `tags`, `capabilities`, `metadata.<key>`, `createdAfter`, and `updatedAfter`, plus
  `AND`, `OR`, parenthesized groups, and `=`, `!=`, `contains`, `>`, and `>=` operators
  where meaningful. `orderBy` is whitelisted for display name, type, publisher, creation
  time, and update time.
- `GET /health`: deployment health with active entry count plus build version, commit,
  and build date metadata. Implemented.
- `GET /metrics`: Prometheus-style operational metrics. Implemented.
- `/admin/*`: implementation-specific management routes; disabled unless an admin token
  is configured. Implemented, including entry lifecycle status management and paginated
  audit event listing plus audit hash-chain verification.
- Go SDK equivalents: `pkg/client` implements public `Search`, `Browse`, `Explore`,
  `Catalog`, and `Health` methods with typed responses from `pkg/ard`. It also exposes
  token-protected admin methods for entry list/upsert/delete, catalog import/export,
  review decisions, lifecycle status, audit listing, and audit hash-chain verification.
  `make test-public-go-client` verifies these public packages from an external Go
  module.
- CLI equivalents: `serve`, `add catalog`, `add mcp`, `add a2a`, `add skill`,
  `add openapi`, `crawl`, `admin`, `browse`, `export catalog`, `health`, `list`,
  `metrics`, `remove`, `verify catalog`, `version`, and `search` are implemented.
  `ardctl health` calls public `/health` without admin credentials, and `ardctl metrics`
  calls public `/metrics` for Prometheus text output. `ardctl browse` calls public
  `/agents` with filter/order/pagination options. `ardctl list --filter` and
  `--order-by` reuse the same deterministic browse parser as public `/agents`.
  `ardctl admin status` manages remote entry lifecycle state, `ardctl admin review
  --reason` handles pending review decisions with optional audit reasons, and `ardctl
  admin audit` lists and verifies admin mutation events.
  `ard-server` runs the same server without exposing management subcommands, while still
  exposing `--version` and `version` for operational inventory.

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
- Validate catalog entry `type` values as media type syntax, not as a fixed allowlist.
  This preserves extension media types while rejecting malformed envelope values.
- Reject unknown catalog root and `host` fields according to the schema. Entry extension
  fields are not rejected by this closed-shape check.
- Treat `trustManifest.sourceDigest` as source artifact integrity metadata. It verifies
  bytes fetched from the entry URL; it is not a signature or identity proof. This is an
  implementation extension to the ARD `trustManifest` schema.
- Treat `trustManifest.attestations[].digest` as attestation document integrity
  metadata. It verifies fetched attestation bytes only; it is not auditor trust,
  freshness, or claim truth verification.
- Treat `trustManifest.provenance[].sourceDigest` as HTTP(S) provenance source
  integrity metadata. It verifies fetched source bytes only; it is not lineage truth,
  publisher identity, or URN source resolution.
- Treat detached compact JWS verification as explicit operator trust-anchor verification.
  It proves the configured Ed25519 key signed the deterministic `trustManifest` payload
  generated by `ard` with `signature` removed. Local files and explicitly supplied HTTPS
  remote JWKS URLs can provide trust anchors, but `ard` does not perform automatic DID,
  SPIFFE, certificate, OIDC, or key-discovery verification.
- Treat HTTP(S), SPIFFE, and `did:web` `trustManifest.identity` trust-domain matching as
  catalog metadata consistency, not as proof of publisher ownership.
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

- DID, SPIFFE, certificate, and trust-anchor discovery depth for MVP.
- Whether to add an embedded non-Postgres development mode.
- Whether to vendor selected upstream spec artifacts, use a git submodule, or fetch pinned
  artifacts during development.
- Whether to replace the MVP JSON ingestion policy with a richer policy engine.

When these decisions are made, update this file in the same task as the code.
