## [2026-06-21 05:29] | Task: Auto Federation Cross-Registry Cursors

### Execution Context

- Agent ID: `codex`
- Base Model: `GPT-5`
- Runtime: `Codex CLI`

### User Query

> Continue building the Go/Cobra/GORM/Gin/Postgres ARD registry and toolkit with real
> verification, milestone commits, and strict alignment with the ARD self-hosted
> registry direction.

### Changes Overview

- Area: search, federation, pagination, E2E coverage, docs.
- Key actions:
  - Added opaque composite page tokens for `federation=auto`.
  - Preserved local registry page tokens, upstream registry page tokens, and
    already-fetched candidates that were not returned on the current score-ranked page.
  - Changed the federation client to return upstream page tokens and to resume only
    upstream registries that explicitly have a next page.
  - Kept upstream requests non-recursive by forcing `federation=none`.
  - Extended integration and real E2E coverage with a paginated local upstream registry.

### Design Intent

Auto federation previously suppressed page tokens whenever upstream results participated
in the merge, because a local-only cursor would have been misleading. This change makes
the cursor truly cross-registry while keeping the external API shape unchanged:
`SearchResponse.pageToken` remains opaque. The token may be longer because it can carry
buffered result candidates, but this avoids skipping lower-ranked results that were
already fetched during a score-ranked merge.

### Files Modified

- `internal/federation/client.go`
- `internal/federation/client_test.go`
- `internal/httpapi/federated_page_token.go`
- `internal/httpapi/router.go`
- `internal/httpapi/router_test.go`
- `internal/httpapi/router_integration_test.go`
- `scripts/test-e2e-artifacts.sh`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/QUALITY_SCORE.md`
- `docs/releases/feature-release-notes.md`
