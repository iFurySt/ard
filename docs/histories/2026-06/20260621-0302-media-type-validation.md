# Media Type Validation

## Request

Continue hardening the Go/Cobra/GORM/Gin/Postgres ARD registry and toolkit with real
verification, milestone commits, and alignment with the ARD specification.

## Changes

- Added media type syntax validation for catalog entry `type` values.
- Added media type syntax validation for trust attestation `mediaType` values.
- Added model tests for rejecting bare tokens such as `mcp` and accepting parameterized
  Skill media types.
- Updated architecture, security, trust, quality, and feature release notes.

## Intent

ARD uses the catalog entry `type` field as a media-type envelope that identifies the
artifact protocol or payload. The validator now rejects malformed envelope values before
persistence while avoiding a fixed allowlist so implementation extensions such as
OpenAPI and parameterized Skill media types remain valid.

## Files

- `internal/ard/models.go`
- `internal/ard/models_test.go`
- `docs/ARCHITECTURE.md`
- `docs/SECURITY.md`
- `docs/TRUST.md`
- `docs/QUALITY_SCORE.md`
- `docs/releases/feature-release-notes.md`
