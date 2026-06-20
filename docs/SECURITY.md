# Security

Use this document to make secure defaults explicit and legible to agents.

## Admin API

- Public ARD discovery routes do not require authentication in the local registry.
- Implementation-specific `/admin/*` routes are disabled by default.
- Set `ARD_ADMIN_TOKEN` or pass `--admin-token` to `ard serve` / `ard-server` to enable
  one full-access admin token.
- Set `ARD_ADMIN_TOKENS_FILE` or pass `--admin-tokens-file` to enable role-scoped admin
  tokens. Running servers reload this file when it changes so operators can rotate
  role tokens without restarting.
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
- Catalog `host` metadata is validated for required display name, absolute
  documentation/logo URIs, and host-level `trustManifest` structure.
- Catalog entry `url` values must be absolute HTTP(S) URLs.
- Catalog entry `updatedAt` must be an RFC3339 date-time when present, and entry
  `metadata` values must be strings, numbers, booleans, or null.
- Catalog entry `identifier` values must be unique within a catalog import.
- `trustManifest.identityType`, when present, must be one of the ARD schema values:
  `spiffe`, `did`, `https`, or `other`.
- `trustManifest.trustSchema` and `trustManifest.signature`, when present, are validated
  for schema shape and value types.
- `trustManifest.attestations` and `trustManifest.provenance`, when present, are
  validated for required fields, enum values, absolute attestation URIs, and digest
  formats.
- Entry HTTP(S) `trustManifest.identity` hosts must match the `urn:air:` publisher
  domain.
- `ard verify catalog --source-digests` fetches URL artifacts and verifies pinned
  `sha256` source digests.
- Source digest verification proves byte integrity for the fetched URL only. It does not
  prove publisher identity, trust schema authority, attestation truth, signature
  validity, runtime safety, or compliance status.
- Detailed trust behavior is in `docs/TRUST.md`.

## Audit Events

- Admin upsert, lifecycle status, and delete operations append persisted audit events.
- Audit events record action, identifier, status when relevant, optional review decision
  reason, source, remote address, request ID, timestamp, previous hash, and event hash.
- Audit events do not record admin bearer tokens or full request bodies.
- Use `ardctl admin audit --verify-chain` or `GET /admin/audit/verify` to verify that
  persisted audit events still match their hash chain.
- The hash chain is tamper-evident metadata inside the same database. It is not a
  replacement for external immutable storage, detached signatures, or database access
  control.

## Request Logging

- HTTP responses include `X-Request-ID`.
- HTTP responses include W3C `traceparent`.
- JSON access logs include request ID, trace ID, span ID, method, path, status, latency,
  and client IP.
- Outbound catalog/artifact fetches and source digest verification forward
  `X-Request-ID` for correlation when present in context.
- Outbound federation, catalog/artifact fetches, source digest verification, and admin
  client requests forward `traceparent` for trace context propagation when present in
  context.
- `ardctl admin --request-id` and `ARD_REQUEST_ID` set the correlation ID for an admin
  operation. If neither is set, `ardctl admin` generates one.
- Trace IDs and span IDs are correlation metadata, not authentication or authorization
  material. Do not use them as trust signals.
- Access logs must not include admin bearer tokens or request bodies.

## Metrics

- `GET /metrics` is public operational telemetry.
- Metrics must use low-cardinality labels and must not expose bearer tokens, request
  bodies, user queries, identifiers, or artifact URLs.
- HTTP duration histograms are labeled only by method, route template, and status.
- Runtime metrics expose process-level goroutine, heap, and GC state only.
- If deployment policy requires private metrics, restrict `/metrics` at the ingress or
  network layer.

## Federation

- `federation=auto` sends the search request body, including query text and filters, to
  configured upstream registry referrals.
- Configure upstream registry referrals only for networks that are acceptable recipients
  of those queries.
- Admin bearer tokens are not forwarded to upstream federation requests.
- `X-Request-ID` is forwarded to upstream federation requests for log correlation.
- Upstream federation requests are forced to `federation=none` to reduce accidental
  recursive data sharing.

## Current Gaps

- Runtime rotation is limited to role-scoped token files; the single legacy admin token
  is still static until restart.
- No externally anchored or signed audit log yet.
- No signed policy bundle or external policy engine yet.
- No attestation document fetch or content verification yet.
- No detached signature, DID, SPIFFE, certificate, or key-resolution verification yet.

## Scope

Dependency, SBOM, and provenance integration guidance lives in `docs/SUPPLY_CHAIN_SECURITY.md`.
