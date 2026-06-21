# OIDC Key Discovery

## User Request

Continue building the Go/Cobra/GORM/Gin/Postgres ARD registry and toolkit with real
verification and milestone commits.

## Changes

- Added opt-in `ard verify catalog --jws-discover-oidc`.
- Resolved HTTPS `trustManifest.identity` values as OpenID Connect issuer URLs.
- Fetched `<issuer>/.well-known/openid-configuration`, required the discovered `issuer`
  to match the entry identity, and loaded OKP/Ed25519 keys from `jwks_uri`.
- Reused the existing detached JWS verification path and recorded the real `jwks_uri` as
  the key source.
- Extended the public surface checker to pin `verify catalog` flags.
- Updated README, trust, security, architecture, quality, and release notes.

## Design Intent

OIDC discovery is a pragmatic second key-discovery slice because it has a standard HTTPS
metadata document and a required `jwks_uri`. The implementation keeps discovery
explicit, validates issuer binding before trusting keys, and keeps JWS verification on
the existing Ed25519 path rather than adding broader JWT/OIDC claim semantics.

## Important Files

- `internal/verify/signature.go`
- `internal/verify/signature_test.go`
- `internal/cli/verify.go`
- `internal/cli/verify_test.go`
- `internal/tools/publicsurface/main.go`
- `docs/TRUST.md`
- `docs/SECURITY.md`
