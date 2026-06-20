## [2026-06-21 06:07] | Task: Release Attestations

## Request

> Continue building the Go/Cobra/GORM/Gin/Postgres ARD registry and toolkit with real
> verification, milestone commits, and strict alignment with the self-hosted enterprise
> distribution goal.

## Changes

- Added a `v*` tag-triggered GitHub Actions release workflow.
- The release workflow packages `dist/` artifacts, verifies SHA-256 checksums, publishes
  a GitHub release, and generates signed GitHub artifact attestations for release
  provenance plus SPDX SBOM.
- Added `internal/tools/workflowcheck` and `make check-workflows` to validate expected CI
  and release workflow invariants.
- Added workflow checking to the main CI path.
- Pinned new release workflow actions to immutable commit SHAs and made the checker
  enforce those references.
- Updated deployment, supply-chain, architecture, quality, README, and release-note docs.

## Design Notes

The release workflow follows GitHub's current artifact-attestation guidance by using
`actions/attest@v4` with OIDC-backed signing permissions. Branch CI cannot mint release
attestations without a tag, so the repository now verifies the workflow shape locally and
in CI while the signed provenance itself is created by the tag release path.

## Files

- `.github/workflows/release.yml`
- `.github/workflows/ci.yml`
- `Makefile`
- `internal/tools/workflowcheck/`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/DEPLOYMENT.md`
- `docs/REPO_COLLAB_GUIDE.md`
- `docs/SUPPLY_CHAIN_SECURITY.md`
- `docs/QUALITY_SCORE.md`
- `docs/releases/feature-release-notes.md`
