#!/usr/bin/env bash
set -euo pipefail

postgres_container="${ARD_E2E_POSTGRES_CONTAINER:-ard-e2e-postgres}"
postgres_port="${ARD_E2E_POSTGRES_PORT:-55440}"
fixture_port="${ARD_E2E_FIXTURE_PORT:-18087}"
registry_port="${ARD_E2E_REGISTRY_PORT:-18088}"
admin_token="${ARD_E2E_ADMIN_TOKEN:-test-token}"
database_url="postgres://ard:ard@127.0.0.1:${postgres_port}/ard?sslmode=disable"
registry_url="http://127.0.0.1:${registry_port}"
export_file="$(mktemp /tmp/ard-e2e-export-XXXXXX.json)"
conformance_bin="${ARD_CONFORMANCE_BIN:-../ard-spec/conformance/bin/conformance-test}"

mcp_card_url="https://raw.githubusercontent.com/clauxel/agentmemory-mcp/main/server.json"
skill_url="https://raw.githubusercontent.com/iFurySt/open-codex-browser-use/main/skills/open-browser-use/SKILL.md"

cleanup() {
  if [ -n "${registry_pid:-}" ]; then
    kill "${registry_pid}" >/dev/null 2>&1 || true
    wait "${registry_pid}" >/dev/null 2>&1 || true
  fi
  if [ -n "${fixture_pid:-}" ]; then
    kill "${fixture_pid}" >/dev/null 2>&1 || true
    wait "${fixture_pid}" >/dev/null 2>&1 || true
  fi
  docker rm -f "${postgres_container}" >/dev/null 2>&1 || true
  rm -f "${export_file}"
}
trap cleanup EXIT

make build

docker rm -f "${postgres_container}" >/dev/null 2>&1 || true
docker run \
  -d \
  --name "${postgres_container}" \
  -e POSTGRES_USER=ard \
  -e POSTGRES_PASSWORD=ard \
  -e POSTGRES_DB=ard \
  -p "${postgres_port}:5432" \
  postgres:16 >/dev/null

for _ in $(seq 1 60); do
  if docker exec "${postgres_container}" pg_isready -U ard -d ard >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
docker exec "${postgres_container}" pg_isready -U ard -d ard >/dev/null

python3 -m http.server "${fixture_port}" --directory internal/adapters/testdata >/tmp/ard-e2e-fixtures.log 2>&1 &
fixture_pid=$!
for _ in $(seq 1 30); do
  if curl -fsS "http://127.0.0.1:${fixture_port}/a2a-agent-card.json" >/dev/null 2>&1; then
    break
  fi
  sleep 0.5
done
curl -fsS "http://127.0.0.1:${fixture_port}/a2a-agent-card.json" >/dev/null

bin/ard-server \
  --database-url "${database_url}" \
  --addr "127.0.0.1:${registry_port}" \
  --admin-token "${admin_token}" >/tmp/ard-e2e-registry.log 2>&1 &
registry_pid=$!
for _ in $(seq 1 30); do
  if curl -fsS "${registry_url}/health" >/dev/null 2>&1; then
    break
  fi
  sleep 0.5
done
curl -fsS "${registry_url}/health" >/dev/null

if bin/ardctl admin list --registry-url "${registry_url}" >/tmp/ard-e2e-no-token.log 2>&1; then
  echo "admin list unexpectedly succeeded without token" >&2
  exit 1
fi
grep -q "admin token is required" /tmp/ard-e2e-no-token.log

bin/ardctl admin add catalog ./internal/catalog/testdata/acme-ai-catalog.json \
  --registry-url "${registry_url}" \
  --admin-token "${admin_token}"
bin/ardctl admin add mcp "${mcp_card_url}" \
  --registry-url "${registry_url}" \
  --admin-token "${admin_token}"
bin/ardctl admin add skill "${skill_url}" \
  --publisher github.com \
  --registry-url "${registry_url}" \
  --admin-token "${admin_token}"
bin/ardctl admin add a2a "http://127.0.0.1:${fixture_port}/a2a-agent-card.json" \
  --publisher example.com \
  --registry-url "${registry_url}" \
  --admin-token "${admin_token}"

bin/ardctl admin list --kind mcp --registry-url "${registry_url}" --admin-token "${admin_token}" --json >/tmp/ard-e2e-list-mcp.json
grep -q "Agentmemory MCP" /tmp/ard-e2e-list-mcp.json
grep -q "Weather Data Node" /tmp/ard-e2e-list-mcp.json

bin/ardctl admin export catalog --registry-url "${registry_url}" --admin-token "${admin_token}" -o "${export_file}"
grep -q "Agentmemory MCP" "${export_file}"
grep -q "open-browser-use" "${export_file}"
grep -q "Hello World Agent" "${export_file}"
bin/ard verify catalog "${export_file}" --json | grep -q '"valid": true'

if [ -x "${conformance_bin}" ]; then
  "${conformance_bin}" manifest "${export_file}"
fi

bin/ardctl search memory --registry-url "${registry_url}" --kind mcp --json | grep -q "Agentmemory MCP"
bin/ardctl search browser --registry-url "${registry_url}" --kind skill --json | grep -q "open-browser-use"
bin/ardctl search hello --registry-url "${registry_url}" --kind a2a --json | grep -q "Hello World Agent"

bin/ardctl admin remove urn:air:raw.githubusercontent.com:server:agentmemory-mcp \
  --registry-url "${registry_url}" \
  --admin-token "${admin_token}" \
  --yes | grep -q "remote removed urn:air:raw.githubusercontent.com:server:agentmemory-mcp"

bin/ardctl admin list --kind mcp --registry-url "${registry_url}" --admin-token "${admin_token}" --json >/tmp/ard-e2e-list-after-remove.json
if grep -q "Agentmemory MCP" /tmp/ard-e2e-list-after-remove.json; then
  echo "removed MCP entry still listed" >&2
  exit 1
fi
if bin/ardctl search memory --registry-url "${registry_url}" --kind mcp --json | grep -q "Agentmemory MCP"; then
  echo "removed MCP entry still searchable" >&2
  exit 1
fi

if [ -x "${conformance_bin}" ]; then
  "${conformance_bin}" registry "${registry_url}"
else
  echo "skipping ard-spec conformance; set ARD_CONFORMANCE_BIN to enable it" >&2
fi
