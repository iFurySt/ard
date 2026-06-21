# Policy Trust Metadata Gates

## Request

Continue building the Go/Cobra/Gin/GORM/Postgres ARD registry and toolkit with
spec-aligned governance, real tests, and milestone commits.

## Changes

- Added `requireTrustManifest` to ingestion policy.
- Added `requireSourceDigestForURLArtifacts` to ingestion policy.
- Added `requireJWSSignature` to ingestion policy.
- Applied these gates through the existing local add/crawl and remote admin import policy
  path before persistence.
- Added unit coverage for required trust manifest, URL source digest, JWS signature, and
  embedded data exemption behavior.
- Added Postgres admin API integration coverage for policy denial and successful
  trusted import.
- Extended real E2E coverage with a live MCP URL import that is rejected without
  `--pin-source-digest` and accepted with it.
- Updated README, policy, security, architecture, quality, and release documentation.

## Design Intent

Enterprise registries need a way to prevent untrusted or unpinned catalog entries from
being persisted accidentally. These policy gates are intentionally static field-presence
checks that run before persistence.

They do not fetch artifacts, verify digests, verify JWS signatures, resolve keys, or
prove identity. Deep verification remains explicit through `ard verify catalog`.

## Files

- `internal/policy/policy.go`
- `internal/policy/policy_test.go`
- `internal/httpapi/router_integration_test.go`
- `scripts/test-e2e-artifacts.sh`
- `README.md`
- `docs/POLICY.md`
- `docs/SECURITY.md`
- `docs/ARCHITECTURE.md`
- `docs/QUALITY_SCORE.md`
- `docs/releases/feature-release-notes.md`
