package verify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ifuryst/ard/internal/ard"
)

func TestVerifyAttestationDigests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		_, _ = response.Write([]byte("soc2-attestation"))
	}))
	t.Cleanup(server.Close)

	results, err := VerifyAttestationDigests(context.Background(), attestationDigestCatalog(server.URL, testSHA256("soc2-attestation")))
	if err != nil {
		t.Fatalf("verify attestation digest: %v", err)
	}
	if len(results) != 1 || !results[0].Verified || results[0].Type != "SOC2-Type2" {
		t.Fatalf("unexpected results: %#v", results)
	}
}

func TestVerifyAttestationDigestsRejectsMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		_, _ = response.Write([]byte("changed-attestation"))
	}))
	t.Cleanup(server.Close)

	_, err := VerifyAttestationDigests(context.Background(), attestationDigestCatalog(server.URL, testSHA256("soc2-attestation")))
	if err == nil || !strings.Contains(err.Error(), "attestations[0].digest mismatch") {
		t.Fatalf("expected mismatch error, got %v", err)
	}
}

func TestVerifyAttestationDigestsCanRequirePinnedAttestations(t *testing.T) {
	_, err := VerifyAttestationDigestsWithOptions(context.Background(), attestationDigestCatalog("https://example.com/soc2.pdf", ""), AttestationDigestOptions{
		RequirePinnedAttestations: true,
	})
	if err == nil || !strings.Contains(err.Error(), "trustManifest.attestations[0].digest required") {
		t.Fatalf("expected missing attestation digest error, got %v", err)
	}
}

func TestVerifyAttestationDigestsSkipsUnpinnedAttestations(t *testing.T) {
	results, err := VerifyAttestationDigests(context.Background(), attestationDigestCatalog("https://example.com/soc2.pdf", ""))
	if err != nil {
		t.Fatalf("verify unpinned attestation: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("unexpected results: %#v", results)
	}
}

func attestationDigestCatalog(uri string, digest string) ard.Catalog {
	attestation := map[string]any{
		"type":      "SOC2-Type2",
		"uri":       uri,
		"mediaType": "application/pdf",
	}
	if digest != "" {
		attestation["digest"] = digest
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
					"identity":     "https://example.com",
					"attestations": []any{attestation},
				},
			},
		},
	}
}

func testSHA256(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}
