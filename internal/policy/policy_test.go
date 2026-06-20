package policy

import (
	"errors"
	"testing"

	"github.com/ifuryst/ard/internal/ard"
	"github.com/ifuryst/ard/internal/store"
)

func TestEvaluateCatalogDeniesPublisher(t *testing.T) {
	entry := testEntry("urn:air:blocked.example.com:server:weather", ard.TypeMCPServerCard)
	_, _, err := Policy{DenyPublishers: []string{"blocked.example.com"}}.EvaluateCatalog(ard.Catalog{
		SpecVersion: "1.0",
		Entries:     []ard.CatalogEntry{entry},
	})
	if err == nil {
		t.Fatal("expected policy denial")
	}
	var denied DeniedError
	if !errors.As(err, &denied) {
		t.Fatalf("expected DeniedError, got %T", err)
	}
	if denied.Identifier != entry.Identifier {
		t.Fatalf("unexpected denied identifier: %s", denied.Identifier)
	}
}

func TestEvaluateCatalogMarksPendingByPublisher(t *testing.T) {
	entry := testEntry("urn:air:review.example.com:server:weather", ard.TypeMCPServerCard)
	statuses, evaluations, err := Policy{PendingPublishers: []string{"review.example.com"}}.EvaluateCatalog(ard.Catalog{
		SpecVersion: "1.0",
		Entries:     []ard.CatalogEntry{entry},
	})
	if err != nil {
		t.Fatalf("evaluate catalog: %v", err)
	}
	if got := statuses[entry.Identifier]; got != store.LifecycleStatusPending {
		t.Fatalf("expected pending status, got %s", got)
	}
	if len(evaluations) != 1 || evaluations[0].Reason != "publisher requires review" {
		t.Fatalf("unexpected evaluations: %#v", evaluations)
	}
}

func TestEvaluateCatalogDefaultStatus(t *testing.T) {
	entry := testEntry("urn:air:example.com:server:weather", ard.TypeMCPServerCard)
	statuses, _, err := Policy{DefaultStatus: store.LifecycleStatusPending}.EvaluateCatalog(ard.Catalog{
		SpecVersion: "1.0",
		Entries:     []ard.CatalogEntry{entry},
	})
	if err != nil {
		t.Fatalf("evaluate catalog: %v", err)
	}
	if got := statuses[entry.Identifier]; got != store.LifecycleStatusPending {
		t.Fatalf("expected default pending status, got %s", got)
	}
}

func testEntry(identifier string, mediaType string) ard.CatalogEntry {
	return ard.CatalogEntry{
		Identifier:            identifier,
		DisplayName:           "Weather",
		Type:                  mediaType,
		URL:                   "https://example.com/weather.json",
		RepresentativeQueries: []string{"weather now", "weather tomorrow"},
	}
}
