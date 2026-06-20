# Host Metadata Validation

## Request

Continue building the Go/Cobra/GORM/Gin/Postgres ARD registry and toolkit with
spec-aligned validation, real verification, and milestone commits.

## Changes

- Added catalog `host` validation for required `displayName`.
- Added absolute URI validation for `host.documentationUrl` and `host.logoUrl`.
- Applied existing `trustManifest` structure validation to host-level trust metadata.
- Kept HTTP(S) identity host-to-publisher matching scoped to catalog entries with
  `urn:air:` identifiers.
- Added focused model tests for valid host metadata and invalid host fields.
- Updated trust, security, architecture, quality, and release documentation.

## Design Notes

Catalog `host.identifier` is not a catalog entry `urn:air:` identifier, so host
`trustManifest.identity` is validated for URI shape and trust metadata structure without
forcing it to match an entry publisher domain. Entry-level trust identities still align
HTTP(S) hosts with the `urn:air:` publisher.

## Files

- `internal/ard/models.go`
- `internal/ard/models_test.go`
- `docs/TRUST.md`
- `docs/SECURITY.md`
- `docs/ARCHITECTURE.md`
- `docs/QUALITY_SCORE.md`
- `docs/releases/feature-release-notes.md`
