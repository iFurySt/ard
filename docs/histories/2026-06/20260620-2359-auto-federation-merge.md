## [2026-06-20 23:59] | Task: Add Auto Federation Merge

### Execution Context

- Agent ID: `Codex`
- Base Model: `GPT-5`
- Runtime: `local CLI`

### User Query

> Continue building the Go/Cobra/GORM/Gin/Postgres ARD registry toward a neutral
> self-hosted registry and toolkit, with real verification and milestone commits.

### Changes Overview

- Area: ARD registry federation.
- Key actions:
  - Added a bounded federation client for server-side `federation=auto` upstream search.
  - Merged upstream results with local results using local-first ordering and identifier
    deduplication.
  - Kept `federation=referrals` as client-followed registry referral behavior.
  - Extended integration and E2E coverage with a local upstream registry.
  - Updated architecture, reliability, security, README, and quality docs.

### Design Intent

The upstream ARD spec says `auto` federation should return one merged response while
`referrals` should let clients choose which registries to follow. This change implements
the smallest operationally safe server-side merge: active registry referrals only, at
most three upstreams, forced non-recursive upstream searches, bounded response reads, and
best-effort upstream failure handling so local search remains useful.

### Files Modified

- `internal/federation/client.go`
- `internal/federation/client_test.go`
- `internal/httpapi/router.go`
- `internal/httpapi/router_test.go`
- `internal/httpapi/router_integration_test.go`
- `scripts/test-e2e-artifacts.sh`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/QUALITY_SCORE.md`
- `docs/RELIABILITY.md`
- `docs/SECURITY.md`
