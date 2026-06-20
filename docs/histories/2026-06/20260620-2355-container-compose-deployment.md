## [2026-06-20 23:55] | Task: Add Container Compose Deployment

### Execution Context

- Agent ID: `Codex`
- Base Model: `GPT-5`
- Runtime: `local CLI`

### User Query

> Continue building the Go/Cobra/GORM/Gin/Postgres ARD registry toward a neutral
> self-hosted registry and toolkit, with real verification and milestone commits.

### Changes Overview

- Area: Deployment and release readiness.
- Key actions:
  - Added a multi-stage Dockerfile that builds `ard`, `ardctl`, and `ard-server`.
  - Added a Docker Compose stack for a local registry and Postgres deployment.
  - Added `make docker-build` and `make test-compose`.
  - Added CI coverage for the compose deployment path.
  - Documented binary, container, and compose deployment behavior.

### Design Intent

Self-hosted enterprise adoption needs a boring deployment path, not only local binaries.
This milestone keeps the image default focused on the registry server while still
including the operational CLI binaries in the image. The compose test verifies that a
fresh containerized registry can migrate Postgres, accept admin catalog import, serve
public search, and expose metrics.

### Files Modified

- `Dockerfile`
- `.dockerignore`
- `infra/compose.yaml`
- `scripts/test-compose.sh`
- `Makefile`
- `.github/workflows/ci.yml`
- `README.md`
- `AGENTS.md`
- `docs/DEPLOYMENT.md`
- `docs/ARCHITECTURE.md`
- `docs/QUALITY_SCORE.md`
- `docs/REPO_COLLAB_GUIDE.md`
- `docs/releases/feature-release-notes.md`
