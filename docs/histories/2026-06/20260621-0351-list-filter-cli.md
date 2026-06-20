# List Filter CLI

## Request

Continue building the Go/Cobra/GORM/Gin/Postgres ARD registry and toolkit with real
verification, milestone commits, and alignment with the ARD specification.

## Changes

- Moved deterministic list `filter` and `orderBy` parsing from the HTTP layer into the
  store layer so registry and CLI code share one implementation.
- Added `ardctl list --filter` and `ardctl list --order-by` for local registry inventory.
- Kept unsupported filter/order fields rejected with explicit errors.
- Tightened `createdAfter` and `updatedAfter` filter parsing so those fields only accept
  the `>` operator, matching the executed Postgres predicate.
- Moved parser unit tests to `internal/store`.
- Added E2E coverage for local `ardctl list --filter --order-by --json` against the real
  Postgres-backed artifact onboarding flow.
- Updated README, architecture, quality, and feature release notes.

## Intent

Deterministic browsing should be a registry feature and a local operator workflow, not an
HTTP-only implementation detail. Sharing the parser keeps `GET /agents` and local
`ardctl list` behavior consistent while preserving safe Postgres query construction.

## Files

- `internal/store/list_query.go`
- `internal/store/list_query_test.go`
- `internal/httpapi/router.go`
- `internal/httpapi/router_test.go`
- `internal/cli/list.go`
- `scripts/test-e2e-artifacts.sh`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/QUALITY_SCORE.md`
- `docs/releases/feature-release-notes.md`
