#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
workdir="$(mktemp -d /tmp/ard-public-go-client-XXXXXX)"
cleanup() {
  rm -rf "${workdir}"
}
trap cleanup EXIT

cd "${workdir}"
go mod init example.com/ard-public-go-client-check >/dev/null
go mod edit -require github.com/ifuryst/ard@v0.0.0
go mod edit -replace "github.com/ifuryst/ard=${repo_root}"

cat >client_test.go <<'GO'
package ardpublicclientcheck

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ifuryst/ard/pkg/ard"
	"github.com/ifuryst/ard/pkg/client"
)

func TestPublicGoClientImportsFromExternalModule(t *testing.T) {
	identifier := "urn:air:example.com:server:weather"
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		switch {
		case request.URL.Path == "/search":
			if request.Method != http.MethodPost {
				t.Fatalf("unexpected search method: %s", request.Method)
			}
			_, _ = response.Write([]byte(`{"results":[{"identifier":"urn:air:example.com:server:weather","displayName":"Weather","type":"application/mcp-server-card+json","url":"https://example.com/mcp.json","score":100,"source":"local"}],"pageToken":"next-search"}`))
		case request.URL.Path == "/agents":
			if got := request.URL.Query().Get("filter"); got != "publisherId = 'example.com'" {
				t.Fatalf("unexpected browse filter: %s", got)
			}
			_, _ = response.Write([]byte(`{"items":[{"identifier":"urn:air:example.com:server:weather","displayName":"Weather","type":"application/mcp-server-card+json","url":"https://example.com/mcp.json"}],"total":1,"pageToken":"next-agent"}`))
		case request.URL.Path == "/explore":
			_, _ = response.Write([]byte(`{"resultType":"facets","facets":{"type":{"buckets":[{"value":"application/mcp-server-card+json","count":1}]}}}`))
		case request.URL.Path == "/.well-known/ai-catalog.json":
			_, _ = response.Write([]byte(`{"specVersion":"1.0","host":{"displayName":"Example"},"entries":[{"identifier":"urn:air:example.com:server:weather","displayName":"Weather","type":"application/mcp-server-card+json","url":"https://example.com/mcp.json"}]}`))
		case request.URL.Path == "/health":
			_, _ = response.Write([]byte(`{"status":"ok","entries":1,"version":"v0.1.0","commit":"abc123","buildDate":"2026-06-21T00:00:00Z"}`))
		case request.URL.Path == "/metrics":
			_, _ = response.Write([]byte("# TYPE ard_http_requests_total counter\nard_http_requests_total 1\n"))
		case request.Method == http.MethodGet && request.URL.Path == "/admin/entries":
			if request.Header.Get("Authorization") != "Bearer token" {
				http.Error(response, "missing admin token", http.StatusUnauthorized)
				return
			}
			query := request.URL.Query()
			if query.Get("kind") != "mcp" || query.Get("status") != "active" || query.Get("pageToken") != "next-admin" {
				t.Fatalf("unexpected admin list query: %s", request.URL.RawQuery)
			}
			_, _ = response.Write([]byte(`{"items":[{"identifier":"urn:air:example.com:server:weather","displayName":"Weather","type":"application/mcp-server-card+json","url":"https://example.com/mcp.json","metadata":{"ard.status":"active"}}],"total":1,"pageToken":"later-admin"}`))
		case request.Method == http.MethodGet && request.URL.Path == "/admin/reviews":
			requireAdmin(t, request)
			_, _ = response.Write([]byte(`{"items":[{"identifier":"urn:air:example.com:server:weather","displayName":"Weather","type":"application/mcp-server-card+json","url":"https://example.com/mcp.json","metadata":{"ard.status":"pending"}}],"total":1}`))
		case request.Method == http.MethodGet && request.URL.Path == "/admin/catalog":
			requireAdmin(t, request)
			_, _ = response.Write([]byte(`{"specVersion":"1.0","host":{"displayName":"Example"},"entries":[{"identifier":"urn:air:example.com:server:weather","displayName":"Weather","type":"application/mcp-server-card+json","url":"https://example.com/mcp.json"}]}`))
		case request.Method == http.MethodPost && request.URL.Path == "/admin/entries":
			requireAdmin(t, request)
			if !strings.Contains(string(readBody(t, request)), identifier) {
				t.Fatal("admin upsert entry body omitted identifier")
			}
			response.WriteHeader(http.StatusCreated)
			_, _ = response.Write([]byte(`{"identifier":"urn:air:example.com:server:weather","displayName":"Weather","type":"application/mcp-server-card+json","url":"https://example.com/mcp.json"}`))
		case request.Method == http.MethodPost && request.URL.Path == "/admin/catalogs":
			requireAdmin(t, request)
			if !strings.Contains(string(readBody(t, request)), `"specVersion":"1.0"`) {
				t.Fatal("admin catalog import body omitted spec version")
			}
			response.WriteHeader(http.StatusCreated)
			_, _ = response.Write([]byte(`{"entries":1}`))
		case request.Method == http.MethodPatch && request.URL.EscapedPath() == "/admin/entries/urn:air:example.com:server:weather/status":
			requireAdmin(t, request)
			var payload map[string]string
			if err := json.Unmarshal(readBody(t, request), &payload); err != nil {
				t.Fatalf("decode status body: %v", err)
			}
			if payload["status"] != "disabled" {
				t.Fatalf("unexpected status payload: %#v", payload)
			}
			_, _ = response.Write([]byte(`{"identifier":"urn:air:example.com:server:weather","status":"disabled"}`))
		case request.Method == http.MethodPost && request.URL.EscapedPath() == "/admin/reviews/urn:air:example.com:server:weather/approve":
			requireAdmin(t, request)
			_, _ = response.Write([]byte(`{"identifier":"urn:air:example.com:server:weather","status":"active","reason":"reviewed","approvals":1,"requiredApprovals":1}`))
		case request.Method == http.MethodPost && request.URL.EscapedPath() == "/admin/reviews/urn:air:example.com:server:weather/reject":
			requireAdmin(t, request)
			_, _ = response.Write([]byte(`{"identifier":"urn:air:example.com:server:weather","status":"disabled"}`))
		case request.Method == http.MethodGet && request.URL.Path == "/admin/audit":
			requireAdmin(t, request)
			_, _ = response.Write([]byte(`{"items":[{"id":"event-1","action":"entry.status","identifier":"urn:air:example.com:server:weather","status":"disabled","source":"admin-api","hash":"abc","createdAt":"2026-06-21T00:00:00Z"}],"total":1,"pageToken":"next-audit"}`))
		case request.Method == http.MethodGet && request.URL.Path == "/admin/audit/verify":
			if request.Header.Get("Authorization") != "Bearer token" {
				http.Error(response, "missing admin token", http.StatusUnauthorized)
				return
			}
			_, _ = response.Write([]byte(`{"valid":true,"total":1,"lastHash":"abc"}`))
		case request.Method == http.MethodDelete && request.URL.EscapedPath() == "/admin/entries/urn:air:example.com:server:weather":
			requireAdmin(t, request)
			response.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()

	registry, err := client.New(server.URL)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	results, err := registry.Search(context.Background(), ard.SearchRequest{
		Query: ard.SearchQuery{Text: "weather"},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results.Results) != 1 || results.Results[0].Type != ard.TypeMCPServerCard || results.PageToken != "next-search" {
		t.Fatalf("unexpected search response: %#v", results)
	}
	list, err := registry.Browse(context.Background(), client.BrowseOptions{
		PageSize:  1,
		PageToken: "cursor",
		Filter:    "publisherId = 'example.com'",
		OrderBy:   "displayName DESC",
	})
	if err != nil {
		t.Fatalf("browse: %v", err)
	}
	if len(list.Items) != 1 || list.PageToken != "next-agent" {
		t.Fatalf("unexpected browse response: %#v", list)
	}
	explore, err := registry.Explore(context.Background(), ard.ExploreRequest{
		ResultType: ard.ExploreResultType{Facets: []ard.ExploreFacetRequest{{Field: "type"}}},
	})
	if err != nil {
		t.Fatalf("explore: %v", err)
	}
	if len(explore.Facets["type"].Buckets) != 1 {
		t.Fatalf("unexpected explore response: %#v", explore)
	}
	catalog, err := registry.Catalog(context.Background())
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	if catalog.SpecVersion != "1.0" || len(catalog.Entries) != 1 {
		t.Fatalf("unexpected catalog response: %#v", catalog)
	}
	health, err := registry.Health(context.Background())
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	if health.Status != "ok" || health.Commit != "abc123" {
		t.Fatalf("unexpected health response: %#v", health)
	}
	metrics, err := registry.Metrics(context.Background())
	if err != nil {
		t.Fatalf("metrics: %v", err)
	}
	if !strings.Contains(metrics, "ard_http_requests_total") {
		t.Fatalf("unexpected metrics response: %s", metrics)
	}
	if err := ard.ValidateCatalogEntry(catalog.Entries[0]); err != nil {
		t.Fatalf("validate catalog entry: %v", err)
	}
	if publisher := ard.Publisher(identifier); publisher != "example.com" {
		t.Fatalf("unexpected publisher: %s", publisher)
	}

	adminRegistry, err := client.New(server.URL, client.WithAdminToken("token"))
	if err != nil {
		t.Fatalf("new admin client: %v", err)
	}
	adminList, err := adminRegistry.AdminList(context.Background(), client.AdminListOptions{
		Kind:      "mcp",
		Type:      ard.TypeMCPServerCard,
		Status:    "active",
		PageSize:  1,
		PageToken: "next-admin",
	})
	if err != nil {
		t.Fatalf("admin list: %v", err)
	}
	if len(adminList.Items) != 1 || adminList.PageToken != "later-admin" {
		t.Fatalf("unexpected admin list response: %#v", adminList)
	}
	reviews, err := adminRegistry.AdminReviews(context.Background(), client.AdminReviewOptions{PageSize: 1})
	if err != nil {
		t.Fatalf("admin reviews: %v", err)
	}
	if len(reviews.Items) != 1 {
		t.Fatalf("unexpected admin reviews response: %#v", reviews)
	}
	adminCatalog, err := adminRegistry.AdminExportCatalog(context.Background())
	if err != nil {
		t.Fatalf("admin catalog: %v", err)
	}
	if len(adminCatalog.Entries) != 1 {
		t.Fatalf("unexpected admin catalog response: %#v", adminCatalog)
	}
	entry := ard.CatalogEntry{
		Identifier:  identifier,
		DisplayName: "Weather",
		Type:        ard.TypeMCPServerCard,
		URL:         "https://example.com/mcp.json",
	}
	upserted, err := adminRegistry.AdminUpsertEntry(context.Background(), entry)
	if err != nil {
		t.Fatalf("admin upsert entry: %v", err)
	}
	if upserted.Identifier != identifier {
		t.Fatalf("unexpected upserted entry: %#v", upserted)
	}
	imported, err := adminRegistry.AdminUpsertCatalog(context.Background(), ard.Catalog{SpecVersion: "1.0", Entries: []ard.CatalogEntry{entry}})
	if err != nil {
		t.Fatalf("admin upsert catalog: %v", err)
	}
	if imported.Entries != 1 {
		t.Fatalf("unexpected import response: %#v", imported)
	}
	status, err := adminRegistry.AdminSetStatus(context.Background(), identifier, "disabled")
	if err != nil {
		t.Fatalf("admin set status: %v", err)
	}
	if status.Status != "disabled" {
		t.Fatalf("unexpected status response: %#v", status)
	}
	approved, err := adminRegistry.AdminApproveReview(context.Background(), identifier, "reviewed")
	if err != nil {
		t.Fatalf("admin approve: %v", err)
	}
	if approved.Status != "active" || approved.Reason != "reviewed" {
		t.Fatalf("unexpected approve response: %#v", approved)
	}
	rejected, err := adminRegistry.AdminRejectReview(context.Background(), identifier, "")
	if err != nil {
		t.Fatalf("admin reject: %v", err)
	}
	if rejected.Status != "disabled" {
		t.Fatalf("unexpected reject response: %#v", rejected)
	}
	audit, err := adminRegistry.AdminAudit(context.Background(), client.AdminAuditOptions{PageSize: 1})
	if err != nil {
		t.Fatalf("admin audit: %v", err)
	}
	if len(audit.Items) != 1 || audit.PageToken != "next-audit" {
		t.Fatalf("unexpected audit response: %#v", audit)
	}
	verification, err := adminRegistry.AdminVerifyAudit(context.Background())
	if err != nil {
		t.Fatalf("admin verify audit: %v", err)
	}
	if !verification.Valid {
		t.Fatalf("unexpected audit verification: %#v", verification)
	}
	if err := adminRegistry.AdminDeleteEntry(context.Background(), identifier); err != nil {
		t.Fatalf("admin delete: %v", err)
	}
}

func TestPublicGoClientHTTPErrorType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		http.Error(response, "bad search", http.StatusBadRequest)
	}))
	defer server.Close()

	registry, err := client.New(server.URL)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	_, err = registry.Search(context.Background(), ard.SearchRequest{Query: ard.SearchQuery{Text: "weather"}})
	if err == nil {
		t.Fatal("expected search error")
	}
	var httpError client.HTTPError
	if !errors.As(err, &httpError) {
		t.Fatalf("expected client.HTTPError, got %T: %v", err, err)
	}
	if httpError.StatusCode != http.StatusBadRequest || !strings.Contains(err.Error(), "bad search") {
		t.Fatalf("unexpected HTTP error: %#v %v", httpError, err)
	}
}

func requireAdmin(t *testing.T, request *http.Request) {
	t.Helper()
	if request.Header.Get("Authorization") != "Bearer token" {
		t.Fatalf("missing admin token: %s", request.Header.Get("Authorization"))
	}
}

func readBody(t *testing.T, request *http.Request) []byte {
	t.Helper()
	body, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return body
}
GO

go test ./...
