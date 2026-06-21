# TLS Certificate Key Discovery

## User Request

Continue building the Go/Cobra/GORM/Gin/Postgres ARD registry and toolkit with real
verification and milestone commits.

## Changes

- Added opt-in `ard verify catalog --jws-discover-tls-cert`.
- Discovered keys from HTTPS `trustManifest.identity` TLS leaf certificates using normal
  Go TLS verification.
- Accepted only Ed25519 leaf certificate public keys and reused the existing detached
  JWS verification path.
- Added verifier tests with a real local TLS server carrying an Ed25519 certificate.
- Extended CLI tests and public surface checks for the new `verify catalog` flag.
- Updated README, trust, security, architecture, quality, and release notes.

## Design Intent

This is the smallest certificate-discovery slice that can be verified end to end without
inventing a custom PKI policy. It proves the verified HTTPS endpoint presented the
Ed25519 key used for `trustManifest.signature` at verification time. SPIFFE, custom
certificate policy, revocation, and non-Ed25519 certificate keys remain separate design
work.

## Important Files

- `internal/verify/signature.go`
- `internal/verify/signature_test.go`
- `internal/cli/verify.go`
- `internal/cli/verify_test.go`
- `internal/tools/publicsurface/main.go`
- `docs/TRUST.md`
- `docs/SECURITY.md`
