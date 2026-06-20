# Supply Chain Security

This document records the template's current supply-chain posture and the controls to add when the project becomes real.

## Current State

This template no longer ships default GitHub Actions supply-chain scanning or release provenance workflows.

The remaining defaults are:

- Do not commit secrets, tokens, or local private configuration.
- Commit auditable dependency manifests and lockfiles once the real project stack exists.
- Pin new GitHub Actions to immutable commit SHAs instead of floating tags.

## Tooling To Add Later

- `actions/dependency-review-action`: reviews pull-request dependency changes.
- `google/osv-scanner-action`: scans for known open source vulnerabilities.
- `anchore/sbom-action`: generates an SPDX SBOM artifact.
- `actions/attest-build-provenance`: generates signed build provenance for release artifacts.

## Limits And Assumptions

- Dependency Review is available for public repositories and private repositories with GitHub Advanced Security.
- There is no automated dependency audit, SBOM, or provenance output right now.
- Reintroduce supply-chain automation after the project stack is known.
- OpenSSF Scorecard is intentionally not enabled by default because a new template repository has no real branch protection, release history, or SAST posture to score. Add it back after repository rules are configured.

## What To Do When The Project Becomes Real

- Add ecosystem-specific lockfiles and keep them committed.
- Make the build deterministic and produce explicit versioned artifacts.
- Gate production deployment on release artifact provenance verification when possible.
- Consider verifying attestations in the deployment environment or cluster admission layer.
