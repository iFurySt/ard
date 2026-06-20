# Attestation Digest Verification

## Request

Continue building the Go/Cobra/Gin/GORM/Postgres ARD registry and toolkit with
spec-aligned trust verification, real tests, and milestone commits.

## Changes

- Added `ard verify catalog --attestation-digests`.
- Added `ard verify catalog --require-attestation-digests`.
- Added attestation document fetching and SHA-256 verification for
  `trustManifest.attestations[].digest`.
- Reused the existing bounded HTTP fetch, retry, request ID, and traceparent propagation
  path used by source digest verification.
- Added unit coverage for successful verification, mismatch rejection, strict digest
  requirements, and unpinned attestation skipping.
- Added CLI coverage using a real local HTTP attestation document.
- Updated README, trust, security, reliability, architecture, quality, and release
  documentation.

## Design Intent

The ARD trust manifest supports attestation digests, but shape validation alone cannot
tell operators whether the referenced document changed. This change adds an explicit
verification gate that validates bytes fetched from attestation URIs.

This is intentionally narrower than compliance verification. It proves fetched document
integrity only; it does not prove auditor trust, freshness, issuer identity, or whether
the attestation claims are true.

## Files

- `internal/verify/attestation_digest.go`
- `internal/verify/attestation_digest_test.go`
- `internal/cli/verify.go`
- `internal/cli/verify_test.go`
- `README.md`
- `docs/TRUST.md`
- `docs/SECURITY.md`
- `docs/RELIABILITY.md`
- `docs/ARCHITECTURE.md`
- `docs/QUALITY_SCORE.md`
- `docs/releases/feature-release-notes.md`
