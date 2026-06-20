# Reliability

Define the operational bar for the repository here.

## Health

- `GET /health` returns registry process health and the active entry count.
- Health checks must not require admin authentication.

## Request Correlation

- Every HTTP response includes `X-Request-ID`.
- If a request supplies `X-Request-ID`, the registry preserves it.
- If absent, the registry generates a UUID request ID.
- Admin mutation audit events store the request ID so operators can correlate an API
  response, access log line, and audit event.

## Logging

- The registry emits one JSON access log event per HTTP request.
- Access log events include timestamp, level, event name, request ID, method, path,
  status, latency, and client IP.
- Access logs must not include bearer tokens or request bodies.

## Current Gaps

- Metrics and tracing are not implemented yet.
- Request correlation IDs are not propagated to outbound artifact fetches yet.
- There is no documented dashboard or incident response workflow yet.
