# Trust Schema Validation

## Request

Continue building the Go/Cobra/GORM/Gin/Postgres ARD registry and toolkit with
spec-aligned validation, real verification, and milestone commits.

## Changes

- Added `trustManifest.trustSchema` object validation.
- Required `trustSchema.identifier` and `trustSchema.version`.
- Validated optional `trustSchema.governanceUri` as an absolute URI.
- Validated optional `trustSchema.verificationMethods` as a string array.
- Validated `trustManifest.signature` as a string when present.
- Added focused model tests and updated security, trust, architecture, quality, and
  release documentation.

## Design Notes

This is schema-shape validation only. It does not resolve trust schema authorities,
verify detached JWS signatures, resolve DID/SPIFFE identities, or validate keys.

## Files

- `internal/ard/models.go`
- `internal/ard/models_test.go`
- `docs/TRUST.md`
- `docs/SECURITY.md`
- `docs/ARCHITECTURE.md`
- `docs/QUALITY_SCORE.md`
- `docs/releases/feature-release-notes.md`
