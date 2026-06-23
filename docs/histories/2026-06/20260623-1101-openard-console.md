## [2026-06-23 11:01] | Task: OpenARD Console

### Execution Context

- Agent ID: `codex`
- Base Model: `GPT-5`
- Runtime: `Codex CLI`

### User Query

> Build an administrator-facing OpenARD Console by reusing the managed-agents web
> console styling where practical, replacing the functionality with OpenARD registry
> administration workflows, and validate with browser automation before committing.

### Changes Overview

- Area: Web console
- Key actions:
  - Added a Vite/React workspace for `OpenARD Console`.
  - Reused the reference console's CDS component primitives and styling approach while
    keeping font assets product-neutral.
  - Added administrator pages for Overview, Discover, Catalog, Add Resource, Reviews,
    Audit Log, Operations, and Settings.
  - Added optional registry-hosted static console serving at `/console`.
  - Bundled the console into the Docker image and compose deployment path.
  - Added shared resource detail inspection from Discover, Catalog, Reviews, and
    registry referral cards.
  - Added npm workspace scripts plus Makefile targets for console development, linting,
    and builds.
  - Documented local console development and browser verification expectations.

### Design Intent

The console is scoped to platform administrators instead of end users. The first slice
uses existing OpenARD HTTP APIs before adding new server features, so the UI can already
manage lifecycle status, reviews, audit inspection, JSON imports, public discovery, and
operations checks without widening backend scope.

The console build is also deployable as a same-origin registry asset through
`--console-dir` / `ARD_CONSOLE_DIR`, reducing local setup to one server process after the
frontend has been built.

The container image builds that console asset during Docker builds and serves it from
`/usr/share/openard/console`, so the Compose stack exposes the administrator UI at
`/console` without a separate frontend process.

Follow-up detail inspection keeps the UI on existing API contracts: list, search, and
review responses already include the resource entry shape, so administrators can inspect
summary fields, representative queries, trust manifests, metadata, and raw entry JSON
before taking lifecycle or governance actions without adding a new server endpoint.

### Files Modified

- `package.json`
- `Makefile`
- `Dockerfile`
- `infra/compose.yaml`
- `scripts/test-compose.sh`
- `apps/console/`
- `internal/cli/`
- `internal/config/`
- `internal/httpapi/`
- `docs/FRONTEND.md`
- `docs/histories/2026-06/20260623-1101-openard-console.md`
