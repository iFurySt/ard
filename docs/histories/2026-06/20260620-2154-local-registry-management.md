## [2026-06-20 21:54] | Task: Add Local Registry Management Commands

### User Request

> Continue building the Go/Cobra/GORM/Gin/Postgres ARD registry and management toolkit
> with real verification and milestone commits.

### Changes

- Added `ard list` / `ardctl list` for local Postgres registry inventory.
- Added `ard remove` / `ardctl remove` for deleting one persisted catalog entry by
  `urn:air:` identifier.
- Added `--kind`, `--limit`, and `--json` to list workflows.
- Added `--yes` confirmation and `--missing-ok` to remove workflows.
- Added `Store.ListEntries` with media-type filtering.
- Added `Store.DeleteEntry`.
- Added reusable `ard.ValidateIdentifier`.
- Updated README, architecture notes, quality score, and integration coverage.

### Design Notes

Listing and removal are local management operations over Postgres. They do not call the
remote registry API. Removal requires `--yes` so destructive scripted usage is explicit
and accidental deletion through a copied command is less likely.

### Verification

- Passed: `make fmt`
- Passed: `go test ./...`
- Passed: `make build`
- Passed: `make test-integration`
- Passed: Postgres E2E management flow:
  - Imported catalog and MCP fixtures.
  - `ardctl list --kind mcp --json` returned both MCP entries.
  - `ardctl remove <identifier>` failed without `--yes`.
  - `ardctl remove <identifier> --yes` removed the selected entry.
  - Follow-up list, export, and registry search confirmed the removed entry was absent.
  - A retained MCP entry remained searchable through `ard-server`.
  - Upstream `ard-spec` registry conformance passed against the local server.
