package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
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
	var upstreamMu sync.Mutex
	upstreamRequests := []ard.SearchRequest{}
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/search" {
			http.Error(response, "unexpected upstream path", http.StatusNotFound)
			return
		}
		var upstreamRequest ard.SearchRequest
		if err := json.NewDecoder(request.Body).Decode(&upstreamRequest); err != nil {
			http.Error(response, err.Error(), http.StatusBadRequest)
			return
		}
		upstreamMu.Lock()
		upstreamRequests = append(upstreamRequests, upstreamRequest)
		upstreamMu.Unlock()
		if upstreamRequest.Federation != "none" {
			http.Error(response, "upstream federation must be none", http.StatusBadRequest)
			return
		}
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(ard.SearchResponse{
			Results: []ard.SearchResult{
				{
					CatalogEntry: ard.CatalogEntry{
						Identifier:  "urn:air:upstream.example.com:server:remote-weather",
						DisplayName: "Remote Weather MCP",
						Type:        ard.TypeMCPServerCard,
						URL:         "https://upstream.example.com/mcp/weather.json",
					},
					Score:  72,
					Source: "upstream-test",
				},
			},
		})
	}))
	t.Cleanup(upstreamServer.Close)
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
			{
				Identifier:            "urn:air:example.com:server:weather-archive",
				DisplayName:           "Archive Weather MCP",
				Type:                  ard.TypeMCPServerCard,
				URL:                   "https://example.com/mcp/weather-archive.json",
				Description:           "Paginationneedle historical weather forecast MCP server.",
				RepresentativeQueries: []string{"paginationneedle weather archive", "historical forecast"},
			},
			{
				Identifier:            "urn:air:example.com:server:weather-current",
				DisplayName:           "Current Weather MCP",
				Type:                  ard.TypeMCPServerCard,
				URL:                   "https://example.com/mcp/weather-current.json",
				Description:           "Paginationneedle current weather forecast MCP server.",
				RepresentativeQueries: []string{"paginationneedle current weather", "current forecast"},
			},
			{
				Identifier:  "urn:air:upstream.example.com:registry:public",
				DisplayName: "Public Upstream Registry",
				Type:        ard.TypeAIRegistry,
				URL:         upstreamServer.URL,
				Description: "Upstream ARD registry for referral-mode federation.",
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

	if len(parsed.Referrals) != 0 {
		t.Fatalf("expected no referrals without federation mode, got %#v", parsed.Referrals)
	}

	firstPageBody, _ := json.Marshal(ard.SearchRequest{
		Query: ard.SearchQuery{
			Text: "paginationneedle",
			Filter: ard.Filter{
				"type": []string{ard.TypeMCPServerCard},
			},
		},
		Federation: "none",
		PageSize:   1,
	})
	firstPageRequest := httptest.NewRequest(http.MethodPost, "/search", bytes.NewReader(firstPageBody))
	firstPageRequest.Header.Set("Content-Type", "application/json")
	firstPageResponse := httptest.NewRecorder()
	router.ServeHTTP(firstPageResponse, firstPageRequest)
	if firstPageResponse.Code != http.StatusOK {
		t.Fatalf("expected first page HTTP 200, got %d: %s", firstPageResponse.Code, firstPageResponse.Body.String())
	}
	var firstPage ard.SearchResponse
	if err := json.Unmarshal(firstPageResponse.Body.Bytes(), &firstPage); err != nil {
		t.Fatalf("decode first page response: %v", err)
	}
	if len(firstPage.Results) != 1 || firstPage.PageToken == "" {
		t.Fatalf("expected one result and next page token, got %#v", firstPage)
	}
	secondPageRequestBody, _ := json.Marshal(ard.SearchRequest{
		Query: ard.SearchQuery{
			Text: "paginationneedle",
			Filter: ard.Filter{
				"type": []string{ard.TypeMCPServerCard},
			},
		},
		Federation: "none",
		PageSize:   1,
		PageToken:  firstPage.PageToken,
	})
	secondPageRequest := httptest.NewRequest(http.MethodPost, "/search", bytes.NewReader(secondPageRequestBody))
	secondPageRequest.Header.Set("Content-Type", "application/json")
	secondPageResponse := httptest.NewRecorder()
	router.ServeHTTP(secondPageResponse, secondPageRequest)
	if secondPageResponse.Code != http.StatusOK {
		t.Fatalf("expected second page HTTP 200, got %d: %s", secondPageResponse.Code, secondPageResponse.Body.String())
	}
	var secondPage ard.SearchResponse
	if err := json.Unmarshal(secondPageResponse.Body.Bytes(), &secondPage); err != nil {
		t.Fatalf("decode second page response: %v", err)
	}
	if len(secondPage.Results) != 1 {
		t.Fatalf("expected one second-page result, got %#v", secondPage)
	}
	if firstPage.Results[0].Identifier == secondPage.Results[0].Identifier {
		t.Fatalf("expected second page to advance, got same identifier %s", secondPage.Results[0].Identifier)
	}
	invalidPageBody, _ := json.Marshal(ard.SearchRequest{
		Query:     ard.SearchQuery{Text: "paginationneedle"},
		PageToken: "not-a-valid-page-token",
	})
	invalidPageRequest := httptest.NewRequest(http.MethodPost, "/search", bytes.NewReader(invalidPageBody))
	invalidPageRequest.Header.Set("Content-Type", "application/json")
	invalidPageResponse := httptest.NewRecorder()
	router.ServeHTTP(invalidPageResponse, invalidPageRequest)
	if invalidPageResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid page token HTTP 400, got %d: %s", invalidPageResponse.Code, invalidPageResponse.Body.String())
	}

	federatedBody, _ := json.Marshal(ard.SearchRequest{
		Query: ard.SearchQuery{
			Text: "weather",
		},
		Federation: "referrals",
		PageSize:   5,
	})
	federatedRequest := httptest.NewRequest(http.MethodPost, "/search", bytes.NewReader(federatedBody))
	federatedRequest.Header.Set("Content-Type", "application/json")
	federatedResponse := httptest.NewRecorder()
	router.ServeHTTP(federatedResponse, federatedRequest)
	if federatedResponse.Code != http.StatusOK {
		t.Fatalf("expected federated HTTP 200, got %d: %s", federatedResponse.Code, federatedResponse.Body.String())
	}
	var federated ard.SearchResponse
	if err := json.Unmarshal(federatedResponse.Body.Bytes(), &federated); err != nil {
		t.Fatalf("decode federated response: %v", err)
	}
	if len(federated.Referrals) != 1 {
		t.Fatalf("expected one referral, got %#v", federated.Referrals)
	}
	if federated.Referrals[0].Type != ard.TypeAIRegistry {
		t.Fatalf("unexpected referral type: %s", federated.Referrals[0].Type)
	}

	autoBody, _ := json.Marshal(ard.SearchRequest{
		Query: ard.SearchQuery{
			Text: "weather",
		},
		Federation: "auto",
		PageSize:   5,
	})
	autoRequest := httptest.NewRequest(http.MethodPost, "/search", bytes.NewReader(autoBody))
	autoRequest.Header.Set("Content-Type", "application/json")
	autoResponse := httptest.NewRecorder()
	router.ServeHTTP(autoResponse, autoRequest)
	if autoResponse.Code != http.StatusOK {
		t.Fatalf("expected auto federation HTTP 200, got %d: %s", autoResponse.Code, autoResponse.Body.String())
	}
	var auto ard.SearchResponse
	if err := json.Unmarshal(autoResponse.Body.Bytes(), &auto); err != nil {
		t.Fatalf("decode auto federation response: %v", err)
	}
	foundRemote := false
	for _, result := range auto.Results {
		if result.Identifier == "urn:air:upstream.example.com:server:remote-weather" {
			foundRemote = true
		}
	}
	if !foundRemote {
		t.Fatalf("expected auto federation remote result, got %#v", auto.Results)
	}
	upstreamMu.Lock()
	defer upstreamMu.Unlock()
	if len(upstreamRequests) == 0 {
		t.Fatal("expected at least one upstream federation request")
	}
	for _, request := range upstreamRequests {
		if request.Federation != "none" {
			t.Fatalf("expected all upstream requests to use federation=none, got %#v", upstreamRequests)
		}
		if request.PageToken != "" {
			t.Fatalf("expected upstream requests to omit local page token, got %#v", upstreamRequests)
		}
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
	pagedListRequest := httptest.NewRequest(http.MethodGet, "/agents?pageSize=1", nil)
	pagedListResponse := httptest.NewRecorder()
	router.ServeHTTP(pagedListResponse, pagedListRequest)
	if pagedListResponse.Code != http.StatusOK {
		t.Fatalf("expected paged /agents HTTP 200, got %d: %s", pagedListResponse.Code, pagedListResponse.Body.String())
	}
	var firstListPage ard.ListResponse
	if err := json.Unmarshal(pagedListResponse.Body.Bytes(), &firstListPage); err != nil {
		t.Fatalf("decode first list page: %v", err)
	}
	if len(firstListPage.Items) != 1 || firstListPage.PageToken == "" {
		t.Fatalf("expected one list item and next token, got %#v", firstListPage)
	}
	nextListRequest := httptest.NewRequest(http.MethodGet, "/agents?pageSize=1&pageToken="+firstListPage.PageToken, nil)
	nextListResponse := httptest.NewRecorder()
	router.ServeHTTP(nextListResponse, nextListRequest)
	if nextListResponse.Code != http.StatusOK {
		t.Fatalf("expected second /agents page HTTP 200, got %d: %s", nextListResponse.Code, nextListResponse.Body.String())
	}
	var secondListPage ard.ListResponse
	if err := json.Unmarshal(nextListResponse.Body.Bytes(), &secondListPage); err != nil {
		t.Fatalf("decode second list page: %v", err)
	}
	if len(secondListPage.Items) != 1 {
		t.Fatalf("expected one second-page list item, got %#v", secondListPage)
	}
	if firstListPage.Items[0].Identifier == secondListPage.Items[0].Identifier {
		t.Fatalf("expected /agents page token to advance, got same identifier %s", secondListPage.Items[0].Identifier)
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

func TestRouterAdminRBACWithPostgres(t *testing.T) {
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
	identifiers := []string{
		"urn:air:rbac.example.com:server:publisher-weather",
		"urn:air:rbac.example.com:server:review-weather",
	}
	for _, identifier := range identifiers {
		if _, err := registryStore.DeleteEntry(ctx, identifier); err != nil {
			t.Fatalf("clean %s: %v", identifier, err)
		}
	}

	router := NewRouterWithOptions(registryStore, Options{
		AdminTokens: []AdminToken{
			{Name: "reader", Token: "reader-token", Role: "reader"},
			{Name: "publisher", Token: "publisher-token", Role: "publisher"},
			{Name: "reviewer", Token: "reviewer-token", Role: "reviewer"},
			{Name: "operator", Token: "operator-token", Role: "operator"},
		},
	})

	entry := ard.CatalogEntry{
		Identifier:            "urn:air:rbac.example.com:server:publisher-weather",
		DisplayName:           "RBAC Publisher Weather",
		Type:                  ard.TypeMCPServerCard,
		URL:                   "https://rbac.example.com/mcp/publisher-weather.json",
		Description:           "Weather MCP server added through a publisher token.",
		RepresentativeQueries: []string{"weather through publisher token", "publisher token forecast"},
	}
	entryBody, _ := json.Marshal(entry)
	readerCreateRequest := httptest.NewRequest(http.MethodPost, "/admin/entries", bytes.NewReader(entryBody))
	readerCreateRequest.Header.Set("Authorization", "Bearer reader-token")
	readerCreateRequest.Header.Set("Content-Type", "application/json")
	readerCreateResponse := httptest.NewRecorder()
	router.ServeHTTP(readerCreateResponse, readerCreateRequest)
	if readerCreateResponse.Code != http.StatusForbidden {
		t.Fatalf("expected reader create HTTP 403, got %d: %s", readerCreateResponse.Code, readerCreateResponse.Body.String())
	}

	publisherCreateRequest := httptest.NewRequest(http.MethodPost, "/admin/entries", bytes.NewReader(entryBody))
	publisherCreateRequest.Header.Set("Authorization", "Bearer publisher-token")
	publisherCreateRequest.Header.Set("Content-Type", "application/json")
	publisherCreateResponse := httptest.NewRecorder()
	router.ServeHTTP(publisherCreateResponse, publisherCreateRequest)
	if publisherCreateResponse.Code != http.StatusCreated {
		t.Fatalf("expected publisher create HTTP 201, got %d: %s", publisherCreateResponse.Code, publisherCreateResponse.Body.String())
	}

	readerListRequest := httptest.NewRequest(http.MethodGet, "/admin/entries?kind=mcp", nil)
	readerListRequest.Header.Set("Authorization", "Bearer reader-token")
	readerListResponse := httptest.NewRecorder()
	router.ServeHTTP(readerListResponse, readerListRequest)
	if readerListResponse.Code != http.StatusOK {
		t.Fatalf("expected reader list HTTP 200, got %d: %s", readerListResponse.Code, readerListResponse.Body.String())
	}

	pendingCatalog := ard.Catalog{
		SpecVersion: "1.0",
		Entries: []ard.CatalogEntry{
			{
				Identifier:            "urn:air:rbac.example.com:server:review-weather",
				DisplayName:           "RBAC Review Weather",
				Type:                  ard.TypeMCPServerCard,
				URL:                   "https://rbac.example.com/mcp/review-weather.json",
				Description:           "Weather MCP server awaiting reviewer approval.",
				RepresentativeQueries: []string{"weather review token", "review token forecast"},
			},
		},
	}
	if err := registryStore.UpsertCatalogWithStatuses(ctx, pendingCatalog, "rbac-test", map[string]string{
		"urn:air:rbac.example.com:server:review-weather": store.LifecycleStatusPending,
	}); err != nil {
		t.Fatalf("upsert pending review entry: %v", err)
	}

	operatorApproveRequest := httptest.NewRequest(http.MethodPost, "/admin/reviews/urn:air:rbac.example.com:server:review-weather/approve", nil)
	operatorApproveRequest.Header.Set("Authorization", "Bearer operator-token")
	operatorApproveResponse := httptest.NewRecorder()
	router.ServeHTTP(operatorApproveResponse, operatorApproveRequest)
	if operatorApproveResponse.Code != http.StatusForbidden {
		t.Fatalf("expected operator approve HTTP 403, got %d: %s", operatorApproveResponse.Code, operatorApproveResponse.Body.String())
	}

	reviewerApproveRequest := httptest.NewRequest(http.MethodPost, "/admin/reviews/urn:air:rbac.example.com:server:review-weather/approve", nil)
	reviewerApproveRequest.Header.Set("Authorization", "Bearer reviewer-token")
	reviewerApproveResponse := httptest.NewRecorder()
	router.ServeHTTP(reviewerApproveResponse, reviewerApproveRequest)
	if reviewerApproveResponse.Code != http.StatusOK {
		t.Fatalf("expected reviewer approve HTTP 200, got %d: %s", reviewerApproveResponse.Code, reviewerApproveResponse.Body.String())
	}

	statusBody, _ := json.Marshal(map[string]string{"status": store.LifecycleStatusDisabled})
	reviewerStatusRequest := httptest.NewRequest(http.MethodPatch, "/admin/entries/urn:air:rbac.example.com:server:publisher-weather/status", bytes.NewReader(statusBody))
	reviewerStatusRequest.Header.Set("Authorization", "Bearer reviewer-token")
	reviewerStatusRequest.Header.Set("Content-Type", "application/json")
	reviewerStatusResponse := httptest.NewRecorder()
	router.ServeHTTP(reviewerStatusResponse, reviewerStatusRequest)
	if reviewerStatusResponse.Code != http.StatusForbidden {
		t.Fatalf("expected reviewer status HTTP 403, got %d: %s", reviewerStatusResponse.Code, reviewerStatusResponse.Body.String())
	}

	operatorStatusRequest := httptest.NewRequest(http.MethodPatch, "/admin/entries/urn:air:rbac.example.com:server:publisher-weather/status", bytes.NewReader(statusBody))
	operatorStatusRequest.Header.Set("Authorization", "Bearer operator-token")
	operatorStatusRequest.Header.Set("Content-Type", "application/json")
	operatorStatusResponse := httptest.NewRecorder()
	router.ServeHTTP(operatorStatusResponse, operatorStatusRequest)
	if operatorStatusResponse.Code != http.StatusOK {
		t.Fatalf("expected operator status HTTP 200, got %d: %s", operatorStatusResponse.Code, operatorStatusResponse.Body.String())
	}

	publisherDeleteRequest := httptest.NewRequest(http.MethodDelete, "/admin/entries/urn:air:rbac.example.com:server:publisher-weather", nil)
	publisherDeleteRequest.Header.Set("Authorization", "Bearer publisher-token")
	publisherDeleteResponse := httptest.NewRecorder()
	router.ServeHTTP(publisherDeleteResponse, publisherDeleteRequest)
	if publisherDeleteResponse.Code != http.StatusForbidden {
		t.Fatalf("expected publisher delete HTTP 403, got %d: %s", publisherDeleteResponse.Code, publisherDeleteResponse.Body.String())
	}

	operatorDeleteRequest := httptest.NewRequest(http.MethodDelete, "/admin/entries/urn:air:rbac.example.com:server:publisher-weather", nil)
	operatorDeleteRequest.Header.Set("Authorization", "Bearer operator-token")
	operatorDeleteResponse := httptest.NewRecorder()
	router.ServeHTTP(operatorDeleteResponse, operatorDeleteRequest)
	if operatorDeleteResponse.Code != http.StatusNoContent {
		t.Fatalf("expected operator delete HTTP 204, got %d: %s", operatorDeleteResponse.Code, operatorDeleteResponse.Body.String())
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
	newSearchRequest := func() *http.Request {
		request := httptest.NewRequest(http.MethodPost, "/search", bytes.NewReader(searchBody))
		request.Header.Set("Content-Type", "application/json")
		return request
	}
	searchResponse := httptest.NewRecorder()
	router.ServeHTTP(searchResponse, newSearchRequest())
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

	reviewListRequest := httptest.NewRequest(http.MethodGet, "/admin/reviews", nil)
	reviewListRequest.Header.Set("Authorization", "Bearer test-token")
	reviewListResponse := httptest.NewRecorder()
	router.ServeHTTP(reviewListResponse, reviewListRequest)
	if reviewListResponse.Code != http.StatusOK {
		t.Fatalf("expected review list HTTP 200, got %d: %s", reviewListResponse.Code, reviewListResponse.Body.String())
	}
	var reviews ard.ListResponse
	if err := json.Unmarshal(reviewListResponse.Body.Bytes(), &reviews); err != nil {
		t.Fatalf("decode review list: %v", err)
	}
	if len(reviews.Items) == 0 {
		t.Fatalf("expected pending review entries, got %#v", reviews)
	}

	approveRequest := httptest.NewRequest(http.MethodPost, "/admin/reviews/urn:air:review.example.com:server:policy-weather/approve", nil)
	approveRequest.Header.Set("Authorization", "Bearer test-token")
	approveResponse := httptest.NewRecorder()
	router.ServeHTTP(approveResponse, approveRequest)
	if approveResponse.Code != http.StatusOK {
		t.Fatalf("expected approve HTTP 200, got %d: %s", approveResponse.Code, approveResponse.Body.String())
	}
	approveAgainRequest := httptest.NewRequest(http.MethodPost, "/admin/reviews/urn:air:review.example.com:server:policy-weather/approve", nil)
	approveAgainRequest.Header.Set("Authorization", "Bearer test-token")
	approveAgainResponse := httptest.NewRecorder()
	router.ServeHTTP(approveAgainResponse, approveAgainRequest)
	if approveAgainResponse.Code != http.StatusConflict {
		t.Fatalf("expected approve active entry HTTP 409, got %d: %s", approveAgainResponse.Code, approveAgainResponse.Body.String())
	}

	searchResponse = httptest.NewRecorder()
	router.ServeHTTP(searchResponse, newSearchRequest())
	if searchResponse.Code != http.StatusOK {
		t.Fatalf("expected search after approve HTTP 200, got %d: %s", searchResponse.Code, searchResponse.Body.String())
	}
	if err := json.Unmarshal(searchResponse.Body.Bytes(), &search); err != nil {
		t.Fatalf("decode approved search: %v", err)
	}
	if len(search.Results) == 0 {
		t.Fatalf("expected approved entry to become searchable")
	}

	updatedEntry := pendingEntry
	updatedEntry.Description = "Quarantine policy update requires review."
	updatedBody, _ := json.Marshal(updatedEntry)
	updateRequest := httptest.NewRequest(http.MethodPost, "/admin/entries", bytes.NewReader(updatedBody))
	updateRequest.Header.Set("Authorization", "Bearer test-token")
	updateRequest.Header.Set("Content-Type", "application/json")
	updateResponse := httptest.NewRecorder()
	router.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusCreated {
		t.Fatalf("expected policy pending update HTTP 201, got %d: %s", updateResponse.Code, updateResponse.Body.String())
	}
	updatedStored, found, err := registryStore.GetEntry(ctx, pendingEntry.Identifier, true)
	if err != nil {
		t.Fatalf("get policy pending update: %v", err)
	}
	if !found {
		t.Fatal("expected policy pending update to exist")
	}
	if updatedStored.Description != "Quarantine policy update requires review." {
		t.Fatalf("expected update content to persist, got %#v", updatedStored)
	}
	if updatedStored.Metadata["ard.status"] != store.LifecycleStatusPending {
		t.Fatalf("expected policy update to become pending, got %#v", updatedStored.Metadata)
	}
	searchResponse = httptest.NewRecorder()
	router.ServeHTTP(searchResponse, newSearchRequest())
	if searchResponse.Code != http.StatusOK {
		t.Fatalf("expected search after pending update HTTP 200, got %d: %s", searchResponse.Code, searchResponse.Body.String())
	}
	if err := json.Unmarshal(searchResponse.Body.Bytes(), &search); err != nil {
		t.Fatalf("decode pending update search: %v", err)
	}
	if len(search.Results) != 0 {
		t.Fatalf("expected pending update to be hidden from search, got %#v", search.Results)
	}

	statusBody, _ := json.Marshal(map[string]string{"status": store.LifecycleStatusPending})
	statusRequest := httptest.NewRequest(http.MethodPatch, "/admin/entries/urn:air:review.example.com:server:policy-weather/status", bytes.NewReader(statusBody))
	statusRequest.Header.Set("Authorization", "Bearer test-token")
	statusRequest.Header.Set("Content-Type", "application/json")
	statusResponse := httptest.NewRecorder()
	router.ServeHTTP(statusResponse, statusRequest)
	if statusResponse.Code != http.StatusOK {
		t.Fatalf("expected reset pending HTTP 200, got %d: %s", statusResponse.Code, statusResponse.Body.String())
	}

	rejectRequest := httptest.NewRequest(http.MethodPost, "/admin/reviews/urn:air:review.example.com:server:policy-weather/reject", nil)
	rejectRequest.Header.Set("Authorization", "Bearer test-token")
	rejectResponse := httptest.NewRecorder()
	router.ServeHTTP(rejectResponse, rejectRequest)
	if rejectResponse.Code != http.StatusOK {
		t.Fatalf("expected reject HTTP 200, got %d: %s", rejectResponse.Code, rejectResponse.Body.String())
	}
	rejectAgainRequest := httptest.NewRequest(http.MethodPost, "/admin/reviews/urn:air:review.example.com:server:policy-weather/reject", nil)
	rejectAgainRequest.Header.Set("Authorization", "Bearer test-token")
	rejectAgainResponse := httptest.NewRecorder()
	router.ServeHTTP(rejectAgainResponse, rejectAgainRequest)
	if rejectAgainResponse.Code != http.StatusConflict {
		t.Fatalf("expected reject disabled entry HTTP 409, got %d: %s", rejectAgainResponse.Code, rejectAgainResponse.Body.String())
	}

	searchResponse = httptest.NewRecorder()
	router.ServeHTTP(searchResponse, newSearchRequest())
	if searchResponse.Code != http.StatusOK {
		t.Fatalf("expected search after reject HTTP 200, got %d: %s", searchResponse.Code, searchResponse.Body.String())
	}
	if err := json.Unmarshal(searchResponse.Body.Bytes(), &search); err != nil {
		t.Fatalf("decode rejected search: %v", err)
	}
	if len(search.Results) != 0 {
		t.Fatalf("expected rejected entry to be hidden from search, got %#v", search.Results)
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
