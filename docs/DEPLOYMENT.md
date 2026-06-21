# Deployment

`ard` ships as Go binaries and as a container image for the registry server.

## Binaries

Build all entrypoints locally:

```sh
make build
```

- `bin/ard`: combined toolkit and local server.
- `bin/ardctl`: client and management CLI.
- `bin/ard-server`: dedicated registry server.

Inspect embedded build metadata:

```sh
bin/ard version
bin/ardctl version --json
bin/ard-server --version
```

Build versioned release archives:

```sh
make package
```

`make package` writes `dist/ard_<version>_<os>_<arch>.tar.gz` archives for Linux and
macOS on amd64 and arm64 by default. Each archive contains `ard`, `ardctl`,
`ard-server`, `README.md`, `LICENSE`, and `VERSION`. The script also writes
`dist/sbom.spdx.json` with the Go module dependency SBOM and `dist/checksums.txt` with
SHA-256 hashes for every archive and the SBOM.

Release archives embed `version`, `commit`, and `buildDate` into every binary. The
server also prints that metadata on startup and returns it from `GET /health`.

Generate only the SBOM:

```sh
make sbom
```

Useful overrides:

```sh
VERSION=v0.1.0 PLATFORMS='linux/amd64 darwin/arm64' make package
VERSION=v0.1.0 COMMIT="$(git rev-parse --short=12 HEAD)" make build
```

## Release Publishing

Push a `v*` tag to publish a GitHub release:

```sh
git tag v0.1.0
git push origin v0.1.0
```

The release workflow packages the binaries with `VERSION` set to the tag name and
`COMMIT` set to the triggering commit, verifies `dist/checksums.txt`, generates GitHub
artifact attestations for release provenance and the SPDX SBOM, and uploads the `dist/`
artifacts to the GitHub release.

## Scheduled E2E

`.github/workflows/e2e.yml` can be run manually and also runs weekly. It executes
`make test-e2e`, which starts a temporary Postgres-backed registry, imports live MCP,
Skill, and OpenAPI artifacts, checks the A2A fixture, exercises policy, federation,
health, metrics, and the external Go admin SDK flow, then tears down its local
containers and processes.

Consumers can verify a downloaded archive with:

```sh
shasum -a 256 -c checksums.txt
gh attestation verify ./ard_v0.1.0_linux_amd64.tar.gz -R iFurySt/ard
gh attestation verify ./ard_v0.1.0_linux_amd64.tar.gz -R iFurySt/ard \
  --predicate-type https://spdx.dev/Document/v2.3
```

Run the server against Postgres:

```sh
DATABASE_URL='postgres://ard:ard@localhost:5432/ard?sslmode=disable' \
ARD_ADMIN_TOKEN='change-me' \
bin/ard-server --addr :8080
```

## Container Image

Build the local image:

```sh
make docker-build
```

The image defaults to `ard-server --addr :8080` and also includes `ard` and `ardctl`
for operational use.

`make docker-build` passes the same `VERSION`, `COMMIT`, and `BUILD_DATE` overrides as
`make build`. Docker Compose also forwards those variables into the image build when
they are set.

Expected environment:

- `DATABASE_URL`: Postgres connection URL.
- `ARD_ADMIN_TOKEN`: optional full-access admin token.
- `ARD_ADMIN_TOKENS_FILE`: optional role-scoped token file path. The running server
  reloads this file when it changes.
- `ARD_POLICY_FILE`: optional ingestion policy file path.

## Compose

Start a local registry and Postgres:

```sh
docker compose -f infra/compose.yaml up --build
```

The compose file exposes the registry at `http://127.0.0.1:18080` by default.

Useful overrides:

```sh
ARD_REGISTRY_PORT=8080 \
ARD_ADMIN_TOKEN='change-me' \
VERSION=v0.1.0 \
COMMIT="$(git rev-parse --short=12 HEAD)" \
docker compose -f infra/compose.yaml up --build
```

Run the automated compose verification:

```sh
make test-compose
```

The verification builds the image, starts Postgres and the registry, imports the checked-in
catalog fixture through the admin API, searches through the public API, checks metrics, and
then removes the compose stack and volume.

## Operations Notes

- Run the admin API behind TLS and a trusted ingress outside local development.
- Keep Postgres backups and migrations under the deployment owner's control.
- Treat `ARD_ADMIN_TOKEN`, role token files, and policy files as deployment secrets or
  reviewed configuration.
- Rotate role token files with an atomic write-and-rename so the server sees complete
  JSON. Invalid updates are ignored and the last valid token set remains active.
- Binary release archives include an SPDX SBOM and SHA-256 checksums. Tagged GitHub
  releases generate signed provenance and SBOM attestations.
