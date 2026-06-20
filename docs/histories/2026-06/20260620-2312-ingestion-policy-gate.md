## [2026-06-20 23:12] | Task: Ingestion Policy Gate

### User Request

> Continue building the Go/Cobra/GORM/Gin/Postgres ARD registry toward a neutral B2B
> registry and management toolkit, with real verification, milestone commits, and real
> artifacts where relevant.

### Changes

- Added optional JSON ingestion policy support through `ARD_POLICY_FILE` and
  `--policy-file`.
- Added `internal/policy` with deny and pending rules by publisher and media type.
- Applied policy to local `ard add`, local `ard crawl`, and remote admin imports.
- Added store support for initial lifecycle status on newly imported entries.
- Kept re-import behavior from overwriting existing lifecycle status.
- Returned `POLICY_DENIED` with HTTP 403 for denied remote admin imports.
- Extended integration tests for policy denial and pending lifecycle behavior.
- Extended the real artifact E2E script to verify policy pending and deny behavior using
  the real Open Browser Use Skill.
- Added `docs/POLICY.md` and updated README, architecture, security, and quality docs.

### Design Notes

The policy file is an MVP ingestion gate, not a full policy engine. Deny rules prevent
persistence. Pending rules persist new entries with lifecycle status `pending`, which
keeps them out of public search, browse, explore, and catalog export until an admin
activates them.

### Verification

- Passed: `make fmt-check`
- Passed: `make test`
- Passed: `make build`
- Passed: `make test-integration`
- Passed: `make test-e2e`
- `make test-e2e` enabled a policy file, imported the real Open Browser Use Skill with
  a pending publisher, verified the pending entry is not publicly searchable, and verified
  a denied publisher is rejected before persistence.
- Upstream `ard-spec` manifest and registry conformance passed. The manifest check still
  reports the expected OpenAPI extension media type warning.
