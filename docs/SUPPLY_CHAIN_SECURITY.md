# Supply Chain Security

This document records the template's current supply-chain posture and the controls to add when the project becomes real.

## Current State

This project now ships a tag-driven GitHub release workflow with artifact attestations.

The remaining defaults are:

- Do not commit secrets, tokens, or local private configuration.
- Commit auditable dependency manifests and lockfiles once the real project stack exists.
- Pin new GitHub Actions to immutable commit SHAs instead of floating tags.
- `make package` produces explicit versioned binary archives and SHA-256 checksums under
  `dist/`.
- `make sbom` and `make package` produce `dist/sbom.spdx.json`, an SPDX 2.3 SBOM for the
  Go module dependency graph.
- Pushing a `v*` tag publishes the `dist/` artifacts to GitHub Releases.
- The release workflow uses GitHub artifact attestations to sign release provenance and
  the SPDX SBOM predicate for archive subjects.
- `make check-workflows` parses local workflow YAML and fails if release permissions,
  tag triggers, attestation steps, checksum verification, or release publishing drift.
- New release workflow actions are pinned to immutable commit SHAs; updating them should
  update `internal/tools/workflowcheck` in the same change.

## Tooling To Add Later

- `actions/dependency-review-action`: reviews pull-request dependency changes.
- `google/osv-scanner-action`: scans for known open source vulnerabilities.

## Limits And Assumptions

- Dependency Review is available for public repositories and private repositories with GitHub Advanced Security.
- There is no automated dependency audit right now.
- Signed attestations are generated only by the `v*` tag release workflow. Branch CI
  validates workflow shape and package output but does not mint attestations.
- OpenSSF Scorecard is intentionally not enabled by default because a new template repository has no real branch protection, release history, or SAST posture to score. Add it back after repository rules are configured.

## What To Do When The Project Becomes Real

- Add ecosystem-specific lockfiles and keep them committed.
- Keep release packaging reproducible enough for CI and produce explicit versioned artifacts.
- Consider replacing or validating the repository-native SPDX generator with a dedicated
  SBOM scanner once release automation matures.
- Gate production deployment on release artifact provenance verification when possible.
- Consider verifying attestations in the deployment environment or cluster admission layer.
