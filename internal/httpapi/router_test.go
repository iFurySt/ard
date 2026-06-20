package httpapi

import (
	"testing"

	"github.com/ifuryst/ard/internal/ard"
)

func TestMergeSearchResultsKeepsLocalFirstAndDeduplicates(t *testing.T) {
	local := []ard.SearchResult{
		{
			CatalogEntry: ard.CatalogEntry{
				Identifier:  "urn:air:example.com:server:weather",
				DisplayName: "Local Weather",
				Type:        ard.TypeMCPServerCard,
			},
			Score: 90,
		},
	}
	upstream := []ard.SearchResult{
		{
			CatalogEntry: ard.CatalogEntry{
				Identifier:  "urn:air:example.com:server:weather",
				DisplayName: "Duplicate Weather",
				Type:        ard.TypeMCPServerCard,
			},
			Score: 99,
		},
		{
			CatalogEntry: ard.CatalogEntry{
				Identifier:  "urn:air:upstream.example.com:server:remote-weather",
				DisplayName: "Remote Weather",
				Type:        ard.TypeMCPServerCard,
			},
			Score: 80,
		},
	}

	results := mergeSearchResults(local, upstream, 2)
	if len(results) != 2 {
		t.Fatalf("expected two merged results, got %#v", results)
	}
	if results[0].DisplayName != "Local Weather" {
		t.Fatalf("expected local result first, got %#v", results)
	}
	if results[1].Identifier != "urn:air:upstream.example.com:server:remote-weather" {
		t.Fatalf("expected deduplicated upstream result, got %#v", results)
	}
}
