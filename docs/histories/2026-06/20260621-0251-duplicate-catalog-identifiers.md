# Duplicate Catalog Identifiers

## Request

Continue hardening the Go/Cobra/GORM/Gin/Postgres ARD registry and toolkit with real
verification, milestone commits, and alignment with the ARD specification.

## Changes

- Added catalog-level validation that rejects duplicate `identifier` values within one
  `ai-catalog.json` import.
- Added a model unit test covering duplicate detection and indexed error output.
- Updated architecture, security, quality, and feature release notes to document the new
  catalog validation behavior.

## Intent

An ARD catalog entry identifier is the stable logical resource name. Accepting two
entries with the same identifier in one catalog would make ingestion order decide which
resource wins and would hide ambiguity from operators. The validator now fails the
catalog before persistence so CLI, crawl, and admin import paths share the same behavior.

## Files

- `internal/ard/models.go`
- `internal/ard/models_test.go`
- `docs/ARCHITECTURE.md`
- `docs/SECURITY.md`
- `docs/QUALITY_SCORE.md`
- `docs/releases/feature-release-notes.md`
