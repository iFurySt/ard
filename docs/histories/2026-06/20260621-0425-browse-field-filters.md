# Browse Field Filters

## Request

Continue the Go/Cobra/Gin/GORM/Postgres ARD implementation with real verification,
small milestones, and stronger registry/client discovery behavior.

## Changes

- Extended the shared list filter parser with `tags`, `capabilities`, and
  `metadata.<key>` equality filters.
- Added Postgres-backed filtering for JSONB array fields and scalar metadata keys.
- Covered parser behavior, store filtering, and public `/agents` filtering in focused
  tests.
- Extended the real E2E flow to verify local `ardctl list` and remote `ardctl browse`
  against the imported Skill artifact using tag, capability, and metadata filters.
- Updated README, architecture, quality, and release notes.

## Intent

Enterprise and agent-platform consumers need deterministic inventory by capability and
metadata, not only broad publisher or media-type filters. Keeping this in the store layer
preserves one filter grammar across public HTTP browse, remote CLI browse, and local CLI
list workflows.

## Files

- `internal/store/list_query.go`
- `internal/store/list_query_test.go`
- `internal/store/postgres.go`
- `internal/store/postgres_integration_test.go`
- `internal/httpapi/router_integration_test.go`
- `scripts/test-e2e-artifacts.sh`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/QUALITY_SCORE.md`
- `docs/releases/feature-release-notes.md`
