# Quality Score

Track quality by product area and architectural layer so agents can prioritize the weakest parts of the system.

## Suggested Scale

- `A`: strong coverage, stable behavior, clear docs, low operational risk.
- `B`: acceptable but still has known gaps.
- `C`: works but needs targeted hardening.
- `D`: fragile or underspecified.

## Current Snapshot

| Area | Score | Why | Next Step |
| --- | --- | --- | --- |
| Product surface | C | CLI and registry can import catalogs, crawl well-known catalogs, onboard MCP/A2A/Skill artifacts, list and remove entries, export catalogs, search, browse, and explore. Combined, CLI-only, and server-only entrypoints are available. Policy and richer governance flows are still missing. | Define the first enterprise registry workflow end to end, including publish/update and approval states. |
| Architecture docs | B | `docs/ARCHITECTURE.md` now describes the Go/Cobra/Gin/GORM/Postgres stack and package boundaries. | Keep adapter, API, and storage decisions updated as public SDK boundaries emerge. |
| Testing | C | Unit tests, Postgres integration tests, build checks, and upstream ARD conformance runs cover core ingestion/search paths. | Add repeatable E2E scripts for artifact onboarding with pinned real MCP/A2A/Skill fixtures. |
| Observability | D | Health check exists, but structured logs, metrics, traces, and operational dashboards are not implemented. | Add request logging and a documented local observability workflow. |
| Security | C | Validation enforces `urn:air:`, value/reference exclusivity, URLs, and basic catalog shape. Auth, authorization, signatures, and trust manifest verification are still pending. | Add registry auth and trust verification design before exposing write APIs beyond local CLI use. |
