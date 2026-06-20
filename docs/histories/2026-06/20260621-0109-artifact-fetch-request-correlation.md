## [2026-06-21 01:09] | Task: Artifact Fetch Request Correlation

### Execution Context

- Agent ID: Codex
- Base Model: GPT-5
- Runtime: Codex CLI

### User Query

> Continue building the Go/Cobra/GORM/Gin/Postgres ARD registry and management toolkit
> with real verification, milestone commits, and live protocol-oriented tests.

### Changes Overview

- Area: Observability.
- Key actions:
  - Added shared request-ID context helpers.
  - Propagated `X-Request-ID` to outbound catalog fetches, artifact onboarding fetches,
    source digest verification fetches, and federation upstream requests.
  - Added `ardctl admin --request-id` / `ARD_REQUEST_ID`, with generated operation IDs
    by default.
  - Extended unit tests and the real artifact E2E script to verify remote artifact fetch
    request-ID propagation.
  - Updated reliability, security, architecture, quality, README, and release notes.

### Design Intent

Request correlation should be context-owned instead of duplicated in each outbound
client. A small shared helper keeps federation, catalog loading, adapters, source digest
verification, and admin CLI behavior aligned while avoiding auth-token propagation.

`ardctl admin` now uses one operation-level request ID for both preflight remote artifact
fetches and the subsequent admin API call, so operators can connect artifact server logs,
registry access logs, and admin audit events.

### Files Modified

- `internal/requestid/requestid.go`
- `internal/adapters/source.go`
- `internal/catalog/loader.go`
- `internal/verify/source_digest.go`
- `internal/federation/client.go`
- `internal/httpapi/middleware.go`
- `internal/httpapi/router.go`
- `internal/cli/admin.go`
- `scripts/test-e2e-artifacts.sh`
- `docs/ARCHITECTURE.md`
- `docs/RELIABILITY.md`
- `docs/SECURITY.md`
- `docs/QUALITY_SCORE.md`
- `docs/releases/feature-release-notes.md`
- `README.md`
