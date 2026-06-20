# Entry URL Validation

## Request

Continue building the Go/Cobra/GORM/Gin/Postgres ARD registry and toolkit with
spec-aligned validation, real verification, and milestone commits.

## Changes

- Tightened catalog entry `url` validation from general URI parsing to absolute
  HTTP(S) URL validation.
- Rejected relative artifact paths, non-HTTP schemes, and URLs without hosts.
- Added focused model tests for accepted local HTTP URLs and rejected invalid URL forms.
- Updated security, trust, architecture, quality, and release documentation.

## Design Notes

The ARD schema describes entry `url` as an HTTP URL reference to retrieve the artifact.
Host documentation/logo URLs and attestation URIs remain general absolute URIs; this
change only applies to catalog entry artifact URLs.

## Files

- `internal/ard/models.go`
- `internal/ard/models_test.go`
- `docs/TRUST.md`
- `docs/SECURITY.md`
- `docs/ARCHITECTURE.md`
- `docs/QUALITY_SCORE.md`
- `docs/releases/feature-release-notes.md`
