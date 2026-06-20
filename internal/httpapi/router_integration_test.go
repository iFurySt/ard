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
	"github.com/ifuryst/ard/internal/policy"
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

func TestRouterAdminAPIWithPostgres(t *testing.T) {
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
	if _, err := registryStore.DeleteEntry(ctx, "urn:air:example.com:server:admin-weather"); err != nil {
		t.Fatalf("clean admin entry: %v", err)
	}

	publicRouter := NewRouter(registryStore)
	publicRequest := httptest.NewRequest(http.MethodGet, "/admin/entries", nil)
	publicResponse := httptest.NewRecorder()
	publicRouter.ServeHTTP(publicResponse, publicRequest)
	if publicResponse.Code != http.StatusNotFound {
		t.Fatalf("expected admin routes to be absent without token, got %d", publicResponse.Code)
	}

	router := NewRouterWithOptions(registryStore, Options{AdminToken: "test-token"})
	unauthorizedRequest := httptest.NewRequest(http.MethodGet, "/admin/entries", nil)
	unauthorizedResponse := httptest.NewRecorder()
	router.ServeHTTP(unauthorizedResponse, unauthorizedRequest)
	if unauthorizedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected HTTP 401 without token, got %d", unauthorizedResponse.Code)
	}

	entry := ard.CatalogEntry{
		Identifier:            "urn:air:example.com:server:admin-weather",
		DisplayName:           "Admin Weather MCP",
		Type:                  ard.TypeMCPServerCard,
		URL:                   "https://example.com/mcp/admin-weather.json",
		Description:           "Weather MCP server added through the admin API.",
		RepresentativeQueries: []string{"what is the weather", "get an admin forecast"},
	}
	entryBody, _ := json.Marshal(entry)
	createRequest := httptest.NewRequest(http.MethodPost, "/admin/entries", bytes.NewReader(entryBody))
	createRequest.Header.Set("Authorization", "Bearer test-token")
	createRequest.Header.Set("Content-Type", "application/json")
	createResponse := httptest.NewRecorder()
	router.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected create HTTP 201, got %d: %s", createResponse.Code, createResponse.Body.String())
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/admin/entries?kind=mcp", nil)
	listRequest.Header.Set("Authorization", "Bearer test-token")
	listResponse := httptest.NewRecorder()
	router.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected list HTTP 200, got %d: %s", listResponse.Code, listResponse.Body.String())
	}
	var list ard.ListResponse
	if err := json.Unmarshal(listResponse.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if list.Total < 1 {
		t.Fatalf("expected admin entry in list, got %#v", list)
	}

	statusBody, _ := json.Marshal(map[string]string{"status": "disabled"})
	statusRequest := httptest.NewRequest(
		http.MethodPatch,
		"/admin/entries/urn:air:example.com:server:admin-weather/status",
		bytes.NewReader(statusBody),
	)
	statusRequest.Header.Set("Authorization", "Bearer test-token")
	statusRequest.Header.Set("Content-Type", "application/json")
	statusRequest.Header.Set("X-Request-ID", "disable-admin-weather")
	statusResponse := httptest.NewRecorder()
	router.ServeHTTP(statusResponse, statusRequest)
	if statusResponse.Code != http.StatusOK {
		t.Fatalf("expected status HTTP 200, got %d: %s", statusResponse.Code, statusResponse.Body.String())
	}

	publicSearchBody, _ := json.Marshal(ard.SearchRequest{
		Query: ard.SearchQuery{
			Text: "admin forecast",
			Filter: ard.Filter{
				"type": []string{ard.TypeMCPServerCard},
			},
		},
		PageSize: 10,
	})
	publicSearchRequest := httptest.NewRequest(http.MethodPost, "/search", bytes.NewReader(publicSearchBody))
	publicSearchRequest.Header.Set("Content-Type", "application/json")
	publicSearchResponse := httptest.NewRecorder()
	router.ServeHTTP(publicSearchResponse, publicSearchRequest)
	if publicSearchResponse.Code != http.StatusOK {
		t.Fatalf("expected public search HTTP 200, got %d: %s", publicSearchResponse.Code, publicSearchResponse.Body.String())
	}
	var publicSearch ard.SearchResponse
	if err := json.Unmarshal(publicSearchResponse.Body.Bytes(), &publicSearch); err != nil {
		t.Fatalf("decode public search: %v", err)
	}
	for _, result := range publicSearch.Results {
		if result.Identifier == "urn:air:example.com:server:admin-weather" {
			t.Fatalf("disabled entry should not appear in public search: %#v", result)
		}
	}

	disabledListRequest := httptest.NewRequest(http.MethodGet, "/admin/entries?status=disabled", nil)
	disabledListRequest.Header.Set("Authorization", "Bearer test-token")
	disabledListResponse := httptest.NewRecorder()
	router.ServeHTTP(disabledListResponse, disabledListRequest)
	if disabledListResponse.Code != http.StatusOK {
		t.Fatalf("expected disabled list HTTP 200, got %d: %s", disabledListResponse.Code, disabledListResponse.Body.String())
	}
	var disabledList ard.ListResponse
	if err := json.Unmarshal(disabledListResponse.Body.Bytes(), &disabledList); err != nil {
		t.Fatalf("decode disabled list: %v", err)
	}
	foundDisabledAdmin := false
	for _, entry := range disabledList.Items {
		if entry.Identifier == "urn:air:example.com:server:admin-weather" {
			foundDisabledAdmin = true
			if entry.Metadata["ard.status"] != "disabled" {
				t.Fatalf("expected disabled lifecycle metadata, got %#v", entry.Metadata)
			}
		}
	}
	if !foundDisabledAdmin {
		t.Fatalf("expected disabled lifecycle metadata, got %#v", disabledList)
	}

	statusBody, _ = json.Marshal(map[string]string{"status": "active"})
	statusRequest = httptest.NewRequest(
		http.MethodPatch,
		"/admin/entries/urn:air:example.com:server:admin-weather/status",
		bytes.NewReader(statusBody),
	)
	statusRequest.Header.Set("Authorization", "Bearer test-token")
	statusRequest.Header.Set("Content-Type", "application/json")
	statusRequest.Header.Set("X-Request-ID", "activate-admin-weather")
	statusResponse = httptest.NewRecorder()
	router.ServeHTTP(statusResponse, statusRequest)
	if statusResponse.Code != http.StatusOK {
		t.Fatalf("expected status reactivate HTTP 200, got %d: %s", statusResponse.Code, statusResponse.Body.String())
	}

	exportRequest := httptest.NewRequest(http.MethodGet, "/admin/catalog", nil)
	exportRequest.Header.Set("Authorization", "Bearer test-token")
	exportResponse := httptest.NewRecorder()
	router.ServeHTTP(exportResponse, exportRequest)
	if exportResponse.Code != http.StatusOK {
		t.Fatalf("expected export HTTP 200, got %d: %s", exportResponse.Code, exportResponse.Body.String())
	}
	var exported ard.Catalog
	if err := json.Unmarshal(exportResponse.Body.Bytes(), &exported); err != nil {
		t.Fatalf("decode export: %v", err)
	}
	if err := ard.ValidateCatalog(exported); err != nil {
		t.Fatalf("exported admin catalog should validate: %v", err)
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/admin/entries/urn:air:example.com:server:admin-weather", nil)
	deleteRequest.Header.Set("Authorization", "Bearer test-token")
	deleteResponse := httptest.NewRecorder()
	router.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("expected delete HTTP 204, got %d: %s", deleteResponse.Code, deleteResponse.Body.String())
	}

	auditRequest := httptest.NewRequest(http.MethodGet, "/admin/audit?pageSize=10", nil)
	auditRequest.Header.Set("Authorization", "Bearer test-token")
	auditResponse := httptest.NewRecorder()
	router.ServeHTTP(auditResponse, auditRequest)
	if auditResponse.Code != http.StatusOK {
		t.Fatalf("expected audit HTTP 200, got %d: %s", auditResponse.Code, auditResponse.Body.String())
	}
	var audit struct {
		Items []store.AuditEvent `json:"items"`
	}
	if err := json.Unmarshal(auditResponse.Body.Bytes(), &audit); err != nil {
		t.Fatalf("decode audit: %v", err)
	}
	seen := map[string]bool{}
	seenRequestID := false
	for _, event := range audit.Items {
		if event.Identifier == "urn:air:example.com:server:admin-weather" {
			seen[event.Action] = true
			if event.Action == "entry.status" && event.Status == "" {
				t.Fatalf("expected status audit event to include status: %#v", event)
			}
			if event.Action == "entry.status" && event.RequestID == "activate-admin-weather" {
				seenRequestID = true
			}
		}
	}
	for _, action := range []string{"entry.upsert", "entry.status", "entry.delete"} {
		if !seen[action] {
			t.Fatalf("expected audit action %s, got %#v", action, audit.Items)
		}
	}
	if !seenRequestID {
		t.Fatalf("expected status audit event to include request id, got %#v", audit.Items)
	}
}

func TestRouterAdminPolicyWithPostgres(t *testing.T) {
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
	if _, err := registryStore.DeleteEntry(ctx, "urn:air:review.example.com:server:policy-weather"); err != nil {
		t.Fatalf("clean pending policy entry: %v", err)
	}
	if _, err := registryStore.DeleteEntry(ctx, "urn:air:blocked.example.com:server:blocked-weather"); err != nil {
		t.Fatalf("clean blocked policy entry: %v", err)
	}

	router := NewRouterWithOptions(registryStore, Options{
		AdminToken: "test-token",
		Policy: &policy.Policy{
			PendingPublishers: []string{"review.example.com"},
			DenyPublishers:    []string{"blocked.example.com"},
		},
	})

	pendingEntry := ard.CatalogEntry{
		Identifier:            "urn:air:review.example.com:server:policy-weather",
		DisplayName:           "Quarantine Policy MCP",
		Type:                  ard.TypeMCPServerCard,
		URL:                   "https://review.example.com/weather.json",
		Description:           "Quarantine policy test resource.",
		RepresentativeQueries: []string{"quarantine policy", "quarantine review"},
	}
	pendingBody, _ := json.Marshal(pendingEntry)
	pendingRequest := httptest.NewRequest(http.MethodPost, "/admin/entries", bytes.NewReader(pendingBody))
	pendingRequest.Header.Set("Authorization", "Bearer test-token")
	pendingRequest.Header.Set("Content-Type", "application/json")
	pendingResponse := httptest.NewRecorder()
	router.ServeHTTP(pendingResponse, pendingRequest)
	if pendingResponse.Code != http.StatusCreated {
		t.Fatalf("expected policy pending create HTTP 201, got %d: %s", pendingResponse.Code, pendingResponse.Body.String())
	}

	searchBody, _ := json.Marshal(ard.SearchRequest{Query: ard.SearchQuery{Text: "quarantine"}, PageSize: 10})
	searchRequest := httptest.NewRequest(http.MethodPost, "/search", bytes.NewReader(searchBody))
	searchRequest.Header.Set("Content-Type", "application/json")
	searchResponse := httptest.NewRecorder()
	router.ServeHTTP(searchResponse, searchRequest)
	if searchResponse.Code != http.StatusOK {
		t.Fatalf("expected search HTTP 200, got %d: %s", searchResponse.Code, searchResponse.Body.String())
	}
	var search ard.SearchResponse
	if err := json.Unmarshal(searchResponse.Body.Bytes(), &search); err != nil {
		t.Fatalf("decode search: %v", err)
	}
	if len(search.Results) != 0 {
		t.Fatalf("expected policy pending entry to be hidden from search, got %#v", search.Results)
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/admin/entries?status=pending", nil)
	listRequest.Header.Set("Authorization", "Bearer test-token")
	listResponse := httptest.NewRecorder()
	router.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected pending list HTTP 200, got %d: %s", listResponse.Code, listResponse.Body.String())
	}
	var list ard.ListResponse
	if err := json.Unmarshal(listResponse.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode pending list: %v", err)
	}
	foundPending := false
	for _, entry := range list.Items {
		if entry.Identifier == pendingEntry.Identifier {
			foundPending = true
			if entry.Metadata["ard.status"] != store.LifecycleStatusPending {
				t.Fatalf("expected pending lifecycle metadata, got %#v", entry.Metadata)
			}
		}
	}
	if !foundPending {
		t.Fatalf("expected pending entry in admin list, got %#v", list)
	}

	blockedEntry := pendingEntry
	blockedEntry.Identifier = "urn:air:blocked.example.com:server:blocked-weather"
	blockedEntry.DisplayName = "Blocked Weather MCP"
	blockedBody, _ := json.Marshal(blockedEntry)
	blockedRequest := httptest.NewRequest(http.MethodPost, "/admin/entries", bytes.NewReader(blockedBody))
	blockedRequest.Header.Set("Authorization", "Bearer test-token")
	blockedRequest.Header.Set("Content-Type", "application/json")
	blockedResponse := httptest.NewRecorder()
	router.ServeHTTP(blockedResponse, blockedRequest)
	if blockedResponse.Code != http.StatusForbidden {
		t.Fatalf("expected policy deny HTTP 403, got %d: %s", blockedResponse.Code, blockedResponse.Body.String())
	}
}
