## [2026-06-20 21:25] | Task: Explore And Agents Endpoints

### Execution Context

- Agent ID: `codex`
- Base Model: `GPT-5`
- Runtime: `local workspace`

### User Query

> Continue the Go ARD registry implementation with spec-aligned, verified milestones.

### Changes Overview

- Area: registry HTTP API.
- Key actions:
  - Implemented `GET /agents` with `items` and `total`.
  - Implemented `POST /explore` facets for fields such as `type`, `publisher`,
    `tags`, `capabilities`, and `metadata.*`.
  - Added ARD explore and list response models.
  - Added Postgres-backed HTTP integration coverage for browse and explore.

### Design Intent

This turns two optional ARD registry endpoints from placeholders into useful B2B
introspection surfaces. `/agents` supports deterministic browsing for portals and
operators, while `/explore` gives clients type/publisher/capability facets without
federating.

### Verification

- `go test ./...`
- `make test-integration`
- `make build`
- Imported the upstream `ard-spec` conformance catalog into a temporary Postgres 16
  Docker container.
- Confirmed `GET /agents?pageSize=5` returned 3 real catalog entries.
- Confirmed `POST /explore` returned real `type` and `publisher` facet buckets.
- Ran upstream `ard-spec` conformance registry probe and passed with `/agents`,
  `/search`, and `/explore` all returning implemented `200` responses.

### Files Modified

- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/histories/2026-06/20260620-2125-explore-agents-endpoints.md`
- `internal/ard/models.go`
- `internal/httpapi/router.go`
- `internal/httpapi/router_integration_test.go`
- `internal/store/postgres.go`
