# Admin Authorization

Admin routes are disabled unless at least one admin token is configured.

## Single Admin Token

For local trials or small deployments, use one full-access token:

```sh
ard-server --admin-token "$ARD_ADMIN_TOKEN"
```

`ARD_ADMIN_TOKEN` is still supported and creates one `admin` role token named
`default-admin`.

## Role Token File

For shared environments, use a token file:

```sh
ard-server --admin-tokens-file ./admin-tokens.json
```

The file can also be selected with `ARD_ADMIN_TOKENS_FILE`.

```json
{
  "version": "1",
  "tokens": [
    { "name": "reader", "token": "reader-token", "role": "reader" },
    { "name": "publisher", "token": "publisher-token", "role": "publisher" },
    { "name": "reviewer", "token": "reviewer-token", "role": "reviewer" },
    { "name": "operator", "token": "operator-token", "role": "operator" },
    { "name": "admin", "token": "admin-token", "role": "admin" }
  ]
}
```

Do not commit real token files.

Running servers reload the role token file when its modification time or size changes.
Write rotations atomically in deployment automation, for example by writing a new file
and renaming it over the old path. If a changed file is invalid, the server keeps the
last valid token set until the file is fixed.

## Roles

| Role | Allowed Admin Operations |
| --- | --- |
| `reader` | List entries, pending reviews, audit events, and export catalog. |
| `publisher` | `reader` permissions plus add/upsert entries and catalogs. |
| `reviewer` | `reader` permissions plus approve or reject pending reviews. |
| `operator` | `reader` permissions plus lifecycle status changes and deletion. |
| `admin` | All admin operations. |

Tokens are matched with constant-time comparison. Token names and roles are for local
authorization only; tokens are never logged, exported, or written to audit events. The
single `ARD_ADMIN_TOKEN` / `--admin-token` value is read at startup; use a role token
file when runtime rotation is required.
