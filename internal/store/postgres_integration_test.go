package store

import (
	"context"
	"os"
	"testing"
	"time"

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
		[]string{
			"urn:air:acme.com:server:weather",
			"urn:air:acme.com:agent:assistant",
			"urn:air:acme.com:server:governed-update",
			"urn:air:review.acme.com:server:pending-weather",
		},
	).Error; err != nil {
		t.Fatalf("clean entries: %v", err)
	}

	catalog := ard.Catalog{
		SpecVersion: "1.0",
		Entries: []ard.CatalogEntry{
			{
				Identifier:   "urn:air:acme.com:server:weather",
				DisplayName:  "Weather Data Node",
				Type:         ard.TypeMCPServerCard,
				URL:          "https://api.acme.com/mcp/weather.json",
				Description:  "Enterprise weather MCP server for live telemetry.",
				Tags:         []string{"weather", "internal"},
				Capabilities: []string{"WeatherTool", "ForecastTool"},
				Metadata: map[string]any{
					"adapter": "mcp",
					"tier":    "gold",
				},
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
	pendingEntry, found, err := registryStore.GetEntry(ctx, "urn:air:review.acme.com:server:pending-weather", true)
	if err != nil {
		t.Fatalf("get pending entry: %v", err)
	}
	if !found {
		t.Fatal("expected pending entry to exist")
	}
	if got := pendingEntry.Metadata["ard.status"]; got != LifecycleStatusPending {
		t.Fatalf("unexpected pending lifecycle metadata: %#v", pendingEntry.Metadata)
	}
	_, found, err = registryStore.GetEntry(ctx, "urn:air:review.acme.com:server:missing", true)
	if err != nil {
		t.Fatalf("get missing entry: %v", err)
	}
	if found {
		t.Fatal("expected missing entry to report not found")
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

	filteredListed, _, err := registryStore.ListEntries(ctx, ListOptions{
		Limit: 10,
		Filter: ListFilter{
			Tags:         []string{"weather"},
			Capabilities: []string{"ForecastTool"},
			Metadata: map[string][]string{
				"adapter": {"mcp"},
				"tier":    {"gold"},
			},
		},
	})
	if err != nil {
		t.Fatalf("list entries with tag/capability/metadata filters: %v", err)
	}
	if len(filteredListed) != 1 || filteredListed[0].Identifier != "urn:air:acme.com:server:weather" {
		t.Fatalf("expected filtered weather entry, got %#v", filteredListed)
	}

	richFilter, err := ParseListFilterExpression("type != 'application/a2a-agent-card+json' AND displayName contains 'Weather' AND publisherId contains 'acme' AND tags contains 'weath' AND capabilities != 'BlockedTool' AND metadata.adapter != 'skill' AND metadata.tier contains 'go'")
	if err != nil {
		t.Fatalf("parse rich list filter: %v", err)
	}
	richFilteredListed, _, err := registryStore.ListEntries(ctx, ListOptions{
		Limit:  10,
		Filter: richFilter,
	})
	if err != nil {
		t.Fatalf("list entries with rich filters: %v", err)
	}
	if len(richFilteredListed) != 1 || richFilteredListed[0].Identifier != "urn:air:acme.com:server:weather" {
		t.Fatalf("expected rich filtered weather entry, got %#v", richFilteredListed)
	}

	groupedFilter, err := ParseListFilterExpression("(type = 'application/a2a-agent-card+json' AND publisherId = 'acme.com') OR (displayName contains 'Weather' AND metadata.tier = 'gold')")
	if err != nil {
		t.Fatalf("parse grouped list filter: %v", err)
	}
	groupedFilteredListed, _, err := registryStore.ListEntries(ctx, ListOptions{
		Limit:  10,
		Filter: groupedFilter,
	})
	if err != nil {
		t.Fatalf("list entries with grouped filters: %v", err)
	}
	if !containsCatalogEntry(groupedFilteredListed, "urn:air:acme.com:agent:assistant") || !containsCatalogEntry(groupedFilteredListed, "urn:air:acme.com:server:weather") {
		t.Fatalf("expected grouped filter to return weather plus Acme A2A entry, got %#v", groupedFilteredListed)
	}

	governedCatalog := ard.Catalog{
		SpecVersion: "1.0",
		Entries: []ard.CatalogEntry{
			{
				Identifier:            "urn:air:acme.com:server:governed-update",
				DisplayName:           "Governed Weather MCP",
				Type:                  ard.TypeMCPServerCard,
				URL:                   "https://api.acme.com/mcp/governed-weather.json",
				Description:           "Governed active weather MCP server.",
				RepresentativeQueries: []string{"governed weather active", "governed forecast"},
			},
		},
	}
	if err := registryStore.UpsertCatalog(ctx, governedCatalog, "integration-test"); err != nil {
		t.Fatalf("upsert governed active catalog: %v", err)
	}
	governedResults, err := registryStore.Search(ctx, ard.SearchRequest{
		Query:    ard.SearchQuery{Text: "governed"},
		PageSize: 10,
	}, "integration-test")
	if err != nil {
		t.Fatalf("search governed active entry: %v", err)
	}
	if len(governedResults) != 1 || governedResults[0].Identifier != "urn:air:acme.com:server:governed-update" {
		t.Fatalf("expected governed active entry to be searchable, got %#v", governedResults)
	}
	governedUpdate := governedCatalog
	governedUpdate.Entries[0].Description = "Governed pending update weather MCP server."
	if err := registryStore.UpsertCatalogWithStatuses(ctx, governedUpdate, "integration-test", map[string]string{
		"urn:air:acme.com:server:governed-update": LifecycleStatusPending,
	}); err != nil {
		t.Fatalf("upsert governed pending update: %v", err)
	}
	governedEntry, found, err := registryStore.GetEntry(ctx, "urn:air:acme.com:server:governed-update", true)
	if err != nil {
		t.Fatalf("get governed pending update: %v", err)
	}
	if !found {
		t.Fatal("expected governed entry to exist")
	}
	if governedEntry.Description != "Governed pending update weather MCP server." {
		t.Fatalf("expected governed update content to persist, got %#v", governedEntry)
	}
	if got := governedEntry.Metadata["ard.status"]; got != LifecycleStatusPending {
		t.Fatalf("expected governed update to require review, got %#v", governedEntry.Metadata)
	}
	governedResults, err = registryStore.Search(ctx, ard.SearchRequest{
		Query:    ard.SearchQuery{Text: "governed"},
		PageSize: 10,
	}, "integration-test")
	if err != nil {
		t.Fatalf("search governed pending update: %v", err)
	}
	if len(governedResults) != 0 {
		t.Fatalf("expected governed pending update to be hidden from search, got %#v", governedResults)
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

func containsCatalogEntry(entries []ard.CatalogEntry, identifier string) bool {
	for _, entry := range entries {
		if entry.Identifier == identifier {
			return true
		}
	}
	return false
}

func TestPostgresAuditHashChainVerification(t *testing.T) {
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
	if err := registryStore.db.Exec("DELETE FROM audit_event_records").Error; err != nil {
		t.Fatalf("clean audit events: %v", err)
	}
	defer func() {
		_ = registryStore.db.Exec("DELETE FROM audit_event_records").Error
	}()

	firstTime := time.Date(2026, 6, 21, 0, 40, 0, 0, time.UTC)
	secondTime := firstTime.Add(time.Second)
	if err := registryStore.RecordAuditEvent(ctx, AuditEvent{
		ID:         "00000000-0000-0000-0000-000000000001",
		Action:     "entry.upsert",
		Identifier: "urn:air:audit.example.com:server:one",
		RequestID:  "audit-chain-one",
		Source:     "test",
		CreatedAt:  firstTime,
	}); err != nil {
		t.Fatalf("record first audit event: %v", err)
	}
	if err := registryStore.RecordAuditEvent(ctx, AuditEvent{
		ID:         "00000000-0000-0000-0000-000000000002",
		Action:     "entry.status",
		Identifier: "urn:air:audit.example.com:server:one",
		Status:     LifecycleStatusDisabled,
		RequestID:  "audit-chain-two",
		Source:     "test",
		CreatedAt:  secondTime,
	}); err != nil {
		t.Fatalf("record second audit event: %v", err)
	}

	verification, err := registryStore.VerifyAuditChain(ctx)
	if err != nil {
		t.Fatalf("verify audit chain: %v", err)
	}
	if !verification.Valid || verification.Total != 2 || verification.LastHash == "" {
		t.Fatalf("expected valid audit chain, got %#v", verification)
	}
	events, _, err := registryStore.ListAuditEvents(ctx, 10)
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	if len(events) != 2 || events[0].Hash == "" || events[1].Hash == "" {
		t.Fatalf("expected audit hashes in listed events, got %#v", events)
	}

	if err := registryStore.db.Model(&AuditEventRecord{}).
		Where("id = ?", "00000000-0000-0000-0000-000000000001").
		Update("status", LifecycleStatusPending).Error; err != nil {
		t.Fatalf("tamper audit event: %v", err)
	}
	verification, err = registryStore.VerifyAuditChain(ctx)
	if err != nil {
		t.Fatalf("verify tampered audit chain: %v", err)
	}
	if verification.Valid || verification.FirstInvalidEventID != "00000000-0000-0000-0000-000000000001" {
		t.Fatalf("expected tampered audit chain to fail on first event, got %#v", verification)
	}
	if err := registryStore.BackfillAuditChain(ctx); err != nil {
		t.Fatalf("backfill tampered audit chain: %v", err)
	}
	verification, err = registryStore.VerifyAuditChain(ctx)
	if err != nil {
		t.Fatalf("verify tampered audit chain after backfill: %v", err)
	}
	if verification.Valid {
		t.Fatalf("backfill should not repair a tampered non-empty audit hash, got %#v", verification)
	}
}
