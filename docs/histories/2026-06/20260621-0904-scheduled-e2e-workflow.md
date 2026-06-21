## [2026-06-21 09:04] | Task: Scheduled E2E Workflow

## Request

Continue hardening the Go/Cobra/Gin/GORM/Postgres ARD registry and toolkit with real
verification, milestone commits, and strict alignment with the self-hosted enterprise
direction.

## Changes

- Added a GitHub Actions E2E workflow that runs `make test-e2e` on manual dispatch and
  on a weekly schedule.
- Pinned the E2E workflow's GitHub Actions to immutable action SHAs.
- Extended the repository workflow checker so `make check-workflows` requires the E2E
  workflow, its triggers, read-only contents permission, pinned actions, and live
  artifact E2E gate.
- Updated README, collaboration, deployment, supply-chain, quality, and release-note
  documentation to describe the scheduled/manual E2E posture.

## Intent

`make test-e2e` is intentionally outside the pull-request CI path because it depends on
live MCP, Skill, and OpenAPI artifacts. A scheduled/manual workflow makes that external
drift visible while keeping normal CI deterministic enough for contributor feedback.

## Files

- `.github/workflows/e2e.yml`
- `internal/tools/workflowcheck/`
- `README.md`
- `docs/REPO_COLLAB_GUIDE.md`
- `docs/QUALITY_SCORE.md`
- `docs/SUPPLY_CHAIN_SECURITY.md`
- `docs/DEPLOYMENT.md`
- `docs/releases/feature-release-notes.md`
