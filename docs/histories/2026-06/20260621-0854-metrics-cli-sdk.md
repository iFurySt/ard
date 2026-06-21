## [2026-06-21 08:54] | Task: Metrics CLI and SDK

### Execution Context

- Agent ID: `codex`
- Base Model: `GPT-5`
- Runtime: `Codex CLI`

### User Query

> Continue building the Go/Cobra/GORM/Gin/Postgres ARD registry and management toolkit
> with real verification, milestone commits, and strict alignment with the ARD spec.

### Changes Overview

- Area: observability and public client operations
- Key actions:
  - Added `pkg/client.Metrics(ctx)` for the public `/metrics` endpoint.
  - Added `ard metrics` and `ardctl metrics` to print Prometheus text metrics.
  - Added unit coverage for SDK and CLI metrics behavior.
  - Extended external-module SDK checks to call `Metrics`.
  - Extended the real E2E artifact workflow to check live registry metrics through
    `ardctl metrics`.
  - Updated README, architecture, reliability, SDK compatibility, quality, and release
    notes.

### Design Intent

The registry already exposes public Prometheus metrics, but CLI and SDK users needed to
call the endpoint manually. Returning raw Prometheus text keeps the SDK stable without
inventing an ARD-specific parsed metrics model.

### Files Modified

- `pkg/client/client.go`
- `pkg/client/client_test.go`
- `internal/cli/metrics.go`
- `internal/cli/metrics_test.go`
- `internal/cli/root.go`
- `internal/cli/root_test.go`
- `scripts/test-public-go-client.sh`
- `scripts/test-e2e-artifacts.sh`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/RELIABILITY.md`
- `docs/SDK_COMPATIBILITY.md`
- `docs/QUALITY_SCORE.md`
- `docs/releases/feature-release-notes.md`
