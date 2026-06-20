## [2026-06-20 22:53] | Task: OpenAPI Onboarding

### User Request

> Continue building the Go/Cobra/GORM/Gin/Postgres ARD registry and toolkit with
> spec-aligned artifact onboarding, real verification, milestone commits, and real
> artifacts where relevant.

### Changes

- Added canonical OpenAPI ARD media type handling with `application/openapi+json`.
- Added `LoadOpenAPI` adapter for JSON/YAML OpenAPI and Swagger documents.
- Extracted OpenAPI title, description, version, tags, server URL, path count, and
  operation capabilities into ARD catalog entries.
- Added local `ard add openapi`.
- Added remote `ardctl admin add openapi`.
- Added `--kind openapi` filtering for search and admin listing.
- Added a checked-in OpenAPI YAML fixture and adapter test.
- Extended `make test-e2e` to ingest the real Swagger Petstore OpenAPI document and
  verify public search and exported catalog behavior.
- Updated README, architecture, collaboration, and quality docs.

### Design Notes

The adapter only translates OpenAPI metadata into ARD discovery metadata. It does not
execute API operations. The entry type uses the IETF-style `application/openapi+json`
canonical media type while the fetch layer accepts JSON/YAML and OAI/IETF OpenAPI media
types for compatibility.

### Verification

- Passed: `make fmt-check`
- Passed: `make test`
- Passed: `make build`
- Passed: `make test-integration`
- Passed: `make test-e2e`
- `make test-e2e` ingested the real Swagger Petstore OpenAPI document, found it through
  public `--kind openapi` search, and exported it in the catalog.
- Upstream `ard-spec` manifest and registry conformance passed. The manifest check
  reported one warning because `application/openapi+json` is an implementation extension
  media type rather than a current upstream ARD standard discovery media type.
