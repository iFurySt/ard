# Go SDK Compatibility

`ard` exposes two public Go packages:

- `github.com/ifuryst/ard/pkg/ard`
- `github.com/ifuryst/ard/pkg/client`

Everything under `internal/` is private implementation detail. It can change without
notice.

## Current Status

The Go SDK is pre-1.0. Use tagged releases for production dependencies once release
automation starts.

Before `v1.0.0`:

- Patch releases should be backward compatible.
- Minor releases may change public API shape when upstream ARD drafts or registry
  behavior require it.
- Any intentional breaking SDK change should be called out in release notes.
- `pkg/ard` tracks the ARD data model shape implemented by this registry.

At `v1.0.0` and later:

- Public packages follow Go module semantic import versioning.
- Backward-incompatible public API changes require a new major module path.
- Deprecated APIs should stay available for at least one minor release when practical.

## Compatibility Boundaries

Stable public surface:

- Exported types, constants, and functions in `pkg/ard`.
- Exported types, options, and methods in `pkg/client`.
- JSON request and response shapes that mirror implemented registry APIs.

Explicitly unstable surface:

- Packages under `internal/`.
- CLI output intended for humans unless `--json` is used.
- Opaque `pageToken` values. Clients must store and replay them, not parse them.
- Implementation-specific metadata keys unless documented in repository docs.

Additive response fields are allowed before `v1.0.0`. For example, `pkg/client`
`HealthResponse` includes optional `version`, `commit`, and `buildDate` fields so
operators can identify the registry binary answering a request.
`Metrics` returns raw Prometheus text from the public `/metrics` endpoint so callers do
not have to depend on an unstable parsed metrics model.

## Validation

`make test-public-go-client` creates a temporary external module, imports the public SDK,
and exercises the public discovery, catalog, health, metrics, explore, admin
list/upsert/status, review, audit, delete, validation helper, publisher helper, and
`HTTPError` surfaces. CI runs this check on every push and pull request.
