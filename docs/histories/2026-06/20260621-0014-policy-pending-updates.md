## [2026-06-21 00:14] | Task: Apply Pending Policy To Updates

### Execution Context

- Agent ID: `Codex`
- Base Model: `GPT-5`
- Runtime: `local CLI`

### User Query

> Continue building the Go/Cobra/GORM/Gin/Postgres ARD registry toward a neutral
> self-hosted registry and toolkit, with real verification and milestone commits.

### Changes Overview

- Area: Ingestion policy and lifecycle governance.
- Key actions:
  - Updated catalog upsert behavior so policy results of `pending` or `disabled` apply
    to existing entries as well as new entries.
  - Preserved the prior behavior that ordinary re-imports do not reactivate or otherwise
    overwrite existing lifecycle status.
  - Added Postgres integration coverage for pending updates to active entries.
  - Added admin API integration and real Skill E2E coverage for pending update review.
  - Updated policy, architecture, quality, and release documentation.

### Design Intent

Pending policy should protect both first publication and later updates. Otherwise, an
already-approved entry could be changed by a policy-matching import while remaining
publicly active. The chosen behavior is intentionally conservative: if policy evaluates
an upsert to a non-active lifecycle state, the stored entry is updated but hidden until
review; normal imports still preserve existing lifecycle status.

### Files Modified

- `internal/store/postgres.go`
- `internal/store/postgres_integration_test.go`
- `internal/httpapi/router_integration_test.go`
- `scripts/test-e2e-artifacts.sh`
- `docs/POLICY.md`
- `docs/ARCHITECTURE.md`
- `docs/QUALITY_SCORE.md`
- `docs/SECURITY.md`
- `docs/releases/feature-release-notes.md`
