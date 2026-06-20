## [2026-06-21 00:06] | Task: Add Page Token Pagination

### Execution Context

- Agent ID: `Codex`
- Base Model: `GPT-5`
- Runtime: `local CLI`

### User Query

> Continue building the Go/Cobra/GORM/Gin/Postgres ARD registry toward a neutral
> self-hosted registry and toolkit, with real verification and milestone commits.

### Changes Overview

- Area: ARD search, browse, and admin list pagination.
- Key actions:
  - Added opaque offset page token helpers.
  - Implemented `pageToken` for local search results and list responses.
  - Returned `INVALID_ARGUMENT` for malformed page tokens.
  - Added CLI `--page-token` flags for search, local list, admin list, and review list.
  - Extended integration and E2E coverage for second-page retrieval.

### Design Intent

The ARD spec defines root-level `pageSize` and `pageToken` for search and list-style
responses. The project already modeled those fields but ignored them. This milestone
implements a simple opaque offset token so clients can page through local registry
results without learning database details. Auto federation does not forward local page
tokens to upstream registries because upstream tokens are registry-specific.

### Files Modified

- `internal/pagination/token.go`
- `internal/pagination/token_test.go`
- `internal/store/postgres.go`
- `internal/httpapi/router.go`
- `internal/httpapi/router_integration_test.go`
- `internal/federation/client.go`
- `internal/cli/search.go`
- `internal/cli/list.go`
- `internal/cli/admin.go`
- `scripts/test-e2e-artifacts.sh`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/QUALITY_SCORE.md`
- `docs/releases/feature-release-notes.md`
