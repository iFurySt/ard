# Security

Use this document to make secure defaults explicit and legible to agents.

## Admin API

- Public ARD discovery routes do not require authentication in the local registry.
- Implementation-specific `/admin/*` routes are disabled by default.
- Set `ARD_ADMIN_TOKEN` or pass `--admin-token` to `ard serve` / `ard-server` to enable
  one full-access admin token.
- Set `ARD_ADMIN_TOKENS_FILE` or pass `--admin-tokens-file` to enable role-scoped admin
  tokens.
- Admin requests must send `Authorization: Bearer <token>`.
- Supported roles are `reader`, `publisher`, `reviewer`, `operator`, and `admin`.
- Do not log, commit, export, or echo admin tokens.
- Run admin routes behind TLS and a trusted ingress in shared environments. The built-in
  bearer token system is an MVP management guard, not a full enterprise identity layer.
- Detailed role behavior is in `docs/ADMIN_AUTH.md`.

## Lifecycle Governance

- Entries are `active` by default when imported.
- Admin users can set entries to `pending`, `disabled`, or back to `active`.
- Public search, browse, explore, and catalog export only expose `active` entries.
- Lifecycle status is implementation metadata and should not be treated as a substitute
  for role-based authorization, policy decisions, or signed trust verification.

## Ingestion Policy

- Set `ARD_POLICY_FILE` or pass `--policy-file` to apply a JSON ingestion policy.
- Policy can deny entries by publisher or media type before persistence.
- Policy can move new or updated entries to `pending` by publisher or media type.
- Denied entries must not be persisted or exposed through public discovery.
- Policy files should be versioned with the deployment configuration and reviewed like
  code.

## Trust Verification

- `--pin-source-digest` can add `trustManifest.sourceDigest` for URL artifacts.
- `ard verify catalog --source-digests` fetches URL artifacts and verifies pinned
  `sha256` source digests.
- Source digest verification proves byte integrity for the fetched URL only. It does not
  prove publisher identity, signature validity, runtime safety, or compliance status.
- Detailed trust behavior is in `docs/TRUST.md`.

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

## Metrics

- `GET /metrics` is public operational telemetry.
- Metrics must use low-cardinality labels and must not expose bearer tokens, request
  bodies, user queries, identifiers, or artifact URLs.
- If deployment policy requires private metrics, restrict `/metrics` at the ingress or
  network layer.

## Federation

- `federation=auto` sends the search request body, including query text and filters, to
  configured upstream registry referrals.
- Configure upstream registry referrals only for networks that are acceptable recipients
  of those queries.
- Admin bearer tokens are not forwarded to upstream federation requests.
- Upstream federation requests are forced to `federation=none` to reduce accidental
  recursive data sharing.

## Current Gaps

- No token rotation workflow yet.
- No tamper-evident audit log yet.
- No signed policy bundle or external policy engine yet.
- No detached signature, DID, SPIFFE, certificate, or key-resolution verification yet.

## Scope

Dependency, SBOM, and provenance integration guidance lives in `docs/SUPPLY_CHAIN_SECURITY.md`.
