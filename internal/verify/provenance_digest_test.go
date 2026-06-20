package verify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ifuryst/ard/internal/ard"
)

func TestVerifyProvenanceDigests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		_, _ = response.Write([]byte("source-artifact"))
	}))
	t.Cleanup(server.Close)

	results, err := VerifyProvenanceDigests(context.Background(), provenanceDigestCatalog(server.URL, testSHA256("source-artifact")))
	if err != nil {
		t.Fatalf("verify provenance digest: %v", err)
	}
	if len(results) != 1 || !results[0].Verified || results[0].Relation != "publishedFrom" {
		t.Fatalf("unexpected results: %#v", results)
	}
}

func TestVerifyProvenanceDigestsRejectsMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		_, _ = response.Write([]byte("changed-source"))
	}))
	t.Cleanup(server.Close)

	_, err := VerifyProvenanceDigests(context.Background(), provenanceDigestCatalog(server.URL, testSHA256("source-artifact")))
	if err == nil || !strings.Contains(err.Error(), "provenance[0].sourceDigest mismatch") {
		t.Fatalf("expected mismatch error, got %v", err)
	}
}

func TestVerifyProvenanceDigestsRequiresHTTPSourceIDForPinnedDigest(t *testing.T) {
	_, err := VerifyProvenanceDigests(context.Background(), provenanceDigestCatalog("urn:air:example.com:source:build", testSHA256("source-artifact")))
	if err == nil || !strings.Contains(err.Error(), "sourceDigest verification requires HTTP(S) sourceId") {
		t.Fatalf("expected sourceId error, got %v", err)
	}
}

func TestVerifyProvenanceDigestsCanRequirePinnedHTTPSourceIDs(t *testing.T) {
	_, err := VerifyProvenanceDigestsWithOptions(context.Background(), provenanceDigestCatalog("https://example.com/source.json", ""), ProvenanceDigestOptions{
		RequirePinnedHTTPSourceIDs: true,
	})
	if err == nil || !strings.Contains(err.Error(), "trustManifest.provenance[0].sourceDigest required for HTTP(S) sourceId") {
		t.Fatalf("expected missing provenance digest error, got %v", err)
	}
}

func TestVerifyProvenanceDigestsRequirementSkipsURNSourceIDs(t *testing.T) {
	results, err := VerifyProvenanceDigestsWithOptions(context.Background(), provenanceDigestCatalog("urn:air:example.com:source:build", ""), ProvenanceDigestOptions{
		RequirePinnedHTTPSourceIDs: true,
	})
	if err != nil {
		t.Fatalf("verify unpinned URN provenance: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("unexpected results: %#v", results)
	}
}

func provenanceDigestCatalog(sourceID string, digest string) ard.Catalog {
	link := map[string]any{
		"relation": "publishedFrom",
		"sourceId": sourceID,
	}
	if digest != "" {
		link["sourceDigest"] = digest
	}
	return ard.Catalog{
		SpecVersion: "1.0",
		Entries: []ard.CatalogEntry{
			{
				Identifier:  "urn:air:example.com:agent:test",
				DisplayName: "Test Agent",
				Type:        ard.TypeA2AAgentCard,
				URL:         "https://example.com/agent-card.json",
				TrustManifest: map[string]any{
					"identity":   "https://example.com",
					"provenance": []any{link},
				},
			},
		},
	}
}
