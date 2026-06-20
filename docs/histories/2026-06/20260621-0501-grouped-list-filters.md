## [2026-06-21 05:01] | Task: Grouped list filters

### Execution Context

- Agent ID: `codex`
- Base Model: `GPT-5`
- Runtime: `local CLI`

### User Query

> Continue building the Go/Cobra/GORM/Gin/Postgres ARD registry and toolkit with real verification, milestone commits, and real tests for MCP/Skill/A2A-related flows.

### Changes Overview

- Area: public browse and local registry inventory filters
- Key actions:
  - Replaced the linear `AND` parser path with a recursive filter parser that supports
    `OR` and parenthesized groups.
  - Added an expression tree to `ListFilter` while preserving field-oriented
    compatibility data for existing programmatic callers.
  - Built Postgres SQL conditions recursively from filter expressions.
  - Added parser, Postgres integration, HTTP integration, and real E2E coverage.

### Design Intent

Operators need practical registry inventory queries such as "OpenAPI entries or Skills
matching this metadata" without exporting catalogs or issuing multiple requests. The
parser keeps the grammar intentionally small and SQL-backed: no arbitrary functions,
wildcard fields, or user-supplied column names.

### Files Modified

- `internal/store/list_query.go`
- `internal/store/postgres.go`
- `internal/store/list_query_test.go`
- `internal/store/postgres_integration_test.go`
- `internal/httpapi/router_integration_test.go`
- `scripts/test-e2e-artifacts.sh`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/QUALITY_SCORE.md`
- `docs/releases/feature-release-notes.md`
