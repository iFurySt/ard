package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ifuryst/ard/internal/ard"
	"github.com/ifuryst/ard/internal/store"
)

func TestRouterSearchWithPostgres(t *testing.T) {
	databaseURL := os.Getenv("ARD_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set ARD_TEST_DATABASE_URL to run Postgres integration tests")
	}
	ctx := context.Background()
	registryStore, err := store.Open(databaseURL)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer registryStore.Close()
	if err := registryStore.AutoMigrate(); err != nil {
		t.Fatalf("migrate store: %v", err)
	}
	if err := registryStore.UpsertCatalog(ctx, ard.Catalog{
		SpecVersion: "1.0",
		Entries: []ard.CatalogEntry{
			{
				Identifier:            "urn:air:example.com:server:weather",
				DisplayName:           "Example Weather MCP",
				Type:                  ard.TypeMCPServerCard,
				URL:                   "https://example.com/mcp/weather.json",
				Description:           "Weather forecast MCP server.",
				RepresentativeQueries: []string{"what is the weather", "get a forecast"},
			},
		},
	}, "router-test"); err != nil {
		t.Fatalf("upsert catalog: %v", err)
	}

	router := NewRouter(registryStore)
	requestBody, _ := json.Marshal(ard.SearchRequest{
		Query: ard.SearchQuery{
			Text: "weather",
			Filter: ard.Filter{
				"type": []string{ard.TypeMCPServerCard},
			},
		},
		PageSize: 5,
	})
	request := httptest.NewRequest(http.MethodPost, "/search", bytes.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d: %s", response.Code, response.Body.String())
	}

	var parsed ard.SearchResponse
	if err := json.Unmarshal(response.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(parsed.Results) == 0 {
		t.Fatal("expected at least one search result")
	}
	if parsed.Results[0].Type != ard.TypeMCPServerCard {
		t.Fatalf("unexpected type: %s", parsed.Results[0].Type)
	}
}

func TestRouterAgentsAndExploreWithPostgres(t *testing.T) {
	databaseURL := os.Getenv("ARD_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set ARD_TEST_DATABASE_URL to run Postgres integration tests")
	}
	ctx := context.Background()
	registryStore, err := store.Open(databaseURL)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer registryStore.Close()
	if err := registryStore.AutoMigrate(); err != nil {
		t.Fatalf("migrate store: %v", err)
	}
	if err := registryStore.UpsertCatalog(ctx, ard.Catalog{
		SpecVersion: "1.0",
		Entries: []ard.CatalogEntry{
			{
				Identifier:            "urn:air:example.com:agent:travel",
				DisplayName:           "Travel Agent",
				Type:                  ard.TypeA2AAgentCard,
				URL:                   "https://example.com/a2a/travel.json",
				Description:           "Travel booking A2A agent.",
				RepresentativeQueries: []string{"book a flight", "plan a trip"},
			},
			{
				Identifier:            "urn:air:example.com:server:weather-facet",
				DisplayName:           "Weather Facet MCP",
				Type:                  ard.TypeMCPServerCard,
				URL:                   "https://example.com/mcp/weather-facet.json",
				Description:           "Weather forecast MCP server.",
				RepresentativeQueries: []string{"what is the weather", "get a forecast"},
			},
		},
	}, "router-explore-test"); err != nil {
		t.Fatalf("upsert catalog: %v", err)
	}

	router := NewRouter(registryStore)
	listRequest := httptest.NewRequest(http.MethodGet, "/agents?pageSize=10", nil)
	listResponse := httptest.NewRecorder()
	router.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected /agents HTTP 200, got %d: %s", listResponse.Code, listResponse.Body.String())
	}

	var list ard.ListResponse
	if err := json.Unmarshal(listResponse.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if list.Total < 2 {
		t.Fatalf("expected at least 2 entries, got total %d", list.Total)
	}

	exploreBody, _ := json.Marshal(ard.ExploreRequest{
		ResultType: ard.ExploreResultType{
			Facets: []ard.ExploreFacetRequest{{Field: "type"}},
		},
	})
	exploreRequest := httptest.NewRequest(http.MethodPost, "/explore", bytes.NewReader(exploreBody))
	exploreRequest.Header.Set("Content-Type", "application/json")
	exploreResponse := httptest.NewRecorder()
	router.ServeHTTP(exploreResponse, exploreRequest)
	if exploreResponse.Code != http.StatusOK {
		t.Fatalf("expected /explore HTTP 200, got %d: %s", exploreResponse.Code, exploreResponse.Body.String())
	}

	var explored ard.ExploreResponse
	if err := json.Unmarshal(exploreResponse.Body.Bytes(), &explored); err != nil {
		t.Fatalf("decode explore response: %v", err)
	}
	if explored.ResultType != "facets" {
		t.Fatalf("unexpected result type: %s", explored.ResultType)
	}
	if len(explored.Facets["type"].Buckets) == 0 {
		t.Fatalf("expected type facet buckets: %#v", explored)
	}
}
