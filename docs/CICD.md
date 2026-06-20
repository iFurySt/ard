# CI/CD Guide

This template no longer ships default GitHub Actions CI/CD scaffolding.

## Current State

- There are no default workflows under `.github/workflows/`.
- The repository no longer provides `make ci`, `scripts/ci.sh`, or `scripts/release-package.sh`.
- Add testing, build, scanning, release, and deployment workflows later when the real project stack is known.

## Design Principle

CI/CD should serve the real project instead of preserving placeholder automation in the template.

Once the stack is known, start with the smallest real validation path, then add build artifacts, supply-chain scanning, release, and deployment. Pin new GitHub Actions to commit SHAs instead of floating tags.

## Recommended Customization Sequence

1. Define the project's own local validation command.
2. Add a minimal pull-request gate that runs real tests, lint, or smoke checks.
3. Add packaging, SBOM, and provenance after a real deliverable exists.
4. Add environment-specific deployment jobs after a real runtime and target environment exist.
5. Document all pipeline entry points and release artifacts in this file.

## When Adding CI/CD Back

- Do not restore workflows that only package placeholder metadata.
- Do not expose stale or unmaintained commands in `Makefile`.
- If release automation is added, update `docs/SUPPLY_CHAIN_SECURITY.md` and `docs/releases/README.md` in the same change.
