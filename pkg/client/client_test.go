package client

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
)

func TestClientPublicRegistryFlow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("User-Agent"); got != "ard-test-client" {
			t.Fatalf("unexpected user agent: %s", got)
		}
		if got := request.Header.Get("X-Test-Client"); got != "sdk" {
			t.Fatalf("missing custom header: %s", got)
		}
		response.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/search":
			if request.Method != http.MethodPost {
				t.Fatalf("unexpected search method: %s", request.Method)
			}
			_, _ = response.Write([]byte(`{"results":[{"identifier":"urn:air:example.com:server:weather","displayName":"Weather","type":"application/mcp-server-card+json","url":"https://example.com/mcp.json","score":91,"source":"local"}],"pageToken":"next-search"}`))
		case "/agents":
			query := request.URL.Query()
			if got := query.Get("pageSize"); got != "5" {
				t.Fatalf("unexpected pageSize: %s", got)
			}
			if got := query.Get("pageToken"); got != "next-agent" {
				t.Fatalf("unexpected pageToken: %s", got)
			}
			if got := query.Get("filter"); got != "publisherId = 'example.com'" {
				t.Fatalf("unexpected filter: %s", got)
			}
			if got := query.Get("orderBy"); got != "displayName DESC" {
				t.Fatalf("unexpected orderBy: %s", got)
			}
			_, _ = response.Write([]byte(`{"items":[{"identifier":"urn:air:example.com:server:weather","displayName":"Weather","type":"application/mcp-server-card+json","url":"https://example.com/mcp.json"}],"total":1}`))
		case "/explore":
			if request.Method != http.MethodPost {
				t.Fatalf("unexpected explore method: %s", request.Method)
			}
			_, _ = response.Write([]byte(`{"resultType":"facets","facets":{"type":{"buckets":[{"value":"application/mcp-server-card+json","count":1}]}}}`))
		case "/.well-known/ai-catalog.json":
			_, _ = response.Write([]byte(`{"specVersion":"1.0","host":{"displayName":"Example"},"entries":[{"identifier":"urn:air:example.com:server:weather","displayName":"Weather","type":"application/mcp-server-card+json","url":"https://example.com/mcp.json"}]}`))
		case "/health":
			_, _ = response.Write([]byte(`{"status":"ok","entries":1,"version":"v0.1.0","commit":"abc123","buildDate":"2026-06-21T00:00:00Z"}`))
		case "/metrics":
			if got := request.Header.Get("Accept"); got != "text/plain" {
				t.Fatalf("unexpected metrics accept header: %s", got)
			}
			response.Header().Set("Content-Type", "text/plain; version=0.0.4")
			_, _ = response.Write([]byte("# TYPE ard_http_requests_total counter\nard_http_requests_total 1\n"))
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()

	registry, err := New(server.URL, WithUserAgent("ard-test-client"), WithHeader("X-Test-Client", "sdk"))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	search, err := registry.Search(context.Background(), ard.SearchRequest{
		Query: ard.SearchQuery{Text: "weather"},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(search.Results) != 1 || search.Results[0].Score != 91 || search.PageToken != "next-search" {
		t.Fatalf("unexpected search response: %#v", search)
	}

	list, err := registry.Browse(context.Background(), BrowseOptions{
		PageSize:  5,
		PageToken: "next-agent",
		Filter:    "publisherId = 'example.com'",
		OrderBy:   "displayName DESC",
	})
	if err != nil {
		t.Fatalf("browse: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].Identifier != "urn:air:example.com:server:weather" {
		t.Fatalf("unexpected browse response: %#v", list)
	}

	explore, err := registry.Explore(context.Background(), ard.ExploreRequest{
		ResultType: ard.ExploreResultType{
			Facets: []ard.ExploreFacetRequest{{Field: "type"}},
		},
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
	if health.Status != "ok" || health.Entries != 1 || health.Version != "v0.1.0" || health.Commit != "abc123" {
		t.Fatalf("unexpected health response: %#v", health)
	}
	metrics, err := registry.Metrics(context.Background())
	if err != nil {
		t.Fatalf("metrics: %v", err)
	}
	if !strings.Contains(metrics, "ard_http_requests_total") {
		t.Fatalf("unexpected metrics response: %s", metrics)
	}
}

func TestClientAdminRegistryFlow(t *testing.T) {
	identifier := "urn:air:example.com:server:weather"
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("Authorization"); got != "Bearer admin-token" {
			t.Fatalf("unexpected authorization header: %s", got)
		}
		response.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/admin/entries":
			query := request.URL.Query()
			if got := query.Get("kind"); got != "mcp" {
				t.Fatalf("unexpected kind: %s", got)
			}
			if got := query.Get("type"); got != ard.TypeMCPServerCard {
				t.Fatalf("unexpected type: %s", got)
			}
			if got := query.Get("status"); got != "active" {
				t.Fatalf("unexpected status: %s", got)
			}
			if got := query.Get("pageSize"); got != "5" {
				t.Fatalf("unexpected pageSize: %s", got)
			}
			if got := query.Get("pageToken"); got != "next-admin" {
				t.Fatalf("unexpected pageToken: %s", got)
			}
			_, _ = response.Write([]byte(`{"items":[{"identifier":"urn:air:example.com:server:weather","displayName":"Weather","type":"application/mcp-server-card+json","url":"https://example.com/mcp.json"}],"total":1}`))
		case request.Method == http.MethodGet && request.URL.Path == "/admin/reviews":
			query := request.URL.Query()
			if got := query.Get("pageSize"); got != "7" {
				t.Fatalf("unexpected review pageSize: %s", got)
			}
			if got := query.Get("pageToken"); got != "next-review" {
				t.Fatalf("unexpected review pageToken: %s", got)
			}
			_, _ = response.Write([]byte(`{"items":[{"identifier":"urn:air:example.com:server:weather","displayName":"Weather","type":"application/mcp-server-card+json","url":"https://example.com/mcp.json","metadata":{"ard.status":"pending"}}],"total":1}`))
		case request.Method == http.MethodGet && request.URL.Path == "/admin/catalog":
			_, _ = response.Write([]byte(`{"specVersion":"1.0","host":{"displayName":"Example"},"entries":[{"identifier":"urn:air:example.com:server:weather","displayName":"Weather","type":"application/mcp-server-card+json","url":"https://example.com/mcp.json"}]}`))
		case request.Method == http.MethodPost && request.URL.Path == "/admin/entries":
			body := readTestBody(t, request)
			if !strings.Contains(string(body), identifier) {
				t.Fatalf("entry upsert body did not include identifier: %s", body)
			}
			response.WriteHeader(http.StatusCreated)
			_, _ = response.Write([]byte(`{"identifier":"urn:air:example.com:server:weather","displayName":"Weather","type":"application/mcp-server-card+json","url":"https://example.com/mcp.json"}`))
		case request.Method == http.MethodPost && request.URL.Path == "/admin/catalogs":
			body := readTestBody(t, request)
			if !strings.Contains(string(body), `"specVersion":"1.0"`) {
				t.Fatalf("catalog upsert body did not include spec version: %s", body)
			}
			response.WriteHeader(http.StatusCreated)
			_, _ = response.Write([]byte(`{"entries":1}`))
		case request.Method == http.MethodPatch && request.URL.EscapedPath() == "/admin/entries/urn:air:example.com:server:weather/status":
			var payload map[string]string
			if err := json.Unmarshal(readTestBody(t, request), &payload); err != nil {
				t.Fatalf("decode status payload: %v", err)
			}
			if payload["status"] != "disabled" {
				t.Fatalf("unexpected status payload: %#v", payload)
			}
			_, _ = response.Write([]byte(`{"identifier":"urn:air:example.com:server:weather","status":"disabled"}`))
		case request.Method == http.MethodPost && request.URL.EscapedPath() == "/admin/reviews/urn:air:example.com:server:weather/approve":
			var payload map[string]string
			if err := json.Unmarshal(readTestBody(t, request), &payload); err != nil {
				t.Fatalf("decode approve payload: %v", err)
			}
			if payload["reason"] != "reviewed" {
				t.Fatalf("unexpected approve payload: %#v", payload)
			}
			_, _ = response.Write([]byte(`{"identifier":"urn:air:example.com:server:weather","status":"active","reason":"reviewed","approvals":2,"requiredApprovals":2}`))
		case request.Method == http.MethodPost && request.URL.EscapedPath() == "/admin/reviews/urn:air:example.com:server:weather/reject":
			_, _ = response.Write([]byte(`{"identifier":"urn:air:example.com:server:weather","status":"disabled"}`))
		case request.Method == http.MethodGet && request.URL.Path == "/admin/audit":
			query := request.URL.Query()
			if got := query.Get("pageSize"); got != "2" {
				t.Fatalf("unexpected audit pageSize: %s", got)
			}
			if got := query.Get("pageToken"); got != "next-audit" {
				t.Fatalf("unexpected audit pageToken: %s", got)
			}
			_, _ = response.Write([]byte(`{"items":[{"id":"event-1","action":"entry.status","identifier":"urn:air:example.com:server:weather","status":"disabled","source":"admin-api","hash":"abc","createdAt":"2026-06-21T00:00:00Z"}],"total":1,"pageToken":"later"}`))
		case request.Method == http.MethodGet && request.URL.Path == "/admin/audit/verify":
			_, _ = response.Write([]byte(`{"valid":true,"total":1,"lastHash":"abc"}`))
		case request.Method == http.MethodDelete && request.URL.EscapedPath() == "/admin/entries/urn:air:example.com:server:weather":
			response.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()

	registry, err := New(server.URL, WithAdminToken("admin-token"))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	entry := ard.CatalogEntry{
		Identifier:  identifier,
		DisplayName: "Weather",
		Type:        ard.TypeMCPServerCard,
		URL:         "https://example.com/mcp.json",
	}

	list, err := registry.AdminList(context.Background(), AdminListOptions{
		Kind:      "mcp",
		Type:      ard.TypeMCPServerCard,
		Status:    "active",
		PageSize:  5,
		PageToken: "next-admin",
	})
	if err != nil {
		t.Fatalf("admin list: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].Identifier != identifier {
		t.Fatalf("unexpected admin list response: %#v", list)
	}

	reviews, err := registry.AdminReviews(context.Background(), AdminReviewOptions{PageSize: 7, PageToken: "next-review"})
	if err != nil {
		t.Fatalf("admin reviews: %v", err)
	}
	if len(reviews.Items) != 1 {
		t.Fatalf("unexpected admin reviews response: %#v", reviews)
	}

	catalog, err := registry.AdminExportCatalog(context.Background())
	if err != nil {
		t.Fatalf("admin export catalog: %v", err)
	}
	if len(catalog.Entries) != 1 {
		t.Fatalf("unexpected admin catalog response: %#v", catalog)
	}

	upserted, err := registry.AdminUpsertEntry(context.Background(), entry)
	if err != nil {
		t.Fatalf("admin upsert entry: %v", err)
	}
	if upserted.Identifier != identifier {
		t.Fatalf("unexpected upserted entry: %#v", upserted)
	}

	imported, err := registry.AdminUpsertCatalog(context.Background(), ard.Catalog{SpecVersion: "1.0", Entries: []ard.CatalogEntry{entry}})
	if err != nil {
		t.Fatalf("admin upsert catalog: %v", err)
	}
	if imported.Entries != 1 {
		t.Fatalf("unexpected import response: %#v", imported)
	}

	status, err := registry.AdminSetStatus(context.Background(), identifier, "disabled")
	if err != nil {
		t.Fatalf("admin set status: %v", err)
	}
	if status.Status != "disabled" {
		t.Fatalf("unexpected status response: %#v", status)
	}

	approved, err := registry.AdminApproveReview(context.Background(), identifier, "reviewed")
	if err != nil {
		t.Fatalf("admin approve review: %v", err)
	}
	if approved.Status != "active" || approved.Reason != "reviewed" || approved.Approvals != 2 || approved.RequiredApprovals != 2 {
		t.Fatalf("unexpected approve response: %#v", approved)
	}

	rejected, err := registry.AdminRejectReview(context.Background(), identifier, "")
	if err != nil {
		t.Fatalf("admin reject review: %v", err)
	}
	if rejected.Status != "disabled" {
		t.Fatalf("unexpected reject response: %#v", rejected)
	}

	audit, err := registry.AdminAudit(context.Background(), AdminAuditOptions{PageSize: 2, PageToken: "next-audit"})
	if err != nil {
		t.Fatalf("admin audit: %v", err)
	}
	if len(audit.Items) != 1 || audit.Items[0].Hash != "abc" || audit.PageToken != "later" {
		t.Fatalf("unexpected audit response: %#v", audit)
	}

	verification, err := registry.AdminVerifyAudit(context.Background())
	if err != nil {
		t.Fatalf("admin verify audit: %v", err)
	}
	if !verification.Valid || verification.LastHash != "abc" {
		t.Fatalf("unexpected audit verification: %#v", verification)
	}

	if err := registry.AdminDeleteEntry(context.Background(), identifier); err != nil {
		t.Fatalf("admin delete entry: %v", err)
	}
}

func TestClientHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		http.Error(response, "bad search", http.StatusBadRequest)
	}))
	defer server.Close()

	registry, err := New(server.URL)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	_, err = registry.Search(context.Background(), ard.SearchRequest{Query: ard.SearchQuery{Text: "weather"}})
	if err == nil {
		t.Fatal("expected search to fail")
	}
	var httpError HTTPError
	if !errors.As(err, &httpError) {
		t.Fatalf("expected HTTPError, got %T: %v", err, err)
	}
	if httpError.StatusCode != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", httpError.StatusCode)
	}
	if !strings.Contains(err.Error(), "bad search") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func readTestBody(t *testing.T, request *http.Request) []byte {
	t.Helper()
	body, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	return body
}

func TestNewRejectsRelativeRegistryURL(t *testing.T) {
	if _, err := New("localhost:8080"); err == nil {
		t.Fatal("expected relative registry URL to be rejected")
	}
}
