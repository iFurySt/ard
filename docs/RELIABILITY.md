# Reliability

Define the operational bar for the repository here.

## Health

- `GET /health` returns registry process health, active entry count, and build metadata
  fields: `version`, `commit`, and `buildDate`.
- `ardctl health --registry-url <url>` calls the public health endpoint and supports
  `--json` for automation.
- Health checks must not require admin authentication.

## Request Correlation

- Every HTTP response includes `X-Request-ID`.
- If a request supplies `X-Request-ID`, the registry preserves it.
- If absent, the registry generates a UUID request ID.
- Admin mutation audit events store the request ID so operators can correlate an API
  response, access log line, and audit event.
- Server-side `federation=auto` upstream requests propagate the inbound `X-Request-ID`
  so local and upstream registry logs can be correlated.
- Outbound catalog fetches, artifact onboarding fetches, source digest verification,
  attestation digest verification, and provenance digest verification propagate
  `X-Request-ID` when their context carries one.
- `ardctl admin` generates an operation-level request ID by default and also accepts
  `--request-id` or `ARD_REQUEST_ID`, allowing a remote artifact fetch and the following
  admin API mutation to share one correlation ID.

## Trace Context

- Every HTTP response includes W3C `traceparent`.
- If a request supplies a valid `traceparent`, the registry preserves the incoming trace
  ID, creates a local span ID, and returns that service span.
- If absent or invalid, the registry generates a new trace context.
- JSON access logs include `traceId` and `spanId`.
- Server-side federation, outbound catalog/artifact fetches, source digest verification,
  attestation digest verification, provenance digest verification, and `ardctl admin`
  requests propagate `traceparent` when their context carries one.

## Logging

- The registry emits one JSON access log event per HTTP request.
- Access log events include timestamp, level, event name, request ID, trace ID, span ID,
  method, path, status, latency, and client IP.
- Access logs must not include bearer tokens or request bodies.

## Metrics

- `GET /metrics` returns Prometheus text format without requiring admin authentication.
- `ardctl metrics --registry-url <url>` calls the public metrics endpoint and prints the
  raw Prometheus text for local inspection or script handoff.
- Metrics include registry uptime, in-flight HTTP requests, request totals, and HTTP
  duration histograms by method, route template, and status.
- Runtime metrics include goroutine count, heap allocation, heap system memory, next GC
  target, completed GC cycles, and the last GC timestamp.
- Metrics labels must stay low-cardinality. Use route templates or `unmatched`, not raw
  URLs or identifiers.
- Metrics must not include bearer tokens, request bodies, search text, or remote artifact
  URLs.

## Federation

- `federation=auto` upstream traversal is intentionally shallow and bounded.
- The registry queries at most three active upstream registry referrals per search.
- Upstream HTTP requests use a 10 second client timeout, a bounded response reader, and
  `federation=none` to avoid recursive registry fan-out.
- Upstream HTTP requests propagate `X-Request-ID`, but do not forward admin bearer
  tokens.
- Upstream failures are ignored for the current search response so local search remains
  available.

## Current Gaps

- Trace export, sampling policy, and backend integration are not implemented yet.
- There is no documented dashboard or incident response workflow yet.
