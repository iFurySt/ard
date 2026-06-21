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

func TestValidateRequiredApprovals(t *testing.T) {
	if got := (Policy{}).NormalizedRequiredApprovals(); got != 1 {
		t.Fatalf("expected empty requiredApprovals to normalize to 1, got %d", got)
	}
	if got := (Policy{RequiredApprovals: 0}).NormalizedRequiredApprovals(); got != 1 {
		t.Fatalf("expected zero requiredApprovals to normalize to 1, got %d", got)
	}
	if got := (Policy{RequiredApprovals: 2}).NormalizedRequiredApprovals(); got != 2 {
		t.Fatalf("expected requiredApprovals 2, got %d", got)
	}
	if err := (Policy{RequiredApprovals: -1}).Validate(); err == nil {
		t.Fatal("expected negative requiredApprovals to fail validation")
	}
}

func TestEvaluateCatalogRequiresTrustManifest(t *testing.T) {
	entry := testEntry("urn:air:example.com:server:weather", ard.TypeMCPServerCard)
	_, _, err := Policy{RequireTrustManifest: true}.EvaluateCatalog(ard.Catalog{
		SpecVersion: "1.0",
		Entries:     []ard.CatalogEntry{entry},
	})
	if err == nil || !errors.Is(err, DeniedError{Identifier: entry.Identifier, Reason: "trustManifest required"}) {
		t.Fatalf("expected trustManifest denial, got %v", err)
	}

	entry.TrustManifest = map[string]any{"identity": "https://example.com"}
	statuses, _, err := Policy{RequireTrustManifest: true}.EvaluateCatalog(ard.Catalog{
		SpecVersion: "1.0",
		Entries:     []ard.CatalogEntry{entry},
	})
	if err != nil {
		t.Fatalf("evaluate trusted entry: %v", err)
	}
	if got := statuses[entry.Identifier]; got != store.LifecycleStatusActive {
		t.Fatalf("expected active status, got %s", got)
	}
}

func TestEvaluateCatalogRequiresSourceDigestForURLArtifacts(t *testing.T) {
	entry := testEntry("urn:air:example.com:server:weather", ard.TypeMCPServerCard)
	entry.TrustManifest = map[string]any{"identity": "https://example.com"}
	_, _, err := Policy{RequireSourceDigestForURLArtifacts: true}.EvaluateCatalog(ard.Catalog{
		SpecVersion: "1.0",
		Entries:     []ard.CatalogEntry{entry},
	})
	if err == nil || !errors.Is(err, DeniedError{Identifier: entry.Identifier, Reason: "sourceDigest required for url delivery"}) {
		t.Fatalf("expected sourceDigest denial, got %v", err)
	}

	entry.TrustManifest["sourceDigest"] = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if _, _, err := (Policy{RequireSourceDigestForURLArtifacts: true}).EvaluateCatalog(ard.Catalog{
		SpecVersion: "1.0",
		Entries:     []ard.CatalogEntry{entry},
	}); err != nil {
		t.Fatalf("expected pinned URL artifact to pass: %v", err)
	}

	embedded := testEntry("urn:air:example.com:skill:weather", ard.TypeAISkill)
	embedded.URL = ""
	embedded.Data = map[string]any{"markdown": "# Weather"}
	if _, _, err := (Policy{RequireSourceDigestForURLArtifacts: true}).EvaluateCatalog(ard.Catalog{
		SpecVersion: "1.0",
		Entries:     []ard.CatalogEntry{embedded},
	}); err != nil {
		t.Fatalf("embedded data should not require sourceDigest: %v", err)
	}
}

func TestEvaluateCatalogRequiresJWSSignature(t *testing.T) {
	entry := testEntry("urn:air:example.com:server:weather", ard.TypeMCPServerCard)
	entry.TrustManifest = map[string]any{"identity": "https://example.com"}
	_, _, err := Policy{RequireJWSSignature: true}.EvaluateCatalog(ard.Catalog{
		SpecVersion: "1.0",
		Entries:     []ard.CatalogEntry{entry},
	})
	if err == nil || !errors.Is(err, DeniedError{Identifier: entry.Identifier, Reason: "trustManifest.signature required"}) {
		t.Fatalf("expected signature denial, got %v", err)
	}

	entry.TrustManifest["signature"] = "detached-jws-placeholder"
	if _, _, err := (Policy{RequireJWSSignature: true}).EvaluateCatalog(ard.Catalog{
		SpecVersion: "1.0",
		Entries:     []ard.CatalogEntry{entry},
	}); err != nil {
		t.Fatalf("expected signed trust manifest to pass: %v", err)
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
