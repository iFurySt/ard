package client

import (
	"context"
	"errors"
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
			_, _ = response.Write([]byte(`{"status":"ok","entries":1}`))
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
	if health.Status != "ok" || health.Entries != 1 {
		t.Fatalf("unexpected health response: %#v", health)
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

func TestNewRejectsRelativeRegistryURL(t *testing.T) {
	if _, err := New("localhost:8080"); err == nil {
		t.Fatal("expected relative registry URL to be rejected")
	}
}
