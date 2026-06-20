## [2026-06-21 00:31] | Task: Admin Token File Reload

### User Request

> Continue the Go/Cobra/GORM/Gin/Postgres ARD implementation with real verification and
> milestone commits.

### Changes

- Added runtime reload for role-scoped admin token files.
- Kept `ARD_ADMIN_TOKEN` / `--admin-token` as a static startup token.
- Preserved the last valid token-file configuration when a changed file is invalid.
- Kept startup validation for token files so bad initial configuration still fails fast.
- Added unit coverage for successful rotation and invalid reload preservation.
- Expanded the real artifact E2E script to rotate a running server's token file, verify
  the new reader token works, and verify the old reader token is rejected.
- Updated security, deployment, architecture, README, quality, and release notes.

### Design Intent

Enterprise operators need token rotation without restarting a registry. Reloading the
role token file keeps the MVP bearer-token model simple while making the self-hosted
control plane more operationally realistic. Invalid updates keep the last valid token set
so a partial write does not lock operators out.

### Files Touched

- `internal/httpapi/auth.go`
- `internal/httpapi/auth_test.go`
- `internal/httpapi/router.go`
- `internal/cli/serve.go`
- `internal/cli/serve_test.go`
- `scripts/test-e2e-artifacts.sh`
- `README.md`
- `docs/ADMIN_AUTH.md`
- `docs/ARCHITECTURE.md`
- `docs/DEPLOYMENT.md`
- `docs/QUALITY_SCORE.md`
- `docs/SECURITY.md`
- `docs/releases/feature-release-notes.md`
