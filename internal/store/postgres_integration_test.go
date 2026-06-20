package store

import (
	"context"
	"os"
	"testing"

	"github.com/ifuryst/ard/internal/ard"
)

func TestPostgresImportAndSearch(t *testing.T) {
	databaseURL := os.Getenv("ARD_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set ARD_TEST_DATABASE_URL to run Postgres integration tests")
	}
	ctx := context.Background()
	registryStore, err := Open(databaseURL)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer registryStore.Close()
	if err := registryStore.AutoMigrate(); err != nil {
		t.Fatalf("migrate store: %v", err)
	}
	if err := registryStore.db.Exec(
		"DELETE FROM catalog_entry_records WHERE identifier IN ?",
		[]string{"urn:air:acme.com:server:weather", "urn:air:acme.com:agent:assistant"},
	).Error; err != nil {
		t.Fatalf("clean entries: %v", err)
	}

	catalog := ard.Catalog{
		SpecVersion: "1.0",
		Entries: []ard.CatalogEntry{
			{
				Identifier:            "urn:air:acme.com:server:weather",
				DisplayName:           "Weather Data Node",
				Type:                  ard.TypeMCPServerCard,
				URL:                   "https://api.acme.com/mcp/weather.json",
				Description:           "Enterprise weather MCP server for live telemetry.",
				Capabilities:          []string{"WeatherTool", "ForecastTool"},
				RepresentativeQueries: []string{"what is the current wind speed in Chicago", "get the 5-day forecast for Seattle"},
			},
			{
				Identifier:            "urn:air:acme.com:agent:assistant",
				DisplayName:           "Corporate Assistant",
				Type:                  ard.TypeA2AAgentCard,
				URL:                   "https://api.acme.com/agents/assistant.json",
				Description:           "General-purpose corporate A2A assistant.",
				RepresentativeQueries: []string{"draft an email", "summarize unread messages"},
			},
		},
	}
	if err := registryStore.UpsertCatalog(ctx, catalog, "integration-test"); err != nil {
		t.Fatalf("upsert catalog: %v", err)
	}

	results, err := registryStore.Search(ctx, ard.SearchRequest{
		Query: ard.SearchQuery{
			Text: "weather forecast",
			Filter: ard.Filter{
				"type": []string{ard.TypeMCPServerCard},
			},
		},
		PageSize: 10,
	}, "")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one MCP result, got %d", len(results))
	}
	if results[0].Identifier != "urn:air:acme.com:server:weather" {
		t.Fatalf("unexpected result: %#v", results[0])
	}
	if results[0].Score <= 0 {
		t.Fatalf("expected positive relevance score, got %d", results[0].Score)
	}
}
