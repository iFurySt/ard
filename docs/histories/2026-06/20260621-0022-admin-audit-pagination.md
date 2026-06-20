## [2026-06-21 00:22] | Task: Admin Audit Pagination

### User Request

> Continue the Go/Cobra/GORM/Gin/Postgres ARD implementation with real verification and
> milestone commits.

### Changes

- Added opaque `pageToken` pagination to persisted admin audit event listing.
- Updated `/admin/audit` to accept `pageToken`, return `pageToken`, and reject invalid
  tokens with `INVALID_ARGUMENT`.
- Added `ardctl admin audit --page-token` for remote audit pagination.
- Expanded Postgres integration coverage for audit page advancement and invalid tokens.
- Expanded the real artifact E2E script to page admin audit results through `ardctl`.
- Updated README, architecture notes, quality score, and release notes.

### Design Intent

Admin audit is the last list-like management surface that only accepted a limit. This
change keeps audit pagination aligned with search, public browse, admin list, and review
pagination while preserving opaque implementation-owned tokens.

### Files Touched

- `internal/store/postgres.go`
- `internal/httpapi/router.go`
- `internal/cli/admin.go`
- `internal/httpapi/router_integration_test.go`
- `scripts/test-e2e-artifacts.sh`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/QUALITY_SCORE.md`
- `docs/releases/feature-release-notes.md`
