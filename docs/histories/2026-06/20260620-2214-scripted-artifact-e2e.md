## [2026-06-20 22:14] | Task: Script Artifact Onboarding E2E

### User Request

> Continue building the Go/Cobra/GORM/Gin/Postgres ARD implementation with real
> verification. When MCP, A2A, or Skills are involved, test with real artifacts where
> possible.

### Changes

- Added `scripts/test-e2e-artifacts.sh`.
- Added `make test-e2e`.
- The E2E script starts:
  - a temporary Postgres 16 container,
  - a temporary local HTTP server for the A2A fixture,
  - a temporary `ard-server` with admin token protection.
- The script imports through `ardctl admin`:
  - the checked-in ARD catalog fixture,
  - the real remote Agentmemory MCP server card,
  - the real remote Open Browser Use Skill,
  - the checked-in A2A agent card fixture.
- The script verifies admin list, export, local catalog validation, public search, remote
  removal, and optional upstream `ard-spec` conformance when the conformance binary is
  available.
- Updated README, collaboration guide, and quality score.

### Design Notes

The default CI path remains deterministic and avoids live external artifact dependencies.
`make test-e2e` is a heavier local gate for release candidates and protocol-touching
changes. It keeps the real MCP/Skill checks versioned and repeatable instead of relying
on chat transcripts.

### Verification

- Passed: `make fmt-check`
- Passed: `make test`
- Passed: `make build`
- Passed: `make test-integration`
- Passed: `make test-e2e`
- `make test-e2e` verified live onboarding for the real Agentmemory MCP server card,
  the real Open Browser Use Skill, and the checked-in A2A agent card fixture.
- The local catalog export passed `ard verify catalog`.
- The upstream `ard-spec` conformance binary was available locally and passed both
  manifest and registry validation with zero critical errors.
