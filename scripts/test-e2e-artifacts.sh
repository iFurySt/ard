#!/usr/bin/env bash
set -euo pipefail

postgres_container="${ARD_E2E_POSTGRES_CONTAINER:-ard-e2e-postgres}"
postgres_port="${ARD_E2E_POSTGRES_PORT:-55440}"
fixture_port="${ARD_E2E_FIXTURE_PORT:-18087}"
registry_port="${ARD_E2E_REGISTRY_PORT:-18088}"
upstream_port="${ARD_E2E_UPSTREAM_PORT:-18089}"
admin_token="${ARD_E2E_ADMIN_TOKEN:-test-token}"
database_url="postgres://ard:ard@127.0.0.1:${postgres_port}/ard?sslmode=disable"
registry_url="http://127.0.0.1:${registry_port}"
export_file="$(mktemp /tmp/ard-e2e-export-XXXXXX.json)"
referral_catalog_file="$(mktemp /tmp/ard-e2e-referral-catalog-XXXXXX.json)"
policy_file="$(mktemp /tmp/ard-e2e-policy-XXXXXX.json)"
tokens_file="$(mktemp /tmp/ard-e2e-tokens-XXXXXX.json)"
mcp_card_file="$(mktemp /tmp/ard-e2e-mcp-card-XXXXXX.json)"
skill_file="$(mktemp /tmp/ard-e2e-skill-XXXXXX.md)"
openapi_file="$(mktemp /tmp/ard-e2e-openapi-XXXXXX.json)"
fixture_server_file="$(mktemp /tmp/ard-e2e-fixture-XXXXXX.py)"
upstream_server_file="$(mktemp /tmp/ard-e2e-upstream-XXXXXX.py)"
conformance_bin="${ARD_CONFORMANCE_BIN:-../ard-spec/conformance/bin/conformance-test}"

mcp_card_url="https://raw.githubusercontent.com/clauxel/agentmemory-mcp/main/server.json"
skill_url="https://raw.githubusercontent.com/iFurySt/open-codex-browser-use/main/skills/open-browser-use/SKILL.md"
skill_fallback="internal/adapters/testdata/open-browser-use/SKILL.md"
openapi_url="https://petstore3.swagger.io/api/v3/openapi.json"

cleanup() {
  if [ -n "${registry_pid:-}" ]; then
    kill "${registry_pid}" >/dev/null 2>&1 || true
    wait "${registry_pid}" >/dev/null 2>&1 || true
  fi
  if [ -n "${fixture_pid:-}" ]; then
    kill "${fixture_pid}" >/dev/null 2>&1 || true
    wait "${fixture_pid}" >/dev/null 2>&1 || true
  fi
  if [ -n "${upstream_pid:-}" ]; then
    kill "${upstream_pid}" >/dev/null 2>&1 || true
    wait "${upstream_pid}" >/dev/null 2>&1 || true
  fi
  docker rm -f "${postgres_container}" >/dev/null 2>&1 || true
  rm -f "${export_file}" "${referral_catalog_file}" "${policy_file}" "${tokens_file}" "${mcp_card_file}" "${skill_file}" "${openapi_file}" "${fixture_server_file}" "${upstream_server_file}"
}
trap cleanup EXIT

fetch_with_retry() {
  local url="$1"
  local output="$2"
  local partial="${output}.part"

  for _ in $(seq 1 5); do
    rm -f "${partial}"
    if curl -fsSL "${url}" -o "${partial}"; then
      mv "${partial}" "${output}"
      break
    fi
    sleep 1
  done
  rm -f "${partial}"
  if [ ! -s "${output}" ]; then
    echo "failed to fetch ${url}" >&2
    return 1
  fi
}

make build

cat >"${policy_file}" <<'JSON'
{
  "version": "1",
  "pendingPublishers": ["pending.example.com"],
  "denyPublishers": ["blocked.example.com"]
}
JSON
cat >"${tokens_file}" <<'JSON'
{
  "version": "1",
  "tokens": [
    {"name": "reader", "token": "reader-token", "role": "reader"},
    {"name": "publisher", "token": "publisher-token", "role": "publisher"},
    {"name": "reviewer", "token": "reviewer-token", "role": "reviewer"},
    {"name": "operator", "token": "operator-token", "role": "operator"}
  ]
}
JSON
cat >"${referral_catalog_file}" <<JSON
{
  "specVersion": "1.0",
  "entries": [
    {
      "identifier": "urn:air:agent.localhost:registry:e2e-upstream",
      "displayName": "E2E Upstream Registry",
      "type": "application/ai-registry+json",
      "url": "http://127.0.0.1:${upstream_port}/search",
      "description": "Local upstream registry referral used by the E2E flow."
    }
  ]
}
JSON
cat >"${upstream_server_file}" <<'PY'
import json
import sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        if self.path != "/search":
            self.send_response(404)
            self.end_headers()
            return
        length = int(self.headers.get("content-length", "0"))
        if length:
            self.rfile.read(length)
        request_id = self.headers.get("x-request-id", "")
        traceparent = self.headers.get("traceparent", "")
        if request_id:
            print(json.dumps({"event": "upstream_request", "requestId": request_id, "traceparent": traceparent}), flush=True)
        payload = {
            "results": [
                {
                    "identifier": "urn:air:upstream.localhost:server:federated-weather",
                    "displayName": "Federated Weather MCP",
                    "type": "application/mcp-server-card+json",
                    "url": "https://upstream.localhost/mcp/weather.json",
                    "description": "MCP result returned by the E2E upstream registry.",
                    "score": 90,
                    "source": "e2e-upstream"
                }
            ]
        }
        data = json.dumps(payload).encode("utf-8")
        self.send_response(200)
        self.send_header("content-type", "application/json")
        self.send_header("content-length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def log_message(self, format, *args):
        return


ThreadingHTTPServer(("127.0.0.1", int(sys.argv[1])), Handler).serve_forever()
PY

cat >"${fixture_server_file}" <<'PY'
import json
import sys
from functools import partial
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer


class Handler(SimpleHTTPRequestHandler):
    def do_GET(self):
        request_id = self.headers.get("x-request-id", "")
        traceparent = self.headers.get("traceparent", "")
        if request_id:
            print(json.dumps({"event": "fixture_request", "requestId": request_id, "traceparent": traceparent, "path": self.path}), flush=True)
        return super().do_GET()

    def log_message(self, format, *args):
        return


directory = sys.argv[2]
handler = partial(Handler, directory=directory)
ThreadingHTTPServer(("127.0.0.1", int(sys.argv[1])), handler).serve_forever()
PY

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

python3 "${fixture_server_file}" "${fixture_port}" internal/adapters/testdata >/tmp/ard-e2e-fixtures.log 2>&1 &
fixture_pid=$!
for _ in $(seq 1 30); do
  if curl -fsS "http://127.0.0.1:${fixture_port}/a2a-agent-card.json" >/dev/null 2>&1; then
    break
  fi
  sleep 0.5
done
curl -fsS "http://127.0.0.1:${fixture_port}/a2a-agent-card.json" >/dev/null

python3 "${upstream_server_file}" "${upstream_port}" >/tmp/ard-e2e-upstream.log 2>&1 &
upstream_pid=$!
for _ in $(seq 1 30); do
  if curl -fsS -X POST "http://127.0.0.1:${upstream_port}/search" >/dev/null 2>&1; then
    break
  fi
  sleep 0.5
done
curl -fsS -X POST "http://127.0.0.1:${upstream_port}/search" >/dev/null

bin/ard-server \
  --database-url "${database_url}" \
  --addr "127.0.0.1:${registry_port}" \
  --admin-token "${admin_token}" \
  --admin-tokens-file "${tokens_file}" \
  --policy-file "${policy_file}" >/tmp/ard-e2e-registry.log 2>&1 &
registry_pid=$!
for _ in $(seq 1 30); do
  if curl -fsS "${registry_url}/health" >/dev/null 2>&1; then
    break
  fi
  sleep 0.5
done
curl -fsS "${registry_url}/health" >/dev/null

fetch_with_retry "${mcp_card_url}" "${mcp_card_file}"
if ! fetch_with_retry "${skill_url}" "${skill_file}"; then
  echo "falling back to checked-in Open Browser Use Skill fixture" >&2
  cp "${skill_fallback}" "${skill_file}"
fi
fetch_with_retry "${openapi_url}" "${openapi_file}"

if bin/ardctl admin list --registry-url "${registry_url}" >/tmp/ard-e2e-no-token.log 2>&1; then
  echo "admin list unexpectedly succeeded without token" >&2
  exit 1
fi
grep -q "admin token is required" /tmp/ard-e2e-no-token.log

bin/ardctl admin add catalog ./internal/catalog/testdata/acme-ai-catalog.json \
  --registry-url "${registry_url}" \
  --admin-token "${admin_token}"
bin/ardctl admin add catalog "${referral_catalog_file}" \
  --registry-url "${registry_url}" \
  --admin-token "${admin_token}"
bin/ardctl admin add mcp "${mcp_card_url}" \
  --publisher raw.githubusercontent.com \
  --pin-source-digest \
  --registry-url "${registry_url}" \
  --admin-token "${admin_token}"
bin/ardctl admin add skill "${skill_file}" \
  --publisher github.com \
  --registry-url "${registry_url}" \
  --admin-token "${admin_token}"
bin/ardctl admin add openapi "${openapi_file}" \
  --publisher petstore3.swagger.io \
  --registry-url "${registry_url}" \
  --admin-token "${admin_token}"
bin/ardctl admin add a2a "http://127.0.0.1:${fixture_port}/a2a-agent-card.json" \
  --publisher example.com \
  --registry-url "${registry_url}" \
  --admin-token "publisher-token" \
  --request-id "ard-e2e-artifact-fetch"
grep -q '"requestId": "ard-e2e-artifact-fetch"' /tmp/ard-e2e-fixtures.log
grep -q '"traceparent": "00-' /tmp/ard-e2e-fixtures.log

bin/ardctl admin list --kind mcp --registry-url "${registry_url}" --admin-token "${admin_token}" --json >/tmp/ard-e2e-list-mcp.json
grep -q "Agentmemory MCP" /tmp/ard-e2e-list-mcp.json
grep -q "Weather Data Node" /tmp/ard-e2e-list-mcp.json
bin/ardctl admin list --registry-url "${registry_url}" --admin-token "${admin_token}" --limit 1 --json >/tmp/ard-e2e-admin-list-page1.json
admin_page_token="$(python3 -c 'import json; print(json.load(open("/tmp/ard-e2e-admin-list-page1.json")).get("pageToken", ""))')"
if [ -z "${admin_page_token}" ]; then
  echo "admin list did not return a page token" >&2
  exit 1
fi
bin/ardctl admin list --registry-url "${registry_url}" --admin-token "${admin_token}" --limit 1 --page-token "${admin_page_token}" --json >/tmp/ard-e2e-admin-list-page2.json
grep -q '"items"' /tmp/ard-e2e-admin-list-page2.json
bin/ardctl admin list --kind mcp --registry-url "${registry_url}" --admin-token "reader-token" --json >/tmp/ard-e2e-reader-list-mcp.json
grep -q "Agentmemory MCP" /tmp/ard-e2e-reader-list-mcp.json
if bin/ardctl admin remove urn:air:raw.githubusercontent.com:server:agentmemory-mcp \
  --registry-url "${registry_url}" \
  --admin-token "reader-token" \
  --yes >/tmp/ard-e2e-rbac-deny.log 2>&1; then
  echo "reader token unexpectedly removed an entry" >&2
  exit 1
fi
grep -q "PERMISSION_DENIED" /tmp/ard-e2e-rbac-deny.log

bin/ardctl admin export catalog --registry-url "${registry_url}" --admin-token "${admin_token}" -o "${export_file}"
grep -q "Agentmemory MCP" "${export_file}"
grep -q "open-browser-use" "${export_file}"
grep -q "Swagger Petstore - OpenAPI 3.0" "${export_file}"
grep -q "Hello World Agent" "${export_file}"
bin/ard verify catalog "${export_file}" --json | grep -q '"valid": true'
bin/ard verify catalog "${export_file}" --source-digests --json >/tmp/ard-e2e-verify-digests.json
grep -q '"sourceDigestsVerified": 1' /tmp/ard-e2e-verify-digests.json
grep -q '"verified": true' /tmp/ard-e2e-verify-digests.json

if [ -x "${conformance_bin}" ]; then
  "${conformance_bin}" manifest "${export_file}"
fi

bin/ardctl search memory --registry-url "${registry_url}" --kind mcp --json | grep -q "Agentmemory MCP"
bin/ardctl search agent --registry-url "${registry_url}" --limit 1 --json >/tmp/ard-e2e-search-page1.json
search_page_token="$(python3 -c 'import json; print(json.load(open("/tmp/ard-e2e-search-page1.json")).get("pageToken", ""))')"
if [ -z "${search_page_token}" ]; then
  echo "search did not return a page token" >&2
  exit 1
fi
bin/ardctl search agent --registry-url "${registry_url}" --limit 1 --page-token "${search_page_token}" --json >/tmp/ard-e2e-search-page2.json
grep -q '"results"' /tmp/ard-e2e-search-page2.json
bin/ardctl search memory --registry-url "${registry_url}" --kind mcp --federation referrals --json >/tmp/ard-e2e-referrals-search.json
grep -q '"referrals"' /tmp/ard-e2e-referrals-search.json
grep -q "E2E Upstream Registry" /tmp/ard-e2e-referrals-search.json
bin/ardctl search federated --registry-url "${registry_url}" --kind mcp --federation auto --json >/tmp/ard-e2e-auto-search.json
grep -q "Federated Weather MCP" /tmp/ard-e2e-auto-search.json
bin/ardctl search "weather federated" --registry-url "${registry_url}" --kind mcp --federation auto --json >/tmp/ard-e2e-auto-ranked-search.json
python3 - <<'PY'
import json
data = json.load(open("/tmp/ard-e2e-auto-ranked-search.json"))
first = data["results"][0]["displayName"]
if first != "Federated Weather MCP":
    raise SystemExit(f"expected upstream ranked first, got {first!r}")
PY
curl -fsS \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: ard-e2e-auto-federation" \
  -H "traceparent: 00-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-bbbbbbbbbbbbbbbb-01" \
  -d '{"query":{"text":"federated","filter":{"type":["application/mcp-server-card+json"]}},"federation":"auto","pageSize":10}' \
  "${registry_url}/search" >/tmp/ard-e2e-auto-correlation.json
grep -q "Federated Weather MCP" /tmp/ard-e2e-auto-correlation.json
grep -q '"requestId": "ard-e2e-auto-federation"' /tmp/ard-e2e-upstream.log
grep -q '"traceparent": "00-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-' /tmp/ard-e2e-upstream.log
bin/ardctl search browser --registry-url "${registry_url}" --kind skill --json | grep -q "open-browser-use"
bin/ardctl search pet --registry-url "${registry_url}" --kind openapi --json | grep -q "Swagger Petstore - OpenAPI 3.0"
bin/ardctl search hello --registry-url "${registry_url}" --kind a2a --json | grep -q "Hello World Agent"

bin/ardctl admin add skill "${skill_file}" \
  --publisher pending.example.com \
  --registry-url "${registry_url}" \
  --admin-token "${admin_token}" | grep -q "remote imported"
bin/ardctl admin list --status pending --registry-url "${registry_url}" --admin-token "${admin_token}" --json >/tmp/ard-e2e-policy-pending.json
grep -q "urn:air:pending.example.com:skill:open-browser-use" /tmp/ard-e2e-policy-pending.json
bin/ardctl admin review list --registry-url "${registry_url}" --admin-token "${admin_token}" --json >/tmp/ard-e2e-review-list.json
grep -q "urn:air:pending.example.com:skill:open-browser-use" /tmp/ard-e2e-review-list.json
if bin/ardctl search pending.example --registry-url "${registry_url}" --kind skill --json | grep -q "pending.example.com"; then
  echo "policy pending entry is publicly searchable" >&2
  exit 1
fi
bin/ardctl admin review approve urn:air:pending.example.com:skill:open-browser-use \
  --registry-url "${registry_url}" \
  --admin-token "reviewer-token" | grep -q "remote approved urn:air:pending.example.com:skill:open-browser-use"
bin/ardctl search pending.example --registry-url "${registry_url}" --kind skill --json | grep -q "pending.example.com"
bin/ardctl admin add skill "${skill_file}" \
  --publisher pending.example.com \
  --registry-url "${registry_url}" \
  --admin-token "${admin_token}" | grep -q "remote imported"
bin/ardctl admin list --status pending --registry-url "${registry_url}" --admin-token "${admin_token}" --json >/tmp/ard-e2e-policy-pending-update.json
grep -q "urn:air:pending.example.com:skill:open-browser-use" /tmp/ard-e2e-policy-pending-update.json
if bin/ardctl search pending.example --registry-url "${registry_url}" --kind skill --json | grep -q "pending.example.com"; then
  echo "policy pending update is publicly searchable" >&2
  exit 1
fi
bin/ardctl admin review approve urn:air:pending.example.com:skill:open-browser-use \
  --registry-url "${registry_url}" \
  --admin-token "reviewer-token" | grep -q "remote approved urn:air:pending.example.com:skill:open-browser-use"
bin/ardctl admin status urn:air:pending.example.com:skill:open-browser-use pending \
  --registry-url "${registry_url}" \
  --admin-token "operator-token" | grep -q "remote set urn:air:pending.example.com:skill:open-browser-use status to pending"
bin/ardctl admin review reject urn:air:pending.example.com:skill:open-browser-use \
  --registry-url "${registry_url}" \
  --admin-token "reviewer-token" | grep -q "remote rejected urn:air:pending.example.com:skill:open-browser-use"
if bin/ardctl search pending.example --registry-url "${registry_url}" --kind skill --json | grep -q "pending.example.com"; then
  echo "rejected review entry is publicly searchable" >&2
  exit 1
fi
if bin/ardctl admin add skill "${skill_file}" \
  --publisher blocked.example.com \
  --registry-url "${registry_url}" \
  --admin-token "${admin_token}" >/tmp/ard-e2e-policy-deny.log 2>&1; then
  echo "policy denied publisher unexpectedly imported" >&2
  exit 1
fi
grep -q "POLICY_DENIED" /tmp/ard-e2e-policy-deny.log

bin/ardctl admin status urn:air:github.com:skill:open-browser-use disabled \
  --registry-url "${registry_url}" \
  --admin-token "${admin_token}" | grep -q "remote set urn:air:github.com:skill:open-browser-use status to disabled"
bin/ardctl admin list --status disabled --registry-url "${registry_url}" --admin-token "${admin_token}" --json >/tmp/ard-e2e-disabled.json
grep -q "open-browser-use" /tmp/ard-e2e-disabled.json
grep -q '"ard.status":"disabled"' /tmp/ard-e2e-disabled.json
if bin/ardctl search browser --registry-url "${registry_url}" --kind skill --json | grep -q "open-browser-use"; then
  echo "disabled skill entry still searchable" >&2
  exit 1
fi
bin/ardctl admin status urn:air:github.com:skill:open-browser-use active \
  --registry-url "${registry_url}" \
  --admin-token "${admin_token}" | grep -q "remote set urn:air:github.com:skill:open-browser-use status to active"
bin/ardctl search browser --registry-url "${registry_url}" --kind skill --json | grep -q "open-browser-use"
bin/ardctl admin audit --registry-url "${registry_url}" --admin-token "${admin_token}" --json >/tmp/ard-e2e-audit.json
grep -q '"action":"entry.status"' /tmp/ard-e2e-audit.json
grep -q '"identifier":"urn:air:github.com:skill:open-browser-use"' /tmp/ard-e2e-audit.json
grep -q '"requestId":"' /tmp/ard-e2e-audit.json
grep -q '"hash":"' /tmp/ard-e2e-audit.json
bin/ardctl admin audit --registry-url "${registry_url}" --admin-token "${admin_token}" --verify-chain | grep -q "remote audit chain valid"
bin/ardctl admin audit --registry-url "${registry_url}" --admin-token "${admin_token}" --limit 1 --json >/tmp/ard-e2e-audit-page1.json
audit_page_token="$(python3 -c 'import json; print(json.load(open("/tmp/ard-e2e-audit-page1.json")).get("pageToken", ""))')"
if [ -z "${audit_page_token}" ]; then
  echo "admin audit did not return a page token" >&2
  exit 1
fi
bin/ardctl admin audit --registry-url "${registry_url}" --admin-token "${admin_token}" --limit 1 --page-token "${audit_page_token}" --json >/tmp/ard-e2e-audit-page2.json
grep -q '"items"' /tmp/ard-e2e-audit-page2.json

cat >"${tokens_file}" <<'JSON'
{
  "version": "1",
  "tokens": [
    {"name": "rotated-reader", "token": "rotated-reader-token", "role": "reader"}
  ]
}
JSON
rotated_token_ready=0
for _ in $(seq 1 20); do
  if bin/ardctl admin list --kind mcp --registry-url "${registry_url}" --admin-token "rotated-reader-token" --json >/tmp/ard-e2e-rotated-reader.json 2>/tmp/ard-e2e-rotated-reader.err; then
    rotated_token_ready=1
    break
  fi
  sleep 0.2
done
if [ "${rotated_token_ready}" != "1" ]; then
  echo "rotated admin token did not become active" >&2
  cat /tmp/ard-e2e-rotated-reader.err >&2 || true
  exit 1
fi
grep -q "Weather Data Node" /tmp/ard-e2e-rotated-reader.json
if bin/ardctl admin list --kind mcp --registry-url "${registry_url}" --admin-token "reader-token" --json >/tmp/ard-e2e-old-reader.json 2>/tmp/ard-e2e-old-reader.err; then
  echo "old reader token remained active after token file rotation" >&2
  exit 1
fi
grep -q "HTTP 401" /tmp/ard-e2e-old-reader.err

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
