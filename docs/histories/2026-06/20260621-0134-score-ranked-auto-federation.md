## [2026-06-21 01:34] | Task: Score-Ranked Auto Federation

### Execution Context

- Agent ID: Codex
- Base Model: GPT-5
- Runtime: Codex CLI

### User Query

> Continue building the Go/Cobra/GORM/Gin/Postgres ARD registry and management toolkit
> with real verification, milestone commits, and product-quality federation behavior.

### Changes Overview

- Area: Search and federation.
- Key actions:
  - Changed `federation=auto` result merging from local-first to descending `score`.
  - Preserved local preference when local and upstream results share the same identifier.
  - Added deterministic tie-breakers for stable merged output.
  - Added unit, Postgres integration, and real artifact E2E coverage for ranked
    federation output.
  - Updated README, architecture, quality, and release notes.

### Design Intent

`score` is the ARD search relevance signal, so auto-federated results should respect it
across local and upstream registries. Local duplicate preference avoids replacing a
self-hosted registry's own copy of an entry with an upstream copy while still allowing
high-scoring upstream results to surface above lower-scoring local matches.

### Files Modified

- `internal/httpapi/router.go`
- `internal/httpapi/router_test.go`
- `internal/httpapi/router_integration_test.go`
- `scripts/test-e2e-artifacts.sh`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/QUALITY_SCORE.md`
- `docs/releases/feature-release-notes.md`
