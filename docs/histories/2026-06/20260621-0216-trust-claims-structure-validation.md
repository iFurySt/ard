# Trust Claims Structure Validation

## Request

Continue building the Go/Cobra/GORM/Gin/Postgres ARD registry and toolkit with
spec-aligned validation, real verification, and milestone commits.

## Changes

- Added `trustManifest.attestations` validation for array shape, object entries,
  required `type`, `uri`, and `mediaType` fields, absolute attestation URIs, and
  optional `digest` format.
- Added `trustManifest.provenance` validation for array shape, object entries, required
  `relation` and `sourceId` fields, supported relation enum values, and optional
  `sourceDigest` format.
- Added focused model tests for valid and invalid trust claim metadata.
- Updated trust, security, architecture, quality, and release documentation.

## Design Notes

This keeps verification at the catalog-schema and metadata-consistency layer. It rejects
malformed claims before persistence, but does not fetch attestation documents, verify
claim truth, validate detached JWS signatures, resolve keys, or prove DID/SPIFFE
identity.

## Files

- `internal/ard/models.go`
- `internal/ard/models_test.go`
- `docs/TRUST.md`
- `docs/SECURITY.md`
- `docs/ARCHITECTURE.md`
- `docs/QUALITY_SCORE.md`
- `docs/releases/feature-release-notes.md`
