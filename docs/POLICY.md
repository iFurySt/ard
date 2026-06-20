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
- Re-importing an existing active entry that matches a pending rule updates the stored
  metadata but moves the entry back to `pending`, hiding it from public discovery until
  review approval.
- `defaultStatus` can be `active`, `pending`, or `disabled`; empty defaults to `active`.
- Deny rules win over pending rules.
- Re-importing an existing entry without a pending or disabled policy result updates its
  metadata but does not overwrite its existing lifecycle status.
- Public search, browse, explore, and catalog export only expose `active` entries.
- Pending entries can be listed with `ardctl admin review list`.
- `ardctl admin review approve IDENTIFIER` makes a pending entry active.
- `ardctl admin review reject IDENTIFIER` disables a pending entry.

Policy is an MVP ingestion gate. It is not a replacement for RBAC, signed trust
manifests, or a full policy engine.
