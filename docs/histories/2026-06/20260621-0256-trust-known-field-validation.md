# Trust Known Field Validation

## Request

Continue hardening the Go/Cobra/GORM/Gin/Postgres ARD registry and toolkit with real
verification, milestone commits, and alignment with the ARD specification.

## Changes

- Added known-field validation for top-level `trustManifest` metadata.
- Added known-field validation for nested `trustManifest.trustSchema`, attestation, and
  provenance objects.
- Preserved `trustManifest.sourceDigest` as the documented implementation extension used
  by URL artifact digest pinning and verification.
- Added model tests covering unknown trust metadata fields and the accepted
  `sourceDigest` extension.
- Updated architecture, security, trust, quality, and feature release notes.

## Intent

The upstream ARD JSON Schema marks trust metadata objects with
`additionalProperties: false`. Rejecting unsupported trust fields before persistence
keeps catalogs predictable and avoids silently storing governance or provenance claims
that the registry does not understand. `sourceDigest` remains accepted because this
repository already documents and verifies it as a local source-integrity extension.

## Files

- `internal/ard/models.go`
- `internal/ard/models_test.go`
- `docs/ARCHITECTURE.md`
- `docs/SECURITY.md`
- `docs/TRUST.md`
- `docs/QUALITY_SCORE.md`
- `docs/releases/feature-release-notes.md`
