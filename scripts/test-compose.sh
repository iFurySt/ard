#!/usr/bin/env bash
set -euo pipefail

project_name="${ARD_COMPOSE_PROJECT_NAME:-ard-compose-test}"
registry_port="${ARD_COMPOSE_REGISTRY_PORT:-18080}"
admin_token="${ARD_COMPOSE_ADMIN_TOKEN:-compose-admin-token}"
registry_url="http://127.0.0.1:${registry_port}"

compose() {
  ARD_REGISTRY_PORT="${registry_port}" \
    ARD_ADMIN_TOKEN="${admin_token}" \
    docker compose -p "${project_name}" -f infra/compose.yaml "$@"
}

cleanup() {
  compose down -v --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT

compose down -v --remove-orphans >/dev/null 2>&1 || true
compose up -d --build

for _ in $(seq 1 60); do
  if curl -fsS "${registry_url}/health" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
curl -fsS "${registry_url}/health" >/tmp/ard-compose-health.json

bin/ardctl admin add catalog ./internal/catalog/testdata/acme-ai-catalog.json \
  --registry-url "${registry_url}" \
  --admin-token "${admin_token}"
bin/ardctl search weather --registry-url "${registry_url}" --kind mcp --json >/tmp/ard-compose-search.json
grep -q "Weather Data Node" /tmp/ard-compose-search.json

curl -fsS "${registry_url}/metrics" >/tmp/ard-compose-metrics.txt
grep -q "ard_http_requests_total" /tmp/ard-compose-metrics.txt
