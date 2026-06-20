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
- Client flow: `ard search` sends spec-shaped `SearchRequest` bodies to a registry.
- Catalog ingestion: `ard add catalog` loads local or remote `ai-catalog.json` files,
  validates them, and persists entries.
- Catalog export: `ardctl export catalog` writes persisted registry entries as a
  spec-shaped `ai-catalog.json` for backup, migration, or well-known publication.
- Local registry management: `ardctl list` and `ardctl remove` inspect and prune
  persisted catalog entries.
- Admin API: when `ARD_ADMIN_TOKEN` or `--admin-token` is configured, Gin exposes
  protected `/admin/*` routes for entry listing, entry upsert, catalog upsert, catalog
  export, lifecycle status changes, audit event listing, and deletion. `ardctl admin` is
  the first client for those remote routes.
- Lifecycle governance: persisted entries have an implementation-owned lifecycle status
  of `active`, `pending`, or `disabled`. Public discovery, search, explore, and catalog
  export only expose `active` entries; admin list can include and filter all statuses.
- Audit log: admin mutations append persisted events for upsert, status changes, and
  deletion with action, identifier, status, source, remote address, and timestamp.
- Artifact onboarding: `ard add mcp`, `ard add a2a`, `ard add skill`, and
  `ard add openapi` translate real MCP server cards, A2A agent cards, Skill markdown
  files, and OpenAPI documents into ARD catalog entries.
- Verification engine: initial schema-level checks cover `urn:air:`, required fields,
  `url`/`data` exclusivity, URL syntax, and representative query count.

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
- `infra/`: Docker, deployment, and environment definitions.
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
   domain-anchored `urn:air:` identifiers, publisher domains, and trust metadata.
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
- Federation traversal should be bounded by depth, registry count, response size, and
  timeout controls.
- Secrets and tokens may be used during request scope only; they must not be stored or
  emitted in plain text.
- Admin API routes must remain disabled by default and require `Authorization: Bearer`
  when enabled.
- Inactive lifecycle states are implementation metadata, not ARD catalog schema fields.
  Do not export disabled or pending entries through public catalog/search surfaces.
- Specification behavior should be derived from `ards-project/ard-spec`, especially
  `spec/ard.md`, `spec/schemas/`, ADRs, and `conformance/`.

## API Targets

- `GET /.well-known/ai-catalog.json`: advertise this registry and any configured catalog
  entries. Implemented.
- `POST /search`: ARD search endpoint. Implemented.
- `POST /explore`: optional; implemented for local facet aggregation.
- `GET /agents`: optional deterministic browse endpoint; implemented for basic listing.
- `GET /health`: deployment health. Implemented.
- `/admin/*`: implementation-specific management routes; disabled unless an admin token
  is configured. Implemented, including entry lifecycle status management and audit event
  listing.
- CLI equivalents: `serve`, `add catalog`, `add mcp`, `add a2a`, `add skill`,
  `add openapi`, `crawl`, `admin`, `export catalog`, `list`, `remove`, `verify catalog`,
  and `search` are implemented. `ardctl admin status` manages remote entry lifecycle
  state, and `ardctl admin audit` lists admin mutation events. `ard-server` runs the same
  server without exposing management subcommands.

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
- Keep `score` strictly as semantic relevance, not a trust or safety signal.
- Support web ingestion of `ai-catalog.json` catalogs as a required registry capability.
- Keep `/explore` local-only and optional; if unsupported, return `501`.
- Keep federation controlled by root-level `SearchRequest.federation`.

Do not vendor or fork the upstream spec content casually. If the implementation needs
schemas or conformance tools in-repo, add a pinned, documented copy under a clearly named
third-party or generated directory and record the source commit.

## Open Decisions

- Ranking strategy for the first release.
- Trust manifest verification depth for MVP.
- Release packaging details for the combined `ard`, CLI-only `ardctl`, and server-only
  `ard-server` binaries.
- Whether to add an embedded non-Postgres development mode.
- Whether to vendor selected upstream spec artifacts, use a git submodule, or fetch pinned
  artifacts during development.

When these decisions are made, update this file in the same task as the code.
