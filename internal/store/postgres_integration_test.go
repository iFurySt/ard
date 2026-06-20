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
		[]string{"urn:air:acme.com:server:weather", "urn:air:acme.com:agent:assistant", "urn:air:review.acme.com:server:pending-weather"},
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

	pendingCatalog := ard.Catalog{
		SpecVersion: "1.0",
		Entries: []ard.CatalogEntry{
			{
				Identifier:            "urn:air:review.acme.com:server:pending-weather",
				DisplayName:           "Quarantine Review MCP",
				Type:                  ard.TypeMCPServerCard,
				URL:                   "https://review.acme.com/mcp/pending-weather.json",
				Description:           "Quarantine MCP server awaiting policy review.",
				RepresentativeQueries: []string{"quarantine lookup", "quarantine review"},
			},
		},
	}
	if err := registryStore.UpsertCatalogWithStatuses(ctx, pendingCatalog, "integration-test", map[string]string{
		"urn:air:review.acme.com:server:pending-weather": LifecycleStatusPending,
	}); err != nil {
		t.Fatalf("upsert pending catalog: %v", err)
	}

	pendingResults, err := registryStore.Search(ctx, ard.SearchRequest{
		Query: ard.SearchQuery{
			Text: "quarantine",
		},
		PageSize: 10,
	}, "integration-test")
	if err != nil {
		t.Fatalf("search pending entry: %v", err)
	}
	if len(pendingResults) != 0 {
		t.Fatalf("expected pending entry to be hidden from search, got %d", len(pendingResults))
	}

	results, err := registryStore.Search(ctx, ard.SearchRequest{
		Query: ard.SearchQuery{
			Text: "weather forecast",
			Filter: ard.Filter{
				"type": []string{ard.TypeMCPServerCard},
			},
		},
		PageSize: 10,
	}, "integration-test")
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

	updated, err := registryStore.SetEntryStatus(ctx, "urn:air:acme.com:server:weather", LifecycleStatusDisabled)
	if err != nil {
		t.Fatalf("disable entry: %v", err)
	}
	if !updated {
		t.Fatal("expected weather entry to be disabled")
	}

	results, err = registryStore.Search(ctx, ard.SearchRequest{
		Query: ard.SearchQuery{
			Text: "weather forecast",
			Filter: ard.Filter{
				"type": []string{ard.TypeMCPServerCard},
			},
		},
		PageSize: 10,
	}, "integration-test")
	if err != nil {
		t.Fatalf("search after disable: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected disabled entry to be hidden from search, got %d", len(results))
	}

	disabled, total, err := registryStore.ListEntries(ctx, ListOptions{
		Limit:                    10,
		Status:                   LifecycleStatusDisabled,
		IncludeLifecycleMetadata: true,
	})
	if err != nil {
		t.Fatalf("list disabled entries: %v", err)
	}
	if total < 1 || len(disabled) < 1 {
		t.Fatalf("expected disabled entries, got total=%d len=%d", total, len(disabled))
	}
	foundDisabled := false
	for _, entry := range disabled {
		if entry.Identifier == "urn:air:acme.com:server:weather" {
			foundDisabled = true
			if got := entry.Metadata["ard.status"]; got != LifecycleStatusDisabled {
				t.Fatalf("unexpected lifecycle metadata: %#v", entry.Metadata)
			}
		}
	}
	if !foundDisabled {
		t.Fatalf("expected disabled weather entry in admin list: %#v", disabled)
	}

	listed, total, err := registryStore.ListEntries(ctx, ListOptions{Limit: 10, Type: ard.TypeMCPServerCard})
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}
	for _, entry := range listed {
		if entry.Identifier == "urn:air:acme.com:server:weather" {
			t.Fatalf("disabled entry should not be listed publicly: %#v", entry)
		}
	}

	updated, err = registryStore.SetEntryStatus(ctx, "urn:air:acme.com:server:weather", LifecycleStatusActive)
	if err != nil {
		t.Fatalf("reactivate entry: %v", err)
	}
	if !updated {
		t.Fatal("expected weather entry to be reactivated")
	}

	listed, total, err = registryStore.ListEntries(ctx, ListOptions{Limit: 10, Type: ard.TypeMCPServerCard})
	if err != nil {
		t.Fatalf("list entries after reactivation: %v", err)
	}
	if total < 1 || len(listed) < 1 {
		t.Fatalf("expected listed active MCP entries, got total=%d len=%d", total, len(listed))
	}
	for _, entry := range listed {
		if entry.Type != ard.TypeMCPServerCard {
			t.Fatalf("expected MCP entry after type filter, got %s", entry.Type)
		}
	}

	exported, err := registryStore.ExportCatalog(ctx, &ard.HostInfo{DisplayName: "Integration Registry"})
	if err != nil {
		t.Fatalf("export catalog: %v", err)
	}
	if exported.SpecVersion != "1.0" {
		t.Fatalf("unexpected exported spec version: %s", exported.SpecVersion)
	}
	if exported.Host == nil || exported.Host.DisplayName != "Integration Registry" {
		t.Fatalf("unexpected exported host: %#v", exported.Host)
	}
	if len(exported.Entries) < 2 {
		t.Fatalf("expected exported entries, got %d", len(exported.Entries))
	}
	if err := ard.ValidateCatalog(exported); err != nil {
		t.Fatalf("exported catalog should validate: %v", err)
	}

	removed, err := registryStore.DeleteEntry(ctx, "urn:air:acme.com:agent:assistant")
	if err != nil {
		t.Fatalf("delete entry: %v", err)
	}
	if !removed {
		t.Fatal("expected assistant entry to be removed")
	}
	removed, err = registryStore.DeleteEntry(ctx, "urn:air:acme.com:agent:assistant")
	if err != nil {
		t.Fatalf("delete missing entry: %v", err)
	}
	if removed {
		t.Fatal("expected second delete to report missing entry")
	}
}
