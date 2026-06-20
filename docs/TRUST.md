# Trust Verification

`ard` currently implements an MVP trust verification path for artifact source integrity.

## Source Digest Pinning

Artifact add commands can pin a URL artifact digest into `trustManifest`:

```sh
ard add mcp https://example.com/mcp/server.json --pin-source-digest
ardctl admin add mcp https://example.com/mcp/server.json \
  --pin-source-digest \
  --registry-url https://registry.example.com \
  --admin-token "$ARD_ADMIN_TOKEN"
```

Pinning writes:

```json
{
  "trustManifest": {
    "identity": "https://example.com",
    "sourceDigest": "sha256:<hex>"
  }
}
```

`--pin-source-digest` requires a URL source. Local files are embedded as `data`, so the
original source bytes are not available from an exported catalog for later URL integrity
checks.

## Verification

Use:

```sh
ard verify catalog ./ai-catalog.json --source-digests
```

When `--source-digests` is enabled, `ard` fetches each URL entry that has
`trustManifest.sourceDigest`, computes `sha256`, and fails if the digest does not match.

## Current Scope

- Implemented: `trustManifest.identity` presence validation.
- Implemented: `trustManifest.identityType` type and enum validation against the ARD
  schema values: `spiffe`, `did`, `https`, and `other`.
- Implemented: URL `trustManifest.identity` host must match the `urn:air:` publisher
  domain.
- Implemented: `trustManifest.attestations` structure validation for required fields,
  absolute `uri` values, and optional `digest` format.
- Implemented: `trustManifest.provenance` structure validation for required fields,
  supported relation values, and optional `sourceDigest` format.
- Implemented: `trustManifest.sourceDigest` type and format validation.
- Implemented: URL artifact source digest verification.
- Implemented: admin audit event hash chaining and chain verification.
- Not implemented yet: attestation document fetch or content verification.
- Not implemented yet: detached JWS signature verification.
- Not implemented yet: DID, SPIFFE, certificate, or key resolution.
- Not implemented yet: externally anchored or signed audit trails.

URL identity host matching is a metadata consistency check. It rejects entries that claim
`urn:air:acme.com:*` while pointing `trustManifest.identity` at a different HTTP(S)
host. It does not prove domain ownership, certificate identity, or signature validity.
