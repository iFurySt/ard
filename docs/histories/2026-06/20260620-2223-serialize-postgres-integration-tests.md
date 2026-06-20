## [2026-06-20 22:23] | Task: Serialize Postgres Integration Tests

### User Request

> Continue the Go/Cobra/GORM/Gin/Postgres ARD implementation with real verification and
> push milestone changes after validation.

### Changes

- Updated `scripts/test-integration.sh` to run package tests with `go test -p 1 ./...`
  whenever `ARD_TEST_DATABASE_URL` is set or the script starts its own Postgres
  container.

### Design Notes

The integration test packages share one Postgres database URL. GitHub Actions exposed a
race where two packages could call GORM `AutoMigrate` for the same table at the same
time, causing a duplicate Postgres catalog type error. Serializing package execution
keeps the integration command deterministic without weakening the actual database-backed
coverage.

### Verification

- Passed: `make fmt-check`
- Passed: `make test`
- Passed: `make build`
- Passed: `make test-integration`
