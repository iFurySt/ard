# Supply Chain Security

This document records the template's current supply-chain posture and the controls to add when the project becomes real.

## Current State

This template no longer ships default GitHub Actions supply-chain scanning or release provenance workflows.

The remaining defaults are:

- Do not commit secrets, tokens, or local private configuration.
- Commit auditable dependency manifests and lockfiles once the real project stack exists.
- Pin new GitHub Actions to immutable commit SHAs instead of floating tags.
- `make package` produces explicit versioned binary archives and SHA-256 checksums under
  `dist/`.
- `make sbom` and `make package` produce `dist/sbom.spdx.json`, an SPDX 2.3 SBOM for the
  Go module dependency graph.

## Tooling To Add Later

- `actions/dependency-review-action`: reviews pull-request dependency changes.
- `google/osv-scanner-action`: scans for known open source vulnerabilities.
- `actions/attest-build-provenance`: generates signed build provenance for release artifacts.

## Limits And Assumptions

- Dependency Review is available for public repositories and private repositories with GitHub Advanced Security.
- There is no automated dependency audit or signed provenance output right now.
- Binary archives are checksummed but not signed or attested yet.
- Reintroduce supply-chain automation after the project stack is known.
- OpenSSF Scorecard is intentionally not enabled by default because a new template repository has no real branch protection, release history, or SAST posture to score. Add it back after repository rules are configured.

## What To Do When The Project Becomes Real

- Add ecosystem-specific lockfiles and keep them committed.
- Keep release packaging reproducible enough for CI and produce explicit versioned artifacts.
- Sign release checksums and add provenance attestations before publishing public tags.
- Consider replacing or validating the repository-native SPDX generator with a dedicated
  SBOM scanner once release automation matures.
- Gate production deployment on release artifact provenance verification when possible.
- Consider verifying attestations in the deployment environment or cluster admission layer.
