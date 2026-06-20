## [2026-06-20 21:34] | Task: Add Artifact Onboarding Adapters

### User Request

> Continue building the Go/Cobra/GORM/Gin/Postgres ARD implementation with real
> validation, small milestones, and real MCP/A2A/Skill artifacts where those protocols
> are involved.

### Changes

- Added `internal/adapters/` for protocol-specific artifact normalization:
  - MCP server cards become `application/mcp-server-card+json` catalog entries.
  - A2A agent cards become `application/a2a-agent-card+json` catalog entries.
  - Skill markdown files become `text/markdown; profile="urn:air:agent-skills"`
    catalog entries.
- Added `ard add mcp`, `ard add a2a`, and `ard add skill`.
- Added `--identifier` and `--publisher` flags for enterprise namespace control.
- Added adapter fixtures based on real public artifact shapes:
  - Agentmemory MCP server card.
  - A2A Hello World agent card.
  - Open Browser Use Skill frontmatter.
- Updated README, architecture notes, and quality score to reflect artifact onboarding.

### Design Notes

Adapters only translate discovery metadata into ARD `CatalogEntry` records. They do not
execute MCP tools, call A2A agents, or run Skill instructions. This keeps registry
management separate from protocol runtimes and lets the existing catalog validation,
Postgres upsert, and search paths remain shared.

Remote artifacts are stored by `url`; local artifacts are embedded as `data`. Generated
identifiers use `urn:air:<publisher>:<namespace>:<slug>` with `agent.localhost` as the
local fallback, and explicit `--identifier` / `--publisher` flags override inference.

### Verification

- Passed: `make fmt`
- Passed: `go test ./...`
- Passed: `make build`
- Passed: `make test-integration`
- Passed: real Postgres E2E import/search/conformance flow using:
  - Remote MCP card:
    `https://raw.githubusercontent.com/clauxel/agentmemory-mcp/main/server.json`
  - Remote Skill:
    `https://raw.githubusercontent.com/iFurySt/open-codex-browser-use/main/skills/open-browser-use/SKILL.md`
  - A2A agent card served over local HTTP from the checked-in fixture.
- Passed: upstream `ard-spec` registry conformance against the local registry with
  `/agents`, `/search`, and `/explore`.
