## [2026-06-20 21:44] | Task: Separate Command Entrypoints

### User Request

> Build the Go/Cobra/GORM/Gin/Postgres version with entrypoints under `cmd/`, and keep
> CLI and server separable while preserving real verification and milestone commits.

### Changes

- Added `cmd/ardctl` as the CLI/client-only entrypoint.
- Added `cmd/ard-server` as the server-only registry entrypoint.
- Kept `cmd/ard` as the combined toolkit entrypoint for local convenience.
- Refactored server startup so `ard serve` and `ard-server` share the same Gin/GORM
  execution path.
- Updated `make build` to build all three binaries.
- Added Cobra root tests proving:
  - `ard` includes `serve`.
  - `ardctl` excludes `serve` but keeps management/client commands.
  - `ard-server` runs at the root and exposes no management subcommands.
- Updated README, architecture notes, and quality score.

### Design Notes

The split keeps one implementation of command behavior in `internal/cli` while exposing
different operational surfaces through `cmd/`. Enterprises can package and deploy only
`ard-server` in registry environments while keeping `ardctl` as the administrative/client
tool.

### Verification

- Passed: `make fmt`
- Passed: `go test ./...`
- Passed: `make build`
- Passed: `make test-integration`
- Passed: dedicated entrypoint E2E with Postgres:
  - `ardctl --help` exposes management/client commands.
  - `ardctl serve --help` fails, proving `serve` is not exposed by the CLI-only binary.
  - `ard-server --help` exposes the server root.
  - `ardctl add catalog` imports a real catalog fixture into Postgres.
  - `ard-server` serves the registry on localhost.
  - `ardctl search` finds the imported MCP catalog entry through the registry.
  - Upstream `ard-spec` registry conformance passes against the `ard-server` process.
