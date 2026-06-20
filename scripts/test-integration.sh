#!/usr/bin/env bash
set -euo pipefail

container_name="ard-postgres-test"
port="${ARD_TEST_POSTGRES_PORT:-55432}"
database_url="postgres://ard:ard@127.0.0.1:${port}/ard_test?sslmode=disable"

cleanup() {
  docker rm -f "${container_name}" >/dev/null 2>&1 || true
}
trap cleanup EXIT

cleanup
docker run \
  --name "${container_name}" \
  -e POSTGRES_USER=ard \
  -e POSTGRES_PASSWORD=ard \
  -e POSTGRES_DB=ard_test \
  -p "${port}:5432" \
  -d postgres:16 >/dev/null

for _ in $(seq 1 60); do
  if docker exec "${container_name}" pg_isready -U ard -d ard_test >/dev/null 2>&1; then
    ARD_TEST_DATABASE_URL="${database_url}" go test ./...
    exit 0
  fi
  sleep 1
done

echo "postgres did not become ready" >&2
exit 1
