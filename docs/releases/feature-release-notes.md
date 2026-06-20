# Feature Release Notes

## 2026-06

| Date | Area | User Impact | Change Summary |
| --- | --- | --- | --- |
| 2026-06-21 | Governance | Reviewers can record why a pending entry was approved or rejected. | Added review decision `reason` support to the admin API, `ardctl admin review --reason`, persisted audit events, hash-chain verification, integration tests, and real E2E coverage. |
| 2026-06-21 | Testing | Local E2E and compose verification are less likely to fail due to port conflicts or Docker frontend mirror issues. | Switched test scripts to dynamic ports and unique project/container names, and removed the unused Dockerfile frontend directive. |
| 2026-06-21 | Search | Auto-federated responses no longer expose local-only page tokens as if they were federated cursors. | Suppressed `pageToken` when `federation=auto` merges upstream results, with Postgres integration and real E2E coverage. |
| 2026-06-21 | Search | Auto-federated search results are ranked by semantic score across local and upstream registries. | Changed `federation=auto` merge behavior from local-first to score-ranked with local duplicate preference, with unit, integration, and real E2E coverage. |
| 2026-06-21 | Observability | Operators can correlate registry work with upstream services using standard trace context. | Added W3C `traceparent` handling for inbound HTTP, JSON access logs, federation, catalog/artifact fetches, source digest verification, and admin client requests, with unit and real E2E coverage. |
| 2026-06-21 | Observability | Operators can inspect HTTP latency distribution from the existing metrics endpoint. | Added low-cardinality Prometheus histogram buckets for HTTP request duration by method, route template, and status, with unit and compose coverage. |
| 2026-06-21 | Observability | Operators can inspect process-level runtime pressure from the existing metrics endpoint. | Added Go runtime gauges for goroutines, heap allocation, heap system memory, next GC target, GC cycles, and last GC time, with unit and compose coverage. |
| 2026-06-21 | Observability | Operators can correlate remote artifact fetches with the admin API mutations they trigger. | Added shared request-ID context propagation for catalog/artifact/source-digest fetches and `ardctl admin --request-id`, with unit and real E2E coverage. |
| 2026-06-21 | Observability | Operators can correlate local and upstream logs for server-side federated searches. | Propagated inbound `X-Request-ID` to bounded `federation=auto` upstream requests, with unit, integration, and real E2E coverage. |
| 2026-06-21 | Security | Operators can detect persisted admin audit event tampering. | Added `previousHash`/`hash` fields to audit events, `/admin/audit/verify`, and `ardctl admin audit --verify-chain`, with integration and real E2E coverage. |
| 2026-06-21 | Security | Operators can rotate role-scoped admin tokens without restarting the registry. | Added runtime reload for admin token files, preserving the last valid token set on invalid updates, with unit and real E2E coverage. |
| 2026-06-21 | Operations | Operators can page through large admin audit trails instead of only reading the first batch. | Added opaque `pageToken` support to `/admin/audit` and `ardctl admin audit --page-token`, with integration and real E2E coverage. |
| 2026-06-21 | Governance | Policy-pending updates to existing entries now require review before becoming publicly discoverable. | Updated catalog upsert lifecycle handling so pending/disabled policy results apply to existing entries, with integration and E2E coverage. |
| 2026-06-21 | Search | Clients can page through larger search and list responses instead of only receiving the first page. | Added opaque `pageToken` support for search, public listing, and admin list/review flows, plus CLI `--page-token` flags. |
| 2026-06-20 | Deployment | Operators can build and verify a local containerized ARD registry backed by Postgres. | Added a Dockerfile, Docker Compose stack, `make docker-build`, `make test-compose`, and deployment documentation. |

## 2026-04

| Date | Area | User Impact | Change Summary |
| --- | --- | --- | --- |
| 2026-04-08 | Template | Introduced the base harness repository template for future services and products. | Added agent entry docs, execution-plan scaffolding, change-history templates, and docs checks. |
