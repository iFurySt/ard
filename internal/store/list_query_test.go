package store

import (
	"strings"
	"testing"

	"github.com/ifuryst/ard/internal/ard"
)

func TestParseListFilterExpression(t *testing.T) {
	filter, err := ParseListFilterExpression("type = 'application/mcp-server-card+json', 'application/a2a-agent-card+json' AND displayName = 'Weather' AND publisherId = 'example.com' AND tags = 'weather' AND capabilities = 'ForecastTool' AND metadata.adapter = 'mcp' AND createdAfter > '2026-01-01'")
	if err != nil {
		t.Fatalf("parse list filter: %v", err)
	}
	if len(filter.Types) != 2 || filter.Types[0] != ard.TypeMCPServerCard || filter.Types[1] != ard.TypeA2AAgentCard {
		t.Fatalf("unexpected type filters: %#v", filter.Types)
	}
	if len(filter.DisplayName) != 1 || filter.DisplayName[0] != "Weather" {
		t.Fatalf("unexpected displayName filters: %#v", filter.DisplayName)
	}
	if len(filter.PublisherIDs) != 1 || filter.PublisherIDs[0] != "example.com" {
		t.Fatalf("unexpected publisher filters: %#v", filter.PublisherIDs)
	}
	if len(filter.Tags) != 1 || filter.Tags[0] != "weather" {
		t.Fatalf("unexpected tag filters: %#v", filter.Tags)
	}
	if len(filter.Capabilities) != 1 || filter.Capabilities[0] != "ForecastTool" {
		t.Fatalf("unexpected capability filters: %#v", filter.Capabilities)
	}
	if got := filter.Metadata["adapter"]; len(got) != 1 || got[0] != "mcp" {
		t.Fatalf("unexpected metadata filters: %#v", filter.Metadata)
	}
	if filter.CreatedAfter == nil {
		t.Fatal("expected createdAfter filter to parse")
	}
}

func TestParseListFilterExpressionRejectsUnsupportedFields(t *testing.T) {
	_, err := ParseListFilterExpression("score = '100'")
	if err == nil {
		t.Fatal("expected unsupported filter field to be rejected")
	}
	if !strings.Contains(err.Error(), `unsupported filter field "score"`) {
		t.Fatalf("unexpected filter error: %v", err)
	}

	_, err = ParseListFilterExpression("updatedAfter = '2026-01-01T00:00:00Z'")
	if err == nil {
		t.Fatal("expected unsupported timestamp operator to be rejected")
	}
	if !strings.Contains(err.Error(), `filter field "updatedAfter" only supports >`) {
		t.Fatalf("unexpected timestamp operator error: %v", err)
	}

	_, err = ParseListFilterExpression("metadata. = 'skill'")
	if err == nil {
		t.Fatal("expected empty metadata key to be rejected")
	}
	if !strings.Contains(err.Error(), "metadata filter key must not be empty") {
		t.Fatalf("unexpected metadata key error: %v", err)
	}
}

func TestParseListOrderBy(t *testing.T) {
	order, err := ParseListOrderBy("updated_at DESC")
	if err != nil {
		t.Fatalf("parse orderBy: %v", err)
	}
	if order.Field != "updatedAt" || order.Direction != "DESC" {
		t.Fatalf("unexpected orderBy: %#v", order)
	}

	if _, err := ParseListOrderBy("score DESC"); err == nil {
		t.Fatal("expected unsupported orderBy field to be rejected")
	}
	if _, err := ParseListOrderBy("displayName SIDEWAYS"); err == nil {
		t.Fatal("expected unsupported orderBy direction to be rejected")
	}
}
