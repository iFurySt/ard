# Public Go Client SDK

## Request

Continue the Go/Cobra/Gin/GORM/Postgres ARD implementation with real verification,
small milestones, and a reusable client surface for agent platforms.

## Changes

- Added `pkg/ard` as the public Go model package for ARD catalog, search, browse, and
  explore types plus validation helpers.
- Added `pkg/client` as a public HTTP client for unauthenticated registry discovery
  surfaces: search, browse, explore, well-known catalog, and health.
- Added SDK unit tests for request construction, typed responses, custom headers, user
  agent handling, and HTTP error reporting.
- Added `make test-public-go-client`, which creates a temporary external module and
  verifies third-party imports of `github.com/ifuryst/ard/pkg/ard` and
  `github.com/ifuryst/ard/pkg/client`.
- Added the public import check to CI and documented the SDK in README, architecture,
  quality, product, and release notes.

## Intent

`ardctl` is useful for operators, but agent platforms need an embeddable client. The
initial SDK intentionally covers only public discovery APIs and keeps admin management
outside the stable public contract until those workflows settle further.

## Files

- `pkg/ard/models.go`
- `pkg/ard/models_test.go`
- `pkg/client/client.go`
- `pkg/client/client_test.go`
- `scripts/test-public-go-client.sh`
- `Makefile`
- `.github/workflows/ci.yml`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/QUALITY_SCORE.md`
- `docs/PRODUCT_SENSE.md`
- `docs/releases/feature-release-notes.md`
