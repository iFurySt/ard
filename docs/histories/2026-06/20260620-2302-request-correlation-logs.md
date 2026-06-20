## [2026-06-20 23:02] | Task: Request Correlation And Logs

### User Request

> Continue building the Go/Cobra/GORM/Gin/Postgres ARD registry with real enterprise
> readiness, verified milestones, commits, and pushes.

### Changes

- Added HTTP request ID middleware.
- Preserved caller-provided `X-Request-ID` or generated a UUID when absent.
- Returned `X-Request-ID` on every HTTP response.
- Added JSON access logs with timestamp, level, event, request ID, method, path, status,
  latency, and client IP.
- Added `requestId` to persisted admin audit events.
- Updated `ardctl admin audit` tabular output to include request IDs.
- Extended HTTP middleware and Postgres integration tests.
- Extended the real artifact E2E script to verify audit events include request IDs.
- Updated reliability, security, architecture, and quality docs.

### Design Notes

Request correlation is intentionally middleware-owned so all public and admin routes get
the same behavior. Access logs avoid request bodies and bearer tokens. Admin audit events
store request IDs so operators can correlate response headers, access logs, and mutation
events.

### Verification

- Passed: `make fmt-check`
- Passed: `make test`
- Passed: `make build`
- Passed: `make test-integration`
- Passed: `make test-e2e`
- `make test-e2e` verified admin audit events include `requestId` while live
  MCP/Skill/OpenAPI onboarding and the checked-in A2A fixture still pass.
- Upstream `ard-spec` manifest and registry conformance passed. The manifest check still
  reports the expected OpenAPI extension media type warning.
