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
  "denyTypes": []
}
```

## Behavior

- `denyPublishers` and `denyTypes` reject matching entries before persistence.
- `pendingPublishers` and `pendingTypes` persist matching new entries with lifecycle
  status `pending`.
- `defaultStatus` can be `active`, `pending`, or `disabled`; empty defaults to `active`.
- Deny rules win over pending rules.
- Re-importing an existing entry updates its metadata but does not overwrite its existing
  lifecycle status.
- Public search, browse, explore, and catalog export only expose `active` entries.

Policy is an MVP ingestion gate. It is not a replacement for RBAC, signed trust
manifests, or a full policy engine.
