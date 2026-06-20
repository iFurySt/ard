# Security

Use this document to make secure defaults explicit and legible to agents.

## Admin API

- Public ARD discovery routes do not require authentication in the local registry.
- Implementation-specific `/admin/*` routes are disabled by default.
- Set `ARD_ADMIN_TOKEN` or pass `--admin-token` to `ard serve` / `ard-server` to enable
  admin routes.
- Admin requests must send `Authorization: Bearer <token>`.
- Do not log, commit, export, or echo admin tokens.
- Run admin routes behind TLS and a trusted ingress in shared environments. The built-in
  bearer token is an MVP management guard, not a full enterprise identity layer.

## Lifecycle Governance

- Entries are `active` by default when imported.
- Admin users can set entries to `pending`, `disabled`, or back to `active`.
- Public search, browse, explore, and catalog export only expose `active` entries.
- Lifecycle status is implementation metadata and should not be treated as a substitute
  for role-based authorization, policy decisions, or signed trust verification.

## Audit Events

- Admin upsert, lifecycle status, and delete operations append persisted audit events.
- Audit events record action, identifier, status when relevant, source, remote address,
  request ID, and timestamp.
- Audit events do not record admin bearer tokens or request bodies.
- The current audit log is an MVP event trail, not a complete tamper-evident audit
  system.

## Request Logging

- HTTP responses include `X-Request-ID`.
- JSON access logs include request ID, method, path, status, latency, and client IP.
- Access logs must not include admin bearer tokens or request bodies.

## Current Gaps

- No role-based authorization yet.
- No token rotation workflow yet.
- No tamper-evident audit log yet.
- No signature or trust manifest verification beyond schema-level validation yet.

## Scope

Dependency, SBOM, and provenance integration guidance lives in `docs/SUPPLY_CHAIN_SECURITY.md`.
