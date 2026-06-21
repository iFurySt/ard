# Policy Verify Preflight

## Request

Continue building the Go/Cobra/Gin/GORM/Postgres ARD registry and toolkit with
spec-aligned governance, real tests, and milestone commits.

## Changes

- Added ingestion policy evaluation to `ard verify catalog` through the existing
  `--policy-file` / `ARD_POLICY_FILE` root setting.
- Made policy evaluation fail verification on policy denial without opening the database
  or mutating registry state.
- Added JSON output for successful policy preflight, including per-entry policy status
  and reason.
- Added CLI tests for successful policy evaluation and policy-denied verification.
- Extended real E2E coverage so policy verification rejects an unpinned catalog before
  import.
- Updated README, policy, security, architecture, quality, and release documentation.

## Design Intent

Operators need the same policy rules available before import, especially in CI or review
pipelines. This keeps `verify catalog` as the no-persistence preflight surface while
preserving the existing ingestion behavior for local add/crawl and remote admin imports.

Policy preflight remains a static policy evaluation. It does not fetch artifacts, verify
digests, verify JWS signatures, resolve keys, or prove identity.

## Files

- `internal/cli/root.go`
- `internal/cli/verify.go`
- `internal/cli/verify_test.go`
- `internal/policy/policy.go`
- `scripts/test-e2e-artifacts.sh`
- `README.md`
- `docs/POLICY.md`
- `docs/SECURITY.md`
- `docs/ARCHITECTURE.md`
- `docs/QUALITY_SCORE.md`
- `docs/releases/feature-release-notes.md`
