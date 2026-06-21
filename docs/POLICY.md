# Policy

`ard` supports an optional JSON ingestion policy file for local imports, crawl imports,
and remote admin imports.

Set it with `--policy-file` or `ARD_POLICY_FILE`.

## Example

```json
{
  "version": "1",
  "defaultStatus": "active",
  "pendingPublishers": ["review.example.com"],
  "denyPublishers": ["blocked.example.com"],
  "pendingTypes": ["application/openapi+json"],
  "denyTypes": [],
  "requiredApprovals": 2,
  "requireTrustManifest": true,
  "requireSourceDigestForURLArtifacts": true,
  "requireJWSSignature": true
}
```

## Behavior

- `denyPublishers` and `denyTypes` reject matching entries before persistence.
- `pendingPublishers` and `pendingTypes` persist matching new entries with lifecycle
  status `pending`.
- `requireTrustManifest` rejects entries that do not carry `trustManifest`.
- `requireSourceDigestForURLArtifacts` rejects URL-delivered entries that do not carry
  `trustManifest.sourceDigest`. Embedded `data` entries are exempt.
- `requireJWSSignature` rejects entries that do not carry `trustManifest.signature`.
- Re-importing an existing active entry that matches a pending rule updates the stored
  metadata but moves the entry back to `pending`, hiding it from public discovery until
  review approval.
- `defaultStatus` can be `active`, `pending`, or `disabled`; empty defaults to `active`.
- Deny rules win over pending rules.
- Re-importing an existing entry without a pending or disabled policy result updates its
  metadata but does not overwrite its existing lifecycle status.
- `requiredApprovals` sets how many distinct reviewer tokens must approve a pending
  entry before it becomes active. Empty or `0` means `1`.
- Public search, browse, explore, and catalog export only expose `active` entries.
- Pending entries can be listed with `ardctl admin review list`.
- `ardctl admin review approve IDENTIFIER --reason "reviewed publisher and digest"`
  records one reviewer approval. If more approvals are required, the entry remains
  `pending`; when the threshold is reached, it becomes `active`.
- Duplicate approvals from the same reviewer token are rejected.
- `ardctl admin review reject IDENTIFIER --reason "not approved for production"`
  disables a pending entry and records the reason on the review audit event.
- Review reasons are decision metadata, not ARD catalog entry metadata.

Policy is an MVP ingestion gate. Trust metadata requirements check field presence only.
They do not fetch artifacts, verify digests, verify JWS signatures, resolve keys, or
prove identity. Use `ard verify catalog` for explicit verification before promotion or
release.

Policy is not a replacement for RBAC, signed trust manifests, or a full policy engine.
