# Release Dry Run

## User Request

Continue toward a real Go/Cobra/GORM/Gin/Postgres ARD registry and toolkit with
milestone verification before public release work proceeds.

## Changes

- Added `scripts/release-dry-run.sh` for a pre-tag release rehearsal.
- Added `VERSION=v0.1.0 make release-dry-run`.
- The dry run validates release version shape, formatting, public API/CLI surface,
  workflow invariants, external Go SDK import coverage, release packaging, SHA-256
  checksums, archive contents, and packaged binary version metadata for the local
  platform.
- Documented the workflow in README, deployment docs, collaboration guidance, quality
  score, and release notes.

## Design Intent

The release dry run intentionally reuses `make package`, which is the same packaging
entry point used by the tagged GitHub release workflow. That keeps pre-tag checks close
to the actual publishing path while avoiding tag creation, GitHub release creation, or
artifact attestation requests during local validation.

## Important Files

- `Makefile`
- `scripts/release-dry-run.sh`
- `docs/DEPLOYMENT.md`
- `docs/REPO_COLLAB_GUIDE.md`
- `docs/ARCHITECTURE.md`
- `docs/QUALITY_SCORE.md`
- `docs/releases/feature-release-notes.md`
