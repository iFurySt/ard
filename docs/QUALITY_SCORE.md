# Quality Score

Track quality by product area and architectural layer so agents can prioritize the weakest parts of the system.

## Suggested Scale

- `A`: strong coverage, stable behavior, clear docs, low operational risk.
- `B`: acceptable but still has known gaps.
- `C`: works but needs targeted hardening.
- `D`: fragile or underspecified.

## Current Snapshot

| Area | Score | Why | Next Step |
| --- | --- | --- | --- |
| Product surface | C | CLI and registry can import catalogs, crawl well-known catalogs, onboard MCP/A2A/Skill/OpenAPI artifacts, list and remove entries, export catalogs, search, browse, explore, return federation referrals, run bounded `federation=auto` upstream merge, and manage remote admin APIs. Combined, CLI-only, and server-only entrypoints are available. Admin lifecycle status can hide inactive entries from public discovery, admin mutation audit events are queryable, and an MVP ingestion policy can deny or pend entries for review. Richer multi-step approval workflows are still missing. | Define publish/update approval semantics and improve federation pagination/ranking behavior. |
| Architecture docs | B | `docs/ARCHITECTURE.md` now describes the Go/Cobra/Gin/GORM/Postgres stack, package boundaries, and container distribution shape. | Keep adapter, API, storage, and deployment decisions updated as public SDK boundaries and release automation emerge. |
| Testing | B | Unit tests, Postgres integration tests, build checks, upstream ARD conformance runs, GitHub Actions CI, `make test-e2e`, and `make test-compose` cover core ingestion/search/admin/artifact-onboarding and deployment paths, including live MCP, Skill, and OpenAPI artifacts plus a checked-in A2A fixture. | Consider an optional scheduled E2E workflow once external artifact availability and rate limits are better understood. |
| Observability | B | Health check exists. HTTP responses include request IDs, JSON access logs are emitted, admin audit events include request IDs for correlation, and `/metrics` exposes Prometheus-style uptime, request, in-flight, and latency counters. Traces, dashboards, runtime metrics, and outbound request correlation are not implemented. | Add tracing, runtime metrics, and a documented local observability workflow. |
| Security | C | Validation enforces `urn:air:`, value/reference exclusivity, URLs, basic catalog shape, and minimal `trustManifest` structure. URL artifacts can be pinned and verified with `trustManifest.sourceDigest`. Admin API routes are disabled by default and guarded by bearer tokens when enabled, with optional role-scoped reader, publisher, reviewer, operator, and admin tokens. Lifecycle status prevents disabled or pending entries from appearing in public discovery. Ingestion policy can deny or pend entries before persistence. Admin mutation events are logged with request IDs, but token rotation, tamper-evident audit trails, detached signatures, and identity proof verification are still pending. | Add token rotation guidance and signature/identity verification design before broader network exposure. |
