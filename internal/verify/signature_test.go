package verify

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ifuryst/ard/internal/ard"
)

func TestVerifySignatures(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	trustManifest := map[string]any{
		"identity":     "https://example.com",
		"identityType": "https",
	}
	trustManifest["signature"] = testDetachedJWS(t, "acme-ed25519", trustManifest, privateKey)

	results, err := VerifySignatures(signedCatalog(trustManifest), SignatureOptions{
		TrustAnchors: TrustAnchors{Keys: []TrustAnchorKey{
			{
				KeyID:     "acme-ed25519",
				Algorithm: "EdDSA",
				PublicKey: base64.RawURLEncoding.EncodeToString(publicKey),
			},
		}},
	})
	if err != nil {
		t.Fatalf("verify signature: %v", err)
	}
	if len(results) != 1 || !results[0].Verified || results[0].KeyID != "acme-ed25519" {
		t.Fatalf("unexpected results: %#v", results)
	}
}

func TestVerifySignaturesRejectsTampering(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	trustManifest := map[string]any{
		"identity":     "https://example.com",
		"identityType": "https",
	}
	trustManifest["signature"] = testDetachedJWS(t, "acme-ed25519", trustManifest, privateKey)
	trustManifest["identityType"] = "other"

	_, err = VerifySignatures(signedCatalog(trustManifest), SignatureOptions{
		TrustAnchors: TrustAnchors{Keys: []TrustAnchorKey{
			{
				KeyID:     "acme-ed25519",
				Algorithm: "EdDSA",
				PublicKey: base64.RawURLEncoding.EncodeToString(publicKey),
			},
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "signature verification failed") {
		t.Fatalf("expected signature verification failure, got %v", err)
	}
}

func TestVerifySignaturesCanRequireSignatures(t *testing.T) {
	_, err := VerifySignatures(signedCatalog(map[string]any{
		"identity": "https://example.com",
	}), SignatureOptions{RequireSignatures: true})
	if err == nil || !strings.Contains(err.Error(), "trustManifest.signature is required") {
		t.Fatalf("expected missing signature error, got %v", err)
	}
}

func TestVerifySignaturesRejectsUnknownKeyID(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	trustManifest := map[string]any{
		"identity": "https://example.com",
	}
	trustManifest["signature"] = testDetachedJWS(t, "missing-ed25519", trustManifest, privateKey)

	_, err = VerifySignatures(signedCatalog(trustManifest), SignatureOptions{
		TrustAnchors: TrustAnchors{Keys: []TrustAnchorKey{
			{
				KeyID:     "acme-ed25519",
				Algorithm: "EdDSA",
				PublicKey: base64.RawURLEncoding.EncodeToString(publicKey),
			},
		}},
	})
	if err == nil || !strings.Contains(err.Error(), `no trust anchor found for kid "missing-ed25519"`) {
		t.Fatalf("expected unknown key error, got %v", err)
	}
}

func signedCatalog(trustManifest map[string]any) ard.Catalog {
	return ard.Catalog{
		SpecVersion: "1.0",
		Entries: []ard.CatalogEntry{
			{
				Identifier:    "urn:air:example.com:agent:test",
				DisplayName:   "Test Agent",
				Type:          ard.TypeA2AAgentCard,
				URL:           "https://example.com/agent-card.json",
				TrustManifest: trustManifest,
			},
		},
	}
}

func testDetachedJWS(t *testing.T, keyID string, trustManifest map[string]any, privateKey ed25519.PrivateKey) string {
	t.Helper()
	protected, err := json.Marshal(jwsProtectedHeader{
		Algorithm: "EdDSA",
		KeyID:     keyID,
	})
	if err != nil {
		t.Fatalf("marshal protected header: %v", err)
	}
	protectedPart := base64.RawURLEncoding.EncodeToString(protected)
	payload, err := canonicalTrustManifestPayload(trustManifest)
	if err != nil {
		t.Fatalf("canonical payload: %v", err)
	}
	signingInput := []byte(protectedPart + "." + base64.RawURLEncoding.EncodeToString(payload))
	signature := ed25519.Sign(privateKey, signingInput)
	return protectedPart + ".." + base64.RawURLEncoding.EncodeToString(signature)
}
