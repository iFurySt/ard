# Search Federation Validation

## Request

Continue hardening the Go/Cobra/GORM/Gin/Postgres ARD registry and toolkit with real
verification, milestone commits, and alignment with the ARD specification.

## Changes

- Added shared `SearchRequest` validation for required `query.text`.
- Added `federation` enum validation for `auto`, `referrals`, and `none`.
- Updated the HTTP search handler to return `400 INVALID_ARGUMENT` for unsupported
  federation modes instead of silently normalizing them to `auto`.
- Added model and Postgres-backed HTTP integration tests.
- Updated architecture, quality, and feature release notes.

## Intent

The ARD OpenAPI schema defines `SearchRequest.federation` as an enum. Silent fallback on
unknown values can turn a client typo into unintended upstream federation. The registry
now fails malformed search requests before executing local search or contacting referrals.

## Files

- `internal/ard/models.go`
- `internal/ard/models_test.go`
- `internal/httpapi/router.go`
- `internal/httpapi/router_integration_test.go`
- `docs/ARCHITECTURE.md`
- `docs/QUALITY_SCORE.md`
- `docs/releases/feature-release-notes.md`
