package verify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ifuryst/ard/internal/ard"
	"github.com/ifuryst/ard/internal/requestid"
)

func TestVerifySourceDigests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		_, _ = response.Write([]byte("artifact"))
	}))
	t.Cleanup(server.Close)

	results, err := VerifySourceDigests(context.Background(), ard.Catalog{
		SpecVersion: "1.0",
		Entries: []ard.CatalogEntry{
			{
				Identifier:  "urn:air:example.com:agent:test",
				DisplayName: "Test Agent",
				Type:        ard.TypeA2AAgentCard,
				URL:         server.URL,
				TrustManifest: map[string]any{
					"identity":     "https://example.com",
					"sourceDigest": "sha256:c7c5c1d70c5dec4416ab6158afd0b223ef40c29b1dc1f97ed9428b94d4cadb1c",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("verify source digest: %v", err)
	}
	if len(results) != 1 || !results[0].Verified {
		t.Fatalf("unexpected results: %#v", results)
	}
}

func TestVerifySourceDigestsPropagatesRequestID(t *testing.T) {
	seenRequestID := ""
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		seenRequestID = request.Header.Get(requestid.Header)
		_, _ = response.Write([]byte("artifact"))
	}))
	t.Cleanup(server.Close)

	_, err := VerifySourceDigests(requestid.With(context.Background(), "verify-source-request"), ard.Catalog{
		SpecVersion: "1.0",
		Entries: []ard.CatalogEntry{
			{
				Identifier:  "urn:air:example.com:agent:test",
				DisplayName: "Test Agent",
				Type:        ard.TypeA2AAgentCard,
				URL:         server.URL,
				TrustManifest: map[string]any{
					"identity":     "https://example.com",
					"sourceDigest": "sha256:c7c5c1d70c5dec4416ab6158afd0b223ef40c29b1dc1f97ed9428b94d4cadb1c",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("verify source digest: %v", err)
	}
	if seenRequestID != "verify-source-request" {
		t.Fatalf("expected request ID propagation, got %q", seenRequestID)
	}
}

func TestVerifySourceDigestsRejectsMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		_, _ = response.Write([]byte("changed"))
	}))
	t.Cleanup(server.Close)

	_, err := VerifySourceDigests(context.Background(), ard.Catalog{
		SpecVersion: "1.0",
		Entries: []ard.CatalogEntry{
			{
				Identifier:  "urn:air:example.com:agent:test",
				DisplayName: "Test Agent",
				Type:        ard.TypeA2AAgentCard,
				URL:         server.URL,
				TrustManifest: map[string]any{
					"identity":     "https://example.com",
					"sourceDigest": "sha256:b461ef6b49651b421d8e5b6b668150b849cbf5b3b88f621f3039e9a7219e7f6f",
				},
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "sourceDigest mismatch") {
		t.Fatalf("expected mismatch error, got %v", err)
	}
}
