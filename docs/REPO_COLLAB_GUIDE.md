# Repository Collaboration Guide

This document defines the default collaboration model for an agent-first repository. Add stack-specific rules in adjacent docs instead of bloating this file.

## Development Principles

- Prefer boring, legible, well-instrumented technology over opaque complexity.
- Optimize for agent legibility: if important knowledge only exists in chat, tickets, or human memory, it effectively does not exist.
- Keep code, docs, tests, config, and release notes synchronized.
- Fix the environment when an agent repeatedly fails; do not rely on prompt retries as the main strategy.
- When fixing a bug, check whether tests and docs should be expanded so the same class of bug is caught once and stays caught.

## Documentation Discipline

- `AGENTS.md` is a routing layer, not a giant policy document.
- `docs/` is the source of truth for repository-local knowledge.
- If behavior changes, update the corresponding docs in the same change.
- Prefer adding a new focused doc over appending unrelated rules to a large catch-all file.

## Git And Review

- Keep commits scoped and descriptive.
- Before a commit or PR, verify that docs, examples, scripts, and histories reflect the final behavior.
- For large or risky work, land changes behind an execution plan checked into `docs/exec-plans/`.
- Prefer review comments and follow-up tasks that cite repository files instead of private context.

## Testing And Validation

- Every meaningful code change should leave behind stronger verification than before.
- Prefer repository-native commands and scripts that agents can run directly.
- If the app has a UI, make it locally bootable and testable in an isolated worktree.
- If the app has logs, metrics, or traces, expose them in a local workflow agents can query.

## CI/CD And Release Posture

- GitHub Actions CI runs formatting checks, unit tests, builds, and Postgres integration
  tests on pushes to `main` and pull requests.
- CI uses a Postgres 16 service and passes `ARD_TEST_DATABASE_URL` into
  `make test-integration`; local runs without that variable still use Docker.
- Release packaging is not automated yet. When release automation is added, document the
  release artifacts and operational assumptions in the same change.

## Configuration Hygiene

- Keep examples and runtime defaults aligned.
- Document every environment variable or external dependency that is required to boot the project.
- Avoid hidden setup steps; encode them in scripts or versioned markdown.
