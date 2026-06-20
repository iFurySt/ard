## [2026-06-20 22:05] | Task: Add Remote Admin Client

### User Request

> Continue building the Go/Cobra/GORM/Gin/Postgres ARD registry and management toolkit
> with real verification, milestone commits, and enterprise-oriented registry behavior.

### Changes

- Added `ard admin` / `ardctl admin` for the remote token-protected admin API.
- Added remote admin commands:
  - `admin list`
  - `admin add catalog`
  - `admin add mcp`
  - `admin add a2a`
  - `admin add skill`
  - `admin export catalog`
  - `admin remove`
- Reused existing catalog loading and MCP/A2A/Skill adapters before POSTing entries to
  the remote admin API.
- Added `--registry-url` and `--admin-token`, with `ARD_ADMIN_TOKEN` fallback.
- Added admin HTTP client tests.
- Updated README, architecture notes, and quality score.

### Design Notes

The admin API became useful only after a first-party client could exercise it. Keeping the
client under `internal/cli` lets the command surface evolve quickly before any public SDK
contract is promoted under `packages/`.

### Verification

- Passed: `make fmt`
- Passed: `go test ./...`
- Passed: `make build`
- Passed: `make test-integration`
- Passed: remote admin client E2E with Postgres and `ard-server --admin-token`:
  - `ardctl admin list` failed without an admin token.
  - `ardctl admin add catalog` imported the checked-in catalog fixture.
  - `ardctl admin add mcp` imported the real remote Agentmemory MCP server card.
  - `ardctl admin add skill` imported the real remote Open Browser Use Skill.
  - `ardctl admin add a2a` imported the checked-in A2A agent card fixture.
  - `ardctl admin list --kind mcp --json` returned imported MCP entries.
  - `ardctl admin export catalog` wrote a valid `ai-catalog.json`.
  - Local `ard verify catalog` passed on the exported catalog.
  - Upstream `ard-spec` manifest conformance passed on the exported catalog.
  - Public `ardctl search` found the remotely imported MCP entry.
  - `ardctl admin remove` removed the remotely imported MCP entry.
  - Follow-up admin list and public search confirmed the removed entry was gone.
  - Upstream `ard-spec` registry conformance passed against the server.
