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
- The built image is a local distribution artifact today. Release publishing and signed
  provenance are still future supply-chain work.
