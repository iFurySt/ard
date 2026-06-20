# Provenance Digest Verification

## Request

Continue building the Go/Cobra/Gin/GORM/Postgres ARD registry and toolkit with
spec-aligned trust verification, real tests, and milestone commits.

## Changes

- Added `ard verify catalog --provenance-digests`.
- Added `ard verify catalog --require-provenance-digests`.
- Added provenance source fetching and SHA-256 verification for
  `trustManifest.provenance[].sourceDigest` when `sourceId` is an HTTP(S) URL.
- Reused the existing bounded HTTP fetch, retry, request ID, and traceparent propagation
  path used by source and attestation digest verification.
- Added unit coverage for successful verification, mismatch rejection, strict digest
  requirements, non-retrievable URN source IDs, and pinned non-HTTP source rejection.
- Added CLI coverage using a real local HTTP provenance source.
- Updated README, trust, security, reliability, architecture, quality, and release
  documentation.

## Design Intent

The ARD schema defines provenance links with a flexible `sourceId` string and optional
`sourceDigest`. This verifier keeps the boundary narrow: it verifies bytes only when the
source can be fetched directly through an HTTP(S) `sourceId`.

This does not extend the ARD schema and does not claim to resolve arbitrary URN source
identifiers. It proves fetched source byte integrity only; it does not prove lineage
truth, publisher identity, source freshness, or whether the provenance relation is
semantically correct.

## Files

- `internal/verify/provenance_digest.go`
- `internal/verify/provenance_digest_test.go`
- `internal/cli/verify.go`
- `internal/cli/verify_test.go`
- `README.md`
- `docs/TRUST.md`
- `docs/SECURITY.md`
- `docs/RELIABILITY.md`
- `docs/ARCHITECTURE.md`
- `docs/QUALITY_SCORE.md`
- `docs/releases/feature-release-notes.md`
