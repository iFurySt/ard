## [2026-06-21 00:56] | Task: Federation Request Correlation

### User Request

> Continue the Go/Cobra/GORM/Gin/Postgres ARD implementation with real verification and
> milestone commits.

### Changes

- Added federation client support for propagating `X-Request-ID` from context.
- Updated `federation=auto` search handling to pass the inbound request ID to upstream
  registry searches.
- Preserved the existing guard that admin bearer tokens are not forwarded upstream.
- Added federation unit coverage for request ID propagation and non-recursive upstream
  requests.
- Added Postgres router integration coverage proving upstream auto-federation receives
  the same request ID.
- Expanded the real artifact E2E script to send a fixed request ID through
  `federation=auto` and verify the local upstream registry logs it.
- Updated reliability, security, architecture, quality, and release notes.

### Design Intent

Server-side federation should remain bounded and privacy-conscious while still being
operable. Propagating only `X-Request-ID` gives operators a low-risk way to correlate
local and upstream logs without leaking admin authorization or local pagination state.

### Files Touched

- `internal/federation/client.go`
- `internal/federation/client_test.go`
- `internal/httpapi/router.go`
- `internal/httpapi/router_integration_test.go`
- `scripts/test-e2e-artifacts.sh`
- `docs/ARCHITECTURE.md`
- `docs/QUALITY_SCORE.md`
- `docs/RELIABILITY.md`
- `docs/SECURITY.md`
- `docs/releases/feature-release-notes.md`
