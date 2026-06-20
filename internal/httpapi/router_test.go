package httpapi

import (
	"testing"

	"github.com/ifuryst/ard/internal/ard"
)

func TestMergeSearchResultsRanksByScoreAndDeduplicates(t *testing.T) {
	local := []ard.SearchResult{
		{
			CatalogEntry: ard.CatalogEntry{
				Identifier:  "urn:air:example.com:server:weather",
				DisplayName: "Local Weather",
				Type:        ard.TypeMCPServerCard,
			},
			Score: 90,
		},
		{
			CatalogEntry: ard.CatalogEntry{
				Identifier:  "urn:air:example.com:server:forecast",
				DisplayName: "Local Forecast",
				Type:        ard.TypeMCPServerCard,
			},
			Score: 99,
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
			Score: 95,
		},
	}

	results := mergeSearchResults(local, upstream, 3)
	if len(results) != 3 {
		t.Fatalf("expected three merged results, got %#v", results)
	}
	if results[0].DisplayName != "Local Forecast" {
		t.Fatalf("expected highest-scoring local result first, got %#v", results)
	}
	if results[1].Identifier != "urn:air:upstream.example.com:server:remote-weather" {
		t.Fatalf("expected higher-scoring upstream result second, got %#v", results)
	}
	if results[2].DisplayName != "Local Weather" {
		t.Fatalf("expected local duplicate to win dedupe, got %#v", results)
	}
}

func TestMergeSearchResultsPageBuffersUnreturnedCandidates(t *testing.T) {
	buffered := []autoFederationBufferedResult{
		{
			Result: ard.SearchResult{
				CatalogEntry: ard.CatalogEntry{
					Identifier:  "urn:air:buffer.example.com:server:weather",
					DisplayName: "Buffered Weather",
					Type:        ard.TypeMCPServerCard,
				},
				Score: 80,
			},
		},
	}
	local := []ard.SearchResult{
		{
			CatalogEntry: ard.CatalogEntry{
				Identifier:  "urn:air:local.example.com:server:weather",
				DisplayName: "Local Weather",
				Type:        ard.TypeMCPServerCard,
			},
			Score: 99,
		},
	}
	upstream := []ard.SearchResult{
		{
			CatalogEntry: ard.CatalogEntry{
				Identifier:  "urn:air:remote.example.com:server:weather",
				DisplayName: "Remote Weather",
				Type:        ard.TypeMCPServerCard,
			},
			Score: 90,
		},
	}

	results, nextBuffered := mergeSearchResultsPage(buffered, local, upstream, 1)
	if len(results) != 1 || results[0].DisplayName != "Local Weather" {
		t.Fatalf("expected highest-scoring result only, got %#v", results)
	}
	if len(nextBuffered) != 2 {
		t.Fatalf("expected two buffered candidates, got %#v", nextBuffered)
	}
	if nextBuffered[0].Result.DisplayName != "Remote Weather" || nextBuffered[1].Result.DisplayName != "Buffered Weather" {
		t.Fatalf("expected unreturned candidates to remain score ordered, got %#v", nextBuffered)
	}
}
